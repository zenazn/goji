// +build go1.3

package graceful

import (
	"net"
	"net/http"
	"time"
)

// About 200 years, also known as "forever"
const forever time.Duration = 200 * 365 * 24 * time.Hour

func (srv *Server) Serve(l net.Listener) error {
	l = WrapListener(l)

	// Spawn a shadow http.Server to do the actual servering. We do this
	// because we need to sketch on some of the parameters you passed in,
	// and it's nice to keep our sketching to ourselves.
	shadow := *(*http.Server)(srv)

	if shadow.ReadTimeout == 0 {
		shadow.ReadTimeout = forever
	}

	go func() {
		<-kill
		shadow.SetKeepAlivesEnabled(false)
		l.Close()
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
