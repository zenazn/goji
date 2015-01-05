// +build !go1.3

package graceful

import (
	"net/http"
	"sync/atomic"
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
	m := middleware(h)
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
	atomic.StoreInt32(&closing, 0)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})
	testClose(t, h, false)
}

func TestClose(t *testing.T) {
	atomic.StoreInt32(&closing, 0)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt32(&closing, 1)
	})
	testClose(t, h, true)
}

func TestCloseWriteHeader(t *testing.T) {
	atomic.StoreInt32(&closing, 0)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt32(&closing, 1)
		w.WriteHeader(200)
	})
	testClose(t, h, true)
}

func TestCloseWrite(t *testing.T) {
	atomic.StoreInt32(&closing, 0)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt32(&closing, 1)
		w.Write([]byte{})
	})
	testClose(t, h, true)
}
