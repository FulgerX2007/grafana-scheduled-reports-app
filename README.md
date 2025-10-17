<div align="center">
  <img src="src/img/logo.png" alt="Scheduled Reports Logo" width="200"/>

  # Scheduled Reports for Grafana

  A comprehensive Grafana app plugin for automated dashboard reporting with PDF generation and email delivery.
</div>

## Features

- 📅 **Scheduled Reports**: Create recurring reports with fixed schedules (daily 8:00 AM, weekly Monday 9:00 AM, monthly 1st at 10:00 AM) or custom cron expressions
- 📊 **High-Fidelity Rendering**: Uses Chromium/Chrome browser for pixel-perfect dashboard rendering with full JavaScript support
- 📧 **Email Delivery**: Send reports via SMTP with customizable subjects, bodies, and template variables
- 🔄 **Run History**: Complete audit trail with execution status, duration, errors, and downloadable artifacts
- ⚙️ **Flexible Configuration**: Per-organization SMTP, renderer settings, rate limits, and retention policies
- 🎨 **Template Variables**: Dynamic email content with dashboard metadata, time ranges, and execution details
- 🔒 **Multi-tenancy**: Complete organization isolation with automatic service account authentication
- 📖 **Built-in Documentation**: Interactive documentation page within the plugin

## Prerequisites

- **Grafana 10.0 or higher** (10.3+ recommended for managed service accounts)
- **Chromium/Chrome browser** for PDF rendering (can be standalone binary or system package)
- **SMTP server** for email delivery (Gmail, SendGrid, corporate mail server, etc.)
- Go 1.21+ (for building from source)
- Node.js 22+ (for building from source)

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/yourusername/scheduled-reports-app.git
cd scheduled-reports-app
make install
make build
```

### 2. Configure Environment Variables (Optional)

Copy the example environment file and configure SMTP settings if needed:

```bash
cp .env.example .env
```

Edit `.env` and set (optional):
- `GF_SMTP_HOST`, `GF_SMTP_USER`, `GF_SMTP_PASSWORD`: Your SMTP settings
- `GF_SMTP_FROM_ADDRESS`, `GF_SMTP_FROM_NAME`: Email sender details

### 3. Start with Docker Compose

```bash
docker compose up -d
```

This will start:
- Grafana on http://localhost:3000 (default credentials: admin/admin)
- Plugin automatically loaded with Chromium browser included

### 4. Enable Plugin

In Grafana:
1. Go to **Administration → Plugins**
2. Find **"Scheduled Reports"** in the list
3. Click the plugin, then click **"Enable"**

**Authentication**: The plugin uses Grafana's managed service accounts (via IAM permissions in plugin.json). For Grafana 10.3+, authentication is automatic. For earlier versions, you may need to manually create a service account token in Settings.

## Development

### Building

```bash
# Install dependencies
make install

# Build both frontend and backend
make build

# Build only backend
make build-backend

# Build only frontend
make build-frontend
```

### Running in Development Mode

```bash
# Start with file watching
make dev
```

### Testing

```bash
# Run all tests
make test

# Run Go tests only
go test -v ./...

# Run frontend tests only
npm test
```

## Configuration

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Grafana Configuration
GF_GRAFANA_URL=http://localhost:3000

# Plugin Data Path (where SQLite DB and artifacts are stored)
GF_PLUGIN_APP_DATA_PATH=./data

# SMTP Configuration (optional if using Grafana's SMTP)
GF_SMTP_HOST=smtp.gmail.com:587
GF_SMTP_USER=your-email@gmail.com
GF_SMTP_PASSWORD=your-app-password
GF_SMTP_FROM_ADDRESS=noreply@example.com
GF_SMTP_FROM_NAME=Grafana Reports
```

### Plugin Settings

After enabling the plugin, go to **Apps → Scheduled Reports → Settings** to configure:

#### SMTP Configuration
Configure email delivery with either Grafana's built-in SMTP or custom settings:
- **Use Grafana SMTP**: Toggle to use Grafana's SMTP configuration (from environment variables)
- **Custom SMTP**: Host, port, username, password, From address
- **TLS Settings**: Enable TLS, skip TLS verification (for self-signed certificates)
- **Test Button**: Send test email to verify SMTP configuration

#### Renderer Configuration
Configure the Chromium rendering engine for PDF generation:
- **Backend**: Chromium (default and only supported renderer)
- **Chromium Path**: Path to Chrome/Chromium binary (e.g., `./chrome-linux64/chrome`)
  - Leave empty for auto-detection (searches common system paths)
- **Timeout**: Maximum render time in milliseconds (default: 30000)
- **Delay**: Wait time after page load for queries to complete (default: 5000)
- **Viewport**: Browser window dimensions (default: 1920x1080)
- **Device Scale Factor**: Rendering quality multiplier (1.0-4.0, default: 2.0)
- **Headless Mode**: Run browser without GUI (recommended: enabled)
- **No Sandbox**: Disable Chrome sandbox (required for Docker/root)
- **Disable GPU**: Disable GPU acceleration (recommended for servers)
- **Skip TLS Verify**: Skip certificate verification (for self-signed certificates)
- **Version Check Button**: Verify Chromium installation and display version

#### Limits and Quotas
Control plugin resource usage per organization:
- **Max Recipients**: Maximum email recipients per schedule (default: 100)
- **Max Attachment Size**: Maximum PDF file size in MB (default: 10)
- **Max Concurrent Renders**: Simultaneous rendering limit (default: 5)
- **Retention Days**: Keep artifacts for N days, then auto-delete (default: 30)

## Rendering Backend

The plugin uses **Chromium/Chrome** for direct PDF generation with full JavaScript support.

### Features

- **Direct PDF generation**: Chrome's native print-to-PDF functionality
- **Full JavaScript support**: Modern Chromium engine handles complex dashboards
- **High rendering fidelity**: Pixel-perfect rendering of Grafana dashboards
- **Per-organization browser reuse**: Efficient resource management
- **Configurable quality**: Adjust viewport size and scale factor

### Configuration

```json
{
  "chromium_path": "./chrome-linux64/chrome",  // Path to Chrome binary (optional, auto-detected)
  "headless": true,                             // Run in headless mode (recommended)
  "no_sandbox": true,                           // Required for Docker/containerized environments
  "disable_gpu": true,                          // Recommended for servers without GPU
  "viewport_width": 1920,                       // Browser viewport width
  "viewport_height": 1080,                      // Browser viewport height
  "device_scale_factor": 2.0,                   // Higher = better quality (1.0-4.0)
  "timeout_ms": 30000,                          // Maximum rendering time
  "delay_ms": 5000,                             // Wait time for queries to complete
  "skip_tls_verify": true                       // Skip TLS certificate verification (for self-signed certs)
}
```

### Installing Chrome/Chromium

**Option 1: Standalone Chrome** (recommended for servers)
```bash
# Download Chrome for Testing (includes ChromeDriver)
wget https://storage.googleapis.com/chrome-for-testing-public/.../chrome-linux64.zip
unzip chrome-linux64.zip
# Configure plugin to use: ./chrome-linux64/chrome
```

**Option 2: System Package**
```bash
# Debian/Ubuntu
apt-get install chromium-browser

# RHEL/CentOS
yum install chromium

# macOS
brew install chromium
```

**Option 3: Download Chrome Stable**
```bash
# Debian/Ubuntu
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
dpkg -i google-chrome-stable_current_amd64.deb
```

## Usage

### Creating a Schedule

1. Navigate to **Apps → Scheduled Reports**
2. Click **"New Schedule"** button
3. Configure the schedule:

   **Basic Settings**:
   - **Name**: Descriptive name (e.g., "Daily Sales Dashboard")
   - **Dashboard**: Select dashboard from dropdown (auto-loads variables)
   - **Format**: PDF or HTML (PDF recommended for most use cases)
   - **Enabled**: Toggle to enable/disable the schedule

   **Time Range**:
   - **From**: Start time (e.g., "now-24h", "now-7d", "2024-01-01")
   - **To**: End time (e.g., "now")

   **Schedule Interval**:
   - **Daily**: Runs every day at 8:00 AM
   - **Weekly**: Runs every Monday at 9:00 AM
   - **Monthly**: Runs on the 1st of each month at 10:00 AM
   - **Custom (Cron)**: Use cron expression for custom timing

   **Dashboard Variables**:
   - Automatically populated from selected dashboard
   - Modify values as needed (e.g., set `env=production`, `region=us-east`)

   **Email Recipients**:
   - **To**: Primary recipients (comma-separated)
   - **CC**: Carbon copy (optional)
   - **BCC**: Blind carbon copy (optional)
   - **Subject**: Email subject line (supports template variables)
   - **Body**: Email message (supports template variables)

4. Click **"Create"** to save the schedule

### Template Variables

Use these placeholders in email subject and body:

- `{{schedule.name}}` - Schedule name
- `{{dashboard.title}}` - Dashboard title
- `{{timerange}}` - Time range
- `{{run.started_at}}` - Execution start time

### Running Reports Manually

- Click the ▶️ icon next to any schedule to run it immediately
- View execution status in the Run History

### Viewing Run History

- Click the 🕐 icon next to any schedule
- See all executions with status, duration, and errors
- Download generated PDFs/HTMLs

## Architecture

### Frontend (React + TypeScript)
- `src/pages/` - Page components (Schedules, Settings, etc.)
- `src/components/` - Reusable UI components
- `src/types/` - TypeScript type definitions

### Backend (Go)
- `pkg/api/` - HTTP API handlers
- `pkg/cron/` - Scheduler and job execution
- `pkg/render/` - Chromium rendering system
  - `interface.go` - Backend interface definition
  - `chromium_renderer.go` - Chromium/Chrome PDF renderer (go-rod)
- `pkg/mail/` - SMTP email sender
- `pkg/store/` - SQLite database operations
- `pkg/model/` - Data models

### Data Storage
- SQLite database in `$GF_PLUGIN_APP_DATA_PATH/reporting.db`
- Artifacts in `$GF_PLUGIN_APP_DATA_PATH/artifacts/`

## Troubleshooting

### Rendering Fails

**Problem**: Dashboard rendering fails or produces blank PDFs

**Diagnostic Steps**:
1. **Test Chromium Installation**:
   ```bash
   # Test installed Chrome/Chromium
   chromium --version
   google-chrome --version
   ./chrome-linux64/chrome --version
   ```

2. **Check Chromium Version in Settings**:
   - Go to Settings page and click **"Check Chromium Version"**
   - Should display version number (e.g., "131.0.6778.204")

3. **Review Plugin Logs**:
   ```bash
   # Docker
   docker logs grafana | grep -i "chromium\|render\|pdf"

   # System installation
   tail -f /var/log/grafana/grafana.log | grep -i "chromium\|render"
   ```

**Common Solutions**:
- **Configure Chromium Path**: Set explicit path in Settings
  - Standalone: `./chrome-linux64/chrome` (relative to plugin directory)
  - System: `/usr/bin/chromium` or `/usr/bin/google-chrome`
  - Auto-detect: Leave empty (searches standard locations)

- **Enable Required Flags** (in Settings):
  - ✅ **No Sandbox**: Required for Docker and root environments
  - ✅ **Disable GPU**: Required for servers without display
  - ✅ **Headless**: Always enabled for server rendering
  - ✅ **Skip TLS Verify**: For self-signed Grafana certificates

- **Adjust Timeouts**:
  - Increase **Timeout** (default: 30s) for slow dashboards
  - Increase **Delay** (default: 5s) for dashboards with slow queries

- **Check Permissions**:
  - Ensure service account has access to dashboard
  - Verify dashboard is in accessible folder
  - Check organization ID matches

### PDF Shows Login Page

**Problem**: Generated PDF contains Grafana login page instead of dashboard

**Cause**: Authentication token not working or missing

**Solutions**:
1. **For Grafana 10.3+**: Managed service accounts should work automatically
   - Restart Grafana to refresh token: `systemctl restart grafana-server`
   - Check Settings page for service account status

2. **For Earlier Versions**: Create manual service account token
   - Go to **Administration → Service accounts**
   - Create new service account with **Admin** or **Viewer** role
   - Generate token and paste into **Settings → Service Account Token**

3. **Verify Token Permissions**:
   - Service account needs read access to dashboards and datasources
   - Check IAM permissions in plugin.json are applied

### Email Not Sending

**Problem**: Reports generate successfully but emails don't arrive

**Diagnostic Steps**:
1. **Test SMTP Settings**:
   - Click **"Send Test Email"** button in Settings
   - Check for error messages

2. **Review Logs**:
   ```bash
   tail -f /var/log/grafana/grafana.log | grep -i "smtp\|mail\|email"
   ```

**Common Solutions**:
- **Verify SMTP Configuration**:
  - Test connection: `telnet smtp.gmail.com 587`
  - For Gmail: Use App Password, not regular password
  - Check firewall allows outbound SMTP (port 587/465/25)

- **Check Recipient Addresses**:
  - Ensure email addresses are valid
  - Check spam/junk folders
  - Verify no typos in addresses

- **Toggle TLS Settings**:
  - Try with TLS enabled and disabled
  - Enable "Skip TLS Verification" for self-signed certificates

### Schedule Not Running

**Problem**: Schedules are created but never execute

**Diagnostic Steps**:
1. **Check Schedule Status**:
   - Verify schedule is **Enabled**
   - Check **Next Run** time is in the future
   - Ensure cron expression is valid (use preview)

2. **Review Scheduler Logs**:
   ```bash
   tail -f /var/log/grafana/grafana.log | grep -i "scheduler\|cron"
   ```

**Common Solutions**:
- **Restart Grafana**: Scheduler starts on plugin load
  ```bash
  systemctl restart grafana-server
  # or
  docker-compose restart grafana
  ```

- **Verify Cron Expression**: Use online cron validators
  - Daily 8AM: `0 8 * * *`
  - Weekly Mon 9AM: `0 9 * * 1`
  - Monthly 1st 10AM: `0 10 1 * *`

- **Check Database**: Ensure SQLite database is writable
  ```bash
  ls -la $GF_PLUGIN_APP_DATA_PATH/reporting.db
  ```

### High Memory Usage

**Problem**: Grafana consuming excessive memory

**Cause**: Browser instances not being released or too many concurrent renders

**Solutions**:
- **Reduce Concurrent Renders**: Lower in Settings (default: 5)
- **Decrease Retention Days**: Clean up old artifacts sooner
- **Monitor Browser Processes**:
  ```bash
  ps aux | grep chrome
  ```
- **Restart Grafana Periodically**: Release accumulated resources

## Production Deployment

### Building for Production

```bash
make build
make sign  # If you have a plugin signing key
make dist  # Creates distribution zip
```

### Installation from Release

Download the appropriate release archive for your platform from GitHub releases and extract it:

```bash
unzip scheduled-reports-app-X.Y.Z.linux-amd64.zip -d /var/lib/grafana/plugins/
systemctl restart grafana-server
```

Configure Grafana to allow unsigned plugins:
```ini
[plugins]
allow_loading_unsigned_plugins = scheduled-reports-app
```

### Local Development Installation

For local development, you need to ensure only the correct binary is present:

```bash
# Build the plugin
make build

# Copy to Grafana plugins directory (ensure gpx_reporting exists, not gpx_reporting_linux_amd64)
mkdir -p /var/lib/grafana/plugins/scheduled-reports-app
rsync -av --exclude='gpx_reporting_*' dist/ /var/lib/grafana/plugins/scheduled-reports-app/

# Restart Grafana
systemctl restart grafana-server
```

### Recommended Settings

- Enable managed service accounts feature in Grafana 10.3+
- Configure rate limits to prevent abuse
- Set up artifact retention to manage disk space
- Use TLS for SMTP connections
- Monitor plugin logs and metrics

## API Reference

### Schedules

```bash
# List schedules
GET /api/plugins/scheduled-reports-app/resources/api/schedules

# Create schedule
POST /api/plugins/scheduled-reports-app/resources/api/schedules

# Get schedule
GET /api/plugins/scheduled-reports-app/resources/api/schedules/{id}

# Update schedule
PUT /api/plugins/scheduled-reports-app/resources/api/schedules/{id}

# Delete schedule
DELETE /api/plugins/scheduled-reports-app/resources/api/schedules/{id}

# Run now
POST /api/plugins/scheduled-reports-app/resources/api/schedules/{id}/run

# Get runs
GET /api/plugins/scheduled-reports-app/resources/api/schedules/{id}/runs
```

### Settings

```bash
# Get settings
GET /api/plugins/scheduled-reports-app/resources/api/settings

# Update settings
POST /api/plugins/scheduled-reports-app/resources/api/settings
```

### Artifacts

```bash
# Download artifact
GET /api/plugins/scheduled-reports-app/resources/api/runs/{id}/artifact
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Apache 2.0

## Support

- **Documentation**:
  - Built-in: **Apps → Scheduled Reports → Documentation** (comprehensive user guide)
  - Developer Guide: See [CLAUDE.md](./CLAUDE.md) for development guidance
  - Quick Start: See [QUICKSTART.md](./QUICKSTART.md) for 5-minute setup
  - Setup Guide: See [SETUP_GUIDE.md](./SETUP_GUIDE.md) for detailed configuration
  - Authentication: See [AUTHENTICATION.md](./AUTHENTICATION.md) for service account setup and troubleshooting
- **Issues**: Report bugs and feature requests at [GitHub Issues](https://github.com/FulgerX2007/grafana-sheduled-reports/issues)
- **Repository**: [https://github.com/FulgerX2007/grafana-sheduled-reports](https://github.com/FulgerX2007/grafana-sheduled-reports)
