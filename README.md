# Scheduled Reports for Grafana

<div align="center">
  <img src="src/img/logo.png" alt="Scheduled Reports Logo" width="200"/>

  **Automated dashboard reporting with PDF generation and email delivery for Grafana OSS**

  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
  [![Grafana](https://img.shields.io/badge/Grafana-11.6%2B-orange)](https://grafana.com)
</div>

## Overview

Scheduled Reports is a comprehensive Grafana app plugin that brings enterprise-grade automated dashboard reporting to Grafana OSS. Schedule recurring reports, render dashboards to PDF using Chromium, and deliver them via email - all managed through an intuitive UI.

## ✨ Key Features

### 📅 Flexible Scheduling
- **Fixed Schedules**: Daily (8:00 AM), Weekly (Monday 9:00 AM), Monthly (1st at 10:00 AM)
- **Custom Cron**: Full cron expression support with timezone awareness
- **Timezone Support**: Schedule reports in any timezone
- **Next Run Preview**: See upcoming 5 executions before saving
- **Manual Execution**: Trigger any report on-demand

### 📊 High-Fidelity Rendering
- **Chromium-Based**: Uses go-rod for pixel-perfect dashboard rendering
- **Direct PDF Generation**: Chrome DevTools Protocol for native PDF output
- **Full JavaScript Support**: Handles complex dashboards with animations and dynamic content
- **Configurable Quality**: Adjust viewport size, scale factor, and rendering delays
- **Per-Org Browser Reuse**: Efficient resource management with lazy initialization

### 📧 Email Delivery
- **SMTP Support**: Works with Gmail, SendGrid, corporate mail servers, or Grafana's SMTP
- **Template Variables**: Dynamic email content with `{{schedule.name}}`, `{{dashboard.title}}`, `{{timerange}}`
- **Multiple Recipients**: To, CC, BCC support with domain whitelisting
- **Attachment Management**: PDF attachments with size limits and fallback to download links
- **Test Function**: Verify SMTP configuration before going live

### 🔄 Complete Audit Trail
- **Run History**: Track every execution with status, duration, and errors
- **Artifact Storage**: Download generated PDFs at any time
- **Retention Policies**: Auto-delete old artifacts based on configurable retention days
- **Error Details**: Full error messages for troubleshooting

### ⚙️ Enterprise-Grade Configuration
- **Multi-tenancy**: Complete organization isolation for all data
- **Per-Org Settings**: SMTP, renderer, and limits configured independently
- **Service Account Auth**: Automatic authentication via Grafana's managed service accounts (IAM)
- **Rate Limiting**: Control max concurrent renders, recipients, and attachment sizes
- **Domain Whitelisting**: Restrict email recipients to approved domains

### 📖 Built-in Documentation
- Interactive documentation page within the plugin
- Context-sensitive help throughout the UI
- Comprehensive troubleshooting guides

## 📋 Prerequisites

- **Grafana**: Version 11.6.0 or higher (11.6+ recommended for managed service accounts)
- **Chromium/Chrome**: Required for PDF rendering
  - System package: `chromium-browser` or `google-chrome`
  - Standalone binary: Chrome for Testing (auto-downloaded in Docker)
  - Auto-detected from common paths if not configured
- **SMTP Server**: For email delivery (Gmail, SendGrid, etc.)
- **Node.js**: 22+ (for building from source)
- **Go**: 1.21+ (for building from source)

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Clone repository
git clone https://github.com/FulgerX2007/grafana-scheduled-reports-app.git
cd grafana-scheduled-reports-app

# Start Grafana with plugin
docker compose up -d

# Access Grafana
open http://localhost:3000
# Default credentials: admin/admin
```

### Option 2: Install from Release

```bash
# Download latest release
wget https://github.com/FulgerX2007/grafana-scheduled-reports-app/releases/latest/download/grafana-scheduled-reports-app-linux-amd64.zip

# Extract to Grafana plugins directory
unzip grafana-scheduled-reports-app-linux-amd64.zip -d /var/lib/grafana/plugins/

# Restart Grafana
systemctl restart grafana-server
```

### Enable Plugin

1. Navigate to **Administration → Plugins** in Grafana
2. Find **"Scheduled Reports"**
3. Click **Enable**
4. Go to **Apps → Scheduled Reports**

## 📖 Usage

### Creating Your First Schedule

1. **Navigate** to Apps → Scheduled Reports
2. Click **"New Schedule"**
3. **Configure Schedule**:
   - **Name**: Descriptive name (e.g., "Daily Sales Report")
   - **Dashboard**: Select from dropdown (auto-loads template variables)
   - **Time Range**: e.g., "Last 24 hours" (now-24h to now)
   - **Schedule**: Choose Daily/Weekly/Monthly or custom cron
   - **Timezone**: Select appropriate timezone
   - **Variables**: Modify dashboard variable values if needed
   - **Recipients**: Enter email addresses (comma-separated)
   - **Email**: Customize subject and body with template variables
4. Click **"Create"**

### Template Variables in Emails

Use these placeholders in email subject/body:

- `{{schedule.name}}` - Schedule name
- `{{dashboard.title}}` - Dashboard title
- `{{timerange}}` - Time range (e.g., "Last 24 hours")
- `{{run.started_at}}` - Execution start time

**Example Subject**:
```
{{schedule.name}} - {{dashboard.title}} ({{timerange}})
```

**Example Body**:
```
Hello,

Please find attached the {{dashboard.title}} report for {{timerange}}.

Generated at: {{run.started_at}}

Best regards,
Automated Reporting System
```

### Configuring Plugin Settings

Navigate to **Apps → Scheduled Reports → Settings** to configure:

#### SMTP Configuration
- **Use Grafana SMTP**: Toggle to use Grafana's built-in SMTP
- **Custom SMTP**: Configure host, port, username, password, and from address
- **TLS Settings**: Enable TLS and optionally skip verification for self-signed certificates
- **Test Button**: Send test email to verify configuration

#### Renderer Configuration
- **Chromium Path**: Path to Chrome binary (leave empty for auto-detection)
  - Auto-detect searches: `/usr/bin/chromium`, `/usr/bin/google-chrome`, `./chrome-linux64/chrome`
- **Timeout**: Maximum render time (default: 30000ms)
- **Delay**: Wait after page load for queries to complete (default: 5000ms)
- **Viewport**: Browser dimensions (default: 1920x1080)
- **Device Scale Factor**: Quality multiplier 1.0-4.0 (default: 2.0, higher = better quality)
- **Headless**: Run browser without GUI (recommended: enabled)
- **No Sandbox**: Required for Docker/containerized environments
- **Disable GPU**: Required for servers without display
- **Skip TLS Verify**: Skip certificate verification for self-signed Grafana certificates
- **Version Check**: Verify Chromium installation and display version

#### Limits and Quotas
- **Max Recipients**: Maximum email recipients per schedule (default: 100)
- **Max Attachment Size**: Maximum PDF size in MB (default: 10)
- **Max Concurrent Renders**: Simultaneous rendering limit (default: 5)
- **Retention Days**: Artifact retention period (default: 30)
- **Allowed Domains**: Whitelist for recipient email domains (empty = all allowed)

## 🏗️ Architecture

### Frontend (React + TypeScript)

```
src/
├── components/           # Reusable UI components
│   ├── AppConfig.tsx    # Main app configuration
│   ├── CronEditor.tsx   # Cron expression editor with preview
│   ├── DashboardPicker.tsx  # Dashboard selection dropdown
│   ├── RecipientsEditor.tsx  # Email recipients management
│   └── VariablesEditor.tsx   # Dashboard variables editor
├── pages/
│   ├── Schedules/       # Schedule list and edit pages
│   ├── RunHistory/      # Execution history viewer
│   ├── Settings/        # Plugin configuration
│   ├── Templates/       # Report templates (future feature)
│   └── Documentation/   # Built-in user guide
├── types/
│   └── types.ts         # TypeScript type definitions
└── plugin.json          # Plugin manifest
```

### Backend (Go)

```
pkg/
├── api/                 # HTTP API handlers
│   └── handlers.go      # REST endpoints for schedules, runs, settings
├── auth/                # Authentication helpers
├── cron/                # Scheduler and job execution
│   └── scheduler.go     # Cron scheduler with timezone support
├── mail/                # SMTP email sender
│   └── mailer.go        # Email delivery with template support
├── model/               # Data models
│   ├── models.go        # Schedule, Run, Settings, Template
│   └── validation.go    # Input validation
├── pdf/                 # PDF assembly (future: multi-page support)
│   └── pdf.go           # PDF manipulation utilities
├── render/              # Rendering backends
│   ├── interface.go     # Backend interface definition
│   └── chromium_renderer.go  # Chromium-based PDF renderer (go-rod)
└── store/               # Data persistence
    ├── store.go         # SQLite database operations
    └── writequeue.go    # Async write queue for performance
```

### Database Schema (SQLite)

**Tables**:
- `schedules`: Report configurations with cron expressions
- `runs`: Execution history with status and artifacts
- `settings`: Per-organization SMTP and renderer configuration
- `templates`: Report templates (future feature)

All tables include `org_id` for multi-tenancy and `created_at`/`updated_at` timestamps.

## 🔧 Development

### Building from Source

```bash
# Install dependencies
npm install
go mod download

# Build frontend
npm run build

# Build backend
go build -o dist/gpx_reporting ./cmd/backend

# Or use Makefile
make install  # Install dependencies
make build    # Build both frontend and backend
```

### Development Mode

```bash
# Terminal 1: Watch frontend changes
npm run dev

# Terminal 2: Run Grafana with plugin
docker compose up grafana
```

### Running Tests

```bash
# Frontend tests
npm test

# Backend tests
go test ./...

# Integration tests (requires Chromium)
go test -tags=integration ./pkg/render/

# End-to-end tests
See E2E_TESTING.md for Playwright setup
```

## 🔌 API Reference

Base path: `/api/plugins/sheduled-reports-app/resources/api`

### Schedules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/schedules` | List all schedules for current org |
| POST | `/schedules` | Create new schedule |
| GET | `/schedules/:id` | Get schedule by ID |
| PUT | `/schedules/:id` | Update schedule |
| DELETE | `/schedules/:id` | Delete schedule |
| POST | `/schedules/:id/run` | Trigger immediate execution |
| GET | `/schedules/:id/runs` | Get run history for schedule |

### Runs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/runs/:id` | Get run details |
| GET | `/runs/:id/artifact` | Download PDF artifact |

### Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/settings` | Get org settings |
| POST | `/settings` | Update org settings |
| POST | `/smtp/test` | Test SMTP configuration |
| GET | `/chromium/check-version` | Verify Chromium installation |
| GET | `/service-account/status` | Check service account status |
| POST | `/service-account/test-token` | Test service account token |

### Example: Create Schedule

```bash
curl -X POST "http://localhost:3000/api/plugins/sheduled-reports-app/resources/api/schedules" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Sales Report",
    "dashboard_uid": "dashboard-uid",
    "range_from": "now-24h",
    "range_to": "now",
    "interval_type": "daily",
    "timezone": "America/New_York",
    "recipients": {
      "to": ["team@example.com"]
    },
    "email_subject": "Daily Sales Report - {{dashboard.title}}",
    "email_body": "Please find attached the daily sales report.",
    "enabled": true
  }'
```

## 🐛 Troubleshooting

### Rendering Fails

**Symptoms**: Blank PDFs, rendering errors, timeouts

**Solutions**:
1. **Verify Chromium**: Go to Settings → Click "Check Chromium Version"
2. **Check Path**: Set explicit Chromium path in Settings
3. **Enable Flags**: Ensure "No Sandbox" and "Disable GPU" are enabled for servers
4. **Increase Timeouts**: Raise timeout/delay for slow dashboards
5. **Check Logs**: `docker logs grafana | grep -i render`

### PDF Shows Login Page

**Cause**: Service account authentication not working

**Solutions**:
1. Go to Settings → Check "Service Account Status"
2. For Grafana 11.6+: Restart Grafana to refresh token
3. For older versions: Manually create service account token

### Email Not Sending

**Solutions**:
1. Click "Send Test Email" in Settings to diagnose
2. For Gmail: Use App Password, not regular password
3. Check firewall allows outbound SMTP (port 587/465/25)
4. Enable "Skip TLS Verification" for self-signed certificates

### Schedule Not Running

**Solutions**:
1. Verify schedule is **Enabled**
2. Check **Next Run** time is in future
3. Validate cron expression syntax
4. Restart Grafana to reload scheduler: `systemctl restart grafana-server`

## 📚 Documentation

- **[QUICKSTART.md](./QUICKSTART.md)** - 5-minute setup guide
- **[SETUP_GUIDE.md](./SETUP_GUIDE.md)** - Detailed installation and configuration
- **[AUTHENTICATION.md](./AUTHENTICATION.md)** - Service account setup and troubleshooting
- **[E2E_TESTING.md](./E2E_TESTING.md)** - Playwright end-to-end testing guide
- **[BUILD.md](./BUILD.md)** - Building and packaging instructions
- **[SECURITY.md](./SECURITY.md)** - Security considerations and best practices
- **[CLAUDE.md](./CLAUDE.md)** - Developer guidance (for Claude Code)
- **[GRAFANA_CATALOG_SUBMISSION.md](./GRAFANA_CATALOG_SUBMISSION.md)** - Publishing to Grafana catalog

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Commit with semantic commit format (`feat:`, `fix:`, `docs:`, etc.)
5. Push to your fork and submit a pull request

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Grafana Plugin SDK](https://grafana.com/developers/plugin-tools)
- Rendering powered by [go-rod](https://github.com/go-rod/rod)
- Email delivery via [gomail](https://gopkg.in/gomail.v2)
- Cron parsing with [robfig/cron](https://github.com/robfig/cron)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/FulgerX2007/grafana-scheduled-reports-app/issues)
- **Discussions**: [GitHub Discussions](https://github.com/FulgerX2007/grafana-scheduled-reports-app/discussions)
- **Documentation**: [Built-in docs](http://localhost:3000/a/sheduled-reports-app/documentation) (Apps → Scheduled Reports → Documentation)

---

<div align="center">
  Made with ❤️ by <a href="https://github.com/FulgerX2007">Andrian Iliev</a>
</div>
