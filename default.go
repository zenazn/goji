package goji

import (
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

// The default web.Mux.
var DefaultMux *web.Mux

func init() {
	DefaultMux = web.New()

	DefaultMux.Use(middleware.RequestId)
	DefaultMux.Use(middleware.Logger)
	DefaultMux.Use(middleware.Recoverer)
	DefaultMux.Use(middleware.AutomaticOptions)
}

// Append the given middleware to the default Mux's middleware stack. See the
// documentation for web.Mux.Use for more information.
func Use(middleware interface{}) {
	DefaultMux.Use(middleware)
}

// Insert the given middleware into the default Mux's middleware stack. See the
// documentation for web.Mux.Insert for more information.
func Insert(middleware, before interface{}) error {
	return DefaultMux.Insert(middleware, before)
}

// Remove the given middleware from the default Mux's middleware stack. See the
// documentation for web.Mux.Abandon for more information.
func Abandon(middleware interface{}) error {
	return DefaultMux.Abandon(middleware)
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

// Set the NotFound handler for the default Mux. See the documentation for
// web.Mux.NotFound for more information.
func NotFound(handler interface{}) {
	DefaultMux.NotFound(handler)
}
