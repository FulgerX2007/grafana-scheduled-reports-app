package render

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yourusername/sheduled-reports-app/pkg/model"
)

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name            string
		grafanaURL      string
		config          model.RendererConfig
		wantWidth       int
		wantHeight      int
		wantTimeout     int
		wantScaleFactor float64
		wantHeadless    bool
	}{
		{
			name:            "default values",
			grafanaURL:      "http://localhost:3000",
			config:          model.RendererConfig{},
			wantWidth:       1920,
			wantHeight:      1080,
			wantTimeout:     30000,
			wantScaleFactor: 2.0,
			wantHeadless:    true,
		},
		{
			name:       "custom values",
			grafanaURL: "http://grafana:3000",
			config: model.RendererConfig{
				ViewportWidth:     1280,
				ViewportHeight:    720,
				TimeoutMS:         60000,
				DeviceScaleFactor: 3.0,
				Headless:          true,
			},
			wantWidth:       1280,
			wantHeight:      720,
			wantTimeout:     60000,
			wantScaleFactor: 3.0,
			wantHeadless:    true,
		},
		{
			name:       "with chromium settings",
			grafanaURL: "http://localhost:3000",
			config: model.RendererConfig{
				ChromiumPath:  "/usr/bin/chromium",
				NoSandbox:     true,
				DisableGPU:    true,
				SkipTLSVerify: true,
			},
			wantWidth:       1920,
			wantHeight:      1080,
			wantTimeout:     30000,
			wantScaleFactor: 2.0,
			wantHeadless:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewChromiumRenderer(tt.grafanaURL, tt.config)

			if r == nil {
				t.Fatal("NewRenderer returned nil")
			}

			if r.grafanaURL != tt.grafanaURL {
				t.Errorf("grafanaURL = %v, want %v", r.grafanaURL, tt.grafanaURL)
			}

			if r.config.ViewportWidth != tt.wantWidth {
				t.Errorf("ViewportWidth = %v, want %v", r.config.ViewportWidth, tt.wantWidth)
			}

			if r.config.ViewportHeight != tt.wantHeight {
				t.Errorf("ViewportHeight = %v, want %v", r.config.ViewportHeight, tt.wantHeight)
			}

			if r.config.TimeoutMS != tt.wantTimeout {
				t.Errorf("TimeoutMS = %v, want %v", r.config.TimeoutMS, tt.wantTimeout)
			}

			if r.config.DeviceScaleFactor != tt.wantScaleFactor {
				t.Errorf("DeviceScaleFactor = %v, want %v", r.config.DeviceScaleFactor, tt.wantScaleFactor)
			}

			if r.config.Headless != tt.wantHeadless {
				t.Errorf("Headless = %v, want %v", r.config.Headless, tt.wantHeadless)
			}

			if r.browser != nil {
				t.Error("browser should be nil (lazy initialization)")
			}
		})
	}
}

func TestBuildDashboardURL(t *testing.T) {
	tests := []struct {
		name       string
		grafanaURL string
		schedule   *model.Schedule
		wantErr    bool
		contains   []string
	}{
		{
			name:       "basic URL",
			grafanaURL: "http://localhost:3000",
			schedule: &model.Schedule{
				DashboardUID: "abc123",
				RangeFrom:    "now-7d",
				RangeTo:      "now",
				OrgID:        1,
				Timezone:     "UTC",
				Variables:    model.VariableList{},
			},
			wantErr: false,
			contains: []string{
				"/d/abc123",
				"from=now-7d",
				"to=now",
				"kiosk=1",
				"tz=UTC",
			},
		},
		{
			name:       "with variables",
			grafanaURL: "http://localhost:3000",
			schedule: &model.Schedule{
				DashboardUID: "test-dash",
				RangeFrom:    "now-24h",
				RangeTo:      "now",
				OrgID:        2,
				Timezone:     "America/New_York",
				Variables: model.VariableList{
					{Name: "server", Value: "web-01"},
					{Name: "environment", Value: "production"},
				},
			},
			wantErr: false,
			contains: []string{
				"/d/test-dash",
				"var-server=web-01",
				"var-environment=production",
			},
		},
		{
			name:       "with duplicate variables",
			grafanaURL: "http://localhost:3000",
			schedule: &model.Schedule{
				DashboardUID: "test-dash",
				RangeFrom:    "now-24h",
				RangeTo:      "now",
				OrgID:        1,
				Timezone:     "UTC",
				Variables: model.VariableList{
					{Name: "rat", Value: "1", IsOriginal: true},  // Original from dashboard
					{Name: "rat", Value: "2", IsOriginal: false}, // User-created duplicate
					{Name: "server", Value: "web-01", IsOriginal: true},
				},
			},
			wantErr: false,
			contains: []string{
				"/d/test-dash",
				"var-rat=1",
				"var-rat=2",
				"var-server=web-01",
			},
		},
		{
			name:       "with subpath",
			grafanaURL: "http://localhost:3000/dna",
			schedule: &model.Schedule{
				DashboardUID: "dash-uid",
				RangeFrom:    "now-1h",
				RangeTo:      "now",
				OrgID:        1,
				Timezone:     "UTC",
				Variables:    model.VariableList{},
			},
			wantErr: false,
			contains: []string{
				"/dna/d/dash-uid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewChromiumRenderer(tt.grafanaURL, model.RendererConfig{})

			url, err := r.buildDashboardURL(tt.schedule)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildDashboardURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for _, substr := range tt.contains {
					if !contains(url, substr) {
						t.Errorf("buildDashboardURL() = %v, should contain %v", url, substr)
					}
				}
			}
		})
	}
}

func TestGetServiceAccountToken(t *testing.T) {
	tests := []struct {
		name       string
		envToken   string
		wantErr    bool
		setupEnv   bool
		cleanupEnv bool
	}{
		{
			name:       "token from environment",
			envToken:   "test-token-12345",
			wantErr:    false,
			setupEnv:   true,
			cleanupEnv: true,
		},
		{
			name:       "no token available",
			envToken:   "",
			wantErr:    true,
			setupEnv:   false,
			cleanupEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv {
				os.Setenv("GF_PLUGIN_APP_CLIENT_SECRET", tt.envToken)
			}

			if tt.cleanupEnv {
				defer os.Unsetenv("GF_PLUGIN_APP_CLIENT_SECRET")
			}

			ctx := context.Background()
			r := NewChromiumRenderer("http://localhost:3000", model.RendererConfig{})
			token, err := r.getServiceAccountToken(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("getServiceAccountToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token != tt.envToken {
				t.Errorf("getServiceAccountToken() = %v, want %v", token, tt.envToken)
			}
		})
	}
}

func TestRendererClose(t *testing.T) {
	r := NewChromiumRenderer("http://localhost:3000", model.RendererConfig{})

	// Should not error when browser is nil
	err := r.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Test is primarily for nil-safety
	// Browser initialization requires actual Chromium, tested in integration tests
}

func TestRendererConfig_Defaults(t *testing.T) {
	config := model.RendererConfig{}
	r := NewChromiumRenderer("http://localhost:3000", config)

	expectedDefaults := map[string]interface{}{
		"ViewportWidth":     1920,
		"ViewportHeight":    1080,
		"TimeoutMS":         30000,
		"DeviceScaleFactor": 2.0,
		"Headless":          true,
	}

	if r.config.ViewportWidth != expectedDefaults["ViewportWidth"] {
		t.Errorf("Default ViewportWidth = %v, want %v", r.config.ViewportWidth, expectedDefaults["ViewportWidth"])
	}

	if r.config.ViewportHeight != expectedDefaults["ViewportHeight"] {
		t.Errorf("Default ViewportHeight = %v, want %v", r.config.ViewportHeight, expectedDefaults["ViewportHeight"])
	}

	if r.config.TimeoutMS != expectedDefaults["TimeoutMS"] {
		t.Errorf("Default TimeoutMS = %v, want %v", r.config.TimeoutMS, expectedDefaults["TimeoutMS"])
	}

	if r.config.DeviceScaleFactor != expectedDefaults["DeviceScaleFactor"] {
		t.Errorf("Default DeviceScaleFactor = %v, want %v", r.config.DeviceScaleFactor, expectedDefaults["DeviceScaleFactor"])
	}

	if r.config.Headless != expectedDefaults["Headless"] {
		t.Errorf("Default Headless = %v, want %v", r.config.Headless, expectedDefaults["Headless"])
	}
}

func TestRendererConfig_CustomChromiumPath(t *testing.T) {
	config := model.RendererConfig{
		ChromiumPath: "/custom/path/chromium",
		NoSandbox:    true,
		DisableGPU:   true,
	}

	r := NewChromiumRenderer("http://localhost:3000", config)

	if r.config.ChromiumPath != "/custom/path/chromium" {
		t.Errorf("ChromiumPath = %v, want /custom/path/chromium", r.config.ChromiumPath)
	}

	if !r.config.NoSandbox {
		t.Error("NoSandbox should be true")
	}

	if !r.config.DisableGPU {
		t.Error("DisableGPU should be true")
	}
}

func TestRendererConfig_TLSVerification(t *testing.T) {
	config := model.RendererConfig{
		SkipTLSVerify: true,
	}

	r := NewChromiumRenderer("http://localhost:3000", config)

	if !r.config.SkipTLSVerify {
		t.Error("SkipTLSVerify should be true")
	}
}

func TestScheduleTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		urlParam string // URL-encoded version
	}{
		{
			name:     "UTC",
			timezone: "UTC",
			urlParam: "tz=UTC",
		},
		{
			name:     "America/New_York",
			timezone: "America/New_York",
			urlParam: "tz=America", // Partial match (URL encoding will have %2F)
		},
		{
			name:     "Europe/London",
			timezone: "Europe/London",
			urlParam: "tz=Europe", // Partial match (URL encoding will have %2F)
		},
	}

	r := NewChromiumRenderer("http://localhost:3000", model.RendererConfig{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &model.Schedule{
				DashboardUID: "test",
				RangeFrom:    "now-1h",
				RangeTo:      "now",
				OrgID:        1,
				Timezone:     tt.timezone,
				Variables:    model.VariableList{},
			}

			url, err := r.buildDashboardURL(schedule)
			if err != nil {
				t.Errorf("buildDashboardURL() error = %v", err)
				return
			}

			// Check for either exact or encoded version
			if !contains(url, tt.urlParam) && !contains(url, "tz="+tt.timezone) {
				t.Errorf("URL should contain timezone parameter (got %v)", url)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewRenderer(b *testing.B) {
	config := model.RendererConfig{}

	for i := 0; i < b.N; i++ {
		_ = NewChromiumRenderer("http://localhost:3000", config)
	}
}

func BenchmarkBuildDashboardURL(b *testing.B) {
	r := NewChromiumRenderer("http://localhost:3000", model.RendererConfig{})
	schedule := &model.Schedule{
		DashboardUID: "bench-test",
		RangeFrom:    "now-24h",
		RangeTo:      "now",
		OrgID:        1,
		Timezone:     "UTC",
		Variables: model.VariableList{
			{Name: "var1", Value: "value1"},
			{Name: "var2", Value: "value2"},
			{Name: "var3", Value: "value3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.buildDashboardURL(schedule)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Table-driven test for localhost conversion
func TestLocalhostConversion(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		wantHost string
	}{
		{
			name:     "localhost:3000",
			inputURL: "http://localhost:3000",
			wantHost: "grafana:3000",
		},
		{
			name:     "127.0.0.1:3000",
			inputURL: "http://127.0.0.1:3000",
			wantHost: "grafana:3000",
		},
		{
			name:     "custom host",
			inputURL: "http://grafana.example.com:3000",
			wantHost: "grafana.example.com:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewChromiumRenderer(tt.inputURL, model.RendererConfig{})
			schedule := &model.Schedule{
				DashboardUID: "test",
				RangeFrom:    "now-1h",
				RangeTo:      "now",
				OrgID:        1,
				Timezone:     "UTC",
				Variables:    model.VariableList{},
			}

			url, err := r.buildDashboardURL(schedule)
			if err != nil {
				t.Fatalf("buildDashboardURL() error = %v", err)
			}

			if !contains(url, tt.wantHost) {
				t.Errorf("URL %v should contain host %v", url, tt.wantHost)
			}
		})
	}
}

// Test delay configuration
func TestRenderDelay(t *testing.T) {
	config := model.RendererConfig{
		DelayMS: 5000,
	}

	r := NewChromiumRenderer("http://localhost:3000", config)

	if r.config.DelayMS != 5000 {
		t.Errorf("DelayMS = %v, want 5000", r.config.DelayMS)
	}
}

// Test timeout configuration
func TestRenderTimeout(t *testing.T) {
	config := model.RendererConfig{
		TimeoutMS: 60000,
	}

	r := NewChromiumRenderer("http://localhost:3000", config)

	if r.config.TimeoutMS != 60000 {
		t.Errorf("TimeoutMS = %v, want 60000", r.config.TimeoutMS)
	}

	// Verify timeout is properly set
	duration := time.Duration(r.config.TimeoutMS) * time.Millisecond
	expected := 60 * time.Second

	if duration != expected {
		t.Errorf("Timeout duration = %v, want %v", duration, expected)
	}
}
