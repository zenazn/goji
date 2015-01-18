package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
	"time"
)

// These tests can probably be DRY'd up a bunch

func chHandler(ch chan string, s string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- s
	})
}

var methods = []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH",
	"POST", "PUT", "TRACE", "OTHER"}

func TestMethods(t *testing.T) {
	t.Parallel()
	m := New()
	ch := make(chan string, 1)

	m.Connect("/", chHandler(ch, "CONNECT"))
	m.Delete("/", chHandler(ch, "DELETE"))
	m.Head("/", chHandler(ch, "HEAD"))
	m.Get("/", chHandler(ch, "GET"))
	m.Options("/", chHandler(ch, "OPTIONS"))
	m.Patch("/", chHandler(ch, "PATCH"))
	m.Post("/", chHandler(ch, "POST"))
	m.Put("/", chHandler(ch, "PUT"))
	m.Trace("/", chHandler(ch, "TRACE"))
	m.Handle("/", chHandler(ch, "OTHER"))

	for _, method := range methods {
		r, _ := http.NewRequest(method, "/", nil)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		select {
		case val := <-ch:
			if val != method {
				t.Errorf("Got %q, expected %q", val, method)
			}
		case <-time.After(5 * time.Millisecond):
			t.Errorf("Timeout waiting for method %q", method)
		}
	}
}

type testPattern struct{}

func (t testPattern) Prefix() string {
	return ""
}

func (t testPattern) Match(r *http.Request, c *C) bool {
	return true
}
func (t testPattern) Run(r *http.Request, c *C) {
}

var _ Pattern = testPattern{}

func TestPatternTypes(t *testing.T) {
	t.Parallel()
	m := New()

	m.Get("/hello/carl", http.NotFound)
	m.Get("/hello/:name", http.NotFound)
	m.Get(regexp.MustCompile(`^/hello/(?P<name>.+)$`), http.NotFound)
	m.Get(testPattern{}, http.NotFound)
}

type testHandler chan string

func (t testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t <- "http"
}
func (t testHandler) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	t <- "httpc"
}

var testHandlerTable = map[string]string{
	"/a": "http fn",
	"/b": "http handler",
	"/c": "web fn",
	"/d": "web handler",
	"/e": "httpc",
}

func TestHandlerTypes(t *testing.T) {
	t.Parallel()
	m := New()
	ch := make(chan string, 1)

	m.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		ch <- "http fn"
	})
	m.Get("/b", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- "http handler"
	}))
	m.Get("/c", func(c C, w http.ResponseWriter, r *http.Request) {
		ch <- "web fn"
	})
	m.Get("/d", HandlerFunc(func(c C, w http.ResponseWriter, r *http.Request) {
		ch <- "web handler"
	}))
	m.Get("/e", testHandler(ch))

	for route, response := range testHandlerTable {
		r, _ := http.NewRequest("GET", route, nil)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		select {
		case resp := <-ch:
			if resp != response {
				t.Errorf("Got %q, expected %q", resp, response)
			}
		case <-time.After(5 * time.Millisecond):
			t.Errorf("Timeout waiting for path %q", route)
		}

	}
}

// The idea behind this test is to comprehensively test if routes are being
// applied in the right order. We define a special pattern type that always
// matches so long as it's greater than or equal to the global test index. By
// incrementing this index, we can invalidate all routes up to some point, and
// therefore test the routing guarantee that Goji provides: for any path P, if
// both A and B match P, and if A was inserted before B, then Goji will route to
// A before it routes to B.
var rsRoutes = []string{
	"/",
	"/a",
	"/a",
	"/b",
	"/ab",
	"/",
	"/ba",
	"/b",
	"/a",
}

var rsTests = []struct {
	key     string
	results []int
}{
	{"/", []int{0, 5, 5, 5, 5, 5, -1, -1, -1, -1}},
	{"/a", []int{0, 1, 2, 5, 5, 5, 8, 8, 8, -1}},
	{"/b", []int{0, 3, 3, 3, 5, 5, 7, 7, -1, -1}},
	{"/ab", []int{0, 1, 2, 4, 4, 5, 8, 8, 8, -1}},
	{"/ba", []int{0, 3, 3, 3, 5, 5, 6, 7, -1, -1}},
	{"/c", []int{0, 5, 5, 5, 5, 5, -1, -1, -1, -1}},
	{"nope", []int{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1}},
}

type rsPattern struct {
	i       int
	counter *int
	prefix  string
	ichan   chan int
}

func (rs rsPattern) Prefix() string {
	return rs.prefix
}
func (rs rsPattern) Match(_ *http.Request, _ *C) bool {
	return rs.i >= *rs.counter
}
func (rs rsPattern) Run(_ *http.Request, _ *C) {
}

func (rs rsPattern) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	rs.ichan <- rs.i
}

var _ Pattern = rsPattern{}
var _ http.Handler = rsPattern{}

func TestRouteSelection(t *testing.T) {
	t.Parallel()
	m := New()
	counter := 0
	ichan := make(chan int, 1)
	m.NotFound(func(w http.ResponseWriter, r *http.Request) {
		ichan <- -1
	})

	for i, s := range rsRoutes {
		pat := rsPattern{
			i:       i,
			counter: &counter,
			prefix:  s,
			ichan:   ichan,
		}
		m.Get(pat, pat)
	}

	for _, test := range rsTests {
		var n int
		for counter, n = range test.results {
			r, _ := http.NewRequest("GET", test.key, nil)
			w := httptest.NewRecorder()
			m.ServeHTTP(w, r)
			actual := <-ichan
			if n != actual {
				t.Errorf("Expected %q @ %d to be %d, got %d",
					test.key, counter, n, actual)
			}
		}
	}
}

func TestNotFound(t *testing.T) {
	t.Parallel()
	m := New()

	r, _ := http.NewRequest("post", "/", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}

	m.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "I'm a teapot!", http.StatusTeapot)
	})

	r, _ = http.NewRequest("POST", "/", nil)
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusTeapot {
		t.Errorf("Expected a teapot, got %d", w.Code)
	}
}

func TestPrefix(t *testing.T) {
	t.Parallel()
	m := New()
	ch := make(chan string, 1)

	m.Handle("/hello/*", func(w http.ResponseWriter, r *http.Request) {
		ch <- r.URL.Path
	})

	r, _ := http.NewRequest("GET", "/hello/world", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	select {
	case val := <-ch:
		if val != "/hello/world" {
			t.Errorf("Got %q, expected /hello/world", val)
		}
	case <-time.After(5 * time.Millisecond):
		t.Errorf("Timeout waiting for hello")
	}
}

var validMethodsTable = map[string][]string{
	"/hello/carl":       {"DELETE", "GET", "HEAD", "PATCH", "POST", "PUT"},
	"/hello/bob":        {"DELETE", "GET", "HEAD", "PATCH", "PUT"},
	"/hola/carl":        {"DELETE", "GET", "HEAD", "PUT"},
	"/hola/bob":         {"DELETE"},
	"/does/not/compute": {},
}

func TestValidMethods(t *testing.T) {
	t.Parallel()
	m := New()
	ch := make(chan []string, 1)

	m.NotFound(func(c C, w http.ResponseWriter, r *http.Request) {
		if c.Env == nil {
			ch <- []string{}
			return
		}
		methods, ok := c.Env[ValidMethodsKey]
		if !ok {
			ch <- []string{}
			return
		}
		ch <- methods.([]string)
	})

	m.Get("/hello/carl", http.NotFound)
	m.Post("/hello/carl", http.NotFound)
	m.Head("/hello/bob", http.NotFound)
	m.Get("/hello/:name", http.NotFound)
	m.Put("/hello/:name", http.NotFound)
	m.Patch("/hello/:name", http.NotFound)
	m.Get("/:greet/carl", http.NotFound)
	m.Put("/:greet/carl", http.NotFound)
	m.Delete("/:greet/:anyone", http.NotFound)

	for path, eMethods := range validMethodsTable {
		r, _ := http.NewRequest("BOGUS", path, nil)
		m.ServeHTTP(httptest.NewRecorder(), r)
		aMethods := <-ch
		if !reflect.DeepEqual(eMethods, aMethods) {
			t.Errorf("For %q, expected %v, got %v", path, eMethods,
				aMethods)
		}
	}

	// This should also work when c.Env has already been initalized
	m.Use(func(c *C, h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Env = make(map[interface{}]interface{})
			h.ServeHTTP(w, r)
		})
	})
	for path, eMethods := range validMethodsTable {
		r, _ := http.NewRequest("BOGUS", path, nil)
		m.ServeHTTP(httptest.NewRecorder(), r)
		aMethods := <-ch
		if !reflect.DeepEqual(eMethods, aMethods) {
			t.Errorf("For %q, expected %v, got %v", path, eMethods,
				aMethods)
		}
	}
}
