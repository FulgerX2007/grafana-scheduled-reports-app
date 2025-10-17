import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

export const DocumentationPage: React.FC = () => {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <img src="public/plugins/sheduled-reports-app/img/logo.png" alt="Scheduled Reports" className={styles.logo} />
        <h1>Scheduled Reports Documentation</h1>
      </div>

      <section className={styles.section}>
        <h2>Overview</h2>
        <p>
          The Scheduled Reports plugin allows you to schedule automatic generation and delivery of dashboard
          reports via email. Reports can be generated as PDF or HTML and sent on a daily, weekly, monthly,
          or custom cron schedule.
        </p>
      </section>

      <section className={styles.section}>
        <h2>Recent Updates</h2>

        <h3>October 2025 - Timezone and Validation Improvements</h3>
        <ul>
          <li>
            <strong>Timezone-Aware Scheduling:</strong> Schedules now run at the correct time in their configured
            timezone. Daily schedules run at midnight (00:00), weekly on Monday at midnight, and monthly on the
            1st at midnight in the schedule's timezone.
          </li>
          <li>
            <strong>Automatic CRON Expression Generation:</strong> Existing schedules without CRON expressions
            are automatically updated with the correct CRON format based on their interval type (daily/weekly/monthly).
          </li>
          <li>
            <strong>CRON Expression Validation:</strong> Invalid CRON expressions are now validated when creating
            or updating schedules, preventing configuration errors.
          </li>
          <li>
            <strong>SMTP Configuration Fix:</strong> Resolved port validation errors when configuring SMTP with
            common ports like 587. Settings now properly initialize with default values.
          </li>
          <li>
            <strong>Variable Editor Enhancement:</strong> Duplicate button is now hidden for select-type dashboard
            variables (those with predefined options), improving UI clarity.
          </li>
        </ul>
        <p>
          <em>Note:</em> If you created schedules before this update, they have been automatically migrated to use
          proper CRON expressions. Your next run times will now execute at the exact scheduled time in your timezone,
          not at the time the schedule was last triggered.
        </p>
      </section>

      <section className={styles.section}>
        <h2>Getting Started</h2>

        <h3>1. Configure Plugin Settings</h3>
        <p>Before creating schedules, configure the plugin in the Settings page:</p>
        <ul>
          <li><strong>Service Account Authentication:</strong> Automatic for Grafana 10.3+ (uses managed service accounts via IAM permissions)</li>
          <li><strong>SMTP Settings:</strong> Configure email delivery with Grafana's SMTP or custom SMTP server</li>
          <li><strong>Chromium Renderer:</strong> Configure Chrome/Chromium browser for PDF generation</li>
          <li><strong>Usage Limits:</strong> Set quotas for recipients, file sizes, concurrent renders, and retention</li>
        </ul>

        <h3>2. Create a Schedule</h3>
        <ol>
          <li>Click "New Schedule" button</li>
          <li>Fill in the schedule details (see below)</li>
          <li>Click "Create" to save</li>
        </ol>
      </section>

      <section className={styles.section}>
        <h2>Schedule Configuration</h2>

        <h3>Basic Information</h3>
        <ul>
          <li><strong>Name:</strong> A descriptive name for your schedule (e.g., "Daily Sales Report")</li>
          <li><strong>Dashboard:</strong> Select the dashboard to report</li>
          <li><strong>Format:</strong> Choose PDF or HTML output</li>
          <li><strong>Enabled:</strong> Enable or disable the schedule</li>
        </ul>

        <h3>Time Range</h3>
        <ul>
          <li><strong>From:</strong> Start of the time range (e.g., "now-24h", "now-7d", "2024-01-01")</li>
          <li><strong>To:</strong> End of the time range (e.g., "now")</li>
        </ul>

        <h3>Schedule Intervals</h3>
        <ul>
          <li><strong>Daily:</strong> Runs every day at midnight (00:00) in the configured timezone</li>
          <li><strong>Weekly:</strong> Runs every Monday at midnight (00:00) in the configured timezone</li>
          <li><strong>Monthly:</strong> Runs on the 1st of each month at midnight (00:00) in the configured timezone</li>
          <li><strong>Custom (Cron):</strong> Use cron expressions for precise scheduling (any time/day combination)</li>
        </ul>

        <h4>Timezone Support</h4>
        <p>
          Schedules now support timezone-aware execution. When you set a schedule to run "daily", it will execute
          at midnight (00:00) in the configured timezone, not 24 hours from the last run. This ensures consistent
          execution times regardless of when the schedule was created or last triggered.
        </p>
        <ul>
          <li><strong>Timezone Configuration:</strong> Set in the schedule's timezone field (e.g., "America/New_York", "Europe/London", "Asia/Tokyo")</li>
          <li><strong>Default Timezone:</strong> UTC if no timezone is specified</li>
          <li><strong>Invalid Timezones:</strong> Automatically fall back to UTC with a warning in logs</li>
          <li><strong>Next Run Display:</strong> Shows the next execution time in UTC in the schedule list</li>
        </ul>

        <p>
          <em>Example:</em> A daily schedule with timezone "America/New_York" will run at midnight Eastern Time
          (00:00 ET), which corresponds to 05:00 UTC (during EST) or 04:00 UTC (during EDT).
        </p>

        <h3>Cron Expression Format</h3>
        <p>Cron expressions use 5 fields: <code>minute hour day-of-month month day-of-week</code></p>

        <div className={styles.codeBlock}>
          <table>
            <thead>
              <tr>
                <th>Expression</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><code>0 8 * * *</code></td>
                <td>Every day at 8:00 AM</td>
              </tr>
              <tr>
                <td><code>0 9 * * 1</code></td>
                <td>Every Monday at 9:00 AM</td>
              </tr>
              <tr>
                <td><code>0 18 * * 1-5</code></td>
                <td>Every weekday at 6:00 PM</td>
              </tr>
              <tr>
                <td><code>0 0 1 * *</code></td>
                <td>First day of every month at midnight</td>
              </tr>
              <tr>
                <td><code>*/15 * * * *</code></td>
                <td>Every 15 minutes</td>
              </tr>
              <tr>
                <td><code>0 8 * * 0</code></td>
                <td>Every Sunday at 8:00 AM</td>
              </tr>
            </tbody>
          </table>
        </div>

        <h3>Dashboard Variables</h3>
        <p>
          When you select a dashboard, its template variables are automatically loaded into the editor.
          You can modify the values that will be applied when rendering the report.
        </p>
        <p>
          <strong>Example:</strong> If your dashboard has variables like "datacenter", "environment", or "region",
          they will appear in the Variables section. Set them to specific values (e.g., <code>datacenter=us-west</code>,
          <code>environment=production</code>) to customize the report output.
        </p>

        <h3>Email Configuration</h3>
        <ul>
          <li><strong>Recipients:</strong> Comma-separated email addresses (To, CC, BCC)</li>
          <li><strong>Subject:</strong> Email subject line (supports template variables)</li>
          <li><strong>Body:</strong> Email body text (supports template variables)</li>
        </ul>

        <h4>Template Variables</h4>
        <p>You can use the following variables in email subject and body:</p>
        <ul>
          <li><code>{'{{schedule.name}}'}</code> - Name of the schedule</li>
          <li><code>{'{{dashboard.title}}'}</code> - Dashboard title</li>
          <li><code>{'{{timerange}}'}</code> - Time range used for the report</li>
          <li><code>{'{{run.started_at}}'}</code> - When the report generation started</li>
        </ul>
      </section>

      <section className={styles.section}>
        <h2>Managing Schedules</h2>

        <h3>Schedule Actions</h3>
        <ul>
          <li><strong>▶ Run Now:</strong> Execute the schedule immediately</li>
          <li><strong>⏸ Pause/Resume:</strong> Disable or enable the schedule</li>
          <li><strong>✏️ Edit:</strong> Modify schedule configuration</li>
          <li><strong>🕐 History:</strong> View past report runs and results</li>
          <li><strong>🗑️ Delete:</strong> Remove the schedule permanently</li>
        </ul>

        <h3>Run History</h3>
        <p>
          The Run History page shows all past executions of a schedule, including:
        </p>
        <ul>
          <li>Execution time and duration</li>
          <li>Status (completed, failed, running)</li>
          <li>Number of pages rendered</li>
          <li>File size</li>
          <li>Error messages (if failed)</li>
          <li>Download button for successful reports</li>
        </ul>
      </section>


      <section className={styles.section}>
        <h2>Settings</h2>

        <h3>SMTP Configuration</h3>
        <p>
          You can either use Grafana's SMTP settings or configure custom SMTP settings for the plugin:
        </p>
        <ul>
          <li><strong>Use Grafana SMTP:</strong> Uses environment variables from Grafana configuration</li>
          <li><strong>Custom SMTP:</strong> Configure separate SMTP settings for reporting</li>
        </ul>

        <h3>Chromium Renderer Configuration</h3>
        <p>The plugin uses Chrome/Chromium browser for high-fidelity PDF generation with full JavaScript support:</p>
        <ul>
          <li><strong>Chromium Path:</strong> Path to Chrome/Chromium binary (auto-detected if empty)</li>
          <li><strong>Timeout:</strong> Maximum rendering time in milliseconds (default: 30000ms / 30 seconds)</li>
          <li><strong>Delay:</strong> Wait time after page load for queries to complete (default: 5000ms / 5 seconds)</li>
          <li><strong>Viewport:</strong> Browser viewport dimensions (default: 1920x1080)</li>
          <li><strong>Device Scale Factor:</strong> Quality multiplier for higher resolution (default: 2.0, range: 1.0-4.0)</li>
          <li><strong>Headless Mode:</strong> Run browser without GUI (always enabled for servers)</li>
          <li><strong>No Sandbox:</strong> Disable Chrome sandbox (required for Docker/containerized environments)</li>
          <li><strong>Disable GPU:</strong> Disable GPU acceleration (recommended for servers without display)</li>
          <li><strong>Skip TLS Verify:</strong> Skip certificate verification (for self-signed Grafana certificates)</li>
        </ul>
        <p>
          <strong>Tip:</strong> Use the <em>"Check Chromium Version"</em> button in Settings to verify your Chrome/Chromium installation.
        </p>

        <h3>Limits</h3>
        <ul>
          <li><strong>Max Recipients:</strong> Maximum number of email recipients per schedule</li>
          <li><strong>Max Attachment Size:</strong> Maximum report file size in MB</li>
          <li><strong>Max Concurrent Renders:</strong> Number of reports that can render simultaneously</li>
          <li><strong>Retention Days:</strong> How long to keep report artifacts</li>
        </ul>
      </section>

      <section className={styles.section}>
        <h2>Troubleshooting</h2>

        <h3>Reports Not Being Generated</h3>
        <ul>
          <li>Check that the schedule is enabled</li>
          <li>Verify the "Next Run" time is in the future</li>
          <li>Check the Run History for error messages</li>
          <li>Verify the service account token is configured</li>
        </ul>

        <h3>Rendering Errors</h3>
        <ul>
          <li>Verify Chrome/Chromium is installed: Use "Check Chromium Version" button in Settings</li>
          <li>Configure Chromium path if auto-detection fails</li>
          <li>Enable required flags: No Sandbox (Docker), Disable GPU (servers), Skip TLS Verify (self-signed certs)</li>
          <li>Increase timeout (default: 30s) for slow dashboards</li>
          <li>Increase delay (default: 5s) for dashboards with slow queries</li>
          <li>Verify service account has access to the dashboard</li>
          <li>Check plugin logs: <code>docker logs grafana | grep -i chromium</code></li>
        </ul>

        <h3>Email Delivery Issues</h3>
        <ul>
          <li>Verify SMTP configuration is correct</li>
          <li>Check email addresses are valid</li>
          <li>Look for error messages in Run History</li>
          <li>Test SMTP settings with "Test Email" button (if available)</li>
        </ul>

        <h3>Dashboard Variables Not Working</h3>
        <ul>
          <li>Ensure variable names match exactly (case-sensitive)</li>
          <li>Check that variables are defined in the dashboard</li>
          <li>Use the format <code>var-variableName</code> in the configuration</li>
        </ul>

        <h3>Schedule Running at Wrong Time</h3>
        <ul>
          <li><strong>Check Timezone Configuration:</strong> Verify the schedule's timezone is set correctly (e.g., "America/New_York")</li>
          <li><strong>Verify CRON Expression:</strong> Use the preview in the schedule editor to see next 5 run times</li>
          <li><strong>Migration Note:</strong> Schedules created before October 2025 have been automatically migrated to use proper CRON expressions</li>
          <li><strong>Next Run Time:</strong> Displayed in UTC in the schedule list - convert to your local timezone to verify</li>
        </ul>

        <h3>SMTP Port Validation Errors</h3>
        <ul>
          <li><strong>Common Ports:</strong> Port 587 (STARTTLS), 465 (SSL/TLS), and 25 (plain) are all supported</li>
          <li><strong>Configuration Reset:</strong> If you see validation errors, try reloading the Settings page</li>
          <li><strong>Default Values:</strong> Settings now initialize with sensible defaults (port 587, TLS enabled)</li>
        </ul>
      </section>

      <section className={styles.section}>
        <h2>Best Practices</h2>

        <ul>
          <li><strong>Use descriptive names:</strong> Make it easy to identify schedules at a glance</li>
          <li><strong>Set appropriate time ranges:</strong> Match the schedule interval (e.g., "now-24h" for daily)</li>
          <li><strong>Test before scheduling:</strong> Use "Run Now" to verify reports look correct</li>
          <li><strong>Monitor Run History:</strong> Check for failed runs regularly</li>
          <li><strong>Use template variables:</strong> Make email content dynamic and informative</li>
          <li><strong>Set retention policies:</strong> Clean up old reports to save storage</li>
          <li><strong>Limit recipients:</strong> Keep recipient lists focused to reduce email volume</li>
        </ul>
      </section>

      <section className={styles.section}>
        <h2>Technical Details</h2>

        <h3>Architecture</h3>
        <p>The Scheduled Reports plugin consists of:</p>
        <ul>
          <li><strong>Frontend:</strong> React/TypeScript UI for managing schedules and settings</li>
          <li><strong>Backend:</strong> Go service with cron scheduler and background workers</li>
          <li><strong>Database:</strong> SQLite for storing schedules, run history, and settings</li>
          <li><strong>Renderer:</strong> Chromium/Chrome browser (via go-rod) for direct PDF generation</li>
          <li><strong>Email:</strong> SMTP client for report delivery</li>
          <li><strong>Authentication:</strong> Grafana managed service accounts (IAM permissions)</li>
        </ul>

        <h3>Data Storage</h3>
        <ul>
          <li><strong>Database:</strong> /var/lib/grafana/plugin-data/reporting.db</li>
          <li><strong>Artifacts:</strong> /var/lib/grafana/plugin-data/artifacts/org_[id]/</li>
        </ul>

        <h3>Security</h3>
        <ul>
          <li>Service account tokens are stored securely in Grafana's encrypted settings</li>
          <li>All schedules are scoped by organization ID</li>
          <li>Users must have Editor role to create schedules</li>
          <li>Admin role required to modify plugin settings</li>
        </ul>
      </section>

      <section className={styles.section}>
        <h2>Support</h2>
        <p>
          For additional help or to report issues, please contact your Grafana administrator
          or visit the plugin documentation repository.
        </p>
      </section>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(3)};
    max-width: 1200px;
    margin: 0 auto;

    h1 {
      margin-bottom: ${theme.spacing(3)};
      border-bottom: 2px solid ${theme.colors.border.medium};
      padding-bottom: ${theme.spacing(2)};
    }
  `,
  header: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(3)};
    border-bottom: 2px solid ${theme.colors.border.medium};
    padding-bottom: ${theme.spacing(2)};

    h1 {
      margin: 0;
      border: none;
      padding: 0;
    }
  `,
  logo: css`
    width: 64px;
    height: 64px;
    object-fit: contain;
  `,
  section: css`
    margin-bottom: ${theme.spacing(4)};

    h2 {
      margin-top: ${theme.spacing(4)};
      margin-bottom: ${theme.spacing(2)};
      color: ${theme.colors.primary.text};
    }

    h3 {
      margin-top: ${theme.spacing(3)};
      margin-bottom: ${theme.spacing(1.5)};
      color: ${theme.colors.text.primary};
    }

    h4 {
      margin-top: ${theme.spacing(2)};
      margin-bottom: ${theme.spacing(1)};
    }

    p {
      margin-bottom: ${theme.spacing(2)};
      line-height: 1.6;
    }

    ul, ol {
      margin-bottom: ${theme.spacing(2)};
      padding-left: ${theme.spacing(3)};
      line-height: 1.8;
    }

    li {
      margin-bottom: ${theme.spacing(0.5)};
    }

    code {
      background: ${theme.colors.background.secondary};
      padding: ${theme.spacing(0.25)} ${theme.spacing(0.75)};
      border-radius: ${theme.shape.radius.default};
      font-family: ${theme.typography.fontFamilyMonospace};
      font-size: 0.9em;
      color: ${theme.colors.primary.text};
    }

    strong {
      font-weight: ${theme.typography.fontWeightMedium};
      color: ${theme.colors.text.primary};
    }

    table {
      width: 100%;
      border-collapse: collapse;
      margin: ${theme.spacing(2)} 0;
    }

    th {
      background: ${theme.colors.background.secondary};
      padding: ${theme.spacing(1)} ${theme.spacing(2)};
      text-align: left;
      border-bottom: 2px solid ${theme.colors.border.medium};
      font-weight: ${theme.typography.fontWeightMedium};
    }

    td {
      padding: ${theme.spacing(1)} ${theme.spacing(2)};
      border-bottom: 1px solid ${theme.colors.border.weak};
    }
  `,
  codeBlock: css`
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    margin: ${theme.spacing(2)} 0;
    overflow-x: auto;
  `,
});
