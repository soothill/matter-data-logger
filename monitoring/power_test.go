// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package monitoring_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/interfaces"
)

type mockErrorScanner struct{}

func (s *mockErrorScanner) GetDeviceByID(_ string) *discovery.Device {
	return &discovery.Device{
		Name: "Test Device",
	}
}

type mockErrorMatterClient struct{}

func (c *mockErrorMatterClient) ReadPower(_ context.Context, _ *discovery.Device) (*interfaces.PowerReading, error) {
	return nil, errors.New("test error")
}

func TestPowerMonitorErrorPath(t *testing.T) {
	scanner := &mockErrorScanner{}
	monitor := monitoring.NewPowerMonitor(100*time.Millisecond, scanner, 10)

	device := &discovery.Device{
		Name: "Test Device",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx, []*discovery.Device{device})

	select {
	case <-monitor.Readings():
		t.Fatal("received a reading when an error was expected")
	case <-time.After(200 * time.Millisecond):
		// Expected timeout
	}

	monitor.Stop()
}
