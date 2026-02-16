# Sentinel

**Cross-platform file system watcher with real-time webhook notifications and web-based management interface.**

Sentinel monitors directories for file changes (create, modify, delete, rename) and delivers instant HTTP POST webhook notifications. Built as a single binary combining a Go backend service with an Angular 21 web interface for complete management and monitoring.

---

## ✨ Features

### Core Capabilities
- **Real-time monitoring** — Detects file events instantly with fsnotify (local) or polling (network shares)
- **Dual monitoring modes** — Watch mode (fsnotify) for local directories, Poll mode for network/UNC paths
- **Webhook delivery** — Sends structured JSON payloads via HTTP POST with retry and backoff
- **Web management UI** — Modern Angular 21 interface for configuration and monitoring
- **Multiple watchers** — Monitor multiple directories simultaneously with independent configurations
- **WebSocket streaming** — Real-time event notifications pushed to connected web clients
- **Event logging** — SQLite database tracks all file events and webhook deliveries
- **Excel export** — Export event logs to Excel format with filtering and search
- **Cross-platform service** — Runs as a Windows Service, Linux systemd unit, or macOS LaunchDaemon

### Management & Monitoring
- **Dashboard** — Overview with system stats, active watchers, recent events, and webhook success rates
- **Watcher control** — Start, stop, restart individual watchers without affecting others
- **Path validation** — Verify directory paths before saving configuration
- **Webhook testing** — Send test requests to validate endpoint configuration
- **Advanced filtering** — Include/exclude file patterns with glob support
- **Log viewer** — Searchable, paginated event history with detailed webhook responses
- **Graceful restarts** — Apply configuration changes with visual countdown overlay
- **Multi-language** — Full support for English and Turkish (i18n)

---

## 🚀 Quick Start

### Prerequisites

- Go 1.22+ (for building from source)
- Node.js 20.19+ or 22.12+ (for building Angular UI)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/<your-username>/sentinel.git
cd sentinel

# Build Angular frontend
cd web/sentinel-ui
npm install
npm run build

# Build Go backend (embeds Angular dist)
cd ../..

# Copy config file
cp sentinel.yaml.example sentinel.yaml
# Edit sentinel.yaml with your settings

go build -o sentinel ./cmd/sentinel

# Run in interactive mode
./sentinel
```

The web interface will be available at `http://localhost:8080` (default port).

### Install as System Service

```bash
# Linux systemd
sudo ./sentinel install
sudo systemctl start sentinel
sudo systemctl enable sentinel

# Windows
sentinel.exe install
sc start Sentinel

# macOS launchd
sudo ./sentinel install
sudo launchctl load /Library/LaunchDaemons/com.sentinel.service.plist
```

---

## 📁 Configuration

Sentinel uses a `sentinel.yaml` configuration file. Copy `sentinel.yaml.example` to `sentinel.yaml` and edit it to match your environment.

### Minimal Example

```yaml
watchers:
  - name: "uploads"
    path: "/data/uploads"
    mode: "watch"
    recursive: true
    filters:
      include: ["*.json", "*.xml", "*.csv"]
      exclude: ["*.tmp", "*.swp"]
    webhook:
      url: "https://api.example.com/hooks/uploads"
      headers:
        Authorization: "Bearer your-token-here"
      timeout: 10s
      retry:
        max_attempts: 3
        backoff: 2s

server:
  port: 8080
  host: "0.0.0.0"

logging:
  level: "info"
```

### Network Shares

For network drives or UNC paths, use **poll mode**:

```yaml
watchers:
  - name: "network-reports"
    path: "\\\\fileserver\\shared\\reports"
    mode: "poll"
    poll_interval: 5s
    recursive: true
    webhook:
      url: "https://api.example.com/hooks/reports"
```

**Why polling?** fsnotify doesn't reliably detect changes on network file systems. Polling mode checks for file modifications at regular intervals using hash comparison.

---

## 🌐 Web Interface

Access the web UI at `http://localhost:8080` (or configured port).

### Dashboard
- System overview with uptime and statistics
- Active watcher count and webhook success rate
- Real-time event stream (last 20 events)
- Grouped events by watcher directory

### Setup Page
- Add, edit, delete watchers
- Start/stop/restart individual watchers
- Validate directory paths before saving
- Test webhook endpoints with mock payloads
- View watcher-specific event logs

### Logs Page
- Unified event log across all watchers
- Filter by watcher, event type, date range
- Search by filename
- Pagination with customizable page size
- Export filtered logs to Excel

### Real-time Updates
WebSocket connection provides live updates:
- New file events appear instantly
- Webhook delivery status updates
- Watcher state changes (started/stopped)

---

## 📡 API Endpoints

The backend exposes a REST API for programmatic control:

### Watchers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/watchers` | List all watchers with status |
| POST | `/api/v1/watchers` | Create new watcher |
| GET | `/api/v1/watchers/:id` | Get watcher details |
| PUT | `/api/v1/watchers/:id` | Update watcher configuration |
| DELETE | `/api/v1/watchers/:id` | Delete watcher |

### Control

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/watchers/:id/start` | Start a watcher |
| POST | `/api/v1/watchers/:id/stop` | Stop a watcher |
| POST | `/api/v1/watchers/:id/restart` | Restart a watcher |
| POST | `/api/v1/watchers/validate-path` | Validate directory path |
| POST | `/api/v1/watchers/:id/test` | Test webhook endpoint |

### Logs & Stats

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/logs` | Get all event logs (paginated) |
| GET | `/api/v1/watchers/:id/logs` | Get watcher-specific logs |
| GET | `/api/v1/watchers/:id/logs/export` | Export logs to Excel |
| GET | `/api/v1/watchers/:id/stats` | Get webhook statistics |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/status` | System status and health |
| POST | `/api/v1/service/restart` | Restart the entire service |
| GET | `/api/v1/config/pending` | Check for pending config changes |

### WebSocket

| Endpoint | Description |
|----------|-------------|
| GET `/ws` | WebSocket connection for real-time events |

---

## 📦 Webhook Payload

Sentinel sends this JSON payload for each file system event:

```json
{
  "watcher_id": 1,
  "watcher_name": "uploads",
  "event": "created",
  "file_name": "report.csv",
  "file_path": "/data/uploads/report.csv",
  "file_size": 204800,
  "timestamp": "2025-02-14T10:30:01Z"
}
```

### Event Types

| Event | Description |
|-------|-------------|
| `created` | New file added to directory |
| `modified` | Existing file content changed |
| `deleted` | File removed from directory |
| `renamed` | File renamed (includes old_path field) |

### Webhook Response

Your endpoint should return:
- **Status 200-299** → Success (logged and marked as delivered)
- **Status 400-499** → Client error (no retry, logged as failed)
- **Status 500-599** → Server error (retried with exponential backoff)

---

## 🔄 Service Management

### Interactive Mode

```bash
# Run in foreground with console output
./sentinel

# Access web UI at http://localhost:8080
```

Press `Ctrl+C` for graceful shutdown.

### Service Mode

#### Linux (systemd)

```bash
# Install service
sudo ./sentinel install

# Start service
sudo systemctl start sentinel

# Enable auto-start on boot
sudo systemctl enable sentinel

# View logs
sudo journalctl -u sentinel -f

# Stop service
sudo systemctl stop sentinel

# Uninstall service
sudo ./sentinel uninstall
```

#### Windows

```powershell
# Run as Administrator

# Install service
sentinel.exe install

# Start service
sc start Sentinel
# Or use Services panel (services.msc)

# Stop service
sc stop Sentinel

# Uninstall service
sentinel.exe uninstall
```

#### macOS (launchd)

```bash
# Install service
sudo ./sentinel install

# Load service
sudo launchctl load /Library/LaunchDaemons/com.sentinel.service.plist

# Unload service
sudo launchctl unload /Library/LaunchDaemons/com.sentinel.service.plist

# Uninstall service
sudo ./sentinel uninstall
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    SENTINEL BINARY                          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              GO BACKEND SERVICE                      │   │
│  │                                                      │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌────────┐│   │
│  │  │   Watcher    │───▶│ Event Queue  │───▶│Webhook ││   │
│  │  │   Manager    │    │  (buffered)  │    │Dispatch││   │
│  │  │              │    │              │    │        ││   │
│  │  │  - fsnotify  │    │  Debounce    │    │ Retry  ││   │
│  │  │  - poller    │    │  500ms       │    │ Logic  ││   │
│  │  └──────┬───────┘    └──────┬───────┘    └───┬────┘│   │
│  │         │                   │                 │     │   │
│  │         ▼                   ▼                 ▼     │   │
│  │  ┌────────────────────────────────────────────────┐│   │
│  │  │            SQLite Database                     ││   │
│  │  │  - Watchers  - Events  - Webhook Logs         ││   │
│  │  └────────────────────────────────────────────────┘│   │
│  │         │                                           │   │
│  │         ▼                                           │   │
│  │  ┌────────────────┐         ┌──────────────────┐   │   │
│  │  │  REST API      │◀───────▶│  WebSocket Hub   │   │   │
│  │  │  (Gin)         │         │  (gorilla/ws)    │   │   │
│  │  └────────┬───────┘         └────────┬─────────┘   │   │
│  └───────────┼──────────────────────────┼─────────────┘   │
│              │                          │                 │
│              ▼                          ▼                 │
│  ┌─────────────────────────────────────────────────────┐  │
│  │          ANGULAR 21 WEB INTERFACE                   │  │
│  │          (Embedded via Go embed.FS)                 │  │
│  │                                                     │  │
│  │  Dashboard  │  Setup  │  Logs  │  Real-time WS     │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                        │
                        ▼
                ┌───────────────┐
                │  Your API     │
                │  Endpoint     │
                └───────────────┘
```

### Key Components

- **Watcher Manager** — Lifecycle management for multiple watchers
- **FSWatcher** — fsnotify-based local file monitoring with debouncing
- **Poller** — Hash-based polling for network shares
- **Webhook Dispatcher** — Concurrent worker pool with retry logic
- **WebSocket Hub** — Real-time event broadcasting to connected clients
- **GORM + SQLite** — Event persistence and query layer
- **Gin Router** — REST API and static file serving
- **Angular 21** — Standalone components, signals, Material Design

---

## 🛠️ Technology Stack

### Backend (Go 1.22+)
- **Web Framework:** gin-gonic/gin
- **File Monitoring:** fsnotify/fsnotify
- **Database:** GORM + github.com/glebarez/sqlite (pure Go, no CGO)
- **WebSocket:** gorilla/websocket
- **Configuration:** spf13/viper (YAML)
- **Service:** kardianos/service (Windows/Linux/macOS)
- **Excel Export:** qax-os/excelize

### Frontend (Angular 21)
- **Framework:** Angular 21 standalone components
- **UI Library:** Angular Material
- **State Management:** Signals (Angular built-in)
- **HTTP Client:** HttpClient with interceptors
- **WebSocket:** Native WebSocket API
- **i18n:** Angular i18n with English and Turkish
- **Animations:** Angular animations

### Database
- **Engine:** SQLite (embedded, single file)
- **Location:** `sentinel.db` in working directory
- **Tables:** watchers, events, webhook_logs

---

## 🔧 Development

### Project Structure

```
sentinel/
├── cmd/sentinel/main.go              # Entry point
├── internal/
│   ├── api/                          # REST API handlers
│   ├── config/                       # Configuration management
│   ├── watcher/                      # File monitoring engine
│   ├── webhook/                      # HTTP POST dispatcher
│   ├── ws/                           # WebSocket hub
│   ├── storage/                      # Database layer (GORM)
│   ├── export/                       # Excel export
│   └── service/                      # OS service wrapper
├── web/sentinel-ui/                  # Angular 21 app
│   ├── src/app/
│   │   ├── core/                     # Services, interceptors, models
│   │   ├── features/                 # Dashboard, Setup, Logs
│   │   └── shared/                   # Reusable components, pipes
│   └── dist/                         # Build output (embedded)
├── sentinel.yaml                     # Configuration file
└── sentinel.db                       # SQLite database
```

### Building

```bash
# Backend only (for testing without UI)
go build -o sentinel ./cmd/sentinel

# Full build (Angular + Go)
./build-all.sh
```

### Running in Development

**Backend:**
```bash
go run cmd/sentinel/main.go
```

**Frontend (separate dev server):**
```bash
cd web/sentinel-ui
npm start
# Access at http://localhost:4200
# Uses proxy.conf.json to forward API calls to :8080
```

---

## 🔐 Security

### Built-in Protections

✅ **SQL Injection Protection** — All database operations use GORM ORM with parameterized queries. No raw SQL concatenation.

✅ **XSS Protection** — Angular's built-in DomSanitizer actively prevents cross-site scripting attacks. No `innerHTML` or unsafe DOM manipulation.

✅ **Path Traversal Protection** — Directory paths validated before use. Embedded frontend prevents directory traversal attacks.

✅ **Input Validation** — All API endpoints validate inputs. Frontend uses reactive forms with validators.

✅ **No Hard-coded Secrets** — All credentials, tokens, and API keys loaded from `sentinel.yaml` configuration file.

✅ **CORS Configuration** — Cross-Origin Resource Sharing settings fully configurable via `sentinel.yaml`.

✅ **TLS/HTTPS Support** — Optional HTTPS mode with certificate configuration for encrypted connections.

✅ **Error Handling** — Detailed errors logged server-side only. Generic error messages returned to clients (no information disclosure).

### Important Security Notes

⚠️ **Localhost Deployment Only**

Sentinel is designed for **localhost deployment** as a file monitoring service. It is **not designed for internet exposure**:

- No built-in authentication/authorization system
- WebSocket connections have basic origin validation but no user authentication
- API endpoints are unprotected (suitable for localhost only)

**Production Deployment Recommendations:**

1. **Use Specific CORS Origins** — In production, replace `allowed_origins: ["*"]` with exact domains:
   ```yaml
   server:
     cors:
       allowed_origins: ["https://sentinel.yourdomain.com"]
   ```

2. **Enable TLS/HTTPS** — For any network-accessible deployment:
   ```yaml
   server:
     tls:
       enabled: true
       cert_file: "/path/to/cert.pem"
       key_file: "/path/to/key.pem"
   ```

3. **Firewall Protection** — If exposing to network, use firewall rules to restrict access:
   ```bash
   # Allow only specific IP range
   sudo ufw allow from 192.168.1.0/24 to any port 8080
   ```

4. **Reverse Proxy with Auth** — For remote access, deploy behind nginx/Apache with Basic Auth:
   ```nginx
   location / {
       auth_basic "Sentinel Access";
       auth_basic_user_file /etc/nginx/.htpasswd;
       proxy_pass http://localhost:8080;
   }
   ```

5. **Webhook Security** — Always use HTTPS webhooks with proper authentication:
   ```yaml
   webhook:
     url: "https://api.example.com/hooks/uploads"
     headers:
       Authorization: "Bearer secret-token-here"
   ```

### Security Audit Results

The codebase has been audited for common vulnerabilities:

| Vulnerability | Status | Notes |
|---------------|--------|-------|
| SQL Injection | ✅ Protected | GORM ORM, no raw SQL |
| XSS | ✅ Protected | Angular sanitization active |
| CSRF | ✅ Protected | HttpClient auto-handles CSRF tokens |
| Path Traversal | ✅ Protected | Path validation implemented |
| Hard-coded Secrets | ✅ None found | All externalized to config |
| SSRF (Webhooks) | ✅ Protected | HTTP client validates URLs |
| Info Disclosure | ✅ Protected | Generic error messages to clients |

**Overall Security Rating:** Excellent for intended use case (localhost service)

---

## 🐛 Troubleshooting

### "Watcher failed to start"
- Verify directory path exists and is accessible
- Check file system permissions
- For network paths, use `mode: "poll"` instead of `mode: "watch"`

### "Webhook delivery failed"
- Test webhook endpoint with `/api/v1/watchers/:id/test`
- Verify network connectivity and firewall rules
- Check webhook URL and authentication headers
- Review retry configuration (max_attempts, backoff)

### "Database locked" error
- Ensure only one Sentinel instance is running
- Check file permissions on `sentinel.db`
- SQLite doesn't support multiple concurrent writers

### High CPU usage
- Reduce polling frequency (`poll_interval`) for network watchers
- Use more specific file filters (include/exclude patterns)
- Limit recursive watching to necessary directories

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

<p align="center">
  <b>Sentinel</b> — Real-time file monitoring, powerful webhooks, beautiful management.
</p>
