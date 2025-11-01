// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Format represents the export format
type Format string

const (
	FormatCSV  Format = "csv"
	FormatJSON Format = "json"
)

// ExportOptions configures data export
type ExportOptions struct {
	Format      Format
	StartTime   time.Time
	EndTime     time.Time
	VIN         string        // Optional: filter by specific vehicle
	Measurement string        // InfluxDB measurement to query
	Fields      []string      // Fields to include in export
	GroupBy     time.Duration // Optional: aggregate data by interval
}

// DataPoint represents a single data point for export
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	VIN       string                 `json:"vin,omitempty"`
	Fields    map[string]interface{} `json:"fields"`
}

// Exporter handles data export operations
type Exporter struct {
	queryAPI api.QueryAPI
	bucket   string
	org      string
}

// New creates a new exporter
func New(client influxdb2.Client, org, bucket string) *Exporter {
	return &Exporter{
		queryAPI: client.QueryAPI(org),
		bucket:   bucket,
		org:      org,
	}
}

// Export exports data according to the specified options
func (e *Exporter) Export(ctx context.Context, writer io.Writer, opts ExportOptions) error {
	// Build Flux query
	query := e.buildQuery(opts)

	// Execute query
	result, err := e.queryAPI.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}
	defer result.Close()

	// Collect data points
	dataPoints, err := e.collectDataPoints(result, opts)
	if err != nil {
		return fmt.Errorf("collecting data points: %w", err)
	}

	// Write in requested format
	switch opts.Format {
	case FormatCSV:
		return e.writeCSV(writer, dataPoints, opts)
	case FormatJSON:
		return e.writeJSON(writer, dataPoints)
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

// buildQuery constructs a Flux query based on export options
func (e *Exporter) buildQuery(opts ExportOptions) string {
	query := fmt.Sprintf(`from(bucket: "%s")`, e.bucket)

	// Time range
	query += fmt.Sprintf(`
  |> range(start: %s, stop: %s)`,
		opts.StartTime.Format(time.RFC3339),
		opts.EndTime.Format(time.RFC3339))

	// Filter by measurement
	if opts.Measurement != "" {
		query += fmt.Sprintf(`
  |> filter(fn: (r) => r._measurement == "%s")`, opts.Measurement)
	}

	// Filter by VIN if specified
	if opts.VIN != "" {
		query += fmt.Sprintf(`
  |> filter(fn: (r) => r.vin == "%s")`, opts.VIN)
	}

	// Filter by fields if specified
	if len(opts.Fields) > 0 {
		query += `
  |> filter(fn: (r) => `
		for i, field := range opts.Fields {
			if i > 0 {
				query += " or "
			}
			query += fmt.Sprintf(`r._field == "%s"`, field)
		}
		query += ")"
	}

	// Aggregate if GroupBy is specified
	if opts.GroupBy > 0 {
		windowDuration := opts.GroupBy.String()
		query += fmt.Sprintf(`
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)`, windowDuration)
	}

	// Pivot to get all fields in one record
	query += `
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")`

	return query
}

// collectDataPoints parses query results into DataPoints
func (e *Exporter) collectDataPoints(result *api.QueryTableResult, opts ExportOptions) ([]DataPoint, error) {
	var points []DataPoint

	for result.Next() {
		record := result.Record()

		point := DataPoint{
			Timestamp: record.Time(),
			Fields:    make(map[string]interface{}),
		}

		// Extract VIN if present
		if vin, ok := record.ValueByKey("vin").(string); ok {
			point.VIN = vin
		}

		// Extract all fields (excluding metadata)
		for key, value := range record.Values() {
			// Skip metadata fields
			if key == "_time" || key == "_start" || key == "_stop" ||
				key == "_measurement" || key == "result" || key == "table" {
				continue
			}

			// Skip VIN (already extracted)
			if key == "vin" {
				continue
			}

			// Include field if it's in the requested fields or no fields specified
			if len(opts.Fields) == 0 || contains(opts.Fields, key) {
				point.Fields[key] = value
			}
		}

		points = append(points, point)
	}

	if result.Err() != nil {
		return nil, result.Err()
	}

	return points, nil
}

// writeCSV writes data points as CSV
func (e *Exporter) writeCSV(writer io.Writer, points []DataPoint, opts ExportOptions) error {
	if len(points) == 0 {
		return nil
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Determine all unique field names
	fieldNames := make(map[string]bool)
	for _, point := range points {
		for field := range point.Fields {
			fieldNames[field] = true
		}
	}

	// Create header row
	header := []string{"timestamp"}
	if opts.VIN == "" {
		header = append(header, "vin")
	}
	for field := range fieldNames {
		header = append(header, field)
	}

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write data rows
	for _, point := range points {
		row := []string{point.Timestamp.Format(time.RFC3339)}

		if opts.VIN == "" {
			row = append(row, point.VIN)
		}

		for i := 1; i < len(header); i++ {
			fieldName := header[i]
			if fieldName == "vin" {
				continue
			}

			value := point.Fields[fieldName]
			row = append(row, fmt.Sprintf("%v", value))
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}

// writeJSON writes data as JSON (accepts any type)
func (e *Exporter) writeJSON(writer io.Writer, data interface{}) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
