// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/soothill/matter-data-logger/pkg/logger"
)

// Watcher handles hot reloading of the configuration file.
type Watcher struct {
	path       string
	configChan chan<- *Config
	reloadChan chan os.Signal
	cancelFunc context.CancelFunc
}

// NewWatcher creates a new configuration watcher.
func NewWatcher(path string, configChan chan<- *Config) *Watcher {
	return &Watcher{
		path:       path,
		configChan: configChan,
		reloadChan: make(chan os.Signal, 1),
	}
}

// Start begins watching for SIGHUP signals to trigger a configuration reload.
func (w *Watcher) Start(ctx context.Context) {
	ctx, w.cancelFunc = context.WithCancel(ctx)
	signal.Notify(w.reloadChan, syscall.SIGHUP)

	go w.watch(ctx)
}

// Stop stops the configuration watcher.
func (w *Watcher) Stop() {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	signal.Stop(w.reloadChan)
}

// watch listens for reload signals and reloads the configuration.
func (w *Watcher) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.reloadChan:
			logger.Info().Msg("SIGHUP received, reloading configuration")
			cfg, err := Load(w.path)
			if err != nil {
				logger.Error().Err(err).Msg("failed to reload configuration")
				continue
			}
			w.configChan <- cfg
			logger.Info().Msg("configuration reloaded successfully")
		}
	}
}
