package web

import (
	"log"
	"net/http"
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
			if matchRoute(rm.routes[si], m, &methods, r, c) {
				rm.routes[si].handler.ServeHTTPC(*c, w, r)
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

func (rt *router) compile() *routeMachine {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	sm := routeMachine{
		sm:     compile(rt.routes),
		routes: rt.routes,
	}
	rt.setMachine(&sm)
	return &sm
}

func (rt *router) route(c *C, w http.ResponseWriter, r *http.Request) {
	rm := rt.getMachine()
	if rm == nil {
		rm = rt.compile()
	}

	methods, ok := rm.route(c, w, r)
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
