package web

import (
	"log"
	"net/http"
	"regexp"
	"sort"
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

// The key used to communicate to the NotFound handler what methods would have
// been allowed if they'd been provided.
const ValidMethodsKey = "goji.web.validMethods"

var validMethodsMap = map[string]method{
	"CONNECT": mCONNECT,
	"DELETE":  mDELETE,
	"GET":     mGET,
	"HEAD":    mHEAD,
	"OPTIONS": mOPTIONS,
	"PATCH":   mPATCH,
	"POST":    mPOST,
	"PUT":     mPUT,
	"TRACE":   mTRACE,
}

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

// A Pattern determines whether or not a given request matches some criteria.
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
	// should mutate or create c.UrlParams if necessary, unless dryrun is
	// set.
	Match(r *http.Request, c *C, dryrun bool) bool
}

func parsePattern(p interface{}) Pattern {
	switch p.(type) {
	case Pattern:
		return p.(Pattern)
	case *regexp.Regexp:
		return parseRegexpPattern(p.(*regexp.Regexp))
	case string:
		return parseStringPattern(p.(string))
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
	if method, ok := validMethodsMap[mname]; ok {
		return method
	}
	return mIDK
}

func (rt *router) route(c C, w http.ResponseWriter, r *http.Request) {
	m := httpMethod(r.Method)
	var methods method
	for _, route := range rt.routes {
		if !strings.HasPrefix(r.URL.Path, route.prefix) ||
			!route.pattern.Match(r, &c, false) {

			continue
		}

		if route.method&m != 0 {
			route.handler.ServeHTTPC(c, w, r)
			return
		} else if route.pattern.Match(r, &c, true) {
			methods |= route.method
		}
	}

	if methods == 0 {
		rt.notFound.ServeHTTPC(c, w, r)
		return
	}

	var methodsList = make([]string, 0)
	for mname, meth := range validMethodsMap {
		if methods&meth != 0 {
			methodsList = append(methodsList, mname)
		}
	}
	sort.Strings(methodsList)

	if c.Env == nil {
		c.Env = map[string]interface{}{
			ValidMethodsKey: methodsList,
		}
	} else {
		c.Env[ValidMethodsKey] = methodsList
	}
	rt.notFound.ServeHTTPC(c, w, r)
}

func (rt *router) handleUntyped(p interface{}, m method, h interface{}) {
	pat := parsePattern(p)
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

/*
Dispatch to the given handler when the pattern matches, regardless of HTTP
method. See the documentation for type Mux for a description of what types are
accepted for pattern and handler.

This method is commonly used to implement sub-routing: an admin application, for
instance, can expose a single handler that is attached to the main Mux by
calling Handle("/admin*", adminHandler) or similar. Note that this function
doesn't strip this prefix from the path before forwarding it on (e.g., the
handler will see the full path, including the "/admin" part), but this
functionality can easily be performed by an extra middleware layer.
*/
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
//
// All GET handlers also transparently serve HEAD requests, since net/http will
// take care of all the fiddly bits for you. If you wish to provide an alternate
// implementation of HEAD, you should add a handler explicitly and place it
// above your GET handler.
func (m *router) Get(pattern interface{}, handler interface{}) {
	m.handleUntyped(pattern, mGET|mHEAD, handler)
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

// Set the fallback (i.e., 404) handler for this mux. See the documentation for
// type Mux for a description of what types are accepted for handler.
//
// As a convenience, the context environment variable "goji.web.validMethods"
// (also available as the constant ValidMethodsKey) will be set to the list of
// HTTP methods that could have been routed had they been provided on an
// otherwise identical request.
func (m *router) NotFound(handler interface{}) {
	m.notFound = parseHandler(handler)
}
