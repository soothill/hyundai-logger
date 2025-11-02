// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Config defines retry behavior
type Config struct {
	MaxAttempts       int
	InitialDelayMs    int
	MaxDelayMs        int
	BackoffMultiplier float64
}

// Retrier handles retry logic with exponential backoff
type Retrier struct {
	config Config
}

// New creates a new Retrier
func New(config Config) *Retrier {
	return &Retrier{
		config: config,
	}
}

// Do executes a function with retry logic and exponential backoff
func (r *Retrier) Do(ctx context.Context, operation func() error) error {
	_, err := r.doRetry(ctx, func() (interface{}, error) {
		return nil, operation()
	})
	return err
}

// DoWithResult executes a function with retry logic and returns a result
func (r *Retrier) DoWithResult(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	return r.doRetry(ctx, operation)
}

// doRetry is the core retry logic used by both Do and DoWithResult
func (r *Retrier) doRetry(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var result interface{}

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Try the operation
		var err error
		result, err = operation()
		if err == nil {
			return result, nil // Success
		}

		lastErr = err

		// Check if we should retry
		if attempt >= r.config.MaxAttempts {
			break
		}

		// Check if context is canceled
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context canceled after %d attempts: %w", attempt, ctx.Err())
		}

		// Calculate backoff delay
		delay := r.calculateDelay(attempt)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled after %d attempts: %w", attempt, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for a given attempt using exponential backoff
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff: initialDelay * (multiplier ^ (attempt - 1))
	delayMs := float64(r.config.InitialDelayMs) * math.Pow(r.config.BackoffMultiplier, float64(attempt-1))

	// Cap at max delay
	if delayMs > float64(r.config.MaxDelayMs) {
		delayMs = float64(r.config.MaxDelayMs)
	}

	return time.Duration(delayMs) * time.Millisecond
}

// GetAttemptDelay returns the delay that would be used for a specific attempt (for logging/testing)
func (r *Retrier) GetAttemptDelay(attempt int) time.Duration {
	return r.calculateDelay(attempt)
}
