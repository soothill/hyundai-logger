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
)

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

// errorTracker tracks consecutive errors for alerting
type errorTracker struct {
	mu                    sync.Mutex
	consecutiveErrors     int
	lastError             error
	lastErrorTime         time.Time
	lastSuccessTime       time.Time
	alertThreshold        int
	hasAlerted            bool
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
	errorTracker        *errorTracker
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
		errorTracker: &errorTracker{
			alertThreshold: alertThreshold,
		},
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

	for {
		// Calculate next poll interval dynamically
		interval := l.calculatePollInterval()
		l.logger.Info("Next poll scheduled in %v", interval)

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			l.logger.Info("Poll loop stopped due to context cancellation")
			return
		case <-l.stopCh:
			timer.Stop()
			l.logger.Info("Poll loop stopped due to stop signal")
			return
		case <-timer.C:
			l.pollAllVehicles(ctx, vehicles)
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

// pollAllVehicles collects data from all vehicles
func (l *Logger) pollAllVehicles(ctx context.Context, vehicles []api.Vehicle) {
	startTime := time.Now()
	l.logger.LogPollStart()

	hasErrors := false
	var lastErr error

	for _, vehicle := range vehicles {
		if err := l.pollVehicle(ctx, vehicle); err != nil {
			l.logger.Error("Error polling vehicle %s: %v", vehicle.VIN, err)
			hasErrors = true
			lastErr = err
		}
	}

	duration := time.Since(startTime)
	l.logger.LogPollComplete(duration)

	// Track success/failure for alerting
	if hasErrors {
		l.recordError(lastErr)
	} else {
		l.recordSuccess()
	}
}

// pollVehicle collects data from a single vehicle
func (l *Logger) pollVehicle(ctx context.Context, vehicle api.Vehicle) error {
	l.logger.Info("Polling vehicle: %s", vehicle.VIN)

	// Get vehicle status
	status, err := l.apiClient.GetVehicleStatus(ctx, vehicle.VehicleID)
	if err != nil {
		l.logger.LogError("Getting vehicle status", err)
		return fmt.Errorf("getting vehicle status: %w", err)
	}

	// Store vehicle status
	if err := l.db.InsertVehicleStatus(ctx, status); err != nil {
		l.logger.LogError("Storing vehicle status", err)
	} else {
		l.logger.LogDataCollection(vehicle.VIN, status.Odometer, status.FuelLevel)
	}

	// Store EV status if available
	if status.EV != nil {
		// Update charging state tracking
		l.lastCharging[vehicle.VIN] = status.EV.Charging

		if err := l.db.InsertEVStatus(ctx, status.Timestamp, vehicle.VehicleID, vehicle.VIN, status.EV); err != nil {
			l.logger.LogError("Storing EV status", err)
		} else {
			l.logger.LogEVData(vehicle.VIN, status.EV.BatteryLevel, status.EV.Charging)

			// Log charging state changes
			if status.EV.Charging {
				l.logger.Info("Vehicle %s is charging at %.1f kW (Battery: %.1f%%)",
					vehicle.VIN, status.EV.ChargingPower, status.EV.BatteryLevel)
			}
		}
	}

	// Get and store location
	location, err := l.apiClient.GetVehicleLocation(ctx, vehicle.VehicleID)
	if err != nil {
		l.logger.Error("Warning: failed to get location: %v", err)
	} else {
		if err := l.db.InsertLocation(ctx, location); err != nil {
			l.logger.LogError("Storing location", err)
		} else {
			l.logger.LogLocation(vehicle.VIN, location.Location.Latitude, location.Location.Longitude)
		}
	}

	return nil
}

// GetStats returns statistics about collected data
func (l *Logger) GetStats(ctx context.Context) (*Stats, error) {
	query := `
		SELECT
			COUNT(DISTINCT vehicle_id) as vehicle_count,
			COUNT(*) as status_count,
			MIN(time) as first_record,
			MAX(time) as last_record
		FROM vehicle_status
	`

	var stats Stats
	err := l.db.pool.QueryRow(ctx, query).Scan(
		&stats.VehicleCount,
		&stats.StatusCount,
		&stats.FirstRecord,
		&stats.LastRecord,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// Stats represents collection statistics
type Stats struct {
	VehicleCount int
	StatusCount  int64
	FirstRecord  time.Time
	LastRecord   time.Time
}

// recordError tracks an error and triggers alerts if threshold is reached
func (l *Logger) recordError(err error) {
	l.errorTracker.mu.Lock()
	defer l.errorTracker.mu.Unlock()

	l.errorTracker.consecutiveErrors++
	l.errorTracker.lastError = err
	l.errorTracker.lastErrorTime = time.Now()

	l.logger.Error("Poll error (%d consecutive failures): %v", l.errorTracker.consecutiveErrors, err)

	// Check if we should send an alert
	if l.errorTracker.consecutiveErrors >= l.errorTracker.alertThreshold && !l.errorTracker.hasAlerted {
		l.sendAlert()
		l.errorTracker.hasAlerted = true
	}
}

// recordSuccess resets error tracking on successful poll
func (l *Logger) recordSuccess() {
	l.errorTracker.mu.Lock()
	defer l.errorTracker.mu.Unlock()

	// If we had errors and now succeeded, log recovery
	if l.errorTracker.consecutiveErrors > 0 {
		l.logger.Info("Poll recovered after %d consecutive failures", l.errorTracker.consecutiveErrors)
	}

	l.errorTracker.consecutiveErrors = 0
	l.errorTracker.lastSuccessTime = time.Now()
	l.errorTracker.hasAlerted = false
}

// sendAlert sends an email alert about persistent errors
func (l *Logger) sendAlert() {
	if l.alerter == nil {
		return
	}

	l.errorTracker.mu.Lock()
	errorCount := l.errorTracker.consecutiveErrors
	lastError := l.errorTracker.lastError
	lastSuccess := l.errorTracker.lastSuccessTime
	l.errorTracker.mu.Unlock()

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
