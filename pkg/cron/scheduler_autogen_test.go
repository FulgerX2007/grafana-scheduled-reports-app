package cron

import (
	"testing"
	"time"

	"github.com/yourusername/sheduled-reports-app/pkg/model"
)

// TestAutoGenerateCronExpr tests that cron_expr is auto-generated when empty
func TestAutoGenerateCronExpr(t *testing.T) {
	scheduler := &Scheduler{}

	tests := []struct {
		name            string
		intervalType    string
		cronExpr        string // Empty to test auto-generation
		timezone        string
		validateNextRun func(t *testing.T, nextRun time.Time, tz string)
	}{
		{
			name:         "Daily with empty cron_expr auto-generates 0 0 * * *",
			intervalType: "daily",
			cronExpr:     "", // Empty - should auto-generate
			timezone:     "America/New_York",
			validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
				loc, _ := time.LoadLocation(tz)
				localTime := nextRun.In(loc)

				// Should run at midnight, NOT at current time + 24 hours
				if localTime.Hour() != 0 || localTime.Minute() != 0 {
					t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d - auto-generation failed!", tz, localTime.Hour(), localTime.Minute())
				}

				// Should be tomorrow or later
				now := time.Now().In(loc)
				if nextRun.Before(now) {
					t.Errorf("Next run should be in the future, got %v", nextRun)
				}
			},
		},
		{
			name:         "Weekly with empty cron_expr auto-generates 0 0 * * 1",
			intervalType: "weekly",
			cronExpr:     "", // Empty - should auto-generate
			timezone:     "Europe/London",
			validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
				loc, _ := time.LoadLocation(tz)
				localTime := nextRun.In(loc)

				// Should run on Monday at midnight
				if localTime.Weekday() != time.Monday {
					t.Errorf("Expected Monday in %s, got %s", tz, localTime.Weekday())
				}
				if localTime.Hour() != 0 || localTime.Minute() != 0 {
					t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
				}
			},
		},
		{
			name:         "Monthly with empty cron_expr auto-generates 0 0 1 * *",
			intervalType: "monthly",
			cronExpr:     "", // Empty - should auto-generate
			timezone:     "Asia/Tokyo",
			validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
				loc, _ := time.LoadLocation(tz)
				localTime := nextRun.In(loc)

				// Should run on 1st of month at midnight
				if localTime.Day() != 1 {
					t.Errorf("Expected 1st day of month in %s, got day %d", tz, localTime.Day())
				}
				if localTime.Hour() != 0 || localTime.Minute() != 0 {
					t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
				}
			},
		},
		{
			name:         "Unknown interval type defaults to daily",
			intervalType: "unknown",
			cronExpr:     "", // Empty - should auto-generate daily
			timezone:     "UTC",
			validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
				if nextRun.Hour() != 0 || nextRun.Minute() != 0 {
					t.Errorf("Expected midnight (00:00) for unknown interval type, got %02d:%02d", nextRun.Hour(), nextRun.Minute())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &model.Schedule{
				ID:           1,
				CronExpr:     tt.cronExpr, // Empty to test auto-generation
				Timezone:     tt.timezone,
				IntervalType: tt.intervalType,
			}

			// Calculate next run - should auto-generate cron expression
			nextRun := scheduler.calculateNextRun(schedule)

			// Verify next run is in the future
			if !nextRun.After(time.Now()) {
				t.Errorf("Next run %v should be in the future", nextRun)
			}

			// Run custom validation
			if tt.validateNextRun != nil {
				tt.validateNextRun(t, nextRun, tt.timezone)
			}

			// Log for debugging
			loc, _ := time.LoadLocation(tt.timezone)
			localTime := nextRun.In(loc)
			t.Logf("Interval type: %s", tt.intervalType)
			t.Logf("Cron expr (before): '%s' (empty)", tt.cronExpr)
			t.Logf("Next run (UTC): %v", nextRun.UTC().Format("2006-01-02 15:04:05 MST"))
			t.Logf("Next run (%s): %v", tt.timezone, localTime.Format("2006-01-02 15:04:05 MST"))
		})
	}
}

// TestFixForUserIssue specifically tests the reported bug scenario
func TestFixForUserIssue(t *testing.T) {
	// This test simulates the user's reported issue:
	// They ran a schedule at 15/10/2025 22:35:57 and next run was set to 16/10/2025 22:35:57
	// Instead, it should be set to 16/10/2025 00:00:00 (midnight)

	scheduler := &Scheduler{}

	// Simulate a "daily" schedule with empty cron_expr (old schedule before frontend update)
	schedule := &model.Schedule{
		ID:           1,
		CronExpr:     "", // Empty - this is the bug! Old schedules don't have cron_expr
		Timezone:     "UTC",
		IntervalType: "daily",
	}

	// Calculate next run
	nextRun := scheduler.calculateNextRun(schedule)

	// Verify it's at midnight, not current_time + 24 hours
	if nextRun.Hour() != 0 || nextRun.Minute() != 0 || nextRun.Second() != 0 {
		t.Errorf("FAILED: Next run should be at midnight (00:00:00), got %02d:%02d:%02d",
			nextRun.Hour(), nextRun.Minute(), nextRun.Second())
		t.Errorf("This means the auto-generation is not working! Old schedules will still have the bug.")
	} else {
		t.Logf("SUCCESS: Next run is at midnight (00:00:00), auto-generation is working!")
	}

	t.Logf("Next run: %v", nextRun.Format("2006-01-02 15:04:05"))
}
