package param

import (
	"errors"
	"fmt"
	"log"
)

// Testing log.Fatal in tests is... not a thing. Allow tests to stub it out.
var pebkacTesting bool

const errPrefix = "param/parse: "
const yourFault = " This is a bug in your use of the param library."

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
