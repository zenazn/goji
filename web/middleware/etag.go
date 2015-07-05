package middleware

import (
	"crypto/md5"
	"net/http"
	"fmt"
)

type etagResponseWriter struct {
	status int
	w http.ResponseWriter
	r *http.Request
}

func (e *etagResponseWriter) Header() http.Header {
	return e.w.Header()
}

func (e *etagResponseWriter) WriteHeader(s int) {
	e.w.WriteHeader(s)
	e.status = s
}

func (e *etagResponseWriter) Write(b []byte) (int, error) {
	eTag := fmt.Sprintf("%x", md5.Sum(b))
	
	if 200 <= e.status && e.status < 300 {
		if e.r.Header.Get("If-None-Match") == eTag {
			e.w.WriteHeader(http.StatusNotModified)
			b = []byte{}
		} else {
			e.w.Header()["ETag"] = []string{eTag}
		}
	}

	size, err := e.w.Write(b)
	return size, err
}

// ETag is a middleware that calculates an ETag for each request.
// If the ETag matches with the "If-None-Match" header, it will return 
// a HTTP 304 Not Modified status.
func ETag(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&etagResponseWriter{http.StatusOK, w, r}, r)
	}

	return http.HandlerFunc(fn)
}
