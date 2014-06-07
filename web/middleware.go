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
	router internalRouter
}

type internalRouter interface {
	route(*C, http.ResponseWriter, *http.Request)
}

/*
Constructing a middleware stack involves a lot of allocations: at the very least
each layer will have to close over the layer after (inside) it, and perhaps a
context object. Instead of doing this on every request, let's cache fully
assembled middleware stacks (the "c" stands for "cached").

A lot of the complexity here (in particular the "pool" parameter, and the
behavior of release() and invalidate() below) is due to the fact that when the
middleware stack is mutated we need to create a "cache barrier," where no
cStack created before the middleware stack mutation is returned to the active
cache pool (and is therefore eligible for subsequent reuse). The way we do this
is a bit ugly: each cStack maintains a pointer to the pool it originally came
from, and will only return itself to that pool. If the mStack's pool has been
rotated since then (meaning that this cStack is invalid), it will either try
(and likely fail) to insert itself into the stale pool, or it will drop the
cStack on the floor.
*/
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
	m.pool = make(chan *cStack, mPoolSize)
}

func (m *mStack) newStack() *cStack {
	m.lock.Lock()
	defer m.lock.Unlock()

	cs := cStack{}
	router := m.router

	cs.m = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.route(&cs.C, w, r)
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
	// architectures I care about it happens to be atomic.
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
	cs.C = C{}
	if cs.pool != m.pool {
		return
	}
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

	copy(m.stack[i:], m.stack[i+1:])
	m.stack = m.stack[:len(m.stack)-1 : len(m.stack)]

	m.invalidate()
	return nil
}
