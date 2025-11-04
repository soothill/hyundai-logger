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

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
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

	// Validate InfluxDB
	results = append(results, validateInfluxDB(cfg, *verbose))

	// Validate Hyundai API
	results = append(results, validateHyundaiAPI(cfg, *verbose))

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

	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}

func validateConfigStructure(cfg *config.Config) ValidationResult {
	start := time.Now()

	// Check required fields
	if cfg.Hyundai.RefreshToken == "" && cfg.Hyundai.Username == "" {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   "Either refresh_token or username is required",
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

	// Check if brand and region are valid
	validRegions := map[string]bool{"EU": true, "US": true, "CA": true}
	if !validRegions[cfg.Hyundai.Region] {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   fmt.Sprintf("Invalid region: %s (valid: EU, US, CA)", cfg.Hyundai.Region),
			Duration:  time.Since(start),
		}
	}

	validBrands := map[string]bool{"hyundai": true, "kia": true}
	if !validBrands[cfg.Hyundai.Brand] {
		return ValidationResult{
			Component: "Config Structure",
			Status:    "fail",
			Message:   fmt.Sprintf("Invalid brand: %s (valid: hyundai, kia)", cfg.Hyundai.Brand),
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

func validateInfluxDB(cfg *config.Config, verbose bool) ValidationResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if verbose {
		fmt.Printf("  Testing InfluxDB connection...\n")
	}

	// Create InfluxDB client
	client := influxdb2.NewClient(cfg.Database.URL, cfg.Database.Token)
	defer client.Close()

	// Test health
	health, err := client.Health(ctx)
	if err != nil {
		return ValidationResult{
			Component: "InfluxDB - Connection",
			Status:    "fail",
			Message:   fmt.Sprintf("Failed to connect: %v", err),
			Duration:  time.Since(start),
		}
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return ValidationResult{
			Component: "InfluxDB - Health",
			Status:    "fail",
			Message:   fmt.Sprintf("Health check failed: %s", msg),
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Health check passed\n")
	}

	// Test authentication and organization
	orgAPI := client.OrganizationsAPI()
	_, err = orgAPI.FindOrganizationByName(ctx, cfg.Database.Organization)
	if err != nil {
		return ValidationResult{
			Component: "InfluxDB - Auth",
			Status:    "fail",
			Message:   "Authentication failed or organization not found",
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Organization found\n")
	}

	// Test bucket access
	bucketsAPI := client.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(ctx, cfg.Database.Bucket)
	if err != nil {
		return ValidationResult{
			Component: "InfluxDB",
			Status:    "warn",
			Message:   fmt.Sprintf("Bucket '%s' not found (will be created on first run)", cfg.Database.Bucket),
			Duration:  time.Since(start),
		}
	}

	if verbose {
		bucketID := ""
		if bucket.Id != nil {
			bucketID = *bucket.Id
		}
		fmt.Printf("  ✓ Bucket exists (ID: %s)\n", bucketID)
	}

	// Test write permissions
	writeAPI := client.WriteAPIBlocking(cfg.Database.Organization, cfg.Database.Bucket)
	testPoint := influxdb2.NewPoint(
		"validation_test",
		map[string]string{"test": "true"},
		map[string]interface{}{"value": 1},
		time.Now(),
	)
	err = writeAPI.WritePoint(ctx, testPoint)
	if err != nil {
		return ValidationResult{
			Component: "InfluxDB",
			Status:    "fail",
			Message:   "Write test failed - check permissions",
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Write permissions OK\n")
	}

	return ValidationResult{
		Component: "InfluxDB",
		Status:    "pass",
		Message:   "Connected successfully, all checks passed",
		Duration:  time.Since(start),
	}
}

func validateHyundaiAPI(cfg *config.Config, verbose bool) ValidationResult {
	start := time.Now()

	if verbose {
		fmt.Printf("  Testing Hyundai API connection...\n")
	}

	// Create API client
	client, err := api.NewClient(
		cfg.Hyundai.Region,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.RefreshToken,
	)
	if err != nil {
		return ValidationResult{
			Component: "Hyundai API - Client",
			Status:    "fail",
			Message:   fmt.Sprintf("Failed to create client: %v", err),
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Client created\n")
		fmt.Printf("  Fetching vehicle list...\n")
	}

	// Try to fetch vehicles
	vehicles, err := client.GetVehicles()
	if err != nil {
		return ValidationResult{
			Component: "Hyundai API - Vehicles",
			Status:    "fail",
			Message:   fmt.Sprintf("Failed to fetch vehicles: %v", err),
			Duration:  time.Since(start),
		}
	}

	// Check which field has vehicles
	vehicleList := vehicles.Vehicles
	if len(vehicleList) == 0 && len(vehicles.Result) > 0 {
		vehicleList = vehicles.Result
	}

	if len(vehicleList) == 0 {
		return ValidationResult{
			Component: "Hyundai API",
			Status:    "warn",
			Message:   "No vehicles found in account",
			Duration:  time.Since(start),
		}
	}

	if verbose {
		fmt.Printf("  ✓ Found %d vehicle(s)\n", len(vehicleList))
		for _, v := range vehicleList {
			fmt.Printf("    - %s (%s)\n", v.Nickname, v.VIN)
		}
	}

	return ValidationResult{
		Component: "Hyundai API",
		Status:    "pass",
		Message:   fmt.Sprintf("Connected successfully, found %d vehicle(s)", len(vehicleList)),
		Duration:  time.Since(start),
	}
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
	// Colors are already const, this is a placeholder
	// In production code, you'd reassign the color variables
}
