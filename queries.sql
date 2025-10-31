-- Example SQL Queries for Hyundai Logger Database
-- Connect to the database: psql -h localhost -U hyundai_logger -d hyundai_data

-- ========================================
-- Basic Queries
-- ========================================

-- Get all vehicles
SELECT * FROM vehicles;

-- Get latest status for all vehicles
SELECT DISTINCT ON (vehicle_id)
    vehicle_id, vin, time, odometer, fuel_level,
    engine_running, battery_level, doors_locked
FROM vehicle_status
ORDER BY vehicle_id, time DESC;

-- Get latest location for all vehicles
SELECT DISTINCT ON (vehicle_id)
    vehicle_id, vin, time, latitude, longitude, speed
FROM vehicle_location
ORDER BY vehicle_id, time DESC;

-- ========================================
-- Time-Series Analysis
-- ========================================

-- Fuel level over the last 7 days (hourly buckets)
SELECT
    time_bucket('1 hour', time) AS hour,
    AVG(fuel_level) as avg_fuel_level,
    MIN(fuel_level) as min_fuel_level,
    MAX(fuel_level) as max_fuel_level
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '7 days'
GROUP BY hour
ORDER BY hour;

-- Odometer change per day
SELECT
    time_bucket('1 day', time) AS day,
    MAX(odometer) - MIN(odometer) as distance_traveled,
    COUNT(*) as readings
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day;

-- Battery voltage monitoring (12V battery health)
SELECT
    time_bucket('1 hour', time) AS hour,
    AVG(battery_voltage) as avg_voltage,
    MIN(battery_voltage) as min_voltage,
    MAX(battery_voltage) as max_voltage
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '7 days'
GROUP BY hour
ORDER BY hour;

-- ========================================
-- EV-Specific Queries
-- ========================================

-- EV battery level over time
SELECT
    time,
    battery_level,
    charging,
    range_km,
    charging_power
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '7 days'
ORDER BY time;

-- Charging sessions
SELECT
    time_bucket('1 hour', time) AS session_time,
    MAX(battery_level) - MIN(battery_level) as charge_gained,
    AVG(charging_power) as avg_power,
    COUNT(*) as readings
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND charging = true
  AND time > NOW() - INTERVAL '30 days'
GROUP BY session_time
HAVING MAX(battery_level) - MIN(battery_level) > 0
ORDER BY session_time;

-- EV range efficiency (km per % of battery)
SELECT
    time_bucket('1 day', time) AS day,
    AVG(range_km / NULLIF(battery_level, 0)) as km_per_percent
FROM ev_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND battery_level > 10
  AND time > NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day;

-- ========================================
-- Location and Movement Analysis
-- ========================================

-- Recent trips (locations with movement)
SELECT
    time,
    latitude,
    longitude,
    speed,
    heading
FROM vehicle_location
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND speed > 0
  AND time > NOW() - INTERVAL '1 day'
ORDER BY time;

-- Daily distance traveled (approximate using GPS)
WITH location_pairs AS (
    SELECT
        time,
        latitude,
        longitude,
        LAG(latitude) OVER (ORDER BY time) as prev_lat,
        LAG(longitude) OVER (ORDER BY time) as prev_lon,
        LAG(time) OVER (ORDER BY time) as prev_time
    FROM vehicle_location
    WHERE vehicle_id = 'YOUR_VEHICLE_ID'
      AND time > NOW() - INTERVAL '7 days'
)
SELECT
    DATE(time) as day,
    -- Haversine distance approximation
    SUM(
        6371 * 2 * ASIN(SQRT(
            POWER(SIN(RADIANS(latitude - prev_lat) / 2), 2) +
            COS(RADIANS(prev_lat)) * COS(RADIANS(latitude)) *
            POWER(SIN(RADIANS(longitude - prev_lon) / 2), 2)
        ))
    ) as distance_km
FROM location_pairs
WHERE prev_lat IS NOT NULL
GROUP BY DATE(time)
ORDER BY day;

-- Most common parking locations (clustering)
SELECT
    ROUND(latitude::numeric, 3) as lat_cluster,
    ROUND(longitude::numeric, 3) as lon_cluster,
    COUNT(*) as visit_count,
    MIN(time) as first_visit,
    MAX(time) as last_visit
FROM vehicle_location
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND speed = 0
GROUP BY lat_cluster, lon_cluster
HAVING COUNT(*) > 5
ORDER BY visit_count DESC;

-- ========================================
-- Maintenance and Alerts
-- ========================================

-- Low tire pressure alerts
SELECT
    time, vin,
    tire_fl_psi, tire_fr_psi, tire_rl_psi, tire_rr_psi
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND (tire_fl_psi < 30 OR tire_fr_psi < 30 OR tire_rl_psi < 30 OR tire_rr_psi < 30)
  AND time > NOW() - INTERVAL '30 days'
ORDER BY time DESC;

-- Battery warning events
SELECT
    time, battery_voltage, battery_level
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND (battery_warning_light = true OR battery_voltage < 12.0)
  AND time > NOW() - INTERVAL '30 days'
ORDER BY time DESC;

-- Door left unlocked
SELECT
    time, doors_locked
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND doors_locked = false
  AND engine_running = false
  AND time > NOW() - INTERVAL '7 days'
ORDER BY time DESC;

-- ========================================
-- Statistics and Summaries
-- ========================================

-- Overall statistics
SELECT
    vehicle_id,
    COUNT(*) as total_readings,
    MIN(time) as first_reading,
    MAX(time) as last_reading,
    MAX(odometer) - MIN(odometer) as total_distance,
    AVG(fuel_level) as avg_fuel_level,
    AVG(battery_voltage) as avg_battery_voltage
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
GROUP BY vehicle_id;

-- Climate control usage
SELECT
    time_bucket('1 day', time) AS day,
    AVG(CASE WHEN climate_active THEN 1 ELSE 0 END) * 100 as climate_usage_pct,
    AVG(climate_target_temp) as avg_target_temp,
    AVG(climate_interior_temp) as avg_interior_temp,
    AVG(climate_exterior_temp) as avg_exterior_temp
FROM vehicle_status
WHERE vehicle_id = 'YOUR_VEHICLE_ID'
  AND time > NOW() - INTERVAL '30 days'
GROUP BY day
ORDER BY day;

-- ========================================
-- Data Retention and Cleanup
-- ========================================

-- Show database size
SELECT
    pg_size_pretty(pg_database_size('hyundai_data')) as database_size;

-- Show table sizes
SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Delete data older than 1 year (example - be careful!)
-- DELETE FROM vehicle_status WHERE time < NOW() - INTERVAL '1 year';
-- DELETE FROM ev_status WHERE time < NOW() - INTERVAL '1 year';
-- DELETE FROM vehicle_location WHERE time < NOW() - INTERVAL '1 year';

-- Create continuous aggregate for daily summaries (improves query performance)
CREATE MATERIALIZED VIEW IF NOT EXISTS vehicle_daily_summary
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    vehicle_id,
    AVG(odometer) as avg_odometer,
    AVG(fuel_level) as avg_fuel,
    AVG(battery_voltage) as avg_battery_voltage,
    COUNT(*) as reading_count
FROM vehicle_status
GROUP BY day, vehicle_id;

-- Refresh continuous aggregate
SELECT add_continuous_aggregate_policy('vehicle_daily_summary',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');
