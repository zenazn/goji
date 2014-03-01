package param

import (
	"net/url"
	"testing"
)

type Crazy struct {
	A     *Crazy
	B     *Crazy
	Value int
	Slice []int
	Map   map[string]Crazy
}

func TestCrazy(t *testing.T) {
	t.Parallel()

	c := Crazy{}
	err := Parse(url.Values{
		"A[B][B][A][Value]":       {"1"},
		"B[A][A][Slice][]":        {"3", "1", "4"},
		"B[Map][hello][A][Value]": {"8"},
		"A[Value]":                {"2"},
		"A[Slice][]":              {"9", "1", "1"},
		"Value":                   {"42"},
	}, &c)
	if err != nil {
		t.Error("Error parsing craziness: ", err)
	}

	// Exhaustively checking everything here is going to be a huge pain, so
	// let's just hope for the best, pretend NPEs don't exist, and hope that
	// this test covers enough stuff that it's actually useful.
	assertEqual(t, "c.A.B.B.A.Value", 1, c.A.B.B.A.Value)
	assertEqual(t, "c.A.Value", 2, c.A.Value)
	assertEqual(t, "c.Value", 42, c.Value)
	assertEqual(t, `c.B.Map["hello"].A.Value`, 8, c.B.Map["hello"].A.Value)

	assertEqual(t, "c.A.B.B.B", (*Crazy)(nil), c.A.B.B.B)
	assertEqual(t, "c.A.B.A", (*Crazy)(nil), c.A.B.A)
	assertEqual(t, "c.A.A", (*Crazy)(nil), c.A.A)

	if c.Slice != nil || c.Map != nil {
		t.Error("Map and Slice should not be set")
	}

	sl := c.B.A.A.Slice
	if len(sl) != 3 || sl[0] != 3 || sl[1] != 1 || sl[2] != 4 {
		t.Error("Something is wrong with c.B.A.A.Slice")
	}
	sl = c.A.Slice
	if len(sl) != 3 || sl[0] != 9 || sl[1] != 1 || sl[2] != 1 {
		t.Error("Something is wrong with c.A.Slice")
	}
}
