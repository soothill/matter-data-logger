// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package slacknotifier provides a simple client for sending notifications to Slack
// via Incoming Webhooks.
//
// It supports basic text messages and formatted attachments with severity levels.
//
// # Features
//
//   - Simple API for sending messages and alerts
//   - Support for Slack attachments with color-coded severity
//   - Context-aware HTTP requests with configurable timeouts
//   - Graceful handling of disabled notifiers (empty webhook URL)
//
// # Usage
//
//	// Create a new notifier
//	notifier := slacknotifier.New("https://hooks.slack.com/services/...")
//
//	// Check if the notifier is enabled
//	if notifier.IsEnabled() {
//	    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	    defer cancel()
//
//	    // Send a simple message
//	    err := notifier.SendMessage(ctx, "Hello, Slack!")
//	    if err != nil {
//	        log.Fatalf("Failed to send message: %v", err)
//	    }
//
//	    // Send a formatted alert
//	    err = notifier.SendAlert(ctx, "warning", "High CPU Usage", "CPU is at 90%")
//	    if err != nil {
//	        log.Fatalf("Failed to send alert: %v", err)
//	    }
//	}
package slacknotifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Notifier sends notifications to Slack via webhook
type Notifier struct {
	webhookURL string
	client     *http.Client
	enabled    bool
}

// Message represents a Slack webhook message payload
type Message struct {
	Text        string       `json:"text,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Block represents a Slack block element
type Block struct {
	Type string `json:"type"`
	Text *Text  `json:"text,omitempty"`
}

// Text represents text within a Slack block
type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color  string `json:"color,omitempty"`
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	Footer string `json:"footer,omitempty"`
	Ts     int64  `json:"ts,omitempty"`
}

// New creates a new Slack notifier
func New(webhookURL string) *Notifier {
	enabled := webhookURL != ""

	return &Notifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: enabled,
	}
}

// IsEnabled returns whether Slack notifications are enabled
func (s *Notifier) IsEnabled() bool {
	return s.enabled
}

// UpdateWebhookURL updates the webhook URL for the notifier.
func (s *Notifier) UpdateWebhookURL(webhookURL string) {
	s.webhookURL = webhookURL
	s.enabled = webhookURL != ""
}

// SendMessage sends a simple text message to Slack
func (s *Notifier) SendMessage(ctx context.Context, message string) error {
	if !s.enabled {
		return nil
	}

	payload := Message{
		Text: message,
	}

	return s.sendPayload(ctx, payload)
}

// SendAlert sends a formatted alert to Slack
func (s *Notifier) SendAlert(ctx context.Context, severity, title, message string) error {
	if !s.enabled {
		return nil
	}

	// Map severity to color
	color := severityToColor(severity)

	payload := Message{
		Attachments: []Attachment{
			{
				Color:  color,
				Title:  title,
				Text:   message,
				Footer: "Matter Data Logger",
				Ts:     time.Now().Unix(),
			},
		},
	}

	return s.sendPayload(ctx, payload)
}

// sendPayload sends a payload to the Slack webhook
func (s *Notifier) sendPayload(ctx context.Context, payload Message) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// severityToColor maps severity levels to Slack colors
func severityToColor(severity string) string {
	switch severity {
	case "danger", "error":
		return "danger" // Red
	case "warning", "warn":
		return "warning" // Yellow
	case "good", "success":
		return "good" // Green
	default:
		return "#808080" // Gray
	}
}
