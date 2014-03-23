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

