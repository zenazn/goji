package web

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type method int

const (
	mCONNECT method = 1 << iota
	mDELETE
	mGET
	mHEAD
	mOPTIONS
	mPATCH
	mPOST
	mPUT
	mTRACE
	// We only natively support the methods above, but we pass through other
	// methods. This constant pretty much only exists for the sake of mALL.
	mIDK

	mALL method = mCONNECT | mDELETE | mGET | mHEAD | mOPTIONS | mPATCH |
		mPOST | mPUT | mTRACE | mIDK
)

type route struct {
	// Theory: most real world routes have a string prefix which is both
	// cheap(-ish) to test against and pretty selective. And, conveniently,
	// both regexes and string patterns give us this out-of-box.
	prefix  string
	method  method
	pattern Pattern
	handler Handler
}

type router struct {
	lock     sync.Mutex
	routes   []route
	notFound Handler
}

// A pattern determines whether or not a given request matches some criteria.
// They are often used in routes, which are essentially (pattern, methodSet,
// handler) tuples. If the method and pattern match, the given handler is used.
//
// Built-in implementations of this interface are used to implement regular
// expression and string matching.
type Pattern interface {
	// In practice, most real-world routes have a string prefix that can be
	// used to quickly determine if a pattern is an eligible match. The
	// router uses the result of this function to optimize away calls to the
	// full Match function, which is likely much more expensive to compute.
	// If your Pattern does not support prefixes, this function should
	// return the empty string.
	Prefix() string
	// Returns true if the request satisfies the pattern. This function is
	// free to examine both the request and the context to make this
	// decision. After it is certain that the request matches, this function
	// should mutate or create c.UrlParams if necessary.
	Match(r *http.Request, c *C) bool
}

func parsePattern(p interface{}, isPrefix bool) Pattern {
	switch p.(type) {
	case Pattern:
		return p.(Pattern)
	case *regexp.Regexp:
		return parseRegexpPattern(p.(*regexp.Regexp))
	case string:
		return parseStringPattern(p.(string), isPrefix)
	default:
		log.Fatalf("Unknown pattern type %v. Expected a web.Pattern, "+
			"regexp.Regexp, or a string.", p)
	}
	panic("log.Fatalf does not return")
}

type netHttpWrap struct {
	http.Handler
}

func (h netHttpWrap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(w, r)
}
func (h netHttpWrap) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(w, r)
}

func parseHandler(h interface{}) Handler {
	switch h.(type) {
	case Handler:
		return h.(Handler)
	case http.Handler:
		return netHttpWrap{h.(http.Handler)}
	case func(c C, w http.ResponseWriter, r *http.Request):
		f := h.(func(c C, w http.ResponseWriter, r *http.Request))
		return HandlerFunc(f)
	case func(w http.ResponseWriter, r *http.Request):
		f := h.(func(w http.ResponseWriter, r *http.Request))
		return netHttpWrap{http.HandlerFunc(f)}
	default:
		log.Fatalf("Unknown handler type %v. Expected a web.Handler, "+
			"a http.Handler, or a function with signature func(C, "+
			"http.ResponseWriter, *http.Request) or "+
			"func(http.ResponseWriter, http.Request)", h)
	}
	panic("log.Fatalf does not return")
}

func httpMethod(mname string) method {
	switch strings.ToUpper(mname) {
	case "CONNECT":
		return mCONNECT
	case "DELETE":
		return mDELETE
	case "GET":
		return mGET
	case "HEAD":
		return mHEAD
	case "OPTIONS":
		return mOPTIONS
	case "PATCH":
		return mPATCH
	case "POST":
		return mPOST
	case "PUT":
		return mPUT
	case "TRACE":
		return mTRACE
	default:
		return mIDK
	}
}

func (rt *router) route(c C, w http.ResponseWriter, r *http.Request) {
	m := httpMethod(r.Method)
	for _, route := range rt.routes {
		if route.method&m == 0 ||
			!strings.HasPrefix(r.URL.Path, route.prefix) ||
			!route.pattern.Match(r, &c) {
			continue
		}
		route.handler.ServeHTTPC(c, w, r)
		return
	}

	rt.notFound.ServeHTTPC(c, w, r)
}

func (rt *router) handleUntyped(p interface{}, m method, h interface{}) {
	pat := parsePattern(p, false)
	rt.handle(pat, m, parseHandler(h))
}

func (rt *router) handle(p Pattern, m method, h Handler) {
	// We're being a little sloppy here: we assume that pointer assignments
	// are atomic, and that there is no way a locked append here can affect
	// another goroutine which looked at rt.routes without a lock.
	rt.lock.Lock()
	defer rt.lock.Unlock()
	rt.routes = append(rt.routes, route{
		prefix:  p.Prefix(),
		method:  m,
		pattern: p,
		handler: h,
	})
}

// This is a bit silly, but I've renamed the method receivers in the public
// functions here "m" instead of the standard "rt", since they will eventually
// be shown on the documentation for the Mux that they are included in.

// Dispatch to the given handler when the pattern matches, regardless of HTTP
// method. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (m *router) Handle(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mALL, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// CONNECT. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (m *router) Connect(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mCONNECT, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// DELETE. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (m *router) Delete(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mDELETE, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// GET. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Get(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mGET, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// HEAD. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Head(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mHEAD, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// OPTIONS. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (m *router) Options(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mOPTIONS, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// PATCH. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Patch(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mPATCH, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// POST. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Post(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mPOST, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// PUT. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Put(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mPUT, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// TRACE. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (m *router) Trace(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mTRACE, handler)
}

/*
Dispatch to the given handler when the given (Sinatra-like) pattern matches a
prefix of the path. This function explicitly takes a string parameter since you
can implement this behavior with a regular expression using the standard
Handle() function.

This function is probably most helpful when implementing sub-routing: an admin
application, for instance, can expose a single handler, and can be hooked up all
at once by attaching a sub-route at "/admin".

Notably, this function does not strip the matched prefix from downstream
handlers, so in the above example the handler would recieve requests with path
"/admin/foo/bar", for example, instead of just "/foo/bar". Luckily, this is a
problem easily surmountable by middleware.

See the documentation for type Mux for a description of what types are accepted
for handler.
*/
func (m *router) Sub(pattern string, handler interface{}) {
	pat := parsePattern(pattern, true)
	m.handle(pat, mALL, parseHandler(handler))
}

// Set the fallback (i.e., 404) handler for this mux. See the documentation for
// type Mux for a description of what types are accepted for handler.
func (m *router) NotFound(handler interface{}) {
	m.notFound = parseHandler(handler)
}
