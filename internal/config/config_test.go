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

	cfg, err := Load(configPath)
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

func TestLoad_InvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
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

	_, err := Load(configPath)
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

	cfg, err := Load(configPath)
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

	if err := cfg.Validate(); err != nil {
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
			err := tt.config.Validate()
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
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid schedule",
			config: RateLimitConfig{
				PollIntervalMinutes: 5,
				RequestsPerHour:     100,
				Schedule: ScheduleConfig{
					Enabled: true,
					Periods: []PeriodConfig{
						{Name: "daytime", StartHour: 6, EndHour: 22, IntervalMinutes: 5},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "schedule enabled but no periods",
			config: RateLimitConfig{
				PollIntervalMinutes: 5,
				RequestsPerHour:     100,
				Schedule: ScheduleConfig{
					Enabled: true,
					Periods: []PeriodConfig{},
				},
			},
			wantErr: true,
			errMsg:  "no periods are defined",
		},
		{
			name: "invalid start hour",
			config: RateLimitConfig{
				PollIntervalMinutes: 5,
				RequestsPerHour:     100,
				Schedule: ScheduleConfig{
					Enabled: true,
					Periods: []PeriodConfig{
						{Name: "invalid", StartHour: -1, EndHour: 22, IntervalMinutes: 5},
					},
				},
			},
			wantErr: true,
			errMsg:  "start_hour must be between 0 and 23",
		},
		{
			name: "invalid interval minutes",
			config: RateLimitConfig{
				PollIntervalMinutes: 5,
				RequestsPerHour:     100,
				Schedule: ScheduleConfig{
					Enabled: true,
					Periods: []PeriodConfig{
						{Name: "invalid", StartHour: 6, EndHour: 22, IntervalMinutes: 0},
					},
				},
			},
			wantErr: true,
			errMsg:  "interval_minutes must be greater than 0",
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
				RateLimit: tt.config,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !containsString(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidate_ChargingConfig(t *testing.T) {
	cfg := &Config{
		Hyundai: HyundaiConfig{
			Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
		},
		Database: DatabaseConfig{
			URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
		},
		RateLimit: RateLimitConfig{
			PollIntervalMinutes: 5,
			RequestsPerHour:     100,
			ChargingConfig: ChargingConfig{
				Enabled:         true,
				IntervalMinutes: 0, // Invalid
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid charging interval")
	}
	if !containsString(err.Error(), "charging interval_minutes") {
		t.Errorf("Expected error about charging interval, got: %v", err)
	}
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

			err := cfg.Validate()
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
	cfg := &RateLimitConfig{
		PollIntervalMinutes: 10,
		RequestsPerHour:     100,
		Schedule: ScheduleConfig{
			Enabled: false,
		},
	}

	cfg.precomputeIntervals()

	// All hours should have default interval
	for hour := 0; hour < 24; hour++ {
		if cfg.intervalByHour[hour] != 10 {
			t.Errorf("Hour %d: expected interval 10, got %d", hour, cfg.intervalByHour[hour])
		}
	}
}

func TestPrecomputeIntervals_WithSchedule(t *testing.T) {
	cfg := &RateLimitConfig{
		PollIntervalMinutes: 10,
		RequestsPerHour:     100,
		Schedule: ScheduleConfig{
			Enabled: true,
			Periods: []PeriodConfig{
				{Name: "daytime", StartHour: 6, EndHour: 22, IntervalMinutes: 5},
				{Name: "nighttime", StartHour: 22, EndHour: 6, IntervalMinutes: 15},
			},
		},
	}

	cfg.precomputeIntervals()

	// Check daytime hours (6-21)
	for hour := 6; hour < 22; hour++ {
		if cfg.intervalByHour[hour] != 5 {
			t.Errorf("Daytime hour %d: expected interval 5, got %d", hour, cfg.intervalByHour[hour])
		}
	}

	// Check nighttime hours (22-23, 0-5)
	nightHours := []int{22, 23, 0, 1, 2, 3, 4, 5}
	for _, hour := range nightHours {
		if cfg.intervalByHour[hour] != 15 {
			t.Errorf("Nighttime hour %d: expected interval 15, got %d", hour, cfg.intervalByHour[hour])
		}
	}
}

func TestGetCurrentInterval(t *testing.T) {
	cfg := &RateLimitConfig{
		PollIntervalMinutes: 10,
		RequestsPerHour:     100,
		Schedule: ScheduleConfig{
			Enabled: true,
			Periods: []PeriodConfig{
				{Name: "daytime", StartHour: 6, EndHour: 22, IntervalMinutes: 5},
			},
		},
	}

	cfg.precomputeIntervals()

	tests := []struct {
		hour     int
		expected int
	}{
		{hour: 0, expected: 10},  // nighttime (default)
		{hour: 6, expected: 5},   // daytime
		{hour: 12, expected: 5},  // daytime
		{hour: 21, expected: 5},  // daytime
		{hour: 22, expected: 10}, // nighttime (default)
		{hour: -1, expected: 10}, // invalid (returns default)
		{hour: 24, expected: 10}, // invalid (returns default)
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := cfg.GetCurrentInterval(tt.hour)
			if got != tt.expected {
				t.Errorf("GetCurrentInterval(%d) = %d, want %d", tt.hour, got, tt.expected)
			}
		})
	}
}

func TestChargingConfig_Methods(t *testing.T) {
	cfg := &ChargingConfig{
		Enabled:         true,
		IntervalMinutes: 2,
	}

	if !cfg.IsEnabled() {
		t.Error("Expected IsEnabled() to return true")
	}

	if cfg.GetIntervalMinutes() != 2 {
		t.Errorf("Expected GetIntervalMinutes() to return 2, got %d", cfg.GetIntervalMinutes())
	}

	cfg.Enabled = false
	if cfg.IsEnabled() {
		t.Error("Expected IsEnabled() to return false")
	}
}

func TestValidate_RetryConfigDefaults(t *testing.T) {
	cfg := &Config{
		Hyundai: HyundaiConfig{
			Username: "user", Password: "pass", PIN: "1234", Brand: "hyundai", Region: "US",
		},
		Database: DatabaseConfig{
			URL: "http://localhost:8086", Token: "token", Organization: "org", Bucket: "bucket",
		},
		RateLimit: RateLimitConfig{PollIntervalMinutes: 5, RequestsPerHour: 100},
		Retry: RetryConfig{
			MaxAttempts:       0, // Should default to 3
			InitialDelayMs:    -100, // Should default to 1000
			MaxDelayMs:        500, // Should default to 60000 (since < InitialDelayMs)
			BackoffMultiplier: 0.5, // Should default to 2.0
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Check defaults were applied
	if cfg.Retry.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts default 3, got %d", cfg.Retry.MaxAttempts)
	}
	if cfg.Retry.InitialDelayMs != 1000 {
		t.Errorf("Expected InitialDelayMs default 1000, got %d", cfg.Retry.InitialDelayMs)
	}
	if cfg.Retry.MaxDelayMs != 60000 {
		t.Errorf("Expected MaxDelayMs default 60000, got %d", cfg.Retry.MaxDelayMs)
	}
	if cfg.Retry.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier default 2.0, got %f", cfg.Retry.BackoffMultiplier)
	}
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

	err := cfg.Validate()
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
