// Package retry provides utility functions for retrying operations.
package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

// Option is a function that configures the retry behavior.
type Option func(*config)

type config struct {
	maxAttempts int
	delay       time.Duration
	multiplier  float64
	maxDelay    time.Duration
	jitter      float64
	ctx         context.Context
	retryIf     func(error) bool
	onRetry     func(attempt int, err error)
}

// WithMaxAttempts sets the maximum number of attempts before giving up.
// Default is 3 attempts.
func WithMaxAttempts(attempts int) Option {
	return func(c *config) {
		if attempts > 0 {
			c.maxAttempts = attempts
		}
	}
}

// WithDelay sets the initial delay between retries.
func WithDelay(delay time.Duration) Option {
	return func(c *config) {
		c.delay = delay
	}
}

// WithBackoff sets an exponential multiplier for the delay.
// For example, if delay is 1s and multiplier is 2, delays will be 1s, 2s, 4s...
func WithBackoff(initial time.Duration, multiplier float64) Option {
	return func(c *config) {
		c.delay = initial
		if multiplier >= 1.0 {
			c.multiplier = multiplier
		}
	}
}

// WithMaxDelay caps the maximum delay between retries.
// Useful to prevent exponential backoff from growing unbounded.
//
// Example:
//
//	retry.WithMaxDelay(30 * time.Second) // delay will never exceed 30s
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(c *config) {
		c.maxDelay = maxDelay
	}
}

// WithJitter adds random jitter to the delay to prevent thundering herd.
// The factor should be between 0.0 and 1.0, where:
//   - 0.0 means no jitter (deterministic delay)
//   - 1.0 means full jitter (delay can vary from 0 to 2x the base delay)
//   - 0.5 means delay varies from 0.5x to 1.5x the base delay (recommended)
//
// Example:
//
//	retry.WithJitter(0.5) // ±50% jitter
func WithJitter(factor float64) Option {
	return func(c *config) {
		if factor > 0 && factor <= 1.0 {
			c.jitter = factor
		}
	}
}

// WithContext sets a context for the retry operation.
// If the context is canceled, the retry stops immediately and returns the context error.
func WithContext(ctx context.Context) Option {
	return func(c *config) {
		c.ctx = ctx
	}
}

// If allows specifying a custom condition to determine if an error should trigger a retry.
// If the function returns false, Do will return immediately with the error.
func If(condition func(err error) bool) Option {
	return func(c *config) {
		c.retryIf = condition
	}
}

// WithOnRetry sets a callback that is invoked before each retry attempt.
// Useful for logging retry attempts with the error that triggered the retry.
//
// Example:
//
//	retry.WithOnRetry(func(attempt int, err error) {
//	    log.Printf("retry attempt %d: %v", attempt, err)
//	})
func WithOnRetry(fn func(attempt int, err error)) Option {
	return func(c *config) {
		c.onRetry = fn
	}
}

// newConfig creates a default config and applies options.
func newConfig(opts ...Option) *config {
	cfg := &config{
		maxAttempts: 3,
		delay:       0,
		multiplier:  1.0,
		jitter:      0,
		maxDelay:    0,
		ctx:         context.Background(),
		retryIf: func(_ error) bool {
			return true // Retry on all errors by default
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// applyJitter applies jitter to the delay using the configured factor.
// Returns a duration in the range [delay*(1-factor), delay*(1+factor)].
func applyJitter(delay time.Duration, factor float64) time.Duration {
	if factor <= 0 {
		return delay
	}
	// Generate random value in [-factor, +factor]
	jitterRange := float64(delay) * factor
	jittered := float64(delay) + (rand.Float64()*2-1)*jitterRange
	if jittered < 0 {
		jittered = 0
	}
	return time.Duration(jittered)
}

// calculateDelay computes the next delay with backoff, jitter, and max cap.
func calculateDelay(delay time.Duration, cfg *config) time.Duration {
	// Apply jitter
	d := applyJitter(delay, cfg.jitter)

	// Apply max delay cap
	if cfg.maxDelay > 0 && d > cfg.maxDelay {
		d = cfg.maxDelay
	}

	return d
}

// Do executes the given action. If the action returns an error, it will retry
// according to the provided options.
func Do(action func() error, opts ...Option) error {
	_, err := DoWithResult(func() (struct{}, error) {
		return struct{}{}, action()
	}, opts...)
	return err
}

// DoWithResult executes the given action that returns a value. If the action returns an error,
// it will retry according to the provided options. Returns the result value from the
// first successful attempt, or zero value with the last error if all attempts fail.
//
// Example:
//
//	result, err := retry.DoWithResult(func() (string, error) {
//	    return fetchData()
//	}, retry.WithMaxAttempts(3), retry.WithBackoff(time.Second, 2.0))
func DoWithResult[T any](action func() (T, error), opts ...Option) (T, error) {
	cfg := newConfig(opts...)

	var result T
	var err error
	delay := cfg.delay

	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		// Check context before attempting
		if errCtx := cfg.ctx.Err(); errCtx != nil {
			return result, errCtx
		}

		result, err = action()
		if err == nil {
			return result, nil
		}

		// If the error shouldn't be retried, return immediately
		if !cfg.retryIf(err) {
			return result, err
		}

		// Don't wait if this was the last attempt
		if attempt == cfg.maxAttempts {
			break
		}

		// Call onRetry callback if configured
		if cfg.onRetry != nil {
			cfg.onRetry(attempt, err)
		}

		// Wait for the delay or context cancellation with proper timer cleanup
		if delay > 0 {
			waitDuration := calculateDelay(delay, cfg)
			timer := time.NewTimer(waitDuration)
			select {
			case <-timer.C:
			case <-cfg.ctx.Done():
				timer.Stop()
				return result, cfg.ctx.Err()
			}
			timer.Stop()

			// Apply backoff multiplier for next attempt
			if cfg.multiplier > 1.0 {
				delay = time.Duration(float64(delay) * cfg.multiplier)
			}
		} else {
			// If no delay, just check context
			select {
			case <-cfg.ctx.Done():
				return result, cfg.ctx.Err()
			default:
			}
		}
	}

	return result, err
}
