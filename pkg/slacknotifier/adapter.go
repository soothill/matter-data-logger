// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package notifications

import (
	"context"
	"fmt"
	"github.com/soothill/matter-data-logger/pkg/slacknotifier"
)

// SlackNotifierAdapter adapts the slacknotifier.Notifier to the
// storage.Notifier interface.
type SlackNotifierAdapter struct {
	notifier *slacknotifier.Notifier
}

// NewSlackNotifierAdapter creates a new adapter.
func NewSlackNotifierAdapter(notifier *slacknotifier.Notifier) *SlackNotifierAdapter {
	return &SlackNotifierAdapter{notifier: notifier}
}

// SendInfluxDBFailure sends an alert when InfluxDB connection fails
func (a *SlackNotifierAdapter) SendInfluxDBFailure(ctx context.Context, err error) error {
	return a.notifier.SendAlert(ctx, "danger", "⚠️ InfluxDB Connection Failure",
		fmt.Sprintf("Failed to connect to InfluxDB: %v\nData will be cached locally until connection is restored.", err))
}

// SendInfluxDBRecovery sends an alert when InfluxDB connection recovers
func (a *SlackNotifierAdapter) SendInfluxDBRecovery(ctx context.Context) error {
	return a.notifier.SendAlert(ctx, "good", "✅ InfluxDB Connection Restored",
		"Connection to InfluxDB has been restored. Cached data will be replayed.")
}

// SendCacheWarning sends an alert when cache usage is high
func (a *SlackNotifierAdapter) SendCacheWarning(ctx context.Context, cacheSize int64, maxSize int64) error {
	percentage := float64(cacheSize) / float64(maxSize) * 100
	return a.notifier.SendAlert(ctx, "warning", "⚠️ Local Cache Usage High",
		fmt.Sprintf("Cache size: %d bytes (%.1f%% of max %d bytes)\nInfluxDB may be unavailable for an extended period.",
			cacheSize, percentage, maxSize))
}

// IsEnabled returns whether Slack notifications are enabled
func (a *SlackNotifierAdapter) IsEnabled() bool {
	return a.notifier.IsEnabled()
}
