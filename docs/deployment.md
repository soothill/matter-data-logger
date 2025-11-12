# Deployment Best Practices

This document provides guidance for deploying the Matter Data Logger in a production environment.

## Resource Requirements

-   **CPU**: A single vCPU is generally sufficient for monitoring a moderate number of devices (up to 50).
-   **Memory**: 256MB of RAM is a good starting point. Monitor memory usage and adjust as needed.
-   **Disk**: The application itself has a small footprint. Ensure you have enough disk space for the local cache, especially if InfluxDB connectivity is unreliable.

## Scaling Considerations

-   **Horizontal Scaling**: For a large number of devices, you can run multiple instances of the logger. However, be aware that this may result in duplicate data if not managed carefully. A potential strategy is to partition devices by network segment or use a service discovery mechanism to assign devices to specific logger instances.
-   **InfluxDB**: Ensure your InfluxDB instance is properly sized to handle the write load from the logger.

## Backup and Restore

-   **InfluxDB**: Follow the official InfluxDB documentation for backing up and restoring your data.
-   **Local Cache**: The local cache is intended for temporary data storage and should not be considered a long-term backup solution.

## Meta-Monitoring

It is recommended to monitor the health and performance of the Matter Data Logger itself. Key metrics to watch include:

-   `matter_data_logger_app_uptime_seconds`: Monitor the uptime of the application.
-   `matter_data_logger_app_goroutines`: Track the number of goroutines to detect potential leaks.
-   `matter_data_logger_storage_writes_total{success="false"}`: Alert on failed writes to InfluxDB.
-   `matter_data_logger_storage_buffer_size`: Monitor the size of the write buffer to detect potential backpressure.
