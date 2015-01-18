package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenazn/goji/web"
)

func testOptions(r *http.Request, f func(*web.C, http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	var c web.C

	h := func(w http.ResponseWriter, r *http.Request) {
		f(&c, w, r)
	}
	m := AutomaticOptions(&c, http.HandlerFunc(h))
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	return w
}

var optionsTestEnv = map[interface{}]interface{}{
	web.ValidMethodsKey: []string{
		"hello",
		"world",
	},
}

func TestAutomaticOptions(t *testing.T) {
	t.Parallel()

	// Shouldn't interfere with normal requests
	r, _ := http.NewRequest("GET", "/", nil)
	rr := testOptions(r,
		func(c *web.C, w http.ResponseWriter, r *http.Request) {
			w.Write([]byte{'h', 'i'})
		},
	)
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.String() != "hi" {
		t.Errorf("body was %q, should be %q", rr.Body.String(), "hi")
	}
	allow := rr.HeaderMap.Get("Allow")
	if allow != "" {
		t.Errorf("Allow header was set to %q, should be empty", allow)
	}

	// If we respond non-404 to an OPTIONS request, also don't interfere
	r, _ = http.NewRequest("OPTIONS", "/", nil)
	rr = testOptions(r,
		func(c *web.C, w http.ResponseWriter, r *http.Request) {
			c.Env = optionsTestEnv
			w.Write([]byte{'h', 'i'})
		},
	)
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.String() != "hi" {
		t.Errorf("body was %q, should be %q", rr.Body.String(), "hi")
	}
	allow = rr.HeaderMap.Get("Allow")
	if allow != "" {
		t.Errorf("Allow header was set to %q, should be empty", allow)
	}

	// Provide options if we 404. Make sure we nom the output bytes
	r, _ = http.NewRequest("OPTIONS", "/", nil)
	rr = testOptions(r,
		func(c *web.C, w http.ResponseWriter, r *http.Request) {
			c.Env = optionsTestEnv
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte{'h', 'i'})
		},
	)
	if rr.Code != http.StatusOK {
		t.Errorf("status is %d, not 200", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("body was %q, should be empty", rr.Body.String())
	}
	allow = rr.HeaderMap.Get("Allow")
	correctHeaders := "hello, world, OPTIONS"
	if allow != "hello, world, OPTIONS" {
		t.Errorf("Allow header should be %q, was %q", correctHeaders,
			allow)
	}

	// If we somehow 404 without giving a list of valid options, don't do
	// anything
	r, _ = http.NewRequest("OPTIONS", "/", nil)
	rr = testOptions(r,
		func(c *web.C, w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte{'h', 'i'})
		},
	)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status is %d, not 404", rr.Code)
	}
	if rr.Body.String() != "hi" {
		t.Errorf("body was %q, should be %q", rr.Body.String(), "hi")
	}
	allow = rr.HeaderMap.Get("Allow")
	if allow != "" {
		t.Errorf("Allow header was set to %q, should be empty", allow)
	}
}
