// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package config provides configuration management for the Matter data logger.
package config

import (
	"fmt"
	"os"
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides (for secrets and runtime config)
	if url := os.Getenv("INFLUXDB_URL"); url != "" {
		cfg.InfluxDB.URL = url
	}
	if token := os.Getenv("INFLUXDB_TOKEN"); token != "" {
		cfg.InfluxDB.Token = token
	}
	if org := os.Getenv("INFLUXDB_ORG"); org != "" {
		cfg.InfluxDB.Organization = org
	}
	if bucket := os.Getenv("INFLUXDB_BUCKET"); bucket != "" {
		cfg.InfluxDB.Bucket = bucket
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}
	if interval := os.Getenv("MATTER_DISCOVERY_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			cfg.Matter.DiscoveryInterval = duration
		}
	}
	if interval := os.Getenv("MATTER_POLL_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			cfg.Matter.PollInterval = duration
		}
	}

	// Set defaults
	if cfg.Matter.DiscoveryInterval == 0 {
		cfg.Matter.DiscoveryInterval = 5 * time.Minute
	}
	if cfg.Matter.PollInterval == 0 {
		cfg.Matter.PollInterval = 30 * time.Second
	}
	if cfg.Matter.ServiceType == "" {
		cfg.Matter.ServiceType = "_matter._tcp"
	}
	if cfg.Matter.Domain == "" {
		cfg.Matter.Domain = "local."
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate InfluxDB configuration
	if c.InfluxDB.URL == "" {
		return fmt.Errorf("influxdb.url is required")
	}
	if c.InfluxDB.Token == "" {
		return fmt.Errorf("influxdb.token is required")
	}
	if c.InfluxDB.Organization == "" {
		return fmt.Errorf("influxdb.organization is required")
	}
	if c.InfluxDB.Bucket == "" {
		return fmt.Errorf("influxdb.bucket is required")
	}

	// Validate Matter configuration
	if c.Matter.DiscoveryInterval < time.Second {
		return fmt.Errorf("matter.discovery_interval must be at least 1 second")
	}
	if c.Matter.PollInterval < time.Second {
		return fmt.Errorf("matter.poll_interval must be at least 1 second")
	}
	if c.Matter.DiscoveryInterval < c.Matter.PollInterval {
		return fmt.Errorf("matter.discovery_interval should be greater than or equal to matter.poll_interval")
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true,
		"warning": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error, fatal, panic")
	}

	return nil
}
