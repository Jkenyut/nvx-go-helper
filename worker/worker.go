package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Job represents a generic job input.
type Job[T any] struct {
	ID   int // Unique identifier
	Data T   // Payload
}

// Result represents the output of processing a Job.
type Result[R any] struct {
	ID    int   // Matches Job.ID
	Value R     // Success result
	Err   error // Error result
}

// WorkerPoolConfig holds configuration options.
type WorkerPoolConfig struct {
	NumWorkers    int           // Concurrent workers (default: 2)
	WorkerTimeout time.Duration // Per-job timeout (default: 15s)
	GlobalTimeout time.Duration // Global pool timeout (default: 30s)
	StopOnError   bool          // Cancel all on first error
}

// ErrSkipped indicates a job was not processed.
var ErrSkipped = fmt.Errorf("job not processed (cancelled or skipped)")

// RunGenericWorkerPoolStream executes jobs concurrently and streams results.
// It guarantees 1:1 result mapping for every job ID.
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
