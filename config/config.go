// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package config provides configuration management for the Matter data logger.
//
// This package handles loading, validating, and managing application configuration
// from YAML files with environment variable overrides. It supports comprehensive
// validation of all configuration parameters to ensure safe operation.
//
// # Configuration Sources
//
// Configuration is loaded in the following order of precedence:
//  1. YAML configuration file (default: config.yaml)
//  2. Environment variable overrides
//  3. Default values for optional settings
//
// # Environment Variables
//
// The following environment variables can override YAML configuration:
//   - INFLUXDB_URL: InfluxDB server URL
//   - INFLUXDB_TOKEN: InfluxDB authentication token
//   - INFLUXDB_ORG: InfluxDB organization name
//   - INFLUXDB_BUCKET: InfluxDB bucket name
//   - LOG_LEVEL: Logging level (debug, info, warn, error, fatal, panic)
//   - MATTER_DISCOVERY_INTERVAL: Device discovery interval (e.g., "5m")
//   - MATTER_POLL_INTERVAL: Power reading poll interval (e.g., "30s")
//   - SLACK_WEBHOOK_URL: Slack webhook URL for notifications
//   - CACHE_DIRECTORY: Local cache directory path
//
// # Security Features
//
// The configuration system includes several security validations:
//   - HTTPS enforcement for non-local InfluxDB connections
//   - Minimum token length validation (8 characters)
//   - URL format validation
//   - Sensible limits on intervals and buffer sizes
//
// # Example Usage
//
//	cfg, err := config.Load("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Configuration is validated and ready to use
//	fmt.Printf("InfluxDB: %s\n", cfg.InfluxDB.URL)
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	InfluxDB      InfluxDBConfig      `yaml:"influxdb"`
	Matter        MatterConfig        `yaml:"matter"`
	Logging       LoggingConfig       `yaml:"logging"`
	Notifications NotificationsConfig `yaml:"notifications"`
	Cache         CacheConfig         `yaml:"cache"`
}

// InfluxDBConfig holds InfluxDB connection settings
type InfluxDBConfig struct {
	URL          string `yaml:"url"`
	Token        string `yaml:"token"`
	Organization string `yaml:"organization"`
	Bucket       string `yaml:"bucket"`
}

// MatterConfig holds Matter device discovery settings
type MatterConfig struct {
	DiscoveryInterval    time.Duration `yaml:"discovery_interval"`
	PollInterval         time.Duration `yaml:"poll_interval"`
	ServiceType          string        `yaml:"service_type"`
	Domain               string        `yaml:"domain"`
	ReadingsChannelSize  int           `yaml:"readings_channel_size"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// NotificationsConfig holds notification settings
type NotificationsConfig struct {
	SlackWebhookURL string `yaml:"slack_webhook_url"`
}

// CacheConfig holds local cache settings
type CacheConfig struct {
	Directory string        `yaml:"directory"`
	MaxSize   int64         `yaml:"max_size"` // bytes
	MaxAge    time.Duration `yaml:"max_age"`
}

// Load reads configuration from a YAML file and applies environment variable overrides
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides and defaults
	cfg.applyEnvironmentOverrides()
	cfg.setDefaults()

	// Validate configuration
	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// applyEnvironmentOverrides applies environment variable overrides to the configuration
func (c *Config) applyEnvironmentOverrides() {
	if url := os.Getenv("INFLUXDB_URL"); url != "" {
		c.InfluxDB.URL = url
	}
	if token := os.Getenv("INFLUXDB_TOKEN"); token != "" {
		c.InfluxDB.Token = token
	}
	if org := os.Getenv("INFLUXDB_ORG"); org != "" {
		c.InfluxDB.Organization = org
	}
	if bucket := os.Getenv("INFLUXDB_BUCKET"); bucket != "" {
		c.InfluxDB.Bucket = bucket
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if interval := os.Getenv("MATTER_DISCOVERY_INTERVAL"); interval != "" {
		duration, parseErr := time.ParseDuration(interval)
		if parseErr == nil {
			c.Matter.DiscoveryInterval = duration
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse MATTER_DISCOVERY_INTERVAL '%s': %v\n", interval, parseErr)
		}
	}
	if interval := os.Getenv("MATTER_POLL_INTERVAL"); interval != "" {
		duration, parseErr := time.ParseDuration(interval)
		if parseErr == nil {
			c.Matter.PollInterval = duration
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse MATTER_POLL_INTERVAL '%s': %v\n", interval, parseErr)
		}
	}
	if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
		c.Notifications.SlackWebhookURL = webhookURL
	}
	if cacheDir := os.Getenv("CACHE_DIRECTORY"); cacheDir != "" {
		c.Cache.Directory = cacheDir
	}
}

// setDefaults sets default values for configuration fields if not provided
func (c *Config) setDefaults() {
	if c.Matter.DiscoveryInterval == 0 {
		c.Matter.DiscoveryInterval = 5 * time.Minute
	}
	if c.Matter.PollInterval == 0 {
		c.Matter.PollInterval = 30 * time.Second
	}
	if c.Matter.ServiceType == "" {
		c.Matter.ServiceType = "_matter._tcp"
	}
	if c.Matter.Domain == "" {
		c.Matter.Domain = "local."
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Cache.Directory == "" {
		c.Cache.Directory = "/var/cache/matter-data-logger"
	}
	if c.Cache.MaxSize == 0 {
		c.Cache.MaxSize = 100 * 1024 * 1024 // 100 MB
	}
	if c.Cache.MaxAge == 0 {
		c.Cache.MaxAge = 24 * time.Hour
	}
	if c.Matter.ReadingsChannelSize == 0 {
		c.Matter.ReadingsChannelSize = 100
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if validateErr := c.validateInfluxDB(); validateErr != nil {
		return validateErr
	}

	if validateErr := c.validateMatter(); validateErr != nil {
		return validateErr
	}

	if validateErr := c.validateLogging(); validateErr != nil {
		return validateErr
	}

	return nil
}

// validateInfluxDB validates the InfluxDB configuration
func (c *Config) validateInfluxDB() error {
	if c.InfluxDB.URL == "" {
		return fmt.Errorf("influxdb.url is required")
	}

	// Validate URL format and security
	parsedURL, parseErr := url.Parse(c.InfluxDB.URL)
	if parseErr != nil {
		return fmt.Errorf("influxdb.url is not a valid URL: %w", parseErr)
	}

	// Check for HTTPS in production-like URLs (not localhost/127.0.0.1)
	if securityErr := validateURLSecurity(parsedURL); securityErr != nil {
		return securityErr
	}

	if c.InfluxDB.Token == "" {
		return fmt.Errorf("influxdb.token is required")
	}

	// Validate token format (basic check for minimum length)
	if len(c.InfluxDB.Token) < 8 {
		return fmt.Errorf("influxdb.token must be at least 8 characters long")
	}

	if c.InfluxDB.Organization == "" {
		return fmt.Errorf("influxdb.organization is required")
	}
	if c.InfluxDB.Bucket == "" {
		return fmt.Errorf("influxdb.bucket is required")
	}

	return nil
}

// validateURLSecurity checks if the URL uses HTTPS for non-local connections
func validateURLSecurity(parsedURL *url.URL) error {
	if parsedURL.Scheme != "http" {
		return nil
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	isLocal := hostname == "localhost" ||
		hostname == "127.0.0.1" ||
		hostname == "::1" ||
		strings.HasPrefix(hostname, "192.168.") ||
		strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.")

	if !isLocal {
		return fmt.Errorf("influxdb.url must use HTTPS for non-local connections (got %s). Using HTTP transmits credentials in plaintext and is a security risk", parsedURL.Scheme)
	}

	return nil
}

// validateMatter validates the Matter configuration
func (c *Config) validateMatter() error {
	if c.Matter.DiscoveryInterval < time.Second {
		return fmt.Errorf("matter.discovery_interval must be at least 1 second")
	}
	if c.Matter.DiscoveryInterval > 24*time.Hour {
		return fmt.Errorf("matter.discovery_interval must not exceed 24 hours")
	}
	if c.Matter.PollInterval < time.Second {
		return fmt.Errorf("matter.poll_interval must be at least 1 second")
	}
	if c.Matter.PollInterval > 1*time.Hour {
		return fmt.Errorf("matter.poll_interval must not exceed 1 hour")
	}
	if c.Matter.DiscoveryInterval < c.Matter.PollInterval {
		return fmt.Errorf("matter.discovery_interval should be greater than or equal to matter.poll_interval")
	}
	// Only validate channel size if explicitly set (non-zero)
	// Zero value is allowed and will be set to default (100)
	if c.Matter.ReadingsChannelSize != 0 {
		if c.Matter.ReadingsChannelSize < 1 {
			return fmt.Errorf("matter.readings_channel_size must be at least 1")
		}
		if c.Matter.ReadingsChannelSize > 10000 {
			return fmt.Errorf("matter.readings_channel_size must not exceed 10000 (got %d)", c.Matter.ReadingsChannelSize)
		}
	}

	return nil
}

// validateLogging validates the logging configuration
func (c *Config) validateLogging() error {
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true,
		"warning": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error, fatal, panic")
	}

	return nil
}
