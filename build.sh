#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Plugin information
PLUGIN_ID="scheduled-reports-app"
PLUGIN_VERSION=$(grep '"version":' src/plugin.json | head -1 | sed 's/.*"version": *"\([^"]*\)".*/\1/')

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Building Grafana Reporting Plugin${NC}"
echo -e "${BLUE}  Version: ${PLUGIN_VERSION}${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Clean previous builds
echo -e "${YELLOW}[1/7]${NC} Cleaning previous builds..."
rm -rf dist/
mkdir -p dist/
echo -e "${GREEN}✓${NC} Clean complete"
echo ""

# Step 2: Install frontend dependencies
echo -e "${YELLOW}[2/7]${NC} Installing frontend dependencies..."
if ! npm install --silent; then
    echo -e "${RED}✗ Failed to install frontend dependencies${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Frontend dependencies installed"
echo ""

# Step 3: Build frontend
echo -e "${YELLOW}[3/7]${NC} Building frontend..."
if ! npm run build; then
    echo -e "${RED}✗ Frontend build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Frontend built successfully"
echo ""

# Step 4: Build backend
echo -e "${YELLOW}[4/7]${NC} Building backend..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        GOARCH="amd64"
        ;;
    aarch64|arm64)
        GOARCH="arm64"
        ;;
    armv7l)
        GOARCH="arm"
        ;;
    *)
        echo -e "${RED}✗ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        GOOS="linux"
        ;;
    darwin)
        GOOS="darwin"
        ;;
    mingw*|msys*|cygwin*)
        GOOS="windows"
        ;;
    *)
        echo -e "${RED}✗ Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

# Binary name with platform (matches plugin.json executable field with {{.OS}}_{{.ARCH}})
PLUGIN_BINARY_NAME="gpx_reporting_${GOOS}_${GOARCH}"
if [ "$GOOS" = "windows" ]; then
    PLUGIN_BINARY_NAME="${PLUGIN_BINARY_NAME}.exe"
fi

echo "  Building for: ${GOOS}/${GOARCH}"
echo "  Output: $PLUGIN_BINARY_NAME"

# Build the backend with platform-specific name
if ! CGO_ENABLED=1 GOOS=$GOOS GOARCH=$GOARCH go build -o dist/$PLUGIN_BINARY_NAME ./cmd/backend; then
    echo -e "${RED}✗ Backend build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Backend built successfully"
echo ""

# Step 5: Download Chrome if missing
echo -e "${YELLOW}[5/7]${NC} Checking Chrome binary..."

# Function to download Chrome for Testing
download_chrome() {
    echo "  Chrome not found - downloading Chrome for Testing..."

    # Chrome for Testing version (stable)
    CHROME_VERSION="141.0.7390.76"
    CHROME_URL="https://storage.googleapis.com/chrome-for-testing-public/${CHROME_VERSION}/linux64/chrome-linux64.zip"

    echo "  Downloading from: $CHROME_URL"

    # Download Chrome
    if ! wget -q --show-progress -O chrome-linux64.zip "$CHROME_URL"; then
        echo -e "${RED}✗ Failed to download Chrome${NC}"
        echo "  Please download manually from: https://googlechromelabs.github.io/chrome-for-testing/"
        return 1
    fi

    echo "  Extracting Chrome..."
    if ! unzip -q chrome-linux64.zip; then
        echo -e "${RED}✗ Failed to extract Chrome${NC}"
        rm -f chrome-linux64.zip
        return 1
    fi

    # Make executables
    chmod +x chrome-linux64/chrome 2>/dev/null || true
    chmod +x chrome-linux64/chrome_crashpad_handler 2>/dev/null || true
    chmod +x chrome-linux64/chrome_sandbox 2>/dev/null || true

    # Cleanup
    rm chrome-linux64.zip

    echo -e "${GREEN}✓${NC} Chrome downloaded successfully ($(du -sh chrome-linux64 | cut -f1))"
    return 0
}

# Check if Chrome exists and is valid
if [ ! -d "chrome-linux64" ] || [ ! -f "chrome-linux64/chrome" ]; then
    download_chrome || echo "  Continuing without Chrome (must be installed separately)"
elif [ -z "$(ls -A chrome-linux64)" ]; then
    # Directory exists but is empty
    echo "  Chrome directory is empty - downloading..."
    rmdir chrome-linux64
    download_chrome || echo "  Continuing without Chrome (must be installed separately)"
else
    echo -e "${GREEN}✓${NC} Chrome binary found ($(du -sh chrome-linux64 | cut -f1))"
fi
echo ""

# Step 6: Package plugin
echo -e "${YELLOW}[6/7]${NC} Packaging plugin..."

# Verify all required files exist
if [ ! -f "dist/module.js" ]; then
    echo -e "${RED}✗ Frontend build output missing (dist/module.js)${NC}"
    exit 1
fi

if [ ! -f "dist/$PLUGIN_BINARY_NAME" ]; then
    echo -e "${RED}✗ Backend build output missing (dist/$PLUGIN_BINARY_NAME)${NC}"
    exit 1
fi

# Copy additional files
cp -r src/img dist/ 2>/dev/null || true
cp src/plugin.json dist/
cp README.md dist/ 2>/dev/null || true
cp check-chrome-deps.sh dist/ 2>/dev/null || true

# Copy Go manifest files and source files (required by Grafana catalog validator)
cp go.mod dist/
cp go.sum dist/

# Copy Go source files using rsync (preserves directory structure)
echo "  Copying Go source files..."
rsync -av --include='*.go' --include='*/' --exclude='*' pkg/ dist/pkg/ 2>/dev/null || true
rsync -av --include='*.go' --include='*/' --exclude='*' cmd/ dist/cmd/ 2>/dev/null || true

# Copy Chrome binary if it exists
if [ -d "chrome-linux64" ]; then
    echo "  Including Chrome binary from chrome-linux64/"
    cp -r chrome-linux64 dist/
    # Make Chrome binary executable
    if [ -f "dist/chrome-linux64/chrome" ]; then
        chmod +x dist/chrome-linux64/chrome
    fi
    # Make check script executable
    if [ -f "dist/check-chrome-deps.sh" ]; then
        chmod +x dist/check-chrome-deps.sh
    fi
else
    echo "  Note: chrome-linux64/ not found - Chrome must be installed separately"
fi

# Make plugin binary executable
chmod +x dist/$PLUGIN_BINARY_NAME

# Show package contents
echo "  Package contents:"
ls -lh dist/ | grep -v "^total" | awk '{printf "    %s %s\n", $9, $5}'

echo -e "${GREEN}✓${NC} Plugin packaged"
echo ""

# Step 7: Create archive
echo -e "${YELLOW}[7/7]${NC} Creating distribution archive..."

ARCHIVE_NAME="${PLUGIN_ID}-${PLUGIN_VERSION}-${GOOS}-${GOARCH}.zip"

cd dist
if ! zip -r ../$ARCHIVE_NAME . -q; then
    echo -e "${RED}✗ Failed to create archive${NC}"
    exit 1
fi
cd ..

ARCHIVE_SIZE=$(du -h "$ARCHIVE_NAME" | cut -f1)
echo -e "${GREEN}✓${NC} Archive created: ${ARCHIVE_NAME} (${ARCHIVE_SIZE})"
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ Build Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Plugin files:"
echo "  • dist/               - Plugin directory"
echo "  • $ARCHIVE_NAME       - Distribution archive"
echo ""
echo "Installation:"
echo "  1. Extract to Grafana plugins directory:"
echo "     unzip $ARCHIVE_NAME -d /var/lib/grafana/plugins/$PLUGIN_ID/"
echo ""
echo "  2. Configure Grafana to allow unsigned plugins:"
echo "     [plugins]"
echo "     allow_loading_unsigned_plugins = $PLUGIN_ID"
echo ""
echo "  3. Restart Grafana:"
echo "     sudo systemctl restart grafana-server"
echo ""
echo "  4. Install Chrome/Chromium for PDF rendering:"
if [ -d "chrome-linux64" ]; then
    echo "     ✓ Chrome binary included in archive (chrome-linux64/)"
    echo "     • No additional installation needed"
    echo "     • Chrome path will be auto-detected"
    echo ""
    echo "  5. Verify Chrome installation (optional):"
    echo "     cd /var/lib/grafana/plugins/scheduled-reports-app/"
    echo "     ./check-chrome-deps.sh"
else
    echo "     ⚠ Chrome binary NOT included in archive"
    echo "     • Option A: Download standalone Chrome:"
    echo "       wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb"
    echo "       Or use chrome-linux64 from Chrome for Testing"
    echo "     • Option B: System package:"
    echo "       apt-get install chromium-browser  # Debian/Ubuntu"
    echo "       yum install chromium              # RHEL/CentOS"
fi
echo ""
echo "  6. Configure plugin in Grafana UI:"
echo "     • Settings → Set Grafana URL: https://your-host:3000/dna"
echo "     • Settings → Chromium Path: (leave empty for auto-detect)"
echo "     • Settings → Enable: Skip TLS Verify (if using self-signed certificates)"
echo "     • Note: Essential Chrome flags are always enabled automatically"
echo ""
