package web

import (
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
const ValidMethodsKey = "goji.web.ValidMethods"

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

func httpMethod(mname string) method {
	if method, ok := validMethodsMap[mname]; ok {
		return method
	}
	return mIDK
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

func (rt *router) getMatch(c *C, w http.ResponseWriter, r *http.Request) Match {
	rm := rt.getMachine()
	if rm == nil {
		rm = rt.compile()
	}

	methods, route := rm.route(c, w, r)
	if route != nil {
		return Match{
			Pattern: route.pattern,
			Handler: route.handler,
		}
	}

	if methods == 0 {
		return Match{Handler: rt.notFound}
	}

	var methodsList = make([]string, 0)
	for mname, meth := range validMethodsMap {
		if methods&meth != 0 {
			methodsList = append(methodsList, mname)
		}
	}
	sort.Strings(methodsList)

	if c.Env == nil {
		c.Env = map[interface{}]interface{}{
			ValidMethodsKey: methodsList,
		}
	} else {
		c.Env[ValidMethodsKey] = methodsList
	}
	return Match{Handler: rt.notFound}
}

func (rt *router) route(c *C, w http.ResponseWriter, r *http.Request) {
	match := GetMatch(*c)
	if match.Handler == nil {
		match = rt.getMatch(c, w, r)
	}
	match.Handler.ServeHTTPC(*c, w, r)
}

func (rt *router) handleUntyped(p PatternType, m method, h HandlerType) {
	rt.handle(ParsePattern(p), m, parseHandler(h))
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

func (rt *router) unhandle(p PatternType, m method) {
	pp := ParsePattern(p).Prefix()

	rt.lock.Lock()
	defer rt.lock.Unlock()

	// Find the route we want to remove
	for i := 0; i < len(rt.routes); i++ {
		r := rt.routes[i]
		if r.prefix <= pp && strings.HasPrefix(r.prefix, pp) && r.method&m != 0 {
			// Found it
			if i == len(rt.routes) {
				// last element, so reslice up to it
				rt.routes = rt.routes[:i]
			} else if i >= 0 {
				// use some append magic to create a pair of slices around the removed element
				rt.routes = append(rt.routes[:i], rt.routes[i+1:]...)
				i-- // decrement, so we don't overincrement
			}
		}
	}
	rt.setMachine(nil)
}
