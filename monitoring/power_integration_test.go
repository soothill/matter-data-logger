// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring_test

import (
	"context"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/stretchr/testify/assert"
)

type mockScanner struct{}

func (s *mockScanner) GetDeviceByID(deviceID string) *discovery.Device {
	return &discovery.Device{
		Name: "Test Device",
	}
}

func TestPowerMonitorIntegration(t *testing.T) {
	scanner := &mockScanner{}
	monitor := monitoring.NewPowerMonitor(100*time.Millisecond, scanner, 10)

	device := &discovery.Device{
		Name: "Test Device",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx, []*discovery.Device{device})

	select {
	case reading := <-monitor.Readings():
		assert.NotNil(t, reading)
		assert.Equal(t, "Test Device", reading.DeviceName)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for reading")
	}

	monitor.Stop()
}
