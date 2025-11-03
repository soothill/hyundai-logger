#!/bin/bash

# Hyundai Logger Setup Script
# This script helps set up the environment and obtain the refresh token

set -e

echo "==========================================="
echo "   Hyundai Logger Setup Assistant"
echo "==========================================="
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check for Python 3
if ! command_exists python3; then
    echo "❌ Python 3 is not installed. Please install Python 3.6 or later."
    exit 1
fi
echo "✅ Python 3 found: $(python3 --version)"

# Check for Go
if ! command_exists go; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi
echo "✅ Go found: $(go version)"

# Check for Chrome/Chromium
if command_exists google-chrome; then
    echo "✅ Google Chrome found"
elif command_exists chromium-browser; then
    echo "✅ Chromium browser found"
elif command_exists chromium; then
    echo "✅ Chromium found"
else
    echo "❌ Chrome/Chromium not found. Required for token fetching."
    echo "   Please install Chrome or Chromium browser."
    exit 1
fi

# Install Python dependencies
echo ""
echo "Installing Python dependencies..."
pip3 install --user selenium requests || {
    echo "Failed to install Python packages. Try with sudo:"
    echo "  sudo pip3 install selenium requests"
    exit 1
}
echo "✅ Python dependencies installed"

# Check for InfluxDB
echo ""
echo "Checking for InfluxDB..."
if command_exists influx; then
    echo "✅ InfluxDB CLI found"
else
    echo "⚠️  InfluxDB CLI not found. Make sure InfluxDB is running."
    echo "   You can still continue if InfluxDB is running elsewhere."
fi

# Build the application
echo ""
echo "Building Hyundai Logger..."
make build || {
    echo "Build failed. Trying direct go build..."
    go build -o build/hyundai-logger cmd/hyundai-logger/main.go
}
echo "✅ Build successful"

# Get user input for configuration
echo ""
echo "==========================================="
echo "   Configuration Setup"
echo "==========================================="
echo ""
echo "Please provide your configuration details:"
echo ""

# Region selection
echo "Select your region:"
echo "1) EU (Europe)"
echo "2) US (United States)"
echo "3) CA (Canada)"
read -p "Enter choice [1-3]: " region_choice

case $region_choice in
    1) REGION="EU";;
    2) REGION="US";;
    3) REGION="CA";;
    *) echo "Invalid choice"; exit 1;;
esac

# Brand selection
echo ""
echo "Select your brand:"
echo "1) Hyundai"
echo "2) Kia"
read -p "Enter choice [1-2]: " brand_choice

case $brand_choice in
    1) BRAND="hyundai";;
    2) BRAND="kia";;
    *) echo "Invalid choice"; exit 1;;
esac

# PIN
echo ""
read -p "Enter your 4-digit PIN (from mobile app): " PIN
if [[ ! "$PIN" =~ ^[0-9]{4}$ ]]; then
    echo "Invalid PIN format. Must be 4 digits."
    exit 1
fi

# InfluxDB configuration
echo ""
echo "InfluxDB Configuration:"
read -p "InfluxDB URL [http://localhost:8086]: " INFLUX_URL
INFLUX_URL=${INFLUX_URL:-http://localhost:8086}

read -p "InfluxDB Organization [hyundai]: " INFLUX_ORG
INFLUX_ORG=${INFLUX_ORG:-hyundai}

read -p "InfluxDB Bucket [vehicle_data]: " INFLUX_BUCKET
INFLUX_BUCKET=${INFLUX_BUCKET:-vehicle_data}

read -p "InfluxDB Token: " INFLUX_TOKEN
if [ -z "$INFLUX_TOKEN" ]; then
    echo "InfluxDB token is required"
    exit 1
fi

# Create config file
echo ""
echo "Creating configuration file..."
cat > config.yaml << EOF
# Hyundai Logger Configuration
# Generated on $(date)

hyundai:
  pin: "$PIN"
  brand: "$BRAND"
  region: "$REGION"
  refresh_token: ""  # Will be filled after token fetch

database:
  url: "$INFLUX_URL"
  token: "$INFLUX_TOKEN"
  organization: "$INFLUX_ORG"
  bucket: "$INFLUX_BUCKET"
  batch_size: 100
  flush_interval_seconds: 10

rate_limit:
  requests_per_hour: 12
  poll_interval_minutes: 5
  periods:
    - start_hour: 6
      end_hour: 22
      interval_minutes: 5
    - start_hour: 22
      end_hour: 6
      interval_minutes: 15

charging_config:
  enabled: true
  interval_minutes: 2
  fast_charge_interval_minutes: 1
  trickle_threshold_kw: 3.0
  fast_charge_threshold_kw: 20.0

alerts:
  enabled: false
EOF

echo "✅ Configuration file created: config.yaml"

# Initialize database
echo ""
read -p "Initialize InfluxDB schema? [y/n]: " init_db
if [[ "$init_db" == "y" || "$init_db" == "Y" ]]; then
    echo "Initializing database..."
    ./build/hyundai-logger -init-db -config config.yaml || {
        echo "⚠️  Database initialization failed. You can try again later with:"
        echo "   ./build/hyundai-logger -init-db"
    }
fi

# Get refresh token
echo ""
echo "==========================================="
echo "   OAuth Token Setup"
echo "==========================================="
echo ""
echo "Now we need to get your refresh token."
echo "This will open Chrome and navigate to the Hyundai/Kia login page."
echo ""
read -p "Ready to get your refresh token? [y/n]: " get_token

if [[ "$get_token" == "y" || "$get_token" == "Y" ]]; then
    echo "Starting token fetcher..."
    python3 scripts/get_refresh_token.py "$REGION" "$BRAND" || {
        echo ""
        echo "⚠️  Token fetch failed. You can try again with:"
        echo "   python3 scripts/get_refresh_token.py $REGION $BRAND"
        echo ""
        echo "After getting the token, add it to config.yaml:"
        echo "   refresh_token: \"your-token-here\""
        exit 1
    }
    
    # Try to extract token from file and update config
    if [ -f "$HOME/.hyundai-logger/tokens.json" ]; then
        REFRESH_TOKEN=$(python3 -c "import json; print(json.load(open('$HOME/.hyundai-logger/tokens.json'))['refresh_token'])" 2>/dev/null)
        if [ ! -z "$REFRESH_TOKEN" ]; then
            # Update config file with token
            sed -i.bak "s/refresh_token: \"\"/refresh_token: \"$REFRESH_TOKEN\"/" config.yaml
            echo "✅ Refresh token added to config.yaml"
        fi
    fi
fi

echo ""
echo "==========================================="
echo "   Setup Complete!"
echo "==========================================="
echo ""
echo "You can now run the logger with:"
echo "  ./build/hyundai-logger -config config.yaml"
echo ""
echo "Or run with verbose output:"
echo "  ./build/hyundai-logger -config config.yaml -verbose"
echo ""
echo "To run as a service, check the README for systemd setup."
echo ""
echo "⚠️  Remember: Excessive polling can drain your 12V battery!"
echo "   Default is 12 requests per hour (safe)"
echo ""
