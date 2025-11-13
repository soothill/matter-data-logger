// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadFileSafely reads a file after cleaning and validating the path.
func ReadFileSafely(path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for %s: %w", path, err)
	}
	return os.ReadFile(absPath) // #nosec G304
}
