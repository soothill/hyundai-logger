package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
	"github.com/soothill/hyundai-logger/internal/database"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	initDB := flag.Bool("init-db", false, "Initialize database schema")
	getToken := flag.Bool("get-token", false, "Run interactive token fetcher")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Handle token fetcher mode
	if *getToken {
		fmt.Println("Please use the Python script to obtain refresh token:")
		fmt.Println("python3 scripts/get_refresh_token.py <REGION> <BRAND>")
		fmt.Println("\nExample: python3 scripts/get_refresh_token.py EU hyundai")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewClient(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize database schema if requested
	if *initDB {
		if initErr := db.InitializeSchema(); initErr != nil {
			log.Fatalf("Failed to initialize database schema: %v", initErr)
		}
		fmt.Println("Database schema initialized successfully")
		os.Exit(0)
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
		log.Fatalf("Failed to create API client: %v", err)
	}

	// Start the logger
	logger := &VehicleLogger{
		config:    cfg,
		apiClient: apiClient,
		dbClient:  db,
		verbose:   *verbose,
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start logging
	fmt.Println("Starting Hyundai Vehicle Logger...")
	fmt.Printf("Region: %s, Brand: %s\n", cfg.Hyundai.Region, cfg.Hyundai.Brand)
	fmt.Printf("Poll interval: %v\n", cfg.GetPollInterval())
	fmt.Printf("Database: %s (org: %s, bucket: %s)\n",
		cfg.Database.URL, cfg.Database.Organization, cfg.Database.Bucket)

	// Run the logger
	logger.wg.Add(1)
	go logger.Run()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down gracefully...")
	logger.Stop()

	// Wait for goroutine to finish
	logger.wg.Wait()
	fmt.Println("Shutdown complete")
}

// VehicleLogger handles the main logging loop
type VehicleLogger struct {
	config           *config.Config
	apiClient        *api.Client
	dbClient         *database.Client
	verbose          bool
	stopChan         chan bool
	chargingVehicles map[string]bool // Track which vehicles are currently charging
	chargingMu       sync.RWMutex    // Protects chargingVehicles map
	wg               sync.WaitGroup  // Tracks running goroutines
}

// Run starts the logging loop
func (vl *VehicleLogger) Run() {
	defer vl.wg.Done()

	// Panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in polling loop: %v", r)
		}
	}()

	vl.stopChan = make(chan bool)
	vl.chargingVehicles = make(map[string]bool)

	// Initial delay to prevent immediate polling on startup
	time.Sleep(30 * time.Second)

	// Get vehicles with retry logic
	var vehicles *api.VehiclesResponse
	var vehicleErr error
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		vehicles, vehicleErr = vl.apiClient.GetVehicles()
		if vehicleErr == nil {
			break
		}

		log.Printf("Failed to get vehicles (attempt %d/%d): %v", attempt, maxRetries, vehicleErr)
		if attempt < maxRetries {
			// Exponential backoff: 5s, 10s, 20s
			backoff := time.Duration(5*attempt) * time.Second
			log.Printf("Retrying in %v...", backoff)

			select {
			case <-vl.stopChan:
				return // Exit if stop requested during retry
			case <-time.After(backoff):
				// Continue to next retry
			}
		}
	}

	if vehicleErr != nil {
		log.Printf("Failed to get vehicles after %d attempts, exiting: %v", maxRetries, vehicleErr)
		return
	}

	if len(vehicles.Vehicles) == 0 && len(vehicles.Result) == 0 {
		log.Println("No vehicles found in account")
		return
	}

	// Use Result field if Vehicles is empty (different API responses)
	vehicleList := vehicles.Vehicles
	if len(vehicleList) == 0 {
		vehicleList = vehicles.Result
	}

	fmt.Printf("Found %d vehicle(s)\n", len(vehicleList))
	for _, v := range vehicleList {
		fmt.Printf("- %s (%s) - VIN: %s\n", v.Nickname, v.VehicleModel, v.VIN)
	}

	// Main polling loop
	for {
		select {
		case <-vl.stopChan:
			return
		default:
			// Poll each vehicle
			for _, vehicle := range vehicleList {
				if err := vl.pollVehicle(vehicle); err != nil {
					log.Printf("Error polling vehicle %s: %v", vehicle.Nickname, err)
				}
			}

			// Determine next poll interval based on charging status
			interval := vl.getNextPollInterval()
			if vl.verbose {
				if len(vl.chargingVehicles) > 0 {
					fmt.Printf("Vehicle(s) charging - using faster interval: %v\n", interval)
				} else {
					fmt.Printf("Waiting %v until next poll...\n", interval)
				}
			}

			select {
			case <-vl.stopChan:
				return
			case <-time.After(interval):
				// Continue to next poll
			}
		}
	}
}

// Stop stops the logging loop
func (vl *VehicleLogger) Stop() {
	if vl.stopChan != nil {
		close(vl.stopChan)
	}
}

// pollVehicle polls a single vehicle and logs its data
func (vl *VehicleLogger) pollVehicle(vehicle api.Vehicle) error {
	if vl.verbose {
		fmt.Printf("Polling vehicle: %s (VIN: %s)\n", vehicle.Nickname, vehicle.VIN)
	}

	// Get vehicle status
	status, err := vl.apiClient.GetVehicleStatus(vehicle.VehicleID)
	if err != nil {
		// Try to refresh status if cached data fails
		if vl.verbose {
			fmt.Println("Cached status failed, attempting refresh...")
		}

		if refreshErr := vl.apiClient.RefreshVehicleStatus(vehicle.VehicleID); refreshErr != nil {
			return fmt.Errorf("failed to refresh status: %w", refreshErr)
		}

		// Wait a bit for the refresh to complete
		time.Sleep(5 * time.Second)

		// Try getting status again
		status, err = vl.apiClient.GetVehicleStatus(vehicle.VehicleID)
		if err != nil {
			return fmt.Errorf("failed to get status after refresh: %w", err)
		}
	}

	// Get location separately if not included in status
	if status.VehicleLocation.Latitude == 0 && status.VehicleLocation.Longitude == 0 {
		location, err := vl.apiClient.GetLocation(vehicle.VehicleID)
		if err == nil {
			status.VehicleLocation = *location
		} else if vl.verbose {
			fmt.Printf("Warning: Could not get location: %v\n", err)
		}
	}

	// Update charging status for this vehicle (protected by mutex)
	isCharging := vl.shouldUseFastPolling(status)
	vl.chargingMu.Lock()
	vl.chargingVehicles[vehicle.VIN] = isCharging
	vl.chargingMu.Unlock()

	if isCharging && vl.verbose {
		chargingPower := 0.0
		if status.EVStatus != nil {
			chargingPower = status.EVStatus.ChargingPower
		}
		fmt.Printf("EV is charging at %.1f kW - faster polling enabled\n", chargingPower)
	}

	// Log the data to InfluxDB
	if err := vl.dbClient.WriteVehicleStatus(vehicle, status); err != nil {
		return fmt.Errorf("failed to write to database: %w", err)
	}

	if vl.verbose {
		vl.printStatus(vehicle, status)
	}

	return nil
}

// getNextPollInterval determines the appropriate poll interval based on charging status
func (vl *VehicleLogger) getNextPollInterval() time.Duration {
	// Check charging status with read lock
	vl.chargingMu.RLock()
	defer vl.chargingMu.RUnlock()

	// If any vehicle is charging, use faster polling
	if len(vl.chargingVehicles) > 0 {
		for _, isCharging := range vl.chargingVehicles {
			if isCharging {
				// Use charging interval from config
				return time.Duration(vl.config.ChargingMode.IntervalMinutes) * time.Minute
			}
		}
	}

	// Default to configured poll interval
	return vl.config.GetPollInterval()
}

// shouldUseFastPolling checks if we should use faster polling (e.g., during EV charging)
func (vl *VehicleLogger) shouldUseFastPolling(status *api.VehicleStatus) bool {
	if !vl.config.ChargingMode.Enabled {
		return false
	}

	if status.EVStatus != nil && status.EVStatus.BatteryCharge {
		// Vehicle is actively charging
		return true
	}

	return false
}

// printStatus prints the vehicle status to console
func (vl *VehicleLogger) printStatus(vehicle api.Vehicle, status *api.VehicleStatus) {
	fmt.Printf("\n=== Vehicle Status: %s ===\n", vehicle.Nickname)
	fmt.Printf("Last Update: %s\n", status.LastUpdateTime.Format(time.RFC3339))

	// General status
	fmt.Printf("\nGeneral:\n")
	fmt.Printf("  Engine: %v\n", status.VehicleStatus.Engine)
	fmt.Printf("  Locked: %v\n", status.VehicleStatus.Locked)
	fmt.Printf("  Fuel Level: %d%%\n", status.VehicleStatus.FuelLevel)
	fmt.Printf("  12V Battery: %.1fV\n", status.VehicleStatus.BatteryVoltage)
	fmt.Printf("  Odometer: %d %s\n", status.OdometerStatus.Value, status.OdometerStatus.Unit)

	// Location
	if status.VehicleLocation.Latitude != 0 || status.VehicleLocation.Longitude != 0 {
		fmt.Printf("\nLocation:\n")
		fmt.Printf("  Coordinates: %.6f, %.6f\n", status.VehicleLocation.Latitude, status.VehicleLocation.Longitude)
		fmt.Printf("  Speed: %.1f km/h\n", status.VehicleLocation.Speed)
	}

	// EV Status (if applicable)
	if status.EVStatus != nil {
		fmt.Printf("\nEV Status:\n")
		fmt.Printf("  Battery Level: %d%%\n", status.EVStatus.BatteryLevel)
		fmt.Printf("  Charging: %v\n", status.EVStatus.BatteryCharge)
		fmt.Printf("  Plugged In: %v\n", status.EVStatus.PluggedIn)
		if status.EVStatus.BatteryCharge {
			fmt.Printf("  Charging Power: %.1f kW\n", status.EVStatus.ChargingPower)
			fmt.Printf("  Est. Time to Full: %d min\n", status.EVStatus.EstimatedChargeTime)
		}
		fmt.Printf("  EV Range: %.1f km\n", status.EVStatus.RangeEV)
	}

	// Climate
	fmt.Printf("\nClimate:\n")
	fmt.Printf("  AC Active: %v\n", status.Climate.Active)
	fmt.Printf("  Interior Temp: %.1f°C\n", status.Climate.InteriorTemp)
	fmt.Printf("  Exterior Temp: %.1f°C\n", status.Climate.ExteriorTemp)

	// Doors
	fmt.Printf("\nDoors:\n")
	fmt.Printf("  Front Left: %v, Front Right: %v\n", status.DoorStatus.FrontLeft, status.DoorStatus.FrontRight)
	fmt.Printf("  Rear Left: %v, Rear Right: %v\n", status.DoorStatus.BackLeft, status.DoorStatus.BackRight)
	fmt.Printf("  Trunk: %v, Hood: %v\n", status.DoorStatus.Trunk, status.DoorStatus.Hood)

	// Tires
	fmt.Printf("\nTire Pressure:\n")
	fmt.Printf("  Front: L=%.1f PSI, R=%.1f PSI\n", status.TireStatus.FrontLeftPSI, status.TireStatus.FrontRightPSI)
	fmt.Printf("  Rear:  L=%.1f PSI, R=%.1f PSI\n", status.TireStatus.RearLeftPSI, status.TireStatus.RearRightPSI)
}
