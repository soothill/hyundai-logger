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
	db        *database.Client
	startTime time.Time
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	fmt.Print(banner)

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	apiClient, err := api.NewClient(
		cfg.Hyundai.Region,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.RefreshToken,
	)
	if err != nil {
		fmt.Printf("Error creating API client: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.NewClient(cfg.Database)
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

	fmt.Printf("  API Client:          ✓ Authenticated\n")
}

func (cli *InteractiveCLI) listVehicles() {
	fmt.Println("\nFetching vehicles...")

	vehiclesResp, err := cli.apiClient.GetVehicles()
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	// Use Vehicles or Result field depending on which is populated
	vehicleList := vehiclesResp.Vehicles
	if len(vehicleList) == 0 {
		vehicleList = vehiclesResp.Result
	}

	if len(vehicleList) == 0 {
		fmt.Println("No vehicles found in account.")
		return
	}

	fmt.Printf("\nFound %d vehicle(s):\n\n", len(vehicleList))

	for i, vehicle := range vehicleList {
		fmt.Printf("  %d. %s %s\n", i+1, vehicle.Nickname, vehicle.VehicleModel)
		fmt.Printf("     VIN:        %s\n", vehicle.VIN)
		fmt.Printf("     Vehicle ID: %s\n", vehicle.VehicleID)
		if vehicle.Year != "" {
			fmt.Printf("     Year:       %s\n", vehicle.Year)
		}
		fmt.Println()
	}
}

func (cli *InteractiveCLI) pollNow(args []string) {
	var targetVIN string
	if len(args) > 0 {
		targetVIN = args[0]
		fmt.Printf("\nPolling vehicle %s...\n", targetVIN)
	} else {
		fmt.Println("\nPolling all vehicles...")
	}

	start := time.Now()

	// Get vehicles
	vehiclesResp, err := cli.apiClient.GetVehicles()
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	// Use Vehicles or Result field depending on which is populated
	vehicleList := vehiclesResp.Vehicles
	if len(vehicleList) == 0 {
		vehicleList = vehiclesResp.Result
	}

	// Filter if VIN specified
	if targetVIN != "" {
		found := false
		for _, v := range vehicleList {
			if v.VIN == targetVIN || v.VehicleID == targetVIN {
				vehicleList = []api.Vehicle{v}
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
	for i, vehicle := range vehicleList {
		fmt.Printf("\n[%d/%d] Polling %s %s (%s)...\n",
			i+1, len(vehicleList), vehicle.Nickname, vehicle.VehicleModel, vehicle.VIN)

		pollStart := time.Now()

		// Get status
		status, err := cli.apiClient.GetVehicleStatus(vehicle.VehicleID)
		if err != nil {
			fmt.Printf("  ✗ Status: Failed (%v)\n", err)
		} else {
			fmt.Printf("  ✓ Status: OK (%s)\n", time.Since(pollStart).Round(time.Millisecond))
			fmt.Printf("    - Odometer: %d %s\n", status.OdometerStatus.Value, status.OdometerStatus.Unit)
			if status.EVStatus != nil {
				fmt.Printf("    - Battery: %d%%\n", status.EVStatus.BatteryLevel)
				if status.EVStatus.BatteryCharge {
					fmt.Printf("    - Charging: Yes (%.1f kW)\n", status.EVStatus.ChargingPower)
				} else {
					fmt.Printf("    - Charging: No\n")
				}
			}
			// Location is included in status
			if status.VehicleLocation.Latitude != 0 || status.VehicleLocation.Longitude != 0 {
				fmt.Printf("    - Location: %.6f, %.6f\n",
					status.VehicleLocation.Latitude, status.VehicleLocation.Longitude)
			}
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

	// Get vehicles to find the ID
	vehiclesResp, err := cli.apiClient.GetVehicles()
	if err != nil {
		fmt.Printf("Error: Failed to fetch vehicles: %v\n", err)
		return
	}

	// Use Vehicles or Result field depending on which is populated
	vehicleList := vehiclesResp.Vehicles
	if len(vehicleList) == 0 {
		vehicleList = vehiclesResp.Result
	}

	var targetVehicle *api.Vehicle
	for _, v := range vehicleList {
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
	status, err := cli.apiClient.GetVehicleStatus(targetVehicle.VehicleID)
	if err != nil {
		fmt.Printf("Error: Failed to get status: %v\n", err)
		return
	}

	// Display information
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  %s %s\n", targetVehicle.Nickname, targetVehicle.VehicleModel)
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n  VIN:        %s\n", targetVehicle.VIN)
	fmt.Printf("  Updated:    %s\n", status.LastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Odometer:   %d %s\n", status.OdometerStatus.Value, status.OdometerStatus.Unit)
	fmt.Printf("  Fuel Level: %d%%\n", status.VehicleStatus.FuelLevel)

	if status.VehicleStatus.Engine {
		fmt.Printf("  Engine:     Running\n")
	} else {
		fmt.Printf("  Engine:     Off\n")
	}

	if status.EVStatus != nil {
		fmt.Println("\n  Electric Vehicle Status:")
		fmt.Printf("    Battery:     %d%%\n", status.EVStatus.BatteryLevel)
		fmt.Printf("    Range:       %.1f km\n", status.EVStatus.RangeEV)
		if status.EVStatus.BatteryCharge {
			fmt.Printf("    Charging:    Yes\n")
			fmt.Printf("    Power:       %.1f kW\n", status.EVStatus.ChargingPower)
		} else {
			fmt.Printf("    Charging:    No\n")
		}
		fmt.Printf("    Plugged In:  %v\n", status.EVStatus.PluggedIn)
	}
}

func (cli *InteractiveCLI) showMetrics() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         Metrics                                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Printf("\n  API Client:\n")
	fmt.Printf("    Region:         %s\n", cli.apiClient.Region)
	fmt.Printf("    Brand:          %s\n", cli.apiClient.Brand)

	fmt.Printf("\n  System:\n")
	fmt.Printf("    Uptime:         %s\n", time.Since(cli.startTime).Round(time.Second))
	fmt.Printf("    Poll Interval:  %d minutes\n", cli.config.RateLimit.PollIntervalMinutes)
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
	if len(cli.config.RateLimit.Periods) > 0 {
		fmt.Printf("    Time Periods:    %d configured\n", len(cli.config.RateLimit.Periods))
	}

	fmt.Printf("\n  Charging Mode:\n")
	fmt.Printf("    Enabled:         %v\n", cli.config.ChargingMode.Enabled)
	if cli.config.ChargingMode.Enabled {
		fmt.Printf("    Interval:        %d minutes\n", cli.config.ChargingMode.IntervalMinutes)
		fmt.Printf("    Fast Interval:   %d minutes\n", cli.config.ChargingMode.FastChargeIntervalMinutes)
	}

	fmt.Printf("\n  Database:\n")
	fmt.Printf("    URL:      %s\n", cli.config.Database.URL)
	fmt.Printf("    Org:      %s\n", cli.config.Database.Organization)
	fmt.Printf("    Bucket:   %s\n", cli.config.Database.Bucket)

	fmt.Printf("\n  Alerts:\n")
	fmt.Printf("    Email:    %v\n", cli.config.Alerts.Enabled)
}

func (cli *InteractiveCLI) showCircuitBreaker() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Circuit Breaker Status                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	fmt.Println("\n  Note: Circuit breaker functionality has been removed in the refactored code.")
	fmt.Println("  The API client now uses simpler error handling with retries.")
	fmt.Printf("\n  API Client Status:\n")
	fmt.Printf("    Region:       %s\n", cli.apiClient.Region)
	fmt.Printf("    Brand:        %s\n", cli.apiClient.Brand)
	fmt.Printf("    Initialized:  ✓\n")
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
	vehiclesResp, err := cli.apiClient.GetVehicles()
	if err != nil {
		fmt.Printf("  ✗ Fetch Vehicles: FAILED (%v)\n", err)
		return
	}

	// Use Vehicles or Result field depending on which is populated
	vehicleList := vehiclesResp.Vehicles
	if len(vehicleList) == 0 {
		vehicleList = vehiclesResp.Result
	}

	vehicleTime := time.Since(start)
	fmt.Printf("  ✓ Fetch Vehicles: OK (%d vehicles in %s)\n",
		len(vehicleList), vehicleTime.Round(time.Millisecond))
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
