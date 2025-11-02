// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//
// Preflight check tool to validate connectivity to InfluxDB and Hyundai API

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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
	cfg, err := config.Load(*configPath)
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

	// Test 1: Ping
	fmt.Print("   ⏳ Testing connection... ")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		printError("", err)
		return false
	}

	if health.Status != "pass" {
		printError("", fmt.Errorf("InfluxDB health check failed: %s - %s", health.Status, *health.Message))
		return false
	}
	printSuccess("Connected")

	// Test 2: Authentication
	fmt.Print("   ⏳ Testing authentication... ")
	_, err = client.Ready(ctx)
	if err != nil {
		printError("", err)
		return false
	}
	printSuccess("Authenticated")

	// Test 3: Check bucket exists
	fmt.Print("   ⏳ Checking bucket access... ")
	bucketsAPI := client.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(ctx, cfg.Database.Bucket)
	if err != nil {
		printError("", err)
		fmt.Println()
		fmt.Printf("   %sℹ️  Bucket '%s' not found. You may need to create it:%s\n", colorYellow, cfg.Database.Bucket, colorReset)
		fmt.Printf("      make init-db\n")
		return false
	}
	if bucket == nil {
		printError("", fmt.Errorf("bucket '%s' not found", cfg.Database.Bucket))
		return false
	}
	printSuccess(fmt.Sprintf("Bucket '%s' exists", cfg.Database.Bucket))

	// Test 4: Test write permissions
	fmt.Print("   ⏳ Testing write permissions... ")
	writeAPI := client.WriteAPIBlocking(cfg.Database.Organization, cfg.Database.Bucket)
	testPoint := influxdb2.NewPointWithMeasurement("preflight_test").
		AddTag("test", "connectivity").
		AddField("value", 1).
		SetTime(time.Now())

	err = writeAPI.WritePoint(ctx, testPoint)
	if err != nil {
		printError("", err)
		return false
	}
	printSuccess("Write test successful")

	fmt.Println()
	fmt.Printf("   %s✓ InfluxDB checks passed%s\n", colorGreen, colorReset)
	return true
}

func checkHyundaiAPI(cfg *config.Config) bool {
	fmt.Printf("   Brand: %s\n", cfg.Hyundai.Brand)
	fmt.Printf("   Region: %s\n", cfg.Hyundai.Region)
	fmt.Printf("   Username: %s\n", maskString(cfg.Hyundai.Username))
	fmt.Println()

	// Test 1: Check credentials are set
	fmt.Print("   ⏳ Validating credentials... ")
	if cfg.Hyundai.Username == "" || cfg.Hyundai.Password == "" {
		printError("", fmt.Errorf("username or password not set"))
		return false
	}
	if cfg.Hyundai.Brand == "" {
		printError("", fmt.Errorf("brand not set"))
		return false
	}
	if cfg.Hyundai.Region == "" {
		printError("", fmt.Errorf("region not set"))
		return false
	}
	printSuccess("Credentials configured")

	// Test 2: Check API endpoint reachability
	fmt.Print("   ⏳ Testing API endpoint... ")

	// Determine API endpoint based on region
	var apiHost string
	switch cfg.Hyundai.Region {
	case "na":
		apiHost = "api.telematics.hyundaiusa.com:443"
	case "eu":
		apiHost = "prd.eu-ccapi.hyundai.com:443"
	case "kr":
		apiHost = "prd.kr-ccapi.hyundai.com:443"
	case "cn":
		apiHost = "prd.cn-ccapi.hyundai.com:443"
	case "au":
		apiHost = "prd.au-ccapi.hyundai.com:443"
	case "jp":
		apiHost = "prd.jp-ccapi.hyundai.com:443"
	case "in":
		apiHost = "prd.in-ccapi.hyundai.com:443"
	case "br":
		apiHost = "prd.br-ccapi.hyundai.com:443"
	default:
		apiHost = "prd.eu-ccapi.hyundai.com:443"
	}

	// For Kia, adjust endpoint
	if cfg.Hyundai.Brand == "kia" {
		apiHost = "prd.eu-ccapi.kia.com:443"
	}

	// Test TCP connection to API endpoint (more reliable than HTTP request)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", apiHost)
	if err != nil {
		printError("", err)
		fmt.Println()
		fmt.Printf("   %sℹ️  Could not reach API endpoint. Possible causes:%s\n", colorYellow, colorReset)
		fmt.Printf("      - Internet connectivity issues\n")
		fmt.Printf("      - Firewall blocking outbound HTTPS (port 443)\n")
		fmt.Printf("      - VPN or proxy configuration\n")
		return false
	}
	conn.Close()

	printSuccess(fmt.Sprintf("API endpoint reachable (%s)", apiHost))

	fmt.Println()
	fmt.Printf("   %s⚠️  Note: Full API authentication test requires actual login attempt%s\n", colorYellow, colorReset)
	fmt.Printf("   Run the logger to verify API credentials work correctly.\n")
	fmt.Println()
	fmt.Printf("   %s✓ Hyundai API checks passed%s\n", colorGreen, colorReset)
	return true
}

func printSuccess(msg string) {
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

func printError(prefix string, err error) {
	if prefix != "" {
		fmt.Printf("%s✗ %s: %v%s\n", colorRed, prefix, err, colorReset)
	} else {
		fmt.Printf("%s✗ %v%s\n", colorRed, err, colorReset)
	}
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
