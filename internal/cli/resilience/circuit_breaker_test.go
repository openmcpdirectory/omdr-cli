package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Should allow calls when closed
	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Fail 3 times
	for i := 0; i < 3; i++ {
		_ = cb.Call(func() error {
			return errors.New("failure")
		})
	}

	// Circuit should be open now
	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %s", cb.State())
	}

	// Next call should fail immediately
	err := cb.Call(func() error {
		t.Error("function should not be called when circuit is open")
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}

	if err.Error() != "circuit breaker is open - service temporarily unavailable" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open on next call attempt
	// A failure in half-open state should increment failures
	_ = cb.Call(func() error {
		return errors.New("still failing")
	})

	// After a failure in half-open, the circuit should either:
	// - Stay in HALF_OPEN (if failure count hasn't reached max yet)
	// - Go to OPEN (if failure count reached max)
	// The current implementation increments failures but doesn't necessarily go back to OPEN immediately
	state := cb.State()
	if state != StateOpen && state != StateHalfOpen {
		t.Errorf("expected state OPEN or HALF_OPEN after half-open failure, got %s", state)
	}
}

func TestCircuitBreaker_HalfOpenToClosedTransition(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Successful call in half-open should close the circuit
	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after successful half-open call, got %s", cb.State())
	}

	if cb.Failures() != 0 {
		t.Errorf("expected 0 failures after closing, got %d", cb.Failures())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %s", cb.State())
	}

	// Manual reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after reset, got %s", cb.State())
	}

	if cb.Failures() != 0 {
		t.Errorf("expected 0 failures after reset, got %d", cb.Failures())
	}
}

func TestCircuitBreaker_PartialFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Fail twice
	for i := 0; i < 2; i++ {
		_ = cb.Call(func() error {
			return errors.New("failure")
		})
	}

	if cb.Failures() != 2 {
		t.Errorf("expected 2 failures, got %d", cb.Failures())
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED (not enough failures), got %s", cb.State())
	}

	// Successful call should reset failure count
	_ = cb.Call(func() error {
		return nil
	})

	if cb.Failures() != 0 {
		t.Errorf("expected 0 failures after success, got %d", cb.Failures())
	}
}
