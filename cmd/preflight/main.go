// Preflight check tool to validate connectivity to InfluxDB and Hyundai API
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
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBlue   = "\033[34m"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("%s  Hyundai Logger - Preflight Check%s\n", colorBlue, colorReset)
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	// Load configuration
	fmt.Printf("📋 Loading configuration from: %s\n", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		printError("Failed to load configuration", err)
		os.Exit(1)
	}
	printSuccess("Configuration loaded")
	fmt.Println()

	allPassed := true

	// Check InfluxDB connectivity
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("1️⃣  InfluxDB Connectivity Check")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if !checkInfluxDB(cfg) {
		allPassed = false
	}
	fmt.Println()

	// Check Hyundai API connectivity
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("2️⃣  Hyundai API Connectivity Check")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if !checkHyundaiAPI(cfg) {
		allPassed = false
	}
	fmt.Println()

	// Print summary
	fmt.Println("═══════════════════════════════════════════════════════")
	if allPassed {
		fmt.Printf("%s✓ All preflight checks PASSED%s\n", colorGreen, colorReset)
		fmt.Println()
		fmt.Println("Your system is ready to run hyundai-logger!")
		os.Exit(0)
	} else {
		fmt.Printf("%s✗ Some preflight checks FAILED%s\n", colorRed, colorReset)
		fmt.Println()
		fmt.Println("Please fix the issues above before running hyundai-logger.")
		os.Exit(1)
	}
}

func checkInfluxDB(cfg *config.Config) bool {
	fmt.Printf("   URL: %s\n", cfg.Database.URL)
	fmt.Printf("   Organization: %s\n", cfg.Database.Organization)
	fmt.Printf("   Bucket: %s\n", cfg.Database.Bucket)
	fmt.Println()

	// Create InfluxDB client
	client := influxdb2.NewClient(cfg.Database.URL, cfg.Database.Token)
	defer client.Close()

	// Test connection
	fmt.Print("   ⏳ Testing connection... ")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		printError("", err)
		return false
	}

	if health.Status != "pass" {
		printError("", fmt.Errorf("health check failed: %s", health.Message))
		return false
	}
	printSuccess("Connected")

	// Test authentication
	fmt.Print("   ⏳ Testing authentication... ")
	orgAPI := client.OrganizationsAPI()
	_, err = orgAPI.FindOrganizationByName(ctx, cfg.Database.Organization)
	if err != nil {
		printError("", fmt.Errorf("authentication failed or organization not found"))
		return false
	}
	printSuccess("Authenticated")

	// Test bucket access
	fmt.Print("   ⏳ Testing bucket access... ")
	bucketsAPI := client.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(ctx, cfg.Database.Bucket)
	if err != nil {
		printWarning(fmt.Sprintf("Bucket '%s' not found (will be created on first run)", cfg.Database.Bucket))
	} else {
		printSuccess(fmt.Sprintf("Bucket exists (ID: %s)", bucket.Id))
	}

	// Test write permission
	fmt.Print("   ⏳ Testing write permissions... ")
	writeAPI := client.WriteAPIBlocking(cfg.Database.Organization, cfg.Database.Bucket)
	testPoint := influxdb2.NewPoint(
		"preflight_test",
		map[string]string{"test": "true"},
		map[string]interface{}{"value": 1},
		time.Now(),
	)
	err = writeAPI.WritePoint(ctx, testPoint)
	if err != nil {
		printError("", fmt.Errorf("write test failed"))
		return false
	}
	printSuccess("Write permissions OK")

	return true
}

func checkHyundaiAPI(cfg *config.Config) bool {
	fmt.Printf("   Region: %s\n", cfg.Hyundai.Region)
	fmt.Printf("   Brand: %s\n", cfg.Hyundai.Brand)
	fmt.Printf("   Username: %s\n", maskString(cfg.Hyundai.Username))
	fmt.Println()

	// Create API client
	fmt.Print("   ⏳ Creating API client... ")
	client, err := api.NewClient(
		cfg.Hyundai.Region,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.RefreshToken,
	)
	if err != nil {
		printError("", err)
		fmt.Println()
		fmt.Printf("   %sNote:%s If you're getting auth errors, try running:\n", colorYellow, colorReset)
		fmt.Printf("         make oauth-manual\n")
		return false
	}
	printSuccess("Client created")

	// Test GetVehicles
	fmt.Print("   ⏳ Testing vehicle list access... ")
	vehicles, err := client.GetVehicles()
	if err != nil {
		printError("", err)
		fmt.Println()
		fmt.Printf("   %sPossible causes:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Invalid or expired refresh token\n")
		fmt.Printf("      2. Missing or invalid device_id (EU only)\n")
		fmt.Printf("      3. Network connectivity issues\n")
		fmt.Println()
		fmt.Printf("   %sFix:%s Run manual OAuth flow to get fresh tokens:\n", colorYellow, colorReset)
		fmt.Printf("         make oauth-manual\n")
		return false
	}

	// Check which field has vehicles
	vehicleList := vehicles.Vehicles
	if len(vehicleList) == 0 && len(vehicles.Result) > 0 {
		vehicleList = vehicles.Result
	}

	if len(vehicleList) == 0 {
		printWarning("No vehicles found in account")
	} else {
		printSuccess(fmt.Sprintf("Found %d vehicle(s)", len(vehicleList)))
		for _, v := range vehicleList {
			fmt.Printf("      - %s (%s)\n", v.Nickname, v.VIN)
		}
	}

	return true
}

func printSuccess(msg string) {
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, msg)
}

func printError(prefix string, err error) {
	if prefix != "" {
		fmt.Printf("%s✗%s %s: %v\n", colorRed, colorReset, prefix, err)
	} else {
		fmt.Printf("%s✗%s %v\n", colorRed, colorReset, err)
	}
}

func printWarning(msg string) {
	fmt.Printf("%s⚠%s  %s\n", colorYellow, colorReset, msg)
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
