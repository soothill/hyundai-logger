// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/soothill/hyundai-logger/internal/alerts"
	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/errortracker"
	"github.com/soothill/hyundai-logger/internal/metrics"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"golang.org/x/sync/errgroup"
)

// vehiclePollResult holds the result of polling a single vehicle
type vehiclePollResult struct {
	vehicle api.Vehicle
	points  []*write.Point
	err     error
}

// LoggerInterface defines the logging interface
type LoggerInterface interface {
	Info(format string, v ...interface{})
	Error(format string, v ...interface{})
	LogAuthentication(success bool, region string)
	LogVehicleDiscovery(count int)
	LogVehicleInfo(year int, make, model, vin string)
	LogPollStart()
	LogPollComplete(duration time.Duration)
	LogDataCollection(vin string, odometer, fuelLevel float64)
	LogEVData(vin string, batteryLevel float64, charging bool)
	LogLocation(vin string, lat, lon float64)
	LogError(operation string, err error)
}

// RateLimitConfig defines the interface for rate limit configuration
type RateLimitConfig interface {
	GetCurrentInterval(currentHour int) int
}

// ChargingConfig defines the interface for charging configuration
type ChargingConfig interface {
	IsEnabled() bool
	GetIntervalMinutes() int
}

// Logger handles periodic data collection and logging
type Logger struct {
	db                  *DB
	apiClient           *api.Client
	logger              LoggerInterface
	rateLimitCfg        RateLimitConfig
	chargingCfg         ChargingConfig
	lastCharging        map[string]bool  // Track charging state per VIN
	stopCh              chan struct{}
	stoppedCh           chan struct{}
	alerter             *alerts.Alerter
	errorTracker        *errortracker.Tracker
	metrics             *metrics.Collector
}

// NewLogger creates a new data logger
func NewLogger(db *DB, apiClient *api.Client, rateLimitCfg RateLimitConfig, chargingCfg ChargingConfig, alerter *alerts.Alerter, alertThreshold int, logger LoggerInterface) *Logger {
	return &Logger{
		db:           db,
		apiClient:    apiClient,
		logger:       logger,
		rateLimitCfg: rateLimitCfg,
		chargingCfg:  chargingCfg,
		lastCharging: make(map[string]bool),
		stopCh:       make(chan struct{}),
		stoppedCh:    make(chan struct{}),
		alerter:      alerter,
		errorTracker: errortracker.New(alertThreshold),
		metrics:      metrics.New(),
	}
}

// Start begins the data collection loop
func (l *Logger) Start(ctx context.Context) error {
	l.logger.Info("Starting data logger...")

	// Authenticate first
	if err := l.apiClient.Authenticate(ctx); err != nil {
		l.logger.LogAuthentication(false, "unknown")
		return fmt.Errorf("initial authentication: %w", err)
	}
	l.logger.LogAuthentication(true, "configured region")

	// Get vehicles
	vehicles, err := l.apiClient.GetVehicles(ctx)
	if err != nil {
		l.logger.LogError("Fetching vehicles", err)
		return fmt.Errorf("fetching vehicles: %w", err)
	}

	if len(vehicles) == 0 {
		l.logger.Error("No vehicles found in account")
		return fmt.Errorf("no vehicles found in account")
	}

	l.logger.LogVehicleDiscovery(len(vehicles))
	l.metrics.SetTotalVehicles(len(vehicles))

	// Store vehicle information
	for _, vehicle := range vehicles {
		if err := l.db.UpsertVehicle(ctx, vehicle); err != nil {
			l.logger.Error("Failed to store vehicle %s: %v", vehicle.VIN, err)
		} else {
			l.logger.LogVehicleInfo(vehicle.Year, vehicle.Make, vehicle.Model, vehicle.VIN)
		}
	}

	// Start polling loop
	go l.pollLoop(ctx, vehicles)

	return nil
}

// Stop gracefully stops the data logger
func (l *Logger) Stop() {
	l.logger.Info("Stopping data logger...")
	close(l.stopCh)
	<-l.stoppedCh
	l.logger.Info("Data logger stopped")
}

// pollLoop periodically collects data from all vehicles with dynamic intervals
func (l *Logger) pollLoop(ctx context.Context, vehicles []api.Vehicle) {
	defer close(l.stoppedCh)

	// Do an immediate poll
	l.pollAllVehicles(ctx, vehicles)

	pollCount := 0
	for {
		// Calculate next poll interval dynamically
		interval := l.calculatePollInterval()
		l.logger.Info("Next poll scheduled in %v", interval)

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			l.logger.Info("Poll loop stopped due to context cancellation")
			l.LogMetrics() // Log final metrics
			return
		case <-l.stopCh:
			timer.Stop()
			l.logger.Info("Poll loop stopped due to stop signal")
			l.LogMetrics() // Log final metrics
			return
		case <-timer.C:
			l.pollAllVehicles(ctx, vehicles)
			pollCount++

			// Log metrics every 10 polls
			if pollCount%10 == 0 {
				l.LogMetrics()
			}
		}
	}
}

// calculatePollInterval determines the appropriate poll interval based on time and charging state
func (l *Logger) calculatePollInterval() time.Duration {
	currentHour := time.Now().Hour()

	// Check if any vehicle is charging
	isAnyCharging := false
	for _, charging := range l.lastCharging {
		if charging {
			isAnyCharging = true
			break
		}
	}

	// If charging detection is enabled and a vehicle is charging, use charging interval
	if l.chargingCfg.IsEnabled() && isAnyCharging {
		intervalMinutes := l.chargingCfg.GetIntervalMinutes()
		l.logger.Info("Vehicle charging detected - using fast polling interval: %d minutes", intervalMinutes)
		return time.Duration(intervalMinutes) * time.Minute
	}

	// Otherwise, use scheduled interval based on current time
	intervalMinutes := l.rateLimitCfg.GetCurrentInterval(currentHour)
	return time.Duration(intervalMinutes) * time.Minute
}

// pollAllVehicles collects data from all vehicles in parallel with batch writing
func (l *Logger) pollAllVehicles(ctx context.Context, vehicles []api.Vehicle) {
	startTime := time.Now()
	l.logger.LogPollStart()
	l.metrics.RecordPollStart()

	// Use errgroup for parallel execution with proper error handling
	g, gCtx := errgroup.WithContext(ctx)

	// Channel to collect results from parallel operations
	resultChan := make(chan vehiclePollResult, len(vehicles))

	// Launch goroutine for each vehicle
	for _, vehicle := range vehicles {
		vehicle := vehicle // Capture loop variable for goroutine
		g.Go(func() error {
			points, err := l.pollVehicleCollectPoints(gCtx, vehicle)
			resultChan <- vehiclePollResult{
				vehicle: vehicle,
				points:  points,
				err:     err,
			}
			return nil // Don't stop other goroutines on error
		})
	}

	// Wait for all goroutines to complete
	_ = g.Wait()
	close(resultChan)

	// Collect all points and errors
	var allPoints []*write.Point
	var mu sync.Mutex
	hasErrors := false
	var lastErr error
	errorCount := 0

	for result := range resultChan {
		if result.err != nil {
			l.logger.Error("Error polling vehicle %s: %v", result.vehicle.VIN, result.err)
			l.metrics.RecordVehiclePoll(result.vehicle.VIN, false)
			hasErrors = true
			lastErr = result.err
			errorCount++
		} else {
			l.metrics.RecordVehiclePoll(result.vehicle.VIN, true)
			// Collect points for batch write
			mu.Lock()
			allPoints = append(allPoints, result.points...)
			mu.Unlock()
		}
	}

	// Batch write all collected points in a single operation
	if len(allPoints) > 0 {
		l.logger.Info("Writing %d data points in batch", len(allPoints))
		if err := l.db.WriteBatch(ctx, allPoints); err != nil {
			l.logger.Error("Failed to write batch to database: %v", err)
			hasErrors = true
			lastErr = err
		} else {
			l.logger.Info("Successfully wrote %d points to database", len(allPoints))
		}
	}

	duration := time.Since(startTime)
	l.logger.LogPollComplete(duration)
	l.logger.Info("Polled %d vehicles in parallel in %s", len(vehicles), duration.Round(time.Millisecond))

	if errorCount > 0 {
		l.logger.Error("Completed poll with %d/%d vehicles reporting errors", errorCount, len(vehicles))
	}

	// Record poll metrics
	l.metrics.RecordPollComplete(duration, !hasErrors)

	// Track success/failure for alerting
	if hasErrors {
		l.recordError(lastErr)
		l.metrics.RecordError(lastErr.Error())
	} else {
		l.recordSuccess()
	}

	// Update charging vehicle count
	chargingCount := 0
	for _, charging := range l.lastCharging {
		if charging {
			chargingCount++
		}
	}
	l.metrics.UpdateVehiclesCharging(chargingCount)
}

// pollVehicleCollectPoints collects data from a single vehicle and returns points for batch writing
func (l *Logger) pollVehicleCollectPoints(ctx context.Context, vehicle api.Vehicle) ([]*write.Point, error) {
	l.logger.Info("Polling vehicle: %s", vehicle.VIN)

	var allPoints []*write.Point

	// Get vehicle status
	status, err := l.apiClient.GetVehicleStatus(ctx, vehicle.VehicleID)
	if err != nil {
		l.logger.LogError("Getting vehicle status", err)
		return nil, fmt.Errorf("getting vehicle status: %w", err)
	}

	// Collect vehicle status points (6 points)
	statusPoints := l.db.CollectVehicleStatusPoints(status)
	allPoints = append(allPoints, statusPoints...)
	l.logger.LogDataCollection(vehicle.VIN, status.Odometer, status.FuelLevel)

	// Collect EV status point if available
	if status.EV != nil {
		// Update charging state tracking
		wasCharging := l.lastCharging[vehicle.VIN]
		l.lastCharging[vehicle.VIN] = status.EV.Charging

		// Record charging detection
		if status.EV.Charging && !wasCharging {
			l.metrics.RecordChargingDetection()
		}

		evPoint := l.db.CollectEVStatusPoint(status.Timestamp, vehicle.VehicleID, vehicle.VIN, status.EV)
		if evPoint != nil {
			allPoints = append(allPoints, evPoint)
			l.logger.LogEVData(vehicle.VIN, status.EV.BatteryLevel, status.EV.Charging)

			// Log charging state changes
			if status.EV.Charging {
				l.logger.Info("Vehicle %s is charging at %.1f kW (Battery: %.1f%%)",
					vehicle.VIN, status.EV.ChargingPower, status.EV.BatteryLevel)
			}
		}
	}

	// Get and collect location point
	location, err := l.apiClient.GetVehicleLocation(ctx, vehicle.VehicleID)
	if err != nil {
		l.logger.Error("Warning: failed to get location: %v", err)
	} else {
		locationPoint := l.db.CollectLocationPoint(location)
		allPoints = append(allPoints, locationPoint)
		l.logger.LogLocation(vehicle.VIN, location.Location.Latitude, location.Location.Longitude)
	}

	return allPoints, nil
}

// GetMetrics returns current application metrics
func (l *Logger) GetMetrics() metrics.Stats {
	return l.metrics.GetStats()
}

// GetMetricsCollector returns the metrics collector for external use (e.g., Prometheus)
func (l *Logger) GetMetricsCollector() *metrics.Collector {
	return l.metrics
}

// LogMetrics logs the current metrics to the logger
func (l *Logger) LogMetrics() {
	stats := l.metrics.GetStats()
	l.logger.Info("\n%s", metrics.FormatStats(stats))
}

// recordError tracks an error and triggers alerts if threshold is reached
func (l *Logger) recordError(err error) {
	shouldAlert := l.errorTracker.RecordError(err)

	consecutiveErrors := l.errorTracker.GetConsecutiveErrors()
	l.logger.Error("Poll error (%d consecutive failures): %v", consecutiveErrors, err)

	// Send alert if threshold reached
	if shouldAlert {
		l.sendAlert()
	}
}

// recordSuccess resets error tracking on successful poll
func (l *Logger) recordSuccess() {
	wasRecovery := l.errorTracker.RecordSuccess()

	// If we had errors and now succeeded, log recovery
	if wasRecovery {
		l.logger.Info("Poll recovered from previous failures")
	}
}

// sendAlert sends an email alert about persistent errors
func (l *Logger) sendAlert() {
	if l.alerter == nil {
		return
	}

	errorCount, lastError, lastSuccess, _ := l.errorTracker.GetStats()

	subject := fmt.Sprintf("🚨 Hyundai Logger Alert: %d Consecutive Failures", errorCount)
	body := alerts.FormatErrorAlert(
		"Polling Failure",
		errorCount,
		lastError.Error(),
		lastSuccess,
	)

	if err := l.alerter.SendAlert(subject, body); err != nil {
		l.logger.Error("Failed to send alert email: %v", err)
	} else {
		l.logger.Info("Alert email sent successfully")
	}
}
