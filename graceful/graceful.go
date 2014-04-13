/*
Package graceful implements graceful shutdown for HTTP servers by closing idle
connections after receiving a signal. By default, this package listens for
interrupts (i.e., SIGINT), but when it detects that it is running under Einhorn
it will additionally listen for SIGUSR2 as well, giving your application
automatic support for graceful upgrades.

It's worth mentioning explicitly that this package is a hack to shim graceful
shutdown behavior into the net/http package provided in Go 1.2. It was written
by carefully reading the sequence of function calls net/http happened to use as
of this writing and finding enough surface area with which to add appropriate
behavior. There's a very good chance that this package will cease to work in
future versions of Go, but with any luck the standard library will add support
of its own by then (https://code.google.com/p/go/issues/detail?id=4674).

If you're interested in figuring out how this package works, we suggest you read
the documentation for WrapConn() and net.go.
*/
package graceful

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

/*
You might notice that these methods look awfully similar to the methods of the
same name from the go standard library--that's because they were stolen from
there! If go were more like, say, Ruby, it'd actually be possible to shim just
the Serve() method, since we can do everything we want from there. However, it's
not possible to get the other methods which call Serve() (ListenAndServe(), say)
to call your shimmed copy--they always call the original.

Since I couldn't come up with a better idea, I just copy-and-pasted both
ListenAndServe and ListenAndServeTLS here more-or-less verbatim. "Oh well!"
*/

// Type Server is exactly the same as an http.Server, but provides more graceful
// implementations of its methods.
type Server http.Server

func (srv *Server) Serve(l net.Listener) (err error) {
	go func() {
		<-kill
		l.Close()
	}()
	l = WrapListener(l)

	// Spawn a shadow http.Server to do the actual servering. We do this
	// because we need to sketch on some of the parameters you passed in,
	// and it's nice to keep our sketching to ourselves.
	shadow := *(*http.Server)(srv)

	if shadow.ReadTimeout == 0 {
		shadow.ReadTimeout = forever
	}
	shadow.Handler = Middleware(shadow.Handler)

	err = shadow.Serve(l)

	// We expect an error when we close the listener, so we indiscriminately
	// swallow Serve errors when we're in a shutdown state.
	select {
	case <-kill:
		return nil
	default:
		return err
	}
}

// About 200 years, also known as "forever"
const forever time.Duration = 200 * 365 * 24 * time.Hour

func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	return srv.Serve(l)
}

func (srv *Server) ListenAndServeTLS(certFile, keyFile string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, config)
	return srv.Serve(tlsListener)
}

// ListenAndServe behaves exactly like the net/http function of the same name.
func ListenAndServe(addr string, handler http.Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// ListenAndServeTLS behaves exactly like the net/http function of the same name.
func ListenAndServeTLS(addr, certfile, keyfile string, handler http.Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(certfile, keyfile)
}

// Serve behaves exactly like the net/http function of the same name.
func Serve(l net.Listener, handler http.Handler) error {
	server := &Server{Handler: handler}
	return server.Serve(l)
}
