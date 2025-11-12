// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package errors

import "fmt"

// DiscoveryError represents an error that occurred during device discovery.
type DiscoveryError struct {
	Op  string
	Err error
}

// Error returns the string representation of the error.
func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("discovery operation '%s' failed: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *DiscoveryError) Unwrap() error {
	return e.Err
}

// StorageError represents an error that occurred in the storage layer.
type StorageError struct {
	Op  string
	Err error
}

// Error returns the string representation of the error.
func (e *StorageError) Error() string {
	return fmt.Sprintf("storage operation '%s' failed: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *StorageError) Unwrap() error {
	return e.Err
}
