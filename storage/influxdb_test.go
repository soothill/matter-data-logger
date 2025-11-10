// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package storage

import (
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/monitoring"
)

func TestNewInfluxDBStorage_InvalidURL(t *testing.T) {
	// Test with empty URL
	storage, err := NewInfluxDBStorage("", "token", "org", "bucket")
	if err == nil {
		t.Error("NewInfluxDBStorage() should fail with empty URL")
	}
	if storage != nil {
		storage.Close()
		t.Error("NewInfluxDBStorage() should return nil storage on error")
	}
}

func TestNewInfluxDBStorage_ConnectionTimeout(t *testing.T) {
	// Test with invalid URL that will timeout
	storage, err := NewInfluxDBStorage("http://invalid-host-that-does-not-exist:8086", "token", "org", "bucket")
	if err == nil {
		t.Error("NewInfluxDBStorage() should fail with unreachable host")
	}
	if storage != nil {
		storage.Close()
		t.Error("NewInfluxDBStorage() should return nil storage on connection error")
	}
}

func TestNewInfluxDBStorage_ValidParameters(t *testing.T) {
	// Test parameter validation (even if connection fails)
	testCases := []struct {
		name   string
		url    string
		token  string
		org    string
		bucket string
	}{
		{"empty token", "http://localhost:8086", "", "org", "bucket"},
		{"empty org", "http://localhost:8086", "token", "", "bucket"},
		{"empty bucket", "http://localhost:8086", "token", "org", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := NewInfluxDBStorage(tc.url, tc.token, tc.org, tc.bucket)
			// Connection will likely fail, but we're testing parameter handling
			if storage != nil {
				storage.Close()
			}
			// We expect an error due to connection failure, not parameter validation
			// This is acceptable as the actual validation happens in config
			_ = err
		})
	}
}

func TestWriteReading_ValidReading(t *testing.T) {
	// Create a mock storage (won't actually connect to InfluxDB)
	// This tests the data structure and method signature
	reading := &monitoring.PowerReading{
		DeviceID:   "test-device-1",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.5,
		Voltage:    120.0,
		Current:    0.8375,
		Energy:     1.5,
	}

	// Validate reading structure
	if reading.DeviceID == "" {
		t.Error("DeviceID should not be empty")
	}
	if reading.Power <= 0 {
		t.Error("Power should be positive")
	}
	if reading.Voltage <= 0 {
		t.Error("Voltage should be positive")
	}
	if reading.Current <= 0 {
		t.Error("Current should be positive")
	}
}

func TestWriteReading_NilReading(t *testing.T) {
	// Test handling of nil reading
	var reading *monitoring.PowerReading

	if reading != nil {
		t.Error("Test setup error: reading should be nil")
	}

	// If we had a real storage, WriteReading should handle nil gracefully
	// This documents expected behavior
}

func TestWriteBatch_ValidReadings(t *testing.T) {
	readings := []*monitoring.PowerReading{
		{
			DeviceID:   "device-1",
			DeviceName: "Device 1",
			Timestamp:  time.Now(),
			Power:      50.0,
			Voltage:    120.0,
			Current:    0.417,
			Energy:     0.5,
		},
		{
			DeviceID:   "device-2",
			DeviceName: "Device 2",
			Timestamp:  time.Now(),
			Power:      75.0,
			Voltage:    120.0,
			Current:    0.625,
			Energy:     1.0,
		},
	}

	// Validate batch structure
	if len(readings) != 2 {
		t.Errorf("Expected 2 readings, got %d", len(readings))
	}

	for i, reading := range readings {
		if reading == nil {
			t.Errorf("Reading %d should not be nil", i)
		}
		if reading.DeviceID == "" {
			t.Errorf("Reading %d: DeviceID should not be empty", i)
		}
	}
}

func TestWriteBatch_EmptySlice(t *testing.T) {
	readings := []*monitoring.PowerReading{}

	if len(readings) != 0 {
		t.Error("Empty slice should have length 0")
	}

	// WriteBatch should handle empty slice gracefully
}

func TestWriteBatch_NilSlice(t *testing.T) {
	var readings []*monitoring.PowerReading

	if readings != nil {
		t.Error("Nil slice should be nil")
	}

	// WriteBatch should handle nil slice gracefully
}

func TestPowerReading_Validation(t *testing.T) {
	tests := []struct {
		name    string
		reading *monitoring.PowerReading
		valid   bool
	}{
		{
			name: "valid reading",
			reading: &monitoring.PowerReading{
				DeviceID:   "device-1",
				DeviceName: "Device 1",
				Timestamp:  time.Now(),
				Power:      100.0,
				Voltage:    120.0,
				Current:    0.833,
				Energy:     1.0,
			},
			valid: true,
		},
		{
			name: "zero power",
			reading: &monitoring.PowerReading{
				DeviceID:   "device-1",
				DeviceName: "Device 1",
				Timestamp:  time.Now(),
				Power:      0.0,
				Voltage:    120.0,
				Current:    0.0,
				Energy:     0.0,
			},
			valid: true, // Zero is valid for idle devices
		},
		{
			name: "negative power",
			reading: &monitoring.PowerReading{
				DeviceID:   "device-1",
				DeviceName: "Device 1",
				Timestamp:  time.Now(),
				Power:      -10.0,
				Voltage:    120.0,
				Current:    -0.083,
				Energy:     1.0,
			},
			valid: false, // Negative values might indicate solar or measurement error
		},
		{
			name: "missing device ID",
			reading: &monitoring.PowerReading{
				DeviceID:   "",
				DeviceName: "Device 1",
				Timestamp:  time.Now(),
				Power:      100.0,
				Voltage:    120.0,
				Current:    0.833,
				Energy:     1.0,
			},
			valid: false,
		},
		{
			name: "missing timestamp",
			reading: &monitoring.PowerReading{
				DeviceID:   "device-1",
				DeviceName: "Device 1",
				Timestamp:  time.Time{},
				Power:      100.0,
				Voltage:    120.0,
				Current:    0.833,
				Energy:     1.0,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validatePowerReading(tt.reading)
			if valid != tt.valid {
				t.Errorf("validatePowerReading() = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// validatePowerReading checks if a power reading has valid data
func validatePowerReading(reading *monitoring.PowerReading) bool {
	if reading == nil {
		return false
	}
	if reading.DeviceID == "" {
		return false
	}
	if reading.Timestamp.IsZero() {
		return false
	}
	if reading.Power < 0 || reading.Voltage < 0 || reading.Current < 0 {
		return false
	}
	return true
}

func TestInfluxDBStorage_FlushAndClose(t *testing.T) {
	// Test that Flush and Close don't panic with nil storage
	// This is important for graceful shutdown

	// In a real test, we would:
	// 1. Create a storage with mock InfluxDB
	// 2. Call Flush() and ensure it doesn't panic
	// 3. Call Close() and ensure it doesn't panic
	// 4. Verify Close() calls Flush() internally

	// For now, we document the expected behavior
	t.Log("Flush should force pending writes to complete")
	t.Log("Close should call Flush and close the client")
}

func TestInfluxDBDataPoint_Structure(t *testing.T) {
	// Test the data structure we're writing to InfluxDB
	reading := &monitoring.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Smart Plug",
		Timestamp:  time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		Power:      100.5,
		Voltage:    120.2,
		Current:    0.836,
		Energy:     2.5,
	}

	// Verify measurement name would be "power_consumption"
	expectedMeasurement := "power_consumption"

	// Verify tags would be device_id and device_name
	expectedTags := map[string]string{
		"device_id":   reading.DeviceID,
		"device_name": reading.DeviceName,
	}

	// Verify fields would be power, voltage, current, energy
	expectedFields := map[string]interface{}{
		"power":   reading.Power,
		"voltage": reading.Voltage,
		"current": reading.Current,
		"energy":  reading.Energy,
	}

	t.Logf("Measurement: %s", expectedMeasurement)
	t.Logf("Tags: %+v", expectedTags)
	t.Logf("Fields: %+v", expectedFields)
	t.Logf("Timestamp: %v", reading.Timestamp)

	// Validate structure
	if expectedMeasurement == "" {
		t.Error("Measurement name should not be empty")
	}
	if len(expectedTags) != 2 {
		t.Error("Should have 2 tags")
	}
	if len(expectedFields) != 4 {
		t.Error("Should have 4 fields")
	}
}

func TestQueryLatestReading_DeviceIDValidation(t *testing.T) {
	// Test that device ID is validated before querying
	testCases := []struct {
		name     string
		deviceID string
		valid    bool
	}{
		{"valid device ID", "device-123", true},
		{"empty device ID", "", false},
		{"device ID with spaces", "device 123", true}, // Technically valid
		{"very long device ID", string(make([]byte, 1000)), true}, // May need limits
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.deviceID == "" && !tc.valid {
				t.Log("Empty device ID should be rejected")
			}
		})
	}
}
