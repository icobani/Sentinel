package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sentinel/sentinel/internal/api"
	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/service"

	frontend "github.com/sentinel/sentinel"
)

var (
	version   = "dev"
	buildTime = ""
	portFlag  int
)

func main() {
	// Setup logging with sensible defaults before config is loaded.
	setupLogging()

	// Parse flags
	flag.IntVar(&portFlag, "port", 0, "Override server port (e.g., -port 8085)")
	flag.Usage = func() { printUsage() }
	flag.Parse()

	args := flag.Args()

	// Parse command line arguments
	if len(args) > 0 {
		command := args[0]

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
		case "version":
			fmt.Printf("Sentinel File Watcher v%s\n", version)
			return
		case "help":
			printUsage()
			return
		case "run":
			// Continue to run in interactive mode
			break
		default:
			// If it's not a known command, treat it as a config file path.
			// This is also the path used when Windows SCM starts the binary
			// with the config path as a command-line argument (set by Install).
			runApplication(command)
			return
		}
	}

	// Default: run with default config path.
	runApplication("./sentinel.yaml")
}

// resolveConfigPath converts a possibly-relative config path to an absolute
// one. When the binary is launched by Windows SCM the working directory is
// C:\Windows\System32, so relative paths would not work. We resolve relative
// paths relative to the executable's directory instead.
func resolveConfigPath(configPath string) string {
	if filepath.IsAbs(configPath) {
		return configPath
	}
	exePath, err := os.Executable()
	if err != nil {
		return configPath
	}
	return filepath.Join(filepath.Dir(exePath), configPath)
}

// runApplication runs the main application logic
func runApplication(configPath string) {
	// Always work with an absolute config path so that the process works
	// correctly regardless of the current working directory (important when
	// started by Windows SCM from C:\Windows\System32).
	configPath = resolveConfigPath(configPath)

	slog.Info("Starting Sentinel file watcher service", "version", version, "config", configPath)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Override port if specified via flag
	if portFlag > 0 {
		cfg.Server.Port = portFlag
		slog.Info("Port overridden via command line flag", "port", portFlag)
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
	subFS, err := fs.Sub(frontend.FrontendFS, "frontend/dist/browser")
	if err != nil {
		slog.Error("Failed to create sub filesystem for frontend", "error", err)
		os.Exit(1)
	}

	// Setup router with embedded frontend
	router := api.SetupRouter(cfg, subFS)

	// Detect whether we are running under a service manager (SCM / systemd /
	// launchd) or interactively. When launched by Windows SCM the process is
	// NOT interactive and must call service.Run() to register with SCM,
	// otherwise the SCM will report Error 1053 (service did not respond in
	// time).
	if service.IsInteractive() {
		// Interactive mode: handle SIGINT/SIGTERM ourselves.
		if err := service.RunInteractive(cfg, router); err != nil {
			slog.Error("Application error", "error", err)
			os.Exit(1)
		}
	} else {
		// Service mode: register with the OS service manager.
		if err := service.Run(cfg, router); err != nil {
			slog.Error("Service run error", "error", err)
			os.Exit(1)
		}
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
	fmt.Println("Flags:")
	fmt.Println("  -port <port>  Override server port (e.g., -port 8085)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sentinel                           # Run with default config (./sentinel.yaml)")
	fmt.Println("  sentinel -port 8085                # Run on port 8085")
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

// setupLogging configures the logging system with default settings.
// Called before config is loaded to ensure we have logging during startup.
func setupLogging() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// reconfigureLogging reconfigures the logging system with the level from
// config. In interactive mode logs go to stdout (and optionally a file).
// In service mode, logging is configured inside service.Run() which routes
// output to the platform's service logger (Windows Event Log, syslog, etc.).
func reconfigureLogging(cfg *config.Config) {
	var level slog.Level

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

	opts := &slog.HandlerOptions{Level: level}

	var writer io.Writer = os.Stdout

	// If a log file is configured, write to both stdout and the file.
	if cfg.Logging.File != "" {
		logFile, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Warn("Failed to open log file, logging to stdout only",
				"path", cfg.Logging.File, "error", err)
		} else {
			writer = io.MultiWriter(os.Stdout, logFile)
		}
	}

	handler := slog.NewTextHandler(writer, opts)
	slog.SetDefault(slog.New(handler))

	slog.Info("Logging reconfigured", "level", cfg.Logging.Level)
}
