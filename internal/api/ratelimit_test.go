// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *RateLimitError
		wantSubstr string
	}{
		{
			name: "with retry after",
			err: &RateLimitError{
				RetryAfter: 60 * time.Second,
				Message:    "too many requests",
			},
			wantSubstr: "retry after 1m0s",
		},
		{
			name: "without retry after",
			err: &RateLimitError{
				RetryAfter: 0,
				Message:    "rate limited",
			},
			wantSubstr: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !containsStr(msg, tt.wantSubstr) {
				t.Errorf("Error message %q does not contain %q", msg, tt.wantSubstr)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	rateLimitErr := &RateLimitError{Message: "test"}

	if !IsRateLimitError(rateLimitErr) {
		t.Error("Expected IsRateLimitError to return true for RateLimitError")
	}

	regularErr := http.ErrHandlerTimeout
	if IsRateLimitError(regularErr) {
		t.Error("Expected IsRateLimitError to return false for regular error")
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	tests := []struct {
		header   string
		expected time.Duration
	}{
		{"60", 60 * time.Second},
		{"120", 120 * time.Second},
		{"1", 1 * time.Second},
		{"3600", 3600 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := parseRetryAfter(tt.header)
			if got != tt.expected {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.header, got, tt.expected)
			}
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test with a future time
	future := time.Now().Add(2 * time.Minute)
	httpDate := future.Format(http.TimeFormat)

	duration := parseRetryAfter(httpDate)

	// Should be approximately 2 minutes (allow some tolerance)
	if duration < 115*time.Second || duration > 125*time.Second {
		t.Errorf("parseRetryAfter with HTTP date = %v, want ~2m", duration)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	tests := []string{
		"",
		"invalid",
		"not-a-number",
		"abc123",
	}

	for _, header := range tests {
		t.Run(header, func(t *testing.T) {
			got := parseRetryAfter(header)
			expected := 60 * time.Second // Default

			if got != expected {
				t.Errorf("parseRetryAfter(%q) = %v, want default %v", header, got, expected)
			}
		})
	}
}

func TestCheckRateLimit_429(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
	}
	resp.Header.Set("Retry-After", "60")

	err := checkRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error for 429 status")
	}

	rateLimitErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("Expected *RateLimitError, got %T", err)
	}

	if rateLimitErr.RetryAfter != 60*time.Second {
		t.Errorf("Expected RetryAfter 60s, got %v", rateLimitErr.RetryAfter)
	}
}

func TestCheckRateLimit_403WithRateLimitHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
	}
	resp.Header.Set("X-RateLimit-Remaining", "0")
	resp.Header.Set("X-RateLimit-Reset", "120")

	err := checkRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error for 403 with rate limit headers")
	}

	rateLimitErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("Expected *RateLimitError, got %T", err)
	}

	if rateLimitErr.RetryAfter != 120*time.Second {
		t.Errorf("Expected RetryAfter 120s, got %v", rateLimitErr.RetryAfter)
	}
}

func TestCheckRateLimit_403WithoutRateLimitHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
	}

	err := checkRateLimit(resp)

	if err != nil {
		t.Errorf("Expected no error for 403 without rate limit headers, got %v", err)
	}
}

func TestCheckRateLimit_200(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
	}

	err := checkRateLimit(resp)

	if err != nil {
		t.Errorf("Expected no error for 200 status, got %v", err)
	}
}

func TestGetRateLimitHeaders_Standard(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("X-RateLimit-Limit", "1000")
	resp.Header.Set("X-RateLimit-Remaining", "500")
	resp.Header.Set("X-RateLimit-Reset", "1609459200")

	limit, remaining, reset := getRateLimitHeaders(resp)

	if limit != "1000" {
		t.Errorf("Expected limit '1000', got %q", limit)
	}
	if remaining != "500" {
		t.Errorf("Expected remaining '500', got %q", remaining)
	}
	if reset != "1609459200" {
		t.Errorf("Expected reset '1609459200', got %q", reset)
	}
}

func TestGetRateLimitHeaders_Alternative(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	// Some APIs use alternative header names without X- prefix
	resp.Header.Set("RateLimit-Limit", "2000")
	resp.Header.Set("RateLimit-Remaining", "1000")
	resp.Header.Set("RateLimit-Reset", "3600")

	limit, remaining, reset := getRateLimitHeaders(resp)

	if limit != "2000" {
		t.Errorf("Expected limit '2000', got %q", limit)
	}
	if remaining != "1000" {
		t.Errorf("Expected remaining '1000', got %q", remaining)
	}
	if reset != "3600" {
		t.Errorf("Expected reset '3600', got %q", reset)
	}
}

func TestGetRateLimitHeaders_Empty(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	limit, remaining, reset := getRateLimitHeaders(resp)

	if limit != "" || remaining != "" || reset != "" {
		t.Errorf("Expected empty headers, got limit=%q remaining=%q reset=%q", limit, remaining, reset)
	}
}

func TestExtractRateLimitInfo_WithHeaders(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
	}
	resp.Header.Set("X-RateLimit-Limit", "100")
	resp.Header.Set("X-RateLimit-Remaining", "0")
	resp.Header.Set("X-RateLimit-Reset", "60")
	resp.Header.Set("Retry-After", "30")

	info := ExtractRateLimitInfo(resp)

	if info == nil {
		t.Fatal("Expected non-nil RateLimitInfo")
	}

	if info.Limit != "100" {
		t.Errorf("Expected Limit '100', got %q", info.Limit)
	}
	if info.Remaining != "0" {
		t.Errorf("Expected Remaining '0', got %q", info.Remaining)
	}
	if info.Reset != "60" {
		t.Errorf("Expected Reset '60', got %q", info.Reset)
	}
	if info.RetryAfter != 30*time.Second {
		t.Errorf("Expected RetryAfter 30s, got %v", info.RetryAfter)
	}
	if info.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusTooManyRequests, info.StatusCode)
	}
}

func TestExtractRateLimitInfo_Nil(t *testing.T) {
	info := ExtractRateLimitInfo(nil)

	if info != nil {
		t.Error("Expected nil for nil response")
	}
}

func TestShouldBackoff_BelowThreshold(t *testing.T) {
	// 10 remaining out of 100 = 10% (below 20% threshold)
	if !ShouldBackoff("100", "10") {
		t.Error("Expected true when below 20% threshold")
	}
}

func TestShouldBackoff_AboveThreshold(t *testing.T) {
	// 30 remaining out of 100 = 30% (above 20% threshold)
	if ShouldBackoff("100", "30") {
		t.Error("Expected false when above 20% threshold")
	}
}

func TestShouldBackoff_AtThreshold(t *testing.T) {
	// 20 remaining out of 100 = 20% (at threshold, should not backoff)
	if ShouldBackoff("100", "20") {
		t.Error("Expected false when exactly at 20% threshold")
	}
}

func TestShouldBackoff_EmptyHeaders(t *testing.T) {
	tests := []struct {
		name      string
		limit     string
		remaining string
	}{
		{"both empty", "", ""},
		{"limit empty", "", "50"},
		{"remaining empty", "100", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ShouldBackoff(tt.limit, tt.remaining) {
				t.Error("Expected false for empty headers")
			}
		})
	}
}

func TestShouldBackoff_ZeroLimit(t *testing.T) {
	if ShouldBackoff("0", "10") {
		t.Error("Expected false for zero limit")
	}
}

func TestShouldBackoff_InvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		limit     string
		remaining string
	}{
		{"non-numeric limit", "abc", "50"},
		{"non-numeric remaining", "100", "xyz"},
		{"both non-numeric", "abc", "xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic and should return false
			result := ShouldBackoff(tt.limit, tt.remaining)
			if result {
				t.Error("Expected false for invalid values")
			}
		})
	}
}

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
