// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/soothill/matter-data-logger/app"
)

// setupDebugSignalHandlers sets up debug signal handlers (SIGUSR1, SIGUSR2)
// SIGUSR1: Dump current application state (devices, monitoring stats)
// SIGUSR2: Dump goroutine stack traces
//
// Usage:
//
//	kill -USR1 <pid>  # Dump application state
//	kill -USR2 <pid>  # Dump goroutine stack traces
func setupDebugSignalHandlers(application *app.App) {
	debugSigChan := make(chan os.Signal, 2) // Buffer for 2 signals
	signal.Notify(debugSigChan, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for sig := range debugSigChan {
			switch sig {
			case syscall.SIGUSR1:
				application.DumpApplicationState()
			case syscall.SIGUSR2:
				app.DumpGoroutineStackTraces()
			}
		}
	}()
}
