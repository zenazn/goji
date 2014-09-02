// +build go1.3

package graceful

import (
	"net"
	"net/http"
)

func (srv *Server) Serve(l net.Listener) error {
	l = WrapListener(l)

	// Spawn a shadow http.Server to do the actual servering. We do this
	// because we need to sketch on some of the parameters you passed in,
	// and it's nice to keep our sketching to ourselves.
	shadow := *(*http.Server)(srv)

	cs := shadow.ConnState
	shadow.ConnState = func(nc net.Conn, s http.ConnState) {
		if c, ok := nc.(*conn); ok {
			// There are a few other states defined, most notably
			// StateActive. Unfortunately it doesn't look like it's
			// possible to make use of StateActive to implement
			// graceful shutdown, since StateActive is set after a
			// complete request has been read off the wire with an
			// intent to process it. If we were to race a graceful
			// shutdown against a connection that was just read off
			// the wire (but not yet in StateActive), we would
			// accidentally close the connection out from underneath
			// an active request.
			//
			// We already needed to work around this for Go 1.2 by
			// shimming out a full net.Conn object, so we can just
			// fall back to the old behavior there.
			//
			// I started a golang-nuts thread about this here:
			// https://groups.google.com/forum/#!topic/golang-nuts/Xi8yjBGWfCQ
			// I'd be very eager to find a better way to do this, so
			// reach out to me if you have any ideas.
			switch s {
			case http.StateIdle:
				c.markIdle()
			case http.StateHijacked:
				c.hijack()
			}
		}
		if cs != nil {
			cs(nc, s)
		}
	}

	go func() {
		<-kill
		l.Close()
		shadow.SetKeepAlivesEnabled(false)
		idleSet.killall()
	}()

	err := shadow.Serve(l)

	// We expect an error when we close the listener, so we indiscriminately
	// swallow Serve errors when we're in a shutdown state.
	select {
	case <-kill:
		return nil
	default:
		return err
	}
}
