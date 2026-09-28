package syncutil

import (
	"context"
	"errors"
	"fmt"
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

func TestForEach_Success(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var sum atomic.Int64
	var activeWorkers atomic.Int32
	var maxWorkers atomic.Int32

	const limit = 3
	err := ForEach(context.Background(), items, limit, func(ctx context.Context, item int) error {
		cur := activeWorkers.Add(1)
		for {
			old := maxWorkers.Load()
			if cur <= old || maxWorkers.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		sum.Add(int64(item))
		activeWorkers.Add(-1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int64(55), sum.Load())
	assert.LessOrEqual(t, maxWorkers.Load(), int32(limit), "Worker count must not exceed limit")
}

func TestForEach_Empty(t *testing.T) {
	err := ForEach(context.Background(), []string{}, 2, func(ctx context.Context, item string) error {
		return errors.New("should not be called")
	})
	assert.NoError(t, err)
}

func TestForEach_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	items := []int{1, 2, 3}
	err := ForEach(ctx, items, 2, func(ctx context.Context, item int) error {
		return nil
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestForEach_Error(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	errExpected := errors.New("boom")

	err := ForEach(context.Background(), items, 2, func(ctx context.Context, item int) error {
		if item == 3 {
			return errExpected
		}
		return nil
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errExpected)
}

func TestMap_Success(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	results, err := Map(context.Background(), items, 2, func(ctx context.Context, item int) (string, error) {
		return fmt.Sprintf("val-%d", item*2), nil
	})

	require.NoError(t, err)
	expected := []string{"val-2", "val-4", "val-6", "val-8", "val-10"}
	assert.Equal(t, expected, results)
}

func TestMap_Error(t *testing.T) {
	items := []int{1, 2, 3, 4}
	targetErr := errors.New("map error")

	results, err := Map(context.Background(), items, 2, func(ctx context.Context, item int) (string, error) {
		if item == 2 {
			return "", targetErr
		}
		return fmt.Sprintf("%d", item), nil
	})

	assert.ErrorIs(t, err, targetErr)
	assert.Nil(t, results)
}
