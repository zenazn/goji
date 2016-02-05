package middleware

import (
	"github.com/zenazn/goji/web"
	"net/http"
)

// URLQueryKey is the context key for the URL Query
const URLQueryKey string = "urlquery"

// URLQuery is a middleware to parse the URL Query parameters just once,
// and store the resulting url.Values in the context.
func URLQuery(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if c.Env == nil {
			c.Env = make(map[interface{}]interface{})
		}
		c.Env[URLQueryKey] = r.URL.Query()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
