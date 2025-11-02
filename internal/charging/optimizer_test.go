// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package charging

import (
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.NotChargingInterval != 5*time.Minute {
		t.Errorf("expected NotChargingInterval 5m, got %v", config.NotChargingInterval)
	}

	if config.FastChargeInterval != 1*time.Minute {
		t.Errorf("expected FastChargeInterval 1m, got %v", config.FastChargeInterval)
	}

	if config.FastChargePowerKW != 50.0 {
		t.Errorf("expected FastChargePowerKW 50, got %v", config.FastChargePowerKW)
	}
}

func TestNewWithDefaults(t *testing.T) {
	optimizer := NewWithDefaults()

	if optimizer == nil {
		t.Fatal("expected optimizer, got nil")
	}

	if optimizer.config.NotChargingInterval != 5*time.Minute {
		t.Error("default config not applied")
	}
}

func TestDetectPhase_NotCharging(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:     false,
			BatteryLevel: 75.0,
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseNotCharging {
		t.Errorf("expected PhaseNotCharging, got %v", phase)
	}
}

func TestDetectPhase_FastCharge(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   50.0,
			ChargingPower:  100.0, // >50 kW
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseFastCharge {
		t.Errorf("expected PhaseFastCharge, got %v", phase)
	}
}

func TestDetectPhase_NormalCharge(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   60.0,
			ChargingPower:  11.0, // 7-50 kW
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseNormalCharge {
		t.Errorf("expected PhaseNormalCharge, got %v", phase)
	}
}

func TestDetectPhase_TricklePower(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   80.0,
			ChargingPower:  3.0, // <7 kW
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseTrickle {
		t.Errorf("expected PhaseTrickle, got %v", phase)
	}
}

func TestDetectPhase_TrickleBattery(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   96.0, // >95%
			ChargingPower:  10.0,
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseTrickle {
		t.Errorf("expected PhaseTrickle, got %v", phase)
	}
}

func TestDetectPhase_Complete(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   100.0,
			ChargingPower:  2.0,
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseComplete {
		t.Errorf("expected PhaseComplete, got %v", phase)
	}
}

func TestDetectPhase_NoEVStatus(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: nil, // No EV data
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseNotCharging {
		t.Errorf("expected PhaseNotCharging for non-EV, got %v", phase)
	}
}

func TestDetectPhase_NoPowerData(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   70.0,
			ChargingPower:  0, // No power data
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseNormalCharge {
		t.Errorf("expected PhaseNormalCharge (fallback), got %v", phase)
	}
}

func TestGetIntervalForPhase(t *testing.T) {
	optimizer := NewWithDefaults()

	tests := []struct {
		phase    ChargingPhase
		expected time.Duration
	}{
		{PhaseNotCharging, 5 * time.Minute},
		{PhaseFastCharge, 1 * time.Minute},
		{PhaseNormalCharge, 2 * time.Minute},
		{PhaseTrickle, 5 * time.Minute},
		{PhaseComplete, 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			interval := optimizer.GetIntervalForPhase(tt.phase)
			if interval != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, interval)
			}
		})
	}
}

func TestGetInterval(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   50.0,
			ChargingPower:  100.0, // Fast charge
		},
	}

	interval := optimizer.GetInterval(status)
	if interval != 1*time.Minute {
		t.Errorf("expected 1m for fast charge, got %v", interval)
	}
}

func TestGetPhaseDescription(t *testing.T) {
	optimizer := NewWithDefaults()

	tests := []struct {
		phase       ChargingPhase
		expectedStr string
	}{
		{PhaseNotCharging, "Not charging"},
		{PhaseFastCharge, "Fast charging (>50 kW)"},
		{PhaseNormalCharge, "Normal charging (7-50 kW)"},
		{PhaseTrickle, "Trickle charging (<7 kW or >95%)"},
		{PhaseComplete, "Charging complete (≥100%)"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			desc := optimizer.GetPhaseDescription(tt.phase)
			if desc != tt.expectedStr {
				t.Errorf("expected %q, got %q", tt.expectedStr, desc)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	optimizer := NewWithDefaults()

	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   75.5,
			ChargingPower:  50.5,
		},
	}

	stats := optimizer.GetStats(status)

	if stats.CurrentPhase != PhaseFastCharge {
		t.Errorf("expected PhaseFastCharge, got %v", stats.CurrentPhase)
	}

	if stats.BatteryLevel != 75.5 {
		t.Errorf("expected battery level 75.5, got %v", stats.BatteryLevel)
	}

	if stats.ChargingPower != 50.5 {
		t.Errorf("expected charging power 50.5, got %v", stats.ChargingPower)
	}

	if !stats.IsCharging {
		t.Error("expected IsCharging to be true")
	}

	if stats.Interval != 1*time.Minute {
		t.Errorf("expected interval 1m, got %v", stats.Interval)
	}

	if stats.PhaseDescription != "Fast charging (>50 kW)" {
		t.Errorf("unexpected phase description: %s", stats.PhaseDescription)
	}
}

func TestEstimateTimeToFull(t *testing.T) {
	optimizer := NewWithDefaults()

	tests := []struct {
		name     string
		status   *api.VehicleStatus
		expected time.Duration
	}{
		{
			name: "Not charging",
			status: &api.VehicleStatus{
				EV: &api.EVStatus{
					Charging: false,
				},
			},
			expected: 0,
		},
		{
			name: "No power data",
			status: &api.VehicleStatus{
				EV: &api.EVStatus{
					Charging:       true,
					ChargingPower:  0,
				},
			},
			expected: 0,
		},
		{
			name: "Already full",
			status: &api.VehicleStatus{
				EV: &api.EVStatus{
					Charging:       true,
					BatteryLevel:   100.0,
					ChargingPower:  10.0,
				},
			},
			expected: 0,
		},
		{
			name: "50% charged, 64 kWh battery, 50 kW charging",
			status: &api.VehicleStatus{
				EV: &api.EVStatus{
					Charging:        true,
					BatteryLevel:    50.0,
					ChargingPower:   50.0,
					BatteryCapacity: 64.0,
				},
			},
			// Remaining: 32 kWh, Power: 50 kW, Efficiency: 0.8
			// Time: 32 / (50 * 0.8) = 0.8 hours = 48 minutes
			expected: 48 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate := optimizer.EstimateTimeToFull(tt.status)

			// Allow some tolerance for floating point math
			if estimate != 0 && tt.expected != 0 {
				diff := estimate - tt.expected
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Minute {
					t.Errorf("expected ~%v, got %v", tt.expected, estimate)
				}
			} else if estimate != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, estimate)
			}
		})
	}
}

func TestShouldNotify(t *testing.T) {
	optimizer := NewWithDefaults()

	tests := []struct {
		name     string
		oldPhase ChargingPhase
		newPhase ChargingPhase
		expected bool
	}{
		{"Same phase", PhaseFastCharge, PhaseFastCharge, false},
		{"Start charging", PhaseNotCharging, PhaseFastCharge, true},
		{"Stop charging", PhaseFastCharge, PhaseNotCharging, true},
		{"Charging complete", PhaseNormalCharge, PhaseComplete, true},
		{"Enter trickle from fast", PhaseFastCharge, PhaseTrickle, true},
		{"Enter trickle from normal", PhaseNormalCharge, PhaseTrickle, true},
		{"Fast to normal", PhaseFastCharge, PhaseNormalCharge, false},
		{"Normal to fast", PhaseNormalCharge, PhaseFastCharge, false},
		{"Trickle to complete", PhaseTrickle, PhaseComplete, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.ShouldNotify(tt.oldPhase, tt.newPhase)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCustomConfig(t *testing.T) {
	config := OptimizerConfig{
		NotChargingInterval:  10 * time.Minute,
		FastChargeInterval:   30 * time.Second,
		NormalChargeInterval: 1 * time.Minute,
		TrickleInterval:      3 * time.Minute,
		CompleteInterval:     15 * time.Minute,
		FastChargePowerKW:    100.0,
		TrickleChargePowerKW: 5.0,
		TrickleBatteryLevel:  90.0,
		CompleteBatteryLevel: 98.0,
	}

	optimizer := New(config)

	// Test with custom fast charge threshold
	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   60.0,
			ChargingPower:  75.0, // Would be fast with default, normal with custom
		},
	}

	phase := optimizer.DetectPhase(status)
	if phase != PhaseNormalCharge {
		t.Errorf("expected PhaseNormalCharge with custom threshold, got %v", phase)
	}

	interval := optimizer.GetInterval(status)
	if interval != 1*time.Minute {
		t.Errorf("expected 1m with custom config, got %v", interval)
	}
}

func BenchmarkDetectPhase(b *testing.B) {
	optimizer := NewWithDefaults()
	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   75.0,
			ChargingPower:  50.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.DetectPhase(status)
	}
}

func BenchmarkGetInterval(b *testing.B) {
	optimizer := NewWithDefaults()
	status := &api.VehicleStatus{
		EV: &api.EVStatus{
			Charging:       true,
			BatteryLevel:   75.0,
			ChargingPower:  50.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.GetInterval(status)
	}
}
