// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatConstants(t *testing.T) {
	if FormatCSV != "csv" {
		t.Errorf("expected FormatCSV to be 'csv', got %s", FormatCSV)
	}
	if FormatJSON != "json" {
		t.Errorf("expected FormatJSON to be 'json', got %s", FormatJSON)
	}
}

func TestBuildQuery(t *testing.T) {
	exporter := &Exporter{
		bucket: "test-bucket",
		org:    "test-org",
	}

	tests := []struct {
		name     string
		opts     ExportOptions
		contains []string
	}{
		{
			name: "Basic query",
			opts: ExportOptions{
				StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
				Measurement: "vehicle_status",
			},
			contains: []string{
				`from(bucket: "test-bucket")`,
				"range(start:",
				"2024-01-01T00:00:00Z",
				"2024-01-31T23:59:59Z",
				`r._measurement == "vehicle_status"`,
				"pivot",
			},
		},
		{
			name: "Query with VIN filter",
			opts: ExportOptions{
				StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Measurement: "vehicle_status",
				VIN:         "TEST123",
			},
			contains: []string{
				`r.vin == "TEST123"`,
			},
		},
		{
			name: "Query with field filter",
			opts: ExportOptions{
				StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Measurement: "vehicle_status",
				Fields:      []string{"battery_level", "odometer"},
			},
			contains: []string{
				`r._field == "battery_level"`,
				`r._field == "odometer"`,
			},
		},
		{
			name: "Query with aggregation",
			opts: ExportOptions{
				StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Measurement: "vehicle_status",
				GroupBy:     5 * time.Minute,
			},
			contains: []string{
				"aggregateWindow",
				"every: 5m0s",
				"fn: mean",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := exporter.buildQuery(tt.opts)

			for _, expected := range tt.contains {
				if !strings.Contains(query, expected) {
					t.Errorf("query does not contain %q\nQuery:\n%s", expected, query)
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		value    string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"test"}, "test", true},
	}

	for _, tt := range tests {
		result := contains(tt.slice, tt.value)
		if result != tt.expected {
			t.Errorf("contains(%v, %q) = %v, expected %v", tt.slice, tt.value, result, tt.expected)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	exporter := &Exporter{}

	points := []DataPoint{
		{
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 75.5,
				"odometer":      12345.6,
			},
		},
		{
			Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 74.2,
				"odometer":      12350.1,
			},
		},
	}

	var buf bytes.Buffer
	err := exporter.writeJSON(&buf, points)
	if err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	// Verify valid JSON
	var result []DataPoint
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 data points, got %d", len(result))
	}

	// Verify first point
	if result[0].VIN != "TEST123" {
		t.Errorf("expected VIN TEST123, got %s", result[0].VIN)
	}

	if result[0].Fields["battery_level"] != 75.5 {
		t.Errorf("expected battery_level 75.5, got %v", result[0].Fields["battery_level"])
	}
}

func TestWriteCSV(t *testing.T) {
	exporter := &Exporter{}

	points := []DataPoint{
		{
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 75.5,
				"odometer":      12345.6,
			},
		},
		{
			Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 74.2,
				"odometer":      12350.1,
			},
		},
	}

	var buf bytes.Buffer
	opts := ExportOptions{Format: FormatCSV}
	err := exporter.writeCSV(&buf, points, opts)
	if err != nil {
		t.Fatalf("writeCSV failed: %v", err)
	}

	output := buf.String()

	// Verify header
	if !strings.Contains(output, "timestamp") {
		t.Error("CSV output missing timestamp header")
	}
	if !strings.Contains(output, "vin") {
		t.Error("CSV output missing vin header")
	}

	// Verify data rows
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// Verify first data row contains VIN
	if !strings.Contains(lines[1], "TEST123") {
		t.Error("First data row missing VIN")
	}

	// Verify timestamp format
	if !strings.Contains(lines[1], "2024-01-01T12:00:00Z") {
		t.Error("First data row has incorrect timestamp")
	}
}

func TestWriteCSV_SingleVehicle(t *testing.T) {
	exporter := &Exporter{}

	points := []DataPoint{
		{
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 75.5,
			},
		},
	}

	var buf bytes.Buffer
	opts := ExportOptions{
		Format: FormatCSV,
		VIN:    "TEST123", // Single vehicle filter
	}
	err := exporter.writeCSV(&buf, points, opts)
	if err != nil {
		t.Fatalf("writeCSV failed: %v", err)
	}

	output := buf.String()

	// When filtering by single VIN, VIN column should not be included
	if strings.Contains(output, "vin") {
		t.Error("CSV output should not include VIN column when filtering by single VIN")
	}
}

func TestWriteCSV_EmptyData(t *testing.T) {
	exporter := &Exporter{}

	var buf bytes.Buffer
	opts := ExportOptions{Format: FormatCSV}
	err := exporter.writeCSV(&buf, []DataPoint{}, opts)
	if err != nil {
		t.Fatalf("writeCSV failed with empty data: %v", err)
	}

	if buf.Len() > 0 {
		t.Error("Expected no output for empty data")
	}
}

func TestExportOptions(t *testing.T) {
	opts := ExportOptions{
		Format:      FormatJSON,
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		VIN:         "TEST123",
		Measurement: "vehicle_status",
		Fields:      []string{"battery_level", "odometer"},
		GroupBy:     5 * time.Minute,
	}

	if opts.Format != FormatJSON {
		t.Errorf("expected Format JSON, got %s", opts.Format)
	}

	if opts.VIN != "TEST123" {
		t.Errorf("expected VIN TEST123, got %s", opts.VIN)
	}

	if len(opts.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(opts.Fields))
	}

	if opts.GroupBy != 5*time.Minute {
		t.Errorf("expected GroupBy 5m, got %v", opts.GroupBy)
	}
}

func TestDataPoint(t *testing.T) {
	point := DataPoint{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		VIN:       "TEST123",
		Fields: map[string]interface{}{
			"battery_level": 75.5,
			"odometer":      12345.6,
			"charging":      true,
		},
	}

	if point.VIN != "TEST123" {
		t.Errorf("expected VIN TEST123, got %s", point.VIN)
	}

	if point.Fields["battery_level"] != 75.5 {
		t.Errorf("expected battery_level 75.5, got %v", point.Fields["battery_level"])
	}

	if point.Fields["charging"] != true {
		t.Errorf("expected charging true, got %v", point.Fields["charging"])
	}

	// Test JSON marshaling
	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("failed to marshal DataPoint: %v", err)
	}

	var unmarshaled DataPoint
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal DataPoint: %v", err)
	}

	if unmarshaled.VIN != point.VIN {
		t.Error("VIN not preserved in JSON roundtrip")
	}
}

func TestNew(t *testing.T) {
	// This test just verifies the constructor doesn't panic
	// We can't test with a real InfluxDB client without mocking
	exporter := &Exporter{
		bucket: "test-bucket",
		org:    "test-org",
	}

	if exporter.bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", exporter.bucket)
	}

	if exporter.org != "test-org" {
		t.Errorf("expected org test-org, got %s", exporter.org)
	}
}

// Benchmark tests
func BenchmarkBuildQuery(b *testing.B) {
	exporter := &Exporter{
		bucket: "test-bucket",
		org:    "test-org",
	}

	opts := ExportOptions{
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		Measurement: "vehicle_status",
		Fields:      []string{"battery_level", "odometer", "charging_power"},
		GroupBy:     5 * time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exporter.buildQuery(opts)
	}
}

func BenchmarkWriteJSON(b *testing.B) {
	exporter := &Exporter{}

	points := make([]DataPoint, 100)
	for i := 0; i < 100; i++ {
		points[i] = DataPoint{
			Timestamp: time.Date(2024, 1, 1, 12, i, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 75.5,
				"odometer":      12345.6 + float64(i),
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.writeJSON(&buf, points)
	}
}

func BenchmarkWriteCSV(b *testing.B) {
	exporter := &Exporter{}

	points := make([]DataPoint, 100)
	for i := 0; i < 100; i++ {
		points[i] = DataPoint{
			Timestamp: time.Date(2024, 1, 1, 12, i, 0, 0, time.UTC),
			VIN:       "TEST123",
			Fields: map[string]interface{}{
				"battery_level": 75.5,
				"odometer":      12345.6 + float64(i),
			},
		}
	}

	opts := ExportOptions{Format: FormatCSV}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.writeCSV(&buf, points, opts)
	}
}
