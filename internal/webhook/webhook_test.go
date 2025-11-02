// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := Config{
		Enabled: true,
		Timeout: 5 * time.Second,
	}

	notifier := New(config)

	if notifier == nil {
		t.Fatal("expected notifier, got nil")
	}

	if notifier.client.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", notifier.client.Timeout)
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	config := Config{
		Enabled: true,
	}

	notifier := New(config)

	if notifier.client.Timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", notifier.client.Timeout)
	}
}

func TestSendAlert_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	notifier := New(config)
	alert := Alert{
		Level:   AlertLevelInfo,
		Title:   "Test",
		Message: "Test message",
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendSlack(t *testing.T) {
	// Create test server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Enabled: true,
		Slack: SlackConfig{
			Enabled:    true,
			WebhookURL: server.URL,
			Channel:    "#test",
			Username:   "TestBot",
			IconEmoji:  ":robot:",
		},
	}

	notifier := New(config)
	alert := Alert{
		Level:      AlertLevelWarning,
		Title:      "Test Alert",
		Message:    "Test message",
		VehicleVIN: "ABC123",
		Timestamp:  time.Now(),
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("no payload received")
	}

	if receivedPayload["channel"] != "#test" {
		t.Errorf("expected channel #test, got %v", receivedPayload["channel"])
	}

	if receivedPayload["username"] != "TestBot" {
		t.Errorf("expected username TestBot, got %v", receivedPayload["username"])
	}
}

func TestSendDiscord(t *testing.T) {
	// Create test server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Enabled: true,
		Discord: DiscordConfig{
			Enabled:    true,
			WebhookURL: server.URL,
			Username:   "TestBot",
			AvatarURL:  "https://example.com/avatar.png",
		},
	}

	notifier := New(config)
	alert := Alert{
		Level:      AlertLevelError,
		Title:      "Test Alert",
		Message:    "Test message",
		VehicleVIN: "DEF456",
		Timestamp:  time.Now(),
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("no payload received")
	}

	if receivedPayload["username"] != "TestBot" {
		t.Errorf("expected username TestBot, got %v", receivedPayload["username"])
	}

	embeds, ok := receivedPayload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in payload")
	}
}

func TestSendGeneric(t *testing.T) {
	// Create test server
	var receivedPayload map[string]interface{}
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Enabled: true,
		Generic: []GenericConfig{
			{
				Name:   "TestWebhook",
				URL:    server.URL,
				Method: "POST",
				Headers: map[string]string{
					"X-Custom-Header": "test-value",
				},
			},
		},
	}

	notifier := New(config)
	alert := Alert{
		Level:      AlertLevelCritical,
		Title:      "Critical Alert",
		Message:    "Critical message",
		VehicleVIN: "GHI789",
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"battery_level": 10,
		},
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("no payload received")
	}

	if receivedPayload["level"] != "critical" {
		t.Errorf("expected level critical, got %v", receivedPayload["level"])
	}

	if receivedPayload["title"] != "Critical Alert" {
		t.Errorf("expected title 'Critical Alert', got %v", receivedPayload["title"])
	}

	if receivedHeaders.Get("X-Custom-Header") != "test-value" {
		t.Errorf("expected custom header, got %v", receivedHeaders.Get("X-Custom-Header"))
	}
}

func TestGetSlackColor(t *testing.T) {
	notifier := New(Config{})

	tests := []struct {
		level    AlertLevel
		expected string
	}{
		{AlertLevelInfo, "#36a64f"},
		{AlertLevelWarning, "#ff9900"},
		{AlertLevelError, "#ff0000"},
		{AlertLevelCritical, "#8b0000"},
		{"unknown", "#808080"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			color := notifier.getSlackColor(tt.level)
			if color != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, color)
			}
		})
	}
}

func TestGetDiscordColor(t *testing.T) {
	notifier := New(Config{})

	tests := []struct {
		level    AlertLevel
		expected int
	}{
		{AlertLevelInfo, 3581519},
		{AlertLevelWarning, 16750080},
		{AlertLevelError, 16711680},
		{AlertLevelCritical, 9109504},
		{"unknown", 8421504},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			color := notifier.getDiscordColor(tt.level)
			if color != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, color)
			}
		})
	}
}

func TestSendAlert_ServerError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer server.Close()

	config := Config{
		Enabled: true,
		Slack: SlackConfig{
			Enabled:    true,
			WebhookURL: server.URL,
		},
	}

	notifier := New(config)
	alert := Alert{
		Level:   AlertLevelInfo,
		Title:   "Test",
		Message: "Test",
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSendAlert_MultipleWebhooks(t *testing.T) {
	slackCalled := false
	discordCalled := false
	genericCalled := false

	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slackCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer slackServer.Close()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discordCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer discordServer.Close()

	genericServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		genericCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer genericServer.Close()

	config := Config{
		Enabled: true,
		Slack: SlackConfig{
			Enabled:    true,
			WebhookURL: slackServer.URL,
		},
		Discord: DiscordConfig{
			Enabled:    true,
			WebhookURL: discordServer.URL,
		},
		Generic: []GenericConfig{
			{
				Name: "Test",
				URL:  genericServer.URL,
			},
		},
	}

	notifier := New(config)
	alert := Alert{
		Level:   AlertLevelInfo,
		Title:   "Test",
		Message: "Test",
	}

	err := notifier.SendAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slackCalled {
		t.Error("slack webhook not called")
	}

	if !discordCalled {
		t.Error("discord webhook not called")
	}

	if !genericCalled {
		t.Error("generic webhook not called")
	}
}
