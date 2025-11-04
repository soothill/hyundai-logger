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

logging:
  level: info
  file: /tmp/test.log

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
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
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

	// Set environment variables
	os.Setenv("HYUNDAI_USERNAME", "envuser")
	os.Setenv("HYUNDAI_PASSWORD", "envpass")
	os.Setenv("INFLUXDB_URL", "http://env:8086")
	os.Setenv("POLL_INTERVAL_MINUTES", "10")
	os.Setenv("REQUESTS_PER_HOUR", "200")

	defer func() {
		os.Unsetenv("HYUNDAI_USERNAME")
		os.Unsetenv("HYUNDAI_PASSWORD")
		os.Unsetenv("INFLUXDB_URL")
		os.Unsetenv("POLL_INTERVAL_MINUTES")
		os.Unsetenv("REQUESTS_PER_HOUR")
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
	if cfg.RateLimit.PollIntervalMinutes != 10 {
		t.Errorf("Expected poll interval 10 from env, got %d", cfg.RateLimit.PollIntervalMinutes)
	}
	if cfg.RateLimit.RequestsPerHour != 200 {
		t.Errorf("Expected requests per hour 200 from env, got %d", cfg.RateLimit.RequestsPerHour)
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
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "username is required",
		},
		{
			name: "missing password",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
			},
			wantErr: "password is required",
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
			wantErr: "database url is required",
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
			wantErr: "poll interval must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
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
	tests := []struct {
		name    string
		alerts  AlertsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid alerts config",
			alerts: AlertsConfig{
				Enabled:           true,
				SMTPHost:          "smtp.example.com",
				SMTPPort:          587,
				FromEmail:         "from@example.com",
				ToEmail:           "to@example.com",
				AlertThreshold:    5,
				AlertCooldownMins: 60,
			},
			wantErr: false,
		},
		{
			name: "missing smtp host",
			alerts: AlertsConfig{
				Enabled:   true,
				SMTPPort:  587,
				FromEmail: "from@example.com",
				ToEmail:   "to@example.com",
			},
			wantErr: true,
			errMsg:  "smtp_host is not set",
		},
		{
			name: "missing from email",
			alerts: AlertsConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				ToEmail:  "to@example.com",
			},
			wantErr: true,
			errMsg:  "from_email is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Hyundai: HyundaiConfig{
					Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
				},
				Database: DatabaseConfig{
					URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
				},
				RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
				Alerts:    tt.alerts,
			}

			err := cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !containsString(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
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
	cfg := &Config{
		Hyundai: HyundaiConfig{
			Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
		},
		Database: DatabaseConfig{
			URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
		},
		RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
		Alerts: AlertsConfig{
			Enabled:           true,
			SMTPHost:          "smtp.example.com",
			SMTPPort:          587,
			FromEmail:         "from@example.com",
			ToEmail:           "to@example.com",
			AlertThreshold:    0, // Should default to 5
			AlertCooldownMins: 0, // Should default to 60
		},
	}

	err := cfg.validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Check defaults were applied
	if cfg.Alerts.AlertThreshold != 5 {
		t.Errorf("Expected AlertThreshold default 5, got %d", cfg.Alerts.AlertThreshold)
	}
	if cfg.Alerts.AlertCooldownMins != 60 {
		t.Errorf("Expected AlertCooldownMins default 60, got %d", cfg.Alerts.AlertCooldownMins)
	}
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
