// +build !go1.3

package graceful

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/zenazn/goji/graceful/listener"
)

// Middleware provides functionality similar to net/http.Server's
// SetKeepAlivesEnabled in Go 1.3, but in Go 1.2.
func middleware(h http.Handler) http.Handler {
	if h == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, cn := w.(http.CloseNotifier)
		_, fl := w.(http.Flusher)
		_, hj := w.(http.Hijacker)
		_, rf := w.(io.ReaderFrom)

		bw := basicWriter{ResponseWriter: w}

		if cn && fl && hj && rf {
			h.ServeHTTP(&fancyWriter{bw}, r)
		} else {
			h.ServeHTTP(&bw, r)
		}
		if !bw.headerWritten {
			bw.maybeClose()
		}
	})
}

type basicWriter struct {
	http.ResponseWriter
	headerWritten bool
}

func (b *basicWriter) maybeClose() {
	b.headerWritten = true
	if atomic.LoadInt32(&closing) != 0 {
		b.ResponseWriter.Header().Set("Connection", "close")
	}
}

func (b *basicWriter) WriteHeader(code int) {
	b.maybeClose()
	b.ResponseWriter.WriteHeader(code)
}

func (b *basicWriter) Write(buf []byte) (int, error) {
	if !b.headerWritten {
		b.maybeClose()
	}
	return b.ResponseWriter.Write(buf)
}

func (b *basicWriter) Unwrap() http.ResponseWriter {
	return b.ResponseWriter
}

// Optimize for the common case of a ResponseWriter that supports all three of
// CloseNotifier, Flusher, and Hijacker.
type fancyWriter struct {
	basicWriter
}

func (f *fancyWriter) CloseNotify() <-chan bool {
	cn := f.basicWriter.ResponseWriter.(http.CloseNotifier)
	return cn.CloseNotify()
}
func (f *fancyWriter) Flush() {
	fl := f.basicWriter.ResponseWriter.(http.Flusher)
	fl.Flush()
}
func (f *fancyWriter) Hijack() (c net.Conn, b *bufio.ReadWriter, e error) {
	hj := f.basicWriter.ResponseWriter.(http.Hijacker)
	c, b, e = hj.Hijack()

	if e == nil {
		e = listener.Disown(c)
	}

	return
}
func (f *fancyWriter) ReadFrom(r io.Reader) (int64, error) {
	rf := f.basicWriter.ResponseWriter.(io.ReaderFrom)
	if !f.basicWriter.headerWritten {
		f.basicWriter.maybeClose()
	}
	return rf.ReadFrom(r)
}

var _ http.CloseNotifier = &fancyWriter{}
var _ http.Flusher = &fancyWriter{}
var _ http.Hijacker = &fancyWriter{}
var _ io.ReaderFrom = &fancyWriter{}
