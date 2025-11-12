// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDiscoveryError(t *testing.T) {
	baseErr := fmt.Errorf("network unreachable")
	err := &DiscoveryError{Op: "mDNS scan", Err: baseErr}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "discovery") || !strings.Contains(errMsg, "mDNS scan") {
		t.Errorf("Error() = %q, want message containing 'discovery' and 'mDNS scan'", errMsg)
	}

	// Test Unwrap()
	if !errors.Is(err, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}

	// Test errors.As()
	var de *DiscoveryError
	if !errors.As(err, &de) {
		t.Error("errors.As() should extract DiscoveryError")
	}
	if de.Op != "mDNS scan" {
		t.Errorf("DiscoveryError.Op = %q, want %q", de.Op, "mDNS scan")
	}
}

func TestStorageError(t *testing.T) {
	baseErr := fmt.Errorf("connection timeout")
	err := &StorageError{Op: "write", Err: baseErr}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "storage") || !strings.Contains(errMsg, "write") {
		t.Errorf("Error() = %q, want message containing 'storage' and 'write'", errMsg)
	}

	// Test Unwrap()
	if !errors.Is(err, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}

	// Test errors.As()
	var se *StorageError
	if !errors.As(err, &se) {
		t.Error("errors.As() should extract StorageError")
	}
	if se.Op != "write" {
		t.Errorf("StorageError.Op = %q, want %q", se.Op, "write")
	}
}
