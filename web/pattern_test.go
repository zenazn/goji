package web

import (
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"code.google.com/p/go.net/context"
)

func pt(url string, match bool, params map[string]string) patternTest {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	return patternTest{
		r:     req,
		match: match,
		c:     context.Background(),
		cout:  params,
	}
}

type patternTest struct {
	r     *http.Request
	match bool
	c     context.Context
	cout  map[string]string
}

var patternTests = []struct {
	pat    Pattern
	prefix string
	tests  []patternTest
}{
	// Regexp tests
	{parseRegexpPattern(regexp.MustCompile("^/hello$")),
		"/hello", []patternTest{
			pt("/hello", true, nil),
			pt("/hell", false, nil),
			pt("/hello/", false, nil),
			pt("/hello/world", false, nil),
			pt("/world", false, nil),
		}},
	{parseRegexpPattern(regexp.MustCompile("^/hello/(?P<name>[a-z]+)$")),
		"/hello/", []patternTest{
			pt("/hello/world", true, map[string]string{
				"name": "world",
			}),
			pt("/hello/", false, nil),
			pt("/hello/my/love", false, nil),
		}},
	{parseRegexpPattern(regexp.MustCompile(`^/a(?P<a>\d+)/b(?P<b>\d+)/?$`)),
		"/a", []patternTest{
			pt("/a1/b2", true, map[string]string{
				"a": "1",
				"b": "2",
			}),
			pt("/a9001/b007/", true, map[string]string{
				"a": "9001",
				"b": "007",
			}),
			pt("/a/b", false, nil),
			pt("/a", false, nil),
			pt("/squirrel", false, nil),
		}},
	{parseRegexpPattern(regexp.MustCompile(`^/hello/([a-z]+)$`)),
		"/hello/", []patternTest{
			pt("/hello/world", true, map[string]string{
				"$1": "world",
			}),
			pt("/hello/", false, nil),
		}},
	{parseRegexpPattern(regexp.MustCompile("/hello")),
		"/hello", []patternTest{
			pt("/hello", true, nil),
			pt("/hell", false, nil),
			pt("/hello/", true, nil),
			pt("/hello/world", true, nil),
			pt("/world/hello", false, nil),
		}},

	// String pattern tests
	{parseStringPattern("/hello"),
		"/hello", []patternTest{
			pt("/hello", true, nil),
			pt("/hell", false, nil),
			pt("/hello/", false, nil),
			pt("/hello/world", false, nil),
		}},
	{parseStringPattern("/hello/:name"),
		"/hello/", []patternTest{
			pt("/hello/world", true, map[string]string{
				"name": "world",
			}),
			pt("/hell", false, nil),
			pt("/hello/", false, nil),
			pt("/hello/my/love", false, nil),
		}},
	{parseStringPattern("/a/:a/b/:b"),
		"/a/", []patternTest{
			pt("/a/1/b/2", true, map[string]string{
				"a": "1",
				"b": "2",
			}),
			pt("/a", false, nil),
			pt("/a//b/", false, nil),
			pt("/a/1/b/2/3", false, nil),
		}},

	// String prefix tests
	{parseStringPattern("/user/:user*"),
		"/user/", []patternTest{
			pt("/user/bob", true, map[string]string{
				"user": "bob",
			}),
			pt("/user/bob/friends/123", true, map[string]string{
				"user": "bob",
			}),
			pt("/user/", false, nil),
			pt("/user//", false, nil),
		}},
	{parseStringPattern("/user/:user/*"),
		"/user/", []patternTest{
			pt("/user/bob/friends/123", true, map[string]string{
				"user": "bob",
			}),
			pt("/user/bob", false, nil),
			pt("/user/", false, nil),
			pt("/user//", false, nil),
		}},
	{parseStringPattern("/user/:user/friends*"),
		"/user/", []patternTest{
			pt("/user/bob/friends", true, map[string]string{
				"user": "bob",
			}),
			pt("/user/bob/friends/123", true, map[string]string{
				"user": "bob",
			}),
			// This is a little unfortunate
			pt("/user/bob/friends123", true, map[string]string{
				"user": "bob",
			}),
			pt("/user/bob/enemies", false, nil),
		}},
}

func TestPatterns(t *testing.T) {
	t.Parallel()

	for _, pt := range patternTests {
		p := pt.pat.Prefix()
		if p != pt.prefix {
			t.Errorf("Expected prefix %q for %v, got %q", pt.prefix,
				pt.pat, p)
		} else {
			for _, test := range pt.tests {
				runTest(t, pt.pat, test)
			}
		}
	}
}

func runTest(t *testing.T, p Pattern, test patternTest) {
	cout, result := p.Match(test.r, test.c)
	if result != test.match {
		t.Errorf("Expected match(%v, %#v) to return %v", p,
			test.r.URL.Path, test.match)
		return
	}

	if !reflect.DeepEqual(URLParams(cout), test.cout) {
		t.Errorf("Expected a context of %v, instead got %v", test.cout,
			test.c)
	}
}
