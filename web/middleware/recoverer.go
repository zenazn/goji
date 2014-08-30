package middleware

import (
	"bytes"
	"log"
	"net/http"
	"runtime/debug"
	"code.google.com/p/go.net/context"

	"github.com/zenazn/goji/web"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible.
//
// Recoverer prints a request ID if one is provided.
func Recoverer(h web.Handler) web.Handler {
	fn := func(c context.Context, w http.ResponseWriter, r *http.Request) {
		reqID := GetReqID(c)

		defer func() {
			if err := recover(); err != nil {
				printPanic(reqID, err)
				debug.PrintStack()
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		h.ServeHTTPC(c, w, r)
	}

	return web.HandlerFunc(fn)
}

func printPanic(reqID string, err interface{}) {
	var buf bytes.Buffer

	if reqID != "" {
		cW(&buf, bBlack, "[%s] ", reqID)
	}
	cW(&buf, bRed, "panic: %+v", err)

	log.Print(buf.String())
}
