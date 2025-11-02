// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// AdaptiveRateLimiter wraps a rate.Limiter with dynamic adjustment capabilities
type AdaptiveRateLimiter struct {
	limiter       *rate.Limiter
	mu            sync.RWMutex
	baseRate      rate.Limit // Original configured rate
	currentRate   rate.Limit // Current adjusted rate
	backoffFactor float64    // How much to reduce rate when hitting limits
	recoveryRate  float64    // How quickly to recover to base rate
	minRate       rate.Limit // Minimum rate to prevent complete throttling

	// Metrics
	rateLimitHits    int64
	dynamicAdjustments int64
	lastAdjustment   time.Time
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(requestsPerHour int) *AdaptiveRateLimiter {
	rps := float64(requestsPerHour) / 3600.0
	baseRate := rate.Limit(rps)

	return &AdaptiveRateLimiter{
		limiter:       rate.NewLimiter(baseRate, 1),
		baseRate:      baseRate,
		currentRate:   baseRate,
		backoffFactor: 0.5, // Reduce to 50% on rate limit
		recoveryRate:  1.1, // Recover by 10% over time
		minRate:       rate.Limit(rps * 0.1), // Never go below 10% of base rate
		lastAdjustment: time.Now(),
	}
}

// Wait blocks until the rate limiter allows an event
func (a *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	a.mu.RLock()
	limiter := a.limiter
	a.mu.RUnlock()

	return limiter.Wait(ctx)
}

// Allow reports whether an event may happen now
func (a *AdaptiveRateLimiter) Allow() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.limiter.Allow()
}

// HandleRateLimitResponse adjusts the rate limiter based on a 429 response
func (a *AdaptiveRateLimiter) HandleRateLimitResponse(retryAfter time.Duration, remaining, limit string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.rateLimitHits++
	a.dynamicAdjustments++
	a.lastAdjustment = time.Now()

	// Calculate new rate based on rate limit headers if available
	if remaining != "" && limit != "" {
		// Parse remaining and limit to calculate safe rate
		a.adjustBasedOnHeaders(remaining, limit)
	} else {
		// Fall back to backoff factor
		newRate := a.currentRate * rate.Limit(a.backoffFactor)
		if newRate < a.minRate {
			newRate = a.minRate
		}
		a.currentRate = newRate
		a.limiter.SetLimit(newRate)
	}
}

// adjustBasedOnHeaders calculates a new rate based on X-RateLimit headers
func (a *AdaptiveRateLimiter) adjustBasedOnHeaders(remaining, limit string) {
	// Try to parse the headers
	var remainingVal, limitVal int
	fmt.Sscanf(remaining, "%d", &remainingVal)
	fmt.Sscanf(limit, "%d", &limitVal)

	if limitVal > 0 {
		// Calculate a safe rate: use 80% of the remaining quota
		// distributed over 1 hour to be conservative
		safeRate := float64(remainingVal) * 0.8 / 3600.0
		newRate := rate.Limit(safeRate)

		if newRate < a.minRate {
			newRate = a.minRate
		}
		if newRate > a.baseRate {
			newRate = a.baseRate
		}

		a.currentRate = newRate
		a.limiter.SetLimit(newRate)
	}
}

// GradualRecovery should be called periodically to recover back to base rate
func (a *AdaptiveRateLimiter) GradualRecovery() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Only recover if we're below base rate and it's been a while since last adjustment
	if a.currentRate < a.baseRate && time.Since(a.lastAdjustment) > 5*time.Minute {
		newRate := a.currentRate * rate.Limit(a.recoveryRate)
		if newRate > a.baseRate {
			newRate = a.baseRate
		}

		a.currentRate = newRate
		a.limiter.SetLimit(newRate)
		a.lastAdjustment = time.Now()
	}
}

// GetStats returns statistics about the rate limiter
func (a *AdaptiveRateLimiter) GetStats() RateLimiterStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return RateLimiterStats{
		BaseRate:           float64(a.baseRate),
		CurrentRate:        float64(a.currentRate),
		RateLimitHits:      a.rateLimitHits,
		DynamicAdjustments: a.dynamicAdjustments,
		LastAdjustment:     a.lastAdjustment,
	}
}

// Reset resets the rate limiter to its base rate
func (a *AdaptiveRateLimiter) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.currentRate = a.baseRate
	a.limiter.SetLimit(a.baseRate)
	a.lastAdjustment = time.Now()
}

// RateLimiterStats holds statistics about rate limiting
type RateLimiterStats struct {
	BaseRate           float64   // Base requests per second
	CurrentRate        float64   // Current adjusted requests per second
	RateLimitHits      int64     // Total 429 responses received
	DynamicAdjustments int64     // Total adjustments made
	LastAdjustment     time.Time // Last time rate was adjusted
}

// String returns a human-readable string representation
func (s RateLimiterStats) String() string {
	return fmt.Sprintf("Rate Limiter Stats: base=%.4f rps, current=%.4f rps, hits=%d, adjustments=%d, last_adjustment=%s",
		s.BaseRate, s.CurrentRate, s.RateLimitHits, s.DynamicAdjustments, s.LastAdjustment.Format(time.RFC3339))
}
