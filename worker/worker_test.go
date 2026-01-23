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
	jobs := []Job[int]{}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{},
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
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 1, Data: 300}, // Duplicate ID
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{},
	)

	count := 0
	errorCount := 0
	for res := range results {
		count++
		if res.Err != nil {
			errorCount++
			// Verify it's the duplicate error
			if res.Err.Error() != "duplicate job ID detected: 1 (all jobs rejected)" {
				t.Errorf("Unexpected error: %v", res.Err)
			}
		}
	}

	// All jobs should receive error
	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}

	if errorCount != len(jobs) {
		t.Errorf("Expected all %d jobs to have errors, got %d", len(jobs), errorCount)
	}
}

// TestNormalOperation tests basic functionality
func TestNormalOperation(t *testing.T) {
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 3, Data: 300},
		{ID: 4, Data: 400},
		{ID: 5, Data: 500},
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:    3,
			GlobalTimeout: 5 * time.Second, // Ensure enough time
		},
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

	// Verify 1:1 mapping
	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}

	if successCount != len(jobs) {
		t.Errorf("Expected %d successful results, got %d", len(jobs), successCount)
	}

	// Verify all job IDs received
	for _, job := range jobs {
		if !resultMap[job.ID] {
			t.Errorf("Missing result for job ID %d", job.ID)
		}
	}
}

// TestParentContextCancelled tests early exit when parent context is cancelled
func TestParentContextCancelled(t *testing.T) {
	// Create cancelled context
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		time.Sleep(100 * time.Millisecond) // This should never execute
		return fmt.Sprintf("result-%d", data), nil
	}

	startTime := time.Now()
	results := RunGenericWorkerPoolStream(
		parentCtx,
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{},
	)

	count := 0
	for res := range results {
		count++
		if res.Err != ErrSkipped {
			t.Errorf("Expected ErrSkipped, got %v", res.Err)
		}
	}
	elapsed := time.Since(startTime)

	// Should return immediately
	if elapsed > 50*time.Millisecond {
		t.Errorf("Expected immediate return (<50ms), took %v", elapsed)
	}

	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
}

// TestPanicRecovery tests panic handling
func TestPanicRecovery(t *testing.T) {
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will panic
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		if data == 200 {
			panic("intentional panic")
		}
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{NumWorkers: 2},
	)

	count := 0
	panicCount := 0
	successCount := 0

	for res := range results {
		count++
		if res.Err != nil {
			if res.ID == 2 {
				// Verify panic was caught
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

	// At least some should succeed
	if successCount == 0 {
		t.Error("Expected at least one successful result")
	}

	t.Logf("Panics: %d, Success: %d", panicCount, successCount)
}

// TestStopOnError tests StopOnError mode
func TestStopOnError(t *testing.T) {
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will error
		{ID: 3, Data: 300},
		{ID: 4, Data: 400},
		{ID: 5, Data: 500},
	}

	var processedCount int32

	workerFunc := func(ctx context.Context, data int) (string, error) {
		atomic.AddInt32(&processedCount, 1)
		time.Sleep(50 * time.Millisecond) // Simulate work
		if data == 200 {
			return "", errors.New("intentional error")
		}
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:  2,
			StopOnError: true,
		},
	)

	count := 0
	errorCount := 0
	skippedCount := 0

	for res := range results {
		count++
		if res.Err != nil {
			if res.Err == ErrSkipped {
				skippedCount++
			} else {
				errorCount++
			}
		}
	}

	// All jobs should get results (1:1 guarantee)
	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}

	// Should have at least one error
	if errorCount == 0 {
		t.Error("Expected at least one error")
	}

	// Some jobs should be skipped due to StopOnError
	if skippedCount == 0 {
		t.Error("Expected some jobs to be skipped with StopOnError=true")
	}

	t.Logf("Processed: %d, Errors: %d, Skipped: %d", processedCount, errorCount, skippedCount)
}

// TestGlobalTimeout tests global timeout
func TestGlobalTimeout(t *testing.T) {
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200},
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		// Simulate long work that will timeout
		select {
		case <-time.After(5 * time.Second):
			return fmt.Sprintf("result-%d", data), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	startTime := time.Now()
	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:    2,
			GlobalTimeout: 100 * time.Millisecond,
		},
	)

	count := 0
	for range results {
		count++
	}
	elapsed := time.Since(startTime)

	// Should timeout in ~100ms, not 5s
	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected timeout around 100ms, took %v", elapsed)
	}

	// All jobs should get results
	if count != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), count)
	}
}

// TestWorkerTimeout tests per-worker timeout
func TestWorkerTimeout(t *testing.T) {
	jobs := []Job[int]{
		{ID: 1, Data: 100},
		{ID: 2, Data: 200}, // This will timeout
		{ID: 3, Data: 300},
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		if data == 200 {
			// Simulate work that exceeds worker timeout
			select {
			case <-time.After(5 * time.Second):
				return fmt.Sprintf("result-%d", data), nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return fmt.Sprintf("result-%d", data), nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:    2,
			WorkerTimeout: 100 * time.Millisecond,
		},
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

	// At least one task should timeout (might be more due to timing)
	if timeoutCount == 0 {
		t.Error("Expected at least one timeout")
	}

	// Should have at least one success
	if successCount == 0 {
		t.Error("Expected at least one successful result")
	}

	t.Logf("Timeouts: %d, Success: %d", timeoutCount, successCount)
}

// TestNoDuplicateResults verifies no duplicate results even under high concurrency
func TestNoDuplicateResults(t *testing.T) {
	const numJobs = 100

	jobs := make([]Job[int], numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = Job[int]{ID: i, Data: i * 10}
	}

	workerFunc := func(ctx context.Context, data int) (int, error) {
		// Minimal work to stress concurrency
		return data * 2, nil
	}

	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{NumWorkers: 10},
	)

	resultMap := make(map[int]int) // ID -> count
	for res := range results {
		resultMap[res.ID]++
	}

	// Check for duplicates
	duplicateCount := 0
	for id, count := range resultMap {
		if count > 1 {
			t.Errorf("Job ID %d received %d times (duplicate!)", id, count)
			duplicateCount++
		}
	}

	if duplicateCount > 0 {
		t.Errorf("Found %d duplicate results", duplicateCount)
	}

	// Verify all jobs received exactly one result
	if len(resultMap) != numJobs {
		t.Errorf("Expected %d unique results, got %d", numJobs, len(resultMap))
	}
}

// TestLargeDatasetStopOnError tests that StopOnError works efficiently with 1M jobs
func TestLargeDatasetStopOnError(t *testing.T) {
	const numJobs = 1000000
	jobs := make([]Job[int], numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = Job[int]{ID: i, Data: i}
	}

	workerFunc := func(ctx context.Context, data int) (string, error) {
		// Error early on
		if data == 2 {
			return "", errors.New("intentional error")
		}
		return fmt.Sprintf("%d", data), nil
	}

	startTime := time.Now()
	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:  4,
			StopOnError: true,
		},
	)

	successCount := 0
	failCount := 0
	skippedCount := 0

	for res := range results {
		if res.Err != nil {
			if res.Err == ErrSkipped {
				skippedCount++
			} else {
				failCount++
				if res.Err.Error() != "intentional error" {
					t.Errorf("Unexpected error: %v", res.Err)
				}
			}
		} else {
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	t.Logf("Processed 1M jobs with early error in %v", elapsed)

	// Should be very fast (<< 1s) because it stops early
	// But 1M jobs overhead (map checks, channel sends) takes ~1s on some machines
	if elapsed > 3*time.Second {
		t.Errorf("StopOnError with 1M jobs took too long: %v", elapsed)
	}

	if failCount == 0 {
		t.Error("Expected at least one failure")
	}

	// Most jobs should be skipped
	if skippedCount < numJobs-2000 { // Allow some slack for concurrent workers
		t.Errorf("Expected most jobs to be skipped, got %d skipped", skippedCount)
	}
}

// TestLargeDatasetTimeout tests that timeout works with 1M jobs
func TestLargeDatasetTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	const numJobs = 1000000
	jobs := make([]Job[int], numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = Job[int]{ID: i, Data: i}
	}

	// Simulate slow work that DEFINITELY causes timeout
	// 4 workers * 30s timeout = max processing capacity is low
	// We want to verify it halts at 30s (default logic) or specified global timeout

	workerFunc := func(ctx context.Context, data int) (string, error) {
		time.Sleep(10 * time.Microsecond)
		return fmt.Sprintf("%d", data), nil
	}

	startTime := time.Now()
	results := RunGenericWorkerPoolStream(
		context.Background(),
		jobs,
		workerFunc,
		nil,
		WorkerPoolConfig{
			NumWorkers:    4,
			GlobalTimeout: 100 * time.Millisecond, // Fast timeout
		},
	)

	skippedCount := 0
	successCount := 0

	for res := range results {
		switch res.Err {
		case ErrSkipped:
			skippedCount++
		case nil:
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	t.Logf("Processed 1M jobs with timeout in %v", elapsed)

	// Allow some time for overhead of skipping 1M jobs (can take ~1s+)
	if elapsed > 3*time.Second {
		t.Errorf("GlobalTimeout failed, took %v", elapsed)
	}

	if skippedCount == 0 {
		t.Error("Expected skipped jobs due to timeout")
	}
}

// BenchmarkWorkerPool benchmarks worker pool performance
func BenchmarkWorkerPool(b *testing.B) {
	jobs := make([]Job[int], 100)
	for i := 0; i < 100; i++ {
		jobs[i] = Job[int]{ID: i, Data: i}
	}

	workerFunc := func(ctx context.Context, data int) (int, error) {
		// Simulate minimal work
		return data * 2, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := RunGenericWorkerPoolStream(
			context.Background(),
			jobs,
			workerFunc,
			nil,
			WorkerPoolConfig{NumWorkers: 10},
		)

		for range results {
			// Drain channel
		}
	}
}
