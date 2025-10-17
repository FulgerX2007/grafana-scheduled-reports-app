#!/bin/bash
# Chrome Dependency Checker
# This script helps diagnose Chrome rendering issues

echo "=========================================="
echo "  Chrome Dependency Checker"
echo "=========================================="
echo ""

# Check if Chrome binary exists
CHROME_PATH="${1:-./chrome-linux64/chrome}"
echo "[1] Checking Chrome binary..."
if [ -f "$CHROME_PATH" ]; then
    echo "    ✓ Chrome binary found: $CHROME_PATH"
    if [ -x "$CHROME_PATH" ]; then
        echo "    ✓ Chrome binary is executable"
    else
        echo "    ✗ Chrome binary is NOT executable"
        echo "    Fix: chmod +x $CHROME_PATH"
    fi
else
    echo "    ✗ Chrome binary not found at: $CHROME_PATH"
    echo "    Please specify correct path: $0 /path/to/chrome"
    exit 1
fi
echo ""

# Check Chrome version
echo "[2] Checking Chrome version..."
VERSION=$($CHROME_PATH --version 2>&1)
if [ $? -eq 0 ]; then
    echo "    ✓ $VERSION"
else
    echo "    ✗ Failed to get Chrome version"
    echo "    Error: $VERSION"
fi
echo ""

# Check required shared libraries
echo "[3] Checking required shared libraries..."
if command -v ldd >/dev/null 2>&1; then
    MISSING=$(ldd "$CHROME_PATH" 2>&1 | grep "not found")
    if [ -z "$MISSING" ]; then
        echo "    ✓ All shared libraries found"
    else
        echo "    ✗ Missing shared libraries:"
        echo "$MISSING" | sed 's/^/      /'
        echo ""
        echo "    Install missing libraries:"
        echo "    Ubuntu/Debian: apt-get install -y libglib2.0-0 libnss3 libxss1 libasound2 libxtst6 libgtk-3-0"
        echo "    RHEL/CentOS:   yum install -y glib2 nss libXScrnSaver alsa-lib libXtst gtk3"
    fi
else
    echo "    ⚠ ldd not available, skipping library check"
fi
echo ""

# Check /tmp space
echo "[4] Checking disk space..."
TMP_SPACE=$(df -h /tmp 2>/dev/null | tail -1 | awk '{print $4}')
if [ -n "$TMP_SPACE" ]; then
    echo "    ✓ /tmp available space: $TMP_SPACE"
else
    echo "    ⚠ Could not check /tmp space"
fi
echo ""

# Check if running in Docker
echo "[5] Checking environment..."
if [ -f /.dockerenv ]; then
    echo "    ⚠ Running in Docker container"
    echo "    Ensure these flags are set:"
    echo "      - --no-sandbox (required)"
    echo "      - --disable-dev-shm-usage (required if /dev/shm is limited)"
else
    echo "    ✓ Not running in Docker"
fi
echo ""

# Try to launch Chrome in test mode
echo "[6] Testing Chrome launch..."
echo "    Attempting to launch Chrome with plugin flags..."
CHROME_OUTPUT=$($CHROME_PATH --headless=new --no-sandbox --disable-gpu --disable-dev-shm-usage --disable-crash-reporter --no-first-run --no-default-browser-check --dump-dom about:blank 2>&1)
if [ $? -eq 0 ]; then
    echo "    ✓ Chrome launched successfully"
else
    echo "    ✗ Chrome failed to launch"
    echo "    Error output:"
    echo "$CHROME_OUTPUT" | head -20 | sed 's/^/      /'
fi
echo ""

echo "=========================================="
echo "  Summary"
echo "=========================================="
echo ""
echo "If Chrome launch failed, common fixes:"
echo "  1. Install missing system libraries (see section 3)"
echo "  2. Ensure Chrome binary is executable: chmod +x $CHROME_PATH"
echo "  3. Increase /tmp disk space if low"
echo "  4. In Docker: add --security-opt seccomp=unconfined"
echo "  5. Check system logs: journalctl -xe"
echo ""
