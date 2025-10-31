# Quick Start Guide

Get up and running with Hyundai Logger in 5 minutes!

## ⚠️ WARNING: 12V Battery Drain Risk

**IMPORTANT:** Excessive polling can drain your vehicle's 12V battery! This application wakes up your vehicle's systems every time it polls for data, consuming battery power.

**Safe default:** 12 requests per hour (every 5 minutes) ✅
**Never exceed:** 60 requests per hour ⚠️

The default configuration uses safe, conservative settings. Do NOT modify polling rates unless you understand the risks!

## Before You Begin: Gather Your Credentials

You'll need the following information from your Hyundai Bluelink or Kia Connect app:

### 1. Username and Password
Use your Bluelink/Kia Connect app login credentials.

### 2. PIN (4-digit)
This is your **remote services PIN** used in the mobile app for commands like remote start or door lock.

**How to find your PIN:**
- Open the Hyundai Bluelink (myHyundai) or Kia Connect app
- Go to: **Settings → Profile → Change PIN**
- If you don't have a PIN set, create one in the app first
- **Note:** This is NOT your vehicle ignition PIN - it's the app-specific PIN

### 3. Region
Choose based on where you **registered your account**:
- **US** - United States (myHyundai)
- **CA** - Canada (myHyundai)
- **EU** - Europe (Bluelink)

**Important:** Use your account's registration region, not your current location.

## Method 1: Docker (Easiest)

### Using Make Commands (Recommended)

1. **Install Docker and Docker Compose**

2. **Deploy with a single command:**
   ```bash
   make docker-deploy
   ```

   This will:
   - Check for .env file (creates from .env.example if missing)
   - Build the Docker image
   - Start InfluxDB, Hyundai Logger, and Grafana
   - Set up all volumes for persistent data

3. **View logs:**
   ```bash
   make docker-logs
   ```

4. **Other useful commands:**
   ```bash
   make docker-status   # Check container status
   make docker-stop     # Stop all containers
   make docker-restart  # Restart containers
   ```

### Using Docker Compose Directly

If you prefer not to use Make:

1. **Install Docker and Docker Compose**

2. **Configure credentials:**
   ```bash
   cp .env.example .env
   nano .env  # Add your credentials (see comments in file for guidance)
   ```

3. **Start everything:**
   ```bash
   docker-compose up -d
   ```

4. **Check logs:**
   ```bash
   docker-compose logs -f hyundai-logger
   ```

That's it! Your data is being logged to InfluxDB. Access the InfluxDB UI at http://localhost:8086

**Data Persistence:** All data is stored in Docker volumes, and your .env and config.yaml remain on the host for easy editing and upgrades.

## Method 2: Manual Installation

### Step 1: Install Prerequisites

**Ubuntu/Debian:**
```bash
# Install InfluxDB v2
wget https://dl.influxdata.com/influxdb/releases/influxdb2-2.7.0-amd64.deb
sudo dpkg -i influxdb2-2.7.0-amd64.deb
sudo systemctl start influxdb
sudo systemctl enable influxdb

# Install Go (if not already installed)
sudo apt-get update
sudo apt-get install golang-go
```

**macOS:**
```bash
# Install InfluxDB v2
brew install influxdb
brew services start influxdb

# Install Go (if not already installed)
brew install go
```

### Step 2: Set Up InfluxDB

**Option A: Web UI (Recommended)**
1. Open http://localhost:8086
2. Click "Get Started"
3. Create initial user with username/password
4. Set organization name to `hyundai`
5. Set bucket name to `vehicle_data`
6. Set retention to 90 days
7. Copy the generated API token - you'll need this!

**Option B: CLI**
```bash
influx setup \
  --username admin \
  --password changeme123 \
  --org hyundai \
  --bucket vehicle_data \
  --retention 90d \
  --force
```

### Step 3: Configure Application

```bash
# Copy environment template
cp .env.example .env

# Edit with your credentials
nano .env
```

**Required settings:**
- `HYUNDAI_USERNAME`: Your Bluelink email address
- `HYUNDAI_PASSWORD`: Your Bluelink password
- `HYUNDAI_PIN`: Your 4-digit remote services PIN (from app)
- `HYUNDAI_BRAND`: Either `hyundai` or `kia`
- `HYUNDAI_REGION`: Region code (`US`, `CA`, or `EU`)
- `INFLUXDB_URL`: Usually `http://localhost:8086`
- `INFLUXDB_TOKEN`: The token from Step 2
- `INFLUXDB_ORG`: Usually `hyundai`
- `INFLUXDB_BUCKET`: Usually `vehicle_data`

See the comments in `.env.example` for detailed guidance!

### Step 4: Build and Run

```bash
# Install dependencies
make install

# Build the application
make build

# Initialize database bucket (creates bucket if it doesn't exist)
./scripts/init-influxdb.sh
# or: make init-db

# Start logging
./hyundai-logger
```

## Verify It's Working

### Check the logs
You should see output like:
```
Hyundai Logger v1.0.0
Loaded configuration from config.yaml
Successfully authenticated with Hyundai API
Found 1 vehicle(s)
Stored vehicle: 2023 Hyundai IONIQ 5 (VIN: ...)
Polling vehicle data...
Stored status for ... - Odometer: 5432.1, Fuel: 85.0%
```

### Query the database

**Via Web UI:**
Visit http://localhost:8086 and use the Data Explorer to query your data.

**Via CLI:**
```bash
influx query --org hyundai 'from(bucket: "vehicle_data")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_status")
  |> last()'
```

**Example Flux query (in InfluxDB UI):**
```flux
// See latest vehicle status
from(bucket: "vehicle_data")
  |> range(start: -24h)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_status")
  |> last()

// See EV battery level over time
from(bucket: "vehicle_data")
  |> range(start: -7d)
  |> filter(fn: (r) => r["_measurement"] == "vehicle_ev")
  |> filter(fn: (r) => r["_field"] == "battery_level")
```

## Common Issues

### Authentication Failed
- Verify credentials work in the official Hyundai Bluelink app
- Double-check your 4-digit PIN (Settings → Profile → Change PIN in the app)
- **Ensure your region matches where you registered your account** (not current location)
- For EU: May need stamp authentication (see README)

### Wrong PIN Error
- This is the **app remote services PIN**, not your vehicle ignition PIN
- Find it in the app: Settings → Profile → Change PIN
- If you haven't set one, create it in the app first

### Database Connection Failed
```bash
# Check if InfluxDB is running
sudo systemctl status influxdb

# Check InfluxDB health
curl http://localhost:8086/health

# Verify with CLI
influx ping --host http://localhost:8086
```

### No Data Being Logged
- Check if your vehicle is powered on
- Verify API rate limiting isn't too restrictive
- Look for errors in the logs

## Next Steps

### Run as a Service (Linux)

1. Edit the service file:
   ```bash
   nano hyundai-logger.service
   # Update User, Group, and paths
   ```

2. Install and start:
   ```bash
   sudo cp hyundai-logger.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable hyundai-logger
   sudo systemctl start hyundai-logger
   ```

3. Check status:
   ```bash
   sudo systemctl status hyundai-logger
   sudo journalctl -u hyundai-logger -f
   ```

### View Your Data

Open the InfluxDB UI at http://localhost:8086 and use the Data Explorer to:
- Browse your measurements (vehicle_status, vehicle_ev, vehicle_location, etc.)
- Create custom Flux queries
- Visualize your vehicle data with graphs
- Set up dashboards for monitoring

See the README.md for example Flux queries to get started!

### Set Up Grafana (Optional)

Grafana is included in docker-compose.yml:
```bash
# Access at http://localhost:3000
# Default: admin/admin

# Add InfluxDB as a data source:
# URL: http://influxdb:8086
# Organization: hyundai
# Token: (from your INFLUXDB_TOKEN in .env)
# Default Bucket: vehicle_data
```

## Configuration Tips

### Battery-Safe Settings (RECOMMENDED)
To avoid draining your 12V battery, use conservative polling:
```yaml
rate_limit:
  requests_per_hour: 12  # Safe: every 5 minutes
  poll_interval_minutes: 5

  # Intelligent overnight scheduling (less frequent polling at night)
  periods:
    - start_hour: 6
      end_hour: 22
      interval_minutes: 5    # Daytime: every 5 minutes
    - start_hour: 22
      end_hour: 6
      interval_minutes: 15   # Nighttime: every 15 minutes
```

### More Frequent Updates (USE WITH CAUTION)
Only if you need more frequent data and accept the battery drain risk:
```yaml
rate_limit:
  requests_per_hour: 24  # Acceptable: every 2.5 minutes
  poll_interval_minutes: 2
```

**WARNING:** Never exceed 60 requests per hour unless actively monitoring for a short period!

## Getting Help

- Check the detailed [README.md](README.md)
- Visit the InfluxDB UI at http://localhost:8086 to explore your data
- Look at the logs for error messages
- Review example Flux queries in the README

## What Data Is Collected?

At your configured interval (default: every 5 minutes), the logger collects:

**Basic Status:**
- Odometer reading
- Fuel level
- Door lock status
- Engine status

**Climate:**
- Interior/exterior temperature
- Climate control settings
- Heater/AC status

**Vehicle Health:**
- 12V battery voltage
- Tire pressure (all 4 tires)
- Washer fluid level

**Location:**
- GPS coordinates
- Speed and heading
- Altitude

**EV-Specific (if applicable):**
- High-voltage battery level and capacity
- Charging status and power
- Estimated range
- Charging completion time
- Plugged-in status

All data is stored in InfluxDB v2 as time-series measurements for efficient analysis and visualization!
