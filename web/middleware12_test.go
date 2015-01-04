// +build !go1.3

package web

import "testing"

// These tests were pretty sketchtacular to start with, but they aren't even
// guaranteed to pass with Go 1.3's sync.Pool. Let's keep them here for now; if
// they start spuriously failing later we can delete them outright.

func TestCaching(t *testing.T) {
	ch := make(chan string)
	st := makeStack(ch)
	cs1 := st.alloc()
	cs2 := st.alloc()
	if cs1 == cs2 {
		t.Fatal("cs1 and cs2 are the same")
	}
	st.release(cs2)
	cs3 := st.alloc()
	if cs2 != cs3 {
		t.Fatalf("Expected cs2 to equal cs3")
	}
	st.release(cs1)
	st.release(cs3)
	cs4 := st.alloc()
	cs5 := st.alloc()
	if cs4 != cs1 {
		t.Fatal("Expected cs4 to equal cs1")
	}
	if cs5 != cs3 {
		t.Fatal("Expected cs5 to equal cs3")
	}
}

func TestInvalidation(t *testing.T) {
	ch := make(chan string)
	st := makeStack(ch)
	cs1 := st.alloc()
	cs2 := st.alloc()
	st.release(cs1)
	st.invalidate()
	cs3 := st.alloc()
	if cs3 == cs1 {
		t.Fatal("Expected cs3 to be fresh, instead got cs1")
	}
	st.release(cs2)
	cs4 := st.alloc()
	if cs4 == cs2 {
		t.Fatal("Expected cs4 to be fresh, instead got cs2")
	}
}
