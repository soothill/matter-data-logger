# TODO

This file lists potential improvements for the Matter Data Logger.

## Libraries to Extract

- [ ] **Slack Notifier**: The `pkg/notifications/slack.go` file could be extracted into a separate, reusable Go module.
- [ ] **InfluxDB Client**: The `storage/influxdb.go` file could be generalized and moved into its own library.
- [ ] **Local Cache**: The file-based cache in `storage/cache.go` could be a standalone library.
- [ ] **mDNS Scanner**: The discovery logic in `discovery/scanner.go` could be extracted into a generic mDNS scanning library.

## Feature Enhancements

- [ ] **More Notification Channels**: Add support for other notification services like email, PagerDuty, or generic webhooks.
- [ ] **More Storage Backends**: Implement support for other time-series databases like Prometheus, OpenTSDB, or VictoriaMetrics.
- [ ] **Configuration Validation**: Improve the configuration validation to provide more detailed error messages and suggestions.

## Testing

- [ ] **More Comprehensive Tests**: Add more unit and integration tests to improve code coverage and reliability.
