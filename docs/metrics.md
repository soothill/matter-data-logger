# Metrics Documentation

This document provides a reference for the Prometheus metrics exposed by the Matter Data Logger.

## Metric Naming Convention

Metrics follow the standard Prometheus naming convention:

`namespace_subsystem_metric_unit`

- **Namespace**: `matter_data_logger`
- **Subsystem**: The component the metric belongs to (e.g., `discovery`, `monitoring`, `storage`).

---

## Exposed Metrics

### Discovery (`discovery`)

- **`matter_data_logger_discovery_devices_discovered_total`**
  - **Type**: Counter
  - **Description**: The total number of Matter devices discovered since the application started.
  - **Labels**: None

- **`matter_data_logger_discovery_last_scan_duration_seconds`**
  - **Type**: Gauge
  - **Description**: The duration of the last mDNS discovery scan in seconds.
  - **Labels**: None

### Monitoring (`monitoring`)

- **`matter_data_logger_monitoring_power_readings_total`**
  - **Type**: Counter
  - **Description**: The total number of power readings taken from Matter devices.
  - **Labels**:
    - `device_id`: The unique identifier of the device.
    - `success`: `true` if the reading was successful, `false` otherwise.

- **`matter_data_logger_monitoring_active_devices`**
  - **Type**: Gauge
  - **Description**: The current number of devices being actively monitored for power consumption.
  - **Labels**: None

- **`matter_data_logger_monitoring_power_reading_value`**
  - **Type**: Gauge
  - **Description**: The last power reading value in watts.
  - **Labels**:
    - `device_id`: The unique identifier of the device.

### Storage (`storage`)

- **`matter_data_logger_storage_writes_total`**
  - **Type**: Counter
  - **Description**: The total number of write operations to the storage backend (InfluxDB).
  - **Labels**:
    - `success`: `true` if the write was successful, `false` otherwise.

- **`matter_data_logger_storage_write_duration_seconds`**
  - **Type**: Histogram
  - **Description**: A histogram of the duration of write operations to the storage backend.
  - **Labels**: None

- **`matter_data_logger_storage_buffer_size`**
  - **Type**: Gauge
  - **Description**: The current number of data points in the write buffer.
  - **Labels**: None

### Application (`app`)

- **`matter_data_logger_app_uptime_seconds`**
  - **Type**: Gauge
  - **Description**: The uptime of the application in seconds.
  - **Labels**: None

- **`matter_data_logger_app_goroutines`**
  - **Type**: Gauge
  - **Description**: The current number of running goroutines.
  - **Labels**: None

---

## Example Usage

To query the total number of power readings for a specific device:

```promql
sum(matter_data_logger_monitoring_power_readings_total{device_id="your-device-id"})
```

To get the 99th percentile of storage write duration:

```promql
histogram_quantile(0.99, sum(rate(matter_data_logger_storage_write_duration_seconds_bucket[5m])) by (le))
