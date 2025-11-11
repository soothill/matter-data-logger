// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

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

	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

const (
	defaultCacheDir  = "/var/cache/matter-data-logger"
	cacheFilePrefix  = "cache_"
	cacheFileExt     = ".json"
	defaultMaxSize   = 100 * 1024 * 1024 // 100 MB
	defaultMaxAge    = 24 * time.Hour
	replayBatchSize  = 100
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
	Reading   *monitoring.PowerReading `json:"reading"`
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
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
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
func (lc *LocalCache) Write(reading *monitoring.PowerReading) error {
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

	if err := os.WriteFile(filename, data, 0644); err != nil {
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
		data, err := os.ReadFile(file)
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
		data, err := os.ReadFile(file)
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
	storage      *InfluxDBStorage
	cache        *LocalCache
	notifier     Notifier
	ctx          context.Context
	cancel       context.CancelFunc
	replayWg     sync.WaitGroup
	cacheEnabled bool
	cacheMutex   sync.RWMutex
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	SendInfluxDBFailure(ctx context.Context, err error) error
	SendInfluxDBRecovery(ctx context.Context) error
	SendCacheWarning(ctx context.Context, cacheSize, maxSize int64) error
	IsEnabled() bool
}

// NewCachingStorage creates a new caching storage wrapper
func NewCachingStorage(storage *InfluxDBStorage, cache *LocalCache, notifier Notifier) *CachingStorage {
	ctx, cancel := context.WithCancel(context.Background())

	cs := &CachingStorage{
		storage:      storage,
		cache:        cache,
		notifier:     notifier,
		ctx:          ctx,
		cancel:       cancel,
		cacheEnabled: false,
	}

	// Start background health monitoring and replay goroutine
	cs.replayWg.Add(1)
	go cs.monitorAndReplay()

	return cs
}

// WriteReading writes a reading, falling back to cache if InfluxDB is unavailable
// The context can be used for cancellation and timeout control
func (cs *CachingStorage) WriteReading(ctx context.Context, reading *monitoring.PowerReading) error {
	// Try to write to InfluxDB first
	err := cs.storage.WriteReading(ctx, reading)
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
			if notifyErr := cs.notifier.SendInfluxDBFailure(alertCtx, err); notifyErr != nil {
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
		if notifyErr := cs.notifier.SendCacheWarning(alertCtx, cacheSize, maxSize); notifyErr != nil {
			logger.Error().Err(notifyErr).Msg("Failed to send cache warning alert")
		}
	}

	return nil
}

// WriteBatch writes multiple readings
// The context can be used for cancellation and timeout control
func (cs *CachingStorage) WriteBatch(ctx context.Context, readings []*monitoring.PowerReading) error {
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

	ticker := time.NewTicker(healthCheckInterval)
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
				if notifyErr := cs.notifier.SendInfluxDBRecovery(alertCtx); notifyErr != nil {
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
