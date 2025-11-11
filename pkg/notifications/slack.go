// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package notifications provides alerting capabilities via various channels.
//
// This package implements notification delivery for critical system events such as
// InfluxDB connectivity issues, cache warnings, and device discovery failures.
// Notifications help operators respond to issues before they impact data collection.
//
// # Notification Channels
//
// Currently supported:
//   - Slack: Webhook-based notifications with formatted attachments
//
// Future channels could include:
//   - Email (SMTP)
//   - PagerDuty
//   - Prometheus Alertmanager
//   - Generic webhooks
//
// # Slack Integration
//
// Slack notifications use Incoming Webhooks for message delivery. The webhook URL
// is configured via SLACK_WEBHOOK_URL environment variable or YAML config.
//
// Message Features:
//   - Color-coded severity levels (red/yellow/green)
//   - Formatted attachments with titles and timestamps
//   - Context-aware timeout handling (10 second HTTP timeout)
//   - Graceful degradation when webhook is not configured
//
// # Alert Severity Levels
//
// Three severity levels with corresponding colors:
//   - danger/error: Red - Critical failures requiring immediate attention
//   - warning/warn: Yellow - Issues that may impact functionality
//   - good/success: Green - Recovery notifications
//
// # Automatic Notifications
//
// The system sends automatic notifications for:
//   - InfluxDB connection failure (on first failure only)
//   - InfluxDB connection recovery (after successful reconnection)
//   - Cache usage warnings (when cache reaches 80% capacity)
//   - Device discovery failures (when mDNS discovery fails)
//
// # Error Handling
//
// Notification failures are logged but do not block the main application:
//   - Failed notifications are logged as errors
//   - HTTP timeouts are enforced (10 seconds)
//   - Context cancellation is respected
//   - Disabled notifiers (empty webhook URL) skip sending silently
//
// # Thread Safety
//
// The SlackNotifier is thread-safe and can be shared across multiple goroutines.
// Each notification uses its own HTTP request with context for cancellation.
//
// # Example Usage
//
// Basic Slack notification:
//
//	notifier := notifications.NewSlackNotifier("https://hooks.slack.com/...")
//
//	if notifier.IsEnabled() {
//	    ctx := context.Background()
//	    notifier.SendMessage(ctx, "Matter data logger started")
//	}
//
// Formatted alert with severity:
//
//	notifier.SendAlert(ctx, "warning", "High CPU Usage",
//	    "CPU usage is above 80% for the last 5 minutes")
//
// Automatic failure notification:
//
//	if err := storage.WriteReading(ctx, reading); err != nil {
//	    notifier.SendInfluxDBFailure(ctx, err)
//	    // Fall back to cache
//	}
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
