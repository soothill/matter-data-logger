// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package slacknotifier

import (
	"context"
	"fmt"
)

// Adapter adapts the slacknotifier.Notifier to the
// storage.Notifier interface.
type Adapter struct {
	notifier *Notifier
}

// NewAdapter creates a new adapter.
func NewAdapter(notifier *Notifier) *Adapter {
	return &Adapter{notifier: notifier}
}

// SendInfluxDBFailure sends an alert when InfluxDB connection fails
func (a *Adapter) SendInfluxDBFailure(ctx context.Context, err error) error {
	return a.notifier.SendAlert(ctx, "danger", "⚠️ InfluxDB Connection Failure",
		fmt.Sprintf("Failed to connect to InfluxDB: %v\nData will be cached locally until connection is restored.", err))
}

// SendInfluxDBRecovery sends an alert when InfluxDB connection recovers
func (a *Adapter) SendInfluxDBRecovery(ctx context.Context) error {
	return a.notifier.SendAlert(ctx, "good", "✅ InfluxDB Connection Restored",
		"Connection to InfluxDB has been restored. Cached data will be replayed.")
}

// SendCacheWarning sends an alert when cache usage is high
func (a *Adapter) SendCacheWarning(ctx context.Context, cacheSize int64, maxSize int64) error {
	percentage := float64(cacheSize) / float64(maxSize) * 100
	return a.notifier.SendAlert(ctx, "warning", "⚠️ Local Cache Usage High",
		fmt.Sprintf("Cache size: %d bytes (%.1f%% of max %d bytes)\nInfluxDB may be unavailable for an extended period.",
			cacheSize, percentage, maxSize))
}

// IsEnabled returns whether Slack notifications are enabled
func (a *Adapter) IsEnabled() bool {
	return a.notifier.IsEnabled()
}

// SendAlert sends a notification with the given level, title, and message.
func (a *Adapter) SendAlert(ctx context.Context, level, title, message string) error {
	return a.notifier.SendAlert(ctx, level, title, message)
}
