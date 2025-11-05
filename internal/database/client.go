package database

import (
	"context"
	"fmt"
	"regexp"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
)

// vinRegex validates VIN format (alphanumeric, typically 17 characters)
var vinRegex = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

// sanitizeVIN validates and sanitizes a VIN to prevent injection attacks
func sanitizeVIN(vin string) string {
	// VINs are 17 alphanumeric characters, excluding I, O, and Q
	if vinRegex.MatchString(vin) {
		return vin
	}
	// If exact match fails, allow alphanumeric only (for test/dev VINs)
	if len(vin) > 0 && len(vin) <= 20 {
		// Only allow alphanumeric characters
		for _, c := range vin {
			if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return "" // Invalid character found
			}
		}
		return vin
	}
	return ""
}

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
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", msg)
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

// HealthCheck checks if the database is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	health, err := c.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("failed to check InfluxDB health: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	return nil
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
		bucketID := ""
		if bucket.Id != nil {
			bucketID = *bucket.Id
		}
		fmt.Printf("Bucket already exists: %s (ID: %s)\n", bucket.Name, bucketID)
	}

	return nil
}

// WriteVehicleStatus writes vehicle status data to InfluxDB
func (c *Client) WriteVehicleStatus(vehicle api.Vehicle, status *api.VehicleStatus) error {
	timestamp := time.Now()
	tags := map[string]string{
		"vin":        vehicle.VIN,
		"vehicle_id": vehicle.VehicleID,
		"nickname":   vehicle.Nickname,
		"model":      vehicle.VehicleModel,
		"year":       vehicle.Year,
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
			"engine":          status.VehicleStatus.Engine,
			"locked":          status.VehicleStatus.Locked,
			"fuel_level":      status.VehicleStatus.FuelLevel,
			"low_fuel_light":  status.VehicleStatus.LowFuelLight,
			"battery_voltage": status.VehicleStatus.BatteryVoltage,
			"air_condition":   status.VehicleStatus.AirCondition,
			"defrost":         status.VehicleStatus.DefrostStatus,
			"trunk_open":      status.VehicleStatus.TrunkOpen,
			"hood_open":       status.VehicleStatus.HoodOpen,
			"remote_start":    status.VehicleStatus.RemoteStartStatus.RemoteStartActive,
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
				"battery_level":         status.EVStatus.BatteryLevel,
				"battery_capacity":      status.EVStatus.BatteryCapacity,
				"charging":              status.EVStatus.BatteryCharge,
				"plugged_in":            status.EVStatus.PluggedIn,
				"charging_power":        status.EVStatus.ChargingPower,
				"estimated_charge_time": status.EVStatus.EstimatedChargeTime,
				"target_charge_level":   status.EVStatus.TargetChargeLevel,
				"range_ev_km":           status.EVStatus.RangeEV,
				"range_ev_miles":        status.EVStatus.RangeEV * 0.621371,
				"charging_current":      status.EVStatus.ChargingCurrent,
				"charging_voltage":      status.EVStatus.ChargingVoltage,
				"charge_mode":           status.EVStatus.ChargeMode,
				"charge_status":         status.EVStatus.ChargeStatus,
				"estimated_full_time":   status.EVStatus.EstimatedFullChargeTime,
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
			"front_left_psi":     status.TireStatus.FrontLeftPSI,
			"front_right_psi":    status.TireStatus.FrontRightPSI,
			"rear_left_psi":      status.TireStatus.RearLeftPSI,
			"rear_right_psi":     status.TireStatus.RearRightPSI,
			"front_left_status":  status.TireStatus.FrontLeftStatus,
			"front_right_status": status.TireStatus.FrontRightStatus,
			"rear_left_status":   status.TireStatus.RearLeftStatus,
			"rear_right_status":  status.TireStatus.RearRightStatus,
		},
		timestamp)
	c.writeAPI.WritePoint(p)

	// Note: Flush is handled automatically by the InfluxDB client based on
	// BatchSize and FlushInterval configuration. Manual flush removed to
	// improve performance and allow proper batching.

	return nil
}

// QueryLatestStatus queries the latest vehicle status from the database
func (c *Client) QueryLatestStatus(vin string) (map[string]interface{}, error) {
	queryAPI := c.client.QueryAPI(c.org)

	// Sanitize VIN to prevent injection attacks
	// VINs are alphanumeric, typically 17 characters
	sanitizedVIN := sanitizeVIN(vin)
	if sanitizedVIN == "" {
		return nil, fmt.Errorf("invalid VIN format")
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r["vin"] == "%s")
			|> filter(fn: (r) => r["_measurement"] == "vehicle_status")
			|> last()
	`, c.bucket, sanitizedVIN)

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	for result.Next() {
		record := result.Record()
		data[record.Field()] = record.Value()
	}

	// Check for query errors
	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return data, nil
}

// GetBatteryHistory retrieves battery level history for EV
func (c *Client) GetBatteryHistory(vin string, hours int) ([]*write.Point, error) {
	queryAPI := c.client.QueryAPI(c.org)

	// Sanitize VIN to prevent injection attacks
	sanitizedVIN := sanitizeVIN(vin)
	if sanitizedVIN == "" {
		return nil, fmt.Errorf("invalid VIN format")
	}

	// Validate hours parameter
	if hours < 1 || hours > 8760 { // 1 hour to 1 year
		return nil, fmt.Errorf("invalid hours: must be between 1 and 8760")
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -%dh)
			|> filter(fn: (r) => r["vin"] == "%s")
			|> filter(fn: (r) => r["_measurement"] == "vehicle_ev")
			|> filter(fn: (r) => r["_field"] == "battery_level")
			|> aggregateWindow(every: 10m, fn: mean, createEmpty: false)
	`, c.bucket, hours, sanitizedVIN)

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	var points []*write.Point
	for result.Next() {
		record := result.Record()
		p := influxdb2.NewPoint(
			"battery_history",
			map[string]string{"vin": sanitizedVIN},
			map[string]interface{}{"level": record.Value()},
			record.Time(),
		)
		points = append(points, p)
	}

	// Check for query errors
	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return points, nil
}
