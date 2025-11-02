// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
	"github.com/soothill/hyundai-logger/internal/database"
	"github.com/soothill/hyundai-logger/internal/retry"
)

const banner = `
╔════════════════════════════════════════════════════════════════╗
║           Hyundai Logger - Interactive Mode                    ║
║                                                                ║
║  Type 'help' for available commands, 'exit' to quit          ║
╚════════════════════════════════════════════════════════════════╝
`

type InteractiveCLI struct {
	config    *config.Config
	apiClient *api.Client
	db        *database.DB
	startTime time.Time
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	fmt.Print(banner)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	retryConfig := retry.Config{
		MaxAttempts:       cfg.Retry.MaxAttempts,
		InitialDelayMs:    cfg.Retry.InitialDelayMs,
		MaxDelayMs:        cfg.Retry.MaxDelayMs,
		BackoffMultiplier: cfg.Retry.BackoffMultiplier,
	}

	apiClient := api.NewClient(
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Region,
		cfg.RateLimit.RequestsPerHour,
		retryConfig,
	)

	// Connect to database
	ctx := context.Background()
	db, err := database.New(ctx, cfg.Database.URL, cfg.Database.Token,
		cfg.Database.Organization, cfg.Database.Bucket)
	if err != nil {
		fmt.Printf("Warning: Database connection failed: %v\n", err)
		db = nil
	}

	cli := &InteractiveCLI{
		config:    cfg,
		apiClient: apiClient,
		db:        db,
		startTime: time.Now(),
	}

	// Run interactive loop
	cli.run()
}

func (cli *InteractiveCLI) run() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		command := parts[0]
		args := parts[1:]

		switch command {
		case "help", "?":
			cli.showHelp()
		case "status":
			cli.showStatus()
		case "vehicles", "list":
			cli.listVehicles()
		case "poll":
			cli.pollNow(args)
		case "vehicle":
			cli.showVehicle(args)
		case "metrics":
			cli.showMetrics()
		case "config":
			cli.showConfig()
		case "circuit":
			cli.showCircuitBreaker()
		case "test":
			cli.testConnection(args)
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return
		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", command)
		}
	}
}

func (cli *InteractiveCLI) showHelp() {
	help := `
Available Commands:
  help, ?           Show this help message
  status            Show current system status
  vehicles, list    List all vehicles in account
  poll [VIN]        Poll vehicle(s) now (all if no VIN specified)
  vehicle <VIN>     Show detailed information for a vehicle
  metrics           Show detailed metrics and statistics
  config            Show current configuration
  circuit           Show circuit breaker status
  test <component>  Test component (api, database, all)
  clear, cls        Clear screen
  exit, quit, q     Exit interactive mode

Examples:
  > status
  > poll
  > poll 5NPE24AF1KH123456
  > vehicle 5NPE24AF1KH123456
  > test api
`
	fmt.Println(help)
}

func (cli *InteractiveCLI) showStatus() {
	uptime := time.Since(cli.startTime)

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      System Status                             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("  Uptime:              %s\n", uptime.Round(time.Second))
	fmt.Printf("  Config File:         %s\n", "config.yaml")
	fmt.Printf("  API Region:          %s\n", cli.config.Hyundai.Region)
	fmt.Printf("  API Brand:           %s\n", cli.config.Hyundai.Brand)
	fmt.Printf("  Poll Interval:       %d minutes\n", cli.config.RateLimit.PollIntervalMinutes)
	fmt.Printf("  Requests/Hour Limit: %d\n", cli.config.RateLimit.RequestsPerHour)

	if cli.db != nil {
		fmt.Printf("  Database:            ✓ Connected (%s)\n", cli.config.Database.URL)
	} else {
		fmt.Printf("  Database:            ✗ Not connected\n")
	}

	state, _, _ := cli.apiClient.GetCircuitBreakerStats()
	fmt.Printf("  Circuit Breaker:     %s\n", state)
}

func (cli *InteractiveCLI) listVehicles() {
	fmt.Println("\nFetching vehicles...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Authenticate if needed
	if err := cli.apiClient.Authenticate(ctx); err != nil {
		fmt.Printf("Error: Authentication failed: %v\n", err)
		return
	}

	vehicles, err := cli.apiClient.GetVehicles(ctx)
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	if len(vehicles) == 0 {
		fmt.Println("No vehicles found in account.")
		return
	}

	fmt.Printf("\nFound %d vehicle(s):\n\n", len(vehicles))

	for i, vehicle := range vehicles {
		fmt.Printf("  %d. %s %s %d\n", i+1, vehicle.Make, vehicle.Model, vehicle.Year)
		fmt.Printf("     VIN:        %s\n", vehicle.VIN)
		fmt.Printf("     Vehicle ID: %s\n", vehicle.VehicleID)
		fmt.Println()
	}
}

func (cli *InteractiveCLI) pollNow(args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Authenticate
	fmt.Println("Authenticating...")
	if err := cli.apiClient.Authenticate(ctx); err != nil {
		fmt.Printf("Error: Authentication failed: %v\n", err)
		return
	}

	var targetVIN string
	if len(args) > 0 {
		targetVIN = args[0]
		fmt.Printf("\nPolling vehicle %s...\n", targetVIN)
	} else {
		fmt.Println("\nPolling all vehicles...")
	}

	start := time.Now()

	// Get vehicles
	vehicles, err := cli.apiClient.GetVehicles(ctx)
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	// Filter if VIN specified
	if targetVIN != "" {
		found := false
		for _, v := range vehicles {
			if v.VIN == targetVIN || v.VehicleID == targetVIN {
				vehicles = []api.Vehicle{v}
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Error: Vehicle %s not found\n", targetVIN)
			return
		}
	}

	// Poll each vehicle
	for i, vehicle := range vehicles {
		fmt.Printf("\n[%d/%d] Polling %s %s (%s)...\n",
			i+1, len(vehicles), vehicle.Make, vehicle.Model, vehicle.VIN)

		pollStart := time.Now()

		// Get status
		status, err := cli.apiClient.GetVehicleStatus(ctx, vehicle.VehicleID)
		if err != nil {
			fmt.Printf("  ✗ Status: Failed (%v)\n", err)
		} else {
			fmt.Printf("  ✓ Status: OK (%s)\n", time.Since(pollStart).Round(time.Millisecond))
			fmt.Printf("    - Odometer: %.1f km\n", status.Odometer)
			if status.EV != nil {
				fmt.Printf("    - Battery: %.1f%%\n", status.EV.BatteryLevel)
				if status.EV.Charging {
					fmt.Printf("    - Charging: Yes (%.1f kW)\n", status.EV.ChargingPower)
				} else {
					fmt.Printf("    - Charging: No\n")
				}
			}
		}

		// Get location
		pollStart = time.Now()
		location, err := cli.apiClient.GetVehicleLocation(ctx, vehicle.VehicleID)
		if err != nil {
			fmt.Printf("  ✗ Location: Failed (%v)\n", err)
		} else {
			fmt.Printf("  ✓ Location: OK (%s)\n", time.Since(pollStart).Round(time.Millisecond))
			fmt.Printf("    - Lat/Lon: %.6f, %.6f\n",
				location.Location.Latitude, location.Location.Longitude)
		}
	}

	duration := time.Since(start)
	fmt.Printf("\n✓ Poll complete in %s\n", duration.Round(time.Millisecond))
}

func (cli *InteractiveCLI) showVehicle(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Please specify a VIN")
		fmt.Println("Usage: vehicle <VIN>")
		return
	}

	vin := args[0]
	fmt.Printf("\nFetching detailed information for %s...\n", vin)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Authenticate
	if err := cli.apiClient.Authenticate(ctx); err != nil {
		fmt.Printf("Error: Authentication failed: %v\n", err)
		return
	}

	// Get vehicles to find the ID
	vehicles, err := cli.apiClient.GetVehicles(ctx)
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	var targetVehicle *api.Vehicle
	for _, v := range vehicles {
		if v.VIN == vin || v.VehicleID == vin {
			targetVehicle = &v
			break
		}
	}

	if targetVehicle == nil {
		fmt.Printf("Error: Vehicle %s not found\n", vin)
		return
	}

	// Get status
	status, err := cli.apiClient.GetVehicleStatus(ctx, targetVehicle.VehicleID)
	if err != nil {
		fmt.Printf("Error: Failed to get status: %v\n", err)
		return
	}

	// Display information
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  %s %s %d\n", targetVehicle.Make, targetVehicle.Model, targetVehicle.Year)
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n  VIN:        %s\n", status.VIN)
	fmt.Printf("  Updated:    %s\n", status.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Odometer:   %.1f km\n", status.Odometer)
	fmt.Printf("  Fuel Level: %.1f%%\n", status.FuelLevel)

	if status.Engine.Running {
		fmt.Printf("  Engine:     Running (Range: %.1f km)\n", status.Engine.RangeKM)
	} else {
		fmt.Printf("  Engine:     Off (Range: %.1f km)\n", status.Engine.RangeKM)
	}

	if status.EV != nil {
		fmt.Println("\n  Electric Vehicle Status:")
		fmt.Printf("    Battery:     %.1f%%\n", status.EV.BatteryLevel)
		fmt.Printf("    Range:       %.1f km\n", status.EV.RangeKM)
		if status.EV.Charging {
			fmt.Printf("    Charging:    Yes\n")
			fmt.Printf("    Power:       %.1f kW\n", status.EV.ChargingPower)
		} else {
			fmt.Printf("    Charging:    No\n")
		}
	}
}

func (cli *InteractiveCLI) showMetrics() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         Metrics                                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	state, failures, lastFail := cli.apiClient.GetCircuitBreakerStats()

	fmt.Printf("\n  Circuit Breaker:\n")
	fmt.Printf("    State:          %s\n", state)
	fmt.Printf("    Failures:       %d\n", failures)
	if !lastFail.IsZero() {
		fmt.Printf("    Last Failure:   %s\n", lastFail.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n  System:\n")
	fmt.Printf("    Uptime:         %s\n", time.Since(cli.startTime).Round(time.Second))
	fmt.Printf("    Config Reloads: N/A (requires logger integration)\n")
}

func (cli *InteractiveCLI) showConfig() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Configuration                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n  Hyundai Account:\n")
	fmt.Printf("    Username: %s\n", cli.config.Hyundai.Username)
	fmt.Printf("    Brand:    %s\n", cli.config.Hyundai.Brand)
	fmt.Printf("    Region:   %s\n", cli.config.Hyundai.Region)

	fmt.Printf("\n  Rate Limiting:\n")
	fmt.Printf("    Poll Interval:   %d minutes\n", cli.config.RateLimit.PollIntervalMinutes)
	fmt.Printf("    Requests/Hour:   %d\n", cli.config.RateLimit.RequestsPerHour)
	fmt.Printf("    Schedule:        %v\n", cli.config.RateLimit.Schedule.Enabled)
	fmt.Printf("    Charging Detect: %v\n", cli.config.RateLimit.ChargingConfig.Enabled)

	fmt.Printf("\n  Database:\n")
	fmt.Printf("    URL:      %s\n", cli.config.Database.URL)
	fmt.Printf("    Org:      %s\n", cli.config.Database.Organization)
	fmt.Printf("    Bucket:   %s\n", cli.config.Database.Bucket)

	fmt.Printf("\n  Alerts:\n")
	fmt.Printf("    Email:    %v\n", cli.config.Alerts.Enabled)
	fmt.Printf("    Webhooks: %v\n", cli.config.Webhooks.Enabled)
}

func (cli *InteractiveCLI) showCircuitBreaker() {
	state, failures, lastFail := cli.apiClient.GetCircuitBreakerStats()

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Circuit Breaker Status                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	statusIcon := "✓"
	if state.String() != "closed" {
		statusIcon = "⚠"
	}

	fmt.Printf("\n  %s State:        %s\n", statusIcon, state)
	fmt.Printf("    Failures:     %d\n", failures)

	if !lastFail.IsZero() {
		fmt.Printf("    Last Failure: %s (%s ago)\n",
			lastFail.Format("2006-01-02 15:04:05"),
			time.Since(lastFail).Round(time.Second))
	} else {
		fmt.Printf("    Last Failure: Never\n")
	}

	if state.String() == "open" {
		fmt.Println("\n  ⚠ Circuit is OPEN - API calls are being blocked")
		fmt.Println("    The circuit will attempt to close automatically")
	} else if state.String() == "half-open" {
		fmt.Println("\n  ⚠ Circuit is HALF-OPEN - Testing if service recovered")
	} else {
		fmt.Println("\n  ✓ Circuit is CLOSED - Operating normally")
	}
}

func (cli *InteractiveCLI) testConnection(args []string) {
	if len(args) == 0 {
		args = []string{"all"}
	}

	component := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch component {
	case "api":
		cli.testAPI(ctx)
	case "database", "db":
		cli.testDatabase(ctx)
	case "all":
		cli.testAPI(ctx)
		fmt.Println()
		cli.testDatabase(ctx)
	default:
		fmt.Printf("Unknown component: %s\n", component)
		fmt.Println("Available: api, database, all")
	}
}

func (cli *InteractiveCLI) testAPI(ctx context.Context) {
	fmt.Println("Testing API connection...")

	start := time.Now()

	if err := cli.apiClient.Authenticate(ctx); err != nil {
		fmt.Printf("  ✗ Authentication: FAILED (%v)\n", err)
		return
	}

	authTime := time.Since(start)
	fmt.Printf("  ✓ Authentication: OK (%s)\n", authTime.Round(time.Millisecond))

	start = time.Now()
	vehicles, err := cli.apiClient.GetVehicles(ctx)
	if err != nil {
		fmt.Printf("  ✗ Fetch Vehicles: FAILED (%v)\n", err)
		return
	}

	vehicleTime := time.Since(start)
	fmt.Printf("  ✓ Fetch Vehicles: OK (%d vehicles in %s)\n",
		len(vehicles), vehicleTime.Round(time.Millisecond))
}

func (cli *InteractiveCLI) testDatabase(ctx context.Context) {
	if cli.db == nil {
		fmt.Println("  ✗ Database: Not connected")
		return
	}

	fmt.Println("Testing database connection...")

	start := time.Now()
	if err := cli.db.HealthCheck(ctx); err != nil {
		fmt.Printf("  ✗ Health Check: FAILED (%v)\n", err)
		return
	}

	duration := time.Since(start)
	fmt.Printf("  ✓ Health Check: OK (%s)\n", duration.Round(time.Millisecond))
}
