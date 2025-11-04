// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package configreload

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/soothill/hyundai-logger/internal/config"
)

// ReloadFunc is called when configuration is successfully reloaded
type ReloadFunc func(oldConfig, newConfig *config.Config) error

// Logger interface for logging reload events
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// Reloader watches configuration file and triggers reloads
type Reloader struct {
	configPath    string
	currentConfig *config.Config
	mu            sync.RWMutex
	watcher       *fsnotify.Watcher
	logger        Logger
	reloadFuncs   []ReloadFunc
	debounce      time.Duration
	lastReload    time.Time
}

// Config holds reloader configuration
type Config struct {
	ConfigPath string
	Logger     Logger
	Debounce   time.Duration // Minimum time between reloads (prevents rapid fire)
}

// New creates a new configuration reloader
func New(cfg Config) (*Reloader, error) {
	if cfg.Debounce == 0 {
		cfg.Debounce = 1 * time.Second // Default debounce
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating file watcher: %w", err)
	}

	// Load initial configuration
	initialConfig, err := config.LoadConfig(cfg.ConfigPath)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("loading initial config: %w", err)
	}

	r := &Reloader{
		configPath:    cfg.ConfigPath,
		currentConfig: initialConfig,
		watcher:       watcher,
		logger:        cfg.Logger,
		debounce:      cfg.Debounce,
		lastReload:    time.Now(),
	}

	// Watch the configuration file
	if err := watcher.Add(cfg.ConfigPath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("watching config file: %w", err)
	}

	return r, nil
}

// GetConfig returns the current configuration
func (r *Reloader) GetConfig() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentConfig
}

// OnReload registers a function to be called when configuration is reloaded
func (r *Reloader) OnReload(fn ReloadFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reloadFuncs = append(r.reloadFuncs, fn)
}

// Start begins watching the configuration file for changes
func (r *Reloader) Start(ctx context.Context) error {
	r.logger.Info("Configuration hot reload enabled for: %s", r.configPath)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Configuration reloader shutting down")
			return ctx.Err()

		case event, ok := <-r.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed unexpectedly")
			}

			// Only care about write/create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r.logger.Debug("Configuration file changed: %s", event.Name)
				r.handleReload()
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			r.logger.Error("Configuration watcher error: %v", err)
		}
	}
}

// handleReload processes a configuration reload
func (r *Reloader) handleReload() {
	// Debounce: prevent rapid reloads
	r.mu.Lock()
	if time.Since(r.lastReload) < r.debounce {
		r.logger.Debug("Reload debounced (too soon after last reload)")
		r.mu.Unlock()
		return
	}
	r.lastReload = time.Now()
	r.mu.Unlock()

	r.logger.Info("Reloading configuration from: %s", r.configPath)

	// Load new configuration
	newConfig, err := config.LoadConfig(r.configPath)
	if err != nil {
		r.logger.Error("Failed to load new configuration: %v", err)
		r.logger.Error("Keeping current configuration")
		return
	}

	// Get old config for comparison
	r.mu.RLock()
	oldConfig := r.currentConfig
	r.mu.RUnlock()

	// Call reload hooks
	for _, fn := range r.reloadFuncs {
		if err := fn(oldConfig, newConfig); err != nil {
			r.logger.Error("Reload hook failed: %v", err)
			r.logger.Error("Aborting configuration reload")
			return
		}
	}

	// Apply new configuration
	r.mu.Lock()
	r.currentConfig = newConfig
	r.mu.Unlock()

	r.logger.Info("Configuration reloaded successfully")
	r.logConfigChanges(oldConfig, newConfig)
}

// logConfigChanges logs notable configuration changes
func (r *Reloader) logConfigChanges(old, new *config.Config) {
	// Log poll interval changes
	if old.RateLimit.PollIntervalMinutes != new.RateLimit.PollIntervalMinutes {
		r.logger.Info("  Poll interval: %dm → %dm",
			old.RateLimit.PollIntervalMinutes,
			new.RateLimit.PollIntervalMinutes)
	}

	// Log requests per hour changes
	if old.RateLimit.RequestsPerHour != new.RateLimit.RequestsPerHour {
		r.logger.Info("  Requests per hour: %d → %d",
			old.RateLimit.RequestsPerHour,
			new.RateLimit.RequestsPerHour)
	}

	// DISABLED: 	// Log schedule changes
	// DISABLED: 	if old.RateLimit.Schedule.Enabled != new.RateLimit.Schedule.Enabled {
	// DISABLED: 		r.logger.Info("  Time-based schedule: %v → %v",
	// DISABLED: 			old.RateLimit.Schedule.Enabled,
	// DISABLED: 			new.RateLimit.Schedule.Enabled)
	// DISABLED: 	}
	// DISABLED:
	// DISABLED: 	// Log charging config changes
	// DISABLED: 	if old.RateLimit.ChargingConfig.Enabled != new.RateLimit.ChargingConfig.Enabled {
	// DISABLED: 		r.logger.Info("  Charging detection: %v → %v",
	// DISABLED: 			old.RateLimit.ChargingConfig.Enabled,
	// DISABLED: 			new.RateLimit.ChargingConfig.Enabled)
	// DISABLED: 	}
	// DISABLED:
	// DISABLED: 	if old.RateLimit.ChargingConfig.IntervalMinutes != new.RateLimit.ChargingConfig.IntervalMinutes {
	// DISABLED: 		r.logger.Info("  Charging interval: %dm → %dm",
	// DISABLED: 			old.RateLimit.ChargingConfig.IntervalMinutes,
	// DISABLED: 			new.RateLimit.ChargingConfig.IntervalMinutes)
	// DISABLED: 	}

	// Log alert changes
	if old.Alerts.Enabled != new.Alerts.Enabled {
		r.logger.Info("  Email alerts: %v → %v",
			old.Alerts.Enabled,
			new.Alerts.Enabled)
	}
}

// Close stops watching the configuration file
func (r *Reloader) Close() error {
	if r.watcher != nil {
		return r.watcher.Close()
	}
	return nil
}

// ForceReload immediately reloads the configuration
func (r *Reloader) ForceReload() error {
	r.logger.Info("Force reloading configuration")

	newConfig, err := config.LoadConfig(r.configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	r.mu.RLock()
	oldConfig := r.currentConfig
	r.mu.RUnlock()

	// Call reload hooks
	for _, fn := range r.reloadFuncs {
		if err := fn(oldConfig, newConfig); err != nil {
			return fmt.Errorf("reload hook failed: %w", err)
		}
	}

	r.mu.Lock()
	r.currentConfig = newConfig
	r.lastReload = time.Now()
	r.mu.Unlock()

	r.logger.Info("Configuration force reloaded successfully")
	r.logConfigChanges(oldConfig, newConfig)

	return nil
}

// Stats holds statistics about configuration reloads
type Stats struct {
	CurrentConfigPath string
	LastReloadTime    time.Time
	ReloadHooksCount  int
}

// GetStats returns reloader statistics
func (r *Reloader) GetStats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return Stats{
		CurrentConfigPath: r.configPath,
		LastReloadTime:    r.lastReload,
		ReloadHooksCount:  len(r.reloadFuncs),
	}
}
