// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package region

import (
	"strings"
	"testing"
)

func TestRegionConstants(t *testing.T) {
	if RegionNorthAmerica != "na" {
		t.Errorf("expected 'na', got %s", RegionNorthAmerica)
	}
	if RegionEurope != "eu" {
		t.Errorf("expected 'eu', got %s", RegionEurope)
	}
	if RegionKorea != "kr" {
		t.Errorf("expected 'kr', got %s", RegionKorea)
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		region       Region
		expectError  bool
		expectedURL  string
		expectedUnit string
	}{
		{RegionNorthAmerica, false, "telematics.hyundaiusa.com", "imperial"},
		{RegionEurope, false, "eu.bluelinky.com", "metric"},
		{RegionKorea, false, "kr.bluelinky.com", "metric"},
		{RegionChina, false, "cn.bluelinky.com", "metric"},
		{RegionAustralia, false, "au.bluelinky.com", "metric"},
		{RegionJapan, false, "jp.bluelinky.com", "metric"},
		{RegionIndia, false, "in.bluelinky.com", "metric"},
		{RegionBrazil, false, "br.bluelinky.com", "metric"},
		{"invalid", true, "", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			config, err := GetConfig(tt.region)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(config.APIBaseURL, tt.expectedURL) {
				t.Errorf("expected URL to contain %s, got %s", tt.expectedURL, config.APIBaseURL)
			}

			if config.MeasurementUnits != tt.expectedUnit {
				t.Errorf("expected units %s, got %s", tt.expectedUnit, config.MeasurementUnits)
			}

			if config.Region != tt.region {
				t.Errorf("expected region %s, got %s", tt.region, config.Region)
			}
		})
	}
}

func TestGetConfigOrDefault(t *testing.T) {
	// Valid region
	config := GetConfigOrDefault(RegionEurope)
	if config.Region != RegionEurope {
		t.Errorf("expected Europe, got %s", config.Region)
	}

	// Invalid region - should return North America
	config = GetConfigOrDefault("invalid")
	if config.Region != RegionNorthAmerica {
		t.Errorf("expected North America as default, got %s", config.Region)
	}
}

func TestParseRegion(t *testing.T) {
	tests := []struct {
		input    string
		expected Region
		hasError bool
	}{
		{"na", RegionNorthAmerica, false},
		{"NA", RegionNorthAmerica, false},
		{"  na  ", RegionNorthAmerica, false},
		{"usa", RegionNorthAmerica, false},
		{"us", RegionNorthAmerica, false},
		{"canada", RegionNorthAmerica, false},
		{"north america", RegionNorthAmerica, false},
		{"eu", RegionEurope, false},
		{"europe", RegionEurope, false},
		{"uk", RegionEurope, false},
		{"kr", RegionKorea, false},
		{"korea", RegionKorea, false},
		{"south korea", RegionKorea, false},
		{"cn", RegionChina, false},
		{"china", RegionChina, false},
		{"au", RegionAustralia, false},
		{"australia", RegionAustralia, false},
		{"jp", RegionJapan, false},
		{"japan", RegionJapan, false},
		{"in", RegionIndia, false},
		{"india", RegionIndia, false},
		{"br", RegionBrazil, false},
		{"brazil", RegionBrazil, false},
		{"brasil", RegionBrazil, false},
		{"invalid", Region(""), true},
		{"", Region(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			region, err := ParseRegion(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if region != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, region)
			}
		})
	}
}

func TestRegionString(t *testing.T) {
	region := RegionKorea
	if region.String() != "kr" {
		t.Errorf("expected 'kr', got %s", region.String())
	}
}

func TestRegionDisplayName(t *testing.T) {
	tests := []struct {
		region   Region
		expected string
	}{
		{RegionNorthAmerica, "North America"},
		{RegionEurope, "Europe"},
		{RegionKorea, "South Korea"},
		{RegionChina, "China"},
		{RegionAustralia, "Australia"},
		{RegionJapan, "Japan"},
		{RegionIndia, "India"},
		{RegionBrazil, "Brazil"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			name := tt.region.DisplayName()
			if name != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, name)
			}
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	regions := SupportedRegions()

	if len(regions) != 8 {
		t.Errorf("expected 8 regions, got %d", len(regions))
	}

	// Check that all expected regions are present
	regionMap := make(map[Region]bool)
	for _, region := range regions {
		regionMap[region] = true
	}

	expectedRegions := []Region{
		RegionNorthAmerica,
		RegionEurope,
		RegionKorea,
		RegionChina,
		RegionAustralia,
		RegionJapan,
		RegionIndia,
		RegionBrazil,
	}

	for _, expected := range expectedRegions {
		if !regionMap[expected] {
			t.Errorf("expected region %s not found in supported regions", expected)
		}
	}
}

func TestIsMetric(t *testing.T) {
	tests := []struct {
		region   Region
		expected bool
	}{
		{RegionNorthAmerica, false}, // US uses imperial
		{RegionEurope, true},
		{RegionKorea, true},
		{RegionChina, true},
		{RegionAustralia, true},
		{RegionJapan, true},
		{RegionIndia, true},
		{RegionBrazil, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			isMetric := tt.region.IsMetric()
			if isMetric != tt.expected {
				t.Errorf("expected IsMetric=%v, got %v", tt.expected, isMetric)
			}
		})
	}
}

func TestConvertDistance(t *testing.T) {
	// Test metric region
	value, unit := RegionEurope.ConvertDistance(100.0)
	if value != 100.0 {
		t.Errorf("expected 100, got %f", value)
	}
	if unit != "km" {
		t.Errorf("expected km, got %s", unit)
	}

	// Test imperial region
	value, unit = RegionNorthAmerica.ConvertDistance(100.0)
	expectedMiles := 100.0 * 0.621371
	tolerance := 0.01
	if value < expectedMiles-tolerance || value > expectedMiles+tolerance {
		t.Errorf("expected ~%f miles, got %f", expectedMiles, value)
	}
	if unit != "mi" {
		t.Errorf("expected mi, got %s", unit)
	}
}

func TestConvertTemperature(t *testing.T) {
	// Test metric region
	value, unit := RegionEurope.ConvertTemperature(25.0)
	if value != 25.0 {
		t.Errorf("expected 25, got %f", value)
	}
	if unit != "°C" {
		t.Errorf("expected °C, got %s", unit)
	}

	// Test imperial region
	value, unit = RegionNorthAmerica.ConvertTemperature(25.0)
	expectedF := 77.0 // 25°C = 77°F
	if value != expectedF {
		t.Errorf("expected %f°F, got %f", expectedF, value)
	}
	if unit != "°F" {
		t.Errorf("expected °F, got %s", unit)
	}

	// Test freezing point
	value, _ = RegionNorthAmerica.ConvertTemperature(0.0)
	if value != 32.0 {
		t.Errorf("expected 32°F, got %f", value)
	}
}

func TestNewValidator(t *testing.T) {
	// Valid region
	validator, err := NewValidator(RegionEurope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("expected validator, got nil")
	}

	// Invalid region
	_, err = NewValidator("invalid")
	if err == nil {
		t.Error("expected error for invalid region")
	}
}

func TestValidateURL(t *testing.T) {
	validator, _ := NewValidator(RegionEurope)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://api.eu.bluelinky.com/v1/vehicles", true},
		{"https://auth.eu.bluelinky.com/oauth/token", true},
		{"https://api.eu.BLUELINKY.com/v1/vehicles", true}, // Case insensitive
		{"https://api.na.bluelinky.com/v1/vehicles", false},
		{"https://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			valid := validator.ValidateURL(tt.url)
			if valid != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, valid)
			}
		})
	}
}

func TestGetRegionFromURL(t *testing.T) {
	tests := []struct {
		url          string
		expectedRegion Region
		expectError  bool
	}{
		{"https://api.telematics.hyundaiusa.com/v1/vehicles", RegionNorthAmerica, false},
		{"https://api.eu.bluelinky.com/v1/vehicles", RegionEurope, false},
		{"https://auth.kr.bluelinky.com/oauth/token", RegionKorea, false},
		{"https://api.cn.bluelinky.com/v1/vehicles", RegionChina, false},
		{"https://api.au.bluelinky.com/v1/vehicles", RegionAustralia, false},
		{"https://api.jp.bluelinky.com/v1/vehicles", RegionJapan, false},
		{"https://api.in.bluelinky.com/v1/vehicles", RegionIndia, false},
		{"https://api.br.bluelinky.com/v1/vehicles", RegionBrazil, false},
		{"https://example.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			region, err := GetRegionFromURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if region != tt.expectedRegion {
				t.Errorf("expected %s, got %s", tt.expectedRegion, region)
			}
		})
	}
}

func TestAllRegionsHaveConfig(t *testing.T) {
	// Ensure all defined region constants have a config
	testRegions := []Region{
		RegionNorthAmerica,
		RegionEurope,
		RegionKorea,
		RegionChina,
		RegionAustralia,
		RegionJapan,
		RegionIndia,
		RegionBrazil,
	}

	for _, region := range testRegions {
		config, err := GetConfig(region)
		if err != nil {
			t.Errorf("region %s missing configuration: %v", region, err)
		}

		// Verify required fields
		if config.APIBaseURL == "" {
			t.Errorf("region %s missing APIBaseURL", region)
		}
		if config.AuthBaseURL == "" {
			t.Errorf("region %s missing AuthBaseURL", region)
		}
		if config.Locale == "" {
			t.Errorf("region %s missing Locale", region)
		}
		if config.Language == "" {
			t.Errorf("region %s missing Language", region)
		}
		if config.Timezone == "" {
			t.Errorf("region %s missing Timezone", region)
		}
		if config.MeasurementUnits == "" {
			t.Errorf("region %s missing MeasurementUnits", region)
		}
	}
}

// Benchmark tests
func BenchmarkGetConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetConfig(RegionNorthAmerica)
	}
}

func BenchmarkParseRegion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseRegion("north america")
	}
}

func BenchmarkConvertDistance(b *testing.B) {
	region := RegionNorthAmerica
	for i := 0; i < b.N; i++ {
		_, _ = region.ConvertDistance(100.0)
	}
}
