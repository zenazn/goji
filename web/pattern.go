package web

import (
	"log"
	"net/http"
	"regexp"
)

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
	switch v := p.(type) {
	case Pattern:
		return v
	case *regexp.Regexp:
		return parseRegexpPattern(v)
	case string:
		return parseStringPattern(v)
	default:
		log.Fatalf("Unknown pattern type %v. Expected a web.Pattern, "+
			"regexp.Regexp, or a string.", p)
	}
	panic("log.Fatalf does not return")
}
