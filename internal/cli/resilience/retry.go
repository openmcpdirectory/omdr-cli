package resilience

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig defines configuration for retry logic with exponential backoff
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns sensible defaults for retry logic
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
// It will retry on errors up to MaxAttempts times, with delays growing exponentially
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		// Execute function
		if err := fn(); err != nil {
			lastErr = err

			// Check if error is retryable
			if !IsRetryableError(err) {
				return err
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}

			// Continue to next attempt
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("max retry attempts (%d) reached: %w", config.MaxAttempts, lastErr)
}

// IsRetryableError determines if an error should trigger a retry
// Network errors and 5xx server errors are retryable
// 4xx client errors are not retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are retryable
	if contains(errStr, "connection refused") ||
		contains(errStr, "timeout") ||
		contains(errStr, "temporary failure") ||
		contains(errStr, "no such host") {
		return true
	}

	// 5xx server errors are retryable
	if contains(errStr, "500") ||
		contains(errStr, "502") ||
		contains(errStr, "503") ||
		contains(errStr, "504") {
		return true
	}

	// 4xx client errors are not retryable
	if contains(errStr, "400") ||
		contains(errStr, "401") ||
		contains(errStr, "403") ||
		contains(errStr, "404") {
		return false
	}

	// Default: retry on unknown errors
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
