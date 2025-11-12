// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License
//
// stress_test.go adds stress tests for race conditions.

package monitoring

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

// TestPowerMonitor_Stress runs a stress test on the PowerMonitor to detect race conditions.
// It simulates multiple devices being added and monitored concurrently.
func TestPowerMonitor_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	logger.Initialize("debug")
	scanner := discovery.NewScanner("_matter._tcp", "local.")
	monitor := NewPowerMonitor(30*time.Second, scanner, 100)

	var wg sync.WaitGroup
	numDevices := 50
	numUpdates := 10

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start monitoring in a separate goroutine
	go func() {
		monitor.Start(ctx, []*discovery.Device{})
	}()

	// Concurrently add and update devices
	for i := 0; i < numDevices; i++ {
		wg.Add(1)
		go func(deviceID int) {
			defer wg.Done()
			for j := 0; j < numUpdates; j++ {
				device := &discovery.Device{
					Name: fmt.Sprintf("device-%d", deviceID),
				}
				monitor.StartMonitoringDevice(ctx, device)
				time.Sleep(10 * time.Millisecond) // Small delay to interleave operations
			}
		}(i)
	}

	wg.Wait()

	// Let it run for a bit to ensure monitoring goroutines are active
	time.Sleep(2 * time.Second)

	// Stop the monitor and check for any race conditions reported by the detector.
	// The race detector is enabled by running tests with the -race flag.
}
