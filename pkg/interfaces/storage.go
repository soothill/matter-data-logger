// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package interfaces defines abstract interfaces for core system components.
// This package promotes loose coupling and testability by allowing
// dependency injection and easy mocking in tests.
package interfaces

import (
	"context"
	"time"
)

// PowerReading represents a power consumption measurement.
// This is redeclared here to avoid circular dependencies.
type PowerReading struct {
	DeviceID   string
	DeviceName string
	Timestamp  time.Time
	Power      float64 // Power in watts
	Voltage    float64 // Voltage in volts
	Current    float64 // Current in amperes
	Energy     float64 // Cumulative energy in kWh
}

// TimeSeriesStorage defines the interface for time-series data persistence.
// Implementations should handle power readings and provide health checks.
type TimeSeriesStorage interface {
	// WriteReading writes a single power reading to storage
	WriteReading(reading *PowerReading) error

	// WriteBatch writes multiple readings to storage efficiently
	WriteBatch(readings []*PowerReading) error

	// Flush ensures all pending writes are completed
	Flush()

	// Close gracefully shuts down the storage connection
	Close()

	// Health checks if the storage backend is healthy
	Health(ctx context.Context) error

	// QueryLatestReading retrieves the most recent reading for a device
	QueryLatestReading(ctx context.Context, deviceID string) (*PowerReading, error)
}
