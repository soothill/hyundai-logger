// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package mockapi

import (
	"context"
	"testing"
	"time"
)

func TestDefaultMockConfig(t *testing.T) {
	config := DefaultMockConfig()

	if config.NumVehicles != 1 {
		t.Errorf("expected 1 vehicle, got %d", config.NumVehicles)
	}

	if config.InitialBattery != 75.0 {
		t.Errorf("expected initial battery 75%%, got %f%%", config.InitialBattery)
	}

	if config.ChargingPowerKW != 50.0 {
		t.Errorf("expected charging power 50kW, got %f", config.ChargingPowerKW)
	}

	if !config.RandomEvents {
		t.Error("expected random events enabled")
	}
}

func TestNewMockClient(t *testing.T) {
	config := DefaultMockConfig()
	config.NumVehicles = 3

	client := NewMockClient(config)

	if client == nil {
		t.Fatal("expected mock client, got nil")
	}

	if len(client.vehicles) != 3 {
		t.Errorf("expected 3 vehicles, got %d", len(client.vehicles))
	}

	// Verify vehicle data
	if client.vehicles[0].Make != "Hyundai" {
		t.Errorf("expected make Hyundai, got %s", client.vehicles[0].Make)
	}

	if client.vehicles[0].VIN == "" {
		t.Error("expected non-empty VIN")
	}
}

func TestAuthenticate(t *testing.T) {
	config := DefaultMockConfig()
	config.AuthDelay = 10 * time.Millisecond

	client := NewMockClient(config)
	ctx := context.Background()

	if client.authenticated {
		t.Error("client should not be authenticated initially")
	}

	err := client.Authenticate(ctx)
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	if !client.authenticated {
		t.Error("client should be authenticated after Authenticate()")
	}
}

func TestAuthenticateTimeout(t *testing.T) {
	config := DefaultMockConfig()
	config.AuthDelay = 1 * time.Second

	client := NewMockClient(config)

	// Create context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Authenticate(ctx)
	if err == nil {
		t.Error("expected timeout error")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestGetVehicles(t *testing.T) {
	config := DefaultMockConfig()
	config.NumVehicles = 2

	client := NewMockClient(config)
	ctx := context.Background()

	// Should fail before authentication
	_, err := client.GetVehicles(ctx)
	if err == nil {
		t.Error("expected error before authentication")
	}

	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	// Should succeed after authentication
	vehicles, err := client.GetVehicles(ctx)
	if err != nil {
		t.Fatalf("GetVehicles failed: %v", err)
	}

	if len(vehicles) != 2 {
		t.Errorf("expected 2 vehicles, got %d", len(vehicles))
	}

	// Verify vehicle structure
	if vehicles[0].VehicleID == "" {
		t.Error("expected non-empty vehicle ID")
	}

	if vehicles[0].Model == "" {
		t.Error("expected non-empty model")
	}
}

func TestGetVehicleStatus(t *testing.T) {
	config := DefaultMockConfig()
	config.InitialBattery = 80.0
	config.InitialOdometer = 10000.0

	client := NewMockClient(config)
	ctx := context.Background()

	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	vehicles, _ := client.GetVehicles(ctx)
	vehicleID := vehicles[0].VehicleID

	// Get status
	status, err := client.GetVehicleStatus(ctx, vehicleID)
	if err != nil {
		t.Fatalf("GetVehicleStatus failed: %v", err)
	}

	// Verify status
	if status.VIN == "" {
		t.Error("expected non-empty VIN")
	}

	if status.EV == nil {
		t.Fatal("expected EV status, got nil")
	}

	if status.EV.BatteryLevel != 80.0 {
		t.Errorf("expected battery 80%%, got %f%%", status.EV.BatteryLevel)
	}

	if status.Odometer != 10000.0 {
		t.Errorf("expected odometer 10000, got %f", status.Odometer)
	}

	if status.Location == nil {
		t.Error("expected location, got nil")
	}
}

func TestGetVehicleStatusInvalidID(t *testing.T) {
	client := NewMockClient(DefaultMockConfig())
	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	_, err := client.GetVehicleStatus(ctx, "invalid-id")
	if err == nil {
		t.Error("expected error for invalid vehicle ID")
	}
}

func TestChargingSimulation(t *testing.T) {
	config := DefaultMockConfig()
	config.InitialBattery = 50.0
	config.ChargingPowerKW = 100.0 // Fast charging
	config.BatteryCapacityKWh = 64.0
	config.RandomEvents = false // Disable random events

	client := NewMockClient(config)
	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	vehicles, _ := client.GetVehicles(ctx)
	vehicleID := vehicles[0].VehicleID

	// Start charging
	client.SetCharging(true)

	// Get initial status
	status1, _ := client.GetVehicleStatus(ctx, vehicleID)
	initialBattery := status1.EV.BatteryLevel

	// Wait a bit for charging simulation
	time.Sleep(100 * time.Millisecond)

	// Get updated status
	status2, _ := client.GetVehicleStatus(ctx, vehicleID)
	newBattery := status2.EV.BatteryLevel

	// Battery should have increased
	if newBattery <= initialBattery {
		t.Errorf("expected battery to increase, got %f -> %f", initialBattery, newBattery)
	}

	if !status2.EV.Charging {
		t.Error("expected charging to be true")
	}

	if status2.EV.ChargingPower <= 0 {
		t.Error("expected positive charging power")
	}
}

func TestDrivingSimulation(t *testing.T) {
	config := DefaultMockConfig()
	config.InitialBattery = 80.0
	config.InitialOdometer = 10000.0
	config.DrivingSpeedKmh = 100.0 // Fast driving for faster test
	config.DrivingEnabled = true
	config.RandomEvents = false

	client := NewMockClient(config)
	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	vehicles, _ := client.GetVehicles(ctx)
	vehicleID := vehicles[0].VehicleID

	// Ensure not charging
	client.SetCharging(false)

	// Get initial status
	status1, _ := client.GetVehicleStatus(ctx, vehicleID)
	initialOdometer := status1.Odometer
	initialBattery := status1.EV.BatteryLevel

	// Wait for driving simulation
	time.Sleep(100 * time.Millisecond)

	// Get updated status
	status2, _ := client.GetVehicleStatus(ctx, vehicleID)
	newOdometer := status2.Odometer
	newBattery := status2.EV.BatteryLevel

	// Odometer should have increased
	if newOdometer <= initialOdometer {
		t.Errorf("expected odometer to increase, got %f -> %f", initialOdometer, newOdometer)
	}

	// Battery should have decreased
	if newBattery >= initialBattery {
		t.Errorf("expected battery to decrease, got %f -> %f", initialBattery, newBattery)
	}

	if status2.EV.Charging {
		t.Error("expected charging to be false")
	}
}

func TestCalculateRange(t *testing.T) {
	config := DefaultMockConfig()
	config.BatteryCapacityKWh = 64.0

	client := NewMockClient(config)
	client.SetBatteryLevel(100.0)

	rangeKm := client.calculateRange()

	// 64 kWh / 19 kWh/100km * 100 = ~337 km
	expectedRange := (64.0 / 19.0) * 100.0

	tolerance := 1.0
	if rangeKm < expectedRange-tolerance || rangeKm > expectedRange+tolerance {
		t.Errorf("expected range ~%f km, got %f km", expectedRange, rangeKm)
	}
}

func TestCalculateTimeToFull(t *testing.T) {
	config := DefaultMockConfig()
	config.BatteryCapacityKWh = 64.0
	config.ChargingPowerKW = 50.0

	client := NewMockClient(config)
	client.SetBatteryLevel(50.0)
	client.SetCharging(true)
	client.chargingPower = 50.0

	timeToFull := client.calculateTimeToFull()

	// 50% of 64 kWh = 32 kWh remaining
	// 32 kWh / (50 kW * 0.8 efficiency) = 0.8 hours = 48 minutes
	expectedTime := 48 * time.Minute

	tolerance := 2 * time.Minute
	diff := timeToFull - expectedTime
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("expected time to full ~%v, got %v", expectedTime, timeToFull)
	}
}

func TestCalculateTimeToFullNotCharging(t *testing.T) {
	client := NewMockClient(DefaultMockConfig())
	client.SetBatteryLevel(50.0)
	client.SetCharging(false)

	timeToFull := client.calculateTimeToFull()

	if timeToFull != 0 {
		t.Errorf("expected 0 time when not charging, got %v", timeToFull)
	}
}

func TestSetMethods(t *testing.T) {
	client := NewMockClient(DefaultMockConfig())

	// Test SetCharging
	client.SetCharging(true)
	if !client.charging {
		t.Error("SetCharging(true) failed")
	}

	client.SetCharging(false)
	if client.charging {
		t.Error("SetCharging(false) failed")
	}

	// Test SetBatteryLevel
	client.SetBatteryLevel(85.5)
	if client.batteryLevel != 85.5 {
		t.Errorf("SetBatteryLevel(85.5) failed, got %f", client.batteryLevel)
	}

	// Test bounds
	client.SetBatteryLevel(150.0)
	if client.batteryLevel != 100.0 {
		t.Error("SetBatteryLevel should cap at 100")
	}

	client.SetBatteryLevel(-10.0)
	if client.batteryLevel != 0.0 {
		t.Error("SetBatteryLevel should floor at 0")
	}

	// Test SetOdometer
	client.SetOdometer(99999.9)
	if client.odometer != 99999.9 {
		t.Errorf("SetOdometer failed, got %f", client.odometer)
	}
}

func TestGetConfig(t *testing.T) {
	config := DefaultMockConfig()
	config.NumVehicles = 5
	config.ChargingPowerKW = 150.0

	client := NewMockClient(config)
	returnedConfig := client.GetConfig()

	if returnedConfig.NumVehicles != 5 {
		t.Errorf("expected 5 vehicles, got %d", returnedConfig.NumVehicles)
	}

	if returnedConfig.ChargingPowerKW != 150.0 {
		t.Errorf("expected 150kW, got %f", returnedConfig.ChargingPowerKW)
	}
}

func TestMultipleVehicles(t *testing.T) {
	config := DefaultMockConfig()
	config.NumVehicles = 3

	client := NewMockClient(config)
	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	vehicles, err := client.GetVehicles(ctx)
	if err != nil {
		t.Fatalf("GetVehicles failed: %v", err)
	}

	if len(vehicles) != 3 {
		t.Fatalf("expected 3 vehicles, got %d", len(vehicles))
	}

	// Each vehicle should have unique ID and VIN
	ids := make(map[string]bool)
	vins := make(map[string]bool)

	for _, vehicle := range vehicles {
		if ids[vehicle.VehicleID] {
			t.Errorf("duplicate vehicle ID: %s", vehicle.VehicleID)
		}
		ids[vehicle.VehicleID] = true

		if vins[vehicle.VIN] {
			t.Errorf("duplicate VIN: %s", vehicle.VIN)
		}
		vins[vehicle.VIN] = true

		// Should be able to get status for each vehicle
		_, err := client.GetVehicleStatus(ctx, vehicle.VehicleID)
		if err != nil {
			t.Errorf("failed to get status for vehicle %s: %v", vehicle.VehicleID, err)
		}
	}
}

func TestBatteryFullStopsCharging(t *testing.T) {
	config := DefaultMockConfig()
	config.RandomEvents = false
	config.ChargingPowerKW = 1000.0 // Very fast to reach 100% quickly

	client := NewMockClient(config)
	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	client.SetBatteryLevel(99.0)
	client.SetCharging(true)

	vehicles, _ := client.GetVehicles(ctx)
	vehicleID := vehicles[0].VehicleID

	// Poll a few times
	for i := 0; i < 5; i++ {
		time.Sleep(50 * time.Millisecond)
		status, _ := client.GetVehicleStatus(ctx, vehicleID)

		if status.EV.BatteryLevel >= 100.0 {
			// Should stop charging at 100%
			time.Sleep(100 * time.Millisecond)
			status2, _ := client.GetVehicleStatus(ctx, vehicleID)

			if status2.EV.Charging {
				t.Error("charging should stop at 100%")
			}
			return
		}
	}
}

// Benchmark tests
func BenchmarkGetVehicleStatus(b *testing.B) {
	client := NewMockClient(DefaultMockConfig())
	ctx := context.Background()
	client.Authenticate(ctx)
	vehicles, _ := client.GetVehicles(ctx)
	vehicleID := vehicles[0].VehicleID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetVehicleStatus(ctx, vehicleID)
	}
}

func BenchmarkUpdateState(b *testing.B) {
	client := NewMockClient(DefaultMockConfig())
	client.lastPollTime = time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.updateState()
	}
}
