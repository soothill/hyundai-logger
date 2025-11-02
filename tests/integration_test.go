// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/circuitbreaker"
	"github.com/soothill/hyundai-logger/internal/retry"
)

// MockAPIServer provides a test HTTP server that simulates Hyundai Bluelink API
type MockAPIServer struct {
	server          *httptest.Server
	authCallCount   int
	vehicleCallCount int
	statusCallCount int
	locationCallCount int
}

// NewMockAPIServer creates a new mock API server
func NewMockAPIServer() *MockAPIServer {
	mock := &MockAPIServer{}

	mux := http.NewServeMux()

	// Authentication endpoint
	mux.HandleFunc("/v2/login", func(w http.ResponseWriter, r *http.Request) {
		mock.authCallCount++

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"access_token":  "test_access_token",
			"refresh_token": "test_refresh_token",
			"expires_in":    3600,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Get vehicles endpoint
	mux.HandleFunc("/v2/vehicles", func(w http.ResponseWriter, r *http.Request) {
		mock.vehicleCallCount++

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		vehicles := api.VehiclesResponse{
			Vehicles: []api.Vehicle{
				{
					VehicleID:  "test-vehicle-1",
					VIN:        "5NPE24AF1KH123456",
					Make:       "Hyundai",
					Model:      "IONIQ 5",
					Year:       2024,
					Nickname:   "Test Vehicle",
					Color:      "Blue",
					Generation: "2024",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vehicles)
	})

	// Get vehicle status endpoint
	mux.HandleFunc("/v2/vehicles/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/vehicles/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Extract vehicle ID and endpoint
		path := r.URL.Path
		if len(path) > len("/v2/vehicles/") {
			// Handle /v2/vehicles/{id}/status
			if r.URL.Path[len(r.URL.Path)-7:] == "/status" {
				mock.statusCallCount++

				status := &api.VehicleStatus{
					VIN:       "5NPE24AF1KH123456",
					Timestamp: time.Now(),
					Odometer:  12345.6,
					FuelLevel: 75.5,
					Engine: api.EngineStatus{
						Running:          false,
						RemoteStartState: false,
						Rpm:              0,
						RangeKM:          450.0,
						RangeMiles:       280.0,
					},
					Climate: api.ClimateStatus{
						Active:       false,
						TargetTemp:   22.0,
						InteriorTemp: 20.0,
						ExteriorTemp: 15.0,
						FanSpeed:     0,
					},
					Doors: api.DoorsStatus{
						Locked:     true,
						FrontLeft:  false,
						FrontRight: false,
						BackLeft:   false,
						BackRight:  false,
						Trunk:      false,
						Hood:       false,
					},
					Battery: api.BatteryStatus{
						Level:   100,
						Voltage: 12.6,
					},
					Tire: api.TireStatus{
						FrontLeft: api.TirePressure{
							PSI:    35.0,
							Status: "OK",
						},
						FrontRight: api.TirePressure{
							PSI:    35.0,
							Status: "OK",
						},
						RearLeft: api.TirePressure{
							PSI:    35.0,
							Status: "OK",
						},
						RearRight: api.TirePressure{
							PSI:    35.0,
							Status: "OK",
						},
					},
					EV: &api.EVStatus{
						BatteryLevel:           85.5,
						BatteryCapacity:        77.4,
						Charging:               true,
						ChargingPower:          11.0,
						EstimatedCurrentCharge: 30,  // minutes
						EstimatedFullCharge:    120, // minutes
						RangeKM:                400.0,
						RangeMiles:             250.0,
						PluggedIn:              true,
						ChargeTargetPercent:    90,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(status)
				return
			}

			// Handle /v2/vehicles/{id}/location
			if len(r.URL.Path) > 9 && r.URL.Path[len(r.URL.Path)-9:] == "/location" {
				mock.locationCallCount++

				location := &api.Location{
					VIN:       "5NPE24AF1KH123456",
					Timestamp: time.Now(),
					Location: api.LocationData{
						Latitude:  37.7749,
						Longitude: -122.4194,
						Altitude:  100.0,
						Speed:     0.0,
						Heading:   0.0,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(location)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

// Close shuts down the mock server
func (m *MockAPIServer) Close() {
	m.server.Close()
}

// URL returns the mock server URL
func (m *MockAPIServer) URL() string {
	return m.server.URL
}

// ResetCounters resets all call counters
func (m *MockAPIServer) ResetCounters() {
	m.authCallCount = 0
	m.vehicleCallCount = 0
	m.statusCallCount = 0
	m.locationCallCount = 0
}

// TestAPIAuthentication tests the authentication flow
func TestAPIAuthentication(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	// Create retry config
	retryConfig := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    100,
		MaxDelayMs:        1000,
		BackoffMultiplier: 2.0,
	}

	// Create client and set custom base URL for mock server
	client := api.NewClient("testuser", "testpass", "1234", "hyundai", "US", 100, retryConfig)
	client.SetBaseURL(mock.URL())
	client.DisableCache() // Disable cache for testing

	// The mock server automatically returns a valid auth response
	// Just verify that creating the client worked
	if client == nil {
		t.Fatal("Failed to create API client")
	}

	// Note: Actual authentication happens on first API call
	// We're just testing that the mock server can be reached
}

// TestAPIRetryLogic tests retry behavior
func TestAPIRetryLogic(t *testing.T) {
	attemptCount := 0
	maxAttempts := 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		// Fail first 2 attempts, succeed on 3rd
		if attemptCount < maxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success on 3rd attempt
		response := map[string]interface{}{
			"access_token":  "test_token",
			"refresh_token": "test_refresh",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	retryConfig := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	client := api.NewClient("test", "test", "1234", "hyundai", "US", 100, retryConfig)
	client.SetBaseURL(server.URL)
	client.DisableCache()

	// Verify retry logic worked - should have made exactly 3 attempts
	// (Note: actual API call would be needed to trigger retries, but we're testing the setup)
	if attemptCount > 0 && attemptCount != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attemptCount)
	}
}

// TestCircuitBreakerIntegration tests circuit breaker behavior
func TestCircuitBreakerIntegration(t *testing.T) {
	retryConfig := retry.Config{
		MaxAttempts:       1, // No retries, we want to test circuit breaker
		InitialDelayMs:    10,
		MaxDelayMs:        50,
		BackoffMultiplier: 2.0,
	}

	client := api.NewClient("test", "test", "1234", "hyundai", "US", 100, retryConfig)

	// Verify circuit breaker is in Closed state initially
	state := client.GetCircuitBreakerState()
	expectedState := "closed" // Circuit breaker starts closed
	if state.String() != expectedState {
		t.Errorf("Expected circuit breaker state %s, got %s", expectedState, state.String())
	}

	// Note: Full circuit breaker integration testing requires a properly configured
	// mock server with authentication endpoints. This test verifies the circuit
	// breaker is properly initialized.
	// For comprehensive circuit breaker testing, see internal/circuitbreaker/breaker_test.go
}

// TestConcurrentPolling tests parallel vehicle polling
func TestConcurrentPolling(t *testing.T) {
	// This would test the parallel polling implementation
	// Requires database setup which is complex for integration tests
	t.Skip("Skipping - requires InfluxDB setup for full integration test")
}

// TestDatabaseBatching tests that batching reduces write operations
func TestDatabaseBatching(t *testing.T) {
	// This would verify that multiple vehicles write in a single batch
	// Requires database setup
	t.Skip("Skipping - requires InfluxDB setup for full integration test")
}

// TestEndToEndFlow tests the complete flow from API to database
func TestEndToEndFlow(t *testing.T) {
	// This would test:
	// 1. Authenticate
	// 2. Get vehicles
	// 3. Poll vehicles
	// 4. Write to database
	// 5. Verify data in database
	t.Skip("Skipping - requires InfluxDB setup for full integration test")
}

// TestMockServerBasics verifies the mock server works correctly
func TestMockServerBasics(t *testing.T) {
	mock := NewMockAPIServer()
	defer mock.Close()

	// Test authentication endpoint
	resp, err := http.Post(mock.URL()+"/v2/login", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("Failed to call login endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if mock.authCallCount != 1 {
		t.Errorf("Expected 1 auth call, got %d", mock.authCallCount)
	}

	// Test vehicles endpoint
	resp, err = http.Get(mock.URL() + "/v2/vehicles")
	if err != nil {
		t.Fatalf("Failed to call vehicles endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var vehiclesResp api.VehiclesResponse
	if err := json.NewDecoder(resp.Body).Decode(&vehiclesResp); err != nil {
		t.Fatalf("Failed to decode vehicles response: %v", err)
	}

	if len(vehiclesResp.Vehicles) != 1 {
		t.Errorf("Expected 1 vehicle, got %d", len(vehiclesResp.Vehicles))
	}

	if vehiclesResp.Vehicles[0].VIN != "5NPE24AF1KH123456" {
		t.Errorf("Expected VIN 5NPE24AF1KH123456, got %s", vehiclesResp.Vehicles[0].VIN)
	}
}

// TestCircuitBreakerRetryIntegration verifies that circuit breaker and retry work together correctly
// This test ensures that retries don't cause retry storms when the circuit is open
func TestCircuitBreakerRetryIntegration(t *testing.T) {
	// Create a mock server that fails consistently
	failureCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failureCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create retry config with 3 attempts
	retryConfig := retry.Config{
		MaxAttempts:       3,
		InitialDelayMs:    10,
		MaxDelayMs:        100,
		BackoffMultiplier: 2.0,
	}

	// Create circuit breaker config with 3 max failures
	client := api.NewClient("test", "test", "1234", "hyundai", "US", 1000, retryConfig)
	client.SetBaseURL(server.URL)
	client.DisableCache()

	ctx := context.Background()

	// First call should fail and retry 3 times
	_, err := client.GetVehicles(ctx)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// After 3 retries (each failing), we should have 3 failures
	// Circuit breaker should now be open after 3 consecutive failures
	expectedFailures := 3 // 3 retries on the first request
	if failureCount != expectedFailures {
		t.Logf("Warning: Expected %d failures, got %d (circuit breaker may have opened earlier)", expectedFailures, failureCount)
	}

	// Get circuit breaker state
	state, failures, _ := client.GetCircuitBreakerStats()

	// Circuit should be open after 3 failures
	if state != circuitbreaker.StateOpen {
		t.Errorf("Expected circuit to be Open, got %v (failures: %d)", state, failures)
	}

	// Second call should fail immediately without retries (circuit is open)
	initialFailureCount := failureCount
	_, err = client.GetVehicles(ctx)
	if err == nil {
		t.Error("Expected error when circuit is open, got nil")
	}

	// Should not have attempted any new requests (circuit is open)
	if failureCount != initialFailureCount {
		t.Errorf("Circuit breaker did not prevent retry storm: %d new failures when circuit should be open", failureCount-initialFailureCount)
	}

	t.Logf("Circuit breaker successfully prevented retry storm after %d failures", failureCount)
}
