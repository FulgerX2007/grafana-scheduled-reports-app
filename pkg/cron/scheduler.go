package cron

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/robfig/cron/v3"
	"github.com/yourusername/scheduled-reports-app/pkg/mail"
	"github.com/yourusername/scheduled-reports-app/pkg/model"
	"github.com/yourusername/scheduled-reports-app/pkg/render"
	"github.com/yourusername/scheduled-reports-app/pkg/store"
)

// Scheduler handles report scheduling
type Scheduler struct {
	store         *store.Store
	cron          *cron.Cron
	grafanaURL    string
	artifactsPath string
	workerPool    chan struct{}
	baseCtx       context.Context           // Context with Grafana config for background jobs
	renderers     map[int64]render.Backend  // Per-org renderer instances for browser reuse
	settingsCache map[int64]*model.Settings // Per-org settings cache to reduce DB reads
	cacheMutex    sync.RWMutex              // Protects settingsCache
}

// NewScheduler creates a new scheduler instance
func NewScheduler(st *store.Store, grafanaURL, artifactsPath string, maxConcurrent int) *Scheduler {
	return &Scheduler{
		store:         st,
		cron:          cron.New(cron.WithSeconds()),
		grafanaURL:    grafanaURL,
		artifactsPath: artifactsPath,
		workerPool:    make(chan struct{}, maxConcurrent),
		baseCtx:       context.Background(), // Will be updated when plugin starts
		renderers:     make(map[int64]render.Backend),
		settingsCache: make(map[int64]*model.Settings),
	}
}

// SetContext sets the base context for the scheduler (should be called on plugin initialization)
func (s *Scheduler) SetContext(ctx context.Context) {
	s.baseCtx = ctx
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	// Add a job that runs every minute to check for due schedules
	cronExpr := "0 * * * * *" // Every minute at second 0
	entryID, err := s.cron.AddFunc(cronExpr, s.checkDueSchedules)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.cron.Start()
	log.Printf("Scheduler started with cron expression '%s' (entry ID: %d)", cronExpr, entryID)
	log.Printf("Scheduler will check for due schedules every minute")
	log.Printf("Current time: %s", time.Now().Format(time.RFC3339))

	return nil
}

// Stop stops the scheduler and cleans up browser instances
func (s *Scheduler) Stop() {
	s.cron.Stop()

	// Close all browser instances
	for orgID, renderer := range s.renderers {
		if err := renderer.Close(); err != nil {
			log.Printf("Failed to close renderer for org %d: %v", orgID, err)
		}
	}

	log.Println("Scheduler stopped and browsers closed")
}

// getCachedSettings retrieves settings for an organization, using cache when possible
func (s *Scheduler) getCachedSettings(orgID int64) (*model.Settings, error) {
	// Try to read from cache first (read lock)
	s.cacheMutex.RLock()
	cached, exists := s.settingsCache[orgID]
	s.cacheMutex.RUnlock()

	if exists {
		log.Printf("[CACHE] Using cached settings for org %d", orgID)
		return cached, nil
	}

	// Cache miss - fetch from database (write lock for cache update)
	log.Printf("[CACHE] Settings cache miss for org %d, fetching from database", orgID)
	settings, err := s.store.GetSettings(orgID)
	if err != nil {
		return nil, err
	}

	if settings != nil {
		// Store in cache for future use
		s.cacheMutex.Lock()
		s.settingsCache[orgID] = settings
		s.cacheMutex.Unlock()
		log.Printf("[CACHE] Cached settings for org %d", orgID)
	}

	return settings, nil
}

// checkDueSchedules checks for schedules that are due and executes them
func (s *Scheduler) checkDueSchedules() {
	log.Printf("[CRON] Checking for due schedules at %s", time.Now().Format(time.RFC3339))

	schedules, err := s.store.GetDueSchedules()
	if err != nil {
		log.Printf("[CRON] ERROR: Failed to get due schedules: %v", err)
		return
	}

	if len(schedules) == 0 {
		log.Printf("[CRON] No due schedules found")
		return
	}

	log.Printf("[CRON] Found %d due schedule(s)", len(schedules))
	for _, schedule := range schedules {
		log.Printf("[CRON] Processing schedule ID=%d, Name='%s', NextRunAt=%v",
			schedule.ID, schedule.Name, schedule.NextRunAt)

		// Update next run time immediately to prevent duplicate execution
		nextRun := s.calculateNextRun(schedule)
		schedule.NextRunAt = &nextRun
		log.Printf("[CRON] Updated schedule ID=%d next run to: %s", schedule.ID, nextRun.Format(time.RFC3339))

		if err := s.store.UpdateSchedule(schedule); err != nil {
			log.Printf("[CRON] ERROR: Failed to update schedule %d next run time: %v", schedule.ID, err)
			continue
		}

		// Execute in worker pool
		log.Printf("[CRON] Triggering execution for schedule ID=%d", schedule.ID)
		go s.executeSchedule(schedule)
	}
}

// ExecuteSchedule executes a schedule immediately (for manual runs)
func (s *Scheduler) ExecuteSchedule(schedule *model.Schedule) {
	go s.executeSchedule(schedule)
}

// executeSchedule executes a single schedule
func (s *Scheduler) executeSchedule(schedule *model.Schedule) {
	log.Printf("[EXECUTE] Starting execution for schedule ID=%d, Name='%s'", schedule.ID, schedule.Name)

	// Acquire worker slot
	s.workerPool <- struct{}{}
	defer func() { <-s.workerPool }()

	log.Printf("[EXECUTE] Acquired worker slot for schedule ID=%d", schedule.ID)

	// Create run record
	run := &model.Run{
		ScheduleID: schedule.ID,
		OrgID:      schedule.OrgID,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	if err := s.store.CreateRun(run); err != nil {
		log.Printf("[EXECUTE] ERROR: Failed to create run record for schedule ID=%d: %v", schedule.ID, err)
		return
	}

	log.Printf("[EXECUTE] Created run record ID=%d for schedule ID=%d", run.ID, schedule.ID)

	// Execute with retries
	err := s.executeWithRetry(schedule, run, 3)

	// Update run record
	now := time.Now()
	run.FinishedAt = &now

	if err != nil {
		run.Status = "failed"
		run.ErrorText = err.Error()
		log.Printf("Schedule %d execution failed: %v", schedule.ID, err)
	} else {
		run.Status = "completed"
	}

	if err := s.store.UpdateRun(run); err != nil {
		log.Printf("Failed to update run record: %v", err)
	}

	// Update schedule last run time
	schedule.LastRunAt = &run.StartedAt
	if err := s.store.UpdateSchedule(schedule); err != nil {
		log.Printf("Failed to update schedule last run time: %v", err)
	}
}

// executeWithRetry executes a schedule with retry logic
func (s *Scheduler) executeWithRetry(schedule *model.Schedule, run *model.Run, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			log.Printf("Retrying schedule %d (attempt %d/%d) after %v", schedule.ID, attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		err := s.executeScheduleOnce(schedule, run)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Schedule %d execution attempt %d failed: %v", schedule.ID, attempt+1, err)
	}

	return fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

// executeScheduleOnce executes a schedule once
func (s *Scheduler) executeScheduleOnce(schedule *model.Schedule, run *model.Run) error {
	// Use the base context which has Grafana config
	ctx := s.baseCtx

	// Get settings (from cache to reduce DB reads)
	settings, err := s.getCachedSettings(schedule.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("no settings configured for org %d", schedule.OrgID)
	}

	// Use configured Grafana URL from settings, fall back to scheduler default
	grafanaURL := s.grafanaURL
	if settings.RendererConfig.GrafanaURL != "" {
		grafanaURL = settings.RendererConfig.GrafanaURL
		log.Printf("DEBUG: Using configured Grafana URL from settings: %s", grafanaURL)
	} else {
		log.Printf("DEBUG: Using default Grafana URL: %s", grafanaURL)
	}

	log.Printf("DEBUG: Rendering with grafanaURL=%s using Chromium backend (managed service account)", grafanaURL)

	// Get or create renderer for this org (reuse renderer instance)
	renderer, exists := s.renderers[schedule.OrgID]
	if !exists {
		// Create new renderer
		var err error
		renderer, err = render.NewBackend(grafanaURL, settings.RendererConfig)
		if err != nil {
			return fmt.Errorf("failed to create renderer: %w", err)
		}
		s.renderers[schedule.OrgID] = renderer
		log.Printf("Created new Chromium renderer for org %d with URL %s", schedule.OrgID, grafanaURL)
	}

	// Render dashboard (token will be retrieved from context inside renderer)
	renderedData, err := renderer.RenderDashboard(ctx, schedule)
	if err != nil {
		return fmt.Errorf("failed to render dashboard: %w", err)
	}

	run.RenderedPages = 1

	// Generate PDF (always PDF format)
	reportData := renderedData
	filename := fmt.Sprintf("%s-%s.pdf", schedule.Name, time.Now().Format("2006-01-02-150405"))
	log.Printf("DEBUG: Using PDF directly from Chromium backend (%d bytes)", len(reportData))

	run.Bytes = int64(len(reportData))

	// Calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(reportData))
	run.Checksum = checksum

	// Save artifact directly to database as BLOB
	run.ArtifactData = reportData
	log.Printf("Report saved to database (%d bytes, checksum=%s)", len(reportData), checksum)

	// Update run record with artifact data BEFORE sending email
	// This makes the download button available immediately in the UI
	if err := s.store.UpdateRun(run); err != nil {
		log.Printf("WARNING: Failed to update run record with artifact data: %v", err)
		// Continue anyway - we'll try to update again after email attempt
	} else {
		log.Printf("Run record updated with artifact data (download now available in UI)")
	}

	// Send email (optional - report is already saved to database)
	if settings.SMTPConfig == nil {
		log.Printf("SMTP not configured for org %d - report saved to database (available for download)", schedule.OrgID)
		run.EmailSent = false
		run.EmailError = "SMTP not configured"
		// Update run with email status
		if err := s.store.UpdateRun(run); err != nil {
			log.Printf("WARNING: Failed to update run record with email status: %v", err)
		}
		return nil // Report generation succeeded, email delivery is optional
	}

	smtpConfig := *settings.SMTPConfig
	mailer := mail.NewMailer(smtpConfig)

	// Interpolate template variables
	vars := map[string]string{
		"schedule.name":   schedule.Name,
		"dashboard.title": schedule.DashboardTitle,
		"timerange":       fmt.Sprintf("%s to %s", schedule.RangeFrom, schedule.RangeTo),
		"run.started_at":  run.StartedAt.Format(time.RFC1123),
	}

	subject := mail.InterpolateTemplate(schedule.EmailSubject, vars)
	body := mail.InterpolateTemplate(schedule.EmailBody, vars)

	// Try to send email, but don't fail the entire run if it fails
	log.Printf("Attempting to send email for schedule %d to %d recipient(s)...", schedule.ID, len(schedule.Recipients.To))
	if err := mailer.SendReport(schedule.Recipients, subject, body, reportData, filename); err != nil {
		log.Printf("Failed to send email for schedule %d: %v - report saved to %s (available for download)", schedule.ID, err, artifactPath)
		run.EmailSent = false
		run.EmailError = err.Error()
		// Update run with email failure status
		if err := s.store.UpdateRun(run); err != nil {
			log.Printf("WARNING: Failed to update run record with email error: %v", err)
		}
		return nil // Report generation succeeded, email delivery failed but report is available
	}

	log.Printf("Email sent successfully for schedule %d to %d recipient(s)", schedule.ID, len(schedule.Recipients.To))
	run.EmailSent = true
	run.EmailError = "" // Clear any previous error
	// Update run with email success status
	if err := s.store.UpdateRun(run); err != nil {
		log.Printf("WARNING: Failed to update run record with email success: %v", err)
	}
	return nil
}

// CalculateNextRun calculates the next run time for a schedule (exported for use in handlers)
func (s *Scheduler) CalculateNextRun(schedule *model.Schedule) time.Time {
	return s.calculateNextRun(schedule)
}

// calculateNextRun calculates the next run time for a schedule
func (s *Scheduler) calculateNextRun(schedule *model.Schedule) time.Time {
	// Load the schedule's timezone (default to UTC if not set or invalid)
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		log.Printf("Failed to load timezone %s for schedule %d: %v, using UTC", schedule.Timezone, schedule.ID, err)
		loc = time.UTC
	}

	// Get current time in the schedule's timezone
	now := time.Now().In(loc)

	// Auto-generate cron expression from interval_type if not set
	cronExpression := schedule.CronExpr
	if cronExpression == "" {
		switch schedule.IntervalType {
		case "daily":
			cronExpression = "0 0 * * *" // Every day at midnight
		case "weekly":
			cronExpression = "0 0 * * 1" // Every Monday at midnight
		case "monthly":
			cronExpression = "0 0 1 * *" // First day of month at midnight
		default:
			// Unknown interval type, default to daily
			cronExpression = "0 0 * * *"
		}
		log.Printf("Auto-generated cron expression '%s' for schedule %d (interval_type: %s)", cronExpression, schedule.ID, schedule.IntervalType)
	}

	// Parse cron expression using gorhill/cronexpr
	expr, err := cronexpr.Parse(cronExpression)
	if err != nil {
		log.Printf("Failed to parse cron expression '%s' for schedule %d: %v, falling back to 1 hour", cronExpression, schedule.ID, err)
		nextRun := now.Add(1 * time.Hour)
		return nextRun.UTC().Truncate(time.Second)
	}

	// Calculate next run in the schedule's timezone
	nextRun := expr.Next(now)

	// Convert to UTC for storage (SQLite stores timestamps in UTC)
	// Strip monotonic clock reading by truncating to second precision
	return nextRun.UTC().Truncate(time.Second)
}

// ClearRendererCache closes and removes renderer instances for the given org ID
// This forces new renderers to be created with updated settings on next render
func (s *Scheduler) ClearRendererCache(orgID int64) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Close existing renderer if it exists
	if renderer, exists := s.renderers[orgID]; exists {
		if err := renderer.Close(); err != nil {
			log.Printf("Warning: Failed to close renderer for org %d: %v", orgID, err)
		}
		delete(s.renderers, orgID)
		log.Printf("Cleared renderer cache for org %d", orgID)
	}

	// Also clear settings cache to force reload
	delete(s.settingsCache, orgID)
	log.Printf("Cleared settings cache for org %d", orgID)

	return nil
}
