package graceful

import (
	"os"
	"os/signal"
	"sync"
	"time"
)

// This is the channel that the connections select on. When it is closed, the
// connections should gracefully exit.
var kill = make(chan struct{})

// This is the channel that the Wait() function selects on. It should only be
// closed once all the posthooks have been called.
var wait = make(chan struct{})

// Whether new requests should be accepted. When false, new requests are refused.
var acceptingRequests bool = true

// This is the WaitGroup that indicates when all the connections have gracefully
// shut down.
var wg sync.WaitGroup
var wgLock sync.Mutex

// This lock protects the list of pre- and post- hooks below.
var hookLock sync.Mutex
var prehooks = make([]func(), 0)
var posthooks = make([]func(), 0)

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
	hookLock.Lock()
	defer hookLock.Unlock()

	prehooks = append(prehooks, f)
}

// PostHook registers a function to be called after all of this package's normal
// shutdown actions. All listeners will be called in the order they were added,
// from a single goroutine, and are guaranteed to be called after all listening
// connections have been closed, but before Wait() returns.
//
// If you've Hijack()ed any connections that must be gracefully shut down in
// some other way (since this library disowns all hijacked connections), it's
// reasonable to use a PostHook() to signal and wait for them.
func PostHook(f func()) {
	hookLock.Lock()
	defer hookLock.Unlock()

	posthooks = append(posthooks, f)
}

func waitForSignal() {
	<-sigchan

	// Prevent servicing of any new requests.
	wgLock.Lock()
	acceptingRequests = false
	wgLock.Unlock()

	hookLock.Lock()
	defer hookLock.Unlock()

	for _, f := range prehooks {
		f()
	}

	var finished chan struct{} = make(chan struct{}, 1)
	go shutdownWatcher(finished)

	close(kill)
	wg.Wait()
	close(finished)

	for _, f := range posthooks {
		f()
	}

	close(wait)
}

func shutdownWatcher(finished chan struct{}) {
	select {
	case <-finished:
		return
	case <-time.After(7 * time.Second):
		os.Exit(-1)
	}
}

// Wait for all connections to gracefully shut down. This is commonly called at
// the bottom of the main() function to prevent the program from exiting
// prematurely.
func Wait() {
	<-wait
}
