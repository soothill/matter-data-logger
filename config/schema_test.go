// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func TestValidateWithSchema_ValidConfig(t *testing.T) {
	// Create a temporary valid config
	validConfig := `influxdb:
  url: http://localhost:8086
  token: test-token-12345
  organization: my-org
  bucket: power-data

matter:
  discovery_interval: 5m
  poll_interval: 30s
  service_type: _matter._tcp
  domain: local.
  readings_channel_size: 1000

logging:
  level: info

notifications:
  slack_webhook_url: https://hooks.slack.com/services/TEST/WEBHOOK/URL

cache:
  directory: ./cache
  max_size: 104857600
  max_age: 24h
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(validConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Validate should pass
	err = ValidateWithSchema(tmpFile)
	if err != nil {
		t.Errorf("ValidateWithSchema() with valid config failed: %v", err)
	}
}

func TestValidateWithSchema_MissingRequired(t *testing.T) {
	// Config missing required fields
	invalidConfig := `influxdb:
  url: http://localhost:8086

logging:
  level: info
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Validate should fail
	err = ValidateWithSchema(tmpFile)
	if err == nil {
		t.Error("ValidateWithSchema() should fail with missing required fields")
	}
}

func TestValidateWithSchema_InvalidType(t *testing.T) {
	// Config with wrong type
	invalidConfig := `influxdb:
  url: http://localhost:8086
  token: test-token
  organization: my-org
  bucket: power-data

matter:
  discovery_interval: not-a-duration
  poll_interval: 30s

logging:
  level: info
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Validate should fail
	err = ValidateWithSchema(tmpFile)
	if err == nil {
		t.Error("ValidateWithSchema() should fail with invalid duration format")
	}
}

func TestValidateWithSchema_InvalidLogLevel(t *testing.T) {
	// Config with invalid enum value
	invalidConfig := `influxdb:
  url: http://localhost:8086
  token: test-token-12345
  organization: my-org
  bucket: power-data

matter:
  discovery_interval: 5m
  poll_interval: 30s

logging:
  level: invalid-level
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Validate should fail
	err = ValidateWithSchema(tmpFile)
	if err == nil {
		t.Error("ValidateWithSchema() should fail with invalid log level")
	}
}

func TestValidateWithSchema_MinimumValues(t *testing.T) {
	// Config with values below minimum
	invalidConfig := `influxdb:
  url: http://localhost:8086
  token: short
  organization: my-org
  bucket: power-data

matter:
  discovery_interval: 5m
  poll_interval: 30s
  readings_channel_size: 5

logging:
  level: info
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidConfig), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Validate should fail
	err = ValidateWithSchema(tmpFile)
	if err == nil {
		t.Error("ValidateWithSchema() should fail with values below minimum")
	}
}

func TestValidateWithSchema_FileNotFound(t *testing.T) {
	err := ValidateWithSchema("nonexistent-file.yaml")
	if err == nil {
		t.Error("ValidateWithSchema() should fail with nonexistent file")
	}
}

func TestValidateWithSchema_InvalidYAML(t *testing.T) {
	invalidYAML := `influxdb:
  url: http://localhost:8086
  token: [invalid yaml structure
`

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(tmpFile, []byte(invalidYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	err = ValidateWithSchema(tmpFile)
	if err == nil {
		t.Error("ValidateWithSchema() should fail with invalid YAML")
	}
}

func TestGetSchemaJSON(t *testing.T) {
	schema := GetSchemaJSON()
	if schema == "" {
		t.Error("GetSchemaJSON() returned empty string")
	}

	// Should be valid JSON
	if len(schema) < 100 {
		t.Error("GetSchemaJSON() returned suspiciously short schema")
	}

	// Should contain expected fields
	if !contains(schema, "$schema") {
		t.Error("GetSchemaJSON() should contain $schema field")
	}
	if !contains(schema, "influxdb") {
		t.Error("GetSchemaJSON() should contain influxdb definition")
	}
}

func TestFormatValidationErrors(t *testing.T) {
	// Test with empty errors
	err := formatValidationErrors(nil)
	if err != nil {
		t.Errorf("formatValidationErrors(nil) should return nil, got %v", err)
	}

	err = formatValidationErrors([]gojsonschema.ResultError{})
	if err != nil {
		t.Errorf("formatValidationErrors([]) should return nil, got %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInside(s, substr)))
}

func containsInside(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
