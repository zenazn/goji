package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/zenazn/goji/web"
)

func testUrlQuery(r *http.Request, f func(*web.C, http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	var c web.C

	h := func(w http.ResponseWriter, r *http.Request) {
		f(&c, w, r)
	}
	m := UrlQuery(&c, http.HandlerFunc(h))
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)

	return w
}

func TestUrlQuery(t *testing.T) {
	type testcase struct {
		url            string
		expectedParams url.Values
	}

	// we're not testing url.Query() here, but rather that the results of the query
	// appear in the context
	testcases := []testcase{
		testcase{"/", url.Values{}},
		testcase{"/?a=1&b=2&a=3", url.Values{"a": []string{"1", "3"}, "b": []string{"2"}}},
		testcase{"/?x=1&y=2&z=3#freddyishere", url.Values{"x": []string{"1"}, "y": []string{"2"}, "z": []string{"3"}}},
	}

	for _, tc := range testcases {
		r, _ := http.NewRequest("GET", tc.url, nil)
		testUrlQuery(r,
			func(c *web.C, w http.ResponseWriter, r *http.Request) {
				params := c.Env[UrlQueryKey].(url.Values)
				if !reflect.DeepEqual(params, tc.expectedParams) {
					t.Errorf("GET %s, UrlQuery middleware found %v, should be %v", tc.url, params, tc.expectedParams)
				}

				w.Write([]byte{'h', 'i'})
			},
		)
	}
}
