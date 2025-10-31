# Hyundai Logger - Polling Implementation Details

## Quick Reference for Polling System

### Location of Polling Code

**Primary File**: `/home/darren/hyundai-logger/internal/database/logger.go`

**Supporting Files**:
- `/internal/api/client.go` - API calls with rate limiting
- `/internal/config/config.go` - Configuration for poll interval
- `/cmd/hyundai-logger/main.go` - Initialization of polling

---

## Polling Loop Structure

### Entry Point: Logger.Start() Method (Line 49-86)

```go
func (l *Logger) Start(ctx context.Context) error {
    // 1. Authenticate with API
    if err := l.apiClient.Authenticate(ctx); err != nil {
        return fmt.Errorf("initial authentication: %w", err)
    }
    
    // 2. Get vehicles
    vehicles, err := l.apiClient.GetVehicles(ctx)
    if err != nil {
        return fmt.Errorf("fetching vehicles: %w", err)
    }
    
    // 3. Store vehicle info
    for _, vehicle := range vehicles {
        if err := l.db.UpsertVehicle(ctx, vehicle); err != nil {
            l.logger.Error("Failed to store vehicle %s: %v", vehicle.VIN, err)
        }
    }
    
    // 4. Start polling goroutine (THIS IS KEY)
    go l.pollLoop(ctx, vehicles)
    
    return nil
}
```

### Core Polling Loop: Logger.pollLoop() Method (Line 97-118)

```go
func (l *Logger) pollLoop(ctx context.Context, vehicles []api.Vehicle) {
    defer close(l.stoppedCh)  // Signal when done
    
    // Create ticker with configured interval
    ticker := time.NewTicker(l.pollInterval)
    defer ticker.Stop()
    
    // IMMEDIATE POLL ON STARTUP (before waiting for interval)
    l.pollAllVehicles(ctx, vehicles)
    
    // Main polling loop
    for {
        select {
        // Context cancelled (graceful shutdown from main)
        case <-ctx.Done():
            l.logger.Info("Poll loop stopped due to context cancellation")
            return
            
        // Stop signal received (from Stop() method)
        case <-l.stopCh:
            l.logger.Info("Poll loop stopped due to stop signal")
            return
            
        // Timer fired - time to poll
        case <-ticker.C:
            l.pollAllVehicles(ctx, vehicles)
        }
    }
}
```

### Poll All Vehicles: Logger.pollAllVehicles() Method (Line 121-133)

```go
func (l *Logger) pollAllVehicles(ctx context.Context, vehicles []api.Vehicle) {
    startTime := time.Now()
    l.logger.LogPollStart()
    
    // Sequential polling of each vehicle
    for _, vehicle := range vehicles {
        if err := l.pollVehicle(ctx, vehicle); err != nil {
            l.logger.Error("Error polling vehicle %s: %v", vehicle.VIN, err)
        }
    }
    
    // Log completion
    duration := time.Since(startTime)
    l.logger.LogPollComplete(duration)
}
```

### Poll Single Vehicle: Logger.pollVehicle() Method (Line 136-175)

```go
func (l *Logger) pollVehicle(ctx context.Context, vehicle api.Vehicle) error {
    l.logger.Info("Polling vehicle: %s", vehicle.VIN)
    
    // STEP 1: Get main vehicle status (engine, climate, battery, tires, etc.)
    status, err := l.apiClient.GetVehicleStatus(ctx, vehicle.VehicleID)
    if err != nil {
        return fmt.Errorf("getting vehicle status: %w", err)
    }
    
    // Store main status
    if err := l.db.InsertVehicleStatus(ctx, status); err != nil {
        l.logger.LogError("Storing vehicle status", err)
    } else {
        l.logger.LogDataCollection(vehicle.VIN, status.Odometer, status.FuelLevel)
    }
    
    // STEP 2: If EV vehicle, store EV status (CHARGING + BATTERY LEVEL)
    if status.EV != nil {
        if err := l.db.InsertEVStatus(ctx, status.Timestamp, vehicle.VehicleID, vehicle.VIN, status.EV); err != nil {
            l.logger.LogError("Storing EV status", err)
        } else {
            l.logger.LogEVData(vehicle.VIN, status.EV.BatteryLevel, status.EV.Charging)
        }
    }
    
    // STEP 3: Get location
    location, err := l.apiClient.GetVehicleLocation(ctx, vehicle.VehicleID)
    if err != nil {
        l.logger.Error("Warning: failed to get location: %v", err)
    } else {
        if err := l.db.InsertLocation(ctx, location); err != nil {
            l.logger.LogError("Storing location", err)
        } else {
            l.logger.LogLocation(vehicle.VIN, location.Location.Latitude, location.Location.Longitude)
        }
    }
    
    return nil
}
```

---

## Polling Configuration

### From config.yaml

```yaml
rate_limit:
  requests_per_hour: 12       # Limit: 12 requests per hour
  poll_interval_minutes: 5     # Frequency: Every 5 minutes
```

### From internal/config/config.go

```go
type RateLimitConfig struct {
    RequestsPerHour     int `yaml:"requests_per_hour"`     // Default: 12
    PollIntervalMinutes int `yaml:"poll_interval_minutes"`  // Default: 5
}
```

### How it's used in internal/database/logger.go

```go
// In NewLogger (line 37-46)
func NewLogger(db *DB, apiClient *api.Client, pollIntervalMinutes int, logger LoggerInterface) *Logger {
    return &Logger{
        db:           db,
        apiClient:    apiClient,
        logger:       logger,
        pollInterval: time.Duration(pollIntervalMinutes) * time.Minute,  // Convert to time.Duration
        stopCh:       make(chan struct{}),
        stoppedCh:    make(chan struct{}),
    }
}
```

---

## Rate Limiting Details

### Implementation in internal/api/client.go

```go
// NewClient method (line 32-52)
func NewClient(username, password, pin, brand, region string, requestsPerHour int) *Client {
    // Convert hourly requests to per-second rate
    rps := float64(requestsPerHour) / 3600.0
    // Create token bucket with burst of 1
    limiter := rate.NewLimiter(rate.Limit(rps), 1)
    
    return &Client{
        httpClient:  ...,
        rateLimiter: limiter,
        ...
    }
}

// Every API call (line 84-86)
func (c *Client) Authenticate(ctx context.Context) error {
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limiter: %w", err)
    }
    // ... make API call
}
```

### Rate Limiting Behavior

With **12 requests/hour** (current default):

```
rps = 12 / 3600 = 0.00333 tokens/second
     = 1 token every 300 seconds
     = 1 token every 5 minutes
```

So with both polling every 5 minutes AND rate limit of 12/hour:
- Each poll cycle calls: GetVehicleStatus + GetVehicleLocation + (optional GetOdometer)
- That's ~2-3 API calls per vehicle per poll
- With 1 vehicle: 2-3 requests every 5 minutes = 24-36 requests/hour
- But rate limiter will throttle - only allows 12/hour
- So effective interval becomes ~10-15 minutes per vehicle

**Important**: Rate limit is GLOBAL, not per-vehicle. All API calls (across all vehicles) compete for the same token bucket.

---

## Charging Data and Battery Level (EV Vehicles)

### What Gets Collected

Every poll cycle, for EV vehicles, these fields are captured:

```go
type EVStatus struct {
    BatteryLevel         float64   // STATE OF CHARGE (0-100%)
    BatteryCapacity      float64   // Total capacity in kWh
    Charging             bool      // IS CURRENTLY CHARGING?
    ChargingPower        float64   // Current charging power in kW
    EstimatedCurrentCharge int     // Minutes to reach current charge %
    EstimatedFullCharge  int       // Minutes to reach 100%
    RangeKM              float64   // Estimated driving range (km)
    RangeMiles           float64   // Estimated driving range (miles)
    PluggedIn            bool      // Is charger physically connected?
    ChargeTargetPercent  int       // User's target charge percentage
    ChargeEndTime        time.Time // Timestamp when charging will complete
}
```

### Where It's Stored

**Database Table**: `ev_status` (Hypertable)

```sql
CREATE TABLE ev_status (
    time TIMESTAMPTZ,
    vehicle_id VARCHAR(100),
    vin VARCHAR(17),
    
    -- KEY FIELDS FOR YOUR USE CASE
    battery_level DOUBLE PRECISION,        -- SoC %
    charging BOOLEAN,                      -- Charging state
    charging_power DOUBLE PRECISION,       -- Power in kW
    charge_end_time TIMESTAMPTZ,           -- When it finishes
    
    -- ... other fields
    PRIMARY KEY (time, vehicle_id)
);
```

### Example Query to Get Latest EV Status

```sql
SELECT DISTINCT ON (vehicle_id)
    vehicle_id, vin, time,
    battery_level,
    charging,
    charging_power,
    charge_end_time,
    range_km
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
ORDER BY vehicle_id, time DESC;
```

### Logging Output

When polling an EV, you'll see in logs:

```
INFO:   EV data for VINXXXXXX - Battery: 85.5%, Status: charging
```

(Or "Status: not charging" if not currently charging)

---

## Polling Lifecycle and Graceful Shutdown

### Start Sequence

1. `main()` creates `dataLogger`
2. `dataLogger.Start(ctx)` called
3. Inside Start():
   - Authenticate
   - Get vehicles
   - Store vehicles
   - **Launch polling goroutine** with `go l.pollLoop(ctx, vehicles)`
4. Returns control to main
5. Main waits for signal

### Polling Loop Runs

- Immediately polls all vehicles (even before first interval)
- Then waits for ticker interval (5 minutes by default)
- Each interval, polls all vehicles again
- Rate limiter throttles API calls

### Shutdown Sequence

1. Signal received (Ctrl+C or SIGTERM)
2. Main calls `dataLogger.Stop()`
3. Inside Stop():
   - Close `stopCh` channel
   - Wait for `stoppedCh` to confirm goroutine exited
4. Context cancelled
5. Database connections closed
6. Log file closed

```go
// From internal/database/logger.go line 89-94
func (l *Logger) Stop() {
    l.logger.Info("Stopping data logger...")
    close(l.stopCh)      // Signal polling loop to stop
    <-l.stoppedCh        // Wait for goroutine to finish
    l.logger.Info("Data logger stopped")
}
```

---

## Key Polling Characteristics

| Aspect | Details |
|--------|---------|
| **Scheduling** | Time-based ticker (not cron) |
| **Interval** | Configurable, default 5 minutes |
| **Vehicles** | Sequential polling (not parallel) |
| **Data per poll** | Status + Location (+ EV status if EV) |
| **Startup behavior** | Immediate poll, then wait for interval |
| **Rate limiting** | Token bucket with global limit |
| **Graceful shutdown** | Channel-based signal handling |
| **Multi-vehicle** | All polled in one cycle, not separately |
| **EV detection** | Checks if `status.EV != nil` |
| **Charging state** | Logged every poll if EV |
| **Battery SoC** | Logged every poll if EV |

---

## Code Flow Diagram

```
main.go
  ├── Load config (poll_interval_minutes = 5)
  ├── Create API client (requests_per_hour = 12)
  ├── Create dataLogger with these values
  └── Call dataLogger.Start(ctx)
      │
      ├── Authenticate with API
      ├── Get vehicles list
      ├── Store vehicles in DB
      └── Launch polling goroutine
          │
          └── pollLoop(ctx, vehicles)
              │
              ├── IMMEDIATE: pollAllVehicles()
              │   └── For each vehicle:
              │       └── pollVehicle()
              │           ├── GetVehicleStatus()  [rate limited]
              │           ├── InsertVehicleStatus()
              │           ├── If EV: InsertEVStatus()  [CHARGING + BATTERY]
              │           ├── GetVehicleLocation()  [rate limited]
              │           └── InsertLocation()
              │
              └── Loop: Wait for ticker.C (5 min interval)
                  └── Same as above
```

---

## Performance Considerations

### Time Per Poll Cycle

- Status API call: ~1-2 seconds
- Location API call: ~1-2 seconds
- Database inserts: ~100-500ms
- Per vehicle: ~2-5 seconds total

With 1 vehicle: ~2-5 seconds per poll
With N vehicles: ~(2-5) * N seconds per poll

### Rate Limiting Impact

- Request limit: 12/hour = 1 every 5 minutes
- Default poll interval: 5 minutes
- If polling 1 vehicle with 2 API calls per poll:
  - Want: 2 calls every 5 min = 24/hour
  - Limited to: 12/hour
  - Actual interval: ~10 minutes between polls

### Database Impact

- Insert ~45 columns per vehicle_status record
- Insert ~13 columns per ev_status record (if EV)
- Insert ~8 columns per location record
- Hypertables automatically chunk by day
- Indexes optimized for (vehicle_id, time DESC)

---

## Files to Modify for Polling Changes

1. **Poll interval**: `config.yaml` → `poll_interval_minutes`
2. **Rate limit**: `config.yaml` → `requests_per_hour`
3. **Poll logic**: `internal/database/logger.go` → `pollVehicle()` method
4. **API calls**: `internal/api/client.go` → individual `Get*()` methods
5. **Data stored**: `internal/database/database.go` → `Insert*()` methods

