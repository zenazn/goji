package web

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"code.google.com/p/go.net/context"
)

type method int

type methodSet int

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
	// If the request satisfies the pattern the new context is returned by run
	Match(r *http.Request, ctx context.Context) (context.Context, bool)
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

type netHTTPWrap func(w http.ResponseWriter, r *http.Request)

func (h netHTTPWrap) ServeHTTPC(c context.Context, w http.ResponseWriter, r *http.Request) {
	h(w, r)
}

func parseHandler(h interface{}) Handler {
	switch f := h.(type) {
	case Handler:
		return f
	case http.Handler:
		return netHTTPWrap(f.ServeHTTP)
	case func(c context.Context, w http.ResponseWriter, r *http.Request):
		return HandlerFunc(f)
	case func(w http.ResponseWriter, r *http.Request):
		return netHTTPWrap(f)
	default:
		log.Panicf("Unknown handler type %T. Expected a web.Handler, "+
			"a http.Handler, or a function with signature func(context.Context, "+
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

func matchRoute(route route, m method, ms *methodSet, r *http.Request, c context.Context) (context.Context, bool) {
	nc, ok := route.pattern.Match(r, c)
	if !ok {
		return c, false
	}

	if m == mOPTIONS {
		*ms |= methodSet(route.method)
	}

	if route.method&m != 0 {
		return nc, true
	}
	return nc, false
}

func (rm routeMachine) route(c context.Context, w http.ResponseWriter, r *http.Request) (methodSet, bool) {
	m := httpMethod(r.Method)
	var methods methodSet
	p := r.URL.Path

	if len(rm.sm) == 0 {
		return methods, false
	}

	var i int
	for {
		sm := rm.sm[i].mode
		if sm&smSetCursor != 0 {
			si := rm.sm[i].i
			p = r.URL.Path[si:]
			i++
			continue
		}

		length := int(sm & smLengthMask)
		match := false
		if length <= len(p) {
			bs := rm.sm[i].bs
			switch length {
			case 3:
				if p[2] != bs[2] {
					break
				}
				fallthrough
			case 2:
				if p[1] != bs[1] {
					break
				}
				fallthrough
			case 1:
				if p[0] != bs[0] {
					break
				}
				fallthrough
			case 0:
				p = p[length:]
				match = true
			}
		}

		if match && sm&smRoute != 0 {
			si := rm.sm[i].i
			if c, ok := matchRoute(rm.routes[si], m, &methods, r, c); ok {
				rm.routes[si].handler.ServeHTTPC(c, w, r)
				return 0, true
			}
			i++
		} else if (match && sm&smJumpOnMatch != 0) ||
			(!match && sm&smJumpOnMatch == 0) {

			if sm&smFail != 0 {
				return methods, false
			}
			i = int(rm.sm[i].i)
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
func (rt *router) Compile() *routeMachine {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	sm := routeMachine{
		sm:     compile(rt.routes),
		routes: rt.routes,
	}
	rt.setMachine(&sm)
	return &sm
}

func (rt *router) route(c context.Context, w http.ResponseWriter, r *http.Request) {
	rm := rt.getMachine()
	if rm == nil {
		rm = rt.Compile()
	}

	ms, ok := rm.route(c, w, r)
	if ok {
		return
	}

	if ms != 0 {
		c = context.WithValue(c, validMethodsKey, ms)
	}

	rt.notFound.ServeHTTPC(c, w, r)
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
