// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package config

import (
	"context"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soothill/matter-data-logger/pkg/logger"
)

const (
	// debounceDuration is the time to wait for file system events to settle
	debounceDuration = 500 * time.Millisecond
)

// ReloadedConfig represents a successfully reloaded configuration
type ReloadedConfig struct {
	Config *Config
	Error  error
}

// Watcher monitors a configuration file for changes and reloads it
type Watcher struct {
	configPath string
	watcher    *fsnotify.Watcher
	// Reloaded channel sends new configurations or errors
	Reloaded chan ReloadedConfig
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWatcher creates a new Watcher
func NewWatcher(configPath string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cw := &Watcher{
		configPath: configPath,
		watcher:    watcher,
		Reloaded:   make(chan ReloadedConfig),
		ctx:        ctx,
		cancel:     cancel,
	}

	if err := cw.watcher.Add(configPath); err != nil {
		cw.watcher.Close()
		return nil, fmt.Errorf("failed to add config file to watcher: %w", err)
	}

	go cw.run()

	return cw, nil
}

// Close stops the watcher
func (cw *Watcher) Close() {
	cw.cancel()
	cw.watcher.Close()
	close(cw.Reloaded)
}

// run starts the event loop for the watcher
func (cw *Watcher) run() {
	var lastEventTime time.Time
	for {
		select {
		case <-cw.ctx.Done():
			logger.Info().Msg("Config watcher shutting down")
			return
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			// Only react to Write or Create events on the config file itself
			// and debounce events to avoid multiple reloads for a single save operation
			if event.Name == cw.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				if time.Since(lastEventTime) < debounceDuration {
					continue
				}
				lastEventTime = time.Now()

				logger.Info().Str("event", event.String()).Msg("Config file changed, reloading...")
				newCfg, err := Load(cw.configPath)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to reload configuration")
					cw.Reloaded <- ReloadedConfig{Error: fmt.Errorf("failed to reload config: %w", err)}
					continue
				}
				logger.Info().Msg("Configuration reloaded successfully")
				cw.Reloaded <- ReloadedConfig{Config: newCfg}
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			logger.Error().Err(err).Msg("Config watcher error")
			cw.Reloaded <- ReloadedConfig{Error: fmt.Errorf("config watcher error: %w", err)}
		}
	}
}
