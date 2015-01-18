package middleware

import (
	"net/http"

	"github.com/zenazn/goji/web"
)

type envInit struct {
	c *web.C
	h http.Handler
}

func (e envInit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e.c.Env == nil {
		e.c.Env = make(map[interface{}]interface{})
	}
	e.h.ServeHTTP(w, r)
}

// EnvInit is a middleware that allocates an environment map if it is nil. While
// it's impossible in general to ensure that Env is never nil in a middleware
// stack, in most common cases placing this middleware at the top of the stack
// will eliminate the need for repetative nil checks.
func EnvInit(c *web.C, h http.Handler) http.Handler {
	return envInit{c, h}
}
