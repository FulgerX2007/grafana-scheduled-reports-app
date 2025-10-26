//go:build integration
// +build integration

package render

import (
    "context"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
)

// TestRenderDashboard_Integration tests actual browser rendering
// Run with: go test -tags=integration -v ./pkg/render/
func TestRenderDashboard_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Check if Chromium is available
    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    // Create a mock Grafana server
    mockGrafana := httptest.NewServer(
        http.HandlerFunc(
            func(w http.ResponseWriter, r *http.Request) {
                // Check Authorization header
                auth := r.Header.Get("Authorization")
                if auth == "" {
                    t.Error("Authorization header not set")
                    http.Error(w, "Unauthorized", http.StatusUnauthorized)
                    return
                }

                if auth != "Bearer test-token-123" {
                    t.Errorf("Wrong authorization header: %v", auth)
                }

                // Return a simple HTML page
                w.Header().Set("Content-Type", "text/html")
                w.Write(
                    []byte(`
<!DOCTYPE html>
<html>
<head><title>Test Dashboard</title></head>
<body style="background-color: #f0f0f0;">
	<h1>Test Dashboard</h1>
	<div id="content">This is a test dashboard for integration testing</div>
</body>
</html>
		`),
                )
            },
        ),
    )
    defer mockGrafana.Close()

    // Set up environment
    os.Setenv("GF_PLUGIN_SA_TOKEN", "test-token-123")
    defer os.Unsetenv("GF_PLUGIN_SA_TOKEN")

    // Create renderer
    config := model.RendererConfig{
        TimeoutMS:         30000,
        ViewportWidth:     1920,
        ViewportHeight:    1080,
        DeviceScaleFactor: 1.0,
        Headless:          true,
        NoSandbox:         true, // Required for CI environments
        DisableGPU:        true,
    }

    r := NewChromiumRenderer(mockGrafana.URL, config)
    defer r.Close()

    // Create test schedule
    schedule := &model.Schedule{
        DashboardUID: "test-dash",
        RangeFrom:    "now-1h",
        RangeTo:      "now",
        OrgID:        1,
        Timezone:     "UTC",
        Variables:    model.VariableList{},
    }

    // Render dashboard
    ctx := context.Background()
    imageData, err := r.RenderDashboard(ctx, schedule)

    if err != nil {
        t.Fatalf("RenderDashboard() error = %v", err)
    }

    // Verify PNG format
    if len(imageData) < 8 {
        t.Fatal("Image data too small")
    }

    if string(imageData[1:4]) != "PNG" {
        t.Errorf("Image is not a PNG, got magic bytes: %v", imageData[0:8])
    }

    t.Logf("Successfully rendered dashboard: %d bytes", len(imageData))
}

// TestBrowserReuse_Integration tests browser instance reuse
func TestBrowserReuse_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    config := model.RendererConfig{
        TimeoutMS:      30000,
        ViewportWidth:  1280,
        ViewportHeight: 720,
        Headless:       true,
        NoSandbox:      true,
        DisableGPU:     true,
    }

    r := NewChromiumRenderer("http://example.com", config)
    defer r.Close()

    // First getBrowser call should initialize
    browser1, err := r.getBrowser()
    if err != nil {
        t.Fatalf("First getBrowser() error = %v", err)
    }

    if browser1 == nil {
        t.Fatal("Browser is nil after first getBrowser()")
    }

    // Second call should return the same instance
    browser2, err := r.getBrowser()
    if err != nil {
        t.Fatalf("Second getBrowser() error = %v", err)
    }

    if browser1 != browser2 {
        t.Error("getBrowser() returned different instances, expected reuse")
    }

    t.Log("Browser instance reused successfully")
}

// TestBrowserCleanup_Integration tests proper cleanup
func TestBrowserCleanup_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    config := model.RendererConfig{
        Headless:   true,
        NoSandbox:  true,
        DisableGPU: true,
    }

    r := NewChromiumRenderer("http://example.com", config)

    // Initialize browser
    _, err := r.getBrowser()
    if err != nil {
        t.Fatalf("getBrowser() error = %v", err)
    }

    // Close should not error
    err = r.Close()
    if err != nil {
        t.Errorf("Close() error = %v", err)
    }

    // Multiple closes should be safe
    err = r.Close()
    if err != nil {
        t.Errorf("Second Close() error = %v", err)
    }

    t.Log("Browser cleanup successful")
}

// TestRenderTimeout_Integration tests timeout handling
func TestRenderTimeout_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    // Create a slow server
    slowServer := httptest.NewServer(
        http.HandlerFunc(
            func(w http.ResponseWriter, r *http.Request) {
                time.Sleep(5 * time.Second) // Longer than timeout
                w.Write([]byte("Too slow"))
            },
        ),
    )
    defer slowServer.Close()

    os.Setenv("GF_PLUGIN_SA_TOKEN", "test-token")
    defer os.Unsetenv("GF_PLUGIN_SA_TOKEN")

    config := model.RendererConfig{
        TimeoutMS:  1000, // 1 second timeout
        Headless:   true,
        NoSandbox:  true,
        DisableGPU: true,
    }

    r := NewChromiumRenderer(slowServer.URL, config)
    defer r.Close()

    schedule := &model.Schedule{
        DashboardUID: "slow-dash",
        RangeFrom:    "now-1h",
        RangeTo:      "now",
        OrgID:        1,
        Timezone:     "UTC",
        Variables:    model.VariableList{},
    }

    ctx := context.Background()
    _, err := r.RenderDashboard(ctx, schedule)

    if err == nil {
        t.Error("Expected timeout error, got nil")
    }

    t.Logf("Timeout handled correctly: %v", err)
}

// TestMultiplePagesSequential_Integration tests sequential page rendering
func TestMultiplePagesSequential_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    requestCount := 0
    mockServer := httptest.NewServer(
        http.HandlerFunc(
            func(w http.ResponseWriter, r *http.Request) {
                requestCount++
                w.Header().Set("Content-Type", "text/html")
                w.Write(
                    []byte(`
<!DOCTYPE html>
<html><body><h1>Request ` + string(rune(requestCount)) + `</h1></body></html>
		`),
                )
            },
        ),
    )
    defer mockServer.Close()

    os.Setenv("GF_PLUGIN_SA_TOKEN", "test-token")
    defer os.Unsetenv("GF_PLUGIN_SA_TOKEN")

    config := model.RendererConfig{
        TimeoutMS:  30000,
        Headless:   true,
        NoSandbox:  true,
        DisableGPU: true,
    }

    r := NewChromiumRenderer(mockServer.URL, config)
    defer r.Close()

    // Render multiple schedules sequentially
    for i := 0; i < 3; i++ {
        schedule := &model.Schedule{
            DashboardUID: "test-dash",
            RangeFrom:    "now-1h",
            RangeTo:      "now",
            OrgID:        int64(i + 1),
            Timezone:     "UTC",
            Variables:    model.VariableList{},
        }

        ctx := context.Background()
        imageData, err := r.RenderDashboard(ctx, schedule)

        if err != nil {
            t.Fatalf("RenderDashboard() iteration %d error = %v", i, err)
        }

        if len(imageData) < 8 || string(imageData[1:4]) != "PNG" {
            t.Errorf("Invalid PNG data in iteration %d", i)
        }

        t.Logf("Iteration %d: rendered %d bytes", i, len(imageData))
    }

    if requestCount != 3 {
        t.Errorf("Expected 3 requests, got %d", requestCount)
    }
}

// TestRenderWithVariables_Integration tests dashboard variable rendering
func TestRenderWithVariables_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    if !isChromiumAvailable() {
        t.Skip("Chromium not available, skipping integration test")
    }

    var receivedURL string
    mockServer := httptest.NewServer(
        http.HandlerFunc(
            func(w http.ResponseWriter, r *http.Request) {
                receivedURL = r.URL.String()
                w.Header().Set("Content-Type", "text/html")
                w.Write([]byte(`<!DOCTYPE html><html><body>Test</body></html>`))
            },
        ),
    )
    defer mockServer.Close()

    os.Setenv("GF_PLUGIN_SA_TOKEN", "test-token")
    defer os.Unsetenv("GF_PLUGIN_SA_TOKEN")

    config := model.RendererConfig{
        TimeoutMS:  30000,
        Headless:   true,
        NoSandbox:  true,
        DisableGPU: true,
    }

    r := NewChromiumRenderer(mockServer.URL, config)
    defer r.Close()

    schedule := &model.Schedule{
        DashboardUID: "var-test",
        RangeFrom:    "now-6h",
        RangeTo:      "now",
        OrgID:        1,
        Timezone:     "America/New_York",
        Variables: model.VariableList{
            {Name: "server", Value: "web-01"},
            {Name: "environment", Value: "production"},
            {Name: "region", Value: "us-east-1"},
        },
    }

    ctx := context.Background()
    _, err := r.RenderDashboard(ctx, schedule)

    if err != nil {
        t.Fatalf("RenderDashboard() error = %v", err)
    }

    // Verify variables in URL
    expectedParams := []string{
        "var-server=web-01",
        "var-environment=production",
        "var-region=us-east-1",
        "tz=America%2FNew_York",
        "from=now-6h",
        "to=now",
    }

    for _, param := range expectedParams {
        if !contains(receivedURL, param) {
            t.Errorf("URL should contain parameter %v: %v", param, receivedURL)
        }
    }

    t.Logf("Variables rendered correctly in URL: %v", receivedURL)
}

// Helper function to check if Chromium is available
func isChromiumAvailable() bool {
    // Try to find Chromium in common locations
    paths := []string{
        "/usr/bin/chromium",
        "/usr/bin/chromium-browser",
        "/usr/bin/google-chrome",
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
    }

    for _, path := range paths {
        if _, err := os.Stat(path); err == nil {
            return true
        }
    }

    // Also check if rod can auto-detect
    return os.Getenv("SKIP_CHROMIUM_CHECK") == "1"
}
