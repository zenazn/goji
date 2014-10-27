package web

import "net/http"

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
