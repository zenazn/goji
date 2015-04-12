// +build go1.3

package graceful

import (
	"log"
	"net"
	"net/http"

	"github.com/zenazn/goji/graceful/listener"
)

// This is a slightly hacky shim to disable keepalives when shutting a server
// down. We could have added extra functionality in listener or signal.go to
// deal with this case, but this seems simpler.
type gracefulServer struct {
	net.Listener
	s *http.Server
}

func (g gracefulServer) Close() error {
	g.s.SetKeepAlivesEnabled(false)
	return g.Listener.Close()
}

// A chaining http.ConnState wrapper
type connState func(net.Conn, http.ConnState)

func (c connState) Wrap(nc net.Conn, s http.ConnState) {
	// There are a few other states defined, most notably StateActive.
	// Unfortunately it doesn't look like it's possible to make use of
	// StateActive to implement graceful shutdown, since StateActive is set
	// after a complete request has been read off the wire with an intent to
	// process it. If we were to race a graceful shutdown against a
	// connection that was just read off the wire (but not yet in
	// StateActive), we would accidentally close the connection out from
	// underneath an active request.
	//
	// We already needed to work around this for Go 1.2 by shimming out a
	// full net.Conn object, so we can just fall back to the old behavior
	// there.
	//
	// I started a golang-nuts thread about this here:
	// https://groups.google.com/forum/#!topic/golang-nuts/Xi8yjBGWfCQ
	// I'd be very eager to find a better way to do this, so reach out to me
	// if you have any ideas.
	switch s {
	case http.StateIdle:
		if err := listener.MarkIdle(nc); err != nil {
			log.Printf("error marking conn as idle: %v", err)
		}
	case http.StateHijacked:
		if err := listener.Disown(nc); err != nil {
			log.Printf("error disowning hijacked conn: %v", err)
		}
	}
	if c != nil {
		c(nc, s)
	}
}

// Serve behaves like the method on net/http.Server with the same name.
func (srv *Server) Serve(l net.Listener) error {
	// Spawn a shadow http.Server to do the actual servering. We do this
	// because we need to sketch on some of the parameters you passed in,
	// and it's nice to keep our sketching to ourselves.
	shadow := *(*http.Server)(srv)
	shadow.ConnState = connState(shadow.ConnState).Wrap

	l = gracefulServer{l, &shadow}
	wrap := listener.Wrap(l, listener.Automatic)
	appendListener(wrap)

	err := shadow.Serve(wrap)
	return peacefulError(err)
}
