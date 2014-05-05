package bind

import (
	"syscall"
)

func closeOnExec(fd int) {
	syscall.CloseOnExec(syscall.Handle(fd))
}
