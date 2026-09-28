// Package syncutil provides high-performance synchronization and concurrency primitives
// including reference-counted KeyedMutex and bounded parallel slice processors.
package syncutil

import (
	"context"
	"errors"
	"sync"
)

// ErrZeroLimit is returned when concurrency limit is less than or equal to zero.
var ErrZeroLimit = errors.New("syncutil: concurrency limit must be greater than zero")

// ForEach executes fn concurrently across items with a maximum of limit goroutines running at once.
// If any invocation returns an error or if the context is cancelled, execution stops early and the error is returned.
func ForEach[T any](ctx context.Context, items []T, limit int, fn func(context.Context, T) error) error {
	if len(items) == 0 {
		return nil
	}
	if limit <= 0 {
		return ErrZeroLimit
	}

	sem := make(chan struct{}, limit)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

forEachLoop:
	for _, item := range items {
		select {
		case <-ctx.Done():
			break forEachLoop
		case sem <- struct{}{}:
			wg.Add(1)
			go func(val T) {
				defer func() {
					<-sem
					wg.Done()
				}()

				if err := fn(ctx, val); err != nil {
					once.Do(func() {
						firstErr = err
						cancel()
					})
				}
			}(item)
		}
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

// Map executes fn concurrently on each item with a concurrency limit and preserves the original item ordering in the output slice.
// If an error occurs, execution terminates early and the error is returned.
func Map[T any, R any](ctx context.Context, items []T, limit int, fn func(context.Context, T) (R, error)) ([]R, error) {
	if len(items) == 0 {
		return []R{}, nil
	}
	if limit <= 0 {
		return nil, ErrZeroLimit
	}

	results := make([]R, len(items))
	sem := make(chan struct{}, limit)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

mapLoop:
	for idx, item := range items {
		select {
		case <-ctx.Done():
			break mapLoop
		case sem <- struct{}{}:
			wg.Add(1)
			go func(i int, val T) {
				defer func() {
					<-sem
					wg.Done()
				}()

				res, err := fn(ctx, val)
				if err != nil {
					once.Do(func() {
						firstErr = err
						cancel()
					})
					return
				}
				results[i] = res
			}(idx, item)
		}
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
