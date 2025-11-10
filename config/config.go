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

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
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

	return &cfg, nil
}
