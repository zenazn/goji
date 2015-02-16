package web_test

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

func Example() {
	m := web.New()

	// Use your favorite HTTP verbs and the interfaces you know and love
	// from net/http:
	m.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Why hello there!\n")
	})
	m.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("password") != "god" {
			http.Error(w, "Hack the planet!", 401)
		}
	})

	// Handlers can optionally take a context parameter, which contains
	// (among other things) a set of bound parameters.
	hello := func(c web.C, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!\n", c.URLParams["name"])
	}

	// Bind parameters using pattern strings...
	m.Get("/hello/:name", hello)

	// ...or use regular expressions if you need additional power.
	bonjour := regexp.MustCompile(`^/bonjour/(?P<name>[A-Za-z]+)$`)
	m.Get(bonjour, hello)

	// Middleware are a great abstraction for performing logic on every
	// request. Some middleware use the Goji context object to set
	// request-scoped variables.
	logger := func(h http.Handler) http.Handler {
		wrap := func(w http.ResponseWriter, r *http.Request) {
			log.Println("Before request")
			h.ServeHTTP(w, r)
			log.Println("After request")
		}
		return http.HandlerFunc(wrap)
	}
	auth := func(c *web.C, h http.Handler) http.Handler {
		wrap := func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie("user"); err == nil {
				c.Env["user"] = cookie.Value
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(wrap)
	}

	// A Middleware stack is a flexible way to assemble the common
	// components of your application, like request loggers and
	// authentication. There is an ecosystem of open-source middleware for
	// Goji, so there's a chance someone has already written the middleware
	// you are looking for!
	m.Use(middleware.EnvInit)
	m.Use(logger)
	m.Use(auth)
}
