// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

// Package logger provides structured logging using zerolog.
package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Initialize sets up the global logger with the specified level
func Initialize(level string) {
	// Parse log level
	logLevel, err := parseLogLevel(level)
	if err != nil {
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
	default:
		return zerolog.InfoLevel, nil
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
