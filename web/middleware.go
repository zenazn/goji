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
	name string
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
	m http.Handler
}

func (s *cStack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.C = C{}
	s.m.ServeHTTP(w, r)
}
func (s *cStack) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	s.C = c
	s.m.ServeHTTP(w, r)
}

func (m *mStack) appendLayer(name string, fn interface{}) {
	var ml mLayer
	ml.name = name
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

func (m *mStack) findLayer(name string) int {
	for i, middleware := range m.stack {
		if middleware.name == name {
			return i
		}
	}
	return -1
}

func (m *mStack) invalidate() {
	old := m.pool
	m.pool = make(chan *cStack, mPoolSize)
	close(old)
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
	select {
	case cs := <-m.pool:
		if cs == nil {
			return m.alloc()
		}
		return cs
	default:
		return m.newStack()
	}
}

func (m *mStack) release(cs *cStack) {
	// It's possible that the pool has been invalidated and therefore
	// closed, in which case we'll start panicing, which is dumb. I'm not
	// sure this is actually better than just grabbing a lock, but whatever.
	defer func() {
		recover()
	}()
	select {
	case m.pool <- cs:
	default:
	}
}

// Append the given middleware to the middleware stack. See the documentation
// for type Mux for a list of valid middleware types.
//
// No attempt is made to enforce the uniqueness of middleware names.
func (m *mStack) Use(name string, middleware interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.appendLayer(name, middleware)
	m.invalidate()
}

// Insert the given middleware immediately before a given existing middleware in
// the stack. See the documentation for type Mux for a list of valid middleware
// types. Returns an error if no middleware has the name given by "before."
//
// No attempt is made to enforce the uniqueness of names. If the insertion point
// is ambiguous, the first (outermost) one is chosen.
func (m *mStack) Insert(name string, middleware interface{}, before string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	i := m.findLayer(before)
	if i < 0 {
		return fmt.Errorf("web: unknown middleware %v", before)
	}

	m.appendLayer(name, middleware)
	inserted := m.stack[len(m.stack)-1]
	copy(m.stack[i+1:], m.stack[i:])
	m.stack[i] = inserted

	m.invalidate()
	return nil
}

// Remove the given named middleware from the middleware stack. Returns an error
// if there is no middleware by that name.
//
// If the name of the middleware to delete is ambiguous, the first (outermost)
// one is chosen.
func (m *mStack) Abandon(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	i := m.findLayer(name)
	if i < 0 {
		return fmt.Errorf("web: unknown middleware %v", name)
	}

	copy(m.stack[i:], m.stack[i+1:])
	m.stack = m.stack[:len(m.stack)-1 : len(m.stack)]

	m.invalidate()
	return nil
}

// Returns a list of middleware currently in use.
func (m *mStack) Middleware() []string {
	m.lock.Lock()
	defer m.lock.Unlock()
	stack := make([]string, len(m.stack))
	for i, ml := range m.stack {
		stack[i] = ml.name
	}
	return stack
}
