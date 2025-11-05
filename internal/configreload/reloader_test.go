// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package configreload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/config"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	infoLogs  []string
	errorLogs []string
	debugLogs []string
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.infoLogs = append(m.infoLogs, format)
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.errorLogs = append(m.errorLogs, format)
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.debugLogs = append(m.debugLogs, format)
}

// createTempConfig creates a temporary configuration file for testing
func createTempConfig(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return configPath
}

const testConfigV1 = `
hyundai:
  username: "test@example.com"
  password: "password123"
  pin: "1234"
  brand: "hyundai"
  region: "US"

rate_limit:
  requests_per_hour: 50
  poll_interval_minutes: 5
  schedule:
    enabled: false
  charging:
    enabled: true
    interval_minutes: 2

database:
  url: "http://localhost:8086"
  token: "test-token"
  organization: "test-org"
  bucket: "test-bucket"

logging:
  level: "info"

retry:
  max_attempts: 3
  initial_delay_ms: 1000
  max_delay_ms: 60000
  backoff_multiplier: 2.0

alerts:
  enabled: false

webhooks:
  enabled: false
`

const testConfigV2 = `
hyundai:
  username: "test@example.com"
  password: "password123"
  pin: "1234"
  brand: "hyundai"
  region: "US"

rate_limit:
  requests_per_hour: 60
  poll_interval_minutes: 10
  schedule:
    enabled: true
    periods:
      - name: "Daytime"
        start_hour: 6
        end_hour: 22
        interval_minutes: 5
  charging:
    enabled: true
    interval_minutes: 3

database:
  url: "http://localhost:8086"
  token: "test-token"
  organization: "test-org"
  bucket: "test-bucket"

logging:
  level: "debug"

retry:
  max_attempts: 3
  initial_delay_ms: 1000
  max_delay_ms: 60000
  backoff_multiplier: 2.0

alerts:
  enabled: true
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  from_email: "test@example.com"
  to_email: "alert@example.com"

webhooks:
  enabled: true
`

func TestNew(t *testing.T) {
	// Clear environment variables that might interfere
	oldEnvVars := map[string]string{
		"HYUNDAI_USERNAME": os.Getenv("HYUNDAI_USERNAME"),
		"HYUNDAI_PASSWORD": os.Getenv("HYUNDAI_PASSWORD"),
		"INFLUXDB_URL":     os.Getenv("INFLUXDB_URL"),
		"INFLUXDB_TOKEN":   os.Getenv("INFLUXDB_TOKEN"),
		"INFLUXDB_ORG":     os.Getenv("INFLUXDB_ORG"),
		"INFLUXDB_BUCKET":  os.Getenv("INFLUXDB_BUCKET"),
	}
	defer func() {
		for k, v := range oldEnvVars {
			if v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()

	for k := range oldEnvVars {
		_ = os.Unsetenv(k)
	}

	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		_ = reloader.Close()
	}()

	if reloader == nil {
		t.Fatal("expected reloader, got nil")
	}

	cfg := reloader.GetConfig()
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if cfg.Hyundai.Username != "test@example.com" {
		t.Errorf("unexpected username: %s", cfg.Hyundai.Username)
	}
}

func TestNew_InvalidPath(t *testing.T) {
	logger := &mockLogger{}

	_, err := New(Config{
		ConfigPath: "/nonexistent/config.yaml",
		Logger:     logger,
	})

	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestGetConfig(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	cfg1 := reloader.GetConfig()
	cfg2 := reloader.GetConfig()

	// Should return same config instance
	if cfg1 != cfg2 {
		t.Error("GetConfig should return same config instance")
	}
}

func TestOnReload(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	callCount := 0
	reloader.OnReload(func(old, new *config.Config) error {
		callCount++
		return nil
	})

	if len(reloader.reloadFuncs) != 1 {
		t.Errorf("expected 1 reload func, got %d", len(reloader.reloadFuncs))
	}
}

func TestForceReload(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	oldConfig := reloader.GetConfig()

	// Update config file
	if err := os.WriteFile(configPath, []byte(testConfigV2), 0644); err != nil {
		t.Fatal(err)
	}

	// Force reload
	if err := reloader.ForceReload(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newConfig := reloader.GetConfig()

	// Check that config changed
	if oldConfig.RateLimit.RequestsPerHour == newConfig.RateLimit.RequestsPerHour {
		t.Error("config should have changed after reload")
	}

	if newConfig.RateLimit.RequestsPerHour != 60 {
		t.Errorf("expected requests per hour 60, got %d", newConfig.RateLimit.RequestsPerHour)
	}
}

func TestForceReload_WithHook(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	hookCalled := false
	var capturedOld, capturedNew *config.Config

	reloader.OnReload(func(old, new *config.Config) error {
		hookCalled = true
		capturedOld = old
		capturedNew = new
		return nil
	})

	// Update config file
	if err := os.WriteFile(configPath, []byte(testConfigV2), 0644); err != nil {
		t.Fatal(err)
	}

	// Force reload
	if err := reloader.ForceReload(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hookCalled {
		t.Error("reload hook should have been called")
	}

	if capturedOld.RateLimit.RequestsPerHour != 50 {
		t.Errorf("old config should have 50 req/hr, got %d", capturedOld.RateLimit.RequestsPerHour)
	}

	if capturedNew.RateLimit.RequestsPerHour != 60 {
		t.Errorf("new config should have 60 req/hr, got %d", capturedNew.RateLimit.RequestsPerHour)
	}
}

func TestForceReload_HookError(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	oldConfig := reloader.GetConfig()

	reloader.OnReload(func(old, new *config.Config) error {
		return fmt.Errorf("hook error")
	})

	// Update config file
	if err := os.WriteFile(configPath, []byte(testConfigV2), 0644); err != nil {
		t.Fatal(err)
	}

	// Force reload should fail
	err = reloader.ForceReload()
	if err == nil {
		t.Error("expected error from hook, got nil")
	}

	// Config should not have changed
	currentConfig := reloader.GetConfig()
	if currentConfig != oldConfig {
		t.Error("config should not have changed when hook fails")
	}
}

func TestAutoReload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auto-reload test in short mode")
	}

	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
		Debounce:   100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching in background
	go func() {
		_ = reloader.Start(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(200 * time.Millisecond)

	oldConfig := reloader.GetConfig()

	// Update config file
	if err := os.WriteFile(configPath, []byte(testConfigV2), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for reload (with generous timeout)
	time.Sleep(2 * time.Second)

	newConfig := reloader.GetConfig()

	// Check that config changed
	if oldConfig.RateLimit.RequestsPerHour == newConfig.RateLimit.RequestsPerHour {
		t.Error("config should have changed after file update")
	}

	if newConfig.RateLimit.RequestsPerHour != 60 {
		t.Errorf("expected requests per hour 60, got %d", newConfig.RateLimit.RequestsPerHour)
	}
}

func TestDebounce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping debounce test in short mode")
	}

	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
		Debounce:   1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	reloadCount := 0
	reloader.OnReload(func(old, new *config.Config) error {
		reloadCount++
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = reloader.Start(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	// Trigger multiple writes quickly
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(configPath, []byte(testConfigV2), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for potential reloads
	time.Sleep(2 * time.Second)

	// Should have been debounced to fewer reloads
	if reloadCount > 2 {
		t.Errorf("expected ≤2 reloads due to debouncing, got %d", reloadCount)
	}
}

func TestGetStats(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reloader.Close() }()

	reloader.OnReload(func(old, new *config.Config) error {
		return nil
	})

	stats := reloader.GetStats()

	if stats.CurrentConfigPath != configPath {
		t.Errorf("unexpected config path: %s", stats.CurrentConfigPath)
	}

	if stats.ReloadHooksCount != 1 {
		t.Errorf("expected 1 hook, got %d", stats.ReloadHooksCount)
	}

	if stats.LastReloadTime.IsZero() {
		t.Error("last reload time should not be zero")
	}
}

func TestClose(t *testing.T) {
	configPath := createTempConfig(t, testConfigV1)
	logger := &mockLogger{}

	reloader, err := New(Config{
		ConfigPath: configPath,
		Logger:     logger,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = reloader.Close()
	if err != nil {
		t.Errorf("unexpected error closing reloader: %v", err)
	}

	// Closing again should be safe
	err = reloader.Close()
	if err != nil {
		t.Errorf("unexpected error closing reloader twice: %v", err)
	}
}
