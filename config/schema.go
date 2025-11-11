// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON []byte

// ValidateWithSchema validates a configuration file against the JSON schema.
// This provides comprehensive validation of structure, types, and constraints.
//
// Example usage:
//
//	err := config.ValidateWithSchema("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
func ValidateWithSchema(configPath string) error {
	// Load the embedded schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)

	// Load the config file
	configData, err := readConfigForSchema(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Convert YAML to JSON for validation
	var configObj interface{}
	err = yaml.Unmarshal(configData, &configObj)
	if err != nil {
		return fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Convert to JSON for schema validation
	configJSON, err := json.Marshal(configObj)
	if err != nil {
		return fmt.Errorf("failed to convert config to JSON: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(configJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	if !result.Valid() {
		return formatValidationErrors(result.Errors())
	}

	return nil
}

// readConfigForSchema reads config file for schema validation
func readConfigForSchema(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// formatValidationErrors formats JSON schema validation errors into a readable message
func formatValidationErrors(errors []gojsonschema.ResultError) error {
	if len(errors) == 0 {
		return nil
	}

	msg := "configuration validation errors:\n"
	for i, err := range errors {
		msg += fmt.Sprintf("  %d. %s: %s\n", i+1, err.Field(), err.Description())
	}

	return fmt.Errorf("%s", msg)
}

// GetSchemaJSON returns the embedded JSON schema as a string.
// Useful for documentation or external tools.
func GetSchemaJSON() string {
	return string(schemaJSON)
}
