// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/soothill/hyundai-logger/internal/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// DB represents the InfluxDB database connection
type DB struct {
	client influxdb2.Client
	org    string
	bucket string
}

// New creates a new InfluxDB connection
func New(ctx context.Context, url, token, org, bucket string) (*DB, error) {
	// Create InfluxDB client
	client := influxdb2.NewClient(url, token)

	// Test the connection by checking health
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		return nil, fmt.Errorf("InfluxDB health check failed: %s", health.Message)
	}

	return &DB{
		client: client,
		org:    org,
		bucket: bucket,
	}, nil
}

// Close closes the InfluxDB client
func (db *DB) Close() {
	db.client.Close()
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
func (db *DB) UpsertVehicle(ctx context.Context, vehicle api.Vehicle) error {
	writeAPI := db.client.WriteAPIBlocking(db.org, db.bucket)

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

	return writeAPI.WritePoint(ctx, p)
}

// InsertVehicleStatus inserts a vehicle status record
func (db *DB) InsertVehicleStatus(ctx context.Context, status *api.VehicleStatus) error {
	writeAPI := db.client.WriteAPIBlocking(db.org, db.bucket)

	// Create multiple points for different aspects of vehicle status
	points := make([]*write.Point, 0)

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

	// Write all points
	return writeAPI.WritePoint(ctx, points...)
}

// InsertEVStatus inserts an EV status record
func (db *DB) InsertEVStatus(ctx context.Context, timestamp time.Time, vehicleID, vin string, evStatus *api.EVStatus) error {
	if evStatus == nil {
		return nil // Not an EV
	}

	writeAPI := db.client.WriteAPIBlocking(db.org, db.bucket)

	p := influxdb2.NewPoint("vehicle_ev",
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

	return writeAPI.WritePoint(ctx, p)
}

// InsertLocation inserts a vehicle location record
func (db *DB) InsertLocation(ctx context.Context, location *api.Location) error {
	writeAPI := db.client.WriteAPIBlocking(db.org, db.bucket)

	p := influxdb2.NewPoint("vehicle_location",
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

	return writeAPI.WritePoint(ctx, p)
}

// GetLatestStatus retrieves the latest status for a vehicle
func (db *DB) GetLatestStatus(ctx context.Context, vehicleID string) (*api.VehicleStatus, error) {
	queryAPI := db.client.QueryAPI(db.org)

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -24h)
		|> filter(fn: (r) => r["_measurement"] == "vehicle_status")
		|> filter(fn: (r) => r["vin"] == "%s")
		|> last()
	`, db.bucket, vehicleID)

	result, err := queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	if !result.Next() {
		return nil, nil // No data found
	}

	// Parse result into VehicleStatus (simplified)
	status := &api.VehicleStatus{}
	if result.Record().Time().Unix() > 0 {
		status.Timestamp = result.Record().Time()
	}

	return status, nil
}
