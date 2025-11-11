// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package errors provides structured error types for the Matter Power Data Logger.
//
// This package defines custom error types that provide better error handling,
// inspection, and debugging capabilities compared to plain string errors.
//
// # Benefits of Structured Errors
//
//   - Type-safe error inspection with errors.As() and errors.Is()
//   - Context-rich error messages with operation and underlying error details
//   - Consistent error formatting across the application
//   - Better error wrapping and unwrapping support
//   - Enhanced logging with structured error fields
//
// # Example Usage
//
//	err := errors.NewDiscoveryError("mDNS scan", fmt.Errorf("network unreachable"))
//	if errors.IsDiscoveryError(err) {
//	    log.Printf("Discovery failed: %v", err)
//	}
//
//	var discoveryErr *errors.DiscoveryError
//	if errors.As(err, &discoveryErr) {
//	    log.Printf("Failed operation: %s", discoveryErr.Op)
//	}
package errors

import (
	"errors"
	"fmt"
)

// DiscoveryError represents an error during device discovery operations.
type DiscoveryError struct {
	Op  string // Operation being performed (e.g., "mDNS scan", "parse TXT record")
	Err error  // Underlying error
}

func (e *DiscoveryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("discovery %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("discovery %s failed", e.Op)
}

func (e *DiscoveryError) Unwrap() error {
	return e.Err
}

// NewDiscoveryError creates a new discovery error.
func NewDiscoveryError(op string, err error) *DiscoveryError {
	return &DiscoveryError{Op: op, Err: err}
}

// IsDiscoveryError checks if an error is a DiscoveryError.
func IsDiscoveryError(err error) bool {
	var de *DiscoveryError
	return errors.As(err, &de)
}

// StorageError represents an error during storage operations.
type StorageError struct {
	Op       string // Operation being performed (e.g., "write", "read", "query")
	DeviceID string // Device ID involved in the operation (if applicable)
	Err      error  // Underlying error
}

func (e *StorageError) Error() string {
	if e.DeviceID != "" {
		return fmt.Sprintf("storage %s (device=%s): %v", e.Op, e.DeviceID, e.Err)
	}
	if e.Err != nil {
		return fmt.Sprintf("storage %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("storage %s failed", e.Op)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// NewStorageError creates a new storage error.
func NewStorageError(op string, deviceID string, err error) *StorageError {
	return &StorageError{Op: op, DeviceID: deviceID, Err: err}
}

// IsStorageError checks if an error is a StorageError.
func IsStorageError(err error) bool {
	var se *StorageError
	return errors.As(err, &se)
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field string // Configuration field that caused the error
	Value string // Invalid value (optional, may be redacted for sensitive fields)
	Err   error  // Underlying error or description
}

func (e *ConfigError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("config error in field %q (value=%q): %v", e.Field, e.Value, e.Err)
	}
	if e.Err != nil {
		return fmt.Sprintf("config error in field %q: %v", e.Field, e.Err)
	}
	return fmt.Sprintf("config error in field %q", e.Field)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new configuration error.
func NewConfigError(field string, value string, err error) *ConfigError {
	return &ConfigError{Field: field, Value: value, Err: err}
}

// IsConfigError checks if an error is a ConfigError.
func IsConfigError(err error) bool {
	var ce *ConfigError
	return errors.As(err, &ce)
}

// MonitoringError represents an error during power monitoring operations.
type MonitoringError struct {
	Op       string // Operation being performed (e.g., "read power", "start monitoring")
	DeviceID string // Device ID involved
	Err      error  // Underlying error
}

func (e *MonitoringError) Error() string {
	if e.DeviceID != "" {
		return fmt.Sprintf("monitoring %s (device=%s): %v", e.Op, e.DeviceID, e.Err)
	}
	if e.Err != nil {
		return fmt.Sprintf("monitoring %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("monitoring %s failed", e.Op)
}

func (e *MonitoringError) Unwrap() error {
	return e.Err
}

// NewMonitoringError creates a new monitoring error.
func NewMonitoringError(op string, deviceID string, err error) *MonitoringError {
	return &MonitoringError{Op: op, DeviceID: deviceID, Err: err}
}

// IsMonitoringError checks if an error is a MonitoringError.
func IsMonitoringError(err error) bool {
	var me *MonitoringError
	return errors.As(err, &me)
}

// ValidationError represents a data validation error.
type ValidationError struct {
	Field   string // Field that failed validation
	Value   any    // Invalid value
	Reason  string // Why validation failed
	Details error  // Additional details (optional)
}

func (e *ValidationError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("validation error: field %q with value %v: %s (%v)", e.Field, e.Value, e.Reason, e.Details)
	}
	return fmt.Sprintf("validation error: field %q with value %v: %s", e.Field, e.Value, e.Reason)
}

func (e *ValidationError) Unwrap() error {
	return e.Details
}

// NewValidationError creates a new validation error.
func NewValidationError(field string, value any, reason string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Reason: reason}
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// NetworkError represents a network-related error.
type NetworkError struct {
	Op   string // Operation being performed (e.g., "connect", "mDNS broadcast")
	Addr string // Network address (if applicable)
	Err  error  // Underlying error
}

func (e *NetworkError) Error() string {
	if e.Addr != "" {
		return fmt.Sprintf("network %s (%s): %v", e.Op, e.Addr, e.Err)
	}
	if e.Err != nil {
		return fmt.Sprintf("network %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("network %s failed", e.Op)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// NewNetworkError creates a new network error.
func NewNetworkError(op string, addr string, err error) *NetworkError {
	return &NetworkError{Op: op, Addr: addr, Err: err}
}

// IsNetworkError checks if an error is a NetworkError.
func IsNetworkError(err error) bool {
	var ne *NetworkError
	return errors.As(err, &ne)
}

// NotificationError represents an error sending notifications.
type NotificationError struct {
	Type string // Notification type (e.g., "slack", "email")
	Err  error  // Underlying error
}

func (e *NotificationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("notification %s: %v", e.Type, e.Err)
	}
	return fmt.Sprintf("notification %s failed", e.Type)
}

func (e *NotificationError) Unwrap() error {
	return e.Err
}

// NewNotificationError creates a new notification error.
func NewNotificationError(notifType string, err error) *NotificationError {
	return &NotificationError{Type: notifType, Err: err}
}

// IsNotificationError checks if an error is a NotificationError.
func IsNotificationError(err error) bool {
	var ne *NotificationError
	return errors.As(err, &ne)
}

// Sentinel errors for common conditions
var (
	// ErrDeviceNotFound indicates a device was not found
	ErrDeviceNotFound = errors.New("device not found")

	// ErrDeviceOffline indicates a device is offline or unreachable
	ErrDeviceOffline = errors.New("device offline")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timeout")

	// ErrCircuitBreakerOpen indicates the circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker open")

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConnectionClosed indicates a connection was closed
	ErrConnectionClosed = errors.New("connection closed")

	// ErrNoPermission indicates insufficient permissions
	ErrNoPermission = errors.New("permission denied")
)
