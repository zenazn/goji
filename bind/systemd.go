// +build !windows

package bind

import (
	"os"
	"syscall"
)

const systemdMinFd = 3

var systemdNumFds int

// Unfortunately this can't be a normal init function, because their execution
// order is undefined, and we need to run before the init() in bind.go.
func systemdInit() {
	pid, err := envInt("LISTEN_PID")
	if err != nil || pid != os.Getpid() {
		return
	}

	systemdNumFds, err = envInt("LISTEN_FDS")
	if err != nil {
		systemdNumFds = 0
		return
	}

	// Prevent fds from leaking to our children
	for i := 0; i < systemdNumFds; i++ {
		syscall.CloseOnExec(systemdMinFd + i)
	}
}

func usingSystemd() bool {
	return systemdNumFds > 0
}
