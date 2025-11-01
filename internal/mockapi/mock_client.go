// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package mockapi

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
)

// MockClient simulates a Hyundai Bluelink API client for development
type MockClient struct {
	config         MockConfig
	authenticated  bool
	vehicles       []api.Vehicle
	lastPollTime   time.Time
	batteryLevel   float64
	odometer       float64
	charging       bool
	chargingPower  float64
	location       api.LocationData
}

// MockConfig configures the mock client behavior
type MockConfig struct {
	NumVehicles        int           // Number of mock vehicles to simulate
	InitialBattery     float64       // Starting battery level (0-100)
	InitialOdometer    float64       // Starting odometer in km
	DrivingEnabled     bool          // Simulate driving (odometer increases)
	DrivingSpeedKmh    float64       // Average driving speed
	ChargingEnabled    bool          // Simulate charging
	ChargingPowerKW    float64       // Charging power in kW
	BatteryCapacityKWh float64       // Battery capacity in kWh
	RandomEvents       bool          // Add random charging/driving events
	AuthDelay          time.Duration // Simulated authentication delay
	PollDelay          time.Duration // Simulated polling delay
}

// DefaultMockConfig returns sensible defaults
func DefaultMockConfig() MockConfig {
	return MockConfig{
		NumVehicles:        1,
		InitialBattery:     75.0,
		InitialOdometer:    12345.0,
		DrivingEnabled:     true,
		DrivingSpeedKmh:    50.0,
		ChargingEnabled:    true,
		ChargingPowerKW:    50.0,
		BatteryCapacityKWh: 64.0,
		RandomEvents:       true,
		AuthDelay:          100 * time.Millisecond,
		PollDelay:          50 * time.Millisecond,
	}
}

// NewMockClient creates a new mock API client
func NewMockClient(config MockConfig) *MockClient {
	if config.NumVehicles == 0 {
		config.NumVehicles = 1
	}

	client := &MockClient{
		config:       config,
		batteryLevel: config.InitialBattery,
		odometer:     config.InitialOdometer,
		charging:     false,
		location: api.LocationData{
			Latitude:  37.5665, // Seoul, South Korea (Hyundai HQ)
			Longitude: 126.9780,
		},
	}

	// Generate mock vehicles
	client.vehicles = make([]api.Vehicle, config.NumVehicles)
	for i := 0; i < config.NumVehicles; i++ {
		client.vehicles[i] = api.Vehicle{
			VehicleID: fmt.Sprintf("mock-vehicle-%d", i+1),
			VIN:       fmt.Sprintf("MOCK%dVIN123456789%02d", i+1, i+1),
			Make:      "Hyundai",
			Model:     []string{"Ioniq 5", "Ioniq 6", "Kona Electric"}[i%3],
			Year:      2024,
			Nickname:  fmt.Sprintf("Mock Vehicle %d", i+1),
		}
	}

	return client
}

// Authenticate simulates API authentication
func (m *MockClient) Authenticate(ctx context.Context) error {
	// Simulate network delay
	select {
	case <-time.After(m.config.AuthDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	m.authenticated = true
	return nil
}

// GetVehicles returns mock vehicles
func (m *MockClient) GetVehicles(ctx context.Context) ([]api.Vehicle, error) {
	if !m.authenticated {
		return nil, fmt.Errorf("not authenticated")
	}

	// Simulate network delay
	select {
	case <-time.After(m.config.PollDelay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return m.vehicles, nil
}

// GetVehicleStatus returns simulated vehicle status
func (m *MockClient) GetVehicleStatus(ctx context.Context, vehicleID string) (*api.VehicleStatus, error) {
	if !m.authenticated {
		return nil, fmt.Errorf("not authenticated")
	}

	// Simulate network delay
	select {
	case <-time.After(m.config.PollDelay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Update simulated state
	m.updateState()

	// Find vehicle
	var vehicle *api.Vehicle
	for i := range m.vehicles {
		if m.vehicles[i].VehicleID == vehicleID {
			vehicle = &m.vehicles[i]
			break
		}
	}

	if vehicle == nil {
		return nil, fmt.Errorf("vehicle not found: %s", vehicleID)
	}

	// Build status
	status := &api.VehicleStatus{
		VIN:       vehicle.VIN,
		Timestamp: time.Now(),
		Odometer:  m.odometer,
		Location:  &m.location,
		EV: &api.EVStatus{
			BatteryLevel:    m.batteryLevel,
			Charging:        m.charging,
			ChargingPower:   m.chargingPower,
			BatteryCapacity: m.config.BatteryCapacityKWh,
			RangeKM:         m.calculateRange(),
			PluggedIn:       m.charging,
			ChargeEndTime:   time.Now().Add(m.calculateTimeToFull()),
		},
	}

	m.lastPollTime = time.Now()

	return status, nil
}

// updateState simulates vehicle state changes over time
func (m *MockClient) updateState() {
	now := time.Now()

	// First update - initialize last poll time
	if m.lastPollTime.IsZero() {
		m.lastPollTime = now
		return
	}

	elapsed := now.Sub(m.lastPollTime)

	// Random events
	if m.config.RandomEvents {
		// 5% chance to toggle charging state each poll
		if rand.Float64() < 0.05 {
			m.charging = !m.charging
		}
	}

	// Update based on state
	if m.charging && m.config.ChargingEnabled {
		// Simulate charging
		energyAdded := (m.config.ChargingPowerKW * elapsed.Hours()) // kWh
		batteryGain := (energyAdded / m.config.BatteryCapacityKWh) * 100.0

		m.batteryLevel += batteryGain
		if m.batteryLevel > 100.0 {
			m.batteryLevel = 100.0
			m.charging = false // Stop charging when full
		}

		m.chargingPower = m.config.ChargingPowerKW

		// Slow down near 100%
		if m.batteryLevel > 80.0 {
			m.chargingPower *= (100.0 - m.batteryLevel) / 20.0
			if m.chargingPower < 7.0 {
				m.chargingPower = 7.0 // Trickle charge
			}
		}
	} else if m.config.DrivingEnabled && !m.charging {
		// Simulate driving
		distance := m.config.DrivingSpeedKmh * elapsed.Hours()
		m.odometer += distance

		// Consume battery while driving
		// Assume 19 kWh/100km efficiency
		energyUsed := (distance / 100.0) * 19.0
		batteryLoss := (energyUsed / m.config.BatteryCapacityKWh) * 100.0

		m.batteryLevel -= batteryLoss
		if m.batteryLevel < 0.0 {
			m.batteryLevel = 0.0
		}

		m.chargingPower = 0.0

		// Start charging if battery is low
		if m.config.RandomEvents && m.batteryLevel < 30.0 {
			m.charging = true
		}

		// Update location (simulate movement)
		m.location.Latitude += (rand.Float64() - 0.5) * 0.01
		m.location.Longitude += (rand.Float64() - 0.5) * 0.01
	} else {
		m.chargingPower = 0.0
	}
}

// calculateRange estimates remaining range based on battery level
func (m *MockClient) calculateRange() float64 {
	// Assume 19 kWh/100km efficiency
	remainingKWh := (m.batteryLevel / 100.0) * m.config.BatteryCapacityKWh
	rangeKm := (remainingKWh / 19.0) * 100.0
	return rangeKm
}

// calculateTimeToFull estimates time to full charge
func (m *MockClient) calculateTimeToFull() time.Duration {
	if !m.charging || m.chargingPower <= 0 {
		return 0
	}

	remainingPercent := 100.0 - m.batteryLevel
	remainingKWh := (remainingPercent / 100.0) * m.config.BatteryCapacityKWh

	// Assume 80% charging efficiency
	hoursToFull := remainingKWh / (m.chargingPower * 0.8)

	return time.Duration(hoursToFull * float64(time.Hour))
}

// SetCharging manually sets charging state (for testing)
func (m *MockClient) SetCharging(charging bool) {
	m.charging = charging
}

// SetBatteryLevel manually sets battery level (for testing)
func (m *MockClient) SetBatteryLevel(level float64) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}
	m.batteryLevel = level
}

// SetOdometer manually sets odometer (for testing)
func (m *MockClient) SetOdometer(odometer float64) {
	m.odometer = odometer
}

// GetConfig returns the current mock configuration
func (m *MockClient) GetConfig() MockConfig {
	return m.config
}
