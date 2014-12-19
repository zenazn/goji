package middleware

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/zenazn/goji/web"
)

const ParsedJSONKey = "parsedJSON"

var (
	contentTypeHeader = http.CanonicalHeaderKey("Content-Type")
	applicationJSON   = "application/json"
)

// ParseJSON is a helper middleware that makes writing API based on JSON easier.
//
// ParseJSON parse JSON-encoded data from the HTTP request body and populate
// Env[ParsedJSONKey]. This middleware will parse JSON only if the request have
// a "Content-Type" header with the value "application/json".
//
// The Env[ParsedJSONKey] is interface{}.
func ParseJSON(c *web.C, h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		shouldParseJSON := false

		// search for a "Content-Type: application/json" header
		for name, contents := range r.Header {
			if name != contentTypeHeader {
				// this is not the header we are looking for
				continue
			}
			for _, content := range contents {
				if content != applicationJSON {
					// this is not the content-type we are looking for
					continue
				}
				shouldParseJSON = true
				break
			}
		}

		if shouldParseJSON {
			// do not forget to close the Body
			defer r.Body.Close()

			// read the whole request Body
			rawJSON, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// use a generic interface{} to store the parsed data
			var parsedJSON interface{}

			// parse the JSON-encoded data
			err = json.Unmarshal(rawJSON, &parsedJSON)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// populate the Env parameter
			if c.Env == nil {
				c.Env = make(map[string]interface{})
			}
			c.Env[ParsedJSONKey] = parsedJSON
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(handler)
}
