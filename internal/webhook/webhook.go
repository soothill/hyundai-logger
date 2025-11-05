// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"

	// Default webhook timeout
	defaultWebhookTimeout = 10 * time.Second

	// Alert level colors (hex format for Slack, used for both Slack and Discord)
	colorGreen   = "#36a64f" // Info
	colorOrange  = "#ff9900" // Warning
	colorRed     = "#ff0000" // Error
	colorDarkRed = "#8b0000" // Critical
	colorGray    = "#808080" // Default/Unknown

	// Discord color codes (decimal equivalents)
	discordColorGreen   = 3581519  // #36a64f
	discordColorOrange  = 16750080 // #ff9900
	discordColorRed     = 16711680 // #ff0000
	discordColorDarkRed = 9109504  // #8b0000
	discordColorGray    = 8421504  // #808080
)

// Alert represents a notification to be sent
type Alert struct {
	Level      AlertLevel
	Title      string
	Message    string
	Timestamp  time.Time
	VehicleVIN string
	Metadata   map[string]interface{}
}

// Config holds webhook configuration
type Config struct {
	Enabled bool
	Slack   SlackConfig
	Discord DiscordConfig
	Generic []GenericConfig
	Timeout time.Duration
}

// SlackConfig holds Slack webhook configuration
type SlackConfig struct {
	Enabled    bool
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
}

// DiscordConfig holds Discord webhook configuration
type DiscordConfig struct {
	Enabled    bool
	WebhookURL string
	Username   string
	AvatarURL  string
}

// GenericConfig holds generic webhook configuration
type GenericConfig struct {
	Name     string
	URL      string
	Method   string
	Headers  map[string]string
	Template string
}

// Notifier sends notifications via webhooks
type Notifier struct {
	config Config
	client *http.Client
}

// New creates a new webhook notifier
func New(config Config) *Notifier {
	if config.Timeout == 0 {
		config.Timeout = defaultWebhookTimeout
	}

	return &Notifier{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SendAlert sends an alert to all configured webhooks
func (n *Notifier) SendAlert(ctx context.Context, alert Alert) error {
	if !n.config.Enabled {
		return nil
	}

	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	var errors []error

	// Send to Slack
	if n.config.Slack.Enabled {
		if err := n.sendSlack(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("slack: %w", err))
		}
	}

	// Send to Discord
	if n.config.Discord.Enabled {
		if err := n.sendDiscord(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("discord: %w", err))
		}
	}

	// Send to generic webhooks
	for _, webhook := range n.config.Generic {
		if err := n.sendGeneric(ctx, alert, webhook); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", webhook.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("webhook errors: %v", errors)
	}

	return nil
}

// sendSlack sends a notification to Slack
func (n *Notifier) sendSlack(ctx context.Context, alert Alert) error {
	payload := map[string]interface{}{
		"text": fmt.Sprintf("*%s*: %s", alert.Title, alert.Message),
	}

	if n.config.Slack.Channel != "" {
		payload["channel"] = n.config.Slack.Channel
	}

	if n.config.Slack.Username != "" {
		payload["username"] = n.config.Slack.Username
	}

	if n.config.Slack.IconEmoji != "" {
		payload["icon_emoji"] = n.config.Slack.IconEmoji
	}

	// Add color based on alert level
	color := n.getSlackColor(alert.Level)
	attachments := []map[string]interface{}{
		{
			"color": color,
			"fields": []map[string]interface{}{
				{
					"title": "Level",
					"value": string(alert.Level),
					"short": true,
				},
				{
					"title": "Time",
					"value": alert.Timestamp.Format("2006-01-02 15:04:05"),
					"short": true,
				},
			},
		},
	}

	if alert.VehicleVIN != "" {
		attachments[0]["fields"] = append(
			attachments[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{
				"title": "Vehicle",
				"value": alert.VehicleVIN,
				"short": true,
			},
		)
	}

	payload["attachments"] = attachments

	return n.sendJSON(ctx, n.config.Slack.WebhookURL, payload)
}

// sendDiscord sends a notification to Discord
func (n *Notifier) sendDiscord(ctx context.Context, alert Alert) error {
	payload := map[string]interface{}{
		"content": fmt.Sprintf("**%s**\n%s", alert.Title, alert.Message),
	}

	if n.config.Discord.Username != "" {
		payload["username"] = n.config.Discord.Username
	}

	if n.config.Discord.AvatarURL != "" {
		payload["avatar_url"] = n.config.Discord.AvatarURL
	}

	// Add embed with color based on alert level
	color := n.getDiscordColor(alert.Level)
	fields := []map[string]interface{}{
		{
			"name":   "Level",
			"value":  string(alert.Level),
			"inline": true,
		},
		{
			"name":   "Time",
			"value":  alert.Timestamp.Format("2006-01-02 15:04:05"),
			"inline": true,
		},
	}

	if alert.VehicleVIN != "" {
		fields = append(fields, map[string]interface{}{
			"name":   "Vehicle",
			"value":  alert.VehicleVIN,
			"inline": true,
		})
	}

	embeds := []map[string]interface{}{
		{
			"title":       alert.Title,
			"description": alert.Message,
			"color":       color,
			"fields":      fields,
			"timestamp":   alert.Timestamp.Format(time.RFC3339),
		},
	}

	payload["embeds"] = embeds

	return n.sendJSON(ctx, n.config.Discord.WebhookURL, payload)
}

// sendHTTPJSON sends a JSON payload via HTTP with custom method and headers
func (n *Notifier) sendHTTPJSON(ctx context.Context, method, url string, payload interface{}, headers map[string]string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendGeneric sends a notification to a generic webhook
func (n *Notifier) sendGeneric(ctx context.Context, alert Alert, config GenericConfig) error {
	payload := map[string]interface{}{
		"level":       string(alert.Level),
		"title":       alert.Title,
		"message":     alert.Message,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"vehicle_vin": alert.VehicleVIN,
		"metadata":    alert.Metadata,
	}

	method := config.Method
	if method == "" {
		method = "POST"
	}

	return n.sendHTTPJSON(ctx, method, config.URL, payload, config.Headers)
}

// sendJSON sends a JSON payload to a webhook URL
func (n *Notifier) sendJSON(ctx context.Context, url string, payload interface{}) error {
	return n.sendHTTPJSON(ctx, "POST", url, payload, nil)
}

// getSlackColor returns a Slack color for the alert level
func (n *Notifier) getSlackColor(level AlertLevel) string {
	switch level {
	case AlertLevelInfo:
		return colorGreen
	case AlertLevelWarning:
		return colorOrange
	case AlertLevelError:
		return colorRed
	case AlertLevelCritical:
		return colorDarkRed
	default:
		return colorGray
	}
}

// getDiscordColor returns a Discord color (decimal) for the alert level
func (n *Notifier) getDiscordColor(level AlertLevel) int {
	switch level {
	case AlertLevelInfo:
		return discordColorGreen
	case AlertLevelWarning:
		return discordColorOrange
	case AlertLevelError:
		return discordColorRed
	case AlertLevelCritical:
		return discordColorDarkRed
	default:
		return discordColorGray
	}
}

// TestConnection tests if the webhook is reachable
func (n *Notifier) TestConnection(ctx context.Context) error {
	testAlert := Alert{
		Level:     AlertLevelInfo,
		Title:     "Test Notification",
		Message:   "This is a test notification from Hyundai Logger",
		Timestamp: time.Now(),
	}

	return n.SendAlert(ctx, testAlert)
}
