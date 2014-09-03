package middleware

import (
	"net/http"
	"strings"
	"code.google.com/p/go.net/context"

	"github.com/zenazn/goji/web"
)

// AutomaticOptions is a NotFound handler that automatically returns an
// appropriate "Allow" header when the request method is OPTIONS and the
// request would have otherwise been 404'd.
func AutomaticOptions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != "OPTIONS" {
		http.NotFound(w, r)
		return
	}
	methods := addMethod(web.ValidMethods(ctx), "OPTIONS")
	w.Header().Set("Allow", strings.Join(methods, ", "))
	w.WriteHeader(http.StatusOK)
}

func addMethod(methods []string, method string) []string {
	for _, m := range methods {
		if m == method {
			return methods
		}
	}
	return append(methods, method)
}
