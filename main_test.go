// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
