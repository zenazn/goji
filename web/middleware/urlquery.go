package middleware

import (
	"github.com/zenazn/goji/web"
	"net/http"
)

// UrlQueryKey is the context key for the URL Query
const UrlQueryKey string = "urlquery"

// UrlQuery is a middleware to parse the URL Query parameters just once,
// and store the resulting url.Values in the context.
func UrlQuery(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if c.Env == nil {
			c.Env = make(map[interface{}]interface{})
		}
		c.Env[UrlQueryKey] = r.URL.Query()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
