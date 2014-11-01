package middleware

import (
	"net/http"

	"github.com/zenazn/goji/web"
)

type subrouter struct {
	c *web.C
	h http.Handler
}

func (s subrouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.c.URLParams != nil {
		if path, ok := s.c.URLParams["*"]; ok {
			oldpath := r.URL.Path
			r.URL.Path = path
			defer func() {
				r.URL.Path = oldpath
			}()
		}
	}
	s.h.ServeHTTP(w, r)
}

// SubRouter is a helper middleware that makes writing sub-routers easier.
//
// If you register a sub-router under a key like "/admin/*", Goji's router will
// automatically set c.URLParams["*"] to the unmatched path suffix. This
// middleware will help you set the request URL's Path to this unmatched suffix,
// allowing you to write sub-routers with no knowledge of what routes the parent
// router matches.
func SubRouter(c *web.C, h http.Handler) http.Handler {
	return subrouter{c, h}
}
