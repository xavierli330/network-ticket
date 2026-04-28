package client

import "time"

// RetryConfig holds exponential backoff parameters for push retries.
type RetryConfig struct {
	MaxAttempts  int
	BaseInterval time.Duration
	MaxInterval  time.Duration
}

// DefaultRetryConfig returns sensible defaults: 5 attempts, 1s base, 30s cap.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		BaseInterval: time.Second,
		MaxInterval:  30 * time.Second,
	}
}

// Backoff returns the delay before the next retry attempt.
// attempt is zero-indexed (0 = first retry delay = BaseInterval).
func Backoff(attempt int, cfg RetryConfig) time.Duration {
	delay := cfg.BaseInterval
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > cfg.MaxInterval {
			return cfg.MaxInterval
		}
	}
	return delay
}
