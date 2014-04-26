package middleware

import (
	"net/http"

	"github.com/zenazn/goji/web"
)

// EnvInit is a middleware that allocates an environment map if one does not
// already exist. This is necessary because Goji does not guarantee that Env is
// present when running middleware (it avoids forcing the map allocation). Note
// that other middleware should check Env for nil in order to maximize
// compatibility (when EnvInit is not used, or when another middleware layer
// blanks out Env), but for situations in which the user controls the middleware
// stack and knows EnvInit is present, this middleware can eliminate a lot of
// boilerplate.
func EnvInit(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if c.Env == nil {
			c.Env = make(map[string]interface{})
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
