package main

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/zenazn/goji/web"
)
//Inline MiddleWare example
func InlineMiddleWare(h web.HandlerFunc) web.HandlerFunc {
    return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
        
        w.Write([]byte("Doing some fancy middleware operations - BEFORE \n"))
        h.ServeHTTPC(c, w, r)
        w.Write([]byte("Doing some fancy middleware operations - AFTER \n"))
    })
}

// PlainText sets the content-type of responses to text/plain.
func PlainText(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Nobody will ever guess this!
const Password = "admin:admin"

// SuperSecure is HTTP Basic Auth middleware for super-secret admin page. Shhhh!
func SuperSecure(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			pleaseAuth(w)
			return
		}

		password, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil || string(password) != Password {
			pleaseAuth(w)
			return
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func pleaseAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Gritter"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Go away!\n"))
}
