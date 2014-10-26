package web

import (
	"net/http"
)

/*
Mux is an HTTP multiplexer, much like net/http's ServeMux.

Routes may be added using any of the various HTTP-method-specific functions.
When processing a request, when iterating in insertion order the first route
that matches both the request's path and method is used.

There are two other differences worth mentioning between web.Mux and
http.ServeMux. First, string patterns (i.e., Sinatra-like patterns) must match
exactly: the "rooted subtree" behavior of ServeMux is not implemented. Secondly,
unlike ServeMux, Mux does not support Host-specific patterns.

If you require any of these features, remember that you are free to mix and
match muxes at any part of the stack.

In order to provide a sane API, many functions on Mux take interface{}'s. This
is obviously not a very satisfying solution, but it's probably the best we can
do for now. Instead of duplicating documentation on each method, the types
accepted by those functions are documented here.

A middleware (the untyped parameter in Use() and Insert()) must be one of the
following types:
	- func(http.Handler) http.Handler
	- func(c *web.C, http.Handler) http.Handler
All of the route-adding functions on Mux take two untyped parameters: pattern
and handler. Pattern must be one of the following types:
	- string. It will be interpreted as a Sinatra-like pattern. In
	  particular, the following syntax is recognized:
		- a path segment starting with with a colon will match any
		  string placed at that position. e.g., "/:name" will match
		  "/carl", binding "name" to "carl".
		- a pattern ending with an asterisk will match any prefix of
		  that route. For instance, "/admin/*" will match "/admin/" and
		  "/admin/secret/lair". This is similar to Sinatra's wildcard,
		  but may only appear at the very end of the string and is
		  therefore significantly less powerful.
	- regexp.Regexp. The library assumes that it is a Perl-style regexp that
	  is anchored on the left (i.e., the beginning of the string). If your
	  regexp is not anchored on the left, a hopefully-identical
	  left-anchored regexp will be created and used instead.
	- web.Pattern
Handler must be one of the following types:
	- http.Handler
	- web.Handler
	- func(w http.ResponseWriter, r *http.Request)
	- func(c web.C, w http.ResponseWriter, r *http.Request)
*/
type Mux struct {
	ms mStack
	router
}

// New creates a new Mux without any routes or middleware.
func New() *Mux {
	mux := Mux{
		ms: mStack{
			stack: make([]mLayer, 0),
			pool:  makeCPool(),
		},
		router: router{
			routes:   make([]route, 0),
			notFound: parseHandler(http.NotFound),
		},
	}
	mux.ms.router = &mux.router
	return &mux
}

// ServeHTTP processes HTTP requests. It make Muxes satisfy net/http.Handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stack := m.ms.alloc()
	stack.ServeHTTP(w, r)
	m.ms.release(stack)
}

// ServeHTTPC creates a context dependent request with the given Mux. Satisfies
// the web.Handler interface.
func (m *Mux) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	stack := m.ms.alloc()
	stack.ServeHTTPC(c, w, r)
	m.ms.release(stack)
}

// Middleware Stack functions

// Append the given middleware to the middleware stack. See the documentation
// for type Mux for a list of valid middleware types.
//
// No attempt is made to enforce the uniqueness of middlewares. It is illegal to
// call this function concurrently with active requests.
func (m *Mux) Use(middleware interface{}) {
	m.ms.Use(middleware)
}

// Insert the given middleware immediately before a given existing middleware in
// the stack. See the documentation for type Mux for a list of valid middleware
// types. Returns an error if no middleware has the name given by "before."
//
// No attempt is made to enforce the uniqueness of middlewares. If the insertion
// point is ambiguous, the first (outermost) one is chosen. It is illegal to
// call this function concurrently with active requests.
func (m *Mux) Insert(middleware, before interface{}) error {
	return m.ms.Insert(middleware, before)
}

// Remove the given middleware from the middleware stack. Returns an error if
// no such middleware can be found.
//
// If the name of the middleware to delete is ambiguous, the first (outermost)
// one is chosen. It is illegal to call this function concurrently with active
// requests.
func (m *Mux) Abandon(middleware interface{}) error {
	return m.ms.Abandon(middleware)
}
