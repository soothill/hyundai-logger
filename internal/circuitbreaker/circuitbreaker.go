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

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when too many requests are made in half-open state
	ErrTooManyRequests = errors.New("too many requests")
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed means the circuit breaker is closed (normal operation)
	StateClosed State = iota
	// StateOpen means the circuit breaker is open (rejecting requests)
	StateOpen
	// StateHalfOpen means the circuit breaker is testing if the service recovered
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
	// MaxFailures is the maximum number of failures before opening the circuit
	MaxFailures int
	// Timeout is how long to wait before transitioning from Open to Half-Open
	Timeout time.Duration
	// MaxRequests is the maximum number of requests allowed in Half-Open state
	MaxRequests int
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures: 5,
		Timeout:     60 * time.Second,
		MaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu            sync.RWMutex
	config        Config
	state         State
	failures      int
	successes     int
	lastFailTime  time.Time
	lastStateTime time.Time
	halfOpenReqs  int
}

// New creates a new circuit breaker
func New(config Config) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRequests <= 0 {
		config.MaxRequests = 1
	}

	return &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		lastStateTime: time.Now(),
	}
}

// Call executes the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if we can proceed
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
		// Check if timeout has passed
		if time.Since(cb.lastStateTime) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenReqs = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests to test if service recovered
		if cb.halfOpenReqs >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenReqs++
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state reopens the circuit
		cb.setState(StateOpen)
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.successes++

	switch cb.state {
	case StateClosed:
		// Reset failure counter on success
		cb.failures = 0

	case StateHalfOpen:
		// Transition back to closed if requests succeed in half-open state
		cb.setState(StateClosed)
		cb.failures = 0
		cb.halfOpenReqs = 0
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(state State) {
	if cb.state != state {
		cb.state = state
		cb.lastStateTime = time.Now()
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:         cb.state,
		Failures:      cb.failures,
		Successes:     cb.successes,
		LastFailTime:  cb.lastFailTime,
		LastStateTime: cb.lastStateTime,
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenReqs = 0
}

// Stats holds circuit breaker statistics
type Stats struct {
	State         State
	Failures      int
	Successes     int
	LastFailTime  time.Time
	LastStateTime time.Time
}
