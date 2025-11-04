package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Hyundai      HyundaiConfig      `yaml:"hyundai"`
	Database     DatabaseConfig     `yaml:"database"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	Alerts       AlertsConfig       `yaml:"alerts"`
	ChargingMode ChargingModeConfig `yaml:"charging_config"`
}

// HyundaiConfig represents Hyundai API configuration
type HyundaiConfig struct {
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	PIN          string `yaml:"pin"`
	Brand        string `yaml:"brand"`         // hyundai or kia
	Region       string `yaml:"region"`        // EU, US, CA
	RefreshToken string `yaml:"refresh_token"` // New: OAuth refresh token
	DeviceID     string `yaml:"device_id"`     // New: Stored device ID
}

// DatabaseConfig represents InfluxDB configuration
type DatabaseConfig struct {
	URL           string `yaml:"url"`
	Token         string `yaml:"token"`
	Organization  string `yaml:"organization"`
	Bucket        string `yaml:"bucket"`
	BatchSize     int    `yaml:"batch_size"`
	FlushInterval int    `yaml:"flush_interval_seconds"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerHour     int                `yaml:"requests_per_hour"`
	PollIntervalMinutes int                `yaml:"poll_interval_minutes"`
	Periods             []TimePeriodConfig `yaml:"periods"`
}

// TimePeriodConfig represents time-based polling configuration
type TimePeriodConfig struct {
	StartHour       int `yaml:"start_hour"`
	EndHour         int `yaml:"end_hour"`
	IntervalMinutes int `yaml:"interval_minutes"`
}

// AlertsConfig represents email alerts configuration
type AlertsConfig struct {
	Enabled           bool   `yaml:"enabled"`
	SMTPHost          string `yaml:"smtp_host"`
	SMTPPort          int    `yaml:"smtp_port"`
	SMTPUsername      string `yaml:"smtp_username"`
	SMTPPassword      string `yaml:"smtp_password"`
	FromEmail         string `yaml:"from_email"`
	ToEmail           string `yaml:"to_email"`
	AlertThreshold    int    `yaml:"alert_threshold"`
	AlertCooldownMins int    `yaml:"alert_cooldown_mins"`
}

// ChargingModeConfig represents EV charging detection configuration
type ChargingModeConfig struct {
	Enabled                   bool    `yaml:"enabled"`
	IntervalMinutes           int     `yaml:"interval_minutes"`
	FastChargeIntervalMinutes int     `yaml:"fast_charge_interval_minutes"`
	TrickleThresholdKW        float64 `yaml:"trickle_threshold_kw"`
	FastChargeThresholdKW     float64 `yaml:"fast_charge_threshold_kw"`
}

// LoadConfig loads configuration from file or environment
func LoadConfig(path string) (*Config, error) {
	config := &Config{
		// Set defaults
		RateLimit: RateLimitConfig{
			RequestsPerHour:     12,
			PollIntervalMinutes: 5,
		},
		Database: DatabaseConfig{
			BatchSize:     100,
			FlushInterval: 10,
		},
		ChargingMode: ChargingModeConfig{
			Enabled:                   true,
			IntervalMinutes:           2,
			FastChargeIntervalMinutes: 1,
			TrickleThresholdKW:        3.0,
			FastChargeThresholdKW:     20.0,
		},
	}

	// Load from file if exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	config.loadFromEnv()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	// Hyundai credentials
	if val := os.Getenv("HYUNDAI_USERNAME"); val != "" {
		c.Hyundai.Username = val
	}
	if val := os.Getenv("HYUNDAI_PASSWORD"); val != "" {
		c.Hyundai.Password = val
	}
	if val := os.Getenv("HYUNDAI_PIN"); val != "" {
		c.Hyundai.PIN = val
	}
	if val := os.Getenv("HYUNDAI_BRAND"); val != "" {
		c.Hyundai.Brand = val
	}
	if val := os.Getenv("HYUNDAI_REGION"); val != "" {
		c.Hyundai.Region = val
	}
	if val := os.Getenv("HYUNDAI_REFRESH_TOKEN"); val != "" {
		c.Hyundai.RefreshToken = val
	}
	if val := os.Getenv("HYUNDAI_DEVICE_ID"); val != "" {
		c.Hyundai.DeviceID = val
	}

	// Database configuration
	if val := os.Getenv("INFLUXDB_URL"); val != "" {
		c.Database.URL = val
	}
	if val := os.Getenv("INFLUXDB_TOKEN"); val != "" {
		c.Database.Token = val
	}
	if val := os.Getenv("INFLUXDB_ORG"); val != "" {
		c.Database.Organization = val
	}
	if val := os.Getenv("INFLUXDB_BUCKET"); val != "" {
		c.Database.Bucket = val
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	// Check for refresh token or username/password
	if c.Hyundai.RefreshToken == "" && c.Hyundai.Username == "" {
		return fmt.Errorf("either refresh_token or username must be provided")
	}

	if c.Hyundai.PIN == "" {
		return fmt.Errorf("PIN is required")
	}

	if c.Hyundai.Brand == "" {
		c.Hyundai.Brand = "hyundai" // Default
	}

	if c.Hyundai.Region == "" {
		c.Hyundai.Region = "EU" // Default
	}

	// Validate region
	validRegions := map[string]bool{"EU": true, "US": true, "CA": true}
	if !validRegions[c.Hyundai.Region] {
		return fmt.Errorf("invalid region: %s (valid: EU, US, CA)", c.Hyundai.Region)
	}

	// Validate brand
	validBrands := map[string]bool{"hyundai": true, "kia": true}
	if !validBrands[c.Hyundai.Brand] {
		return fmt.Errorf("invalid brand: %s (valid: hyundai, kia)", c.Hyundai.Brand)
	}

	// Validate database config
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
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

	// Validate rate limits
	if c.RateLimit.RequestsPerHour > 60 {
		fmt.Printf("WARNING: Request rate of %d/hour may drain your 12V battery!\n", c.RateLimit.RequestsPerHour)
	}

	return nil
}

// GetPollInterval returns the current poll interval based on time
func (c *Config) GetPollInterval() time.Duration {
	now := time.Now()
	hour := now.Hour()

	// Check time-based periods
	for _, period := range c.RateLimit.Periods {
		if isInPeriod(hour, period.StartHour, period.EndHour) {
			return time.Duration(period.IntervalMinutes) * time.Minute
		}
	}

	// Default interval
	return time.Duration(c.RateLimit.PollIntervalMinutes) * time.Minute
}

// isInPeriod checks if current hour is within a time period
func isInPeriod(current, start, end int) bool {
	if start <= end {
		return current >= start && current < end
	}
	// Handle overnight periods (e.g., 22:00 to 06:00)
	return current >= start || current < end
}
