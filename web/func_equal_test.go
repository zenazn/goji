package web

import (
	"testing"
)

// To tell you the truth, I'm not actually sure how many of these cases are
// needed. Presumably someone with more patience than I could comb through
// http://golang.org/s/go11func and figure out what all the different cases I
// ought to test are, but I think this test includes all the cases I care about
// and is at least reasonably thorough.

func a() string {
	return "A"
}
func b() string {
	return "B"
}
func mkFn(s string) func() string {
	return func() string {
		return s
	}
}

var c = mkFn("C")
var d = mkFn("D")
var e = a
var f = c
var g = mkFn("D")

type Type string

func (t *Type) String() string {
	return string(*t)
}

var t1 = Type("hi")
var t2 = Type("bye")
var t1f = t1.String
var t2f = t2.String

var funcEqualTests = []struct {
	a, b   func() string
	result bool
}{
	{a, a, true},
	{a, b, false},
	{b, b, true},
	{a, c, false},
	{c, c, true},
	{c, d, false},
	{a, e, true},
	{a, f, false},
	{c, f, true},
	{e, f, false},
	{d, g, false},
	{t1f, t1f, true},
	{t1f, t2f, false},
}

func TestFuncEqual(t *testing.T) {
	t.Parallel()

	for _, test := range funcEqualTests {
		r := funcEqual(test.a, test.b)
		if r != test.result {
			t.Errorf("funcEqual(%v, %v) should have been %v",
				test.a, test.b, test.result)
		}
	}
	h := mkFn("H")
	i := h
	j := mkFn("H")
	k := a
	if !funcEqual(h, i) {
		t.Errorf("h and i should have been equal")
	}
	if funcEqual(h, j) {
		t.Errorf("h and j should not have been equal")
	}
	if !funcEqual(a, k) {
		t.Errorf("a and k should have been equal")
	}
}
