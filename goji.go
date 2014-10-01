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
