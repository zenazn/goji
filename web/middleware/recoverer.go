package middleware

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/zenazn/goji/web"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible.
//
// Recoverer prints a request ID if one is provided.
func Recoverer(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqID := GetReqID(*c)

		defer func() {
			if err := recover(); err != nil {
				printPanic(reqID, err)
				printStack(debug.Stack())
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func printPanic(reqID string, err interface{}) {
	var buf bytes.Buffer

	if reqID != "" {
		cW(&buf, bBlack, "[%s] ", reqID)
	}
	cW(&buf, bRed, "panic: %+v", err)

	log.Print(buf.String())
}

func printStack(stack []byte) {
	// skip callers on stack
	split := bytes.Split(stack, []byte{0x0a})
	stack = bytes.Join(split[6:], []byte{0x0a})

	os.Stderr.Write(stack)
}
