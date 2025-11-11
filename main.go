// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Matter Power Data Logger discovers Matter devices and logs their power consumption.
package main

import (
	"context"
	"flag"
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
	"github.com/soothill/matter-data-logger/storage"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	metricsPort := flag.String("metrics-port", "9090", "Port for Prometheus metrics endpoint")
	healthCheck := flag.Bool("health-check", false, "Perform health check and exit")
	flag.Parse()

	// If health-check flag is set, perform check and exit
	if *healthCheck {
		os.Exit(performHealthCheck())
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

	// Initialize InfluxDB storage
	var db *storage.InfluxDBStorage
	db, err = storage.NewInfluxDBStorage(
		cfg.InfluxDB.URL,
		cfg.InfluxDB.Token,
		cfg.InfluxDB.Organization,
		cfg.InfluxDB.Bucket,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize InfluxDB")
	}
	defer db.Close()

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessCheckHandler(w, r, db)
	})

	// Create HTTP server with localhost binding for security
	// Bind to localhost to prevent exposing metrics to external networks
	// If you need external access, configure a reverse proxy with authentication
	server := &http.Server{
		Addr:    "localhost:" + *metricsPort,
		Handler: mux,
	}

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

	// Create power monitor
	monitor := monitoring.NewPowerMonitor(cfg.Matter.PollInterval)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
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
				writeErr := db.WriteReading(reading)
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
	performInitialDiscovery(ctx, scanner, monitor)

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
			performPeriodicDiscovery(ctx, scanner, monitor)
		}
	}
}

// performInitialDiscovery performs the initial device discovery and starts monitoring
func performInitialDiscovery(ctx context.Context, scanner *discovery.Scanner, monitor *monitoring.PowerMonitor) {
	logger.Info().Msg("Performing initial device discovery")
	start := time.Now()
	devices, discoverErr := scanner.Discover(ctx, 10*time.Second)
	metrics.DiscoveryDuration.Observe(time.Since(start).Seconds())

	if discoverErr != nil {
		logger.Error().Err(discoverErr).Msg("Initial discovery failed")
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
func performPeriodicDiscovery(ctx context.Context, scanner *discovery.Scanner, monitor *monitoring.PowerMonitor) {
	logger.Info().Msg("Performing periodic device discovery")
	start := time.Now()
	newDevices, discoverErr := scanner.Discover(ctx, 10*time.Second)
	metrics.DiscoveryDuration.Observe(time.Since(start).Seconds())

	if discoverErr != nil {
		logger.Error().Err(discoverErr).Msg("Discovery failed")
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

// performGracefulShutdown handles graceful shutdown of all components
func performGracefulShutdown(server *http.Server, monitor *monitoring.PowerMonitor, cancel context.CancelFunc) {
	logger.Info().Msg("Initiating graceful shutdown...")

	// Shutdown HTTP server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
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
func performCleanup(db *storage.InfluxDBStorage, wg *sync.WaitGroup) {
	// Flush InfluxDB with timeout
	flushCtx, flushCancel := context.WithTimeout(context.Background(), 10*time.Second)
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
