// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger provides structured logging with file and console output
type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger
	file     *os.File
}

// Config represents logger configuration
type Config struct {
	LogFilePath string // Full path to log file (e.g., /var/log/hyundai-logger/hyundai-logger.log)
	LogLevel    string // debug, info, warn, error
	LogToFile   bool   // Write to file
	LogToStdout bool   // Write to stdout
}

// New creates a new logger instance
func New(cfg Config) (*Logger, error) {
	var writers []io.Writer

	// Add stdout if requested
	if cfg.LogToStdout {
		writers = append(writers, os.Stdout)
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

		writers = append(writers, logFile)
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout) // Default to stdout
	}

	multiWriter := io.MultiWriter(writers...)

	logger := &Logger{
		infoLog:  log.New(multiWriter, "INFO:  ", log.Ldate|log.Ltime|log.LUTC),
		errorLog: log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile),
		debugLog: log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile),
		file:     logFile,
	}

	return logger, nil
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLog.Printf(format, v...)
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLog.Printf(format, v...)
}

// Debug logs debug messages
func (l *Logger) Debug(format string, v ...interface{}) {
	l.debugLog.Printf(format, v...)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.errorLog.Printf(format, v...)
	l.Close()
	os.Exit(1)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetStdLogger returns a standard log.Logger for compatibility
func (l *Logger) GetStdLogger() *log.Logger {
	return l.infoLog
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(version, region, brand string, pollInterval int) {
	l.Info("========================================")
	l.Info("Hyundai Logger v%s", version)
	l.Info("========================================")
	l.Info("Started at: %s", time.Now().UTC().Format(time.RFC3339))
	l.Info("Region: %s, Brand: %s", region, brand)
	l.Info("Poll interval: %d minutes", pollInterval)
	l.Info("========================================")
}

// LogShutdown logs application shutdown
func (l *Logger) LogShutdown() {
	l.Info("========================================")
	l.Info("Hyundai Logger Shutdown")
	l.Info("Stopped at: %s", time.Now().UTC().Format(time.RFC3339))
	l.Info("========================================")
}

// LogError logs an error with context
func (l *Logger) LogError(operation string, err error) {
	if err != nil {
		l.Error("%s failed: %v", operation, err)
	}
}

// LogVehicleDiscovery logs vehicle discovery information
func (l *Logger) LogVehicleDiscovery(count int) {
	l.Info("Found %d vehicle(s) in account", count)
}

// LogVehicleInfo logs individual vehicle information
func (l *Logger) LogVehicleInfo(year int, make, model, vin string) {
	l.Info("Vehicle: %d %s %s (VIN: %s)", year, make, model, vin)
}

// LogDataCollection logs data collection events
func (l *Logger) LogDataCollection(vin string, odometer, fuelLevel float64) {
	l.Info("Collected data for %s - Odometer: %.1f, Fuel: %.1f%%", vin, odometer, fuelLevel)
}

// LogEVData logs EV-specific data
func (l *Logger) LogEVData(vin string, batteryLevel float64, charging bool) {
	chargingStatus := "not charging"
	if charging {
		chargingStatus = "charging"
	}
	l.Info("EV data for %s - Battery: %.1f%%, Status: %s", vin, batteryLevel, chargingStatus)
}

// LogLocation logs location data
func (l *Logger) LogLocation(vin string, lat, lon float64) {
	l.Info("Location for %s - Lat: %.6f, Lon: %.6f", vin, lat, lon)
}

// LogPollStart logs the start of a polling cycle
func (l *Logger) LogPollStart() {
	l.Info("Starting vehicle data poll...")
}

// LogPollComplete logs the completion of a polling cycle
func (l *Logger) LogPollComplete(duration time.Duration) {
	l.Info("Poll completed in %.2f seconds", duration.Seconds())
}

// LogDatabaseInit logs database initialization
func (l *Logger) LogDatabaseInit() {
	l.Info("Initializing database schema...")
}

// LogDatabaseReady logs database ready state
func (l *Logger) LogDatabaseReady() {
	l.Info("Database connection established and ready")
}

// LogAuthentication logs authentication events
func (l *Logger) LogAuthentication(success bool, region string) {
	if success {
		l.Info("Successfully authenticated with Hyundai API (%s region)", region)
	} else {
		l.Error("Failed to authenticate with Hyundai API (%s region)", region)
	}
}

// LogRateLimit logs rate limiting events
func (l *Logger) LogRateLimit(requestsPerHour int) {
	l.Info("Rate limiting: %d requests per hour", requestsPerHour)
}
