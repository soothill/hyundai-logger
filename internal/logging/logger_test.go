// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package logging

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "stdout only",
			config: Config{
				LogToStdout: true,
				LogLevel:    "info",
			},
			wantErr: false,
		},
		{
			name: "file only",
			config: Config{
				LogFilePath: filepath.Join(t.TempDir(), "test.log"),
				LogToFile:   true,
				LogLevel:    "debug",
			},
			wantErr: false,
		},
		{
			name: "both stdout and file",
			config: Config{
				LogFilePath: filepath.Join(t.TempDir(), "test.log"),
				LogToFile:   true,
				LogToStdout: true,
				LogLevel:    "warn",
			},
			wantErr: false,
		},
		{
			name: "default (no outputs specified)",
			config: Config{
				LogLevel: "info",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer logger.Close()
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "debug",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test different log levels
	logger.Debug("debug message: %s", "test")
	logger.Info("info message: %s", "test")
	logger.Error("error message: %s", "test")

	logger.Close()

	// Read log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify all levels were logged
	if !strings.Contains(logContent, "debug message") {
		t.Error("Debug message not found in log")
	}
	if !strings.Contains(logContent, "info message") {
		t.Error("Info message not found in log")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("Error message not found in log")
	}
}

func TestLogFiltering(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Create logger with info level (should filter out debug)
	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Debug("debug message should be filtered")
	logger.Info("info message should appear")

	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Debug should be filtered out
	if strings.Contains(logContent, "debug message should be filtered") {
		t.Error("Debug message should have been filtered at info level")
	}

	// Info should appear
	if !strings.Contains(logContent, "info message should appear") {
		t.Error("Info message should appear at info level")
	}
}

func TestStructuredLogging(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log with structured fields
	logger.LogStartup("1.0.0", "US", "hyundai", 5)
	logger.LogVehicleDiscovery(3)
	logger.LogVehicleInfo(2023, "Hyundai", "Ioniq 5", "VIN123")
	logger.LogDataCollection("VIN123", 12345.6, 75.5)
	logger.LogEVData("VIN123", 85.5, true)
	logger.LogLocation("VIN123", 37.7749, -122.4194)
	logger.LogPollComplete(5 * time.Second)
	logger.LogAuthentication(true, "US")
	logger.LogRateLimit(100)

	logger.Close()

	// Read and parse JSON log entries
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	validEntries := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Failed to parse JSON log entry: %v\nLine: %s", err, line)
			continue
		}

		validEntries++

		// Verify required fields exist
		if _, ok := entry["level"]; !ok {
			t.Error("Log entry missing 'level' field")
		}
		if _, ok := entry["time"]; !ok {
			t.Error("Log entry missing 'time' field")
		}
		if _, ok := entry["message"]; !ok {
			t.Error("Log entry missing 'message' field")
		}
	}

	if validEntries == 0 {
		t.Error("No valid JSON log entries found")
	}
}

func TestLogStartup(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogStartup("1.2.3", "EU", "kia", 10)
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["version"] != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got %v", entry["version"])
	}
	if entry["region"] != "EU" {
		t.Errorf("Expected region 'EU', got %v", entry["region"])
	}
	if entry["brand"] != "kia" {
		t.Errorf("Expected brand 'kia', got %v", entry["brand"])
	}
	// JSON numbers are float64
	if entry["poll_interval_minutes"] != float64(10) {
		t.Errorf("Expected poll_interval_minutes 10, got %v", entry["poll_interval_minutes"])
	}
}

func TestLogVehicleInfo(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogVehicleInfo(2024, "Kia", "EV6", "TESTVIN123")
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["year"] != float64(2024) {
		t.Errorf("Expected year 2024, got %v", entry["year"])
	}
	if entry["make"] != "Kia" {
		t.Errorf("Expected make 'Kia', got %v", entry["make"])
	}
	if entry["model"] != "EV6" {
		t.Errorf("Expected model 'EV6', got %v", entry["model"])
	}
	if entry["vin"] != "TESTVIN123" {
		t.Errorf("Expected vin 'TESTVIN123', got %v", entry["vin"])
	}
}

func TestLogEVData(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogEVData("VIN456", 92.5, true)
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["vin"] != "VIN456" {
		t.Errorf("Expected vin 'VIN456', got %v", entry["vin"])
	}
	if entry["battery_level"] != 92.5 {
		t.Errorf("Expected battery_level 92.5, got %v", entry["battery_level"])
	}
	if entry["charging"] != true {
		t.Errorf("Expected charging true, got %v", entry["charging"])
	}
}

func TestLogLocation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.LogLocation("VIN789", 51.5074, -0.1278)
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["vin"] != "VIN789" {
		t.Errorf("Expected vin 'VIN789', got %v", entry["vin"])
	}
	if entry["latitude"] != 51.5074 {
		t.Errorf("Expected latitude 51.5074, got %v", entry["latitude"])
	}
	if entry["longitude"] != -0.1278 {
		t.Errorf("Expected longitude -0.1278, got %v", entry["longitude"])
	}
}

func TestLogPollComplete(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	duration := 3*time.Second + 500*time.Millisecond
	logger.LogPollComplete(duration)
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check duration_seconds field
	if entry["duration_seconds"] != 3.5 {
		t.Errorf("Expected duration_seconds 3.5, got %v", entry["duration_seconds"])
	}
}

func TestWithContext(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create logger with context
	ctxLogger := logger.WithContext(map[string]interface{}{
		"request_id": "req-123",
		"operation":  "test-op",
	})

	ctxLogger.Info("test message with context")
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["request_id"] != "req-123" {
		t.Errorf("Expected request_id 'req-123', got %v", entry["request_id"])
	}
	if entry["operation"] != "test-op" {
		t.Errorf("Expected operation 'test-op', got %v", entry["operation"])
	}
}

func TestLogError(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	testErr := errors.New("test error occurred")
	logger.LogError("database_write", testErr)
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["operation"] != "database_write" {
		t.Errorf("Expected operation 'database_write', got %v", entry["operation"])
	}
	if entry["level"] != "error" {
		t.Errorf("Expected level 'error', got %v", entry["level"])
	}
}

func TestLogFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "nested", "log", "directory")
	logFile := filepath.Join(logDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger with nested directory: %v", err)
	}

	logger.Info("test message")
	logger.Close()

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("test message")

	// First close should not return error
	if err := logger.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestDefaultLogLevel(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Create logger without specifying log level
	logger, err := New(Config{
		LogFilePath: logFile,
		LogToFile:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Default level should be info, so debug should be filtered
	if strings.Contains(logContent, "debug message") {
		t.Error("Debug message should be filtered at default (info) level")
	}
	if !strings.Contains(logContent, "info message") {
		t.Error("Info message should appear at default (info) level")
	}
}
