// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package debuglog

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: true,
		Colorize:   false,
	})

	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected 'test message' in output, got: %s", output)
	}
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault()
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	if logger.level != LevelInfo {
		t.Errorf("expected level Info, got %v", logger.level)
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output: &buf,
		Level:  LevelInfo,
	})

	logger.Debug("should not appear")
	if buf.Len() > 0 {
		t.Error("debug message should not appear at Info level")
	}

	logger.SetLevel(LevelDebug)
	logger.Debug("should appear")

	if buf.Len() == 0 {
		t.Error("debug message should appear after setting Debug level")
	}
}

func TestGetLevel(t *testing.T) {
	logger := New(Config{
		Level: LevelWarn,
	})

	if logger.GetLevel() != LevelWarn {
		t.Errorf("expected Warn level, got %v", logger.GetLevel())
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	output := buf.String()

	if !strings.Contains(output, "DEBUG") {
		t.Error("expected DEBUG in output")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("expected INFO in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("expected WARN in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR in output")
	}

	if !strings.Contains(output, "debug msg") {
		t.Error("expected 'debug msg' in output")
	}
}

func TestLogFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelWarn,
		Timestamps: false,
	})

	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should appear")
	logger.Error("should appear")

	output := buf.String()

	if strings.Contains(output, "DEBUG") {
		t.Error("DEBUG should be filtered")
	}
	if strings.Contains(output, "INFO") {
		t.Error("INFO should be filtered")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("WARN should appear")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("ERROR should appear")
	}
}

func TestTimestamps(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelInfo,
		Timestamps: true,
		Colorize:   false,
	})

	logger.Info("test")

	output := buf.String()

	// Should contain timestamp in format YYYY-MM-DD HH:MM:SS
	if !strings.Contains(output, "-") || !strings.Contains(output, ":") {
		t.Errorf("expected timestamp in output, got: %s", output)
	}
}

func TestTimer(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:      &buf,
		Level:       LevelDebug,
		Timestamps:  false,
		Colorize:    false,
		IncludeTime: false,
	})

	timer := logger.StartTimer("test_operation")
	time.Sleep(10 * time.Millisecond)
	timer.End("completed successfully")

	output := buf.String()

	if !strings.Contains(output, "test_operation") {
		t.Error("expected operation name in output")
	}
	if !strings.Contains(output, "completed successfully") {
		t.Error("expected completion message in output")
	}
	if !strings.Contains(output, "ms") {
		t.Error("expected timing information in output")
	}
}

func TestTimerCheckpoints(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:      &buf,
		Level:       LevelDebug,
		Timestamps:  false,
		Colorize:    false,
		IncludeTime: true,
	})

	timer := logger.StartTimer("complex_operation")
	time.Sleep(5 * time.Millisecond)
	timer.Checkpoint("step1")
	time.Sleep(5 * time.Millisecond)
	timer.Checkpoint("step2")
	timer.End("done")

	output := buf.String()

	if !strings.Contains(output, "step1") {
		t.Error("expected checkpoint 'step1' in output")
	}
	if !strings.Contains(output, "step2") {
		t.Error("expected checkpoint 'step2' in output")
	}
	if !strings.Contains(output, "Total") {
		t.Error("expected 'Total' in output")
	}
}

func TestContextLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithVehicleID(ctx, "VIN-456")

	logger.InfoContext(ctx, "test message")

	output := buf.String()

	if !strings.Contains(output, "req:req-123") {
		t.Error("expected request ID in output")
	}
	if !strings.Contains(output, "vin:VIN-456") {
		t.Error("expected vehicle ID in output")
	}
	if !strings.Contains(output, "test message") {
		t.Error("expected 'test message' in output")
	}
}

func TestHTTPRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	// Successful request
	logger.HTTPRequest("GET", "https://api.example.com", 100*time.Millisecond, 200)
	output := buf.String()

	if !strings.Contains(output, "GET") {
		t.Error("expected method in output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected status code in output")
	}

	// Failed request
	buf.Reset()
	logger.HTTPRequest("POST", "https://api.example.com", 200*time.Millisecond, 500)
	output = buf.String()

	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR level for failed request")
	}
	if !strings.Contains(output, "500") {
		t.Error("expected status code 500 in output")
	}
}

func TestAPICallLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	details := map[string]interface{}{
		"vehicle_id": "ABC123",
		"points":     5,
	}

	logger.APICall("GetVehicleStatus", 150*time.Millisecond, details)

	output := buf.String()

	if !strings.Contains(output, "GetVehicleStatus") {
		t.Error("expected operation name in output")
	}
	if !strings.Contains(output, "150ms") || !strings.Contains(output, "ms") {
		t.Error("expected duration in output")
	}
	if !strings.Contains(output, "vehicle_id") {
		t.Error("expected details in output")
	}
}

func TestDatabaseLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	// Successful write
	logger.DatabaseWrite("test-bucket", 10, 50*time.Millisecond, nil)
	output := buf.String()

	if !strings.Contains(output, "test-bucket") {
		t.Error("expected bucket name in output")
	}
	if !strings.Contains(output, "10 points") {
		t.Error("expected point count in output")
	}

	// Failed write
	buf.Reset()
	logger.DatabaseWrite("test-bucket", 10, 50*time.Millisecond,
		&testError{msg: "connection failed"})
	output = buf.String()

	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR level for failed write")
	}
	if !strings.Contains(output, "connection failed") {
		t.Error("expected error message in output")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestCircuitBreakerLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelWarn,
		Timestamps: false,
		Colorize:   false,
	})

	logger.CircuitBreakerStateChange("api_client", "closed", "open")

	output := buf.String()

	if !strings.Contains(output, "Circuit breaker") {
		t.Error("expected 'Circuit breaker' in output")
	}
	if !strings.Contains(output, "closed → open") {
		t.Error("expected state transition in output")
	}
}

func TestRetryLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelWarn,
		Timestamps: false,
		Colorize:   false,
	})

	logger.RetryAttempt("api_call", 2, 3, 1*time.Second, &testError{msg: "timeout"})

	output := buf.String()

	if !strings.Contains(output, "Retry 2/3") {
		t.Error("expected retry count in output")
	}
	if !strings.Contains(output, "timeout") {
		t.Error("expected error in output")
	}
}

func TestCacheLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   false,
	})

	logger.CacheHit("status:ABC123")
	logger.CacheMiss("status:XYZ789")

	output := buf.String()

	if !strings.Contains(output, "Cache HIT") {
		t.Error("expected cache hit in output")
	}
	if !strings.Contains(output, "Cache MISS") {
		t.Error("expected cache miss in output")
	}
}

func TestRateLimitLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelWarn,
		Timestamps: false,
		Colorize:   false,
	})

	resetTime := time.Now().Add(1 * time.Hour)
	logger.RateLimit("GetVehicleStatus", 50, 10, resetTime)

	output := buf.String()

	if !strings.Contains(output, "Rate limit") {
		t.Error("expected 'Rate limit' in output")
	}
	if !strings.Contains(output, "10/50") {
		t.Error("expected remaining count in output")
	}
}

func TestColorize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelDebug,
		Timestamps: false,
		Colorize:   true,
	})

	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	output := buf.String()

	// Should contain ANSI escape codes
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI color codes in output")
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Output:     &buf,
		Level:      LevelInfo,
		Timestamps: false,
		Colorize:   false,
	})

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				logger.Info("goroutine %d message %d", n, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 100 {
		t.Errorf("expected 100 log lines, got %d", len(lines))
	}
}
