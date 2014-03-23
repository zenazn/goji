package middleware

import (
	"bytes"
	"log"
	"net/http"
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
		reqId := GetReqId(*c)

		defer func() {
			if err := recover(); err != nil {
				printPanic(reqId, err)
				debug.PrintStack()
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func printPanic(reqId string, err interface{}) {
	var buf bytes.Buffer

	if reqId != "" {
		cW(&buf, bBlack, "[%s] ", reqId)
	}
	cW(&buf, bRed, "panic: %#v", err)

	log.Print(buf.String())
}
