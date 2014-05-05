package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/zenazn/goji/web"
)

// Key to use when setting the request ID.
const RequestIDKey = "reqID"

var prefix string
var reqid uint64

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var buf [12]byte
	rand.Read(buf[:])
	b64 := base64.StdEncoding.EncodeToString(buf[:])
	// Strip out annoying characters. We have something like a billion to
	// one chance of having enough from 12 bytes of entropy
	b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)

	prefix = fmt.Sprintf("%s/%s", hostname, b64[0:8])
}

// RequestID is a middleware that injects a request ID into the context of each
// request. A request ID is a string of the form "host.example.com/random-0001",
// where "random" is a base62 random string that uniquely identifies this go
// process, and where the last number is an atomically incremented request
// counter.
func RequestID(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if c.Env == nil {
			c.Env = make(map[string]interface{})
		}
		myid := atomic.AddUint64(&reqid, 1)
		c.Env[RequestIDKey] = fmt.Sprintf("%s-%06d", prefix, myid)

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// GetReqID returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
func GetReqID(c web.C) string {
	if c.Env == nil {
		return ""
	}
	v, ok := c.Env[RequestIDKey]
	if !ok {
		return ""
	}
	if reqID, ok := v.(string); ok {
		return reqID
	}
	return ""
}
