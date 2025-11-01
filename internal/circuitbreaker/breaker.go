// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed allows requests through
	StateClosed State = iota
	// StateOpen blocks requests (circuit is broken)
	StateOpen
	// StateHalfOpen allows a test request to check recovery
	StateHalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	// MaxFailures is the number of consecutive failures before opening
	MaxFailures int
	// Timeout is how long to wait in Open state before trying HalfOpen
	Timeout time.Duration
	// HalfOpenMaxRequests is max requests allowed in HalfOpen state
	HalfOpenMaxRequests int
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxFailures:         5,
		Timeout:             60 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu                  sync.RWMutex
	state               State
	failures            int
	lastFailureTime     time.Time
	lastStateChange     time.Time
	halfOpenRequests    int
	config              Config
	onStateChange       func(from, to State)
}

// New creates a new circuit breaker with the given configuration
func New(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// NewWithDefaults creates a circuit breaker with default configuration
func NewWithDefaults() *CircuitBreaker {
	return New(DefaultConfig())
}

// OnStateChange sets a callback for state changes
func (cb *CircuitBreaker) OnStateChange(fn func(from, to State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned in HalfOpen when max requests exceeded
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can execute
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil

	case StateOpen:
		// Check if we should transition to HalfOpen
		if time.Since(cb.lastStateChange) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result of the request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Request failed
		cb.failures++
		cb.lastFailureTime = time.Now()

		switch cb.state {
		case StateClosed:
			// Check if we should open
			if cb.failures >= cb.config.MaxFailures {
				cb.setState(StateOpen)
			}

		case StateHalfOpen:
			// Failed in HalfOpen, go back to Open
			cb.setState(StateOpen)
		}
	} else {
		// Request succeeded
		switch cb.state {
		case StateClosed:
			// Reset failure count on success
			cb.failures = 0

		case StateHalfOpen:
			// Success in HalfOpen, go back to Closed
			cb.failures = 0
			cb.setState(StateClosed)
		}
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Call the state change callback if set
	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// GetState returns the current state (thread-safe)
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns current statistics (thread-safe)
func (cb *CircuitBreaker) GetStats() (state State, failures int, lastFailure time.Time) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, cb.failures, cb.lastFailureTime
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.halfOpenRequests = 0
	cb.lastFailureTime = time.Time{}
}
