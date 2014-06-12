package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenazn/goji/web"
)

func TestNoCache(t *testing.T) {

	rr := httptest.NewRecorder()
	s := web.New()

	s.Use(NoCache)
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	s.ServeHTTP(rr, r)

	for k, v := range noCacheHeaders {
		if rr.HeaderMap[k][0] != v {
			t.Errorf("%s header not set by middleware.", k)
		}
	}
}
