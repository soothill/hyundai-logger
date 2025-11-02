// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RateLimitError represents a 429 Too Many Requests error
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited: %s (retry after %s)", e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("rate limited: %s", e.Message)
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// parseRetryAfter parses the Retry-After header
// Supports both delay-seconds and HTTP-date formats
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 60 * time.Second // Default to 60 seconds if not specified
	}

	// Try to parse as seconds (most common)
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as HTTP-date (RFC 1123)
	if t, err := http.ParseTime(header); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	// Default to 60 seconds if we can't parse
	return 60 * time.Second
}

// checkRateLimit checks the response for rate limiting and returns appropriate error
func checkRateLimit(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &RateLimitError{
			RetryAfter: retryAfter,
			Message:    "API rate limit exceeded",
		}
	}

	// Check for other rate limit indicators
	if resp.StatusCode == http.StatusForbidden {
		// Some APIs return 403 for rate limits
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			retryAfter := parseRetryAfter(resp.Header.Get("X-RateLimit-Reset"))
			return &RateLimitError{
				RetryAfter: retryAfter,
				Message:    "API rate limit exceeded (403)",
			}
		}
	}

	return nil
}

// getRateLimitHeaders extracts rate limit information from response headers
func getRateLimitHeaders(resp *http.Response) (limit, remaining, reset string) {
	// Standard headers
	limit = resp.Header.Get("X-RateLimit-Limit")
	remaining = resp.Header.Get("X-RateLimit-Remaining")
	reset = resp.Header.Get("X-RateLimit-Reset")

	// Alternative header names (some APIs use different conventions)
	if limit == "" {
		limit = resp.Header.Get("RateLimit-Limit")
	}
	if remaining == "" {
		remaining = resp.Header.Get("RateLimit-Remaining")
	}
	if reset == "" {
		reset = resp.Header.Get("RateLimit-Reset")
	}

	return
}

// RateLimitInfo contains detailed rate limit information
type RateLimitInfo struct {
	Limit      string
	Remaining  string
	Reset      string
	RetryAfter time.Duration
	StatusCode int
}

// ExtractRateLimitInfo extracts comprehensive rate limit information from a response
func ExtractRateLimitInfo(resp *http.Response) *RateLimitInfo {
	if resp == nil {
		return nil
	}

	limit, remaining, reset := getRateLimitHeaders(resp)

	retryAfter := time.Duration(0)
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}

	return &RateLimitInfo{
		Limit:      limit,
		Remaining:  remaining,
		Reset:      reset,
		RetryAfter: retryAfter,
		StatusCode: resp.StatusCode,
	}
}

// ShouldBackoff determines if we should back off based on rate limit headers
// Returns true if remaining requests are running low (below 20% of limit)
func ShouldBackoff(limit, remaining string) bool {
	if limit == "" || remaining == "" {
		return false
	}

	limitVal, err := strconv.Atoi(limit)
	if err != nil {
		return false // Invalid limit value
	}

	remainingVal, err := strconv.Atoi(remaining)
	if err != nil {
		return false // Invalid remaining value
	}

	if limitVal == 0 {
		return false
	}

	// Back off if we're below 20% of our quota
	threshold := float64(limitVal) * 0.2
	return float64(remainingVal) < threshold
}
