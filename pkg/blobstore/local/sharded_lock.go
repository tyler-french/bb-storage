package local

import (
	"sort"
	"sync"
)

// ShardedLock is a lock that shards its internal locks based on a key.
// This allows for fine-grained locking to reduce contention.
type ShardedLock struct {
	Shards []sync.RWMutex
}

// NewShardedLock creates a new ShardedLock with the specified number of shards.
// The number of shards must be a power of 2 for efficient key distribution.
func NewShardedLock(numShards int) *ShardedLock {
	if numShards <= 0 || (numShards&(numShards-1)) != 0 {
		panic("number of shards must be a positive power of 2")
	}
	return &ShardedLock{
		Shards: make([]sync.RWMutex, numShards),
	}
}

// GetShard returns the shard index for a given key.
func (l *ShardedLock) GetShard(key Key) int {
	// Use the first 4 bytes of the key to determine the shard
	// The key is a SHA-256 hash, so it's already well-distributed
	// We use & (len(l.Shards) - 1) to ensure the result is within bounds
	// This works because len(l.Shards) is always a power of 2
	hash := uint32(key[0]) | uint32(key[1])<<8 | uint32(key[2])<<16 | uint32(key[3])<<24
	return int(hash & uint32(len(l.Shards)-1))
}

// RLock acquires a read lock for the given key.
func (l *ShardedLock) RLock(key Key) {
	l.Shards[l.GetShard(key)].RLock()
}

// RUnlock releases a read lock for the given key.
func (l *ShardedLock) RUnlock(key Key) {
	l.Shards[l.GetShard(key)].RUnlock()
}

// Lock acquires a write lock for the given key.
func (l *ShardedLock) Lock(key Key) {
	l.Shards[l.GetShard(key)].Lock()
}

// Unlock releases a write lock for the given key.
func (l *ShardedLock) Unlock(key Key) {
	l.Shards[l.GetShard(key)].Unlock()
}

// GlobalLock acquires write locks for all shards.
// This should be used sparingly as it blocks all operations.
func (l *ShardedLock) GlobalLock() {
	for i := range l.Shards {
		l.Shards[i].Lock()
	}
}

// GlobalUnlock releases write locks for all shards.
func (l *ShardedLock) GlobalUnlock() {
	for i := range l.Shards {
		l.Shards[i].Unlock()
	}
}

// MultiLock acquires write locks for multiple keys in a consistent order to prevent deadlocks.
// The locks must be released with MultiUnlock.
func (l *ShardedLock) MultiLock(keys ...Key) {
	// Gather shard indices for all keys, avoiding duplicates
	shardIndices := make(map[int]struct{})
	for _, key := range keys {
		shardIndices[l.GetShard(key)] = struct{}{}
	}

	// Sort the indices to ensure consistent locking order
	indices := make([]int, 0, len(shardIndices))
	for idx := range shardIndices {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	// Lock all shards in sorted order
	for _, idx := range indices {
		l.Shards[idx].Lock()
	}
}

// MultiUnlock releases write locks for multiple keys.
func (l *ShardedLock) MultiUnlock(keys ...Key) {
	// Track which shards we've already unlocked to avoid redundant unlocks
	unlocked := make(map[int]struct{})
	for _, key := range keys {
		idx := l.GetShard(key)
		if _, ok := unlocked[idx]; !ok {
			l.Shards[idx].Unlock()
			unlocked[idx] = struct{}{}
		}
	}
}

// MultiRLock acquires read locks for multiple keys in a consistent order to prevent deadlocks.
// The locks must be released with MultiRUnlock.
func (l *ShardedLock) MultiRLock(keys ...Key) {
	// Gather shard indices for all keys, avoiding duplicates
	shardIndices := make(map[int]struct{})
	for _, key := range keys {
		shardIndices[l.GetShard(key)] = struct{}{}
	}

	// Sort the indices to ensure consistent locking order
	indices := make([]int, 0, len(shardIndices))
	for idx := range shardIndices {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	// Lock all shards in sorted order
	for _, idx := range indices {
		l.Shards[idx].RLock()
	}
}

// MultiRUnlock releases read locks for multiple keys.
func (l *ShardedLock) MultiRUnlock(keys ...Key) {
	// Track which shards we've already unlocked to avoid redundant unlocks
	unlocked := make(map[int]struct{})
	for _, key := range keys {
		idx := l.GetShard(key)
		if _, ok := unlocked[idx]; !ok {
			l.Shards[idx].RUnlock()
			unlocked[idx] = struct{}{}
		}
	}
}
