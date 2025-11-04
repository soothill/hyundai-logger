// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	EventAuthentication EventType = "authentication"
	EventAuthorization  EventType = "authorization"
	EventVehicleAccess  EventType = "vehicle_access"
	EventDataExport     EventType = "data_export"
	EventConfigChange   EventType = "config_change"
	EventAPIKeyCreate   EventType = "apikey_create"
	EventAPIKeyRevoke   EventType = "apikey_revoke"
	EventAPIKeyUse      EventType = "apikey_use"
	EventSystemStart    EventType = "system_start"
	EventSystemStop     EventType = "system_stop"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType EventType              `json:"event_type"`
	Severity  Severity               `json:"severity"`
	Actor     string                 `json:"actor"`    // User/service that performed the action
	Resource  string                 `json:"resource"` // Resource affected
	Action    string                 `json:"action"`   // Action performed
	Result    string                 `json:"result"`   // success/failure
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	HMAC      string                 `json:"hmac"` // Tamper-proof signature
}

// Logger handles audit logging
type Logger struct {
	mu         sync.Mutex
	writer     io.Writer
	hmacKey    []byte
	eventCount int64
}

// Config configures the audit logger
type Config struct {
	OutputPath string // Path to audit log file
	HMACKey    []byte // Secret key for HMAC signatures
}

// NewLogger creates a new audit logger
func NewLogger(config Config) (*Logger, error) {
	var writer io.Writer

	if config.OutputPath == "" || config.OutputPath == "-" {
		writer = os.Stdout
	} else {
		// Open file in append mode with restricted permissions
		f, err := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("opening audit log file: %w", err)
		}
		writer = f
	}

	// Generate HMAC key if not provided
	hmacKey := config.HMACKey
	if len(hmacKey) == 0 {
		// Generate default key (in production, this should come from secure storage)
		hmacKey = []byte("default-audit-hmac-key-change-in-production")
	}

	return &Logger{
		writer:  writer,
		hmacKey: hmacKey,
	}, nil
}

// Log records an audit event
func (l *Logger) Log(event *AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate unique ID
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit-%d-%d", event.Timestamp.Unix(), l.eventCount)
		l.eventCount++
	}

	// Generate HMAC for tamper detection
	event.HMAC = l.generateHMAC(event)

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling audit event: %w", err)
	}

	// Write to log
	data = append(data, '\n')
	if _, err := l.writer.Write(data); err != nil {
		return fmt.Errorf("writing audit event: %w", err)
	}

	return nil
}

// generateHMAC creates a tamper-proof signature for an audit event
func (l *Logger) generateHMAC(event *AuditEvent) string {
	// Create message to sign (without the HMAC field itself)
	message := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.Severity,
		event.Actor,
		event.Resource,
		event.Action,
		event.Result,
	)

	// Create HMAC
	h := hmac.New(sha256.New, l.hmacKey)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC verifies that an audit event has not been tampered with
func (l *Logger) VerifyHMAC(event *AuditEvent) bool {
	expectedHMAC := l.generateHMAC(event)
	return hmac.Equal([]byte(event.HMAC), []byte(expectedHMAC))
}

// LogAuthentication logs an authentication event
func (l *Logger) LogAuthentication(actor, result string, details map[string]interface{}) error {
	event := &AuditEvent{
		EventType: EventAuthentication,
		Severity:  SeverityInfo,
		Actor:     actor,
		Resource:  "authentication",
		Action:    "login",
		Result:    result,
		Details:   details,
	}

	if result != "success" {
		event.Severity = SeverityWarning
	}

	return l.Log(event)
}

// LogVehicleAccess logs vehicle access events
func (l *Logger) LogVehicleAccess(actor, vehicleID, action, result string, details map[string]interface{}) error {
	event := &AuditEvent{
		EventType: EventVehicleAccess,
		Severity:  SeverityInfo,
		Actor:     actor,
		Resource:  fmt.Sprintf("vehicle:%s", vehicleID),
		Action:    action,
		Result:    result,
		Details:   details,
	}

	return l.Log(event)
}

// LogDataExport logs data export events
func (l *Logger) LogDataExport(actor, format string, recordCount int, result string) error {
	event := &AuditEvent{
		EventType: EventDataExport,
		Severity:  SeverityInfo,
		Actor:     actor,
		Resource:  "data",
		Action:    "export",
		Result:    result,
		Details: map[string]interface{}{
			"format":       format,
			"record_count": recordCount,
		},
	}

	return l.Log(event)
}

// LogConfigChange logs configuration changes
func (l *Logger) LogConfigChange(actor, config, oldValue, newValue, result string) error {
	event := &AuditEvent{
		EventType: EventConfigChange,
		Severity:  SeverityWarning, // Config changes are important
		Actor:     actor,
		Resource:  fmt.Sprintf("config:%s", config),
		Action:    "update",
		Result:    result,
		Details: map[string]interface{}{
			"old_value": oldValue,
			"new_value": newValue,
		},
	}

	return l.Log(event)
}

// LogAPIKeyCreate logs API key creation
func (l *Logger) LogAPIKeyCreate(actor, keyName string, scopes []string) error {
	event := &AuditEvent{
		EventType: EventAPIKeyCreate,
		Severity:  SeverityInfo,
		Actor:     actor,
		Resource:  fmt.Sprintf("apikey:%s", keyName),
		Action:    "create",
		Result:    "success",
		Details: map[string]interface{}{
			"scopes": scopes,
		},
	}

	return l.Log(event)
}

// LogAPIKeyRevoke logs API key revocation
func (l *Logger) LogAPIKeyRevoke(actor, keyName, reason string) error {
	event := &AuditEvent{
		EventType: EventAPIKeyRevoke,
		Severity:  SeverityWarning,
		Actor:     actor,
		Resource:  fmt.Sprintf("apikey:%s", keyName),
		Action:    "revoke",
		Result:    "success",
		Details: map[string]interface{}{
			"reason": reason,
		},
	}

	return l.Log(event)
}

// LogAPIKeyUse logs API key usage
func (l *Logger) LogAPIKeyUse(keyName, action, resource string, ipAddress string) error {
	event := &AuditEvent{
		EventType: EventAPIKeyUse,
		Severity:  SeverityInfo,
		Actor:     fmt.Sprintf("apikey:%s", keyName),
		Resource:  resource,
		Action:    action,
		Result:    "success",
		IPAddress: ipAddress,
	}

	return l.Log(event)
}

// LogSystemEvent logs system lifecycle events
func (l *Logger) LogSystemEvent(eventType EventType, details map[string]interface{}) error {
	event := &AuditEvent{
		EventType: eventType,
		Severity:  SeverityInfo,
		Actor:     "system",
		Resource:  "system",
		Action:    string(eventType),
		Result:    "success",
		Details:   details,
	}

	return l.Log(event)
}

// Close closes the audit logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Reader reads and verifies audit logs
type Reader struct {
	hmacKey []byte
}

// NewReader creates a new audit log reader
func NewReader(hmacKey []byte) *Reader {
	return &Reader{
		hmacKey: hmacKey,
	}
}

// ReadEvents reads audit events from a file
func (r *Reader) ReadEvents(path string) ([]*AuditEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading audit log: %w", err)
	}

	var events []*AuditEvent
	lines := 0

	// Parse JSON lines
	decoder := json.NewDecoder(io.Reader(io.NopCloser(io.Reader(&io.LimitedReader{
		R: io.NopCloser(io.Reader(&limitedBytesReader{data: data})),
		N: int64(len(data)),
	}))))

	for {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parsing audit event at line %d: %w", lines+1, err)
		}
		lines++
		events = append(events, &event)
	}

	return events, nil
}

// VerifyIntegrity verifies the integrity of audit events
func (r *Reader) VerifyIntegrity(events []*AuditEvent) (valid, invalid int) {
	logger := &Logger{hmacKey: r.hmacKey}

	for _, event := range events {
		if logger.VerifyHMAC(event) {
			valid++
		} else {
			invalid++
		}
	}

	return valid, invalid
}

// Helper type for reading bytes
type limitedBytesReader struct {
	data []byte
	pos  int
}

func (r *limitedBytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *limitedBytesReader) Close() error {
	return nil
}
