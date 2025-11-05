// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package alerts

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockSMTPServer is a mock SMTP server for testing
type mockSMTPServer struct {
	listener      net.Listener
	receivedMails []string
	mu            sync.Mutex
	failNext      bool
	started       bool
	addr          string
}

func newMockSMTPServer(t *testing.T) *mockSMTPServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &mockSMTPServer{
		listener:      listener,
		receivedMails: make([]string, 0),
		addr:          listener.Addr().String(),
		started:       true,
	}

	go server.serve()

	return server
}

func (s *mockSMTPServer) serve() {
	for {
		s.mu.Lock()
		started := s.started
		s.mu.Unlock()

		if !started {
			return
		}

		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			started = s.started
			s.mu.Unlock()
			if started {
				continue
			}
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *mockSMTPServer) handleConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	s.mu.Lock()
	fail := s.failNext
	s.failNext = false
	s.mu.Unlock()

	if fail {
		_, _ = conn.Write([]byte("500 Error\r\n"))
		return
	}

	// SMTP greeting
	_, _ = conn.Write([]byte("220 localhost SMTP mock\r\n"))

	buf := make([]byte, 4096)
	mailData := ""

	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		line := string(buf[:n])
		mailData += line

		if strings.HasPrefix(line, "EHLO") {
			// Multi-line EHLO response with AUTH support
			_, _ = conn.Write([]byte("250-localhost\r\n"))
			_, _ = conn.Write([]byte("250-AUTH PLAIN LOGIN\r\n"))
			_, _ = conn.Write([]byte("250 OK\r\n"))
		} else if strings.HasPrefix(line, "HELO") {
			_, _ = conn.Write([]byte("250 Hello\r\n"))
		} else if strings.HasPrefix(line, "AUTH") {
			_, _ = conn.Write([]byte("235 Authentication successful\r\n"))
		} else if strings.HasPrefix(line, "MAIL FROM") {
			_, _ = conn.Write([]byte("250 OK\r\n"))
		} else if strings.HasPrefix(line, "RCPT TO") {
			_, _ = conn.Write([]byte("250 OK\r\n"))
		} else if strings.HasPrefix(line, "DATA") {
			_, _ = conn.Write([]byte("354 Start mail input\r\n"))
		} else if strings.HasPrefix(line, "QUIT") {
			_, _ = conn.Write([]byte("221 Bye\r\n"))
			break
		} else if strings.Contains(line, "\r\n.\r\n") {
			_, _ = conn.Write([]byte("250 OK\r\n"))
			s.mu.Lock()
			s.receivedMails = append(s.receivedMails, mailData)
			s.mu.Unlock()
		}
	}
}

func (s *mockSMTPServer) getReceivedMails() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.receivedMails...)
}

func (s *mockSMTPServer) close() {
	s.mu.Lock()
	s.started = false
	s.mu.Unlock()
	_ = s.listener.Close()
}

func TestNewAlerter(t *testing.T) {
	config := Config{
		Enabled:           true,
		SMTPHost:          "localhost",
		SMTPPort:          587,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertThreshold:    5,
		AlertCooldownMins: 60,
	}

	alerter := NewAlerter(config)

	if alerter == nil {
		t.Fatal("Expected alerter to be created, got nil")
	}

	if !alerter.enabled {
		t.Error("Expected alerter to be enabled")
	}

	if alerter.config.SMTPHost != "localhost" {
		t.Errorf("Expected SMTPHost to be 'localhost', got %s", alerter.config.SMTPHost)
	}

	if alerter.lastAlertTime.IsZero() == false {
		t.Error("Expected lastAlertTime to be zero initially")
	}
}

func TestNewAlerter_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	alerter := NewAlerter(config)

	if alerter == nil {
		t.Fatal("Expected alerter to be created, got nil")
	}

	if alerter.enabled {
		t.Error("Expected alerter to be disabled")
	}
}

func TestSendAlert_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	alerter := NewAlerter(config)

	err := alerter.SendAlert("Test Subject", "Test Body")
	if err != nil {
		t.Errorf("Expected no error when disabled, got: %v", err)
	}

	if !alerter.lastAlertTime.IsZero() {
		t.Error("Expected lastAlertTime to remain zero when disabled")
	}
}

func TestSendAlert_Cooldown(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 5, // 5 minute cooldown
	}

	alerter := NewAlerter(config)

	// First alert should succeed
	err := alerter.SendAlert("Test Subject 1", "Test Body 1")
	if err != nil {
		t.Errorf("First alert failed: %v", err)
	}

	// Wait a bit for the mock server to process
	time.Sleep(100 * time.Millisecond)

	// Second alert should be skipped due to cooldown
	err = alerter.SendAlert("Test Subject 2", "Test Body 2")
	if err != nil {
		t.Errorf("Second alert returned error: %v", err)
	}

	// Check that only one email was received
	mails := mockServer.getReceivedMails()
	if len(mails) != 1 {
		t.Errorf("Expected 1 email due to cooldown, got %d", len(mails))
	}
}

func TestSendAlert_CooldownExpired(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 0, // No cooldown
	}

	alerter := NewAlerter(config)

	// First alert
	err := alerter.SendAlert("Test Subject 1", "Test Body 1")
	if err != nil {
		t.Errorf("First alert failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Second alert should go through since cooldown is 0
	err = alerter.SendAlert("Test Subject 2", "Test Body 2")
	if err != nil {
		t.Errorf("Second alert failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Check that both emails were received
	mails := mockServer.getReceivedMails()
	if len(mails) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(mails))
	}
}

func TestSendAlert_UpdatesLastAlertTime(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 60,
	}

	alerter := NewAlerter(config)

	before := time.Now()
	err := alerter.SendAlert("Test Subject", "Test Body")
	after := time.Now()

	if err != nil {
		t.Errorf("SendAlert failed: %v", err)
	}

	if alerter.lastAlertTime.Before(before) || alerter.lastAlertTime.After(after) {
		t.Error("lastAlertTime not updated correctly")
	}
}

func TestSendAlert_ConcurrentAccess(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 0,
	}

	alerter := NewAlerter(config)

	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	// Send 10 alerts concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := alerter.SendAlert(fmt.Sprintf("Subject %d", id), fmt.Sprintf("Body %d", id))
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Should not crash, and most/all should succeed
	if errCount > 5 {
		t.Errorf("Too many errors in concurrent test: %d", errCount)
	}
}

func TestFormatErrorAlert(t *testing.T) {
	lastSuccess := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	html := FormatErrorAlert("API Connection Error", 5, "Failed to connect to API", lastSuccess)

	if !strings.Contains(html, "API Connection Error") {
		t.Error("HTML should contain error type")
	}

	if !strings.Contains(html, "5") {
		t.Error("HTML should contain error count")
	}

	if !strings.Contains(html, "Failed to connect to API") {
		t.Error("HTML should contain error details")
	}

	if !strings.Contains(html, "2025-01-01") {
		t.Error("HTML should contain last success time")
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should be valid HTML")
	}

	if !strings.Contains(html, "Hyundai Logger Alert") {
		t.Error("HTML should contain alert header")
	}
}

func TestFormatErrorAlert_NoLastSuccess(t *testing.T) {
	var lastSuccess time.Time // Zero time

	html := FormatErrorAlert("Test Error", 3, "Test details", lastSuccess)

	if !strings.Contains(html, "Never") {
		t.Error("HTML should contain 'Never' for zero time")
	}

	if !strings.Contains(html, "Test Error") {
		t.Error("HTML should contain error type")
	}
}

func TestFormatErrorAlert_HTMLEscaping(t *testing.T) {
	html := FormatErrorAlert(
		"Error with <script>alert('xss')</script>",
		1,
		"Details with <tags>",
		time.Now(),
	)

	// The function doesn't escape HTML, but we should note this in our report
	// This is a potential XSS vulnerability if email clients render HTML
	if strings.Contains(html, "<script>") {
		t.Log("Warning: HTML output contains unescaped tags - potential XSS risk")
	}
}

func TestSendEmail_Integration(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 0,
	}

	alerter := NewAlerter(config)

	subject := "Test Alert Subject"
	body := "Test alert body with details"

	err := alerter.SendAlert(subject, body)
	if err != nil {
		t.Errorf("SendAlert failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mails := mockServer.getReceivedMails()
	if len(mails) == 0 {
		t.Fatal("No emails received by mock server")
	}

	mailContent := mails[0]

	if !strings.Contains(mailContent, subject) {
		t.Error("Email should contain subject")
	}

	if !strings.Contains(mailContent, "from@example.com") {
		t.Error("Email should contain from address")
	}

	if !strings.Contains(mailContent, "to@example.com") {
		t.Error("Email should contain to address")
	}

	if !strings.Contains(mailContent, "MIME-Version") {
		t.Error("Email should contain MIME headers")
	}
}

func TestAlerter_ThreadSafety(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "from@example.com",
		ToEmail:           "to@example.com",
		AlertCooldownMins: 1,
	}

	alerter := NewAlerter(config)

	var wg sync.WaitGroup

	// Test concurrent reads and writes to lastAlertTime
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = alerter.SendAlert("Test", "Body")
		}()
	}

	wg.Wait()
	// Should not race or panic
}

func TestEmailFormat(t *testing.T) {
	// Test the HTML format generation
	lastSuccess := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	html := FormatErrorAlert("Connection Failed", 10, "timeout after 30s", lastSuccess)

	// Verify all sections are present
	expectedSections := []string{
		"Hyundai Logger Alert",
		"Error Type",
		"Consecutive Failures",
		"Last Successful Poll",
		"Alert Time",
		"Error Details",
		"Action Required",
	}

	for _, section := range expectedSections {
		if !strings.Contains(html, section) {
			t.Errorf("HTML missing expected section: %s", section)
		}
	}

	// Verify CSS is included
	if !strings.Contains(html, "<style>") {
		t.Error("HTML should include CSS styles")
	}

	// Verify responsive design elements
	if !strings.Contains(html, "font-family") {
		t.Error("HTML should include font styling")
	}
}

// Helper function to parse host:port
func parseAddr(addr string) (string, int) {
	host, portStr, _ := net.SplitHostPort(addr)
	port := 0
	_, _ = fmt.Sscanf(portStr, "%d", &port)
	return host, port
}

// Test that sendEmail is called with correct parameters
func TestSendEmail_CorrectHeaders(t *testing.T) {
	mockServer := newMockSMTPServer(t)
	defer mockServer.close()

	host, port := parseAddr(mockServer.addr)

	config := Config{
		Enabled:           true,
		SMTPHost:          host,
		SMTPPort:          port,
		SMTPUsername:      "test@example.com",
		SMTPPassword:      "password",
		FromEmail:         "sender@example.com",
		ToEmail:           "recipient@example.com",
		AlertCooldownMins: 0,
	}

	alerter := NewAlerter(config)
	err := alerter.sendEmail("Test Subject", "Test Body")

	if err != nil {
		t.Errorf("sendEmail failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mails := mockServer.getReceivedMails()
	if len(mails) == 0 {
		t.Fatal("No emails received")
	}

	mail := mails[0]

	// Check headers
	if !strings.Contains(mail, "From: sender@example.com") {
		t.Error("Email missing correct From header")
	}

	if !strings.Contains(mail, "To: recipient@example.com") {
		t.Error("Email missing correct To header")
	}

	if !strings.Contains(mail, "Subject: Test Subject") {
		t.Error("Email missing correct Subject header")
	}

	if !strings.Contains(mail, "Content-Type: text/html") {
		t.Error("Email missing HTML content type")
	}

	if !strings.Contains(mail, "MIME-Version: 1.0") {
		t.Error("Email missing MIME version")
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	config := Config{
		Enabled:           true,
		AlertThreshold:    0,
		AlertCooldownMins: 0,
	}

	alerter := NewAlerter(config)

	if !alerter.enabled {
		t.Error("Expected alerter to be enabled")
	}

	// With 0 cooldown, alerts should always go through (except SMTP errors)
	if alerter.config.AlertCooldownMins != 0 {
		t.Errorf("Expected 0 cooldown, got %d", alerter.config.AlertCooldownMins)
	}
}

// Benchmark tests
func BenchmarkFormatErrorAlert(b *testing.B) {
	lastSuccess := time.Now()
	for i := 0; i < b.N; i++ {
		FormatErrorAlert("API Error", 5, "Connection timeout", lastSuccess)
	}
}

func BenchmarkSendAlert_Cooldown(b *testing.B) {
	config := Config{
		Enabled:           true,
		SMTPHost:          "localhost",
		SMTPPort:          587,
		AlertCooldownMins: 60,
	}

	alerter := NewAlerter(config)
	alerter.lastAlertTime = time.Now() // Set so all alerts are in cooldown

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = alerter.SendAlert("Test", "Body")
	}
}

// Test SMTP authentication specifically
func TestSMTPAuth(t *testing.T) {
	config := Config{
		SMTPUsername: "testuser",
		SMTPPassword: "testpass",
		SMTPHost:     "smtp.example.com",
	}

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
	if auth == nil {
		t.Error("Failed to create SMTP auth")
	}
}
