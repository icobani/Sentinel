package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/viper"
)

var (
	Version   = "dev"
	BuildTime = ""
)

// Config represents the application configuration
type Config struct {
	Watchers []WatcherConfig `mapstructure:"watchers"`
	Server   ServerConfig    `mapstructure:"server"`
	Logging  LoggingConfig   `mapstructure:"logging"`
	Database DatabaseConfig  `mapstructure:"database"`
}

// WatcherConfig represents a single watcher configuration
type WatcherConfig struct {
	Name         string        `mapstructure:"name"`
	Path         string        `mapstructure:"path"`
	Mode         string        `mapstructure:"mode"` // "watch" or "poll"
	Recursive    bool          `mapstructure:"recursive"`
	PollInterval string        `mapstructure:"poll_interval"`
	Filters      FilterConfig  `mapstructure:"filters"`
	Webhook      WebhookConfig `mapstructure:"webhook"`
}

// FilterConfig represents file filtering rules
type FilterConfig struct {
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL           string            `mapstructure:"url"`
	Headers       map[string]string `mapstructure:"headers"`
	Timeout       string            `mapstructure:"timeout"`
	Retry         RetryConfig       `mapstructure:"retry"`
	AttachFile    bool              `mapstructure:"attach_file"`
	SkipTLSVerify bool              `mapstructure:"skip_tls_verify"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts int    `mapstructure:"max_attempts"`
	Backoff     string `mapstructure:"backoff"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int        `mapstructure:"port"`
	Host string     `mapstructure:"host"`
	CORS CORSConfig `mapstructure:"cors"`
	TLS  TLSConfig  `mapstructure:"tls"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path      string `mapstructure:"path"`
	Retention string `mapstructure:"retention"` // e.g., "30d", "7d", "24h"
}

var AppConfig *Config

// Load loads the configuration from the specified file
func Load(configPath string) (*Config, error) {
	SetConfigFilePath(configPath)
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("server.port", 8083)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.cors.allowed_origins", []string{"*"})
	viper.SetDefault("server.tls.enabled", false)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("database.path", "./sentinel.db")
	viper.SetDefault("database.retention", "30d")

	if err := viper.ReadInConfig(); err != nil {
		// Check if the error is because the file doesn't exist
		if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			// Create default config file
			slog.Info("Config file not found, creating default configuration", "path", configPath)
			if err := createDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
			// Try reading again
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read newly created config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	AppConfig = &config
	slog.Info("Configuration loaded successfully", "path", configPath)
	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but cert_file or key_file not specified")
		}
	}

	for i, watcher := range cfg.Watchers {
		if watcher.Name == "" {
			return fmt.Errorf("watcher %d: name is required", i)
		}
		if watcher.Path == "" {
			return fmt.Errorf("watcher %s: path is required", watcher.Name)
		}
		if watcher.Webhook.URL == "" {
			return fmt.Errorf("watcher %s: webhook URL is required", watcher.Name)
		}

		// Validate mode
		if watcher.Mode != "" && watcher.Mode != "watch" && watcher.Mode != "poll" {
			return fmt.Errorf("watcher %s: invalid mode '%s' (must be 'watch' or 'poll')", watcher.Name, watcher.Mode)
		}

		// Validate timeout
		if watcher.Webhook.Timeout != "" {
			if _, err := time.ParseDuration(watcher.Webhook.Timeout); err != nil {
				return fmt.Errorf("watcher %s: invalid timeout duration: %w", watcher.Name, err)
			}
		}

		// Validate backoff
		if watcher.Webhook.Retry.Backoff != "" {
			if _, err := time.ParseDuration(watcher.Webhook.Retry.Backoff); err != nil {
				return fmt.Errorf("watcher %s: invalid backoff duration: %w", watcher.Name, err)
			}
		}

		// Validate poll_interval for poll mode
		if watcher.Mode == "poll" && watcher.PollInterval != "" {
			if _, err := time.ParseDuration(watcher.PollInterval); err != nil {
				return fmt.Errorf("watcher %s: invalid poll_interval duration: %w", watcher.Name, err)
			}
		}
	}

	return nil
}

// Reload reloads the configuration
func Reload(configPath string) error {
	newConfig, err := Load(configPath)
	if err != nil {
		return err
	}
	AppConfig = newConfig
	slog.Info("Configuration reloaded successfully")
	return nil
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	defaultConfig := `# Sentinel Configuration File
# This file was automatically generated with default values.
# You can add watchers through the web UI or by editing this file directly.

# Server configuration
server:
  port: 8083                    # Web UI and API port
  host: "0.0.0.0"               # Listen address (0.0.0.0 = all interfaces)
  cors:
    allowed_origins: ["*"]      # CORS allowed origins (* = all origins)
  tls:
    enabled: false              # Enable HTTPS (requires cert_file and key_file)
    cert_file: ""               # Path to TLS certificate file
    key_file: ""                # Path to TLS private key file

# Logging configuration
logging:
  level: "info"                 # Log level: debug, info, warn, error
  output: "stdout"              # Output: stdout or file
  file: ""                      # Log file path (only used if output = file)

# Database configuration
database:
  path: "./sentinel.db"         # SQLite database file path
  retention: "30d"              # Event log retention period (e.g., 30d, 7d, 24h)

# File watchers configuration
# Add watchers through the web UI at http://localhost:8083
# or edit this section manually following the example format:
#
# watchers:
#   - name: "example-watcher"
#     path: "/path/to/watch"
#     mode: "watch"             # "watch" for local dirs, "poll" for network shares
#     recursive: true
#     filters:
#       include: ["*.json", "*.xml"]
#       exclude: ["*.tmp"]
#     webhook:
#       url: "https://api.example.com/webhook"
#       headers:
#         Authorization: "Bearer your-token"
#       timeout: "10s"
#       retry:
#         max_attempts: 3
#         backoff: "2s"

watchers: []
`

	// Write the default config to file
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}

	slog.Info("Default configuration file created successfully", "path", configPath)
	return nil
}
