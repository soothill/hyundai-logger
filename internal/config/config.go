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
	Webhooks WebhooksConfig  `yaml:"webhooks"`
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

	// Pre-computed lookup table for O(1) interval lookups [0-23 hours]
	intervalByHour      [24]int
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
	URL          string `yaml:"url"`
	Token        string `yaml:"token"`
	Organization string `yaml:"organization"`
	Bucket       string `yaml:"bucket"`
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

// WebhooksConfig defines webhook notification configuration
type WebhooksConfig struct {
	Enabled bool               `yaml:"enabled"`
	Slack   SlackWebhookConfig `yaml:"slack"`
	Discord DiscordWebhookConfig `yaml:"discord"`
	Generic []GenericWebhookConfig `yaml:"generic"`
}

// SlackWebhookConfig defines Slack webhook configuration
type SlackWebhookConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
	IconEmoji  string `yaml:"icon_emoji"`
}

// DiscordWebhookConfig defines Discord webhook configuration
type DiscordWebhookConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Username   string `yaml:"username"`
	AvatarURL  string `yaml:"avatar_url"`
}

// GenericWebhookConfig defines generic webhook configuration
type GenericWebhookConfig struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
}

// setEnvString sets a string value from an environment variable if present
func setEnvString(envKey string, target *string) {
	if val := os.Getenv(envKey); val != "" {
		*target = val
	}
}

// setEnvInt sets an integer value from an environment variable if present
func setEnvInt(envKey string, target *int) {
	if val := os.Getenv(envKey); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			*target = intVal
		}
	}
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
	setEnvString("HYUNDAI_USERNAME", &cfg.Hyundai.Username)
	setEnvString("HYUNDAI_PASSWORD", &cfg.Hyundai.Password)
	setEnvString("HYUNDAI_PIN", &cfg.Hyundai.PIN)
	setEnvString("HYUNDAI_BRAND", &cfg.Hyundai.Brand)
	setEnvString("HYUNDAI_REGION", &cfg.Hyundai.Region)

	setEnvString("INFLUXDB_URL", &cfg.Database.URL)
	setEnvString("INFLUXDB_TOKEN", &cfg.Database.Token)
	setEnvString("INFLUXDB_ORG", &cfg.Database.Organization)
	setEnvString("INFLUXDB_BUCKET", &cfg.Database.Bucket)

	setEnvInt("POLL_INTERVAL_MINUTES", &cfg.RateLimit.PollIntervalMinutes)
	setEnvInt("REQUESTS_PER_HOUR", &cfg.RateLimit.RequestsPerHour)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Pre-compute interval lookup table for O(1) access
	cfg.RateLimit.precomputeIntervals()

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

	if c.Database.URL == "" {
		return fmt.Errorf("database url is required")
	}
	if c.Database.Token == "" {
		return fmt.Errorf("database token is required")
	}
	if c.Database.Organization == "" {
		return fmt.Errorf("database organization is required")
	}
	if c.Database.Bucket == "" {
		return fmt.Errorf("database bucket is required")
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

// precomputeIntervals pre-computes interval lookup table for O(1) access
func (r *RateLimitConfig) precomputeIntervals() {
	// Initialize all hours with default interval
	for i := 0; i < 24; i++ {
		r.intervalByHour[i] = r.PollIntervalMinutes
	}

	// If schedule is not enabled, we're done
	if !r.Schedule.Enabled {
		return
	}

	// Apply each period to the lookup table
	for _, period := range r.Schedule.Periods {
		if period.StartHour <= period.EndHour {
			// Normal period (e.g., 6:00 to 22:00)
			for hour := period.StartHour; hour < period.EndHour; hour++ {
				r.intervalByHour[hour] = period.IntervalMinutes
			}
		} else {
			// Period spans midnight (e.g., 22:00 to 6:00)
			for hour := period.StartHour; hour < 24; hour++ {
				r.intervalByHour[hour] = period.IntervalMinutes
			}
			for hour := 0; hour < period.EndHour; hour++ {
				r.intervalByHour[hour] = period.IntervalMinutes
			}
		}
	}
}

// GetCurrentInterval returns the appropriate poll interval for the current time
// Uses pre-computed lookup table for O(1) performance
func (r *RateLimitConfig) GetCurrentInterval(currentHour int) int {
	if currentHour < 0 || currentHour > 23 {
		return r.PollIntervalMinutes
	}
	return r.intervalByHour[currentHour]
}

