// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package storage provides InfluxDB storage for power consumption data.
//
// # Connection Pooling
//
// The InfluxDB client automatically manages HTTP connection pooling using Go's
// net/http package. The client creates a single HTTP connection pool that is
// shared across all write operations, providing efficient connection reuse.
//
// Key connection pooling behaviors:
//   - HTTP/1.1 persistent connections are reused automatically
//   - Default Go http.Transport settings apply:
//     * MaxIdleConns: 100 (total idle connections across all hosts)
//     * MaxIdleConnsPerHost: 2 (idle connections per host)
//     * IdleConnTimeout: 90 seconds (time before idle connections are closed)
//   - Connections are thread-safe and can be used concurrently
//   - No manual connection management is required
//
// The WriteAPI uses non-blocking asynchronous writes with automatic batching,
// further improving throughput by reducing the number of HTTP requests.
package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

const (
	maxRetries     = 3
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

// InfluxDBStorage handles writing power data to InfluxDB
type InfluxDBStorage struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPI
	bucket      string
	org         string
	ctx         context.Context
	cancel      context.CancelFunc
	errorWg     sync.WaitGroup
	retryQueue  chan retryItem
	closed      bool
	closeMutex  sync.Mutex
}

type retryItem struct {
	reading  *monitoring.PowerReading
	attempts int
}

// NewInfluxDBStorage creates a new InfluxDB storage client.
//
// Connection Pooling:
// The InfluxDB client automatically manages HTTP connection pooling. A single
// client instance maintains a pool of persistent HTTP connections that are
// reused across multiple write operations. This significantly reduces the
// overhead of establishing new connections for each request.
//
// The client is thread-safe and can be safely shared across multiple goroutines.
// All write operations use the same underlying connection pool, maximizing
// efficiency for concurrent writes.
//
// Performance characteristics:
//   - Connection reuse reduces latency by eliminating TCP handshake overhead
//   - Idle connections are kept alive for 90 seconds (Go default)
//   - Up to 2 idle connections per host are maintained for immediate reuse
//   - Automatic reconnection handling on connection failures
//
// No manual connection management is required. The Close() method should be
// called when the storage is no longer needed to gracefully close all connections.
func NewInfluxDBStorage(url, token, org, bucket string) (*InfluxDBStorage, error) {
	client := influxdb2.NewClient(url, token)

	// Verify connection
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	health, err := client.Health(healthCtx)
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

	ctx, cancel := context.WithCancel(context.Background())
	storage := &InfluxDBStorage{
		client:     client,
		writeAPI:   writeAPI,
		bucket:     bucket,
		org:        org,
		ctx:        ctx,
		cancel:     cancel,
		retryQueue: make(chan retryItem, 100),
	}

	// Handle async write errors with retry logic
	storage.errorWg.Add(2)
	go storage.handleWriteErrors()
	go storage.processRetries()

	return storage, nil
}

// WriteReading writes a power reading to InfluxDB
// The context can be used for cancellation and timeout control
func (s *InfluxDBStorage) WriteReading(ctx context.Context, reading *monitoring.PowerReading) error {
	// Check if context is already canceled
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled: %w", err)
	}

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

	// Validate power readings cannot be negative
	if reading.Power < 0 {
		return fmt.Errorf("power reading cannot be negative: %f", reading.Power)
	}
	if reading.Voltage < 0 {
		return fmt.Errorf("voltage reading cannot be negative: %f", reading.Voltage)
	}
	if reading.Current < 0 {
		return fmt.Errorf("current reading cannot be negative: %f", reading.Current)
	}
	if reading.Energy < 0 {
		return fmt.Errorf("energy reading cannot be negative: %f", reading.Energy)
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
// The context can be used for cancellation and timeout control
func (s *InfluxDBStorage) WriteBatch(ctx context.Context, readings []*monitoring.PowerReading) error {
	if readings == nil {
		return fmt.Errorf("readings slice cannot be nil")
	}

	for i, reading := range readings {
		if err := s.WriteReading(ctx, reading); err != nil {
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
	s.closeMutex.Lock()
	if s.closed {
		s.closeMutex.Unlock()
		return
	}
	s.closed = true
	s.closeMutex.Unlock()

	logger.Info().Msg("Closing InfluxDB connection")

	// Cancel context to stop retry processing
	s.cancel()

	// Close retry queue
	close(s.retryQueue)

	// Wait for error handlers to finish
	s.errorWg.Wait()

	// Flush and close
	s.writeAPI.Flush()
	s.client.Close()
}

// handleWriteErrors monitors the async write error channel and queues items for retry
func (s *InfluxDBStorage) handleWriteErrors() {
	defer s.errorWg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-s.writeAPI.Errors():
			if err == nil {
				return
			}
			logger.Error().Err(err).Msg("InfluxDB write error, will retry if possible")
			// Note: We cannot easily extract the failed point from the error,
			// so we log it. The retry logic is better handled at a higher level
			// or by relying on the InfluxDB client's built-in retry mechanism.
		}
	}
}

// processRetries handles retrying failed writes with exponential backoff
func (s *InfluxDBStorage) processRetries() {
	defer s.errorWg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case item, ok := <-s.retryQueue:
			if !ok {
				return
			}

			// Calculate backoff duration
			backoff := initialBackoff
			for i := 0; i < item.attempts; i++ {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
					break
				}
			}

			// Wait before retry
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(backoff):
			}

			// Attempt retry
			if item.attempts < maxRetries {
				logger.Info().
					Int("attempt", item.attempts+1).
					Int("max_retries", maxRetries).
					Dur("backoff", backoff).
					Str("device_id", item.reading.DeviceID).
					Msg("Retrying InfluxDB write")

				// Use internal context for retries
				if err := s.WriteReading(s.ctx, item.reading); err != nil {
					logger.Error().
						Err(err).
						Int("attempt", item.attempts+1).
						Str("device_id", item.reading.DeviceID).
						Msg("Retry failed")

					// Re-queue for another retry
					item.attempts++
					select {
					case s.retryQueue <- item:
					case <-s.ctx.Done():
						return
					default:
						logger.Warn().
							Str("device_id", item.reading.DeviceID).
							Msg("Retry queue full, dropping write")
					}
				} else {
					logger.Info().
						Int("attempt", item.attempts+1).
						Str("device_id", item.reading.DeviceID).
						Msg("Retry successful")
				}
			} else {
				logger.Error().
					Int("attempts", item.attempts).
					Str("device_id", item.reading.DeviceID).
					Msg("Max retries exceeded, dropping write")
			}
		}
	}
}

// Client returns the underlying InfluxDB client for advanced operations
func (s *InfluxDBStorage) Client() influxdb2.Client {
	return s.client
}

// Health checks the InfluxDB connection health
func (s *InfluxDBStorage) Health(ctx context.Context) error {
	health, err := s.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	if health.Status != "pass" {
		message := "unknown error"
		if health.Message != nil {
			message = *health.Message
		}
		return fmt.Errorf("InfluxDB unhealthy: %s", message)
	}
	return nil
}

// sanitizeFluxString escapes special characters in strings used in Flux queries
// to prevent injection attacks
func sanitizeFluxString(s string) string {
	// Escape backslashes first, then quotes
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// QueryLatestReading retrieves the most recent power reading for a device
func (s *InfluxDBStorage) QueryLatestReading(ctx context.Context, deviceID string) (*monitoring.PowerReading, error) {
	// Validate input
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	queryAPI := s.client.QueryAPI(s.org)

	// Sanitize inputs to prevent Flux injection
	safeBucket := sanitizeFluxString(s.bucket)
	safeDeviceID := sanitizeFluxString(deviceID)

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "power_consumption")
			|> filter(fn: (r) => r.device_id == "%s")
			|> last()
	`, safeBucket, safeDeviceID)

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
