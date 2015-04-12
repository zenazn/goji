package graceful

import (
	"crypto/tls"
	"net"
	"net/http"
)

// Most of the code here is lifted straight from net/http

// A Server is exactly the same as an http.Server, but provides more graceful
// implementations of its methods.
type Server http.Server

// ListenAndServe behaves like the method on net/http.Server with the same name.
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

// ListenAndServeTLS behaves almost exactly like the net/http function of the
// same name. Unlike net/http, however, this function defaults to enforcing TLS
// 1.0 or higher in order to address the POODLE vulnerability. Users who wish to
// enable SSLv3 must do so by explicitly instantiating a server with an
// appropriately configured TLSConfig property.
func ListenAndServeTLS(addr, certfile, keyfile string, handler http.Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(certfile, keyfile)
}

// Serve behaves exactly like the net/http function of the same name.
func Serve(l net.Listener, handler http.Handler) error {
	server := &Server{Handler: handler}
	return server.Serve(l)
}
