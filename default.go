package goji

import (
	"github.com/zenazn/goji/web"
)

// The default web.Mux.
var DefaultMux *web.Mux

func init() {
	DefaultMux = web.New()
}

// Append the given middleware to the default Mux's middleware stack. See the
// documentation for web.Mux.Use for more informatino.
func Use(name string, middleware interface{}) {
	DefaultMux.Use(name, middleware)
}

// Insert the given middleware into the default Mux's middleware stack. See the
// documentation for web.Mux.Insert for more informatino.
func Insert(name string, middleware interface{}, before string) error {
	return DefaultMux.Insert(name, middleware, before)
}

// Remove the given middleware from the default Mux's middleware stack. See the
// documentation for web.Mux.Abandon for more informatino.
func Abandon(name string) error {
	return DefaultMux.Abandon(name)
}

// Retrieve the list of middleware from the default Mux's middleware stack. See
// the documentation for web.Mux.Middleware() for more information.
func Middleware() []string {
	return DefaultMux.Middleware()
}

// Add a route to the default Mux. See the documentation for web.Mux for more
// information about what types this function accepts.
func Handle(pattern interface{}, handler interface{}) {
	DefaultMux.Handle(pattern, handler)
}

// Add a CONNECT route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Connect(pattern interface{}, handler interface{}) {
	DefaultMux.Connect(pattern, handler)
}

// Add a DELETE route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Delete(pattern interface{}, handler interface{}) {
	DefaultMux.Delete(pattern, handler)
}

// Add a GET route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Get(pattern interface{}, handler interface{}) {
	DefaultMux.Get(pattern, handler)
}

// Add a HEAD route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Head(pattern interface{}, handler interface{}) {
	DefaultMux.Head(pattern, handler)
}

// Add a OPTIONS route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Options(pattern interface{}, handler interface{}) {
	DefaultMux.Options(pattern, handler)
}

// Add a PATCH route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Patch(pattern interface{}, handler interface{}) {
	DefaultMux.Patch(pattern, handler)
}

// Add a POST route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Post(pattern interface{}, handler interface{}) {
	DefaultMux.Post(pattern, handler)
}

// Add a PUT route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Put(pattern interface{}, handler interface{}) {
	DefaultMux.Put(pattern, handler)
}

// Add a TRACE route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Trace(pattern interface{}, handler interface{}) {
	DefaultMux.Trace(pattern, handler)
}

// Add a sub-route to the default Mux. See the documentation for web.Mux.Sub
// for more information.
func Sub(pattern string, handler interface{}) {
	DefaultMux.Sub(pattern, handler)
}

// Set the NotFound handler for the default Mux. See the documentation for
// web.Mux.NotFound for more information.
func NotFound(handler interface{}) {
	DefaultMux.NotFound(handler)
}
