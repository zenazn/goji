package graceful

import (
	"runtime"
	"sync"
)

type connShard struct {
	mu sync.Mutex
	// We sort of abuse this field to also act as a "please shut down" flag.
	// If it's nil, you should die at your earliest opportunity.
	set map[*conn]struct{}
}

type connSet struct {
	// This is an incrementing connection counter so we round-robin
	// connections across shards. Use atomic when touching it.
	id     uint64
	shards []*connShard
}

var idleSet connSet

// We pretty aggressively preallocate set entries in the hopes that we never
// have to allocate memory with the lock held. This is definitely a premature
// optimization and is probably misguided, but luckily it costs us essentially
// nothing.
const prealloc = 2048

func init() {
	// To keep the expected contention rate constant we'd have to grow this
	// as numcpu**2. In practice, CPU counts don't generally grow without
	// bound, and contention is probably going to be small enough that
	// nobody cares anyways.
	idleSet.shards = make([]*connShard, 2*runtime.NumCPU())
	for i := range idleSet.shards {
		idleSet.shards[i] = &connShard{
			set: make(map[*conn]struct{}, prealloc),
		}
	}
}

func (cs connSet) markIdle(c *conn) {
	c.busy = false
	shard := cs.shards[int(c.id%uint64(len(cs.shards)))]
	shard.mu.Lock()
	if shard.set == nil {
		shard.mu.Unlock()
		c.die = true
	} else {
		shard.set[c] = struct{}{}
		shard.mu.Unlock()
	}
}

func (cs connSet) markBusy(c *conn) {
	c.busy = true
	shard := cs.shards[int(c.id%uint64(len(cs.shards)))]
	shard.mu.Lock()
	if shard.set == nil {
		shard.mu.Unlock()
		c.die = true
	} else {
		delete(shard.set, c)
		shard.mu.Unlock()
	}
}

func (cs connSet) killall() {
	for _, shard := range cs.shards {
		shard.mu.Lock()
		set := shard.set
		shard.set = nil
		shard.mu.Unlock()

		for conn := range set {
			conn.closeIfIdle()
		}
	}
}
