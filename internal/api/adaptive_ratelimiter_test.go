// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"context"
	"testing"
	"time"
)

func TestNewAdaptiveRateLimiter(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100) // 100 requests per hour

	if limiter == nil {
		t.Fatal("Expected limiter to be created, got nil")
	}

	stats := limiter.GetStats()
	expectedRps := 100.0 / 3600.0

	if stats.BaseRate != expectedRps {
		t.Errorf("Expected base rate %.6f, got %.6f", expectedRps, stats.BaseRate)
	}
	if stats.CurrentRate != expectedRps {
		t.Errorf("Expected current rate %.6f, got %.6f", expectedRps, stats.CurrentRate)
	}
}

func TestAdaptiveRateLimiter_Wait(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(3600) // 1 request per second for easy testing
	ctx := context.Background()

	// First request should be immediate
	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if duration > 100*time.Millisecond {
		t.Errorf("First request took too long: %v", duration)
	}

	// Second request should wait ~1 second
	start = time.Now()
	err = limiter.Wait(ctx)
	duration = time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
	if duration < 900*time.Millisecond || duration > 1100*time.Millisecond {
		t.Errorf("Second request wait time unexpected: %v (expected ~1s)", duration)
	}
}

func TestAdaptiveRateLimiter_Allow(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(3600) // 1 request per second

	// First request should be allowed
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}

	// Immediate second request should be denied (burst is 1)
	if limiter.Allow() {
		t.Error("Immediate second request should be denied")
	}

	// Wait for capacity to replenish
	time.Sleep(1100 * time.Millisecond)

	// Now should be allowed again
	if !limiter.Allow() {
		t.Error("Request after waiting should be allowed")
	}
}

func TestAdaptiveRateLimiter_HandleRateLimitResponse(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100) // 100 requests per hour
	initialRate := limiter.GetStats().CurrentRate

	// Simulate 429 response
	limiter.HandleRateLimitResponse(60*time.Second, "", "")

	stats := limiter.GetStats()

	// Rate should have been reduced
	if stats.CurrentRate >= initialRate {
		t.Errorf("Expected rate to be reduced, got %.6f (was %.6f)", stats.CurrentRate, initialRate)
	}

	// Metrics should be updated
	if stats.RateLimitHits != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", stats.RateLimitHits)
	}
	if stats.DynamicAdjustments != 1 {
		t.Errorf("Expected 1 dynamic adjustment, got %d", stats.DynamicAdjustments)
	}
}

func TestAdaptiveRateLimiter_HandleRateLimitResponse_WithHeaders(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(1000) // 1000 requests per hour

	// Simulate 429 response with headers indicating 500 remaining out of 1000 limit
	limiter.HandleRateLimitResponse(60*time.Second, "500", "1000")

	stats := limiter.GetStats()

	// Rate should be adjusted based on headers
	// Expected: (500 * 0.8) / 3600 = 0.111... rps
	expectedRate := (500.0 * 0.8) / 3600.0

	if stats.CurrentRate < expectedRate*0.9 || stats.CurrentRate > expectedRate*1.1 {
		t.Errorf("Expected rate ~%.6f, got %.6f", expectedRate, stats.CurrentRate)
	}
}

func TestAdaptiveRateLimiter_GradualRecovery(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100) // 100 requests per hour

	// Simulate rate limit and reduction
	limiter.HandleRateLimitResponse(60*time.Second, "", "")
	reducedRate := limiter.GetStats().CurrentRate

	// Manually set last adjustment time to past to trigger recovery
	limiter.mu.Lock()
	limiter.lastAdjustment = time.Now().Add(-10 * time.Minute)
	limiter.mu.Unlock()

	// Trigger gradual recovery
	limiter.GradualRecovery()

	newRate := limiter.GetStats().CurrentRate

	// Rate should have recovered somewhat
	if newRate <= reducedRate {
		t.Errorf("Expected rate to recover, got %.6f (was %.6f)", newRate, reducedRate)
	}
}

func TestAdaptiveRateLimiter_GradualRecovery_TooSoon(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100)

	// Simulate rate limit
	limiter.HandleRateLimitResponse(60*time.Second, "", "")
	rateAfterLimit := limiter.GetStats().CurrentRate

	// Try to recover immediately (should not recover yet)
	limiter.GradualRecovery()

	rateAfterRecovery := limiter.GetStats().CurrentRate

	// Rate should not have changed (too soon)
	if rateAfterRecovery != rateAfterLimit {
		t.Errorf("Rate should not recover so soon, got %.6f (was %.6f)", rateAfterRecovery, rateAfterLimit)
	}
}

func TestAdaptiveRateLimiter_Reset(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100)
	baseRate := limiter.GetStats().BaseRate

	// Simulate multiple rate limits
	limiter.HandleRateLimitResponse(60*time.Second, "", "")
	limiter.HandleRateLimitResponse(60*time.Second, "", "")

	reducedRate := limiter.GetStats().CurrentRate

	// Verify rate was reduced
	if reducedRate >= baseRate {
		t.Error("Expected rate to be reduced")
	}

	// Reset the limiter
	limiter.Reset()

	stats := limiter.GetStats()

	// Rate should be back to base
	if stats.CurrentRate != baseRate {
		t.Errorf("Expected rate to be reset to base (%.6f), got %.6f", baseRate, stats.CurrentRate)
	}
}

func TestAdaptiveRateLimiter_MinRateFloor(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100)

	// Simulate many rate limits to push rate very low
	for i := 0; i < 20; i++ {
		limiter.HandleRateLimitResponse(60*time.Second, "", "")
	}

	stats := limiter.GetStats()

	// Rate should not go below minRate (10% of base)
	minRate := limiter.minRate
	if stats.CurrentRate < float64(minRate) {
		t.Errorf("Rate dropped below minimum: %.6f < %.6f", stats.CurrentRate, float64(minRate))
	}
}

func TestAdaptiveRateLimiter_MultipleHits(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100)

	// Simulate multiple 429 responses
	for i := 0; i < 5; i++ {
		limiter.HandleRateLimitResponse(60*time.Second, "", "")
	}

	stats := limiter.GetStats()

	if stats.RateLimitHits != 5 {
		t.Errorf("Expected 5 rate limit hits, got %d", stats.RateLimitHits)
	}
	if stats.DynamicAdjustments != 5 {
		t.Errorf("Expected 5 dynamic adjustments, got %d", stats.DynamicAdjustments)
	}
}

func TestAdaptiveRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(1) // Very low rate to force waiting

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := limiter.Wait(ctx)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestRateLimiterStats_String(t *testing.T) {
	stats := RateLimiterStats{
		BaseRate:           0.027778,
		CurrentRate:        0.013889,
		RateLimitHits:      5,
		DynamicAdjustments: 3,
		LastAdjustment:     time.Now(),
	}

	str := stats.String()

	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Verify key information is present
	if !contains(str, "base=") || !contains(str, "current=") || !contains(str, "hits=") {
		t.Errorf("String representation missing key information: %s", str)
	}
}

func TestAdaptiveRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(3600) // High rate for concurrent access
	ctx := context.Background()

	done := make(chan bool)
	errors := make(chan error, 10)

	// Multiple goroutines calling Wait
	for i := 0; i < 10; i++ {
		go func() {
			err := limiter.Wait(ctx)
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestAdaptiveRateLimiter_RecoveryDoesNotExceedBase(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100)
	baseRate := limiter.GetStats().BaseRate

	// Simulate rate limit
	limiter.HandleRateLimitResponse(60*time.Second, "", "")

	// Force many recovery attempts
	limiter.mu.Lock()
	limiter.lastAdjustment = time.Now().Add(-1 * time.Hour)
	limiter.mu.Unlock()

	for i := 0; i < 100; i++ {
		limiter.GradualRecovery()
		limiter.mu.Lock()
		limiter.lastAdjustment = time.Now().Add(-1 * time.Hour)
		limiter.mu.Unlock()
	}

	finalRate := limiter.GetStats().CurrentRate

	// Rate should not exceed base rate
	if finalRate > baseRate {
		t.Errorf("Rate exceeded base rate: %.6f > %.6f", finalRate, baseRate)
	}

	// Should be at or very close to base rate
	if finalRate < baseRate*0.99 {
		t.Errorf("Rate did not fully recover: %.6f < %.6f", finalRate, baseRate)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkAdaptiveRateLimiter_Wait(b *testing.B) {
	limiter := NewAdaptiveRateLimiter(36000000) // Very high rate to minimize waiting
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Wait(ctx)
	}
}

func BenchmarkAdaptiveRateLimiter_Allow(b *testing.B) {
	limiter := NewAdaptiveRateLimiter(36000000) // Very high rate
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow()
	}
}

func BenchmarkAdaptiveRateLimiter_HandleRateLimitResponse(b *testing.B) {
	limiter := NewAdaptiveRateLimiter(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.HandleRateLimitResponse(60*time.Second, "500", "1000")
	}
}

func BenchmarkAdaptiveRateLimiter_ConcurrentWait(b *testing.B) {
	limiter := NewAdaptiveRateLimiter(36000000) // Very high rate to minimize blocking
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Wait(ctx)
		}
	})
}

func BenchmarkAdaptiveRateLimiter_ConcurrentAllow(b *testing.B) {
	limiter := NewAdaptiveRateLimiter(36000000) // Very high rate
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Allow()
		}
	})
}
