package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterMiddleware(t *testing.T) {
	t.Parallel()

	m := New()
	ch := make(chan string, 1)
	m.Get("/a", chHandler(ch, "a"))
	m.Get("/b", chHandler(ch, "b"))
	m.Use(m.Router)
	m.Use(func(c *C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			m := GetMatch(*c)
			if rp := m.RawPattern(); rp != "/a" {
				t.Fatalf("RawPattern was not /a: %v", rp)
			}
			r.URL.Path = "/b"
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})

	r, _ := http.NewRequest("GET", "/a", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if v := <-ch; v != "a" {
		t.Errorf("Routing was not frozen! %s", v)
	}
}
