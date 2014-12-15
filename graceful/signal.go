package graceful

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"

	"github.com/zenazn/goji/graceful/listener"
)

var mu sync.Mutex // protects everything that follows
var listeners = make([]*listener.T, 0)
var prehooks = make([]func(), 0)
var posthooks = make([]func(), 0)
var closing int32

var wait = make(chan struct{})
var stdSignals = []os.Signal{os.Interrupt}
var sigchan = make(chan os.Signal, 1)

func init() {
	go waitForSignal()
}

// HandleSignals installs signal handlers for a set of standard signals. By
// default, this set only includes keyboard interrupts, however when the package
// detects that it is running under Einhorn, a SIGUSR2 handler is installed as
// well.
func HandleSignals() {
	AddSignal(stdSignals...)
}

// AddSignal adds the given signal to the set of signals that trigger a graceful
// shutdown.
func AddSignal(sig ...os.Signal) {
	signal.Notify(sigchan, sig...)
}

// ResetSignals resets the list of signals that trigger a graceful shutdown.
func ResetSignals() {
	signal.Stop(sigchan)
}

type userShutdown struct{}

func (u userShutdown) String() string {
	return "application initiated shutdown"
}
func (u userShutdown) Signal() {}

// Shutdown manually triggers a shutdown from your application. Like Wait(),
// blocks until all connections have gracefully shut down.
func Shutdown() {
	sigchan <- userShutdown{}
	<-wait
}

// PreHook registers a function to be called before any of this package's normal
// shutdown actions. All listeners will be called in the order they were added,
// from a single goroutine.
func PreHook(f func()) {
	mu.Lock()
	defer mu.Unlock()

	prehooks = append(prehooks, f)
}

// PostHook registers a function to be called after all of this package's normal
// shutdown actions. All listeners will be called in the order they were added,
// from a single goroutine, and are guaranteed to be called after all listening
// connections have been closed, but before Wait() returns.
//
// If you've Hijacked any connections that must be gracefully shut down in some
// other way (since this library disowns all hijacked connections), it's
// reasonable to use a PostHook to signal and wait for them.
func PostHook(f func()) {
	mu.Lock()
	defer mu.Unlock()

	posthooks = append(posthooks, f)
}

func waitForSignal() {
	<-sigchan

	mu.Lock()
	defer mu.Unlock()

	for _, f := range prehooks {
		f()
	}

	atomic.StoreInt32(&closing, 1)
	var wg sync.WaitGroup
	wg.Add(len(listeners))
	for _, l := range listeners {
		go func(l *listener.T) {
			defer wg.Done()
			l.Close()
			l.Drain()
		}(l)
	}
	wg.Wait()

	for _, f := range posthooks {
		f()
	}

	close(wait)
}

// Wait for all connections to gracefully shut down. This is commonly called at
// the bottom of the main() function to prevent the program from exiting
// prematurely.
func Wait() {
	<-wait
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
		if oe.Op == "accept" && oe.Err.Error() == errClosing {
			return nil
		}
	}
	return err
}
