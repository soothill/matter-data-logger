[comment]: # (Copyright (c) 2025 Darren Soothill)
[comment]: # (Licensed under the MIT License)

# Code Improvements TODO

This document tracks code improvement opportunities identified through comprehensive codebase analysis.

## Project Overview
- **Type**: Go 1.24.0 (toolchain 1.24.8) Matter Power Data Logger
- **LOC**: ~2,507 lines
- **Test Coverage**: 67% average (main: 2.2%, storage: 8.1%, discovery: 93.8%, monitoring: 79.5%)
- **Purpose**: Discovers Matter devices via mDNS, monitors power consumption, stores in InfluxDB

---

## Priority Levels

- 🔴 **CRITICAL**: Fix immediately (security/data loss risks)
- 🟠 **HIGH**: Fix soon (testing, reliability, major bugs)
- 🟡 **MEDIUM**: Important improvements (code quality, performance)
- 🟢 **LOW**: Nice to have (features, enhancements)

---

## 🔴 CRITICAL PRIORITY

### 1. ✅ Remove Hardcoded Secrets (SECURITY) - COMPLETED
- [x] **File**: `docker-compose.yml:20`
- [x] **Issue**: Contains hardcoded `my-super-secret-auth-token` and `adminpassword`
- [x] **Risk**: Secrets in version control, security vulnerability
- [x] **Fix**: Completed in commit `968cb22`
  - Removed hardcoded secrets
  - Added `.env` file with docker-compose
  - Added warning comment about secrets management
  - Documented how to generate random tokens

### 2. ✅ Update Security-Critical Dependencies - COMPLETED
- [x] **Files**: `go.mod`
- [x] **Issue**: Outdated `golang.org/x/crypto v0.21.0` and `golang.org/x/net v0.23.0`
- [x] **Risk**: Known security vulnerabilities
- [x] **Fix**: Completed in commit `968cb22`
  - Updated `golang.org/x/crypto` to v0.43.0
  - Updated `golang.org/x/net` to v0.46.0

### 3. ✅ Add Retry Logic for InfluxDB Writes - COMPLETED
- [x] **File**: `storage/influxdb.go`
- [x] **Issue**: Failed writes are logged but not retried
- [x] **Risk**: Data loss during temporary network issues
- [x] **Fix**: Completed in commit `968cb22` - Implemented exponential backoff retry logic for transient failures

### 4. ✅ Enforce TLS for Production InfluxDB - COMPLETED
- [x] **File**: `config/config.go`
- [x] **Issue**: No validation warning for HTTP vs HTTPS URLs
- [x] **Risk**: Credentials transmitted in plaintext
- [x] **Fix**: Completed in commit `968cb22` - Added TLS validation for production connections

### 5. ✅ Add Vulnerability Scanning to CI/CD - COMPLETED
- [x] **Files**: `.github/workflows/ci.yml`
- [x] **Issue**: No automated CVE scanning
- [x] **Risk**: Security vulnerabilities go unnoticed
- [x] **Fix**: Completed in commit `968cb22` - Added `govulncheck` step to CI pipeline

---

## 🟠 HIGH PRIORITY

### 6. ✅ Add Tests for Main Package - COMPLETED (23.3% Coverage)
- [x] **File**: `main.go`, `main_test.go`
- [x] **Issue**: No tests for application initialization, signal handling, graceful shutdown
- [x] **Progress**: Completed in commits `c09c86c`, `922b94d`, `13a7bdd`:
  - ✅ Health check endpoint (TestHealthCheckHandler)
  - ✅ Readiness check endpoint (TestReadinessCheckHandler_NoInfluxDB, TestReadinessCheckHandler_Healthy)
  - ✅ Health check function (TestPerformHealthCheck)
  - ✅ Graceful shutdown (TestPerformGracefulShutdown)
  - ✅ Cleanup function (TestPerformCleanup)
  - ✅ Initial discovery (TestPerformInitialDiscovery_NoDevices)
  - ✅ Periodic discovery (TestPerformPeriodicDiscovery_NoDevices)
  - ✅ Component initialization (TestInitializeComponents, TestInitializeComponents_WithSlackWebhook)
  - ✅ Configuration loading (TestMain_ConfigFileHandling)
  - ✅ Refactored initializeComponents() to return errors for better testability
- **Result**: Coverage improved from 0% → 23.3%
- **Note**: Remaining gaps are architecturally difficult to test:
  - main() function (runs in infinite loop)
  - Complete signal handling (requires process signals)
  - Full integration startup flow (requires real InfluxDB)

### 7. ✅ Improve Storage Package Test Coverage - COMPLETED
- [x] **Files**: `storage/influxdb.go`, `storage/influxdb_test.go`, `storage/influxdb_integration_test.go`
- [x] **Issue**: Critical data persistence layer poorly tested (was 8.1%)
- [x] **Fix**: Completed - Added comprehensive integration tests achieving 72.3% coverage:
  - ✅ Actual write operations with real InfluxDB container
  - ✅ Connection and health checks
  - ✅ Validation error handling (nil, empty fields, negative values)
  - ✅ Batch write operations
  - ✅ Query operations (QueryLatestReading)
  - ✅ Close/Flush idempotency
  - ✅ Client accessor method
  - ✅ Context handling with timeouts
  - ✅ Added testcontainers-go for integration testing
  - ✅ Fixed Close() bug (double-close panic)
  - ✅ Added Makefile targets: `make test-integration`, `make test-integration-coverage`
- **Result**: Coverage improved from 8.1% to 72.3% (8.9x improvement!)
- **Note**: Remaining gaps (28%) are in retry logic and error handling goroutines which are difficult to test reliably

### 8. ✅ Improve Discovery Package Test Coverage - COMPLETED
- [x] **File**: `discovery/discovery.go`, `discovery/discovery_test.go`
- [x] **Issue**: Core functionality insufficiently tested
- [x] **Fix**: Completed in commit `c09c86c` - Comprehensive tests added achieving 93.8% coverage (exceeds 85% target):
  - ✅ mDNS discovery with timeout and cancellation
  - ✅ Service entry parsing edge cases
  - ✅ Power measurement cluster detection
  - ✅ Device ID generation with fallbacks
  - ✅ IPv6 address handling
  - ✅ Multiple discovery runs
  - ✅ Device filtering and retrieval

### 9. ✅ Add Input Validation for Power Readings - COMPLETED
- [x] **File**: `storage/influxdb.go` (WriteReading method)
- [x] **Issue**: Accepts negative power/voltage/current values
- [x] **Fix**: Completed in commit `968cb22` - Added validation to reject negative values

### 10. ✅ Add Comprehensive Configuration Validation - COMPLETED
- [x] **File**: `config/config.go`
- [x] **Issue**: Missing validation for URLs, intervals, token formats
- [x] **Fix**: Completed in commit `078afed`
  - URL format parsing
  - Poll interval maximum limits
  - Environment variable parsing error logging
  - Token format validation

### 11. ✅ Fix Unbuffered Discovery Channel - COMPLETED
- [x] **File**: `discovery/discovery.go:77-109`
- [x] **Issue**: `entries` channel unbuffered, could block zeroconf resolver
- [x] **Fix**: Completed in commit `968cb22` - Channel now buffered with size 10

### 12. ✅ Implement HTTP Server Graceful Shutdown - COMPLETED
- [x] **File**: `main.go:245-263`
- [x] **Issue**: HTTP server didn't shut down gracefully, could terminate mid-request
- [x] **Fix**: Completed in commit `fcec935`
  - Added `server.Shutdown()` with 5-second timeout
  - Extracted `performGracefulShutdown()` helper function
  - Integrated with signal handling (SIGTERM/SIGINT)
  - Waits for in-flight requests before stopping

### 12a. ✅ Fix PowerMonitor Channel Closure - COMPLETED
- [x] **File**: `monitoring/power.go:207-229`
- [x] **Issue**: Readings channel never closed, data writer goroutine blocked forever
- [x] **Risk**: Application hangs on shutdown, requires force kill
- [x] **Fix**: Completed in commit `fcec935`
  - Added `Stop()` method that closes readings channel
  - Cancels all device monitoring goroutines
  - Waits for all goroutines to finish before closing channel
  - Prevents double-stop with `stopped` flag

### 12b. ✅ Add InfluxDB Flush Timeout - COMPLETED
- [x] **File**: `main.go:265-290`
- [x] **Issue**: InfluxDB flush could block indefinitely on shutdown
- [x] **Fix**: Completed in commit `fcec935`
  - Wrapped flush in goroutine with 10-second timeout
  - Logs warning if timeout occurs
  - Extracted `performCleanup()` helper function
  - Prevents infinite wait during shutdown

### 12c. ✅ Fix Discovery Nil Slice Return - COMPLETED
- [x] **File**: `discovery/discovery.go:79`
- [x] **Issue**: `Discover()` returned nil instead of empty slice when no devices found
- [x] **Fix**: Completed in commit `fcec935`
  - Changed `var discoveredDevices []*Device` to `make([]*Device, 0)`
  - Consistent with Go best practices (return empty slice, not nil)
  - Fixed failing test `TestScanner_Discover_MultipleRuns`

### 12d. ✅ Fix Code Quality Issues - COMPLETED
- [x] **Files**: `config/config_test.go`, `discovery/discovery_test.go`, `monitoring/power_test.go`
- [x] **Issue**: Variable shadowing and staticcheck warnings
- [x] **Fix**: Completed in commit `fcec935`
  - Fixed 4 instances of variable shadowing in config tests
  - Removed unnecessary nil checks in discovery tests
  - Added explicit returns after Fatal() calls to satisfy staticcheck
  - All linter warnings resolved

### 12e. ✅ Define Interfaces for External Dependencies - COMPLETED
- [x] **Files**: `pkg/interfaces/storage.go`, `pkg/interfaces/discovery.go`, `pkg/interfaces/monitoring.go`
- [x] **Issue**: Direct dependencies on concrete types (InfluxDB, zeroconf)
- [x] **Impact**: Hard to mock, tight coupling
- [x] **Fix**: Completed in commit `c09c86c` - Created comprehensive interface package:
  - ✅ `TimeSeriesStorage` interface (storage.go) - defines storage contract
  - ✅ `DeviceScanner` and `DeviceCapabilities` interfaces (discovery.go) - defines discovery contract
  - ✅ `PowerMonitor` interface (monitoring.go) - defines monitoring contract
  - ✅ Includes `Device` and `PowerReading` types to avoid circular dependencies
  - ✅ Full method documentation for implementers
  - ✅ Foundation for better unit testing and mocking

### 13. ✅ Fix Goroutine Lifecycle Management - COMPLETED
- [x] **Files**: `main.go`, `monitoring/power.go`
- [x] **Issue**: Goroutines not properly tracked, could leak on shutdown
- [x] **Fix**: Completed in commit `fcec935`
  - Added `sync.WaitGroup` to track all goroutines
  - PowerMonitor.Stop() method ensures all monitoring goroutines finish
  - Data writer and HTTP server goroutines properly tracked
  - Graceful shutdown waits for all goroutines to complete

### 14. ✅ Add Context to Async Operations - COMPLETED
- [x] **File**: `storage/influxdb.go`, `storage/cache.go`, `main.go`
- [x] **Issue**: `WriteReading` didn't accept context for cancellation
- [x] **Fix**: Completed in commit `59f8d7c`
  - Added context parameter to WriteReading() and WriteBatch()
  - Added context checks for early cancellation
  - Propagated context through CachingStorage wrapper
  - Updated all call sites in main.go to pass context
  - Enables proper timeout control and cancellation

---

## 🟡 MEDIUM PRIORITY

### 15. ✅ Extract Magic Numbers to Constants - COMPLETED
- [x] **Files**: `main.go`, `monitoring/power.go`
- [x] **Issue**: Hard-coded buffer sizes (100), timeouts (10s, 5s, 2s)
- [x] **Fix**: Completed in commit `2581011`
  - main.go constants: signalChannelSize, discoveryTimeout, alertContextTimeout, readinessCheckTimeout, shutdownTimeout, flushTimeout
  - monitoring/power.go constants: readingsChannelSize, simulatedBaseLoadMin, simulatedLoadRange, simulatedVariation, simulatedBaseVoltage, simulatedVoltageVar
  - All magic numbers replaced with named constants
  - Improved code maintainability and readability

### 16. ✅ Fix parseLogLevel Error Handling - COMPLETED
- [x] **File**: `pkg/logger/logger.go`
- [x] **Issue**: Returned nil instead of error when parsing failed
- [x] **Fix**: Completed in commit `b971ac7`
  - Created errInvalidLogLevel error variable
  - parseLogLevel() now returns error for invalid levels
  - Empty string treated as valid (defaults to info without warning)
  - Initialize() logs warning with structured fields when invalid level provided
  - Example: "WRN Invalid log level, defaulting to info invalid_level=invalid using=info"

### 17. ✅ Initialize Global Logger Safely - COMPLETED
- [x] **File**: `pkg/logger/logger.go`
- [x] **Issue**: Global `log` not initialized, could panic before Initialize()
- [x] **Fix**: Completed in commit `3fc2756`
  - Added init() function with safe default initialization
  - Logger defaults to info level writing to stdout
  - Prevents panics if logger functions called before Initialize()
  - Logger reconfigured when Initialize() called with custom settings

### 18. Add Rate Limiting on Health Endpoints
- [ ] **Files**: `main.go:64-76`, `main.go:209-235`
- [ ] **Issue**: No rate limiting, could be DoS target
- [ ] **Fix**: Add rate limiting middleware using `golang.org/x/time/rate`

### 19. ✅ Secure Metrics Endpoint - COMPLETED
- [x] **File**: `main.go:76`
- [x] **Issue**: Prometheus metrics exposed to network without auth
- [x] **Risk**: Could leak device information to external networks
- [x] **Fix**: Completed in commit `fcec935`
  - HTTP server now binds to `localhost:9090` instead of `:9090`
  - Metrics only accessible from local machine
  - Added comments explaining security rationale
  - For external access, users must configure reverse proxy with auth

### 20. Improve Flux Query Safety
- [ ] **File**: `storage/influxdb.go:150-176`
- [ ] **Issue**: String concatenation for queries, injection risk remains
- [ ] **Fix**: Use parameterized queries or InfluxDB query builder API

### 21. Add Circuit Breaker for InfluxDB
- [ ] **File**: `storage/influxdb.go`
- [ ] **Issue**: Continues writing even when InfluxDB consistently fails
- [ ] **Fix**: Implement circuit breaker (e.g., `github.com/sony/gobreaker`)

### 22. ✅ Use Consistent Error Wrapping - COMPLETED
- [x] **Files**: Multiple
- [x] **Issue**: Inconsistent use of `%w` vs `%v` in error formatting
- [x] **Fix**: Completed in commit `bfa4003`
  - Fixed storage/cache.go:323 to use `%w` for both errors in combined error message
  - Ensures proper error chain unwrapping with errors.Is() and errors.As()
  - All tests passing, linter clean

### 23. ✅ Add Context Checks in Monitoring Loops - COMPLETED
- [x] **File**: `monitoring/power.go:124-149`, `main.go`, `storage/cache.go`
- [x] **Issue**: Doesn't check context before expensive operations
- [x] **Fix**: Completed in commit `e67fa94`
  - Added context checks after ticker fires in monitoring/power.go
  - Added context checks before periodic discovery in main.go
  - Added context checks before health check operations in storage/cache.go
  - Improves graceful shutdown responsiveness by exiting immediately when context is cancelled

### 24. ✅ Fix Device Name Staleness - COMPLETED
- [x] **File**: `monitoring/power.go`, `discovery/discovery.go`, `pkg/interfaces/discovery.go`
- [x] **Issue**: Device name copied at start, won't update if renamed
- [x] **Fix**: Completed in commit `43426f8`
  - Added GetDeviceByID() method to Scanner
  - PowerMonitor now queries Scanner for fresh device info on each reading
  - Ensures power readings reflect current device names even if devices are renamed
  - Updated all tests with mock scanner implementation

### 25. ✅ Add Missing Error Context - COMPLETED
- [x] **Files**: `storage/cache.go`
- [x] **Issue**: Many errors lack operation context
- [x] **Fix**: Completed in commit `afb2789`
  - Added error context to WriteBatch operation
  - Now includes reading index (i+1/total) and device_id in error messages
  - Pattern: `fmt.Errorf("failed to write reading %d/%d (device_id=%s): %w", ...)`
  - Improves debugging by showing exactly which reading in a batch failed

### 26. ✅ Update All Outdated Dependencies - COMPLETED
- [x] **File**: `go.mod`, `go.sum`
- [x] **Issue**: 20+ dependencies had newer versions
- [x] **Fix**: Completed in commit `c87dfb7`
  - prometheus/client_golang: v1.19.0 → v1.23.2
  - prometheus/client_model: v0.5.0 → v0.6.2
  - prometheus/common: v0.48.0 → v0.66.1
  - prometheus/procfs: v0.12.0 → v0.16.1
  - rs/zerolog: v1.32.0 → v1.34.0
  - google.golang.org/grpc: v1.75.1 → v1.76.0
  - OpenTelemetry packages: v1.37.0 → v1.38.0
  - Updated 20+ transitive dependencies for security and compatibility
  - All tests passing, linter clean

### 27. ✅ Replace Deprecated golang/protobuf - ADDRESSED
- [x] **File**: `go.mod`
- [x] **Issue**: `github.com/golang/protobuf v1.5.4` is deprecated
- [x] **Status**: Addressed in commit `c87dfb7`
  - The deprecated package is only present as a transitive dependency for backward compatibility
  - Main module does not directly use deprecated package (verified with `go mod why`)
  - Updated grpc to v1.76.0 which uses modern google.golang.org/protobuf
  - This is the expected and correct situation for Go modules

### 28. ✅ Make Channel Sizes Configurable - COMPLETED
- [x] **Files**: `config/config.go`, `monitoring/power.go`, `main.go`, test files
- [x] **Issue**: Readings channel buffer size (100) was hardcoded
- [x] **Fix**: Completed in commit `ce8c672`
  - Added ReadingsChannelSize field to MatterConfig
  - Default value: 100 (maintains existing behavior)
  - Validation: 1-10000 range when explicitly set, 0 for default
  - Updated NewPowerMonitor to accept channelSize parameter
  - Updated all callers in main.go and test files
  - Users can now tune channel size via config.yaml based on workload

### 29. Add Metrics Cardinality Limits
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Issue**: Unbounded cardinality with device_id labels
- [ ] **Fix**: Add device count limits or remove device_name label

### 30. ✅ Document InfluxDB Connection Pooling - COMPLETED
- [x] **File**: `storage/influxdb.go`
- [x] **Issue**: Connection pooling behavior was undocumented
- [x] **Fix**: Completed in commit `b4ffc1f`
  - Added comprehensive package-level documentation
  - Documented HTTP connection pooling via net/http
  - Explained default Go http.Transport settings (MaxIdleConns, IdleConnTimeout, etc.)
  - Documented thread-safety and connection reuse behavior
  - Added performance characteristics to NewInfluxDBStorage function
  - No code changes needed - client already handles pooling efficiently

---

## 🟢 LOW PRIORITY (Nice to Have)

### 31. Add Package-Level Documentation
- [ ] **Files**: All packages
- [ ] **Issue**: Missing package doc comments for godoc
- [ ] **Fix**: Add package-level comments describing purpose and usage

### 32. Add Integration Tests
- [ ] **Issue**: Only unit tests exist, no end-to-end tests
- [ ] **Fix**: Add integration tests using:
  - Testcontainers for InfluxDB
  - Mock mDNS server
  - Full startup/shutdown cycle

### 33. Add Benchmark Tests
- [ ] **Issue**: No performance benchmarks
- [ ] **Fix**: Add benchmarks for:
  - Power reading generation
  - InfluxDB write performance
  - Discovery parsing
  - Metrics updates

### 34. Add Fuzz Tests
- [ ] **Files**: `discovery/discovery.go`, `storage/influxdb.go`
- [ ] **Issue**: No fuzzing for input parsing
- [ ] **Fix**: Add Go 1.18+ fuzz tests for:
  - TXT record parsing
  - Device ID generation
  - Flux query sanitization

### 35. Add Error Path Testing
- [ ] **Issue**: Many error branches not tested
- [ ] **Fix**: Add negative tests forcing error conditions

### 36. Add Race Condition Stress Tests
- [ ] **Issue**: Concurrent access might have undiscovered races
- [ ] **Fix**: Add stress tests with many goroutines

### 37. Create Architecture Decision Records
- [ ] **Issue**: No documentation of design decisions
- [ ] **Fix**: Document decisions like:
  - Why zeroconf library
  - Why async InfluxDB writes
  - Why channel-based architecture

### 38. Generate Metrics Documentation
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Issue**: Metrics not exported in structured way
- [ ] **Fix**: Generate metrics docs for operators

### 39. Expand Troubleshooting Guide
- [ ] **File**: `README.md`
- [ ] **Issue**: Minimal troubleshooting section
- [ ] **Fix**: Cover:
  - Firewall issues
  - mDNS permission problems
  - Configuration errors
  - Device discovery verification

### 40. Add Deployment Best Practices Docs
- [ ] **Issue**: No production deployment guidance
- [ ] **Fix**: Document:
  - Resource requirements
  - Scaling considerations
  - Backup/restore
  - Meta-monitoring

### 41. Add Extension Examples
- [ ] **Issue**: No examples for extending the system
- [ ] **Fix**: Add examples for:
  - New storage backends
  - Additional Matter clusters
  - Custom metrics

### 42. Improve Complex Code Comments
- [ ] **File**: `discovery/discovery.go:86-98`
- [ ] **Issue**: Complex concurrent code lacks detailed comments
- [ ] **Fix**: Explain synchronization strategy

### 43. Add Configuration Hot Reload
- [ ] **File**: `config/config.go`
- [ ] **Issue**: Requires restart for config changes
- [ ] **Fix**: Add SIGHUP handler for reload

### 44. Add Configuration Schema
- [ ] **Issue**: No JSON Schema for validation
- [ ] **Fix**: Add schema file and validation

### 45. Support Configuration Profiles
- [ ] **Issue**: Single config, no dev/staging/prod profiles
- [ ] **Fix**: Support multiple config files

### 46. Add Observability Configuration
- [ ] **Issue**: Missing config for metrics interval, log format, tracing
- [ ] **Fix**: Add observability config section

### 47. Add Resource Limits Configuration
- [ ] **Issue**: No limits for max devices, buffer sizes, batch sizes
- [ ] **Fix**: Add resource limits config

### 48. Extract Business Logic from Main
- [ ] **File**: `main.go:147-206`
- [ ] **Issue**: Discovery logic in main function
- [ ] **Fix**: Extract to application/service layer

### 49. Add Graceful Degradation
- [ ] **Issue**: InfluxDB down = silent data loss
- [ ] **Fix**: Add fallback storage or fail loudly

### 50. Enhance Metrics Help Text
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Fix**: Add units and examples to help text

### 51. Define Structured Error Types
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

### 52. Add Alerting System
- [ ] **Description**: No alerts when devices offline or readings anomalous
- [ ] **Fix**: Integrate with Alertmanager or add webhook system

### 53. Add Device Discovery Events
- [ ] **Description**: No notification when devices appear/disappear
- [ ] **Fix**: Add event system or webhooks

### 54. Add Historical Data Export
- [ ] **Description**: No export/backup functionality
- [ ] **Fix**: Add CLI command or API for data export

### 55. Add Device Metadata Persistence
- [ ] **Description**: Device names, locations lost on restart
- [ ] **Fix**: Persist metadata (JSON file or database)

### 56. Add Resource Limit Enforcement
- [ ] **Description**: No limits on goroutines, memory, file descriptors
- [ ] **Fix**: Add resource monitoring and limits

### 57. Add Discovery Fallback Mode
- [ ] **Description**: mDNS failure prevents startup
- [ ] **Fix**: Allow running without discovery for testing

### 58. Add Debug Signal Handlers
- [ ] **Description**: No SIGUSR1/SIGUSR2 for debugging
- [ ] **Fix**: Add handlers to dump goroutines, device state

### 59. Add Docker Secrets Support
- [ ] **File**: `docker-compose.yml`
- [ ] **Issue**: Uses environment variables in compose
- [ ] **Fix**: Use Docker secrets or external secret management

### 60. Add Incremental Discovery
- [ ] **File**: `discovery/discovery.go`
- [ ] **Issue**: Full network scan every time
- [ ] **Fix**: Support continuous/incremental discovery

---

## Test Coverage Goals

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| main | 23.3% | 80% | 🔶 Reasonable coverage (architectural limits) |
| storage | 72.3% | 85% | 🔶 Close to target (with integration tests) |
| discovery | 93.8% | 85% | ✅ Exceeds target |
| monitoring | 79.5% | 90% | 🔶 Close to target |
| config | 89.8% | 95% | 🔶 Close to target |
| metrics | 100.0% | 100% | ✅ Perfect |
| logger | 90.9% | 100% | 🔶 Close to target |

**Overall Coverage**: 67% (Updated 2025-11-11)

---

## Dependency Updates Needed

- `golang.org/x/net` (security)
- `golang.org/x/crypto` (security)
- `golang.org/x/sys`
- `github.com/golang/protobuf` (deprecated)
- 20+ other indirect dependencies

---

## Completion Tracking

- Total Items: 65
- **Completed**: 30 items ✅
- **In Progress**: 0 items
- **Remaining**: 35 items

### By Priority:
- Critical (🔴): 5/5 completed (100%) ✅
- High (🟠): 14/14 completed (100%) ✅ **ALL HIGH PRIORITY ITEMS DONE!**
- Medium (🟡): 11/16 completed (69%) 🎯 **SIGNIFICANT PROGRESS!**
- Low (🟢): 0/22 completed (0%)
- Features (🌟): 0/8 completed (0%)

### Recently Completed Items (current session - 2025-11-11):
19. ✅ Update All Outdated Dependencies (#26) - Completed in commit `c87dfb7`
20. ✅ Address Deprecated golang/protobuf (#27) - Completed in commit `c87dfb7`
21. ✅ Extract Magic Numbers to Constants (#15) - Completed in commit `2581011`
22. ✅ Fix parseLogLevel Error Handling (#16) - Completed in commit `b971ac7`
23. ✅ Initialize Global Logger Safely (#17) - Completed in commit `3fc2756`
24. ✅ Add Context to Async Operations (#14) - Completed in commit `59f8d7c`
25. ✅ Add Tests for Main Package (#6) - 0% → 23.3% - Completed in commits `922b94d`, `13a7bdd`
26. ✅ Use Consistent Error Wrapping (#22) - Completed in commit `bfa4003`
27. ✅ Add Context Checks in Monitoring Loops (#23) - Completed in commit `e67fa94`
28. ✅ Fix Device Name Staleness (#24) - Completed in commit `43426f8`
29. ✅ Add Missing Error Context (#25) - Completed in commit `afb2789`
30. ✅ Make Channel Sizes Configurable (#28) - Completed in commit `ce8c672`
31. ✅ Document InfluxDB Connection Pooling (#30) - Completed in commit `b4ffc1f`

### Previously Completed Items (commit `c09c86c`):
16. ✅ Define Interfaces for External Dependencies (#12e)
17. ✅ Improve Discovery Package Test Coverage (#8)
18. ✅ Improve Storage Package Test Coverage (#7) - 8.1% → 72.3%

### Previously Completed Items (commit `fcec935`):
9. ✅ Implement HTTP Server Graceful Shutdown (#12)
10. ✅ Fix PowerMonitor Channel Closure (#12a)
11. ✅ Add InfluxDB Flush Timeout (#12b)
12. ✅ Fix Discovery Nil Slice Return (#12c)
13. ✅ Fix Code Quality Issues - Variable Shadowing & Staticcheck (#12d)
14. ✅ Fix Goroutine Lifecycle Management (#13)
15. ✅ Secure Metrics Endpoint - Localhost Binding (#19)

### Previously Completed Items:
1. ✅ Remove Hardcoded Secrets (commit `968cb22`)
2. ✅ Update Security-Critical Dependencies (commit `968cb22`)
3. ✅ Add Retry Logic for InfluxDB Writes (commit `968cb22`)
4. ✅ Enforce TLS for Production InfluxDB (commit `968cb22`)
5. ✅ Add Vulnerability Scanning to CI/CD (commit `968cb22`)
6. ✅ Add Input Validation for Power Readings (commit `968cb22`)
7. ✅ Add Comprehensive Configuration Validation (commit `078afed`)
8. ✅ Fix Unbuffered Discovery Channel (commit `968cb22`)

---

## Notes

This TODO list was generated through comprehensive static analysis of the codebase on 2025-11-11. Items are organized by priority, with critical security and data loss issues at the top.

**Last Updated**: 2025-11-11 (27 items completed - ALL HIGH PRIORITY ITEMS DONE! Medium priority 50% complete! 🎉)

The codebase is generally well-structured and follows good Go practices. These improvements will enhance security, reliability, testability, and maintainability.

### Recent Progress (commit `c09c86c`)
**Major improvements to testability and test coverage:**
- ✅ Created comprehensive interfaces package for better testability
  - `pkg/interfaces/storage.go` - TimeSeriesStorage interface
  - `pkg/interfaces/discovery.go` - DeviceScanner & DeviceCapabilities interfaces
  - `pkg/interfaces/monitoring.go` - PowerMonitor interface
- ✅ Added initial main package tests (0% → 2.2% coverage)
  - Health check endpoints
  - Graceful shutdown function
  - Cleanup function
- ✅ Discovery package now exceeds target (93.8% coverage, target: 85%)
  - Comprehensive test coverage for all discovery scenarios
  - Edge case handling verified

**Test Status**:
- All 96 tests passing ✅
- golangci-lint clean ✅
- No race conditions detected ✅
- Overall coverage: 67%
- Discovery: 93.8% (exceeds 85% target) 🎉

**Recommended Approach**:
1. ✅ ~~Start with all 🔴 CRITICAL items~~ (COMPLETED!)
2. ✅ ~~Continue with 🟠 HIGH priority items~~ (COMPLETED - 14/14 done, 100%! 🎉)
3. 🎯 **NOW**: Address 🟡 MEDIUM items in batches (8/16 completed - 50%!)
4. Consider 🟢 LOW and 🌟 FEATURE items as time permits

**Next Focus Areas**:
- Rate limiting on health endpoints (#18)
- Improve Flux query safety (#20)
- Circuit breaker for InfluxDB (#21)
- Make channel sizes configurable (#28)
- Add metrics cardinality limits (#29)
