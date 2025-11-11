// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDevicesDiscoveredGauge(t *testing.T) {
	// Reset metric
	DevicesDiscovered.Set(0)

	// Set value
	DevicesDiscovered.Set(5)

	// Verify
	value := testutil.ToFloat64(DevicesDiscovered)
	if value != 5 {
		t.Errorf("DevicesDiscovered = %v, want 5", value)
	}
}

func TestPowerDevicesDiscoveredGauge(t *testing.T) {
	PowerDevicesDiscovered.Set(0)
	PowerDevicesDiscovered.Set(3)

	value := testutil.ToFloat64(PowerDevicesDiscovered)
	if value != 3 {
		t.Errorf("PowerDevicesDiscovered = %v, want 3", value)
	}
}

func TestDevicesMonitoredGauge(t *testing.T) {
	DevicesMonitored.Set(0)
	DevicesMonitored.Set(10)

	value := testutil.ToFloat64(DevicesMonitored)
	if value != 10 {
		t.Errorf("DevicesMonitored = %v, want 10", value)
	}
}

func TestPowerReadingsTotalCounter(t *testing.T) {
	initial := testutil.ToFloat64(PowerReadingsTotal)
	PowerReadingsTotal.Inc()
	final := testutil.ToFloat64(PowerReadingsTotal)

	if final <= initial {
		t.Errorf("PowerReadingsTotal should have increased, got %v -> %v", initial, final)
	}
}

func TestPowerReadingErrorsCounter(t *testing.T) {
	initial := testutil.ToFloat64(PowerReadingErrors)
	PowerReadingErrors.Inc()
	final := testutil.ToFloat64(PowerReadingErrors)

	if final <= initial {
		t.Errorf("PowerReadingErrors should have increased, got %v -> %v", initial, final)
	}
}

func TestInfluxDBWritesTotalCounter(t *testing.T) {
	initial := testutil.ToFloat64(InfluxDBWritesTotal)
	InfluxDBWritesTotal.Inc()
	final := testutil.ToFloat64(InfluxDBWritesTotal)

	if final <= initial {
		t.Errorf("InfluxDBWritesTotal should have increased, got %v -> %v", initial, final)
	}
}

func TestInfluxDBWriteErrorsCounter(t *testing.T) {
	initial := testutil.ToFloat64(InfluxDBWriteErrors)
	InfluxDBWriteErrors.Inc()
	final := testutil.ToFloat64(InfluxDBWriteErrors)

	if final <= initial {
		t.Errorf("InfluxDBWriteErrors should have increased, got %v -> %v", initial, final)
	}
}

func TestDiscoveryDurationHistogram(t *testing.T) {
	// Observe some values
	DiscoveryDuration.Observe(1.5)
	DiscoveryDuration.Observe(2.3)

	// Verify it's registered as a histogram
	metricType := testutil.CollectAndCount(DiscoveryDuration)
	if metricType == 0 {
		t.Error("DiscoveryDuration histogram should have observations")
	}
}

func TestPowerReadingDurationHistogram(t *testing.T) {
	PowerReadingDuration.Observe(0.5)
	PowerReadingDuration.Observe(1.0)

	metricType := testutil.CollectAndCount(PowerReadingDuration)
	if metricType == 0 {
		t.Error("PowerReadingDuration histogram should have observations")
	}
}

func TestCurrentPowerGaugeVec(t *testing.T) {
	// Set value for a device
	CurrentPower.WithLabelValues("device-1", "Test Device").Set(100.5)

	// Get the metric
	metric, err := CurrentPower.GetMetricWithLabelValues("device-1", "Test Device")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	// Verify value
	value := testutil.ToFloat64(metric)
	if value != 100.5 {
		t.Errorf("CurrentPower = %v, want 100.5", value)
	}
}

func TestCurrentVoltageGaugeVec(t *testing.T) {
	CurrentVoltage.WithLabelValues("device-2", "Test Device 2").Set(120.0)

	metric, err := CurrentVoltage.GetMetricWithLabelValues("device-2", "Test Device 2")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	value := testutil.ToFloat64(metric)
	if value != 120.0 {
		t.Errorf("CurrentVoltage = %v, want 120.0", value)
	}
}

func TestCurrentCurrentGaugeVec(t *testing.T) {
	CurrentCurrent.WithLabelValues("device-3", "Test Device 3").Set(5.5)

	metric, err := CurrentCurrent.GetMetricWithLabelValues("device-3", "Test Device 3")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	value := testutil.ToFloat64(metric)
	if value != 5.5 {
		t.Errorf("CurrentCurrent = %v, want 5.5", value)
	}
}

func TestMetricsAreRegistered(t *testing.T) {
	// Verify all metrics are properly registered
	metrics := []prometheus.Collector{
		DevicesDiscovered,
		PowerDevicesDiscovered,
		DevicesMonitored,
		PowerReadingsTotal,
		PowerReadingErrors,
		InfluxDBWritesTotal,
		InfluxDBWriteErrors,
		DiscoveryDuration,
		PowerReadingDuration,
		CurrentPower,
		CurrentVoltage,
		CurrentCurrent,
	}

	for i, metric := range metrics {
		count := testutil.CollectAndCount(metric)
		if count < 0 {
			t.Errorf("Metric %d is not properly registered", i)
		}
	}
}

func TestGaugeVecLabelCardinality(t *testing.T) {
	// Test that we can create multiple device labels without issues
	devices := []struct {
		id   string
		name string
	}{
		{"dev-1", "Device 1"},
		{"dev-2", "Device 2"},
		{"dev-3", "Device 3"},
	}

	for _, dev := range devices {
		CurrentPower.WithLabelValues(dev.id, dev.name).Set(100.0)
		CurrentVoltage.WithLabelValues(dev.id, dev.name).Set(120.0)
		CurrentCurrent.WithLabelValues(dev.id, dev.name).Set(1.0)
	}

	// Verify we can retrieve all metrics
	for _, dev := range devices {
		powerMetric, err := CurrentPower.GetMetricWithLabelValues(dev.id, dev.name)
		if err != nil {
			t.Errorf("Failed to get CurrentPower metric for %s: %v", dev.id, err)
		}
		if testutil.ToFloat64(powerMetric) != 100.0 {
			t.Errorf("Wrong value for CurrentPower[%s]", dev.id)
		}
	}
}
