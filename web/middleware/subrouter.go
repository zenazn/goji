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
			oldmatch := web.GetMatch(*s.c)
			r.URL.Path = path
			if oldmatch.Handler != nil {
				delete(s.c.Env, web.MatchKey)
			}

			defer func() {
				r.URL.Path = oldpath

				if s.c.Env == nil {
					return
				}
				if oldmatch.Handler != nil {
					s.c.Env[web.MatchKey] = oldmatch
				} else {
					delete(s.c.Env, web.MatchKey)
				}
			}()
		}
	}
	s.h.ServeHTTP(w, r)
}

/*
SubRouter is a helper middleware that makes writing sub-routers easier.

If you register a sub-router under a key like "/admin/*", Goji's router will
automatically set c.URLParams["*"] to the unmatched path suffix. This middleware
will help you set the request URL's Path to this unmatched suffix, allowing you
to write sub-routers with no knowledge of what routes the parent router matches.

Since Go's regular expressions do not allow you to create a capturing group
named "*", SubRouter also accepts the string "_". For instance, to duplicate the
semantics of the string pattern "/foo/*", you might use the regular expression
"^/foo(?P<_>/.*)$".

This middleware is Match-aware: it will un-set any explicit routing information
contained in the Goji context in order to prevent routing loops when using
explicit routing with sub-routers. See the documentation for Mux.Router for
more.
*/
func SubRouter(c *web.C, h http.Handler) http.Handler {
	return subrouter{c, h}
}
