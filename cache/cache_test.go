package cache

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

func TestCache_BasicOperations(t *testing.T) {
	c := New[string, string](WithCleanupInterval(100 * time.Millisecond))
	defer c.Close()

	// Get non-existent
	val, ok := c.Get("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, val)

	// Set and Get
	c.Set("key1", "val1", 1*time.Minute)
	val, ok = c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "val1", val)
	assert.Equal(t, 1, c.Len())

	// Has
	assert.True(t, c.Has("key1"))
	assert.False(t, c.Has("nonexistent"))

	// Delete
	c.Delete("key1")
	val, ok = c.Get("key1")
	assert.False(t, ok)
	assert.Empty(t, val)
	assert.Equal(t, 0, c.Len())

	// Clear
	c.Set("k1", "v1", 1*time.Minute)
	c.Set("k2", "v2", 1*time.Minute)
	assert.Equal(t, 2, c.Len())
	c.Clear()
	assert.Equal(t, 0, c.Len())
}

func TestCache_TTLExpiration(t *testing.T) {
	c := New[string, int](WithCleanupInterval(50 * time.Millisecond))
	defer c.Close()

	// Expire fast
	c.Set("short", 42, 30*time.Millisecond)
	// Never expire (ttl <= 0 means no expiration)
	c.Set("forever", 100, 0)

	val, ok := c.Get("short")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	time.Sleep(60 * time.Millisecond)

	// "short" should be expired when fetched
	val, ok = c.Get("short")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// "forever" must still be alive
	val, ok = c.Get("forever")
	assert.True(t, ok)
	assert.Equal(t, 100, val)
}

func TestCache_BackgroundCleanup(t *testing.T) {
	c := New[string, int](WithCleanupInterval(30 * time.Millisecond))
	defer c.Close()

	c.Set("exp1", 1, 20*time.Millisecond)
	c.Set("exp2", 2, 20*time.Millisecond)
	c.Set("keep", 3, 500*time.Millisecond)

	assert.Equal(t, 3, c.Len())

	// Wait for background eviction cycle
	time.Sleep(80 * time.Millisecond)

	// Expired keys should have been purged by cleaner
	assert.Equal(t, 1, c.Len())
	val, ok := c.Get("keep")
	assert.True(t, ok)
	assert.Equal(t, 3, val)
}

func TestCache_GetOrCompute_Singleflight(t *testing.T) {
	c := New[string, string]()
	defer c.Close()

	var computeCount atomic.Int32
	var wg sync.WaitGroup

	const numGoroutines = 50
	results := make([]string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val, err := c.GetOrCompute(context.Background(), "flight_key", 1*time.Minute, func(ctx context.Context) (string, error) {
				computeCount.Add(1)
				time.Sleep(50 * time.Millisecond) // simulate heavy DB or remote query
				return "computed_result", nil
			})
			require.NoError(t, err)
			results[idx] = val
		}(i)
	}

	wg.Wait()

	// Exactly 1 compute execution should occur across all 50 concurrent requests
	assert.Equal(t, int32(1), computeCount.Load(), "Compute func must only be invoked once")
	for _, res := range results {
		assert.Equal(t, "computed_result", res)
	}

	// Subsequent call should hit cache directly without compute
	val, ok := c.Get("flight_key")
	assert.True(t, ok)
	assert.Equal(t, "computed_result", val)
}

func TestCache_GetOrCompute_Error(t *testing.T) {
	c := New[string, string]()
	defer c.Close()

	customErr := errors.New("computation failed")
	val, err := c.GetOrCompute(context.Background(), "err_key", 1*time.Minute, func(ctx context.Context) (string, error) {
		return "", customErr
	})

	assert.ErrorIs(t, err, customErr)
	assert.Empty(t, val)

	// Key should not be cached on error
	_, ok := c.Get("err_key")
	assert.False(t, ok)
}

func TestCache_ConcurrentRace(t *testing.T) {
	c := New[int, int](WithCleanupInterval(20 * time.Millisecond))
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := j % 10
				c.Set(key, workerID*1000+j, 50*time.Millisecond)
				_, _ = c.Get(key)
				_ = c.Has(key)
				if j%5 == 0 {
					c.Delete(key)
				}
				if j%10 == 0 {
					_, _ = c.GetOrCompute(context.Background(), key, 50*time.Millisecond, func(ctx context.Context) (int, error) {
						return 999, nil
					})
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_Close_Idempotent(t *testing.T) {
	c := New[string, string](WithCleanupInterval(50 * time.Millisecond))
	assert.NoError(t, c.Close())
	assert.NoError(t, c.Close()) // Second call should not panic or error
}

func BenchmarkCache_Get(b *testing.B) {
	c := New[string, string]()
	defer c.Close()

	c.Set("bench", "value", 1*time.Hour)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Get("bench")
		}
	})
}

func BenchmarkCache_Set(b *testing.B) {
	c := New[string, string]()
	defer c.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			c.Set(fmt.Sprintf("key_%d", i%100), "value", 1*time.Hour)
		}
	})
}
