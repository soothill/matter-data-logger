// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package metrics provides Prometheus instrumentation for monitoring Matter device
// discovery, power readings, and InfluxDB storage operations. All metrics are
// automatically registered with Prometheus and exposed via the /metrics endpoint.
//
// The metrics include counters for tracking total operations and errors, gauges
// for current device counts and readings, histograms for operation durations,
// and gauge vectors for per-device power measurements.
//
// # Cardinality Considerations
//
// Several metrics use labels to track per-device measurements (device_id, device_name).
// Each unique combination of label values creates a new time series in Prometheus.
//
// Cardinality calculation:
//   - CurrentPower: 1 time series per device
//   - CurrentVoltage: 1 time series per device
//   - CurrentCurrent: 1 time series per device
//   - Total: 3 × number_of_devices time series
//
// Example cardinality growth:
//   - 10 devices: 30 time series
//   - 100 devices: 300 time series
//   - 1,000 devices: 3,000 time series
//   - 10,000 devices: 30,000 time series
//
// Cardinality limits:
//   - The application is designed for small to medium Matter home networks
//   - Typical home has 10-100 Matter devices
//   - Enterprise deployments should consider label reduction or aggregation
//
// Best practices for high-cardinality scenarios:
//   - Use aggregation queries in Prometheus (sum, avg by room/floor)
//   - Implement metric retention policies to drop old time series
//   - Consider removing device_name label if device_id is sufficient
//   - Use Prometheus recording rules to pre-aggregate data
//   - Monitor Prometheus memory usage and time series count
//
// To check current cardinality:
//
//	curl http://localhost:9090/metrics | grep matter_current_ | wc -l
//
// Note: device_name label is included for human-readable dashboards but
// increases cardinality. For very large deployments (1000+ devices),
// consider removing it and using device_id only.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DevicesDiscovered tracks the total number of Matter devices discovered
	DevicesDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_devices_discovered_total",
		Help: "Total number of Matter devices discovered via mDNS (count, includes all device types)",
	})

	// PowerDevicesDiscovered tracks the number of devices with power measurement capability
	PowerDevicesDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_power_devices_discovered_total",
		Help: "Total number of Matter devices with Electrical Measurement (0x0B04) or Power Measurement (0x0091) clusters (count)",
	})

	// DevicesMonitored tracks the number of devices currently being monitored
	DevicesMonitored = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "matter_devices_monitored",
		Help: "Number of devices currently being actively monitored for power consumption (count, actively polling)",
	})

	// PowerReadingsTotal tracks the total number of power readings collected
	PowerReadingsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_power_readings_total",
		Help: "Total number of power readings successfully collected from devices (count, monotonically increasing)",
	})

	// PowerReadingErrors tracks the number of failed power readings
	PowerReadingErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_power_reading_errors_total",
		Help: "Total number of failed power reading attempts (count, includes timeouts and device errors)",
	})

	// InfluxDBWritesTotal tracks the total number of writes to InfluxDB
	InfluxDBWritesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_influxdb_writes_total",
		Help: "Total number of successful writes to InfluxDB (count, excludes cached writes during outages)",
	})

	// InfluxDBWriteErrors tracks the number of failed writes to InfluxDB
	InfluxDBWriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matter_influxdb_write_errors_total",
		Help: "Total number of failed InfluxDB write attempts (count, triggers local cache fallback)",
	})

	// DiscoveryDuration tracks how long device discovery takes
	DiscoveryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "matter_discovery_duration_seconds",
		Help:    "Duration of mDNS device discovery operation in seconds (histogram with buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)",
		Buckets: prometheus.DefBuckets,
	})

	// PowerReadingDuration tracks how long it takes to read power from a device
	PowerReadingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "matter_power_reading_duration_seconds",
		Help:    "Duration of single device power reading operation in seconds (histogram, typical range: 0.001-0.1s)",
		Buckets: prometheus.DefBuckets,
	})

	// CurrentPower tracks the current power consumption per device.
	//
	// Cardinality Warning: Creates 1 time series per device.
	// Labels:
	//   - device_id: Unique device identifier (required)
	//   - device_name: Human-readable device name (increases cardinality)
	//
	// For deployments with >1000 devices, consider using device_id only
	// and looking up names in a separate system.
	CurrentPower = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_power_watts",
		Help: "Current power consumption per device in watts (W). Labels: device_id, device_name. Typical range: 0-5000W for household devices. High cardinality: 1 series per device.",
	}, []string{"device_id", "device_name"})

	// CurrentVoltage tracks the current voltage per device.
	//
	// Cardinality Warning: Creates 1 time series per device.
	// Labels:
	//   - device_id: Unique device identifier (required)
	//   - device_name: Human-readable device name (increases cardinality)
	CurrentVoltage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_voltage_volts",
		Help: "Current voltage per device in volts (V). Labels: device_id, device_name. Typical range: 110-240V AC. High cardinality: 1 series per device.",
	}, []string{"device_id", "device_name"})

	// CurrentCurrent tracks the current current (amperage) per device.
	//
	// Cardinality Warning: Creates 1 time series per device.
	// Labels:
	//   - device_id: Unique device identifier (required)
	//   - device_name: Human-readable device name (increases cardinality)
	CurrentCurrent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "matter_current_amperage_amps",
		Help: "Current amperage per device in amps (A). Labels: device_id, device_name. Typical range: 0-20A for household devices. High cardinality: 1 series per device.",
	}, []string{"device_id", "device_name"})
)
