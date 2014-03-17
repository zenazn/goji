package web

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type regexpPattern struct {
	re    *regexp.Regexp
	names []string
}

func (p regexpPattern) Prefix() string {
	prefix, _ := p.re.LiteralPrefix()
	return prefix
}
func (p regexpPattern) Match(r *http.Request, c *C) bool {
	matches := p.re.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		return false
	}

	if c.UrlParams == nil && len(matches) > 0 {
		c.UrlParams = make(map[string]string, len(matches)-1)
	}
	for i := 1; i < len(matches); i++ {
		c.UrlParams[p.names[i]] = matches[i]
	}
	return true
}

func parseRegexpPattern(re *regexp.Regexp) regexpPattern {
	rnames := re.SubexpNames()
	// We have to make our own copy since package regexp forbids us
	// from scribbling over the slice returned by SubexpNames().
	names := make([]string, len(rnames))
	for i, rname := range rnames {
		if rname == "" {
			rname = fmt.Sprintf("$%d", i)
		}
		names[i] = rname
	}
	return regexpPattern{
		re:    re,
		names: names,
	}
}

type stringPattern struct {
	raw      string
	pats     []string
	literals []string
	isPrefix bool
}

func (s stringPattern) Prefix() string {
	return s.literals[0]
}

func (s stringPattern) Match(r *http.Request, c *C) bool {
	path := r.URL.Path
	matches := make([]string, len(s.pats))
	for i := 0; i < len(s.pats); i++ {
		if !strings.HasPrefix(path, s.literals[i]) {
			return false
		}
		path = path[len(s.literals[i]):]

		m := strings.IndexRune(path, '/')
		if m == -1 {
			m = len(path)
		}
		if m == 0 {
			// Empty strings are not matches, otherwise routes like
			// "/:foo" would match the path "/"
			return false
		}
		matches[i] = path[:m]
		path = path[m:]
	}
	// There's exactly one more literal than pat.
	if s.isPrefix {
		if strings.HasPrefix(path, s.literals[len(s.pats)]) {
			return false
		}
	} else {
		if path != s.literals[len(s.pats)] {
			return false
		}
	}

	if c.UrlParams == nil && len(matches) > 0 {
		c.UrlParams = make(map[string]string, len(matches)-1)
	}
	for i, match := range matches {
		c.UrlParams[s.pats[i]] = match
	}
	return true
}

func parseStringPattern(s string, isPrefix bool) stringPattern {
	matches := patternRe.FindAllStringSubmatchIndex(s, -1)
	pats := make([]string, len(matches))
	literals := make([]string, len(matches)+1)
	n := 0
	for i, match := range matches {
		a, b := match[2], match[3]
		literals[i] = s[n : a-1] // Need to leave off the colon
		pats[i] = s[a:b]
		n = b
	}
	literals[len(matches)] = s[n:]
	return stringPattern{
		raw:      s,
		pats:     pats,
		literals: literals,
		isPrefix: isPrefix,
	}
}
