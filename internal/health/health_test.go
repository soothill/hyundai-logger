// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/metrics"
)

func TestNewHandler(t *testing.T) {
	collector := metrics.New()
	handler := NewHandler(collector, "1.0.0")

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", handler.version)
	}
	if handler.metricsCollector != collector {
		t.Error("Expected metrics collector to be set")
	}
}

func TestAddChecker(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	mockChecker := &mockChecker{
		name:   "test",
		status: StatusHealthy,
	}

	handler.AddChecker(mockChecker)

	if len(handler.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(handler.checkers))
	}
}

func TestCheckHealthy(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	mockChecker := &mockChecker{
		name:   "database",
		status: StatusHealthy,
	}
	handler.AddChecker(mockChecker)

	response := handler.Check(context.Background())

	if response.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", response.Status)
	}
	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", response.Version)
	}
	if len(response.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(response.Components))
	}
	if response.Components["database"].Status != StatusHealthy {
		t.Errorf("Expected database status healthy, got %s", response.Components["database"].Status)
	}
}

func TestCheckDegraded(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	handler.AddChecker(&mockChecker{
		name:   "database",
		status: StatusHealthy,
	})
	handler.AddChecker(&mockChecker{
		name:   "api",
		status: StatusDegraded,
	})

	response := handler.Check(context.Background())

	if response.Status != StatusDegraded {
		t.Errorf("Expected overall status degraded, got %s", response.Status)
	}
}

func TestCheckUnhealthy(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	handler.AddChecker(&mockChecker{
		name:   "database",
		status: StatusHealthy,
	})
	handler.AddChecker(&mockChecker{
		name:   "api",
		status: StatusUnhealthy,
	})

	response := handler.Check(context.Background())

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected overall status unhealthy, got %s", response.Status)
	}
}

func TestCheckWithMetrics(t *testing.T) {
	collector := metrics.New()
	collector.SetTotalVehicles(5)
	collector.UpdateVehiclesCharging(2)
	collector.RecordPollStart()
	collector.RecordPollComplete(5*time.Second, true)

	handler := NewHandler(collector, "1.0.0")

	response := handler.Check(context.Background())

	if response.Metrics == nil {
		t.Fatal("Expected metrics to be present")
	}
	if response.Metrics.TotalVehicles != 5 {
		t.Errorf("Expected 5 total vehicles, got %d", response.Metrics.TotalVehicles)
	}
	if response.Metrics.VehiclesCharging != 2 {
		t.Errorf("Expected 2 vehicles charging, got %d", response.Metrics.VehiclesCharging)
	}
	if response.Metrics.TotalPolls != 1 {
		t.Errorf("Expected 1 total poll, got %d", response.Metrics.TotalPolls)
	}
}

func TestCheckConsecutiveErrors(t *testing.T) {
	collector := metrics.New()

	// Record 3 consecutive failed polls (which increments consecutive errors)
	for i := 0; i < 3; i++ {
		collector.RecordPollStart()
		collector.RecordPollComplete(1*time.Second, false)
	}

	handler := NewHandler(collector, "1.0.0")

	response := handler.Check(context.Background())

	// 3 consecutive errors should trigger degraded status
	if response.Status != StatusDegraded {
		t.Errorf("Expected degraded status with 3 errors, got %s", response.Status)
	}
	if response.Metrics.ConsecutiveErrors != 3 {
		t.Errorf("Expected 3 consecutive errors, got %d", response.Metrics.ConsecutiveErrors)
	}
}

func TestCheckManyConsecutiveErrors(t *testing.T) {
	collector := metrics.New()

	// Record 5 consecutive failed polls (which increments consecutive errors)
	for i := 0; i < 5; i++ {
		collector.RecordPollStart()
		collector.RecordPollComplete(1*time.Second, false)
	}

	handler := NewHandler(collector, "1.0.0")

	response := handler.Check(context.Background())

	// 5 consecutive errors should trigger unhealthy status
	if response.Status != StatusUnhealthy {
		t.Errorf("Expected unhealthy status with 5 errors, got %s", response.Status)
	}
}

func TestHTTPHandler(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")
	handler.AddChecker(&mockChecker{
		name:   "test",
		status: StatusHealthy,
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HTTPHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", response.Status)
	}
}

func TestHTTPHandlerUnhealthy(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")
	handler.AddChecker(&mockChecker{
		name:   "test",
		status: StatusUnhealthy,
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HTTPHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %s", response.Status)
	}
}

func TestHTTPHandlerDegraded(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")
	handler.AddChecker(&mockChecker{
		name:   "test",
		status: StatusDegraded,
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HTTPHandler()(w, req)

	// Degraded still returns 200
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200 for degraded, got %d", w.Code)
	}
}

func TestLivenessHandler(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	handler.LivenessHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Expected 'OK', got %s", w.Body.String())
	}
}

func TestReadinessHandlerReady(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")
	handler.AddChecker(&mockChecker{
		name:   "test",
		status: StatusHealthy,
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler.ReadinessHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
	if w.Body.String() != "READY" {
		t.Errorf("Expected 'READY', got %s", w.Body.String())
	}
}

func TestReadinessHandlerNotReady(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")
	handler.AddChecker(&mockChecker{
		name:   "test",
		status: StatusDegraded,
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	handler.ReadinessHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code 503, got %d", w.Code)
	}
	if w.Body.String() != "NOT READY" {
		t.Errorf("Expected 'NOT READY', got %s", w.Body.String())
	}
}

func TestDatabaseChecker(t *testing.T) {
	// Healthy database
	checker := NewDatabaseChecker(func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "influxdb" {
		t.Errorf("Expected name 'influxdb', got %s", checker.Name())
	}

	health := checker.Check(context.Background())
	if health.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
	if health.Message != "connected" {
		t.Errorf("Expected message 'connected', got %s", health.Message)
	}
}

func TestDatabaseCheckerUnhealthy(t *testing.T) {
	// Unhealthy database
	checker := NewDatabaseChecker(func(ctx context.Context) error {
		return errors.New("connection failed")
	})

	health := checker.Check(context.Background())
	if health.Status != StatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %s", health.Status)
	}
	if health.Message != "connection failed" {
		t.Errorf("Expected error message, got %s", health.Message)
	}
}

func TestDatabaseCheckerResponseTime(t *testing.T) {
	// Slow database
	checker := NewDatabaseChecker(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	health := checker.Check(context.Background())
	if health.ResponseMs < 100 {
		t.Errorf("Expected response time >= 100ms, got %dms", health.ResponseMs)
	}
}

func TestUptime(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	response := handler.Check(context.Background())

	if response.Uptime == "" {
		t.Error("Expected uptime to be set")
	}
}

func TestMultipleCheckers(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

	handler.AddChecker(&mockChecker{name: "database", status: StatusHealthy})
	handler.AddChecker(&mockChecker{name: "api", status: StatusHealthy})
	handler.AddChecker(&mockChecker{name: "cache", status: StatusHealthy})

	response := handler.Check(context.Background())

	if len(response.Components) != 3 {
		t.Errorf("Expected 3 components, got %d", len(response.Components))
	}

	// Check all components are present
	if _, ok := response.Components["database"]; !ok {
		t.Error("Expected database component")
	}
	if _, ok := response.Components["api"]; !ok {
		t.Error("Expected api component")
	}
	if _, ok := response.Components["cache"]; !ok {
		t.Error("Expected cache component")
	}
}

func TestPollSuccessRate(t *testing.T) {
	collector := metrics.New()

	// Record some polls
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, true)
	collector.RecordPollStart()
	collector.RecordPollComplete(1*time.Second, false)

	handler := NewHandler(collector, "1.0.0")
	response := handler.Check(context.Background())

	if response.Metrics == nil {
		t.Fatal("Expected metrics")
	}

	// 2 successful out of 3 = 66.67%
	expectedRate := 66.67
	if response.Metrics.PollSuccessRate < expectedRate-1 || response.Metrics.PollSuccessRate > expectedRate+1 {
		t.Errorf("Expected success rate ~%.2f%%, got %.2f%%", expectedRate, response.Metrics.PollSuccessRate)
	}
}

// Mock checker for testing
type mockChecker struct {
	name    string
	status  Status
	message string
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context) ComponentHealth {
	return ComponentHealth{
		Status:      m.status,
		Message:     m.message,
		LastChecked: time.Now(),
		ResponseMs:  10,
	}
}
