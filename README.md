# Hyundai Logger

A Go application that connects to the Hyundai Bluelink API to log vehicle data into a TimescaleDB time-series database. This tool respects API rate limits to avoid draining your vehicle's 12V battery.

## Features

- Connects to Hyundai Bluelink API (US, CA, EU regions supported)
- Rate limiting to protect vehicle battery (configurable requests per hour)
- Logs comprehensive vehicle data:
  - Engine status and range
  - Climate control settings
  - Door lock status
  - Battery voltage and level
  - Tire pressure
  - Fuel level and odometer
  - EV-specific data (battery level, charging status, range)
  - GPS location tracking
- TimescaleDB integration for efficient time-series data storage
- Automatic hypertable creation for optimal performance
- Graceful shutdown handling
- Configuration via YAML file or environment variables

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12+ with TimescaleDB extension
- Hyundai Bluelink account with valid credentials

## Installation

1. Clone or download this project:

```bash
cd hyundai-logger
```

2. Install dependencies:

```bash
go mod download
```

3. Set up PostgreSQL with TimescaleDB:

```bash
# Install TimescaleDB (Ubuntu/Debian example)
sudo apt-get install timescaledb-postgresql-14

# Create database and user
sudo -u postgres psql
```

```sql
CREATE DATABASE hyundai_data;
CREATE USER hyundai_logger WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE hyundai_data TO hyundai_logger;
\c hyundai_data
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

4. Configure the application:

```bash
cp .env.example .env
# Edit .env with your credentials
```

Or edit `config.yaml` with your settings.

5. Initialize the database schema:

```bash
go run cmd/hyundai-logger/main.go -init-db
```

## Configuration

### Using config.yaml

Edit [config.yaml](config.yaml) with your settings:

```yaml
hyundai:
  username: "your-email@example.com"
  password: "your-password"
  pin: "1234"
  brand: "hyundai"  # or "kia"
  region: "US"      # US, CA, EU

rate_limit:
  requests_per_hour: 12  # Recommended: 12-24 per hour
  poll_interval_minutes: 5

database:
  host: "localhost"
  port: 5432
  user: "hyundai_logger"
  password: "your-db-password"
  dbname: "hyundai_data"
```

### Using Environment Variables

Environment variables take precedence over config.yaml:

```bash
export HYUNDAI_USERNAME="your-email@example.com"
export HYUNDAI_PASSWORD="your-password"
export HYUNDAI_PIN="1234"
export HYUNDAI_BRAND="hyundai"
export HYUNDAI_REGION="US"
export DB_PASSWORD="your-db-password"
```

## Usage

### Initialize Database (First Time Only)

```bash
go run cmd/hyundai-logger/main.go -init-db
```

### Run the Logger

```bash
go run cmd/hyundai-logger/main.go
```

Or build and run:

```bash
go build -o hyundai-logger cmd/hyundai-logger/main.go
./hyundai-logger
```

### Run as a Service (systemd)

Create `/etc/systemd/system/hyundai-logger.service`:

```ini
[Unit]
Description=Hyundai Vehicle Data Logger
After=network.target postgresql.service

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/hyundai-logger
ExecStart=/path/to/hyundai-logger/hyundai-logger
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable hyundai-logger
sudo systemctl start hyundai-logger
sudo systemctl status hyundai-logger
```

## Database Schema

The application creates the following tables:

### `vehicles`
Stores vehicle information (make, model, VIN, etc.)

### `vehicle_status` (Hypertable)
Time-series data for:
- Engine status
- Climate control
- Door locks
- Battery voltage
- Tire pressure
- Fuel level
- Odometer readings

### `ev_status` (Hypertable)
EV-specific time-series data:
- Battery level and capacity
- Charging status and power
- Estimated range
- Charge completion time

### `vehicle_location` (Hypertable)
GPS location history:
- Latitude/longitude
- Altitude
- Speed and heading

## Querying Data

### Get latest vehicle status

```sql
SELECT * FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
ORDER BY time DESC
LIMIT 1;
```

### Get battery level over time

```sql
SELECT time, battery_level
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '7 days'
ORDER BY time;
```

### Get average fuel consumption

```sql
SELECT
  time_bucket('1 day', time) AS day,
  AVG(fuel_level) as avg_fuel_level,
  MAX(odometer) - MIN(odometer) as distance
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day;
```

### Track vehicle location history

```sql
SELECT time, latitude, longitude, speed
FROM vehicle_location
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '1 day'
ORDER BY time;
```

## Rate Limiting

The application implements rate limiting to avoid excessive API calls that can drain your vehicle's 12V battery. The default setting is 12 requests per hour (one every 5 minutes).

**Recommended settings:**
- **Conservative:** 12 requests/hour (5-minute intervals)
- **Moderate:** 24 requests/hour (2.5-minute intervals)
- **Maximum:** 60 requests/hour (1-minute intervals) - Not recommended

## Important Notes

### Unofficial API
This application uses reverse-engineered Hyundai Bluelink API endpoints. It is **not officially supported** by Hyundai and may stop working if they change their API.

### Battery Drain Warning
Excessive API polling can drain your vehicle's 12V battery. Use conservative rate limits (12-24 requests/hour maximum).

### EU Region
EU regions require stamp authentication (cryptographic signatures). The current implementation provides basic support but may need additional configuration.

### Authentication Tokens
Access tokens are obtained during each startup. For production use, consider implementing token refresh logic to avoid repeated authentications.

## Troubleshooting

### Authentication Failures

1. Verify your Bluelink credentials work in the official app
2. Check your region setting matches your account
3. For EU users, ensure stamp authentication is properly configured

### Database Connection Issues

1. Verify PostgreSQL is running: `sudo systemctl status postgresql`
2. Test connection: `psql -h localhost -U hyundai_logger -d hyundai_data`
3. Check TimescaleDB extension: `SELECT * FROM pg_extension WHERE extname = 'timescaledb';`

### No Data Being Logged

1. Check logs for API errors
2. Verify rate limiting isn't too restrictive
3. Ensure your vehicle is powered on and connected

## Development

### Project Structure

```
hyundai-logger/
├── cmd/
│   └── hyundai-logger/     # Main application
│       └── main.go
├── internal/
│   ├── api/                # Hyundai API client
│   │   ├── client.go
│   │   └── types.go
│   ├── config/             # Configuration management
│   │   └── config.go
│   └── database/           # Database operations
│       ├── database.go
│       └── logger.go
├── config.yaml             # Configuration file
├── .env.example            # Environment variables template
├── go.mod
└── README.md
```

### Adding New Data Points

1. Add fields to types in [internal/api/types.go](internal/api/types.go)
2. Update database schema in [internal/database/database.go](internal/database/database.go)
3. Update insert queries to include new fields
4. Run `-init-db` to update schema (or manually alter tables)

## License

This project is provided as-is for educational and personal use. Use at your own risk.

## Disclaimer

This application is not affiliated with, endorsed by, or connected to Hyundai Motor Company. Use of this application may violate Hyundai's terms of service. The developers are not responsible for any consequences of using this application, including but not limited to account suspension, vehicle battery drain, or data loss.
