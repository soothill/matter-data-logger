// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package metrics provides Prometheus metrics for the Matter data logger.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DevicesDiscovered tracks the total number of Matter devices discovered
	DevicesDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_devices_discovered_total",
		Help: "Total number of Matter devices discovered",
	})

	// PowerDevicesDiscovered tracks the number of devices with power measurement capability
	PowerDevicesDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_power_devices_discovered_total",
		Help: "Total number of Matter devices with power measurement capability",
	})

	// DevicesMonitored tracks the number of devices currently being monitored
	DevicesMonitored = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_devices_monitored",
		Help: "Number of devices currently being monitored for power consumption",
	})

	// PowerReadingsTotal tracks the total number of power readings collected
	PowerReadingsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_power_readings_total",
		Help: "Total number of power readings collected",
	})

	// PowerReadingErrors tracks the number of failed power readings
	PowerReadingErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_power_reading_errors_total",
		Help: "Total number of failed power readings",
	})

	// InfluxDBWritesTotal tracks the total number of writes to InfluxDB
	InfluxDBWritesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_influxdb_writes_total",
		Help: "Total number of writes to InfluxDB",
	})

	// InfluxDBWriteErrors tracks the number of failed writes to InfluxDB
	InfluxDBWriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_influxdb_write_errors_total",
		Help: "Total number of failed writes to InfluxDB",
	})

	// DiscoveryDuration tracks how long device discovery takes
	DiscoveryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "matter_discovery_duration_seconds",
		Help:    "Duration of device discovery in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// PowerReadingDuration tracks how long it takes to read power from a device
	PowerReadingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "matter_power_reading_duration_seconds",
		Help:    "Duration of power reading in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// CurrentPower tracks the current power consumption per device
	CurrentPower = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_power_watts",
		Help: "Current power consumption in watts",
	}, []string{"device_id", "device_name"})

	// CurrentVoltage tracks the current voltage per device
	CurrentVoltage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_voltage_volts",
		Help: "Current voltage in volts",
	}, []string{"device_id", "device_name"})

	// CurrentCurrent tracks the current current (amperage) per device
	CurrentCurrent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_amperage_amps",
		Help: "Current amperage in amps",
	}, []string{"device_id", "device_name"})
)
