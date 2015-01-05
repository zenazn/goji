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
		path, ok := s.c.URLParams["*"]
		if !ok {
			path, ok = s.c.URLParams["_"]
		}
		if ok {
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
//
// Since Go's regular expressions do not allow you to create a capturing group
// named "*", SubRouter also accepts the string "_". For instance, to duplicate
// the semantics of the string pattern "/foo/*", you might use the regular
// expression "^/foo(?P<_>/.*)$".
func SubRouter(c *web.C, h http.Handler) http.Handler {
	return subrouter{c, h}
}
