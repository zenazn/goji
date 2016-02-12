Goji
====

[![GoDoc](https://godoc.org/github.com/zenazn/goji/web?status.svg)](https://godoc.org/github.com/zenazn/goji/web) [![Build Status](https://travis-ci.org/zenazn/goji.svg?branch=master)](https://travis-ci.org/zenazn/goji)

Goji is a minimalistic web framework that values composability and simplicity.

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
        fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
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
* URL patterns (both Sinatra style `/foo/:bar` patterns and regular expressions,
  as well as [custom patterns][pattern])
* Reconfigurable middleware stack
* Context/environment object threaded through middleware and handlers
* Automatic support for [Einhorn][einhorn], systemd, and [more][bind]
* [Graceful shutdown][graceful], and zero-downtime graceful reload when combined
  with Einhorn.
* High in antioxidants

[einhorn]: https://github.com/stripe/einhorn
[bind]: http://godoc.org/github.com/zenazn/goji/bind
[graceful]: http://godoc.org/github.com/zenazn/goji/graceful
[pattern]: https://godoc.org/github.com/zenazn/goji/web#Pattern


Is it any good?
---------------

Maybe!

There are [plenty][revel] of [other][gorilla] [good][pat] [Go][martini]
[web][gocraft] [frameworks][tiger] out there. Goji is by no means especially
novel, nor is it uniquely good. The primary difference between Goji and other
frameworks—and the primary reason I think Goji is any good—is its philosophy:

Goji first of all attempts to be simple. It is of the Sinatra and Flask school
of web framework design, and not the Rails/Django one. If you want me to tell
you what directory you should put your models in, or if you want built-in flash
sessions, you won't have a good time with Goji.

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

[Yeah][bench1], [it is][bench2]. Goji is among the fastest HTTP routers out
there, and is very gentle on the garbage collector.

But that's sort of missing the point. Almost all Go routers are fast enough for
almost all purposes. In my opinion, what matters more is how simple and flexible
the routing semantics are.

Goji provides results indistinguishable from naively trying routes one after
another. This means that a route added before another route will be attempted
before that route as well. This is perhaps the most simple and most intuitive
interface a router can provide, and makes routes very easy to understand and
debug.

Goji's router is also very flexible: in addition to the standard Sinatra-style
patterns and regular expression patterns, you can define [custom
patterns][pattern] to perform whatever custom matching logic you desire. Custom
patterns of course are fully compatible with the routing semantics above.

It's easy (and quite a bit of fun!) to get carried away by microbenchmarks, but
at the end of the day you're not going to miss those extra hundred nanoseconds
on a request. What matters is that you aren't compromising on the API for a
handful of CPU cycles.

[bench1]: https://gist.github.com/zenazn/c5c8528efe1a00634096
[bench2]: https://github.com/julienschmidt/go-http-routing-benchmark


Third-Party Libraries
---------------------

Goji is already compatible with a great many third-party libraries that are
themselves compatible with `net/http`, however some library authors have gone
out of their way to include Goji compatibility specifically, perhaps by
integrating more tightly with Goji's `web.C` or by providing a custom pattern
type. An informal list of such libraries is maintained [on the wiki][third];
feel free to add to it as you see fit.

[third]: https://github.com/zenazn/goji/wiki/Third-Party-Libraries


Contributing
------------

Please do! I love pull requests, and I love pull requests that include tests
even more. Goji's core packages have pretty good code coverage (yay code
coverage gamification!), and if you have the time to write tests I'd like to
keep it that way.

In addition to contributing code, I'd love to know what you think about Goji.
Please open an issue or send me an email with your thoughts; it'd mean a lot to
me.
