/*
Package graceful implements graceful shutdown for HTTP servers by closing idle
connections after receiving a signal. By default, this package listens for
interrupts (i.e., SIGINT), but when it detects that it is running under Einhorn
it will additionally listen for SIGUSR2 as well, giving your application
automatic support for graceful restarts/code upgrades.
*/
package graceful

import (
	"net"
	"runtime"
	"sync/atomic"

	"github.com/zenazn/goji/graceful/listener"
)

// WrapListener wraps an arbitrary net.Listener for use with graceful shutdowns.
// In the background, it uses the listener sub-package to Wrap the listener in
// Deadline mode. If another mode of operation is desired, you should call
// listener.Wrap yourself: this function is smart enough to not double-wrap
// listeners.
func WrapListener(l net.Listener) net.Listener {
	if lt, ok := l.(*listener.T); ok {
		appendListener(lt)
		return lt
	}

	lt := listener.Wrap(l, listener.Deadline)
	appendListener(lt)
	return lt
}

func appendListener(l *listener.T) {
	mu.Lock()
	defer mu.Unlock()

	listeners = append(listeners, l)
}

const errClosing = "use of closed network connection"

// During graceful shutdown, calls to Accept will start returning errors. This
// is inconvenient, since we know these sorts of errors are peaceful, so we
// silently swallow them.
func peacefulError(err error) error {
	if atomic.LoadInt32(&closing) == 0 {
		return err
	}
	// Unfortunately Go doesn't really give us a better way to select errors
	// than this, so *shrug*.
	if oe, ok := err.(*net.OpError); ok {
		errOp := "accept"
		if runtime.GOOS == "windows" {
			errOp = "AcceptEx"
		}
		if oe.Op == errOp && oe.Err.Error() == errClosing {
			return nil
		}
	}
	return err
}
