package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDo_Success(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_RetryThenSuccess(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	}, WithMaxAttempts(5))
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_ExhaustedRetries(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return errors.New("always fail")
	}, WithMaxAttempts(3))
	assert.Error(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, "always fail", err.Error())
}

func TestDo_WithDelay(t *testing.T) {
	start := time.Now()
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	}, WithMaxAttempts(3), WithDelay(50*time.Millisecond))
	elapsed := time.Since(start)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(90)) // 2 delays * 50ms
}

func TestDo_WithBackoff(t *testing.T) {
	start := time.Now()
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	}, WithMaxAttempts(3), WithBackoff(20*time.Millisecond, 2.0))
	elapsed := time.Since(start)
	assert.NoError(t, err)
	// First delay: 20ms, second delay: 40ms → total ~60ms
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(50))
}

func TestDo_WithContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Do(func() error {
		return errors.New("should not succeed")
	}, WithContext(ctx), WithMaxAttempts(5))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestDo_WithContext_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := Do(func() error {
		return errors.New("always fail")
	}, WithContext(ctx), WithMaxAttempts(100), WithDelay(20*time.Millisecond))
	assert.Error(t, err)
}

func TestDo_RetryIf(t *testing.T) {
	calls := 0
	permanentErr := errors.New("permanent")

	err := Do(func() error {
		calls++
		if calls == 1 {
			return errors.New("temporary")
		}
		return permanentErr
	}, WithMaxAttempts(5), If(func(err error) bool {
		return err.Error() != "permanent"
	}))
	assert.ErrorIs(t, err, permanentErr)
	assert.Equal(t, 2, calls) // Stopped after permanent error
}

func TestDo_WithJitter(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	}, WithMaxAttempts(3), WithDelay(10*time.Millisecond), WithJitter(0.5))
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_WithMaxDelay(t *testing.T) {
	start := time.Now()
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 4 {
			return errors.New("temp")
		}
		return nil
	}, WithMaxAttempts(5), WithBackoff(50*time.Millisecond, 10.0), WithMaxDelay(60*time.Millisecond))
	elapsed := time.Since(start)
	assert.NoError(t, err)
	// Without max delay: 50 + 500 + 5000 = 5550ms
	// With max delay capped at 60ms: 50 + 60 + 60 = 170ms
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestDo_WithOnRetry(t *testing.T) {
	var attempts []int
	var errs []error

	err := Do(func() error {
		return errors.New("fail")
	}, WithMaxAttempts(3), WithOnRetry(func(attempt int, err error) {
		attempts = append(attempts, attempt)
		errs = append(errs, err)
	}))
	assert.Error(t, err)
	assert.Equal(t, []int{1, 2}, attempts) // Called on attempt 1 and 2 (not on last attempt)
	assert.Len(t, errs, 2)
}

func TestDoWithResult_Success(t *testing.T) {
	result, err := DoWithResult(func() (string, error) {
		return "hello", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestDoWithResult_RetryThenSuccess(t *testing.T) {
	calls := 0
	result, err := DoWithResult(func() (int, error) {
		calls++
		if calls < 3 {
			return 0, errors.New("temp")
		}
		return 42, nil
	}, WithMaxAttempts(5))
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestDoWithResult_ExhaustedRetries(t *testing.T) {
	result, err := DoWithResult(func() (string, error) {
		return "", errors.New("always fail")
	}, WithMaxAttempts(2))
	assert.Error(t, err)
	assert.Equal(t, "", result)
}

func TestDoWithResult_WithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := DoWithResult(func() (int, error) {
		return 0, errors.New("fail")
	}, WithContext(ctx), WithMaxAttempts(5))
	assert.Error(t, err)
	assert.Equal(t, 0, result)
}

func TestDoWithResult_WithJitter(t *testing.T) {
	var callCount int32
	result, err := DoWithResult(func() (string, error) {
		if atomic.AddInt32(&callCount, 1) < 3 {
			return "", errors.New("temp")
		}
		return "done", nil
	}, WithMaxAttempts(5), WithDelay(5*time.Millisecond), WithJitter(1.0))
	assert.NoError(t, err)
	assert.Equal(t, "done", result)
}

func TestJitterBounds(t *testing.T) {
	// Invalid jitter values should be ignored
	cfg := newConfig(WithJitter(0.0))
	assert.Equal(t, 0.0, cfg.jitter)

	cfg = newConfig(WithJitter(-1.0))
	assert.Equal(t, 0.0, cfg.jitter)

	cfg = newConfig(WithJitter(1.1))
	assert.Equal(t, 0.0, cfg.jitter)

	// Valid jitter
	cfg = newConfig(WithJitter(0.5))
	assert.Equal(t, 0.5, cfg.jitter)
}

func TestApplyJitter(t *testing.T) {
	// No jitter → same delay
	d := applyJitter(100*time.Millisecond, 0.0)
	assert.Equal(t, 100*time.Millisecond, d)

	// With jitter → within range
	for i := 0; i < 100; i++ {
		d := applyJitter(100*time.Millisecond, 0.5)
		assert.GreaterOrEqual(t, int64(d), int64(0))
		assert.LessOrEqual(t, int64(d), int64(200*time.Millisecond))
	}
}

func TestWithMaxAttempts_Invalid(t *testing.T) {
	cfg := newConfig(WithMaxAttempts(0))
	assert.Equal(t, 3, cfg.maxAttempts) // default
	cfg = newConfig(WithMaxAttempts(-1))
	assert.Equal(t, 3, cfg.maxAttempts) // default
}

func TestWithBackoff_InvalidMultiplier(t *testing.T) {
	cfg := newConfig(WithBackoff(time.Second, 0.5))
	assert.Equal(t, 1.0, cfg.multiplier) // default, not applied
	assert.Equal(t, time.Second, cfg.delay)
}
