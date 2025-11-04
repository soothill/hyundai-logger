// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package export

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMonthlyReport(t *testing.T) {
	report := MonthlyReport{
		Year:              2024,
		Month:             time.January,
		VIN:               "",
		TotalDistance:     1234.5,
		TotalChargingCost: 45.67,
		ChargingSessions:  12,
		TotalEnergyUsed:   234.5,
		AverageEfficiency: 19.0,
		Vehicles: map[string]VehicleStats{
			"TEST123": {
				Make:              "Hyundai",
				Model:             "Ioniq 5",
				VIN:               "TEST123",
				TotalDistance:     1234.5,
				TotalChargingCost: 45.67,
				ChargingSessions:  12,
				TotalEnergyUsed:   234.5,
				AverageEfficiency: 19.0,
			},
		},
	}

	if report.Year != 2024 {
		t.Errorf("expected year 2024, got %d", report.Year)
	}

	if report.Month != time.January {
		t.Errorf("expected month January, got %s", report.Month)
	}

	if report.TotalDistance != 1234.5 {
		t.Errorf("expected distance 1234.5, got %f", report.TotalDistance)
	}

	if report.ChargingSessions != 12 {
		t.Errorf("expected 12 sessions, got %d", report.ChargingSessions)
	}

	if len(report.Vehicles) != 1 {
		t.Errorf("expected 1 vehicle, got %d", len(report.Vehicles))
	}

	// Test JSON marshaling
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal MonthlyReport: %v", err)
	}

	var unmarshaled MonthlyReport
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal MonthlyReport: %v", err)
	}

	if unmarshaled.Year != report.Year {
		t.Error("Year not preserved in JSON roundtrip")
	}
}

func TestVehicleStats(t *testing.T) {
	stats := VehicleStats{
		Make:              "Hyundai",
		Model:             "Ioniq 5",
		VIN:               "TEST123",
		TotalDistance:     1234.5,
		TotalChargingCost: 45.67,
		ChargingSessions:  12,
		TotalEnergyUsed:   234.5,
		AverageEfficiency: 19.0,
	}

	if stats.Make != "Hyundai" {
		t.Errorf("expected make Hyundai, got %s", stats.Make)
	}

	if stats.Model != "Ioniq 5" {
		t.Errorf("expected model Ioniq 5, got %s", stats.Model)
	}

	if stats.VIN != "TEST123" {
		t.Errorf("expected VIN TEST123, got %s", stats.VIN)
	}

	if stats.AverageEfficiency != 19.0 {
		t.Errorf("expected efficiency 19.0, got %f", stats.AverageEfficiency)
	}

	// Test JSON marshaling
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal VehicleStats: %v", err)
	}

	var unmarshaled VehicleStats
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal VehicleStats: %v", err)
	}

	if unmarshaled.VIN != stats.VIN {
		t.Error("VIN not preserved in JSON roundtrip")
	}
}

func TestTripSummary(t *testing.T) {
	trip := TripSummary{
		StartTime:     time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
		EndTime:       time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
		Distance:      45.6,
		Duration:      "1h30m0s",
		StartOdometer: 12345.0,
		EndOdometer:   12390.6,
	}

	if trip.Distance != 45.6 {
		t.Errorf("expected distance 45.6, got %f", trip.Distance)
	}

	if trip.Duration != "1h30m0s" {
		t.Errorf("expected duration 1h30m0s, got %s", trip.Duration)
	}

	calculatedDistance := trip.EndOdometer - trip.StartOdometer
	tolerance := 0.001 // Allow small floating point errors
	diff := calculatedDistance - trip.Distance
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("odometer distance (%f) doesn't match trip distance (%f)", calculatedDistance, trip.Distance)
	}

	// Test JSON marshaling
	data, err := json.Marshal(trip)
	if err != nil {
		t.Fatalf("failed to marshal TripSummary: %v", err)
	}

	var unmarshaled TripSummary
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal TripSummary: %v", err)
	}

	if unmarshaled.Distance != trip.Distance {
		t.Error("Distance not preserved in JSON roundtrip")
	}
}

func TestChargingSession(t *testing.T) {
	session := ChargingSession{
		StartTime:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		EndTime:      time.Date(2024, 1, 1, 11, 30, 0, 0, time.UTC),
		Duration:     "1h30m0s",
		EnergyAdded:  32.5,
		Cost:         8.12,
		StartBattery: 20.0,
		EndBattery:   70.0,
		PeakPower:    150.0,
		AveragePower: 100.0,
	}

	if session.EnergyAdded != 32.5 {
		t.Errorf("expected energy 32.5, got %f", session.EnergyAdded)
	}

	if session.Cost != 8.12 {
		t.Errorf("expected cost 8.12, got %f", session.Cost)
	}

	batteryGain := session.EndBattery - session.StartBattery
	if batteryGain != 50.0 {
		t.Errorf("expected battery gain 50%%, got %f%%", batteryGain)
	}

	if session.PeakPower < session.AveragePower {
		t.Error("Peak power should be >= average power")
	}

	// Test JSON marshaling
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("failed to marshal ChargingSession: %v", err)
	}

	var unmarshaled ChargingSession
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ChargingSession: %v", err)
	}

	if unmarshaled.EnergyAdded != session.EnergyAdded {
		t.Error("EnergyAdded not preserved in JSON roundtrip")
	}
}

func TestNewReporter(t *testing.T) {
	// This test just verifies the constructor doesn't panic
	reporter := &Reporter{
		bucket:     "test-bucket",
		org:        "test-org",
		costPerKWh: 0.25,
	}

	if reporter.bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", reporter.bucket)
	}

	if reporter.org != "test-org" {
		t.Errorf("expected org test-org, got %s", reporter.org)
	}

	if reporter.costPerKWh != 0.25 {
		t.Errorf("expected costPerKWh 0.25, got %f", reporter.costPerKWh)
	}
}

func TestConvertTripsToDataPoints(t *testing.T) {
	trips := []TripSummary{
		{
			StartTime:     time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
			Distance:      45.6,
			Duration:      "1h30m0s",
			StartOdometer: 12345.0,
			EndOdometer:   12390.6,
		},
		{
			StartTime:     time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC),
			Distance:      30.2,
			Duration:      "1h0m0s",
			StartOdometer: 12390.6,
			EndOdometer:   12420.8,
		},
	}

	points := convertTripsToDataPoints(trips)

	if len(points) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(points))
	}

	// Verify first point
	if points[0].Timestamp != trips[0].StartTime {
		t.Error("First point timestamp doesn't match trip start time")
	}

	if points[0].Fields["distance_km"] != trips[0].Distance {
		t.Errorf("expected distance %f, got %v", trips[0].Distance, points[0].Fields["distance_km"])
	}

	if points[0].Fields["duration"] != trips[0].Duration {
		t.Errorf("expected duration %s, got %v", trips[0].Duration, points[0].Fields["duration"])
	}

	// Verify second point
	if points[1].Fields["start_odometer"] != trips[1].StartOdometer {
		t.Errorf("expected start odometer %f, got %v", trips[1].StartOdometer, points[1].Fields["start_odometer"])
	}
}

func TestEfficiencyCalculation(t *testing.T) {
	// Test realistic efficiency calculation
	stats := VehicleStats{
		TotalDistance:   500.0, // 500 km
		TotalEnergyUsed: 95.0,  // 95 kWh
	}

	// Calculate kWh per 100km
	efficiency := (stats.TotalEnergyUsed / stats.TotalDistance) * 100.0

	expectedEfficiency := 19.0 // 19 kWh/100km is typical for EVs
	if efficiency != expectedEfficiency {
		t.Errorf("expected efficiency %f kWh/100km, got %f", expectedEfficiency, efficiency)
	}

	stats.AverageEfficiency = efficiency

	// Verify efficiency is reasonable (10-25 kWh/100km for most EVs)
	if stats.AverageEfficiency < 10.0 || stats.AverageEfficiency > 25.0 {
		t.Errorf("efficiency %f kWh/100km seems unrealistic", stats.AverageEfficiency)
	}
}

func TestChargingCostCalculation(t *testing.T) {
	costPerKWh := 0.25 // $0.25 per kWh

	session := ChargingSession{
		EnergyAdded: 32.0, // 32 kWh
	}

	// Calculate cost
	session.Cost = session.EnergyAdded * costPerKWh

	expectedCost := 8.0 // $8.00
	if session.Cost != expectedCost {
		t.Errorf("expected cost $%f, got $%f", expectedCost, session.Cost)
	}
}

func TestMonthlyReportAggregation(t *testing.T) {
	// Simulate aggregating multiple vehicles
	vehicle1 := VehicleStats{
		VIN:               "VIN1",
		TotalDistance:     500.0,
		TotalChargingCost: 25.0,
		ChargingSessions:  5,
		TotalEnergyUsed:   95.0,
	}

	vehicle2 := VehicleStats{
		VIN:               "VIN2",
		TotalDistance:     300.0,
		TotalChargingCost: 15.0,
		ChargingSessions:  3,
		TotalEnergyUsed:   57.0,
	}

	report := MonthlyReport{
		Year:  2024,
		Month: time.January,
		Vehicles: map[string]VehicleStats{
			"VIN1": vehicle1,
			"VIN2": vehicle2,
		},
	}

	// Aggregate totals
	report.TotalDistance = vehicle1.TotalDistance + vehicle2.TotalDistance
	report.TotalChargingCost = vehicle1.TotalChargingCost + vehicle2.TotalChargingCost
	report.ChargingSessions = vehicle1.ChargingSessions + vehicle2.ChargingSessions
	report.TotalEnergyUsed = vehicle1.TotalEnergyUsed + vehicle2.TotalEnergyUsed

	if report.TotalDistance != 800.0 {
		t.Errorf("expected total distance 800, got %f", report.TotalDistance)
	}

	if report.TotalChargingCost != 40.0 {
		t.Errorf("expected total cost 40, got %f", report.TotalChargingCost)
	}

	if report.ChargingSessions != 8 {
		t.Errorf("expected 8 sessions, got %d", report.ChargingSessions)
	}

	if report.TotalEnergyUsed != 152.0 {
		t.Errorf("expected total energy 152, got %f", report.TotalEnergyUsed)
	}

	// Calculate overall efficiency
	if report.TotalDistance > 0 && report.TotalEnergyUsed > 0 {
		report.AverageEfficiency = (report.TotalEnergyUsed / report.TotalDistance) * 100.0
	}

	expectedEfficiency := 19.0
	if report.AverageEfficiency != expectedEfficiency {
		t.Errorf("expected efficiency %f, got %f", expectedEfficiency, report.AverageEfficiency)
	}
}

// Benchmark tests
func BenchmarkConvertTripsToDataPoints(b *testing.B) {
	trips := make([]TripSummary, 100)
	for i := 0; i < 100; i++ {
		trips[i] = TripSummary{
			StartTime:     time.Date(2024, 1, 1, 8, i, 0, 0, time.UTC),
			EndTime:       time.Date(2024, 1, 1, 9, i, 0, 0, time.UTC),
			Distance:      45.6,
			Duration:      "1h0m0s",
			StartOdometer: float64(12345 + i*45),
			EndOdometer:   float64(12390 + i*45),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = convertTripsToDataPoints(trips)
	}
}
