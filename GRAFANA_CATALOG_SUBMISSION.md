# Grafana Catalog Submission Guide

This guide covers the complete process for submitting the Scheduled Reports plugin to the Grafana plugin catalog to obtain a **community signature** (removing the `rootUrls` restriction).

## Table of Contents

1. [Overview](#overview)
2. [Signature Types](#signature-types)
3. [Prerequisites](#prerequisites)
4. [Preparation Steps](#preparation-steps)
5. [Submission Process](#submission-process)
6. [After Submission](#after-submission)
7. [Common Issues](#common-issues)

## Overview

**Why Submit Publicly?**
- **No rootUrls restriction**: Plugin works on any Grafana instance
- **Trusted signature**: Users don't need to configure `allow_loading_unsigned_plugins`
- **Discoverability**: Listed in Grafana plugin catalog
- **Wider adoption**: Available to millions of Grafana users

**Current Status**: Private signature with `signatureType: "private"` (requires rootUrls)

**Target**: Community signature with `signatureType: "community"` (no rootUrls needed)

## Signature Types

| Type | Signed By | rootUrls Required? | Visibility | Review Required? |
|------|-----------|-------------------|------------|------------------|
| **private** | Your organization | ✅ Yes | Private | ❌ No |
| **community** | Grafana Labs | ❌ No | Public catalog | ✅ Yes |
| **commercial** | Grafana Labs | ❌ No | Public catalog | ✅ Yes (paid) |
| **grafana** | Grafana Labs | ❌ No | Built-in | N/A (internal) |

## Prerequisites

### 1. Grafana Cloud Account

Create a free account at: https://grafana.com/auth/sign-up

This is required to:
- Submit plugins for review
- Access plugin management dashboard
- Receive submission status updates

### 2. Public Repository

Your plugin source code must be in a **publicly accessible Git repository**.

**Current Issue**: Repository is on corporate GitLab (gitlab.tech.orange) which may not be public.

**Solution**: Mirror to public GitHub

```bash
# 1. Create new repository on GitHub
# Go to: https://github.com/new
# Name: grafana-scheduled-reports-app
# Visibility: Public

# 2. Add GitHub as remote
git remote add github https://github.com/FulgerX2007/grafana-scheduled-reports-app.git

# 3. Push all branches and tags
git push github --all
git push github --tags

# 4. Verify repository is publicly accessible
# Visit: https://github.com/FulgerX2007/grafana-scheduled-reports-app
```

### 3. Plugin Requirements

- ✅ No Angular dependencies (deprecated)
- ✅ Not a fork or derivative work
- ✅ Unique functionality (not duplicating existing plugins)
- ✅ Production-ready code quality
- ✅ Comprehensive documentation
- ✅ Valid plugin.json metadata

## Preparation Steps

### Step 1: Capture Screenshots (REQUIRED)

Grafana requires screenshots to showcase your plugin.

**What to capture:**
1. **Schedules List** - Main landing page showing list of schedules
2. **Create/Edit Schedule** - Form with all configuration options
3. **Settings Page** - Plugin configuration (SMTP, renderer, etc.)
4. **Run History** (Optional) - Execution history with status

**How to capture:**

```bash
# 1. Start Grafana with plugin
docker compose up -d

# 2. Open browser to http://localhost:3000
# 3. Navigate to Apps → Scheduled Reports
# 4. Take screenshots of each page
# 5. Save to src/img/screenshots/

# Recommended dimensions: 1920x1080 or 1280x720
# Format: PNG
# File names:
#   - schedules-list.png
#   - create-schedule.png
#   - settings.png
```

**Screenshot Tips:**
- Use a clean Grafana instance without clutter
- Show the plugin in action with sample data
- Ensure text is readable
- Avoid sensitive information (emails, domains, etc.)
- Use light theme for better visibility

### Step 2: Optimize Logo

**Current Issue**: Logo is 786KB (too large for fast loading)

**Target**: < 100KB

```bash
# Option 1: Using ImageMagick
convert src/img/logo.png -resize 512x512 -quality 85 src/img/logo-optimized.png
mv src/img/logo-optimized.png src/img/logo.png

# Option 2: Using online tools
# Upload to: https://tinypng.com or https://squoosh.app
# Download optimized version
# Replace src/img/logo.png

# Option 3: Using pngquant
pngquant --quality=65-80 src/img/logo.png --output src/img/logo.png --force
```

### Step 3: Update Documentation

**Update README.md**

Replace all occurrences of old repository URLs:

```bash
# Find and replace in README.md:
# FROM: https://github.com/FulgerX2007/grafana-scheduled-reports-app
# FROM: https://github.com/FulgerX2007/grafana-scheduled-reports
# TO:   https://github.com/FulgerX2007/grafana-scheduled-reports-app
```

**Ensure README includes:**
- Clear description of plugin functionality
- Installation instructions
- Configuration guide
- Usage examples
- Troubleshooting section
- Screenshots (embedded from src/img/screenshots/)
- License information

### Step 4: Create Sample Dashboard (Recommended)

Help reviewers test your plugin by providing a sample dashboard:

```bash
# Create provisioning directory structure
mkdir -p provisioning/dashboards

# Export a simple test dashboard from Grafana
# Save as: provisioning/dashboards/sample-dashboard.json
```

**Sample dashboard should:**
- Be simple and self-contained
- Use TestData datasource (built-in)
- Demonstrate typical use case
- Include template variables (to test variable support)

### Step 5: Update plugin.json

Verify all metadata is correct:

```json
{
  "id": "scheduled-reports-app",
  "name": "Scheduled Reports",
  "version": "1.0.0",  // Bump to stable version
  "info": {
    "description": "Clear, concise description",
    "author": {
      "name": "Your Name",
      "url": "https://github.com/FulgerX2007"
    },
    "keywords": ["reporting", "pdf", "email", "scheduler"],
    "logos": {
      "small": "img/logo.png",  // Must be < 100KB
      "large": "img/logo.png"
    },
    "screenshots": [  // Must have at least 1 screenshot
      {
        "name": "Schedules List",
        "path": "img/screenshots/schedules-list.png"
      }
    ],
    "links": [
      {
        "name": "GitHub Repository",
        "url": "https://github.com/FulgerX2007/grafana-scheduled-reports-app"
      }
    ]
  }
}
```

### Step 6: Run Plugin Validator

Validate your plugin before submission:

```bash
# Build plugin
npm run build
go build -o dist/gpx_reporting ./cmd/backend

# Run validator
npx @grafana/plugin-validator dist/

# Fix any errors or warnings reported
```

**Common validation issues:**
- Missing metadata fields
- Oversized images
- Invalid plugin structure
- Missing README
- No screenshots

### Step 7: Build Release Package

Create the plugin ZIP archive for submission:

```bash
# 1. Ensure clean build
npm run build
go build -o dist/gpx_reporting ./cmd/backend

# 2. Create ZIP (from project root)
cd dist
zip -r ../grafana-scheduled-reports-app-1.0.0.zip . \
  -x "*.map" \
  -x "gpx_reporting_*"  # Exclude platform-specific binaries if any
cd ..

# 3. Calculate SHA256 hash
sha256sum grafana-scheduled-reports-app-1.0.0.zip > plugin-sha256.txt

# 4. Upload to GitHub Release
# Go to: https://github.com/FulgerX2007/grafana-scheduled-reports-app/releases/new
# Tag: v1.0.0
# Upload: grafana-scheduled-reports-app-1.0.0.zip
```

## Submission Process

### Step 1: Create GitHub Release

1. Go to: https://github.com/FulgerX2007/grafana-scheduled-reports-app/releases/new
2. Tag version: `v1.0.0`
3. Release title: `Scheduled Reports v1.0.0`
4. Description: Include release notes and features
5. Upload the ZIP file
6. Publish release
7. Copy the direct download URL (e.g., `https://github.com/.../releases/download/v1.0.0/grafana-scheduled-reports-app-1.0.0.zip`)

### Step 2: Log in to Grafana Cloud

1. Visit: https://grafana.com
2. Click "Sign In" (top right)
3. Log in with your Grafana Cloud account
4. Ensure you're an **organization administrator**

### Step 3: Submit Plugin

1. Navigate to: **Org Settings → My Plugins**
   - Direct URL: https://grafana.com/orgs/[your-org]/plugins
2. Click **"Submit New Plugin"** button
3. Fill out the submission form:

**Form Fields:**

- **Plugin ZIP URL**: Direct download URL from GitHub release
  - Example: `https://github.com/FulgerX2007/grafana-scheduled-reports-app/releases/download/v1.0.0/grafana-scheduled-reports-app-1.0.0.zip`

- **Source Code Repository**: GitHub repository URL
  - Example: `https://github.com/FulgerX2007/grafana-scheduled-reports-app`

- **SHA256 Hash**: From `plugin-sha256.txt`

- **OS & Architecture**: Select `Linux` and `amd64` (or all platforms you support)

- **Testing Instructions**: Comprehensive guide for reviewers

**Example Testing Instructions:**

```markdown
## Prerequisites
- Grafana 11.6.0 or higher
- Chromium browser (auto-downloaded by plugin or use system package)
- SMTP server (or use Gmail with app password)

## Installation
1. Extract ZIP to Grafana plugins directory
2. Enable plugin in Grafana UI: Administration → Plugins → Scheduled Reports
3. Navigate to Apps → Scheduled Reports

## Testing Steps

### 1. Configure Plugin
- Go to Apps → Scheduled Reports → Settings
- Configure SMTP settings (or toggle "Use Grafana SMTP")
- Click "Send Test Email" to verify SMTP works
- Configure Chromium path if needed (or leave empty for auto-detect)
- Click "Check Chromium Version" to verify renderer

### 2. Create Schedule
- Click "New Schedule" button
- Select a dashboard (use any existing dashboard)
- Set time range: "Last 24 hours"
- Set schedule: "Daily" (or custom cron)
- Add recipient email
- Click "Create"

### 3. Run Report
- Click ▶️ (play) icon next to schedule
- Wait for execution to complete (~30 seconds)
- Check email inbox for PDF report
- Click 🕐 (history) icon to view run details
- Download PDF artifact from run history

### 4. Verify Multi-tenancy
- Switch organizations in Grafana
- Confirm schedules are isolated per organization

## Sample Dashboard
Use the TestData datasource with any dashboard, or import:
provisioning/dashboards/sample-dashboard.json

## Expected Behavior
- PDF contains rendered dashboard with correct time range
- Email is delivered with PDF attachment
- Run history shows "Success" status
- Service account authentication is automatic (Grafana 11.3+)

## Common Issues
- If rendering fails, ensure Chromium is installed: `chromium --version`
- If email fails, verify SMTP settings with test button
- If login page appears in PDF, check Settings → Service Account status
```

- **Provisioning Configuration** (Optional): If you have provisioning files
  - Upload `provisioning/` directory contents

4. Click **"Submit"** button

### Step 4: Wait for Review

**Review Timeline:**
- **Automated validation**: 5-15 minutes (checks plugin structure, metadata)
- **Manual review**: 1-4 weeks (security audit, code quality, functional testing)

**Review Process:**
1. **Automated checks** - Plugin validator runs automatically
2. **Security review** - Manual code inspection for vulnerabilities
3. **Functional testing** - Reviewer installs and tests plugin
4. **Approval or feedback** - You'll receive email notification

**You'll be notified via email with:**
- ✅ Approval: Plugin is signed and published
- ❌ Rejection: Specific issues to fix
- 💬 Feedback: Requested changes or questions

## After Submission

### If Approved

Congratulations! Your plugin is now **community signed**:

1. **Update your plugin.json**: Remove `rootUrls` signing from CI
2. **Manifest will show**:
   ```json
   {
     "signatureType": "community",
     "signedByOrg": "grafana",
     "signedByOrgName": "Grafana Labs"
   }
   ```
3. **Plugin works on any Grafana instance** - no rootUrls restriction!
4. **Listed in catalog**: https://grafana.com/grafana/plugins/scheduled-reports-app

**Update CI/CD:**
- Remove private signing step from `.gitlab-ci.yml`
- Grafana will sign new versions when you submit updates
- Keep building and releasing on GitHub

### If Rejected

Common rejection reasons and fixes:

**Reason: Code Quality Issues**
- Fix linting errors: `npm run lint -- --fix`
- Add tests: `npm test`
- Improve code organization

**Reason: Security Concerns**
- Remove hardcoded credentials
- Fix dependency vulnerabilities: `npm audit fix`
- Update to secure libraries

**Reason: Missing Documentation**
- Expand README with more details
- Add inline code comments
- Create user guide

**Reason: Duplicate Functionality**
- Highlight unique features
- Explain how it differs from similar plugins
- Consider if merging with existing plugin is better

**Reason: Angular Dependencies**
- Migrate to React
- Remove `@grafana/ui` Angular components
- Use modern Grafana SDK

**After Fixing Issues:**
1. Make required changes
2. Create new release (v1.0.1)
3. Re-submit with updated ZIP URL
4. Reference previous submission in notes

### Updating Published Plugin

To release new versions:

1. **Make changes** and test locally
2. **Update version** in `src/plugin.json` (e.g., `1.1.0`)
3. **Create tag**: `git tag v1.1.0 && git push github v1.1.0`
4. **Build and upload** new release to GitHub
5. **Submit update** via Grafana Cloud:
   - Go to: Org Settings → My Plugins → [Your Plugin]
   - Click "Submit Update"
   - Provide new ZIP URL and changelog
6. **Wait for approval** (faster than initial submission, typically 1-2 days)

## Common Issues

### Issue: Repository Not Public

**Error**: "Repository is not accessible"

**Solution**:
```bash
# Make GitHub repository public
# Go to: Settings → Danger Zone → Change visibility → Make public
```

### Issue: ZIP File Too Large

**Error**: "Archive exceeds size limit"

**Solution**:
```bash
# Exclude unnecessary files from ZIP
cd dist
zip -r ../plugin.zip . \
  -x "*.map" \
  -x "node_modules/*" \
  -x ".git/*" \
  -x "chrome-linux64/*"  # Chromium is too large
```

**Note**: For Chromium, document that users must install separately or provide download script.

### Issue: Invalid Plugin Structure

**Error**: "Plugin manifest not found"

**Solution**: Ensure `plugin.json` is at the root of the ZIP:
```
plugin.zip/
  ├── plugin.json      # Must be here
  ├── module.js
  ├── gpx_reporting
  └── img/
```

### Issue: SHA256 Mismatch

**Error**: "Hash does not match"

**Solution**: Recalculate hash of the exact ZIP file you uploaded:
```bash
sha256sum grafana-scheduled-reports-app-1.0.0.zip
```

### Issue: Validator Fails

**Error**: "Plugin validation failed"

**Solution**: Run validator locally first:
```bash
npx @grafana/plugin-validator dist/
# Fix all errors before submitting
```

### Issue: Screenshots Not Showing

**Error**: "Screenshots required"

**Solution**:
- Add at least 1 screenshot to `src/img/screenshots/`
- Update `plugin.json` with screenshot paths
- Rebuild: `npm run build`
- Screenshots must be in the ZIP

## Resources

- **Official Guide**: https://grafana.com/developers/plugin-tools/publish-a-plugin/publish-a-plugin
- **Plugin Validator**: https://github.com/grafana/plugin-validator
- **Plugin Examples**: https://github.com/grafana/grafana-plugin-examples
- **Community Forum**: https://community.grafana.com/c/plugin-development
- **Plugin SDK Docs**: https://grafana.com/developers/plugin-tools

## Checklist

Before submitting, verify:

- [ ] Plugin builds without errors: `npm run build && go build ./cmd/backend`
- [ ] Plugin validator passes: `npx @grafana/plugin-validator dist/`
- [ ] Public GitHub repository exists and is accessible
- [ ] All screenshots are captured (at least 1, recommended 3+)
- [ ] Logo is optimized (< 100KB)
- [ ] README is comprehensive and up-to-date
- [ ] plugin.json metadata is complete and accurate
- [ ] Version is bumped to stable (1.0.0)
- [ ] GitHub release is created with ZIP file
- [ ] SHA256 hash is calculated
- [ ] Testing instructions are detailed
- [ ] Grafana Cloud account is set up
- [ ] You've tested the plugin in Grafana 11.6.0+

Good luck with your submission! 🚀
