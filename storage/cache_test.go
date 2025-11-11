// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/monitoring"
)

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

	reading := &monitoring.PowerReading{
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
		reading := &monitoring.PowerReading{
			DeviceID:   "test-device",
			DeviceName: "Test Device",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Power:      float64(100 + i*10),
			Voltage:    120.0,
			Current:    0.833,
			Energy:     1.0,
		}

		if err := cache.Write(reading); err != nil {
			t.Fatalf("Write() error = %v", err)
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

	reading := &monitoring.PowerReading{
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

	reading := &monitoring.PowerReading{
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

	reading := &monitoring.PowerReading{
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

	reading := &monitoring.PowerReading{
		DeviceID:   "test-device",
		DeviceName: "Test Device",
		Timestamp:  time.Now(),
		Power:      100.0,
		Voltage:    120.0,
		Current:    0.833,
		Energy:     1.0,
	}

	// First write should succeed
	if err := cache.Write(reading); err != nil {
		t.Fatalf("First Write() error = %v", err)
	}

	// Second write should fail (cache full)
	err = cache.Write(reading)
	if err == nil {
		t.Error("Expected error for cache full, got nil")
	}
}

// Mock notifier for testing
type mockNotifier struct {
	influxFailureCalled  bool
	influxRecoveryCalled bool
	cacheWarningCalled   bool
}

func (m *mockNotifier) SendInfluxDBFailure(_ context.Context, _ error) error {
	m.influxFailureCalled = true
	return nil
}

func (m *mockNotifier) SendInfluxDBRecovery(_ context.Context) error {
	m.influxRecoveryCalled = true
	return nil
}

func (m *mockNotifier) SendCacheWarning(_ context.Context, _, _ int64) error {
	m.cacheWarningCalled = true
	return nil
}

func (m *mockNotifier) IsEnabled() bool {
	return true
}

func TestCachingStorage_WriteReading_Success(t *testing.T) {
	// This test requires a real InfluxDB connection
	// For unit testing, we test the cache fallback logic
	t.Skip("Requires integration test with real InfluxDB")
}

func TestCachingStorage_WriteReading_CacheFallback(t *testing.T) {
	// Test that writing to cache works when InfluxDB fails
	// This would require mocking the InfluxDB storage
	t.Skip("Requires mocking InfluxDB storage")
}
