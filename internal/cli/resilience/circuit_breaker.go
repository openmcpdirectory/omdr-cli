package resilience

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are rejected
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern
// It prevents cascading failures by failing fast when a service is down
type CircuitBreaker struct {
	maxFailures int
	timeout     time.Duration

	mu           sync.Mutex
	state        CircuitState
	failures     int
	lastFailTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker
// maxFailures: number of consecutive failures before opening the circuit
// timeout: duration to wait before transitioning from Open to HalfOpen
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       StateClosed,
	}
}

// Call executes a function with circuit breaker protection
// Returns an error if the circuit is open or if the function fails
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if circuit should transition from Open to HalfOpen
	if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.timeout {
		cb.state = StateHalfOpen
		cb.failures = 0
	}

	// Reject if circuit is open
	if cb.state == StateOpen {
		cb.mu.Unlock()
		return errors.New("circuit breaker is open - service temporarily unavailable")
	}

	cb.mu.Unlock()

	// Execute function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Open circuit if max failures reached
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}

		return err
	}

	// Success - reset or close circuit
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
	cb.failures = 0

	return nil
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
}
