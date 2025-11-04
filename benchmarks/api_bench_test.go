// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package benchmarks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
)

// createTestAPIServer creates a test HTTP server for benchmarking
func createTestAPIServer() *httptest.Server {
	mux := http.NewServeMux()

	// Login endpoint
	mux.HandleFunc("/v2/login", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"access_token":  "test_token",
			"refresh_token": "test_refresh",
			"expires_in":    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Vehicles endpoint
	mux.HandleFunc("/v2/vehicles", func(w http.ResponseWriter, r *http.Request) {
		vehicles := api.VehiclesResponse{
			Vehicles: []api.Vehicle{
				{
					VehicleID:    "test-1",
					VIN:          "5NPE24AF1KH123456",
					Nickname:     "Test Car",
					VehicleName:  "Hyundai",
					VehicleModel: "IONIQ 5",
					Year:         "2024",
					Color:        "Blue",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vehicles)
	})

	// Vehicle status endpoint
	mux.HandleFunc("/v2/vehicles/test-1/status", func(w http.ResponseWriter, r *http.Request) {
		status := &api.VehicleStatus{
			LastUpdateTime: time.Now(),
			OdometerStatus: api.OdometerStatus{
				Value: 12345,
				Unit:  "km",
			},
			EVStatus: &api.EVStatus{
				BatteryLevel:  85,
				BatteryCharge: true,
				RangeEV:       400.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	// Vehicle location endpoint
	mux.HandleFunc("/v2/vehicles/test-1/location", func(w http.ResponseWriter, r *http.Request) {
		location := &api.Location{
			Latitude:  37.7749,
			Longitude: -122.4194,
			Altitude:  0,
			Speed:     0,
			Heading:   0,
			Time:      time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(location)
	})

	return httptest.NewServer(mux)
}

// BenchmarkAPIClientCreation benchmarks client instantiation
func BenchmarkAPIClientCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = api.NewClient("US", "hyundai", "user", "pass", "1234", "")
	}
}

// BenchmarkCacheHit benchmarks cache hit performance
func BenchmarkCacheHit(b *testing.B) {
	// This benchmark would require modifying the API client to accept a custom base URL
	// or using dependency injection. For now, we'll skip the actual HTTP call
	// and focus on testing cache performance separately in cache_bench_test.go
	b.Skip("Requires API client refactoring to accept custom base URL")
}

// BenchmarkCacheMiss benchmarks cache miss performance
func BenchmarkCacheMiss(b *testing.B) {
	b.Skip("Requires API client refactoring to accept custom base URL")
}

// BenchmarkRateLimiterWait benchmarks rate limiter performance
func BenchmarkRateLimiterWait(b *testing.B) {
	// High requests per hour to minimize rate limit delays in benchmark
	client, err := api.NewClient("US", "hyundai", "user", "pass", "1234", "")
	if err != nil {
		b.Skip("Cannot create API client without valid credentials")
	}
	_ = client

	// We can't easily benchmark the rate limiter without making actual requests
	// because it's internal to the doRequest method
	b.Skip("Rate limiter is internal to doRequest - requires refactoring for direct benchmarking")
}

// BenchmarkJSONMarshal benchmarks JSON marshaling performance
func BenchmarkJSONMarshal(b *testing.B) {
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
		EVStatus: &api.EVStatus{
			BatteryLevel:  85,
			BatteryCharge: true,
			RangeEV:       400.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(status)
	}
}

// BenchmarkJSONUnmarshal benchmarks JSON unmarshaling performance
func BenchmarkJSONUnmarshal(b *testing.B) {
	data := []byte(`{
		"lastUpdateTime": "2024-01-01T00:00:00Z",
		"vehicleStatus": {
			"engine": false,
			"doorLock": true,
			"fuelLevel": 75,
			"battery": 12.6
		},
		"odometer": {
			"value": 12345,
			"unit": "km"
		},
		"evStatus": {
			"batteryLevel": 85,
			"batteryPlugin": true,
			"drvDistance": 400.0
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var status api.VehicleStatus
		_ = json.Unmarshal(data, &status)
	}
}

// BenchmarkHTTPServerResponse benchmarks mock server response time
func BenchmarkHTTPServerResponse(b *testing.B) {
	server := createTestAPIServer()
	defer server.Close()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL + "/v2/vehicles")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkContextCreation benchmarks context creation overhead
func BenchmarkContextCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		cancel()
		_ = ctx
	}
}

// BenchmarkStructAllocation benchmarks VehicleStatus struct allocation
func BenchmarkStructAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &api.VehicleStatus{
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
			EVStatus: &api.EVStatus{
				BatteryLevel:  85,
				BatteryCharge: true,
				RangeEV:       400.0,
			},
		}
	}
}
