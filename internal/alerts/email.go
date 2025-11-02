// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package alerts

import (
	"fmt"
	"net/smtp"
	"sync"
	"time"
)

// Config defines email alerting configuration
type Config struct {
	Enabled           bool
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	FromEmail         string
	ToEmail           string
	AlertThreshold    int
	AlertCooldownMins int
}

// Alerter handles email notifications for persistent errors
type Alerter struct {
	config        Config
	mu            sync.Mutex
	lastAlertTime time.Time
	enabled       bool
}

// NewAlerter creates a new email alerter
func NewAlerter(config Config) *Alerter {
	return &Alerter{
		config:  config,
		enabled: config.Enabled,
	}
}

// SendAlert sends an email alert if conditions are met
func (a *Alerter) SendAlert(subject, body string) error {
	if !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check cooldown period
	if time.Since(a.lastAlertTime) < time.Duration(a.config.AlertCooldownMins)*time.Minute {
		return nil // Skip alert due to cooldown
	}

	// Send email
	if err := a.sendEmail(subject, body); err != nil {
		return fmt.Errorf("sending alert email: %w", err)
	}

	a.lastAlertTime = time.Now()
	return nil
}

// sendEmail sends an email via SMTP
func (a *Alerter) sendEmail(subject, body string) error {
	// Build email message
	msg := fmt.Sprintf("From: %s\r\n", a.config.FromEmail)
	msg += fmt.Sprintf("To: %s\r\n", a.config.ToEmail)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	msg += "\r\n"
	msg += body

	// SMTP authentication
	auth := smtp.PlainAuth("", a.config.SMTPUsername, a.config.SMTPPassword, a.config.SMTPHost)

	// Send email
	addr := fmt.Sprintf("%s:%d", a.config.SMTPHost, a.config.SMTPPort)
	err := smtp.SendMail(addr, auth, a.config.FromEmail, []string{a.config.ToEmail}, []byte(msg))
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}

// FormatErrorAlert creates a formatted HTML email body for error alerts
func FormatErrorAlert(errorType string, errorCount int, errorDetails string, lastSuccess time.Time) string {
	body := `
<!DOCTYPE html>
<html>
<head>
<style>
body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
.alert-box { background-color: #f8d7da; border: 1px solid #f5c6cb; border-radius: 4px; padding: 20px; margin: 20px 0; }
.alert-header { color: #721c24; font-size: 24px; font-weight: bold; margin-bottom: 10px; }
.info-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
.info-table th { background-color: #f2f2f2; text-align: left; padding: 10px; border: 1px solid #ddd; }
.info-table td { padding: 10px; border: 1px solid #ddd; }
.details { background-color: #f8f9fa; padding: 15px; border-radius: 4px; font-family: monospace; white-space: pre-wrap; }
</style>
</head>
<body>
<div class="alert-box">
<div class="alert-header">⚠️ Hyundai Logger Alert</div>
<p>Your Hyundai logger has encountered persistent errors that require attention.</p>
</div>

<table class="info-table">
<tr>
<th>Error Type</th>
<td>%s</td>
</tr>
<tr>
<th>Consecutive Failures</th>
<td>%d</td>
</tr>
<tr>
<th>Last Successful Poll</th>
<td>%s</td>
</tr>
<tr>
<th>Alert Time</th>
<td>%s</td>
</tr>
</table>

<h3>Error Details:</h3>
<div class="details">%s</div>

<p><strong>Action Required:</strong> Please check your logger service, network connectivity, and Hyundai Bluelink API credentials.</p>

<hr>
<p style="color: #666; font-size: 12px;">This is an automated alert from Hyundai Logger. Do not reply to this email.</p>
</body>
</html>
`

	lastSuccessStr := "Never"
	if !lastSuccess.IsZero() {
		lastSuccessStr = lastSuccess.Format("2006-01-02 15:04:05 MST")
	}

	//nolint:staticcheck // SA5009: False positive - format string is valid but staticcheck can't parse HTML content
	return fmt.Sprintf(body,
		errorType,
		errorCount,
		lastSuccessStr,
		time.Now().Format("2006-01-02 15:04:05 MST"),
		errorDetails,
	)
}
