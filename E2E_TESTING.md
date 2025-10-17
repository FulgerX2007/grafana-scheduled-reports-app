# End-to-End (E2E) Testing Guide

This document provides comprehensive information about E2E testing for the Scheduled Reports Grafana plugin.

## Overview

The plugin uses [Playwright](https://playwright.dev/) for E2E testing to ensure it works correctly across different Grafana versions and real-world usage scenarios. E2E tests simulate actual user interactions with the plugin running in a live Grafana instance.

## Why E2E Testing?

- **Real-world validation**: Tests the plugin in an environment similar to production
- **Multi-version compatibility**: Ensures the plugin works with different Grafana versions
- **Regression prevention**: Catches breaking changes before they reach users
- **Integration verification**: Tests full workflows including rendering and email delivery
- **Submission readiness**: Required for Grafana plugin marketplace submission

## Test Structure

```
e2e/
├── fixtures/
│   └── auth.ts              # Authentication helpers
├── tests/
│   ├── plugin-health.spec.ts    # Plugin loading and enablement
│   ├── schedule-crud.spec.ts    # Schedule CRUD operations
│   ├── report-execution.spec.ts # Report generation and download
│   └── settings.spec.ts         # Settings configuration
playwright.config.ts         # Playwright configuration
docker-compose.e2e.yml       # E2E test environment
```

## Test Coverage

### 1. Plugin Health Checks (`plugin-health.spec.ts`)
- Plugin loads successfully
- Plugin can be enabled
- Navigation to plugin app works
- Settings page is accessible

### 2. Schedule CRUD (`schedule-crud.spec.ts`)
- Create new schedule with all required fields
- List all schedules
- View schedule details
- Edit existing schedule
- Delete schedule

### 3. Report Execution (`report-execution.spec.ts`)
- Run schedule manually
- View run history
- Download PDF/HTML artifacts
- End-to-end report generation workflow

### 4. Settings Configuration (`settings.spec.ts`)
- SMTP configuration
- Renderer settings
- Usage limits configuration

## Test Environment

The E2E environment consists of three services:

1. **Grafana** (`localhost:3000`)
   - Admin credentials: admin/admin
   - Plugin pre-loaded from `./dist` directory
   - Test dashboard provisioned automatically

2. **Grafana Image Renderer** (`localhost:8081`)
   - Headless Chrome-based renderer
   - Used for generating PDF/HTML reports

3. **MailHog** (`localhost:8025`)
   - SMTP test server
   - Web UI for viewing sent emails
   - Simulates email delivery

## Running Tests

### Prerequisites

```bash
# Install dependencies
npm install

# Build plugin
npm run build
go build -o dist/gpx_reporting ./cmd/backend
```

### Local Development

```bash
# Start E2E environment
npm run e2e:setup

# Run tests (headless)
npm run test:e2e

# Run tests with UI (interactive mode)
npm run test:e2e:ui

# Run tests in headed browser
npm run test:e2e:headed

# Debug specific test
npm run test:e2e:debug

# View test report
npm run test:e2e:report

# Teardown environment
npm run e2e:teardown
```

### Manual Docker Commands

```bash
# Start services
docker compose -f docker-compose.e2e.yml up -d

# Check service health
docker compose -f docker-compose.e2e.yml ps

# View logs
docker compose -f docker-compose.e2e.yml logs -f

# Stop services
docker compose -f docker-compose.e2e.yml down -v
```

## CI/CD Integration

### GitHub Actions Workflow

The `.github/workflows/e2e-tests.yml` workflow runs automatically on:
- Push to `main` or `develop` branches
- Pull requests targeting `main` or `develop`
- Manual trigger via workflow_dispatch

### Matrix Testing

Tests run against multiple Grafana versions:
- 10.0.0 (minimum supported version)
- 11.0.0 (current stable)
- latest (bleeding edge)

### Artifacts

When tests fail, the following are uploaded as artifacts:
- Playwright HTML report
- Test videos
- Screenshots
- Docker logs

## Writing New Tests

### Test Template

```typescript
import { test, expect } from '../fixtures/auth';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ authenticatedPage: page }) => {
    // Setup code runs before each test
    await page.goto('/a/sheduled-reports-app');
  });

  test('should do something', async ({ authenticatedPage: page }) => {
    // Your test code
    await page.click('button:has-text("New Schedule")');
    await expect(page.locator('h1')).toHaveText('Create Schedule');
  });
});
```

### Best Practices

1. **Use semantic locators**: Prefer text content and ARIA labels over CSS selectors
   ```typescript
   // Good
   await page.click('button:has-text("Save")');
   await page.locator('input[aria-label="Schedule name"]').fill('My Schedule');

   // Avoid
   await page.click('.btn-primary');
   ```

2. **Wait for network idle**: Let async operations complete
   ```typescript
   await page.waitForLoadState('networkidle');
   ```

3. **Handle conditional elements**: Check visibility before interacting
   ```typescript
   const modal = page.locator('[role="dialog"]');
   if (await modal.isVisible()) {
     await modal.locator('button:has-text("Close")').click();
   }
   ```

4. **Use appropriate timeouts**: Give operations time to complete
   ```typescript
   await expect(page.locator('text=Success')).toBeVisible({ timeout: 10000 });
   ```

5. **Clean up test data**: Delete created resources after tests
   ```typescript
   test.afterEach(async ({ authenticatedPage: page }) => {
     // Cleanup code
   });
   ```

## Debugging Tests

### Interactive UI Mode

```bash
npm run test:e2e:ui
```
- Time-travel debugging
- Watch mode
- Pick specific tests
- See network requests

### Debug Mode

```bash
npm run test:e2e:debug
```
- Runs with Playwright Inspector
- Step through test code
- Set breakpoints
- Inspect page state

### Headed Mode

```bash
npm run test:e2e:headed
```
- Watch browser in real-time
- See what the test sees
- Useful for understanding failures

### Screenshots and Videos

Videos and screenshots are automatically captured on failure:
```
test-results/
└── feature-name-test-name-chromium/
    ├── video.webm
    └── test-failed-1.png
```

## Troubleshooting

### Tests Failing Locally

1. **Check services are running**:
   ```bash
   docker compose -f docker-compose.e2e.yml ps
   ```

2. **Verify Grafana is healthy**:
   ```bash
   curl http://localhost:3000/api/health
   ```

3. **Check plugin is loaded**:
   - Visit http://localhost:3000/plugins
   - Search for "Scheduled Reports"

4. **View service logs**:
   ```bash
   docker compose -f docker-compose.e2e.yml logs grafana
   ```

### Slow Tests

- Increase timeouts in `playwright.config.ts`
- Reduce parallelism with `workers: 1`
- Check Docker resource limits

### Flaky Tests

- Add explicit waits for async operations
- Use `waitForLoadState('networkidle')`
- Increase retry count for unreliable assertions

### CI Failures

- Check uploaded artifacts for videos and logs
- Compare Grafana versions between local and CI
- Verify Docker resource availability in CI

## Performance Considerations

- Tests run in parallel by default (configurable)
- Each test starts with fresh authentication
- Docker services have health checks
- Automatic cleanup after test runs

## Updating Tests

When plugin functionality changes:

1. Update affected test files
2. Run tests locally to verify
3. Update this documentation if needed
4. Ensure CI passes before merging

## Resources

- [Playwright Documentation](https://playwright.dev/)
- [Grafana Plugin Development](https://grafana.com/developers/plugin-tools/)
- [Grafana Plugin E2E](https://github.com/grafana/plugin-tools/tree/main/packages/plugin-e2e)
- [Testing Best Practices](https://playwright.dev/docs/best-practices)
