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

## 🟠 HIGH PRIORITY (Newly Identified)

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
