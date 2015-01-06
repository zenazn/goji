package graceful

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zenazn/goji/graceful/listener"
)

var mu sync.Mutex // protects everything that follows
var listeners = make([]*listener.T, 0)
var prehooks = make([]func(), 0)
var posthooks = make([]func(), 0)
var closing int32
var doubleKick, timeout time.Duration

var wait = make(chan struct{})
var stdSignals = []os.Signal{os.Interrupt}
var sigchan = make(chan os.Signal, 1)

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

// Shutdown manually triggers a shutdown from your application. Like Wait,
// blocks until all connections have gracefully shut down.
func Shutdown() {
	shutdown(false)
}

// ShutdownNow triggers an immediate shutdown from your application. All
// connections (not just those that are idle) are immediately closed, even if
// they are in the middle of serving a request.
func ShutdownNow() {
	shutdown(true)
}

// DoubleKickWindow sets the length of the window during which two back-to-back
// signals are treated as an especially urgent or forceful request to exit
// (i.e., ShutdownNow instead of Shutdown). Signals delivered more than this
// duration apart are treated as separate requests to exit gracefully as usual.
//
// Setting DoubleKickWindow to 0 disables the feature.
func DoubleKickWindow(d time.Duration) {
	if d < 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	doubleKick = d
}

// Timeout sets the maximum amount of time package graceful will wait for
// connections to gracefully shut down after receiving a signal. After this
// timeout, connections will be forcefully shut down (similar to calling
// ShutdownNow).
//
// Setting Timeout to 0 disables the feature.
func Timeout(d time.Duration) {
	if d < 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	timeout = d
}

// Wait for all connections to gracefully shut down. This is commonly called at
// the bottom of the main() function to prevent the program from exiting
// prematurely.
func Wait() {
	<-wait
}

func init() {
	go sigLoop()
}
func sigLoop() {
	var last time.Time
	for {
		<-sigchan
		now := time.Now()
		mu.Lock()
		force := doubleKick != 0 && now.Sub(last) < doubleKick
		if t := timeout; t != 0 && !force {
			go func() {
				time.Sleep(t)
				shutdown(true)
			}()
		}
		mu.Unlock()
		go shutdown(force)
		last = now
	}
}

var preOnce, closeOnce, forceOnce, postOnce, notifyOnce sync.Once

func shutdown(force bool) {
	preOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, f := range prehooks {
			f()
		}
	})

	if force {
		forceOnce.Do(func() {
			closeListeners(force)
		})
	} else {
		closeOnce.Do(func() {
			closeListeners(force)
		})
	}

	postOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, f := range posthooks {
			f()
		}
	})

	notifyOnce.Do(func() {
		close(wait)
	})
}

func closeListeners(force bool) {
	atomic.StoreInt32(&closing, 1)

	var wg sync.WaitGroup
	defer wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	wg.Add(len(listeners))

	for _, l := range listeners {
		go func(l *listener.T) {
			defer wg.Done()
			l.Close()
			if force {
				l.DrainAll()
			} else {
				l.Drain()
			}
		}(l)
	}
}
