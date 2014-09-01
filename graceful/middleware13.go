// +build go1.3

package graceful

import "net/http"

// Middleware is a stub that does nothing. When used with versions of Go before
// Go 1.3, it provides functionality similar to net/http.Server's
// SetKeepAlivesEnabled.
func Middleware(h http.Handler) http.Handler {
	return h
}
