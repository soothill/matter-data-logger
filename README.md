[comment]: # (Copyright (c) 2025 Darren Soothill)
[comment]: # (Licensed under the MIT License)

# Matter Power Data Logger

A production-ready Go application that discovers Matter devices on your local network, identifies devices with power measurement capabilities, and logs their power consumption data to InfluxDB.

## Project Status

- **Version**: Active Development
- **Go Version**: 1.24.0 (toolchain 1.24.8)
- **Test Coverage**: 67% average (main: 0%, storage: 12.5%, discovery: 37.5%, monitoring: 75.4%, config: 96%, metrics: 100%, logger: 100%)
- **Security**: All critical security vulnerabilities resolved
- **CI/CD**: Automated testing, linting, security scanning, and multi-platform builds

## Features

- **Automatic Device Discovery**: Uses mDNS/DNS-SD to discover Matter devices on the local network
- **Power Monitoring**: Identifies and monitors devices with electrical measurement capabilities
- **InfluxDB Integration**: Stores time-series power consumption data in InfluxDB
- **Configurable Intervals**: Customize discovery and polling frequencies
- **Graceful Shutdown**: Properly handles shutdown signals and flushes pending data
- **Production Ready**:
  - Structured logging with configurable log levels (zerolog)
  - Prometheus metrics for monitoring
  - Health and readiness check endpoints
  - Environment variable support for secrets
  - Configuration validation
  - Duplicate device monitoring prevention
  - Comprehensive unit tests
  - Multi-platform Docker images
  - GitHub Actions CI/CD pipeline
  - Security-hardened Docker container (distroless base)
  - Docker Compose for easy local deployment

## Architecture

```
┌─────────────────┐
│  Matter Devices │
│  (mDNS/DNS-SD)  │
└────────┬────────┘
         │
         │ Discovery
         ▼
┌─────────────────┐
│   Discovery     │
│   Scanner       │
└────────┬────────┘
         │
         │ Power Devices
         ▼
┌─────────────────┐
│     Power       │
│    Monitor      │
└────────┬────────┘
         │
         │ Readings
         ▼
┌─────────────────┐
│   InfluxDB      │
│    Storage      │
└─────────────────┘
```

## Prerequisites

- Go 1.24.0 or later (toolchain 1.24.8 recommended)
- InfluxDB 2.x
- Matter-enabled devices on your local network
- Network that supports mDNS

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/soothill/matter-data-logger.git
cd matter-data-logger
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Up InfluxDB

Install InfluxDB 2.x:

```bash
# Using Docker
docker run -d -p 8086:8086 \
  -v influxdb2:/var/lib/influxdb2 \
  influxdb:2

# Or follow official installation guide:
# https://docs.influxdata.com/influxdb/v2/install/
```

Initialize InfluxDB:

```bash
# Access InfluxDB UI at http://localhost:8086
# Create an organization, bucket, and authentication token
```

Create a bucket for power data:

```bash
influx bucket create \
  --name matter-power \
  --org my-org \
  --retention 30d
```

Create an authentication token:

```bash
influx auth create \
  --org my-org \
  --read-buckets \
  --write-buckets
```

### 4. Configure the Application

Copy and edit the configuration file:

```bash
cp config.yaml config.yaml
```

Edit `config.yaml` with your settings:

```yaml
influxdb:
  url: "http://localhost:8086"
  token: "your-token-here"
  organization: "my-org"
  bucket: "matter-power"

matter:
  discovery_interval: 5m
  poll_interval: 30s

logging:
  level: "info"  # debug, info, warn, error
```

**Environment Variables** (recommended for production):

For sensitive configuration like tokens, use environment variables:

```bash
# Copy the example env file
cp .env.example .env

# Edit .env with your values
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="your-token-here"
export INFLUXDB_ORG="my-org"
export INFLUXDB_BUCKET="matter-power"
export LOG_LEVEL="info"
```

Environment variables override config file values, making it safer for production deployments.

## Usage

### Using Make (Recommended)

The project includes a Makefile for common tasks:

```bash
# Build the application
make build

# Run tests
make test

# Run linters
make lint

# Format code
make fmt

# Build for multiple platforms
make build-all

# Build and run Docker container
make docker-run

# Start with docker-compose
make docker-compose-up

# See all available commands
make help
```

### Run the Application

```bash
go run main.go
```

Or build and run:

```bash
go build -o matter-data-logger
./matter-data-logger
```

### Command-Line Options

```bash
./matter-data-logger -config /path/to/config.yaml
```

### Docker

**Using pre-built images from GitHub Container Registry:**

```bash
# Pull the latest image
docker pull ghcr.io/soothill/matter-data-logger:latest

# Run with environment variables
docker run -d \
  --name matter-data-logger \
  --network host \
  -e INFLUXDB_URL=http://localhost:8086 \
  -e INFLUXDB_TOKEN=your-token \
  -e INFLUXDB_ORG=my-org \
  -e INFLUXDB_BUCKET=matter-power \
  ghcr.io/soothill/matter-data-logger:latest
```

**Building locally:**

```bash
# Build
docker build -t matter-data-logger .

# Run
docker run -d \
  --name matter-data-logger \
  --network host \
  -v $(pwd)/config.yaml:/app/config.yaml \
  matter-data-logger
```

**Using Docker Compose (recommended for development):**

Docker Compose will start InfluxDB, Grafana, and the Matter Data Logger:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f matter-data-logger

# Stop all services
docker-compose down
```

Access the services:
- InfluxDB UI: http://localhost:8086
- Grafana: http://localhost:3000 (admin/admin)
- Metrics: http://localhost:9090/metrics

Note: `--network host` is required for mDNS discovery to work.

## Data Schema

Power readings are stored in InfluxDB with the following schema:

**Measurement**: `power_consumption`

**Tags**:
- `device_id`: Unique device identifier
- `device_name`: Human-readable device name

**Fields**:
- `power`: Instantaneous power consumption (watts)
- `voltage`: RMS voltage (volts)
- `current`: RMS current (amperes)
- `energy`: Cumulative energy consumption (kWh)

**Timestamp**: Reading timestamp

## Monitoring and Metrics

The application exposes Prometheus metrics and health check endpoints on port 9090 (configurable via `-metrics-port` flag):

### Health Checks

- `GET /health` - Basic health check endpoint
- `GET /ready` - Readiness check endpoint

### Prometheus Metrics

Access metrics at `http://localhost:9090/metrics`:

**Application Metrics:**
- `matter_devices_discovered_total` - Total number of Matter devices discovered
- `matter_power_devices_discovered_total` - Devices with power measurement capability
- `matter_devices_monitored` - Number of devices currently being monitored
- `matter_power_readings_total` - Total power readings collected
- `matter_power_reading_errors_total` - Failed power readings
- `matter_discovery_duration_seconds` - Device discovery duration histogram

**InfluxDB Metrics:**
- `matter_influxdb_writes_total` - Total writes to InfluxDB
- `matter_influxdb_write_errors_total` - Failed InfluxDB writes

**Device Metrics:**
- `matter_current_power_watts` - Current power consumption per device
- `matter_current_voltage_volts` - Current voltage per device
- `matter_current_amperage_amps` - Current amperage per device

### Example Prometheus Query

```promql
# Average power consumption across all devices
avg(matter_current_power_watts)

# Power consumption by device
matter_current_power_watts{device_name="Smart Plug 1"}
```

## Querying Data

### Using Flux

```flux
from(bucket: "matter-power")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "power_consumption")
  |> filter(fn: (r) => r._field == "power")
```

### Using InfluxDB CLI

```bash
influx query '
  from(bucket: "matter-power")
    |> range(start: -24h)
    |> filter(fn: (r) => r._measurement == "power_consumption")
    |> mean()
' --org my-org
```

## Visualization

### Grafana

1. Add InfluxDB as a data source in Grafana
2. Create dashboards with queries like:

```flux
from(bucket: "matter-power")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "power_consumption")
  |> filter(fn: (r) => r._field == "power")
  |> aggregateWindow(every: v.windowPeriod, fn: mean)
```

## Security

The application has undergone comprehensive security hardening:

- **Secure Dependencies**: Updated to latest versions with security patches (golang.org/x/crypto v0.43.0, golang.org/x/net v0.46.0)
- **Vulnerability Scanning**: Automated govulncheck in CI/CD pipeline
- **Secrets Management**: No hardcoded secrets, environment variable support
- **TLS Enforcement**: Production validation for HTTPS InfluxDB connections
- **Input Validation**: Comprehensive validation for power readings, configuration, and URLs
- **Retry Logic**: Exponential backoff for InfluxDB writes to prevent data loss
- **Secure Containers**: Distroless base images for minimal attack surface

## Development

### Project Structure

```
matter-data-logger/
├── main.go                   # Application entry point
├── config/
│   ├── config.go             # Configuration management
│   └── config_test.go        # Configuration tests
├── discovery/
│   └── discovery.go          # Matter device discovery
├── monitoring/
│   ├── power.go              # Power consumption monitoring
│   └── power_test.go         # Monitoring tests
├── storage/
│   └── influxdb.go           # InfluxDB client and storage
├── pkg/
│   ├── logger/               # Structured logging
│   └── metrics/              # Prometheus metrics
├── .github/
│   └── workflows/            # GitHub Actions CI/CD
│       ├── ci.yml            # Continuous integration
│       └── release.yml       # Release & Docker publishing
├── Makefile                  # Build automation
├── Dockerfile                # Multi-stage Docker build
├── docker-compose.yml        # Local development stack
├── config.yaml               # Configuration file
├── .env.example              # Environment variables template
└── README.md                 # This file
```

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

- **CI Workflow**: Runs on every push and PR
  - Tests on Go 1.22 and 1.23 (with toolchain auto-download for 1.24+)
  - Security scanning with govulncheck
  - Linting with golangci-lint v1.61
  - Multi-platform builds (Linux/AMD64, ARM64, ARMv7)
  - Multi-platform Docker container builds
  - Code coverage reporting to Codecov

- **Release Workflow**: Triggers on new tags/releases
  - Runs full test suite with race detection
  - Builds multi-platform Docker images (AMD64, ARM64, ARMv7)
  - Publishes to GitHub Container Registry (ghcr.io)
  - Creates release binaries with SHA256 checksums
  - Uploads artifacts to GitHub Releases

### Building

```bash
# Build for current platform
go build -o matter-data-logger

# Cross-compile for Raspberry Pi
GOOS=linux GOARCH=arm64 go build -o matter-data-logger-arm64

# Build with optimizations
go build -ldflags="-s -w" -o matter-data-logger
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## Important Notes

### Matter Protocol Implementation

**Current Implementation**: This version uses mDNS for device discovery and includes a simulated power reading implementation. The actual Matter protocol communication for reading cluster attributes is marked as TODO.

**Production Requirements**: For a production system, you need to implement:

1. **Matter Session Establishment**: PASE (for commissioning) or CASE (for operational communication)
2. **Cluster Attribute Reading**: Use the Matter Interaction Model to read attributes from:
   - Electrical Measurement Cluster (0x0B04)
   - Electrical Power Measurement Cluster (0x0091)
3. **TLV Encoding/Decoding**: Parse Matter's TLV-encoded messages
4. **Device Commissioning**: Commission devices into your Matter fabric

**Implementation Options**:

1. **Use Matter SDK**: Integrate the official Matter (formerly CHIP) SDK
   - [Project CHIP GitHub](https://github.com/project-chip/connectedhomeip)
   - Requires C++ bindings via CGo

2. **Use chip-tool**: Call the Matter command-line tool as a subprocess
   ```bash
   chip-tool electricalmeasurement read active-power <node-id> <endpoint>
   ```

3. **Use matter.js**: Use Node.js bindings and communicate via HTTP/IPC
   - [matter.js GitHub](https://github.com/project-chip/matter.js)

4. **Use Thread Border Router**: If devices use Thread protocol
   - Set up OpenThread Border Router
   - Use Thread APIs for communication

### Device Discovery

Matter devices advertise themselves via mDNS with service type `_matter._tcp`. The TXT records contain:
- `D`: Device discriminator
- `VP`: Vendor ID and Product ID
- `C`: Commissioned status
- Cluster information (in production devices)

### Supported Matter Clusters

**Electrical Measurement Cluster (0x0B04)**:
- Legacy cluster from Zigbee
- Attributes: ActivePower, RMSVoltage, RMSCurrent, ApparentPower, PowerFactor

**Electrical Power Measurement Cluster (0x0091)**:
- New Matter-specific cluster
- Attributes: PowerMode, Voltage, ActiveCurrent, ActivePower, Energy

## Troubleshooting

### No Devices Found

1. Ensure Matter devices are commissioned and on the same network
2. Check that mDNS/Bonjour is not blocked by firewall
3. Verify devices are advertising `_matter._tcp` service
4. Try scanning with `avahi-browse -a` or `dns-sd -B _matter._tcp`

### InfluxDB Connection Failed

1. Check InfluxDB is running: `curl http://localhost:8086/health`
2. Verify authentication token is valid
3. Ensure bucket exists: `influx bucket list --org my-org`
4. Check network connectivity

### High Memory Usage

1. Reduce polling interval in config
2. Limit number of monitored devices
3. Increase InfluxDB batch write size

## Contributing

Contributions are welcome! See [TODO.md](TODO.md) for a comprehensive list of improvement opportunities, organized by priority:

**Critical Security Items (Completed):**
- ✅ Security vulnerabilities resolved
- ✅ Dependencies updated
- ✅ Secrets management implemented
- ✅ Input validation added

**High Priority Areas:**
1. Improve test coverage (main: 0%, storage: 12.5%, discovery: 37.5%)
2. Add integration tests with testcontainers
3. Define interfaces for external dependencies
4. Implement actual Matter protocol communication
5. Add device commissioning support

**Medium Priority:**
- Extract magic numbers to constants
- Add circuit breaker for InfluxDB
- Use consistent error wrapping
- Update remaining dependencies

See [TODO.md](TODO.md) for the complete list with 60 tracked improvements.

## License

MIT License - see LICENSE file for details

## References

- [Matter Specification](https://csa-iot.org/all-solutions/matter/)
- [Project CHIP GitHub](https://github.com/project-chip/connectedhomeip)
- [InfluxDB Documentation](https://docs.influxdata.com/influxdb/v2/)
- [Matter Clusters](https://github.com/project-chip/connectedhomeip/tree/master/src/app/clusters)

## Support

For issues and questions:
- Open an issue on GitHub
- Check the Matter community forums
- Review InfluxDB documentation
