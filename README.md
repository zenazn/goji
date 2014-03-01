Goji
====

Goji is a minimalistic web framework inspired by Sinatra.


Philosophy
----------

Most of the design decisions in Goji can be traced back to the fundamental
philosopy that the Go standard library got things Mostly Right, and if it
didn't, it at least is good enough that it's not worth fighting.

Therefore, Goji leans heavily on the standard library, and in particular its
interfaces and idioms. You can expect to be able to use most of Goji in exactly
the manner you would use a comparable standard library function, and have it
function in exactly the way you would expect.

Also in this vein, Goji makes use of Go's `flag` package, and in particular the
default global flag set. Third party packages that have global state and squat
on global namespaces is something to be suspicious of, but the `flag` package is
also the closest thing Go has to a unified configuration API, and when used
tastefully it can make everyone's lives a bit easier. Wherever possible, the use
of these flags is opt-out, at the cost of additional complexity for the user.

Goji also makes an attempt to not be magical -- explicit is better than
implicit. Goji does make use of reflection and `interface{}`, but only when an
API would be impossible or cumbersome without it.
