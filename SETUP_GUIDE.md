# Setup Guide - Grafana Scheduled Reports Plugin

## Quick Start (3 Steps)

### Step 1: Enable Feature Toggle

Add to `grafana.ini`:
```ini
[feature_toggles]
enable = externalServiceAccounts
```

**Or** set environment variable:
```bash
export GF_FEATURE_TOGGLES_ENABLE=externalServiceAccounts
```

### Step 2: Restart Grafana

```bash
sudo systemctl restart grafana-server
```

### Step 3: Verify Service Account

1. Open Grafana: Apps → Scheduled Reports → Settings
2. Check "Service Account Authentication" section
3. Should show: **✓ Service Account: Active**

**Done!** Your plugin is now ready to render dashboards.

**Note**: The managed service account will be named `extsvc-scheduled-reports-app` (using `extsvc-` prefix). You can find it in Grafana Admin → Configuration → Service Accounts.

---

## Detailed Setup

### Requirements

- ✅ **Grafana 10.3 or later**
- ✅ **Feature toggle**: `externalServiceAccounts` enabled
- ✅ **Plugin installed** in Grafana plugins directory

### Installation

1. **Copy plugin** to Grafana plugins directory:
   ```bash
   cp -r dist/ /var/lib/grafana/plugins/scheduled-reports-app/
   ```

2. **Enable feature toggle** in `grafana.ini`:
   ```ini
   [feature_toggles]
   enable = externalServiceAccounts
   ```

3. **Restart Grafana**:
   ```bash
   sudo systemctl restart grafana-server
   ```

4. **Verify plugin loaded**:
   ```bash
   tail -f /var/log/grafana/grafana.log | grep "scheduled-reports"
   ```

### Configuration

#### SMTP Settings (For Email Delivery)

1. Go to: **Apps → Scheduled Reports → Settings**
2. Configure **SMTP Configuration** section:
   - **Host**: smtp.gmail.com (or your SMTP server)
   - **Port**: 587 (or 465 for SSL)
   - **Username**: your-email@gmail.com
   - **Password**: your-app-password
   - **From Address**: noreply@example.com
   - **Use TLS**: ✓ (recommended)

#### Renderer Settings

1. Configure **Grafana URL** (Settings page):
   - Example: `https://127.0.0.1:3000` or `https://grafana.example.com`
   - Include subpath if you have one: `https://example.com/grafana`

2. Configure **Chromium Path** (if not auto-detected):
   - Download Chrome for Testing: https://googlechromelabs.github.io/chrome-for-testing/
   - Extract to plugin directory: `./chrome-linux64/chrome`
   - Or use system Chrome: `/usr/bin/google-chrome`

3. Adjust **Rendering Settings** if needed:
   - **Timeout**: 60000ms (increase for slow dashboards)
   - **Delay**: 5000ms (wait time after page load)
   - **Viewport**: 1920x1080 (dashboard resolution)
   - **Scale Factor**: 2.0 (higher = better quality)

### Verification

#### 1. Check Service Account Status

Go to: **Apps → Scheduled Reports → Settings**

**Expected**:
```
✓ Service Account: Active
Token Status: Configured (234 characters)
Grafana automatically manages this service account based on plugin.json IAM permissions
```

#### 2. Test Schedule Run

1. Go to: **Apps → Scheduled Reports**
2. Click: **Create Schedule**
3. Select a dashboard
4. Set schedule to run now
5. Click: **Run Now**
6. Check: **Run History** for successful execution
7. Download: **Artifact** to verify PDF generated correctly

#### 3. Check Logs

Monitor Grafana logs during test run:
```bash
tail -f /var/log/grafana/grafana.log | grep -E "(DEBUG:|ERROR:)"
```

**Expected log output**:
```
DEBUG: Using managed service account token from Grafana SDK (length: 234)
DEBUG: Dashboard URL: https://127.0.0.1:3000/d/abc123?from=...
DEBUG: [Request #1] Injected Authorization Bearer token for: https://...
DEBUG: Page title: My Dashboard - Grafana
DEBUG: PDF generated successfully (12345 bytes)
```

### Troubleshooting

#### Service Account Not Active

**Symptoms**:
- Settings shows "✗ Service Account: Error" or "⚠ Service Account: Not Configured"

**Solutions**:

1. **Check Grafana version**:
   ```bash
   grafana-server -v
   ```
   Must be 10.3 or later

2. **Enable feature toggle**:
   ```ini
   # grafana.ini
   [feature_toggles]
   enable = externalServiceAccounts
   ```

3. **Restart Grafana**:
   ```bash
   sudo systemctl restart grafana-server
   ```

4. **Check logs** for errors:
   ```bash
   tail -100 /var/log/grafana/grafana.log | grep -i "service account"
   ```

#### Getting Login Page Instead of Dashboard

**Symptoms**:
- PDF contains login page instead of dashboard
- Logs show: "ERROR: Got Grafana login page instead of dashboard!"

**Solutions**:

1. **Verify service account is active**:
   - Settings page should show "✓ Service Account: Active"

2. **Check token is being retrieved**:
   ```bash
   tail -f /var/log/grafana/grafana.log | grep "DEBUG: Using"
   ```
   Should see: `DEBUG: Using managed service account token from Grafana SDK`

3. **Restart Grafana** to refresh token:
   ```bash
   sudo systemctl restart grafana-server
   ```

4. **Open Settings page** to cache context (required for background jobs)

#### Chromium Not Found

**Symptoms**:
- Error: "Chrome/Chromium not found"
- Settings shows renderer error

**Solutions**:

1. **Download Chrome for Testing**:
   ```bash
   cd /var/lib/grafana/plugins/scheduled-reports-app/
   wget https://storage.googleapis.com/chrome-for-testing-public/131.0.6778.204/linux64/chrome-linux64.zip
   unzip chrome-linux64.zip
   chmod +x chrome-linux64/chrome
   ```

2. **Configure path in Settings**:
   - Chromium Path: `./chrome-linux64/chrome`

3. **Or install system Chrome**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install chromium-browser

   # CentOS/RHEL
   sudo yum install chromium
   ```

#### SMTP Email Not Sending

**Symptoms**:
- Schedule runs successfully
- No email received
- No error in logs

**Solutions**:

1. **Check SMTP settings**:
   - Verify host, port, username, password are correct
   - For Gmail, use App Password (not regular password)

2. **Test SMTP connection**:
   ```bash
   telnet smtp.gmail.com 587
   ```

3. **Check Grafana logs** for SMTP errors:
   ```bash
   tail -f /var/log/grafana/grafana.log | grep -i "smtp\|mail"
   ```

4. **Verify recipients**:
   - Email addresses must be valid
   - Check spam folder

### Docker Deployment

If running Grafana in Docker:

```dockerfile
# docker-compose.yml
version: '3'
services:
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_FEATURE_TOGGLES_ENABLE=externalServiceAccounts
      - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=scheduled-reports-app
    volumes:
      - ./dist:/var/lib/grafana/plugins/scheduled-reports-app
      - ./chrome-linux64:/opt/chrome  # Bundle Chrome
    ports:
      - "3000:3000"
```

**Important for Docker**:
- Chrome needs `--no-sandbox` flag (automatically enabled)
- May need `--disable-gpu` (automatically enabled)
- Bundle Chrome binary in plugin directory

### Production Deployment Checklist

- ✅ Grafana 10.3+ installed
- ✅ Feature toggle `externalServiceAccounts` enabled
- ✅ Plugin installed in plugins directory
- ✅ Grafana restarted after plugin installation
- ✅ Service account status shows "Active"
- ✅ SMTP configured for email delivery
- ✅ Chromium binary available and executable
- ✅ Grafana URL configured correctly
- ✅ Test schedule runs successfully
- ✅ PDF artifact contains dashboard (not login page)

### Support

- **Documentation**: See `AUTHENTICATION.md` for details
- **Issues**: Check Grafana logs in `/var/log/grafana/grafana.log`
- **Settings**: Verify service account status in Settings page
- **GitHub**: Report issues at repository URL

## Summary

**Minimum Setup**:
1. Enable feature toggle: `externalServiceAccounts`
2. Restart Grafana
3. Verify service account is active

**Full Setup**:
1. Enable feature toggle
2. Restart Grafana
3. Configure SMTP settings
4. Configure Grafana URL
5. Install Chromium (if needed)
6. Test schedule run

That's it! The plugin handles everything else automatically via Grafana's managed service accounts.
