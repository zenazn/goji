package middleware

import (
	"encoding/base64"
	"fmt"
	"github.com/zenazn/goji/web"
	"net/http"
	"strings"
)

type basicAuth struct {
	h    http.Handler
	c    *web.C
	opts *AuthOptions
}

// AuthOptions stores the configuration for HTTP Basic Authentication.
//
// The Realm (scope/namespace), User and Password must be supplied.
// A http.Handler may also be passed to UnauthorizedHandler to override the
// default error handler if you wish to serve a custom template/response.
type AuthOptions struct {
	Realm               string
	User                string
	Password            string
	UnauthorizedHandler http.Handler
}

// Satisfies the http.Handler interface for basicAuth.
func (b basicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if we have a user-provided error handler, else set a default
	if b.opts.UnauthorizedHandler == nil {
		b.opts.UnauthorizedHandler = http.HandlerFunc(defaultUnauthorizedHandler)
		return
	}

	if b.authenticate(r) == false {
		b.requestAuth(w, r)
		return
	}

	// Call the next handler on success.
	b.h.ServeHTTP(w, r)
}

// authenticate validates the user:password combination provided in the request header.
// Returns 'false' if the user has not successfully authenticated.
func (b *basicAuth) authenticate(r *http.Request) bool {
	// Confirm the request is sending Basic Authentication credentials.
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
        return false
	}

	// Get the plain-text username and password from the request
	// The first six characters are skipped e.g. "Basic ".
	str, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return false
	}

	// Split on the first ":" character only, with any subsequent colons assumed to be part
	// of the password. Note that the RFC2617 standard does not place any limitations on
	// allowable characters in the password.
	creds := strings.SplitN(string(str), ":", 2)
	// Validate the user & password match.
	if creds[0] == b.opts.User && creds[1] == b.opts.Password {
		return true
	}

	return false
}

// Require authentication, and serve our error handler otherwise.
func (b *basicAuth) requestAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, b.opts.Realm))
	b.opts.UnauthorizedHandler.ServeHTTP(w, r)
}

// defaultUnauthorizedHandler provides a default HTTP 401 Unauthorized response.
func defaultUnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
	w.Write([]byte("You are not authorized to access this resource."))
}

// BasicAuth provides HTTP middleware for protecting URIs with HTTP Basic Authentication
// as per RFC 2617. The server authenticates a user:password combination provided in the
// "Authorization" HTTP header.
//
// Example:
//
//     package main
//
//     import(
//            "net/http"
//            "github.com/zenazn/goji/web"
//            "github.com/zenazn/goji/web/middleware"
//     )
//
//     func main() {
//          basicOpts := &middleware.AuthOptions{
//                      Realm: "Restricted",
//                      User: "Dave",
//                      Password: "ClearText",
//                  }
//
//          goji.Use(middleware.BasicAuth(basicOpts), middleware.SomeOtherMiddleware)
//          goji.Get("/thing", myHandler)
//  }
//
// Note: HTTP Basic Authentication credentials are sent in plain text, and therefore it does
// not make for a wholly secure authentication mechanism. You should serve your content over
// HTTPS to mitigate this, noting that "Basic Authentication" is meant to be just that: basic!
func BasicAuth(o *AuthOptions) func(*web.C, http.Handler) http.Handler {
	fn := func(c *web.C, h http.Handler) http.Handler {
		return basicAuth{h, c, o}
	}
	return fn
}
