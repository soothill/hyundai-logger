// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package config

import (
	"testing"
)

func TestGetCurrentInterval(t *testing.T) {
	config := RateLimitConfig{
		PollIntervalMinutes: 30,
	}

	// Initialize lookup table
	for i := 0; i < 24; i++ {
		config.intervalByHour[i] = 30
	}

	// Override some hours
	config.intervalByHour[9] = 15  // 9 AM
	config.intervalByHour[17] = 10 // 5 PM

	tests := []struct {
		hour     int
		expected int
	}{
		{9, 15},
		{17, 10},
		{12, 30},
		{0, 30},
		{23, 30},
		{-1, 30},  // Invalid - should return default
		{24, 30},  // Invalid - should return default
	}

	for _, tc := range tests {
		result := config.GetCurrentInterval(tc.hour)
		if result != tc.expected {
			t.Errorf("Hour %d: expected %d, got %d", tc.hour, tc.expected, result)
		}
	}
}

func TestPrecomputeIntervals(t *testing.T) {
	config := &RateLimitConfig{
		PollIntervalMinutes: 30,
		Schedule: ScheduleConfig{
			Enabled: true,
			Periods: []PeriodConfig{
				{
					Name:            "Morning",
					StartHour:       6,
					EndHour:         12,
					IntervalMinutes: 10,
				},
				{
					Name:            "Evening",
					StartHour:       18,
					EndHour:         22,
					IntervalMinutes: 15,
				},
			},
		},
	}

	config.precomputeIntervals()

	// Check morning period (6-12)
	for hour := 6; hour < 12; hour++ {
		if config.intervalByHour[hour] != 10 {
			t.Errorf("Hour %d: expected 10, got %d", hour, config.intervalByHour[hour])
		}
	}

	// Check evening period (18-22)
	for hour := 18; hour < 22; hour++ {
		if config.intervalByHour[hour] != 15 {
			t.Errorf("Hour %d: expected 15, got %d", hour, config.intervalByHour[hour])
		}
	}

	// Check default period (other hours)
	if config.intervalByHour[0] != 30 {
		t.Errorf("Hour 0: expected 30, got %d", config.intervalByHour[0])
	}
	if config.intervalByHour[15] != 30 {
		t.Errorf("Hour 15: expected 30, got %d", config.intervalByHour[15])
	}
}

func TestPrecomputeIntervalsDisabled(t *testing.T) {
	config := &RateLimitConfig{
		PollIntervalMinutes: 30,
		Schedule: ScheduleConfig{
			Enabled: false,
			Periods: []PeriodConfig{
				{
					Name:            "Morning",
					StartHour:       6,
					EndHour:         12,
					IntervalMinutes: 10,
				},
			},
		},
	}

	config.precomputeIntervals()

	// All hours should be default when schedule is disabled
	for hour := 0; hour < 24; hour++ {
		if config.intervalByHour[hour] != 30 {
			t.Errorf("Hour %d: expected 30 (default), got %d", hour, config.intervalByHour[hour])
		}
	}
}

func TestChargingConfigIsEnabled(t *testing.T) {
	config := ChargingConfig{
		Enabled: true,
		IntervalMinutes: 5,
	}

	if !config.IsEnabled() {
		t.Error("Expected IsEnabled to return true")
	}

	config.Enabled = false
	if config.IsEnabled() {
		t.Error("Expected IsEnabled to return false")
	}
}

func TestChargingConfigGetIntervalMinutes(t *testing.T) {
	config := ChargingConfig{
		Enabled: true,
		IntervalMinutes: 7,
	}

	if config.GetIntervalMinutes() != 7 {
		t.Errorf("Expected 7, got %d", config.GetIntervalMinutes())
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "Valid config",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user@example.com",
					Password: "password",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
				},
				RateLimit: RateLimitConfig{
					PollIntervalMinutes: 30,
					RequestsPerHour:     200,
				},
				Database: DatabaseConfig{
					URL:          "http://localhost:8086",
					Token:        "token",
					Organization: "org",
					Bucket:       "bucket",
				},
			},
			shouldError: false,
		},
		{
			name: "Missing username",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "",
					Password: "password",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
				},
			},
			shouldError: true,
		},
		{
			name: "Missing database URL",
			config: Config{
				Hyundai: HyundaiConfig{
					Username: "user@example.com",
					Password: "password",
					PIN:      "1234",
					Brand:    "hyundai",
					Region:   "US",
				},
				Database: DatabaseConfig{
					URL: "",
				},
			},
			shouldError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPeriodOverlap(t *testing.T) {
	config := &RateLimitConfig{
		PollIntervalMinutes: 30,
		Schedule: ScheduleConfig{
			Enabled: true,
			Periods: []PeriodConfig{
				{
					Name:            "Period1",
					StartHour:       6,
					EndHour:         12,
					IntervalMinutes: 10,
				},
				{
					Name:            "Period2",
					StartHour:       10, // Overlaps with Period1
					EndHour:         14,
					IntervalMinutes: 5,
				},
			},
		},
	}

	config.precomputeIntervals()

	// Hour 10 and 11 are in both periods - last one wins
	if config.intervalByHour[10] != 5 {
		t.Errorf("Hour 10: expected 5 (Period2 wins), got %d", config.intervalByHour[10])
	}
	if config.intervalByHour[11] != 5 {
		t.Errorf("Hour 11: expected 5 (Period2 wins), got %d", config.intervalByHour[11])
	}
}
