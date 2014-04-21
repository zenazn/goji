package middleware

import (
	"net/http"
	"strings"

	"github.com/zenazn/goji/web"
)

type autoOptionsState int

const (
	aosInit autoOptionsState = iota
	aosHeaderWritten
	aosProxying
)

// I originally used an httptest.ResponseRecorder here, but package httptest
// adds a flag which I'm not particularly eager to expose. This is essentially a
// ResponseRecorder that has been specialized for the purpose at hand to avoid
// the httptest dependency.
type autoOptionsProxy struct {
	w     http.ResponseWriter
	c     *web.C
	state autoOptionsState
}

func (p *autoOptionsProxy) Header() http.Header {
	return p.w.Header()
}

func (p *autoOptionsProxy) Write(buf []byte) (int, error) {
	switch p.state {
	case aosInit:
		p.state = aosHeaderWritten
	case aosProxying:
		return len(buf), nil
	}
	return p.w.Write(buf)
}

func (p *autoOptionsProxy) WriteHeader(code int) {
	methods := getValidMethods(*p.c)
	switch p.state {
	case aosInit:
		if methods != nil && code == http.StatusNotFound {
			p.state = aosProxying
			break
		}
		p.state = aosHeaderWritten
		fallthrough
	default:
		p.w.WriteHeader(code)
		return
	}

	methods = addMethod(methods, "OPTIONS")
	p.w.Header().Set("Allow", strings.Join(methods, ", "))
	p.w.WriteHeader(http.StatusOK)
}

// AutomaticOptions automatically return an appropriate "Allow" header when the
// request method is OPTIONS and the request would have otherwise been 404'd.
func AutomaticOptions(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w = &autoOptionsProxy{c: c, w: w}
		}

		h.ServeHTTP(w, r)
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
	}
	return nil
}

func addMethod(methods []string, method string) []string {
	for _, m := range methods {
		if m == method {
			return methods
		}
	}
	return append(methods, method)
}
