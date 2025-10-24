# Grafana Catalog Validation Fixes

## Current Status
Branch: `fix/grafana-catalog-validation`

## Issues Summary

### 🔴 CRITICAL - Blocking Review

1. **❌ Filesystem Access Violations** (17 violations)
   - **Impact**: BLOCKS catalog approval
   - **Files affected**:
     - `pkg/api/handlers.go`: lines 280, 611, 613, 693
     - `pkg/cron/scheduler.go`: lines 289, 293
     - `pkg/render/chromium_renderer.go`: lines 33, 60, 129-132, 198, 607
   - **Root cause**: Plugin writes PDF artifacts to filesystem
   - **Grafana rule**: Plugins cannot access filesystem directly
   - **Solution required**:
     - Option A: Store PDFs in database as BLOBs (SQLite/BoltDB)
     - Option B: Use Grafana's storage API (if available)
     - Option C: Remove artifact storage feature (keep only email delivery)

2. **❌ Archive Structure**
   - **Issue**: ZIP must contain directory named `fulgerx2007-scheduled-reports-app/`
   - **Current**: ZIP contains files directly in root
   - **Fix**: Update release workflow packaging steps

3. **❌ Go Manifest Missing**
   - **Issue**: `go.mod` not found at repository root
   - **Current**: `go.mod` exists but validator can't find it
   - **Fix**: Ensure go.mod is included in source submission

### 🟡 HIGH Priority

4. **❌ README Relative Links** (9 links)
   - Convert to absolute GitHub URLs:
     - `./QUICKSTART.md` → `https://github.com/FulgerX2007/grafana-scheduled-reports-app/blob/master/QUICKSTART.md`
     - `./SETUP_GUIDE.md` → absolute
     - `./AUTHENTICATION.md` → absolute
     - `./E2E_TESTING.md` → absolute
     - `./BUILD.md` → absolute
     - `./SECURITY.md` → absolute
     - `./CLAUDE.md` → absolute
     - `./GRAFANA_CATALOG_SUBMISSION.md` → absolute
     - `LICENSE` → `https://github.com/FulgerX2007/grafana-scheduled-reports-app/blob/master/LICENSE`

5. **❌ Invalid Checksum Format**
   - **Issue**: SHA256 provided but only MD5/SHA1 supported
   - **Fix**: Provide MD5 or SHA1 checksum instead

### 🟢 MEDIUM Priority

6. **⚠️ Environment Variable Access** (2 warnings)
   - `GF_URL` accessed in `pkg/api/handlers.go:527`
   - `GRAFANA_HOSTNAME` accessed in `pkg/render/chromium_renderer.go:634`
   - **Fix**: Use Grafana SDK APIs instead

7. **⚠️ Missing CHANGELOG.md**
   - **Fix**: Create CHANGELOG.md with version history

8. **⚠️ Broken Link**
   - `http://localhost:3000/...` in README
   - **Fix**: Remove or make relative to user's instance

### 🔵 LOW Priority (Suggestions)

9. **💡 Sponsorship Link**
   - Consider adding sponsorship link to plugin.json

10. **💡 Build Provenance Attestation**
    - Enable GitHub Actions attestation

## Recommended Action Plan

Due to the **filesystem access violations being a fundamental architectural issue**, we have three options:

### Option A: Refactor to Database Storage (RECOMMENDED)
**Effort**: Medium (4-6 hours)
**Impact**: Maintains all features
**Steps**:
1. Modify `pkg/store` to store PDF BLOBs in SQLite
2. Update handlers to serve PDFs from database
3. Add cleanup/retention logic for database
4. Test artifact download functionality

### Option B: Remove Artifact Storage
**Effort**: Low (1-2 hours)
**Impact**: Loses artifact download feature
**Steps**:
1. Remove file storage code
2. Keep email delivery only
3. Remove download endpoints
4. Update UI to hide download buttons

### Option C: Request Exemption
**Effort**: Unknown
**Impact**: May be rejected
**Steps**:
1. Contact Grafana support
2. Explain use case for filesystem access
3. Wait for decision

## Immediate Next Steps

1. **DECISION NEEDED**: Choose Option A, B, or C for filesystem access
2. Fix README relative links (easy, 10 minutes)
3. Fix archive structure (easy, 15 minutes)
4. Create CHANGELOG.md (easy, 10 minutes)
5. Address chosen filesystem solution (varies)
6. Update release workflow
7. Test locally with validator
8. Resubmit to Grafana

## Files to Modify

- [ ] `README.md` - Fix 9 relative links
- [ ] `.github/workflows/release.yml` - Fix archive packaging
- [ ] `CHANGELOG.md` - Create new file
- [ ] `pkg/api/handlers.go` - Remove filesystem access
- [ ] `pkg/cron/scheduler.go` - Remove filesystem access
- [ ] `pkg/render/chromium_renderer.go` - Remove filesystem access
- [ ] `pkg/store/` - Add BLOB storage (if Option A)
- [ ] `src/plugin.json` - Already fixed (rootUrls removed)

---

**Status**: ✅ Created validation fixes plan
**Next**: Awaiting user decision on filesystem access approach
