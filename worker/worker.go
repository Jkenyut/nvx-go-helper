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
	"fmt"
	"sync"
	"time"
)

// Job represents a generic job input.
// T is the type of data to be processed.
type Job[T any] struct {
	ID   int // Unique identifier (usually index) to map result back to input
	Data T   // Payload to be processed
}

// Result represents the output of processing a Job.
// R is the type of the result value.
type Result[R any] struct {
	ID    int   // Matches Job.ID, allowing O(1) correlation
	Value R     // Success result (if any)
	Err   error // Error result (if any) or panic error
}

// WorkerPoolConfig holds configuration options for the worker pool.
type WorkerPoolConfig struct {
	NumWorkers    int           // Concurrent workers count (default: 2)
	WorkerTimeout time.Duration // Timeout for a single job execution (default: 15s)
	GlobalTimeout time.Duration // Total timeout for the entire batch (default: 30s)
	StopOnError   bool          // If true, the pool shuts down on the first error
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
func RunGenericWorkerPoolStream[T any, R any](
	ctx context.Context,
	jobs []Job[T],
	workerFunc func(context.Context, T) (R, error),
	globalSemaphore chan struct{},
	cfg WorkerPoolConfig,
) <-chan Result[R] {

	if len(jobs) == 0 {
		outCh := make(chan Result[R])
		close(outCh)
		return outCh
	}

	// Validate duplicate IDs
	seenIDs := make(map[int]bool, len(jobs))
	for _, job := range jobs {
		if seenIDs[job.ID] {
			outCh := make(chan Result[R], len(jobs))
			go func() {
				err := fmt.Errorf("duplicate job ID detected: %d (all jobs rejected)", job.ID)
				for _, j := range jobs {
					outCh <- Result[R]{ID: j.ID, Err: err}
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
		outCh := make(chan Result[R], len(jobs))
		go func() {
			for _, job := range jobs {
				outCh <- Result[R]{ID: job.ID, Err: ErrSkipped}
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

	if cfg.GlobalTimeout <= 0 {
		cfg.GlobalTimeout = 30 * time.Second
	}

	if cfg.WorkerTimeout <= 0 {
		cfg.WorkerTimeout = 15 * time.Second
		// Cap at GlobalTimeout if smaller
		if cfg.WorkerTimeout > cfg.GlobalTimeout {
			cfg.WorkerTimeout = cfg.GlobalTimeout
		}
	}

	// Ensure global timeout is safe relative to worker timeout
	if cfg.GlobalTimeout < cfg.WorkerTimeout {
		cfg.GlobalTimeout = cfg.WorkerTimeout * 2
	}

	outCh := make(chan Result[R], len(jobs))
	jobCh := make(chan Job[T])

	poolCtx, cancelPool := context.WithTimeout(ctx, cfg.GlobalTimeout)

	var cancelOnce sync.Once
	safeCancelPool := func() {
		cancelOnce.Do(func() {
			cancelPool()
		})
	}

	var workerWG sync.WaitGroup
	var feederWG sync.WaitGroup
	sentResults := &sync.Map{}

	sendResult := func(result Result[R]) {
		if _, alreadySent := sentResults.LoadOrStore(result.ID, true); !alreadySent {
			outCh <- result
		}
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
					sendResult(Result[R]{ID: job.ID, Err: ErrSkipped})
					continue
				default:
				}

				// Acquire external semaphore if provided
				if globalSemaphore != nil {
					select {
					case globalSemaphore <- struct{}{}:
					case <-poolCtx.Done():
						sendResult(Result[R]{ID: job.ID, Err: ErrSkipped})
						continue
					}
				}

				func() {
					if globalSemaphore != nil {
						defer func() { <-globalSemaphore }()
					}

					defer func() {
						if r := recover(); r != nil {
							sendResult(Result[R]{ID: job.ID, Err: fmt.Errorf("panic: %v", r)})
							if cfg.StopOnError {
								safeCancelPool()
							}
						}
					}()

					taskCtx, cancel := context.WithTimeout(poolCtx, cfg.WorkerTimeout)
					defer cancel()

					res, err := workerFunc(taskCtx, job.Data)

					if err != nil && cfg.StopOnError {
						safeCancelPool()
					}

					sendResult(Result[R]{ID: job.ID, Value: res, Err: err})
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
				sendResult(Result[R]{ID: job.ID, Err: ErrSkipped})
			}
		}
	}()

	// Finalizer
	go func() {
		feederWG.Wait()
		workerWG.Wait()
		cancelPool() // Ensure cleanup
		close(outCh)
	}()

	return outCh
}
