package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenazn/goji/web"
)

func TestSubRouterMatch(t *testing.T) {
	m := web.New()
	m.Use(m.Router)

	m2 := web.New()
	m2.Use(SubRouter)
	m2.Get("/bar", func(w http.ResponseWriter, r *http.Request) {})

	m.Get("/foo/*", m2)

	r, err := http.NewRequest("GET", "/foo/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	// This function will recurse forever if SubRouter + Match didn't work.
	m.ServeHTTP(httptest.NewRecorder(), r)
}
