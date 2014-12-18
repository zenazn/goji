package middleware

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/zenazn/goji/web"
)

const ParsedJsonKey = "parsedJson"

var (
	contentTypeHeader = http.CanonicalHeaderKey("Content-Type")
	applicationJson   = "application/json"
)

// ParseJson is a helper middleware that makes writing API based on json easier.
//
// ParseJson parse json-encoded data from the HTTP request body and populate
// Env[ParsedJsonKey]. This middleware will parse json only if the request have
// a "Content-Type" header with the value "application/json".
//
// The Env[ParsedJsonKey] is map[string]interface{}.
func ParseJson(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		shouldParseJson := false

		// search for a "Content-Type: application/json" header
		for name, contents := range r.Header {
			if name != contentTypeHeader {
				// this is not the header we are looking for
				continue
			}
			for _, content := range contents {
				if content != applicationJson {
					// this is not the content-type we are looking for
					continue
				}
				shouldParseJson = true
				break
			}
		}

		if shouldParseJson {
			// do not forget to close the Body
			defer r.Body.Close()

			// read the whole request Body
			rawJson, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// use a generic interface{} to store the parsed data
			var parsedJson interface{}

			// parse the Json-encoded data
			err = json.Unmarshal(rawJson, &parsedJson)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// populate the Env parameter
			if c.Env == nil {
				c.Env = make(map[string]interface{})
			}
			c.Env[ParsedJsonKey] = parsedJson
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
