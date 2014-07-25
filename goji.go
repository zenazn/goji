/*
Package goji provides an out-of-box web server with reasonable defaults.

Example:
	package main

	import (
		"fmt"
		"net/http"

		"github.com/zenazn/goji"
		"github.com/zenazn/goji/web"
	)

	func hello(c web.C, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
	}

	func main() {
		goji.Get("/hello/:name", hello)
		goji.Serve()
	}

This package exists purely as a convenience to programmers who want to get
started as quickly as possible. It draws almost all of its code from goji's
subpackages, the most interesting of which is goji/web, and where most of the
documentation for the web framework lives.

A side effect of this package's ease-of-use is the fact that it is opinionated.
If you don't like (or have outgrown) its opinions, it should be straightforward
to use the APIs of goji's subpackages to reimplement things to your liking. Both
methods of using this library are equally well supported.

Goji requires Go 1.2 or newer.
*/
package goji

import (
	"flag"
	"log"
	"net/http"

	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
)

func init() {
	bind.WithFlag()
	if fl := log.Flags(); fl&log.Ltime != 0 {
		log.SetFlags(fl | log.Lmicroseconds)
	}
}

// Serve starts Goji using reasonable defaults.
func Serve() {
	if !flag.Parsed() {
		flag.Parse()
	}

	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", DefaultMux)

	listener := bind.Default()
	log.Println("Starting Goji on", listener.Addr())

	graceful.HandleSignals()
	bind.Ready()

	err := graceful.Serve(listener, http.DefaultServeMux)

	if err != nil {
		log.Fatal(err)
	}

	graceful.Wait()
}
