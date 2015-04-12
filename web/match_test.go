package web

import (
	"net/http"
	"regexp"
	"testing"
)

var rawPatterns = []PatternType{
	"/hello/:name",
	regexp.MustCompile("^/hello/(?P<name>[^/]+)$"),
	testPattern{},
}

func TestRawPattern(t *testing.T) {
	t.Parallel()

	for _, p := range rawPatterns {
		m := Match{Pattern: ParsePattern(p)}
		if rp := m.RawPattern(); rp != p {
			t.Errorf("got %#v, expected %#v", rp, p)
		}
	}
}

type httpHandlerOnly struct{}

func (httpHandlerOnly) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

type handlerOnly struct{}

func (handlerOnly) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {}

var rawHandlers = []HandlerType{
	func(w http.ResponseWriter, r *http.Request) {},
	func(c C, w http.ResponseWriter, r *http.Request) {},
	httpHandlerOnly{},
	handlerOnly{},
}

func TestRawHandler(t *testing.T) {
	t.Parallel()

	for _, h := range rawHandlers {
		m := Match{Handler: parseHandler(h)}
		if rh := m.RawHandler(); !funcEqual(rh, h) {
			t.Errorf("got %#v, expected %#v", rh, h)
		}
	}
}
