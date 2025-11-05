// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Clear environment variables that might interfere with test
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

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validConfig := `
hyundai:
  username: testuser
  password: testpass
  pin: "1234"
  brand: hyundai
  region: US

database:
  url: http://localhost:8086
  token: testtoken
  organization: testorg
  bucket: testbucket

rate_limit:
  poll_interval_minutes: 5
  requests_per_hour: 100

alerts:
  enabled: false
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded config
	if cfg.Hyundai.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", cfg.Hyundai.Username)
	}
	if cfg.Database.URL != "http://localhost:8086" {
		t.Errorf("Expected database URL 'http://localhost:8086', got %s", cfg.Database.URL)
	}
	if cfg.RateLimit.PollIntervalMinutes != 5 {
		t.Errorf("Expected poll interval 5, got %d", cfg.RateLimit.PollIntervalMinutes)
	}
}

func TestLoadConfig_InvalidPath(t *testing.T) {
	// LoadConfig no longer errors on missing files - it uses defaults + env vars
	// This allows the application to run with only environment variables
	t.Skip("LoadConfig now silently falls back to defaults for missing files")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
invalid: yaml: content:
  missing: proper
    structure
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	baseConfig := `
hyundai:
  username: yamluser
  password: yamlpass
  pin: "1234"
  brand: hyundai
  region: US

database:
  url: http://localhost:8086
  token: yamltoken
  organization: yamlorg
  bucket: yamlbucket

rate_limit:
  poll_interval_minutes: 5
  requests_per_hour: 100
`

	if err := os.WriteFile(configPath, []byte(baseConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variables (only those supported by loadFromEnv)
	_ = os.Setenv("HYUNDAI_USERNAME", "envuser")
	_ = os.Setenv("HYUNDAI_PASSWORD", "envpass")
	_ = os.Setenv("INFLUXDB_URL", "http://env:8086")
	_ = os.Setenv("INFLUXDB_ORG", "envorg")
	_ = os.Setenv("INFLUXDB_BUCKET", "envbucket")

	defer func() {
		_ = os.Unsetenv("HYUNDAI_USERNAME")
		_ = os.Unsetenv("HYUNDAI_PASSWORD")
		_ = os.Unsetenv("INFLUXDB_URL")
		_ = os.Unsetenv("INFLUXDB_ORG")
		_ = os.Unsetenv("INFLUXDB_BUCKET")
	}()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment variables override YAML
	if cfg.Hyundai.Username != "envuser" {
		t.Errorf("Expected username 'envuser' from env, got %s", cfg.Hyundai.Username)
	}
	if cfg.Hyundai.Password != "envpass" {
		t.Errorf("Expected password 'envpass' from env, got %s", cfg.Hyundai.Password)
	}
	if cfg.Database.URL != "http://env:8086" {
		t.Errorf("Expected database URL 'http://env:8086' from env, got %s", cfg.Database.URL)
	}
	if cfg.Database.Organization != "envorg" {
		t.Errorf("Expected org 'envorg' from env, got %s", cfg.Database.Organization)
	}
	if cfg.Database.Bucket != "envbucket" {
		t.Errorf("Expected bucket 'envbucket' from env, got %s", cfg.Database.Bucket)
	}
	// Note: RateLimit env vars not supported by loadFromEnv, so they use YAML values
	if cfg.RateLimit.PollIntervalMinutes != 5 {
		t.Errorf("Expected poll interval 5 from YAML, got %d", cfg.RateLimit.PollIntervalMinutes)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Hyundai: HyundaiConfig{
			Username: "testuser",
			Password: "testpass",
			PIN:      "1234",
			Brand:    "hyundai",
			Region:   "US",
		},
		Database: DatabaseConfig{
			URL:          "http://localhost:8086",
			Token:        "token",
			Organization: "org",
			Bucket:       "bucket",
		},
		RateLimit: RateLimitConfig{
			PollIntervalMinutes: 5,
			RequestsPerHour:     100,
		},
	}

	// validate is now private
	if err := cfg.validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "missing username",
			config: Config{
				Hyundai: HyundaiConfig{
					Password: "pass",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
					// No username AND no refresh_token
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "either refresh_token or username must be provided",
		},
		{
			name: "missing password",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
					// Password not required if refresh_token exists
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "", // No error - password not required
		},
		{
			name: "missing PIN",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user",
					Password: "pass",
					Brand:    "hyundai",
					Region:   "US",
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "PIN is required",
		},
		{
			name: "missing database URL",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
				},
				Database: DatabaseConfig{
					Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "database URL is required",
		},
		{
			name: "invalid poll interval",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{
					PollIntervalMinutes: 0,
					RequestsPerHour:     100,
				},
			},
			wantErr: "", // No error - poll interval validation removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got '%v'", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.wantErr)
				return
			}
			if err.Error() != tt.wantErr && !containsString(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

func TestValidate_ScheduleConfig(t *testing.T) {
	t.Skip("Schedule functionality removed from config")
}

func TestValidate_ChargingConfig(t *testing.T) {
	t.Skip("ChargingConfig functionality removed from config")
}

func TestValidate_AlertsConfig(t *testing.T) {
	t.Skip("Alerts validation removed from config - alerts settings are no longer validated by the config package")
}

func TestPrecomputeIntervals_NoSchedule(t *testing.T) {
	t.Skip("Schedule functionality removed from config")
}

func TestPrecomputeIntervals_WithSchedule(t *testing.T) {
	t.Skip("Schedule functionality removed from config")
}

func TestGetCurrentInterval(t *testing.T) {
	t.Skip("Schedule functionality removed from config")
}

func TestChargingConfig_Methods(t *testing.T) {
	t.Skip("ChargingConfig functionality removed from config")
}

func TestValidate_RetryConfigDefaults(t *testing.T) {
	t.Skip("RetryConfig functionality removed from config")
}

func TestValidate_AlertsConfigDefaults(t *testing.T) {
	t.Skip("Alerts default values no longer set by validate() - defaults should be set where alerts are used")
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
