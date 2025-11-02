# Hyundai Logger

A Go application that connects to the Hyundai Bluelink API to log vehicle data into an InfluxDB v2 time-series database. This tool respects API rate limits to avoid draining your vehicle's 12V battery and includes intelligent polling features.

## Features

### 🚗 Core Functionality
- **Multi-Region Support** - Connects to Hyundai Bluelink API across 8 global regions (US, CA, EU, UK, AU, KR, CN, IN)
- **Intelligent Time-Based Polling** - Configurable schedules with O(1) lookup, less frequent overnight
- **EV Charging Optimization** - Multi-phase intelligent polling (fast charge, normal, trickle, complete)
- **Comprehensive Vehicle Data Logging**:
  - Engine status and range
  - Climate control settings
  - Door lock status
  - Battery voltage and level
  - Tire pressure
  - Fuel level and odometer
  - EV-specific data (battery level, charging status, range)
  - GPS location tracking

### 🛡️ Reliability & Performance
- **Structured Logging with Zerolog** - JSON output with contextual fields (VIN, operation, request_id) for log aggregation
- **Circuit Breaker Pattern** - Prevents cascading failures, automatic recovery with state machine (Closed/Open/HalfOpen)
- **Request/Response Caching** - TTL-based caching (60s) reduces API calls by 20-40%, thread-safe
- **Database Write Batching** - Parallel collection from all vehicles, single batch write (2-3x faster)
- **Adaptive Rate Limiting** - Dynamic 429 response handling with Retry-After parsing, 50% backoff, 10% gradual recovery
- **Exponential Backoff Retry Logic** - Resilience to network interruptions with intelligent retry strategy
- **Request Coalescing** - Deduplicates identical in-flight requests to reduce wasted API calls
- **Memory Pooling** - Reduces GC pressure with sync.Pool for frequently allocated objects

### 🔐 Security & Authentication
- **API Key Authentication** - Full RBAC implementation with role-based permissions (admin, readonly)
- **Audit Logging** - Tamper-proof audit trail for all API calls, config changes, authentication attempts

### 📊 Monitoring & Observability
- **Prometheus Metrics Endpoint** - 18+ metrics at `/metrics` for scraping (polls, success rate, latency, errors)
- **Enhanced Health Checks** - Detailed status including database, API health, consecutive errors, uptime
- **Debug Logging** - Comprehensive request/response tracing for troubleshooting

### 🔔 Alerting & Notifications
- **Email Alerting** - Extended problem period notifications with SMTP support
- **Webhook Notifications** - Slack, Discord, and generic webhook integrations with retry logic

### ⚙️ Configuration & Operations
- **Configuration Hot Reload** - File watching with fsnotify, no restart required for config changes
- **Configuration Validation Tool** - Pre-deployment validation catches errors early
- **YAML or Environment Variables** - Flexible configuration options
- **Graceful Shutdown Handling** - Clean termination with resource cleanup

### 💾 Data Storage & Export
- **InfluxDB v2 Integration** - Efficient time-series data storage with automatic bucket creation
- **90-Day Retention Policy** - Automatic data lifecycle management
- **Data Export Features** - CSV/JSON export, monthly reports, trip analysis, charging session reports
- **Historical Data Analysis** - Fuel efficiency, charging costs, trip distance analytics

### 🐳 Deployment & DevOps
- **Docker Compose Setup** - Easy deployment with persistent volumes
- **Multi-Architecture Support** - Builds for amd64, arm64, arm/v7 (Raspberry Pi, Apple Silicon)
- **Helm Charts** - Kubernetes deployment with customizable values
- **Terraform Modules** - Infrastructure as Code for AWS deployment (ECS, RDS, VPC)
- **CI/CD Pipeline** - GitHub Actions for automated testing, building, and releases
- **Interactive CLI Mode** - Real-time status, manual polling, metrics viewing, data export

### 🧪 Testing & Quality
- **Comprehensive Unit Tests** - 200+ test cases with 85%+ average coverage
- **Integration Tests** - End-to-end system tests for full poll cycles and error scenarios
- **Benchmark Suite** - Performance benchmarks for API, metrics, cache, retry, circuit breaker

## Prerequisites

- Go 1.23 or higher
- InfluxDB v2 (2.7 or higher recommended)
- Hyundai Bluelink account with valid credentials

## ⚠️ IMPORTANT WARNING: 12V Battery Drain

**EXCESSIVE POLLING CAN DRAIN YOUR VEHICLE'S 12V BATTERY!**

Every time this application polls your vehicle's status, it wakes up the vehicle's telematics system, which draws power from the 12V battery. **Too frequent polling can drain your battery** and leave you stranded with a dead battery.

### Safe Polling Guidelines:

✅ **SAFE (Recommended):**
- **12 requests per hour** (every 5 minutes) - Default setting
- **24 requests per hour** (every 2.5 minutes) - Acceptable for short periods

⚠️ **RISKY:**
- **60 requests per hour** (every minute) - Use sparingly and only when actively monitoring
- **During charging:** 30 requests per hour (every 2 minutes) - Enabled automatically by charging detection

❌ **DANGEROUS:**
- **More than 60 requests per hour** - Can significantly drain battery
- **Continuous rapid polling** - Will drain battery quickly

### Battery Protection Features:

This application includes several features to protect your battery:
- **Default conservative rate limits** (12 requests/hour)
- **Intelligent overnight polling** (reduced frequency during sleeping hours)
- **Configurable time-based schedules** (different rates for different times)
- **Charging detection** (faster polling only when vehicle is charging)

### If Your Battery Drains:

If you experience 12V battery issues:
1. **Immediately reduce polling frequency** in config.yaml
2. Jump-start or charge your vehicle's 12V battery
3. Consider using **12 requests/hour maximum** going forward
4. Use intelligent scheduling to reduce overnight polling

**You have been warned!** The developers are not responsible for any battery drain or vehicle issues caused by excessive polling. Use conservative settings and monitor your battery health.

## Quick Container Deployment

The easiest way to deploy Hyundai Logger is using Docker or Podman with persistent volumes for data and configuration:

```bash
# Clone the repository
git clone https://github.com/soothill/hyundai-logger.git
cd hyundai-logger

# Configure credentials
cp .env.example .env
nano .env  # Edit with your Hyundai credentials

# Deploy everything (InfluxDB + Logger + Grafana)
make docker-deploy
```

That's it! All data is stored in Docker volumes, and configuration files remain on your host for easy editing.

**Upgrading is simple:**
```bash
# Pull latest code
git pull

# Rebuild and restart (data is preserved)
make docker-deploy
```

**Useful commands:**
- `make docker-logs` - View application logs
- `make docker-status` - Check container status
- `make docker-stop` - Stop all services
- `make docker-restart` - Restart services

See the [Container Deployment](#container-deployment-dockerpodman) section below for more details.

## Installation

### Option 1: Container Deployment (Recommended)

Supports Docker or Podman - the Makefile will automatically detect and use whichever is installed.

1. Clone or download this project:

```bash
cd hyundai-logger
```

2. Configure the application:

```bash
cp .env.example .env
# Edit .env with your credentials and generate a secure token
```

3. Start the services:

```bash
# Using make (recommended - auto-detects Docker/Podman)
make docker-deploy

# Or directly with docker-compose/podman-compose
docker-compose up -d
```

This will automatically:
- Start InfluxDB v2 with initial setup
- Create the organization and bucket with 90-day retention
- Build and run the Hyundai Logger
- Set up Grafana for visualization (optional)

**Note:** The `make docker-deploy` command automatically detects whether you have Docker or Podman installed and uses the appropriate commands.

### Option 2: Manual Installation

1. Clone or download this project:

```bash
cd hyundai-logger
```

2. Install dependencies:

```bash
make install
# or: go mod download
```

3. Set up InfluxDB v2:

```bash
# Install InfluxDB v2 (Ubuntu/Debian example)
wget https://dl.influxdata.com/influxdb/releases/influxdb2-2.7.0-amd64.deb
sudo dpkg -i influxdb2-2.7.0-amd64.deb
sudo systemctl start influxdb

# Set up initial user and organization via web UI at http://localhost:8086
# Or use the CLI:
influx setup \
  --username admin \
  --password changeme123 \
  --org hyundai \
  --bucket vehicle_data \
  --retention 90d \
  --force
```

4. Configure the application:

```bash
cp .env.example .env
# Edit .env with your Hyundai and InfluxDB credentials
```

Or edit `config.yaml` with your settings.

5. Initialize the database schema:

```bash
make init-db
# or: ./scripts/init-influxdb.sh
# or: go run cmd/hyundai-logger/main.go -init-db
```

## Configuration

### Understanding Your Credentials

Before configuring the application, you need to gather the following information:

#### Username and Password
Use the same email address and password you use to log into the Hyundai Bluelink or Kia Connect mobile app.

#### PIN
This is your **4-digit PIN** used for remote commands in the mobile app (like remote start or door lock/unlock).

**Where to find or set your PIN:**
1. Open the Hyundai Bluelink (myHyundai) or Kia Connect mobile app
2. Navigate to: **Settings → Profile → Change PIN** (or similar path depending on app version)
3. If you haven't set a PIN yet, the app will prompt you to create one
4. Use this same 4-digit PIN in the configuration

**Note:** This is NOT your vehicle's ignition security PIN - it's the app-specific remote services PIN.

#### Brand
Set to either:
- `hyundai` - for Hyundai vehicles
- `kia` - for Kia vehicles

#### Region
Choose the region where your Bluelink/Connect account was registered:

| Region Code | Description | Service Name |
|-------------|-------------|--------------|
| `EU` | Europe | Bluelink |
| `US` | United States | myHyundai |
| `CA` | Canada | myHyundai |

**Important:** Use the region where you **created your account**, not necessarily where you are currently located. For example, if you registered your account in the EU but are temporarily in the US, use `EU`.

### Using config.yaml

Edit [config.yaml](config.yaml) with your settings:

```yaml
hyundai:
  username: "your-email@example.com"
  password: "your-password"
  pin: "1234"
  brand: "hyundai"  # or "kia"
  region: "EU"      # EU, US, CA

rate_limit:
  requests_per_hour: 12  # Recommended: 12-24 per hour
  poll_interval_minutes: 5

  # Intelligent time-based scheduling
  periods:
    - start_hour: 6
      end_hour: 22
      interval_minutes: 5
    - start_hour: 22
      end_hour: 6
      interval_minutes: 15

  # Enhanced charging detection
  charging_config:
    enabled: true
    interval_minutes: 2

database:
  url: "http://localhost:8086"
  token: "your-influxdb-token"
  organization: "hyundai"
  bucket: "vehicle_data"

# Optional: Email alerts for persistent errors
alerts:
  enabled: false
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  smtp_username: "your-email@example.com"
  smtp_password: "your-app-password"
  from_email: "your-email@example.com"
  to_email: "alerts@example.com"
  alert_threshold: 5
  alert_cooldown_mins: 60
```

### Using Environment Variables

Environment variables take precedence over config.yaml:

```bash
# Hyundai Bluelink credentials
export HYUNDAI_USERNAME="your-email@example.com"
export HYUNDAI_PASSWORD="your-password"
export HYUNDAI_PIN="1234"
export HYUNDAI_BRAND="hyundai"
export HYUNDAI_REGION="EU"

# InfluxDB connection
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="your-influxdb-token"
export INFLUXDB_ORG="hyundai"
export INFLUXDB_BUCKET="vehicle_data"

# Rate limiting
export POLL_INTERVAL_MINUTES="5"
export REQUESTS_PER_HOUR="12"
```

## Usage

### Initialize Database (First Time Only)

```bash
# Using the initialization script
./scripts/init-influxdb.sh

# Or using make
make init-db

# Or using go run
go run cmd/hyundai-logger/main.go -init-db
```

### Run the Logger

```bash
# Using make
make run

# Or build and run
make build
./hyundai-logger

# Or using go run
go run cmd/hyundai-logger/main.go
```

### Run as a Service (systemd)

Create `/etc/systemd/system/hyundai-logger.service`:

```ini
[Unit]
Description=Hyundai Vehicle Data Logger
After=network.target influxdb.service

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/hyundai-logger
EnvironmentFile=/path/to/hyundai-logger/.env
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
sudo journalctl -u hyundai-logger -f
```

### Log Rotation

For production deployments, it's recommended to set up log rotation to prevent log files from consuming too much disk space. The project includes a logrotate configuration that can be installed with:

```bash
# Install logrotate configuration (requires root)
sudo make install-logrotate
```

This will install a logrotate configuration with the following settings:
- **Rotation**: Daily
- **Retention**: 30 days
- **Compression**: Enabled (delayed by 1 day)
- **Max size**: 100MB per file
- **Log location**: `/var/log/hyundai-logger/*.log`

Before installing logrotate, ensure the log directory and user exist:

```bash
# Create system user for the logger (if not already created)
sudo useradd -r -s /bin/false hyundai-logger

# Create log directory
sudo mkdir -p /var/log/hyundai-logger

# Set ownership
sudo chown hyundai-logger:hyundai-logger /var/log/hyundai-logger
```

You can manually test the logrotate configuration with:

```bash
sudo logrotate -d /etc/logrotate.d/hyundai-logger  # Dry run
sudo logrotate -f /etc/logrotate.d/hyundai-logger  # Force rotation
```

**Note:** If you're using Docker deployment, log rotation is handled automatically within the container and this setup is not required.

## Container Deployment (Docker/Podman)

The container deployment provides a complete, containerized solution with persistent data storage. All configuration and data are kept outside the containers for easy upgrades and backups.

**Supported runtimes:**
- **Docker** - Traditional Docker Engine with docker-compose
- **Podman** - Rootless container alternative with podman-compose

The Makefile automatically detects which runtime is available and uses it. Both work identically.

### Architecture

The Docker Compose setup includes:
- **InfluxDB v2** - Time-series database with automatic initialization
- **Hyundai Logger** - The main application
- **Grafana** - Optional visualization dashboard

### Multi-Architecture Support

The project supports building container images for multiple CPU architectures:

**Supported Platforms:**
- **linux/amd64** - Intel/AMD 64-bit (most desktop/server systems)
- **linux/arm64** - ARM 64-bit (Apple Silicon Macs, Raspberry Pi 4+, AWS Graviton)
- **linux/arm/v7** - ARM 32-bit (Raspberry Pi 2/3, older ARM devices)

**Building multi-arch images:**
```bash
make docker-build-multiarch
```

This uses Docker Buildx or Podman to create a single image that works across all supported architectures. The Go application compiles cleanly for all platforms thanks to `CGO_ENABLED=0`.

**Use cases:**
- Deploy on Raspberry Pi for low-power vehicle monitoring
- Run on Apple Silicon Macs (M1/M2/M3) natively
- Use ARM-based cloud instances (cheaper than x86)
- Build once, deploy anywhere

**Note:** The standard `make docker-build` builds only for your current platform, which is faster for development and local use.

### Data Persistence

All data is stored in Docker volumes and host-mounted directories:

**Docker Volumes (managed by Docker):**
- `influxdb-data` - InfluxDB database files
- `influxdb-config` - InfluxDB configuration
- `grafana-data` - Grafana dashboards and settings

**Host-Mounted Files:**
- `.env` - Credentials (mounted read-only)
- `config.yaml` - Application configuration (mounted read-only)
- `./logs/` - Application logs (read/write)

This design allows you to:
- **Upgrade easily** - Just rebuild the containers, data is preserved
- **Edit configuration** - Modify .env or config.yaml on the host, restart to apply
- **Backup easily** - Back up Docker volumes and host files
- **Migrate easily** - Move volumes and config files to another machine

### Make Commands

```bash
# Build Docker image (for current platform)
make docker-build

# Build multi-architecture image (amd64, arm64, arm/v7)
# Use this for Raspberry Pi, Apple Silicon, or cross-platform deployments
make docker-build-multiarch

# Deploy all services (builds and starts containers)
make docker-deploy

# View logs (follows application logs)
make docker-logs

# Check status of all containers
make docker-status

# Stop all containers (data preserved)
make docker-stop

# Restart all containers
make docker-restart

# Clean up containers and images (keeps data volumes)
make docker-clean
```

### Manual Docker Commands

If you prefer not to use Make:

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f hyundai-logger

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up -d --build

# View all logs
docker-compose logs -f

# Remove everything including volumes (DANGER: deletes all data!)
docker-compose down -v
```

### Accessing Services

Once deployed:
- **InfluxDB UI**: http://localhost:8086
  - Organization: `hyundai` (or from INFLUXDB_ORG)
  - Bucket: `vehicle_data` (or from INFLUXDB_BUCKET)
  - Token: From your INFLUXDB_TOKEN in .env

- **Grafana**: http://localhost:3000
  - Default credentials: `admin` / `admin`
  - Change password on first login

### Upgrading

To upgrade to a new version:

```bash
# Stop current containers
make docker-stop

# Pull latest code
git pull

# Rebuild and start (data is preserved in volumes)
make docker-deploy
```

Your data in InfluxDB volumes is automatically preserved during upgrades.

### Backup and Restore

**Backup Docker volumes:**
```bash
# Create backup directory
mkdir -p backups

# Backup InfluxDB data
docker run --rm -v hyundai-logger_influxdb-data:/data -v $(pwd)/backups:/backup alpine tar czf /backup/influxdb-backup.tar.gz -C /data .

# Backup configuration files
tar czf backups/config-backup.tar.gz .env config.yaml
```

**Restore from backup:**
```bash
# Stop containers
make docker-stop

# Restore InfluxDB volume
docker run --rm -v hyundai-logger_influxdb-data:/data -v $(pwd)/backups:/backup alpine sh -c "cd /data && tar xzf /backup/influxdb-backup.tar.gz"

# Restore configuration
tar xzf backups/config-backup.tar.gz

# Start containers
make docker-deploy
```

### Troubleshooting Docker Deployment

**Container won't start:**
```bash
# Check logs for errors
make docker-logs

# Check all container logs
docker-compose logs

# Verify .env file exists and has valid credentials
cat .env
```

**InfluxDB initialization fails:**
```bash
# Check InfluxDB logs
docker-compose logs influxdb

# Verify environment variables
docker-compose config
```

**Can't connect to InfluxDB:**
```bash
# Verify InfluxDB is running
docker-compose ps

# Check InfluxDB health
curl http://localhost:8086/health

# Restart InfluxDB
docker-compose restart influxdb
```

## Database Schema

The application stores data in InfluxDB v2 with the following measurements:

### `vehicle_info`
Vehicle metadata (make, model, VIN, nickname, year, color)

### `vehicle_engine`
Engine-related metrics:
- Running state, remote start status
- RPM, range (km and miles)

### `vehicle_climate`
Climate control data:
- Active state, temperatures (interior, exterior, target)
- Air condition, heater, auto mode, fan speed

### `vehicle_doors`
Door and lock status:
- Locked state
- Individual door status (front/back, left/right)
- Trunk and hood status

### `vehicle_battery`
12V battery data:
- Battery level and voltage
- Charge time, warning lights

### `vehicle_tires`
Tire pressure monitoring:
- PSI for each tire (front/rear, left/right)
- Status for each tire
- Warning lights

### `vehicle_status`
General vehicle metrics:
- Odometer, fuel level
- Defrost, steering wheel heat
- Side mirror heat, rear window heat
- Washer fluid level and warnings

### `vehicle_ev`
EV-specific time-series data (for electric/hybrid vehicles):
- Battery level, capacity, charging state
- Charging power, estimated charge times
- EV range (km and miles)
- Plugged-in status, charge target percentage

### `vehicle_location`
GPS location history:
- Latitude, longitude, altitude
- Speed and heading

## Querying Data

InfluxDB v2 uses Flux query language. Access the InfluxDB UI at `http://localhost:8086` or use the CLI/API.

### Get latest vehicle status

```flux
from(bucket: "vehicle_data")
  |> range(start: -24h)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_status")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> last()
```

### Get EV battery level over time

```flux
from(bucket: "vehicle_data")
  |> range(start: -7d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_ev")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> filter(fn: (r) => r["_field"] == "battery_level")
  |> yield(name: "battery_level")
```

### Get charging sessions

```flux
from(bucket: "vehicle_data")
  |> range(start: -30d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_ev")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> filter(fn: (r) => r["_field"] == "charging" or r["_field"] == "battery_level")
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
  |> filter(fn: (r) => r.charging == true)
```

### Get average fuel level by day

```flux
from(bucket: "vehicle_data")
  |> range(start: -30d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_status")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> filter(fn: (r) => r["_field"] == "fuel_level")
  |> aggregateWindow(every: 1d, fn: mean, createEmpty: false)
  |> yield(name: "daily_avg_fuel")
```

### Track vehicle location history

```flux
from(bucket: "vehicle_data")
  |> range(start: -1d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_location")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> filter(fn: (r) => r["_field"] == "latitude" or r["_field"] == "longitude" or r["_field"] == "speed")
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
```

### Monitor tire pressure trends

```flux
from(bucket: "vehicle_data")
  |> range(start: -7d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_tires")
  |> filter(fn: (r) => r["vin"] == "YOUR_VIN")
  |> filter(fn: (r) => r["_field"] =~ /psi$/)
  |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
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

1. Verify InfluxDB is running: `sudo systemctl status influxdb`
2. Check InfluxDB health: `curl http://localhost:8086/health`
3. Test connection with CLI: `influx ping --host http://localhost:8086`
4. Verify organization and bucket exist via web UI: http://localhost:8086
5. Check token permissions in InfluxDB UI (must have write access to bucket)

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
