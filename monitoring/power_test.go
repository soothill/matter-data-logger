// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
)

func TestNewPowerMonitor(t *testing.T) {
	pollInterval := 30 * time.Second
	monitor := NewPowerMonitor(pollInterval)

	if monitor == nil {
		t.Fatal("NewPowerMonitor() returned nil")
	}

	if monitor.pollInterval != pollInterval {
		t.Errorf("pollInterval = %v, want %v", monitor.pollInterval, pollInterval)
	}

	if monitor.readings == nil {
		t.Error("readings channel is nil")
	}

	if monitor.monitoredDevices == nil {
		t.Error("monitoredDevices map is nil")
	}
}

func TestStartMonitoringDevice(t *testing.T) {
	monitor := NewPowerMonitor(30 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device-1",
		},
	}

	// First start should succeed
	started := monitor.StartMonitoringDevice(ctx, device)
	if !started {
		t.Error("StartMonitoringDevice() should return true for new device")
	}

	// Second start should fail (duplicate)
	started = monitor.StartMonitoringDevice(ctx, device)
	if started {
		t.Error("StartMonitoringDevice() should return false for already monitored device")
	}

	// Check if device is being monitored
	if !monitor.IsMonitoring(device.GetDeviceID()) {
		t.Error("Device should be monitored")
	}

	// Check monitored device count
	count := monitor.GetMonitoredDeviceCount()
	if count != 1 {
		t.Errorf("GetMonitoredDeviceCount() = %d, want 1", count)
	}
}

func TestStopMonitoringDevice(t *testing.T) {
	monitor := NewPowerMonitor(30 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device-2",
		},
	}

	// Start monitoring
	monitor.StartMonitoringDevice(ctx, device)

	// Stop monitoring
	monitor.StopMonitoringDevice(device.GetDeviceID())

	// Check if device is no longer monitored
	if monitor.IsMonitoring(device.GetDeviceID()) {
		t.Error("Device should not be monitored after stop")
	}

	// Check monitored device count
	count := monitor.GetMonitoredDeviceCount()
	if count != 0 {
		t.Errorf("GetMonitoredDeviceCount() = %d, want 0", count)
	}
}

func TestReadPower(t *testing.T) {
	monitor := NewPowerMonitor(30 * time.Second)

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device-3",
		},
	}

	reading, err := monitor.readPower(device)
	if err != nil {
		t.Errorf("readPower() error = %v", err)
	}

	if reading == nil {
		t.Fatal("readPower() returned nil reading")
	}

	if reading.DeviceID != device.GetDeviceID() {
		t.Errorf("reading.DeviceID = %v, want %v", reading.DeviceID, device.GetDeviceID())
	}

	if reading.DeviceName != device.Name {
		t.Errorf("reading.DeviceName = %v, want %v", reading.DeviceName, device.Name)
	}

	if reading.Power <= 0 {
		t.Error("reading.Power should be positive")
	}

	if reading.Voltage <= 0 {
		t.Error("reading.Voltage should be positive")
	}

	if reading.Current <= 0 {
		t.Error("reading.Current should be positive")
	}
}
