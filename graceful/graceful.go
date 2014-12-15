/*
Package graceful implements graceful shutdown for HTTP servers by closing idle
connections after receiving a signal. By default, this package listens for
interrupts (i.e., SIGINT), but when it detects that it is running under Einhorn
it will additionally listen for SIGUSR2 as well, giving your application
automatic support for graceful restarts/code upgrades.
*/
package graceful

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/zenazn/goji/graceful/listener"
)

// Most of the code here is lifted straight from net/http

// Type Server is exactly the same as an http.Server, but provides more graceful
// implementations of its methods.
type Server http.Server

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

// WrapListener wraps an arbitrary net.Listener for use with graceful shutdowns.
// In the background, it uses the listener sub-package to Wrap the listener in
// Deadline mode. If another mode of operation is desired, you should call
// listener.Wrap yourself: this function is smart enough to not double-wrap
// listeners.
func WrapListener(l net.Listener) net.Listener {
	if lt, ok := l.(*listener.T); ok {
		appendListener(lt)
		return lt
	}

	lt := listener.Wrap(l, listener.Deadline)
	appendListener(lt)
	return lt
}
