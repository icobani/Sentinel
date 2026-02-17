# Sentinel

**Cross-platform file system watcher with real-time webhook notifications and web-based management interface.**

Sentinel monitors directories for file changes (create, modify, delete, rename) and delivers instant HTTP POST webhook notifications. Built as a single binary — the Go backend and Angular web interface are embedded together, no separate installations needed.

<!-- TODO: Add a hero screenshot here -->
<!-- ![Sentinel Dashboard](docs/screenshots/dashboard.png) -->

---

## Features

- **Real-time file monitoring** — Detects changes instantly via fsnotify (local) or polling (network shares)
- **Webhook delivery** — HTTP POST with configurable retry and exponential backoff
- **Web management UI** — Dashboard, watcher management, log viewer — all from your browser
- **Multiple watchers** — Monitor many directories simultaneously with independent configs
- **WebSocket live updates** — Events appear in the UI the moment they happen
- **File pattern filtering** — Include/exclude with glob patterns (case-insensitive)
- **Excel export** — Download filtered event logs as `.xlsx`
- **Bulk log management** — Select and delete logs from the UI
- **Auto-cleanup** — Configurable retention period (default 30 days)
- **Cross-platform** — Runs on Windows, macOS, Linux, Raspberry Pi
- **System service** — Install as Windows Service, systemd unit, or macOS LaunchDaemon
- **Multi-language** — English and Turkish (i18n)

---

## Screenshots

### Dashboard

Real-time overview with system stats, active watchers, and recent events grouped by directory.

<!-- ![Dashboard](docs/screenshots/dashboard.png) -->
*Screenshot: Dashboard with summary cards and live event stream*

### Setup

Add, edit, and control watchers. Validate directory paths, test webhook endpoints, start/stop individual watchers.

<!-- ![Setup](docs/screenshots/setup.png) -->
*Screenshot: Watcher list with status indicators and quick actions*

### Watcher Form

Configure watchers with file filters, webhook URL, custom headers, retry settings, and path validation.

<!-- ![Watcher Form](docs/screenshots/watcher-form.png) -->
*Screenshot: Watcher creation form with path validation and webhook test*

### Logs

Searchable, filterable event log with pagination and Excel export. Select and delete logs in bulk.

<!-- ![Logs](docs/screenshots/logs.png) -->
*Screenshot: Log viewer with filters, search, and bulk delete*

---

## Installation

### macOS (Homebrew)

```bash
brew tap icobani/sentinel
brew install icobani/sentinel/sentinel
sentinel
```

On first run, Sentinel automatically creates a default `sentinel.yaml` in the current directory if one doesn't exist.
You can then add watchers through the web UI at `http://localhost:8083` or edit the config file directly.

### Windows (Scoop)

```powershell
scoop bucket add sentinel https://github.com/icobani/scoop-sentinel
scoop install sentinel
```

```powershell
# Copy and edit config
cp ~\scoop\apps\sentinel\current\sentinel.yaml.example sentinel.yaml
# Edit sentinel.yaml, then run
sentinel.exe
```

### Windows (Manual)

Download `sentinel-windows-amd64.zip` from [Releases](https://github.com/icobani/Sentinel/releases/latest), extract, edit `sentinel.yaml`, and run.

```powershell
sentinel.exe

# Or install as Windows Service (run as Administrator)
sentinel.exe install
sentinel.exe start
```

### Ubuntu / Debian (x64)

```bash
curl -sL https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-linux-amd64.tar.gz | tar xz
sudo mv sentinel /usr/local/bin/
sudo cp sentinel.yaml.example /etc/sentinel.yaml
sudo nano /etc/sentinel.yaml
sentinel
```

### Raspberry Pi (ARM64)

```bash
curl -sL https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-linux-arm64.tar.gz | tar xz
sudo mv sentinel /usr/local/bin/
sudo cp sentinel.yaml.example /etc/sentinel.yaml
sudo nano /etc/sentinel.yaml
sentinel
```

### Linux systemd Service

```bash
sudo sentinel install
sudo systemctl start sentinel
sudo systemctl enable sentinel

# View logs
sudo journalctl -u sentinel -f
```

### Build from Source

Requires Go 1.22+ and Node.js 22+.

```bash
git clone https://github.com/icobani/Sentinel.git
cd Sentinel
cd web/sentinel-ui && npm install && npm run build && cd ../..
go build -o sentinel ./cmd/sentinel
./sentinel
```

---

## Updating

### macOS (Homebrew)

```bash
brew update
brew upgrade sentinel
```

### Windows (Scoop)

```powershell
scoop update sentinel
```

### Linux / Raspberry Pi (Manual)

Download the latest release and replace the binary:

```bash
# x64
curl -sL https://github.com/icobani/Sentinel/releases/latest/download/sentinel-linux-amd64.tar.gz | tar xz
sudo mv sentinel /usr/local/bin/

# ARM64 (Raspberry Pi)
curl -sL https://github.com/icobani/Sentinel/releases/latest/download/sentinel-linux-arm64.tar.gz | tar xz
sudo mv sentinel /usr/local/bin/
```

If running as a systemd service, restart after update:

```bash
sudo systemctl restart sentinel
```

Your `sentinel.yaml` and `sentinel.db` are preserved across updates.

---

## Configuration

Copy `sentinel.yaml.example` to `sentinel.yaml` and edit:

```yaml
watchers:
  - name: "uploads"
    path: "/data/uploads"
    mode: "watch"              # "watch" (local) or "poll" (network)
    recursive: true
    filters:
      include: ["*.json", "*.csv"]
      exclude: ["*.tmp"]
    webhook:
      url: "https://api.example.com/hooks/uploads"
      headers:
        Authorization: "Bearer your-token"
      timeout: 10s
      retry:
        max_attempts: 3
        backoff: 2s

server:
  port: 8083
  host: "0.0.0.0"

database:
  path: ./sentinel.db
  retention: 30d               # Auto-delete events older than this
```

For network shares (UNC paths), use `mode: "poll"` — fsnotify doesn't reliably detect changes on network file systems.

---

## How It Works

```
  File Change Detected
         │
         ▼
  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
  │   Watcher    │────▶│  Event Queue │────▶│   Webhook    │────▶ Your API
  │   Engine     │     │  (debounce)  │     │  Dispatcher  │
  └──────────────┘     └──────┬───────┘     └──────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
             ┌────────────┐    ┌──────────────┐
             │   SQLite   │    │  WebSocket   │────▶ Browser UI
             │  Database  │    │  Broadcast   │
             └────────────┘    └──────────────┘
```

1. **Watcher Engine** monitors directories using fsnotify (local) or polling (network)
2. Events are debounced (500ms) and saved to SQLite
3. **Webhook Dispatcher** sends HTTP POST to your endpoint with retry logic
4. **WebSocket** broadcasts events to all connected browser clients in real-time
5. **Web UI** displays everything — dashboard, watcher management, searchable logs

---

## Webhook Payload

Sentinel sends this JSON for each file event:

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

Event types: `created`, `modified`, `deleted`, `renamed`

Your endpoint should return `2xx` for success, `5xx` triggers retry with exponential backoff.

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.22+, Gin, GORM, SQLite |
| Frontend | Angular 21, Material UI, Signals |
| File Watch | fsnotify (local), polling (network) |
| WebSocket | gorilla/websocket |
| Config | Viper (YAML) |
| Service | kardianos/service |
| Export | excelize (Excel) |

---

## License

MIT License — see [LICENSE](LICENSE) for details.
