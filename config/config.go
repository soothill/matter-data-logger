// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package config provides configuration management for the Matter data logger.
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
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
	Matter   MatterConfig   `yaml:"matter"`
	Logging  LoggingConfig  `yaml:"logging"`
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
	DiscoveryInterval time.Duration `yaml:"discovery_interval"`
	PollInterval      time.Duration `yaml:"poll_interval"`
	ServiceType       string        `yaml:"service_type"`
	Domain            string        `yaml:"domain"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string `yaml:"level"`
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
