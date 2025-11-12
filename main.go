// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/soothill/matter-data-logger/app"
	"github.com/soothill/matter-data-logger/config"
	"github.com/soothill/matter-data-logger/pkg/logger"
	"github.com/soothill/matter-data-logger/storage"
)

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

	application, err := app.New(cfg, *metricsPort, *configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create application")
	}

	application.Run()

	setupDebugSignalHandlers(application)
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
