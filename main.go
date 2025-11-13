// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soothill/matter-data-logger/config"
	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/interfaces"
	"github.com/soothill/matter-data-logger/pkg/logger"
	"github.com/soothill/matter-data-logger/pkg/metrics"
	"github.com/soothill/matter-data-logger/pkg/slacknotifier"
	"github.com/soothill/matter-data-logger/storage"
	"golang.org/x/time/rate"
)

const (
	signalChannelSize     = 1
	discoveryTimeout      = 10 * time.Second
	alertContextTimeout   = 5 * time.Second
	readinessCheckTimeout = 2 * time.Second
	shutdownTimeout       = 5 * time.Second
	flushTimeout          = 10 * time.Second
)

// App represents the main application
type App struct {
	cfg           *config.Config
	metricsPort   string
	server        *http.Server
	monitor       *monitoring.PowerMonitor
	scanner       *discovery.Scanner
	db            *storage.CachingStorage
	influxDB      interfaces.TimeSeriesStorage // Changed to interface
	notifier      *slacknotifier.Notifier
	configWatcher *config.Watcher
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	metricsPort := flag.String("metrics-port", "9090", "Port for Prometheus metrics endpoint")
	healthCheck := flag.Bool("health-check", false, "Perform health check and exit")
	validateConfig := flag.Bool("validate-config", false, "Validate configuration file and exit")
	flag.Parse()

	if *healthCheck {
		os.Exit(performHealthCheck(*configPath))
	}

	if *validateConfig {
		os.Exit(performConfigValidation(*configPath))
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Initialize("error")
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Initialize(cfg.Logging.Level)

	logger.Info().Msg("Starting Matter Power Data Logger")
	logger.Info().Dur("discovery_interval", cfg.Matter.DiscoveryInterval).
		Dur("poll_interval", cfg.Matter.PollInterval).
		Msg("Configuration loaded")

	configChan := make(chan *config.Config)
	configWatcher := config.NewWatcher(*configPath, configChan)

	application, err := New(cfg, *metricsPort, configWatcher)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create application")
	}

	application.Run(configChan)

	// Perform initial discovery
	application.DiscoverAndMonitor(context.Background())

	setupDebugSignalHandlers(application)
}

// New creates a new application instance
func New(cfg *config.Config, metricsPort string, configWatcher *config.Watcher) (*App, error) {
	app := &App{
		cfg:           cfg,
		metricsPort:   metricsPort,
		configWatcher: configWatcher,
	}

	var err error
	app.notifier, app.db, app.influxDB, app.server, err = app.initializeComponents()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	app.scanner = discovery.NewScanner(cfg.Matter.ServiceType, cfg.Matter.Domain)
	app.monitor = monitoring.NewPowerMonitor(cfg.Matter.PollInterval, app.scanner, cfg.Matter.ReadingsChannelSize)

	return app, nil
}

// Run starts the application and blocks until shutdown
func (a *App) Run(configChan <-chan *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.cancel = cancel
	defer a.cancel()

	a.configWatcher.Start(ctx)
	defer a.configWatcher.Stop()

	a.startMetricsServer()
	a.setupSignalHandler()
	a.startConfigWatcher(configChan)
	a.startDataWriter(ctx)
	a.DiscoverAndMonitor(ctx)
	a.runMainLoop(ctx)
}

// initializeComponents initializes all application components
func (a *App) initializeComponents() (*slacknotifier.Notifier, *storage.CachingStorage, interfaces.TimeSeriesStorage, *http.Server, error) {
	var err error

	// Initialize Slack notifier
	notifier := slacknotifier.New(a.cfg.Notifications.SlackWebhookURL)
	if notifier.IsEnabled() {
		logger.Info().Msg("Slack notifications enabled")
	} else {
		logger.Info().Msg("Slack notifications disabled (no webhook URL configured)")
	}
	notifierAdapter := slacknotifier.NewAdapter(notifier)

	// Initialize InfluxDB storage
	var influxDB *storage.InfluxDBStorage
	influxDB, err = storage.NewInfluxDBStorage(
		a.cfg.InfluxDB.URL,
		a.cfg.InfluxDB.Token,
		a.cfg.InfluxDB.Organization,
		a.cfg.InfluxDB.Bucket,
	)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize InfluxDB: %w", err)
	}

	// Initialize local cache
	var cache *storage.LocalCache
	cache, err = storage.NewLocalCache(
		a.cfg.Cache.Directory,
		a.cfg.Cache.MaxSize,
		a.cfg.Cache.MaxAge,
	)
	if err != nil {
		influxDB.Close() // Clean up InfluxDB connection
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize local cache: %w", err)
	}
	logger.Info().Str("directory", a.cfg.Cache.Directory).
		Int64("max_size_mb", a.cfg.Cache.MaxSize/(1024*1024)).
		Dur("max_age", a.cfg.Cache.MaxAge).
		Msg("Local cache initialized")

	// Wrap InfluxDB storage with caching layer
	db := storage.NewCachingStorage(influxDB, cache, notifierAdapter)

	// Create rate limiters for health endpoints
	healthLimiter := rate.NewLimiter(10, 20)
	readyLimiter := rate.NewLimiter(10, 20)

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", rateLimitMiddleware(healthLimiter, healthCheckHandler))
	mux.HandleFunc("/ready", rateLimitMiddleware(readyLimiter, func(w http.ResponseWriter, r *http.Request) {
		readinessCheckHandler(w, r, influxDB)
	}))

	server := &http.Server{
		Addr:    "localhost:" + a.metricsPort,
		Handler: mux,
	}

	return notifier, db, influxDB, server, nil
}

// startMetricsServer starts the HTTP server for metrics and health checks
func (a *App) startMetricsServer() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		logger.Info().Str("addr", a.server.Addr).Msg("Starting metrics and health check server (localhost only)")
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("Metrics server failed")
		}
	}()
}

// startDataWriter starts the goroutine that writes power readings to the database
func (a *App) startDataWriter(ctx context.Context) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case <-ctx.Done():
				logger.Info().Msg("Data writer goroutine shutting down")
				return
			case reading, ok := <-a.monitor.Readings():
				if !ok {
					logger.Info().Msg("Readings channel closed, data writer exiting")
					return
				}
				writeErr := a.db.WriteReading(ctx, reading)
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
}

// setupSignalHandler sets up graceful shutdown on interrupt signals
func (a *App) setupSignalHandler() {
	sigChan := make(chan os.Signal, signalChannelSize)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		a.performGracefulShutdown()
	}()
}

// DumpApplicationState dumps current application state to logs
func (a *App) DumpApplicationState() {
	logger.Info().Msg("=== APPLICATION STATE DUMP (SIGUSR1) ===")

	allDevices := a.scanner.GetDevices()
	powerDevices := a.scanner.GetPowerDevices()
	logger.Info().
		Int("total_devices", len(allDevices)).
		Int("power_devices", len(powerDevices)).
		Msg("Device discovery state")

	for _, device := range allDevices {
		logger.Info().
			Str("device_id", device.GetDeviceID()).
			Str("device_name", device.Name).
			Str("address", device.Address.String()).
			Int("port", device.Port).
			Bool("has_power_measurement", device.HasPowerMeasurement()).
			Msg("Discovered device")
	}

	monitoredCount := a.monitor.GetMonitoredDeviceCount()
	logger.Info().
		Int("monitored_devices", monitoredCount).
		Msg("Monitoring state")

	for _, device := range powerDevices {
		deviceID := device.GetDeviceID()
		isMonitoring := a.monitor.IsMonitoring(deviceID)
		logger.Info().
			Str("device_id", deviceID).
			Str("device_name", device.Name).
			Bool("is_monitoring", isMonitoring).
			Msg("Power device monitoring status")
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.Info().
		Uint64("alloc_mb", m.Alloc/1024/1024).
		Uint64("total_alloc_mb", m.TotalAlloc/1024/1024).
		Uint32("num_gc", m.NumGC).
		Int("num_goroutines", runtime.NumGoroutine()).
		Msg("Runtime statistics")

	logger.Info().Msg("=== END STATE DUMP ===")
}

// DumpGoroutineStackTraces dumps all goroutine stack traces to logs
func DumpGoroutineStackTraces() {
	logger.Info().Msg("=== GOROUTINE STACK TRACES (SIGUSR2) ===")
	logger.Info().Int("num_goroutines", runtime.NumGoroutine()).Msg("Current goroutine count")

	buf := make([]byte, 1024*1024) // 1MB buffer
	stackLen := runtime.Stack(buf, true)
	logger.Info().Str("stack_traces", string(buf[:stackLen])).Msg("Full stack trace")

	logger.Info().Msg("=== END STACK TRACES ===")
}

// runMainLoop runs the main discovery loop
func (a *App) runMainLoop(ctx context.Context) {
	discoveryTicker := time.NewTicker(a.cfg.Matter.DiscoveryInterval)
	defer discoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Shutting down")
			a.performCleanup()
			return
		case <-discoveryTicker.C:
			if ctx.Err() != nil {
				return
			}
			a.DiscoverAndMonitor(ctx)
		}
	}
}

// DiscoverAndMonitor performs a device discovery and starts monitoring new devices.
func (a *App) DiscoverAndMonitor(ctx context.Context) {
	logger.Info().Msg("Performing device discovery")
	start := time.Now()
	newDevices, discoverErr := a.scanner.Discover(ctx, discoveryTimeout)
	metrics.DiscoveryDuration.Observe(time.Since(start).Seconds())

	if discoverErr != nil {
		logger.Error().Err(discoverErr).Msg("Discovery failed")
		if a.notifier != nil && a.notifier.IsEnabled() {
			alertCtx, alertCancel := context.WithTimeout(context.Background(), alertContextTimeout)
			defer alertCancel()
			if notifyErr := sendDiscoveryFailure(alertCtx, a.notifier, discoverErr); notifyErr != nil {
				logger.Error().Err(notifyErr).Msg("Failed to send discovery failure alert")
			}
		}
		return
	}

	allDevices := a.scanner.GetDevices()
	logger.Info().Int("total_devices", len(allDevices)).Int("new_devices", len(newDevices)).
		Msg("Discovery complete")
	metrics.DevicesDiscovered.Set(float64(len(allDevices)))

	powerDevices := a.scanner.GetPowerDevices()
	metrics.PowerDevicesDiscovered.Set(float64(len(powerDevices)))

	if len(newDevices) > 0 {
		for _, device := range newDevices {
			if device.HasPowerMeasurement() && !a.monitor.IsMonitoring(device.GetDeviceID()) {
				logger.Info().Str("device_id", device.GetDeviceID()).
					Str("device_name", device.Name).
					Msg("Starting monitoring for new power device")
				a.monitor.StartMonitoringDevice(ctx, device)
			}
		}
		metrics.DevicesMonitored.Set(float64(a.monitor.GetMonitoredDeviceCount()))
	}
}

// performGracefulShutdown handles graceful shutdown of all components
func (a *App) performGracefulShutdown() {
	logger.Info().Msg("Initiating graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	} else {
		logger.Info().Msg("HTTP server stopped")
	}

	a.monitor.Stop()
	a.configWatcher.Stop()
	a.cancel()
}

// performCleanup flushes database and waits for goroutines to finish
func (a *App) performCleanup() {
	flushCtx, flushCancel := context.WithTimeout(context.Background(), flushTimeout)
	defer flushCancel()

	flushDone := make(chan struct{})
	go func() {
		a.db.Flush()
		close(flushDone)
	}()

	select {
	case <-flushDone:
		logger.Info().Msg("InfluxDB flush completed")
	case <-flushCtx.Done():
		logger.Warn().Msg("InfluxDB flush timeout - some data may be lost")
	}

	logger.Info().Msg("Waiting for goroutines to finish...")
	a.wg.Wait()
	logger.Info().Msg("All goroutines finished, exiting")
}

// UpdateConfig updates the application's configuration.
func (a *App) UpdateConfig(newCfg *config.Config) {
	a.cfg = newCfg
	logger.Info().Msg("Application configuration updated")

	// Reconfigure components that depend on dynamic config values
	a.monitor.UpdatePollInterval(a.cfg.Matter.PollInterval)
	a.notifier.UpdateWebhookURL(a.cfg.Notifications.SlackWebhookURL)
	logger.Info().Dur("new_poll_interval", a.cfg.Matter.PollInterval).Msg("Monitor poll interval updated")
}

// startConfigWatcher starts a goroutine to listen for config file changes and reloads
func (a *App) startConfigWatcher(configChan <-chan *config.Config) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case <-a.ctx.Done():
				logger.Info().Msg("Config watcher goroutine shutting down")
				return
			case newCfg := <-configChan:
				a.UpdateConfig(newCfg)
			}
		}
	}()
}

// rateLimitMiddleware wraps an HTTP handler with rate limiting
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

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write([]byte("OK")); writeErr != nil {
		logger.Error().Err(writeErr).Msg("Failed to write health check response")
	}
}

// readinessCheckHandler handles readiness check requests
func readinessCheckHandler(w http.ResponseWriter, _ *http.Request, db interfaces.TimeSeriesStorage) {
	ctx, cancel := context.WithTimeout(context.Background(), readinessCheckTimeout)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		logger.Warn().Err(err).Msg("Readiness check failed: InfluxDB unhealthy")
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
func performHealthCheck(configPath string) int {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: could not load config: %v\n", err)
		return 1
	}

	influxDB, err := storage.NewInfluxDBStorage(
		cfg.InfluxDB.URL,
		cfg.InfluxDB.Token,
		cfg.InfluxDB.Organization,
		cfg.InfluxDB.Bucket,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: could not create InfluxDB client: %v\n", err)
		return 1
	}
	defer influxDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := influxDB.Health(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: InfluxDB is unhealthy: %v\n", err)
		return 1
	}

	fmt.Println("Health check passed: InfluxDB is healthy")
	return 0
}

// performConfigValidation validates the configuration file and returns exit code
func performConfigValidation(configPath string) int {
	logger.Initialize("info")
	logger.Info().Str("path", configPath).Msg("Validating configuration file")

	if err := config.ValidateWithSchema(configPath); err != nil {
		logger.Error().Err(err).Msg("Configuration schema validation failed")
		fmt.Fprintf(os.Stderr, "\n❌ Configuration validation FAILED\n")
		return 1
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error().Err(err).Msg("Configuration validation failed")
		fmt.Fprintf(os.Stderr, "\n❌ Configuration validation FAILED\n")
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		return 1
	}

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

func sendDiscoveryFailure(ctx context.Context, notifier *slacknotifier.Notifier, err error) error {
	return notifier.SendAlert(ctx, "warning", "⚠️ Device Discovery Failure",
		fmt.Sprintf("Failed to discover Matter devices: %v", err))
}
