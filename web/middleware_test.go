package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type iRouter func(*C, http.ResponseWriter, *http.Request)

func (i iRouter) route(c *C, w http.ResponseWriter, r *http.Request) {
	i(c, w, r)
}

func makeStack(ch chan string) *mStack {
	router := func(c *C, w http.ResponseWriter, r *http.Request) {
		ch <- "router"
	}
	return &mStack{
		stack:  make([]mLayer, 0),
		pool:   makeCPool(),
		router: iRouter(router),
	}
}

func chanWare(ch chan string, s string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ch <- s
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func simpleRequest(ch chan string, st *mStack) {
	defer func() {
		ch <- "end"
	}()
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	cs := st.alloc()
	defer st.release(cs)

	cs.ServeHTTP(w, r)
}

func assertOrder(t *testing.T, ch chan string, strings ...string) {
	for i, s := range strings {
		var v string
		select {
		case v = <-ch:
		case <-time.After(5 * time.Millisecond):
			t.Fatalf("Expected %q as %d'th value, but timed out", s,
				i+1)
		}
		if s != v {
			t.Errorf("%d'th value was %q, expected %q", i+1, v, s)
		}
	}
}

func TestSimple(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	st := makeStack(ch)
	st.Use(chanWare(ch, "one"))
	st.Use(chanWare(ch, "two"))
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "two", "router", "end")
}

func TestTypes(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	st := makeStack(ch)
	st.Use(func(h http.Handler) http.Handler {
		return h
	})
	st.Use(func(c *C, h http.Handler) http.Handler {
		return h
	})
}

func TestAddMore(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	st := makeStack(ch)
	st.Use(chanWare(ch, "one"))
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "router", "end")

	st.Use(chanWare(ch, "two"))
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "two", "router", "end")

	st.Use(chanWare(ch, "three"))
	st.Use(chanWare(ch, "four"))
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "two", "three", "four", "router", "end")
}

func TestInsert(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	st := makeStack(ch)
	one := chanWare(ch, "one")
	two := chanWare(ch, "two")
	st.Use(one)
	st.Use(two)
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "two", "router", "end")

	err := st.Insert(chanWare(ch, "sloth"), chanWare(ch, "squirrel"))
	if err == nil {
		t.Error("Expected error when referencing unknown middleware")
	}

	st.Insert(chanWare(ch, "middle"), two)
	err = st.Insert(chanWare(ch, "start"), one)
	if err != nil {
		t.Fatal(err)
	}
	go simpleRequest(ch, st)
	assertOrder(t, ch, "start", "one", "middle", "two", "router", "end")
}

func TestAbandon(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	st := makeStack(ch)
	one := chanWare(ch, "one")
	two := chanWare(ch, "two")
	three := chanWare(ch, "three")
	st.Use(one)
	st.Use(two)
	st.Use(three)
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "two", "three", "router", "end")

	st.Abandon(two)
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "three", "router", "end")

	err := st.Abandon(chanWare(ch, "panda"))
	if err == nil {
		t.Error("Expected error when deleting unknown middleware")
	}

	st.Abandon(one)
	st.Abandon(three)
	go simpleRequest(ch, st)
	assertOrder(t, ch, "router", "end")

	st.Use(one)
	go simpleRequest(ch, st)
	assertOrder(t, ch, "one", "router", "end")
}

func TestContext(t *testing.T) {
	router := func(c *C, w http.ResponseWriter, r *http.Request) {
		if c.Env["reqID"].(int) != 2 {
			t.Error("Request id was not 2 :(")
		}
	}
	st := mStack{
		stack:  make([]mLayer, 0),
		pool:   makeCPool(),
		router: iRouter(router),
	}
	st.Use(func(c *C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if c.Env != nil || c.URLParams != nil {
				t.Error("Expected a clean context")
			}
			c.Env = make(map[interface{}]interface{})
			c.Env["reqID"] = 1

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})

	st.Use(func(c *C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if c.Env == nil {
				t.Error("Expected env from last middleware")
			}
			c.Env["reqID"] = c.Env["reqID"].(int) + 1

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})
	ch := make(chan string)
	go simpleRequest(ch, &st)
	assertOrder(t, ch, "end")
}
