# Sentinel File Watcher - Makefile

# Variables
BINARY_NAME=sentinel
VERSION?=$(shell date +%Y.%m.%d%H%M%S)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

# Directories
BIN_DIR=./bin
MAC_DIR=$(BIN_DIR)/mac
WIN_DIR=$(BIN_DIR)/windows

# Required files for deployment
REQUIRED_FILES=sentinel.yaml

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help SetupForMac SetupForWin clean build-mac build-win package-mac package-win build-frontend setup-all

# Default target
help:
	@echo "$(GREEN)Sentinel File Watcher - Build & Setup$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@echo "  make SetupForWin  - Build for Windows and create setup.zip"
	@echo "  make SetupForMac  - Build for macOS and create setup.zip"
	@echo "  make setup-all    - Build for both macOS and Windows"
	@echo "  make build-win    - Build Windows binary only"
	@echo "  make build-mac    - Build macOS binary only"
	@echo "  make build-frontend - Build Angular frontend (dist/)"
	@echo "  make clean        - Clean build artifacts"
	@echo ""

# ============================================================
# Windows
# ============================================================

# Setup for Windows - full pipeline
SetupForWin: clean build-win package-win
	@echo "$(GREEN)✓ Windows setup package created successfully!$(NC)"
	@echo "$(YELLOW)Package location: $(WIN_DIR)/setup.zip$(NC)"
	@python3 -c "import subprocess; import os; path = os.path.abspath('$(WIN_DIR)/setup.zip'); subprocess.run(['osascript', '-e', f'set the clipboard to (POSIX file \"{path}\") as «class furl»'])" 2>/dev/null || true
	@echo "$(GREEN)✓ File copied to clipboard! You can paste it directly.$(NC)"

# Build for Windows
build-win:
	@echo "$(YELLOW)Building for Windows (amd64)...$(NC)"
	@mkdir -p $(WIN_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(WIN_DIR)/$(BINARY_NAME).exe ./cmd/sentinel
	@echo "$(GREEN)✓ Windows binary built successfully$(NC)"

# Package for Windows
package-win:
	@echo "$(YELLOW)Packaging Windows setup...$(NC)"
	@# Copy required config files
	@for file in $(REQUIRED_FILES); do \
		if [ -f $$file ]; then \
			echo "$(GREEN)  Copying $$file$(NC)"; \
			cp $$file $(WIN_DIR)/; \
		else \
			echo "$(RED)  Warning: $$file not found$(NC)"; \
		fi \
	done
	@# Fix sentinel.yaml paths for Windows defaults
	@echo "$(YELLOW)  Creating Windows-compatible sentinel.yaml...$(NC)"
	@sed -e 's|path: .*sentinel\.db|path: ./sentinel.db|' \
	     -e 's|file: /var/log/sentinel\.log|file: ./sentinel.log|' \
	     -e 's|path: /Users/.*|path: C:\\\\Sentinel\\\\watch1|' \
	     sentinel.yaml > $(WIN_DIR)/sentinel.yaml
	@# Create Windows install script
	@echo "$(YELLOW)  Creating install.bat...$(NC)"
	@printf '@echo off\r\necho ============================================\r\necho  Sentinel File Watcher - Windows Installer\r\necho ============================================\r\necho.\r\n\r\nREM Check admin rights\r\nnet session >nul 2>&1\r\nif %%errorlevel%% neq 0 (\r\n    echo ERROR: Please run as Administrator!\r\n    echo Right-click this file and select "Run as administrator"\r\n    pause\r\n    exit /b 1\r\n)\r\n\r\nset INSTALL_DIR=C:\\Sentinel\r\n\r\necho Installing Sentinel to %%INSTALL_DIR%%...\r\necho.\r\n\r\nREM Create install directory\r\nif not exist "%%INSTALL_DIR%%" mkdir "%%INSTALL_DIR%%"\r\n\r\nREM Copy files\r\ncopy /Y sentinel.exe "%%INSTALL_DIR%%\\"\r\nif not exist "%%INSTALL_DIR%%\\sentinel.yaml" (\r\n    copy /Y sentinel.yaml "%%INSTALL_DIR%%\\"\r\n    echo Config file copied. Edit %%INSTALL_DIR%%\\sentinel.yaml before starting.\r\n) else (\r\n    echo Config file already exists, skipping.\r\n)\r\n\r\nREM Create watch directories\r\nif not exist "%%INSTALL_DIR%%\\watch1" mkdir "%%INSTALL_DIR%%\\watch1"\r\n\r\nREM Install as Windows Service\r\necho.\r\necho Installing as Windows Service...\r\n"%%INSTALL_DIR%%\\sentinel.exe" install "%%INSTALL_DIR%%\\sentinel.yaml"\r\n\r\necho.\r\necho ============================================\r\necho  Installation complete!\r\necho ============================================\r\necho.\r\necho Config file : %%INSTALL_DIR%%\\sentinel.yaml\r\necho Watch dir   : %%INSTALL_DIR%%\\watch1\r\necho Web UI      : http://localhost:8083\r\necho.\r\necho Commands:\r\necho   sentinel start    - Start the service\r\necho   sentinel stop     - Stop the service\r\necho   sentinel restart  - Restart the service\r\necho   sentinel status   - Show service status\r\necho.\r\necho To start now:\r\necho   "%%INSTALL_DIR%%\\sentinel.exe" start\r\necho.\r\npause\r\n' > $(WIN_DIR)/install.bat
	@# Create uninstall script
	@echo "$(YELLOW)  Creating uninstall.bat...$(NC)"
	@printf '@echo off\r\necho Uninstalling Sentinel...\r\n\r\nnet session >nul 2>&1\r\nif %%errorlevel%% neq 0 (\r\n    echo ERROR: Please run as Administrator!\r\n    pause\r\n    exit /b 1\r\n)\r\n\r\nset INSTALL_DIR=C:\\Sentinel\r\n\r\necho Stopping service...\r\n"%%INSTALL_DIR%%\\sentinel.exe" stop 2>nul\r\necho Uninstalling service...\r\n"%%INSTALL_DIR%%\\sentinel.exe" uninstall\r\n\r\necho.\r\necho Service uninstalled. Files remain in %%INSTALL_DIR%%\r\necho Delete the folder manually if no longer needed.\r\npause\r\n' > $(WIN_DIR)/uninstall.bat
	@# Create README
	@echo "$(YELLOW)  Creating README.txt...$(NC)"
	@printf 'Sentinel File Watcher - Windows Setup\r\n======================================\r\n\r\nVersion: $(VERSION)\r\nBuild:   $(BUILD_TIME)\r\n\r\nQUICK INSTALL\r\n-------------\r\n1. Right-click install.bat -> "Run as administrator"\r\n2. Edit C:\\Sentinel\\sentinel.yaml (set watch paths and webhook URLs)\r\n3. Start: C:\\Sentinel\\sentinel.exe start\r\n4. Open: http://localhost:8083\r\n\r\nMANUAL INSTALL\r\n--------------\r\n1. Copy sentinel.exe and sentinel.yaml to your desired directory\r\n2. Edit sentinel.yaml:\r\n   - Set watcher paths (directories to monitor)\r\n   - Set webhook URLs (where to send notifications)\r\n   - Set server port (default: 8083)\r\n3. Install as service:  sentinel.exe install\r\n4. Start service:       sentinel.exe start\r\n\r\nSTANDALONE (no service)\r\n-----------------------\r\nJust run: sentinel.exe\r\nThis runs in foreground with default config (sentinel.yaml)\r\n\r\nCOMMANDS\r\n--------\r\n  sentinel.exe install   - Install as Windows Service\r\n  sentinel.exe uninstall - Uninstall the Windows Service\r\n  sentinel.exe start     - Start the service\r\n  sentinel.exe stop      - Stop the service\r\n  sentinel.exe restart   - Restart the service\r\n  sentinel.exe status    - Show service status\r\n  sentinel.exe run       - Run in foreground (interactive)\r\n  sentinel.exe version   - Show version\r\n\r\nFILES\r\n-----\r\n  sentinel.exe   - Application binary (includes embedded web UI)\r\n  sentinel.yaml  - Configuration file\r\n  sentinel.db    - SQLite database (auto-created on first run)\r\n\r\nWEB UI\r\n------\r\nThe web dashboard is embedded in the binary.\r\nAccess it at: http://localhost:<port> (default 8083)\r\n\r\nFeatures:\r\n  - Dashboard with real-time file events\r\n  - Watcher management (add/edit/delete/start/stop)\r\n  - Event log viewer with filtering and Excel export\r\n  - WebSocket live updates\r\n\r\nCONFIG EXAMPLE\r\n--------------\r\nwatchers:\r\n  - name: "my-watcher"\r\n    path: "C:\\\\Data\\\\uploads"\r\n    mode: "watch"\r\n    recursive: true\r\n    filters:\r\n      include: ["*.csv", "*.json"]\r\n      exclude: ["*.tmp"]\r\n    webhook:\r\n      url: "https://your-api.com/webhook"\r\n      headers:\r\n        Authorization: "Bearer your-token"\r\n      timeout: 10s\r\n      retry:\r\n        max_attempts: 3\r\n        backoff: 2s\r\n' > $(WIN_DIR)/README.txt
	@# Copy URL shortcut
	@if [ -f Sentinel.url ]; then \
		echo "$(GREEN)  Copying Sentinel.url$(NC)"; \
		cp Sentinel.url $(WIN_DIR)/; \
	fi
	@# Create zip package
	@echo "$(YELLOW)  Creating setup.zip...$(NC)"
	@cd $(WIN_DIR) && zip -r setup.zip * -x "*.DS_Store" "setup.zip"
	@echo "$(GREEN)✓ Package created: $(WIN_DIR)/setup.zip$(NC)"

# ============================================================
# macOS
# ============================================================

# Setup for macOS - full pipeline
SetupForMac: clean build-mac package-mac
	@echo "$(GREEN)✓ macOS setup package created successfully!$(NC)"
	@echo "$(YELLOW)Package location: $(MAC_DIR)/setup.zip$(NC)"
	@python3 -c "import subprocess; import os; path = os.path.abspath('$(MAC_DIR)/setup.zip'); subprocess.run(['osascript', '-e', f'set the clipboard to (POSIX file \"{path}\") as «class furl»'])" 2>/dev/null || true
	@echo "$(GREEN)✓ File copied to clipboard! You can paste it directly.$(NC)"

# Build for macOS (detect architecture)
build-mac:
	@echo "$(YELLOW)Building for macOS...$(NC)"
	@mkdir -p $(MAC_DIR)
ifeq ($(shell uname -m),arm64)
	@echo "$(GREEN)  Detected Apple Silicon (ARM64)$(NC)"
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(MAC_DIR)/$(BINARY_NAME) ./cmd/sentinel
else
	@echo "$(GREEN)  Detected Intel (AMD64)$(NC)"
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(MAC_DIR)/$(BINARY_NAME) ./cmd/sentinel
endif
	@chmod +x $(MAC_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)✓ macOS binary built successfully$(NC)"

# Package for macOS
package-mac:
	@echo "$(YELLOW)Packaging macOS setup...$(NC)"
	@# Copy required config files
	@for file in $(REQUIRED_FILES); do \
		if [ -f $$file ]; then \
			echo "$(GREEN)  Copying $$file$(NC)"; \
			cp $$file $(MAC_DIR)/; \
		else \
			echo "$(RED)  Warning: $$file not found$(NC)"; \
		fi \
	done
	@# Create README
	@echo "$(YELLOW)  Creating README.md...$(NC)"
	@echo "# Sentinel File Watcher - macOS Setup\n\n**Version:** $(VERSION)\n**Build:** $(BUILD_TIME)\n\n## Quick Start\n\n\`\`\`bash\n# 1. Edit config\nvi sentinel.yaml\n\n# 2. Run\n./sentinel\n\n# 3. Open browser\nopen http://localhost:8083\n\`\`\`\n\n## Install as Service\n\n\`\`\`bash\nsudo ./sentinel install\nsudo ./sentinel start\n\`\`\`\n\n## Commands\n\n| Command | Description |\n|---------|-------------|\n| \`./sentinel\` | Run in foreground |\n| \`./sentinel install\` | Install as launchd service |\n| \`./sentinel uninstall\` | Uninstall service |\n| \`./sentinel start\` | Start service |\n| \`./sentinel stop\` | Stop service |\n| \`./sentinel restart\` | Restart service |\n| \`./sentinel status\` | Show status |" > $(MAC_DIR)/README.md
	@# Create zip package
	@echo "$(YELLOW)  Creating setup.zip...$(NC)"
	@cd $(MAC_DIR) && zip -r setup.zip * -x "*.DS_Store" "setup.zip"
	@echo "$(GREEN)✓ Package created: $(MAC_DIR)/setup.zip$(NC)"

# ============================================================
# Frontend
# ============================================================

# Build Angular frontend
build-frontend:
	@echo "$(YELLOW)Building Angular frontend...$(NC)"
	@cd web/sentinel-ui && npm run build
	@echo "$(GREEN)✓ Frontend built successfully$(NC)"

# ============================================================
# Common
# ============================================================

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BIN_DIR)
	@echo "$(GREEN)✓ Clean complete$(NC)"

# Build both platforms
setup-all: clean build-mac build-win package-mac package-win
	@echo "$(GREEN)✓ All setup packages created successfully!$(NC)"
	@echo "$(YELLOW)  Mac package:     $(MAC_DIR)/setup.zip$(NC)"
	@echo "$(YELLOW)  Windows package:  $(WIN_DIR)/setup.zip$(NC)"
