# Building the Plugin Locally

This guide explains how to build the Grafana Scheduled Reports plugin from source.

## Prerequisites

- **Node.js** 18+ and npm
- **Go** 1.21+
- **Git**
- **zip** utility (for packaging)
- **Chromium/Chrome** browser for PDF rendering (standalone or system package)

## Quick Start

### Option 1: Using Makefile (Recommended)

```bash
# Show all available commands
make help

# Install dependencies
make install

# Build both frontend and backend
make build

# Create distribution package
make package

# Install directly to Grafana
sudo make install-plugin
```

### Option 2: Using build.sh script

```bash
# Make script executable (first time only)
chmod +x build.sh

# Build plugin and create archive
./build.sh
```

### Option 3: Manual build

```bash
# Install frontend dependencies
npm install

# Build frontend
npm run build

# Build backend
go build -o dist/gpx_reporting_linux_amd64 ./cmd/backend

# Make backend executable
chmod +x dist/gpx_reporting_linux_amd64
```

## Build Script Features

The `build.sh` script automatically:

1. ✓ Cleans previous builds
2. ✓ Installs frontend dependencies
3. ✓ Builds frontend (React/TypeScript)
4. ✓ Builds backend (Go) for your OS/architecture
5. ✓ Packages plugin files
6. ✓ Creates distribution archive

### Supported Platforms

**Operating Systems:**
- Linux (x86_64, arm64, arm)
- macOS (x86_64, arm64)
- Windows (x86_64)

**Output:**
- `dist/` - Plugin directory ready to use
- `scheduled-reports-app-{version}-{os}-{arch}.zip` - Distribution archive

## Installation

### Local Development

Copy the `dist/` folder to Grafana plugins directory:

```bash
# For Linux
sudo cp -r dist /var/lib/grafana/plugins/fulgerx2007-scheduledreports-app

# For macOS with Homebrew
cp -r dist /usr/local/var/lib/grafana/plugins/fulgerx2007-scheduledreports-app

# For Docker
docker cp dist <container>:/var/lib/grafana/plugins/fulgerx2007-scheduledreports-app
```

### Production Installation

Use the distribution archive:

```bash
# Extract to plugins directory
sudo unzip scheduled-reports-app-*.zip -d /var/lib/grafana/plugins/

# Set ownership (Linux)
sudo chown -R grafana:grafana /var/lib/grafana/plugins/fulgerx2007-scheduledreports-app

# Configure Grafana to allow unsigned plugin
# Edit /etc/grafana/grafana.ini:
[plugins]
allow_loading_unsigned_plugins = fulgerx2007-scheduledreports-app

# Restart Grafana
sudo systemctl restart grafana-server
```

## Configuration After Installation

1. **Enable the plugin** in Grafana UI:
   - Go to Administration → Plugins
   - Find "Scheduled Reports"
   - Click "Enable"

2. **Configure Grafana URL** in plugin Settings:
   - Go to Apps → Scheduled Reports → Settings
   - Set "Grafana URL" to your full Grafana URL:
     - Example: `https://127.0.0.1:3000/dna`
     - Include protocol (http/https)
     - Include port if not standard
     - Include subpath from root_url if configured

3. **Configure Chrome/Chromium path** (if using standalone binary):
   - Set "Chromium Path" to your Chrome binary location
   - Example: `./chrome-linux64/chrome`
   - Leave empty to auto-detect system Chrome

4. **Configure SMTP** (if not using Grafana's SMTP)

5. **Set limits** for your organization

## Development Mode

For active development with hot reload:

```bash
# Terminal 1: Frontend watch mode
npm run dev

# Terminal 2: Backend rebuild on change
make build-backend

# Or use the Makefile target
make dev
```

## Troubleshooting

### Build fails with "npm not found"

Install Node.js 18+ from nodejs.org

### Build fails with "go: command not found"

Install Go 1.21+ from golang.org

### Backend build fails with CGO errors

For systems without gcc:
```bash
CGO_ENABLED=0 go build -o dist/gpx_reporting ./cmd/backend
```

Note: CGO is required for SQLite. Install gcc/build-essential if needed.

### Permission denied running build.sh

Make it executable:
```bash
chmod +x build.sh
```

### Plugin not loading in Grafana

1. Check Grafana logs:
```bash
sudo journalctl -u grafana-server -f
```

2. Verify unsigned plugin is allowed in grafana.ini

3. Check file permissions:
```bash
ls -la /var/lib/grafana/plugins/fulgerx2007-scheduledreports-app/
```

### Rendering fails after installation

1. Configure Grafana URL in plugin Settings
2. Verify the URL is accessible from the plugin
3. Check Chrome/Chromium is installed:
   - Test: `chromium --version` or `google-chrome --version`
   - Or verify custom path: `./chrome-linux64/chrome --version`
4. Ensure required Chrome flags are enabled in Settings:
   - No Sandbox (required for Docker)
   - Disable GPU (recommended for servers)
   - Skip TLS Verify (if using self-signed certificates)

## Clean Build

To start fresh:

```bash
# Using Makefile
make clean

# Or manually
rm -rf dist/
rm -f *.zip
```

## Build for Different Platform

Cross-compilation example:

```bash
# For Linux on macOS
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o dist/gpx_reporting_linux_amd64 ./cmd/backend

# For ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -o dist/gpx_reporting_linux_arm64 ./cmd/backend
```

Note: Cross-compilation with CGO requires target platform toolchain.

## Testing Builds

```bash
# Run all tests
make test

# Or manually
go test ./pkg/... -v
npm test
```

## Support

- Documentation: [README.md](./README.md)
- Issues: https://github.com/FulgerX2007/grafana-scheduled-reports-app/issues
