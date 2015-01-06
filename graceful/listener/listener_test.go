package listener

import (
	"net"
	"testing"
	"time"
)

// Helper for tests acting on a single accepted connection
func singleConn(t *testing.T, m mode) (*T, *fakeConn, net.Conn) {
	l := makeFakeListener("net.Listener")
	wl := Wrap(l, m)
	c := makeFakeConn("local", "remote")

	go l.Enqueue(c)
	wc, err := wl.Accept()
	if err != nil {
		t.Fatalf("error accepting connection: %v", err)
	}
	return wl, c, wc
}

func TestAddr(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	if a := l.Addr(); a.String() != "net.Listener" {
		t.Errorf("addr was %v, wanted net.Listener", a)
	}

	if c.LocalAddr() != wc.LocalAddr() {
		t.Errorf("local addresses don't match: %v, %v", c.LocalAddr(),
			wc.LocalAddr())
	}
	if c.RemoteAddr() != wc.RemoteAddr() {
		t.Errorf("remote addresses don't match: %v, %v", c.RemoteAddr(),
			wc.RemoteAddr())
	}
}

func TestBasicCloseIdle(t *testing.T) {
	t.Parallel()
	l, c, _ := singleConn(t, Manual)

	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if !c.Closed() {
		t.Error("idle connection not closed")
	}
}

func TestMark(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	if err := MarkInUse(wc); err != nil {
		t.Fatalf("error marking %v in-use: %v", wc, err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if c.Closed() {
		t.Errorf("manually in-use connection was closed")
	}

	if err := MarkIdle(wc); err != nil {
		t.Fatalf("error marking %v idle: %v", wc, err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if !c.Closed() {
		t.Error("manually idle connection was not closed")
	}
}

func TestDisown(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	if err := Disown(wc); err != nil {
		t.Fatalf("error disowning connection: %v", err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}

	if c.Closed() {
		t.Errorf("disowned connection got closed")
	}
}

func TestDrain(t *testing.T) {
	t.Parallel()
	l, _, wc := singleConn(t, Manual)

	MarkInUse(wc)
	start := time.Now()
	go func() {
		time.Sleep(50 * time.Millisecond)
		MarkIdle(wc)
	}()
	if err := l.Drain(); err != nil {
		t.Fatalf("error draining listener: %v", err)
	}
	end := time.Now()
	if dt := end.Sub(start); dt < 50*time.Millisecond {
		t.Errorf("expected at least 50ms wait, but got %v", dt)
	}
}

func TestDrainAll(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	MarkInUse(wc)
	if err := l.DrainAll(); err != nil {
		t.Fatalf("error draining listener: %v", err)
	}
	if !c.Closed() {
		t.Error("expected in-use connection to be closed")
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()
	_, c, wc := singleConn(t, Manual)
	if err := Disown(c); err == nil {
		t.Error("expected error when disowning unmanaged net.Conn")
	}
	if err := MarkIdle(c); err == nil {
		t.Error("expected error when marking unmanaged net.Conn idle")
	}
	if err := MarkInUse(c); err == nil {
		t.Error("expected error when marking unmanaged net.Conn in use")
	}

	if err := Disown(wc); err != nil {
		t.Fatalf("unexpected error disowning socket: %v", err)
	}
	if err := Disown(wc); err == nil {
		t.Error("expected error disowning socket twice")
	}
}

func TestClose(t *testing.T) {
	t.Parallel()
	l, c, _ := singleConn(t, Manual)
	if err := l.Close(); err != nil {
		t.Fatalf("error while closing listener: %v", err)
	}
	if c.Closed() {
		t.Error("connection closed when listener was?")
	}
}
