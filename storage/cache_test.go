// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/pkg/interfaces"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

func init() {
	logger.Initialize("debug")
}

// mockTimeSeriesStorage is a mock implementation of interfaces.TimeSeriesStorage
type mockTimeSeriesStorage struct {
	mu               sync.Mutex
	writeReadingFunc func(ctx context.Context, reading *interfaces.PowerReading) error
	writeBatchFunc   func(ctx context.Context, readings []*interfaces.PowerReading) error
	flushFunc        func()
	closeFunc        func()
	healthFunc       func(ctx context.Context) error
	queryFunc        func(ctx context.Context, deviceID string) (*interfaces.PowerReading, error)
	clientFunc       func() interface{}
}

func (m *mockTimeSeriesStorage) WriteReading(ctx context.Context, reading *interfaces.PowerReading) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeReadingFunc(ctx, reading)
}

func (m *mockTimeSeriesStorage) WriteBatch(ctx context.Context, readings []*interfaces.PowerReading) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeBatchFunc(ctx, readings)
}

func (m *mockTimeSeriesStorage) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushFunc()
}

func (m *mockTimeSeriesStorage) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeFunc()
}

func (m *mockTimeSeriesStorage) Health(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.healthFunc(ctx)
}

func (m *mockTimeSeriesStorage) QueryLatestReading(ctx context.Context, deviceID string) (*interfaces.PowerReading, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queryFunc(ctx, deviceID)
}

func (m *mockTimeSeriesStorage) Client() interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clientFunc()
}

// mockNotifier is a mock implementation of Notifier
type mockNotifier struct {
	mu                   sync.Mutex
	influxFailureCalled  bool
	influxRecoveryCalled bool
	cacheWarningCalled   bool
	recoveryChan         chan struct{}
}

func newMockNotifier() *mockNotifier {
	return &mockNotifier{
		recoveryChan: make(chan struct{}, 1),
	}
}

func (m *mockNotifier) SendInfluxDBFailure(_ context.Context, _ error) error {
	m.mu.Lock()
	m.influxFailureCalled = true
	m.mu.Unlock()
	return nil
}

func (m *mockNotifier) SendInfluxDBRecovery(_ context.Context) error {
	m.mu.Lock()
	m.influxRecoveryCalled = true
	m.mu.Unlock()
	m.recoveryChan <- struct{}{}
	return nil
}

func (m *mockNotifier) SendCacheWarning(_ context.Context, _, _ int64) error {
	m.mu.Lock()
	m.cacheWarningCalled = true
	m.mu.Unlock()
	return nil
}

func (m *mockNotifier) IsEnabled() bool {
	return true
}

func TestNewLocalCache(t *testing.T) {
	tempDir := t.TempDir()

	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	if cache.cacheDir != tempDir {
		t.Errorf("cacheDir = %v, want %v", cache.cacheDir, tempDir)
	}

	if cache.maxSize != 1024*1024 {
		t.Errorf("maxSize = %v, want %v", cache.maxSize, 1024*1024)
	}

	if cache.maxAge != time.Hour {
		t.Errorf("maxAge = %v, want %v", cache.maxAge, time.Hour)
	}

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestLocalCache_Write(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	err = cache.Write(reading)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if err != nil {
		t.Fatalf("Failed to list cache files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 cache file, got %d", len(files))
	}
}

func TestLocalCache_ListCachedReadings(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	// Write multiple readings
	for i := 0; i < 3; i++ {
		reading := &interfaces.PowerReading{
			DeviceID:   "test-device",
			DeviceName: "Test Device",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Power:      float64(100 + i*10),
			Voltage:    120.0,
			Current:    0.833,
			Energy:     1.0,
		}

		if writeErr := cache.Write(reading); writeErr != nil {
			t.Fatalf("Write() error = %v", writeErr)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	readings, err := cache.ListCachedReadings()
	if err != nil {
		t.Fatalf("ListCachedReadings() error = %v", err)
	}

	if len(readings) != 3 {
		t.Errorf("ListCachedReadings() returned %d readings, want 3", len(readings))
	}

	// Verify readings are sorted by timestamp
	for i := 1; i < len(readings); i++ {
		if readings[i].CachedAt.Before(readings[i-1].CachedAt) {
			t.Error("Readings are not sorted by cached timestamp")
		}
	}
}

func TestLocalCache_DeleteCached(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	if writeErr := cache.Write(reading); writeErr != nil {
		t.Fatalf("Write() error = %v", writeErr)
	}

	readings, err := cache.ListCachedReadings()
	if err != nil {
		t.Fatalf("ListCachedReadings() error = %v", err)
	}

	if len(readings) != 1 {
		t.Fatalf("Expected 1 reading, got %d", len(readings))
	}

	attemptID := readings[0].AttemptID

	err = cache.DeleteCached(attemptID)
	if err != nil {
		t.Errorf("DeleteCached() error = %v", err)
	}

	// Verify reading was deleted
	readings, err = cache.ListCachedReadings()
	if err != nil {
		t.Fatalf("ListCachedReadings() error = %v", err)
	}

	if len(readings) != 0 {
		t.Errorf("Expected 0 readings after delete, got %d", len(readings))
	}
}

func TestLocalCache_CleanupOld(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, 1*time.Second)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	if writeErr := cache.Write(reading); writeErr != nil {
		t.Fatalf("Write() error = %v", writeErr)
	}

	// Wait for reading to become old
	time.Sleep(2 * time.Second)

	err = cache.CleanupOld()
	if err != nil {
		t.Errorf("CleanupOld() error = %v", err)
	}

	// Verify old reading was deleted
	readings, err := cache.ListCachedReadings()
	if err != nil {
		t.Fatalf("ListCachedReadings() error = %v", err)
	}

	if len(readings) != 0 {
		t.Errorf("Expected 0 readings after cleanup, got %d", len(readings))
	}
}

func TestLocalCache_GetCacheSize(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	initialSize := cache.GetCacheSize()
	if initialSize != 0 {
		t.Errorf("Initial cache size = %d, want 0", initialSize)
	}

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	if err := cache.Write(reading); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	sizeAfterWrite := cache.GetCacheSize()
	if sizeAfterWrite == 0 {
		t.Error("Cache size should be > 0 after write")
	}
}

func TestLocalCache_CacheFull(t *testing.T) {
	tempDir := t.TempDir()
	// Set very small max size
	cache, err := NewLocalCache(tempDir, 100, time.Hour)
	if err != nil {
		t.Fatalf("NewLocalCache() error = %v", err)
	}

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	// First write should succeed
	if writeErr := cache.Write(reading); writeErr != nil {
		t.Fatalf("First Write() error = %v", writeErr)
	}

	// Second write should fail (cache full)
	err = cache.Write(reading)
	if err == nil {
		t.Error("Expected error for cache full, got nil")
	}
}

func TestCachingStorage_WriteReading_Success(t *testing.T) {
	// Mock InfluxDB to return success
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return nil },
		healthFunc:       func(_ context.Context) error { return nil },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc:   func(_ context.Context, _ []*interfaces.PowerReading) error { return nil },
		queryFunc:        func(_ context.Context, _ string) (*interfaces.PowerReading, error) { return nil, nil },
		clientFunc:       func() interface{} { return nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mock notifier
	mockNotif := newMockNotifier()

	cs := NewCachingStorage(mockDB, cache, mockNotif, WithHealthCheckInterval(100*time.Millisecond))
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device-success",
		DeviceName: "Test Device Success",
		Timestamp:  time.Now(),
		Power:      100.0,
	}

	err = cs.WriteReading(context.Background(), reading)
	if err != nil {
		t.Errorf("WriteReading() error = %v, want nil", err)
	}

	// Ensure no cache files were written
	files, _ := filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if len(files) != 0 {
		t.Errorf("Expected 0 cache files, got %d", len(files))
	}

	// Ensure no notifications were sent
	if mockNotif.influxFailureCalled || mockNotif.influxRecoveryCalled || mockNotif.cacheWarningCalled {
		t.Error("No notifications should be sent on successful write")
	}
}

func TestCachingStorage_WriteReading_CacheFallback(t *testing.T) {
	// Mock InfluxDB to return an error
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return errors.New("influxdb error") },
		healthFunc:       func(_ context.Context) error { return errors.New("influxdb unhealthy") }, // Simulate unhealthy
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc: func(_ context.Context, _ []*interfaces.PowerReading) error {
			return errors.New("influxdb error")
		},
		queryFunc: func(_ context.Context, _ string) (*interfaces.PowerReading, error) {
			return nil, errors.New("influxdb error")
		},
		clientFunc: func() interface{} { return nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mock notifier
	mockNotif := newMockNotifier()

	cs := NewCachingStorage(mockDB, cache, mockNotif, WithHealthCheckInterval(100*time.Millisecond))
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device-fallback",
		DeviceName: "Test Device Fallback",
		Timestamp:  time.Now(),
		Power:      100.0,
	}

	// First write should fail to InfluxDB and fall back to cache
	err = cs.WriteReading(context.Background(), reading)
	if err != nil {
		t.Errorf("WriteReading() error = %v, want nil (cache should handle)", err)
	}

	// Ensure one cache file was written
	files, _ := filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if len(files) != 1 {
		t.Errorf("Expected 1 cache file, got %d", len(files))
	}

	// Ensure InfluxDB failure notification was sent
	if !mockNotif.influxFailureCalled {
		t.Error("Expected InfluxDB failure notification to be sent")
	}

	// Simulate InfluxDB recovery and replay
	var replayWg sync.WaitGroup
	replayWg.Add(1)

	mockDB.mu.Lock()
	mockDB.healthFunc = func(_ context.Context) error { return nil }
	mockDB.writeReadingFunc = func(_ context.Context, _ *interfaces.PowerReading) error {
		defer replayWg.Done()
		return nil
	}
	mockDB.mu.Unlock()

	// Wait for the replay to complete
	replayWg.Wait()

	// Wait for the recovery notification
	select {
	case <-mockNotif.recoveryChan:
		// notification received
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for recovery notification")
	}

	// Ensure cache is empty after replay
	files, _ = filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if len(files) != 0 {
		t.Errorf("Expected 0 cache files after replay, got %d", len(files))
	}

	// Ensure InfluxDB recovery notification was sent
	mockNotif.mu.Lock()
	if !mockNotif.influxRecoveryCalled {
		t.Error("Expected InfluxDB recovery notification to be sent")
	}
	mockNotif.mu.Unlock()
}

// triggerReplay is a test helper to manually trigger the replay logic
func (cs *CachingStorage) triggerReplay() {
	cs.replayWg.Add(1)
	go func() {
		defer cs.replayWg.Done()
		cs.cacheMutex.RLock()
		cacheEnabled := cs.cacheEnabled
		cs.cacheMutex.RUnlock()

		if !cacheEnabled {
			return
		}

		// Check if InfluxDB is healthy
		healthCtx, healthCancel := context.WithTimeout(cs.ctx, 5*time.Second)
		err := cs.storage.Health(healthCtx)
		healthCancel()

		if err != nil {
			logger.Debug().Err(err).Msg("InfluxDB still unhealthy, keeping cache enabled")
			return
		}

		// InfluxDB is healthy, replay cached data
		logger.Info().Msg("InfluxDB is healthy, replaying cached data")
		if replayErr := cs.replayCachedData(); replayErr != nil {
			logger.Error().Err(replayErr).Msg("Failed to replay cached data")
			return
		}

		// Disable cache
		cs.cacheMutex.Lock()
		cs.cacheEnabled = false
		cs.cacheMutex.Unlock()

		// Send recovery alert
		if cs.notifier != nil && cs.notifier.IsEnabled() {
			alertCtx, alertCancel := context.WithTimeout(cs.ctx, 5*time.Second)
			defer alertCancel()
			if notifyErr := cs.notifier.SendInfluxDBRecovery(alertCtx); notifyErr != nil {
				logger.Error().Err(notifyErr).Msg("Failed to send InfluxDB recovery alert")
			}
		}
	}()
}

func TestCachingStorage_WriteReading_CacheFull(t *testing.T) {
	// Mock InfluxDB to return an error
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return errors.New("influxdb error") },
		healthFunc:       func(_ context.Context) error { return errors.New("influxdb unhealthy") },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc: func(_ context.Context, _ []*interfaces.PowerReading) error {
			return errors.New("influxdb error")
		},
		queryFunc: func(_ context.Context, _ string) (*interfaces.PowerReading, error) {
			return nil, errors.New("influxdb error")
		},
		clientFunc: func() interface{} { return nil },
	}

	// Create a temporary cache directory with very small max size
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 100, time.Hour) // Max size 100 bytes
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mock notifier
	mockNotif := newMockNotifier()

	cs := NewCachingStorage(mockDB, cache, mockNotif)
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device-full",
		DeviceName: "Test Device Full",
		Timestamp:  time.Now(),
		Power:      100.0,
	}

	// First write should succeed (to cache)
	err = cs.WriteReading(context.Background(), reading)
	if err != nil {
		t.Errorf("First WriteReading() error = %v, want nil", err)
	}

	// Second write should fail (InfluxDB error + cache full)
	err = cs.WriteReading(context.Background(), reading)
	if err == nil || !strings.Contains(err.Error(), "cache is full") {
		t.Errorf("Expected cache full error, got %v", err)
	}
}

func TestCachingStorage_CacheWarning(t *testing.T) {
	// Mock InfluxDB to return an error
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return errors.New("influxdb error") },
		healthFunc:       func(_ context.Context) error { return errors.New("influxdb unhealthy") },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc: func(_ context.Context, _ []*interfaces.PowerReading) error {
			return errors.New("influxdb error")
		},
		queryFunc: func(_ context.Context, _ string) (*interfaces.PowerReading, error) {
			return nil, errors.New("influxdb error")
		},
		clientFunc: func() interface{} { return nil },
	}

	// Create a temporary cache directory with a size that triggers warning easily
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 200, time.Hour) // Max size 200 bytes
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mock notifier
	mockNotif := newMockNotifier()

	cs := NewCachingStorage(mockDB, cache, mockNotif)
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	reading := &interfaces.PowerReading{
		DeviceID:   "test-device-warning",
		DeviceName: "Test Device Warning",
		Timestamp:  time.Now(),
		Power:      100.0,
	}

	// Write enough data to trigger cache warning (e.g., 80% of 200 bytes = 160 bytes)
	// A single reading is usually ~150-200 bytes when marshaled
	err = cs.WriteReading(context.Background(), reading)
	if err != nil {
		t.Errorf("WriteReading() error = %v, want nil", err)
	}

	// Check if cache warning was sent
	if !mockNotif.cacheWarningCalled {
		t.Error("Expected cache warning notification to be sent")
	}
}

func TestCachingStorage_Close(t *testing.T) {
	// Mock InfluxDB and ensure Close is called
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return nil },
		healthFunc:       func(_ context.Context) error { return nil },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc:   func(_ context.Context, _ []*interfaces.PowerReading) error { return nil },
		queryFunc:        func(_ context.Context, _ string) (*interfaces.PowerReading, error) { return nil, nil },
		clientFunc:       func() interface{} { return nil },
	}
	closeCalled := make(chan struct{}, 1)
	mockDB.closeFunc = func() { closeCalled <- struct{}{} }

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	cs := NewCachingStorage(mockDB, cache, newMockNotifier())

	cs.Close()

	select {
	case <-closeCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Mock InfluxDB Close() was not called")
	}
}

func TestCachingStorage_Health(t *testing.T) {
	// Mock InfluxDB to return success
	mockDB := &mockTimeSeriesStorage{
		healthFunc:       func(_ context.Context) error { return nil },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return nil },
		writeBatchFunc:   func(_ context.Context, _ []*interfaces.PowerReading) error { return nil },
		queryFunc:        func(_ context.Context, _ string) (*interfaces.PowerReading, error) { return nil, nil },
		clientFunc:       func() interface{} { return nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	cs := NewCachingStorage(mockDB, cache, newMockNotifier())
	defer cs.Close()

	err = cs.Health(context.Background())
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}

	// Mock InfluxDB to return error
	mockDB.healthFunc = func(_ context.Context) error { return errors.New("influxdb unhealthy") }

	err = cs.Health(context.Background())
	if err == nil {
		t.Error("Health() expected error, got nil")
	}
}

func TestCachingStorage_Client(t *testing.T) {
	// Mock InfluxDB and ensure Client is returned
	mockClient := "mock_influxdb_client"
	mockDB := &mockTimeSeriesStorage{
		clientFunc:       func() interface{} { return mockClient },
		flushFunc:        func() {},
		closeFunc:        func() {},
		healthFunc:       func(_ context.Context) error { return nil },
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return nil },
		writeBatchFunc:   func(_ context.Context, _ []*interfaces.PowerReading) error { return nil },
		queryFunc:        func(_ context.Context, _ string) (*interfaces.PowerReading, error) { return nil, nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	cs := NewCachingStorage(mockDB, cache, newMockNotifier())
	defer cs.Close()

	client := cs.Client()
	if client != mockClient {
		t.Errorf("Client() = %v, want %v", client, mockClient)
	}
}

func TestCachingStorage_WriteBatch(t *testing.T) {
	// Mock InfluxDB to return success
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return nil },
		healthFunc:       func(_ context.Context) error { return nil },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc:   func(_ context.Context, _ []*interfaces.PowerReading) error { return nil },
		queryFunc:        func(_ context.Context, _ string) (*interfaces.PowerReading, error) { return nil, nil },
		clientFunc:       func() interface{} { return nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	cs := NewCachingStorage(mockDB, cache, newMockNotifier())
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	readings := []*interfaces.PowerReading{
		{
			DeviceID:   "batch-device-1",
			DeviceName: "Batch Device 1",
			Timestamp:  time.Now(),
			Power:      10.0,
		},
		{
			DeviceID:   "batch-device-2",
			DeviceName: "Batch Device 2",
			Timestamp:  time.Now(),
			Power:      20.0,
		},
	}

	err = cs.WriteBatch(context.Background(), readings)
	if err != nil {
		t.Errorf("WriteBatch() error = %v, want nil", err)
	}

	// Ensure no cache files were written
	files, _ := filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if len(files) != 0 {
		t.Errorf("Expected 0 cache files, got %d", len(files))
	}
}

func TestCachingStorage_WriteBatch_Fallback(t *testing.T) {
	// Mock InfluxDB to return an error
	mockDB := &mockTimeSeriesStorage{
		writeReadingFunc: func(_ context.Context, _ *interfaces.PowerReading) error { return errors.New("influxdb error") },
		healthFunc:       func(_ context.Context) error { return errors.New("influxdb unhealthy") },
		flushFunc:        func() {},
		closeFunc:        func() {},
		writeBatchFunc: func(_ context.Context, _ []*interfaces.PowerReading) error {
			return errors.New("influxdb error")
		},
		queryFunc: func(_ context.Context, _ string) (*interfaces.PowerReading, error) {
			return nil, errors.New("influxdb error")
		},
		clientFunc: func() interface{} { return nil },
	}

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Mock notifier
	mockNotif := newMockNotifier()

	cs := NewCachingStorage(mockDB, cache, mockNotif)
	cs.cb = NewCircuitBreaker(5, 30*time.Second, 1)
	defer cs.Close()

	readings := []*interfaces.PowerReading{
		{
			DeviceID:   "batch-fallback-1",
			DeviceName: "Batch Fallback 1",
			Timestamp:  time.Now(),
			Power:      10.0,
		},
		{
			DeviceID:   "batch-fallback-2",
			DeviceName: "Batch Fallback 2",
			Timestamp:  time.Now(),
			Power:      20.0,
		},
	}

	err = cs.WriteBatch(context.Background(), readings)
	if err != nil {
		t.Errorf("WriteBatch() error = %v, want nil (cache should handle)", err)
	}

	// Ensure cache files were written
	files, _ := filepath.Glob(filepath.Join(tempDir, "cache_*"+".json"))
	if len(files) != 2 {
		t.Errorf("Expected 2 cache files, got %d", len(files))
	}

	// Ensure InfluxDB failure notification was sent
	if !mockNotif.influxFailureCalled {
		t.Error("Expected InfluxDB failure notification to be sent")
	}
}
