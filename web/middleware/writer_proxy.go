package middleware

import (
	"bufio"
	"net"
	"net/http"
)

type writerProxy interface {
	http.ResponseWriter
	maybeWriteHeader()
	status() int
}

type basicWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
}

func (b *basicWriter) WriteHeader(code int) {
	b.code = code
	b.wroteHeader = true
	b.ResponseWriter.WriteHeader(code)
}
func (b *basicWriter) Write(buf []byte) (int, error) {
	b.maybeWriteHeader()
	return b.ResponseWriter.Write(buf)
}
func (b *basicWriter) maybeWriteHeader() {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
}
func (b *basicWriter) status() int {
	return b.code
}
func (b *basicWriter) Unwrap() http.ResponseWriter {
	return b.ResponseWriter
}

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
func (f *fancyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj := f.basicWriter.ResponseWriter.(http.Hijacker)
	return hj.Hijack()
}

var _ http.CloseNotifier = &fancyWriter{}
var _ http.Flusher = &fancyWriter{}
var _ http.Hijacker = &fancyWriter{}
