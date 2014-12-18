package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zenazn/goji/web"
)

func TestParseJsonInvalidInput(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJson)

	invalid_json := strings.NewReader("{")
	r, err := http.NewRequest("GET", "/", invalid_json)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"application/json"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Reponse code is not '400 - Bad request' for invalid Json")
	}
}

func TestParseJsonNoHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJson)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		_, ok := c.Env[ParsedJsonKey]
		if ok {
			t.Errorf("Env is populated but the Content-Type header is not sent")
		}
	})

	json := strings.NewReader(`{}`)
	r, err := http.NewRequest("GET", "/", json)
	if err != nil {
		t.Fatal(err)
	}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}

func TestParseJsonAlternativeContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJson)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		_, ok := c.Env[ParsedJsonKey]
		if ok {
			t.Errorf("Env is populated but the Content-Type header is not set to application/json")
		}
	})

	json := strings.NewReader(`{}`)
	r, err := http.NewRequest("GET", "/", json)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"multipart/form-data"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}

func TestParseJsonPopulatedEnv(t *testing.T) {
	rr := httptest.NewRecorder()
	s := web.New()
	s.Use(ParseJson)
	s.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		temp, ok := c.Env[ParsedJsonKey]
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

	json := strings.NewReader(`{"email":"xyz","password":"zyx"}`)
	r, err := http.NewRequest("GET", "/", json)
	if err != nil {
		t.Fatal(err)
	}
	r.Header["Content-Type"] = []string{"application/json"}

	s.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("Reponse code is not '200 - OK'")
	}
}
