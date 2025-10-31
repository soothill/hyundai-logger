# Hyundai Logger - Project Summary

## Overview
A complete Go application that connects to the Hyundai Bluelink API and logs all available vehicle data into a TimescaleDB time-series database, with built-in rate limiting to protect your vehicle's battery.

## Project Structure

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
│   └── database/
│       ├── database.go            # Database connection and schema
│       └── logger.go              # Data collection and logging logic
├── config.yaml                     # Configuration file
├── .env.example                    # Environment variables template
├── go.mod / go.sum                 # Go dependencies
├── Makefile                        # Build and run commands
├── Dockerfile                      # Docker container definition
├── docker-compose.yml              # Docker Compose setup with TimescaleDB
├── setup.sh                        # Automated setup script
├── hyundai-logger.service          # SystemD service file
├── queries.sql                     # Example SQL queries
├── README.md                       # Complete documentation
├── QUICKSTART.md                   # Quick start guide
└── .gitignore                      # Git ignore rules
```

## Key Features

### 1. API Client ([internal/api/client.go](internal/api/client.go))
- Rate-limited HTTP client using `golang.org/x/time/rate`
- Supports US, CA, and EU regions
- Authentication handling
- Vehicle status retrieval
- Location tracking
- Odometer readings
- EV-specific data collection

### 2. Data Models ([internal/api/types.go](internal/api/types.go))
Complete data structures for:
- Vehicle information (make, model, VIN, year)
- Engine status (running, RPM, range)
- Climate control (temperature, fan speed, mode)
- Door locks and security
- Battery health (12V voltage, level)
- Tire pressure (all 4 tires)
- EV status (battery %, charging, range)
- GPS location (lat/lon, speed, heading)
- Odometer and fuel level

### 3. Database Layer ([internal/database/](internal/database/))

**Schema (database.go):**
- `vehicles` - Vehicle information table
- `vehicle_status` - Main status hypertable (time-series)
- `ev_status` - EV-specific data hypertable
- `vehicle_location` - GPS tracking hypertable
- Automatic hypertable creation with 1-day chunks
- Indexes for optimal query performance

**Logger (logger.go):**
- Periodic data collection
- Configurable polling intervals
- Concurrent vehicle polling
- Error handling and logging
- Graceful shutdown support

### 4. Configuration ([internal/config/config.go](internal/config/config.go))
- YAML file support
- Environment variable overrides
- Validation
- Multiple sources (file + env)

### 5. Main Application ([cmd/hyundai-logger/main.go](cmd/hyundai-logger/main.go))
- Command-line flags (-init-db, -config)
- Signal handling (SIGINT, SIGTERM)
- Graceful shutdown
- Structured logging

## Technical Implementation

### Rate Limiting
```go
// Prevents excessive API calls that drain 12V battery
limiter := rate.NewLimiter(rate.Limit(rps), 1)
if err := c.rateLimiter.Wait(ctx); err != nil {
    return err
}
```

Configurable via:
- `requests_per_hour`: Total API calls per hour
- `poll_interval_minutes`: How often to collect data

**Recommended:** 12 requests/hour = 5-minute intervals

### Time-Series Storage

Uses TimescaleDB hypertables for efficient storage:
```sql
SELECT create_hypertable('vehicle_status', 'time',
    chunk_time_interval => INTERVAL '1 day'
);
```

Benefits:
- Automatic data partitioning
- Fast time-range queries
- Compression support
- Retention policies

### Data Collection Flow

1. **Authentication:** Login to Hyundai API
2. **Vehicle Discovery:** Fetch all vehicles in account
3. **Periodic Polling:**
   - Get vehicle status
   - Get EV status (if applicable)
   - Get GPS location
   - Store all data with timestamp
4. **Error Handling:** Log errors, continue operation

### Database Tables

**vehicle_status** - 44 columns including:
- Engine: running, rpm, range
- Climate: temps, settings, fan speed
- Doors: lock status for all doors
- Battery: voltage, level, warnings
- Tires: pressure for all 4 tires
- Odometer and fuel level

**ev_status** - 13 columns including:
- Battery level and capacity
- Charging status and power
- Range estimates
- Charge target and completion time

**vehicle_location** - 7 columns:
- GPS coordinates
- Altitude, speed, heading
- Timestamp

## Usage Examples

### Basic Operations

```bash
# Build
make build

# Initialize database
./hyundai-logger -init-db

# Run
./hyundai-logger

# Run with custom config
./hyundai-logger -config /path/to/config.yaml
```

### Docker Deployment

```bash
# Start everything
docker-compose up -d

# View logs
docker-compose logs -f hyundai-logger

# Stop
docker-compose down
```

### SystemD Service

```bash
# Install
sudo cp hyundai-logger.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable hyundai-logger
sudo systemctl start hyundai-logger

# Monitor
sudo journalctl -u hyundai-logger -f
```

## Example Queries

### Latest vehicle status
```sql
SELECT * FROM vehicle_status
WHERE vehicle_id = 'YOUR_ID'
ORDER BY time DESC LIMIT 1;
```

### Fuel consumption over time
```sql
SELECT
    time_bucket('1 day', time) AS day,
    MAX(odometer) - MIN(odometer) as distance,
    AVG(fuel_level) as avg_fuel
FROM vehicle_status
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY day;
```

### EV charging sessions
```sql
SELECT
    time,
    battery_level,
    charging_power,
    range_km
FROM ev_status
WHERE charging = true
ORDER BY time;
```

### Location history
```sql
SELECT time, latitude, longitude, speed
FROM vehicle_location
WHERE time > NOW() - INTERVAL '7 days'
ORDER BY time;
```

## Configuration Options

### Hyundai API
- `username`: Bluelink email
- `password`: Bluelink password
- `pin`: 4-digit PIN
- `brand`: hyundai or kia
- `region`: US, CA, EU, etc.

### Rate Limiting
- `requests_per_hour`: 12 (recommended)
- `poll_interval_minutes`: 5 (recommended)

### Database
- `host`: PostgreSQL hostname
- `port`: 5432
- `user`: Database user
- `password`: Database password
- `dbname`: Database name
- `sslmode`: SSL mode (disable for local)

## Security Considerations

1. **Credentials Storage:**
   - Use `.env` file (not committed to git)
   - Or environment variables
   - Never hardcode in source

2. **API Rate Limiting:**
   - Prevents battery drain
   - Respects Hyundai's infrastructure
   - Configurable per use case

3. **Database Security:**
   - Use strong passwords
   - Enable SSL for remote connections
   - Restrict network access

4. **Service Permissions:**
   - Run as non-root user
   - Restrict file system access
   - Use systemd security features

## Extensibility

### Adding New Data Points

1. **Update API types** ([internal/api/types.go](internal/api/types.go)):
   ```go
   type VehicleStatus struct {
       NewField string `json:"newField"`
       // ...
   }
   ```

2. **Update database schema** ([internal/database/database.go](internal/database/database.go)):
   ```sql
   ALTER TABLE vehicle_status
   ADD COLUMN new_field VARCHAR(50);
   ```

3. **Update insert query:**
   ```go
   _, err := db.pool.Exec(ctx,
       "INSERT INTO vehicle_status (..., new_field) VALUES (..., $N)",
       status.NewField,
   )
   ```

### Adding New API Endpoints

1. Add method to [internal/api/client.go](internal/api/client.go)
2. Create new data types if needed
3. Add database storage logic
4. Update logger to call new endpoint

## Monitoring and Maintenance

### Health Checks
```bash
# Check service status
systemctl status hyundai-logger

# View recent logs
journalctl -u hyundai-logger -n 100

# Database connection
psql -h localhost -U hyundai_logger -d hyundai_data -c "SELECT NOW();"
```

### Database Maintenance
```sql
-- Check database size
SELECT pg_size_pretty(pg_database_size('hyundai_data'));

-- Check table sizes
SELECT tablename, pg_size_pretty(pg_total_relation_size(tablename::text))
FROM pg_tables WHERE schemaname = 'public';

-- Clean old data (example)
DELETE FROM vehicle_status WHERE time < NOW() - INTERVAL '1 year';
```

### Performance Tuning

1. **Adjust chunk interval** for hypertables based on data volume
2. **Add indexes** for frequently queried columns
3. **Enable compression** for old data
4. **Set retention policies** to auto-delete old data

## Troubleshooting

### Authentication Issues
- Verify credentials in official app
- Check region matches account
- EU users may need stamp authentication

### Database Connection
- Ensure PostgreSQL is running
- Check TimescaleDB extension is installed
- Verify connection string

### No Data Logging
- Check vehicle is powered on
- Verify API credentials
- Review logs for errors
- Confirm rate limiting settings

### High Database Size
- Implement data retention policies
- Enable TimescaleDB compression
- Archive old data to cold storage

## Performance Characteristics

### Resource Usage
- **Memory:** ~50-100 MB (includes Go runtime)
- **CPU:** Minimal (<1% most of the time)
- **Disk:** Depends on retention and poll frequency
  - ~1 MB per day per vehicle at 5-min intervals
  - ~30 MB per month per vehicle

### Scalability
- Handles multiple vehicles concurrently
- TimescaleDB scales to TBs of data
- Rate limiting prevents API throttling

## Dependencies

### Go Packages
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/joho/godotenv` - .env file support
- `golang.org/x/time/rate` - Rate limiting
- `gopkg.in/yaml.v3` - YAML parsing

### External Services
- PostgreSQL 12+ with TimescaleDB extension
- Hyundai Bluelink API (unofficial)

## Future Enhancements

Potential additions:
1. Web dashboard for real-time monitoring
2. Mobile app integration
3. Alert system (low battery, door unlocked, etc.)
4. Export to other formats (CSV, JSON)
5. Integration with Home Assistant
6. Predictive maintenance using ML
7. Trip detection and analysis
8. Fuel/energy cost tracking

## License and Disclaimer

**Unofficial Implementation:** This uses reverse-engineered APIs and is not endorsed by Hyundai.

**Use at Your Own Risk:** May violate terms of service. Could result in account suspension.

**No Warranty:** Provided as-is for educational purposes.

**Battery Drain:** Excessive polling can drain your vehicle's 12V battery.

## Support

For issues and questions:
1. Check [README.md](README.md) for detailed documentation
2. Review [QUICKSTART.md](QUICKSTART.md) for setup help
3. Examine [queries.sql](queries.sql) for query examples
4. Check application logs for errors

## Credits

This implementation is based on reverse engineering work by:
- Bluelinky (Node.js library)
- hyundai-kia-connect-api (Python library)
- The open-source community

---

**Version:** 1.0.0
**Last Updated:** 2025-10-31
**Go Version:** 1.23+
**TimescaleDB:** 2.0+
