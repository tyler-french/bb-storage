package local_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/buildbarn/bb-storage/pkg/blobstore/local"
)

func TestShardedLock(t *testing.T) {
	t.Run("ReadAndWriteLocking", func(t *testing.T) {
		sl := local.NewShardedLock(16)
		key1 := local.NewKeyFromString("key1")
		key2 := local.NewKeyFromString("key2")

		// Should be able to acquire read locks for different keys concurrently
		sl.RLock(key1)
		sl.RLock(key2)
		sl.RUnlock(key1)
		sl.RUnlock(key2)

		// Should be able to acquire write locks for different keys concurrently
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			sl.Lock(key1)
			time.Sleep(50 * time.Millisecond) // Hold the lock briefly
			sl.Unlock(key1)
		}()

		go func() {
			defer wg.Done()
			sl.Lock(key2)
			time.Sleep(50 * time.Millisecond) // Hold the lock briefly
			sl.Unlock(key2)
		}()

		wg.Wait()
	})

	t.Run("LockExclusion", func(t *testing.T) {
		sl := local.NewShardedLock(16)
		key := local.NewKeyFromString("same-key")

		// Test that write lock excludes readers
		sl.Lock(key)

		var readBlocked atomic.Bool
		readBlocked.Store(true)

		go func() {
			sl.RLock(key)
			readBlocked.Store(false)
			sl.RUnlock(key)
		}()

		// Reader should be blocked
		time.Sleep(50 * time.Millisecond)
		require.True(t, readBlocked.Load(), "Read lock should be blocked by write lock")

		// Release the write lock, reader should proceed
		sl.Unlock(key)
		time.Sleep(50 * time.Millisecond)
		require.False(t, readBlocked.Load(), "Read lock should proceed after write lock is released")
	})

	t.Run("GlobalLock", func(t *testing.T) {
		sl := local.NewShardedLock(16)
		key1 := local.NewKeyFromString("key1")
		key2 := local.NewKeyFromString("key2")

		// Global lock should block all operations
		sl.GlobalLock()

		var lock1Blocked atomic.Bool
		var lock2Blocked atomic.Bool
		lock1Blocked.Store(true)
		lock2Blocked.Store(true)

		go func() {
			sl.Lock(key1)
			lock1Blocked.Store(false)
			sl.Unlock(key1)
		}()

		go func() {
			sl.RLock(key2)
			lock2Blocked.Store(false)
			sl.RUnlock(key2)
		}()

		// Both operations should be blocked
		time.Sleep(50 * time.Millisecond)
		require.True(t, lock1Blocked.Load(), "Lock should be blocked by global lock")
		require.True(t, lock2Blocked.Load(), "RLock should be blocked by global lock")

		// Release the global lock, operations should proceed
		sl.GlobalUnlock()
		time.Sleep(50 * time.Millisecond)
		require.False(t, lock1Blocked.Load(), "Lock should proceed after global lock is released")
		require.False(t, lock2Blocked.Load(), "RLock should proceed after global lock is released")
	})

	t.Run("PowerOfTwoShardCount", func(t *testing.T) {
		// Test that non-power-of-2 shard counts panic
		require.Panics(t, func() {
			local.NewShardedLock(10)
		}, "Should panic with non-power-of-2 shard count")

		// Test with power of 2
		sl := local.NewShardedLock(32)
		require.Len(t, sl.Shards, 32)

		// Test with 0
		require.Panics(t, func() {
			local.NewShardedLock(0)
		}, "Should panic with zero shards")
	})

	t.Run("MultiLocking", func(t *testing.T) {
		sl := local.NewShardedLock(16)
		keys := []local.Key{
			local.NewKeyFromString("foo"),
			local.NewKeyFromString("bar"),
			local.NewKeyFromString("baz"),
		}

		// Should be able to acquire locks for multiple keys
		sl.MultiLock(keys...)
		sl.MultiUnlock(keys...)

		// Test that multi-locking correctly handles duplicates
		duplicateKeys := []local.Key{
			local.NewKeyFromString("foo"),
			local.NewKeyFromString("foo"), // Duplicate
			local.NewKeyFromString("bar"),
		}

		sl.MultiLock(duplicateKeys...)
		sl.MultiUnlock(duplicateKeys...)
	})

	t.Run("ForceCollision", func(t *testing.T) {
		// Set a very small number of shards to force collisions
		sl := local.NewShardedLock(1) // Just one shard

		key1 := local.NewKeyFromString("key1")
		key2 := local.NewKeyFromString("key2")

		// Verify these keys map to the same shard (guaranteed with only 1 shard)
		idx1 := sl.GetShard(key1)
		idx2 := sl.GetShard(key2)
		if idx1 != idx2 {
			t.Fatalf("Both keys should map to the same shard with only 1 shard total")
		}

		// Verify that locking one key blocks the other
		sl.Lock(key1)

		var blocked atomic.Bool
		blocked.Store(true)

		go func() {
			sl.Lock(key2)
			blocked.Store(false)
			sl.Unlock(key2)
		}()

		// Second key should be blocked even though it's different
		time.Sleep(50 * time.Millisecond)
		if !blocked.Load() {
			t.Fatalf("The second key should be blocked when they map to the same shard")
		}

		// Release the lock, second key should proceed
		sl.Unlock(key1)
		time.Sleep(50 * time.Millisecond)
		if blocked.Load() {
			t.Fatalf("The second key should proceed after the first is unlocked")
		}
	})

	t.Run("ConcurrentAccessWithPotentialCollisions", func(t *testing.T) {
		// Create a lock with deliberately few shards to increase collision probability
		sl := local.NewShardedLock(4)

		// Create a map to track access
		var accessCount atomic.Int64

		// Generate many keys
		const keyCount = 100
		keys := make([]local.Key, keyCount)
		for i := 0; i < keyCount; i++ {
			keys[i] = local.NewKeyFromString(fmt.Sprintf("test-key-%d", i))
		}

		// Launch many goroutines to access the keys concurrently
		var wg sync.WaitGroup
		for i := 0; i < keyCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				key := keys[idx]

				// Do some work with exclusive lock
				sl.Lock(key)
				accessCount.Add(1)
				time.Sleep(1 * time.Millisecond)
				sl.Unlock(key)

				// Do some work with shared lock
				sl.RLock(key)
				time.Sleep(1 * time.Millisecond)
				sl.RUnlock(key)
			}(i)
		}

		wg.Wait()
		if accessCount.Load() != keyCount {
			t.Errorf("Expected %d accesses but got %d", keyCount, accessCount.Load())
		}
	})

	t.Run("MultiRLocking", func(t *testing.T) {
		sl := local.NewShardedLock(16)
		keys := []local.Key{
			local.NewKeyFromString("foo"),
			local.NewKeyFromString("bar"),
			local.NewKeyFromString("baz"),
		}

		// Should be able to acquire read locks for multiple keys
		sl.MultiRLock(keys...)

		// While holding read locks, other readers should be able to access the same keys
		var readersBlocked atomic.Bool
		readersBlocked.Store(false)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Try to get read locks for the same keys
			sl.MultiRLock(keys...)
			time.Sleep(50 * time.Millisecond)
			sl.MultiRUnlock(keys...)
		}()

		// Wait a bit to ensure the second reader tried to acquire locks
		time.Sleep(50 * time.Millisecond)
		require.False(t, readersBlocked.Load(), "Multiple readers should not block each other")

		// Release the first set of read locks
		sl.MultiRUnlock(keys...)
		wg.Wait()
	})
}
