// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package slacknotifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		webhookURL  string
		wantEnabled bool
	}{
		{
			name:        "with webhook URL",
			webhookURL:  "https://hooks.slack.com/services/test",
			wantEnabled: true,
		},
		{
			name:        "empty webhook URL",
			webhookURL:  "",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := New(tt.webhookURL)
			if notifier.IsEnabled() != tt.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", notifier.IsEnabled(), tt.wantEnabled)
			}
		})
	}
}

func TestNotifier_SendMessage(t *testing.T) {
	// Create a test server
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := New(server.URL)
	ctx := context.Background()

	err := notifier.SendMessage(ctx, "Test message")
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	if !called {
		t.Error("Expected webhook to be called")
	}
}

func TestNotifier_SendMessage_Disabled(t *testing.T) {
	notifier := New("")
	ctx := context.Background()

	// Should not error when disabled
	err := notifier.SendMessage(ctx, "Test message")
	if err != nil {
		t.Errorf("SendMessage() with disabled notifier error = %v", err)
	}
}

func TestNotifier_SendAlert(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		title    string
		message  string
	}{
		{
			name:     "danger alert",
			severity: "danger",
			title:    "Test Danger",
			message:  "This is a danger alert",
		},
		{
			name:     "warning alert",
			severity: "warning",
			title:    "Test Warning",
			message:  "This is a warning alert",
		},
		{
			name:     "success alert",
			severity: "good",
			title:    "Test Success",
			message:  "This is a success alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			notifier := New(server.URL)
			ctx := context.Background()

			err := notifier.SendAlert(ctx, tt.severity, tt.title, tt.message)
			if err != nil {
				t.Errorf("SendAlert() error = %v", err)
			}
		})
	}
}

func TestNotifier_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := New(server.URL)
	ctx := context.Background()

	err := notifier.SendMessage(ctx, "Test message")
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestNotifier_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(15 * time.Second) // Longer than client timeout
	}))
	defer server.Close()

	notifier := New(server.URL)
	ctx := context.Background()

	err := notifier.SendMessage(ctx, "Test message")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestSeverityToColor(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"danger", "danger"},
		{"error", "danger"},
		{"warning", "warning"},
		{"warn", "warning"},
		{"good", "good"},
		{"success", "good"},
		{"info", "#808080"},
		{"", "#808080"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := severityToColor(tt.severity)
			if got != tt.want {
				t.Errorf("severityToColor(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}
