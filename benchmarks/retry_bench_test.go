// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/retry"
)

// BenchmarkRetrySuccess benchmarks successful operations without retries
func BenchmarkRetrySuccess(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Do(ctx, func() error {
			return nil // Always succeed
		})
	}
}

// BenchmarkRetryFailure benchmarks operations that always fail
func BenchmarkRetryFailure(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    1, // Minimal delay for benchmark
		MaxDelayMs:        10,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Do(ctx, func() error {
			return fmt.Errorf("simulated error")
		})
	}
}

// BenchmarkRetryEventualSuccess benchmarks operations that succeed on 2nd attempt
func BenchmarkRetryEventualSuccess(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    1, // Minimal delay for benchmark
		MaxDelayMs:        10,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		_ = r.Do(ctx, func() error {
			attempts++
			if attempts < 2 {
				return fmt.Errorf("retry needed")
			}
			return nil
		})
	}
}

// BenchmarkRetryWithResult benchmarks DoWithResult method
func BenchmarkRetryWithResult(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.DoWithResult(ctx, func() (interface{}, error) {
			return "result", nil
		})
	}
}

// BenchmarkRetryWithResultFailure benchmarks DoWithResult with failures
func BenchmarkRetryWithResultFailure(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    1,
		MaxDelayMs:        10,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.DoWithResult(ctx, func() (interface{}, error) {
			return nil, fmt.Errorf("error")
		})
	}
}

// BenchmarkRetryParallel benchmarks parallel retry operations
func BenchmarkRetryParallel(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.Do(ctx, func() error {
				return nil
			})
		}
	})
}

// BenchmarkRetryBackoff benchmarks exponential backoff calculation
func BenchmarkRetryBackoff(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       5,
		InitialDelayMs:    10,
		MaxDelayMs:        5000,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts := 0
		_ = r.Do(ctx, func() error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("retry")
			}
			return nil
		})
	}
}

// BenchmarkRetryContextCancellation benchmarks context cancellation handling
func BenchmarkRetryContextCancellation(b *testing.B) {
	config := retry.Config{
		MaxAttempts:       10,
		InitialDelayMs:    100,
		MaxDelayMs:        5000,
		BackoffMultiplier: 2.0,
	}
	r := retry.New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		_ = r.Do(ctx, func() error {
			time.Sleep(10 * time.Millisecond) // Longer than context timeout
			return nil
		})
		cancel()
	}
}
