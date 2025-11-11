// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

//go:build integration
// +build integration

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/testcontainers/testcontainers-go/modules/influxdb"
)

// TestIntegration_WriteReading tests writing a single reading to InfluxDB
func TestIntegration_WriteReading(t *testing.T) {
	ctx := context.Background()

	// Start InfluxDB container
	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection URL
	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	// Create storage
	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create a test reading
	reading := &monitoring.PowerReading{
		DeviceID:   "test-device-1",
		DeviceName: "Test Smart Plug",
		Timestamp:  time.Now(),
		Power:      100.5,
		Voltage:    120.0,
		Current:    0.8375,
		Energy:     1.5,
	}

	// Write reading
	if err := storage.WriteReading(ctx, reading); err != nil {
		t.Fatalf("WriteReading() error = %v", err)
	}

	// Flush to ensure write completes
	storage.Flush()

	// Verify health
	if err := storage.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

// TestIntegration_WriteBatch tests writing multiple readings
func TestIntegration_WriteBatch(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create multiple test readings
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
			Timestamp:  time.Now().Add(1 * time.Second),
			Power:      75.0,
			Voltage:    120.0,
			Current:    0.625,
			Energy:     1.0,
		},
		{
			DeviceID:   "device-3",
			DeviceName: "Device 3",
			Timestamp:  time.Now().Add(2 * time.Second),
			Power:      100.0,
			Voltage:    120.0,
			Current:    0.833,
			Energy:     1.5,
		},
	}

	// Write batch
	if err := storage.WriteBatch(ctx, readings); err != nil {
		t.Fatalf("WriteBatch() error = %v", err)
	}

	// Flush
	storage.Flush()
}

// TestIntegration_WriteReading_ValidationErrors tests validation errors
func TestIntegration_WriteReading_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	tests := []struct {
		name    string
		reading *monitoring.PowerReading
		wantErr bool
	}{
		{
			name:    "nil reading",
			reading: nil,
			wantErr: true,
		},
		{
			name: "empty device ID",
			reading: &monitoring.PowerReading{
				DeviceID:  "",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			reading: &monitoring.PowerReading{
				DeviceID:  "device-1",
				Timestamp: time.Time{},
			},
			wantErr: true,
		},
		{
			name: "negative power",
			reading: &monitoring.PowerReading{
				DeviceID:  "device-1",
				Timestamp: time.Now(),
				Power:     -10.0,
			},
			wantErr: true,
		},
		{
			name: "negative voltage",
			reading: &monitoring.PowerReading{
				DeviceID:  "device-1",
				Timestamp: time.Now(),
				Voltage:   -120.0,
			},
			wantErr: true,
		},
		{
			name: "negative current",
			reading: &monitoring.PowerReading{
				DeviceID:  "device-1",
				Timestamp: time.Now(),
				Current:   -0.5,
			},
			wantErr: true,
		},
		{
			name: "negative energy",
			reading: &monitoring.PowerReading{
				DeviceID:  "device-1",
				Timestamp: time.Now(),
				Energy:    -1.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.WriteReading(ctx, tt.reading)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteReading() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIntegration_QueryLatestReading tests querying the latest reading
func TestIntegration_QueryLatestReading(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Write test readings
	deviceID := "query-test-device"
	readings := []*monitoring.PowerReading{
		{
			DeviceID:   deviceID,
			DeviceName: "Query Test Device",
			Timestamp:  time.Now().Add(-2 * time.Minute),
			Power:      50.0,
			Voltage:    120.0,
			Current:    0.417,
			Energy:     0.5,
		},
		{
			DeviceID:   deviceID,
			DeviceName: "Query Test Device",
			Timestamp:  time.Now().Add(-1 * time.Minute),
			Power:      75.0,
			Voltage:    120.0,
			Current:    0.625,
			Energy:     1.0,
		},
		{
			DeviceID:   deviceID,
			DeviceName: "Query Test Device",
			Timestamp:  time.Now(),
			Power:      100.0,
			Voltage:    120.0,
			Current:    0.833,
			Energy:     1.5,
		},
	}

	for _, reading := range readings {
		if err := storage.WriteReading(ctx, reading); err != nil {
			t.Fatalf("Failed to write test reading: %v", err)
		}
	}

	// Flush to ensure writes complete
	storage.Flush()

	// Wait for data to be queryable
	time.Sleep(2 * time.Second)

	// Query latest reading
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	latest, err := storage.QueryLatestReading(queryCtx, deviceID)
	if err != nil {
		t.Fatalf("QueryLatestReading() error = %v", err)
	}

	// Verify we got a result
	if latest == nil {
		t.Fatal("QueryLatestReading() returned nil")
	}

	if latest.DeviceID != deviceID {
		t.Errorf("DeviceID = %v, want %v", latest.DeviceID, deviceID)
	}
}

// TestIntegration_QueryLatestReading_EmptyDeviceID tests validation
func TestIntegration_QueryLatestReading_EmptyDeviceID(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	_, err = storage.QueryLatestReading(ctx, "")
	if err == nil {
		t.Error("QueryLatestReading() with empty device ID should return error")
	}
}

// TestIntegration_Health tests the health check
func TestIntegration_Health(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test health check
	if err := storage.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}

	// Test health check with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := storage.Health(timeoutCtx); err != nil {
		t.Errorf("Health() with timeout error = %v", err)
	}
}

// TestIntegration_CloseAndFlush tests closing the storage
func TestIntegration_CloseAndFlush(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Write a reading
	reading := &monitoring.PowerReading{
		DeviceID:   "close-test-device",
		DeviceName: "Close Test",
		Timestamp:  time.Now(),
		Power:      50.0,
		Voltage:    120.0,
		Current:    0.417,
		Energy:     0.5,
	}

	if err := storage.WriteReading(ctx, reading); err != nil {
		t.Fatalf("WriteReading() error = %v", err)
	}

	// Test Flush
	storage.Flush()

	// Test Close (should call Flush internally)
	storage.Close()

	// Calling Close multiple times should not panic
	storage.Close()
}

// TestIntegration_Client tests the Client accessor
func TestIntegration_Client(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test Client accessor
	client := storage.Client()
	if client == nil {
		t.Error("Client() returned nil")
	}

	// Verify we can use the client for health check
	health, err := client.Health(ctx)
	if err != nil {
		t.Errorf("Client.Health() error = %v", err)
	}
	if health.Status != "pass" {
		t.Errorf("Client.Health() status = %v, want pass", health.Status)
	}
}

// TestIntegration_WriteBatch_EmptySlice tests empty batch handling
func TestIntegration_WriteBatch_EmptySlice(t *testing.T) {
	ctx := context.Background()

	influxContainer, err := influxdb.Run(ctx,
		"influxdb:2.7-alpine",
		influxdb.WithV2Auth("test-org", "test-bucket", "test-user", "test-password"),
		influxdb.WithV2AdminToken("test-token"),
	)
	if err != nil {
		t.Fatalf("Failed to start InfluxDB container: %v", err)
	}
	defer func() {
		if err := influxContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	url, err := influxContainer.ConnectionUrl(ctx)
	if err != nil {
		t.Fatalf("Failed to get InfluxDB URL: %v", err)
	}

	storage, err := NewInfluxDBStorage(url, "test-token", "test-org", "test-bucket")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test empty slice - should not error
	err = storage.WriteBatch(ctx, []*monitoring.PowerReading{})
	if err != nil {
		t.Errorf("WriteBatch() with empty slice error = %v", err)
	}

	// Test nil slice - should error
	err = storage.WriteBatch(ctx, nil)
	if err == nil {
		t.Error("WriteBatch() with nil slice should return error")
	}
}
