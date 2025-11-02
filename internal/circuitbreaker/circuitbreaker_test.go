// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := New(Config{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
		MaxRequests: 1,
	})

	// Should allow requests when closed
	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed, got %s", cb.GetState())
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := New(Config{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
		MaxRequests: 1,
	})

	testErr := errors.New("test error")

	// Fail 3 times to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Call(func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}
	}

	// Circuit should now be open
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open, got %s", cb.GetState())
	}

	// Next call should fail with circuit open error
	err := cb.Call(func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := New(Config{
		MaxFailures: 2,
		Timeout:     50 * time.Millisecond,
		MaxRequests: 1,
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state to be Open, got %s", cb.GetState())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to half-open
	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error in half-open state, got %v", err)
	}

	// Should be closed again after successful call
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after successful half-open call, got %s", cb.GetState())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := New(Config{
		MaxFailures: 2,
		Timeout:     50 * time.Millisecond,
		MaxRequests: 1,
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open state
	err := cb.Call(func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	// Should be open again
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after half-open failure, got %s", cb.GetState())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := New(Config{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
		MaxRequests: 1,
	})

	testErr := errors.New("test error")

	// Record some successes and failures
	cb.Call(func() error { return nil })
	cb.Call(func() error { return nil })
	cb.Call(func() error { return testErr })

	stats := cb.GetStats()

	if stats.Successes != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.Successes)
	}

	if stats.Failures != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.Failures)
	}

	if stats.State != StateClosed {
		t.Errorf("Expected state Closed, got %s", stats.State)
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := New(Config{
		MaxFailures: 2,
		Timeout:     100 * time.Millisecond,
		MaxRequests: 1,
	})

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Call(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state to be Open, got %s", cb.GetState())
	}

	// Reset the circuit breaker
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after reset, got %s", cb.GetState())
	}

	stats := cb.GetStats()
	if stats.Failures != 0 || stats.Successes != 0 {
		t.Errorf("Expected stats to be reset, got failures=%d, successes=%d", stats.Failures, stats.Successes)
	}
}

func TestCircuitBreakerDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxFailures <= 0 {
		t.Error("MaxFailures should be positive")
	}

	if config.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}

	if config.MaxRequests <= 0 {
		t.Error("MaxRequests should be positive")
	}
}

func TestCircuitBreakerStateString(t *testing.T) {
	states := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
	}

	for _, tc := range states {
		if tc.state.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.state.String())
		}
	}
}
