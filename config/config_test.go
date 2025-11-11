// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"os"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
					ServiceType:       "_matter._tcp",
					Domain:            "local.",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "missing influxdb url",
			config: Config{
				InfluxDB: InfluxDBConfig{
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
		},
		{
			name: "missing influxdb token",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid poll interval",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      500 * time.Millisecond,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent-config.yaml")
	if err == nil {
		t.Error("Load() should fail when file doesn't exist")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a temporary invalid YAML file
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte("invalid: yaml: content:\n  - missing\n  closing")
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	_ = tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Load() should fail with invalid YAML")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	// Create a temporary valid config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`influxdb:
  url: "http://localhost:8086"
  token: "test-token"
  organization: "test-org"
  bucket: "test-bucket"
matter:
  discovery_interval: 5m
  poll_interval: 30s
logging:
  level: "info"
`)
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	_ = tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.InfluxDB.URL != "http://localhost:8086" {
		t.Errorf("InfluxDB.URL = %v, want http://localhost:8086", cfg.InfluxDB.URL)
	}
	if cfg.InfluxDB.Token != "test-token" {
		t.Errorf("InfluxDB.Token = %v, want test-token", cfg.InfluxDB.Token)
	}
	if cfg.Matter.DiscoveryInterval != 5*time.Minute {
		t.Errorf("Matter.DiscoveryInterval = %v, want 5m", cfg.Matter.DiscoveryInterval)
	}
	if cfg.Matter.PollInterval != 30*time.Second {
		t.Errorf("Matter.PollInterval = %v, want 30s", cfg.Matter.PollInterval)
	}
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`influxdb:
  url: "http://localhost:8086"
  token: "file-token"
  organization: "file-org"
  bucket: "file-bucket"
matter:
  discovery_interval: 5m
  poll_interval: 30s
logging:
  level: "info"
`)
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	_ = tmpfile.Close()

	// Set environment variables to override
	_ = os.Setenv("INFLUXDB_URL", "https://env-host:8086")
	_ = os.Setenv("INFLUXDB_TOKEN", "env-token")
	_ = os.Setenv("INFLUXDB_ORG", "env-org")
	_ = os.Setenv("INFLUXDB_BUCKET", "env-bucket")
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("MATTER_DISCOVERY_INTERVAL", "10m")
	_ = os.Setenv("MATTER_POLL_INTERVAL", "1m")

	defer func() {
		_ = os.Unsetenv("INFLUXDB_URL")
		_ = os.Unsetenv("INFLUXDB_TOKEN")
		_ = os.Unsetenv("INFLUXDB_ORG")
		_ = os.Unsetenv("INFLUXDB_BUCKET")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("MATTER_DISCOVERY_INTERVAL")
		_ = os.Unsetenv("MATTER_POLL_INTERVAL")
	}()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment variables override file values
	if cfg.InfluxDB.URL != "https://env-host:8086" {
		t.Errorf("InfluxDB.URL = %v, want https://env-host:8086", cfg.InfluxDB.URL)
	}
	if cfg.InfluxDB.Token != "env-token" {
		t.Errorf("InfluxDB.Token = %v, want env-token", cfg.InfluxDB.Token)
	}
	if cfg.InfluxDB.Organization != "env-org" {
		t.Errorf("InfluxDB.Organization = %v, want env-org", cfg.InfluxDB.Organization)
	}
	if cfg.InfluxDB.Bucket != "env-bucket" {
		t.Errorf("InfluxDB.Bucket = %v, want env-bucket", cfg.InfluxDB.Bucket)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %v, want debug", cfg.Logging.Level)
	}
	if cfg.Matter.DiscoveryInterval != 10*time.Minute {
		t.Errorf("Matter.DiscoveryInterval = %v, want 10m", cfg.Matter.DiscoveryInterval)
	}
	if cfg.Matter.PollInterval != 1*time.Minute {
		t.Errorf("Matter.PollInterval = %v, want 1m", cfg.Matter.PollInterval)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Create a minimal config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`influxdb:
  url: "http://localhost:8086"
  token: "test-token"
  organization: "test-org"
  bucket: "test-bucket"
`)
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	_ = tmpfile.Close()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults are applied
	if cfg.Matter.DiscoveryInterval != 5*time.Minute {
		t.Errorf("Default DiscoveryInterval = %v, want 5m", cfg.Matter.DiscoveryInterval)
	}
	if cfg.Matter.PollInterval != 30*time.Second {
		t.Errorf("Default PollInterval = %v, want 30s", cfg.Matter.PollInterval)
	}
	if cfg.Matter.ServiceType != "_matter._tcp" {
		t.Errorf("Default ServiceType = %v, want _matter._tcp", cfg.Matter.ServiceType)
	}
	if cfg.Matter.Domain != "local." {
		t.Errorf("Default Domain = %v, want local.", cfg.Matter.Domain)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Default log level = %v, want info", cfg.Logging.Level)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "missing organization",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:    "http://localhost:8086",
					Token:  "test-token",
					Bucket: "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{Level: "info"},
			},
		},
		{
			name: "missing bucket",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{Level: "info"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Error("Validate() should fail for missing required fields")
			}
		})
	}
}

func TestValidate_InvalidIntervals(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "discovery_interval less than poll_interval",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 30 * time.Second,
					PollInterval:      1 * time.Minute,
				},
				Logging: LoggingConfig{Level: "info"},
			},
		},
		{
			name: "zero discovery_interval",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 0,
					PollInterval:      30 * time.Second,
				},
				Logging: LoggingConfig{Level: "info"},
			},
		},
		{
			name: "zero poll_interval",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "test-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval: 5 * time.Minute,
					PollInterval:      0,
				},
				Logging: LoggingConfig{Level: "info"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Error("Validate() should fail for invalid intervals")
			}
		})
	}
}
