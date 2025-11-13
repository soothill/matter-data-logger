// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package storage provides persistent storage for power consumption data with local caching.
//
// This package implements a two-tier storage architecture:
//  1. Primary storage: InfluxDB time-series database
//  2. Fallback storage: Local file-based cache
//
// The caching layer provides resilience against InfluxDB outages by automatically
// falling back to local storage when the database is unavailable, then replaying
// cached data when connectivity is restored.
//
// # Architecture
//
// Storage Components:
//   - InfluxDBStorage: Direct InfluxDB client with circuit breaker protection
//   - LocalCache: File-based JSON storage with size and age limits
//   - CachingStorage: Wrapper combining InfluxDB + cache with automatic failover
//
// The CachingStorage wrapper monitors InfluxDB health in the background and
// automatically switches between direct writes and cached writes based on
// availability.
//
// # Automatic Failover
//
// When InfluxDB writes fail:
//  1. Readings are written to local cache (JSON files)
//  2. Slack notification sent (if configured)
//  3. Background health checker polls InfluxDB every 30 seconds
//  4. When healthy, cached readings are replayed in order
//  5. Recovery notification sent
//
// # Circuit Breaker
//
// The InfluxDB storage uses the circuit breaker pattern to prevent cascading
// failures when the database is unavailable:
//   - Trips after 5 requests with 60% failure ratio within 60s window
//   - 30 second timeout before attempting recovery
//   - State transitions are logged for monitoring
//
// # Cache Management
//
// The local cache has configurable limits:
//   - Max size: Default 100 MB (configurable)
//   - Max age: Default 24 hours (configurable)
//   - Old entries are cleaned up automatically on startup
//   - Warning notifications at 80% capacity
//
// # Thread Safety
//
// All storage operations are thread-safe and can be called concurrently from
// multiple goroutines. The LocalCache uses a mutex to protect file operations,
// and CachingStorage uses read-write locks for cache state management.
//
// # Flux Query Security
//
// All Flux queries sanitize user inputs to prevent injection attacks:
//   - Special characters are escaped (quotes, backslashes, newlines)
//   - Input length is limited (1000 characters max)
//   - Null bytes are removed
//
// # Example Usage
//
// Direct InfluxDB storage:
//
//	storage, err := storage.NewInfluxDBStorage(url, token, org, bucket)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer storage.Close()
//
//	reading := &monitoring.PowerReading{
//	    DeviceID:  "device-1",
//	    Power:     100.5,
//	    Timestamp: time.Now(),
//	}
//	storage.WriteReading(context.Background(), reading)
//
// Caching storage with automatic failover:
//
//	influxDB, _ := storage.NewInfluxDBStorage(url, token, org, bucket)
//	cache, _ := storage.NewLocalCache("/var/cache/app", 100*1024*1024, 24*time.Hour)
//	notifier := slacknotifier.New(webhookURL)
//
//	cachingStorage := storage.NewCachingStorage(influxDB, cache, notifier)
//	defer cachingStorage.Close()
//
//	// Writes to InfluxDB, falls back to cache on failure
//	cachingStorage.WriteReading(context.Background(), reading)
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/soothill/matter-data-logger/pkg/interfaces"
	"github.com/soothill/matter-data-logger/pkg/logger"
	"github.com/soothill/matter-data-logger/pkg/util"
)

const (
	defaultCacheDir     = "/var/cache/matter-data-logger"
	cacheFilePrefix     = "cache_"
	cacheFileExt        = ".json"
	defaultMaxSize      = 100 * 1024 * 1024 // 100 MB
	defaultMaxAge       = 24 * time.Hour
	replayBatchSize     = 100
	healthCheckInterval = 30 * time.Second
)

// LocalCache provides file-based caching for power readings
type LocalCache struct {
	cacheDir    string
	maxSize     int64
	maxAge      time.Duration
	mu          sync.Mutex
	currentSize int64
}

// CachedReading represents a power reading stored in cache
type CachedReading struct {
	Reading   *interfaces.PowerReading `json:"reading"`
	CachedAt  time.Time                `json:"cached_at"`
	AttemptID string                   `json:"attempt_id"`
}

// NewLocalCache creates a new local cache
func NewLocalCache(cacheDir string, maxSize int64, maxAge time.Duration) (*LocalCache, error) {
	if cacheDir == "" {
		cacheDir = defaultCacheDir
	}
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	if maxAge <= 0 {
		maxAge = defaultMaxAge
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &LocalCache{
		cacheDir: cacheDir,
		maxSize:  maxSize,
		maxAge:   maxAge,
	}

	// Calculate current cache size
	if err := cache.updateCurrentSize(); err != nil {
		logger.Warn().Err(err).Msg("Failed to calculate initial cache size")
	}

	// Clean up old cache files on startup
	if err := cache.CleanupOld(); err != nil {
		logger.Warn().Err(err).Msg("Failed to cleanup old cache files")
	}

	return cache, nil
}

// Write writes a reading to the cache
func (lc *LocalCache) Write(reading *interfaces.PowerReading) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Check if cache is full
	if lc.currentSize >= lc.maxSize {
		return fmt.Errorf("cache is full (%d >= %d bytes)", lc.currentSize, lc.maxSize)
	}

	cached := &CachedReading{
		Reading:   reading,
		CachedAt:  time.Now(),
		AttemptID: fmt.Sprintf("%d_%s", time.Now().UnixNano(), reading.DeviceID),
	}

	filename := lc.generateFilename(cached.AttemptID)
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal reading: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	lc.currentSize += int64(len(data))
	logger.Debug().
		Str("device_id", reading.DeviceID).
		Str("filename", filepath.Base(filename)).
		Int64("cache_size", lc.currentSize).
		Msg("Written reading to cache")

	return nil
}

// ListCachedReadings returns all cached readings sorted by timestamp
func (lc *LocalCache) ListCachedReadings() ([]*CachedReading, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(lc.cacheDir, cacheFilePrefix+"*"+cacheFileExt))
	if err != nil {
		return nil, fmt.Errorf("failed to list cache files: %w", err)
	}

	var readings []*CachedReading
	for _, file := range files {
		data, err := util.ReadFileSafely(file)
		if err != nil {
			logger.Warn().Err(err).Str("file", file).Msg("Failed to read cache file")
			continue
		}

		var cached CachedReading
		if err := json.Unmarshal(data, &cached); err != nil {
			logger.Warn().Err(err).Str("file", file).Msg("Failed to unmarshal cache file")
			continue
		}

		readings = append(readings, &cached)
	}

	// Sort by cached timestamp
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].CachedAt.Before(readings[j].CachedAt)
	})

	return readings, nil
}

// DeleteCached deletes a specific cached reading
func (lc *LocalCache) DeleteCached(attemptID string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	filename := lc.generateFilename(attemptID)

	// Get file size before deleting
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to stat cache file: %w", err)
	}

	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	lc.currentSize -= info.Size()
	logger.Debug().Str("attempt_id", attemptID).Msg("Deleted cached reading")

	return nil
}

// CleanupOld removes cache files older than maxAge
func (lc *LocalCache) CleanupOld() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(lc.cacheDir, cacheFilePrefix+"*"+cacheFileExt))
	if err != nil {
		return fmt.Errorf("failed to list cache files: %w", err)
	}

	cutoff := time.Now().Add(-lc.maxAge)
	deletedCount := 0

	for _, file := range files {
		data, err := util.ReadFileSafely(file)
		if err != nil {
			continue
		}

		var cached CachedReading
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}

		if cached.CachedAt.Before(cutoff) {
			if err := os.Remove(file); err != nil {
				logger.Warn().Err(err).Str("file", file).Msg("Failed to delete old cache file")
				continue
			}
			deletedCount++
			lc.currentSize -= int64(len(data))
		}
	}

	if deletedCount > 0 {
		logger.Info().Int("count", deletedCount).Msg("Cleaned up old cache files")
	}

	return nil
}

// GetCacheSize returns the current cache size in bytes
func (lc *LocalCache) GetCacheSize() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.currentSize
}

// GetMaxSize returns the maximum cache size
func (lc *LocalCache) GetMaxSize() int64 {
	return lc.maxSize
}

// updateCurrentSize recalculates the current cache size
func (lc *LocalCache) updateCurrentSize() error {
	files, err := filepath.Glob(filepath.Join(lc.cacheDir, cacheFilePrefix+"*"+cacheFileExt))
	if err != nil {
		return fmt.Errorf("failed to list cache files: %w", err)
	}

	var totalSize int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}

	lc.currentSize = totalSize
	return nil
}

// generateFilename generates a cache filename for an attempt ID
func (lc *LocalCache) generateFilename(attemptID string) string {
	return filepath.Join(lc.cacheDir, cacheFilePrefix+attemptID+cacheFileExt)
}

// CachingStorage wraps InfluxDBStorage with local caching support
type CachingStorage struct {
	storage             interfaces.TimeSeriesStorage
	cache               *LocalCache
	notifier            interfaces.Notifier
	cb                  *CircuitBreaker
	ctx                 context.Context
	cancel              context.CancelFunc
	replayWg            sync.WaitGroup
	cacheEnabled        bool
	cacheMutex          sync.RWMutex
	healthCheckInterval time.Duration
}

// CachingStorageOption defines a functional option for configuring CachingStorage.
type CachingStorageOption func(*CachingStorage)

// WithHealthCheckInterval sets a custom health check interval.
func WithHealthCheckInterval(interval time.Duration) CachingStorageOption {
	return func(cs *CachingStorage) {
		cs.healthCheckInterval = interval
	}
}

// NewCachingStorage creates a new caching storage wrapper
func NewCachingStorage(storage interfaces.TimeSeriesStorage, cache *LocalCache, notifier interfaces.Notifier, opts ...CachingStorageOption) *CachingStorage {
	ctx, cancel := context.WithCancel(context.Background())

	cs := &CachingStorage{
		storage:             storage,
		cache:               cache,
		notifier:            notifier,
		ctx:                 ctx,
		cancel:              cancel,
		cacheEnabled:        false,
		healthCheckInterval: healthCheckInterval,
	}

	for _, opt := range opts {
		opt(cs)
	}

	// Start background health monitoring and replay goroutine
	cs.replayWg.Add(1)
	go cs.monitorAndReplay()

	return cs
}

// WriteReading writes a reading, falling back to cache if InfluxDB is unavailable
// The context can be used for cancellation and timeout control
func (cs *CachingStorage) WriteReading(ctx context.Context, reading *interfaces.PowerReading) error {
	// Try to write to InfluxDB first
	err := cs.cb.Execute(ctx, func(ctx context.Context) error {
		return cs.storage.WriteReading(ctx, reading)
	})
	if err == nil {
		return nil
	}

	// If write failed, cache the reading
	logger.Warn().Err(err).Str("device_id", reading.DeviceID).Msg("InfluxDB write failed, caching locally")

	cs.cacheMutex.Lock()
	if !cs.cacheEnabled {
		cs.cacheEnabled = true
		cs.cacheMutex.Unlock()
		// Send alert on first cache activation
		if cs.notifier != nil && cs.notifier.IsEnabled() {
			alertCtx, alertCancel := context.WithTimeout(cs.ctx, 5*time.Second)
			defer alertCancel()
			if notifyErr := cs.sendInfluxDBFailureAlert(alertCtx, err); notifyErr != nil {
				logger.Error().Err(notifyErr).Msg("Failed to send InfluxDB failure alert")
			}
		}
	} else {
		cs.cacheMutex.Unlock()
	}

	if cacheErr := cs.cache.Write(reading); cacheErr != nil {
		return fmt.Errorf("influxdb write failed and cache write failed: influxdb=%w, cache=%w", err, cacheErr)
	}

	// Check cache size and send warning if needed
	cacheSize := cs.cache.GetCacheSize()
	maxSize := cs.cache.GetMaxSize()
	if float64(cacheSize)/float64(maxSize) > 0.8 && cs.notifier != nil && cs.notifier.IsEnabled() {
		alertCtx, alertCancel := context.WithTimeout(cs.ctx, 5*time.Second)
		defer alertCancel()
		if notifyErr := cs.sendCacheWarningAlert(alertCtx, cacheSize, maxSize); notifyErr != nil {
			logger.Error().Err(notifyErr).Msg("Failed to send cache warning alert")
		}
	}

	return nil
}

// WriteBatch writes multiple readings
// The context can be used for cancellation and timeout control
func (cs *CachingStorage) WriteBatch(ctx context.Context, readings []*interfaces.PowerReading) error {
	for i, reading := range readings {
		if err := cs.WriteReading(ctx, reading); err != nil {
			return fmt.Errorf("failed to write reading %d/%d (device_id=%s): %w", i+1, len(readings), reading.DeviceID, err)
		}
	}
	return nil
}

// Flush flushes pending writes
func (cs *CachingStorage) Flush() {
	cs.storage.Flush()
}

// Close closes the storage and stops replay
func (cs *CachingStorage) Close() {
	logger.Info().Msg("Closing caching storage")
	cs.cancel()
	cs.replayWg.Wait()
	cs.storage.Close()
}

// Health checks storage health
func (cs *CachingStorage) Health(ctx context.Context) error {
	return cs.storage.Health(ctx)
}

// Client returns the underlying InfluxDB client
func (cs *CachingStorage) Client() interface{} {
	return cs.storage.Client()
}

// monitorAndReplay monitors InfluxDB health and replays cached data when available
func (cs *CachingStorage) monitorAndReplay() {
	defer cs.replayWg.Done()

	ticker := time.NewTicker(cs.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			// Check context before health check operation
			if cs.ctx.Err() != nil {
				return
			}
			cs.cacheMutex.RLock()
			cacheEnabled := cs.cacheEnabled
			cs.cacheMutex.RUnlock()

			if !cacheEnabled {
				continue
			}

			// Check if InfluxDB is healthy
			healthCtx, healthCancel := context.WithTimeout(cs.ctx, 5*time.Second)
			err := cs.storage.Health(healthCtx)
			healthCancel()

			if err != nil {
				logger.Debug().Err(err).Msg("InfluxDB still unhealthy, keeping cache enabled")
				continue
			}

			// InfluxDB is healthy, replay cached data
			logger.Info().Msg("InfluxDB is healthy, replaying cached data")
			if replayErr := cs.replayCachedData(); replayErr != nil {
				logger.Error().Err(replayErr).Msg("Failed to replay cached data")
				continue
			}

			// Disable cache
			cs.cacheMutex.Lock()
			cs.cacheEnabled = false
			cs.cacheMutex.Unlock()

			// Send recovery alert
			if cs.notifier != nil && cs.notifier.IsEnabled() {
				alertCtx, alertCancel := context.WithTimeout(cs.ctx, 5*time.Second)
				defer alertCancel()
				if notifyErr := cs.sendInfluxDBRecoveryAlert(alertCtx); notifyErr != nil {
					logger.Error().Err(notifyErr).Msg("Failed to send InfluxDB recovery alert")
				}
			}
		}
	}
}

// replayCachedData replays all cached readings to InfluxDB
func (cs *CachingStorage) replayCachedData() error {
	readings, err := cs.cache.ListCachedReadings()
	if err != nil {
		return fmt.Errorf("failed to list cached readings: %w", err)
	}

	if len(readings) == 0 {
		logger.Info().Msg("No cached readings to replay")
		return nil
	}

	logger.Info().Int("count", len(readings)).Msg("Replaying cached readings")

	successCount := 0
	failCount := 0

	for _, cached := range readings {
		// Use internal context for replay operations
		if err := cs.storage.WriteReading(cs.ctx, cached.Reading); err != nil {
			logger.Warn().
				Err(err).
				Str("device_id", cached.Reading.DeviceID).
				Str("attempt_id", cached.AttemptID).
				Msg("Failed to replay cached reading")
			failCount++
			continue
		}

		if err := cs.cache.DeleteCached(cached.AttemptID); err != nil {
			logger.Warn().Err(err).Str("attempt_id", cached.AttemptID).Msg("Failed to delete replayed reading from cache")
		}

		successCount++

		// Batch flush every N readings
		if successCount%replayBatchSize == 0 {
			cs.storage.Flush()
		}
	}

	// Final flush
	cs.storage.Flush()

	logger.Info().
		Int("success", successCount).
		Int("failed", failCount).
		Int("total", len(readings)).
		Msg("Finished replaying cached readings")

	return nil
}

func (cs *CachingStorage) sendInfluxDBFailureAlert(ctx context.Context, err error) error {
	return cs.notifier.SendAlert(ctx, "danger", "⚠️ InfluxDB Connection Failure",
		fmt.Sprintf("Failed to connect to InfluxDB: %v\nData will be cached locally until connection is restored.", err))
}

func (cs *CachingStorage) sendInfluxDBRecoveryAlert(ctx context.Context) error {
	return cs.notifier.SendAlert(ctx, "good", "✅ InfluxDB Connection Restored",
		"Connection to InfluxDB has been restored. Cached data will be replayed.")
}

func (cs *CachingStorage) sendCacheWarningAlert(ctx context.Context, cacheSize, maxSize int64) error {
	percentage := float64(cacheSize) / float64(maxSize) * 100
	return cs.notifier.SendAlert(ctx, "warning", "⚠️ Local Cache Usage High",
		fmt.Sprintf("Cache size: %d bytes (%.1f%% of max %d bytes)\nInfluxDB may be unavailable for an extended period.",
			cacheSize, percentage, maxSize))
}
