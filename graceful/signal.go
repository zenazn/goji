package graceful

import (
	"log"
	"os"
	"os/signal"
	"sync"
)

// This is the channel that the connections select on. When it is closed, the
// connections should gracefully exit.
var kill = make(chan struct{})

// This is the channel that the Wait() function selects on. It should only be
// closed once all the posthooks have been called.
var wait = make(chan struct{})

// This is the WaitGroup that indicates when all the connections have gracefully
// shut down.
var wg sync.WaitGroup

// This lock protects the list of pre- and post- hooks below.
var hookLock sync.Mutex
var prehooks = make([]func(), 0)
var posthooks = make([]func(), 0)

var sigchan = make(chan os.Signal, 1)

func init() {
	AddSignal(os.Interrupt)
	go waitForSignal()
}

// AddSignal adds the given signal to the set of signals that trigger a graceful
// shutdown. Note that for convenience the default interrupt (SIGINT) handler is
// installed at package load time, and unless you call ResetSignals() will be
// listened for in addition to any signals you provide by calling this function.
func AddSignal(sig ...os.Signal) {
	signal.Notify(sigchan, sig...)
}

// ResetSignals resets the list of signals that trigger a graceful shutdown.
// Useful if, for instance, you don't want to use the default interrupt (SIGINT)
// handler. Since we necessarily install the SIGINT handler before you have a
// chance to call ResetSignals(), there will be a brief window during which the
// set of signals this package listens for will not be as you intend. Therefore,
// if you intend on using this function, we encourage you to call it as soon as
// possible.
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
	sig := <-sigchan
	log.Printf("Received %v, gracefully shutting down!", sig)

	hookLock.Lock()
	defer hookLock.Unlock()

	for _, f := range prehooks {
		f()
	}

	close(kill)
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
