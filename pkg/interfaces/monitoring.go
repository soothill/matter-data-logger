// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package interfaces

import (
	"context"
)

// PowerMonitor defines the interface for device power monitoring.
// Implementations should manage concurrent monitoring of multiple devices.
type PowerMonitor interface {
	// Start begins monitoring the given devices
	Start(ctx context.Context, devices []*Device)

	// StartMonitoringDevice starts monitoring a single device
	// Returns true if monitoring started, false if already monitored
	StartMonitoringDevice(ctx context.Context, device *Device) bool

	// StopMonitoringDevice stops monitoring a specific device
	StopMonitoringDevice(deviceID string)

	// IsMonitoring checks if a device is currently being monitored
	IsMonitoring(deviceID string) bool

	// GetMonitoredDeviceCount returns the number of devices being monitored
	GetMonitoredDeviceCount() int

	// Readings returns the channel for receiving power readings
	Readings() <-chan *PowerReading

	// Stop stops all device monitoring and closes the readings channel
	Stop()
}
