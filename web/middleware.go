package web

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Maximum size of the pool of spare middleware stacks
const mPoolSize = 32

type mLayer struct {
	fn   func(*C, http.Handler) http.Handler
	orig interface{}
}

type mStack struct {
	lock   sync.Mutex
	stack  []mLayer
	pool   chan *cStack
	router Handler
}

// Constructing a middleware stack involves a lot of allocations: at the very
// least each layer will have to close over the layer after (inside) it, and
// perhaps a context object. Instead of doing this on every request, let's cache
// fully assembled middleware stacks (the "c" stands for "cached").
type cStack struct {
	C
	m    http.Handler
	pool chan *cStack
}

func (s *cStack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.C = C{}
	s.m.ServeHTTP(w, r)
}
func (s *cStack) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	s.C = c
	s.m.ServeHTTP(w, r)
}

func (m *mStack) appendLayer(fn interface{}) {
	ml := mLayer{orig: fn}
	switch fn.(type) {
	case func(http.Handler) http.Handler:
		unwrapped := fn.(func(http.Handler) http.Handler)
		ml.fn = func(c *C, h http.Handler) http.Handler {
			return unwrapped(h)
		}
	case func(*C, http.Handler) http.Handler:
		ml.fn = fn.(func(*C, http.Handler) http.Handler)
	default:
		log.Fatalf(`Unknown middleware type %v. Expected a function `+
			`with signature "func(http.Handler) http.Handler" or `+
			`"func(*web.C, http.Handler) http.Handler".`, fn)
	}
	m.stack = append(m.stack, ml)
}

func (m *mStack) findLayer(l interface{}) int {
	for i, middleware := range m.stack {
		if funcEqual(l, middleware.orig) {
			return i
		}
	}
	return -1
}

func (m *mStack) invalidate() {
	old := m.pool
	m.pool = make(chan *cStack, mPoolSize)
	close(old)
	// Bleed down the old pool so it gets GC'd
	for _ = range old {
	}
}

func (m *mStack) newStack() *cStack {
	m.lock.Lock()
	defer m.lock.Unlock()

	cs := cStack{}
	router := m.router

	cs.m = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTPC(cs.C, w, r)
	})
	for i := len(m.stack) - 1; i >= 0; i-- {
		cs.m = m.stack[i].fn(&cs.C, cs.m)
	}

	return &cs
}

func (m *mStack) alloc() *cStack {
	// This is a little sloppy: this is only safe if this pointer
	// dereference is atomic. Maybe someday I'll replace it with
	// sync/atomic, but for now I happen to know that on all the
	// architecures I care about it happens to be atomic.
	p := m.pool
	var cs *cStack
	select {
	case cs = <-p:
		// This can happen if we race against an invalidation. It's
		// completely peaceful, so long as we assume we can grab a cStack before
		// our stack blows out.
		if cs == nil {
			return m.alloc()
		}
	default:
		cs = m.newStack()
	}

	cs.pool = p
	return cs
}

func (m *mStack) release(cs *cStack) {
	if cs.pool != m.pool {
		return
	}
	// It's possible that the pool has been invalidated (and closed) between
	// the check above and now, in which case we'll start panicing, which is
	// dumb. I'm not sure this is actually better than just grabbing a lock,
	// but whatever.
	defer func() {
		recover()
	}()
	select {
	case cs.pool <- cs:
	default:
	}
}

// Append the given middleware to the middleware stack. See the documentation
// for type Mux for a list of valid middleware types.
//
// No attempt is made to enforce the uniqueness of middlewares.
func (m *mStack) Use(middleware interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.appendLayer(middleware)
	m.invalidate()
}

// Insert the given middleware immediately before a given existing middleware in
// the stack. See the documentation for type Mux for a list of valid middleware
// types. Returns an error if no middleware has the name given by "before."
//
// No attempt is made to enforce the uniqueness of middlewares. If the insertion
// point is ambiguous, the first (outermost) one is chosen.
func (m *mStack) Insert(middleware, before interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	i := m.findLayer(before)
	if i < 0 {
		return fmt.Errorf("web: unknown middleware %v", before)
	}

	m.appendLayer(middleware)
	inserted := m.stack[len(m.stack)-1]
	copy(m.stack[i+1:], m.stack[i:])
	m.stack[i] = inserted

	m.invalidate()
	return nil
}

// Remove the given middleware from the middleware stack. Returns an error if
// no such middleware can be found.
//
// If the name of the middleware to delete is ambiguous, the first (outermost)
// one is chosen.
func (m *mStack) Abandon(middleware interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	i := m.findLayer(middleware)
	if i < 0 {
		return fmt.Errorf("web: unknown middleware %v", middleware)
	}

	m.stack = m.stack[:i+copy(m.stack[i:], m.stack[i+1:])]

	m.invalidate()
	return nil
}
