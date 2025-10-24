# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-10-24

### Added
- **Flexible Scheduling System**
  - Fixed schedules: Daily (8:00 AM), Weekly (Monday 9:00 AM), Monthly (1st at 10:00 AM)
  - Custom cron expression support with timezone awareness
  - Next run preview showing upcoming 5 executions
  - Manual execution capability for immediate report generation

- **Enhanced Chromium Rendering Backend**
  - Progressive scrolling to trigger all lazy-loaded panels
  - Automatic waiting for network idle and loading indicators
  - Full content capture with dynamic dimension calculation
  - Per-organization browser reuse for optimal performance
  - Support for complex dashboards with animations and dynamic content
  - Configurable viewport size, scale factor, and rendering delays

- **Experimental Playwright Rendering Backend**
  - Alternative rendering engine for Ubuntu/Debian environments
  - Context-based authentication
  - Similar lazy-loading and scroll handling capabilities

- **Email Delivery System**
  - SMTP support with Gmail, SendGrid, and corporate mail servers
  - Template variables for dynamic content (schedule name, dashboard title, timerange)
  - Multiple recipients support (To, CC, BCC)
  - Domain whitelisting for security
  - PDF attachment management with size limits
  - Fallback to download links for large files
  - Test function to verify SMTP configuration

- **Complete Audit Trail**
  - Run history tracking with status, duration, and errors
  - Artifact storage for generated PDFs
  - Configurable retention policies
  - Detailed error messages for troubleshooting

- **Enterprise-Grade Configuration**
  - Multi-tenancy with complete organization isolation
  - Per-organization settings for SMTP, renderer, and limits
  - Service account authentication via Grafana managed service accounts
  - Automatic service account creation with Admin role
  - Rate limiting controls
  - Domain whitelisting for recipient emails

- **Built-in Documentation**
  - Interactive documentation page within the plugin
  - Context-sensitive help throughout the UI
  - Comprehensive troubleshooting guides

- **User Interface**
  - Schedule management with create, edit, delete operations
  - Dashboard picker with template variables auto-loading
  - Cron expression editor with presets
  - Email template editor with variable placeholders
  - Run history viewer with artifact download
  - Settings page for SMTP and renderer configuration
  - Service account status display and management

### Technical Details
- Grafana version requirement: 11.6.0 or higher
- Backend: Go 1.21+ with Grafana Plugin SDK
- Frontend: React 18 + TypeScript with Grafana UI components
- Database: SQLite for schedules, runs, and settings
- Rendering: Chromium (go-rod) and Playwright backends
- Email: SMTP with gomail library
- Scheduler: Cron parsing with robfig/cron library

### Security
- Service account-based authentication
- Per-organization data isolation
- Secure storage for SMTP credentials
- Domain whitelisting for email recipients
- TLS support with optional certificate verification skip

[1.0.0]: https://github.com/FulgerX2007/grafana-scheduled-reports-app/releases/tag/v1.0.0
