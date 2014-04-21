package param

import (
	"errors"
	"fmt"
	"log"
)

// TODO: someday it might be nice to throw typed errors instead of weird strings

// Testing log.Fatal in tests is... not a thing. Allow tests to stub it out.
var pebkacTesting bool

const errPrefix = "param/parse: "
const yourFault = " This is a bug in your use of the param library."

// Panic with a formatted error. The param library uses panics to quickly unwind
// the call stack and return a user error
func perr(format string, a ...interface{}) {
	err := errors.New(errPrefix + fmt.Sprintf(format, a...))
	panic(err)
}

// Problem exists between keyboard and chair. This function is used in cases of
// programmer error, i.e. an inappropriate use of the param library, to
// immediately force the program to halt with a hopefully helpful error message.
func pebkac(format string, a ...interface{}) {
	err := errors.New(errPrefix + fmt.Sprintf(format, a...) + yourFault)
	if pebkacTesting {
		panic(err)
	} else {
		log.Fatal(err)
	}
}
