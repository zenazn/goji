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
	machine  *routeMachine
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
	// decision. Match should not modify either argument, and since it will
	// potentially be called several times over the course of matching a
	// request, it should be reasonably efficient.
	Match(r *http.Request, c *C) bool
	// Run the pattern on the request and context, modifying the context as
	// necessary to bind URL parameters or other parsed state.
	Run(r *http.Request, c *C)
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

type netHTTPWrap struct {
	http.Handler
}

func (h netHTTPWrap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(w, r)
}
func (h netHTTPWrap) ServeHTTPC(c C, w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(w, r)
}

func parseHandler(h interface{}) Handler {
	switch h.(type) {
	case Handler:
		return h.(Handler)
	case http.Handler:
		return netHTTPWrap{h.(http.Handler)}
	case func(c C, w http.ResponseWriter, r *http.Request):
		f := h.(func(c C, w http.ResponseWriter, r *http.Request))
		return HandlerFunc(f)
	case func(w http.ResponseWriter, r *http.Request):
		f := h.(func(w http.ResponseWriter, r *http.Request))
		return netHTTPWrap{http.HandlerFunc(f)}
	default:
		log.Fatalf("Unknown handler type %v. Expected a web.Handler, "+
			"a http.Handler, or a function with signature func(C, "+
			"http.ResponseWriter, *http.Request) or "+
			"func(http.ResponseWriter, *http.Request)", h)
	}
	panic("log.Fatalf does not return")
}

func httpMethod(mname string) method {
	if method, ok := validMethodsMap[mname]; ok {
		return method
	}
	return mIDK
}

type routeMachine struct {
	sm     stateMachine
	routes []route
}

func matchRoute(route route, m method, ms *method, r *http.Request, c *C) bool {
	if !route.pattern.Match(r, c) {
		return false
	}
	*ms |= route.method

	if route.method&m != 0 {
		route.pattern.Run(r, c)
		return true
	}
	return false
}

func (rm routeMachine) route(c *C, w http.ResponseWriter, r *http.Request) (method, bool) {
	m := httpMethod(r.Method)
	var methods method
	p := r.URL.Path

	if len(rm.sm) == 0 {
		return methods, false
	}

	var i int
	for {
		s := rm.sm[i]
		if s.mode&smSetCursor != 0 {
			p = r.URL.Path[s.i:]
			i++
			continue
		}

		length := int(s.mode & smLengthMask)
		match := length <= len(p)
		for j := 0; match && j < length; j++ {
			match = match && p[j] == s.bs[j]
		}

		if match {
			p = p[length:]
		}

		if match && s.mode&smRoute != 0 {
			if matchRoute(rm.routes[s.i], m, &methods, r, c) {
				rm.routes[s.i].handler.ServeHTTPC(*c, w, r)
				return 0, true
			} else {
				i++
			}
		} else if (match && s.mode&smJumpOnMatch != 0) ||
			(!match && s.mode&smJumpOnMatch == 0) {

			if s.mode&smFail != 0 {
				return methods, false
			}
			i = int(s.i)
		} else {
			i++
		}
	}

	return methods, false
}

// Compile the list of routes into bytecode. This only needs to be done once
// after all the routes have been added, and will be called automatically for
// you (at some performance cost on the first request) if you do not call it
// explicitly.
func (rt *router) Compile() {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	sm := routeMachine{
		sm:     compile(rt.routes),
		routes: rt.routes,
	}
	rt.setMachine(&sm)
}

func (rt *router) route(c *C, w http.ResponseWriter, r *http.Request) {
	if rt.machine == nil {
		rt.Compile()
	}

	methods, ok := rt.getMachine().route(c, w, r)
	if ok {
		return
	}

	if methods == 0 {
		rt.notFound.ServeHTTPC(*c, w, r)
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
	rt.notFound.ServeHTTPC(*c, w, r)
}

func (rt *router) handleUntyped(p interface{}, m method, h interface{}) {
	pat := parsePattern(p)
	rt.handle(pat, m, parseHandler(h))
}

func (rt *router) handle(p Pattern, m method, h Handler) {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	// Calculate the sorted insertion point, because there's no reason to do
	// swapping hijinks if we're already making a copy. We need to use
	// bubble sort because we can only compare adjacent elements.
	pp := p.Prefix()
	var i int
	for i = len(rt.routes); i > 0; i-- {
		rip := rt.routes[i-1].prefix
		if rip <= pp || strings.HasPrefix(rip, pp) {
			break
		}
	}

	newRoutes := make([]route, len(rt.routes)+1)
	copy(newRoutes, rt.routes[:i])
	newRoutes[i] = route{
		prefix:  pp,
		method:  m,
		pattern: p,
		handler: h,
	}
	copy(newRoutes[i+1:], rt.routes[i:])

	rt.setMachine(nil)
	rt.routes = newRoutes
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
func (rt *router) Handle(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mALL, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// CONNECT. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (rt *router) Connect(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mCONNECT, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// DELETE. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (rt *router) Delete(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mDELETE, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// GET. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
//
// All GET handlers also transparently serve HEAD requests, since net/http will
// take care of all the fiddly bits for you. If you wish to provide an alternate
// implementation of HEAD, you should add a handler explicitly and place it
// above your GET handler.
func (rt *router) Get(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mGET|mHEAD, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// HEAD. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (rt *router) Head(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mHEAD, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// OPTIONS. See the documentation for type Mux for a description of what types
// are accepted for pattern and handler.
func (rt *router) Options(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mOPTIONS, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// PATCH. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (rt *router) Patch(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mPATCH, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// POST. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (rt *router) Post(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mPOST, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// PUT. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (rt *router) Put(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mPUT, handler)
}

// Dispatch to the given handler when the pattern matches and the HTTP method is
// TRACE. See the documentation for type Mux for a description of what types are
// accepted for pattern and handler.
func (rt *router) Trace(pattern interface{}, handler interface{}) {
	rt.handleUntyped(pattern, mTRACE, handler)
}

// Set the fallback (i.e., 404) handler for this mux. See the documentation for
// type Mux for a description of what types are accepted for handler.
//
// As a convenience, the context environment variable "goji.web.validMethods"
// (also available as the constant ValidMethodsKey) will be set to the list of
// HTTP methods that could have been routed had they been provided on an
// otherwise identical request.
func (rt *router) NotFound(handler interface{}) {
	rt.notFound = parseHandler(handler)
}
