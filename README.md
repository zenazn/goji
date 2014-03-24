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

Goji also includes a [sample application][sample] in the `example` folder which
was artificially constructed to show off all of Goji's features. Check it out!

[sample]: https://github.com/zenazn/goji/tree/master/example


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


Is it any good?
---------------

Maybe!

There are [plenty][revel] of [other][gorilla] [good][pat] [Go][martini]
[web][gocraft] [frameworks][tiger] out there. Goji is by no means especially
novel, nor is it uniquely good. The primary difference between Goji and other
frameworks--and the primary reason I think Goji is any good--is its philosophy:

Goji first of all attempts to be simple. It is of the Sinatra school of web
framework design, and not the Rails one. If you want me to tell you what
directory you should put your models in, or if you want built-in flash sessions,
you won't have a good time with Goji.

Secondly, Goji attempts to be composable. It is fully composable with net/http,
and can be used as a `http.Handler`, or can serve arbitrary `http.Handler`s. At
least a few HTTP frameworks share this property, and is not particularly novel.
The more interesting property in my mind is that Goji is fully composable with
itself: it defines an interface (`web.Handler`) which is both fully compatible
with `http.Handler` and allows Goji to perform a "protocol upgrade" of sorts
when it detects that it is talking to itself (or another `web.Handler`
compatible component). `web.Handler` is at the core of Goji's interfaces and is
what allows it to share request contexts across unrelated objects.

Third, Goji is not magic. One of my favorite existing frameworks is
[Martini][martini], but I rejected it in favor of building Goji because I
thought it was too magical. Goji's web package does not use reflection at all,
which is not in itself a sign of API quality, but to me at least seems to
suggest it.

Finally, Goji gives you enough rope to hang yourself with. One of my other
favorite libraries, [pat][pat], implements Sinatra-like routing in a
particularly elegant way, but because of its reliance on net/http's interfaces,
doesn't allow programmers to thread their own state through the request handling
process. Implementing arbitrary context objects was one of the primary
motivations behind abandoning pat to write Goji.

[revel]: http://revel.github.io/
[gorilla]: http://www.gorillatoolkit.org/
[pat]: https://github.com/bmizerany/pat
[martini]: http://martini.codegangsta.io/
[gocraft]: https://github.com/gocraft/web
[tiger]: https://github.com/rcrowley/go-tigertonic


Is it fast?
-----------

It's not bad: in very informal tests it performed roughly in the middle of the
pack of [one set of benchmarks][bench]. For almost all applications this means
that it's fast enough that it doesn't matter.

I have very little interest in boosting Goji's router's benchmark scores. There
is an obvious solution here--radix trees--and maybe if I get bored I'll
implement one for Goji, but I think the API guarantees and conceptual simplicity
Goji provides are more important (all routes are attempted, one after another,
until a matching route is found). Even if I choose to optimize Goji's router,
Goji's routing semantics will not change.

Plus, Goji provides users with the ability to create their own radix trees: by
using sub-routes you create a tree of routers and match routes in more or less
the same way as a radix tree would. But, again, the real win here in my mind
isn't the performance, but the separation of concerns you get from having your
`/admin` routes and your `/profile` routes far, far away from each other.

Goji's performance isn't all about the router though, it's also about allowing
net/http to perform its built-in optimizations. Perhaps uniquely in the Go web
framework ecosystem, Goji supports net/http's transparent `sendfile(2)` support.

[bench]: https://github.com/cypriss/golang-mux-benchmark/
