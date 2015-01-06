// +build amd64

package listener

import (
	"testing"
	"unsafe"
)

// We pack shards together in an array, but we don't want them packed too
// closely, since we want to give each shard a dedicated CPU cache line. This
// test checks this property for x64 (which has a 64-byte cache line), which
// probably covers the majority of deployments.
//
// As always, this is probably a premature optimization.
func TestShardSize(t *testing.T) {
	s := unsafe.Sizeof(shard{})
	if s < 64 {
		t.Errorf("sizeof(shard) = %d; expected >64", s)
	}
}
