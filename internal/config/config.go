// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Hyundai  HyundaiConfig   `yaml:"hyundai"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Database DatabaseConfig  `yaml:"database"`
	Logging  LoggingConfig   `yaml:"logging"`
	Retry    RetryConfig     `yaml:"retry"`
	Alerts   AlertsConfig    `yaml:"alerts"`
}

type HyundaiConfig struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	PIN       string `yaml:"pin"`
	Brand     string `yaml:"brand"`
	Region    string `yaml:"region"`
	UseStamps bool   `yaml:"use_stamps"`
	StampURL  string `yaml:"stamp_url"`
}

type RateLimitConfig struct {
	RequestsPerHour     int              `yaml:"requests_per_hour"`
	PollIntervalMinutes int              `yaml:"poll_interval_minutes"`
	Schedule            ScheduleConfig   `yaml:"schedule"`
	ChargingConfig      ChargingConfig   `yaml:"charging"`
}

// ScheduleConfig defines time-based polling schedules
type ScheduleConfig struct {
	Enabled     bool            `yaml:"enabled"`
	Periods     []PeriodConfig  `yaml:"periods"`
}

// PeriodConfig defines a time period with specific polling interval
type PeriodConfig struct {
	Name            string `yaml:"name"`
	StartHour       int    `yaml:"start_hour"`    // 0-23
	EndHour         int    `yaml:"end_hour"`      // 0-23
	IntervalMinutes int    `yaml:"interval_minutes"`
}

// ChargingConfig defines polling behavior when vehicle is charging
type ChargingConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalMinutes int  `yaml:"interval_minutes"`
}

// IsEnabled returns whether charging detection is enabled
func (c *ChargingConfig) IsEnabled() bool {
	return c.Enabled
}

// GetIntervalMinutes returns the polling interval when charging
func (c *ChargingConfig) GetIntervalMinutes() int {
	return c.IntervalMinutes
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// RetryConfig defines retry behavior for API calls
type RetryConfig struct {
	MaxAttempts       int     `yaml:"max_attempts"`
	InitialDelayMs    int     `yaml:"initial_delay_ms"`
	MaxDelayMs        int     `yaml:"max_delay_ms"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`
}

// AlertsConfig defines email alerting configuration
type AlertsConfig struct {
	Enabled           bool   `yaml:"enabled"`
	SMTPHost          string `yaml:"smtp_host"`
	SMTPPort          int    `yaml:"smtp_port"`
	SMTPUsername      string `yaml:"smtp_username"`
	SMTPPassword      string `yaml:"smtp_password"`
	FromEmail         string `yaml:"from_email"`
	ToEmail           string `yaml:"to_email"`
	AlertThreshold    int    `yaml:"alert_threshold"`    // Number of consecutive failures before alerting
	AlertCooldownMins int    `yaml:"alert_cooldown_mins"` // Minutes to wait before sending another alert
}

// Load reads configuration from YAML file and environment variables
// Environment variables take precedence over YAML config
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Read YAML config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Override with environment variables if present
	if val := os.Getenv("HYUNDAI_USERNAME"); val != "" {
		cfg.Hyundai.Username = val
	}
	if val := os.Getenv("HYUNDAI_PASSWORD"); val != "" {
		cfg.Hyundai.Password = val
	}
	if val := os.Getenv("HYUNDAI_PIN"); val != "" {
		cfg.Hyundai.PIN = val
	}
	if val := os.Getenv("HYUNDAI_BRAND"); val != "" {
		cfg.Hyundai.Brand = val
	}
	if val := os.Getenv("HYUNDAI_REGION"); val != "" {
		cfg.Hyundai.Region = val
	}

	if val := os.Getenv("DB_HOST"); val != "" {
		cfg.Database.Host = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			cfg.Database.Port = port
		}
	}
	if val := os.Getenv("DB_USER"); val != "" {
		cfg.Database.User = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		cfg.Database.DBName = val
	}
	if val := os.Getenv("DB_SSLMODE"); val != "" {
		cfg.Database.SSLMode = val
	}

	if val := os.Getenv("POLL_INTERVAL_MINUTES"); val != "" {
		if interval, err := strconv.Atoi(val); err == nil {
			cfg.RateLimit.PollIntervalMinutes = interval
		}
	}
	if val := os.Getenv("REQUESTS_PER_HOUR"); val != "" {
		if req, err := strconv.Atoi(val); err == nil {
			cfg.RateLimit.RequestsPerHour = req
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Hyundai.Username == "" {
		return fmt.Errorf("hyundai username is required")
	}
	if c.Hyundai.Password == "" {
		return fmt.Errorf("hyundai password is required")
	}
	if c.Hyundai.PIN == "" {
		return fmt.Errorf("hyundai PIN is required")
	}
	if c.Hyundai.Brand == "" {
		return fmt.Errorf("hyundai brand is required")
	}
	if c.Hyundai.Region == "" {
		return fmt.Errorf("hyundai region is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if c.RateLimit.PollIntervalMinutes <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}
	if c.RateLimit.RequestsPerHour <= 0 {
		return fmt.Errorf("requests per hour must be greater than 0")
	}

	// Validate schedule configuration if enabled
	if c.RateLimit.Schedule.Enabled {
		if len(c.RateLimit.Schedule.Periods) == 0 {
			return fmt.Errorf("schedule is enabled but no periods are defined")
		}
		for i, period := range c.RateLimit.Schedule.Periods {
			if period.StartHour < 0 || period.StartHour > 23 {
				return fmt.Errorf("period %d: start_hour must be between 0 and 23", i)
			}
			if period.EndHour < 0 || period.EndHour > 23 {
				return fmt.Errorf("period %d: end_hour must be between 0 and 23", i)
			}
			if period.IntervalMinutes <= 0 {
				return fmt.Errorf("period %d: interval_minutes must be greater than 0", i)
			}
		}
	}

	// Validate charging configuration if enabled
	if c.RateLimit.ChargingConfig.Enabled {
		if c.RateLimit.ChargingConfig.IntervalMinutes <= 0 {
			return fmt.Errorf("charging interval_minutes must be greater than 0")
		}
	}

	// Validate retry configuration
	if c.Retry.MaxAttempts < 1 {
		c.Retry.MaxAttempts = 3 // Default
	}
	if c.Retry.InitialDelayMs < 0 {
		c.Retry.InitialDelayMs = 1000 // Default
	}
	if c.Retry.MaxDelayMs < c.Retry.InitialDelayMs {
		c.Retry.MaxDelayMs = 60000 // Default
	}
	if c.Retry.BackoffMultiplier <= 1.0 {
		c.Retry.BackoffMultiplier = 2.0 // Default
	}

	// Validate alerts configuration if enabled
	if c.Alerts.Enabled {
		if c.Alerts.SMTPHost == "" {
			return fmt.Errorf("alerts enabled but smtp_host is not set")
		}
		if c.Alerts.SMTPPort <= 0 {
			return fmt.Errorf("alerts enabled but smtp_port is not set")
		}
		if c.Alerts.FromEmail == "" {
			return fmt.Errorf("alerts enabled but from_email is not set")
		}
		if c.Alerts.ToEmail == "" {
			return fmt.Errorf("alerts enabled but to_email is not set")
		}
		if c.Alerts.AlertThreshold < 1 {
			c.Alerts.AlertThreshold = 5 // Default
		}
		if c.Alerts.AlertCooldownMins < 1 {
			c.Alerts.AlertCooldownMins = 60 // Default
		}
	}

	return nil
}

// GetCurrentInterval returns the appropriate poll interval for the current time
func (r *RateLimitConfig) GetCurrentInterval(currentHour int) int {
	// If schedule is not enabled, return default interval
	if !r.Schedule.Enabled {
		return r.PollIntervalMinutes
	}

	// Find the matching period for current hour
	for _, period := range r.Schedule.Periods {
		if r.isHourInPeriod(currentHour, period.StartHour, period.EndHour) {
			return period.IntervalMinutes
		}
	}

	// If no matching period found, return default interval
	return r.PollIntervalMinutes
}

// isHourInPeriod checks if an hour falls within a period
// Handles periods that span midnight (e.g., 22:00 to 6:00)
func (r *RateLimitConfig) isHourInPeriod(hour, start, end int) bool {
	if start <= end {
		// Normal period (e.g., 6:00 to 22:00)
		return hour >= start && hour < end
	}
	// Period spans midnight (e.g., 22:00 to 6:00)
	return hour >= start || hour < end
}

// GetConnectionString returns PostgreSQL connection string
func (c *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
