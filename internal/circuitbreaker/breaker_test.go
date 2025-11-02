// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package circuitbreaker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewWithDefaults()

	if cb.GetState() != StateClosed {
		t.Errorf("expected initial state to be Closed, got %s", cb.GetState())
	}

	state, failures, _ := cb.GetStats()
	if state != StateClosed {
		t.Errorf("expected Closed state, got %s", state)
	}
	if failures != 0 {
		t.Errorf("expected 0 failures, got %d", failures)
	}
}

func TestCircuitBreaker_SuccessfulExecution(t *testing.T) {
	cb := NewWithDefaults()

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("expected state to remain Closed, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_FailedExecution(t *testing.T) {
	cb := NewWithDefaults()

	testErr := errors.New("test error")
	err := cb.Execute(func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("expected state to remain Closed after single failure, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterMaxFailures(t *testing.T) {
	config := Config{
		MaxFailures:         3,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	testErr := errors.New("test error")

	// First 2 failures should keep circuit closed
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
		if cb.GetState() != StateClosed {
			t.Errorf("expected Closed after %d failures, got %s", i+1, cb.GetState())
		}
	}

	// 3rd failure should open the circuit
	_ = cb.Execute(func() error {
		return testErr
	})

	if cb.GetState() != StateOpen {
		t.Errorf("expected Open after 3 failures, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_OpenCircuitBlocksRequests(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Trigger circuit to open
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatal("circuit should be open")
	}

	// Attempt execution - should be blocked
	err := cb.Execute(func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatal("circuit should be open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to HalfOpen
	executed := false
	_ = cb.Execute(func() error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("expected function to execute in HalfOpen state")
	}
}

func TestCircuitBreaker_HalfOpenSuccessClosesCircuit(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Execute successfully in HalfOpen - should close circuit
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("expected Closed after successful HalfOpen, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenFailureOpensCircuit(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Execute with failure in HalfOpen - should go back to Open
	_ = cb.Execute(func() error {
		return testErr
	})

	if cb.GetState() != StateOpen {
		t.Errorf("expected Open after failed HalfOpen, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenMaxRequests(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// First request should succeed and close the circuit
	executionCount := 0
	err1 := cb.Execute(func() error {
		executionCount++
		return nil // Success
	})
	if err1 != nil {
		t.Errorf("first request should succeed, got error: %v", err1)
	}
	if executionCount != 1 {
		t.Error("first request should have executed")
	}

	// Circuit should now be Closed, so second request should succeed too
	err2 := cb.Execute(func() error {
		executionCount++
		return nil
	})
	if err2 != nil {
		t.Errorf("second request should succeed in Closed state, got error: %v", err2)
	}
	if executionCount != 2 {
		t.Error("second request should have executed")
	}

	// Verify circuit is Closed
	if cb.GetState() != StateClosed {
		t.Errorf("expected Closed state after successful HalfOpen, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Fatal("circuit should be open")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("expected Closed after reset, got %s", cb.GetState())
	}

	state, failures, _ := cb.GetStats()
	if failures != 0 {
		t.Errorf("expected 0 failures after reset, got %d", failures)
	}
	if state != StateClosed {
		t.Errorf("expected Closed state after reset, got %s", state)
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	config := Config{
		MaxFailures:         2,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := New(config)

	var transitions []string
	cb.OnStateChange(func(from, to State) {
		transitions = append(transitions, from.String()+"->"+to.String())
	})

	// Trigger state transitions
	testErr := errors.New("test error")

	// Open the circuit (Closed -> Open)
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait and trigger HalfOpen (Open -> HalfOpen)
	time.Sleep(100 * time.Millisecond)
	_ = cb.Execute(func() error {
		return nil // Success to go to Closed
	})
	// HalfOpen -> Closed

	if len(transitions) < 2 {
		t.Errorf("expected at least 2 transitions, got %d: %v", len(transitions), transitions)
	}

	expectedFirst := "closed->open"
	if transitions[0] != expectedFirst {
		t.Errorf("expected first transition '%s', got '%s'", expectedFirst, transitions[0])
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.state.String())
			}
		})
	}
}

func TestCircuitBreaker_ConcurrentExecution(t *testing.T) {
	cb := NewWithDefaults()
	done := make(chan bool)
	var successCount int32
	var failCount int32

	// Run concurrent executions
	for i := 0; i < 50; i++ {
		go func(n int) {
			err := cb.Execute(func() error {
				if n%2 == 0 {
					return nil
				}
				return errors.New("error")
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 50; i++ {
		<-done
	}

	// Just verify it didn't panic
	totalCount := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&failCount)
	if totalCount == 0 {
		t.Error("expected some executions to complete")
	}
}
