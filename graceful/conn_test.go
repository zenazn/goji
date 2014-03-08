package graceful

import (
	"net"
	"time"
)

// Stub out a net.Conn. This is going to be painful.

type fakeAddr struct{}

func (f fakeAddr) Network() string {
	return "fake"
}
func (f fakeAddr) String() string {
	return "fake"
}

type fakeConn struct {
	onRead, onWrite, onClose, onLocalAddr, onRemoteAddr  func()
	onSetDeadline, onSetReadDeadline, onSetWriteDeadline func()
}

// Here's my number, so...
func callMeMaybe(f func()) {
	// I apologize for nothing.
	if f != nil {
		f()
	}
}

func (f fakeConn) Read(b []byte) (int, error) {
	callMeMaybe(f.onRead)
	return len(b), nil
}
func (f fakeConn) Write(b []byte) (int, error) {
	callMeMaybe(f.onWrite)
	return len(b), nil
}
func (f fakeConn) Close() error {
	callMeMaybe(f.onClose)
	return nil
}
func (f fakeConn) LocalAddr() net.Addr {
	callMeMaybe(f.onLocalAddr)
	return fakeAddr{}
}
func (f fakeConn) RemoteAddr() net.Addr {
	callMeMaybe(f.onRemoteAddr)
	return fakeAddr{}
}
func (f fakeConn) SetDeadline(t time.Time) error {
	callMeMaybe(f.onSetDeadline)
	return nil
}
func (f fakeConn) SetReadDeadline(t time.Time) error {
	callMeMaybe(f.onSetReadDeadline)
	return nil
}
func (f fakeConn) SetWriteDeadline(t time.Time) error {
	callMeMaybe(f.onSetWriteDeadline)
	return nil
}
