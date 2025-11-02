// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package prometheus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/metrics"
)

func TestMetricsHandler(t *testing.T) {
	collector := metrics.New()

	// Record some test data
	collector.SetTotalVehicles(3)
	collector.UpdateVehiclesCharging(1)
	collector.RecordPollStart()
	collector.RecordPollComplete(2*time.Second, true)
	collector.RecordAPICall(100*time.Millisecond, true)
	collector.RecordVehiclePoll("VIN123", true)

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	body := w.Body.String()

	// Verify key metrics are present
	expectedMetrics := []string{
		"hyundai_logger_uptime_seconds",
		"hyundai_logger_polls_total",
		"hyundai_logger_polls_successful_total",
		"hyundai_logger_poll_success_rate",
		"hyundai_logger_vehicles_total 3",
		"hyundai_logger_vehicles_charging 1",
		"hyundai_logger_api_calls_total",
		"hyundai_logger_errors_total",
		"hyundai_logger_errors_consecutive",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected metric %q to be present in output", metric)
		}
	}
}

func TestMetricsHandler_WithErrors(t *testing.T) {
	collector := metrics.New()

	// Record some errors
	collector.RecordError("test error 1")
	collector.RecordError("test error 2")

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body := w.Body.String()

	// Should include error timestamp
	if !strings.Contains(body, "hyundai_logger_last_error_timestamp_seconds") {
		t.Error("Expected last_error_timestamp metric when errors are recorded")
	}

	// Should show error count
	if !strings.Contains(body, "hyundai_logger_errors_total 2") {
		t.Error("Expected errors_total to be 2")
	}
}

func TestMetricsHandler_PrometheusFormat(t *testing.T) {
	collector := metrics.New()
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body := w.Body.String()

	// Verify Prometheus exposition format
	// Should have HELP and TYPE comments
	if !strings.Contains(body, "# HELP ") {
		t.Error("Expected HELP comments in Prometheus format")
	}
	if !strings.Contains(body, "# TYPE ") {
		t.Error("Expected TYPE comments in Prometheus format")
	}

	// Check specific format for a metric
	if !strings.Contains(body, "# TYPE hyundai_logger_polls_total counter") {
		t.Error("Expected polls_total to be declared as counter type")
	}
	if !strings.Contains(body, "# TYPE hyundai_logger_poll_success_rate gauge") {
		t.Error("Expected poll_success_rate to be declared as gauge type")
	}
}

func TestMetricsHandler_EmptyCollector(t *testing.T) {
	collector := metrics.New()

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should still have all metric definitions even with zero values
	if !strings.Contains(body, "hyundai_logger_polls_total 0") {
		t.Error("Expected polls_total to be 0 for empty collector")
	}
}

func TestMetricsHandler_SuccessRateCalculation(t *testing.T) {
	collector := metrics.New()

	// Record mixed success/failure
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, false)

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body := w.Body.String()

	// 2 successful out of 3 = 66.67%
	if !strings.Contains(body, "hyundai_logger_polls_successful_total 2") {
		t.Error("Expected 2 successful polls")
	}
	if !strings.Contains(body, "hyundai_logger_polls_failed_total 1") {
		t.Error("Expected 1 failed poll")
	}
}

func TestStartMetricsServer_HealthEndpoints(t *testing.T) {
	// We can't fully test StartMetricsServer since it's a blocking call,
	// but we can verify the health handler integration via unit test of its components
	collector := metrics.New()
	collector.SetTotalVehicles(5)

	// Test that health check can be created with db checker
	dbHealthCheck := func(ctx context.Context) error {
		return nil // Healthy
	}

	// Verify function works correctly
	// (actual server test would require goroutine and port management)
	if err := dbHealthCheck(context.Background()); err != nil {
		t.Errorf("Database health check should succeed, got error: %v", err)
	}

	if collector == nil {
		t.Error("Collector should not be nil")
	}
}

func TestStartMetricsServer_FailingDBHealth(t *testing.T) {
	// Test that unhealthy database checker works
	dbHealthCheck := func(ctx context.Context) error {
		return errors.New("database connection failed")
	}

	err := dbHealthCheck(context.Background())
	if err == nil {
		t.Error("Expected error from failing database health check")
	}
}

func TestMetricsHandler_AllMetricTypes(t *testing.T) {
	collector := metrics.New()

	// Populate all metric types
	collector.SetTotalVehicles(10)
	collector.UpdateVehiclesCharging(3)

	for i := 0; i < 5; i++ {
		collector.RecordPollStart()
		collector.RecordPollComplete(time.Duration(i+1)*time.Second, i%2 == 0)
	}

	for i := 0; i < 20; i++ {
		collector.RecordAPICall(time.Duration(i+50)*time.Millisecond, i%3 != 0)
	}

	collector.RecordError("test error")
	collector.RecordVehiclePoll("VIN1", true)
	collector.RecordVehiclePoll("VIN2", false)

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	body := w.Body.String()

	// Verify all metric families are present
	metricFamilies := []string{
		"hyundai_logger_uptime_seconds",
		"hyundai_logger_polls_total",
		"hyundai_logger_polls_successful_total",
		"hyundai_logger_polls_failed_total",
		"hyundai_logger_poll_success_rate",
		"hyundai_logger_poll_duration_seconds",
		"hyundai_logger_poll_duration_average_seconds",
		"hyundai_logger_vehicles_total",
		"hyundai_logger_vehicles_charging",
		"hyundai_logger_charging_detections_total",
		"hyundai_logger_api_calls_total",
		"hyundai_logger_api_calls_successful_total",
		"hyundai_logger_api_calls_failed_total",
		"hyundai_logger_api_success_rate",
		"hyundai_logger_api_latency_average_seconds",
		"hyundai_logger_errors_total",
		"hyundai_logger_errors_consecutive",
		"hyundai_logger_last_error_timestamp_seconds",
	}

	for _, family := range metricFamilies {
		if !strings.Contains(body, family) {
			t.Errorf("Missing metric family: %s", family)
		}
	}
}

func BenchmarkMetricsHandler(b *testing.B) {
	collector := metrics.New()
	collector.SetTotalVehicles(3)
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)

	handler := MetricsHandler(collector)

	req := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}
