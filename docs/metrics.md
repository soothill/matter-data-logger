# Metrics

This document describes the Prometheus metrics exposed by the Matter Data Logger.

## Gauges

*   `matter_devices_discovered_total`: Total number of Matter devices discovered via mDNS.
*   `matter_power_devices_discovered_total`: Total number of Matter devices with power measurement capability.
*   `matter_devices_monitored`: Number of devices currently being actively monitored for power consumption.

## Counters

*   `matter_power_readings_total`: Total number of power readings successfully collected from devices.
*   `matter_power_reading_errors_total`: Total number of failed power reading attempts.
*   `matter_influxdb_writes_total`: Total number of successful writes to InfluxDB.
*   `matter_influxdb_write_errors_total`: Total number of failed InfluxDB write attempts.

## Histograms

*   `matter_discovery_duration_seconds`: Duration of mDNS device discovery operation in seconds.
*   `matter_power_reading_duration_seconds`: Duration of single device power reading operation in seconds.

## Gauge Vectors

*   `matter_current_power_watts`: Current power consumption per device in watts (W).
*   `matter_current_voltage_volts`: Current voltage per device in volts (V).
*   `matter_current_amperage_amps`: Current amperage per device in amps (A).
