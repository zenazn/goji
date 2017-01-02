package middleware

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestBasicAuthAuthenticate(t *testing.T) {
	// Provide a minimal test implementation.
	b := &basicAuth{
		opts: &AuthOptions{
			Realm:    "Restricted",
			User:     "test",
			Password: "test123",
		},
	}

	r := &http.Request{}
	r.Method = "GET"

	// Provide auth data, but no Authorization header
	if b.authenticate(r) != false {
		t.Fatal("No Authorization header supplied.")
	}

	// Initialise the map for HTTP headers
	r.Header = http.Header(make(map[string][]string))

	// Set a malformed/bad header
	r.Header.Set("Authorization", "    Basic")
	if b.authenticate(r) != false {
		t.Fatal("Malformed Authorization header supplied.")
	}

    // Test correct credentials
    auth := base64.StdEncoding.EncodeToString([]byte(b.opts.User + ":" + b.opts.Password))
	r.Header.Set("Authorization", "Basic " + auth)
	if b.authenticate(r) != true {
		t.Fatal("Failed on correct credentials")
	}
}
