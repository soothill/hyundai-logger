// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/soothill/hyundai-logger/internal/metrics"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status      Status    `json:"status"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
	ResponseMs  int64     `json:"response_ms,omitempty"`
}

// HealthResponse is the JSON response for the health endpoint
type HealthResponse struct {
	Status     Status                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Uptime     string                     `json:"uptime"`
	Version    string                     `json:"version,omitempty"`
	Components map[string]ComponentHealth `json:"components"`
	Metrics    *HealthMetrics             `json:"metrics,omitempty"`
}

// HealthMetrics contains key metrics for the health check
type HealthMetrics struct {
	TotalPolls        int64   `json:"total_polls"`
	SuccessfulPolls   int64   `json:"successful_polls"`
	FailedPolls       int64   `json:"failed_polls"`
	PollSuccessRate   float64 `json:"poll_success_rate"`
	LastPollDuration  string  `json:"last_poll_duration"`
	ConsecutiveErrors int64   `json:"consecutive_errors"`
	TotalVehicles     int     `json:"total_vehicles"`
	VehiclesCharging  int     `json:"vehicles_charging"`
}

// Checker defines the interface for health checks
type Checker interface {
	Check(ctx context.Context) ComponentHealth
	Name() string
}

// DatabaseChecker checks InfluxDB connectivity
type DatabaseChecker struct {
	healthCheck func(context.Context) error
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(healthCheck func(context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		healthCheck: healthCheck,
	}
}

// Name returns the checker name
func (d *DatabaseChecker) Name() string {
	return "influxdb"
}

// Check performs the health check
func (d *DatabaseChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()
	err := d.healthCheck(ctx)
	duration := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:      StatusUnhealthy,
			Message:     err.Error(),
			LastChecked: time.Now(),
			ResponseMs:  duration.Milliseconds(),
		}
	}

	return ComponentHealth{
		Status:      StatusHealthy,
		Message:     "connected",
		LastChecked: time.Now(),
		ResponseMs:  duration.Milliseconds(),
	}
}

// Handler provides HTTP health check endpoints
type Handler struct {
	checkers         []Checker
	metricsCollector *metrics.Collector
	startTime        time.Time
	version          string
}

// NewHandler creates a new health check handler
func NewHandler(collector *metrics.Collector, version string) *Handler {
	return &Handler{
		checkers:         make([]Checker, 0),
		metricsCollector: collector,
		startTime:        time.Now(),
		version:          version,
	}
}

// AddChecker adds a health checker
func (h *Handler) AddChecker(checker Checker) {
	h.checkers = append(h.checkers, checker)
}

// Check performs all health checks and returns the overall status
func (h *Handler) Check(ctx context.Context) HealthResponse {
	components := make(map[string]ComponentHealth)
	overallStatus := StatusHealthy

	// Run all component checks
	for _, checker := range h.checkers {
		health := checker.Check(ctx)
		components[checker.Name()] = health

		// Determine overall status (worst status wins)
		if health.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if health.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	// Get metrics if available
	var healthMetrics *HealthMetrics
	if h.metricsCollector != nil {
		stats := h.metricsCollector.GetStats()
		healthMetrics = &HealthMetrics{
			TotalPolls:        stats.TotalPolls,
			SuccessfulPolls:   stats.SuccessfulPolls,
			FailedPolls:       stats.FailedPolls,
			PollSuccessRate:   stats.PollSuccessRate,
			LastPollDuration:  stats.LastPollDuration.Round(time.Millisecond).String(),
			ConsecutiveErrors: stats.ConsecutiveErrors,
			TotalVehicles:     stats.TotalVehicles,
			VehiclesCharging:  stats.VehiclesCharging,
		}

		// Check for consecutive errors
		if stats.ConsecutiveErrors >= 3 {
			overallStatus = StatusDegraded
		}
		if stats.ConsecutiveErrors >= 5 {
			overallStatus = StatusUnhealthy
		}
	}

	return HealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(h.startTime).Round(time.Second).String(),
		Version:    h.version,
		Components: components,
		Metrics:    healthMetrics,
	}
}

// HTTPHandler returns an HTTP handler for the health endpoint
func (h *Handler) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := h.Check(ctx)

		// Set appropriate HTTP status code
		var statusCode int
		switch response.Status {
		case StatusDegraded:
			statusCode = http.StatusOK // Still return 200 for degraded
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		default:
			statusCode = http.StatusOK
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// LivenessHandler returns a simple liveness check (always returns 200 if server is running)
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// ReadinessHandler returns a readiness check (returns 200 only if system is ready)
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		response := h.Check(ctx)

		if response.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("READY"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("NOT READY"))
		}
	}
}
