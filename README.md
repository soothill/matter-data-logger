# Matter Power Data Logger

A Go application that discovers Matter devices on your local network, identifies devices with power measurement capabilities, and logs their power consumption data to InfluxDB.

## Features

- **Automatic Device Discovery**: Uses mDNS/DNS-SD to discover Matter devices on the local network
- **Power Monitoring**: Identifies and monitors devices with electrical measurement capabilities
- **InfluxDB Integration**: Stores time-series power consumption data in InfluxDB
- **Configurable Intervals**: Customize discovery and polling frequencies
- **Graceful Shutdown**: Properly handles shutdown signals and flushes pending data

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

- Go 1.21 or later
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
```

## Usage

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

Build and run with Docker:

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

## Development

### Project Structure

```
matter-data-logger/
├── main.go              # Application entry point
├── config/
│   └── config.go        # Configuration management
├── discovery/
│   └── discovery.go     # Matter device discovery
├── monitoring/
│   └── power.go         # Power consumption monitoring
├── storage/
│   └── influxdb.go      # InfluxDB client and storage
├── config.yaml          # Configuration file
└── README.md           # This file
```

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

Contributions are welcome! Areas that need work:

1. Implement actual Matter protocol communication
2. Add device commissioning support
3. Support additional Matter clusters
4. Add Prometheus metrics export
5. Improve error handling and retry logic
6. Add unit tests and integration tests

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
