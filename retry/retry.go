package retry

import (
	"context"
	"time"
)

// Option is a function that configures the retry behavior.
type Option func(*config)

type config struct {
	maxAttempts int
	delay       time.Duration
	multiplier  float64
	ctx         context.Context
	retryIf     func(error) bool
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

// WithContext sets a context for the retry operation.
// If the context is canceled, the retry stops immediately and returns the context error.
func WithContext(ctx context.Context) Option {
	return func(c *config) {
		c.ctx = ctx
	}
}

// RetryIf allows specifying a custom condition to determine if an error should trigger a retry.
// If the function returns false, Do will return immediately with the error.
func RetryIf(condition func(err error) bool) Option {
	return func(c *config) {
		c.retryIf = condition
	}
}

// Do executes the given action. If the action returns an error, it will retry
// according to the provided options.
func Do(action func() error, opts ...Option) error {
	// Default configuration
	cfg := &config{
		maxAttempts: 3,
		delay:       0,
		multiplier:  1.0,
		ctx:         context.Background(),
		retryIf: func(err error) bool {
			return true // Retry on all errors by default
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var err error
	delay := cfg.delay

	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		// Check context before attempting
		if errCtx := cfg.ctx.Err(); errCtx != nil {
			return errCtx
		}

		err = action()
		if err == nil {
			return nil
		}

		// If the error shouldn't be retried, return immediately
		if !cfg.retryIf(err) {
			return err
		}

		// Don't wait if this was the last attempt
		if attempt == cfg.maxAttempts {
			break
		}

		// Wait for the delay or context cancellation
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-cfg.ctx.Done():
				return cfg.ctx.Err()
			}

			// Apply backoff multiplier for next attempt
			if cfg.multiplier > 1.0 {
				delay = time.Duration(float64(delay) * cfg.multiplier)
			}
		} else {
			// If no delay, just check context
			select {
			case <-cfg.ctx.Done():
				return cfg.ctx.Err()
			default:
			}
		}
	}

	return err
}
