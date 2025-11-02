// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	if retrier == nil {
		t.Fatal("Expected retrier to be created, got nil")
	}
	if retrier.config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", retrier.config.MaxAttempts)
	}
}

func TestDo_SuccessFirstAttempt(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	attempts := 0
	err := retrier.Do(ctx, func() error {
		attempts++
		return nil // Success on first attempt
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	attempts := 0
	err := retrier.Do(ctx, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third attempt
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDo_FailAllAttempts(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	testError := errors.New("persistent error")
	attempts := 0

	err := retrier.Do(ctx, func() error {
		attempts++
		return testError
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, testError) {
		t.Errorf("Expected error to wrap testError")
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	config := Config{
		MaxAttempts:       5,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	err := retrier.Do(ctx, func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel context after second attempt
		}
		return errors.New("temporary error")
	})

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
	// Should stop after context cancellation
	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts, got %d", attempts)
	}
}

func TestDo_ContextTimeout(t *testing.T) {
	config := Config{
		MaxAttempts:       5,
		InitialDelayMs:    50,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attempts := 0
	err := retrier.Do(ctx, func() error {
		attempts++
		return errors.New("temporary error")
	})

	if err == nil {
		t.Error("Expected error due to context timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestDoWithResult_SuccessFirstAttempt(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	expectedResult := "success"
	attempts := 0

	result, err := retrier.DoWithResult(ctx, func() (interface{}, error) {
		attempts++
		return expectedResult, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, result)
	}
}

func TestDoWithResult_SuccessAfterRetries(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	expectedResult := 42
	attempts := 0

	result, err := retrier.DoWithResult(ctx, func() (interface{}, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary error")
		}
		return expectedResult, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, result)
	}
}

func TestDoWithResult_FailAllAttempts(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	testError := errors.New("persistent error")
	attempts := 0

	result, err := retrier.DoWithResult(ctx, func() (interface{}, error) {
		attempts++
		return nil, testError
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDoWithResult_ContextCanceled(t *testing.T) {
	config := Config{
		MaxAttempts:       5,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	result, err := retrier.DoWithResult(ctx, func() (interface{}, error) {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return nil, errors.New("temporary error")
	})

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestCalculateDelay(t *testing.T) {
	config := Config{
		MaxAttempts:       5,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)

	tests := []struct {
		attempt      int
		expectedMs   int
	}{
		{attempt: 1, expectedMs: 100},  // 100 * 2^0 = 100
		{attempt: 2, expectedMs: 200},  // 100 * 2^1 = 200
		{attempt: 3, expectedMs: 400},  // 100 * 2^2 = 400
		{attempt: 4, expectedMs: 800},  // 100 * 2^3 = 800
		{attempt: 5, expectedMs: 1000}, // 100 * 2^4 = 1600, capped at 1000
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := retrier.calculateDelay(tt.attempt)
			expectedDelay := time.Duration(tt.expectedMs) * time.Millisecond

			if delay != expectedDelay {
				t.Errorf("Attempt %d: expected delay %v, got %v", tt.attempt, expectedDelay, delay)
			}
		})
	}
}

func TestGetAttemptDelay(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    50,
		MaxDelayMs:        500,
		BackoffMultiplier: 3.0,
	}

	retrier := New(config)

	delay1 := retrier.GetAttemptDelay(1)
	delay2 := retrier.GetAttemptDelay(2)
	delay3 := retrier.GetAttemptDelay(3)

	// Verify exponential increase
	expected1 := 50 * time.Millisecond  // 50 * 3^0
	expected2 := 150 * time.Millisecond // 50 * 3^1
	expected3 := 450 * time.Millisecond // 50 * 3^2

	if delay1 != expected1 {
		t.Errorf("Delay 1: expected %v, got %v", expected1, delay1)
	}
	if delay2 != expected2 {
		t.Errorf("Delay 2: expected %v, got %v", expected2, delay2)
	}
	if delay3 != expected3 {
		t.Errorf("Delay 3: expected %v, got %v", expected3, delay3)
	}

	// Verify delay2 > delay1
	if delay2 <= delay1 {
		t.Error("Expected exponential backoff (delay2 should be greater than delay1)")
	}
}

func TestCalculateDelay_MaxDelayCap(t *testing.T) {
	config := Config{
		MaxAttempts:       10,
		InitialDelayMs:    1000,
		MaxDelayMs:        5000,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)

	// Attempt 5: 1000 * 2^4 = 16000ms, should be capped at 5000ms
	delay := retrier.calculateDelay(5)
	expected := 5000 * time.Millisecond

	if delay != expected {
		t.Errorf("Expected delay to be capped at %v, got %v", expected, delay)
	}

	// Verify all high attempts are capped
	for attempt := 6; attempt <= 10; attempt++ {
		delay := retrier.calculateDelay(attempt)
		if delay != expected {
			t.Errorf("Attempt %d: expected delay capped at %v, got %v", attempt, expected, delay)
		}
	}
}

func TestDo_TimingVerification(t *testing.T) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    50,
		MaxDelayMs:        500,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	start := time.Now()
	attempts := 0

	_ = retrier.Do(ctx, func() error {
		attempts++
		return errors.New("always fail")
	})

	duration := time.Since(start)

	// Total expected delays: 50ms (after attempt 1) + 100ms (after attempt 2) = 150ms
	// Allow some tolerance for execution time
	minDuration := 140 * time.Millisecond
	maxDuration := 300 * time.Millisecond

	if duration < minDuration {
		t.Errorf("Duration too short: expected at least %v, got %v", minDuration, duration)
	}
	if duration > maxDuration {
		t.Errorf("Duration too long: expected at most %v, got %v", maxDuration, duration)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDifferentBackoffMultipliers(t *testing.T) {
	tests := []struct {
		name       string
		multiplier float64
		attempt    int
		initialMs  int
		expectedMs int
	}{
		{
			name:       "multiplier 1.5",
			multiplier: 1.5,
			attempt:    3,
			initialMs:  100,
			expectedMs: 225, // 100 * 1.5^2
		},
		{
			name:       "multiplier 3.0",
			multiplier: 3.0,
			attempt:    3,
			initialMs:  100,
			expectedMs: 900, // 100 * 3^2
		},
		{
			name:       "multiplier 1.0 (no backoff)",
			multiplier: 1.0,
			attempt:    5,
			initialMs:  100,
			expectedMs: 100, // 100 * 1^4 = 100 (constant)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				MaxAttempts:       10,
				InitialDelayMs:    tt.initialMs,
				MaxDelayMs:        10000,
				BackoffMultiplier: tt.multiplier,
			}

			retrier := New(config)
			delay := retrier.calculateDelay(tt.attempt)
			expected := time.Duration(tt.expectedMs) * time.Millisecond

			if delay != expected {
				t.Errorf("Expected delay %v, got %v", expected, delay)
			}
		})
	}
}

func BenchmarkDo(b *testing.B) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    1,
		MaxDelayMs:        10,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retrier.Do(ctx, func() error {
			return nil // Immediate success
		})
	}
}

func BenchmarkDoWithRetries(b *testing.B) {
	config := Config{
		MaxAttempts:       3,
		InitialDelayMs:    1,
		MaxDelayMs:        10,
		BackoffMultiplier: 2.0,
	}

	retrier := New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		_ = retrier.Do(ctx, func() error {
			count++
			if count < 3 {
				return errors.New("retry")
			}
			return nil
		})
	}
}
