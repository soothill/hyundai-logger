// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/circuitbreaker"
)

// BenchmarkCircuitBreakerSuccess benchmarks successful operations through circuit breaker
func BenchmarkCircuitBreakerSuccess(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil // Always succeed
		})
	}
}

// BenchmarkCircuitBreakerFailure benchmarks failed operations
func BenchmarkCircuitBreakerFailure(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return fmt.Errorf("simulated error")
		})
	}
}

// BenchmarkCircuitBreakerOpen benchmarks operations when circuit is open
func BenchmarkCircuitBreakerOpen(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             1 * time.Hour, // Long timeout to keep circuit open
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	// Trip the circuit breaker
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return fmt.Errorf("trip circuit")
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}
}

// BenchmarkCircuitBreakerParallel benchmarks parallel operations
func BenchmarkCircuitBreakerParallel(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Execute(func() error {
				return nil
			})
		}
	})
}

// BenchmarkCircuitBreakerGetState benchmarks state checking
func BenchmarkCircuitBreakerGetState(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.GetState()
	}
}

// BenchmarkCircuitBreakerGetStats benchmarks statistics gathering
func BenchmarkCircuitBreakerGetStats(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := circuitbreaker.New(config)

	// Execute some operations to populate stats
	for i := 0; i < 100; i++ {
		_ = cb.Execute(func() error {
			if i%5 == 0 {
				return fmt.Errorf("error")
			}
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = cb.GetStats()
	}
}

// BenchmarkCircuitBreakerMixedWorkload benchmarks realistic mixed success/failure workload
func BenchmarkCircuitBreakerMixedWorkload(b *testing.B) {
	config := circuitbreaker.Config{
		MaxFailures:         5,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	}
	cb := circuitbreaker.New(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 90% success rate
		_ = cb.Execute(func() error {
			if i%10 == 0 {
				return fmt.Errorf("occasional error")
			}
			return nil
		})
	}
}
