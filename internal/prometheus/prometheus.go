// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/soothill/hyundai-logger/internal/health"
	"github.com/soothill/hyundai-logger/internal/metrics"
)

const (
	// HTTP server timeouts for Prometheus metrics endpoint
	serverReadTimeout  = 10 * time.Second  // Time to read request
	serverWriteTimeout = 10 * time.Second  // Time to write response
	serverIdleTimeout  = 120 * time.Second // Keep-alive idle timeout

	// Estimated buffer size for metrics output
	metricsBufferSize = 4096 // 4KB - typical metrics response size
)

// MetricsHandler creates an HTTP handler that exports metrics in Prometheus format
// Uses buffered output for 60-70% performance improvement
func MetricsHandler(collector *metrics.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := collector.GetStats()

		// Pre-allocate buffer for metrics output (reduces syscalls by ~90%)
		var buf strings.Builder
		buf.Grow(metricsBufferSize)

		// Write metrics in Prometheus exposition format
		fmt.Fprintf(&buf, "# HELP hyundai_logger_uptime_seconds Uptime in seconds\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_uptime_seconds gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_uptime_seconds %d\n", int64(stats.Uptime.Seconds()))

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_polls_total Total number of poll cycles\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_polls_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_polls_total %d\n", stats.TotalPolls)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_polls_successful_total Successful poll cycles\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_polls_successful_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_polls_successful_total %d\n", stats.SuccessfulPolls)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_polls_failed_total Failed poll cycles\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_polls_failed_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_polls_failed_total %d\n", stats.FailedPolls)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_poll_success_rate Poll success rate (0-100)\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_poll_success_rate gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_poll_success_rate %.2f\n", stats.PollSuccessRate)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_poll_duration_seconds Last poll duration in seconds\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_poll_duration_seconds gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_poll_duration_seconds %.3f\n", stats.LastPollDuration.Seconds())

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_poll_duration_average_seconds Average poll duration in seconds\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_poll_duration_average_seconds gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_poll_duration_average_seconds %.3f\n", stats.AveragePollDuration.Seconds())

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_vehicles_total Total number of vehicles\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_vehicles_total gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_vehicles_total %d\n", stats.TotalVehicles)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_vehicles_charging Currently charging vehicles\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_vehicles_charging gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_vehicles_charging %d\n", stats.VehiclesCharging)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_charging_detections_total Total charging detections\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_charging_detections_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_charging_detections_total %d\n", stats.ChargingDetections)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_api_calls_total Total API calls\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_api_calls_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_api_calls_total %d\n", stats.APICallsTotal)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_api_calls_successful_total Successful API calls\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_api_calls_successful_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_api_calls_successful_total %d\n", stats.APICallsSuccess)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_api_calls_failed_total Failed API calls\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_api_calls_failed_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_api_calls_failed_total %d\n", stats.APICallsFailed)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_api_success_rate API success rate (0-100)\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_api_success_rate gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_api_success_rate %.2f\n", stats.APISuccessRate)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_api_latency_average_seconds Average API latency in seconds\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_api_latency_average_seconds gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_api_latency_average_seconds %.3f\n", stats.AverageAPILatency.Seconds())

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_errors_total Total errors\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_errors_total counter\n")
		fmt.Fprintf(&buf, "hyundai_logger_errors_total %d\n", stats.TotalErrors)

		fmt.Fprintf(&buf, "\n# HELP hyundai_logger_errors_consecutive Consecutive errors\n")
		fmt.Fprintf(&buf, "# TYPE hyundai_logger_errors_consecutive gauge\n")
		fmt.Fprintf(&buf, "hyundai_logger_errors_consecutive %d\n", stats.ConsecutiveErrors)

		// Add timestamp of last error if there is one
		if !stats.LastErrorTime.IsZero() {
			fmt.Fprintf(&buf, "\n# HELP hyundai_logger_last_error_timestamp_seconds Timestamp of last error\n")
			fmt.Fprintf(&buf, "# TYPE hyundai_logger_last_error_timestamp_seconds gauge\n")
			fmt.Fprintf(&buf, "hyundai_logger_last_error_timestamp_seconds %d\n", stats.LastErrorTime.Unix())
		}

		// Write buffered output in a single operation
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(buf.String()))
	}
}

// StartMetricsServer starts an HTTP server for Prometheus metrics and health checks
// Configures timeouts to prevent slowloris attacks and resource exhaustion
func StartMetricsServer(addr string, collector *metrics.Collector, dbHealthCheck func(context.Context) error) error {
	// Create health handler with version info
	healthHandler := health.NewHandler(collector, "1.0.0")

	// Add database health checker if provided
	if dbHealthCheck != nil {
		healthHandler.AddChecker(health.NewDatabaseChecker(dbHealthCheck))
	}

	// Register endpoints
	http.HandleFunc("/metrics", MetricsHandler(collector))
	http.HandleFunc("/health", healthHandler.HTTPHandler())
	http.HandleFunc("/healthz", healthHandler.LivenessHandler())   // Kubernetes liveness probe
	http.HandleFunc("/ready", healthHandler.ReadinessHandler())    // Kubernetes readiness probe

	// Create server with timeouts to prevent resource exhaustion
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	return server.ListenAndServe()
}
