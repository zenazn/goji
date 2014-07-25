// +build !windows

package bind

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"syscall"
)

const tooBigErr = "bind: einhorn@%d not found (einhorn only passed %d fds)"
const bindErr = "bind: could not bind einhorn@%d: not running under einhorn"
const einhornErr = "bind: einhorn environment initialization error"
const ackErr = "bind: error ACKing to einhorn: %v"

var einhornNumFds int

func envInt(val string) (int, error) {
	return strconv.Atoi(os.Getenv(val))
}

// Unfortunately this can't be a normal init function, because their execution
// order is undefined, and we need to run before the init() in bind.go.
func einhornInit() {
	mpid, err := envInt("EINHORN_MASTER_PID")
	if err != nil || mpid != os.Getppid() {
		return
	}

	einhornNumFds, err = envInt("EINHORN_FD_COUNT")
	if err != nil {
		einhornNumFds = 0
		return
	}

	// Prevent einhorn's fds from leaking to our children
	for i := 0; i < einhornNumFds; i++ {
		syscall.CloseOnExec(einhornFdMap(i))
	}
}

func usingEinhorn() bool {
	return einhornNumFds > 0
}

func einhornFdMap(n int) int {
	name := fmt.Sprintf("EINHORN_FD_%d", n)
	fno, err := envInt(name)
	if err != nil {
		log.Fatal(einhornErr)
	}
	return fno
}

func einhornBind(n int) (net.Listener, error) {
	if !usingEinhorn() {
		return nil, fmt.Errorf(bindErr, n)
	}
	if n >= einhornNumFds || n < 0 {
		return nil, fmt.Errorf(tooBigErr, n, einhornNumFds)
	}

	fno := einhornFdMap(n)
	f := os.NewFile(uintptr(fno), fmt.Sprintf("einhorn@%d", n))
	defer f.Close()
	return net.FileListener(f)
}

// Fun story: this is actually YAML, not JSON.
const ackMsg = `{"command": "worker:ack", "pid": %d}` + "\n"

func einhornAck() {
	if !usingEinhorn() {
		return
	}
	log.Print("bind: ACKing to einhorn")

	ctl, err := net.Dial("unix", os.Getenv("EINHORN_SOCK_PATH"))
	if err != nil {
		log.Fatalf(ackErr, err)
	}
	defer ctl.Close()

	_, err = fmt.Fprintf(ctl, ackMsg, os.Getpid())
	if err != nil {
		log.Fatalf(ackErr, err)
	}
}
