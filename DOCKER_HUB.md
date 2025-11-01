# Hyundai Logger

[![Docker Pulls](https://img.shields.io/docker/pulls/soothill/hyundai-logger)](https://hub.docker.com/r/soothill/hyundai-logger)
[![Docker Image Size](https://img.shields.io/docker/image-size/soothill/hyundai-logger/latest)](https://hub.docker.com/r/soothill/hyundai-logger)

A lightweight Go application that connects to the Hyundai Bluelink API to log vehicle data into InfluxDB. Perfect for monitoring your Hyundai or Kia vehicle with custom dashboards!

## Quick Start

```bash
# 1. Create configuration
cat > .env <<EOF
HYUNDAI_USERNAME=your-email@example.com
HYUNDAI_PASSWORD=your-password
HYUNDAI_PIN=1234
HYUNDAI_BRAND=hyundai
HYUNDAI_REGION=US
INFLUXDB_TOKEN=your-secure-token-here
EOF

# 2. Start with Docker Compose
curl -O https://raw.githubusercontent.com/soothill/hyundai-logger/main/docker-compose.yml
docker-compose up -d
```

That's it! Your vehicle data will start flowing into InfluxDB.

## Multi-Architecture Support

This image supports multiple platforms for maximum compatibility:

| Architecture | Platform | Devices |
|--------------|----------|---------|
| **amd64** | linux/amd64 | Intel/AMD servers, desktops |
| **arm64** | linux/arm64 | Apple Silicon (M1/M2/M3), Raspberry Pi 4+, AWS Graviton |
| **armv7** | linux/arm/v7 | Raspberry Pi 2/3, older ARM devices |

The correct architecture is automatically selected when you pull the image. Perfect for low-power Raspberry Pi deployments!

## Key Features

- ✅ **Battery-Safe Polling** - Intelligent rate limiting to protect your vehicle's 12V battery
- ✅ **Charging Detection** - Automatic increased polling when vehicle is charging
- ✅ **Time-Based Schedules** - Less frequent polling overnight to conserve battery
- ✅ **Comprehensive Data** - Engine, climate, doors, tires, EV stats, GPS location
- ✅ **InfluxDB Integration** - Efficient time-series storage with 90-day retention
- ✅ **Email Alerts** - Get notified of extended connection issues
- ✅ **Grafana Ready** - Built-in dashboard for beautiful visualizations

## Supported Regions

- **US** - United States (myHyundai)
- **CA** - Canada (myHyundai)
- **EU** - Europe (Bluelink)

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `HYUNDAI_USERNAME` | Bluelink email | `user@example.com` |
| `HYUNDAI_PASSWORD` | Bluelink password | `your-password` |
| `HYUNDAI_PIN` | 4-digit remote services PIN | `1234` |
| `HYUNDAI_BRAND` | Vehicle brand | `hyundai` or `kia` |
| `HYUNDAI_REGION` | Account region | `US`, `CA`, or `EU` |
| `INFLUXDB_TOKEN` | InfluxDB API token | Generate in InfluxDB UI |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `INFLUXDB_URL` | `http://influxdb:8086` | InfluxDB server URL |
| `INFLUXDB_ORG` | `hyundai` | InfluxDB organization |
| `INFLUXDB_BUCKET` | `vehicle_data` | InfluxDB bucket name |
| `POLL_INTERVAL_MINUTES` | `5` | Polling interval (5-15 recommended) |
| `REQUESTS_PER_HOUR` | `12` | Max requests per hour (12-24 safe) |

## ⚠️ IMPORTANT: Battery Safety

**Excessive polling can drain your vehicle's 12V battery!**

Every API request wakes your vehicle's telematics system, consuming 12V battery power.

### Safe Settings:
- ✅ **12 requests/hour** (every 5 min) - Recommended default
- ✅ **24 requests/hour** (every 2.5 min) - Acceptable
- ⚠️ **60 requests/hour** (every 1 min) - Use sparingly
- ❌ **More than 60/hour** - Can drain battery quickly

This application includes intelligent features to protect your battery:
- Conservative default rate limits
- Reduced overnight polling
- Automatic charging detection (faster polling only when charging)

## Docker Compose Example

Complete stack with InfluxDB and Grafana:

```yaml
version: '3.8'

services:
  influxdb:
    image: influxdb:2.7
    environment:
      - DOCKER_INFLUXDB_INIT_MODE=setup
      - DOCKER_INFLUXDB_INIT_USERNAME=admin
      - DOCKER_INFLUXDB_INIT_PASSWORD=changeme123
      - DOCKER_INFLUXDB_INIT_ORG=hyundai
      - DOCKER_INFLUXDB_INIT_BUCKET=vehicle_data
      - DOCKER_INFLUXDB_INIT_RETENTION=90d
      - DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=${INFLUXDB_TOKEN}
    volumes:
      - influxdb-data:/var/lib/influxdb2
      - influxdb-config:/etc/influxdb2
    ports:
      - "8086:8086"

  hyundai-logger:
    image: soothill/hyundai-logger:latest
    depends_on:
      - influxdb
    environment:
      - HYUNDAI_USERNAME=${HYUNDAI_USERNAME}
      - HYUNDAI_PASSWORD=${HYUNDAI_PASSWORD}
      - HYUNDAI_PIN=${HYUNDAI_PIN}
      - HYUNDAI_BRAND=${HYUNDAI_BRAND:-hyundai}
      - HYUNDAI_REGION=${HYUNDAI_REGION:-US}
      - INFLUXDB_URL=http://influxdb:8086
      - INFLUXDB_TOKEN=${INFLUXDB_TOKEN}
      - INFLUXDB_ORG=${INFLUXDB_ORG:-hyundai}
      - INFLUXDB_BUCKET=${INFLUXDB_BUCKET:-vehicle_data}
      - POLL_INTERVAL_MINUTES=${POLL_INTERVAL_MINUTES:-5}
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    depends_on:
      - influxdb
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_INSTALL_PLUGINS=
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  influxdb-data:
  influxdb-config:
  grafana-data:
```

## Accessing Services

After deployment:

- **InfluxDB UI**: http://localhost:8086
- **Grafana**: http://localhost:3000 (admin/admin)

## Volume Mounts

For advanced configuration:

```yaml
hyundai-logger:
  image: soothill/hyundai-logger:latest
  volumes:
    - ./config.yaml:/app/config.yaml:ro  # Custom config
    - ./logs:/app/logs                    # Persistent logs
  environment:
    # ... your env vars
```

## Troubleshooting

**Can't connect to Bluelink:**
- Verify credentials work in the official Hyundai/Kia app
- Check region matches your account (US/CA/EU)
- Ensure PIN is your 4-digit remote services PIN

**No data in InfluxDB:**
- Check container logs: `docker logs hyundai-logger`
- Verify InfluxDB is healthy: `curl http://localhost:8086/health`
- Confirm token has write permissions to bucket

**Battery draining:**
- Reduce `REQUESTS_PER_HOUR` to 12 or less
- Increase `POLL_INTERVAL_MINUTES` to 10 or 15
- Check application isn't polling too frequently

## Data Collected

The logger captures comprehensive vehicle data:

- **Engine**: Status, RPM, range
- **Climate**: Temperature, AC/heater status, fan speed
- **Doors**: Lock status, individual door states
- **Battery**: 12V voltage and level
- **Tires**: Pressure for all four tires
- **EV Data**: Battery level, charging status, EV range
- **Location**: GPS coordinates, speed, heading
- **Status**: Odometer, fuel level, defrost, warnings

## Links

- **GitHub**: https://github.com/soothill/hyundai-logger
- **Documentation**: https://github.com/soothill/hyundai-logger/blob/main/README.md
- **Issues**: https://github.com/soothill/hyundai-logger/issues
- **Changelog**: https://github.com/soothill/hyundai-logger/blob/main/CHANGELOG.md

## Disclaimer

This application is not affiliated with Hyundai Motor Company. It uses reverse-engineered API endpoints and is provided as-is for personal use. Use at your own risk and be mindful of your vehicle's battery health.

## License

MIT License - See [LICENSE](https://github.com/soothill/hyundai-logger/blob/main/LICENSE) for details.
