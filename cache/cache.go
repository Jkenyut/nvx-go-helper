// Package cache provides a thread-safe, high-performance in-memory generic cache
// with automatic TTL expiration, background cleanup, and singleflight computation
// to eliminate cache stampede under heavy concurrency.
package cache

import (
	"context"
	"sync"
	"time"
)

type item[V any] struct {
	value     V
	expiresAt int64 // Unix nanoseconds, 0 indicates no expiration
}

func (it item[V]) isExpired(now int64) bool {
	return it.expiresAt > 0 && now >= it.expiresAt
}

type flightCall[V any] struct {
	wg  sync.WaitGroup
	val V
	err error
}

// Option configures cache behavior.
type Option func(*config)

type config struct {
	cleanupInterval time.Duration
}

// WithCleanupInterval sets the period between background expired-key sweeps.
// Set to <= 0 to disable the background cleaner goroutine. Default is 1 minute.
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *config) {
		c.cleanupInterval = interval
	}
}

// Cache is a thread-safe, generic in-memory key-value store.
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]item[V]
	flightMu sync.Mutex
	flights  map[K]*flightCall[V]

	closeOnce sync.Once
	stopChan  chan struct{}
	doneChan  chan struct{}
}

// New creates a new initialized Cache instance.
func New[K comparable, V any](opts ...Option) *Cache[K, V] {
	cfg := config{
		cleanupInterval: 1 * time.Minute,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	c := &Cache[K, V]{
		items:    make(map[K]item[V]),
		flights:  make(map[K]*flightCall[V]),
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	if cfg.cleanupInterval > 0 {
		go c.cleanupLoop(cfg.cleanupInterval)
	} else {
		close(c.doneChan)
	}

	return c
}

// Set stores a key-value pair with a specific TTL.
// A ttl <= 0 means the item will never expire.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	c.items[key] = item[V]{
		value:     value,
		expiresAt: expiresAt,
	}
	c.mu.Unlock()
}

// Get retrieves the value associated with key if present and not expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	now := time.Now().UnixNano()

	c.mu.RLock()
	it, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return zero, false
	}

	if it.isExpired(now) {
		// Lazily remove expired item
		c.mu.Lock()
		if current, ok := c.items[key]; ok && current.isExpired(now) {
			delete(c.items, key)
		}
		c.mu.Unlock()
		return zero, false
	}

	return it.value, true
}

// Has checks whether a non-expired key exists in the cache.
func (c *Cache[K, V]) Has(key K) bool {
	_, found := c.Get(key)
	return found
}

// Delete removes a key and its value from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Clear flushes all items from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	c.items = make(map[K]item[V])
	c.mu.Unlock()
}

// Len returns the current total number of cached entries (including uncollected expired keys).
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// GetOrCompute retrieves the value for key from cache if present. If not present or expired,
// it executes fn using a singleflight mechanism so that concurrent requests for the same key
// share the exact same execution, preventing cache stampedes. On success, the value is cached.
func (c *Cache[K, V]) GetOrCompute(
	ctx context.Context,
	key K,
	ttl time.Duration,
	fn func(context.Context) (V, error),
) (V, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	c.flightMu.Lock()
	call, active := c.flights[key]
	if active {
		// An existing computation is already in flight for this key
		c.flightMu.Unlock()
		call.wg.Wait()
		if call.err != nil {
			var zero V
			return zero, call.err
		}
		return call.val, nil
	}

	// First caller starts the in-flight job
	call = &flightCall[V]{}
	call.wg.Add(1)
	c.flights[key] = call
	c.flightMu.Unlock()

	defer func() {
		c.flightMu.Lock()
		delete(c.flights, key)
		c.flightMu.Unlock()
		call.wg.Done()
	}()

	val, err := fn(ctx)
	call.val = val
	call.err = err

	if err != nil {
		var zero V
		return zero, err
	}

	c.Set(key, val, ttl)
	return val, nil
}

// Close stops the background expiration cleaner goroutine and frees resources.
func (c *Cache[K, V]) Close() error {
	c.closeOnce.Do(func() {
		close(c.stopChan)
		<-c.doneChan
	})
	return nil
}

func (c *Cache[K, V]) cleanupLoop(interval time.Duration) {
	defer close(c.doneChan)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.deleteExpired()
		}
	}
}

func (c *Cache[K, V]) deleteExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	for k, it := range c.items {
		if it.isExpired(now) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
