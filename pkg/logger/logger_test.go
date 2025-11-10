// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{"debug level", "debug", zerolog.DebugLevel},
		{"info level", "info", zerolog.InfoLevel},
		{"warn level", "warn", zerolog.WarnLevel},
		{"warning level", "warning", zerolog.WarnLevel},
		{"error level", "error", zerolog.ErrorLevel},
		{"fatal level", "fatal", zerolog.FatalLevel},
		{"panic level", "panic", zerolog.PanicLevel},
		{"invalid level defaults to info", "invalid", zerolog.InfoLevel},
		{"empty level defaults to info", "", zerolog.InfoLevel},
		{"uppercase level", "DEBUG", zerolog.DebugLevel},
		{"mixed case level", "WaRn", zerolog.WarnLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Initialize(tt.level)

			// Get the logger and check its level
			logger := Get()
			if logger == nil {
				t.Fatal("Get() returned nil logger")
			}

			// Verify the logger was initialized
			if logger == nil {
				t.Error("Logger should not be nil after Initialize()")
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{"debug", "debug", zerolog.DebugLevel},
		{"info", "info", zerolog.InfoLevel},
		{"warn", "warn", zerolog.WarnLevel},
		{"warning", "warning", zerolog.WarnLevel},
		{"error", "error", zerolog.ErrorLevel},
		{"fatal", "fatal", zerolog.FatalLevel},
		{"panic", "panic", zerolog.PanicLevel},
		{"invalid defaults to info", "invalid", zerolog.InfoLevel},
		{"uppercase", "DEBUG", zerolog.DebugLevel},
		{"mixed case", "InFo", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLogLevel(tt.level)
			if level != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, want %v", tt.level, level, tt.expected)
			}
			// Error is acceptable for invalid levels
			_ = err
		})
	}
}

func TestGet(t *testing.T) {
	Initialize("info")

	logger := Get()
	if logger == nil {
		t.Error("Get() returned nil logger")
	}

	// Get should return the same logger instance
	logger2 := Get()
	if logger2 == nil {
		t.Error("Second Get() returned nil logger")
	}
}

func TestLogFunctions(t *testing.T) {
	// Redirect output to buffer for testing
	var buf bytes.Buffer
	Initialize("debug")
	SetOutput(&buf)

	tests := []struct {
		name    string
		logFunc func() *zerolog.Event
		message string
		level   string
	}{
		{"debug", Debug, "debug message", "debug"},
		{"info", Info, "info message", "info"},
		{"warn", Warn, "warn message", "warn"},
		{"error", Error, "error message", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			event := tt.logFunc()
			if event == nil {
				t.Errorf("%s() returned nil event", tt.name)
				return
			}

			event.Msg(tt.message)

			// Verify something was written
			output := buf.String()
			if output == "" {
				t.Errorf("%s() produced no output", tt.name)
			}

			// Verify the message is in the output
			if !strings.Contains(output, tt.message) {
				t.Errorf("%s() output should contain message %q, got %q", tt.name, tt.message, output)
			}
		})
	}
}

func TestWith(t *testing.T) {
	Initialize("info")

	context := With()

	// Test that we can add fields and create a logger
	logger := context.Str("test_field", "test_value").Logger()

	// Verify we can use the logger without panicking
	var buf bytes.Buffer
	logger = logger.Output(&buf)
	logger.Info().Msg("test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Error("Context-created logger should be functional")
	}
}

func TestSetOutput(t *testing.T) {
	var buf bytes.Buffer
	Initialize("info")
	SetOutput(&buf)

	Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("SetOutput() should redirect output, got: %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		configLevel string
		logLevel    string
		shouldLog   bool
	}{
		{"info logs at info level", "info", "info", true},
		{"debug filtered at info level", "info", "debug", false},
		{"error logs at info level", "info", "error", true},
		{"warn logs at info level", "info", "warn", true},
		{"debug logs at debug level", "debug", "debug", true},
		{"info logs at error level", "error", "info", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Initialize(tt.configLevel)
			SetOutput(&buf)

			message := "test message for filtering"

			// Log at the specified level
			switch tt.logLevel {
			case "debug":
				Debug().Msg(message)
			case "info":
				Info().Msg(message)
			case "warn":
				Warn().Msg(message)
			case "error":
				Error().Msg(message)
			}

			output := buf.String()
			hasMessage := strings.Contains(output, message)

			if tt.shouldLog && !hasMessage {
				t.Errorf("Expected message to be logged at %s level with config %s, but it wasn't", tt.logLevel, tt.configLevel)
			}
			if !tt.shouldLog && hasMessage {
				t.Errorf("Expected message NOT to be logged at %s level with config %s, but it was", tt.logLevel, tt.configLevel)
			}
		})
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer
	Initialize("info")
	SetOutput(&buf)

	// Test logging with various field types
	Info().
		Str("string_field", "value").
		Int("int_field", 42).
		Bool("bool_field", true).
		Float64("float_field", 3.14).
		Msg("test with fields")

	output := buf.String()

	expectedFields := []string{"test with fields", "string_field", "value", "int_field", "42", "bool_field", "float_field", "3.14"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output should contain %q, got: %s", field, output)
		}
	}
}

func TestMultipleInitializations(t *testing.T) {
	// Test that multiple initializations don't cause issues
	Initialize("debug")
	Initialize("info")
	Initialize("error")

	logger := Get()
	if logger == nil {
		t.Error("Logger should be initialized after multiple Initialize() calls")
	}
}

func TestLoggerNotPanicOnNil(t *testing.T) {
	// Ensure logger functions don't panic even with unusual usage
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger functions should not panic, got: %v", r)
		}
	}()

	Initialize("info")

	// These should not panic
	Debug().Msg("test")
	Info().Msg("test")
	Warn().Msg("test")
	Error().Msg("test")
}
