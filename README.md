Goji
====

Goji is a minimalistic web framework inspired by Sinatra.

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
