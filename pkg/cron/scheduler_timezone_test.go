package cron

import (
    "testing"
    "time"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
)

// TestCalculateNextRunTimezone tests timezone-aware next run calculation
func TestCalculateNextRunTimezone(t *testing.T) {
    tests := []struct {
        name            string
        cronExpr        string
        timezone        string
        expectedHour    int // Expected hour in the schedule's timezone
        expectedMinute  int
        validateNextRun func(t *testing.T, nextRun time.Time, tz string)
    }{
        {
            name:           "Daily at midnight in America/New_York",
            cronExpr:       "0 0 * * *",
            timezone:       "America/New_York",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                loc, _ := time.LoadLocation(tz)
                localTime := nextRun.In(loc)
                if localTime.Hour() != 0 || localTime.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
                }
            },
        },
        {
            name:           "Daily at midnight in Europe/London",
            cronExpr:       "0 0 * * *",
            timezone:       "Europe/London",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                loc, _ := time.LoadLocation(tz)
                localTime := nextRun.In(loc)
                if localTime.Hour() != 0 || localTime.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
                }
            },
        },
        {
            name:           "Daily at midnight in Asia/Tokyo",
            cronExpr:       "0 0 * * *",
            timezone:       "Asia/Tokyo",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                loc, _ := time.LoadLocation(tz)
                localTime := nextRun.In(loc)
                if localTime.Hour() != 0 || localTime.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
                }
            },
        },
        {
            name:           "Weekly Monday at midnight in America/Los_Angeles",
            cronExpr:       "0 0 * * 1",
            timezone:       "America/Los_Angeles",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                loc, _ := time.LoadLocation(tz)
                localTime := nextRun.In(loc)
                if localTime.Weekday() != time.Monday {
                    t.Errorf("Expected Monday in %s, got %s", tz, localTime.Weekday())
                }
                if localTime.Hour() != 0 || localTime.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
                }
            },
        },
        {
            name:           "Monthly 1st at midnight in UTC",
            cronExpr:       "0 0 1 * *",
            timezone:       "UTC",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                loc, _ := time.LoadLocation(tz)
                localTime := nextRun.In(loc)
                if localTime.Day() != 1 {
                    t.Errorf("Expected 1st day of month in %s, got day %d", tz, localTime.Day())
                }
                if localTime.Hour() != 0 || localTime.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in %s, got %02d:%02d", tz, localTime.Hour(), localTime.Minute())
                }
            },
        },
        {
            name:           "Invalid timezone falls back to UTC",
            cronExpr:       "0 0 * * *",
            timezone:       "Invalid/Timezone",
            expectedHour:   0,
            expectedMinute: 0,
            validateNextRun: func(t *testing.T, nextRun time.Time, tz string) {
                // Should fall back to UTC when timezone is invalid
                // nextRun is already stored in UTC
                if nextRun.Hour() != 0 || nextRun.Minute() != 0 {
                    t.Errorf("Expected midnight (00:00) in UTC (fallback), got %02d:%02d", nextRun.Hour(), nextRun.Minute())
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(
            tt.name, func(t *testing.T) {
                // Create a mock scheduler (no need for real store/renderer)
                scheduler := &Scheduler{}

                // Create test schedule
                schedule := &model.Schedule{
                    ID:           1,
                    CronExpr:     tt.cronExpr,
                    Timezone:     tt.timezone,
                    IntervalType: "cron",
                }

                // Calculate next run
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
                t.Logf("Next run (UTC): %v", nextRun.UTC().Format("2006-01-02 15:04:05 MST"))
                if loc, err := time.LoadLocation(tt.timezone); err == nil {
                    localTime := nextRun.In(loc)
                    t.Logf("Next run (%s): %v", tt.timezone, localTime.Format("2006-01-02 15:04:05 MST"))
                } else {
                    t.Logf("Next run (%s): [invalid timezone, using UTC]", tt.timezone)
                }
            },
        )
    }
}

// TestCalculateNextRunEnabledDisabled tests that next run respects enabled flag
func TestCalculateNextRunEnabledDisabled(t *testing.T) {
    scheduler := &Scheduler{}

    // Test enabled schedule
    enabledSchedule := &model.Schedule{
        ID:           1,
        CronExpr:     "0 0 * * *",
        Timezone:     "America/New_York",
        IntervalType: "daily",
        Enabled:      true,
    }

    nextRun := scheduler.calculateNextRun(enabledSchedule)
    if nextRun.IsZero() {
        t.Error("Enabled schedule should have a calculated next run")
    }

    // Note: The calculateNextRun function doesn't check the Enabled flag.
    // That's handled in the API handlers (handlers.go).
    // This test just verifies the function always calculates when called.
}

// TestTimezoneAwareCronParsing verifies CRON expressions are parsed in correct timezone
func TestTimezoneAwareCronParsing(t *testing.T) {
    // Set up test at a specific time for reproducibility
    // We're testing that "0 0 * * *" means midnight in the schedule's timezone, not UTC

    scheduler := &Scheduler{}

    schedule := &model.Schedule{
        ID:           1,
        CronExpr:     "0 0 * * *", // Daily at midnight
        Timezone:     "America/New_York",
        IntervalType: "daily",
    }

    nextRun := scheduler.calculateNextRun(schedule)

    // Convert to New York time
    loc, err := time.LoadLocation("America/New_York")
    if err != nil {
        t.Fatalf("Failed to load timezone: %v", err)
    }

    nyTime := nextRun.In(loc)

    // Should be midnight in New York
    if nyTime.Hour() != 0 || nyTime.Minute() != 0 {
        t.Errorf("Expected midnight (00:00) in America/New_York, got %02d:%02d", nyTime.Hour(), nyTime.Minute())
    }

    t.Logf("Next run UTC: %v", nextRun.UTC())
    t.Logf("Next run NY:  %v", nyTime)
}
