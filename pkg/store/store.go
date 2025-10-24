package store

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Register SQLite driver
	"github.com/yourusername/scheduled-reports-app/pkg/model"
)

// parseTimestamp parses a timestamp string from SQLite, handling multiple formats
// Formats supported:
// - "2006-01-02 15:04:05" (UTC, no timezone)
// - "2006-01-02 15:04:05 +0300 EEST" (with timezone)
// - "2006-01-02 15:04:05 +0000 UTC" (UTC with explicit timezone)
func parseTimestamp(s string) *time.Time {
	if s == "" {
		return nil
	}

	// Try parsing with various formats
	formats := []string{
		"2006-01-02 15:04:05",           // SQLite standard format (UTC assumed)
		"2006-01-02 15:04:05 -0700 MST", // With timezone offset and name
		"2006-01-02 15:04:05 -0700",     // With timezone offset only
		time.RFC3339,                    // ISO 8601
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	// If all parsing attempts fail, log warning and return nil
	log.Printf("[STORE] WARNING: Failed to parse timestamp: %s", s)
	return nil
}

// Store handles database operations
type Store struct {
	db         *sql.DB
	writeQueue *writeQueue
}

// NewStore creates a new store instance
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure SQLite for concurrent access
	// Enable WAL mode to allow concurrent readers and single writer
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to 5 seconds (retry on lock contention)
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Configure connection pool for SQLite best practices
	db.SetMaxOpenConns(1) // SQLite only supports single writer
	db.SetMaxIdleConns(1)

	log.Println("[STORE] SQLite configured: WAL mode enabled, busy_timeout=5000ms, single writer connection")

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize write queue for serialized write operations
	store.writeQueue = newWriteQueue(store)
	log.Println("[STORE] Write queue initialized for serialized database writes")

	return store, nil
}

// migrate runs database migrations
func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			dashboard_uid TEXT NOT NULL,
			dashboard_title TEXT,
			panel_ids TEXT,
			range_from TEXT NOT NULL,
			range_to TEXT NOT NULL,
			interval_type TEXT NOT NULL,
			cron_expr TEXT,
			timezone TEXT NOT NULL,
			format TEXT NOT NULL DEFAULT 'pdf',
			variables TEXT,
			recipients TEXT NOT NULL,
			email_subject TEXT NOT NULL,
			email_body TEXT NOT NULL,
			template_id INTEGER,
			enabled INTEGER NOT NULL DEFAULT 1,
			last_run_at DATETIME,
			next_run_at DATETIME,
			owner_user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_org_id ON schedules(org_id)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_next_run_at ON schedules(next_run_at)`,
		`CREATE TABLE IF NOT EXISTS runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id INTEGER NOT NULL,
			org_id INTEGER NOT NULL,
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			status TEXT NOT NULL,
			error_text TEXT,
			artifact_path TEXT,
			rendered_pages INTEGER NOT NULL DEFAULT 0,
			bytes INTEGER NOT NULL DEFAULT 0,
			checksum TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runs_schedule_id ON runs(schedule_id)`,
		`CREATE INDEX IF NOT EXISTS idx_runs_org_id ON runs(org_id)`,
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			config TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_org_id ON templates(org_id)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id INTEGER NOT NULL UNIQUE,
			smtp_config TEXT,
			renderer_config TEXT NOT NULL,
			limits TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// Migration: Populate cron_expr for existing schedules based on interval_type
		`UPDATE schedules
		 SET cron_expr = CASE
			WHEN interval_type = 'daily' THEN '0 0 * * *'
			WHEN interval_type = 'weekly' THEN '0 0 * * 1'
			WHEN interval_type = 'monthly' THEN '0 0 1 * *'
			ELSE '0 0 * * *'
		 END
		 WHERE cron_expr IS NULL OR cron_expr = ''`,
		// Migration: Add email_sent field to track email delivery status
		`ALTER TABLE runs ADD COLUMN email_sent INTEGER NOT NULL DEFAULT 0`,
		// Migration: Add email_error field to store email sending errors
		`ALTER TABLE runs ADD COLUMN email_error TEXT`,
		// Migration: Add artifact_data BLOB field to store PDF content directly in database
		// This replaces filesystem storage to comply with Grafana catalog requirements
		`ALTER TABLE runs ADD COLUMN artifact_data BLOB`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			// Ignore "duplicate column" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration failed: %w", err)
			}
			log.Printf("[STORE] Migration warning (ignored): %v", err)
		}
	}

	return nil
}

// CreateSchedule creates a new schedule (queued for serialized execution)
func (s *Store) CreateSchedule(schedule *model.Schedule) error {
	return s.writeQueue.enqueue(opCreateSchedule, schedule)
}

// createScheduleDirect creates a new schedule (direct database access, called by write queue)
func (s *Store) createScheduleDirect(schedule *model.Schedule) error {
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	// Format next_run_at for SQLite compatibility (without timezone suffix)
	var nextRunAtStr interface{}
	if schedule.NextRunAt != nil {
		// Format as SQLite-compatible datetime: "2006-01-02 15:04:05" in UTC
		nextRunAtStr = schedule.NextRunAt.UTC().Format("2006-01-02 15:04:05")
	}

	// Try INSERT with format field first (for backward compatibility with old databases)
	result, err := s.db.Exec(`
		INSERT INTO schedules (
			org_id, name, dashboard_uid, dashboard_title, panel_ids, range_from, range_to,
			interval_type, cron_expr, timezone, format, variables, recipients,
			email_subject, email_body, template_id, enabled, owner_user_id,
			next_run_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		schedule.OrgID, schedule.Name, schedule.DashboardUID, schedule.DashboardTitle,
		schedule.PanelIDs, schedule.RangeFrom, schedule.RangeTo, schedule.IntervalType,
		schedule.CronExpr, schedule.Timezone, "pdf", schedule.Variables,
		schedule.Recipients, schedule.EmailSubject, schedule.EmailBody, schedule.TemplateID,
		schedule.Enabled, schedule.OwnerUserID, nextRunAtStr, now, now,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	schedule.ID = id

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *Store) GetSchedule(orgID, id int64) (*model.Schedule, error) {
	schedule := &model.Schedule{}
	var format string // Backward compatibility - format field removed from model but may exist in old databases
	var lastRunAtStr, nextRunAtStr sql.NullString

	err := s.db.QueryRow(`
		SELECT id, org_id, name, dashboard_uid, dashboard_title, panel_ids, range_from, range_to,
		       interval_type, cron_expr, timezone, format, variables, recipients,
		       email_subject, email_body, template_id, enabled, last_run_at, next_run_at,
		       owner_user_id, created_at, updated_at
		FROM schedules WHERE id = ? AND org_id = ?`,
		id, orgID,
	).Scan(
		&schedule.ID, &schedule.OrgID, &schedule.Name, &schedule.DashboardUID,
		&schedule.DashboardTitle, &schedule.PanelIDs, &schedule.RangeFrom, &schedule.RangeTo,
		&schedule.IntervalType, &schedule.CronExpr, &schedule.Timezone, &format,
		&schedule.Variables, &schedule.Recipients, &schedule.EmailSubject, &schedule.EmailBody,
		&schedule.TemplateID, &schedule.Enabled, &lastRunAtStr, &nextRunAtStr,
		&schedule.OwnerUserID, &schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, err
	}

	// Parse timestamp strings to time.Time, handling multiple formats
	if lastRunAtStr.Valid {
		schedule.LastRunAt = parseTimestamp(lastRunAtStr.String)
	}
	if nextRunAtStr.Valid {
		schedule.NextRunAt = parseTimestamp(nextRunAtStr.String)
	}

	return schedule, nil
}

// ListSchedules retrieves all schedules for an organization
func (s *Store) ListSchedules(orgID int64) ([]*model.Schedule, error) {
	rows, err := s.db.Query(`
		SELECT id, org_id, name, dashboard_uid, dashboard_title, panel_ids, range_from, range_to,
		       interval_type, cron_expr, timezone, format, variables, recipients,
		       email_subject, email_body, template_id, enabled, last_run_at, next_run_at,
		       owner_user_id, created_at, updated_at
		FROM schedules WHERE org_id = ? ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]*model.Schedule, 0)
	for rows.Next() {
		schedule := &model.Schedule{}
		var format string // Backward compatibility - format field removed from model but may exist in old databases
		var lastRunAtStr, nextRunAtStr sql.NullString

		err := rows.Scan(
			&schedule.ID, &schedule.OrgID, &schedule.Name, &schedule.DashboardUID,
			&schedule.DashboardTitle, &schedule.PanelIDs, &schedule.RangeFrom, &schedule.RangeTo,
			&schedule.IntervalType, &schedule.CronExpr, &schedule.Timezone, &format,
			&schedule.Variables, &schedule.Recipients, &schedule.EmailSubject, &schedule.EmailBody,
			&schedule.TemplateID, &schedule.Enabled, &lastRunAtStr, &nextRunAtStr,
			&schedule.OwnerUserID, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse timestamp strings to time.Time
		if lastRunAtStr.Valid {
			schedule.LastRunAt = parseTimestamp(lastRunAtStr.String)
		}
		if nextRunAtStr.Valid {
			schedule.NextRunAt = parseTimestamp(nextRunAtStr.String)
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// UpdateSchedule updates an existing schedule (queued for serialized execution)
func (s *Store) UpdateSchedule(schedule *model.Schedule) error {
	return s.writeQueue.enqueue(opUpdateSchedule, schedule)
}

// updateScheduleDirect updates an existing schedule (direct database access, called by write queue)
func (s *Store) updateScheduleDirect(schedule *model.Schedule) error {
	schedule.UpdatedAt = time.Now()

	// Format timestamps for SQLite compatibility (without timezone suffix)
	var lastRunAtStr, nextRunAtStr interface{}
	if schedule.LastRunAt != nil {
		lastRunAtStr = schedule.LastRunAt.UTC().Format("2006-01-02 15:04:05")
	}
	if schedule.NextRunAt != nil {
		nextRunAtStr = schedule.NextRunAt.UTC().Format("2006-01-02 15:04:05")
	}

	// Include format field for backward compatibility with old databases (always set to 'pdf')
	_, err := s.db.Exec(`
		UPDATE schedules SET
			name = ?, dashboard_uid = ?, dashboard_title = ?, panel_ids = ?,
			range_from = ?, range_to = ?, interval_type = ?, cron_expr = ?,
			timezone = ?, format = ?, variables = ?, recipients = ?,
			email_subject = ?, email_body = ?, template_id = ?, enabled = ?,
			last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ? AND org_id = ?`,
		schedule.Name, schedule.DashboardUID, schedule.DashboardTitle, schedule.PanelIDs,
		schedule.RangeFrom, schedule.RangeTo, schedule.IntervalType, schedule.CronExpr,
		schedule.Timezone, "pdf", schedule.Variables, schedule.Recipients,
		schedule.EmailSubject, schedule.EmailBody, schedule.TemplateID, schedule.Enabled,
		lastRunAtStr, nextRunAtStr, schedule.UpdatedAt, schedule.ID, schedule.OrgID,
	)
	return err
}

// DeleteSchedule deletes a schedule (queued for serialized execution)
func (s *Store) DeleteSchedule(orgID, id int64) error {
	return s.writeQueue.enqueue(opDeleteSchedule, deleteScheduleParams{orgID: orgID, id: id})
}

// deleteScheduleDirect deletes a schedule (direct database access, called by write queue)
func (s *Store) deleteScheduleDirect(orgID, id int64) error {
	_, err := s.db.Exec("DELETE FROM schedules WHERE id = ? AND org_id = ?", id, orgID)
	return err
}

// CreateRun creates a new run record (queued for serialized execution)
func (s *Store) CreateRun(run *model.Run) error {
	return s.writeQueue.enqueue(opCreateRun, run)
}

// createRunDirect creates a new run record (direct database access, called by write queue)
func (s *Store) createRunDirect(run *model.Run) error {
	run.CreatedAt = time.Now()

	result, err := s.db.Exec(`
		INSERT INTO runs (schedule_id, org_id, started_at, status, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		run.ScheduleID, run.OrgID, run.StartedAt, run.Status, run.CreatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	run.ID = id

	return nil
}

// UpdateRun updates a run record (queued for serialized execution)
func (s *Store) UpdateRun(run *model.Run) error {
	return s.writeQueue.enqueue(opUpdateRun, run)
}

// updateRunDirect updates a run record (direct database access, called by write queue)
func (s *Store) updateRunDirect(run *model.Run) error {
	_, err := s.db.Exec(`
		UPDATE runs SET
			finished_at = ?, status = ?, error_text = ?, artifact_path = ?, artifact_data = ?,
			rendered_pages = ?, bytes = ?, checksum = ?, email_sent = ?, email_error = ?
		WHERE id = ?`,
		run.FinishedAt, run.Status, run.ErrorText, run.ArtifactPath, run.ArtifactData,
		run.RenderedPages, run.Bytes, run.Checksum, run.EmailSent, run.EmailError, run.ID,
	)
	return err
}

// GetRun retrieves a run by ID
func (s *Store) GetRun(orgID, id int64) (*model.Run, error) {
	run := &model.Run{}
	var finishedAt sql.NullTime
	var errorText, artifactPath, checksum, emailError sql.NullString
	var artifactData []byte

	err := s.db.QueryRow(`
		SELECT id, schedule_id, org_id, started_at, finished_at, status, error_text,
		       artifact_path, artifact_data, rendered_pages, bytes, checksum, email_sent, email_error, created_at
		FROM runs WHERE id = ? AND org_id = ?`,
		id, orgID,
	).Scan(
		&run.ID, &run.ScheduleID, &run.OrgID, &run.StartedAt, &finishedAt,
		&run.Status, &errorText, &artifactPath, &artifactData, &run.RenderedPages,
		&run.Bytes, &checksum, &run.EmailSent, &emailError, &run.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found")
	}
	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if errorText.Valid {
		run.ErrorText = errorText.String
	}
	if artifactPath.Valid {
		run.ArtifactPath = artifactPath.String
	}
	if len(artifactData) > 0 {
		run.ArtifactData = artifactData
	}
	if checksum.Valid {
		run.Checksum = checksum.String
	}
	if emailError.Valid {
		run.EmailError = emailError.String
	}

	return run, nil
}

// ListRuns retrieves runs for a schedule
func (s *Store) ListRuns(orgID, scheduleID int64) ([]*model.Run, error) {
	rows, err := s.db.Query(`
		SELECT id, schedule_id, org_id, started_at, finished_at, status, error_text,
		       artifact_path, rendered_pages, bytes, checksum, email_sent, email_error, created_at
		FROM runs WHERE schedule_id = ? AND org_id = ? ORDER BY started_at DESC LIMIT 50`,
		scheduleID, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]*model.Run, 0)
	for rows.Next() {
		run := &model.Run{}
		var finishedAt sql.NullTime
		var errorText, artifactPath, checksum, emailError sql.NullString

		err := rows.Scan(
			&run.ID, &run.ScheduleID, &run.OrgID, &run.StartedAt, &finishedAt,
			&run.Status, &errorText, &artifactPath, &run.RenderedPages,
			&run.Bytes, &checksum, &run.EmailSent, &emailError, &run.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert nullable fields
		if finishedAt.Valid {
			run.FinishedAt = &finishedAt.Time
		}
		if errorText.Valid {
			run.ErrorText = errorText.String
		}
		if artifactPath.Valid {
			run.ArtifactPath = artifactPath.String
		}
		if checksum.Valid {
			run.Checksum = checksum.String
		}
		if emailError.Valid {
			run.EmailError = emailError.String
		}

		runs = append(runs, run)
	}

	return runs, nil
}

// GetSettings retrieves settings for an organization
func (s *Store) GetSettings(orgID int64) (*model.Settings, error) {
	settings := &model.Settings{}
	err := s.db.QueryRow(`
		SELECT id, org_id, smtp_config, renderer_config, limits, created_at, updated_at
		FROM settings WHERE org_id = ?`,
		orgID,
	).Scan(
		&settings.ID, &settings.OrgID, &settings.SMTPConfig,
		&settings.RendererConfig, &settings.Limits, &settings.CreatedAt, &settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return settings, err
}

// UpsertSettings creates or updates settings (queued for serialized execution)
func (s *Store) UpsertSettings(settings *model.Settings) error {
	return s.writeQueue.enqueue(opUpsertSettings, settings)
}

// upsertSettingsDirect creates or updates settings (direct database access, called by write queue)
func (s *Store) upsertSettingsDirect(settings *model.Settings) error {
	now := time.Now()
	settings.UpdatedAt = now

	existing, err := s.GetSettings(settings.OrgID)
	if err != nil {
		return err
	}

	if existing == nil {
		settings.CreatedAt = now
		result, err := s.db.Exec(`
			INSERT INTO settings (org_id, smtp_config, renderer_config, limits, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			settings.OrgID, settings.SMTPConfig, settings.RendererConfig,
			settings.Limits, settings.CreatedAt, settings.UpdatedAt,
		)
		if err != nil {
			return err
		}
		id, _ := result.LastInsertId()
		settings.ID = id
	} else {
		_, err := s.db.Exec(`
			UPDATE settings SET
				smtp_config = ?, renderer_config = ?, limits = ?, updated_at = ?
			WHERE org_id = ?`,
			settings.SMTPConfig, settings.RendererConfig,
			settings.Limits, settings.UpdatedAt, settings.OrgID,
		)
		return err
	}

	return nil
}

// GetDueSchedules retrieves schedules that are due to run
func (s *Store) GetDueSchedules() ([]*model.Schedule, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	// Debug: log the query parameters
	log.Printf("[STORE] GetDueSchedules: current time = %s", now)

	rows, err := s.db.Query(`
		SELECT id, org_id, name, dashboard_uid, dashboard_title, panel_ids, range_from, range_to,
		       interval_type, cron_expr, timezone, format, variables, recipients,
		       email_subject, email_body, template_id, enabled, last_run_at, next_run_at,
		       owner_user_id, created_at, updated_at
		FROM schedules
		WHERE enabled = 1 AND (next_run_at IS NULL OR datetime(next_run_at) <= datetime(?))
		ORDER BY next_run_at ASC`,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]*model.Schedule, 0)
	for rows.Next() {
		schedule := &model.Schedule{}
		var format string // Backward compatibility - format field removed from model but may exist in old databases
		var lastRunAtStr, nextRunAtStr sql.NullString

		err := rows.Scan(
			&schedule.ID, &schedule.OrgID, &schedule.Name, &schedule.DashboardUID,
			&schedule.DashboardTitle, &schedule.PanelIDs, &schedule.RangeFrom, &schedule.RangeTo,
			&schedule.IntervalType, &schedule.CronExpr, &schedule.Timezone, &format,
			&schedule.Variables, &schedule.Recipients, &schedule.EmailSubject, &schedule.EmailBody,
			&schedule.TemplateID, &schedule.Enabled, &lastRunAtStr, &nextRunAtStr,
			&schedule.OwnerUserID, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			log.Printf("[STORE] ERROR: Failed to scan schedule row: %v", err)
			return nil, err
		}

		// Parse timestamp strings to time.Time
		if lastRunAtStr.Valid {
			schedule.LastRunAt = parseTimestamp(lastRunAtStr.String)
		}
		if nextRunAtStr.Valid {
			schedule.NextRunAt = parseTimestamp(nextRunAtStr.String)
		}

		log.Printf("[STORE] Found due schedule: ID=%d, Name='%s', NextRunAt=%v", schedule.ID, schedule.Name, schedule.NextRunAt)
		schedules = append(schedules, schedule)
	}

	log.Printf("[STORE] GetDueSchedules: returning %d schedule(s)", len(schedules))
	return schedules, nil
}

// Close closes the database connection and shuts down the write queue
func (s *Store) Close() error {
	// Shutdown write queue first to ensure all pending writes complete
	if s.writeQueue != nil {
		s.writeQueue.shutdown()
	}
	return s.db.Close()
}
