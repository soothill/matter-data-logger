// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Matter Power Data Logger discovers Matter devices and logs their power consumption.
//
// This application automatically discovers Matter-compatible smart home devices on the
// local network via mDNS, monitors their power consumption in real-time, and stores
// the data in InfluxDB for analysis and visualization.
//
// # Application Architecture
//
// The application uses a concurrent, goroutine-based architecture:
//   - Main goroutine: Coordinates startup, shutdown, and periodic discovery
//   - HTTP server goroutine: Serves metrics and health endpoints
//   - Data writer goroutine: Consumes power readings and writes to InfluxDB
//   - Per-device monitor goroutines: Poll individual devices for power data
//   - Background health monitor: Checks InfluxDB availability and replays cache
//
// # Startup Flow
//
//  1. Parse command-line flags (config path, metrics port, health-check mode)
//  2. Load and validate configuration from YAML + environment variables
//  3. Initialize logger with configured log level
//  4. Initialize components:
//     - Slack notifier for alerts
//     - InfluxDB client with circuit breaker
//     - Local cache for offline resilience
//     - HTTP server with rate-limited endpoints
//  5. Start HTTP server for Prometheus metrics and health checks
//  6. Perform initial device discovery via mDNS
//  7. Start monitoring discovered power devices
//  8. Enter main loop for periodic discovery
//
// # Graceful Shutdown
//
// The application handles SIGTERM and SIGINT for graceful shutdown:
//  1. Signal received, cancel main context
//  2. HTTP server stops accepting new connections (5s timeout)
//  3. Power monitor stops all device polling goroutines
//  4. Data writer drains remaining readings channel
//  5. InfluxDB flush with timeout (10s)
//  6. Wait for all goroutines to finish
//  7. Close database connections
//  8. Exit cleanly
//
// # Configuration
//
// Configuration is loaded from config.yaml with environment variable overrides:
//   - InfluxDB connection (URL, token, org, bucket)
//   - Matter discovery settings (intervals, service type)
//   - Logging level
//   - Slack webhook URL for notifications
//   - Local cache settings
//
// See config/config.go for full configuration options.
//
// # HTTP Endpoints
//
// The application exposes three endpoints on localhost:9090 (configurable):
//
// GET /metrics - Prometheus metrics:
//   - Device discovery metrics
//   - Power reading counts
//   - InfluxDB write metrics
//   - Current power/voltage/current per device
//
// GET /health - Basic health check:
//   - Always returns 200 OK if application is running
//   - Rate limited: 10 req/sec with burst of 20
//
// GET /ready - Readiness check:
//   - Returns 200 READY if InfluxDB is healthy
//   - Returns 503 NOT READY if InfluxDB is down
//   - Rate limited: 10 req/sec with burst of 20
//   - Used by Kubernetes/Docker for deployment readiness
//
// # Local Cache and Failover
//
// When InfluxDB is unavailable:
//  1. Readings are automatically written to local JSON cache
//  2. Slack notification sent on first failure
//  3. Background monitor checks InfluxDB health every 30s
//  4. When healthy, cached readings are replayed in order
//  5. Slack recovery notification sent
//  6. Normal operation resumes
//
// # Security Features
//
//   - Rate limiting on health endpoints to prevent DoS
//   - Circuit breaker for InfluxDB to prevent cascade failures
//   - Metrics endpoint bound to localhost only (requires reverse proxy for external access)
//   - TLS validation for non-local InfluxDB connections
//   - Input sanitization for Flux queries
//   - Validation of negative power readings
//
// # Command-Line Usage
//
// Start application with default config:
//
//	./matter-data-logger
//
// Specify custom config file:
//
//	./matter-data-logger -config /path/to/config.yaml
//
// Custom metrics port:
//
//	./matter-data-logger -metrics-port 8080
//
// Health check mode (for Docker/K8s):
//
//	./matter-data-logger -health-check
//
// # Environment Variables
//
// Override configuration via environment variables:
//   - INFLUXDB_URL, INFLUXDB_TOKEN, INFLUXDB_ORG, INFLUXDB_BUCKET
//   - LOG_LEVEL
//   - MATTER_DISCOVERY_INTERVAL, MATTER_POLL_INTERVAL
//   - SLACK_WEBHOOK_URL
//   - CACHE_DIRECTORY
//
// See config/config.go for complete list.
//
// # Development
//
// Run tests:
//
//	make test
//
// Run integration tests (requires Docker):
//
//	make test-integration
//
// Run linter:
//
//	make lint
//
// Build:
//
//	make build
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soothill/matter-data-logger/config"
	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/logger"
	"github.com/soothill/matter-data-logger/pkg/metrics"
	"github.com/soothill/matter-data-logger/pkg/notifications"
	"github.com/soothill/matter-data-logger/storage"
	"golang.org/x/time/rate"
)

const (
	signalChannelSize      = 1
	discoveryTimeout       = 10 * time.Second
	alertContextTimeout    = 5 * time.Second
	readinessCheckTimeout  = 2 * time.Second
	shutdownTimeout        = 5 * time.Second
	flushTimeout           = 10 * time.Second
)

// rateLimitMiddleware wraps an HTTP handler with rate limiting
// Returns HTTP 429 (Too Many Requests) when the rate limit is exceeded
func rateLimitMiddleware(limiter *rate.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			logger.Warn().
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Msg("Rate limit exceeded for health endpoint")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// initializeComponents initializes all application components
// Returns an error instead of calling Fatal() to allow for testing
func initializeComponents(cfg *config.Config, metricsPort string) (*notifications.SlackNotifier, *storage.CachingStorage, *storage.InfluxDBStorage, *http.Server, error) {
	var err error

	// Initialize Slack notifier
	notifier := notifications.NewSlackNotifier(cfg.Notifications.SlackWebhookURL)
	if notifier.IsEnabled() {
		logger.Info().Msg("Slack notifications enabled")
	} else {
		logger.Info().Msg("Slack notifications disabled (no webhook URL configured)")
	}

	// Initialize InfluxDB storage
	var influxDB *storage.InfluxDBStorage
	influxDB, err = storage.NewInfluxDBStorage(
		cfg.InfluxDB.URL,
		cfg.InfluxDB.Token,
		cfg.InfluxDB.Organization,
		cfg.InfluxDB.Bucket,
	)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize InfluxDB: %w", err)
	}

	// Initialize local cache
	var cache *storage.LocalCache
	cache, err = storage.NewLocalCache(
		cfg.Cache.Directory,
		cfg.Cache.MaxSize,
		cfg.Cache.MaxAge,
	)
	if err != nil {
		influxDB.Close() // Clean up InfluxDB connection
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize local cache: %w", err)
	}
	logger.Info().Str("directory", cfg.Cache.Directory).
		Int64("max_size_mb", cfg.Cache.MaxSize/(1024*1024)).
		Dur("max_age", cfg.Cache.MaxAge).
		Msg("Local cache initialized")

	// Wrap InfluxDB storage with caching layer
	db := storage.NewCachingStorage(influxDB, cache, notifier)

	// Create rate limiters for health endpoints
	// 10 requests per second with burst of 20 to prevent abuse/DoS
	healthLimiter := rate.NewLimiter(10, 20)
	readyLimiter := rate.NewLimiter(10, 20)

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", rateLimitMiddleware(healthLimiter, healthCheckHandler))
	mux.HandleFunc("/ready", rateLimitMiddleware(readyLimiter, func(w http.ResponseWriter, r *http.Request) {
		readinessCheckHandler(w, r, influxDB)
	}))

	// Create HTTP server with localhost binding for security
	// Bind to localhost to prevent exposing metrics to external networks
	// If you need external access, configure a reverse proxy with authentication
	server := &http.Server{
		Addr:    "localhost:" + metricsPort,
		Handler: mux,
	}

	return notifier, db, influxDB, server, nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	metricsPort := flag.String("metrics-port", "9090", "Port for Prometheus metrics endpoint")
	healthCheck := flag.Bool("health-check", false, "Perform health check and exit")
	validateConfig := flag.Bool("validate-config", false, "Validate configuration file and exit")
	flag.Parse()

	// If health-check flag is set, perform check and exit
	if *healthCheck {
		os.Exit(performHealthCheck())
	}

	// If validate-config flag is set, validate configuration and exit
	if *validateConfig {
		os.Exit(performConfigValidation(*configPath))
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Initialize("error")
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger with configured level
	logger.Initialize(cfg.Logging.Level)

	logger.Info().Msg("Starting Matter Power Data Logger")
	logger.Info().Dur("discovery_interval", cfg.Matter.DiscoveryInterval).
		Dur("poll_interval", cfg.Matter.PollInterval).
		Msg("Configuration loaded")

	// Initialize components
	notifier, db, influxDB, server, err := initializeComponents(cfg, *metricsPort)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize components")
	}
	defer influxDB.Close()
	defer db.Close()

	// WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Start metrics HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info().Str("addr", server.Addr).Msg("Starting metrics and health check server (localhost only)")
		if serverErr := server.ListenAndServe(); serverErr != nil && serverErr != http.ErrServerClosed {
			logger.Error().Err(serverErr).Msg("Metrics server failed")
		}
	}()

	// Create device scanner
	scanner := discovery.NewScanner(cfg.Matter.ServiceType, cfg.Matter.Domain)

	// Create power monitor with scanner reference for fresh device info
	monitor := monitoring.NewPowerMonitor(cfg.Matter.PollInterval, scanner, cfg.Matter.ReadingsChannelSize)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, signalChannelSize)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		performGracefulShutdown(server, monitor, cancel)
	}()

	// Start data writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				logger.Info().Msg("Data writer goroutine shutting down")
				return
			case reading, ok := <-monitor.Readings():
				if !ok {
					logger.Info().Msg("Readings channel closed, data writer exiting")
					return
				}
				writeErr := db.WriteReading(ctx, reading)
				if writeErr != nil {
					logger.Error().Err(writeErr).Str("device_id", reading.DeviceID).
						Msg("Failed to write reading to InfluxDB")
					metrics.InfluxDBWriteErrors.Inc()
				} else {
					metrics.InfluxDBWritesTotal.Inc()
					metrics.CurrentPower.WithLabelValues(reading.DeviceID, reading.DeviceName).Set(reading.Power)
					metrics.CurrentVoltage.WithLabelValues(reading.DeviceID, reading.DeviceName).Set(reading.Voltage)
					metrics.CurrentCurrent.WithLabelValues(reading.DeviceID, reading.DeviceName).Set(reading.Current)
				}
			}
		}
	}()

	// Initial device discovery
	performInitialDiscovery(ctx, scanner, monitor, notifier)

	// Periodic device discovery
	discoveryTicker := time.NewTicker(cfg.Matter.DiscoveryInterval)
	defer discoveryTicker.Stop()

	// Main loop
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Shutting down")
			performCleanup(db, &wg)
			return

		case <-discoveryTicker.C:
			// Check context before discovery operation
			if ctx.Err() != nil {
				return
			}
			performPeriodicDiscovery(ctx, scanner, monitor, notifier)
		}
	}
}

// performInitialDiscovery performs the initial device discovery and starts monitoring
func performInitialDiscovery(ctx context.Context, scanner *discovery.Scanner, monitor *monitoring.PowerMonitor, notifier *notifications.SlackNotifier) {
	logger.Info().Msg("Performing initial device discovery")
	start := time.Now()
	devices, discoverErr := scanner.Discover(ctx, discoveryTimeout)
	metrics.DiscoveryDuration.Observe(time.Since(start).Seconds())

	if discoverErr != nil {
		logger.Error().Err(discoverErr).Msg("Initial discovery failed")
		// Send Slack alert for discovery failure
		if notifier != nil && notifier.IsEnabled() {
			alertCtx, alertCancel := context.WithTimeout(context.Background(), alertContextTimeout)
			defer alertCancel()
			if notifyErr := notifier.SendDiscoveryFailure(alertCtx, discoverErr); notifyErr != nil {
				logger.Error().Err(notifyErr).Msg("Failed to send discovery failure alert")
			}
		}
	} else {
		logger.Info().Int("count", len(devices)).Msg("Discovered Matter devices")
		metrics.DevicesDiscovered.Set(float64(len(scanner.GetDevices())))
	}

	// Start monitoring power devices
	powerDevices := scanner.GetPowerDevices()
	metrics.PowerDevicesDiscovered.Set(float64(len(powerDevices)))
	logger.Info().Int("count", len(powerDevices)).Msg("Found devices with power measurement capability")

	if len(powerDevices) > 0 {
		monitor.Start(ctx, powerDevices)
		metrics.DevicesMonitored.Set(float64(monitor.GetMonitoredDeviceCount()))
	} else {
		logger.Warn().Msg("No power monitoring devices found. Will retry during periodic discovery")
	}
}

// performPeriodicDiscovery performs periodic device discovery and starts monitoring new devices
func performPeriodicDiscovery(ctx context.Context, scanner *discovery.Scanner, monitor *monitoring.PowerMonitor, notifier *notifications.SlackNotifier) {
	logger.Info().Msg("Performing periodic device discovery")
	start := time.Now()
	newDevices, discoverErr := scanner.Discover(ctx, discoveryTimeout)
	metrics.DiscoveryDuration.Observe(time.Since(start).Seconds())

	if discoverErr != nil {
		logger.Error().Err(discoverErr).Msg("Discovery failed")
		// Send Slack alert for discovery failure (only log, don't block periodic discovery)
		if notifier != nil && notifier.IsEnabled() {
			alertCtx, alertCancel := context.WithTimeout(context.Background(), alertContextTimeout)
			defer alertCancel()
			if notifyErr := notifier.SendDiscoveryFailure(alertCtx, discoverErr); notifyErr != nil {
				logger.Error().Err(notifyErr).Msg("Failed to send discovery failure alert")
			}
		}
		return
	}

	allDevices := scanner.GetDevices()
	logger.Info().Int("total_devices", len(allDevices)).Int("new_devices", len(newDevices)).
		Msg("Discovery complete")
	metrics.DevicesDiscovered.Set(float64(len(allDevices)))

	// Check for new power devices and start monitoring only new ones
	powerDevices := scanner.GetPowerDevices()
	metrics.PowerDevicesDiscovered.Set(float64(len(powerDevices)))

	if len(newDevices) > 0 {
		for _, device := range newDevices {
			if device.HasPowerMeasurement() && !monitor.IsMonitoring(device.GetDeviceID()) {
				logger.Info().Str("device_id", device.GetDeviceID()).
					Str("device_name", device.Name).
					Msg("Starting monitoring for new power device")
				monitor.StartMonitoringDevice(ctx, device)
			}
		}
		metrics.DevicesMonitored.Set(float64(monitor.GetMonitoredDeviceCount()))
	}
}

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write([]byte("OK")); writeErr != nil {
		logger.Error().Err(writeErr).Msg("Failed to write health check response")
	}
}

// readinessCheckHandler handles readiness check requests
func readinessCheckHandler(w http.ResponseWriter, _ *http.Request, db *storage.InfluxDBStorage) {
	// Check InfluxDB connection
	ctx, cancel := context.WithTimeout(context.Background(), readinessCheckTimeout)
	defer cancel()

	if healthErr := db.Health(ctx); healthErr != nil {
		logger.Warn().Err(healthErr).Msg("Readiness check failed: InfluxDB unhealthy")
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, writeErr := w.Write([]byte("NOT READY: InfluxDB unhealthy")); writeErr != nil {
			logger.Error().Err(writeErr).Msg("Failed to write readiness check response")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write([]byte("READY")); writeErr != nil {
		logger.Error().Err(writeErr).Msg("Failed to write readiness check response")
	}
}

// performHealthCheck performs a health check and returns exit code
func performHealthCheck() int {
	// Simple health check for Docker/K8s - just check if we can start
	// In a more sophisticated implementation, this could check connectivity
	return 0
}

// performConfigValidation validates the configuration file and returns exit code
// Returns 0 if configuration is valid, 1 if invalid
func performConfigValidation(configPath string) int {
	// Initialize logger for validation output
	logger.Initialize("info")

	logger.Info().Str("path", configPath).Msg("Validating configuration file")

	// Attempt to load and validate configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error().Err(err).Msg("Configuration validation failed")
		fmt.Fprintf(os.Stderr, "\n❌ Configuration validation FAILED\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintf(os.Stderr, "Please check your configuration file and fix the errors above.\n")
		return 1
	}

	// Configuration is valid - print summary
	logger.Info().Msg("Configuration validation successful")
	fmt.Println("\n✅ Configuration validation PASSED")
	fmt.Println("\nConfiguration summary:")
	fmt.Printf("  InfluxDB URL: %s\n", cfg.InfluxDB.URL)
	fmt.Printf("  InfluxDB Organization: %s\n", cfg.InfluxDB.Organization)
	fmt.Printf("  InfluxDB Bucket: %s\n", cfg.InfluxDB.Bucket)
	fmt.Printf("  Log Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  Discovery Interval: %s\n", cfg.Matter.DiscoveryInterval)
	fmt.Printf("  Poll Interval: %s\n", cfg.Matter.PollInterval)
	fmt.Printf("  Service Type: %s\n", cfg.Matter.ServiceType)
	fmt.Printf("  Domain: %s\n", cfg.Matter.Domain)
	fmt.Printf("  Readings Channel Size: %d\n", cfg.Matter.ReadingsChannelSize)
	fmt.Printf("  Cache Directory: %s\n", cfg.Cache.Directory)
	fmt.Printf("  Cache Max Size: %d MB\n", cfg.Cache.MaxSize/(1024*1024))
	fmt.Printf("  Cache Max Age: %s\n", cfg.Cache.MaxAge)

	if cfg.Notifications.SlackWebhookURL != "" {
		fmt.Println("  Slack Notifications: Enabled")
	} else {
		fmt.Println("  Slack Notifications: Disabled")
	}

	fmt.Println("\nAll validation checks passed. Configuration is ready for use.")
	return 0
}

// performGracefulShutdown handles graceful shutdown of all components
func performGracefulShutdown(server *http.Server, monitor *monitoring.PowerMonitor, cancel context.CancelFunc) {
	logger.Info().Msg("Initiating graceful shutdown...")

	// Shutdown HTTP server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	} else {
		logger.Info().Msg("HTTP server stopped")
	}

	// Stop power monitoring
	monitor.Stop()

	// Cancel main context
	cancel()
}

// performCleanup flushes database and waits for goroutines to finish
func performCleanup(db *storage.CachingStorage, wg *sync.WaitGroup) {
	// Flush InfluxDB with timeout
	flushCtx, flushCancel := context.WithTimeout(context.Background(), flushTimeout)
	defer flushCancel()

	// Note: Current InfluxDB client Flush() doesn't accept context
	// This is a known limitation - wrapping in goroutine with timeout
	flushDone := make(chan struct{})
	go func() {
		db.Flush()
		close(flushDone)
	}()

	select {
	case <-flushDone:
		logger.Info().Msg("InfluxDB flush completed")
	case <-flushCtx.Done():
		logger.Warn().Msg("InfluxDB flush timeout - some data may be lost")
	}

	// Wait for all goroutines to finish
	logger.Info().Msg("Waiting for goroutines to finish...")
	wg.Wait()
	logger.Info().Msg("All goroutines finished, exiting")
}
