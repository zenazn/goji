package graceful

import (
	"io"
	"net"
	"sync"
	"time"
)

type listener struct {
	net.Listener
}

type gracefulConn interface {
	gracefulShutdown()
}

// WrapListener wraps an arbitrary net.Listener for use with graceful shutdowns.
// All net.Conn's Accept()ed by this listener will be auto-wrapped as if
// WrapConn() were called on them.
func WrapListener(l net.Listener) net.Listener {
	return listener{l}
}

func (l listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return WrapConn(conn), nil
}

/*
WrapConn wraps an arbitrary connection for use with graceful shutdowns. The
graceful shutdown process will ensure that this connection is closed before
terminating the process.

In order to use this function, you must call SetReadDeadline() before the call
to Read() you might make to read a new request off the wire.  The connection is
eligible for abrupt closing at any point between when the call to
SetReadDeadline() returns and when the call to Read returns with new data.  It
does not matter what deadline is given to SetReadDeadline()--the default HTTP
server provided by this package sets a deadline far into the future when a
deadline is not provided, for instance.

Unfortunately, this means that it's difficult to use SetReadDeadline() in a
great many perfectly reasonable circumstances, such as to extend a deadline
after more data has been read, without the connection being eligible for
"graceful" termination at an undesirable time. Since this package was written
explicitly to target net/http, which does not as of this writing do any of this,
fixing the semantics here does not seem especially urgent.

As an optimization for net/http over TCP, if the input connection supports the
ReadFrom() function, the returned connection will as well. This allows the net
package to use sendfile(2) on certain platforms in certain circumstances.
*/
func WrapConn(c net.Conn) net.Conn {
	wg.Add(1)

	nc := conn{
		Conn:    c,
		closing: make(chan struct{}),
	}

	if _, ok := c.(io.ReaderFrom); ok {
		c = &sendfile{nc}
	} else {
		c = &nc
	}

	go c.(gracefulConn).gracefulShutdown()

	return c
}

type connstate int

/*
State diagram. (Waiting) is the starting state.

(Waiting) -----Read()-----> Working ---+
    |   ^                   /  |  ^   Read()
    |    \                 /   |  +----+
   kill   SetReadDeadline()   kill
    |                          |  +-----+
    V                          V  V   Read()
  Dead <-SetReadDeadline()-- Dying ----+
    ^
    |
    +--Close()--- [from any state]

*/

const (
	// Waiting for more data, and eligible for killing
	csWaiting connstate = iota
	// In the middle of a connection
	csWorking
	// Kill has been requested, but waiting on request to finish up
	csDying
	// Connection is gone forever. Also used when a connection gets hijacked
	csDead
)

type conn struct {
	net.Conn
	m       sync.Mutex
	state   connstate
	closing chan struct{}
}
type sendfile struct{ conn }

func (c *conn) gracefulShutdown() {
	select {
	case <-kill:
	case <-c.closing:
		return
	}
	c.m.Lock()
	defer c.m.Unlock()

	switch c.state {
	case csWaiting:
		c.unlockedClose(true)
	case csWorking:
		c.state = csDying
	}
}

func (c *conn) unlockedClose(closeConn bool) {
	if closeConn {
		c.Conn.Close()
	}
	close(c.closing)
	wg.Done()
	c.state = csDead
}

// We do some hijinks to support hijacking. The semantics here is that any
// connection that gets hijacked is dead to us: we return the raw net.Conn and
// stop tracking the connection entirely.
type hijackConn interface {
	hijack() net.Conn
}

func (c *conn) hijack() net.Conn {
	c.m.Lock()
	defer c.m.Unlock()
	if c.state != csDead {
		close(c.closing)
		wg.Done()
		c.state = csDead
	}
	return c.Conn
}

func (c *conn) Read(b []byte) (n int, err error) {
	defer func() {
		c.m.Lock()
		defer c.m.Unlock()

		if c.state == csWaiting {
			c.state = csWorking
		}
	}()

	return c.Conn.Read(b)
}
func (c *conn) Close() error {
	defer func() {
		c.m.Lock()
		defer c.m.Unlock()

		if c.state != csDead {
			c.unlockedClose(false)
		}
	}()
	return c.Conn.Close()
}
func (c *conn) SetReadDeadline(t time.Time) error {
	defer func() {
		c.m.Lock()
		defer c.m.Unlock()
		switch c.state {
		case csDying:
			c.unlockedClose(false)
		case csWorking:
			c.state = csWaiting
		}
	}()
	return c.Conn.SetReadDeadline(t)
}

func (s *sendfile) ReadFrom(r io.Reader) (int64, error) {
	// conn.Conn.KHAAAAAAAANNNNNN
	return s.conn.Conn.(io.ReaderFrom).ReadFrom(r)
}
