# Authentication Guide

## Overview

The Scheduled Reports plugin uses **Grafana Managed Service Accounts** for authentication. This is the recommended approach for Grafana app plugins that need to access dashboards and datasources programmatically.

## How It Works

### 1. Automatic Service Account Creation

When the plugin starts, Grafana automatically:
1. Creates a service account for the plugin (named `extsvc-scheduled-reports-app`)
2. Grants permissions defined in `plugin.json` IAM section
3. Makes the token available via the Grafana SDK

### 2. Token Retrieval

The plugin retrieves tokens in this priority order:

1. **Grafana SDK Context** (preferred for Grafana 10.3+):
   ```go
   cfg := backend.GrafanaConfigFromContext(ctx)
   token, _ := cfg.PluginAppClientSecret()
   ```

2. **Environment Variable** (fallback):
   ```bash
   GF_PLUGIN_APP_CLIENT_SECRET=<token>
   ```

### 3. Authentication Flow

```
┌─────────────────────────────────────────────────────────────┐
│ Scheduler Job                                               │
│  └─ Calls: renderer.RenderDashboard(ctx, schedule)         │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ ChromiumRenderer                                            │
│  ├─ Gets token from Grafana SDK or env                     │
│  ├─ Sets up request hijacking                              │
│  └─ Injects "Authorization: Bearer <token>" header         │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Chromium Browser                                            │
│  ├─ Intercepts ALL HTTP requests to Grafana                │
│  ├─ Adds Authorization header to each request              │
│  └─ Navigates to dashboard URL                             │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Grafana                                                     │
│  ├─ Validates Bearer token                                 │
│  ├─ Checks service account permissions                     │
│  ├─ Renders dashboard                                      │
│  └─ Returns content to browser                             │
└─────────────────────────────────────────────────────────────┘
```

## Setup Requirements

### Grafana Version
- **Minimum**: Grafana 10.3 or later
- **Recommended**: Latest stable version

### Feature Toggle

**Required for Grafana 10.3+**

Add to `grafana.ini`:
```ini
[feature_toggles]
enable = externalServiceAccounts
```

Or set environment variable:
```bash
export GF_FEATURE_TOGGLES_ENABLE=externalServiceAccounts
```

### Restart Grafana

After enabling the feature toggle:
```bash
sudo systemctl restart grafana-server
# OR
docker-compose restart grafana
```

## Verification

### Check Service Account Status

1. Navigate to: **Apps → Scheduled Reports → Settings**
2. Look for "Service Account Authentication" section

### Expected Status Indicators

✅ **Success** (Green):
```
✓ Service Account: Active
Token Status: Configured (234 characters)
Grafana automatically manages this service account based on plugin.json IAM permissions
```

❌ **Error** (Red):
```
✗ Service Account: Error
Issue: Failed to retrieve token
Solution: Enable externalServiceAccounts feature toggle and restart Grafana
```

⚠️ **Not Configured** (Yellow):
```
⚠ Service Account: Not Configured
Status: Service account token is empty
Solution: Restart Grafana to allow automatic service account creation
```

## Troubleshooting

### Issue: Service Account Not Active

**Symptoms:**
- Settings shows "✗ Service Account: Error" or "⚠ Not Configured"
- PDFs show login page instead of dashboard

**Solutions:**

1. **Verify Grafana version**:
   ```bash
   grafana-server -v
   ```
   Must be 10.3 or later

2. **Enable feature toggle** in `grafana.ini`:
   ```ini
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

### Issue: Login Page in PDF Instead of Dashboard

**Symptoms:**
- Generated PDF contains Grafana login page
- Logs show: "ERROR: Got Grafana login page instead of dashboard!"

**Solutions:**

1. **Verify service account is active** (Settings page should show "✓ Active")

2. **Check token is being retrieved** (look for this in logs):
   ```bash
   tail -f /var/log/grafana/grafana.log | grep "DEBUG: Using"
   ```
   Should see: `DEBUG: Using managed service account token from Grafana SDK`

3. **Restart Grafana** to refresh token:
   ```bash
   sudo systemctl restart grafana-server
   ```

4. **Open Settings page** in Grafana UI to cache context for background jobs

### Issue: Context Not Available for Background Jobs

**Symptoms:**
- Error: "Grafana config not available in context"
- Token works in API calls but not in scheduled jobs

**Solution:**
Trigger at least one HTTP request to the plugin before running schedules:
1. Open **Apps → Scheduled Reports → Settings** page
2. This caches the Grafana context for background jobs
3. Then run your schedule

### Issue: Permission Denied When Rendering

**Symptoms:**
- Error: "Permission denied" or "Access denied to dashboard"
- Token appears to be valid

**Solutions:**

1. **Check service account permissions** in Grafana:
   - Go to: **Administration → Service accounts**
   - Find: `extsvc-scheduled-reports-app`
   - Verify it has permissions to access the dashboard

2. **Verify dashboard folder permissions**:
   - Service account needs read access to dashboard's folder
   - Check IAM permissions in `plugin.json` are correct

3. **Check organization ID**:
   - Ensure service account and schedule are in same org
   - Review `org_id` field in schedule

## Testing Authentication

### Manual Token Test with curl

Verify token works before using in plugin:

```bash
# Get token from Settings page or logs
TOKEN="eyJrIjoiYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo..."
GRAFANA_URL="https://127.0.0.1:3000"
DASHBOARD_UID="abc123"

# Test 1: Get dashboard metadata
curl -k -H "Authorization: Bearer $TOKEN" \
     "$GRAFANA_URL/api/dashboards/uid/$DASHBOARD_UID"

# Test 2: Access dashboard page (should NOT redirect to login)
curl -k -H "Authorization: Bearer $TOKEN" \
     "$GRAFANA_URL/d/$DASHBOARD_UID?kiosk=tv&orgId=1"
```

**Expected:** JSON response with dashboard data, NOT login page HTML

### Check Logs for Debug Information

Run a schedule and monitor logs:

```bash
# Follow logs
tail -f /var/log/grafana/grafana.log | grep -E "(DEBUG:|ERROR:)"
```

**Look for:**
- ✅ `DEBUG: Using managed service account token from Grafana SDK (length: 234)`
- ✅ `DEBUG: [Request #1] Injected Authorization Bearer token for: https://...`
- ✅ `DEBUG: Page title: My Dashboard - Grafana`
- ❌ `ERROR: Failed to get service account token: no service account token available`
- ❌ `ERROR: Got Grafana login page instead of dashboard!`

## IAM Permissions

The plugin requires these permissions (defined in `plugin.json`):

```json
{
  "iam": {
    "permissions": [
      {
        "action": "datasources:query",
        "scope": "datasources:*"
      },
      {
        "action": "dashboards:read",
        "scope": "dashboards:*"
      },
      {
        "action": "folders:read",
        "scope": "folders:*"
      },
      {
        "action": "annotations:read",
        "scope": "annotations:*"
      }
    ]
  }
}
```

## Docker Deployment

For Docker environments, ensure feature toggle is set:

```yaml
# docker-compose.yml
services:
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_FEATURE_TOGGLES_ENABLE=externalServiceAccounts
      - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=scheduled-reports-app
    volumes:
      - ./dist:/var/lib/grafana/plugins/scheduled-reports-app
```

## Migration from Manual Service Accounts

**Old approach** (before v2.0):
- Manually created service accounts in Grafana UI
- Copied token to plugin settings
- Required admin credentials

**Current approach** (v2.0+):
- Grafana automatically creates service account
- Token retrieved via SDK
- No manual configuration needed

**Migration steps:**
1. Enable `externalServiceAccounts` feature toggle
2. Restart Grafana
3. Verify "Service Account: Active" in Settings
4. Delete old manual service accounts (optional cleanup)

## Additional Resources

- [Grafana Service Accounts Documentation](https://grafana.com/developers/plugin-tools/how-to-guides/app-plugins/use-a-service-account)
- [Feature Toggles Reference](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/feature-toggles/)
- [Plugin IAM Permissions](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin#plugin-signature-levels)

## Summary

✅ **Correct Implementation:**
- Grafana automatically creates managed service account
- Token retrieved via SDK: `cfg.PluginAppClientSecret()`
- No manual configuration needed
- Simple and reliable

**Key Requirements:**
1. Grafana 10.3+
2. `externalServiceAccounts` feature toggle enabled
3. Restart Grafana after plugin installation
4. Verify service account status in Settings

The plugin follows Grafana's recommended best practices for managed service accounts.
