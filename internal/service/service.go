package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/watcher"
	"github.com/sentinel/sentinel/internal/ws"
)

// Program implements the service.Interface
type Program struct {
	cfg         *config.Config
	server      *http.Server
	exit        chan struct{}
	errorLogger chan struct{} // Signal to stop error logger goroutine
}

// newServiceConfig returns the base service configuration (name only).
// Used by Uninstall, Start, Stop, Restart, Status, and Run — operations
// that only need to identify the service, not configure how it launches.
func newServiceConfig() *service.Config {
	return &service.Config{
		Name:        "Sentinel",
		DisplayName: "Sentinel File Watcher",
		Description: "Cross-platform file system watcher and webhook notification service",
	}
}

// Start is called when the service is starting
func (p *Program) Start(s service.Service) error {
	slog.Info("Starting Sentinel service")

	// Start the service in a goroutine
	go p.run()

	return nil
}

// run contains the main service logic
func (p *Program) run() {
	slog.Info("Sentinel service running")

	// Wait for exit signal
	<-p.exit
}

// Stop is called when the service is stopping
func (p *Program) Stop(s service.Service) error {
	slog.Info("Stopping Sentinel service")

	// Stop all watchers
	manager := watcher.GetManager()
	slog.Info("Stopping all watchers...")
	if err := manager.StopAll(); err != nil {
		slog.Error("Error stopping watchers", "error", err)
	}

	// Stop webhook dispatcher
	slog.Info("Stopping webhook dispatcher...")
	if err := manager.GetDispatcher().Stop(); err != nil {
		slog.Error("Error stopping webhook dispatcher", "error", err)
	}

	// Stop WebSocket hub
	slog.Info("Stopping WebSocket hub...")
	hub := ws.GetHub()
	hub.Stop()

	// Shutdown HTTP server
	if p.server != nil {
		slog.Info("Shutting down HTTP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := p.server.Shutdown(ctx); err != nil {
			slog.Error("Server forced to shutdown", "error", err)
		}
	}

	// Close database
	slog.Info("Closing database...")
	storage.Close()

	// Stop error logger goroutine
	if p.errorLogger != nil {
		close(p.errorLogger)
	}

	// Signal exit
	close(p.exit)

	slog.Info("Sentinel service stopped")
	return nil
}

// Install installs the service with the given config path embedded in the
// service arguments and the executable directory as the working directory.
func Install(configPath string) error {
	// Resolve the config path to an absolute path so Windows SCM can find it
	// regardless of the working directory at launch time.
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	// Determine the executable path so we can set a sensible WorkingDirectory.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build a fresh config that includes launch-time Arguments and WorkingDirectory.
	cfg := newServiceConfig()
	cfg.Arguments = []string{absConfigPath}
	cfg.WorkingDirectory = filepath.Dir(exePath)

	prg := &Program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	slog.Info("Service installed successfully",
		"config", absConfigPath,
		"working_dir", cfg.WorkingDirectory,
	)
	return nil
}

// Uninstall removes the service
func Uninstall() error {
	prg := &Program{}
	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	slog.Info("Service uninstalled successfully")
	return nil
}

// Start starts the service
func Start() error {
	prg := &Program{}
	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	slog.Info("Service started successfully")
	return nil
}

// Stop stops the service
func Stop() error {
	prg := &Program{}
	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	slog.Info("Service stopped successfully")
	return nil
}

// Restart restarts the service
func Restart() error {
	prg := &Program{}
	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := s.Restart(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	slog.Info("Service restarted successfully")
	return nil
}

// IsInteractive reports whether the process is running interactively (i.e. not
// as a managed Windows/Linux/macOS service). It delegates to the
// kardianos/service package-level Interactive() function which queries the OS.
func IsInteractive() bool {
	return service.Interactive()
}

// Run runs the service under Windows SCM / systemd / launchd. It must be
// called (instead of RunInteractive) when the binary is launched by the
// service manager, because s.Run() registers the process with the SCM and
// handles Start/Stop callbacks.
func Run(cfg *config.Config, router http.Handler) error {
	prg := &Program{
		cfg:  cfg,
		exit: make(chan struct{}),
	}

	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Setup service logger (writes to Windows Event Log / syslog / launchd)
	errs := make(chan error, 5)
	logger, err := s.Logger(errs)
	if err != nil {
		return fmt.Errorf("failed to create service logger: %w", err)
	}

	// Build a multi-handler: Event Viewer + optional log file
	var handlers []slog.Handler
	handlers = append(handlers, newServiceLogHandler(logger, parseLogLevel(cfg.Logging.Level)))

	if cfg.Logging.File != "" {
		logFile, openErr := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if openErr != nil {
			// Non-fatal: log the issue and continue with Event Log only.
			_ = logger.Warning(fmt.Sprintf("failed to open log file %s: %v", cfg.Logging.File, openErr))
		} else {
			fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
				Level: parseLogLevel(cfg.Logging.Level),
			})
			handlers = append(handlers, fileHandler)
		}
	}

	slog.SetDefault(slog.New(&multiHandler{handlers: handlers}))

	// Initialize error logger stop channel
	prg.errorLogger = make(chan struct{})

	// Log service-level errors in background
	go func() {
		for {
			select {
			case err := <-errs:
				if err != nil {
					slog.Error("Service error", "error", err)
				}
			case <-prg.errorLogger:
				slog.Debug("Error logger goroutine stopping")
				return
			}
		}
	}()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	prg.server = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		var err error
		if cfg.Server.TLS.Enabled {
			slog.Info("Starting HTTPS server", "address", addr)
			slog.Info("Sentinel UI available", "url", fmt.Sprintf("https://localhost:%d", cfg.Server.Port))
			err = prg.server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)

			if err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to start HTTPS server, falling back to HTTP", "error", err)
				cfg.Server.TLS.Enabled = false
				slog.Info("Starting HTTP server", "address", addr)
				slog.Info("Sentinel UI available", "url", fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
				err = prg.server.ListenAndServe()
			}
		} else {
			slog.Info("Starting HTTP server", "address", addr)
			slog.Info("Sentinel UI available", "url", fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
			err = prg.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			_ = logger.Error(err)
		}
	}()

	slog.Info("Server started successfully", "address", addr)

	// Block until the SCM sends a Stop signal
	if err := s.Run(); err != nil {
		return fmt.Errorf("service run error: %w", err)
	}

	return nil
}

// Status returns the service status
func Status() error {
	prg := &Program{}
	s, err := service.New(prg, newServiceConfig())
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	status, err := s.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	switch status {
	case service.StatusRunning:
		fmt.Println("Service Status: Running")
	case service.StatusStopped:
		fmt.Println("Service Status: Stopped")
	case service.StatusUnknown:
		fmt.Println("Service Status: Unknown")
	default:
		fmt.Println("Service Status: Unknown")
	}

	return nil
}

// RunInteractive runs the application in interactive mode (not as a service)
func RunInteractive(cfg *config.Config, router http.Handler) error {
	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		var err error
		if cfg.Server.TLS.Enabled {
			slog.Info("Starting HTTPS server", "address", addr)
			slog.Info("Sentinel UI available", "url", fmt.Sprintf("https://localhost:%d", cfg.Server.Port))
			err = srv.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)

			// If TLS fails, try falling back to HTTP
			if err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to start HTTPS server, falling back to HTTP", "error", err)
				cfg.Server.TLS.Enabled = false
				slog.Info("Starting HTTP server", "address", addr)
				slog.Info("Sentinel UI available", "url", fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
				err = srv.ListenAndServe()
			}
		} else {
			slog.Info("Starting HTTP server", "address", addr)
			slog.Info("Sentinel UI available", "url", fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Server started successfully", "address", addr)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop all watchers
	manager := watcher.GetManager()
	slog.Info("Stopping all watchers...")
	if err := manager.StopAll(); err != nil {
		slog.Error("Error stopping watchers", "error", err)
	}

	// Stop webhook dispatcher
	slog.Info("Stopping webhook dispatcher...")
	if err := manager.GetDispatcher().Stop(); err != nil {
		slog.Error("Error stopping webhook dispatcher", "error", err)
	}

	// Stop WebSocket hub
	slog.Info("Stopping WebSocket hub...")
	hub := ws.GetHub()
	hub.Stop()

	// Close database
	slog.Info("Closing database...")
	storage.Close()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		return err
	}

	slog.Info("Server shutdown complete")
	return nil
}

// Initialize performs application initialization (DB, watchers, etc.)
func Initialize(cfg *config.Config) error {
	// Initialize database
	if err := storage.InitDB(cfg.Database.Path); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load watchers from config file to database
	if err := watcher.LoadWatchersFromConfig(convertToWatcherConfigs(cfg.Watchers)); err != nil {
		return fmt.Errorf("failed to load watchers from config: %w", err)
	}

	// Initialize WebSocket hub
	hub := ws.GetHub()
	slog.Info("WebSocket hub initialized", "clients", hub.GetClientCount())

	// Get watcher manager
	manager := watcher.GetManager()

	// Start webhook dispatcher
	if err := manager.GetDispatcher().Start(); err != nil {
		return fmt.Errorf("failed to start webhook dispatcher: %w", err)
	}

	// Start all watchers from database
	if err := manager.StartAll(); err != nil {
		slog.Warn("Some watchers failed to start", "error", err)
		// Don't return error, continue with the ones that started successfully
	}

	// Configure database retention cleanup (runs on each event insert, throttled)
	storage.InitCleanup(cfg.Database.Retention)

	return nil
}

// convertToWatcherConfigs converts config.WatcherConfig to watcher.WatcherConfig
func convertToWatcherConfigs(cfgs []config.WatcherConfig) []watcher.WatcherConfig {
	result := make([]watcher.WatcherConfig, len(cfgs))
	for i, cfg := range cfgs {
		result[i] = watcher.WatcherConfig{
			Name:         cfg.Name,
			Path:         cfg.Path,
			Mode:         cfg.Mode,
			Recursive:    cfg.Recursive,
			PollInterval: cfg.PollInterval,
		}
		result[i].Filters.Include = cfg.Filters.Include
		result[i].Filters.Exclude = cfg.Filters.Exclude
		result[i].Webhook.URL = cfg.Webhook.URL
		result[i].Webhook.Headers = cfg.Webhook.Headers
		result[i].Webhook.Timeout = cfg.Webhook.Timeout
		result[i].Webhook.AttachFile = cfg.Webhook.AttachFile
		result[i].Webhook.SkipTLSVerify = cfg.Webhook.SkipTLSVerify
		result[i].Webhook.Retry.MaxAttempts = cfg.Webhook.Retry.MaxAttempts
		result[i].Webhook.Retry.Backoff = cfg.Webhook.Retry.Backoff
	}
	return result
}

// parseLogLevel converts a logging level string to the corresponding slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ---------------------------------------------------------------------------
// serviceLogHandler — bridges slog to kardianos/service Logger (Event Viewer)
// ---------------------------------------------------------------------------

type serviceLogHandler struct {
	logger service.Logger
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func newServiceLogHandler(logger service.Logger, level slog.Level) *serviceLogHandler {
	return &serviceLogHandler{logger: logger, level: level}
}

func (h *serviceLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *serviceLogHandler) Handle(_ context.Context, r slog.Record) error {
	msg := r.Message
	// Append structured attributes to the message string.
	r.Attrs(func(a slog.Attr) bool {
		msg += " " + a.Key + "=" + a.Value.String()
		return true
	})

	switch {
	case r.Level >= slog.LevelError:
		return h.logger.Error(msg)
	case r.Level >= slog.LevelWarn:
		return h.logger.Warning(msg)
	default:
		return h.logger.Info(msg)
	}
}

func (h *serviceLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &serviceLogHandler{
		logger: h.logger,
		level:  h.level,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *serviceLogHandler) WithGroup(name string) slog.Handler {
	return &serviceLogHandler{
		logger: h.logger,
		level:  h.level,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

// ---------------------------------------------------------------------------
// multiHandler — fans slog records out to multiple slog.Handler instances
// ---------------------------------------------------------------------------

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}
