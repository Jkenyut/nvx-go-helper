package syncutil

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyedMutex_SerialPerKey(t *testing.T) {
	km := NewKeyedMutex[string]()

	var activeCount atomic.Int32
	var maxActive atomic.Int32
	var wg sync.WaitGroup

	const numGoroutines = 20
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := km.Lock("user_123")
			defer unlock()

			cur := activeCount.Add(1)
			// Track max concurrency for same key
			for {
				oldMax := maxActive.Load()
				if cur <= oldMax || maxActive.CompareAndSwap(oldMax, cur) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			activeCount.Add(-1)
		}()
	}

	wg.Wait()
	// Goroutines sharing the exact same key must run strictly sequentially (maxActive == 1)
	assert.Equal(t, int32(1), maxActive.Load())
	assert.Equal(t, 0, km.Len(), "Key entry must be cleaned up when unused")
}

func TestKeyedMutex_ParallelDifferentKeys(t *testing.T) {
	km := NewKeyedMutex[int]()

	var activeCount atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	const numKeys = 10
	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			unlock := km.Lock(key)
			defer unlock()

			cur := activeCount.Add(1)
			for {
				oldMax := maxConcurrent.Load()
				if cur <= oldMax || maxConcurrent.CompareAndSwap(oldMax, cur) {
					break
				}
			}

			time.Sleep(30 * time.Millisecond)
			activeCount.Add(-1)
		}(i)
	}

	wg.Wait()
	// Different keys should execute concurrently
	assert.Greater(t, maxConcurrent.Load(), int32(1), "Different keys must run in parallel")
	assert.Equal(t, 0, km.Len(), "All key entries must be cleaned up")
}

func TestKeyedMutex_TryLock(t *testing.T) {
	km := NewKeyedMutex[string]()

	unlock1, ok1 := km.TryLock("resource")
	require.True(t, ok1)
	require.NotNil(t, unlock1)

	// Second attempt while locked should fail
	unlock2, ok2 := km.TryLock("resource")
	assert.False(t, ok2)
	assert.Nil(t, unlock2)

	unlock1()

	// Third attempt after unlock should succeed
	unlock3, ok3 := km.TryLock("resource")
	assert.True(t, ok3)
	require.NotNil(t, unlock3)
	unlock3()

	assert.Equal(t, 0, km.Len())
}
