/*
Package bind provides a convenient way to bind to sockets. It exposes a flag in
the default flag set named "bind" which provides syntax to bind TCP and UNIX
sockets. It also supports binding to arbitrary file descriptors passed by a
parent (for instance, systemd), and for binding to Einhorn sockets (including
Einhorn ACK support).

If the value passed to bind contains a colon, as in ":8000" or "127.0.0.1:9001",
it will be treated as a TCP address. If it begins with a "/" or a ".", it will
be treated as a path to a UNIX socket. If it begins with the string "fd@", as in
"fd@3", it will be treated as a file descriptor (useful for use with systemd,
for instance). If it begins with the string "einhorn@", as in "einhorn@0", the
corresponding einhorn socket will be used.

If an option is not explicitly passed, the implementation will automatically
select between using "einhorn@0", "fd@3", and ":8000", depending on whether
Einhorn or systemd (or neither) is detected.

This package is a teensy bit magical, and goes out of its way to Do The Right
Thing in many situations, including in both development and production. If
you're looking for something less magical, you'd probably be better off just
calling net.Listen() the old-fashioned way.
*/
package bind

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var bind string

func init() {
	einhornInit()
	systemdInit()

	defaultBind := ":8000"
	if bind := os.Getenv("GOJI_BIND"); bind != "" {
		defaultBind = bind
	} else if usingEinhorn() {
		defaultBind = "einhorn@0"
	} else if usingSystemd() {
		defaultBind = "fd@3"
	} else if port := os.Getenv("PORT"); port != "" {
		defaultBind = ":" + port
	}
	flag.StringVar(&bind, "bind", defaultBind,
		`Address to bind on. If this value has a colon, as in ":8000" or
		"127.0.0.1:9001", it will be treated as a TCP address. If it
		begins with a "/" or a ".", it will be treated as a path to a
		UNIX socket. If it begins with the string "fd@", as in "fd@3",
		it will be treated as a file descriptor (useful for use with
		systemd, for instance). If it begins with the string "einhorn@",
		as in "einhorn@0", the corresponding einhorn socket will be
		used. If an option is not explicitly passed, the implementation
		will automatically select among "einhorn@0" (Einhorn), "fd@3"
		(systemd), and ":8000" (fallback) based on its environment.`)
}

func listenTo(bind string) (net.Listener, error) {
	if strings.Contains(bind, ":") {
		return net.Listen("tcp", bind)
	} else if strings.HasPrefix(bind, ".") || strings.HasPrefix(bind, "/") {
		return net.Listen("unix", bind)
	} else if strings.HasPrefix(bind, "fd@") {
		fd, err := strconv.Atoi(bind[3:])
		if err != nil {
			return nil, fmt.Errorf("error while parsing fd %v: %v",
				bind, err)
		}
		f := os.NewFile(uintptr(fd), bind)
		defer f.Close()
		return net.FileListener(f)
	} else if strings.HasPrefix(bind, "einhorn@") {
		fd, err := strconv.Atoi(bind[8:])
		if err != nil {
			return nil, fmt.Errorf(
				"error while parsing einhorn %v: %v", bind, err)
		}
		return einhornBind(fd)
	}

	return nil, fmt.Errorf("error while parsing bind arg %v", bind)
}

// Socket parses and binds to the specified address. If Socket encounters an
// error while parsing or binding to the given socket it will exit by calling
// log.Fatal.
func Socket(bind string) net.Listener {
	l, err := listenTo(bind)
	if err != nil {
		log.Fatal(err)
	}
	return l
}

// Default parses and binds to the default socket as given to us by the flag
// module. If there was an error parsing or binding to that socket, Default will
// exit by calling `log.Fatal`.
func Default() net.Listener {
	return Socket(bind)
}

// I'm not sure why you'd ever want to call Ready() more than once, but we may
// as well be safe against it...
var ready sync.Once

// Ready notifies the environment (for now, just Einhorn) that the process is
// ready to receive traffic. Should be called at the last possible moment to
// maximize the chances that a faulty process exits before signaling that it's
// ready.
func Ready() {
	ready.Do(func() {
		einhornAck()
	})
}
