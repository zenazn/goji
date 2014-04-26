package middleware

import (
	"net/http"
	"strings"

	"github.com/zenazn/goji/web"
)

// Key the original value of RemoteAddr is stored under.
const OriginalRemoteAddrKey = "originalRemoteAddr"

var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")

// RealIP is a middleware that sets a http.Request's RemoteAddr to the results
// of parsing either the X-Forwarded-For header or the X-Real-IP header (in that
// order). It places the original value of RemoteAddr in a context environment
// variable.
//
// This middleware should be inserted fairly early in the middleware stack to
// ensure that subsequent layers (e.g., request loggers) which examine the
// RemoteAddr will see the intended value.
//
// You should only use this middleware if you can trust the headers passed to
// you (in particular, the two headers this middleware uses), for example
// because you have placed a reverse proxy like HAProxy or nginx in front of
// Goji. If your reverse proxies are configured to pass along arbitrary header
// values from the client, or if you use this middleware without a reverse
// proxy, malicious clients will be able to make you very sad (or, depending on
// how you're using RemoteAddr, vulnerable to an attack of some sort).
func RealIP(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rip := realIP(r); rip != "" {
			if c.Env == nil {
				c.Env = make(map[string]interface{})
			}
			c.Env[OriginalRemoteAddrKey] = r.RemoteAddr
			r.RemoteAddr = rip
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func realIP(r *http.Request) string {
	var ip string

	if xff := r.Header.Get(xForwardedFor); xff != "" {
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		ip = xrip
	}

	return ip
}
