// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package database

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	hyundaiapi "github.com/soothill/hyundai-logger/internal/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

const (
	// InfluxDB client configuration
	influxHTTPRequestTimeout = 30              // HTTP request timeout in seconds
	influxMaxRetries         = 3               // Maximum number of retries for failed requests
	influxMaxRetryInterval   = 15              // Maximum retry interval in seconds
	influxBatchSize          = 5000            // Batch size for write operations
	influxFlushInterval      = 1000            // Flush interval in milliseconds
)

var (
	// vehicleIDRegex validates vehicle IDs to prevent injection
	vehicleIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// DB represents the InfluxDB database connection
type DB struct {
	client   influxdb2.Client
	writeAPI influxapi.WriteAPIBlocking
	org      string
	bucket   string
}

// New creates a new InfluxDB connection with optimized settings
func New(ctx context.Context, url, token, org, bucket string) (*DB, error) {
	// Create InfluxDB client with optimized configuration
	// - GZip compression: reduces network bandwidth by 60-80%
	// - Connection pooling: improves throughput under concurrent load
	// - Retry logic: handles transient failures automatically
	// - Batching: reduces write overhead
	client := influxdb2.NewClientWithOptions(url, token,
		influxdb2.DefaultOptions().
			SetUseGZip(true).
			SetHTTPRequestTimeout(influxHTTPRequestTimeout).
			SetMaxRetries(influxMaxRetries).
			SetMaxRetryInterval(influxMaxRetryInterval).
			SetBatchSize(influxBatchSize).
			SetFlushInterval(influxFlushInterval))

	// Test the connection by checking health
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	// Create write API once for reuse
	writeAPI := client.WriteAPIBlocking(org, bucket)

	return &DB{
		client:   client,
		writeAPI: writeAPI,
		org:      org,
		bucket:   bucket,
	}, nil
}

// Close closes the InfluxDB client
func (db *DB) Close() {
	db.client.Close()
}

// HealthCheck performs a health check on the database connection
func (db *DB) HealthCheck(ctx context.Context) error {
	health, err := db.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return fmt.Errorf("database unhealthy: %s", msg)
	}

	return nil
}

// InitSchema ensures the bucket exists (InfluxDB is schemaless for data)
func (db *DB) InitSchema(ctx context.Context) error {
	// Get buckets API
	bucketsAPI := db.client.BucketsAPI()

	// Check if bucket exists
	bucket, err := bucketsAPI.FindBucketByName(ctx, db.bucket)
	if err != nil {
		return fmt.Errorf("checking bucket: %w", err)
	}

	if bucket == nil {
		// Create bucket with 90-day retention
		orgAPI := db.client.OrganizationsAPI()
		org, err := orgAPI.FindOrganizationByName(ctx, db.org)
		if err != nil {
			return fmt.Errorf("finding organization: %w", err)
		}
		if org == nil {
			return fmt.Errorf("organization %s not found", db.org)
		}

		retention := 90 * 24 * time.Hour // 90 days
		retentionSeconds := int64(retention.Seconds())
		retentionRule := domain.RetentionRule{
			EverySeconds: retentionSeconds,
		}
		_, err = bucketsAPI.CreateBucketWithName(ctx, org, db.bucket, retentionRule)
		if err != nil {
			return fmt.Errorf("creating bucket: %w", err)
		}
	}

	return nil
}

// UpsertVehicle stores vehicle metadata
// In InfluxDB, we store this as a measurement with tags for vehicle info
func (db *DB) UpsertVehicle(ctx context.Context, vehicle hyundaiapi.Vehicle) error {
	p := influxdb2.NewPoint("vehicle_info",
		map[string]string{
			"vehicle_id": vehicle.VehicleID,
			"vin":        vehicle.VIN,
			"make":       vehicle.Make,
			"model":      vehicle.Model,
		},
		map[string]interface{}{
			"nickname":   vehicle.Nickname,
			"year":       vehicle.Year,
			"color":      vehicle.Color,
			"generation": vehicle.Generation,
		},
		time.Now(),
	)

	return db.writeAPI.WritePoint(ctx, p)
}

// CollectVehicleStatusPoints creates points for a vehicle status record without writing
// Returns the points for batching
func (db *DB) CollectVehicleStatusPoints(status *hyundaiapi.VehicleStatus) []*write.Point {
	// Create multiple points for different aspects of vehicle status
	// Pre-allocate with capacity 6 for better performance
	points := make([]*write.Point, 0, 6)

	// Engine status
	points = append(points, influxdb2.NewPoint("vehicle_engine",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"running":           status.Engine.Running,
			"remote_start":      status.Engine.RemoteStartState,
			"rpm":               status.Engine.Rpm,
			"range_km":          status.Engine.RangeKM,
			"range_miles":       status.Engine.RangeMiles,
		},
		status.Timestamp,
	))

	// Climate status
	points = append(points, influxdb2.NewPoint("vehicle_climate",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"active":          status.Climate.Active,
			"target_temp":     status.Climate.TargetTemp,
			"interior_temp":   status.Climate.InteriorTemp,
			"exterior_temp":   status.Climate.ExteriorTemp,
			"air_condition":   status.Climate.AirCondition,
			"heater":          status.Climate.Heater,
			"auto_mode":       status.Climate.AutoMode,
			"fan_speed":       status.Climate.FanSpeed,
		},
		status.Timestamp,
	))

	// Doors status
	points = append(points, influxdb2.NewPoint("vehicle_doors",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"locked":      status.Doors.Locked,
			"front_left":  status.Doors.FrontLeft,
			"front_right": status.Doors.FrontRight,
			"back_left":   status.Doors.BackLeft,
			"back_right":  status.Doors.BackRight,
			"trunk":       status.Doors.Trunk,
			"hood":        status.Doors.Hood,
		},
		status.Timestamp,
	))

	// Battery status (12V)
	points = append(points, influxdb2.NewPoint("vehicle_battery",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"level":         status.Battery.Level,
			"voltage":       status.Battery.Voltage,
			"charge_time":   status.Battery.ChargeTime,
			"warning_light": status.Battery.WarningLight,
		},
		status.Timestamp,
	))

	// Tire status
	points = append(points, influxdb2.NewPoint("vehicle_tires",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"front_left_psi":    status.Tire.FrontLeft.PSI,
			"front_left_status": status.Tire.FrontLeft.Status,
			"front_right_psi":   status.Tire.FrontRight.PSI,
			"front_right_status": status.Tire.FrontRight.Status,
			"rear_left_psi":     status.Tire.RearLeft.PSI,
			"rear_left_status":  status.Tire.RearLeft.Status,
			"rear_right_psi":    status.Tire.RearRight.PSI,
			"rear_right_status": status.Tire.RearRight.Status,
			"warning_light":     status.Tire.WarningLight,
		},
		status.Timestamp,
	))

	// General status
	points = append(points, influxdb2.NewPoint("vehicle_status",
		map[string]string{
			"vin": status.VIN,
		},
		map[string]interface{}{
			"odometer":            status.Odometer,
			"fuel_level":          status.FuelLevel,
			"defrost":             status.DefrostStatus,
			"steering_wheel_heat": status.SteeringWheelHeat,
			"side_mirror_heat":    status.SideMirrorHeat,
			"rear_window_heat":    status.RearWindowHeat,
			"washer_level":        status.Washer.Level,
			"washer_warning":      status.Washer.WarningLight,
		},
		status.Timestamp,
	))

	return points
}

// InsertVehicleStatus inserts a vehicle status record (kept for backward compatibility)
func (db *DB) InsertVehicleStatus(ctx context.Context, status *hyundaiapi.VehicleStatus) error {
	points := db.CollectVehicleStatusPoints(status)
	return db.writeAPI.WritePoint(ctx, points...)
}

// CollectEVStatusPoint creates a point for EV status without writing
// Returns nil if not an EV
func (db *DB) CollectEVStatusPoint(timestamp time.Time, vehicleID, vin string, evStatus *hyundaiapi.EVStatus) *write.Point {
	if evStatus == nil {
		return nil // Not an EV
	}

	return influxdb2.NewPoint("vehicle_ev",
		map[string]string{
			"vehicle_id": vehicleID,
			"vin":        vin,
		},
		map[string]interface{}{
			"battery_level":           evStatus.BatteryLevel,
			"battery_capacity":        evStatus.BatteryCapacity,
			"charging":                evStatus.Charging,
			"charging_power":          evStatus.ChargingPower,
			"estimated_current_charge": evStatus.EstimatedCurrentCharge,
			"estimated_full_charge":   evStatus.EstimatedFullCharge,
			"range_km":                evStatus.RangeKM,
			"range_miles":             evStatus.RangeMiles,
			"plugged_in":              evStatus.PluggedIn,
			"charge_target_percent":   evStatus.ChargeTargetPercent,
		},
		timestamp,
	)
}

// InsertEVStatus inserts an EV status record (kept for backward compatibility)
func (db *DB) InsertEVStatus(ctx context.Context, timestamp time.Time, vehicleID, vin string, evStatus *hyundaiapi.EVStatus) error {
	p := db.CollectEVStatusPoint(timestamp, vehicleID, vin, evStatus)
	if p == nil {
		return nil
	}
	return db.writeAPI.WritePoint(ctx, p)
}

// CollectLocationPoint creates a point for vehicle location without writing
func (db *DB) CollectLocationPoint(location *hyundaiapi.Location) *write.Point {
	return influxdb2.NewPoint("vehicle_location",
		map[string]string{
			"vin": location.VIN,
		},
		map[string]interface{}{
			"latitude":  location.Location.Latitude,
			"longitude": location.Location.Longitude,
			"altitude":  location.Location.Altitude,
			"speed":     location.Location.Speed,
			"heading":   location.Location.Heading,
		},
		location.Timestamp,
	)
}

// InsertLocation inserts a vehicle location record (kept for backward compatibility)
func (db *DB) InsertLocation(ctx context.Context, location *hyundaiapi.Location) error {
	p := db.CollectLocationPoint(location)
	return db.writeAPI.WritePoint(ctx, p)
}

// WriteBatch writes multiple points in a single batch operation
// This is more efficient than individual writes
func (db *DB) WriteBatch(ctx context.Context, points []*write.Point) error {
	if len(points) == 0 {
		return nil // Nothing to write
	}
	return db.writeAPI.WritePoint(ctx, points...)
}

// escapeFluxString escapes special characters for Flux queries
func escapeFluxString(s string) string {
	// Escape backslashes and quotes for Flux
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// isValidVehicleID validates a vehicle ID to prevent injection
func isValidVehicleID(id string) bool {
	return vehicleIDRegex.MatchString(id)
}

// GetLatestStatus retrieves the latest status for a vehicle
func (db *DB) GetLatestStatus(ctx context.Context, vehicleID string) (*hyundaiapi.VehicleStatus, error) {
	// Validate vehicle ID to prevent injection
	if !isValidVehicleID(vehicleID) {
		return nil, fmt.Errorf("invalid vehicle ID format: only alphanumeric characters, hyphens, and underscores allowed")
	}

	queryAPI := db.client.QueryAPI(db.org)

	// Escape inputs for Flux query
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -24h)
		|> filter(fn: (r) => r["_measurement"] == "vehicle_status")
		|> filter(fn: (r) => r["vin"] == "%s")
		|> last()
	`, escapeFluxString(db.bucket), escapeFluxString(vehicleID))

	result, err := queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	if !result.Next() {
		return nil, nil // No data found
	}

	// Parse result into VehicleStatus (simplified)
	status := &hyundaiapi.VehicleStatus{}
	if result.Record().Time().Unix() > 0 {
		status.Timestamp = result.Record().Time()
	}

	return status, nil
}
