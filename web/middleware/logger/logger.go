package logger

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
	. "github.com/zenazn/goji/web/middleware"
	"github.com/zenazn/goji/web/mutil"
)

type optSetter func(*logger) error

type logger struct {
	useReqId bool
}

func (l logger) middleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var reqID string
		if l.useReqId == true {
			reqID = GetReqID(*c)
		} else {
			reqID = ""
		}

		printStart(reqID, r)

		lw := mutil.WrapWriter(w)

		t1 := time.Now()
		h.ServeHTTP(lw, r)

		if lw.Status() == 0 {
			lw.WriteHeader(http.StatusOK)
		}
		t2 := time.Now()

		printEnd(reqID, lw, t2.Sub(t1))

	}
	return http.HandlerFunc(fn)
}

func New(setters ...optSetter) func(*web.C, http.Handler) http.Handler {
	l := &logger{true}

	for _, s := range setters {
		if err := s(l); err != nil {
			panic(err)
		}
	}

	return l.middleware
}

func UseReqID(flag bool) optSetter {
	return func(l *logger) error {
		l.useReqId = flag
		return nil
	}
}

func printStart(reqID string, r *http.Request) {
	var buf bytes.Buffer

	if reqID != "" {
		buf.WriteString("[" + reqID + "] ")
	}
	buf.WriteString(fmt.Sprintf("Started %s %q from %s", r.Method, r.URL.String(), r.RemoteAddr))

	log.Print(buf.String())
}

func printEnd(reqID string, w mutil.WriterProxy, dt time.Duration) {
	var buf bytes.Buffer

	if reqID != "" {
		buf.WriteString("[" + reqID + "] ")
	}
	buf.WriteString(fmt.Sprintf("Returning %03d in %s", w.Status(), dt))

	log.Print(buf.String())
}
