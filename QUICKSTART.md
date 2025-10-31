# Quick Start Guide

Get up and running with Hyundai Logger in 5 minutes!

## Method 1: Docker (Easiest)

1. **Install Docker and Docker Compose**

2. **Configure credentials:**
   ```bash
   cp .env.example .env
   nano .env  # Add your Hyundai credentials
   ```

3. **Start everything:**
   ```bash
   docker-compose up -d
   ```

4. **Check logs:**
   ```bash
   docker-compose logs -f hyundai-logger
   ```

That's it! Your data is being logged to TimescaleDB.

## Method 2: Manual Installation

### Step 1: Install Prerequisites

**Ubuntu/Debian:**
```bash
# Install PostgreSQL with TimescaleDB
sudo apt-get update
sudo apt-get install postgresql-14 postgresql-client-14
sudo add-apt-repository ppa:timescale/timescaledb-ppa
sudo apt-get update
sudo apt-get install timescaledb-postgresql-14
```

**macOS:**
```bash
brew install postgresql timescaledb
brew services start postgresql
```

### Step 2: Set Up Database

```bash
# Connect to PostgreSQL
sudo -u postgres psql

# Run these SQL commands:
CREATE DATABASE hyundai_data;
CREATE USER hyundai_logger WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE hyundai_data TO hyundai_logger;
\c hyundai_data
CREATE EXTENSION IF NOT EXISTS timescaledb;
\q
```

### Step 3: Configure Application

```bash
# Copy environment template
cp .env.example .env

# Edit with your credentials
nano .env
```

Required settings:
- `HYUNDAI_USERNAME`: Your Bluelink email
- `HYUNDAI_PASSWORD`: Your Bluelink password
- `HYUNDAI_PIN`: Your 4-digit PIN
- `HYUNDAI_REGION`: Your region (US, CA, EU)
- `DB_PASSWORD`: The password you set above

### Step 4: Build and Run

```bash
# Run the setup script
chmod +x setup.sh
./setup.sh

# Initialize database
./hyundai-logger -init-db

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
```bash
psql -h localhost -U hyundai_logger -d hyundai_data
```

```sql
-- See your vehicles
SELECT * FROM vehicles;

-- See latest status
SELECT * FROM vehicle_status ORDER BY time DESC LIMIT 5;
```

## Common Issues

### Authentication Failed
- Verify credentials work in the official Hyundai app
- Check your region setting matches your account
- For EU: May need stamp authentication (see README)

### Database Connection Failed
```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Test connection
psql -h localhost -U hyundai_logger -d hyundai_data
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

Use the example queries in [queries.sql](queries.sql):
```bash
psql -h localhost -U hyundai_logger -d hyundai_data -f queries.sql
```

### Set Up Grafana (Optional)

Grafana is included in docker-compose.yml:
```bash
# Access at http://localhost:3000
# Default: admin/admin

# Add TimescaleDB as a data source:
# Host: timescaledb:5432
# Database: hyundai_data
# User: hyundai_logger
# Password: (from your .env)
```

## Configuration Tips

### Battery-Safe Settings
To avoid draining your 12V battery:
```yaml
rate_limit:
  requests_per_hour: 12  # Max 12 per hour
  poll_interval_minutes: 5
```

### More Frequent Updates
If you need more frequent data:
```yaml
rate_limit:
  requests_per_hour: 24  # Every 2.5 minutes
  poll_interval_minutes: 2
```

## Getting Help

- Check the detailed [README.md](README.md)
- Review [queries.sql](queries.sql) for example queries
- Look at the logs for error messages

## What Data Is Collected?

Every 5 minutes (configurable), the logger collects:

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
- Battery level and capacity
- Charging status and power
- Estimated range
- Charging completion time

All data is stored in TimescaleDB for efficient time-series analysis!
