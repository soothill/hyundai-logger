// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestCollector_InitialState(t *testing.T) {
	c := New()

	stats := c.GetStats()

	if stats.TotalPolls != 0 {
		t.Errorf("expected 0 total polls, got %d", stats.TotalPolls)
	}
	if stats.SuccessfulPolls != 0 {
		t.Errorf("expected 0 successful polls, got %d", stats.SuccessfulPolls)
	}
	if stats.FailedPolls != 0 {
		t.Errorf("expected 0 failed polls, got %d", stats.FailedPolls)
	}
	if stats.PollSuccessRate != 0 {
		t.Errorf("expected 0%% success rate, got %.2f%%", stats.PollSuccessRate)
	}
}

func TestCollector_RecordPollComplete(t *testing.T) {
	c := New()

	// Record successful poll
	c.RecordPollStart()
	c.RecordPollComplete(2*time.Second, true)

	stats := c.GetStats()
	if stats.TotalPolls != 1 {
		t.Errorf("expected 1 total poll, got %d", stats.TotalPolls)
	}
	if stats.SuccessfulPolls != 1 {
		t.Errorf("expected 1 successful poll, got %d", stats.SuccessfulPolls)
	}
	if stats.FailedPolls != 0 {
		t.Errorf("expected 0 failed polls, got %d", stats.FailedPolls)
	}
	if stats.LastPollDuration != 2*time.Second {
		t.Errorf("expected 2s duration, got %v", stats.LastPollDuration)
	}
	if stats.PollSuccessRate != 100.0 {
		t.Errorf("expected 100%% success rate, got %.2f%%", stats.PollSuccessRate)
	}

	// Record failed poll
	c.RecordPollStart()
	c.RecordPollComplete(1*time.Second, false)

	stats = c.GetStats()
	if stats.TotalPolls != 2 {
		t.Errorf("expected 2 total polls, got %d", stats.TotalPolls)
	}
	if stats.SuccessfulPolls != 1 {
		t.Errorf("expected 1 successful poll, got %d", stats.SuccessfulPolls)
	}
	if stats.FailedPolls != 1 {
		t.Errorf("expected 1 failed poll, got %d", stats.FailedPolls)
	}
	if stats.LastPollDuration != 1*time.Second {
		t.Errorf("expected 1s duration, got %v", stats.LastPollDuration)
	}
	if stats.PollSuccessRate != 50.0 {
		t.Errorf("expected 50%% success rate, got %.2f%%", stats.PollSuccessRate)
	}
}

func TestCollector_AveragePollDuration(t *testing.T) {
	c := New()

	c.RecordPollStart()
	c.RecordPollComplete(1*time.Second, true)
	c.RecordPollStart()
	c.RecordPollComplete(3*time.Second, true)
	c.RecordPollStart()
	c.RecordPollComplete(2*time.Second, true)

	stats := c.GetStats()

	// Average should be (1 + 3 + 2) / 3 = 2 seconds
	expected := 2 * time.Second
	if stats.AveragePollDuration != expected {
		t.Errorf("expected average duration %v, got %v", expected, stats.AveragePollDuration)
	}
}

func TestCollector_RecordAPICall(t *testing.T) {
	c := New()

	// Record successful API call
	c.RecordAPICall(500*time.Millisecond, true)

	stats := c.GetStats()
	if stats.APICallsTotal != 1 {
		t.Errorf("expected 1 total API call, got %d", stats.APICallsTotal)
	}
	if stats.APICallsSuccess != 1 {
		t.Errorf("expected 1 successful API call, got %d", stats.APICallsSuccess)
	}
	if stats.APICallsFailed != 0 {
		t.Errorf("expected 0 failed API calls, got %d", stats.APICallsFailed)
	}
	if stats.APISuccessRate != 100.0 {
		t.Errorf("expected 100%% API success rate, got %.2f%%", stats.APISuccessRate)
	}

	// Record failed API call
	c.RecordAPICall(0, false)

	stats = c.GetStats()
	if stats.APICallsTotal != 2 {
		t.Errorf("expected 2 total API calls, got %d", stats.APICallsTotal)
	}
	if stats.APICallsSuccess != 1 {
		t.Errorf("expected 1 successful API call, got %d", stats.APICallsSuccess)
	}
	if stats.APICallsFailed != 1 {
		t.Errorf("expected 1 failed API call, got %d", stats.APICallsFailed)
	}
	if stats.APISuccessRate != 50.0 {
		t.Errorf("expected 50%% API success rate, got %.2f%%", stats.APISuccessRate)
	}
}

func TestCollector_AverageAPILatency(t *testing.T) {
	c := New()

	c.RecordAPICall(100*time.Millisecond, true)
	c.RecordAPICall(300*time.Millisecond, true)
	c.RecordAPICall(200*time.Millisecond, true)

	stats := c.GetStats()

	// Average should be (100 + 300 + 200) / 3 = 200ms
	expected := 200 * time.Millisecond
	if stats.AverageAPILatency != expected {
		t.Errorf("expected average latency %v, got %v", expected, stats.AverageAPILatency)
	}
}

func TestCollector_RecordError(t *testing.T) {
	c := New()

	// Record first error
	c.RecordError("first error")

	stats := c.GetStats()
	if stats.TotalErrors != 1 {
		t.Errorf("expected 1 total error, got %d", stats.TotalErrors)
	}
	if stats.LastError != "first error" {
		t.Errorf("expected 'first error', got '%s'", stats.LastError)
	}

	// Record second error
	c.RecordError("second error")

	stats = c.GetStats()
	if stats.TotalErrors != 2 {
		t.Errorf("expected 2 total errors, got %d", stats.TotalErrors)
	}
	if stats.LastError != "second error" {
		t.Errorf("expected 'second error', got '%s'", stats.LastError)
	}
}

func TestCollector_ConsecutiveErrorsClearsOnSuccess(t *testing.T) {
	c := New()

	// Simulate failed polls (which increment consecutive errors)
	c.RecordPollStart()
	c.RecordPollComplete(1*time.Second, false)
	c.RecordPollStart()
	c.RecordPollComplete(1*time.Second, false)

	stats := c.GetStats()
	if stats.ConsecutiveErrors != 2 {
		t.Errorf("expected 2 consecutive errors, got %d", stats.ConsecutiveErrors)
	}

	// Successful poll clears consecutive errors
	c.RecordPollStart()
	c.RecordPollComplete(1*time.Second, true)

	stats = c.GetStats()
	if stats.ConsecutiveErrors != 0 {
		t.Errorf("expected 0 consecutive errors after successful poll, got %d", stats.ConsecutiveErrors)
	}
}

func TestCollector_RecordVehiclePoll(t *testing.T) {
	c := New()

	// Record successful vehicle poll
	c.RecordVehiclePoll("VIN123", true)
	c.RecordVehiclePoll("VIN123", true)

	// Record failed vehicle poll
	c.RecordVehiclePoll("VIN123", false)

	// We can't get individual vehicle stats, but we can verify it doesn't panic
	stats := c.GetStats()
	if stats.TotalVehicles < 0 {
		t.Error("unexpected negative vehicle count")
	}
}

func TestCollector_RecordCharging(t *testing.T) {
	c := New()

	// Record charging detection
	c.RecordChargingDetection()

	stats := c.GetStats()
	if stats.ChargingDetections != 1 {
		t.Errorf("expected 1 charging detection, got %d", stats.ChargingDetections)
	}

	// Update currently charging count
	c.UpdateVehiclesCharging(1)

	stats = c.GetStats()
	if stats.VehiclesCharging != 1 {
		t.Errorf("expected 1 vehicle charging, got %d", stats.VehiclesCharging)
	}

	// Another detection
	c.RecordChargingDetection()

	stats = c.GetStats()
	if stats.ChargingDetections != 2 {
		t.Errorf("expected 2 charging detections, got %d", stats.ChargingDetections)
	}

	// Update charging count to 0
	c.UpdateVehiclesCharging(0)

	stats = c.GetStats()
	if stats.VehiclesCharging != 0 {
		t.Errorf("expected 0 vehicles charging, got %d", stats.VehiclesCharging)
	}
}

func TestCollector_SetTotalVehicles(t *testing.T) {
	c := New()

	c.SetTotalVehicles(3)

	stats := c.GetStats()
	if stats.TotalVehicles != 3 {
		t.Errorf("expected 3 total vehicles, got %d", stats.TotalVehicles)
	}
}

func TestCollector_Uptime(t *testing.T) {
	c := New()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	stats := c.GetStats()
	if stats.Uptime < 100*time.Millisecond {
		t.Errorf("expected uptime >= 100ms, got %v", stats.Uptime)
	}
	if stats.Uptime > 200*time.Millisecond {
		t.Errorf("expected uptime < 200ms, got %v", stats.Uptime)
	}
}

func TestCollector_FormatStats(t *testing.T) {
	c := New()

	// Add some data
	c.SetTotalVehicles(2)
	c.RecordPollStart()
	c.RecordPollComplete(2*time.Second, true)
	c.RecordAPICall(100*time.Millisecond, true)
	c.RecordChargingDetection()
	c.UpdateVehiclesCharging(1)

	stats := c.GetStats()
	formatted := FormatStats(stats)

	// Check for key sections
	expectedSections := []string{
		"Hyundai Logger Metrics",
		"Uptime:",
		"Poll Metrics:",
		"Vehicle Metrics:",
		"API Metrics:",
		"Error Metrics:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(formatted, section) {
			t.Errorf("expected formatted output to contain '%s'", section)
		}
	}

	// Check for specific values
	if !strings.Contains(formatted, "Total Vehicles: 2") {
		t.Error("expected 'Total Vehicles: 2' in formatted output")
	}
	if !strings.Contains(formatted, "Currently Charging: 1") {
		t.Error("expected 'Currently Charging: 1' in formatted output")
	}
}

func TestCollector_ThreadSafety(t *testing.T) {
	c := New()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				c.RecordPollStart()
				c.RecordPollComplete(1*time.Second, true)
				c.RecordAPICall(100*time.Millisecond, true)
				c.RecordError("test error")
				c.RecordVehiclePoll("VIN123", true)
			}
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				stats := c.GetStats()
				FormatStats(stats)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	stats := c.GetStats()
	// Just verify some data was recorded and didn't panic
	if stats.TotalPolls == 0 {
		t.Error("expected some polls to be recorded")
	}
}

func TestCollector_SuccessRateEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		successes    int64
		failures     int64
		expectedRate float64
	}{
		{"no data", 0, 0, 0.0},
		{"all success", 10, 0, 100.0},
		{"all failures", 0, 10, 0.0},
		{"half and half", 5, 5, 50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()

			for i := int64(0); i < tt.successes; i++ {
				c.RecordPollStart()
				c.RecordPollComplete(1*time.Second, true)
			}
			for i := int64(0); i < tt.failures; i++ {
				c.RecordPollStart()
				c.RecordPollComplete(1*time.Second, false)
			}

			stats := c.GetStats()
			if stats.PollSuccessRate != tt.expectedRate {
				t.Errorf("expected %.2f%% success rate, got %.2f%%", tt.expectedRate, stats.PollSuccessRate)
			}
		})
	}
}
