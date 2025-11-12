// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
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
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name: "missing influxdb url",
			config: Config{
				InfluxDB: InfluxDBConfig{
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "influxdb token too short",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "short",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid poll interval (too short)",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        500 * time.Millisecond, // < 1s
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "discovery interval less than poll interval",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   10 * time.Second,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086",
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "non-local HTTP URL for InfluxDB",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://example.com:8086", // Non-local HTTP, should fail
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "valid HTTPS URL for InfluxDB",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "https://example.com:8086", // Valid HTTPS
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   1024 * 1024,
					MaxAge:    time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid cache max size",
			config: Config{
				InfluxDB: InfluxDBConfig{
					URL:          "http://localhost:8086", // Valid HTTP since local
					Token:        "a-very-secret-token",
					Organization: "test-org",
					Bucket:       "test-bucket",
				},
				Matter: MatterConfig{
					DiscoveryInterval:   5 * time.Minute,
					PollInterval:        30 * time.Second,
					ServiceType:         "_matter._tcp",
					Domain:              "local.",
					ReadingsChannelSize: 100,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Cache: CacheConfig{
					Directory: "/tmp/cache",
					MaxSize:   0, // Invalid: min=1
					MaxAge:    time.Hour,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				// Check if the error is a validation error for more specific assertions
				if vErrs, ok := err.(validator.ValidationErrors); ok {
					// Log specific validation errors for debugging
					for _, vErr := range vErrs {
						t.Logf("Validation error: Field=%s, Tag=%s, Value=%v", vErr.Field(), vErr.Tag(), vErr.Value())
					}
				}
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
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`
default:
  influxdb:
    url: "http://localhost:8086"
    token: "test-token"
    organization: "test-org"
    bucket: "test-bucket"
  matter:
    discovery_interval: 5m
    poll_interval: 30s
  logging:
    level: "info"
  cache:
    directory: "/tmp/cache"
    max_size: 104857600
    max_age: 24h
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
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`
default:
  influxdb:
    url: "http://localhost:8086"
    token: "file-token"
    organization: "file-org"
    bucket: "file-bucket"
  matter:
    discovery_interval: 5m
    poll_interval: 30s
  logging:
    level: "info"
  cache:
    directory: "/tmp/cache_file"
    max_size: 104857600
    max_age: 24h
`)
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	_ = tmpfile.Close()

	// Set environment variables to override
	_ = os.Setenv("INFLUXDB_URL", "https://env-host:8086")
	_ = os.Setenv("INFLUXDB_TOKEN", "env-token-123")
	_ = os.Setenv("INFLUXDB_ORG", "env-org")
	_ = os.Setenv("INFLUXDB_BUCKET", "env-bucket")
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("MATTER_DISCOVERY_INTERVAL", "10m")
	_ = os.Setenv("MATTER_POLL_INTERVAL", "1m")
	_ = os.Setenv("CACHE_DIRECTORY", "/tmp/cache_env")

	defer func() {
		_ = os.Unsetenv("INFLUXDB_URL")
		_ = os.Unsetenv("INFLUXDB_TOKEN")
		_ = os.Unsetenv("INFLUXDB_ORG")
		_ = os.Unsetenv("INFLUXDB_BUCKET")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("MATTER_DISCOVERY_INTERVAL")
		_ = os.Unsetenv("MATTER_POLL_INTERVAL")
		_ = os.Unsetenv("CACHE_DIRECTORY")
	}()

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment variables override file values
	if cfg.InfluxDB.URL != "https://env-host:8086" {
		t.Errorf("InfluxDB.URL = %v, want https://env-host:8086", cfg.InfluxDB.URL)
	}
	if cfg.InfluxDB.Token != "env-token-123" {
		t.Errorf("InfluxDB.Token = %v, want env-token-123", cfg.InfluxDB.Token)
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
	if cfg.Cache.Directory != "/tmp/cache_env" {
		t.Errorf("Cache.Directory = %v, want /tmp/cache_env", cfg.Cache.Directory)
	}
}

func TestLoad_Defaults(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := []byte(`
default:
  influxdb:
    url: "http://localhost:8086"
    token: "test-token-default"
    organization: "test-org-default"
    bucket: "test-bucket-default"
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
	if cfg.Cache.Directory != "/var/cache/matter-data-logger" {
		t.Errorf("Default cache directory = %v, want /var/cache/matter-data-logger", cfg.Cache.Directory)
	}
	if cfg.Cache.MaxSize != 100*1024*1024 {
		t.Errorf("Default cache max size = %v, want 100MB", cfg.Cache.MaxSize)
	}
	if cfg.Cache.MaxAge != 24*time.Hour {
		t.Errorf("Default cache max age = %v, want 24h", cfg.Cache.MaxAge)
	}
	if cfg.Matter.ReadingsChannelSize != 100 {
		t.Errorf("Default readings channel size = %v, want 100", cfg.Matter.ReadingsChannelSize)
	}
}
