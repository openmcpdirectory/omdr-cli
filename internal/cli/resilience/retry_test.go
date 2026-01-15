package resilience

import (
	"context"
	"testing"
	"time"
)

func TestRetryWithBackoff_SuccessFirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := RetryWithBackoff(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return &testError{msg: "temporary failure"}
		}
		return nil
	}

	err := RetryWithBackoff(ctx, config, fn)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	attempts := 0
	fn := func() error {
		attempts++
		return &testError{msg: "persistent failure"}
	}

	err := RetryWithBackoff(ctx, config, fn)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if attempts != config.MaxAttempts {
		t.Errorf("expected %d attempts, got %d", config.MaxAttempts, attempts)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()

	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return &testError{msg: "failure"}
	}

	err := RetryWithBackoff(ctx, config, fn)
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should stop after context cancellation
	if attempts > 2 {
		t.Errorf("expected at most 2 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	var delays []time.Duration
	lastTime := time.Now()

	fn := func() error {
		now := time.Now()
		if attempts > 0 {
			delays = append(delays, now.Sub(lastTime))
		}
		lastTime = now
		attempts++
		return &testError{msg: "failure"}
	}

	_ = RetryWithBackoff(ctx, config, fn)

	// Check that delays are increasing
	if len(delays) != 2 {
		t.Errorf("expected 2 delays, got %d", len(delays))
	}

	// First delay should be ~10ms
	if delays[0] < 8*time.Millisecond || delays[0] > 15*time.Millisecond {
		t.Errorf("first delay out of range: %v", delays[0])
	}

	// Second delay should be ~20ms
	if delays[1] < 18*time.Millisecond || delays[1] > 25*time.Millisecond {
		t.Errorf("second delay out of range: %v", delays[1])
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", &testError{msg: "connection refused"}, true},
		{"timeout", &testError{msg: "timeout"}, true},
		{"500 error", &testError{msg: "server error: 500"}, true},
		{"503 error", &testError{msg: "server error: 503"}, true},
		{"404 error", &testError{msg: "client error: 404"}, false},
		{"401 error", &testError{msg: "client error: 401"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
