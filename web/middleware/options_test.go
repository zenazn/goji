package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"code.google.com/p/go.net/context"

	"github.com/zenazn/goji/web"
)

func testOptions(m http.Handler, method string, path string) *httptest.ResponseRecorder {
	r, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	return w
}

func TestAutomaticOptions(t *testing.T) {
	m := web.New()
	m.NotFound(AutomaticOptions)
	m.Get("/path/1", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("get1"))
	})
	m.Post("/path/1", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("post1"))
	})
	m.Options("/path/1", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("options1"))
	})
	m.Get("/path/2", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("get2"))
	})
	m.Post("/path/2", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("post2"))
	})
	m.Put("/path/*", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("patch2"))
	})

	// Shouldn't interfere with normal requests
	rr := testOptions(m, "GET", "/path/1")
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.String() != "get1" {
		t.Errorf("body was %q, should be %q", rr.Body.String(), "get1")
	}
	allow := rr.HeaderMap.Get("Allow")
	if allow != "" {
		t.Errorf("Allow header was set to %q, should be empty", allow)
	}

	// If we respond to an OPTIONS request, check that we don't interfere
	rr = testOptions(m, "OPTIONS", "/path/1")
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.String() != "options1" {
		t.Errorf("body was %q, should be %q", rr.Body.String(), "options1")
	}
	allow = rr.HeaderMap.Get("Allow")
	if allow != "" {
		t.Errorf("Allow header was set to %q, should be empty", allow)
	}

	// Provide options if we 404. Make sure we have all the registered routes
	rr = testOptions(m, "OPTIONS", "/path/2")
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("body was %q, should be empty", rr.Body.String())
	}
	allow = rr.HeaderMap.Get("Allow")
	correctHeaders := "GET, HEAD, POST, PUT, OPTIONS"
	if allow != correctHeaders {
		t.Errorf("Allow header should be %q, was %q", correctHeaders,
			allow)
	}

}
