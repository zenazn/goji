package listener

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type conn struct {
	net.Conn

	shard *shard
	mode  mode

	mu       sync.Mutex // Protects the state machine below
	busy     bool       // connection is in use (i.e., not idle)
	closed   bool       // connection is closed
	disowned bool       // if true, this connection is no longer under our management
}

// This intentionally looks a lot like the one in package net.
var errClosing = errors.New("use of closed network connection")

func (c *conn) init() error {
	c.shard.wg.Add(1)
	if shouldExit := c.shard.track(c); shouldExit {
		c.Close()
		return errClosing
	}
	return nil
}

func (c *conn) Read(b []byte) (n int, err error) {
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		if c.disowned {
			return
		}

		// This protects against a Close/Read race. We're not really
		// concerned about the general case (it's fundamentally racy),
		// but are mostly trying to prevent a race between a new request
		// getting read off the wire in one thread while the connection
		// is being gracefully shut down in another.
		if c.closed && err == nil {
			n = 0
			err = errClosing
			return
		}

		if c.mode != Manual && !c.busy && !c.closed {
			c.busy = true
			c.shard.markInUse(c)
		}
	}()

	return c.Conn.Read(b)
}

func (c *conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disowned {
		return c.Conn.Close()
	} else if c.closed {
		return errClosing
	}

	c.closed = true
	c.shard.disown(c)
	defer c.shard.wg.Done()

	return c.Conn.Close()
}

func (c *conn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	if !c.disowned && c.mode == Deadline {
		defer c.markIdle()
	}
	c.mu.Unlock()
	return c.Conn.SetReadDeadline(t)
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(c.Conn, r)
}

func (c *conn) markIdle() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.busy {
		return
	}
	c.busy = false

	if exit := c.shard.markIdle(c); exit && !c.closed && !c.disowned {
		c.closed = true
		c.shard.disown(c)
		defer c.shard.wg.Done()
		c.Conn.Close()
		return
	}
}

func (c *conn) markInUse() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.busy && !c.closed && !c.disowned {
		c.busy = true
		c.shard.markInUse(c)
	}
}

func (c *conn) closeIfIdle() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.busy && !c.closed && !c.disowned {
		c.closed = true
		c.shard.disown(c)
		defer c.shard.wg.Done()
		return c.Conn.Close()
	}

	return nil
}

var errAlreadyDisowned = errors.New("listener: conn already disowned")

func (c *conn) disown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.disowned {
		return errAlreadyDisowned
	}

	c.shard.disown(c)
	c.disowned = true
	c.shard.wg.Done()

	return nil
}
