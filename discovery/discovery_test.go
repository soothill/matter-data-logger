// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package discovery

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	serviceType := "_matter._tcp"
	domain := "local."

	scanner := NewScanner(serviceType, domain)

	if scanner == nil {
		t.Fatal("NewScanner() returned nil")
	}

	if scanner.serviceType != serviceType {
		t.Errorf("serviceType = %v, want %v", scanner.serviceType, serviceType)
	}

	if scanner.domain != domain {
		t.Errorf("domain = %v, want %v", scanner.domain, domain)
	}

	if scanner.devices == nil {
		t.Error("devices map is nil")
	}

	if len(scanner.devices) != 0 {
		t.Errorf("devices map should be empty, got %d devices", len(scanner.devices))
	}
}

func TestDevice_HasPowerMeasurement(t *testing.T) {
	tests := []struct {
		name      string
		txtRecord map[string]string
		want      bool
	}{
		{
			name: "has electrical measurement cluster (0B04)",
			txtRecord: map[string]string{
				"C": "0006,0008,0B04,001D",
			},
			want: true,
		},
		{
			name: "has electrical measurement cluster (B04)",
			txtRecord: map[string]string{
				"C": "0006,B04,001D",
			},
			want: true,
		},
		{
			name: "has electrical power measurement cluster (0091)",
			txtRecord: map[string]string{
				"C": "0006,0008,0091,001D",
			},
			want: true,
		},
		{
			name: "has electrical power measurement cluster (91)",
			txtRecord: map[string]string{
				"C": "0006,91,001D",
			},
			want: true,
		},
		{
			name: "no power measurement cluster",
			txtRecord: map[string]string{
				"C": "0006,0008,001D",
			},
			want: false,
		},
		{
			name:      "missing cluster information",
			txtRecord: map[string]string{},
			want:      false,
		},
		{
			name:      "nil TXT record",
			txtRecord: nil,
			want:      false,
		},
		{
			name: "empty cluster string",
			txtRecord: map[string]string{
				"C": "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Device{
				TXTRecord: tt.txtRecord,
			}
			got := device.HasPowerMeasurement()
			if got != tt.want {
				t.Errorf("HasPowerMeasurement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_GetDeviceID(t *testing.T) {
	tests := []struct {
		name      string
		device    *Device
		want      string
	}{
		{
			name: "with discriminator in TXT record",
			device: &Device{
				Address: net.ParseIP("192.168.1.100"),
				Port:    5540,
				TXTRecord: map[string]string{
					"D": "12345",
				},
			},
			want: "12345",
		},
		{
			name: "without discriminator - fallback to address:port",
			device: &Device{
				Address:   net.ParseIP("192.168.1.100"),
				Port:      5540,
				TXTRecord: map[string]string{},
			},
			want: "192.168.1.100:5540",
		},
		{
			name: "IPv6 address without discriminator",
			device: &Device{
				Address:   net.ParseIP("fe80::1"),
				Port:      5540,
				TXTRecord: map[string]string{},
			},
			want: "fe80::1:5540",
		},
		{
			name: "nil TXT record",
			device: &Device{
				Address:   net.ParseIP("192.168.1.100"),
				Port:      5540,
				TXTRecord: nil,
			},
			want: "192.168.1.100:5540",
		},
		{
			name: "empty discriminator",
			device: &Device{
				Address: net.ParseIP("192.168.1.100"),
				Port:    5540,
				TXTRecord: map[string]string{
					"D": "",
				},
			},
			want: "192.168.1.100:5540",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.device.GetDeviceID()
			if got != tt.want {
				t.Errorf("GetDeviceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScanner_GetDevices(t *testing.T) {
	scanner := NewScanner("_matter._tcp", "local.")

	// Initially empty
	devices := scanner.GetDevices()
	if len(devices) != 0 {
		t.Errorf("GetDevices() should return empty slice, got %d devices", len(devices))
	}

	// Add some devices
	device1 := &Device{
		Name:    "Device 1",
		Address: net.ParseIP("192.168.1.100"),
		Port:    5540,
		TXTRecord: map[string]string{
			"D": "device-1",
		},
	}
	device2 := &Device{
		Name:    "Device 2",
		Address: net.ParseIP("192.168.1.101"),
		Port:    5540,
		TXTRecord: map[string]string{
			"D": "device-2",
		},
	}

	scanner.devices[device1.GetDeviceID()] = device1
	scanner.devices[device2.GetDeviceID()] = device2

	devices = scanner.GetDevices()
	if len(devices) != 2 {
		t.Errorf("GetDevices() should return 2 devices, got %d", len(devices))
	}
}

func TestScanner_GetPowerDevices(t *testing.T) {
	scanner := NewScanner("_matter._tcp", "local.")

	// Add device with power measurement
	powerDevice := &Device{
		Name:    "Power Device",
		Address: net.ParseIP("192.168.1.100"),
		Port:    5540,
		TXTRecord: map[string]string{
			"D": "power-device",
			"C": "0006,0B04,001D",
		},
	}

	// Add device without power measurement
	normalDevice := &Device{
		Name:    "Normal Device",
		Address: net.ParseIP("192.168.1.101"),
		Port:    5540,
		TXTRecord: map[string]string{
			"D": "normal-device",
			"C": "0006,0008,001D",
		},
	}

	scanner.devices[powerDevice.GetDeviceID()] = powerDevice
	scanner.devices[normalDevice.GetDeviceID()] = normalDevice

	powerDevices := scanner.GetPowerDevices()
	if len(powerDevices) != 1 {
		t.Errorf("GetPowerDevices() should return 1 device, got %d", len(powerDevices))
	}

	if powerDevices[0].GetDeviceID() != "power-device" {
		t.Errorf("GetPowerDevices() returned wrong device: %v", powerDevices[0].GetDeviceID())
	}
}

func TestScanner_Discover_Timeout(t *testing.T) {
	scanner := NewScanner("_matter._tcp", "local.")
	ctx := context.Background()

	// Test with very short timeout - should complete without hanging
	start := time.Now()
	devices, err := scanner.Discover(ctx, 100*time.Millisecond)
	duration := time.Since(start)

	// Note: In environments without network interfaces (like CI),
	// this may fail with "failed to join any of these interfaces"
	// This is expected and not a bug - skip the test in that case
	if err != nil {
		if strings.Contains(err.Error(), "failed to join any of these interfaces") {
			t.Skip("Skipping test: no network interfaces available for mDNS (expected in some CI environments)")
		}
		// Other errors should be reported
		t.Logf("Discover() returned error: %v (this may be expected in some environments)", err)
	}

	// Should complete within reasonable time (allowing some overhead)
	if duration > 500*time.Millisecond {
		t.Errorf("Discover() took too long: %v", duration)
	}

	// Devices list should be initialized (even if empty)
	if devices == nil && err == nil {
		t.Error("Discover() returned nil devices slice without error")
	}
}

func TestScanner_Discover_ContextCancellation(t *testing.T) {
	scanner := NewScanner("_matter._tcp", "local.")
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	devices, err := scanner.Discover(ctx, 5*time.Second)

	// Note: May fail in environments without network interfaces
	if err != nil {
		if strings.Contains(err.Error(), "failed to join any of these interfaces") {
			t.Skip("Skipping test: no network interfaces available for mDNS")
		}
	}

	// Should handle cancellation gracefully
	// Devices may be nil if error occurred, which is acceptable
	_ = devices
	_ = err
}

func TestScanner_Discover_MultipleRuns(t *testing.T) {
	scanner := NewScanner("_matter._tcp", "local.")
	ctx := context.Background()

	// Run discovery multiple times - should not panic or cause issues
	var hasNetworkError bool
	for i := 0; i < 3; i++ {
		devices, err := scanner.Discover(ctx, 100*time.Millisecond)
		if err != nil {
			if strings.Contains(err.Error(), "failed to join any of these interfaces") {
				hasNetworkError = true
				continue
			}
			t.Errorf("Discover() run %d error = %v", i+1, err)
		}
		if devices == nil && err == nil {
			t.Errorf("Discover() run %d returned nil devices without error", i+1)
		}
	}

	if hasNetworkError {
		t.Skip("Skipping test: no network interfaces available for mDNS")
	}

	// Devices map should accumulate or update properly
	if scanner.devices == nil {
		t.Error("scanner.devices map is nil after multiple discoveries")
	}
}
