// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package region

import (
	"fmt"
	"strings"
)

// Region represents a geographic region for API access
type Region string

const (
	RegionNorthAmerica Region = "na"   // North America (US, Canada)
	RegionEurope       Region = "eu"   // Europe
	RegionKorea        Region = "kr"   // South Korea
	RegionChina        Region = "cn"   // China
	RegionAustralia    Region = "au"   // Australia
	RegionJapan        Region = "jp"   // Japan
	RegionIndia        Region = "in"   // India
	RegionBrazil       Region = "br"   // Brazil
)

// Config contains region-specific configuration
type Config struct {
	Region           Region
	APIBaseURL       string
	AuthBaseURL      string
	Locale           string
	Language         string
	Timezone         string
	MeasurementUnits string // "metric" or "imperial"
}

// Default configurations for each region
var defaultConfigs = map[Region]Config{
	RegionNorthAmerica: {
		Region:           RegionNorthAmerica,
		APIBaseURL:       "https://api.telematics.hyundaiusa.com",
		AuthBaseURL:      "https://auth.telematics.hyundaiusa.com",
		Locale:           "en_US",
		Language:         "en",
		Timezone:         "America/New_York",
		MeasurementUnits: "imperial",
	},
	RegionEurope: {
		Region:           RegionEurope,
		APIBaseURL:       "https://api.eu.bluelinky.com",
		AuthBaseURL:      "https://auth.eu.bluelinky.com",
		Locale:           "en_GB",
		Language:         "en",
		Timezone:         "Europe/London",
		MeasurementUnits: "metric",
	},
	RegionKorea: {
		Region:           RegionKorea,
		APIBaseURL:       "https://api.kr.bluelinky.com",
		AuthBaseURL:      "https://auth.kr.bluelinky.com",
		Locale:           "ko_KR",
		Language:         "ko",
		Timezone:         "Asia/Seoul",
		MeasurementUnits: "metric",
	},
	RegionChina: {
		Region:           RegionChina,
		APIBaseURL:       "https://api.cn.bluelinky.com",
		AuthBaseURL:      "https://auth.cn.bluelinky.com",
		Locale:           "zh_CN",
		Language:         "zh",
		Timezone:         "Asia/Shanghai",
		MeasurementUnits: "metric",
	},
	RegionAustralia: {
		Region:           RegionAustralia,
		APIBaseURL:       "https://api.au.bluelinky.com",
		AuthBaseURL:      "https://auth.au.bluelinky.com",
		Locale:           "en_AU",
		Language:         "en",
		Timezone:         "Australia/Sydney",
		MeasurementUnits: "metric",
	},
	RegionJapan: {
		Region:           RegionJapan,
		APIBaseURL:       "https://api.jp.bluelinky.com",
		AuthBaseURL:      "https://auth.jp.bluelinky.com",
		Locale:           "ja_JP",
		Language:         "ja",
		Timezone:         "Asia/Tokyo",
		MeasurementUnits: "metric",
	},
	RegionIndia: {
		Region:           RegionIndia,
		APIBaseURL:       "https://api.in.bluelinky.com",
		AuthBaseURL:      "https://auth.in.bluelinky.com",
		Locale:           "en_IN",
		Language:         "en",
		Timezone:         "Asia/Kolkata",
		MeasurementUnits: "metric",
	},
	RegionBrazil: {
		Region:           RegionBrazil,
		APIBaseURL:       "https://api.br.bluelinky.com",
		AuthBaseURL:      "https://auth.br.bluelinky.com",
		Locale:           "pt_BR",
		Language:         "pt",
		Timezone:         "America/Sao_Paulo",
		MeasurementUnits: "metric",
	},
}

// GetConfig returns the configuration for a region
func GetConfig(region Region) (Config, error) {
	config, exists := defaultConfigs[region]
	if !exists {
		return Config{}, fmt.Errorf("unknown region: %s", region)
	}
	return config, nil
}

// GetConfigOrDefault returns the configuration for a region, or North America if not found
func GetConfigOrDefault(region Region) Config {
	config, err := GetConfig(region)
	if err != nil {
		return defaultConfigs[RegionNorthAmerica]
	}
	return config
}

// ParseRegion parses a region string into a Region type
func ParseRegion(s string) (Region, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Try direct match
	region := Region(s)
	if _, exists := defaultConfigs[region]; exists {
		return region, nil
	}

	// Try aliases
	aliases := map[string]Region{
		"north america":  RegionNorthAmerica,
		"northamerica":   RegionNorthAmerica,
		"usa":            RegionNorthAmerica,
		"us":             RegionNorthAmerica,
		"canada":         RegionNorthAmerica,
		"ca":             RegionNorthAmerica,
		"europe":         RegionEurope,
		"uk":             RegionEurope,
		"gb":             RegionEurope,
		"de":             RegionEurope,
		"fr":             RegionEurope,
		"korea":          RegionKorea,
		"south korea":    RegionKorea,
		"china":          RegionChina,
		"australia":      RegionAustralia,
		"japan":          RegionJapan,
		"india":          RegionIndia,
		"brasil":         RegionBrazil,
		"brazil":         RegionBrazil,
	}

	if region, exists := aliases[s]; exists {
		return region, nil
	}

	return "", fmt.Errorf("unknown region: %s", s)
}

// String returns the string representation of a region
func (r Region) String() string {
	return string(r)
}

// DisplayName returns a human-readable name for the region
func (r Region) DisplayName() string {
	names := map[Region]string{
		RegionNorthAmerica: "North America",
		RegionEurope:       "Europe",
		RegionKorea:        "South Korea",
		RegionChina:        "China",
		RegionAustralia:    "Australia",
		RegionJapan:        "Japan",
		RegionIndia:        "India",
		RegionBrazil:       "Brazil",
	}

	if name, exists := names[r]; exists {
		return name
	}
	return string(r)
}

// SupportedRegions returns a list of all supported regions
func SupportedRegions() []Region {
	regions := make([]Region, 0, len(defaultConfigs))
	for region := range defaultConfigs {
		regions = append(regions, region)
	}
	return regions
}

// IsMetric returns true if the region uses metric units
func (r Region) IsMetric() bool {
	config, err := GetConfig(r)
	if err != nil {
		return true // Default to metric
	}
	return config.MeasurementUnits == "metric"
}

// ConvertDistance converts distance based on region's measurement system
func (r Region) ConvertDistance(km float64) (value float64, unit string) {
	if r.IsMetric() {
		return km, "km"
	}
	// Convert to miles
	miles := km * 0.621371
	return miles, "mi"
}

// ConvertTemperature converts temperature based on region's measurement system
func (r Region) ConvertTemperature(celsius float64) (value float64, unit string) {
	if r.IsMetric() {
		return celsius, "°C"
	}
	// Convert to Fahrenheit
	fahrenheit := (celsius * 9.0 / 5.0) + 32.0
	return fahrenheit, "°F"
}

// Validator checks if region configuration is valid
type Validator struct {
	config Config
}

// NewValidator creates a new region validator
func NewValidator(region Region) (*Validator, error) {
	config, err := GetConfig(region)
	if err != nil {
		return nil, err
	}
	return &Validator{config: config}, nil
}

// ValidateURL checks if a URL is valid for this region
func (v *Validator) ValidateURL(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, strings.ToLower(v.config.APIBaseURL)) ||
		strings.Contains(url, strings.ToLower(v.config.AuthBaseURL))
}

// GetRegionFromURL attempts to determine the region from a URL
func GetRegionFromURL(url string) (Region, error) {
	url = strings.ToLower(url)

	for region, config := range defaultConfigs {
		if strings.Contains(url, strings.ToLower(config.APIBaseURL)) ||
			strings.Contains(url, strings.ToLower(config.AuthBaseURL)) {
			return region, nil
		}
	}

	return "", fmt.Errorf("cannot determine region from URL: %s", url)
}
