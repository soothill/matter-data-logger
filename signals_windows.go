// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

//go:build windows

package main

import (
	"github.com/soothill/matter-data-logger/discovery"
	"github.com/soothill/matter-data-logger/monitoring"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

// setupDebugSignalHandlers is a no-op on Windows as SIGUSR1/SIGUSR2 don't exist
// On Windows, debug information can be accessed via:
// - HTTP endpoints (if implemented)
// - Log file analysis
// - Windows Performance Monitor
func setupDebugSignalHandlers(scanner *discovery.Scanner, monitor *monitoring.PowerMonitor) {
	// No-op on Windows - SIGUSR1 and SIGUSR2 don't exist
	// Debug signal handlers are only available on Unix-like systems
	logger.Debug().Msg("Debug signal handlers not available on Windows")
}
