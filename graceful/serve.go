// +build !go1.3

package graceful

import (
	"net"
	"net/http"
	"time"

	"github.com/zenazn/goji/graceful/listener"
)

// About 200 years, also known as "forever"
const forever time.Duration = 200 * 365 * 24 * time.Hour

// Serve behaves like the method on net/http.Server with the same name.
func (srv *Server) Serve(l net.Listener) error {
	// Spawn a shadow http.Server to do the actual servering. We do this
	// because we need to sketch on some of the parameters you passed in,
	// and it's nice to keep our sketching to ourselves.
	shadow := *(*http.Server)(srv)

	if shadow.ReadTimeout == 0 {
		shadow.ReadTimeout = forever
	}
	shadow.Handler = middleware(shadow.Handler)

	wrap := listener.Wrap(l, listener.Deadline)
	appendListener(wrap)

	err := shadow.Serve(wrap)
	return peacefulError(err)
}
