package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	pathpkg "path"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sentinel/sentinel/internal/api/controllers"
	"github.com/sentinel/sentinel/internal/api/routes"
	"github.com/sentinel/sentinel/internal/api/services"
	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/storage"
	"github.com/sentinel/sentinel/internal/watcher"
	"github.com/sentinel/sentinel/internal/ws"
)

// SetupRouter configures and returns the Gin router
func SetupRouter(cfg *config.Config, frontendFS fs.FS) *gin.Engine {
	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add custom logging middleware
	router.Use(LoggingMiddleware())

	// Configure CORS
	corsConfig := cors.Config{
		AllowOrigins:  cfg.Server.CORS.AllowedOrigins,
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders: []string{"Content-Length", "Content-Disposition"},
		MaxAge:        12 * time.Hour,
	}

	// Only enable credentials if not using wildcard origins
	hasWildcard := false
	for _, origin := range cfg.Server.CORS.AllowedOrigins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}
	corsConfig.AllowCredentials = !hasWildcard

	router.Use(cors.New(corsConfig))

	// Dependency injection
	db := storage.DB
	manager := watcher.GetManager()
	hub := ws.GetHub()

	// Create services
	watcherSvc := services.NewWatcherService(db, manager, hub)
	logSvc := services.NewLogService(db)
	statusSvc := services.NewStatusService(db, manager)

	// Create controllers
	watcherCtrl := controllers.NewWatcherController(watcherSvc)
	logCtrl := controllers.NewLogController(logSvc)
	statusCtrl := controllers.NewStatusController(statusSvc)
	wsCtrl := controllers.NewWSController(hub, cfg)

	// Register routes
	v1 := router.Group("/api/v1")
	routes.InitWatcherRoutes(v1, watcherCtrl)
	routes.InitLogRoutes(v1, logCtrl)
	routes.InitStatusRoutes(v1, statusCtrl)
	routes.InitWSRoutes(router, wsCtrl)

	// Serve static files from embedded Angular build
	if frontendFS != nil {
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// Don't serve static files for API or WebSocket paths
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"error":   "endpoint not found",
				})
				return
			}

			// Clean the path and remove leading slash
			// Use path package (not filepath) to ensure forward slashes on all OS
			cleanPath := strings.TrimPrefix(pathpkg.Clean(path), "/")
			if cleanPath == "." || cleanPath == "" {
				cleanPath = "index.html"
			}

			// Try to read the file from embedded FS
			data, err := fs.ReadFile(frontendFS, cleanPath)
			if err != nil {
				// If file not found, serve index.html for Angular client-side routing
				data, err = fs.ReadFile(frontendFS, "index.html")
				if err != nil {
					slog.Error("Failed to read index.html from embedded FS", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"error":   "failed to serve application",
					})
					return
				}
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}

			// Determine content type based on file extension
			contentType := getContentType(cleanPath)
			c.Data(http.StatusOK, contentType, data)
		})
	}

	slog.Info("Router configured successfully")
	return router
}

// getContentType returns the appropriate content type based on file extension
func getContentType(p string) string {
	ext := pathpkg.Ext(p)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// LoggingMiddleware provides custom logging for requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Log request
		slog.Info("HTTP request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", duration.String(),
			"ip", c.ClientIP(),
		)
	}
}
