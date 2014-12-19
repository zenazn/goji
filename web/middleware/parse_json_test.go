package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zenazn/goji/web"
)

func TestParseJSONInvalidInput(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJSON)

	invalidJSON := strings.NewReader("{")
	r, err := http.NewRequest("GET", "/", invalidJSON)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"application/json"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Reponse code is not '400 - Bad request' for invalid JSON")
	}
}

func TestParseJSONNoHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJSON)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		_, ok := c.Env[ParsedJSONKey]
		if ok {
			t.Errorf("Env is populated but the Content-Type header is not sent")
		}
	})

	validJSON := strings.NewReader(`{}`)
	r, err := http.NewRequest("GET", "/", validJSON)
	if err != nil {
		t.Fatal(err)
	}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}

func TestParseJSONBadContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJSON)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		_, ok := c.Env[ParsedJSONKey]
		if ok {
			t.Errorf("Env is populated but the Content-Type header is not set to application/json")
		}
	})

	validJSON := strings.NewReader(`[]`)
	r, err := http.NewRequest("GET", "/", validJSON)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"multipart/form-data"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}

func TestParseJSONPopulatedEnv(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJSON)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		temp, ok := c.Env[ParsedJSONKey]
		if !ok {
			t.Errorf("Env is not populated")
		}

		data := temp.(map[string]interface{})

		email, ok := data["email"]
		if !ok {
			t.Errorf("email is missing")
		}
		if email != "xyz" {
			t.Errorf("email is not 'xyz': %s", email)
		}

		password, ok := data["password"]
		if !ok {
			t.Errorf("password is missing")
		}
		if password != "zyx" {
			t.Errorf("password is not 'zyx': %s", password)
		}
	})

	validJSON := strings.NewReader(`{"email":"xyz","password":"zyx"}`)
	r, err := http.NewRequest("GET", "/", validJSON)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"application/json"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}
