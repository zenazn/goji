package graceful

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

/*
Middleware adds graceful shutdown capabilities to the given handler. When a
graceful shutdown is in progress, this middleware intercepts responses to add a
"Connection: close" header to politely inform the client that we are about to go
away.

This package creates a shim http.ResponseWriter that it passes to subsequent
handlers. Unfortunately, there's a great many optional interfaces that this
http.ResponseWriter might implement (e.g., http.CloseNotifier, http.Flusher, and
http.Hijacker), and in order to perfectly proxy all of these options we'd be
left with some kind of awful powerset of ResponseWriters, and that's not even
counting all the other custom interfaces you might be expecting. Instead of
doing that, we have implemented two kinds of proxies: one that contains no
additional methods (i.e., exactly corresponding to the http.ResponseWriter
interface), and one that supports all three of http.CloseNotifier, http.Flusher,
and http.Hijacker. If you find that this is not enough, the original
http.ResponseWriter can be retrieved by calling Unwrap() on the proxy object.

This middleware is automatically applied to every http.Handler passed to this
package, and most users will not need to call this function directly. It is
exported primarily for documentation purposes and in the off chance that someone
really wants more control over their http.Server than we currently provide.
*/
func Middleware(h http.Handler) http.Handler {
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
	select {
	case <-kill:
		b.ResponseWriter.Header().Set("Connection", "close")
	default:
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

	if conn, ok := c.(hijackConn); ok {
		c = conn.hijack()
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
