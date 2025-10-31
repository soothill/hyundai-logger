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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB represents the database connection pool
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new database connection
func New(ctx context.Context, connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Set connection pool settings
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.pool.Close()
}

// InitSchema creates the database schema including TimescaleDB hypertables
func (db *DB) InitSchema(ctx context.Context) error {
	// Enable TimescaleDB extension
	_, err := db.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;")
	if err != nil {
		return fmt.Errorf("creating timescaledb extension: %w", err)
	}

	// Create vehicles table
	_, err = db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS vehicles (
			vehicle_id VARCHAR(100) PRIMARY KEY,
			vin VARCHAR(17) NOT NULL,
			nickname VARCHAR(100),
			year INTEGER,
			make VARCHAR(50),
			model VARCHAR(50),
			color VARCHAR(50),
			generation VARCHAR(50),
			registered_date TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("creating vehicles table: %w", err)
	}

	// Create vehicle_status table (time series)
	_, err = db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS vehicle_status (
			time TIMESTAMPTZ NOT NULL,
			vehicle_id VARCHAR(100) NOT NULL,
			vin VARCHAR(17) NOT NULL,

			-- Engine data
			engine_running BOOLEAN,
			engine_remote_start BOOLEAN,
			engine_rpm INTEGER,
			engine_range_km DOUBLE PRECISION,
			engine_range_miles DOUBLE PRECISION,

			-- Climate data
			climate_active BOOLEAN,
			climate_target_temp DOUBLE PRECISION,
			climate_interior_temp DOUBLE PRECISION,
			climate_exterior_temp DOUBLE PRECISION,
			climate_air_condition BOOLEAN,
			climate_heater BOOLEAN,
			climate_auto_mode BOOLEAN,
			climate_fan_speed INTEGER,

			-- Doors data
			doors_locked BOOLEAN,
			door_front_left BOOLEAN,
			door_front_right BOOLEAN,
			door_back_left BOOLEAN,
			door_back_right BOOLEAN,
			door_trunk BOOLEAN,
			door_hood BOOLEAN,

			-- Battery data
			battery_level DOUBLE PRECISION,
			battery_voltage DOUBLE PRECISION,
			battery_charge_time INTEGER,
			battery_warning_light BOOLEAN,

			-- Tire data
			tire_fl_psi DOUBLE PRECISION,
			tire_fl_status VARCHAR(20),
			tire_fr_psi DOUBLE PRECISION,
			tire_fr_status VARCHAR(20),
			tire_rl_psi DOUBLE PRECISION,
			tire_rl_status VARCHAR(20),
			tire_rr_psi DOUBLE PRECISION,
			tire_rr_status VARCHAR(20),
			tire_warning_light BOOLEAN,

			-- Other status
			odometer DOUBLE PRECISION,
			fuel_level DOUBLE PRECISION,
			defrost_status BOOLEAN,
			steering_wheel_heat BOOLEAN,
			side_mirror_heat BOOLEAN,
			rear_window_heat BOOLEAN,
			washer_level VARCHAR(20),
			washer_warning_light BOOLEAN,

			CONSTRAINT vehicle_status_pkey PRIMARY KEY (time, vehicle_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("creating vehicle_status table: %w", err)
	}

	// Convert to hypertable
	_, err = db.pool.Exec(ctx, `
		SELECT create_hypertable('vehicle_status', 'time',
			if_not_exists => TRUE,
			chunk_time_interval => INTERVAL '1 day'
		);
	`)
	if err != nil {
		return fmt.Errorf("creating vehicle_status hypertable: %w", err)
	}

	// Create EV status table (time series) for electric vehicles
	_, err = db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ev_status (
			time TIMESTAMPTZ NOT NULL,
			vehicle_id VARCHAR(100) NOT NULL,
			vin VARCHAR(17) NOT NULL,

			battery_level DOUBLE PRECISION,
			battery_capacity DOUBLE PRECISION,
			charging BOOLEAN,
			charging_power DOUBLE PRECISION,
			estimated_current_charge INTEGER,
			estimated_full_charge INTEGER,
			range_km DOUBLE PRECISION,
			range_miles DOUBLE PRECISION,
			plugged_in BOOLEAN,
			charge_target_percent INTEGER,
			charge_end_time TIMESTAMPTZ,

			CONSTRAINT ev_status_pkey PRIMARY KEY (time, vehicle_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("creating ev_status table: %w", err)
	}

	// Convert to hypertable
	_, err = db.pool.Exec(ctx, `
		SELECT create_hypertable('ev_status', 'time',
			if_not_exists => TRUE,
			chunk_time_interval => INTERVAL '1 day'
		);
	`)
	if err != nil {
		return fmt.Errorf("creating ev_status hypertable: %w", err)
	}

	// Create location table (time series)
	_, err = db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS vehicle_location (
			time TIMESTAMPTZ NOT NULL,
			vehicle_id VARCHAR(100) NOT NULL,
			vin VARCHAR(17) NOT NULL,

			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			altitude DOUBLE PRECISION,
			speed DOUBLE PRECISION,
			heading DOUBLE PRECISION,

			CONSTRAINT vehicle_location_pkey PRIMARY KEY (time, vehicle_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("creating vehicle_location table: %w", err)
	}

	// Convert to hypertable
	_, err = db.pool.Exec(ctx, `
		SELECT create_hypertable('vehicle_location', 'time',
			if_not_exists => TRUE,
			chunk_time_interval => INTERVAL '1 day'
		);
	`)
	if err != nil {
		return fmt.Errorf("creating vehicle_location hypertable: %w", err)
	}

	// Create indexes for better query performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_vehicle_status_vehicle_time ON vehicle_status (vehicle_id, time DESC);",
		"CREATE INDEX IF NOT EXISTS idx_ev_status_vehicle_time ON ev_status (vehicle_id, time DESC);",
		"CREATE INDEX IF NOT EXISTS idx_vehicle_location_vehicle_time ON vehicle_location (vehicle_id, time DESC);",
	}

	for _, idx := range indexes {
		if _, err := db.pool.Exec(ctx, idx); err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

// UpsertVehicle inserts or updates a vehicle record
func (db *DB) UpsertVehicle(ctx context.Context, vehicle api.Vehicle) error {
	query := `
		INSERT INTO vehicles (vehicle_id, vin, nickname, year, make, model, color, generation, registered_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (vehicle_id) DO UPDATE SET
			nickname = EXCLUDED.nickname,
			year = EXCLUDED.year,
			make = EXCLUDED.make,
			model = EXCLUDED.model,
			color = EXCLUDED.color,
			generation = EXCLUDED.generation,
			registered_date = EXCLUDED.registered_date,
			updated_at = NOW()
	`

	_, err := db.pool.Exec(ctx, query,
		vehicle.VehicleID,
		vehicle.VIN,
		vehicle.Nickname,
		vehicle.Year,
		vehicle.Make,
		vehicle.Model,
		vehicle.Color,
		vehicle.Generation,
		vehicle.RegisteredDate,
	)

	return err
}

// InsertVehicleStatus inserts a vehicle status record
func (db *DB) InsertVehicleStatus(ctx context.Context, status *api.VehicleStatus) error {
	query := `
		INSERT INTO vehicle_status (
			time, vehicle_id, vin,
			engine_running, engine_remote_start, engine_rpm, engine_range_km, engine_range_miles,
			climate_active, climate_target_temp, climate_interior_temp, climate_exterior_temp,
			climate_air_condition, climate_heater, climate_auto_mode, climate_fan_speed,
			doors_locked, door_front_left, door_front_right, door_back_left, door_back_right,
			door_trunk, door_hood,
			battery_level, battery_voltage, battery_charge_time, battery_warning_light,
			tire_fl_psi, tire_fl_status, tire_fr_psi, tire_fr_status,
			tire_rl_psi, tire_rl_status, tire_rr_psi, tire_rr_status, tire_warning_light,
			odometer, fuel_level, defrost_status, steering_wheel_heat, side_mirror_heat,
			rear_window_heat, washer_level, washer_warning_light
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44
		)
		ON CONFLICT (time, vehicle_id) DO NOTHING
	`

	_, err := db.pool.Exec(ctx, query,
		status.Timestamp, status.VIN, status.VIN,
		status.Engine.Running, status.Engine.RemoteStartState, status.Engine.Rpm,
		status.Engine.RangeKM, status.Engine.RangeMiles,
		status.Climate.Active, status.Climate.TargetTemp, status.Climate.InteriorTemp,
		status.Climate.ExteriorTemp, status.Climate.AirCondition, status.Climate.Heater,
		status.Climate.AutoMode, status.Climate.FanSpeed,
		status.Doors.Locked, status.Doors.FrontLeft, status.Doors.FrontRight,
		status.Doors.BackLeft, status.Doors.BackRight, status.Doors.Trunk, status.Doors.Hood,
		status.Battery.Level, status.Battery.Voltage, status.Battery.ChargeTime,
		status.Battery.WarningLight,
		status.Tire.FrontLeft.PSI, status.Tire.FrontLeft.Status,
		status.Tire.FrontRight.PSI, status.Tire.FrontRight.Status,
		status.Tire.RearLeft.PSI, status.Tire.RearLeft.Status,
		status.Tire.RearRight.PSI, status.Tire.RearRight.Status,
		status.Tire.WarningLight,
		status.Odometer, status.FuelLevel, status.DefrostStatus,
		status.SteeringWheelHeat, status.SideMirrorHeat, status.RearWindowHeat,
		status.Washer.Level, status.Washer.WarningLight,
	)

	return err
}

// InsertEVStatus inserts an EV status record
func (db *DB) InsertEVStatus(ctx context.Context, timestamp time.Time, vehicleID, vin string, evStatus *api.EVStatus) error {
	if evStatus == nil {
		return nil // Not an EV
	}

	query := `
		INSERT INTO ev_status (
			time, vehicle_id, vin,
			battery_level, battery_capacity, charging, charging_power,
			estimated_current_charge, estimated_full_charge,
			range_km, range_miles, plugged_in, charge_target_percent, charge_end_time
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (time, vehicle_id) DO NOTHING
	`

	_, err := db.pool.Exec(ctx, query,
		timestamp, vehicleID, vin,
		evStatus.BatteryLevel, evStatus.BatteryCapacity, evStatus.Charging,
		evStatus.ChargingPower, evStatus.EstimatedCurrentCharge,
		evStatus.EstimatedFullCharge, evStatus.RangeKM, evStatus.RangeMiles,
		evStatus.PluggedIn, evStatus.ChargeTargetPercent, evStatus.ChargeEndTime,
	)

	return err
}

// InsertLocation inserts a vehicle location record
func (db *DB) InsertLocation(ctx context.Context, location *api.Location) error {
	query := `
		INSERT INTO vehicle_location (
			time, vehicle_id, vin, latitude, longitude, altitude, speed, heading
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (time, vehicle_id) DO NOTHING
	`

	_, err := db.pool.Exec(ctx, query,
		location.Timestamp, location.VIN, location.VIN,
		location.Location.Latitude, location.Location.Longitude,
		location.Location.Altitude, location.Location.Speed, location.Location.Heading,
	)

	return err
}

// GetLatestStatus retrieves the latest status for a vehicle
func (db *DB) GetLatestStatus(ctx context.Context, vehicleID string) (*api.VehicleStatus, error) {
	query := `
		SELECT
			time, vin, engine_running, engine_remote_start, engine_rpm,
			engine_range_km, engine_range_miles, odometer, fuel_level
		FROM vehicle_status
		WHERE vehicle_id = $1
		ORDER BY time DESC
		LIMIT 1
	`

	var status api.VehicleStatus
	err := db.pool.QueryRow(ctx, query, vehicleID).Scan(
		&status.Timestamp, &status.VIN, &status.Engine.Running,
		&status.Engine.RemoteStartState, &status.Engine.Rpm,
		&status.Engine.RangeKM, &status.Engine.RangeMiles,
		&status.Odometer, &status.FuelLevel,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &status, nil
}
