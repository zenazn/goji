package listener

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestManualRead(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	go c.AllowRead()
	wc.Read(make([]byte, 1024))

	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if !c.Closed() {
		t.Error("Read() should not make connection not-idle")
	}
}

func TestAutomaticRead(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Automatic)

	go c.AllowRead()
	wc.Read(make([]byte, 1024))

	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if c.Closed() {
		t.Error("expected Read() to mark connection as in-use")
	}
}

func TestDeadlineRead(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Deadline)

	go c.AllowRead()
	if _, err := wc.Read(make([]byte, 1024)); err != nil {
		t.Fatalf("error reading from connection: %v", err)
	}

	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if c.Closed() {
		t.Error("expected Read() to mark connection as in-use")
	}
}

func TestDisownedRead(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Deadline)

	if err := Disown(wc); err != nil {
		t.Fatalf("unexpected error disowning conn: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("unexpected error closing listener: %v", err)
	}
	if err := l.Drain(); err != nil {
		t.Fatalf("unexpected error draining listener: %v", err)
	}

	go c.AllowRead()
	if _, err := wc.Read(make([]byte, 1024)); err != nil {
		t.Fatalf("error reading from connection: %v", err)
	}
}

func TestCloseConn(t *testing.T) {
	t.Parallel()
	l, _, wc := singleConn(t, Deadline)

	if err := MarkInUse(wc); err != nil {
		t.Fatalf("error marking conn in use: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Errorf("error closing connection: %v", err)
	}
	// This will hang if wc.Close() doesn't un-track the connection
	if err := l.Drain(); err != nil {
		t.Errorf("error draining listener: %v", err)
	}
}

// Regression test for issue #130.
func TestDisownedClose(t *testing.T) {
	t.Parallel()
	_, c, wc := singleConn(t, Deadline)

	if err := Disown(wc); err != nil {
		t.Fatalf("unexpected error disowning conn: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Errorf("error closing connection: %v", err)
	}
	if !c.Closed() {
		t.Errorf("connection didn't get closed")
	}
}

func TestManualReadDeadline(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Manual)

	if err := MarkInUse(wc); err != nil {
		t.Fatalf("error marking connection in use: %v", err)
	}
	if err := wc.SetReadDeadline(time.Now()); err != nil {
		t.Fatalf("error setting read deadline: %v", err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if c.Closed() {
		t.Error("SetReadDeadline() should not mark manual conn as idle")
	}
}

func TestAutomaticReadDeadline(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Automatic)

	if err := MarkInUse(wc); err != nil {
		t.Fatalf("error marking connection in use: %v", err)
	}
	if err := wc.SetReadDeadline(time.Now()); err != nil {
		t.Fatalf("error setting read deadline: %v", err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if c.Closed() {
		t.Error("SetReadDeadline() should not mark automatic conn as idle")
	}
}

func TestDeadlineReadDeadline(t *testing.T) {
	t.Parallel()
	l, c, wc := singleConn(t, Deadline)

	if err := MarkInUse(wc); err != nil {
		t.Fatalf("error marking connection in use: %v", err)
	}
	if err := wc.SetReadDeadline(time.Now()); err != nil {
		t.Fatalf("error setting read deadline: %v", err)
	}
	if err := l.CloseIdle(); err != nil {
		t.Fatalf("error closing idle connections: %v", err)
	}
	if !c.Closed() {
		t.Error("SetReadDeadline() should mark deadline conn as idle")
	}
}

type readerConn struct {
	fakeConn
}

func (rc *readerConn) ReadFrom(r io.Reader) (int64, error) {
	return 123, nil
}

func TestReadFrom(t *testing.T) {
	t.Parallel()

	l := makeFakeListener("net.Listener")
	wl := Wrap(l, Manual)
	c := &readerConn{
		fakeConn{
			read:   make(chan struct{}),
			write:  make(chan struct{}),
			closed: make(chan struct{}),
			me:     fakeAddr{"tcp", "local"},
			you:    fakeAddr{"tcp", "remote"},
		},
	}

	go l.Enqueue(c)
	wc, err := wl.Accept()
	if err != nil {
		t.Fatalf("error accepting connection: %v", err)
	}

	// The io.MultiReader is a convenient hack to ensure that we're using
	// our ReadFrom, not strings.Reader's WriteTo.
	r := io.MultiReader(strings.NewReader("hello world"))
	if _, err := io.Copy(wc, r); err != nil {
		t.Fatalf("error copying: %v", err)
	}
}
