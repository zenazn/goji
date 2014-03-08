package graceful

import (
	"net/http"
	"testing"
)

type fakeWriter http.Header

func (f fakeWriter) Header() http.Header {
	return http.Header(f)
}
func (f fakeWriter) Write(buf []byte) (int, error) {
	return len(buf), nil
}
func (f fakeWriter) WriteHeader(status int) {}

func testClose(t *testing.T, h http.Handler, expectClose bool) {
	m := Middleware(h)
	r, _ := http.NewRequest("GET", "/", nil)
	w := make(fakeWriter)
	m.ServeHTTP(w, r)

	c, ok := w["Connection"]
	if expectClose {
		if !ok || len(c) != 1 || c[0] != "close" {
			t.Fatal("Expected 'Connection: close'")
		}
	} else {
		if ok {
			t.Fatal("Did not expect Connection header")
		}
	}
}

func TestNormal(t *testing.T) {
	kill = make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})
	testClose(t, h, false)
}

func TestClose(t *testing.T) {
	kill = make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(kill)
	})
	testClose(t, h, true)
}

func TestCloseWriteHeader(t *testing.T) {
	kill = make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(kill)
		w.WriteHeader(200)
	})
	testClose(t, h, true)
}

func TestCloseWrite(t *testing.T) {
	kill = make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(kill)
		w.Write([]byte{})
	})
	testClose(t, h, true)
}
