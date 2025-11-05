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
	zlog zerolog.Logger
	file *os.File
}

// Config represents logger configuration
type Config struct {
	LogFilePath string // Full path to log file
	LogLevel    string // debug, info, warn, error
	LogToFile   bool   // Write to file
	LogToStdout bool   // Write to stdout
}

// New creates a new structured logger with zerolog
func New(cfg Config) (*Logger, error) {
	var writers []io.Writer

	// Add stdout if requested (with console formatting)
	if cfg.LogToStdout {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	var logFile *os.File
	var err error

	// Add file writer if requested (JSON format)
	if cfg.LogToFile && cfg.LogFilePath != "" {
		logDir := filepath.Dir(cfg.LogFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("creating log directory: %w", err)
		}

		logFile, err = os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}

		writers = append(writers, logFile)
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zlog := zerolog.New(multiWriter).With().Timestamp().Logger()

	// Set log level
	switch cfg.LogLevel {
	case "debug":
		zlog = zlog.Level(zerolog.DebugLevel)
	case "info":
		zlog = zlog.Level(zerolog.InfoLevel)
	case "warn":
		zlog = zlog.Level(zerolog.WarnLevel)
	case "error":
		zlog = zlog.Level(zerolog.ErrorLevel)
	default:
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	return &Logger{
		zlog: zlog,
		file: logFile,
	}, nil
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.zlog.Info().Msgf(format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	l.zlog.Error().Msgf(format, v...)
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	l.zlog.Debug().Msgf(format, v...)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.zlog.Fatal().Msgf(format, v...)
	_ = l.Close() // Ignore error since we're exiting
	os.Exit(1)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Structured logging methods with context

// LogStartup logs application startup with structured fields
func (l *Logger) LogStartup(version, region, brand string, pollInterval int) {
	l.zlog.Info().
		Str("version", version).
		Str("region", region).
		Str("brand", brand).
		Int("poll_interval_minutes", pollInterval).
		Msg("Hyundai Logger starting")
}

// LogShutdown logs application shutdown
func (l *Logger) LogShutdown() {
	l.zlog.Info().Msg("Hyundai Logger shutting down")
}

// LogError logs an error with context
func (l *Logger) LogError(operation string, err error) {
	if err != nil {
		l.zlog.Error().
			Str("operation", operation).
			Err(err).
			Msg("Operation failed")
	}
}

// LogVehicleDiscovery logs vehicle discovery
func (l *Logger) LogVehicleDiscovery(count int) {
	l.zlog.Info().
		Int("vehicle_count", count).
		Msg("Discovered vehicles")
}

// LogVehicleInfo logs individual vehicle information
func (l *Logger) LogVehicleInfo(year int, make, model, vin string) {
	l.zlog.Info().
		Int("year", year).
		Str("make", make).
		Str("model", model).
		Str("vin", vin).
		Msg("Vehicle registered")
}

// LogDataCollection logs data collection events
func (l *Logger) LogDataCollection(vin string, odometer, fuelLevel float64) {
	l.zlog.Info().
		Str("vin", vin).
		Float64("odometer", odometer).
		Float64("fuel_level", fuelLevel).
		Msg("Data collected")
}

// LogEVData logs EV-specific data
func (l *Logger) LogEVData(vin string, batteryLevel float64, charging bool) {
	l.zlog.Info().
		Str("vin", vin).
		Float64("battery_level", batteryLevel).
		Bool("charging", charging).
		Msg("EV data collected")
}

// LogLocation logs location data
func (l *Logger) LogLocation(vin string, lat, lon float64) {
	l.zlog.Info().
		Str("vin", vin).
		Float64("latitude", lat).
		Float64("longitude", lon).
		Msg("Location recorded")
}

// LogPollStart logs the start of a polling cycle
func (l *Logger) LogPollStart() {
	l.zlog.Info().Msg("Poll cycle starting")
}

// LogPollComplete logs the completion of a polling cycle
func (l *Logger) LogPollComplete(duration time.Duration) {
	l.zlog.Info().
		Dur("duration", duration).
		Float64("duration_seconds", duration.Seconds()).
		Msg("Poll cycle complete")
}

// LogDatabaseInit logs database initialization
func (l *Logger) LogDatabaseInit() {
	l.zlog.Info().Msg("Initializing database schema")
}

// LogDatabaseReady logs database ready state
func (l *Logger) LogDatabaseReady() {
	l.zlog.Info().Msg("Database connection established")
}

// LogAuthentication logs authentication events
func (l *Logger) LogAuthentication(success bool, region string) {
	l.zlog.Info().
		Bool("success", success).
		Str("region", region).
		Msg("API authentication attempt")
}

// LogRateLimit logs rate limiting configuration
func (l *Logger) LogRateLimit(requestsPerHour int) {
	l.zlog.Info().
		Int("requests_per_hour", requestsPerHour).
		Msg("Rate limiting configured")
}

// WithContext returns a logger with additional context fields
func (l *Logger) WithContext(fields map[string]interface{}) *Logger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{
		zlog: ctx.Logger(),
		file: l.file,
	}
}
