// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package prometheus

import (
	"fmt"
	"net/http"

	"github.com/soothill/hyundai-logger/internal/metrics"
)

// MetricsHandler creates an HTTP handler that exports metrics in Prometheus format
func MetricsHandler(collector *metrics.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := collector.GetStats()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Write metrics in Prometheus exposition format
		fmt.Fprintf(w, "# HELP hyundai_logger_uptime_seconds Uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_uptime_seconds gauge\n")
		fmt.Fprintf(w, "hyundai_logger_uptime_seconds %d\n", int64(stats.Uptime.Seconds()))

		fmt.Fprintf(w, "\n# HELP hyundai_logger_polls_total Total number of poll cycles\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_polls_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_polls_total %d\n", stats.TotalPolls)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_polls_successful_total Successful poll cycles\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_polls_successful_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_polls_successful_total %d\n", stats.SuccessfulPolls)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_polls_failed_total Failed poll cycles\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_polls_failed_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_polls_failed_total %d\n", stats.FailedPolls)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_poll_success_rate Poll success rate (0-100)\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_poll_success_rate gauge\n")
		fmt.Fprintf(w, "hyundai_logger_poll_success_rate %.2f\n", stats.PollSuccessRate)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_poll_duration_seconds Last poll duration in seconds\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_poll_duration_seconds gauge\n")
		fmt.Fprintf(w, "hyundai_logger_poll_duration_seconds %.3f\n", stats.LastPollDuration.Seconds())

		fmt.Fprintf(w, "\n# HELP hyundai_logger_poll_duration_average_seconds Average poll duration in seconds\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_poll_duration_average_seconds gauge\n")
		fmt.Fprintf(w, "hyundai_logger_poll_duration_average_seconds %.3f\n", stats.AveragePollDuration.Seconds())

		fmt.Fprintf(w, "\n# HELP hyundai_logger_vehicles_total Total number of vehicles\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_vehicles_total gauge\n")
		fmt.Fprintf(w, "hyundai_logger_vehicles_total %d\n", stats.TotalVehicles)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_vehicles_charging Currently charging vehicles\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_vehicles_charging gauge\n")
		fmt.Fprintf(w, "hyundai_logger_vehicles_charging %d\n", stats.VehiclesCharging)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_charging_detections_total Total charging detections\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_charging_detections_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_charging_detections_total %d\n", stats.ChargingDetections)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_api_calls_total Total API calls\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_api_calls_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_api_calls_total %d\n", stats.APICallsTotal)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_api_calls_successful_total Successful API calls\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_api_calls_successful_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_api_calls_successful_total %d\n", stats.APICallsSuccess)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_api_calls_failed_total Failed API calls\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_api_calls_failed_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_api_calls_failed_total %d\n", stats.APICallsFailed)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_api_success_rate API success rate (0-100)\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_api_success_rate gauge\n")
		fmt.Fprintf(w, "hyundai_logger_api_success_rate %.2f\n", stats.APISuccessRate)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_api_latency_average_seconds Average API latency in seconds\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_api_latency_average_seconds gauge\n")
		fmt.Fprintf(w, "hyundai_logger_api_latency_average_seconds %.3f\n", stats.AverageAPILatency.Seconds())

		fmt.Fprintf(w, "\n# HELP hyundai_logger_errors_total Total errors\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_errors_total counter\n")
		fmt.Fprintf(w, "hyundai_logger_errors_total %d\n", stats.TotalErrors)

		fmt.Fprintf(w, "\n# HELP hyundai_logger_errors_consecutive Consecutive errors\n")
		fmt.Fprintf(w, "# TYPE hyundai_logger_errors_consecutive gauge\n")
		fmt.Fprintf(w, "hyundai_logger_errors_consecutive %d\n", stats.ConsecutiveErrors)

		// Add timestamp of last error if there is one
		if !stats.LastErrorTime.IsZero() {
			fmt.Fprintf(w, "\n# HELP hyundai_logger_last_error_timestamp_seconds Timestamp of last error\n")
			fmt.Fprintf(w, "# TYPE hyundai_logger_last_error_timestamp_seconds gauge\n")
			fmt.Fprintf(w, "hyundai_logger_last_error_timestamp_seconds %d\n", stats.LastErrorTime.Unix())
		}
	}
}

// StartMetricsServer starts an HTTP server for Prometheus metrics
func StartMetricsServer(addr string, collector *metrics.Collector) error {
	http.HandleFunc("/metrics", MetricsHandler(collector))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})

	return http.ListenAndServe(addr, nil)
}
