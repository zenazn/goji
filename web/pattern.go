package web

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"regexp/syntax"
	"strings"
)

type regexpPattern struct {
	re     *regexp.Regexp
	prefix string
	names  []string
}

func (p regexpPattern) Prefix() string {
	return p.prefix
}
func (p regexpPattern) Match(r *http.Request, c *C) bool {
	return p.match(r, c, false)
}
func (p regexpPattern) Run(r *http.Request, c *C) {
	p.match(r, c, false)
}

func (p regexpPattern) match(r *http.Request, c *C, dryrun bool) bool {
	matches := p.re.FindStringSubmatch(r.URL.Path)
	if matches == nil || len(matches) == 0 {
		return false
	}

	if c == nil || dryrun || len(matches) == 1 {
		return true
	}

	if c.URLParams == nil {
		c.URLParams = make(map[string]string, len(matches)-1)
	}
	for i := 1; i < len(matches); i++ {
		c.URLParams[p.names[i]] = matches[i]
	}
	return true
}

func (p regexpPattern) String() string {
	return fmt.Sprintf("regexpPattern(%v)", p.re)
}

/*
I'm sorry, dear reader. I really am.

The problem here is to take an arbitrary regular expression and:
1. return a regular expression that is just like it, but left-anchored,
   preferring to return the original if possible.
2. determine a string literal prefix that all matches of this regular expression
   have, much like regexp.Regexp.Prefix(). Unfortunately, Prefix() does not work
   in the presence of anchors, so we need to write it ourselves.

What this actually means is that we need to sketch on the internals of the
standard regexp library to forcefully extract the information we want.

Unfortunately, regexp.Regexp hides a lot of its state, so our abstraction is
going to be pretty leaky. The biggest leak is that we blindly assume that all
regular expressions are perl-style, not POSIX. This is probably Mostly True, and
I think most users of the library probably won't be able to notice.
*/
func sketchOnRegex(re *regexp.Regexp) (*regexp.Regexp, string) {
	rawRe := re.String()
	sRe, err := syntax.Parse(rawRe, syntax.Perl)
	if err != nil {
		log.Printf("WARN(web): unable to parse regexp %v as perl. "+
			"This route might behave unexpectedly.", re)
		return re, ""
	}
	sRe = sRe.Simplify()
	p, err := syntax.Compile(sRe)
	if err != nil {
		log.Printf("WARN(web): unable to compile regexp %v. This "+
			"route might behave unexpectedly.", re)
		return re, ""
	}
	if p.StartCond()&syntax.EmptyBeginText == 0 {
		// I hope doing this is always legal...
		newRe, err := regexp.Compile(`\A` + rawRe)
		if err != nil {
			log.Printf("WARN(web): unable to create a left-"+
				"anchored regexp from %v. This route might "+
				"behave unexpectedly", re)
			return re, ""
		}
		re = newRe
	}

	// Run the regular expression more or less by hand :(
	pc := uint32(p.Start)
	atStart := true
	i := &p.Inst[pc]
	var buf bytes.Buffer
Sadness:
	for {
		switch i.Op {
		case syntax.InstEmptyWidth:
			if !atStart {
				break Sadness
			}
		case syntax.InstCapture, syntax.InstNop:
			// nop!
		case syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny,
			syntax.InstRuneAnyNotNL:

			atStart = false
			if len(i.Rune) != 1 ||
				syntax.Flags(i.Arg)&syntax.FoldCase != 0 {
				break Sadness
			}
			buf.WriteRune(i.Rune[0])
		default:
			break Sadness
		}
		pc = i.Out
		i = &p.Inst[pc]
	}
	return re, buf.String()
}

func parseRegexpPattern(re *regexp.Regexp) regexpPattern {
	re, prefix := sketchOnRegex(re)
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
		re:     re,
		prefix: prefix,
		names:  names,
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
	return s.match(r, c, true)
}
func (s stringPattern) Run(r *http.Request, c *C) {
	s.match(r, c, false)
}
func (s stringPattern) match(r *http.Request, c *C, dryrun bool) bool {
	path := r.URL.Path
	var matches map[string]string
	if !dryrun && len(s.pats) > 0 {
		matches = make(map[string]string, len(s.pats))
	}
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
		if !dryrun {
			matches[s.pats[i]] = path[:m]
		}
		path = path[m:]
	}
	// There's exactly one more literal than pat.
	if s.isPrefix {
		if !strings.HasPrefix(path, s.literals[len(s.pats)]) {
			return false
		}
	} else {
		if path != s.literals[len(s.pats)] {
			return false
		}
	}

	if c == nil || dryrun {
		return true
	}

	if c.URLParams == nil {
		c.URLParams = matches
	} else {
		for k, v := range matches {
			c.URLParams[k] = v
		}
	}
	return true
}

func (s stringPattern) String() string {
	return fmt.Sprintf("stringPattern(%q, %v)", s.raw, s.isPrefix)
}

var patternRe = regexp.MustCompile(`/:([^/]+)`)

func parseStringPattern(s string) stringPattern {
	var isPrefix bool
	// Routes that end in an asterisk ("*") are prefix routes
	if len(s) > 0 && s[len(s)-1] == '*' {
		s = s[:len(s)-1]
		isPrefix = true
	}

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
