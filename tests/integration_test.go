// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
)

// MockAPIServer provides a test HTTP server that simulates Hyundai Bluelink API
type MockAPIServer struct {
	server            *httptest.Server
	authCallCount     int
	vehicleCallCount  int
	statusCallCount   int
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
					VehicleID:    "test-vehicle-1",
					VIN:          "5NPE24AF1KH123456",
					VehicleName:  "Hyundai",
					VehicleModel: "IONIQ 5",
					Year:         "2024",
					Nickname:     "Test Vehicle",
					Color:        "Blue",
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
					LastUpdateTime: time.Now(),
					VehicleStatus: api.GeneralStatus{
						Engine:         false,
						Locked:         true,
						FuelLevel:      75,
						BatteryVoltage: 12.6,
					},
					OdometerStatus: api.OdometerStatus{
						Value: 12345,
						Unit:  "km",
					},
					Climate: api.ClimateStatus{
						Active:       false,
						TargetTemp:   22.0,
						InteriorTemp: 20.0,
						ExteriorTemp: 15.0,
						FanSpeed:     0,
					},
					DoorStatus: api.DoorStatus{
						FrontLeft:  false,
						FrontRight: false,
						BackLeft:   false,
						BackRight:  false,
						Trunk:      false,
						Hood:       false,
					},
					TireStatus: api.TireStatus{
						FrontLeftPSI:     35.0,
						FrontRightPSI:    35.0,
						RearLeftPSI:      35.0,
						RearRightPSI:     35.0,
						FrontLeftStatus:  "OK",
						FrontRightStatus: "OK",
						RearLeftStatus:   "OK",
						RearRightStatus:  "OK",
					},
					EVStatus: &api.EVStatus{
						BatteryLevel:        85,
						BatteryCapacity:     77.4,
						BatteryCharge:       true,
						ChargingPower:       11.0,
						EstimatedChargeTime: 30, // minutes
						RangeEV:             400.0,
						PluggedIn:           true,
						TargetChargeLevel:   90,
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
					Latitude:  37.7749,
					Longitude: -122.4194,
					Altitude:  100.0,
					Speed:     0.0,
					Heading:   0.0,
					Time:      time.Now(),
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
	t.Skip("Test skipped: API client refactored - no longer supports custom base URL for mocking. Authentication now happens in NewClient().")
}

// TestAPIRetryLogic tests retry behavior
func TestAPIRetryLogic(t *testing.T) {
	t.Skip("Test skipped: API client refactored - no longer supports SetBaseURL() for testing. Retry logic is now internal to doRequest().")
}

// TestCircuitBreakerIntegration tests circuit breaker behavior
func TestCircuitBreakerIntegration(t *testing.T) {
	t.Skip("Test skipped: Circuit breaker functionality removed from refactored API client. API now uses simpler error handling with retries.")
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
	t.Skip("Test skipped: Circuit breaker functionality removed from refactored API client.")
}
