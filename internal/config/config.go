package config

import (
	"fmt"
	"log/slog"
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
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
	Timeout string            `mapstructure:"timeout"`
	Retry   RetryConfig       `mapstructure:"retry"`
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
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.cors.allowed_origins", []string{"*"})
	viper.SetDefault("server.tls.enabled", false)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("database.path", "./sentinel.db")
	viper.SetDefault("database.retention", "30d")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
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
