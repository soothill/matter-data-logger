// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package notifications provides alerting capabilities via various channels.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/soothill/matter-data-logger/pkg/logger"
)

// SlackNotifier sends notifications to Slack via webhook
type SlackNotifier struct {
	webhookURL string
	client     *http.Client
	enabled    bool
}

// SlackMessage represents a Slack webhook message payload
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Blocks      []SlackBlock `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
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

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	enabled := webhookURL != ""

	return &SlackNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: enabled,
	}
}

// IsEnabled returns whether Slack notifications are enabled
func (s *SlackNotifier) IsEnabled() bool {
	return s.enabled
}

// SendMessage sends a simple text message to Slack
func (s *SlackNotifier) SendMessage(ctx context.Context, message string) error {
	if !s.enabled {
		logger.Debug().Msg("Slack notifications disabled, skipping message")
		return nil
	}

	payload := SlackMessage{
		Text: message,
	}

	return s.sendPayload(ctx, payload)
}

// SendAlert sends a formatted alert to Slack
func (s *SlackNotifier) SendAlert(ctx context.Context, severity, title, message string) error {
	if !s.enabled {
		logger.Debug().Msg("Slack notifications disabled, skipping alert")
		return nil
	}

	// Map severity to color
	color := s.severityToColor(severity)

	payload := SlackMessage{
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

// SendInfluxDBFailure sends an alert when InfluxDB connection fails
func (s *SlackNotifier) SendInfluxDBFailure(ctx context.Context, err error) error {
	return s.SendAlert(ctx, "danger", "⚠️ InfluxDB Connection Failure",
		fmt.Sprintf("Failed to connect to InfluxDB: %v\nData will be cached locally until connection is restored.", err))
}

// SendInfluxDBRecovery sends an alert when InfluxDB connection recovers
func (s *SlackNotifier) SendInfluxDBRecovery(ctx context.Context) error {
	return s.SendAlert(ctx, "good", "✅ InfluxDB Connection Restored",
		"Connection to InfluxDB has been restored. Cached data will be replayed.")
}

// SendCacheWarning sends an alert when cache usage is high
func (s *SlackNotifier) SendCacheWarning(ctx context.Context, cacheSize int64, maxSize int64) error {
	percentage := float64(cacheSize) / float64(maxSize) * 100
	return s.SendAlert(ctx, "warning", "⚠️ Local Cache Usage High",
		fmt.Sprintf("Cache size: %d bytes (%.1f%% of max %d bytes)\nInfluxDB may be unavailable for an extended period.",
			cacheSize, percentage, maxSize))
}

// SendDiscoveryFailure sends an alert when device discovery fails
func (s *SlackNotifier) SendDiscoveryFailure(ctx context.Context, err error) error {
	return s.SendAlert(ctx, "warning", "⚠️ Device Discovery Failure",
		fmt.Sprintf("Failed to discover Matter devices: %v", err))
}

// sendPayload sends a payload to the Slack webhook
func (s *SlackNotifier) sendPayload(ctx context.Context, payload SlackMessage) error {
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

	if len(payload.Attachments) > 0 {
		logger.Debug().Str("title", payload.Attachments[0].Title).Msg("Slack notification sent successfully")
	} else {
		logger.Debug().Str("text", payload.Text).Msg("Slack notification sent successfully")
	}
	return nil
}

// severityToColor maps severity levels to Slack colors
func (s *SlackNotifier) severityToColor(severity string) string {
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
