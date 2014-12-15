package listener

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func init() {
	// Just to make sure we get some variety
	runtime.GOMAXPROCS(4 * runtime.NumCPU())
}

// Chosen by random die roll
const seed = 4611413766552969250

// This is mostly just fuzzing to see what happens.
func TestRace(t *testing.T) {
	t.Parallel()

	l := makeFakeListener("net.Listener")
	wl := Wrap(l, Automatic)

	var flag int32

	go func() {
		for i := 0; ; i++ {
			laddr := fmt.Sprintf("local%d", i)
			raddr := fmt.Sprintf("remote%d", i)
			c := makeFakeConn(laddr, raddr)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						if atomic.LoadInt32(&flag) != 0 {
							return
						}
						panic(r)
					}
				}()
				l.Enqueue(c)
			}()
			wc, err := wl.Accept()
			if err != nil {
				if atomic.LoadInt32(&flag) != 0 {
					return
				}
				t.Fatalf("error accepting connection: %v", err)
			}

			go func() {
				for {
					time.Sleep(50 * time.Millisecond)
					c.AllowRead()
				}
			}()

			go func(i int64) {
				rng := rand.New(rand.NewSource(i + seed))
				buf := make([]byte, 1024)
				for j := 0; j < 1024; j++ {
					if _, err := wc.Read(buf); err != nil {
						if atomic.LoadInt32(&flag) != 0 {
							// Peaceful; the connection has
							// probably been closed while
							// idle
							return
						}
						t.Errorf("error reading in conn %d: %v",
							i, err)
					}
					time.Sleep(time.Duration(rng.Intn(100)) * time.Millisecond)
					// This one is to make sure the connection
					// hasn't closed underneath us
					if _, err := wc.Read(buf); err != nil {
						t.Errorf("error reading in conn %d: %v",
							i, err)
					}
					MarkIdle(wc)
					time.Sleep(time.Duration(rng.Intn(100)) * time.Millisecond)
				}
			}(int64(i))

			time.Sleep(time.Duration(i) * time.Millisecond / 2)
		}
	}()

	if testing.Short() {
		time.Sleep(2 * time.Second)
	} else {
		time.Sleep(10 * time.Second)
	}
	start := time.Now()
	atomic.StoreInt32(&flag, 1)
	wl.Close()
	wl.Drain()
	end := time.Now()
	if dt := end.Sub(start); dt > 300*time.Millisecond {
		t.Errorf("took %v to drain; expected shorter", dt)
	}
}
