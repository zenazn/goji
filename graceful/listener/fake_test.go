package listener

import (
	"net"
	"time"
)

type fakeAddr struct {
	network, addr string
}

func (f fakeAddr) Network() string {
	return f.network
}
func (f fakeAddr) String() string {
	return f.addr
}

type fakeListener struct {
	ch     chan net.Conn
	closed chan struct{}
	addr   net.Addr
}

func makeFakeListener(addr string) *fakeListener {
	a := fakeAddr{"tcp", addr}
	return &fakeListener{
		ch:     make(chan net.Conn),
		closed: make(chan struct{}),
		addr:   a,
	}
}

func (f *fakeListener) Accept() (net.Conn, error) {
	select {
	case c := <-f.ch:
		return c, nil
	case <-f.closed:
		return nil, errClosing
	}
}
func (f *fakeListener) Close() error {
	close(f.closed)
	return nil
}

func (f *fakeListener) Addr() net.Addr {
	return f.addr
}

func (f *fakeListener) Enqueue(c net.Conn) {
	f.ch <- c
}

type fakeConn struct {
	read, write, closed chan struct{}
	me, you             net.Addr
}

func makeFakeConn(me, you string) *fakeConn {
	return &fakeConn{
		read:   make(chan struct{}),
		write:  make(chan struct{}),
		closed: make(chan struct{}),
		me:     fakeAddr{"tcp", me},
		you:    fakeAddr{"tcp", you},
	}
}

func (f *fakeConn) Read(buf []byte) (int, error) {
	select {
	case <-f.read:
		return len(buf), nil
	case <-f.closed:
		return 0, errClosing
	}
}

func (f *fakeConn) Write(buf []byte) (int, error) {
	select {
	case <-f.write:
		return len(buf), nil
	case <-f.closed:
		return 0, errClosing
	}
}

func (f *fakeConn) Close() error {
	close(f.closed)
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	return f.me
}
func (f *fakeConn) RemoteAddr() net.Addr {
	return f.you
}
func (f *fakeConn) SetDeadline(t time.Time) error {
	return nil
}
func (f *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (f *fakeConn) Closed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func (f *fakeConn) AllowRead() {
	f.read <- struct{}{}
}
func (f *fakeConn) AllowWrite() {
	f.write <- struct{}{}
}
