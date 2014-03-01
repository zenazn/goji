package param

import (
	"net/url"
	"strings"
	"testing"
)

type Bad struct {
	Unknown interface{}
}

type Bad2 struct {
	Unknown *interface{}
}

type Bad3 struct {
	BadMap map[int]int
}

// These tests are not parallel so we can frob pebkac behavior in an isolated
// way

func assertPebkac(t *testing.T, err error) {
	if err == nil {
		t.Error("Expected PEBKAC error message")
	} else if !strings.HasSuffix(err.Error(), yourFault) {
		t.Errorf("Expected PEBKAC error, but got: %v", err)
	}
}

func TestBadInputs(t *testing.T) {
	pebkacTesting = true

	err := Parse(url.Values{"Unknown": {"4"}}, Bad{})
	assertPebkac(t, err)

	b := &Bad{}
	err = Parse(url.Values{"Unknown": {"4"}}, &b)
	assertPebkac(t, err)

	pebkacTesting = false
}

func TestBadTypes(t *testing.T) {
	pebkacTesting = true

	err := Parse(url.Values{"Unknown": {"4"}}, &Bad{})
	assertPebkac(t, err)

	err = Parse(url.Values{"Unknown": {"4"}}, &Bad2{})
	assertPebkac(t, err)

	err = Parse(url.Values{"BadMap[llama]": {"4"}}, &Bad3{})
	assertPebkac(t, err)

	pebkacTesting = false
}
