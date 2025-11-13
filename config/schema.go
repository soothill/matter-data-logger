// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/soothill/matter-data-logger/pkg/util"
	"github.com/xeipuuv/gojsonschema"
)

// ValidateWithSchema validates the configuration file against the JSON schema.
func ValidateWithSchema(path string) error {
	schemaPath, err := filepath.Abs("schema.json")
	if err != nil {
		return fmt.Errorf("could not get absolute path for schema: %w", err)
	}
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

	yamlFile, err := util.ReadFileSafely(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData interface{}
	err = yaml.Unmarshal(yamlFile, &configData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	jsonData, err := json.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("failed to validate config schema: %w", err)
	}

	if !result.Valid() {
		fmt.Fprintf(os.Stderr, "Configuration is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			fmt.Fprintf(os.Stderr, "- %s\n", desc)
		}
		return fmt.Errorf("configuration is not valid")
	}

	return nil
}
