// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package metrics

import (
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	collector := New()

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if collector.TotalPolls != 0 {
		t.Errorf("Expected 0 total polls, got %d", collector.TotalPolls)
	}

	if collector.VehiclesPollSuccess == nil {
		t.Error("Expected initialized VehiclesPollSuccess map")
	}

	if collector.VehiclesPollFailed == nil {
		t.Error("Expected initialized VehiclesPollFailed map")
	}

	if collector.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}
}

func TestRecordPollStart(t *testing.T) {
	collector := New()

	collector.RecordPollStart()
	if collector.TotalPolls != 1 {
		t.Errorf("Expected 1 total poll, got %d", collector.TotalPolls)
	}

	collector.RecordPollStart()
	if collector.TotalPolls != 2 {
		t.Errorf("Expected 2 total polls, got %d", collector.TotalPolls)
	}
}

func TestRecordPollComplete(t *testing.T) {
	collector := New()

	duration := 2 * time.Second

	// Successful poll
	collector.RecordPollComplete(duration, true)

	if collector.SuccessfulPolls != 1 {
		t.Errorf("Expected 1 successful poll, got %d", collector.SuccessfulPolls)
	}

	if collector.LastPollDuration != duration {
		t.Errorf("Expected duration %v, got %v", duration, collector.LastPollDuration)
	}

	if collector.ConsecutiveErrors != 0 {
		t.Errorf("Expected 0 consecutive errors, got %d", collector.ConsecutiveErrors)
	}

	// Failed poll
	collector.RecordPollComplete(duration, false)

	if collector.FailedPolls != 1 {
		t.Errorf("Expected 1 failed poll, got %d", collector.FailedPolls)
	}

	if collector.ConsecutiveErrors != 1 {
		t.Errorf("Expected 1 consecutive error, got %d", collector.ConsecutiveErrors)
	}
}

func TestAveragePollDuration(t *testing.T) {
	collector := New()

	// Must call RecordPollStart before RecordPollComplete
	collector.RecordPollStart()
	collector.RecordPollComplete(2*time.Second, true)

	collector.RecordPollStart()
	collector.RecordPollComplete(4*time.Second, true)

	if collector.AveragePollDuration != 3*time.Second {
		t.Errorf("Expected average 3s, got %v", collector.AveragePollDuration)
	}
}

func TestRecordAPICall(t *testing.T) {
	collector := New()

	latency := 100 * time.Millisecond

	// Successful API call
	collector.RecordAPICall(latency, true)

	if collector.APICallsTotal != 1 {
		t.Errorf("Expected 1 API call, got %d", collector.APICallsTotal)
	}

	if collector.APICallsSuccess != 1 {
		t.Errorf("Expected 1 successful API call, got %d", collector.APICallsSuccess)
	}

	if collector.LastAPICallTime.IsZero() {
		t.Error("Expected non-zero last API call time")
	}

	// Failed API call
	collector.RecordAPICall(latency, false)

	if collector.APICallsTotal != 2 {
		t.Errorf("Expected 2 API calls, got %d", collector.APICallsTotal)
	}

	if collector.APICallsFailed != 1 {
		t.Errorf("Expected 1 failed API call, got %d", collector.APICallsFailed)
	}
}

func TestRecordVehiclePoll(t *testing.T) {
	collector := New()

	vin := "TEST123VIN456"

	// Successful vehicle poll
	collector.RecordVehiclePoll(vin, true)

	if collector.VehiclesPollSuccess[vin] != 1 {
		t.Errorf("Expected 1 success for VIN, got %d", collector.VehiclesPollSuccess[vin])
	}

	// Failed vehicle poll
	collector.RecordVehiclePoll(vin, false)

	if collector.VehiclesPollFailed[vin] != 1 {
		t.Errorf("Expected 1 failure for VIN, got %d", collector.VehiclesPollFailed[vin])
	}
}

func TestRecordError(t *testing.T) {
	collector := New()

	testErr := "test error message"

	collector.RecordError(testErr)

	if collector.TotalErrors != 1 {
		t.Errorf("Expected 1 total error, got %d", collector.TotalErrors)
	}

	// RecordError only tracks TotalErrors, not ConsecutiveErrors
	// ConsecutiveErrors is tracked by RecordPollComplete with success=false
	if collector.LastError != testErr {
		t.Errorf("Expected last error '%s', got '%s'", testErr, collector.LastError)
	}

	if collector.LastErrorTime.IsZero() {
		t.Error("Expected non-zero last error time")
	}
}

func TestRecordChargingDetection(t *testing.T) {
	collector := New()

	collector.RecordChargingDetection()

	if collector.ChargingDetections != 1 {
		t.Errorf("Expected 1 charging detection, got %d", collector.ChargingDetections)
	}

	collector.RecordChargingDetection()

	if collector.ChargingDetections != 2 {
		t.Errorf("Expected 2 charging detections, got %d", collector.ChargingDetections)
	}
}

func TestSetTotalVehicles(t *testing.T) {
	collector := New()

	collector.SetTotalVehicles(3)

	if collector.TotalVehicles != 3 {
		t.Errorf("Expected 3 total vehicles, got %d", collector.TotalVehicles)
	}
}

func TestUpdateVehiclesCharging(t *testing.T) {
	collector := New()

	collector.UpdateVehiclesCharging(2)

	if collector.VehiclesCharging != 2 {
		t.Errorf("Expected 2 charging vehicles, got %d", collector.VehiclesCharging)
	}
}

func TestGetUptime(t *testing.T) {
	collector := New()

	// Sleep a bit to ensure uptime > 0
	time.Sleep(10 * time.Millisecond)

	stats := collector.GetStats()

	if stats.Uptime <= 0 {
		t.Errorf("Expected positive uptime, got %v", stats.Uptime)
	}
}

func TestSuccessRate(t *testing.T) {
	collector := New()

	// No polls yet - should be 0
	stats := collector.GetStats()
	if stats.PollSuccessRate != 0 {
		t.Errorf("Expected 0 success rate with no polls, got %f", stats.PollSuccessRate)
	}

	// 3 successful, 1 failed = 75%
	collector.TotalPolls = 4
	collector.SuccessfulPolls = 3
	collector.FailedPolls = 1

	stats = collector.GetStats()
	if stats.PollSuccessRate != 75.0 {
		t.Errorf("Expected 75.0 success rate, got %f", stats.PollSuccessRate)
	}

	// 100% success
	collector.TotalPolls = 10
	collector.SuccessfulPolls = 10
	collector.FailedPolls = 0

	stats = collector.GetStats()
	if stats.PollSuccessRate != 100.0 {
		t.Errorf("Expected 100.0 success rate, got %f", stats.PollSuccessRate)
	}
}

func TestConcurrentAccess(t *testing.T) {
	collector := New()

	done := make(chan bool)

	// Run concurrent operations
	for i := 0; i < 10; i++ {
		go func() {
			collector.RecordPollStart()
			collector.RecordPollComplete(time.Second, true)
			collector.RecordAPICall(100*time.Millisecond, true)
			collector.RecordVehiclePoll("TEST", true)
			collector.RecordError("test")
			collector.GetStats()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic - verify we can read stats
	if collector.TotalPolls != 10 {
		t.Errorf("Expected 10 polls, got %d", collector.TotalPolls)
	}
}
