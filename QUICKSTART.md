# Quick Start Guide

Get the Grafana Reporting Plugin running in 5 minutes.

## Step 1: Build the Plugin

```bash
# Install dependencies
make install

# Build frontend and backend
make build
```

This creates:
- Frontend bundle in `dist/`
- Backend binary at `dist/gpx_reporting`

## Step 2: Start Grafana with Docker

```bash
# Start Grafana and Image Renderer
docker-compose up -d

# Check logs
docker-compose logs -f grafana
```

Access Grafana at http://localhost:3000 (admin/admin)

## Step 3: Enable the Plugin

1. Go to: **Administration → Plugins**
2. Search for "Scheduled Reports"
3. Click on it
4. Click **Enable**

> **Note**: The plugin uses Grafana's managed service accounts (Grafana 10.3+). Authentication is handled automatically - no manual token configuration required.

## Step 4: Create Your First Report

1. Go to: **Apps → Reporting**
2. Click **New Schedule**
3. Fill in:
   - **Name**: "Weekly Dashboard Report"
   - **Dashboard**: Select any dashboard
   - **Format**: PDF
   - **Time Range**: From: `now-7d`, To: `now`
   - **Schedule**: Select "Weekly"
   - **Recipients → To**: Your email address
   - **Subject**: `Weekly Report: {{dashboard.title}}`
4. Click **Create**

## Step 5: Configure SMTP (Required for Email)

1. In the plugin, click **Settings**
2. Uncheck "Use Grafana SMTP" (unless you've configured Grafana's SMTP)
3. Fill in your SMTP details:
   ```
   Host: smtp.gmail.com
   Port: 587
   Username: your-email@gmail.com
   Password: your-app-password
   From: your-email@gmail.com
   Use TLS: ✓
   ```
4. Click **Save Settings**

> **Note**: For Gmail, you need an [App Password](https://support.google.com/accounts/answer/185833)

## Step 6: Test the Report

1. Go back to **Apps → Reporting**
2. Find your schedule
3. Click the **▶️ (play)** button
4. Click **🕐 (history)** to see the run status
5. Check your email!

## What's Next?

### Advanced Scheduling

Use cron expressions for custom schedules:
- Every Monday at 8 AM: `0 8 * * 1`
- Every hour: `0 * * * *`
- Every weekday at 9 AM: `0 9 * * 1-5`

### Dashboard Variables

Add dashboard variable values in the schedule:
- Key: `env`, Value: `production`
- Key: `region`, Value: `us-east-1`

### Templates

Customize email content with placeholders:
- `{{schedule.name}}` - Schedule name
- `{{dashboard.title}}` - Dashboard title
- `{{timerange}}` - Time range
- `{{run.started_at}}` - Execution time

### Troubleshooting

**Rendering fails?**
- Verify the managed service account has proper dashboard permissions
- Try increasing render timeout in Settings
- Verify image renderer is running: `curl http://localhost:8081`

**Email not sending?**
- Verify SMTP credentials
- Check firewall/network settings
- Test SMTP connection separately

**Schedule not running?**
- Ensure schedule is enabled
- Check next run time is in the future
- Review backend logs: `docker-compose logs grafana`

## Need Help?

- Full docs: See [README.md](./README.md)
- Development guide: See [CLAUDE.md](./CLAUDE.md)
- Architecture: See [PLAN.md](./PLAN.md)
