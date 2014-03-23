package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/zenazn/goji/web"
)

// Automatically return an appropriate "Allow" header when the request method is
// OPTIONS and the request would have otherwise been 404'd.
func AutomaticOptions(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// This will probably slow down OPTIONS calls a bunch, but it
		// probably won't happen too much, and it'll just be hitting the
		// 404 route anyways.
		var fw *httptest.ResponseRecorder
		pw := w
		if r.Method == "OPTIONS" {
			fw = httptest.NewRecorder()
			pw = fw
		}

		h.ServeHTTP(pw, r)

		if fw == nil {
			return
		}

		for k, v := range fw.Header() {
			w.Header()[k] = v
		}

		methods := getValidMethods(*c)

		if fw.Code == http.StatusNotFound && methods != nil {
			methods = addMethod(methods, "OPTIONS")
			w.Header().Set("Allow", strings.Join(methods, ", "))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(fw.Code)
			io.Copy(w, fw.Body)
		}
	}

	return http.HandlerFunc(fn)
}

func getValidMethods(c web.C) []string {
	if c.Env == nil {
		return nil
	}
	v, ok := c.Env[web.ValidMethodsKey]
	if !ok {
		return nil
	}
	if methods, ok := v.([]string); ok {
		return methods
	} else {
		return nil
	}
}

// Assumption: the list of methods is teensy, and that anything we could
// possibly want to do here is going to be fast.
func addMethod(methods []string, method string) []string {
	for _, m := range methods {
		if m == method {
			return methods
		}
	}
	return append(methods, method)
}
