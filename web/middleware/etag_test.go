package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"io"
	"github.com/zenazn/goji/web"
)

func TestETag(t *testing.T) {
	s := web.New()
	s.Use(ETag)
	
	s.Get("/", func (w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, World!")
	})

	r, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Status code is %d, not 200", rr.Code)
	}
	
	var eTag string
	if value, ok := rr.Header()["ETag"]; ok {
		eTag = value[0]
	} else {
		t.Error("ETag header should be set by middleware.")
	}
	
	r, _ = http.NewRequest("GET", "/", nil)
	r.Header.Set("If-None-Match", eTag)
	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, r)
	
	if rr.Code != http.StatusNotModified {
		t.Error("Status code should be 304.")
	}
}
