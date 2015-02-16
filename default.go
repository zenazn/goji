package goji

import (
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

// The default web.Mux.
var DefaultMux *web.Mux

func init() {
	DefaultMux = web.New()

	DefaultMux.Use(middleware.RequestID)
	DefaultMux.Use(middleware.Logger)
	DefaultMux.Use(middleware.Recoverer)
	DefaultMux.Use(middleware.AutomaticOptions)
}

// Use appends the given middleware to the default Mux's middleware stack. See
// the documentation for web.Mux.Use for more information.
func Use(middleware web.MiddlewareType) {
	DefaultMux.Use(middleware)
}

// Insert the given middleware into the default Mux's middleware stack. See the
// documentation for web.Mux.Insert for more information.
func Insert(middleware, before web.MiddlewareType) error {
	return DefaultMux.Insert(middleware, before)
}

// Abandon removes the given middleware from the default Mux's middleware stack.
// See the documentation for web.Mux.Abandon for more information.
func Abandon(middleware web.MiddlewareType) error {
	return DefaultMux.Abandon(middleware)
}

// Handle adds a route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Handle(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Handle(pattern, handler)
}

// Connect adds a CONNECT route to the default Mux. See the documentation for
// web.Mux for more information about what types this function accepts.
func Connect(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Connect(pattern, handler)
}

// Delete adds a DELETE route to the default Mux. See the documentation for
// web.Mux for more information about what types this function accepts.
func Delete(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Delete(pattern, handler)
}

// Get adds a GET route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Get(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Get(pattern, handler)
}

// Head adds a HEAD route to the default Mux. See the documentation for web.Mux
// for more information about what types this function accepts.
func Head(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Head(pattern, handler)
}

// Options adds a OPTIONS route to the default Mux. See the documentation for
// web.Mux for more information about what types this function accepts.
func Options(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Options(pattern, handler)
}

// Patch adds a PATCH route to the default Mux. See the documentation for web.Mux
// for more information about what types this function accepts.
func Patch(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Patch(pattern, handler)
}

// Post adds a POST route to the default Mux. See the documentation for web.Mux
// for more information about what types this function accepts.
func Post(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Post(pattern, handler)
}

// Put adds a PUT route to the default Mux. See the documentation for web.Mux for
// more information about what types this function accepts.
func Put(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Put(pattern, handler)
}

// Trace adds a TRACE route to the default Mux. See the documentation for
// web.Mux for more information about what types this function accepts.
func Trace(pattern web.PatternType, handler web.HandlerType) {
	DefaultMux.Trace(pattern, handler)
}

// NotFound sets the NotFound handler for the default Mux. See the documentation
// for web.Mux.NotFound for more information.
func NotFound(handler web.HandlerType) {
	DefaultMux.NotFound(handler)
}
