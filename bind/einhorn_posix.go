// +build !windows

package bind

import (
	"syscall"
)

func closeOnExec(fd int) {
	syscall.CloseOnExec(fd)
}
