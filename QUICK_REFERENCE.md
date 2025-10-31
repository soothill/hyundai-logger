# Hyundai Logger - Quick Reference Guide

## What This Project Does

Continuously monitors a Hyundai/Kia vehicle via the Bluelink API and logs all vehicle data (including charging status and battery level for EVs) to a TimescaleDB time-series database.

## Charging & Battery Support - ANSWER

**YES - FULLY SUPPORTED**

| Data | Location | Type | Frequency |
|------|----------|------|-----------|
| Charging Status | `ev_status.charging` | BOOLEAN | Every poll |
| Battery Level (SoC) | `ev_status.battery_level` | PERCENTAGE (0-100) | Every poll |
| Charging Power | `ev_status.charging_power` | KILOWATTS | Every poll |
| Charge End Time | `ev_status.charge_end_time` | TIMESTAMP | Every poll |
| Plugged In | `ev_status.plugged_in` | BOOLEAN | Every poll |

## Polling Overview

| Aspect | Value |
|--------|-------|
| **Interval** | 5 minutes (configurable) |
| **Rate Limit** | 12 requests/hour (configurable) |
| **Polling Type** | Sequential, all vehicles together |
| **Startup Behavior** | Immediate poll, then wait for interval |
| **Shutdown** | Graceful on SIGINT/SIGTERM |
| **Data Per Cycle** | Status + Location + (EV data if EV) |

## File Locations

### Main Polling Code
- **Polling Loop**: `internal/database/logger.go` - `pollLoop()` method (line 97-118)
- **Vehicle Polling**: `internal/database/logger.go` - `pollVehicle()` method (line 136-175)
- **Rate Limiting**: `internal/api/client.go` - `NewClient()` method (line 32-52)

### Configuration
- **YAML Config**: `config.yaml`
- **Environment Variables**: Prefix `HYUNDAI_`, `DB_`, `POLL_`, `REQUESTS_`

### Data Storage
- **Main Status**: `vehicle_status` table (45+ fields)
- **EV Data**: `ev_status` table (charging + battery fields)
- **Location**: `vehicle_location` table (GPS data)
- **Vehicles**: `vehicles` table (vehicle metadata)

### Documentation
- **Comprehensive Analysis**: `CODEBASE_ANALYSIS.md`
- **Polling Details**: `POLLING_IMPLEMENTATION.md`
- **User Guide**: `README.md`
- **Quick Start**: `QUICKSTART.md`

## Configuration

### Key Settings (config.yaml)

```yaml
rate_limit:
  requests_per_hour: 12          # API rate limit
  poll_interval_minutes: 5        # How often to poll

hyundai:
  username: "email@example.com"   # Bluelink email
  password: "password"            # Bluelink password
  pin: "1234"                     # Vehicle PIN
  brand: "hyundai"                # or "kia"
  region: "US"                    # US, CA, EU

database:
  host: "localhost"
  port: 5432
  user: "hyundai_logger"
  password: "password"
  dbname: "hyundai_data"
```

### Environment Variables (override YAML)

```bash
# Hyundai
export HYUNDAI_USERNAME=email@example.com
export HYUNDAI_PASSWORD=password
export HYUNDAI_PIN=1234
export HYUNDAI_BRAND=hyundai
export HYUNDAI_REGION=US

# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=hyundai_logger
export DB_PASSWORD=password
export DB_NAME=hyundai_data

# Polling
export POLL_INTERVAL_MINUTES=5
export REQUESTS_PER_HOUR=12
```

## How Polling Works (Simplified)

```
1. App starts
   ├── Load config
   ├── Connect to database
   └── Authenticate with Hyundai API

2. Create polling goroutine
   ├── IMMEDIATE: Poll all vehicles
   │   └── For each vehicle:
   │       ├── Get vehicle status
   │       ├── Store in vehicle_status table
   │       ├── If EV: Store in ev_status table (CHARGING + BATTERY)
   │       ├── Get location
   │       └── Store in vehicle_location table
   │
   └── Wait 5 minutes, repeat

3. Rate limiting (applies to all API calls)
   ├── Max 12 requests/hour
   ├── Token bucket algorithm
   └── Each call waits if tokens unavailable

4. On shutdown (Ctrl+C)
   ├── Stop polling loop
   ├── Close database
   └── Exit gracefully
```

## Key Code Snippets

### Getting Latest EV Charging Status

```sql
SELECT DISTINCT ON (vehicle_id)
    time, battery_level, charging, charging_power, charge_end_time
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
ORDER BY vehicle_id, time DESC
LIMIT 1;
```

### Charging History (Last 7 Days)

```sql
SELECT
    time_bucket('1 hour', time) as hour,
    AVG(battery_level) as avg_battery,
    MAX(charging_power) as peak_power
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND charging = true
  AND time > NOW() - INTERVAL '7 days'
GROUP BY hour
ORDER BY hour DESC;
```

## Starting the Application

```bash
# Initialize database (first time only)
go run cmd/hyundai-logger/main.go -init-db

# Run the logger
go run cmd/hyundai-logger/main.go

# With custom config file
go run cmd/hyundai-logger/main.go -config /path/to/config.yaml

# Run as background service
systemctl start hyundai-logger
```

## Important Notes

1. **Rate Limiting**: Tokens are global - all API calls compete for same limit
2. **Startup Poll**: Vehicle is polled immediately on startup (before waiting for interval)
3. **EV Detection**: EV data only collected if vehicle.EV != nil in API response
4. **Graceful Shutdown**: Properly cleans up all connections
5. **Unofficial API**: Uses reverse-engineered endpoints, not officially supported

## Troubleshooting

### No data being logged?
- Check logs at `/var/log/hyundai-logger/hyundai-logger.log`
- Verify Hyundai credentials work in official app
- Check database connection: `psql -h localhost -U hyundai_logger -d hyundai_data`

### API calls failing?
- Verify `region` matches your account's region
- Check if credentials have changed
- For EU: may need stamp authentication configured

### Database errors?
- Verify TimescaleDB extension: `SELECT * FROM pg_extension WHERE extname = 'timescaledb';`
- Check database user has permissions
- Initialize schema: `go run cmd/hyundai-logger/main.go -init-db`

## Modifying Behavior

| To Change | Edit | Field |
|-----------|------|-------|
| Poll interval | config.yaml | `rate_limit.poll_interval_minutes` |
| Rate limit | config.yaml | `rate_limit.requests_per_hour` |
| What data is collected | internal/database/logger.go | `pollVehicle()` method |
| API endpoints | internal/api/client.go | `Get*()` methods |
| Database schema | internal/database/database.go | `InitSchema()` method |

## Architecture Summary

```
main.go (application entry point)
    ↓
apiClient (internal/api/client.go)
    - Authenticates
    - Rate limited
    - Makes API calls
    ↓
dataLogger (internal/database/logger.go)
    - Polling loop
    - Vehicle polling
    - Error handling
    ↓
database (internal/database/database.go)
    - TimescaleDB connection
    - Insert operations
    - Schema definitions
    ↓
PostgreSQL + TimescaleDB
    - vehicle_status (hypertable)
    - ev_status (hypertable) ← EV CHARGING DATA
    - vehicle_location (hypertable)
    - vehicles (table)
```

## Data Flow for EV Charging

```
Hyundai API
    ↓ (every 5 min)
GetVehicleStatus() returns EVStatus
    ├── battery_level (%)
    ├── charging (bool)
    ├── charging_power (kW)
    └── ... other fields
    ↓
InsertEVStatus()
    ↓
ev_status table (TimescaleDB)
    ↓
Query with ev_status.charging = true to find active charges
Query ev_status.battery_level to track SoC over time
Query ev_status.charging_power to analyze charging curves
```

## Summary

This is a complete vehicle telematics logger with:
- Full support for EV charging data (battery level + charging state)
- Intelligent rate limiting to prevent battery drain
- Time-series data storage for analysis
- Graceful startup/shutdown
- Configurable polling and rate limits
- Multi-vehicle support
- Multiple region support (US, CA, EU)

The charging and battery data are automatically collected and stored on every poll cycle for EV vehicles.

