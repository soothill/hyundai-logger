// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
	"github.com/soothill/hyundai-logger/internal/database"
	"github.com/soothill/hyundai-logger/internal/retry"
	"github.com/soothill/hyundai-logger/internal/webhook"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

type ValidationResult struct {
	Component string
	Status    string // "pass", "warn", "fail"
	Message   string
	Duration  time.Duration
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	// Disable colors if requested or not a TTY
	if *noColor || !isTerminal() {
		disableColors()
	}

	fmt.Printf("%s╔════════════════════════════════════════════════════════════════╗%s\n", colorBlue, colorReset)
	fmt.Printf("%s║     Hyundai Logger Configuration Validation Tool              ║%s\n", colorBlue, colorReset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════════╝%s\n\n", colorBlue, colorReset)

	fmt.Printf("Configuration file: %s\n\n", *configPath)

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		printResult(ValidationResult{
			Component: "Configuration File",
			Status:    "fail",
			Message:   err.Error(),
		})
		os.Exit(1)
	}

	printResult(ValidationResult{
		Component: "Configuration File",
		Status:    "pass",
		Message:   "Successfully loaded",
	})

	// Run all validations
	results := []ValidationResult{}

	// Validate configuration structure
	results = append(results, validateConfigStructure(cfg))

	// Validate Hyundai API
	results = append(results, validateHyundaiAPI(cfg, *verbose))

	// Validate InfluxDB
	results = append(results, validateInfluxDB(cfg, *verbose))

	// Validate webhooks
	if cfg.Webhooks.Enabled {
		results = append(results, validateWebhooks(cfg, *verbose)...)
	} else {
		results = append(results, ValidationResult{
			Component: "Webhooks",
			Status:    "warn",
			Message:   "Disabled in configuration",
		})
	}

	// Print summary
	printSummary(results)

	// Exit with appropriate code
	if hasFailures(results) {
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}

func validateConfigStructure(cfg *config.Config) ValidationResult {
	start := time.Now()

	// Check required fields
	if cfg.Hyundai.Username == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Hyundai username is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Hyundai.Password == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Hyundai password is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Hyundai.PIN == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Hyundai PIN is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Database.URL == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Database URL is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Database.Token == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Database token is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Database.Organization == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Database organization is required",
			Duration:  time.Since(start),
		}
	}

	if cfg.Database.Bucket == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Database bucket is required",
			Duration:  time.Since(start),
		}
	}

	// Validate intervals
	if cfg.RateLimit.PollIntervalMinutes <= 0 {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Poll interval must be greater than 0",
			Duration:  time.Since(start),
		}
	}

	if cfg.RateLimit.RequestsPerHour <= 0 {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Requests per hour must be greater than 0",
			Duration:  time.Since(start),
		}
	}

	return ValidationResult{
		Component: "Config Structure",
		Status:    "pass",
		Message:   "All required fields present and valid",
		Duration:  time.Since(start),
	}
}

func validateHyundaiAPI(cfg *config.Config, verbose bool) ValidationResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("  Testing Hyundai API connection...\n")
	}

	retryConfig := retry.Config{
		MaxAttempts:       1, // No retries for validation
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	client := api.NewClient(
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Region,
		cfg.RateLimit.RequestsPerHour,
		retryConfig,
	)

	// Try to authenticate
	if err := client.Authenticate(ctx); err != nil {
		return ValidationResult{
			Component: "Hyundai API - Authentication",
			Status:    "fail",
			Message:   fmt.Sprintf("Login failed: %v", err),
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Authentication successful\n")
		fmt.Printf("  Fetching vehicle list...\n")
	}

	// Try to fetch vehicles
	vehicles, err := client.GetVehicles(ctx)
	if err != nil {
		return ValidationResult{
			Component: "Hyundai API - Vehicles",
			Status:    "fail",
			Message:   fmt.Sprintf("Failed to fetch vehicles: %v", err),
			Duration:  time.Since(start),
		}
	}

	if len(vehicles) == 0 {
		return ValidationResult{
			Component: "Hyundai API",
			Status:    "warn",
			Message:   "No vehicles found in account",
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Found %d vehicle(s)\n", len(vehicles))
	}

	return ValidationResult{
		Component: "Hyundai API",
		Status:    "pass",
		Message:   fmt.Sprintf("Connected successfully, found %d vehicle(s)", len(vehicles)),
		Duration:  time.Since(start),
	}
}

func validateInfluxDB(cfg *config.Config, verbose bool) ValidationResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("  Testing InfluxDB connection...\n")
	}

	db, err := database.New(
		ctx,
		cfg.Database.URL,
		cfg.Database.Token,
		cfg.Database.Organization,
		cfg.Database.Bucket,
	)
	if err != nil {
		return ValidationResult{
			Component: "InfluxDB - Connection",
			Status:    "fail",
			Message:   fmt.Sprintf("Failed to create client: %v", err),
			Duration:  time.Since(start),
		}
	}
	defer db.Close()

	if verbose {
		fmt.Printf("  ✓ Client created\n")
		fmt.Printf("  Checking health...\n")
	}

	// Check health
	if err := db.HealthCheck(ctx); err != nil {
		return ValidationResult{
			Component: "InfluxDB - Health",
			Status:    "fail",
			Message:   fmt.Sprintf("Health check failed: %v", err),
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Health check passed\n")
	}

	return ValidationResult{
		Component: "InfluxDB",
		Status:    "pass",
		Message:   "Connected successfully and healthy",
		Duration:  time.Since(start),
	}
}

func validateWebhooks(cfg *config.Config, verbose bool) []ValidationResult {
	results := []ValidationResult{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate Slack
	if cfg.Webhooks.Slack.Enabled {
		start := time.Now()
		if verbose {
			fmt.Printf("  Testing Slack webhook...\n")
		}

		if cfg.Webhooks.Slack.WebhookURL == "" {
			results = append(results, ValidationResult{
				Component: "Webhook - Slack",
				Status:    "fail",
				Message:   "Webhook URL is required",
				Duration:  time.Since(start),
			})
		} else {
			webhookConfig := webhook.Config{
				Enabled: true,
				Slack: webhook.SlackConfig{
					Enabled:    cfg.Webhooks.Slack.Enabled,
					WebhookURL: cfg.Webhooks.Slack.WebhookURL,
					Channel:    cfg.Webhooks.Slack.Channel,
					Username:   cfg.Webhooks.Slack.Username,
					IconEmoji:  cfg.Webhooks.Slack.IconEmoji,
				},
			}

			notifier := webhook.New(webhookConfig)
			if err := notifier.TestConnection(ctx); err != nil {
				results = append(results, ValidationResult{
					Component: "Webhook - Slack",
					Status:    "fail",
					Message:   fmt.Sprintf("Test failed: %v", err),
					Duration:  time.Since(start),
				})
			} else {
				results = append(results, ValidationResult{
					Component: "Webhook - Slack",
					Status:    "pass",
					Message:   "Test notification sent successfully",
					Duration:  time.Since(start),
				})
			}
		}
	}

	// Validate Discord
	if cfg.Webhooks.Discord.Enabled {
		start := time.Now()
		if verbose {
			fmt.Printf("  Testing Discord webhook...\n")
		}

		if cfg.Webhooks.Discord.WebhookURL == "" {
			results = append(results, ValidationResult{
				Component: "Webhook - Discord",
				Status:    "fail",
				Message:   "Webhook URL is required",
				Duration:  time.Since(start),
			})
		} else {
			webhookConfig := webhook.Config{
				Enabled: true,
				Discord: webhook.DiscordConfig{
					Enabled:    cfg.Webhooks.Discord.Enabled,
					WebhookURL: cfg.Webhooks.Discord.WebhookURL,
					Username:   cfg.Webhooks.Discord.Username,
					AvatarURL:  cfg.Webhooks.Discord.AvatarURL,
				},
			}

			notifier := webhook.New(webhookConfig)
			if err := notifier.TestConnection(ctx); err != nil {
				results = append(results, ValidationResult{
					Component: "Webhook - Discord",
					Status:    "fail",
					Message:   fmt.Sprintf("Test failed: %v", err),
					Duration:  time.Since(start),
				})
			} else {
				results = append(results, ValidationResult{
					Component: "Webhook - Discord",
					Status:    "pass",
					Message:   "Test notification sent successfully",
					Duration:  time.Since(start),
				})
			}
		}
	}

	// Validate Generic webhooks
	for i, generic := range cfg.Webhooks.Generic {
		start := time.Now()
		name := generic.Name
		if name == "" {
			name = fmt.Sprintf("Generic #%d", i+1)
		}

		if verbose {
			fmt.Printf("  Testing %s webhook...\n", name)
		}

		if generic.URL == "" {
			results = append(results, ValidationResult{
				Component: fmt.Sprintf("Webhook - %s", name),
				Status:    "fail",
				Message:   "Webhook URL is required",
				Duration:  time.Since(start),
			})
			continue
		}

		webhookConfig := webhook.Config{
			Enabled: true,
			Generic: []webhook.GenericConfig{
				{
					Name:    generic.Name,
					URL:     generic.URL,
					Method:  generic.Method,
					Headers: generic.Headers,
				},
			},
		}

		notifier := webhook.New(webhookConfig)
		if err := notifier.TestConnection(ctx); err != nil {
			results = append(results, ValidationResult{
				Component: fmt.Sprintf("Webhook - %s", name),
				Status:    "fail",
				Message:   fmt.Sprintf("Test failed: %v", err),
				Duration:  time.Since(start),
			})
		} else {
			results = append(results, ValidationResult{
				Component: fmt.Sprintf("Webhook - %s", name),
				Status:    "pass",
				Message:   "Test notification sent successfully",
				Duration:  time.Since(start),
			})
		}
	}

	return results
}

func printResult(result ValidationResult) {
	var statusIcon, statusColor string

	switch result.Status {
	case "pass":
		statusIcon = "✓"
		statusColor = colorGreen
	case "warn":
		statusIcon = "⚠"
		statusColor = colorYellow
	case "fail":
		statusIcon = "✗"
		statusColor = colorRed
	}

	durationStr := ""
	if result.Duration > 0 {
		durationStr = fmt.Sprintf(" (%s)", result.Duration.Round(time.Millisecond))
	}

	fmt.Printf("%s%s%s %-40s %s%s\n",
		statusColor, statusIcon, colorReset,
		result.Component+":",
		result.Message,
		durationStr,
	)
}

func printSummary(results []ValidationResult) {
	fmt.Printf("\n%s╔════════════════════════════════════════════════════════════════╗%s\n", colorBlue, colorReset)
	fmt.Printf("%s║                       Validation Summary                       ║%s\n", colorBlue, colorReset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════════╝%s\n\n", colorBlue, colorReset)

	passed := 0
	warned := 0
	failed := 0

	for _, result := range results {
		printResult(result)

		switch result.Status {
		case "pass":
			passed++
		case "warn":
			warned++
		case "fail":
			failed++
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Total: %d  ", len(results))
	fmt.Printf("%sPassed: %d%s  ", colorGreen, passed, colorReset)

	if warned > 0 {
		fmt.Printf("%sWarnings: %d%s  ", colorYellow, warned, colorReset)
	}

	if failed > 0 {
		fmt.Printf("%sFailed: %d%s", colorRed, failed, colorReset)
	}

	fmt.Printf("\n\n")

	if failed > 0 {
		fmt.Printf("%s❌ Configuration validation FAILED%s\n", colorRed, colorReset)
		fmt.Printf("Please fix the errors above before running hyundai-logger.\n")
	} else if warned > 0 {
		fmt.Printf("%s⚠️  Configuration validation passed with warnings%s\n", colorYellow, colorReset)
		fmt.Printf("Review the warnings above - the application may not work as expected.\n")
	} else {
		fmt.Printf("%s✅ Configuration validation PASSED%s\n", colorGreen, colorReset)
		fmt.Printf("Your configuration is ready to use!\n")
	}
}

func hasFailures(results []ValidationResult) bool {
	for _, result := range results {
		if result.Status == "fail" {
			return true
		}
	}
	return false
}

func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func disableColors() {
	*(*string)(nil) = ""
}
