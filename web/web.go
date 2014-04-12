/*
Package web is a microframework inspired by Sinatra.

The underlying philosophy behind this package is that net/http is a very good
HTTP library which is only missing a few features. If you disagree with this
statement (e.g., you think that the interfaces it exposes are not especially
good, or if you're looking for a comprehensive "batteries included" feature
list), you're likely not going to have a good time using this library. In that
spirit, we have attempted wherever possible to be compatible with net/http. You
should be able to insert any net/http compliant handler into this library, or
use this library with any other net/http compliant mux.

This package attempts to solve three problems that net/http does not. First, it
allows you to specify URL patterns with Sinatra-like named wildcards and
regexps. Second, it allows you to write reconfigurable middleware stacks. And
finally, it allows you to attach additional context to requests, in a manner
that can be manipulated by both compliant middleware and handlers.

A usage example:

	m := web.New()

Use your favorite HTTP verbs:

	var legacyFooHttpHandler http.Handler // From elsewhere
	m.Get("/foo", legacyFooHttpHandler)
	m.Post("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.Write("Hello world!")
	})

Bind parameters using either Sinatra-like patterns or regular expressions:

	m.Get("/hello/:name", func(c web.C, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
	})
	pattern := regexp.MustCompile(`^/ip/(?P<ip>(?:\d{1,3}\.){3}\d{1,3})$`)
	m.Get(pattern, func(c web.C, w http.ResponseWriter, r *http.Request) {
		fmt.Printf(w, "Info for IP address %s:", c.URLParams["ip"])
	})

Middleware are functions that wrap http.Handlers, just like you'd use with raw
net/http. Middleware functions can optionally take a context parameter, which
will be threaded throughout the middleware stack and to the final handler, even
if not all of these things do not support contexts. Middleware are encouraged to
use the Env parameter to pass data to other middleware and to the final handler:

	m.Use(func(h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			log.Println("Before request")
			h.ServeHTTP(w, r)
			log.Println("After request")
		}
		return http.HandlerFunc(handler)
	})
	m.Use(func(c *web.C, h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("user")
			if err == nil {
				c.Env["user"] = cookie.Raw
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(handler)
	})

	m.Get("/baz", func(c web.C, w http.ResponseWriter, r *http.Request) {
		if user, ok := c.Env["user"], ok {
			w.Write("Hello " + string(user))
		} else {
			w.Write("Hello Stranger!")
		}
	})
*/
package web

import (
	"net/http"
)

/*
C is a per-request context object which is threaded through all compliant middleware
layers and to the final request handler.

As an implementation detail, references to these structs are reused between
requests to reduce allocation churn, but the maps they contain are created fresh
on every request. If you are closing over a context (especially relevant for
middleware), you should not close over either the URLParams or Env objects,
instead accessing them through the context whenever they are required.
*/
type C struct {
	// The parameters parsed by the mux from the URL itself. In most cases,
	// will contain a map from programmer-specified identifiers to the
	// strings that matched those identifiers, but if a unnamed regex
	// capture is used, it will be assigned to the special identifiers "$1",
	// "$2", etc.
	URLParams map[string]string
	// A free-form environment, similar to Rack or PEP 333's environments.
	// Middleware layers are encouraged to pass data to downstream layers
	// and other handlers using this map, and are even more strongly
	// encouraged to document and maybe namespace they keys they use.
	Env map[string]interface{}
}

// Handler is a superset of net/http's http.Handler, which also includes a
// mechanism for serving requests with a context. If your handler does not
// support the use of contexts, we encourage you to use http.Handler instead.
type Handler interface {
	http.Handler
	ServeHTTPC(C, http.ResponseWriter, *http.Request)
}

// HandlerFunc is like net/http's http.HandlerFunc, but supports a context
// object. Implements both http.Handler and web.Handler free of charge.
type HandlerFunc func(C, http.ResponseWriter, *http.Request)

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(C{}, w, r)
}

// ServeHTTPC wraps ServeHTTP with a context parameter.
func (h HandlerFunc) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	h(c, w, r)
}
