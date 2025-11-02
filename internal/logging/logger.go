// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// Logger provides structured logging with zerolog
type Logger struct {
	logger zerolog.Logger
	file   *os.File
}

// Config represents logger configuration
type Config struct {
	LogFilePath string // Full path to log file (e.g., /var/log/hyundai-logger/hyundai-logger.log)
	LogLevel    string // debug, info, warn, error
	LogToFile   bool   // Write to file
	LogToStdout bool   // Write to stdout
}

// New creates a new logger instance with zerolog
func New(cfg Config) (*Logger, error) {
	var writers []io.Writer

	// Add stdout if requested
	if cfg.LogToStdout {
		// Use console writer for human-readable output to stdout
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	var logFile *os.File
	var err error

	// Add file writer if requested
	if cfg.LogToFile && cfg.LogFilePath != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(cfg.LogFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("creating log directory: %w", err)
		}

		// Open log file with append mode
		logFile, err = os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}

		// Write JSON to file for machine parsing
		writers = append(writers, logFile)
	}

	if len(writers) == 0 {
		// Default to console output on stdout
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	multiWriter := io.MultiWriter(writers...)

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	logger := zerolog.New(multiWriter).With().Timestamp().Caller().Logger()

	// Set log level
	level := parseLogLevel(cfg.LogLevel)
	logger = logger.Level(level)

	return &Logger{
		logger: logger,
		file:   logFile,
	}, nil
}

// parseLogLevel converts string log level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.logger.Info().Msgf(format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	l.logger.Debug().Msgf(format, v...)
}

// Warn logs warning messages
func (l *Logger) Warn(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v...)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.logger.Fatal().Msgf(format, v...)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// With returns a logger with contextual fields
func (l *Logger) With() *LogContext {
	return &LogContext{event: l.logger.With()}
}

// LogContext provides a fluent interface for adding contextual fields
type LogContext struct {
	event zerolog.Context
}

// Str adds a string field
func (c *LogContext) Str(key, value string) *LogContext {
	c.event = c.event.Str(key, value)
	return c
}

// Int adds an integer field
func (c *LogContext) Int(key string, value int) *LogContext {
	c.event = c.event.Int(key, value)
	return c
}

// Float64 adds a float64 field
func (c *LogContext) Float64(key string, value float64) *LogContext {
	c.event = c.event.Float64(key, value)
	return c
}

// Bool adds a boolean field
func (c *LogContext) Bool(key string, value bool) *LogContext {
	c.event = c.event.Bool(key, value)
	return c
}

// Err adds an error field
func (c *LogContext) Err(err error) *LogContext {
	c.event = c.event.Err(err)
	return c
}

// Logger returns the logger with the contextual fields
func (c *LogContext) Logger() *Logger {
	return &Logger{logger: c.event.Logger()}
}

// LogStartup logs application startup information with structured fields
func (l *Logger) LogStartup(version, region, brand string, pollInterval int) {
	l.logger.Info().
		Str("version", version).
		Str("region", region).
		Str("brand", brand).
		Int("poll_interval_minutes", pollInterval).
		Msg("Hyundai Logger started")
}

// LogShutdown logs application shutdown
func (l *Logger) LogShutdown() {
	l.logger.Info().Msg("Hyundai Logger shutdown")
}

// LogError logs an error with context
func (l *Logger) LogError(operation string, err error) {
	if err != nil {
		l.logger.Error().
			Err(err).
			Str("operation", operation).
			Msg("Operation failed")
	}
}

// LogVehicleDiscovery logs vehicle discovery information
func (l *Logger) LogVehicleDiscovery(count int) {
	l.logger.Info().
		Int("vehicle_count", count).
		Msg("Discovered vehicles")
}

// LogVehicleInfo logs individual vehicle information
func (l *Logger) LogVehicleInfo(year int, make, model, vin string) {
	l.logger.Info().
		Int("year", year).
		Str("make", make).
		Str("model", model).
		Str("vin", vin).
		Msg("Vehicle information")
}

// LogDataCollection logs data collection events
func (l *Logger) LogDataCollection(vin string, odometer, fuelLevel float64) {
	l.logger.Info().
		Str("vin", vin).
		Float64("odometer", odometer).
		Float64("fuel_level", fuelLevel).
		Msg("Data collected")
}

// LogEVData logs EV-specific data
func (l *Logger) LogEVData(vin string, batteryLevel float64, charging bool) {
	l.logger.Info().
		Str("vin", vin).
		Float64("battery_level", batteryLevel).
		Bool("charging", charging).
		Msg("EV data collected")
}

// LogLocation logs location data
func (l *Logger) LogLocation(vin string, lat, lon float64) {
	l.logger.Info().
		Str("vin", vin).
		Float64("latitude", lat).
		Float64("longitude", lon).
		Msg("Location data")
}

// LogPollStart logs the start of a polling cycle
func (l *Logger) LogPollStart() {
	l.logger.Info().Msg("Starting vehicle data poll")
}

// LogPollComplete logs the completion of a polling cycle
func (l *Logger) LogPollComplete(duration time.Duration) {
	l.logger.Info().
		Float64("duration_seconds", duration.Seconds()).
		Dur("duration", duration).
		Msg("Poll completed")
}

// LogDatabaseInit logs database initialization
func (l *Logger) LogDatabaseInit() {
	l.logger.Info().Msg("Initializing database schema")
}

// LogDatabaseReady logs database ready state
func (l *Logger) LogDatabaseReady() {
	l.logger.Info().Msg("Database connection established and ready")
}

// LogAuthentication logs authentication events
func (l *Logger) LogAuthentication(success bool, region string) {
	if success {
		l.logger.Info().
			Str("region", region).
			Msg("Authentication successful")
	} else {
		l.logger.Error().
			Str("region", region).
			Msg("Authentication failed")
	}
}

// LogRateLimit logs rate limiting events
func (l *Logger) LogRateLimit(requestsPerHour int) {
	l.logger.Info().
		Int("requests_per_hour", requestsPerHour).
		Msg("Rate limiting configured")
}
