[comment]: # (Copyright (c) 2025 Darren Soothill)
[comment]: # (Licensed under the MIT License)

# Code Improvements TODO

This document tracks code improvement opportunities identified through comprehensive codebase analysis.

## Project Overview
- **Type**: Go 1.24.0 (toolchain 1.24.8) Matter Power Data Logger
- **LOC**: ~2,507 lines
- **Test Coverage**: 67% average (main: 25.4%, storage: 72.3%, discovery: 93.8%, monitoring: 79.5%, config: 89.8%, metrics: 100%, logger: 90.9%)
- **Purpose**: Discovers Matter devices via mDNS, monitors power consumption, stores in InfluxDB
- **Code Quality**: All GitHub Actions passing (tests, security, lint, build)

---

## Priority Levels

- 🔴 **CRITICAL**: Fix immediately (security/data loss risks)
- 🟠 **HIGH**: Fix soon (testing, reliability, major bugs)
- 🟡 **MEDIUM**: Important improvements (code quality, performance)
- 🟢 **LOW**: Nice to have (features, enhancements)

---

## ✅ COMPLETED

### 1. Implement Actual Matter Protocol Communication
- [x] **File**: `monitoring/power.go`
- [x] **Issue**: The application currently simulates power readings instead of communicating with real Matter devices. This is a critical feature gap that prevents the logger from performing its primary function.
- [x] **Fix**: Replace the mock data generation with a proper Matter client implementation to read power consumption from discovered devices.

### 2. Implement Dynamic Ticker Updates for Monitoring
- [x] **File**: `monitoring/power.go`
- [x] **Issue**: The monitoring interval is set when a device is first discovered, but there is no mechanism to update it if the configuration changes. This can lead to stale monitoring intervals and inconsistent data collection.
- [x] **Fix**: Implement a mechanism to signal `monitorDevice` goroutines to restart or update their tickers when the monitoring configuration is reloaded.

### 3. Add Integration Tests
- [x] **Issue**: Only unit tests exist, no end-to-end tests
- [x] **Fix**: Add integration tests using:
  - Testcontainers for InfluxDB
  - Mock mDNS server
  - Full startup/shutdown cycle

## ✅ COMPLETED

### 4. Add Benchmark Tests
- [x] **Issue**: No performance benchmarks
- [x] **Fix**: Add benchmarks for:
  - Power reading generation
  - InfluxDB write performance
  - Discovery parsing
  - Metrics updates

### 5. Add Fuzz Tests
- [x] **Files**: `discovery/discovery.go`, `storage/influxdb.go`
- [x] **Issue**: No fuzzing for input parsing
- [x] **Fix**: Add Go 1.18+ fuzz tests for:
  - TXT record parsing
  - Device ID generation
  - Flux query sanitization

### 6. Add Error Path Testing
- [x] **Issue**: Many error branches not tested
- [x] **Fix**: Add negative tests forcing error conditions

## 🟡 IN PROGRESS

### 7. Add Race Condition Stress Tests
- [ ] **Issue**: Concurrent access might have undiscovered races
- [ ] **Fix**: Add stress tests with many goroutines

### 8. Create Architecture Decision Records
- [ ] **Issue**: No documentation of design decisions
- [ ] **Fix**: Document decisions like:
  - Why zeroconf library
  - Why async InfluxDB writes
  - Why channel-based architecture

### 9. Generate Metrics Documentation
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Issue**: Metrics not exported in structured way
- [ ] **Fix**: Generate metrics docs for operators

---

## 🟠 HIGH PRIORITY (Newly Identified)

---

## 🟢 LOW PRIORITY (Nice to Have)

### 10. Expand Troubleshooting Guide
- [ ] **File**: `README.md`
- [ ] **Issue**: Minimal troubleshooting section
- [ ] **Fix**: Cover:
  - Firewall issues
  - mDNS permission problems
  - Configuration errors
  - Device discovery verification

### 11. Add Deployment Best Practices Docs
- [ ] **Issue**: No production deployment guidance
- [ ] **Fix**: Document:
  - Resource requirements
  - Scaling considerations
  - Backup/restore
  - Meta-monitoring

### 12. Add Extension Examples
- [ ] **Issue**: No examples for extending the system
- [ ] **Fix**: Add examples for:
  - New storage backends
  - Additional Matter clusters
  - Custom metrics

### 13. Improve Complex Code Comments
- [ ] **File**: `discovery/discovery.go:86-98`
- [ ] **Issue**: Complex concurrent code lacks detailed comments
- [ ] **Fix**: Explain synchronization strategy

### 14. Add Configuration Hot Reload
- [ ] **File**: `config/config.go`
- [ ] **Issue**: Requires restart for config changes
- [ ] **Fix**: Add SIGHUP handler for reload

### 15. Add Configuration Schema
- [ ] **Issue**: No JSON Schema for validation
- [ ] **Fix**: Add schema file and validation

### 16. Support Configuration Profiles
- [ ] **Issue**: Single config, no dev/staging/prod profiles
- [ ] **Fix**: Support multiple config files

### 17. Add Observability Configuration
- [ ] **Issue**: Missing config for metrics interval, log format, tracing
- [ ] **Fix**: Add observability config section

### 18. Add Resource Limits Configuration
- [ ] **Issue**: No limits for max devices, buffer sizes, batch sizes
- [ ] **Fix**: Add resource limits config

### 19. Extract Business Logic from Main
- [ ] **File**: `main.go:147-206`
- [ ] **Issue**: Discovery logic in main function
- [ ] **Fix**: Extract to application/service layer

### 20. Add Graceful Degradation
- [ ] **Issue**: InfluxDB down = silent data loss
- [ ] **Fix**: Add fallback storage or fail loudly

### 21. Enhance Metrics Help Text
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Fix**: Add units and examples to help text

### 22. Define Structured Error Types
- [ ] **Issue**: Errors are strings, not types
- [ ] **Fix**: Define error types:
  ```go
  type DiscoveryError struct {
      Op string
      Err error
  }
  ```

---

## 🌟 FEATURE ENHANCEMENTS

### 23. Add Alerting System
- [ ] **Description**: No alerts when devices offline or readings anomalous
- [ ] **Fix**: Integrate with Alertmanager or add webhook system

### 24. Add Device Discovery Events
- [ ] **Description**: No notification when devices appear/disappear
- [ ] **Fix**: Add event system or webhooks

### 25. Add Historical Data Export
- [ ] **Description**: No export/backup functionality
- [ ] **Fix**: Add CLI command or API for data export

### 26. Add Device Metadata Persistence
- [ ] **Description**: Device names, locations lost on restart
- [ ] **Fix**: Persist metadata (JSON file or database)

### 27. Add Resource Limit Enforcement
- [ ] **Description**: No limits on goroutines, memory, file descriptors
- [ ] **Fix**: Add resource monitoring and limits

### 28. Add Discovery Fallback Mode
- [ ] **Description**: mDNS failure prevents startup
- [ ] **Fix**: Allow running without discovery for testing

### 29. Add Debug Signal Handlers
- [ ] **Description**: No SIGUSR1/SIGUSR2 for debugging
- [ ] **Fix**: Add handlers to dump goroutines, device state

### 30. Add Docker Secrets Support
- [ ] **File**: `docker-compose.yml`
- [ ] **Issue**: Uses environment variables in compose
- [ ] **Fix**: Use Docker secrets or external secret management

### 31. Add Incremental Discovery
- [ ] **File**: `discovery/discovery.go`
- [ ] **Issue**: Full network scan every time
- [ ] **Fix**: Support continuous/incremental discovery

---

## Test Coverage Status

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| main | 25.4% | 80% | 🔶 Reasonable coverage (architectural limits) |
| storage | 72.3% | 85% | 🔶 Close to target (with integration tests) |
| discovery | 93.8% | 85% | ✅ Exceeds target |
| monitoring | 79.5% | 90% | 🔶 Close to target |
| config | 89.8% | 95% | 🔶 Close to target |
| metrics | 100.0% | 100% | ✅ Perfect |
| logger | 90.9% | 100% | 🔶 Close to target |

**Overall Coverage**: 67% (Updated 2025-11-11)

---

## Completion Tracking

- Total Remaining Items: 31
- **High Priority** (🟠): 2 items
- **Low Priority** (🟢): 21 items
- **Feature Enhancements** (🌟): 8 items

### Previously Completed Work Summary:
- **Critical Priority** (🔴): 5/5 completed (100%) ✅
- **High Priority** (🟠): 14/14 completed (100%) ✅
- **Medium Priority** (🟡): 16/16 completed (100%) ✅

All original critical, high, and medium priority tasks have been completed. The codebase is considered production-ready, and the remaining items focus on implementing core functionality (Matter communication) and further enhancements.
