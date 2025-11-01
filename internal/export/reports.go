// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package export

import (
	"context"
	"fmt"
	"io"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// MonthlyReport contains statistics for a month
type MonthlyReport struct {
	Year              int                     `json:"year"`
	Month             time.Month              `json:"month"`
	VIN               string                  `json:"vin"`
	TotalDistance     float64                 `json:"total_distance_km"`
	TotalChargingCost float64                 `json:"total_charging_cost,omitempty"`
	ChargingSessions  int                     `json:"charging_sessions"`
	TotalEnergyUsed   float64                 `json:"total_energy_kwh,omitempty"`
	AverageEfficiency float64                 `json:"average_efficiency_kwh_per_100km,omitempty"`
	Vehicles          map[string]VehicleStats `json:"vehicles,omitempty"`
}

// VehicleStats contains statistics for a specific vehicle
type VehicleStats struct {
	Make              string  `json:"make"`
	Model             string  `json:"model"`
	VIN               string  `json:"vin"`
	TotalDistance     float64 `json:"total_distance_km"`
	TotalChargingCost float64 `json:"total_charging_cost,omitempty"`
	ChargingSessions  int     `json:"charging_sessions"`
	TotalEnergyUsed   float64 `json:"total_energy_kwh,omitempty"`
	AverageEfficiency float64 `json:"average_efficiency_kwh_per_100km,omitempty"`
}

// TripSummary contains information about a single trip
type TripSummary struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Distance    float64   `json:"distance_km"`
	Duration    string    `json:"duration"`
	StartOdometer float64 `json:"start_odometer_km"`
	EndOdometer   float64 `json:"end_odometer_km"`
}

// ChargingSession contains information about a charging session
type ChargingSession struct {
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      string    `json:"duration"`
	EnergyAdded   float64   `json:"energy_added_kwh"`
	Cost          float64   `json:"cost,omitempty"`
	StartBattery  float64   `json:"start_battery_percent"`
	EndBattery    float64   `json:"end_battery_percent"`
	PeakPower     float64   `json:"peak_power_kw"`
	AveragePower  float64   `json:"average_power_kw"`
}

// Reporter generates various reports from historical data
type Reporter struct {
	exporter     *Exporter
	queryAPI     api.QueryAPI
	bucket       string
	org          string
	costPerKWh   float64
}

// NewReporter creates a new report generator
func NewReporter(client influxdb2.Client, org, bucket string, costPerKWh float64) *Reporter {
	return &Reporter{
		exporter:   New(client, org, bucket),
		queryAPI:   client.QueryAPI(org),
		bucket:     bucket,
		org:        org,
		costPerKWh: costPerKWh,
	}
}

// GenerateMonthlyReport generates a monthly summary report
func (r *Reporter) GenerateMonthlyReport(ctx context.Context, writer io.Writer, year int, month time.Month, format Format) error {
	// Calculate time range for the month
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	// Query for all vehicle data in the month
	query := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r._measurement == "vehicle_status")
  |> filter(fn: (r) => r._field == "odometer" or r._field == "battery_level" or r._field == "charging" or r._field == "charging_power")
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
`, r.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339))

	result, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("querying monthly data: %w", err)
	}
	defer result.Close()

	// Aggregate statistics by VIN
	vehicleStats := make(map[string]*VehicleStats)
	chargingSessions := make(map[string][]*ChargingSession)

	var prevOdometer, prevBattery float64
	var prevCharging bool
	var currentSession *ChargingSession

	for result.Next() {
		record := result.Record()
		vin, _ := record.ValueByKey("vin").(string)

		if _, exists := vehicleStats[vin]; !exists {
			vehicleStats[vin] = &VehicleStats{VIN: vin}
		}

		// Extract values
		odometer, _ := record.ValueByKey("odometer").(float64)
		batteryLevel, _ := record.ValueByKey("battery_level").(float64)
		charging, _ := record.ValueByKey("charging").(bool)
		chargingPower, _ := record.ValueByKey("charging_power").(float64)
		timestamp := record.Time()

		// Calculate distance traveled
		if prevOdometer > 0 && odometer > prevOdometer {
			distance := odometer - prevOdometer
			vehicleStats[vin].TotalDistance += distance
		}

		// Track charging sessions
		if charging && !prevCharging {
			// Start new charging session
			currentSession = &ChargingSession{
				StartTime:    timestamp,
				StartBattery: batteryLevel,
			}
		} else if !charging && prevCharging && currentSession != nil {
			// End charging session
			currentSession.EndTime = timestamp
			currentSession.EndBattery = prevBattery
			currentSession.Duration = timestamp.Sub(currentSession.StartTime).Round(time.Minute).String()

			// Estimate energy added (simplified calculation)
			if currentSession.EndBattery > currentSession.StartBattery {
				batteryDelta := currentSession.EndBattery - currentSession.StartBattery
				// Assume 64 kWh battery capacity (typical for EVs)
				energyAdded := (batteryDelta / 100.0) * 64.0
				currentSession.EnergyAdded = energyAdded
				currentSession.Cost = energyAdded * r.costPerKWh

				vehicleStats[vin].TotalEnergyUsed += energyAdded
				vehicleStats[vin].TotalChargingCost += currentSession.Cost
			}

			chargingSessions[vin] = append(chargingSessions[vin], currentSession)
			vehicleStats[vin].ChargingSessions++
			currentSession = nil
		} else if charging && currentSession != nil {
			// Update peak power during session
			if chargingPower > currentSession.PeakPower {
				currentSession.PeakPower = chargingPower
			}
		}

		prevOdometer = odometer
		prevBattery = batteryLevel
		prevCharging = charging
	}

	if result.Err() != nil {
		return fmt.Errorf("processing results: %w", result.Err())
	}

	// Calculate efficiency for each vehicle
	for _, stats := range vehicleStats {
		if stats.TotalDistance > 0 && stats.TotalEnergyUsed > 0 {
			stats.AverageEfficiency = (stats.TotalEnergyUsed / stats.TotalDistance) * 100.0
		}
	}

	// Create report
	report := MonthlyReport{
		Year:     year,
		Month:    month,
		Vehicles: make(map[string]VehicleStats),
	}

	for vin, stats := range vehicleStats {
		report.Vehicles[vin] = *stats
		report.TotalDistance += stats.TotalDistance
		report.TotalChargingCost += stats.TotalChargingCost
		report.ChargingSessions += stats.ChargingSessions
		report.TotalEnergyUsed += stats.TotalEnergyUsed
	}

	// Calculate overall efficiency
	if report.TotalDistance > 0 && report.TotalEnergyUsed > 0 {
		report.AverageEfficiency = (report.TotalEnergyUsed / report.TotalDistance) * 100.0
	}

	// Export report
	return r.writeReport(writer, report, format)
}

// GenerateMileageReport generates a mileage report for tax deductions
func (r *Reporter) GenerateMileageReport(ctx context.Context, writer io.Writer, start, end time.Time, format Format) error {
	query := fmt.Sprintf(`
from(bucket: "%s")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r._measurement == "vehicle_status")
  |> filter(fn: (r) => r._field == "odometer")
  |> aggregateWindow(every: 1d, fn: first, createEmpty: false)
`, r.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339))

	result, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("querying mileage data: %w", err)
	}
	defer result.Close()

	// Collect daily odometer readings
	type DailyReading struct {
		Date     time.Time
		Odometer float64
		VIN      string
	}

	var readings []DailyReading
	for result.Next() {
		record := result.Record()
		vin, _ := record.ValueByKey("vin").(string)
		odometer, _ := record.Value().(float64)

		readings = append(readings, DailyReading{
			Date:     record.Time(),
			Odometer: odometer,
			VIN:      vin,
		})
	}

	if result.Err() != nil {
		return fmt.Errorf("processing mileage results: %w", result.Err())
	}

	// Calculate trips (simplified: day-to-day changes)
	var trips []TripSummary
	for i := 1; i < len(readings); i++ {
		if readings[i].VIN != readings[i-1].VIN {
			continue
		}

		distance := readings[i].Odometer - readings[i-1].Odometer
		if distance > 0 {
			trip := TripSummary{
				StartTime:     readings[i-1].Date,
				EndTime:       readings[i].Date,
				Distance:      distance,
				Duration:      readings[i].Date.Sub(readings[i-1].Date).Round(time.Minute).String(),
				StartOdometer: readings[i-1].Odometer,
				EndOdometer:   readings[i].Odometer,
			}
			trips = append(trips, trip)
		}
	}

	// Export trips
	if format == FormatCSV {
		return r.writeTripsCSV(writer, trips)
	}
	return r.writeTripsJSON(writer, trips)
}

// writeReport writes a monthly report in the specified format
func (r *Reporter) writeReport(writer io.Writer, report MonthlyReport, format Format) error {
	if format == FormatJSON {
		return r.exporter.writeJSON(writer, report)
	}

	// CSV format for monthly report
	return fmt.Errorf("CSV format not yet implemented for monthly reports")
}

// writeTripsCSV writes trips in CSV format
func (r *Reporter) writeTripsCSV(writer io.Writer, trips []TripSummary) error {
	return r.exporter.writeCSV(writer, convertTripsToDataPoints(trips), ExportOptions{Format: FormatCSV})
}

// writeTripsJSON writes trips in JSON format
func (r *Reporter) writeTripsJSON(writer io.Writer, trips []TripSummary) error {
	return r.exporter.writeJSON(writer, trips)
}

// convertTripsToDataPoints converts trips to data points for CSV export
func convertTripsToDataPoints(trips []TripSummary) []DataPoint {
	points := make([]DataPoint, len(trips))
	for i, trip := range trips {
		points[i] = DataPoint{
			Timestamp: trip.StartTime,
			Fields: map[string]interface{}{
				"end_time":       trip.EndTime.Format(time.RFC3339),
				"distance_km":    trip.Distance,
				"duration":       trip.Duration,
				"start_odometer": trip.StartOdometer,
				"end_odometer":   trip.EndOdometer,
			},
		}
	}
	return points
}
