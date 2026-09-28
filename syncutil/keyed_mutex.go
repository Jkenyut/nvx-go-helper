package syncutil

import (
	"sync"
)

type refMutex struct {
	mu  sync.Mutex
	ref int
}

// KeyedMutex provides per-key mutual exclusion locks with automatic reference-counted memory cleanup.
// This allows concurrent execution across different keys while guaranteeing serialized execution
// for any identical key (e.g. per-user locking or per-order locking).
type KeyedMutex[K comparable] struct {
	mu    sync.Mutex
	locks map[K]*refMutex
}

// NewKeyedMutex instantiates a KeyedMutex.
func NewKeyedMutex[K comparable]() *KeyedMutex[K] {
	return &KeyedMutex[K]{
		locks: make(map[K]*refMutex),
	}
}

// Lock acquires the lock for key and returns an unlock function.
// Usage: defer km.Lock(key)()
func (km *KeyedMutex[K]) Lock(key K) func() {
	km.mu.Lock()
	entry, exists := km.locks[key]
	if !exists {
		entry = &refMutex{}
		km.locks[key] = entry
	}
	entry.ref++
	km.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		km.mu.Lock()
		entry.ref--
		if entry.ref <= 0 {
			delete(km.locks, key)
		}
		km.mu.Unlock()
	}
}

// TryLock attempts to acquire the lock for key without blocking.
// If successful, returns the unlock function and true; otherwise returns nil and false.
func (km *KeyedMutex[K]) TryLock(key K) (func(), bool) {
	km.mu.Lock()
	entry, exists := km.locks[key]
	if !exists {
		entry = &refMutex{}
		km.locks[key] = entry
	}

	if !entry.mu.TryLock() {
		if !exists {
			delete(km.locks, key)
		}
		km.mu.Unlock()
		return nil, false
	}

	entry.ref++
	km.mu.Unlock()

	return func() {
		entry.mu.Unlock()

		km.mu.Lock()
		entry.ref--
		if entry.ref <= 0 {
			delete(km.locks, key)
		}
		km.mu.Unlock()
	}, true
}

// Len returns the current number of active locked keys.
func (km *KeyedMutex[K]) Len() int {
	km.mu.Lock()
	n := len(km.locks)
	km.mu.Unlock()
	return n
}
