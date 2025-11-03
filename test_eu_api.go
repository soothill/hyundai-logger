package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Test program to call EU API directly and see response
func main() {
	// Load tokens from .auth_tokens
	data, err := os.ReadFile(".auth_tokens")
	if err != nil {
		fmt.Printf("Error reading .auth_tokens: %v\n", err)
		return
	}

	var accessToken string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ACCESS_TOKEN=") {
			accessToken = strings.Trim(strings.TrimPrefix(line, "ACCESS_TOKEN="), "\"")
			break
		}
	}

	if accessToken == "" {
		fmt.Println("No access token found")
		return
	}

	fmt.Printf("Using access token: %s...\n", accessToken[:50])

	// Generate device ID
	deviceID := fmt.Sprintf("%064x", time.Now().UnixNano())

	// Generate stamp (XOR-based)
	cfbHyundai, _ := base64.StdEncoding.DecodeString("RFtoRq/vDXJmRndoZaZQyfOot7OrIqGVFj96iY2WL3yyH5Z/pUvlUhqmCxD2t+D65SQ=")
	appIDHyundai := "014d2225-8495-4735-812d-2616334fd15d"
	timestamp := time.Now().Unix()
	rawData := []byte(fmt.Sprintf("%s:%d", appIDHyundai, timestamp))
	result := make([]byte, len(rawData))
	for i := 0; i < len(rawData); i++ {
		result[i] = cfbHyundai[i%len(cfbHyundai)] ^ rawData[i]
	}
	stamp := base64.StdEncoding.EncodeToString(result)

	// Make request to EU vehicles endpoint (v1 is correct)
	endpoint := "https://prd.eu-ccapi.hyundai.com:8080/api/v1/spa/vehicles"
	fmt.Printf("\nCalling: %s\n", endpoint)

	req, err := http.NewRequestWithContext(context.Background(), "GET", endpoint, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("ccsp-service-id", "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a")
	req.Header.Set("ccsp-application-id", "99cfff84-f4e2-4be8-a5ed-e5b755eb6581")
	req.Header.Set("ccsp-device-id", deviceID)
	req.Header.Set("Stamp", stamp)
	req.Header.Set("clientId", "ANDROID")
	req.Header.Set("Host", "prd.eu-ccapi.hyundai.com:8080")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "okhttp/3.12.1")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("\nResponse Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Println("\nResponse Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Println("\nResponse Body:")
	fmt.Println(string(body))

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		fmt.Println("\nParsed JSON:")
		prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
		fmt.Println(string(prettyJSON))
	}

	if resp.StatusCode == 200 {
		fmt.Println("\n✓ SUCCESS!")
	} else {
		fmt.Println("\n✗ FAILED")
	}
}
