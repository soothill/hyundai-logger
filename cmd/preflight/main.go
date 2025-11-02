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
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      URL: %s\n", cfg.Database.URL)
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Check if InfluxDB is running:\n")
		fmt.Printf("         curl -v %s/health\n", cfg.Database.URL)
		fmt.Printf("      2. Verify URL in config.yaml is correct\n")
		fmt.Printf("      3. Check if InfluxDB container is running:\n")
		fmt.Printf("         docker ps | grep influx\n")
		fmt.Printf("      4. Check InfluxDB logs:\n")
		fmt.Printf("         docker logs influxdb\n")
		return false
	}

	if health.Status != "pass" {
		printError("", fmt.Errorf("InfluxDB health check failed: %s - %s", health.Status, *health.Message))
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Health Status: %s\n", health.Status)
		fmt.Printf("      Message: %s\n", *health.Message)
		return false
	}
	printSuccess("Connected")

	// Test 2: Authentication
	fmt.Print("   ⏳ Testing authentication... ")
	_, err = client.Ready(ctx)
	if err != nil {
		printError("", err)
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Organization: %s\n", cfg.Database.Organization)
		fmt.Printf("      Token: %s\n", maskToken(cfg.Database.Token))
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Verify token has correct permissions\n")
		fmt.Printf("      2. Check token in InfluxDB UI: %s\n", cfg.Database.URL)
		fmt.Printf("      3. Generate new token if needed:\n")
		fmt.Printf("         influx auth create --org %s \\\n", cfg.Database.Organization)
		fmt.Printf("           --read-buckets --write-buckets\n")
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
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Bucket: %s\n", cfg.Database.Bucket)
		fmt.Printf("      Organization: %s\n", cfg.Database.Organization)
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sFix:%s Create the bucket:\n", colorYellow, colorReset)
		fmt.Printf("      influx bucket create \\\n")
		fmt.Printf("        --host %s \\\n", cfg.Database.URL)
		fmt.Printf("        --token YOUR_TOKEN \\\n")
		fmt.Printf("        --org %s \\\n", cfg.Database.Organization)
		fmt.Printf("        --name %s \\\n", cfg.Database.Bucket)
		fmt.Printf("        --retention 90d\n")
		return false
	}
	if bucket == nil {
		printError("", fmt.Errorf("bucket '%s' not found", cfg.Database.Bucket))
		fmt.Println()
		fmt.Printf("   %sFix:%s Create the bucket:\n", colorYellow, colorReset)
		fmt.Printf("      influx bucket create \\\n")
		fmt.Printf("        --host %s \\\n", cfg.Database.URL)
		fmt.Printf("        --token YOUR_TOKEN \\\n")
		fmt.Printf("        --org %s \\\n", cfg.Database.Organization)
		fmt.Printf("        --name %s \\\n", cfg.Database.Bucket)
		fmt.Printf("        --retention 90d\n")
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
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      Token may not have write permissions\n")
		fmt.Printf("      Generate a new token with write access in InfluxDB UI\n")
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
		fmt.Println()
		fmt.Printf("   %sFix:%s Set credentials in your config.yaml:\n", colorYellow, colorReset)
		fmt.Printf("      hyundai:\n")
		fmt.Printf("        username: \"your.email@example.com\"\n")
		fmt.Printf("        password: \"your-password\"\n")
		return false
	}
	if cfg.Hyundai.Brand == "" {
		printError("", fmt.Errorf("brand not set"))
		fmt.Println()
		fmt.Printf("   %sFix:%s Set brand in your config.yaml (hyundai, kia, or genesis)\n", colorYellow, colorReset)
		return false
	}
	if cfg.Hyundai.Region == "" {
		printError("", fmt.Errorf("region not set"))
		fmt.Println()
		fmt.Printf("   %sFix:%s Set region in your config.yaml (na, eu, kr, etc.)\n", colorYellow, colorReset)
		return false
	}
	printSuccess("Credentials configured")

	// Test 2: Check basic internet connectivity first
	fmt.Print("   ⏳ Testing internet connectivity... ")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with a reliable public endpoint
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
	if err != nil {
		printError("", fmt.Errorf("no internet connection"))
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Cannot reach internet (tested with 1.1.1.1:443)\n")
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Check your network connection\n")
		fmt.Printf("      2. Verify DNS is working: nslookup google.com\n")
		fmt.Printf("      3. Check if proxy is required: echo $HTTP_PROXY\n")
		fmt.Printf("      4. Test basic connectivity: ping 1.1.1.1\n")
		return false
	}
	conn.Close()
	printSuccess("Internet reachable")

	// Test 3: Check API endpoint reachability
	fmt.Print("   ⏳ Testing Hyundai API endpoint... ")

	// Determine API endpoint based on region
	var apiHost string
	var apiHostname string
	switch cfg.Hyundai.Region {
	case "na":
		apiHostname = "api.telematics.hyundaiusa.com"
		apiHost = apiHostname + ":443"
	case "eu":
		apiHostname = "prd.eu-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "kr":
		apiHostname = "prd.kr-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "cn":
		apiHostname = "prd.cn-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "au":
		apiHostname = "prd.au-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "jp":
		apiHostname = "prd.jp-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "in":
		apiHostname = "prd.in-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	case "br":
		apiHostname = "prd.br-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	default:
		apiHostname = "prd.eu-ccapi.hyundai.com"
		apiHost = apiHostname + ":443"
	}

	// For Kia, adjust endpoint
	if cfg.Hyundai.Brand == "kia" {
		apiHostname = "prd.eu-ccapi.kia.com"
		apiHost = apiHostname + ":443"
	}

	// Test DNS resolution first
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	ips, err := net.DefaultResolver.LookupIP(ctx2, "ip", apiHostname)
	if err != nil {
		printError("", fmt.Errorf("DNS resolution failed"))
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Cannot resolve hostname: %s\n", apiHostname)
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Test DNS resolution: nslookup %s\n", apiHostname)
		fmt.Printf("      2. Try alternative DNS: dig @8.8.8.8 %s\n", apiHostname)
		fmt.Printf("      3. Check /etc/resolv.conf for DNS servers\n")
		return false
	}

	// Test TCP connection to API endpoint
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel3()

	conn, err = dialer.DialContext(ctx3, "tcp", apiHost)
	if err != nil {
		printError("", fmt.Errorf("connection failed"))
		fmt.Println()
		fmt.Printf("   %sDiagnostics:%s\n", colorYellow, colorReset)
		fmt.Printf("      Hostname: %s\n", apiHostname)
		fmt.Printf("      Resolved IPs: %v\n", ips)
		fmt.Printf("      Port: 443 (HTTPS)\n")
		fmt.Printf("      Error: %v\n", err)
		fmt.Println()
		fmt.Printf("   %sTroubleshooting:%s\n", colorYellow, colorReset)
		fmt.Printf("      1. Test direct connection:\n")
		fmt.Printf("         curl -v --connect-timeout 10 https://%s\n", apiHostname)
		fmt.Printf("      2. Check firewall rules:\n")
		fmt.Printf("         sudo iptables -L -n | grep 443\n")
		fmt.Printf("      3. Test with netcat:\n")
		fmt.Printf("         nc -zv %s 443\n", apiHostname)
		fmt.Printf("      4. Check if running in restricted network/container\n")
		fmt.Printf("      5. Try from different network to rule out ISP blocking\n")
		fmt.Println()
		fmt.Printf("   %sNote:%s The API endpoint may have geographic or IP-based restrictions.\n", colorYellow, colorReset)
		fmt.Printf("         If running in Docker, ensure network mode allows external access.\n")
		return false
	}
	conn.Close()

	printSuccess(fmt.Sprintf("API endpoint reachable (%s -> %v)", apiHostname, ips[0]))

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

func maskToken(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
