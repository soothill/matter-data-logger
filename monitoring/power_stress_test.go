// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
)

func TestPowerMonitorRace(t *testing.T) {
	scanner := &mockScanner{}
	monitor := monitoring.NewPowerMonitor(10*time.Millisecond, scanner, 100)

	device := &discovery.Device{
		Name: "Test Device",
	}
	deviceID := device.GetDeviceID()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to start and stop monitoring
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			monitor.StartMonitoringDevice(ctx, device)
			time.Sleep(1 * time.Millisecond)
			monitor.StopMonitoringDevice(deviceID)
		}
	}()

	// Goroutine to read from the channel
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-monitor.Readings():
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()

	wg.Wait()
	monitor.Stop()
}
