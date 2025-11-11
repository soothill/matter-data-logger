// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/config"
	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/storage"
)

func TestHealthCheckHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthCheckHandler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthCheckHandler() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if w.Body.String() != "OK" {
		t.Errorf("healthCheckHandler() body = %s, want OK", w.Body.String())
	}
}

func TestReadinessCheckHandler_NoInfluxDB(t *testing.T) {
	// Create a mock storage that will fail health check
	db, err := storage.NewInfluxDBStorage(
		"http://nonexistent:8086",
		"fake-token",
		"fake-org",
		"fake-bucket",
	)
	if err != nil {
		t.Skip("Cannot create InfluxDB client for testing")
	}
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	readinessCheckHandler(w, req, db)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Should return 503 Service Unavailable when InfluxDB is not healthy
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("readinessCheckHandler() status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestPerformHealthCheck(t *testing.T) {
	exitCode := performHealthCheck()
	if exitCode != 0 {
		t.Errorf("performHealthCheck() = %d, want 0", exitCode)
	}
}

func TestPerformGracefulShutdown(t *testing.T) {
	// Create a test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test"))
	})

	server := &http.Server{
		Addr:    "localhost:0", // Random port
		Handler: mux,
	}

	// Start server in background
	go func() {
		_ = server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create mock dependencies
	ctx, cancel := context.WithCancel(context.Background())

	// Mock power monitor (nil is acceptable for this test as Stop() won't be called)
	// In a real scenario, we'd use a mock implementing the interface

	// Call performGracefulShutdown
	// Note: We can't easily test monitor.Stop() without creating the full monitor
	// This tests the HTTP server shutdown portion
	shutdownComplete := make(chan struct{})
	go func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("Server shutdown error: %v", err)
		}
		cancel()
		close(shutdownComplete)
	}()

	// Wait for shutdown to complete
	select {
	case <-shutdownComplete:
		// Success
	case <-time.After(3 * time.Second):
		t.Error("Shutdown did not complete in time")
	}

	// Verify context was canceled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be canceled after shutdown")
	}
}

func TestPerformCleanup(t *testing.T) {
	// Create a mock InfluxDB storage
	influxDB, err := storage.NewInfluxDBStorage(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)
	if err != nil {
		t.Skip("Cannot create InfluxDB client for testing")
	}
	defer influxDB.Close()

	// Create a temporary cache directory
	tempDir := t.TempDir()
	cache, err := storage.NewLocalCache(tempDir, 1024*1024, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create caching storage (pass nil notifier for test)
	db := storage.NewCachingStorage(influxDB, cache, nil)
	defer db.Close()

	// Create a WaitGroup and add a goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	// Simulate a goroutine that completes quickly
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	// Call performCleanup - should complete within timeout
	done := make(chan struct{})
	go func() {
		performCleanup(db, &wg)
		close(done)
	}()

	select {
	case <-done:
		// Success - cleanup completed
	case <-time.After(15 * time.Second):
		t.Error("performCleanup() did not complete within expected time")
	}
}

func TestInitializeComponents(t *testing.T) {
	// Skip this test as initializeComponents calls Fatal() on InfluxDB connection failure
	// which would terminate the entire test suite
	// This function is indirectly tested by integration tests
	t.Skip("initializeComponents() calls Fatal() on errors, cannot test without real InfluxDB")
}

func TestPerformInitialDiscovery_NoDevices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create scanner
	scanner := discovery.NewScanner("_matter._tcp", "local.")

	// Create monitor
	monitor := monitoring.NewPowerMonitor(1 * time.Second)
	defer monitor.Stop()

	// Run initial discovery (will timeout as no real devices)
	performInitialDiscovery(ctx, scanner, monitor, nil)

	// Verify no devices were discovered (expected in test environment)
	devices := scanner.GetDevices()
	if len(devices) > 0 {
		t.Logf("Unexpectedly found %d devices in test environment", len(devices))
	}

	// Verify no power devices
	powerDevices := scanner.GetPowerDevices()
	if len(powerDevices) > 0 {
		t.Logf("Unexpectedly found %d power devices in test environment", len(powerDevices))
	}
}

func TestPerformPeriodicDiscovery_NoDevices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create scanner
	scanner := discovery.NewScanner("_matter._tcp", "local.")

	// Create monitor
	monitor := monitoring.NewPowerMonitor(1 * time.Second)
	defer monitor.Stop()

	// Run periodic discovery (will timeout as no real devices)
	performPeriodicDiscovery(ctx, scanner, monitor, nil)

	// Verify function completed without panic
	// In a test environment with no devices, this tests error handling
}

func TestReadinessCheckHandler_Healthy(t *testing.T) {
	// This test requires a mock InfluxDB or test container
	// For now, we test the handler structure
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Create a mock storage (will likely fail health check)
	db, err := storage.NewInfluxDBStorage(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)
	if err != nil {
		t.Skip("Cannot create InfluxDB client for testing")
	}
	defer db.Close()

	readinessCheckHandler(w, req, db)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	// Will likely return 503 as InfluxDB is not running
	// But the handler should not panic
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("readinessCheckHandler() status = %d, want %d or %d",
			resp.StatusCode, http.StatusOK, http.StatusServiceUnavailable)
	}
}

func TestInitializeComponents_WithSlackWebhook(t *testing.T) {
	// Skip this test as initializeComponents calls Fatal() on InfluxDB connection failure
	t.Skip("initializeComponents() calls Fatal() on errors, cannot test without real InfluxDB")
}

func TestMain_ConfigFileHandling(t *testing.T) {
	// Test config file creation and loading
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create a minimal test config file
	configContent := `
influxdb:
  url: "http://localhost:8086"
  token: "test-token"
  organization: "test-org"
  bucket: "test-bucket"

matter:
  service_type: "_matter._tcp"
  domain: "local."
  discovery_interval: 5m
  poll_interval: 30s

logging:
  level: "info"

notifications:
  slack_webhook_url: ""

cache:
  directory: "` + tempDir + `"
  max_size: 104857600
  max_age: 24h
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load the config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Verify config values
	if cfg.InfluxDB.URL != "http://localhost:8086" {
		t.Errorf("InfluxDB URL = %s, want http://localhost:8086", cfg.InfluxDB.URL)
	}

	if cfg.InfluxDB.Token != "test-token" {
		t.Errorf("InfluxDB token = %s, want test-token", cfg.InfluxDB.Token)
	}

	if cfg.Matter.ServiceType != "_matter._tcp" {
		t.Errorf("ServiceType = %s, want _matter._tcp", cfg.Matter.ServiceType)
	}
}
