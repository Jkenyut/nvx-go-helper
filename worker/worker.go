// Package worker provides a generic, concurrent worker pool implementation
// with cancellation, timeouts, and ordered result streaming.
//
// It is designed for processing large batches of jobs (e.g., CSV imports,
// data migrations, bulk API calls) where:
//   - Concurrency is needed for speed
//   - Results must be mapped 1:1 to inputs
//   - Global and per-job timeouts are required
//   - Panics must be caught safely
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

// WorkerPoolConfig holds configuration options for the worker pool.
type WorkerPoolConfig struct {
	NumWorkers    int                        // Concurrent workers count (default: 2)
	WorkerTimeout time.Duration              // Timeout for a single job execution (default: 15s)
	GlobalTimeout time.Duration              // Total timeout for the entire batch (default: 30s)
	StopOnError   bool                       // If true, the pool shuts down on the first error
	PreserveOrder bool                       // If true, RunGenericWorkerPool returns results in the exact order of input jobs
	OnProgress    func(completed, total int) // Optional callback for progress tracking
}

// ErrSkipped indicates a job was not processed because the pool was cancelled/timed out,
// or a previous job failed (if StopOnError is true).
var ErrSkipped = fmt.Errorf("job not processed (cancelled or skipped)")

// RunGenericWorkerPoolStream executes a batch of jobs concurrently and streams results.
//
// Key features:
//   - **Ordered Results**: Results are NOT guaranteed to be in order, but each Result contains the ID of the source Job.
//   - **Concurrency Control**: Use cfg.NumWorkers to limit parallelism.
//   - **Timeouts**: Enforces both GlobalTimeout (whole batch) and WorkerTimeout (per item).
//   - **Safety**: Recovers from panics in worker function to prevent crash.
//
// The workerFunc must accept a context (which respects timeouts) and the job data.
// It returns the result R and an error.
//
// Returns a read-only channel of Results. The channel is closed when all jobs are finished or timed out.
func RunGenericWorkerPoolStream[K comparable, T any, R any](
	ctx context.Context,
	jobs []Job[K, T],
	workerFunc func(context.Context, K, T) (R, error),
	globalSemaphore chan struct{},
	cfg WorkerPoolConfig,
) <-chan Result[K, R] {

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
			if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
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

	// Apply configuration defaults
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 2
	}

	if cfg.GlobalTimeout == 0 {
		cfg.GlobalTimeout = 30 * time.Second
	}

	if cfg.WorkerTimeout == 0 {
		cfg.WorkerTimeout = 15 * time.Second
	}

	if cfg.WorkerTimeout > 0 && cfg.GlobalTimeout > 0 && cfg.WorkerTimeout > cfg.GlobalTimeout {
		cfg.WorkerTimeout = cfg.GlobalTimeout
	}

	outCh := make(chan Result[K, R], len(jobs))
	jobCh := make(chan Job[K, T])

	var poolCtx context.Context
	var cancelPool context.CancelCauseFunc

	poolCtx, cancelPool = context.WithCancelCause(ctx)

	var timeoutTimer *time.Timer
	if cfg.GlobalTimeout >= 0 {
		timeoutTimer = time.AfterFunc(cfg.GlobalTimeout, func() {
			cancelPool(fmt.Errorf("global timeout of %v exceeded", cfg.GlobalTimeout))
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
		if cause := context.Cause(poolCtx); cause != nil && cause != context.Canceled {
			return fmt.Errorf("%w: %w", ErrSkipped, cause)
		}
		return ErrSkipped
	}

	// Worker goroutines
	workerWG.Add(cfg.NumWorkers)
	for i := 0; i < cfg.NumWorkers; i++ {
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
				if globalSemaphore != nil {
					select {
					case globalSemaphore <- struct{}{}:
					case <-poolCtx.Done():
						sendResult(Result[K, R]{ID: job.ID, Err: getSkipErr()})
						continue
					}
				}

				func() {
					if globalSemaphore != nil {
						defer func() { <-globalSemaphore }()
					}

					defer func() {
						if r := recover(); r != nil {
							panicErr := fmt.Errorf("panic: %v", r)
							sendResult(Result[K, R]{ID: job.ID, Err: panicErr})
							if cfg.StopOnError {
								safeCancelPool(fmt.Errorf("panic in job %v: %v", job.ID, r))
							}
						}
					}()

					var taskCtx context.Context
					var cancel context.CancelFunc
					if cfg.WorkerTimeout < 0 {
						taskCtx, cancel = context.WithCancel(poolCtx)
					} else {
						taskCtx, cancel = context.WithTimeoutCause(poolCtx, cfg.WorkerTimeout, fmt.Errorf("worker timeout of %v exceeded", cfg.WorkerTimeout))
					}
					defer cancel()

					res, err := workerFunc(taskCtx, job.ID, job.Data)

					if err != nil && cfg.StopOnError {
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
		cancelPool(nil) // Ensure cleanup
		close(outCh)
	}()

	return outCh
}

// RunGenericWorkerPool executes a batch of jobs concurrently and waits for all of them to complete.
// It returns a slice of all results and an aggregated error if any job failed.
func RunGenericWorkerPool[K comparable, T any, R any](
	ctx context.Context,
	jobs []Job[K, T],
	workerFunc func(context.Context, K, T) (R, error),
	globalSemaphore chan struct{},
	cfg WorkerPoolConfig,
) ([]Result[K, R], error) {

	outCh := RunGenericWorkerPoolStream(ctx, jobs, workerFunc, globalSemaphore, cfg)

	var results []Result[K, R]
	var errs []error
	var completed int
	total := len(jobs)

	if cfg.PreserveOrder {
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
			if cfg.OnProgress != nil {
				completed++
				cfg.OnProgress(completed, total)
			}
		}
	} else {
		results = make([]Result[K, R], 0, len(jobs))
		for res := range outCh {
			results = append(results, res)
			if res.Err != nil {
				errs = append(errs, res.Err)
			}
			if cfg.OnProgress != nil {
				completed++
				cfg.OnProgress(completed, total)
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
