[comment]: # (Copyright (c) 2025 Darren Soothill)
[comment]: # (Licensed under the MIT License)

# Code Improvements TODO

This document tracks code improvement opportunities identified through comprehensive codebase analysis.

## Project Overview
- **Type**: Go 1.24.0 (toolchain 1.24.8) Matter Power Data Logger
- **LOC**: ~2,507 lines
- **Test Coverage**: 67% average (main: 0%, storage: 12.5%, discovery: 37.5%)
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

### 6. Add Tests for Main Package (0% Coverage)
- [ ] **File**: `main.go`
- [ ] **Issue**: No tests for application initialization, signal handling, graceful shutdown
- [ ] **Fix**: Create `main_test.go` with tests for:
  - Configuration loading and validation
  - Signal handling (SIGTERM, SIGINT)
  - Graceful shutdown flow
  - Health check endpoints
  - Component integration

### 7. Improve Storage Package Test Coverage (12.5%)
- [ ] **File**: `storage/influxdb.go`, `storage/influxdb_test.go`
- [ ] **Issue**: Critical data persistence layer poorly tested
- [ ] **Fix**: Add integration tests with testcontainers:
  - Actual write operations
  - Connection pooling
  - Error recovery
  - Batch writes
  - Query operations

### 8. Improve Discovery Package Test Coverage (37.5%)
- [ ] **File**: `discovery/discovery.go`
- [ ] **Issue**: Core functionality insufficiently tested
- [ ] **Fix**: Add tests for:
  - Actual mDNS discovery (using mock zeroconf)
  - Service entry parsing edge cases
  - Concurrent device updates
  - IPv6 handling

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

### 12. Define Interfaces for External Dependencies
- [ ] **Files**: Multiple
- [ ] **Issue**: Direct dependencies on concrete types (InfluxDB, zeroconf)
- [ ] **Impact**: Hard to mock, tight coupling
- [ ] **Fix**: Define interfaces:
  ```go
  type TimeSeriesStorage interface {
      WriteReading(ctx context.Context, reading *PowerReading) error
      Close() error
  }
  type DeviceDiscoverer interface {
      Discover(ctx context.Context, service string) ([]*Device, error)
  }
  ```

### 13. Fix Potential Goroutine Leak in Error Handler
- [ ] **File**: `storage/influxdb.go:55-59`
- [ ] **Issue**: Error handling goroutine never terminates properly
- [ ] **Fix**: Add context cancellation and cleanup in Close()

### 14. Add Context to Async Operations
- [ ] **File**: `storage/influxdb.go:70`
- [ ] **Issue**: `WriteReading` doesn't accept context for cancellation
- [ ] **Fix**: Add context parameter to WriteReading and all async operations

---

## 🟡 MEDIUM PRIORITY

### 15. Extract Magic Numbers to Constants
- [ ] **Files**: `main.go:100-123`, `monitoring/power.go:41`
- [ ] **Issue**: Hard-coded buffer sizes (100), timeouts (10s, 5s)
- [ ] **Fix**: Extract to constants:
  ```go
  const (
      ReadingsChannelSize = 100
      DiscoveryTimeout = 10 * time.Second
      HealthCheckTimeout = 5 * time.Second
  )
  ```

### 16. Fix parseLogLevel Error Handling
- [ ] **File**: `pkg/logger/logger.go:38-55`
- [ ] **Issue**: Returns nil instead of error when parsing fails
- [ ] **Fix**: Return actual error or log warning

### 17. Initialize Global Logger Safely
- [ ] **File**: `pkg/logger/logger.go`
- [ ] **Issue**: Global `log` not initialized, could panic before Initialize()
- [ ] **Fix**: Initialize with default logger in init() or use sync.Once

### 18. Add Rate Limiting on Health Endpoints
- [ ] **Files**: `main.go:64-76`, `main.go:209-235`
- [ ] **Issue**: No rate limiting, could be DoS target
- [ ] **Fix**: Add rate limiting middleware using `golang.org/x/time/rate`

### 19. Add Authentication to Metrics Endpoint
- [ ] **File**: `main.go:65`
- [ ] **Issue**: Prometheus metrics exposed without auth
- [ ] **Risk**: Could leak device information
- [ ] **Fix**: Add basic auth or mTLS

### 20. Improve Flux Query Safety
- [ ] **File**: `storage/influxdb.go:150-176`
- [ ] **Issue**: String concatenation for queries, injection risk remains
- [ ] **Fix**: Use parameterized queries or InfluxDB query builder API

### 21. Add Circuit Breaker for InfluxDB
- [ ] **File**: `storage/influxdb.go`
- [ ] **Issue**: Continues writing even when InfluxDB consistently fails
- [ ] **Fix**: Implement circuit breaker (e.g., `github.com/sony/gobreaker`)

### 22. Use Consistent Error Wrapping
- [ ] **Files**: Multiple
- [ ] **Issue**: Inconsistent use of `%w` vs `%v` in error formatting
- [ ] **Fix**: Use `%w` consistently with `fmt.Errorf` for error chains

### 23. Add Context Checks in Monitoring Loops
- [ ] **File**: `monitoring/power.go:124-149`
- [ ] **Issue**: Doesn't check context before expensive operations
- [ ] **Fix**: Add context check after ticker fires

### 24. Fix Device Name Staleness
- [ ] **File**: `monitoring/power.go`
- [ ] **Issue**: Device name copied at start, won't update if renamed
- [ ] **Fix**: Fetch fresh device info or add device update mechanism

### 25. Add Missing Error Context
- [ ] **Files**: Multiple
- [ ] **Issue**: Many errors lack operation context
- [ ] **Fix**: Use pattern: `fmt.Errorf("operation %s failed: %w", op, err)`

### 26. Update All Outdated Dependencies
- [ ] **File**: `go.mod`
- [ ] **Issue**: 20+ dependencies have newer versions
- [ ] **Fix**: Run `go get -u ./...` and test thoroughly

### 27. Replace Deprecated golang/protobuf
- [ ] **File**: `go.mod`
- [ ] **Issue**: `github.com/golang/protobuf v1.5.3` is deprecated
- [ ] **Fix**: Update to `google.golang.org/protobuf`

### 28. Make Channel Sizes Configurable
- [ ] **File**: `monitoring/power.go:41`
- [ ] **Issue**: Readings channel buffer size (100) is arbitrary
- [ ] **Fix**: Make configurable or calculate based on poll interval × device count

### 29. Add Metrics Cardinality Limits
- [ ] **File**: `pkg/metrics/metrics.go`
- [ ] **Issue**: Unbounded cardinality with device_id labels
- [ ] **Fix**: Add device count limits or remove device_name label

### 30. Document InfluxDB Connection Pooling
- [ ] **File**: `storage/influxdb.go`
- [ ] **Note**: Client already handles pooling, document the behavior

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

| Package | Current | Target |
|---------|---------|--------|
| main | 0.0% | 80% |
| storage | 12.5% | 85% |
| discovery | 37.5% | 85% |
| monitoring | 75.4% | 90% |
| config | 96.0% | 95% |
| metrics | 100.0% | 100% |
| logger | 100.0% | 100% |

---

## Dependency Updates Needed

- `golang.org/x/net` (security)
- `golang.org/x/crypto` (security)
- `golang.org/x/sys`
- `github.com/golang/protobuf` (deprecated)
- 20+ other indirect dependencies

---

## Completion Tracking

- Total Items: 60
- **Completed**: 8 items ✅
- **Remaining**: 52 items

### By Priority:
- Critical (🔴): 5/5 completed (100%) ✅
- High (🟠): 3/9 completed (33%)
- Medium (🟡): 0/16 completed (0%)
- Low (🟢): 0/22 completed (0%)
- Features (🌟): 0/8 completed (0%)

### Completed Items:
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

**Last Updated**: 2025-11-11 (8 items completed)

The codebase is generally well-structured and follows good Go practices. These improvements will enhance security, reliability, testability, and maintainability.

**Recommended Approach**:
1. ✅ ~~Start with all 🔴 CRITICAL items~~ (COMPLETED!)
2. Continue with 🟠 HIGH priority items (3/9 done, 6 remaining)
3. Address 🟡 MEDIUM items in batches
4. Consider 🟢 LOW and 🌟 FEATURE items as time permits
