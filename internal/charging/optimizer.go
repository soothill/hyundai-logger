// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package charging

import (
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
)

// ChargingPhase represents the current phase of charging
type ChargingPhase string

const (
	PhaseNotCharging  ChargingPhase = "not_charging"
	PhaseFastCharge   ChargingPhase = "fast_charge"  // >50 kW
	PhaseNormalCharge ChargingPhase = "normal_charge" // 7-50 kW
	PhaseTrickle      ChargingPhase = "trickle"       // <7 kW or >95% battery
	PhaseComplete     ChargingPhase = "complete"      // 100% battery

	// Charging efficiency factor (80% - typical for EV charging)
	// Accounts for energy loss during AC/DC conversion and battery heating
	chargingEfficiency = 0.8
)

// OptimizerConfig holds configuration for the charging optimizer
type OptimizerConfig struct {
	// Base intervals
	NotChargingInterval time.Duration // Default when not charging

	// Charging intervals by phase
	FastChargeInterval   time.Duration // High power charging (>50 kW)
	NormalChargeInterval time.Duration // Normal charging (7-50 kW)
	TrickleInterval      time.Duration // Trickle charge (<7 kW or >95%)
	CompleteInterval     time.Duration // Fully charged

	// Thresholds
	FastChargePowerKW    float64 // Power threshold for fast charging
	TrickleChargePowerKW float64 // Power threshold for trickle charging
	TrickleBatteryLevel  float64 // Battery level threshold for trickle
	CompleteBatteryLevel float64 // Battery level considered "complete"
}

// DefaultConfig returns sensible defaults for the charging optimizer
func DefaultConfig() OptimizerConfig {
	return OptimizerConfig{
		NotChargingInterval:  5 * time.Minute,
		FastChargeInterval:   1 * time.Minute,
		NormalChargeInterval: 2 * time.Minute,
		TrickleInterval:      5 * time.Minute,
		CompleteInterval:     10 * time.Minute,
		FastChargePowerKW:    50.0,
		TrickleChargePowerKW: 7.0,
		TrickleBatteryLevel:  95.0,
		CompleteBatteryLevel: 99.5,
	}
}

// Optimizer calculates optimal polling intervals based on charging state
type Optimizer struct {
	config OptimizerConfig
}

// New creates a new charging optimizer
func New(config OptimizerConfig) *Optimizer {
	return &Optimizer{
		config: config,
	}
}

// NewWithDefaults creates a new charging optimizer with default configuration
func NewWithDefaults() *Optimizer {
	return New(DefaultConfig())
}

// GetInterval calculates the optimal polling interval based on vehicle status
func (o *Optimizer) GetInterval(status *api.VehicleStatus) time.Duration {
	phase := o.DetectPhase(status)
	return o.GetIntervalForPhase(phase)
}

// DetectPhase determines the current charging phase
func (o *Optimizer) DetectPhase(status *api.VehicleStatus) ChargingPhase {
	// No EV status means not an EV or no data
	if status.EV == nil {
		return PhaseNotCharging
	}

	// Not charging
	if !status.EV.Charging {
		return PhaseNotCharging
	}

	// Check battery level first
	batteryLevel := status.EV.BatteryLevel

	// Complete - at or near 100%
	if batteryLevel >= o.config.CompleteBatteryLevel {
		return PhaseComplete
	}

	// Trickle - high battery level (>95%)
	if batteryLevel >= o.config.TrickleBatteryLevel {
		return PhaseTrickle
	}

	// Check charging power if available
	if status.EV.ChargingPower > 0 {
		// Fast charging - high power (>50 kW)
		if status.EV.ChargingPower >= o.config.FastChargePowerKW {
			return PhaseFastCharge
		}

		// Trickle - low power (<7 kW)
		if status.EV.ChargingPower < o.config.TrickleChargePowerKW {
			return PhaseTrickle
		}

		// Normal charging - medium power (7-50 kW)
		return PhaseNormalCharge
	}

	// Fallback: if charging but no power data, assume normal charge
	return PhaseNormalCharge
}

// GetIntervalForPhase returns the polling interval for a given charging phase
func (o *Optimizer) GetIntervalForPhase(phase ChargingPhase) time.Duration {
	switch phase {
	case PhaseNotCharging:
		return o.config.NotChargingInterval
	case PhaseFastCharge:
		return o.config.FastChargeInterval
	case PhaseNormalCharge:
		return o.config.NormalChargeInterval
	case PhaseTrickle:
		return o.config.TrickleInterval
	case PhaseComplete:
		return o.config.CompleteInterval
	default:
		return o.config.NotChargingInterval
	}
}

// GetPhaseDescription returns a human-readable description of the charging phase
func (o *Optimizer) GetPhaseDescription(phase ChargingPhase) string {
	switch phase {
	case PhaseNotCharging:
		return "Not charging"
	case PhaseFastCharge:
		return "Fast charging (>50 kW)"
	case PhaseNormalCharge:
		return "Normal charging (7-50 kW)"
	case PhaseTrickle:
		return "Trickle charging (<7 kW or >95%)"
	case PhaseComplete:
		return "Charging complete (≥100%)"
	default:
		return "Unknown"
	}
}

// Stats holds statistics about charging detection
type Stats struct {
	CurrentPhase    ChargingPhase
	BatteryLevel    float64
	ChargingPower   float64
	IsCharging      bool
	Interval        time.Duration
	PhaseDescription string
}

// GetStats returns current charging statistics
func (o *Optimizer) GetStats(status *api.VehicleStatus) Stats {
	phase := o.DetectPhase(status)
	interval := o.GetIntervalForPhase(phase)

	stats := Stats{
		CurrentPhase:     phase,
		Interval:         interval,
		PhaseDescription: o.GetPhaseDescription(phase),
	}

	if status.EV != nil {
		stats.BatteryLevel = status.EV.BatteryLevel
		stats.ChargingPower = status.EV.ChargingPower
		stats.IsCharging = status.EV.Charging
	}

	return stats
}

// EstimateTimeToFull estimates time remaining to full charge
// Returns 0 if not charging or no power data available
func (o *Optimizer) EstimateTimeToFull(status *api.VehicleStatus) time.Duration {
	if status.EV == nil || !status.EV.Charging {
		return 0
	}

	if status.EV.ChargingPower <= 0 {
		return 0 // No power data available
	}

	// Calculate remaining capacity
	batteryCapacityKWh := status.EV.BatteryCapacity
	if batteryCapacityKWh <= 0 {
		return 0 // No battery capacity data
	}

	remainingPercent := 100.0 - status.EV.BatteryLevel
	if remainingPercent <= 0 {
		return 0 // Already full
	}

	remainingKWh := batteryCapacityKWh * (remainingPercent / 100.0)

	// Time = Energy / Power (accounting for charging efficiency)
	hoursToFull := remainingKWh / (status.EV.ChargingPower * chargingEfficiency)

	return time.Duration(hoursToFull * float64(time.Hour))
}

// ShouldNotify determines if a notification should be sent based on charging state changes
func (o *Optimizer) ShouldNotify(oldPhase, newPhase ChargingPhase) bool {
	// Notify on any phase change except within same category
	if oldPhase == newPhase {
		return false
	}

	// Always notify when starting or stopping charging
	if oldPhase == PhaseNotCharging || newPhase == PhaseNotCharging {
		return true
	}

	// Notify when charging completes
	if newPhase == PhaseComplete {
		return true
	}

	// Notify when entering trickle mode from active charging
	if newPhase == PhaseTrickle && (oldPhase == PhaseFastCharge || oldPhase == PhaseNormalCharge) {
		return true
	}

	// Don't notify for other phase transitions (e.g., fast -> normal)
	return false
}
