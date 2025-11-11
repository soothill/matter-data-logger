# Metrics Documentation

This document provides comprehensive documentation for all Prometheus metrics exported by the Matter Power Data Logger. These metrics enable monitoring, alerting, and performance analysis of device discovery, power readings, and data storage operations.

## Table of Contents

- [Overview](#overview)
- [Accessing Metrics](#accessing-metrics)
- [Metric Categories](#metric-categories)
  - [Discovery Metrics](#discovery-metrics)
  - [Power Reading Metrics](#power-reading-metrics)
  - [Storage Metrics](#storage-metrics)
  - [Performance Metrics](#performance-metrics)
  - [Per-Device Metrics](#per-device-metrics)
- [Example Queries](#example-queries)
- [Alerting Rules](#alerting-rules)
- [Dashboard Templates](#dashboard-templates)
- [Cardinality Considerations](#cardinality-considerations)

## Overview

The Matter Power Data Logger exposes metrics on port 9090 at the `/metrics` endpoint (localhost only for security). These metrics follow Prometheus naming conventions and include:

- **Counters**: Monotonically increasing values (e.g., total readings, errors)
- **Gauges**: Point-in-time values that can increase or decrease (e.g., device count, current power)
- **Histograms**: Distribution of observed values (e.g., operation durations)

All metrics use the `matter_` prefix for easy identification.

## Accessing Metrics

```bash
# View all metrics
curl http://localhost:9090/metrics

# Filter for specific metrics
curl http://localhost:9090/metrics | grep matter_

# Check specific metric
curl http://localhost:9090/metrics | grep matter_devices_discovered_total
```

## Metric Categories

### Discovery Metrics

#### `matter_devices_discovered_total`
- **Type**: Gauge
- **Unit**: count
- **Description**: Total number of Matter devices discovered via mDNS, including all device types
- **Use Case**: Monitor overall device discovery health
- **Typical Range**: 0-100 for home networks, 100-1000 for enterprise

**Example Query**:
```promql
# Current number of discovered devices
matter_devices_discovered_total

# Rate of device discovery changes
rate(matter_devices_discovered_total[5m])
```

#### `matter_power_devices_discovered_total`
- **Type**: Gauge
- **Unit**: count
- **Description**: Number of devices with Electrical Measurement (0x0B04) or Power Measurement (0x0091) clusters
- **Use Case**: Track devices actually being monitored for power consumption
- **Typical Range**: 0-50 for home networks

**Example Query**:
```promql
# Percentage of devices with power measurement
(matter_power_devices_discovered_total / matter_devices_discovered_total) * 100
```

#### `matter_devices_monitored`
- **Type**: Gauge
- **Unit**: count
- **Description**: Number of devices currently being actively polled for power readings
- **Use Case**: Verify monitoring is active for all expected devices
- **Typical Range**: Should equal matter_power_devices_discovered_total

**Example Query**:
```promql
# Devices not being monitored (should be 0)
matter_power_devices_discovered_total - matter_devices_monitored
```

### Power Reading Metrics

#### `matter_power_readings_total`
- **Type**: Counter
- **Unit**: count
- **Description**: Total number of power readings successfully collected from devices
- **Use Case**: Monitor data collection rate and overall system health
- **Typical Value**: Increases by (device_count × samples_per_minute × 60) per hour

**Example Query**:
```promql
# Readings per second (overall rate)
rate(matter_power_readings_total[1m])

# Readings per device per second
rate(matter_power_readings_total[1m]) / matter_devices_monitored

# Total readings today
increase(matter_power_readings_total[24h])
```

#### `matter_power_reading_errors_total`
- **Type**: Counter
- **Unit**: count
- **Description**: Total number of failed power reading attempts (timeouts, device errors, network issues)
- **Use Case**: Identify problematic devices or network issues
- **Alert Threshold**: Error rate > 5% requires investigation

**Example Query**:
```promql
# Error rate as percentage
(rate(matter_power_reading_errors_total[5m]) /
 (rate(matter_power_readings_total[5m]) + rate(matter_power_reading_errors_total[5m]))) * 100

# Errors per minute
rate(matter_power_reading_errors_total[1m]) * 60
```

### Storage Metrics

#### `matter_influxdb_writes_total`
- **Type**: Counter
- **Unit**: count
- **Description**: Total number of successful writes to InfluxDB (excludes cached writes during outages)
- **Use Case**: Monitor data persistence health
- **Typical Value**: Should closely track matter_power_readings_total

**Example Query**:
```promql
# Write rate (points per second)
rate(matter_influxdb_writes_total[1m])

# Writes per hour
increase(matter_influxdb_writes_total[1h])

# Write success percentage
(matter_influxdb_writes_total /
 (matter_influxdb_writes_total + matter_influxdb_write_errors_total)) * 100
```

#### `matter_influxdb_write_errors_total`
- **Type**: Counter
- **Unit**: count
- **Description**: Total number of failed InfluxDB write attempts (triggers local cache fallback)
- **Use Case**: Detect InfluxDB connectivity or performance issues
- **Alert Threshold**: Any errors indicate InfluxDB problems

**Example Query**:
```promql
# Recent write errors (last 5 minutes)
increase(matter_influxdb_write_errors_total[5m])

# Error rate
rate(matter_influxdb_write_errors_total[1m])
```

### Performance Metrics

#### `matter_discovery_duration_seconds`
- **Type**: Histogram
- **Unit**: seconds
- **Description**: Duration of mDNS device discovery operations
- **Buckets**: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 seconds
- **Use Case**: Monitor network performance and discovery efficiency
- **Typical Range**: 0.1-5 seconds for healthy networks

**Example Query**:
```promql
# Average discovery duration
rate(matter_discovery_duration_seconds_sum[5m]) /
rate(matter_discovery_duration_seconds_count[5m])

# 95th percentile discovery time
histogram_quantile(0.95, rate(matter_discovery_duration_seconds_bucket[5m]))

# 99th percentile discovery time
histogram_quantile(0.99, rate(matter_discovery_duration_seconds_bucket[5m]))
```

#### `matter_power_reading_duration_seconds`
- **Type**: Histogram
- **Unit**: seconds
- **Description**: Duration of single device power reading operations
- **Buckets**: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 seconds
- **Use Case**: Identify slow devices or network latency issues
- **Typical Range**: 0.001-0.1 seconds for healthy devices

**Example Query**:
```promql
# Average reading time
rate(matter_power_reading_duration_seconds_sum[5m]) /
rate(matter_power_reading_duration_seconds_count[5m])

# 50th percentile (median)
histogram_quantile(0.50, rate(matter_power_reading_duration_seconds_bucket[5m]))

# 99th percentile (slow readings)
histogram_quantile(0.99, rate(matter_power_reading_duration_seconds_bucket[5m]))

# Percentage of readings taking > 100ms
sum(rate(matter_power_reading_duration_seconds_bucket{le="0.1"}[5m])) /
sum(rate(matter_power_reading_duration_seconds_count[5m])) * 100
```

### Per-Device Metrics

**Cardinality Warning**: These metrics create one time series per device. See [Cardinality Considerations](#cardinality-considerations) for details.

#### `matter_current_power_watts`
- **Type**: Gauge
- **Unit**: watts (W)
- **Labels**: `device_id`, `device_name`
- **Description**: Current power consumption per device
- **Typical Range**: 0-5000W for household devices
- **Use Case**: Real-time power monitoring, anomaly detection, usage trends

**Example Query**:
```promql
# Top 5 power consumers
topk(5, matter_current_power_watts)

# Total household power consumption
sum(matter_current_power_watts)

# Average power per device
avg(matter_current_power_watts)

# Devices consuming > 1000W
matter_current_power_watts > 1000

# Power consumption by device (for specific device)
matter_current_power_watts{device_id="A1B2C3D4"}

# Devices with significant power change (> 100W in 5 min)
abs(delta(matter_current_power_watts[5m])) > 100
```

#### `matter_current_voltage_volts`
- **Type**: Gauge
- **Unit**: volts (V)
- **Labels**: `device_id`, `device_name`
- **Description**: Current voltage per device
- **Typical Range**: 110-240V AC depending on region
- **Use Case**: Monitor power quality, detect voltage fluctuations

**Example Query**:
```promql
# Average voltage across all devices
avg(matter_current_voltage_volts)

# Voltage range (max - min)
max(matter_current_voltage_volts) - min(matter_current_voltage_volts)

# Devices with low voltage (< 110V for US)
matter_current_voltage_volts < 110

# Voltage stability (standard deviation)
stddev(matter_current_voltage_volts)
```

#### `matter_current_amperage_amps`
- **Type**: Gauge
- **Unit**: amps (A)
- **Labels**: `device_id`, `device_name`
- **Description**: Current amperage (current draw) per device
- **Typical Range**: 0-20A for household devices
- **Use Case**: Monitor current draw, detect overload conditions

**Example Query**:
```promql
# Top current consumers
topk(5, matter_current_amperage_amps)

# Total current draw
sum(matter_current_amperage_amps)

# Devices drawing > 15A (approaching circuit limit)
matter_current_amperage_amps > 15

# Power factor (Power / (Voltage × Current)) for a device
matter_current_power_watts{device_id="ABC123"} /
(matter_current_voltage_volts{device_id="ABC123"} *
 matter_current_amperage_amps{device_id="ABC123"})
```

## Example Queries

### System Health

```promql
# Overall system health check
up{job="matter-data-logger"}

# Application uptime
time() - process_start_time_seconds

# Discovery success rate (should be close to 100%)
(1 - (rate(matter_power_reading_errors_total[5m]) /
      (rate(matter_power_readings_total[5m]) +
       rate(matter_power_reading_errors_total[5m])))) * 100
```

### Data Collection

```promql
# Data points per second
rate(matter_power_readings_total[1m])

# Expected vs actual readings (assuming 1 reading/sec/device)
rate(matter_power_readings_total[1m]) / matter_devices_monitored

# Data loss (if writing to InfluxDB fails)
increase(matter_power_reading_errors_total[1h]) +
increase(matter_influxdb_write_errors_total[1h])
```

### Power Analysis

```promql
# Total power consumption (all devices)
sum(matter_current_power_watts)

# Average power per device
avg(matter_current_power_watts)

# Power consumption trend (last hour)
delta(sum(matter_current_power_watts)[1h:1m])

# Daily energy consumption (kWh estimate)
sum(avg_over_time(matter_current_power_watts[24h])) / 1000 * 24

# Peak power consumption (last 24h)
max_over_time(sum(matter_current_power_watts)[24h:1m])
```

### Device Discovery

```promql
# New devices discovered recently
delta(matter_devices_discovered_total[1h])

# Percentage of devices with power measurement
(matter_power_devices_discovered_total / matter_devices_discovered_total) * 100

# Discovery performance degradation
rate(matter_discovery_duration_seconds_sum[5m]) /
rate(matter_discovery_duration_seconds_count[5m]) > 5
```

## Alerting Rules

Below are recommended Prometheus alerting rules for production deployments.

```yaml
groups:
  - name: matter_data_logger
    interval: 30s
    rules:
      # Critical: Application down
      - alert: MatterDataLoggerDown
        expr: up{job="matter-data-logger"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Matter Data Logger is down"
          description: "The Matter Data Logger has been unreachable for 2 minutes."

      # Critical: No devices discovered
      - alert: NoDevicesDiscovered
        expr: matter_devices_discovered_total == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No Matter devices discovered"
          description: "No devices have been discovered for 5 minutes. Check network and mDNS configuration."

      # Critical: High error rate
      - alert: HighPowerReadingErrorRate
        expr: |
          (rate(matter_power_reading_errors_total[5m]) /
           (rate(matter_power_readings_total[5m]) +
            rate(matter_power_reading_errors_total[5m]))) * 100 > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High power reading error rate ({{ $value | humanize }}%)"
          description: "Power reading error rate is above 10% for 5 minutes. Check device connectivity."

      # Critical: InfluxDB write failures
      - alert: InfluxDBWriteFailures
        expr: rate(matter_influxdb_write_errors_total[5m]) > 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "InfluxDB write failures detected"
          description: "InfluxDB writes are failing. Check InfluxDB connectivity and health. Data is being cached locally."

      # Warning: Device disappeared
      - alert: DeviceCountDecreased
        expr: delta(matter_devices_discovered_total[10m]) < -1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Device count decreased by {{ $value }}"
          description: "One or more devices disappeared from the network. Check device power and network connectivity."

      # Warning: Slow discovery
      - alert: SlowDeviceDiscovery
        expr: |
          histogram_quantile(0.95,
            rate(matter_discovery_duration_seconds_bucket[5m])) > 10
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Device discovery is slow ({{ $value | humanizeDuration }})"
          description: "95th percentile discovery time exceeds 10 seconds. Check network performance."

      # Warning: Slow power readings
      - alert: SlowPowerReadings
        expr: |
          histogram_quantile(0.95,
            rate(matter_power_reading_duration_seconds_bucket[5m])) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Power readings are slow ({{ $value | humanizeDuration }})"
          description: "95th percentile reading time exceeds 1 second. Check device and network performance."

      # Warning: No data collection
      - alert: NoDataCollection
        expr: rate(matter_power_readings_total[5m]) == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "No power readings collected"
          description: "No power readings have been collected for 5 minutes despite having monitored devices."

      # Info: High power consumption
      - alert: HighTotalPowerConsumption
        expr: sum(matter_current_power_watts) > 10000
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "High total power consumption ({{ $value | humanize }}W)"
          description: "Total power consumption exceeds 10kW. Review device usage."

      # Info: Unusual power spike
      - alert: PowerConsumptionSpike
        expr: |
          abs(delta(sum(matter_current_power_watts)[5m])) > 2000
        for: 1m
        labels:
          severity: info
        annotations:
          summary: "Power consumption spike detected ({{ $value | humanize }}W change)"
          description: "Power consumption changed by more than 2kW in 5 minutes."
```

## Dashboard Templates

### Grafana Dashboard JSON

Basic dashboard layout for visualizing Matter device metrics:

```json
{
  "dashboard": {
    "title": "Matter Power Data Logger",
    "panels": [
      {
        "title": "Total Power Consumption",
        "targets": [
          {"expr": "sum(matter_current_power_watts)"}
        ],
        "type": "graph"
      },
      {
        "title": "Devices Discovered",
        "targets": [
          {"expr": "matter_devices_discovered_total"},
          {"expr": "matter_power_devices_discovered_total"},
          {"expr": "matter_devices_monitored"}
        ],
        "type": "stat"
      },
      {
        "title": "Power by Device",
        "targets": [
          {"expr": "matter_current_power_watts", "legendFormat": "{{device_name}}"}
        ],
        "type": "graph"
      },
      {
        "title": "Data Collection Rate",
        "targets": [
          {"expr": "rate(matter_power_readings_total[1m])"}
        ],
        "type": "graph"
      },
      {
        "title": "Error Rates",
        "targets": [
          {"expr": "rate(matter_power_reading_errors_total[1m])", "legendFormat": "Reading Errors"},
          {"expr": "rate(matter_influxdb_write_errors_total[1m])", "legendFormat": "Write Errors"}
        ],
        "type": "graph"
      },
      {
        "title": "Discovery Performance",
        "targets": [
          {"expr": "histogram_quantile(0.95, rate(matter_discovery_duration_seconds_bucket[5m]))"}
        ],
        "type": "gauge"
      }
    ]
  }
}
```

### Key Metrics for Operators

**First 5 minutes after deployment**:
1. `matter_devices_discovered_total` - Should be > 0
2. `matter_power_devices_discovered_total` - Should match expected count
3. `matter_devices_monitored` - Should equal power devices count
4. `rate(matter_power_readings_total[1m])` - Should be > 0
5. `matter_influxdb_write_errors_total` - Should remain 0

**Ongoing monitoring**:
1. `sum(matter_current_power_watts)` - Total power consumption
2. Error rates - Should be < 5%
3. Discovery duration - Should be < 5 seconds
4. Reading duration - Should be < 0.1 seconds

## Cardinality Considerations

### What is Cardinality?

Cardinality in Prometheus refers to the number of unique time series. Each unique combination of metric name and label values creates a separate time series. High cardinality can impact Prometheus performance and memory usage.

### Cardinality Impact

The Matter Power Data Logger uses device labels for per-device metrics:

- `matter_current_power_watts{device_id="A", device_name="Light"}` - 1 series
- `matter_current_power_watts{device_id="B", device_name="Fan"}` - 1 series
- **Total**: 3 metrics × number of devices

**Cardinality Growth**:
- 10 devices: 30 time series
- 100 devices: 300 time series
- 1,000 devices: 3,000 time series
- 10,000 devices: 30,000 time series

### Recommendations

**For Home Networks (< 100 devices)**:
- No action needed - cardinality is low

**For Small Enterprise (100-1,000 devices)**:
- Monitor Prometheus memory usage
- Consider 30-day retention limit
- Use recording rules for aggregated queries

**For Large Enterprise (> 1,000 devices)**:
- Consider removing `device_name` label (use only `device_id`)
- Implement aggregation at ingestion time
- Use Prometheus federation or remote storage
- Group devices by room/floor and aggregate

**Check Current Cardinality**:
```bash
# Count current time series
curl -s localhost:9090/metrics | grep -E '^matter_current_' | wc -l

# View specific device metrics
curl -s localhost:9090/metrics | grep matter_current_power_watts
```

**Reduce Cardinality** (if needed):
1. Modify `pkg/metrics/metrics.go` to remove `device_name` label
2. Use device ID only for tracking
3. Maintain device name mapping in external system

### Best Practices

1. **Use Recording Rules** for common aggregations:
```yaml
groups:
  - name: matter_aggregations
    interval: 30s
    rules:
      - record: matter:total_power:sum
        expr: sum(matter_current_power_watts)

      - record: matter:avg_power:avg
        expr: avg(matter_current_power_watts)
```

2. **Set Retention Policies**:
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

storage:
  tsdb:
    retention.time: 30d
    retention.size: 50GB
```

3. **Monitor Prometheus Health**:
```promql
# Time series count
prometheus_tsdb_symbol_table_size_bytes

# Memory usage
process_resident_memory_bytes{job="prometheus"}

# Head cardinality
prometheus_tsdb_head_series
```

## Additional Resources

- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [PromQL Query Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Gallery](https://grafana.com/grafana/dashboards/)
- [Matter Protocol Specification](https://csa-iot.org/all-solutions/matter/)

---

**Last Updated**: 2025-11-11
**Maintainer**: Matter Data Logger Team
**Version**: 1.0
