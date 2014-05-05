package middleware

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
)

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return. When standard output is a TTY, Logger will
// print in color, otherwise it will print in black and white.
//
// Logger prints a request ID if one is provided.
func Logger(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqID := GetReqID(*c)

		printStart(reqID, r)

		lw := wrapWriter(w)

		t1 := time.Now()
		h.ServeHTTP(lw, r)
		lw.maybeWriteHeader()
		t2 := time.Now()

		printEnd(reqID, lw, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func printStart(reqID string, r *http.Request) {
	var buf bytes.Buffer

	if reqID != "" {
		cW(&buf, bBlack, "[%s] ", reqID)
	}
	buf.WriteString("Started ")
	cW(&buf, bMagenta, "%s ", r.Method)
	cW(&buf, nBlue, "%q ", r.URL.String())
	buf.WriteString("from ")
	buf.WriteString(r.RemoteAddr)

	log.Print(buf.String())
}

func printEnd(reqID string, w writerProxy, dt time.Duration) {
	var buf bytes.Buffer

	if reqID != "" {
		cW(&buf, bBlack, "[%s] ", reqID)
	}
	buf.WriteString("Returning ")
	if w.status() < 200 {
		cW(&buf, bBlue, "%03d", w.status())
	} else if w.status() < 300 {
		cW(&buf, bGreen, "%03d", w.status())
	} else if w.status() < 400 {
		cW(&buf, bCyan, "%03d", w.status())
	} else if w.status() < 500 {
		cW(&buf, bYellow, "%03d", w.status())
	} else {
		cW(&buf, bRed, "%03d", w.status())
	}
	buf.WriteString(" in ")
	if dt < 500*time.Millisecond {
		cW(&buf, nGreen, "%s", dt)
	} else if dt < 5*time.Second {
		cW(&buf, nYellow, "%s", dt)
	} else {
		cW(&buf, nRed, "%s", dt)
	}

	log.Print(buf.String())
}
