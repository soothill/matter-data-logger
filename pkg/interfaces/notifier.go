// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package interfaces

import (
	"context"
)

// Notifier defines the interface for sending notifications.
type Notifier interface {
	// SendAlert sends a notification with the given level, title, and message.
	SendAlert(ctx context.Context, level, title, message string) error
	// IsEnabled returns true if the notifier is configured and enabled.
	IsEnabled() bool
}
