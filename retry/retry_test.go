package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetrySuccessOnFirstTry(t *testing.T) {
	attempts := 0
	err := Do(func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetrySuccessAfterMultipleTries(t *testing.T) {
	attempts := 0
	err := Do(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}, WithMaxAttempts(5), WithDelay(10*time.Millisecond))

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryFailure(t *testing.T) {
	attempts := 0
	expectedErr := errors.New("persistent error")
	err := Do(func() error {
		attempts++
		return expectedErr
	}, WithMaxAttempts(4), WithDelay(10*time.Millisecond))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 4, attempts)
}

func TestRetryWithBackoff(t *testing.T) {
	attempts := 0
	start := time.Now()

	err := Do(func() error {
		attempts++
		return errors.New("fail")
	}, WithMaxAttempts(3), WithBackoff(10*time.Millisecond, 2.0))

	duration := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	// Delay 1: 10ms
	// Delay 2: 20ms
	// Total delay should be at least 30ms
	assert.True(t, duration >= 30*time.Millisecond, "Expected duration >= 30ms, got %v", duration)
}

func TestRetryWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	err := Do(func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel on second attempt
		}
		return errors.New("fail")
	}, WithContext(ctx), WithMaxAttempts(5), WithDelay(50*time.Millisecond))

	// It should stop after the second attempt due to context cancellation
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryIf(t *testing.T) {
	attempts := 0
	fatalErr := errors.New("fatal auth error")

	err := Do(func() error {
		attempts++
		if attempts == 2 {
			return fatalErr
		}
		return errors.New("network blip")
	},
		WithMaxAttempts(5),
		WithDelay(10*time.Millisecond),
		RetryIf(func(err error) bool {
			return err != fatalErr
		}))

	// Should stop immediately after the second attempt returning fatalErr
	assert.Error(t, err)
	assert.Equal(t, fatalErr, err)
	assert.Equal(t, 2, attempts)
}
