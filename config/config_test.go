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

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR_MISSING",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvAsIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid int env var",
			key:          "TEST_INT_VAR",
			defaultValue: 100,
			envValue:     "200",
			want:         200,
		},
		{
			name:         "invalid int env var",
			key:          "TEST_INT_VAR_INVALID",
			defaultValue: 100,
			envValue:     "not-a-number",
			want:         100,
		},
		{
			name:         "env var not set",
			key:          "TEST_INT_VAR_MISSING",
			defaultValue: 100,
			envValue:     "",
			want:         100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvAsIntOrDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvAsIntOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
