// Package worker provides a generic, concurrent worker pool implementation
// with cancellation, timeouts, order preservation, and streaming results.
//
// It is designed for processing batches of jobs (e.g., bulk API calls, migrations)
// with idiomatic functional options, panic recovery, and bounded concurrency.
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Job represents a generic job input.
// K is the comparable type of the identifier.
// T is the type of data to be processed.
type Job[K comparable, T any] struct {
	ID   K // Unique identifier to map result back to input
	Data T // Payload to be processed
}

// Result represents the output of processing a Job.
// K is the comparable type of the identifier.
// R is the type of the result value.
type Result[K comparable, R any] struct {
	ID    K     // Matches Job.ID, allowing O(1) correlation
	Value R     // Success result (if any)
	Err   error // Error result (if any) or panic error
}

type config struct {
	numWorkers      int
	workerTimeout   time.Duration
	globalTimeout   time.Duration
	stopOnError     bool
	preserveOrder   bool
	onProgress      func(completed, total int)
	globalSemaphore chan struct{}
}

// Option configures worker pool behavior.
type Option func(*config)

// WithWorkers sets the number of concurrent workers (default: 2).
func WithWorkers(num int) Option {
	return func(c *config) {
		c.numWorkers = num
	}
}

// WithWorkerTimeout sets the execution timeout per job (default: 15s).
func WithWorkerTimeout(d time.Duration) Option {
	return func(c *config) {
		c.workerTimeout = d
	}
}

// WithGlobalTimeout sets the total timeout for the entire batch (default: 30s).
func WithGlobalTimeout(d time.Duration) Option {
	return func(c *config) {
		c.globalTimeout = d
	}
}

// WithStopOnError configures the pool to cancel on the first encountered error.
func WithStopOnError(stop bool) Option {
	return func(c *config) {
		c.stopOnError = stop
	}
}

// WithPreserveOrder configures whether results are returned strictly in the input jobs order.
func WithPreserveOrder(preserve bool) Option {
	return func(c *config) {
		c.preserveOrder = preserve
	}
}

// WithOnProgress sets a progress reporting callback invoked upon each completed job.
func WithOnProgress(fn func(completed, total int)) Option {
	return func(c *config) {
		c.onProgress = fn
	}
}

// WithGlobalSemaphore provides an optional shared semaphore across multiple worker pools.
func WithGlobalSemaphore(sem chan struct{}) Option {
	return func(c *config) {
		c.globalSemaphore = sem
	}
}

func defaultConfig(opts ...Option) config {
	cfg := config{
		numWorkers:    2,
		workerTimeout: 15 * time.Second,
		globalTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.numWorkers <= 0 {
		cfg.numWorkers = 2
	}
	if cfg.globalTimeout == 0 {
		cfg.globalTimeout = 30 * time.Second
	}
	if cfg.workerTimeout == 0 {
		cfg.workerTimeout = 15 * time.Second
	}
	if cfg.workerTimeout > 0 && cfg.globalTimeout > 0 && cfg.workerTimeout > cfg.globalTimeout {
		cfg.workerTimeout = cfg.globalTimeout
	}

	return cfg
}

// ErrSkipped indicates a job was not processed because the pool was cancelled/timed out,
// or a previous job failed (if WithStopOnError is enabled).
var ErrSkipped = fmt.Errorf("job not processed (cancelled or skipped)")

// Stream executes a batch of jobs concurrently using functional options and streams results.
// The channel is closed when all jobs are completed, failed, or cancelled.
func Stream[K comparable, T any, R any](
	ctx context.Context,
	jobs []Job[K, T],
	workerFunc func(context.Context, K, T) (R, error),
	opts ...Option,
) <-chan Result[K, R] {
	cfg := defaultConfig(opts...)

	if len(jobs) == 0 {
		outCh := make(chan Result[K, R])
		close(outCh)
		return outCh
	}

	// Validate duplicate IDs
	seenIDs := make(map[K]bool, len(jobs))
	for _, job := range jobs {
		if seenIDs[job.ID] {
			outCh := make(chan Result[K, R], len(jobs))
			go func() {
				err := fmt.Errorf("duplicate job ID detected: %v (all jobs rejected)", job.ID)
				for _, j := range jobs {
					outCh <- Result[K, R]{ID: j.ID, Err: err}
				}
				close(outCh)
			}()
			return outCh
		}
		seenIDs[job.ID] = true
	}

	// Check parent context
	select {
	case <-ctx.Done():
		outCh := make(chan Result[K, R], len(jobs))
		go func() {
			errToSend := ErrSkipped
			if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, context.Canceled) {
				errToSend = fmt.Errorf("%w: %w", ErrSkipped, cause)
			}
			for _, job := range jobs {
				outCh <- Result[K, R]{ID: job.ID, Err: errToSend}
			}
			close(outCh)
		}()
		return outCh
	default:
	}

	outCh := make(chan Result[K, R], len(jobs))
	jobCh := make(chan Job[K, T])

	poolCtx, cancelPool := context.WithCancelCause(ctx)

	var timeoutTimer *time.Timer
	if cfg.globalTimeout >= 0 {
		timeoutTimer = time.AfterFunc(cfg.globalTimeout, func() {
			cancelPool(fmt.Errorf("global timeout of %v exceeded", cfg.globalTimeout))
		})
	}

	var cancelOnce sync.Once
	safeCancelPool := func(cause error) {
		cancelOnce.Do(func() {
			cancelPool(cause)
		})
	}

	var workerWG sync.WaitGroup
	var feederWG sync.WaitGroup

	sendResult := func(result Result[K, R]) {
		outCh <- result
	}

	getSkipErr := func() error {
		if cause := context.Cause(poolCtx); cause != nil && !errors.Is(cause, context.Canceled) {
			return fmt.Errorf("%w: %w", ErrSkipped, cause)
		}
		return ErrSkipped
	}

	// Worker goroutines
	workerWG.Add(cfg.numWorkers)
	for i := 0; i < cfg.numWorkers; i++ {
		go func() {
			defer workerWG.Done()

			for job := range jobCh {
				// Check context before work
				select {
				case <-poolCtx.Done():
					sendResult(Result[K, R]{ID: job.ID, Err: getSkipErr()})
					continue
				default:
				}

				// Acquire external semaphore if provided
				if cfg.globalSemaphore != nil {
					select {
					case cfg.globalSemaphore <- struct{}{}:
					case <-poolCtx.Done():
						sendResult(Result[K, R]{ID: job.ID, Err: getSkipErr()})
						continue
					}
				}

				func() {
					if cfg.globalSemaphore != nil {
						defer func() { <-cfg.globalSemaphore }()
					}

					defer func() {
						if r := recover(); r != nil {
							panicErr := fmt.Errorf("panic: %v", r)
							sendResult(Result[K, R]{ID: job.ID, Err: panicErr})
							if cfg.stopOnError {
								safeCancelPool(fmt.Errorf("panic in job %v: %v", job.ID, r))
							}
						}
					}()

					var taskCtx context.Context
					var cancel context.CancelFunc
					if cfg.workerTimeout < 0 {
						taskCtx, cancel = context.WithCancel(poolCtx)
					} else {
						taskCtx, cancel = context.WithTimeoutCause(poolCtx, cfg.workerTimeout, fmt.Errorf("worker timeout of %v exceeded", cfg.workerTimeout))
					}
					defer cancel()

					res, err := workerFunc(taskCtx, job.ID, job.Data)

					if err != nil && cfg.stopOnError {
						safeCancelPool(fmt.Errorf("error in job %v: %w", job.ID, err))
					}

					sendResult(Result[K, R]{ID: job.ID, Value: res, Err: err})
				}()
			}
		}()
	}

	// Feeder
	feederWG.Add(1)
	go func() {
		defer feederWG.Done()
		defer close(jobCh)

		for _, job := range jobs {
			select {
			case jobCh <- job:
			case <-poolCtx.Done():
				sendResult(Result[K, R]{ID: job.ID, Err: getSkipErr()})
			}
		}
	}()

	// Finalizer
	go func() {
		feederWG.Wait()
		workerWG.Wait()
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}
		cancelPool(nil)
		close(outCh)
	}()

	return outCh
}

// Run executes a batch of jobs concurrently with functional options and waits for all of them to complete.
// It returns a slice of all results and an aggregated error if any job failed.
func Run[K comparable, T any, R any](
	ctx context.Context,
	jobs []Job[K, T],
	workerFunc func(context.Context, K, T) (R, error),
	opts ...Option,
) ([]Result[K, R], error) {
	cfg := defaultConfig(opts...)
	outCh := Stream(ctx, jobs, workerFunc, opts...)

	var results []Result[K, R]
	var errs []error
	var completed int
	total := len(jobs)

	if cfg.preserveOrder {
		results = make([]Result[K, R], len(jobs))
		indexMap := make(map[K]int, len(jobs))
		for i, job := range jobs {
			indexMap[job.ID] = i
		}
		for res := range outCh {
			if idx, ok := indexMap[res.ID]; ok {
				results[idx] = res
			}
			if res.Err != nil {
				errs = append(errs, res.Err)
			}
			if cfg.onProgress != nil {
				completed++
				cfg.onProgress(completed, total)
			}
		}
	} else {
		results = make([]Result[K, R], 0, len(jobs))
		for res := range outCh {
			results = append(results, res)
			if res.Err != nil {
				errs = append(errs, res.Err)
			}
			if cfg.onProgress != nil {
				completed++
				cfg.onProgress(completed, total)
			}
		}
	}

	var finalErr error
	if len(errs) > 0 {
		var uniqueErrs []error
		seenErrs := make(map[string]bool)
		for _, e := range errs {
			msg := e.Error()
			if !seenErrs[msg] {
				seenErrs[msg] = true
				uniqueErrs = append(uniqueErrs, e)
			}
		}
		finalErr = errors.Join(uniqueErrs...)
	}

	return results, finalErr
}
