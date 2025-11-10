// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package storage provides InfluxDB storage for power consumption data.
package storage

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

// InfluxDBStorage handles writing power data to InfluxDB
type InfluxDBStorage struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	bucket   string
	org      string
}

// NewInfluxDBStorage creates a new InfluxDB storage client
func NewInfluxDBStorage(url, token, org, bucket string) (*InfluxDBStorage, error) {
	client := influxdb2.NewClient(url, token)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		client.Close()
		message := "unknown error"
		if health.Message != nil {
			message = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", message)
	}

	logger.Info().Str("url", url).Str("status", string(health.Status)).Msg("Connected to InfluxDB")

	writeAPI := client.WriteAPI(org, bucket)

	// Handle async write errors
	go func() {
		for err := range writeAPI.Errors() {
			logger.Error().Err(err).Msg("InfluxDB write error")
		}
	}()

	return &InfluxDBStorage{
		client:   client,
		writeAPI: writeAPI,
		bucket:   bucket,
		org:      org,
	}, nil
}

// WriteReading writes a power reading to InfluxDB
func (s *InfluxDBStorage) WriteReading(reading *monitoring.PowerReading) error {
	// Validate input
	if reading == nil {
		return fmt.Errorf("reading cannot be nil")
	}
	if reading.DeviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	if reading.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	p := influxdb2.NewPoint(
		"power_consumption",
		map[string]string{
			"device_id":   reading.DeviceID,
			"device_name": reading.DeviceName,
		},
		map[string]interface{}{
			"power":   reading.Power,
			"voltage": reading.Voltage,
			"current": reading.Current,
			"energy":  reading.Energy,
		},
		reading.Timestamp,
	)

	s.writeAPI.WritePoint(p)
	return nil
}

// WriteBatch writes multiple readings efficiently
func (s *InfluxDBStorage) WriteBatch(readings []*monitoring.PowerReading) error {
	if readings == nil {
		return fmt.Errorf("readings slice cannot be nil")
	}

	for i, reading := range readings {
		if err := s.WriteReading(reading); err != nil {
			return fmt.Errorf("failed to write reading at index %d: %w", i, err)
		}
	}
	return nil
}

// Flush forces all pending writes to complete
func (s *InfluxDBStorage) Flush() {
	s.writeAPI.Flush()
}

// Close closes the InfluxDB client and flushes pending writes
func (s *InfluxDBStorage) Close() {
	logger.Info().Msg("Closing InfluxDB connection")
	s.writeAPI.Flush()
	s.client.Close()
}

// QueryLatestReading retrieves the most recent power reading for a device
func (s *InfluxDBStorage) QueryLatestReading(ctx context.Context, deviceID string) (*monitoring.PowerReading, error) {
	// Validate input
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	queryAPI := s.client.QueryAPI(s.org)

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "power_consumption")
			|> filter(fn: (r) => r.device_id == "%s")
			|> last()
	`, s.bucket, deviceID)

	result, err := queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() {
		_ = result.Close()
	}()

	reading := &monitoring.PowerReading{
		DeviceID: deviceID,
	}

	for result.Next() {
		record := result.Record()

		if name, ok := record.ValueByKey("device_name").(string); ok {
			reading.DeviceName = name
		}

		reading.Timestamp = record.Time()

		switch record.Field() {
		case "power":
			if val, ok := record.Value().(float64); ok {
				reading.Power = val
			}
		case "voltage":
			if val, ok := record.Value().(float64); ok {
				reading.Voltage = val
			}
		case "current":
			if val, ok := record.Value().(float64); ok {
				reading.Current = val
			}
		case "energy":
			if val, ok := record.Value().(float64); ok {
				reading.Energy = val
			}
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query parsing failed: %w", result.Err())
	}

	return reading, nil
}
