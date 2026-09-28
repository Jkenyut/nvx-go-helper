package worker

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// TestEmptyJobs tests the new empty jobs optimization
func TestEmptyJobs(t *testing.T) {
	jobs := []Job[int, int]{}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
	)

	count := 0
	for range results {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 results for empty jobs, got %d", count)
	}
}

// TestDuplicateJobIDs verifies duplicate detection
func TestDuplicateJobIDs(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 1, Data: 300}, // Duplicate ID
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
	)

	count := 0
	errorCount := 0
	for res := range results {
		count++
		if res.Err != nil {
			errorCount++
			if res.Err.Error() != "duplicate job ID detected: 1 (all jobs rejected)" {
				t.Errorf("Unexpected error: %v", res.Err)
			}
		}
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
	if errorCount != len(jobs) {
		t.Errorf("Expected all %d jobs to have errors, got %d", len(jobs), errorCount)
	}
}

// TestNormalOperation tests basic functionality
func TestNormalOperation(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 3, Data: 300},
		{ID: 4, Data: 400},
		{ID: 5, Data: 500},
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(3),
		WithGlobalTimeout(5*time.Second),
	)

	count := 0
	successCount := 0
	resultMap := make(map[int]bool)

	for res := range results {
		count++
		if res.Err == nil {
			successCount++
			resultMap[res.ID] = true
		} else {
			t.Logf("Job ID %d failed with error: %v", res.ID, res.Err)
		}
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
	if successCount != len(jobs) {
		t.Errorf("Expected %d successful results, got %d", len(jobs), successCount)
	}
	for _, job := range jobs {
		if !resultMap[job.ID] {
			t.Errorf("Missing result for job ID %d", job.ID)
		}
	}
}

// TestParentContextCancelled tests early exit when parent context is cancelled
func TestParentContextCancelled(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return fmt.Sprintf("result-%d", data), nil
	}

	startTime := time.Now()
	results := Stream(
		parentCtx,
		jobs,
		workerFunc,
	)

	count := 0
	for res := range results {
		count++
		if !errors.Is(res.Err, ErrSkipped) {
			t.Errorf("Expected ErrSkipped, got %v", res.Err)
		}
	}
	elapsed := time.Since(startTime)

	if elapsed > 50*time.Millisecond {
		t.Errorf("Expected immediate return (<50ms), took %v", elapsed)
	}
	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
}

// TestPanicRecovery tests panic handling
func TestPanicRecovery(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will panic
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		if data == 200 {
			panic("intentional panic")
		}
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
	)

	count := 0
	panicCount := 0
	successCount := 0

	for res := range results {
		count++
		if res.Err != nil {
			if res.ID == 2 {
				if res.Err.Error() != "panic: intentional panic" {
					t.Errorf("Expected panic error, got %v", res.Err)
				}
				panicCount++
			}
		} else {
			successCount++
		}
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
	if panicCount != 1 {
		t.Errorf("Expected 1 panic, got %d", panicCount)
	}
	if successCount == 0 {
		t.Error("Expected at least one successful result")
	}
}

// TestStopOnError tests StopOnError mode
func TestStopOnError(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will error
		{ID: 3, Data: 300},
		{ID: 4, Data: 400},
		{ID: 5, Data: 500},
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		if data == 200 {
			return "", errors.New("intentional error")
		}
		time.Sleep(20 * time.Millisecond)
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
		WithStopOnError(true),
	)

	count := 0
	skippedCount := 0
	errorCount := 0

	for res := range results {
		count++
		if res.Err != nil {
			if errors.Is(res.Err, ErrSkipped) {
				skippedCount++
			} else {
				errorCount++
			}
		}
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
	if errorCount == 0 {
		t.Error("Expected at least one error")
	}
}

// TestWorkerTimeout tests per-worker timeout
func TestWorkerTimeout(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will timeout
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, id, data int) (string, error) {
		if data == 200 {
			select {
			case <-time.After(5 * time.Second):
				return fmt.Sprintf("result-%d", data), nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return fmt.Sprintf("result-%d", data), nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
		WithWorkerTimeout(100*time.Millisecond),
	)

	count := 0
	timeoutCount := 0
	successCount := 0

	for res := range results {
		count++
		if res.Err != nil {
			if errors.Is(res.Err, context.DeadlineExceeded) {
				timeoutCount++
			}
		} else {
			successCount++
		}
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
	if timeoutCount == 0 {
		t.Error("Expected at least one timeout")
	}
	if successCount == 0 {
		t.Error("Expected at least one successful result")
	}
}

// TestNoDuplicateResults verifies no duplicate results even under high concurrency
func TestNoDuplicateResults(t *testing.T) {
	const numJobs = 100

	jobs := make([]Job[int, int], numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = Job[int, int]{ID: i, Data: i * 10}
	}

	workerFunc := func(ctx context.Context, id, data int) (int, error) {
		return data * 2, nil
	}

	results := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(10),
	)

	resultMap := make(map[int]int)
	for res := range results {
		resultMap[res.ID]++
	}

	for id, count := range resultMap {
		if count > 1 {
			t.Errorf("Job ID %d received %d times (duplicate!)", id, count)
		}
	}
}

// BenchmarkWorkerPool benchmarks worker pool performance
func BenchmarkWorkerPool(b *testing.B) {
	jobs := make([]Job[int, int], 100)
	for i := 0; i < 100; i++ {
		jobs[i] = Job[int, int]{ID: i, Data: i}
	}

	workerFunc := func(ctx context.Context, id, data int) (int, error) {
		return data * 2, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := Stream(
			context.Background(),
			jobs,
			workerFunc,
			WithWorkers(10),
		)

		for range results {
			// Drain channel
		}
	}
}

// TestRunSynchronous tests the synchronous Run function
func TestRunSynchronous(t *testing.T) {
	jobs := []Job[string, int]{
		{ID: "A", Data: 1},
		{ID: "B", Data: 2},
		{ID: "C", Data: 3},
	}

	workerFunc := func(ctx context.Context, id string, data int) (int, error) {
		if data == 2 {
			return 0, errors.New("job B failed")
		}
		return data * 10, nil
	}

	results, err := Run(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
	)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if err == nil {
		t.Error("Expected an aggregated error, got nil")
	} else if err.Error() != "job B failed" {
		t.Errorf("Expected 'job B failed', got %v", err)
	}

	for _, res := range results {
		switch res.ID {
		case "B":
			if res.Err == nil {
				t.Error("Expected error for job B")
			}
		case "A":
			if res.Value != 10 {
				t.Errorf("Expected 10 for A, got %d", res.Value)
			}
		case "C":
			if res.Value != 30 {
				t.Errorf("Expected 30 for C, got %d", res.Value)
			}
		}
	}
}

// TestRunOrdered tests the PreserveOrder configuration
func TestRunOrdered(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 10, Data: 1},
		{ID: 20, Data: 2},
		{ID: 30, Data: 3},
		{ID: 40, Data: 4},
		{ID: 50, Data: 5},
	}

	workerFunc := func(ctx context.Context, id, data int) (int, error) {
		sleepTime := time.Duration(100-(data*10)) * time.Millisecond
		time.Sleep(sleepTime)
		return data * 100, nil
	}

	results, err := Run(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(5),
		WithPreserveOrder(true),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != len(jobs) {
		t.Fatalf("Expected %d results, got %d", len(jobs), len(results))
	}

	for i, job := range jobs {
		if results[i].ID != job.ID {
			t.Errorf("Result at index %d has ID %v, expected %v", i, results[i].ID, job.ID)
		}
		if results[i].Value != job.Data*100 {
			t.Errorf("Result at index %d has Value %v, expected %v", i, results[i].Value, job.Data*100)
		}
	}
}

// TestOnProgress tests the OnProgress callback
func TestOnProgress(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 1},
		{ID: 2, Data: 2},
		{ID: 3, Data: 3},
	}

	workerFunc := func(ctx context.Context, id, data int) (int, error) {
		return data * 10, nil
	}

	var progressCount int32
	var finalTotal int32

	onProgress := func(completed, total int) {
		atomic.AddInt32(&progressCount, 1)
		atomic.StoreInt32(&finalTotal, int32(total))
	}

	_, err := Run(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
		WithOnProgress(onProgress),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if atomic.LoadInt32(&progressCount) != int32(len(jobs)) {
		t.Errorf("Expected OnProgress to be called %d times, got %d", len(jobs), progressCount)
	}

	if atomic.LoadInt32(&finalTotal) != int32(len(jobs)) {
		t.Errorf("Expected total to be %d, got %d", len(jobs), finalTotal)
	}
}

func TestStream_WithOptions(t *testing.T) {
	jobs := []Job[int, int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
	}

	workerFunc := func(ctx context.Context, id, data int) (int, error) {
		return data + 1, nil
	}

	sem := make(chan struct{}, 1)
	outCh := Stream(
		context.Background(),
		jobs,
		workerFunc,
		WithWorkers(2),
		WithGlobalSemaphore(sem),
	)

	received := 0
	for res := range outCh {
		if res.Err != nil {
			t.Fatalf("unexpected error: %v", res.Err)
		}
		received++
	}

	if received != 2 {
		t.Errorf("expected 2 streamed results, got %d", received)
	}
}
