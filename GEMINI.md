# GEMINI.md

## Project Overview

This project, "Matter Power Data Logger," is a Go application designed to monitor and log power consumption data from Matter-enabled smart home devices. It discovers these devices on the local network using mDNS, identifies devices with power measurement capabilities, and stores their power consumption data in an InfluxDB time-series database.

The application is built to be production-ready, incorporating several key features:
- **Data Resilience:** A local caching mechanism ensures that data is not lost if the connection to InfluxDB is temporarily unavailable. The data is automatically replayed once the connection is restored.
- **Monitoring and Alerting:** The application exposes Prometheus metrics for monitoring its own health and the data it collects. It can also send alerts to a Slack channel for critical events like discovery failures or database connection issues.
- **Configuration:** The application is configured via a YAML file (`config.yaml`), with support for environment variable overrides for sensitive data.
- **Containerization:** The project includes a `Dockerfile` and `docker-compose.yml` for easy deployment and development.

## Building and Running

The project uses a `Makefile` to streamline common development tasks.

### Building the application

To build the application binary, run:
```bash
make build
```

### Running tests

To run the unit tests, use:
```bash
make test
```
To run integration tests (requires Docker), use:
```bash
make test-integration
```

### Running the application

To build and run the application locally, use:
```bash
make run
```
This will use the `config.yaml` file in the root of the project.

### Running with Docker

To build and run the application in a Docker container, use:
```bash
make docker-run
```

For a full development environment, including InfluxDB and Grafana, you can use Docker Compose:
```bash
make docker-compose-up
```

## Development Conventions

- **Linting:** The project uses `golangci-lint` for code linting. You can run the linter with `make lint`.
- **Testing:** The project has a suite of unit and integration tests. All new code should be accompanied by tests.
- **Documentation:** The `README.md` file is very comprehensive and should be kept up-to-date. The code is also well-documented with comments.
- **Dependencies:** The project uses Go modules for dependency management.
