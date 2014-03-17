Goji
====

Goji is a minimalistic web framework inspired by Sinatra. [Godoc][doc].

[doc]: http://godoc.org/github.com/zenazn/goji

Example
-------

```go
package main

import (
        "fmt"
        "net/http"

        "github.com/zenazn/goji"
        "github.com/zenazn/goji/web"
)

func hello(c web.C, w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, %s!", c.UrlParams["name"])
}

func main() {
        goji.Get("/hello/:name", hello)
        goji.Serve()
}
```


Features
--------

* Compatible with `net/http`
* URL patterns (both Sinatra style `/foo/:bar` patterns and regular expressions)
* Reconfigurable middleware stack
* Context/environment objects threaded through middleware and handlers
* Automatic support for [Einhorn][einhorn], systemd, and [more][bind]
* [Graceful shutdown][graceful], and zero-downtime graceful reload when combined
  with Einhorn.
* Ruby on Rails / jQuery style [parameter parsing][param]

[einhorn]: https://github.com/stripe/einhorn
[bind]: http://godoc.org/github.com/zenazn/goji/bind
[graceful]: http://godoc.org/github.com/zenazn/goji/graceful
[param]: http://godoc.org/github.com/zenazn/goji/param


Todo
----

Goji probably deserves a bit more love before anyone actually tries to use it.
Things that need doing include:

* Support for omitting trailing slashes on routes which include them.
* Tests for `goji/web`. There are currently no tests. This probably means
  `goji/web` is made of bugs. I'm sorry.
* Standard middleware implementations. I'm currently thinking:
  * Request ID assigner: injects a request ID into the environment.
  * Request logger: logs requests as they come in. Preferrably with request IDs
    and maybe even with colors.
  * Request timer: maybe part of the request logger. Time how long each request
    takes and print it. Maybe with color.
  * Error handler: recovers from panics, does something nice with the output.
  * Healthcheck endpoint: Always returns OK.
