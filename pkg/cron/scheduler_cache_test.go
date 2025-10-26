package cron

import (
    "os"
    "sync"
    "testing"
    "time"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/store"
)

// TestCachedSettingsConcurrentAccess tests that multiple concurrent schedule executions
// can access settings without causing database locks
func TestCachedSettingsConcurrentAccess(t *testing.T) {
    // Create temporary database
    dbPath := "test_cache_concurrent.db"
    defer os.Remove(dbPath)

    st, err := store.NewStore(dbPath)
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    defer st.Close()

    // Create settings for org 1
    settings := &model.Settings{
        OrgID: 1,
        SMTPConfig: &model.SMTPConfig{
            Host:     "smtp.example.com",
            Port:     587,
            Username: "test@example.com",
            Password: "password",
            From:     "test@example.com",
        },
        RendererConfig: model.RendererConfig{
            TimeoutMS:         30000,
            DelayMS:           1000,
            ViewportWidth:     1920,
            ViewportHeight:    1080,
            DeviceScaleFactor: 1.0,
        },
        Limits: model.Limits{
            MaxRecipients:        10,
            MaxAttachmentSizeMB:  25,
            MaxConcurrentRenders: 5,
            RetentionDays:        30,
        },
    }

    if err := st.UpsertSettings(settings); err != nil {
        t.Fatalf("Failed to create settings: %v", err)
    }

    // Create scheduler
    scheduler := NewScheduler(st, "http://localhost:3000", "/tmp/artifacts", 10)

    // Simulate 20 concurrent schedule executions all accessing settings
    numConcurrent := 20
    var wg sync.WaitGroup
    errChan := make(chan error, numConcurrent)

    for i := 0; i < numConcurrent; i++ {
        wg.Add(1)
        go func(scheduleNum int) {
            defer wg.Done()

            // Get settings (this is what happens in executeScheduleOnce)
            retrievedSettings, err := scheduler.getCachedSettings(1)
            if err != nil {
                errChan <- err
                return
            }

            if retrievedSettings == nil {
                errChan <- err
                return
            }

            // Verify settings are correct
            if retrievedSettings.OrgID != 1 {
                t.Errorf("Expected OrgID 1, got %d", retrievedSettings.OrgID)
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
        t.Errorf("Got %d errors during concurrent settings access:", len(errors))
        for _, err := range errors {
            t.Errorf("  - %v", err)
        }
    }
}

// TestSettingsCacheHitRatio verifies that the cache is actually being used
func TestSettingsCacheHitRatio(t *testing.T) {
    // Create temporary database
    dbPath := "test_cache_hits.db"
    defer os.Remove(dbPath)

    st, err := store.NewStore(dbPath)
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    defer st.Close()

    // Create settings for org 1
    settings := &model.Settings{
        OrgID: 1,
        SMTPConfig: &model.SMTPConfig{
            Host:     "smtp.example.com",
            Port:     587,
            Username: "test@example.com",
            Password: "password",
            From:     "test@example.com",
        },
        RendererConfig: model.RendererConfig{
            TimeoutMS:         30000,
            DelayMS:           1000,
            ViewportWidth:     1920,
            ViewportHeight:    1080,
            DeviceScaleFactor: 1.0,
        },
        Limits: model.Limits{
            MaxRecipients:        10,
            MaxAttachmentSizeMB:  25,
            MaxConcurrentRenders: 5,
            RetentionDays:        30,
        },
    }

    if err := st.UpsertSettings(settings); err != nil {
        t.Fatalf("Failed to create settings: %v", err)
    }

    // Create scheduler
    scheduler := NewScheduler(st, "http://localhost:3000", "/tmp/artifacts", 10)

    // First access - cache miss (should fetch from DB)
    start := time.Now()
    settings1, err := scheduler.getCachedSettings(1)
    firstAccessTime := time.Since(start)
    if err != nil {
        t.Fatalf("Failed to get settings (first access): %v", err)
    }
    if settings1 == nil {
        t.Fatal("Settings should not be nil")
    }

    // Second access - cache hit (should be faster)
    start = time.Now()
    settings2, err := scheduler.getCachedSettings(1)
    secondAccessTime := time.Since(start)
    if err != nil {
        t.Fatalf("Failed to get settings (second access): %v", err)
    }
    if settings2 == nil {
        t.Fatal("Settings should not be nil")
    }

    // Verify cache hit is faster (should be significantly faster as no DB access)
    t.Logf("First access (cache miss): %v", firstAccessTime)
    t.Logf("Second access (cache hit): %v", secondAccessTime)

    // Cache hit should be at least 10x faster (typically 1000x+)
    if secondAccessTime > firstAccessTime/10 {
        t.Errorf("Cache hit not significantly faster than miss: %v vs %v", secondAccessTime, firstAccessTime)
    }

    // Verify same pointer (cache returns same instance)
    if settings1 != settings2 {
        t.Error("Expected cached settings to be the same instance")
    }
}

// TestSettingsCachePerOrg verifies that cache is properly isolated per organization
func TestSettingsCachePerOrg(t *testing.T) {
    // Create temporary database
    dbPath := "test_cache_per_org.db"
    defer os.Remove(dbPath)

    st, err := store.NewStore(dbPath)
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }
    defer st.Close()

    // Create settings for org 1
    settings1 := &model.Settings{
        OrgID: 1,
        SMTPConfig: &model.SMTPConfig{
            Host:     "smtp1.example.com",
            Port:     587,
            Username: "test1@example.com",
            Password: "password1",
            From:     "test1@example.com",
        },
        RendererConfig: model.RendererConfig{
            TimeoutMS:         30000,
            DelayMS:           1000,
            ViewportWidth:     1920,
            ViewportHeight:    1080,
            DeviceScaleFactor: 1.0,
        },
        Limits: model.Limits{
            MaxRecipients:        10,
            MaxAttachmentSizeMB:  25,
            MaxConcurrentRenders: 5,
            RetentionDays:        30,
        },
    }

    if err := st.UpsertSettings(settings1); err != nil {
        t.Fatalf("Failed to create settings for org 1: %v", err)
    }

    // Create settings for org 2
    settings2 := &model.Settings{
        OrgID: 2,
        SMTPConfig: &model.SMTPConfig{
            Host:     "smtp2.example.com",
            Port:     25,
            Username: "test2@example.com",
            Password: "password2",
            From:     "test2@example.com",
        },
        RendererConfig: model.RendererConfig{
            TimeoutMS:         60000,
            DelayMS:           2000,
            ViewportWidth:     1280,
            ViewportHeight:    720,
            DeviceScaleFactor: 2.0,
        },
        Limits: model.Limits{
            MaxRecipients:        20,
            MaxAttachmentSizeMB:  50,
            MaxConcurrentRenders: 10,
            RetentionDays:        60,
        },
    }

    if err := st.UpsertSettings(settings2); err != nil {
        t.Fatalf("Failed to create settings for org 2: %v", err)
    }

    // Create scheduler
    scheduler := NewScheduler(st, "http://localhost:3000", "/tmp/artifacts", 10)

    // Get settings for org 1
    cachedSettings1, err := scheduler.getCachedSettings(1)
    if err != nil {
        t.Fatalf("Failed to get settings for org 1: %v", err)
    }

    // Get settings for org 2
    cachedSettings2, err := scheduler.getCachedSettings(2)
    if err != nil {
        t.Fatalf("Failed to get settings for org 2: %v", err)
    }

    // Verify settings are different
    if cachedSettings1.SMTPConfig.Host == cachedSettings2.SMTPConfig.Host {
        t.Error("Expected different SMTP hosts for different orgs")
    }

    if cachedSettings1.RendererConfig.TimeoutMS == cachedSettings2.RendererConfig.TimeoutMS {
        t.Error("Expected different timeout values for different orgs")
    }

    // Verify correct settings were returned
    if cachedSettings1.SMTPConfig.Host != "smtp1.example.com" {
        t.Errorf("Expected smtp1.example.com, got %s", cachedSettings1.SMTPConfig.Host)
    }

    if cachedSettings2.SMTPConfig.Host != "smtp2.example.com" {
        t.Errorf("Expected smtp2.example.com, got %s", cachedSettings2.SMTPConfig.Host)
    }
}
