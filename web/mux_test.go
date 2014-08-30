package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.google.com/p/go.net/context"
)

// Sanity check types
var _ http.Handler = &Mux{}
var _ Handler = &Mux{}

// There's... really not a lot to do here.
type testKey int

const greetingKey testKey = 0

func TestIfItWorks(t *testing.T) {
	t.Parallel()

	m := New()
	ch := make(chan string, 1)

	m.Get("/hello/:name", func(c context.Context, w http.ResponseWriter, r *http.Request) {
		greeting := "Hello "
		if g, ok := c.Value(greetingKey).(string); ok {
			greeting = g
		}
		ch <- greeting + URLParams(c)["name"]
	})

	r, _ := http.NewRequest("GET", "/hello/carl", nil)
	m.ServeHTTP(httptest.NewRecorder(), r)
	out := <-ch
	if out != "Hello carl" {
		t.Errorf(`Unexpected response %q, expected "Hello carl"`, out)
	}

	r, _ = http.NewRequest("GET", "/hello/bob", nil)
	ctx := context.WithValue(context.Background(), greetingKey, "Yo ")
	m.ServeHTTPC(ctx, httptest.NewRecorder(), r)
	out = <-ch
	if out != "Yo bob" {
		t.Errorf(`Unexpected response %q, expected "Yo bob"`, out)
	}
}
