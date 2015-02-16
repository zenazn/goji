// +build go1.3

package web

import (
	"crypto/rand"
	"encoding/base64"
	mrand "math/rand"
	"net/http"
	"testing"
)

/*
The core benchmarks here are based on cypriss's mux benchmarks, which can be
found here:
https://github.com/cypriss/golang-mux-benchmark

They happen to play very well into Goji's router's strengths.
*/

type nilRouter struct{}

var helloWorld = []byte("Hello world!\n")

func (_ nilRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(helloWorld)
}

type nilResponse struct{}

func (_ nilResponse) Write(buf []byte) (int, error) {
	return len(buf), nil
}
func (_ nilResponse) Header() http.Header {
	return nil
}
func (_ nilResponse) WriteHeader(code int) {
}

func trivialMiddleware(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

var w nilResponse

func addRoutes(m *Mux, prefix string) {
	m.Get(prefix, nilRouter{})
	m.Post(prefix, nilRouter{})
	m.Get(prefix+"/:id", nilRouter{})
	m.Put(prefix+"/:id", nilRouter{})
	m.Delete(prefix+"/:id", nilRouter{})
}

func randString() string {
	var buf [6]byte
	rand.Reader.Read(buf[:])
	return base64.URLEncoding.EncodeToString(buf[:])
}

func genPrefixes(n int) []string {
	p := make([]string, n)
	for i := range p {
		p[i] = "/" + randString()
	}
	return p
}

func genRequests(prefixes []string) []*http.Request {
	rs := make([]*http.Request, 5*len(prefixes))
	for i, prefix := range prefixes {
		rs[5*i+0], _ = http.NewRequest("GET", prefix, nil)
		rs[5*i+1], _ = http.NewRequest("POST", prefix, nil)
		rs[5*i+2], _ = http.NewRequest("GET", prefix+"/foo", nil)
		rs[5*i+3], _ = http.NewRequest("PUT", prefix+"/foo", nil)
		rs[5*i+4], _ = http.NewRequest("DELETE", prefix+"/foo", nil)
	}
	return rs
}

func permuteRequests(reqs []*http.Request) []*http.Request {
	out := make([]*http.Request, len(reqs))
	perm := mrand.Perm(len(reqs))
	for i, req := range reqs {
		out[perm[i]] = req
	}
	return out
}

func benchN(b *testing.B, n int) {
	m := New()
	prefixes := genPrefixes(n)
	for _, prefix := range prefixes {
		addRoutes(m, prefix)
	}
	m.Compile()
	reqs := permuteRequests(genRequests(prefixes))

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			m.ServeHTTP(w, reqs[i%len(reqs)])
		}
	})
}

func benchM(b *testing.B, n int) {
	m := New()
	m.Get("/", nilRouter{})
	for i := 0; i < n; i++ {
		m.Use(trivialMiddleware)
	}
	r, _ := http.NewRequest("GET", "/", nil)
	m.Compile()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.ServeHTTP(w, r)
		}
	})
}

func BenchmarkStatic(b *testing.B) {
	m := New()
	m.Get("/", nilRouter{})
	r, _ := http.NewRequest("GET", "/", nil)
	m.Compile()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.ServeHTTP(w, r)
		}
	})
}

func BenchmarkRoute5(b *testing.B) {
	benchN(b, 1)
}
func BenchmarkRoute50(b *testing.B) {
	benchN(b, 10)
}
func BenchmarkRoute500(b *testing.B) {
	benchN(b, 100)
}
func BenchmarkRoute5000(b *testing.B) {
	benchN(b, 1000)
}

func BenchmarkMiddleware1(b *testing.B) {
	benchM(b, 1)
}
func BenchmarkMiddleware10(b *testing.B) {
	benchM(b, 10)
}
func BenchmarkMiddleware100(b *testing.B) {
	benchM(b, 100)
}
