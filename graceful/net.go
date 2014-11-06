package graceful

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type listener struct {
	net.Listener
}

// WrapListener wraps an arbitrary net.Listener for use with graceful shutdowns.
// All net.Conn's Accept()ed by this listener will be auto-wrapped as if
// WrapConn() were called on them.
func WrapListener(l net.Listener) net.Listener {
	return listener{l}
}

func (l listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	return WrapConn(conn), err
}

type conn struct {
	mu sync.Mutex
	cs *connSet
	net.Conn
	id        uint64
	busy, die bool
	dead      bool
	hijacked  bool
}

/*
WrapConn wraps an arbitrary connection for use with graceful shutdowns. The
graceful shutdown process will ensure that this connection is closed before
terminating the process.

In order to use this function, you must call SetReadDeadline() before the call
to Read() you might make to read a new request off the wire.  The connection is
eligible for abrupt closing at any point between when the call to
SetReadDeadline() returns and when the call to Read returns with new data. It
does not matter what deadline is given to SetReadDeadline()--if a deadline is
inappropriate, providing one extremely far into the future will suffice.

Unfortunately, this means that it's difficult to use SetReadDeadline() in a
great many perfectly reasonable circumstances, such as to extend a deadline
after more data has been read, without the connection being eligible for
"graceful" termination at an undesirable time. Since this package was written
explicitly to target net/http, which does not as of this writing do any of this,
fixing the semantics here does not seem especially urgent.
*/
func WrapConn(c net.Conn) net.Conn {
	if c == nil {
		return nil
	}

	wgLock.Lock()
	defer wgLock.Unlock()
	wg.Add(1)

	return &conn{
		Conn: c,
		id:   atomic.AddUint64(&idleSet.id, 1),
	}
}

func (c *conn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	if !c.hijacked {
		defer func() {
			c.mu.Lock()
			if c.hijacked {
				// It's a little unclear to me how this case
				// would happen, but we *did* drop the lock, so
				// let's play it safe.
				return
			}

			if c.dead {
				// Dead sockets don't tell tales. This is to
				// prevent the case where a Read manages to suck
				// an entire request off the wire in a race with
				// someone trying to close idle connections.
				// Whoever grabs the conn lock first wins, and
				// if that's the closing process, we need to
				// "take back" the read.
				n = 0
				err = io.EOF
			} else {
				idleSet.markBusy(c)
			}
			c.mu.Unlock()
		}()
	}
	c.mu.Unlock()

	return c.Conn.Read(b)
}

func (c *conn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	if !c.hijacked {
		defer c.markIdle()
	}
	c.mu.Unlock()
	return c.Conn.SetReadDeadline(t)
}

func (c *conn) Close() error {
	kill := false
	c.mu.Lock()
	kill, c.dead = !c.dead, true
	idleSet.markBusy(c)
	c.mu.Unlock()

	if kill {
		defer wg.Done()
	}
	return c.Conn.Close()
}

type writerOnly struct {
	w io.Writer
}

func (w writerOnly) Write(buf []byte) (int, error) {
	return w.w.Write(buf)
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := c.Conn.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(writerOnly{c}, r)
}

func (c *conn) markIdle() {
	kill := false
	c.mu.Lock()
	idleSet.markIdle(c)
	if c.die {
		kill, c.dead = !c.dead, true
	}
	c.mu.Unlock()

	if kill {
		defer wg.Done()
		c.Conn.Close()
	}

}

func (c *conn) closeIfIdle() {
	kill := false
	c.mu.Lock()
	c.die = true
	if !c.busy && !c.hijacked {
		kill, c.dead = !c.dead, true
	}
	c.mu.Unlock()

	if kill {
		defer wg.Done()
		c.Conn.Close()
	}
}

func (c *conn) hijack() net.Conn {
	c.mu.Lock()
	idleSet.markBusy(c)
	c.hijacked = true
	c.mu.Unlock()

	return c.Conn
}
