# Hyundai Logger - Comprehensive Codebase Analysis

## Executive Summary

The **Hyundai Logger** is a Go application that continuously polls the Hyundai Bluelink API to collect vehicle telematics data and stores it in a TimescaleDB time-series database. It implements intelligent rate limiting to prevent excessive API calls that could drain the vehicle's 12V battery, and provides comprehensive logging of vehicle state including charging status and battery level for EVs.

---

## 1. Project Structure and Architecture

### Directory Layout

```
hyundai-logger/
├── cmd/
│   └── hyundai-logger/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── client.go              # Hyundai API client with rate limiting
│   │   └── types.go               # API response data structures
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── database/
│   │   ├── database.go            # Database connection and schema
│   │   └── logger.go              # Data collection polling loop
│   └── logging/
│       └── logger.go              # Structured logging
├── migrations/                      # (Empty - schema created at runtime)
├── config.yaml                     # Configuration file
├── .env.example                    # Environment variables template
├── go.mod / go.sum                 # Go dependencies (v1.23)
├── Makefile                        # Build commands
├── Dockerfile / docker-compose.yml # Container setup
├── setup.sh                        # Automated setup script
├── hyundai-logger.service          # SystemD service file
├── logrotate.conf                  # Log rotation configuration
├── queries.sql                     # Example SQL queries
└── README.md / QUICKSTART.md       # Documentation
```

### Technology Stack

- **Language:** Go 1.23+
- **Database:** PostgreSQL 12+ with TimescaleDB extension
- **HTTP Client:** Standard library `net/http`
- **Rate Limiting:** `golang.org/x/time/rate`
- **Configuration:** YAML parser (`gopkg.in/yaml.v3`)
- **Environment Management:** `github.com/joho/godotenv`
- **Database Driver:** PostgreSQL/PGX v5 (`github.com/jackc/pgx/v5`)

---

## 2. Current Logging and Polling Implementation

### 2.1 Polling Mechanism

The polling system is implemented in `/home/darren/hyundai-logger/internal/database/logger.go`:

#### Polling Loop Architecture

```go
type Logger struct {
    db             *DB
    apiClient      *api.Client
    logger         LoggerInterface
    pollInterval   time.Duration    // Configured interval
    stopCh         chan struct{}     // Stop signal channel
    stoppedCh      chan struct{}     // Confirmation channel
}
```

#### Polling Flow

1. **Initialization** (`Start()` method):
   - Authenticates with Hyundai API
   - Retrieves list of vehicles
   - Stores vehicle information in database
   - Starts polling goroutine with `go l.pollLoop(ctx, vehicles)`

2. **Polling Loop** (`pollLoop()` method):
   - Uses `time.NewTicker(l.pollInterval)` to create periodic ticks
   - Performs **immediate poll on startup** (before waiting for first interval)
   - Then waits for ticker intervals
   - Polls all vehicles on each interval
   - Supports graceful shutdown via channel signals

3. **Per-Vehicle Polling** (`pollVehicle()` method):
   - Fetches vehicle status (engine, climate, doors, battery, tire, fuel, odometer)
   - Stores main status data
   - If EV, fetches and stores EV-specific status (battery level, charging state, range)
   - Fetches and stores GPS location
   - Logs all data collected

#### Polling Configuration

From `/home/darren/hyundai-logger/config.yaml`:

```yaml
rate_limit:
  requests_per_hour: 12       # Total API requests allowed per hour
  poll_interval_minutes: 5     # Polling interval (how often to poll)
```

**Current Default Settings:**
- Poll interval: 5 minutes
- Rate limit: 12 requests/hour (one request every 5 minutes on average)

#### Rate Limiting Implementation

From `/home/darren/hyundai-logger/internal/api/client.go`:

```go
// Rate limiter uses token bucket algorithm
rps := float64(requestsPerHour) / 3600.0  // Convert hourly to per-second rate
limiter := rate.NewLimiter(rate.Limit(rps), 1)  // burst of 1

// Each API call waits for available token
if err := c.rateLimiter.Wait(ctx); err != nil {
    return fmt.Errorf("rate limiter: %w", err)
}
```

**How it works:**
- Tokens generated at constant rate: `requests_per_hour / 3600` tokens/second
- Burst capacity: 1 token (prevents sudden spikes)
- Each API call consumes 1 token
- If no tokens available, call blocks until token available
- With 12 requests/hour: 1 token every 300 seconds (5 minutes)

### 2.2 Shutdown and Cleanup

- Graceful shutdown via `dataLogger.Stop()` method
- Signals listened: `os.Interrupt`, `syscall.SIGTERM`
- Polling goroutine stops cleanly via channel communication
- Database connections properly closed
- Log files properly flushed

---

## 3. Car Data Fetching and Available Information

### 3.1 API Client Structure

From `/home/darren/hyundai-logger/internal/api/client.go`:

#### Supported Regions and Base URLs

```go
URLs := map[string]map[string]string{
    "US": {
        "hyundai": "https://api.telematics.hyundaiusa.com",
        "kia":     "https://api.owners.kia.com",
    },
    "CA": {
        "hyundai": "https://api.telematics.hyundaiusa.com",
        "kia":     "https://api.owners.kia.com",
    },
    "EU": {
        "hyundai": "https://prd.eu-ccapi.hyundai.com:8080",
        "kia":     "https://prd.eu-ccapi.kia.com:8080",
    },
}
```

#### API Endpoints Used

1. **Authentication**: `POST /v2/login`
2. **Get Vehicles**: `GET /v2/vehicles`
3. **Vehicle Status**: `GET /v2/vehicles/{vehicleId}/status`
4. **Vehicle Location**: `GET /v2/vehicles/{vehicleId}/location`
5. **Odometer**: `GET /v2/vehicles/{vehicleId}/odometer`

### 3.2 Complete Car Data Available

From `/home/darren/hyundai-logger/internal/api/types.go`:

#### Vehicle Information
- Vehicle ID, VIN, Nickname
- Year, Make, Model, Color
- Generation, Registration Date

#### Engine Status
- Engine running state
- Remote start state
- RPM
- Range (km and miles)

#### Climate Control
- Active/Inactive status
- Target temperature
- Interior temperature
- Exterior temperature
- AC on/off
- Heater on/off
- Auto mode
- Fan speed level

#### Door Locks & Security
- All doors locked (boolean)
- Individual door status (front-left, front-right, back-left, back-right)
- Trunk status
- Hood status

#### Battery (12V)
- Level (percentage)
- Voltage (volts)
- Charge time
- Warning light status

#### Tire Pressure
- Front-left PSI and status
- Front-right PSI and status
- Rear-left PSI and status
- Rear-right PSI and status
- Tire warning light status

#### Other Status
- Odometer reading
- Fuel level (percentage)
- Defrost status
- Steering wheel heat
- Side mirror heat
- Rear window heat
- Washer fluid level

#### EV-Specific Data (EVStatus struct)

**CRITICAL FOR YOUR USE CASE - Battery and Charging Information:**

```go
type EVStatus struct {
    BatteryLevel         float64   // State of Charge (%)
    BatteryCapacity      float64   // Total capacity (kWh)
    Charging             bool      // Currently charging? [KEY]
    ChargingPower        float64   // Current charging power (kW)
    EstimatedCurrentCharge int     // Time to current charge level
    EstimatedFullCharge  int       // Time to full charge
    RangeKM              float64   // Estimated range in km
    RangeMiles           float64   // Estimated range in miles
    PluggedIn            bool      // Charger connected?
    ChargeTargetPercent  int       // User's target charge %
    ChargeEndTime        time.Time // When charging will complete
}
```

#### GPS Location Data
- Latitude, Longitude
- Altitude
- Speed
- Heading

---

## 4. Configuration and Scheduling Mechanisms

### 4.1 Configuration System

From `/home/darren/hyundai-logger/internal/config/config.go`:

#### YAML Configuration File (config.yaml)

```yaml
# Hyundai Bluelink API Configuration
hyundai:
  username: "your-email@example.com"
  password: "your-password"
  pin: "1234"
  brand: "hyundai"  # or "kia"
  region: "US"      # US, CA, EU, etc.
  use_stamps: false # EU regions may need stamp auth
  stamp_url: ""

# Rate limiting configuration
rate_limit:
  requests_per_hour: 12      # API calls per hour
  poll_interval_minutes: 5    # How often to poll

# Database configuration (TimescaleDB/PostgreSQL)
database:
  host: "localhost"
  port: 5432
  user: "hyundai_logger"
  password: "your-db-password"
  dbname: "hyundai_data"
  sslmode: "disable"

# Logging configuration
logging:
  level: "info"  # debug, info, warn, error
  file: "hyundai-logger.log"
```

#### Environment Variable Overrides

All config values can be overridden by environment variables (takes precedence):

```bash
HYUNDAI_USERNAME           # Hyundai account email
HYUNDAI_PASSWORD           # Hyundai account password
HYUNDAI_PIN                # Hyundai account PIN
HYUNDAI_BRAND              # Brand: hyundai or kia
HYUNDAI_REGION             # Region: US, CA, EU

DB_HOST                    # Database hostname
DB_PORT                    # Database port
DB_USER                    # Database user
DB_PASSWORD                # Database password
DB_NAME                    # Database name
DB_SSLMODE                 # SSL mode

POLL_INTERVAL_MINUTES      # Polling interval
REQUESTS_PER_HOUR          # Rate limit
```

#### Configuration Validation

The `Validate()` method checks:
- All required Hyundai credentials present
- All required database settings present
- Poll interval > 0
- Requests per hour > 0

### 4.2 Scheduling Mechanism

The scheduling is time-based using Go's `time.Ticker`:

```go
ticker := time.NewTicker(l.pollInterval)
defer ticker.Stop()

// Immediate poll on startup
l.pollAllVehicles(ctx, vehicles)

// Then wait for each interval
for {
    select {
    case <-ctx.Done():          // Shutdown via context
        return
    case <-l.stopCh:            // Shutdown signal
        return
    case <-ticker.C:            // Timer fired - poll all vehicles
        l.pollAllVehicles(ctx, vehicles)
    }
}
```

**Important Notes on Scheduling:**
- **Not cron-based** - Simple interval-based scheduler
- **Fixed delay** - Poll every N minutes consistently
- **No job queue** - All vehicles polled synchronously in sequence
- **Rate limiting integrated** - Each API call respects the rate limiter
- **Graceful shutdown** - Stops cleanly on signals

---

## 5. Database Schema and EV Data Storage

From `/home/darren/hyundai-logger/internal/database/database.go`:

### Database Design

Uses **TimescaleDB hypertables** for efficient time-series data storage:

### Table 1: vehicles (Regular Table)

```sql
CREATE TABLE vehicles (
    vehicle_id VARCHAR(100) PRIMARY KEY,
    vin VARCHAR(17) NOT NULL,
    nickname VARCHAR(100),
    year INTEGER,
    make VARCHAR(50),
    model VARCHAR(50),
    color VARCHAR(50),
    generation VARCHAR(50),
    registered_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

Stores static vehicle metadata, upserted on startup.

### Table 2: vehicle_status (Hypertable - Time Series)

```sql
CREATE TABLE vehicle_status (
    time TIMESTAMPTZ,              -- Timestamp of data point
    vehicle_id VARCHAR(100),
    vin VARCHAR(17),
    
    -- Engine data
    engine_running BOOLEAN,
    engine_remote_start BOOLEAN,
    engine_rpm INTEGER,
    engine_range_km DOUBLE PRECISION,
    engine_range_miles DOUBLE PRECISION,
    
    -- Climate data (8 columns)
    climate_active BOOLEAN,
    climate_target_temp DOUBLE PRECISION,
    climate_interior_temp DOUBLE PRECISION,
    climate_exterior_temp DOUBLE PRECISION,
    climate_air_condition BOOLEAN,
    climate_heater BOOLEAN,
    climate_auto_mode BOOLEAN,
    climate_fan_speed INTEGER,
    
    -- Doors data (7 columns)
    doors_locked BOOLEAN,
    door_front_left BOOLEAN,
    door_front_right BOOLEAN,
    door_back_left BOOLEAN,
    door_back_right BOOLEAN,
    door_trunk BOOLEAN,
    door_hood BOOLEAN,
    
    -- Battery data (12V)
    battery_level DOUBLE PRECISION,
    battery_voltage DOUBLE PRECISION,
    battery_charge_time INTEGER,
    battery_warning_light BOOLEAN,
    
    -- Tire data (all 4 tires)
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
    
    PRIMARY KEY (time, vehicle_id)
);

-- Convert to hypertable with 1-day chunks
SELECT create_hypertable('vehicle_status', 'time',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '1 day'
);

-- Index for optimized queries
CREATE INDEX idx_vehicle_status_vehicle_time ON vehicle_status (vehicle_id, time DESC);
```

### Table 3: ev_status (Hypertable - Time Series) - EV SPECIFIC

```sql
CREATE TABLE ev_status (
    time TIMESTAMPTZ,              -- Timestamp of data point
    vehicle_id VARCHAR(100),
    vin VARCHAR(17),
    
    -- BATTERY AND CHARGING STATE
    battery_level DOUBLE PRECISION,        -- State of Charge (%)
    battery_capacity DOUBLE PRECISION,     -- Total capacity (kWh)
    charging BOOLEAN,                      -- Currently charging? [KEY]
    charging_power DOUBLE PRECISION,       -- Current power (kW)
    estimated_current_charge INTEGER,     -- Time to current %
    estimated_full_charge INTEGER,        -- Time to full charge
    
    -- Range and efficiency
    range_km DOUBLE PRECISION,             -- Estimated range (km)
    range_miles DOUBLE PRECISION,          -- Estimated range (miles)
    
    -- Charging details
    plugged_in BOOLEAN,                    -- Charger connected?
    charge_target_percent INTEGER,        -- User's target %
    charge_end_time TIMESTAMPTZ,          -- When charge will complete
    
    PRIMARY KEY (time, vehicle_id)
);

-- Convert to hypertable
SELECT create_hypertable('ev_status', 'time',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '1 day'
);

-- Index for optimized queries
CREATE INDEX idx_ev_status_vehicle_time ON ev_status (vehicle_id, time DESC);
```

**KEY COLUMNS FOR YOUR USE CASE:**
- `ev_status.charging` - Boolean flag indicating if vehicle is actively charging
- `ev_status.battery_level` - State of Charge percentage (0-100%)
- `ev_status.charging_power` - Current charging power in kW
- `ev_status.charge_end_time` - When charging session will complete

### Table 4: vehicle_location (Hypertable - Time Series)

```sql
CREATE TABLE vehicle_location (
    time TIMESTAMPTZ,
    vehicle_id VARCHAR(100),
    vin VARCHAR(17),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    altitude DOUBLE PRECISION,
    speed DOUBLE PRECISION,
    heading DOUBLE PRECISION,
    
    PRIMARY KEY (time, vehicle_id)
);

-- Convert to hypertable
SELECT create_hypertable('vehicle_location', 'time',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '1 day'
);

-- Index
CREATE INDEX idx_vehicle_location_vehicle_time ON vehicle_location (vehicle_id, time DESC);
```

### Data Insertion Operations

From the `Logger.pollVehicle()` method:

```go
// Main status data
db.InsertVehicleStatus(ctx, status)

// EV-specific data (if vehicle is EV)
if status.EV != nil {
    db.InsertEVStatus(ctx, status.Timestamp, vehicle.VehicleID, vehicle.VIN, status.EV)
}

// Location data
db.InsertLocation(ctx, location)
```

---

## 6. Logging Infrastructure

From `/home/darren/hyundai-logger/internal/logging/logger.go`:

### Logger Configuration

```go
type Config struct {
    LogFilePath string // Full path to log file
    LogLevel    string // debug, info, warn, error
    LogToFile   bool   // Write to file
    LogToStdout bool   // Write to stdout (console)
}
```

### Log Output

The logger writes to:
1. **File**: `/var/log/hyundai-logger/hyundai-logger.log` (or configured path)
2. **Stdout**: Console output (if LogToStdout=true)

### Log Levels and Formatting

- **INFO**: General information messages (INFO: prefix)
- **ERROR**: Error messages with file/line info (ERROR: prefix)
- **DEBUG**: Detailed debugging info (DEBUG: prefix)

Each log entry includes:
- Timestamp (UTC)
- Log level prefix
- Source file and line number (for errors/debug)

### Custom Logging Methods

High-level logging methods for specific events:

```go
// Data collection
LogDataCollection(vin, odometer, fuelLevel)      // "Collected data for..."
LogEVData(vin, batteryLevel, charging)           // "EV data for..."
LogLocation(vin, lat, lon)                       // "Location for..."

// Polling cycle
LogPollStart()                                     // "Starting vehicle data poll..."
LogPollComplete(duration)                        // "Poll completed in X seconds"

// Authentication
LogAuthentication(success, region)                // Success/failure message

// Startup/Shutdown
LogStartup(version, region, brand, pollInterval)
LogShutdown()

// Rate limiting
LogRateLimit(requestsPerHour)
```

---

## 7. Main Application Flow

From `/home/darren/hyundai-logger/cmd/hyundai-logger/main.go`:

### Initialization Sequence

1. **Parse flags**
   - `-config`: Path to configuration file (default: "config.yaml")
   - `-init-db`: Initialize database schema and exit

2. **Load configuration**
   - Read YAML file
   - Override with environment variables

3. **Initialize logger**
   - Set up file and console logging
   - Determine log file location

4. **Connect to database**
   - Create connection pool (min 2, max 10 connections)
   - Test connectivity

5. **Optional: Initialize database schema**
   - Create tables if running with `-init-db` flag
   - Enable TimescaleDB extension

6. **Create API client**
   - Initialize with credentials from config
   - Set rate limiting from config

7. **Create data logger**
   - Initialize with API client and database

8. **Start data logger**
   - Authenticate with Hyundai API
   - Fetch vehicles
   - Start polling goroutine

9. **Wait for signals**
   - Listen for SIGINT (Ctrl+C) or SIGTERM
   - Gracefully shut down on signal
   - Stop polling, close connections

### Command Line Examples

```bash
# Initialize database schema
go run cmd/hyundai-logger/main.go -init-db

# Run with default config.yaml
go run cmd/hyundai-logger/main.go

# Run with custom config
go run cmd/hyundai-logger/main.go -config /etc/hyundai-logger/config.yaml

# With environment variables
HYUNDAI_USERNAME=email@example.com \
HYUNDAI_PASSWORD=pass \
HYUNDAI_PIN=1234 \
go run cmd/hyundai-logger/main.go
```

---

## 8. Key Insights and Constraints

### Charging State and SoC Availability

**YES** - Charging state and State of Charge (battery %) are **fully supported**:

- **Charging Status**: Available in `EVStatus.Charging` (boolean)
- **Battery Level (SoC)**: Available in `EVStatus.BatteryLevel` (percentage)
- **Charging Power**: Available in `EVStatus.ChargingPower` (kW)
- **Charge Completion Time**: Available in `EVStatus.ChargeEndTime` (timestamp)
- **Plugged In Status**: Available in `EVStatus.PluggedIn` (boolean)
- **Charge Target**: Available in `EVStatus.ChargeTargetPercent` (%)

These are stored in the `ev_status` table and logged with every poll cycle.

### Rate Limiting Behavior

- **Token Bucket Algorithm**: Smooth rate limiting preventing sudden spikes
- **Current Setup**: 12 requests/hour means one API call every 5 minutes on average
- **Burst Protection**: Burst of 1 token prevents multiple requests in succession
- **Per-Vehicle**: All vehicles polled sequentially within single API call batch, not per vehicle

### Polling Characteristics

- **Fixed Interval**: Simple ticker-based (no variable scheduling)
- **Synchronous**: All vehicles polled in series, not parallel
- **Immediate First Poll**: Polls immediately on startup, then waits for interval
- **Complete Data Set**: Each poll fetches status, location, and vehicle metadata in one cycle
- **EV Detection**: EV status only collected for vehicles with `EVStatus != nil`

### Database Features

- **TimescaleDB Optimization**: Hypertables provide compression, parallel scanning, and automatic partitioning
- **Retention**: No automatic data pruning (manual cleanup via SQL recommended)
- **Query Performance**: Time-bucketing functions available for aggregations
- **Continuous Aggregates**: Pre-computed materialized views supported

### API Limitations

- **Unofficial API**: Uses reverse-engineered endpoints (not officially supported)
- **Token Refresh**: Tokens obtained at startup, no automatic refresh mechanism
- **EU Stamping**: Requires additional cryptographic signature for EU regions (basic support only)
- **Multiple Vehicles**: Full support for multiple vehicles in single account

---

## Summary Table

| Component | Details |
|-----------|---------|
| **Language** | Go 1.23+ |
| **Database** | PostgreSQL + TimescaleDB |
| **Polling Interval** | Configurable (default: 5 minutes) |
| **Rate Limit** | Configurable requests/hour (default: 12) |
| **Data Points** | 40+ per poll (main) + 13+ per EV poll |
| **EV Support** | Full (battery %, charging, range, etc.) |
| **Charging Data** | YES - Charging boolean + multiple related fields |
| **SoC Data** | YES - Battery level percentage stored per poll |
| **Storage Strategy** | Time-series hypertables (1-day chunks) |
| **Indexes** | Vehicle ID + Time DESC for fast lookups |
| **Configuration** | YAML + Environment Variables |
| **Logging** | File + Console, structured with levels |
| **Graceful Shutdown** | Signal handling (SIGINT, SIGTERM) |

---

## File Reference Guide

| File Path | Purpose |
|-----------|---------|
| `/internal/api/client.go` | API client with rate limiting and HTTP methods |
| `/internal/api/types.go` | All data structure definitions including EVStatus |
| `/internal/database/database.go` | Schema definition and database operations |
| `/internal/database/logger.go` | **Polling loop implementation** |
| `/internal/config/config.go` | Configuration loading and validation |
| `/internal/logging/logger.go` | Logging infrastructure |
| `/cmd/hyundai-logger/main.go` | Application entry point and orchestration |
| `config.yaml` | Configuration file template |
| `queries.sql` | Example queries for data analysis |

