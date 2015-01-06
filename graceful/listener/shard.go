package listener

import "sync"

type shard struct {
	l *T

	mu    sync.Mutex
	idle  map[*conn]struct{}
	all   map[*conn]struct{}
	wg    sync.WaitGroup
	drain bool
}

// We pretty aggressively preallocate set entries in the hopes that we never
// have to allocate memory with the lock held. This is definitely a premature
// optimization and is probably misguided, but luckily it costs us essentially
// nothing.
const prealloc = 2048

func (s *shard) init(l *T) {
	s.l = l
	s.idle = make(map[*conn]struct{}, prealloc)
	s.all = make(map[*conn]struct{}, prealloc)
}

func (s *shard) track(c *conn) (shouldClose bool) {
	s.mu.Lock()
	if s.drain {
		s.mu.Unlock()
		return true
	}
	s.all[c] = struct{}{}
	s.idle[c] = struct{}{}
	s.mu.Unlock()
	return false
}

func (s *shard) disown(c *conn) {
	s.mu.Lock()
	delete(s.all, c)
	delete(s.idle, c)
	s.mu.Unlock()
}

func (s *shard) markIdle(c *conn) (shouldClose bool) {
	s.mu.Lock()
	if s.drain {
		s.mu.Unlock()
		return true
	}
	s.idle[c] = struct{}{}
	s.mu.Unlock()
	return false
}

func (s *shard) markInUse(c *conn) {
	s.mu.Lock()
	delete(s.idle, c)
	s.mu.Unlock()
}

func (s *shard) closeConns(all, drain bool) {
	s.mu.Lock()
	if drain {
		s.drain = true
	}
	set := make(map[*conn]struct{}, len(s.all))
	if all {
		for c := range s.all {
			set[c] = struct{}{}
		}
	} else {
		for c := range s.idle {
			set[c] = struct{}{}
		}
	}
	// We have to drop the shard lock here to avoid deadlock: we cannot
	// acquire the shard lock after the connection lock, and the closeIfIdle
	// call below will grab a connection lock.
	s.mu.Unlock()

	for c := range set {
		// This might return an error (from Close), but I don't think we
		// can do anything about it, so let's just pretend it didn't
		// happen. (I also expect that most errors returned in this way
		// are going to be pretty boring)
		if all {
			c.Close()
		} else {
			c.closeIfIdle()
		}
	}
}

func (s *shard) wait() {
	s.wg.Wait()
}
