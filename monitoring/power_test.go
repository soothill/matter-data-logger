// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
)

// mockScanner is a mock implementation of DeviceScanner for testing
type mockScanner struct {
	devices map[string]*discovery.Device
	mu      sync.RWMutex
}

func newMockScanner() *mockScanner {
	return &mockScanner{
		devices: make(map[string]*discovery.Device),
	}
}

func (m *mockScanner) GetDeviceByID(deviceID string) *discovery.Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.devices[deviceID]
}

func (m *mockScanner) addDevice(device *discovery.Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[device.GetDeviceID()] = device
}

func TestNewPowerMonitor(t *testing.T) {
	pollInterval := 30 * time.Second
	scanner := newMockScanner()
	monitor := NewPowerMonitor(pollInterval, scanner, 100)

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
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
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
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
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
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)

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
		return
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

func TestStartMultipleDevices(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	devices := []*discovery.Device{
		{
			Name: "Device 1",
			TXTRecord: map[string]string{
				"D": "device-1",
			},
		},
		{
			Name: "Device 2",
			TXTRecord: map[string]string{
				"D": "device-2",
			},
		},
		{
			Name: "Device 3",
			TXTRecord: map[string]string{
				"D": "device-3",
			},
		},
	}

	monitor.Start(ctx, devices)

	// Give goroutines time to start
	time.Sleep(100 * time.Millisecond)

	count := monitor.GetMonitoredDeviceCount()
	if count != 3 {
		t.Errorf("GetMonitoredDeviceCount() = %d, want 3", count)
	}

	// Verify all devices are being monitored
	for _, device := range devices {
		if !monitor.IsMonitoring(device.GetDeviceID()) {
			t.Errorf("Device %s should be monitored", device.GetDeviceID())
		}
	}
}

func TestReadingsChannel(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(100 * time.Millisecond, scanner, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	monitor.StartMonitoringDevice(ctx, device)

	// Wait for at least one reading
	select {
	case reading := <-monitor.Readings():
		if reading == nil {
			t.Error("Received nil reading from channel")
		} else if reading.DeviceID != device.GetDeviceID() {
			t.Errorf("Reading DeviceID = %v, want %v", reading.DeviceID, device.GetDeviceID())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for reading from channel")
	}
}

func TestContextCancellation(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	monitor.StartMonitoringDevice(ctx, device)

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	if !monitor.IsMonitoring(device.GetDeviceID()) {
		t.Error("Device should be monitored")
	}

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Device should no longer be monitored after context cancellation
	if monitor.IsMonitoring(device.GetDeviceID()) {
		t.Error("Device should not be monitored after context cancellation")
	}
}

func TestConcurrentMonitoring(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start multiple devices concurrently
	numDevices := 10
	done := make(chan bool, numDevices)

	for i := 0; i < numDevices; i++ {
		go func(id int) {
			device := &discovery.Device{
				Name: "Device",
				TXTRecord: map[string]string{
					"D": string(rune('A' + id)),
				},
			}
			monitor.StartMonitoringDevice(ctx, device)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numDevices; i++ {
		<-done
	}

	// Give time for all monitors to start
	time.Sleep(100 * time.Millisecond)

	count := monitor.GetMonitoredDeviceCount()
	if count != numDevices {
		t.Errorf("GetMonitoredDeviceCount() = %d, want %d", count, numDevices)
	}
}

func TestStopNonExistentDevice(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)

	// Stopping a device that doesn't exist should not panic
	monitor.StopMonitoringDevice("nonexistent-device")

	// Should still be safe to check
	if monitor.IsMonitoring("nonexistent-device") {
		t.Error("Nonexistent device should not be monitored")
	}
}

func TestReadingsChannelFull(_ *testing.T) {
	// Create monitor with very small channel buffer
	monitor := &PowerMonitor{
		pollInterval:     1 * time.Millisecond,
		readings:         make(chan *PowerReading, 1), // Small buffer
		monitoredDevices: make(map[string]context.CancelFunc),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	monitor.StartMonitoringDevice(ctx, device)

	// Don't read from channel - let it fill up
	time.Sleep(100 * time.Millisecond)

	// Monitor should handle full channel gracefully (drops readings)
	// This test ensures no panic occurs
}

func TestMonitorDevice_ZeroPollInterval(_ *testing.T) {
	// Test that zero or very short poll interval doesn't cause issues
	scanner := newMockScanner()
	monitor := NewPowerMonitor(1 * time.Nanosecond, scanner, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	// Should not panic with very short interval
	monitor.StartMonitoringDevice(ctx, device)
	time.Sleep(50 * time.Millisecond)
}

func TestReadPower_ConsistentData(t *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	// Read power multiple times and verify consistency
	for i := 0; i < 5; i++ {
		reading, err := monitor.readPower(device)
		if err != nil {
			t.Errorf("readPower() iteration %d error = %v", i, err)
		}

		if reading == nil {
			t.Fatalf("readPower() iteration %d returned nil", i)
			return
		}

		// Verify timestamp is recent
		if time.Since(reading.Timestamp) > 1*time.Second {
			t.Errorf("Reading timestamp is too old: %v", reading.Timestamp)
		}

		// Verify power calculation is consistent (Power = Voltage * Current)
		expectedPower := reading.Voltage * reading.Current
		tolerance := 0.1 // Allow small floating point error
		if reading.Power < expectedPower-tolerance || reading.Power > expectedPower+tolerance {
			t.Errorf("Power calculation inconsistent: Power=%f, Voltage=%f, Current=%f, Expected=%f",
				reading.Power, reading.Voltage, reading.Current, expectedPower)
		}
	}
}

func TestIsMonitoring_ThreadSafety(_ *testing.T) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	monitor.StartMonitoringDevice(ctx, device)

	// Check IsMonitoring from multiple goroutines concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = monitor.IsMonitoring(device.GetDeviceID())
				_ = monitor.GetMonitoredDeviceCount()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests

func BenchmarkReadPower(b *testing.B) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = monitor.readPower(device)
	}
}

func BenchmarkReadPower_Parallel(b *testing.B) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)

	device := &discovery.Device{
		Name: "Test Device",
		TXTRecord: map[string]string{
			"D": "test-device",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = monitor.readPower(device)
		}
	})
}

func BenchmarkIsMonitoring(b *testing.B) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add some devices to monitor
	for i := 0; i < 10; i++ {
		device := &discovery.Device{
			Name: "Device",
			TXTRecord: map[string]string{
				"D": string(rune('A' + i)),
			},
		}
		monitor.StartMonitoringDevice(ctx, device)
	}

	deviceID := "A"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.IsMonitoring(deviceID)
	}
}

func BenchmarkGetMonitoredDeviceCount(b *testing.B) {
	scanner := newMockScanner()
	monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add some devices to monitor
	for i := 0; i < 100; i++ {
		device := &discovery.Device{
			Name: "Device",
			TXTRecord: map[string]string{
				"D": string(rune('A' + i)),
			},
		}
		monitor.StartMonitoringDevice(ctx, device)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetMonitoredDeviceCount()
	}
}

func BenchmarkStartMonitoringDevice(b *testing.B) {
	scanner := newMockScanner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		monitor := NewPowerMonitor(30 * time.Second, scanner, 100)
		device := &discovery.Device{
			Name: "Device",
			TXTRecord: map[string]string{
				"D": string(rune('A' + (i % 26))),
			},
		}
		b.StartTimer()

		monitor.StartMonitoringDevice(ctx, device)
	}
}

func BenchmarkPowerReadingGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &PowerReading{
			DeviceID:   "device-1",
			DeviceName: "Test Device",
			Timestamp:  time.Now(),
			Power:      100.0,
			Voltage:    120.0,
			Current:    0.833,
			Energy:     1.0,
		}
	}
}
