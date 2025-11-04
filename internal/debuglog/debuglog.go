// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package debuglog

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents logging level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides detailed debug logging with timing information
type Logger struct {
	mu          sync.Mutex
	output      io.Writer
	level       Level
	timestamps  bool
	colorize    bool
	includeTime bool
}

// Config holds logger configuration
type Config struct {
	Output      io.Writer // Where to write logs (default: os.Stdout)
	Level       Level     // Minimum level to log
	Timestamps  bool      // Include timestamps
	Colorize    bool      // Use ANSI colors (for terminals)
	IncludeTime bool      // Include detailed timing info
}

// New creates a new debug logger
func New(config Config) *Logger {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	return &Logger{
		output:      config.Output,
		level:       config.Level,
		timestamps:  config.Timestamps,
		colorize:    config.Colorize,
		includeTime: config.IncludeTime,
	}
}

// NewDefault creates a logger with default settings
func NewDefault() *Logger {
	return New(Config{
		Output:      os.Stdout,
		Level:       LevelInfo,
		Timestamps:  true,
		Colorize:    true,
		IncludeTime: false,
	})
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns current logging level
func (l *Logger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// log is the internal logging function
func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we should log this level
	if level < l.level {
		return
	}

	// Build log message
	var msg string

	if l.timestamps {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		msg = fmt.Sprintf("[%s] ", timestamp)
	}

	// Add level with optional color
	if l.colorize {
		msg += l.colorizeLevel(level)
	} else {
		msg += fmt.Sprintf("[%s] ", level.String())
	}

	// Add the actual message
	msg += fmt.Sprintf(format, args...)

	// Write to output
	fmt.Fprintln(l.output, msg)
}

// colorizeLevel returns a colorized level string
func (l *Logger) colorizeLevel(level Level) string {
	const (
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorGray   = "\033[90m"
	)

	switch level {
	case LevelDebug:
		return colorGray + "[DEBUG] " + colorReset
	case LevelInfo:
		return colorBlue + "[INFO]  " + colorReset
	case LevelWarn:
		return colorYellow + "[WARN]  " + colorReset
	case LevelError:
		return colorRed + "[ERROR] " + colorReset
	default:
		return "[UNKNOWN] "
	}
}

// Timer tracks operation timing
type Timer struct {
	logger      *Logger
	operation   string
	start       time.Time
	checkpoints []checkpoint
}

type checkpoint struct {
	name     string
	duration time.Duration
}

// StartTimer begins timing an operation
func (l *Logger) StartTimer(operation string) *Timer {
	return &Timer{
		logger:    l,
		operation: operation,
		start:     time.Now(),
	}
}

// Checkpoint records a timing checkpoint
func (t *Timer) Checkpoint(name string) {
	duration := time.Since(t.start)
	t.checkpoints = append(t.checkpoints, checkpoint{
		name:     name,
		duration: duration,
	})
}

// End completes timing and logs the results
func (t *Timer) End(format string, args ...interface{}) {
	totalDuration := time.Since(t.start)

	if !t.logger.includeTime {
		// Simple log without timing details
		msg := fmt.Sprintf(format, args...)
		t.logger.Debug("%s: %s (%s)", t.operation, msg, totalDuration.Round(time.Millisecond))
		return
	}

	// Detailed timing log
	t.logger.Debug("%s: Starting detailed timing analysis", t.operation)

	for _, cp := range t.checkpoints {
		t.logger.Debug("  → %s: %s", cp.name, cp.duration.Round(time.Millisecond))
	}

	msg := fmt.Sprintf(format, args...)
	t.logger.Debug("  → Total: %s - %s", totalDuration.Round(time.Millisecond), msg)
}

// Context-aware logging
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	vehicleIDKey contextKey = "vehicle_id"
)

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithVehicleID adds vehicle ID to context
func WithVehicleID(ctx context.Context, vehicleID string) context.Context {
	return context.WithValue(ctx, vehicleIDKey, vehicleID)
}

// DebugContext logs with context information
func (l *Logger) DebugContext(ctx context.Context, format string, args ...interface{}) {
	msg := l.formatWithContext(ctx, format, args...)
	l.log(LevelDebug, "%s", msg)
}

// InfoContext logs with context information
func (l *Logger) InfoContext(ctx context.Context, format string, args ...interface{}) {
	msg := l.formatWithContext(ctx, format, args...)
	l.log(LevelInfo, "%s", msg)
}

// WarnContext logs with context information
func (l *Logger) WarnContext(ctx context.Context, format string, args ...interface{}) {
	msg := l.formatWithContext(ctx, format, args...)
	l.log(LevelWarn, "%s", msg)
}

// ErrorContext logs with context information
func (l *Logger) ErrorContext(ctx context.Context, format string, args ...interface{}) {
	msg := l.formatWithContext(ctx, format, args...)
	l.log(LevelError, "%s", msg)
}

// formatWithContext adds context values to log message
func (l *Logger) formatWithContext(ctx context.Context, format string, args ...interface{}) string {
	var prefix string

	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		prefix += fmt.Sprintf("[req:%s] ", requestID)
	}

	if vehicleID, ok := ctx.Value(vehicleIDKey).(string); ok {
		prefix += fmt.Sprintf("[vin:%s] ", vehicleID)
	}

	return prefix + fmt.Sprintf(format, args...)
}

// HTTPRequestLogger logs HTTP request details
func (l *Logger) HTTPRequest(method, url string, duration time.Duration, statusCode int) {
	if statusCode >= 400 {
		l.Error("HTTP %s %s → %d (%s)", method, url, statusCode, duration.Round(time.Millisecond))
	} else {
		l.Debug("HTTP %s %s → %d (%s)", method, url, statusCode, duration.Round(time.Millisecond))
	}
}

// APICallLogger logs API call details with breakdown
func (l *Logger) APICall(operation string, duration time.Duration, details map[string]interface{}) {
	msg := fmt.Sprintf("API %s completed in %s", operation, duration.Round(time.Millisecond))

	if len(details) > 0 {
		msg += " - "
		first := true
		for k, v := range details {
			if !first {
				msg += ", "
			}
			msg += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
	}

	l.Debug("%s", msg)
}

// DatabaseLogger logs database operations
func (l *Logger) DatabaseWrite(bucket string, points int, duration time.Duration, err error) {
	if err != nil {
		l.Error("Database write to %s failed: %d points in %s - %v",
			bucket, points, duration.Round(time.Millisecond), err)
	} else {
		l.Debug("Database write to %s: %d points in %s",
			bucket, points, duration.Round(time.Millisecond))
	}
}

// CircuitBreakerLogger logs circuit breaker state changes
func (l *Logger) CircuitBreakerStateChange(name, oldState, newState string) {
	l.Warn("Circuit breaker '%s' state change: %s → %s", name, oldState, newState)
}

// RetryLogger logs retry attempts
func (l *Logger) RetryAttempt(operation string, attempt int, maxAttempts int, delay time.Duration, err error) {
	l.Warn("Retry %d/%d for %s after %s: %v",
		attempt, maxAttempts, operation, delay.Round(time.Millisecond), err)
}

// CacheLogger logs cache operations
func (l *Logger) CacheHit(key string) {
	l.Debug("Cache HIT: %s", key)
}

// CacheMiss logs cache misses
func (l *Logger) CacheMiss(key string) {
	l.Debug("Cache MISS: %s", key)
}

// RateLimitLogger logs rate limit events
func (l *Logger) RateLimit(operation string, limit int, remaining int, resetTime time.Time) {
	l.Warn("Rate limit: %s - %d/%d remaining, resets at %s",
		operation, remaining, limit, resetTime.Format("15:04:05"))
}
