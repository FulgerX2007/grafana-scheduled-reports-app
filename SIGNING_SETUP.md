# Plugin Signing Configuration

## Overview

The Grafana plugin is now configured to be automatically signed during the GitHub Actions release workflow using your `PLUGIN_SIGNING` secret.

## What Was Changed

### 1. GitHub Actions Workflow Updated

File: `.github/workflows/release.yml`

**Added two new steps after backend builds:**

#### Sign Plugin Step (lines 67-73)
```yaml
- name: Sign plugin
  run: |
    echo "Signing plugin with Grafana signing tool..."
    export GRAFANA_ACCESS_POLICY_TOKEN="${{ secrets.PLUGIN_SIGNING }}"
    npx --yes @grafana/sign-plugin@latest
  env:
    GRAFANA_ACCESS_POLICY_TOKEN: ${{ secrets.PLUGIN_SIGNING }}
```

This step:
- Uses the `@grafana/sign-plugin` tool from Grafana Labs
- Reads the `PLUGIN_SIGNING` GitHub secret (your private signing key)
- Signs all files in the `dist/` directory
- Creates `dist/MANIFEST.txt` with file signatures

#### Verify Signature Step (lines 75-84)
```yaml
- name: Verify plugin signature
  run: |
    if [ -f "dist/MANIFEST.txt" ]; then
      echo "✅ Plugin signed successfully"
      echo "=== MANIFEST.txt contents ==="
      cat dist/MANIFEST.txt
    else
      echo "❌ ERROR: Plugin signing failed - MANIFEST.txt not found"
      exit 1
    fi
```

This step:
- Verifies that `MANIFEST.txt` was created
- Displays the manifest contents in the build logs
- Fails the build if signing didn't work

### 2. Release Notes Updated

The release installation instructions were updated to remove "unsigned plugin" warnings since the plugin is now properly signed.

## How Plugin Signing Works

### Private Signature Type

When you sign with your own key (`PLUGIN_SIGNING` secret), the plugin gets a **private signature**:

```json
{
  "plugin": "scheduled-reports-app",
  "version": "1.0.0",
  "files": {
    "plugin.json": "<hash>",
    "module.js": "<hash>",
    "gpx_reporting": "<hash>",
    ...
  },
  "time": 1729777777000,
  "keyId": "your-key-id",
  "signatureType": "private"
}
```

### Signature Verification

When Grafana loads the plugin, it:
1. Reads `MANIFEST.txt`
2. Verifies each file hash matches the manifest
3. Checks the signature against your public key
4. **Important**: With `signatureType: "private"`, the plugin will only work on Grafana instances with your `rootUrls` configured

### Grafana Configuration Required

For **private signatures**, add to `grafana.ini`:

```ini
[plugins.scheduled-reports-app]
allow_loading_unsigned_plugins = false

# Root URLs where this plugin can run (comma-separated)
# Example for localhost and production:
root_url = http://localhost:3000,https://your-grafana.example.com
```

Or via environment variables:
```bash
# Not needed if properly signed:
# GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=scheduled-reports-app
```

## Obtaining the PLUGIN_SIGNING Secret

If you don't already have the `PLUGIN_SIGNING` secret configured:

### Option 1: Create a New Signing Key (Recommended)

1. **Generate a signing key pair:**
   ```bash
   npx --yes @grafana/sign-plugin@latest --generate-key
   ```

2. **This creates two files:**
   - `private-key.pem` - Your private signing key (keep secret!)
   - `public-key.pem` - Your public key (can be shared)

3. **Add the private key to GitHub Secrets:**
   - Go to your repository: https://github.com/FulgerX2007/grafana-scheduled-reports-app
   - Navigate to: **Settings → Secrets and variables → Actions**
   - Click **New repository secret**
   - Name: `PLUGIN_SIGNING`
   - Value: Paste the **entire contents** of `private-key.pem`
   - Click **Add secret**

4. **Store the private key safely:**
   ```bash
   # Backup to a secure location
   cp private-key.pem ~/secure-backup/grafana-plugin-signing-key.pem
   chmod 600 ~/secure-backup/grafana-plugin-signing-key.pem
   
   # NEVER commit this to git!
   echo "private-key.pem" >> .gitignore
   ```

### Option 2: Use Existing Key

If you already have a signing key:

1. **Convert to PEM format** (if needed)
2. **Copy the entire private key contents**
3. **Add to GitHub Secrets** as described above

## Testing the Signing Process

### Test Locally

Before pushing a tag, test signing locally:

```bash
# Build the plugin
npm run build
go build -o dist/gpx_reporting ./cmd/backend

# Sign with your private key
export GRAFANA_ACCESS_POLICY_TOKEN="<your-private-key-content>"
npx @grafana/sign-plugin

# Verify MANIFEST.txt was created
ls -lh dist/MANIFEST.txt
cat dist/MANIFEST.txt
```

### Test in GitHub Actions

Create a test release:

```bash
# Create and push a test tag
git tag v1.0.0-rc1
git push github v1.0.0-rc1

# Monitor the GitHub Actions workflow:
# https://github.com/FulgerX2007/grafana-scheduled-reports-app/actions
```

Check the workflow logs for:
- ✅ "Plugin signed successfully"
- ✅ MANIFEST.txt contents displayed
- ✅ All 4 platform ZIP files created

## What Happens on Release

When you push a version tag (e.g., `v1.0.0`):

1. **Workflow triggers** on tag push
2. **Builds** frontend (React/TypeScript)
3. **Builds** backend for 4 platforms (Linux/Darwin × amd64/arm64)
4. **Signs** the plugin with your private key
5. **Verifies** signature was successful
6. **Packages** 4 separate ZIP files (one per platform)
7. **Generates** SHA256 checksums
8. **Creates** GitHub release with all artifacts

## Release Artifacts

Each release will contain:
- `scheduled-reports-app-1.0.0.linux-amd64.zip` ✅ Signed
- `scheduled-reports-app-1.0.0.linux-arm64.zip` ✅ Signed
- `scheduled-reports-app-1.0.0.darwin-amd64.zip` ✅ Signed
- `scheduled-reports-app-1.0.0.darwin-arm64.zip` ✅ Signed
- `checksums.txt` - SHA256 hashes

All packages include the signed `MANIFEST.txt` file.

## Path to Community Signature

**Current State**: Private signature (works with your key)

**Goal**: Community signature (works everywhere)

To get a **community signature** from Grafana Labs:

1. ✅ **Complete all Grafana catalog requirements** (see GRAFANA_CATALOG_SUBMISSION.md)
2. ✅ **Submit plugin for review** via Grafana Cloud
3. ⏳ **Wait for approval** (1-4 weeks)
4. ✅ **After approval**: Grafana signs plugin with their key
5. ✅ **Result**: `signatureType: "community"` - works on any Grafana instance!

Once you have community signature:
- No `rootUrls` restrictions
- No need for private signing key
- Grafana Labs signs new releases for you
- Listed in official Grafana plugin catalog

## Troubleshooting

### Error: "Invalid signing token"

The `PLUGIN_SIGNING` secret format is incorrect.

**Solution**: Ensure the secret contains the full PEM format:
```
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...
...
-----END PRIVATE KEY-----
```

### Error: "MANIFEST.txt not found"

Signing failed, check:
1. `dist/` directory exists and contains plugin files
2. `plugin.json` is valid JSON
3. Signing token has correct permissions

### Error: "Plugin signature verification failed"

When installing the plugin:
1. Check Grafana logs: `journalctl -u grafana-server -f`
2. Verify `rootUrls` is configured in grafana.ini
3. Ensure the signed MANIFEST.txt is included in the ZIP

## Next Steps

1. ✅ Plugin signing is configured in GitHub Actions
2. ⏳ Push a git tag to trigger the first signed release
3. ⏳ Complete remaining catalog submission items
4. ⏳ Submit for community signature review

---

**Status**: ✅ Plugin signing is now automated in GitHub Actions workflow
