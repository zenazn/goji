// +build appengine

package goji

import "net/http"

// Serve starts Goji using reasonable defaults.
func Serve() {
	DefaultMux.Compile()
	// Install our handler at the root of the standard net/http default mux.
	// This is required for App Engine, and also allows packages like expvar
	// to continue working as expected.
	http.Handle("/", DefaultMux)
}
