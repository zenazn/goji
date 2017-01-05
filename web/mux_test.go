package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Sanity check types
var _ http.Handler = &Mux{}
var _ Handler = &Mux{}

// There's... really not a lot to do here.

func TestIfItWorks(t *testing.T) {
	t.Parallel()

	m := New()
	ch := make(chan string, 1)

	m.Get("/hello/:name", func(c C, w http.ResponseWriter, r *http.Request) {
		greeting := "Hello "
		if c.Env != nil {
			if g, ok := c.Env["greeting"]; ok {
				greeting = g.(string)
			}
		}
		ch <- greeting + c.URLParams["name"]
	})

	r, _ := http.NewRequest("GET", "/hello/carl", nil)
	m.ServeHTTP(httptest.NewRecorder(), r)
	out := <-ch
	if out != "Hello carl" {
		t.Errorf(`Unexpected response %q, expected "Hello carl"`, out)
	}

	r, _ = http.NewRequest("GET", "/hello/bob", nil)
	env := map[interface{}]interface{}{"greeting": "Yo "}
	m.ServeHTTPC(C{Env: env}, httptest.NewRecorder(), r)
	out = <-ch
	if out != "Yo bob" {
		t.Errorf(`Unexpected response %q, expected "Yo bob"`, out)
	}
}

func TestMiddlewareOrderForNewUse(t *testing.T) {
	t.Parallel()

	m := New()
	ch := make(chan string)

	// Reusing couple of test methods from middleware_test.go
	m.Use(chanWare(ch, "Hello carl"), chanWare(ch, ", this"), chanWare(ch, "is"),
		chanWare(ch, "enhanced"), chanWare(ch, "Use"), chanWare(ch, "method."))

	go simpleRequest(ch, &m.ms)
	assertOrder(t, ch, "Hello carl", ", this", "is", "enhanced", "Use", "method.")
}
