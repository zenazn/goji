/*
Package listener provides a way to incorporate graceful shutdown to any
net.Listener.

This package provides low-level primitives, not a high-level API. If you're
looking for a package that provides graceful shutdown for HTTP servers, I
recommend this package's parent package, github.com/zenazn/goji/graceful.
*/
package listener

import (
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

type mode int8

const (
	// Manual mode is completely manual: users must use use MarkIdle and
	// MarkInUse to indicate when connections are busy servicing requests or
	// are eligible for termination.
	Manual mode = iota
	// Automatic mode is what most users probably want: calling Read on a
	// connection will mark it as in use, but users must manually call
	// MarkIdle to indicate when connections may be safely closed.
	Automatic
	// Deadline mode is like automatic mode, except that calling
	// SetReadDeadline on a connection will also mark it as being idle. This
	// is useful for many servers like net/http, where SetReadDeadline is
	// used to implement read timeouts on new requests.
	Deadline
)

// Wrap a net.Listener, returning a net.Listener which supports idle connection
// tracking and shutdown. Listeners can be placed in to one of three modes,
// exported as variables from this package: most users will probably want the
// "Automatic" mode.
func Wrap(l net.Listener, m mode) *T {
	t := &T{
		l:    l,
		mode: m,
		// To keep the expected contention rate constant we'd have to
		// grow this as numcpu**2. In practice, CPU counts don't
		// generally grow without bound, and contention is probably
		// going to be small enough that nobody cares anyways.
		shards: make([]shard, 2*runtime.NumCPU()),
	}
	for i := range t.shards {
		t.shards[i].init(t)
	}
	return t
}

// T is the type of this package's graceful listeners.
type T struct {
	mu sync.Mutex
	l  net.Listener

	// TODO(carl): a count of currently outstanding connections.
	connCount uint64
	shards    []shard

	mode mode
}

var _ net.Listener = &T{}

// Accept waits for and returns the next connection to the listener. The
// returned net.Conn's idleness is tracked, and idle connections can be closed
// from the associated T.
func (t *T) Accept() (net.Conn, error) {
	c, err := t.l.Accept()
	if err != nil {
		return nil, err
	}

	connID := atomic.AddUint64(&t.connCount, 1)
	shard := &t.shards[int(connID)%len(t.shards)]
	wc := &conn{
		Conn:  c,
		shard: shard,
		mode:  t.mode,
	}

	if err = wc.init(); err != nil {
		return nil, err
	}
	return wc, nil
}

// Addr returns the wrapped listener's network address.
func (t *T) Addr() net.Addr {
	return t.l.Addr()
}

// Close closes the wrapped listener.
func (t *T) Close() error {
	return t.l.Close()
}

// CloseIdle closes all connections that are currently marked as being idle. It,
// however, makes no attempt to wait for in-use connections to die, or to close
// connections which become idle in the future. Call this function if you're
// interested in shedding useless connections, but otherwise wish to continue
// serving requests.
func (t *T) CloseIdle() error {
	for i := range t.shards {
		t.shards[i].closeConns(false, false)
	}
	// Not sure if returning errors is actually useful here :/
	return nil
}

// Drain immediately closes all idle connections, prevents new connections from
// being accepted, and waits for all outstanding connections to finish.
//
// Once a listener has been drained, there is no way to re-enable it. You
// probably want to Close the listener before draining it, otherwise new
// connections will be accepted and immediately closed.
func (t *T) Drain() error {
	for i := range t.shards {
		t.shards[i].closeConns(false, true)
	}
	for i := range t.shards {
		t.shards[i].wait()
	}
	return nil
}

// DrainAll closes all connections currently tracked by this listener (both idle
// and in-use connections), and prevents new connections from being accepted.
// Disowned connections are not closed.
func (t *T) DrainAll() error {
	for i := range t.shards {
		t.shards[i].closeConns(true, true)
	}
	for i := range t.shards {
		t.shards[i].wait()
	}
	return nil
}

var errNotManaged = errors.New("listener: passed net.Conn is not managed by this package")

// Disown causes a connection to no longer be tracked by the listener. The
// passed connection must have been returned by a call to Accept from this
// listener.
func Disown(c net.Conn) error {
	if cn, ok := c.(*conn); ok {
		return cn.disown()
	}
	return errNotManaged
}

// MarkIdle marks the given connection as being idle, and therefore eligible for
// closing at any time. The passed connection must have been returned by a call
// to Accept from this listener.
func MarkIdle(c net.Conn) error {
	if cn, ok := c.(*conn); ok {
		cn.markIdle()
		return nil
	}
	return errNotManaged
}

// MarkInUse marks this connection as being in use, removing it from the set of
// connections which are eligible for closing. The passed connection must have
// been returned by a call to Accept from this listener.
func MarkInUse(c net.Conn) error {
	if cn, ok := c.(*conn); ok {
		cn.markInUse()
		return nil
	}
	return errNotManaged
}
