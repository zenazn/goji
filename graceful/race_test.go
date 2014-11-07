// +build race

package graceful

import "testing"

func TestWaitGroupRace(t *testing.T) {
	go func() {
		go WrapConn(fakeConn{}).Close()
	}()
	Shutdown()
}
