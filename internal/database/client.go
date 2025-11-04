package database

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
)

// Client represents the InfluxDB client
type Client struct {
	client   influxdb2.Client
	writeAPI influxapi.WriteAPI
	config   config.DatabaseConfig
	org      string
	bucket   string
}

// NewClient creates a new InfluxDB client
func NewClient(cfg config.DatabaseConfig) (*Client, error) {
	// Create client
	client := influxdb2.NewClientWithOptions(cfg.URL, cfg.Token,
		influxdb2.DefaultOptions().
			SetBatchSize(uint(cfg.BatchSize)).
			SetFlushInterval(uint(cfg.FlushInterval)))

	// Test connection
	health, err := client.Health(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		return nil, fmt.Errorf("InfluxDB health check failed: %s", health.Message)
	}

	// Get write API
	writeAPI := client.WriteAPI(cfg.Organization, cfg.Bucket)

	return &Client{
		client:   client,
		writeAPI: writeAPI,
		config:   cfg,
		org:      cfg.Organization,
		bucket:   cfg.Bucket,
	}, nil
}

// Close closes the database connection
func (c *Client) Close() {
	if c.writeAPI != nil {
		c.writeAPI.Flush()
	}
	if c.client != nil {
		c.client.Close()
	}
}

// InitializeSchema ensures the bucket exists with proper retention
func (c *Client) InitializeSchema() error {
	bucketsAPI := c.client.BucketsAPI()
	orgAPI := c.client.OrganizationsAPI()

	// Get organization
	org, err := orgAPI.FindOrganizationByName(context.Background(), c.org)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	// Check if bucket exists
	bucket, err := bucketsAPI.FindBucketByName(context.Background(), c.bucket)
	if err != nil {
		// Create bucket with 90-day retention
		retentionRule := domain.RetentionRule{
			EverySeconds: 90 * 24 * 60 * 60, // 90 days in seconds
		}
		_, err = bucketsAPI.CreateBucketWithName(context.Background(), org, c.bucket, retentionRule)
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		fmt.Printf("Created bucket: %s\n", c.bucket)
	} else {
		fmt.Printf("Bucket already exists: %s (ID: %s)\n", bucket.Name, bucket.Id)
	}

	return nil
}

// WriteVehicleStatus writes vehicle status data to InfluxDB
func (c *Client) WriteVehicleStatus(vehicle api.Vehicle, status *api.VehicleStatus) error {
	timestamp := time.Now()
	tags := map[string]string{
		"vin":         vehicle.VIN,
		"vehicle_id":  vehicle.VehicleID,
		"nickname":    vehicle.Nickname,
		"model":       vehicle.VehicleModel,
		"year":        vehicle.Year,
	}

	// Write vehicle metadata
	p := influxdb2.NewPoint("vehicle_info",
		tags,
		map[string]interface{}{
			"color":        vehicle.Color,
			"vehicle_name": vehicle.VehicleName,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Write general vehicle status
	p = influxdb2.NewPoint("vehicle_status",
		tags,
		map[string]interface{}{
			"engine":            status.VehicleStatus.Engine,
			"locked":            status.VehicleStatus.Locked,
			"fuel_level":        status.VehicleStatus.FuelLevel,
			"low_fuel_light":    status.VehicleStatus.LowFuelLight,
			"battery_voltage":   status.VehicleStatus.BatteryVoltage,
			"air_condition":     status.VehicleStatus.AirCondition,
			"defrost":           status.VehicleStatus.DefrostStatus,
			"trunk_open":        status.VehicleStatus.TrunkOpen,
			"hood_open":         status.VehicleStatus.HoodOpen,
			"remote_start":      status.VehicleStatus.RemoteStartStatus.RemoteStartActive,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Write odometer
	p = influxdb2.NewPoint("vehicle_odometer",
		tags,
		map[string]interface{}{
			"value": status.OdometerStatus.Value,
			"unit":  status.OdometerStatus.Unit,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Write location if available
	if status.VehicleLocation.Latitude != 0 || status.VehicleLocation.Longitude != 0 {
		p = influxdb2.NewPoint("vehicle_location",
			tags,
			map[string]interface{}{
				"latitude":  status.VehicleLocation.Latitude,
				"longitude": status.VehicleLocation.Longitude,
				"altitude":  status.VehicleLocation.Altitude,
				"speed":     status.VehicleLocation.Speed,
				"heading":   status.VehicleLocation.Heading,
			},
			timestamp)
		c.writeAPI.WritePoint(p)
	}

	// Write EV status if available
	if status.EVStatus != nil {
		p = influxdb2.NewPoint("vehicle_ev",
			tags,
			map[string]interface{}{
				"battery_level":            status.EVStatus.BatteryLevel,
				"battery_capacity":         status.EVStatus.BatteryCapacity,
				"charging":                 status.EVStatus.BatteryCharge,
				"plugged_in":               status.EVStatus.PluggedIn,
				"charging_power":           status.EVStatus.ChargingPower,
				"estimated_charge_time":    status.EVStatus.EstimatedChargeTime,
				"target_charge_level":      status.EVStatus.TargetChargeLevel,
				"range_ev_km":              status.EVStatus.RangeEV,
				"range_ev_miles":           status.EVStatus.RangeEV * 0.621371,
				"charging_current":         status.EVStatus.ChargingCurrent,
				"charging_voltage":         status.EVStatus.ChargingVoltage,
				"charge_mode":              status.EVStatus.ChargeMode,
				"charge_status":            status.EVStatus.ChargeStatus,
				"estimated_full_time":      status.EVStatus.EstimatedFullChargeTime,
			},
			timestamp)
		c.writeAPI.WritePoint(p)
	}

	// Write climate status
	p = influxdb2.NewPoint("vehicle_climate",
		tags,
		map[string]interface{}{
			"active":           status.Climate.Active,
			"interior_temp_c":  status.Climate.InteriorTemp,
			"exterior_temp_c":  status.Climate.ExteriorTemp,
			"target_temp_c":    status.Climate.TargetTemp,
			"fan_speed":        status.Climate.FanSpeed,
			"defrost":          status.Climate.DefrostActive,
			"rear_defrost":     status.Climate.RearDefrost,
			"steering_heat":    status.Climate.SteeringWheel,
			"side_mirror_heat": status.Climate.SideMirrorHeat,
			"seat_heat_left":   status.Climate.SeatHeatLeft,
			"seat_heat_right":  status.Climate.SeatHeatRight,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Write door status
	p = influxdb2.NewPoint("vehicle_doors",
		tags,
		map[string]interface{}{
			"front_left_open":  status.DoorStatus.FrontLeft,
			"front_right_open": status.DoorStatus.FrontRight,
			"rear_left_open":   status.DoorStatus.BackLeft,
			"rear_right_open":  status.DoorStatus.BackRight,
			"trunk_open":       status.DoorStatus.Trunk,
			"hood_open":        status.DoorStatus.Hood,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Write tire status
	p = influxdb2.NewPoint("vehicle_tires",
		tags,
		map[string]interface{}{
			"front_left_psi":      status.TireStatus.FrontLeftPSI,
			"front_right_psi":     status.TireStatus.FrontRightPSI,
			"rear_left_psi":       status.TireStatus.RearLeftPSI,
			"rear_right_psi":      status.TireStatus.RearRightPSI,
			"front_left_status":   status.TireStatus.FrontLeftStatus,
			"front_right_status":  status.TireStatus.FrontRightStatus,
			"rear_left_status":    status.TireStatus.RearLeftStatus,
			"rear_right_status":   status.TireStatus.RearRightStatus,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Flush writes
	c.writeAPI.Flush()

	return nil
}

// QueryLatestStatus queries the latest vehicle status from the database
func (c *Client) QueryLatestStatus(vin string) (map[string]interface{}, error) {
	queryAPI := c.client.QueryAPI(c.org)
	
	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r["vin"] == "%s")
			|> filter(fn: (r) => r["_measurement"] == "vehicle_status")
			|> last()
	`, c.bucket, vin)

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	for result.Next() {
		record := result.Record()
		data[record.Field()] = record.Value()
	}

	return data, nil
}

// GetBatteryHistory retrieves battery level history for EV
func (c *Client) GetBatteryHistory(vin string, hours int) ([]*write.Point, error) {
	queryAPI := c.client.QueryAPI(c.org)

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -%dh)
			|> filter(fn: (r) => r["vin"] == "%s")
			|> filter(fn: (r) => r["_measurement"] == "vehicle_ev")
			|> filter(fn: (r) => r["_field"] == "battery_level")
			|> aggregateWindow(every: 10m, fn: mean, createEmpty: false)
	`, c.bucket, hours, vin)

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	var points []*write.Point
	for result.Next() {
		record := result.Record()
		p := influxdb2.NewPoint(
			"battery_history",
			map[string]string{"vin": vin},
			map[string]interface{}{"level": record.Value()},
			record.Time(),
		)
		points = append(points, p)
	}

	return points, nil
}
