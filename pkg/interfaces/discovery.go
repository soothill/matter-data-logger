// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package interfaces

import (
	"context"
	"net"
	"time"
)

// Device represents a discovered Matter device.
// This is redeclared here to avoid circular dependencies.
type Device struct {
	Name      string
	Address   net.IP
	Port      int
	TXTRecord map[string]string
	Hostname  string
}

// DeviceScanner defines the interface for Matter device discovery.
// Implementations should support mDNS/DNS-SD discovery protocols.
type DeviceScanner interface {
	// Discover performs a device discovery scan with the given timeout
	Discover(ctx context.Context, timeout time.Duration) ([]*Device, error)

	// GetDevices returns all discovered devices
	GetDevices() []*Device

	// GetPowerDevices returns only devices with power measurement capability
	GetPowerDevices() []*Device

	// GetDeviceByID returns a device by its ID, or nil if not found
	GetDeviceByID(deviceID string) *Device
}

// DeviceCapabilities defines methods for checking device capabilities.
type DeviceCapabilities interface {
	// HasPowerMeasurement checks if the device supports power measurement
	HasPowerMeasurement() bool

	// GetDeviceID returns a unique identifier for the device
	GetDeviceID() string
}
