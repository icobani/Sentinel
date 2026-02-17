package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	configMutex    sync.RWMutex
	pendingRestart bool
	pendingMutex   sync.RWMutex
	configFilePath string
)

// SetConfigFilePath sets the path to the config file
func SetConfigFilePath(path string) {
	configMutex.Lock()
	defer configMutex.Unlock()
	configFilePath = path
}

// GetConfigFilePath returns the current config file path
func GetConfigFilePath() string {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return getConfigFilePathLocked()
}

// getConfigFilePathLocked returns the config file path without acquiring the lock
// Must be called with configMutex already held
func getConfigFilePathLocked() string {
	if configFilePath == "" {
		return "./sentinel.yaml"
	}
	return configFilePath
}

// SetPendingRestart marks that a restart is pending
func SetPendingRestart(pending bool) {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()
	pendingRestart = pending
	if pending {
		slog.Info("Configuration change requires restart")
	}
}

// IsPendingRestart returns whether a restart is pending
func IsPendingRestart() bool {
	pendingMutex.RLock()
	defer pendingMutex.RUnlock()
	return pendingRestart
}

// AddWatcherToConfig adds a new watcher to the config file
func AddWatcherToConfig(watcher WatcherConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	path := getConfigFilePathLocked()

	// Read current config
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get watchers array
	watchers, ok := config["watchers"].([]interface{})
	if !ok {
		watchers = []interface{}{}
	}

	// Convert WatcherConfig to map
	watcherMap := watcherConfigToMap(watcher)

	// Add new watcher
	watchers = append(watchers, watcherMap)
	config["watchers"] = watchers

	// Write back to file
	if err := writeConfigFile(path, config); err != nil {
		return err
	}

	SetPendingRestart(true)
	slog.Info("Watcher added to config file", "name", watcher.Name)
	return nil
}

// UpdateWatcherInConfig updates an existing watcher in the config file
func UpdateWatcherInConfig(name string, watcher WatcherConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	path := getConfigFilePathLocked()

	// Read current config
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get watchers array
	watchers, ok := config["watchers"].([]interface{})
	if !ok {
		return fmt.Errorf("no watchers found in config")
	}

	// Find and update watcher
	found := false
	for i, w := range watchers {
		watcherMap, ok := w.(map[string]interface{})
		if !ok {
			continue
		}
		if watcherMap["name"] == name {
			watchers[i] = watcherConfigToMap(watcher)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("watcher '%s' not found in config", name)
	}

	config["watchers"] = watchers

	// Write back to file
	if err := writeConfigFile(path, config); err != nil {
		return err
	}

	SetPendingRestart(true)
	slog.Info("Watcher updated in config file", "name", watcher.Name)
	return nil
}

// RemoveWatcherFromConfig removes a watcher from the config file
func RemoveWatcherFromConfig(name string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	path := getConfigFilePathLocked()

	// Read current config
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get watchers array
	watchers, ok := config["watchers"].([]interface{})
	if !ok {
		return fmt.Errorf("no watchers found in config")
	}

	// Find and remove watcher
	newWatchers := []interface{}{}
	found := false
	for _, w := range watchers {
		watcherMap, ok := w.(map[string]interface{})
		if !ok {
			newWatchers = append(newWatchers, w)
			continue
		}
		if watcherMap["name"] != name {
			newWatchers = append(newWatchers, w)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("watcher '%s' not found in config", name)
	}

	config["watchers"] = newWatchers

	// Write back to file
	if err := writeConfigFile(path, config); err != nil {
		return err
	}

	SetPendingRestart(true)
	slog.Info("Watcher removed from config file", "name", name)
	return nil
}

// writeConfigFile writes config data to file with proper formatting
func writeConfigFile(path string, config map[string]interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// watcherConfigToMap converts WatcherConfig to a map for YAML serialization
func watcherConfigToMap(wc WatcherConfig) map[string]interface{} {
	result := map[string]interface{}{
		"name":      wc.Name,
		"path":      wc.Path,
		"mode":      wc.Mode,
		"recursive": wc.Recursive,
	}

	if wc.PollInterval != "" {
		result["poll_interval"] = wc.PollInterval
	}

	// Add filters if present
	if len(wc.Filters.Include) > 0 || len(wc.Filters.Exclude) > 0 {
		filters := map[string]interface{}{}
		if len(wc.Filters.Include) > 0 {
			filters["include"] = wc.Filters.Include
		}
		if len(wc.Filters.Exclude) > 0 {
			filters["exclude"] = wc.Filters.Exclude
		}
		result["filters"] = filters
	}

	// Add webhook config
	webhook := map[string]interface{}{
		"url": wc.Webhook.URL,
	}

	if len(wc.Webhook.Headers) > 0 {
		webhook["headers"] = wc.Webhook.Headers
	}

	if wc.Webhook.Timeout != "" {
		webhook["timeout"] = wc.Webhook.Timeout
	}

	if wc.Webhook.AttachFile {
		webhook["attach_file"] = wc.Webhook.AttachFile
	}

	// Add retry config
	if wc.Webhook.Retry.MaxAttempts > 0 || wc.Webhook.Retry.Backoff != "" {
		retry := map[string]interface{}{}
		if wc.Webhook.Retry.MaxAttempts > 0 {
			retry["max_attempts"] = wc.Webhook.Retry.MaxAttempts
		}
		if wc.Webhook.Retry.Backoff != "" {
			retry["backoff"] = wc.Webhook.Retry.Backoff
		}
		webhook["retry"] = retry
	}

	result["webhook"] = webhook

	return result
}

// ReloadConfig reloads the configuration from disk
func ReloadConfig() error {
	path := GetConfigFilePath()
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	var newConfig Config
	if err := viper.Unmarshal(&newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	AppConfig = &newConfig
	SetPendingRestart(false)
	slog.Info("Configuration reloaded", "path", path)
	return nil
}
