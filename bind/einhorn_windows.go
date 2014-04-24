// +build windows

package bind

import (
	"net"
)

func einhornInit()                             {}
func einhornAck()                              {}
func einhornBind(fd int) (net.Listener, error) { return nil, nil }
func usingEinhorn() bool                       { return false }
