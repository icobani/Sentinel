package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/sentinel/sentinel/internal/api"
	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/service"

	frontend "github.com/sentinel/sentinel"
)

var (
	version   = "dev"
	buildTime = ""
)

func main() {
	// Setup logging
	setupLogging()

	// Parse command line arguments
	if len(os.Args) > 1 {
		command := os.Args[1]

		switch command {
		case "install":
			handleInstall()
			return
		case "uninstall":
			handleUninstall()
			return
		case "start":
			handleStart()
			return
		case "stop":
			handleStop()
			return
		case "restart":
			handleRestart()
			return
		case "status":
			handleStatus()
			return
		case "version", "-v", "--version":
			fmt.Printf("Sentinel File Watcher v%s\n", version)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		case "run":
			// Continue to run in interactive mode
			break
		default:
			// If it's not a known command, treat it as a config file path
			// Allow non-existent paths as the config loader will create default config
			runApplication(command)
			return
		}
	}

	// Default: run in interactive mode with default config
	runApplication("./sentinel.yaml")
}

// runApplication runs the main application logic
func runApplication(configPath string) {
	slog.Info("Starting Sentinel file watcher service", "version", version)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set version and build time in config
	config.Version = version
	config.BuildTime = buildTime

	// Reconfigure logging with the level from config
	reconfigureLogging(cfg)

	// Initialize application (DB, watchers, etc.)
	if err := service.Initialize(cfg); err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Create sub FS from embedded frontend (strip dist/browser prefix)
	subFS, err := fs.Sub(frontend.FrontendFS, "dist/browser")
	if err != nil {
		slog.Error("Failed to create sub filesystem for frontend", "error", err)
		os.Exit(1)
	}

	// Setup router with embedded frontend
	router := api.SetupRouter(cfg, subFS)

	// Run in interactive mode (handles graceful shutdown)
	if err := service.RunInteractive(cfg, router); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

// handleInstall installs the service
func handleInstall() {
	configPath := "./sentinel.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	fmt.Println("Installing Sentinel service...")
	if err := service.Install(configPath); err != nil {
		fmt.Printf("Failed to install service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sentinel service installed successfully!")
	fmt.Println("\nTo start the service, run:")
	fmt.Println("  sentinel start")
}

// handleUninstall uninstalls the service
func handleUninstall() {
	fmt.Println("Uninstalling Sentinel service...")
	if err := service.Uninstall(); err != nil {
		fmt.Printf("Failed to uninstall service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sentinel service uninstalled successfully!")
}

// handleStart starts the service
func handleStart() {
	fmt.Println("Starting Sentinel service...")
	if err := service.Start(); err != nil {
		fmt.Printf("Failed to start service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sentinel service started successfully!")
}

// handleStop stops the service
func handleStop() {
	fmt.Println("Stopping Sentinel service...")
	if err := service.Stop(); err != nil {
		fmt.Printf("Failed to stop service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sentinel service stopped successfully!")
}

// handleRestart restarts the service
func handleRestart() {
	fmt.Println("Restarting Sentinel service...")
	if err := service.Restart(); err != nil {
		fmt.Printf("Failed to restart service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Sentinel service restarted successfully!")
}

// handleStatus shows the service status
func handleStatus() {
	if err := service.Status(); err != nil {
		fmt.Printf("Failed to get service status: %v\n", err)
		os.Exit(1)
	}
}

// printUsage prints the usage information
func printUsage() {
	fmt.Printf("Sentinel File Watcher v%s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  sentinel [command] [config-file]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install   - Install as a system service")
	fmt.Println("  uninstall - Uninstall the system service")
	fmt.Println("  start     - Start the service")
	fmt.Println("  stop      - Stop the service")
	fmt.Println("  restart   - Restart the service")
	fmt.Println("  status    - Show service status")
	fmt.Println("  run       - Run in interactive mode (foreground)")
	fmt.Println("  version   - Show version information")
	fmt.Println("  help      - Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sentinel                           # Run with default config (./sentinel.yaml)")
	fmt.Println("  sentinel run                       # Run in interactive mode")
	fmt.Println("  sentinel /path/to/config.yaml      # Run with custom config")
	fmt.Println("  sentinel install                   # Install as service with default config")
	fmt.Println("  sentinel install /path/to/config.yaml  # Install with custom config")
	fmt.Println("  sentinel start                     # Start the service")
	fmt.Println("  sentinel stop                      # Stop the service")
	fmt.Println()
	fmt.Println("Service Management:")
	fmt.Println("  Windows: Installs as Windows Service")
	fmt.Println("  Linux:   Installs as systemd service")
	fmt.Println("  macOS:   Installs as launchd service")
}

// setupLogging configures the logging system with default settings
// This is called before config is loaded to ensure we have logging during startup
func setupLogging() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// reconfigureLogging reconfigures the logging system with the level from config
func reconfigureLogging(cfg *config.Config) {
	var level slog.Level

	// Parse the level string from config
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
		slog.Warn("Unknown log level, defaulting to info", "configured_level", cfg.Logging.Level)
	}

	// Create new handler with the configured level
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logging reconfigured", "level", cfg.Logging.Level)
}
