// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/soothill/hyundai-logger/internal/config"
	"github.com/soothill/hyundai-logger/internal/export"
)

const (
	exitSuccess = 0
	exitError   = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	// Command-line flags
	var (
		configPath  = flag.String("config", "config.yaml", "Path to configuration file")
		reportType  = flag.String("type", "export", "Report type: export, monthly, mileage")
		format      = flag.String("format", "json", "Output format: json, csv")
		outputFile  = flag.String("output", "", "Output file (default: stdout)")
		startTime   = flag.String("start", "", "Start time (RFC3339 format, e.g., 2024-01-01T00:00:00Z)")
		endTime     = flag.String("end", "", "End time (RFC3339 format, e.g., 2024-12-31T23:59:59Z)")
		vin         = flag.String("vin", "", "Filter by specific VIN")
		measurement = flag.String("measurement", "vehicle_status", "InfluxDB measurement name")
		fields      = flag.String("fields", "", "Comma-separated list of fields to export")
		groupBy     = flag.Duration("group-by", 0, "Aggregate data by interval (e.g., 5m, 1h)")
		year        = flag.Int("year", 0, "Year for monthly report (default: current year)")
		month       = flag.Int("month", 0, "Month for monthly report (1-12, default: current month)")
		costPerKWh  = flag.Float64("cost-per-kwh", 0.25, "Electricity cost per kWh for cost calculations")
	)

	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return exitError
	}

	// Create InfluxDB client
	client := influxdb2.NewClient(cfg.Database.URL, cfg.Database.Token)
	defer client.Close()

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to InfluxDB: %v\n", err)
		return exitError
	}

	if health.Status != "pass" {
		fmt.Fprintf(os.Stderr, "InfluxDB health check failed: %s\n", *health.Message)
		return exitError
	}

	fmt.Fprintf(os.Stderr, "Connected to InfluxDB at %s\n", cfg.Database.URL)

	// Parse time range
	var start, end time.Time

	if *startTime == "" {
		// Default: last 30 days
		start = time.Now().AddDate(0, 0, -30)
	} else {
		start, err = time.Parse(time.RFC3339, *startTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid start time: %v\n", err)
			return exitError
		}
	}

	if *endTime == "" {
		end = time.Now()
	} else {
		end, err = time.Parse(time.RFC3339, *endTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid end time: %v\n", err)
			return exitError
		}
	}

	// Determine output destination
	var outputWriter *os.File
	if *outputFile == "" {
		outputWriter = os.Stdout
	} else {
		outputWriter, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return exitError
		}
		defer outputWriter.Close()
	}

	// Execute requested report
	switch *reportType {
	case "export":
		if err := runExport(ctx, client, cfg, outputWriter, start, end, *vin, *measurement, *fields, *groupBy, *format); err != nil {
			fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
			return exitError
		}

	case "monthly":
		if err := runMonthlyReport(ctx, client, cfg, outputWriter, *year, *month, *format, *costPerKWh); err != nil {
			fmt.Fprintf(os.Stderr, "Monthly report failed: %v\n", err)
			return exitError
		}

	case "mileage":
		if err := runMileageReport(ctx, client, cfg, outputWriter, start, end, *format); err != nil {
			fmt.Fprintf(os.Stderr, "Mileage report failed: %v\n", err)
			return exitError
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown report type: %s\n", *reportType)
		fmt.Fprintf(os.Stderr, "Valid types: export, monthly, mileage\n")
		return exitError
	}

	if *outputFile != "" {
		fmt.Fprintf(os.Stderr, "Report written to: %s\n", *outputFile)
	}

	return exitSuccess
}

func runExport(ctx context.Context, client influxdb2.Client, cfg *config.Config, writer *os.File, start, end time.Time, vin, measurement, fieldsStr string, groupBy time.Duration, formatStr string) error {
	exporter := export.New(client, cfg.Database.Organization, cfg.Database.Bucket)

	// Parse format
	var format export.Format
	switch formatStr {
	case "json":
		format = export.FormatJSON
	case "csv":
		format = export.FormatCSV
	default:
		return fmt.Errorf("unsupported format: %s", formatStr)
	}

	// Parse fields
	var fields []string
	if fieldsStr != "" {
		fields = parseFields(fieldsStr)
	}

	opts := export.ExportOptions{
		Format:      format,
		StartTime:   start,
		EndTime:     end,
		VIN:         vin,
		Measurement: measurement,
		Fields:      fields,
		GroupBy:     groupBy,
	}

	fmt.Fprintf(os.Stderr, "Exporting data from %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	if vin != "" {
		fmt.Fprintf(os.Stderr, "Filtering by VIN: %s\n", vin)
	}
	if len(fields) > 0 {
		fmt.Fprintf(os.Stderr, "Fields: %v\n", fields)
	}

	return exporter.Export(ctx, writer, opts)
}

func runMonthlyReport(ctx context.Context, client influxdb2.Client, cfg *config.Config, writer *os.File, year, month int, formatStr string, costPerKWh float64) error {
	reporter := export.NewReporter(client, cfg.Database.Organization, cfg.Database.Bucket, costPerKWh)

	// Default to current month if not specified
	now := time.Now()
	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}

	if month < 1 || month > 12 {
		return fmt.Errorf("invalid month: %d (must be 1-12)", month)
	}

	// Parse format
	var format export.Format
	switch formatStr {
	case "json":
		format = export.FormatJSON
	case "csv":
		format = export.FormatCSV
	default:
		return fmt.Errorf("unsupported format: %s", formatStr)
	}

	fmt.Fprintf(os.Stderr, "Generating monthly report for %s %d\n", time.Month(month), year)
	fmt.Fprintf(os.Stderr, "Electricity cost: $%.2f/kWh\n", costPerKWh)

	return reporter.GenerateMonthlyReport(ctx, writer, year, time.Month(month), format)
}

func runMileageReport(ctx context.Context, client influxdb2.Client, cfg *config.Config, writer *os.File, start, end time.Time, formatStr string) error {
	reporter := export.NewReporter(client, cfg.Database.Organization, cfg.Database.Bucket, 0)

	// Parse format
	var format export.Format
	switch formatStr {
	case "json":
		format = export.FormatJSON
	case "csv":
		format = export.FormatCSV
	default:
		return fmt.Errorf("unsupported format: %s", formatStr)
	}

	fmt.Fprintf(os.Stderr, "Generating mileage report from %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))

	return reporter.GenerateMileageReport(ctx, writer, start, end, format)
}

func parseFields(fieldsStr string) []string {
	// Simple comma-separated parsing
	var fields []string
	current := ""
	for _, ch := range fieldsStr {
		if ch == ',' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else if ch != ' ' {
			current += string(ch)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}
