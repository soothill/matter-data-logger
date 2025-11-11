// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package logger provides structured logging using zerolog.
//
// This package wraps zerolog to provide a consistent logging interface across
// the application with structured JSON logging, configurable log levels, and
// console-friendly formatting for development.
//
// # Logging Levels
//
// Supported log levels (from least to most verbose):
//   - panic: Logs and immediately panics
//   - fatal: Logs and exits with os.Exit(1)
//   - error: Error conditions requiring attention
//   - warn/warning: Warning messages for potential issues
//   - info: Informational messages (default level)
//   - debug: Detailed debugging information
//
// # Configuration
//
// The logger is configured via the Initialize() function, typically called
// during application startup with the log level from configuration:
//
//	logger.Initialize("info")  // Set log level to info
//
// The log level can be set via:
//  1. config.yaml: logging.level
//  2. Environment variable: LOG_LEVEL
//  3. Default: info
//
// # Safe Initialization
//
// The logger uses an init() function to set up a safe default configuration
// that prevents panics if logging functions are called before Initialize().
// This default configuration logs at info level to stdout.
//
// # Structured Logging
//
// The logger supports structured logging with typed fields:
//
//	logger.Info().
//	    Str("device_id", "device-1").
//	    Float64("power", 100.5).
//	    Msg("Power reading recorded")
//
// Common field methods:
//   - Str(), Strs(): String values
//   - Int(), Int64(): Integer values
//   - Float64(): Floating-point values
//   - Bool(): Boolean values
//   - Dur(): Duration values
//   - Time(): Time values
//   - Err(): Error values (special formatting)
//
// # Output Format
//
// The logger outputs human-readable console format with:
//   - RFC3339 timestamps
//   - Color-coded log levels (when terminal supports it)
//   - Caller information (file:line) for debugging
//   - Structured key-value pairs
//
// Example output:
//   2025-11-11T10:30:45-08:00 INF Power reading recorded device_id=device-1 power=100.5
//
// # Global Logger Access
//
// The package provides convenience functions that wrap the global logger:
//   - Debug(), Info(), Warn(), Error(), Fatal()
//   - Get() for advanced usage
//   - With() for creating child loggers with preset fields
//
// # Thread Safety
//
// All logger operations are thread-safe and can be called concurrently from
// multiple goroutines. Zerolog uses lock-free operations for high performance.
//
// # Example Usage
//
// Basic logging:
//
//	logger.Info().Msg("Application started")
//	logger.Warn().Str("config_file", path).Msg("Using default config")
//	logger.Error().Err(err).Msg("Failed to connect to database")
//
// Structured logging with multiple fields:
//
//	logger.Info().
//	    Str("device_id", deviceID).
//	    Str("device_name", deviceName).
//	    Float64("power_w", reading.Power).
//	    Float64("voltage_v", reading.Voltage).
//	    Msg("Power reading")
//
// Child loggers with preset fields:
//
//	deviceLogger := logger.With().Str("device_id", deviceID).Logger()
//	deviceLogger.Info().Msg("Starting monitoring")  // Includes device_id automatically
package logger

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	log                 zerolog.Logger
	errInvalidLogLevel = errors.New("invalid log level")
)

// init initializes the logger with a default configuration to prevent panics
// before Initialize() is called. The logger will be reconfigured when Initialize() is called.
func init() {
	// Set up a default logger that writes to stdout at info level
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
}

// Initialize sets up the global logger with the specified level
func Initialize(level string) {
	// Parse log level
	logLevel, err := parseLogLevel(level)
	if err != nil {
		// Create a temporary logger to warn about invalid level
		tempOutput := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		tempLog := zerolog.New(tempOutput).With().Timestamp().Logger()
		tempLog.Warn().Str("invalid_level", level).Str("using", "info").Msg("Invalid log level, defaulting to info")
		logLevel = zerolog.InfoLevel
	}

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	log = zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()
}

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "panic":
		return zerolog.PanicLevel, nil
	case "":
		// Empty string is acceptable, default to info without warning
		return zerolog.InfoLevel, nil
	default:
		return zerolog.InfoLevel, errInvalidLogLevel
	}
}

// Get returns the global logger instance
func Get() *zerolog.Logger {
	return &log
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info logs an info message
func Info() *zerolog.Event {
	return log.Info()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// With creates a child logger with additional fields
func With() zerolog.Context {
	return log.With()
}

// SetOutput sets the output writer for the logger
func SetOutput(w io.Writer) {
	log = log.Output(w)
}
