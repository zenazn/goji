package graceful

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

var b = make([]byte, 0)

func connify(c net.Conn) *conn {
	switch c.(type) {
	case (*conn):
		return c.(*conn)
	case (*sendfile):
		return &c.(*sendfile).conn
	default:
		panic("IDK")
	}
}

func assertState(t *testing.T, n net.Conn, st connstate) {
	c := connify(n)
	c.m.Lock()
	defer c.m.Unlock()
	if c.state != st {
		t.Fatalf("conn was %v, but expected %v", c.state, st)
	}
}

// Not super happy about making the tests dependent on the passing of time, but
// I'm not really sure what else to do.

func expectCall(t *testing.T, ch <-chan struct{}, name string) {
	select {
	case <-ch:
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("Expected call to %s", name)
	}
}

func TestCounting(t *testing.T) {
	kill = make(chan struct{})
	c := WrapConn(fakeConn{})
	ch := make(chan struct{})

	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		t.Fatal("Expected connection to keep us from quitting")
	case <-time.After(5 * time.Millisecond):
	}

	c.Close()
	expectCall(t, ch, "wg.Wait()")
}

func TestStateTransitions1(t *testing.T) {
	kill = make(chan struct{})
	ch := make(chan struct{})

	onclose := make(chan struct{})
	read := make(chan struct{})
	deadline := make(chan struct{})
	c := WrapConn(fakeConn{
		onClose: func() {
			onclose <- struct{}{}
		},
		onRead: func() {
			read <- struct{}{}
		},
		onSetReadDeadline: func() {
			deadline <- struct{}{}
		},
	})

	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()

	assertState(t, c, csWaiting)

	// Waiting + Read() = Working
	go c.Read(b)
	expectCall(t, read, "c.Read()")
	assertState(t, c, csWorking)

	// Working + SetReadDeadline() = Waiting
	go c.SetReadDeadline(time.Now())
	expectCall(t, deadline, "c.SetReadDeadline()")
	assertState(t, c, csWaiting)

	// Waiting + kill = Dead
	close(kill)
	expectCall(t, onclose, "c.Close()")
	assertState(t, c, csDead)

	expectCall(t, ch, "wg.Wait()")
}

func TestStateTransitions2(t *testing.T) {
	kill = make(chan struct{})
	ch := make(chan struct{})
	onclose := make(chan struct{})
	read := make(chan struct{})
	deadline := make(chan struct{})
	c := WrapConn(fakeConn{
		onClose: func() {
			onclose <- struct{}{}
		},
		onRead: func() {
			read <- struct{}{}
		},
		onSetReadDeadline: func() {
			deadline <- struct{}{}
		},
	})

	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()

	assertState(t, c, csWaiting)

	// Waiting + Read() = Working
	go c.Read(b)
	expectCall(t, read, "c.Read()")
	assertState(t, c, csWorking)

	// Working + Read() = Working
	go c.Read(b)
	expectCall(t, read, "c.Read()")
	assertState(t, c, csWorking)

	// Working + kill = Dying
	close(kill)
	time.Sleep(5 * time.Millisecond)
	assertState(t, c, csDying)

	// Dying + Read() = Dying
	go c.Read(b)
	expectCall(t, read, "c.Read()")
	assertState(t, c, csDying)

	// Dying + SetReadDeadline() = Dead
	go c.SetReadDeadline(time.Now())
	expectCall(t, deadline, "c.SetReadDeadline()")
	assertState(t, c, csDead)

	expectCall(t, ch, "wg.Wait()")
}

func TestHijack(t *testing.T) {
	kill = make(chan struct{})
	fake := fakeConn{}
	c := WrapConn(fake)
	ch := make(chan struct{})

	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()

	cc := connify(c)
	if _, ok := cc.hijack().(fakeConn); !ok {
		t.Error("Expected original connection back out")
	}
	assertState(t, c, csDead)
	expectCall(t, ch, "wg.Wait()")
}

type fakeSendfile struct {
	fakeConn
}

func (f fakeSendfile) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}

func TestReadFrom(t *testing.T) {
	kill = make(chan struct{})
	c := WrapConn(fakeSendfile{})
	r := strings.NewReader("Hello world")

	if rf, ok := c.(io.ReaderFrom); ok {
		rf.ReadFrom(r)
	} else {
		t.Fatal("Expected a ReaderFrom in return")
	}
}
