/*
Package web implements a fast and flexible middleware stack and mux.

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
		w.Write([]byte("Hello world!"))
	})

Bind parameters using either Sinatra-like patterns or regular expressions:

	m.Get("/hello/:name", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
	})
	pattern := regexp.MustCompile(`^/ip/(?P<ip>(?:\d{1,3}\.){3}\d{1,3})$`)
	m.Get(pattern, func(c context.Context, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Info for IP address %s:", c.URLParams["ip"])
	})

Middleware are functions that wrap http.Handlers, just like you'd use with raw
net/http. Middleware functions can optionally take a context parameter, which
will be threaded throughout the middleware stack and to the final handler, even
if not all of these things support contexts. Middleware are encouraged to use
the Env parameter to pass data to other middleware and to the final handler:

	m.Use(func(h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			log.Println("Before request")
			h.ServeHTTP(w, r)
			log.Println("After request")
		}
		return http.HandlerFunc(handler)
	})
	m.Use(func(c context.Context, h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("user")
			if err == nil {
				// Consider using the middleware EnvInit instead
				// of repeating the below check
				if c.Env == nil {
					c.Env = make(map[string]interface{})
				}
				c.Env["user"] = cookie.Value
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(handler)
	})

	m.Get("/baz", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		if user, ok := c.Env["user"].(string); ok {
			w.Write([]byte("Hello " + user))
		} else {
			w.Write([]byte("Hello Stranger!"))
		}
	})
*/
package web

import (
	"net/http"

	"code.google.com/p/go.net/context"
)

// Handler is like net/http's http.Handler, but also includes a
// mechanism for serving requests with a context. If your handler does not
// support the use of contexts, we encourage you to use http.Handler instead.
type Handler interface {
	ServeHTTPC(context.Context, http.ResponseWriter, *http.Request)
}

// HandlerFunc is like net/http's http.HandlerFunc, but supports a context
// object.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// ServeHTTPC wraps ServeHTTP with a context parameter.
func (h HandlerFunc) ServeHTTPC(c context.Context, w http.ResponseWriter, r *http.Request) {
	h(c, w, r)
}
