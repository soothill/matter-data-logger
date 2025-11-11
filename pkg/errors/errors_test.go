// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDiscoveryError(t *testing.T) {
	baseErr := fmt.Errorf("network unreachable")
	err := NewDiscoveryError("mDNS scan", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "discovery") || !strings.Contains(errMsg, "mDNS scan") {
		t.Errorf("Error() = %q, want message containing 'discovery' and 'mDNS scan'", errMsg)
	}

	// Test Unwrap()
	if !errors.Is(err, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}

	// Test IsDiscoveryError()
	if !IsDiscoveryError(err) {
		t.Error("IsDiscoveryError() should return true for DiscoveryError")
	}

	// Test errors.As()
	var de *DiscoveryError
	if !errors.As(err, &de) {
		t.Error("errors.As() should extract DiscoveryError")
	}
	if de.Op != "mDNS scan" {
		t.Errorf("DiscoveryError.Op = %q, want %q", de.Op, "mDNS scan")
	}
}

func TestStorageError(t *testing.T) {
	baseErr := fmt.Errorf("connection timeout")
	err := NewStorageError("write", "device-123", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "storage") || !strings.Contains(errMsg, "write") || !strings.Contains(errMsg, "device-123") {
		t.Errorf("Error() = %q, want message containing 'storage', 'write', and 'device-123'", errMsg)
	}

	// Test Unwrap()
	if !errors.Is(err, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}

	// Test IsStorageError()
	if !IsStorageError(err) {
		t.Error("IsStorageError() should return true for StorageError")
	}

	// Test errors.As()
	var se *StorageError
	if !errors.As(err, &se) {
		t.Error("errors.As() should extract StorageError")
	}
	if se.Op != "write" {
		t.Errorf("StorageError.Op = %q, want %q", se.Op, "write")
	}
	if se.DeviceID != "device-123" {
		t.Errorf("StorageError.DeviceID = %q, want %q", se.DeviceID, "device-123")
	}
}

func TestConfigError(t *testing.T) {
	baseErr := fmt.Errorf("invalid format")
	err := NewConfigError("influxdb.url", "invalid://url", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "config") || !strings.Contains(errMsg, "influxdb.url") {
		t.Errorf("Error() = %q, want message containing 'config' and 'influxdb.url'", errMsg)
	}

	// Test IsConfigError()
	if !IsConfigError(err) {
		t.Error("IsConfigError() should return true for ConfigError")
	}

	// Test errors.As()
	var ce *ConfigError
	if !errors.As(err, &ce) {
		t.Error("errors.As() should extract ConfigError")
	}
	if ce.Field != "influxdb.url" {
		t.Errorf("ConfigError.Field = %q, want %q", ce.Field, "influxdb.url")
	}
}

func TestMonitoringError(t *testing.T) {
	baseErr := fmt.Errorf("device not responding")
	err := NewMonitoringError("read power", "device-456", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "monitoring") || !strings.Contains(errMsg, "read power") || !strings.Contains(errMsg, "device-456") {
		t.Errorf("Error() = %q, want message containing 'monitoring', 'read power', and 'device-456'", errMsg)
	}

	// Test IsMonitoringError()
	if !IsMonitoringError(err) {
		t.Error("IsMonitoringError() should return true for MonitoringError")
	}

	// Test errors.As()
	var me *MonitoringError
	if !errors.As(err, &me) {
		t.Error("errors.As() should extract MonitoringError")
	}
	if me.DeviceID != "device-456" {
		t.Errorf("MonitoringError.DeviceID = %q, want %q", me.DeviceID, "device-456")
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("power", -10.5, "must be non-negative")

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "validation") || !strings.Contains(errMsg, "power") || !strings.Contains(errMsg, "non-negative") {
		t.Errorf("Error() = %q, want message containing 'validation', 'power', and 'non-negative'", errMsg)
	}

	// Test IsValidationError()
	if !IsValidationError(err) {
		t.Error("IsValidationError() should return true for ValidationError")
	}

	// Test errors.As()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Error("errors.As() should extract ValidationError")
	}
	if ve.Field != "power" {
		t.Errorf("ValidationError.Field = %q, want %q", ve.Field, "power")
	}
	if ve.Reason != "must be non-negative" {
		t.Errorf("ValidationError.Reason = %q, want %q", ve.Reason, "must be non-negative")
	}
}

func TestNetworkError(t *testing.T) {
	baseErr := fmt.Errorf("connection refused")
	err := NewNetworkError("connect", "192.168.1.100:5540", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "network") || !strings.Contains(errMsg, "connect") || !strings.Contains(errMsg, "192.168.1.100:5540") {
		t.Errorf("Error() = %q, want message containing 'network', 'connect', and address", errMsg)
	}

	// Test IsNetworkError()
	if !IsNetworkError(err) {
		t.Error("IsNetworkError() should return true for NetworkError")
	}
}

func TestNotificationError(t *testing.T) {
	baseErr := fmt.Errorf("webhook failed")
	err := NewNotificationError("slack", baseErr)

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "notification") || !strings.Contains(errMsg, "slack") {
		t.Errorf("Error() = %q, want message containing 'notification' and 'slack'", errMsg)
	}

	// Test IsNotificationError()
	if !IsNotificationError(err) {
		t.Error("IsNotificationError() should return true for NotificationError")
	}
}

func TestSentinelErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{"ErrDeviceNotFound", ErrDeviceNotFound},
		{"ErrDeviceOffline", ErrDeviceOffline},
		{"ErrTimeout", ErrTimeout},
		{"ErrCircuitBreakerOpen", ErrCircuitBreakerOpen},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrConnectionClosed", ErrConnectionClosed},
		{"ErrNoPermission", ErrNoPermission},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that sentinel errors have non-empty messages
			if tc.err.Error() == "" {
				t.Errorf("%s has empty error message", tc.name)
			}

			// Test that sentinel errors can be wrapped and checked with errors.Is()
			wrapped := fmt.Errorf("operation failed: %w", tc.err)
			if !errors.Is(wrapped, tc.err) {
				t.Errorf("errors.Is() should find wrapped %s", tc.name)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Create a chain of errors
	baseErr := fmt.Errorf("base error")
	discoveryErr := NewDiscoveryError("scan", baseErr)
	storageErr := NewStorageError("write", "device-1", discoveryErr)

	// Test unwrapping works through the chain
	if !errors.Is(storageErr, baseErr) {
		t.Error("errors.Is() should find base error through chain")
	}

	// Test As() works for intermediate types
	var de *DiscoveryError
	if !errors.As(storageErr, &de) {
		t.Error("errors.As() should find DiscoveryError in chain")
	}

	var se *StorageError
	if !errors.As(storageErr, &se) {
		t.Error("errors.As() should find StorageError at top of chain")
	}
}

func TestErrorsWithoutUnderlyingError(t *testing.T) {
	// Test errors can be created without underlying errors
	discoveryErr := NewDiscoveryError("scan", nil)
	if discoveryErr.Error() == "" {
		t.Error("DiscoveryError without underlying error should have message")
	}

	storageErr := NewStorageError("write", "", nil)
	if storageErr.Error() == "" {
		t.Error("StorageError without underlying error should have message")
	}

	configErr := NewConfigError("field", "", nil)
	if configErr.Error() == "" {
		t.Error("ConfigError without underlying error should have message")
	}
}

func TestIsHelperWithWrongType(t *testing.T) {
	// Test that Is helpers return false for wrong error types
	genericErr := fmt.Errorf("generic error")

	if IsDiscoveryError(genericErr) {
		t.Error("IsDiscoveryError() should return false for generic error")
	}

	if IsStorageError(genericErr) {
		t.Error("IsStorageError() should return false for generic error")
	}

	if IsConfigError(genericErr) {
		t.Error("IsConfigError() should return false for generic error")
	}

	if IsMonitoringError(genericErr) {
		t.Error("IsMonitoringError() should return false for generic error")
	}

	if IsValidationError(genericErr) {
		t.Error("IsValidationError() should return false for generic error")
	}

	if IsNetworkError(genericErr) {
		t.Error("IsNetworkError() should return false for generic error")
	}

	if IsNotificationError(genericErr) {
		t.Error("IsNotificationError() should return false for generic error")
	}
}
