package controllers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sentinel/sentinel/internal/config"
	"github.com/sentinel/sentinel/internal/ws"
)

// WSController handles WebSocket connection upgrades
type WSController struct {
	hub      *ws.Hub
	cfg      *config.Config
	upgrader websocket.Upgrader
}

// NewWSController creates a new WebSocket controller instance
func NewWSController(hub *ws.Hub, cfg *config.Config) *WSController {
	ctrl := &WSController{
		hub: hub,
		cfg: cfg,
	}
	ctrl.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     ctrl.checkOrigin,
	}
	return ctrl
}

// checkOrigin validates the WebSocket connection origin against allowed origins
func (ctrl *WSController) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	if ctrl.cfg == nil {
		slog.Warn("AppConfig is nil, allowing all WebSocket origins")
		return true
	}

	allowedOrigins := ctrl.cfg.Server.CORS.AllowedOrigins
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.TrimSuffix(origin, "/") == strings.TrimSuffix(allowed, "/") {
			return true
		}
	}

	slog.Warn("WebSocket origin not allowed", "origin", origin, "allowed", allowedOrigins)
	return false
}

// HandleWS upgrades HTTP connection to WebSocket and registers the client
func (ctrl *WSController) HandleWS(c *gin.Context) {
	conn, err := ctrl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Failed to upgrade WebSocket connection", "error", err, "ip", c.ClientIP())
		return
	}

	client := ws.NewClient(ctrl.hub, conn)
	ctrl.hub.Register(client)
	client.Start()

	slog.Info("WebSocket client connected", "ip", c.ClientIP(), "total_clients", ctrl.hub.GetClientCount())
}
