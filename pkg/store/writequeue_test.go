package store

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/yourusername/sheduled-reports-app/pkg/model"
)

// TestConcurrentWrites tests that multiple concurrent write operations don't cause SQLITE_BUSY errors
func TestConcurrentWrites(t *testing.T) {
	// Create temporary database
	dbPath := "test_concurrent.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Number of concurrent operations
	numSchedules := 10
	numRuns := 5

	var wg sync.WaitGroup
	errChan := make(chan error, numSchedules*(1+numRuns*2)) // 1 create + N run creates/updates

	// Create multiple schedules concurrently
	for i := 0; i < numSchedules; i++ {
		wg.Add(1)
		go func(scheduleNum int) {
			defer wg.Done()

			// Create schedule
			nextRun := time.Now().Add(1 * time.Hour)
			schedule := &model.Schedule{
				OrgID:          1,
				Name:           "Test Schedule",
				DashboardUID:   "test-dashboard",
				DashboardTitle: "Test Dashboard",
				RangeFrom:      "now-1h",
				RangeTo:        "now",
				IntervalType:   "daily",
				Timezone:       "UTC",
				Recipients:     model.Recipients{To: []string{"test@example.com"}},
				EmailSubject:   "Test Report",
				EmailBody:      "Test Body",
				Enabled:        true,
				NextRunAt:      &nextRun,
				OwnerUserID:    1,
			}

			if err := store.CreateSchedule(schedule); err != nil {
				errChan <- err
				return
			}

			// Create multiple runs for this schedule concurrently
			for j := 0; j < numRuns; j++ {
				wg.Add(1)
				go func(runNum int) {
					defer wg.Done()

					// Create run
					run := &model.Run{
						ScheduleID: schedule.ID,
						OrgID:      schedule.OrgID,
						StartedAt:  time.Now(),
						Status:     "running",
					}

					if err := store.CreateRun(run); err != nil {
						errChan <- err
						return
					}

					// Update run
					finishedAt := time.Now()
					run.FinishedAt = &finishedAt
					run.Status = "completed"
					run.ArtifactPath = "/tmp/test.pdf"
					run.RenderedPages = 1
					run.Bytes = 1024
					run.Checksum = "abc123"

					if err := store.UpdateRun(run); err != nil {
						errChan <- err
						return
					}
				}(j)
			}

			// Update schedule's last run time
			lastRun := time.Now()
			schedule.LastRunAt = &lastRun
			if err := store.UpdateSchedule(schedule); err != nil {
				errChan <- err
				return
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors during concurrent writes:", len(errors))
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	}

	// Verify data was written correctly
	schedules, err := store.ListSchedules(1)
	if err != nil {
		t.Fatalf("Failed to list schedules: %v", err)
	}

	if len(schedules) != numSchedules {
		t.Errorf("Expected %d schedules, got %d", numSchedules, len(schedules))
	}

	// Count total runs
	totalRuns := 0
	for _, schedule := range schedules {
		runs, err := store.ListRuns(1, schedule.ID)
		if err != nil {
			t.Fatalf("Failed to list runs for schedule %d: %v", schedule.ID, err)
		}
		totalRuns += len(runs)
	}

	expectedRuns := numSchedules * numRuns
	if totalRuns != expectedRuns {
		t.Errorf("Expected %d total runs, got %d", expectedRuns, totalRuns)
	}
}

// TestWriteQueueShutdown tests that the write queue shuts down gracefully
func TestWriteQueueShutdown(t *testing.T) {
	// Create temporary database
	dbPath := "test_shutdown.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Queue some operations
	for i := 0; i < 5; i++ {
		nextRun := time.Now().Add(1 * time.Hour)
		schedule := &model.Schedule{
			OrgID:          1,
			Name:           "Test Schedule",
			DashboardUID:   "test-dashboard",
			DashboardTitle: "Test Dashboard",
			RangeFrom:      "now-1h",
			RangeTo:        "now",
			IntervalType:   "daily",
			Timezone:       "UTC",
			Recipients:     model.Recipients{To: []string{"test@example.com"}},
			EmailSubject:   "Test Report",
			EmailBody:      "Test Body",
			Enabled:        true,
			NextRunAt:      &nextRun,
			OwnerUserID:    1,
		}

		if err := store.CreateSchedule(schedule); err != nil {
			t.Fatalf("Failed to create schedule: %v", err)
		}
	}

	// Close should complete all pending operations before returning
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Verify all schedules were created
	store2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	schedules, err := store2.ListSchedules(1)
	if err != nil {
		t.Fatalf("Failed to list schedules: %v", err)
	}

	if len(schedules) != 5 {
		t.Errorf("Expected 5 schedules after shutdown, got %d", len(schedules))
	}
}

// BenchmarkConcurrentWrites benchmarks concurrent write performance
func BenchmarkConcurrentWrites(b *testing.B) {
	// Create temporary database
	dbPath := "bench_concurrent.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nextRun := time.Now().Add(1 * time.Hour)
		schedule := &model.Schedule{
			OrgID:          1,
			Name:           "Bench Schedule",
			DashboardUID:   "bench-dashboard",
			DashboardTitle: "Bench Dashboard",
			RangeFrom:      "now-1h",
			RangeTo:        "now",
			IntervalType:   "daily",
			Timezone:       "UTC",
			Recipients:     model.Recipients{To: []string{"bench@example.com"}},
			EmailSubject:   "Bench Report",
			EmailBody:      "Bench Body",
			Enabled:        true,
			NextRunAt:      &nextRun,
			OwnerUserID:    1,
		}

		if err := store.CreateSchedule(schedule); err != nil {
			b.Fatalf("Failed to create schedule: %v", err)
		}
	}
}
