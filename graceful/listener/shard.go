package listener

import "sync"

type shard struct {
	l *T

	mu    sync.Mutex
	set   map[*conn]struct{}
	wg    sync.WaitGroup
	drain bool

	// We pack shards together in an array, but we don't want them packed
	// too closely, since we want to give each shard a dedicated CPU cache
	// line. This amount of padding works out well for the common case of
	// x64 processors (64-bit pointers with a 64-byte cache line).
	_ [12]byte
}

// We pretty aggressively preallocate set entries in the hopes that we never
// have to allocate memory with the lock held. This is definitely a premature
// optimization and is probably misguided, but luckily it costs us essentially
// nothing.
const prealloc = 2048

func (s *shard) init(l *T) {
	s.l = l
	s.set = make(map[*conn]struct{}, prealloc)
}

func (s *shard) markIdle(c *conn) (shouldClose bool) {
	s.mu.Lock()
	if s.drain {
		s.mu.Unlock()
		return true
	}
	s.set[c] = struct{}{}
	s.mu.Unlock()
	return false
}

func (s *shard) markInUse(c *conn) {
	s.mu.Lock()
	delete(s.set, c)
	s.mu.Unlock()
}

func (s *shard) closeIdle(drain bool) {
	s.mu.Lock()
	if drain {
		s.drain = true
	}
	set := s.set
	s.set = make(map[*conn]struct{}, prealloc)
	// We have to drop the shard lock here to avoid deadlock: we cannot
	// acquire the shard lock after the connection lock, and the closeIfIdle
	// call below will grab a connection lock.
	s.mu.Unlock()

	for conn := range set {
		// This might return an error (from Close), but I don't think we
		// can do anything about it, so let's just pretend it didn't
		// happen. (I also expect that most errors returned in this way
		// are going to be pretty boring)
		conn.closeIfIdle()
	}
}

func (s *shard) wait() {
	s.wg.Wait()
}
