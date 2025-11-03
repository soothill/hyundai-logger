# Hyundai Logger - Updated for New Authentication

This is an updated version of the Hyundai Logger that supports the new OAuth2 authentication system with device ID registration.

## ⚠️ Important Changes

Hyundai/Kia have updated their authentication system to use OAuth2 with reCAPTCHA validation. This means:
- Direct username/password authentication no longer works
- You need to obtain a refresh token through browser-based authentication
- Device ID is now required and managed automatically

## 🚀 Quick Start

### Step 1: Obtain Refresh Token

First, you need to obtain a refresh token using the provided Python script:

```bash
# Install Python dependencies
pip3 install selenium requests

# Run the token fetcher
python3 scripts/get_refresh_token.py <REGION> <BRAND>

# Example:
python3 scripts/get_refresh_token.py EU hyundai
```

**Supported Regions:** EU, US, CA  
**Supported Brands:** hyundai, kia

The script will:
1. Open Chrome browser automatically
2. Navigate to the Hyundai/Kia login page
3. Wait for you to complete login (including reCAPTCHA)
4. Capture the authorization code
5. Exchange it for refresh token
6. Save tokens to `~/.hyundai-logger/tokens.json`

### Step 2: Configure the Logger

Edit `config.yaml` with your details:

```yaml
hyundai:
  pin: "1234"  # Your 4-digit PIN from the mobile app
  brand: "hyundai"  # or "kia"
  region: "EU"  # EU, US, or CA
  refresh_token: "YOUR_REFRESH_TOKEN_HERE"  # From step 1

database:
  url: "http://localhost:8086"
  token: "your-influxdb-token"
  organization: "hyundai"
  bucket: "vehicle_data"
```

### Step 3: Initialize Database

```bash
# Build the application
go build -o hyundai-logger cmd/hyundai-logger/main.go

# Initialize InfluxDB schema
./hyundai-logger -init-db
```

### Step 4: Run the Logger

```bash
# Run with config file
./hyundai-logger -config config.yaml

# Or run with environment variables
export HYUNDAI_REFRESH_TOKEN="your-refresh-token"
export HYUNDAI_PIN="1234"
export HYUNDAI_REGION="EU"
export HYUNDAI_BRAND="hyundai"
./hyundai-logger
```

## 🔧 Configuration Options

### Authentication Methods

1. **Refresh Token (Recommended)**
   - Use the Python script to obtain refresh token
   - Token remains valid for extended periods
   - No password needed in config

2. **Existing Tokens**
   - If you already have tokens from another app
   - Place them in `~/.hyundai-logger/tokens.json`

### Environment Variables

All configuration options can be set via environment variables:

```bash
HYUNDAI_USERNAME=your-email@example.com
HYUNDAI_PASSWORD=your-password  # Not needed with refresh token
HYUNDAI_PIN=1234
HYUNDAI_BRAND=hyundai
HYUNDAI_REGION=EU
HYUNDAI_REFRESH_TOKEN=your-refresh-token
HYUNDAI_DEVICE_ID=auto-generated-if-empty

INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=your-token
INFLUXDB_ORG=hyundai
INFLUXDB_BUCKET=vehicle_data
```

## 📊 Data Storage

The logger stores the following data in InfluxDB:
- Vehicle metadata (VIN, model, year, color)
- Vehicle status (engine, doors, locks, fuel level)
- Location (GPS coordinates, speed, heading)
- EV status (battery level, charging state, range)
- Climate control status
- Tire pressure
- Odometer readings

## 🔒 Security Notes

- **Refresh tokens are sensitive** - keep them secure
- Tokens are stored in `~/.hyundai-logger/tokens.json` with 0600 permissions
- Never commit tokens to version control
- Use environment variables for production deployments

## 🐛 Troubleshooting

### "Invalid device ID" Error
- Delete `~/.hyundai-logger/tokens.json`
- Re-run the refresh token script
- The device ID will be regenerated

### Authentication Failed
- Refresh token may have expired
- Re-run `scripts/get_refresh_token.py`

### Chrome Not Found
- Install Chrome or Chromium browser
- The token fetcher requires Chrome for reCAPTCHA handling

### Rate Limiting
- Default: 12 requests per hour (safe for battery)
- Maximum recommended: 24 requests per hour
- Excessive polling can drain your 12V battery!

## 🛠️ Building from Source

```bash
# Clone the repository
git clone https://github.com/soothill/hyundai-logger.git
cd hyundai-logger

# Copy updated files
# (Copy all files from hyundai-logger-updated/ to your repository)

# Install dependencies
go mod download

# Build
go build -o hyundai-logger cmd/hyundai-logger/main.go

# Or use make
make build
```

## 📝 File Structure

```
hyundai-logger-updated/
├── cmd/
│   └── hyundai-logger/
│       └── main.go              # Main application
├── internal/
│   ├── api/
│   │   ├── client.go           # API client with OAuth2
│   │   └── types.go            # API data types
│   ├── auth/
│   │   ├── oauth.go            # OAuth2 authentication
│   │   └── stamp.go            # EU stamp signatures
│   ├── config/
│   │   └── config.go           # Configuration management
│   └── database/
│       └── client.go           # InfluxDB client
├── scripts/
│   └── get_refresh_token.py    # Token fetcher script
├── config.yaml                  # Example configuration
├── go.mod                       # Go dependencies
└── README.md                    # This file
```

## 🔄 Migration from Old Version

If you're migrating from the old version:

1. **Backup your data** - Export from InfluxDB if needed
2. **Get refresh token** - Run the Python script
3. **Update config** - Add `refresh_token` field
4. **Update code** - Replace with new authentication modules
5. **Test** - Run with `-verbose` flag to verify

## ⚠️ Warnings

- **Battery Drain**: Excessive polling WILL drain your 12V battery
- **API Changes**: This uses reverse-engineered APIs that may change
- **Regional Differences**: Some features vary by region
- **Unofficial**: Not affiliated with or endorsed by Hyundai/Kia

## 📄 License

This project is provided as-is for educational purposes. Use at your own risk.

## 🙏 Credits

- Original hyundai-logger concept
- bluelinky project for API research
- hyundai_kia_connect_api for authentication flow
- Community stamp service providers
