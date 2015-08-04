package graceful

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Most of the code here is lifted straight from net/http

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// A Server is exactly the same as an http.Server, but provides more graceful
// implementations of its methods.
type Server http.Server

// ListenAndServe behaves like the method on net/http.Server with the same name.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// ListenAndServeTLS behaves like the method on net/http.Server with the same
// name. Unlike the method of the same name on http.Server, this function
// defaults to enforcing TLS 1.0 or higher in order to address the POODLE
// vulnerability. Users who wish to enable SSLv3 must do so by supplying a
// TLSConfig explicitly.
func (srv *Server) ListenAndServeTLS(certFile, keyFile string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{
		MinVersion: tls.VersionTLS10,
	}
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

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	return srv.Serve(tlsListener)
}

// ListenAndServe behaves exactly like the net/http function of the same name.
func ListenAndServe(addr string, handler http.Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// ListenAndServeTLS behaves almost exactly like the net/http function of the
// same name. Unlike net/http, however, this function defaults to enforcing TLS
// 1.0 or higher in order to address the POODLE vulnerability. Users who wish to
// enable SSLv3 must do so by explicitly instantiating a server with an
// appropriately configured TLSConfig property.
func ListenAndServeTLS(addr, certfile, keyfile string, handler http.Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(certfile, keyfile)
}

// Serve mostly behaves like the net/http function of the same name, except that
// if the passed listener is a net.TCPListener, TCP keep-alives are enabled on
// accepted connections.
func Serve(l net.Listener, handler http.Handler) error {
	if tl, ok := l.(*net.TCPListener); ok {
		l = tcpKeepAliveListener{tl}
	}
	server := &Server{Handler: handler}
	return server.Serve(l)
}
