package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"code.google.com/p/go.net/context"

	"github.com/zenazn/goji/web"
)

type ctxkey string

// Key to use when setting the request ID.
const RequestIDKey ctxkey = "reqID"

var prefix string
var reqid uint64

/*
A quick note on the statistics here: we're trying to calculate the chance that
two randomly generated base62 prefixes will collide. We use the formula from
http://en.wikipedia.org/wiki/Birthday_problem

P[m, n] \approx 1 - e^{-m^2/2n}

We ballpark an upper bound for $m$ by imagining (for whatever reason) a server
that restarts every second over 10 years, for $m = 86400 * 365 * 10 = 315360000$

For a $k$ character base-62 identifier, we have $n(k) = 62^k$

Plugging this in, we find $P[m, n(10)] \approx 5.75%$, which is good enough for
our purposes, and is surely more than anyone would ever need in practice -- a
process that is rebooted a handful of times a day for a hundred years has less
than a millionth of a percent chance of generating two colliding IDs.
*/

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var buf [12]byte
	var b64 string
	for len(b64) < 10 {
		rand.Read(buf[:])
		b64 = base64.StdEncoding.EncodeToString(buf[:])
		b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)
	}

	prefix = fmt.Sprintf("%s/%s", hostname, b64[0:10])
}

// RequestID is a middleware that injects a request ID into the context of each
// request. A request ID is a string of the form "host.example.com/random-0001",
// where "random" is a base62 random string that uniquely identifies this go
// process, and where the last number is an atomically incremented request
// counter.
func RequestID(h web.Handler) web.Handler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		myid := atomic.AddUint64(&reqid, 1)
		c := context.WithValue(ctx, RequestIDKey, fmt.Sprintf("%s-%06d", prefix, myid))
		h.ServeHTTPC(c, w, r)
	}

	return web.HandlerFunc(fn)
}

// GetReqID returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
func GetReqID(c context.Context) string {
	v, ok := c.Value(RequestIDKey).(string)
	if !ok {
		return ""
	}
	return v
}
