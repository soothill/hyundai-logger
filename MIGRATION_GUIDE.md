# Migration Guide - Hyundai Logger Refactor

This guide documents breaking changes introduced in the major refactor (commit `6dd4a76`).

## Table of Contents

1. [API Client Changes](#api-client-changes)
2. [Type Structure Changes](#type-structure-changes)
3. [Configuration Changes](#configuration-changes)
4. [Database Changes](#database-changes)
5. [Testing Changes](#testing-changes)
6. [Docker Build Changes](#docker-build-changes)

---

## API Client Changes

### Removed Features

The circuit breaker pattern has been completely removed. If you were using:

```go
// ❌ OLD - No longer available
stats := client.GetCircuitBreakerStats()
```

**Migration**: Circuit breaker functionality is no longer supported. Implement your own retry/backoff logic if needed.

### Authentication

Authentication is now handled in the constructor rather than as a separate method.

```go
// ❌ OLD
client := api.NewClient(region, brand, username, password, pin)
err := client.Authenticate(ctx)

// ✅ NEW
client, err := api.NewClient(region, brand, username, password, pin, refreshToken)
```

**Breaking changes**:
- `Authenticate()` method removed
- Constructor now returns `(*Client, error)` instead of `*Client`
- Authentication happens automatically during construction

### GetVehicles Method

The method signature and return type have changed.

```go
// ❌ OLD
vehicles, err := client.GetVehicles(ctx)
vehicleList := vehicles.Vehicles

// ✅ NEW
vehiclesResp, err := client.GetVehicles()
// Handle both possible response fields
vehicleList := vehiclesResp.Vehicles
if len(vehicleList) == 0 {
    vehicleList = vehiclesResp.Result
}
```

**Breaking changes**:
- No longer accepts `context.Context` parameter
- Returns `*VehiclesResponse` instead of slice
- Response has two possible fields: `Vehicles` or `Result` (region-dependent)

### Removed Methods

The following methods have been removed:

```go
// ❌ No longer available
client.SetBaseURL(url)
client.DisableCache()
```

**Migration**: These features are no longer supported.

---

## Type Structure Changes

### Vehicle Type

Field names and types have changed:

```go
// ❌ OLD
type Vehicle struct {
    Make  string
    Model string
    Year  int
}

// ✅ NEW
type Vehicle struct {
    VehicleName  string `json:"vehicleName"`
    VehicleModel string `json:"vehicleModel"`
    Year         string `json:"year"`  // Now a string!
}
```

**Migration**:
- Replace `vehicle.Make` → `vehicle.VehicleName`
- Replace `vehicle.Model` → `vehicle.VehicleModel`
- Year is now a string, not an int

### VehicleStatus Type

Major restructuring of the status object:

```go
// ❌ OLD
status.EV.BatteryLevel        // float64
status.EV.Charging            // bool
status.EV.RangeKM             // float64
status.Odometer               // int
status.Timestamp              // time.Time

// ✅ NEW
status.EVStatus.BatteryLevel  // int (not float64!)
status.EVStatus.BatteryCharge // bool (was "Charging")
status.EVStatus.RangeEV       // float64 (was "RangeKM")
status.OdometerStatus.Value   // int
status.LastUpdateTime         // time.Time (was "Timestamp")
```

**Key changes**:
- `EV` → `EVStatus` (field renamed)
- `Odometer` → `OdometerStatus.Value` (now nested)
- `BatteryLevel` changed from `float64` to `int`
- `Charging` → `BatteryCharge`
- `RangeKM` → `RangeEV`
- `Timestamp` → `LastUpdateTime`

### VehicleStatus Structure

The status object is now more nested:

```go
// ❌ OLD
type VehicleStatus struct {
    EV        *EVStatus
    Odometer  int
    Timestamp time.Time
}

// ✅ NEW
type VehicleStatus struct {
    VehicleStatus   GeneralStatus
    VehicleLocation Location
    OdometerStatus  OdometerStatus
    EVStatus        *EVStatus
    LastUpdateTime  time.Time
}

type OdometerStatus struct {
    Value int
    Unit  string
}
```

### EVStatus Type

```go
// ❌ OLD
type EVStatus struct {
    BatteryLevel  float64
    Charging      bool
    RangeKM       float64
}

// ✅ NEW
type EVStatus struct {
    BatteryLevel   int     // Changed from float64!
    BatteryCharge  bool    // Renamed from "Charging"
    RangeEV        float64 // Renamed from "RangeKM"
    ChargingPower  float64 // New field
    PluggedIn      bool    // New field
}
```

---

## Configuration Changes

### Validation Logic

The configuration validation has been updated:

```go
// ❌ OLD - Username was always required
hyundai:
  username: required
  password: required
  pin: required

// ✅ NEW - Either refresh_token OR username required
hyundai:
  refresh_token: "token"  # OR username+password
  pin: required
```

**Changes**:
- `refresh_token` OR `username` must be provided (not both required)
- `password` is optional if `refresh_token` exists
- Poll interval validation removed (no longer enforces minimum)

### Error Messages

Validation error messages have changed:

```go
// OLD: "username is required"
// NEW: "either refresh_token or username must be provided"

// OLD: "database url is required"
// NEW: "database URL is required"
```

### Environment Variables

The following environment variables are still supported:
- `HYUNDAI_USERNAME`
- `HYUNDAI_PASSWORD`
- `INFLUXDB_URL`
- `INFLUXDB_ORG`
- `INFLUXDB_BUCKET`

But these are no longer automatically loaded from environment:
- Rate limit settings (must be in config file)
- Alert settings (must be in config file)

### Removed Config Sections

The following config sections have been removed or changed:
- Schedule functionality removed
- Retry config removed (retry logic is built-in)
- Alert defaults no longer set by config validation

---

## Database Changes

### New Methods

A `HealthCheck` method has been added:

```go
// ✅ NEW
func (c *Client) HealthCheck(ctx context.Context) error {
    health, err := c.client.Health(ctx)
    if err != nil {
        return fmt.Errorf("failed to check InfluxDB health: %w", err)
    }
    if health.Status != "pass" {
        return fmt.Errorf("InfluxDB health check failed")
    }
    return nil
}
```

Use this to verify database connectivity before writing data.

---

## Testing Changes

### Test Structure Updates

Tests need to be updated for new types:

```go
// ❌ OLD
if status.EV.BatteryLevel != 75.0 {
    t.Error("unexpected battery level")
}

// ✅ NEW
if status.EVStatus.BatteryLevel != 75 {  // Note: int, not float
    t.Error("unexpected battery level")
}
```

### MockAPI Changes

The mock client test timing parameters have been adjusted:

```go
// ❌ OLD - Too short, tests would fail
config.ChargingPowerKW = 100.0
time.Sleep(100 * time.Millisecond)

// ✅ NEW - Adjusted for realistic simulation
config.ChargingPowerKW = 5000.0  // Faster for tests
time.Sleep(1 * time.Second)
```

### Skipped Tests

Some tests are now skipped because features were removed:
- Circuit breaker tests
- Custom base URL tests
- Schedule-related tests
- Old validation behavior tests

---

## Docker Build Changes

### Multi-Architecture Builds

The `docker-build-multiarch` target has been fixed:

```bash
# ❌ OLD - Would fail with manifest list error
make docker-build-multiarch  # Used --load which doesn't work for multi-arch

# ✅ NEW - Builds and caches, ready for push
make docker-build-multiarch  # Builds for amd64, arm64, arm/v7
make docker-push             # Push to registry
```

**Important**: Multi-architecture builds cannot be loaded locally. Use:
- `make docker-build` for local single-architecture builds
- `make docker-build-multiarch` + `make docker-push` for multi-arch registry push

---

## Complete Migration Example

Here's a complete example showing old vs new code:

```go
// ❌ OLD CODE
package main

import (
    "context"
    "github.com/soothill/hyundai-logger/internal/api"
)

func main() {
    ctx := context.Background()

    // Old authentication
    client := api.NewClient("US", "hyundai", "user", "pass", "1234")
    if err := client.Authenticate(ctx); err != nil {
        panic(err)
    }

    // Old GetVehicles
    vehicles, err := client.GetVehicles(ctx)
    if err != nil {
        panic(err)
    }

    for _, vehicle := range vehicles.Vehicles {
        // Old status access
        status, _ := client.GetVehicleStatus(vehicle.VehicleID)

        battery := status.EV.BatteryLevel  // float64
        odometer := status.Odometer        // int
        charging := status.EV.Charging     // bool

        println(vehicle.Model, battery, odometer, charging)
    }
}
```

```go
// ✅ NEW CODE
package main

import (
    "github.com/soothill/hyundai-logger/internal/api"
)

func main() {
    // New authentication in constructor
    client, err := api.NewClient("US", "hyundai", "user", "pass", "1234", "")
    if err != nil {
        panic(err)
    }

    // New GetVehicles (no context)
    vehiclesResp, err := client.GetVehicles()
    if err != nil {
        panic(err)
    }

    // Handle both possible response fields
    vehicleList := vehiclesResp.Vehicles
    if len(vehicleList) == 0 {
        vehicleList = vehiclesResp.Result
    }

    for _, vehicle := range vehicleList {
        // New status access
        status, _ := client.GetVehicleStatus(vehicle.VehicleID)

        battery := status.EVStatus.BatteryLevel     // int (not float!)
        odometer := status.OdometerStatus.Value     // nested
        charging := status.EVStatus.BatteryCharge   // renamed field

        println(vehicle.VehicleModel, battery, odometer, charging)
    }
}
```

---

## Summary of Breaking Changes

### Critical Changes (will break compilation)
- ✅ `api.NewClient()` now returns `(*Client, error)`
- ✅ `Authenticate()` method removed
- ✅ `GetVehicles()` signature changed (no context)
- ✅ `status.EV` → `status.EVStatus`
- ✅ `status.Odometer` → `status.OdometerStatus.Value`
- ✅ `EVStatus.BatteryLevel` changed from `float64` to `int`
- ✅ `Vehicle.Year` changed from `int` to `string`

### Moderate Changes (may break runtime behavior)
- ⚠️ `EVStatus.Charging` → `EVStatus.BatteryCharge`
- ⚠️ `EVStatus.RangeKM` → `EVStatus.RangeEV`
- ⚠️ `status.Timestamp` → `status.LastUpdateTime`
- ⚠️ Config validation now accepts `refresh_token` OR `username`
- ⚠️ Multi-arch Docker builds cannot use `--load`

### Minor Changes
- Circuit breaker removed
- Some methods removed (`SetBaseURL`, `DisableCache`, etc.)
- Test timing adjustments
- Error message text changes

---

## Need Help?

If you encounter issues during migration:

1. Check the [test files](./tests/) for examples of updated usage
2. Review [cmd/hyundai-logger/main.go](./cmd/hyundai-logger/main.go) for complete working example
3. Open an issue at [GitHub Issues](https://github.com/soothill/hyundai-logger/issues)

---

**Migration Guide Version**: 1.0
**Last Updated**: 2025-11-04
**Refactor Commit**: `6dd4a76`
