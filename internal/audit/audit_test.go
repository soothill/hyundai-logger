// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	// Test with stdout
	logger, err := NewLogger(Config{OutputPath: "-"})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	// Test with file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err = NewLogger(Config{OutputPath: logPath})
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}

	defer func() { _ = logger.Close() }()

	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestLogEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	event := &AuditEvent{
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     "user@example.com",
		Resource:  "authentication",
		Action:    "login",
		Result:    "success",
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify output
	output := buf.String()
	if output == "" {
		t.Error("expected output, got empty string")
	}

	// Parse JSON
	var logged AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &logged); err != nil {
		t.Fatalf("failed to parse logged event: %v", err)
	}

	if logged.EventType != EventAuthentication {
		t.Errorf("expected event type %s, got %s", EventAuthentication, logged.EventType)
	}

	if logged.Actor != "user@example.com" {
		t.Errorf("expected actor user@example.com, got %s", logged.Actor)
	}

	if logged.HMAC == "" {
		t.Error("HMAC should not be empty")
	}
}

func TestGenerateHMAC(t *testing.T) {
	logger := &Logger{
		hmacKey: []byte("test-hmac-key"),
	}

	event := &AuditEvent{
		ID:        "test-1",
		Timestamp: time.Now(),
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     "test-user",
		Resource:  "test-resource",
		Action:    "test-action",
		Result:    "success",
	}

	hmac1 := logger.generateHMAC(event)
	if hmac1 == "" {
		t.Error("HMAC should not be empty")
	}

	// Same event should produce same HMAC
	hmac2 := logger.generateHMAC(event)
	if hmac1 != hmac2 {
		t.Error("same event should produce same HMAC")
	}

	// Different event should produce different HMAC
	event.Actor = "different-user"
	hmac3 := logger.generateHMAC(event)
	if hmac1 == hmac3 {
		t.Error("different event should produce different HMAC")
	}
}

func TestVerifyHMAC(t *testing.T) {
	logger := &Logger{
		hmacKey: []byte("test-verification-key"),
	}

	event := &AuditEvent{
		ID:        "verify-1",
		Timestamp: time.Now(),
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     "test-user",
		Resource:  "test-resource",
		Action:    "login",
		Result:    "success",
	}

	// Generate HMAC
	event.HMAC = logger.generateHMAC(event)

	// Verify should pass
	if !logger.VerifyHMAC(event) {
		t.Error("HMAC verification should pass for unmodified event")
	}

	// Modify event - should fail
	originalActor := event.Actor
	event.Actor = "tampered-user"
	if logger.VerifyHMAC(event) {
		t.Error("HMAC verification should fail for modified event")
	}

	// Restore and verify again
	event.Actor = originalActor
	if !logger.VerifyHMAC(event) {
		t.Error("HMAC verification should pass after restoring original value")
	}
}

func TestLogAuthentication(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	details := map[string]interface{}{
		"method": "password",
		"ip":     "192.168.1.1",
	}

	err := logger.LogAuthentication("user@example.com", "success", details)
	if err != nil {
		t.Fatalf("failed to log authentication: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventAuthentication {
		t.Errorf("expected event type %s, got %s", EventAuthentication, event.EventType)
	}

	if event.Severity != SeverityInfo {
		t.Errorf("expected severity info, got %s", event.Severity)
	}

	// Test failed authentication
	buf.Reset()
	err = logger.LogAuthentication("user@example.com", "failure", nil)
	if err != nil {
		t.Fatalf("failed to log failed authentication: %v", err)
	}

	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if event.Severity != SeverityWarning {
		t.Errorf("failed auth should have warning severity, got %s", event.Severity)
	}
}

func TestLogVehicleAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	details := map[string]interface{}{
		"operation": "status_check",
	}

	err := logger.LogVehicleAccess("user@example.com", "VIN123", "read", "success", details)
	if err != nil {
		t.Fatalf("failed to log vehicle access: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventVehicleAccess {
		t.Errorf("expected event type %s, got %s", EventVehicleAccess, event.EventType)
	}

	if !strings.Contains(event.Resource, "VIN123") {
		t.Errorf("resource should contain VIN123, got %s", event.Resource)
	}
}

func TestLogDataExport(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	err := logger.LogDataExport("user@example.com", "csv", 1000, "success")
	if err != nil {
		t.Fatalf("failed to log data export: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventDataExport {
		t.Errorf("expected event type %s, got %s", EventDataExport, event.EventType)
	}

	if event.Details["format"] != "csv" {
		t.Errorf("expected format csv, got %v", event.Details["format"])
	}

	if event.Details["record_count"] != float64(1000) { // JSON unmarshals numbers as float64
		t.Errorf("expected record_count 1000, got %v", event.Details["record_count"])
	}
}

func TestLogConfigChange(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	err := logger.LogConfigChange("admin", "poll_interval", "5", "10", "success")
	if err != nil {
		t.Fatalf("failed to log config change: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventConfigChange {
		t.Errorf("expected event type %s, got %s", EventConfigChange, event.EventType)
	}

	if event.Severity != SeverityWarning {
		t.Errorf("config changes should have warning severity, got %s", event.Severity)
	}

	if event.Details["old_value"] != "5" {
		t.Errorf("expected old_value 5, got %v", event.Details["old_value"])
	}
}

func TestLogAPIKeyOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	// Test key creation
	scopes := []string{"read", "write"}
	err := logger.LogAPIKeyCreate("admin", "test-key", scopes)
	if err != nil {
		t.Fatalf("failed to log API key create: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventAPIKeyCreate {
		t.Errorf("expected event type %s, got %s", EventAPIKeyCreate, event.EventType)
	}

	// Test key revocation
	buf.Reset()
	err = logger.LogAPIKeyRevoke("admin", "test-key", "security breach")
	if err != nil {
		t.Fatalf("failed to log API key revoke: %v", err)
	}

	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventAPIKeyRevoke {
		t.Errorf("expected event type %s, got %s", EventAPIKeyRevoke, event.EventType)
	}

	if event.Details["reason"] != "security breach" {
		t.Errorf("expected reason 'security breach', got %v", event.Details["reason"])
	}

	// Test key usage
	buf.Reset()
	err = logger.LogAPIKeyUse("test-key", "read", "vehicle:VIN123", "192.168.1.1")
	if err != nil {
		t.Fatalf("failed to log API key use: %v", err)
	}

	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventAPIKeyUse {
		t.Errorf("expected event type %s, got %s", EventAPIKeyUse, event.EventType)
	}

	if event.IPAddress != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", event.IPAddress)
	}
}

func TestLogSystemEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-key"),
	}

	details := map[string]interface{}{
		"version": "1.0.0",
	}

	err := logger.LogSystemEvent(EventSystemStart, details)
	if err != nil {
		t.Fatalf("failed to log system event: %v", err)
	}

	var event AuditEvent
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if event.EventType != EventSystemStart {
		t.Errorf("expected event type %s, got %s", EventSystemStart, event.EventType)
	}

	if event.Actor != "system" {
		t.Errorf("expected actor 'system', got %s", event.Actor)
	}
}

func TestEventConstants(t *testing.T) {
	events := []EventType{
		EventAuthentication,
		EventAuthorization,
		EventVehicleAccess,
		EventDataExport,
		EventConfigChange,
		EventAPIKeyCreate,
		EventAPIKeyRevoke,
		EventAPIKeyUse,
		EventSystemStart,
		EventSystemStop,
	}

	for _, event := range events {
		if event == "" {
			t.Error("event type should not be empty")
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	severities := []Severity{
		SeverityInfo,
		SeverityWarning,
		SeverityError,
		SeverityCritical,
	}

	for _, severity := range severities {
		if severity == "" {
			t.Error("severity should not be empty")
		}
	}
}

func TestFileLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit-test.log")

	logger, err := NewLogger(Config{
		OutputPath: logPath,
		HMACKey:    []byte("test-file-key"),
	})
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Log multiple events
	for i := 0; i < 5; i++ {
		err := logger.LogAuthentication("user@example.com", "success", nil)
		if err != nil {
			t.Fatalf("failed to log event %d: %v", i, err)
		}
	}

	// Close to flush
	_ = logger.Close()

	// Read file
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 log lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("test-concurrent-key"),
	}

	done := make(chan bool)
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				_ = logger.LogAuthentication("user@example.com", "success", map[string]interface{}{
					"goroutine": id,
					"iteration": j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Should have 100 log entries
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Errorf("expected 100 log entries, got %d", len(lines))
	}
}

// Benchmark tests
func BenchmarkLogEvent(b *testing.B) {
	var buf bytes.Buffer
	logger := &Logger{
		writer:  &buf,
		hmacKey: []byte("benchmark-key"),
	}

	event := &AuditEvent{
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     "user@example.com",
		Resource:  "authentication",
		Action:    "login",
		Result:    "success",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(event)
	}
}

func BenchmarkGenerateHMAC(b *testing.B) {
	logger := &Logger{
		hmacKey: []byte("benchmark-hmac-key"),
	}

	event := &AuditEvent{
		ID:        "bench-1",
		Timestamp: time.Now(),
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     "user@example.com",
		Resource:  "test",
		Action:    "test",
		Result:    "success",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.generateHMAC(event)
	}
}
