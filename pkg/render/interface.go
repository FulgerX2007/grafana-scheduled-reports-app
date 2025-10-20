package render

import (
	"context"

	"github.com/yourusername/scheduled-reports-app/pkg/model"
)

// Backend defines the interface for rendering backends
type Backend interface {
	// RenderDashboard renders a Grafana dashboard to PDF
	RenderDashboard(ctx context.Context, schedule *model.Schedule) ([]byte, error)

	// Close cleans up resources used by the backend
	Close() error

	// Name returns the name of the backend
	Name() string
}

// BackendType represents the rendering backend type (kept for compatibility)
type BackendType string

const (
	// BackendChromium is the Chromium-based renderer using go-rod
	BackendChromium BackendType = "chromium"
	// BackendPlaywright is the Playwright-based renderer (recommended for better reliability)
	BackendPlaywright BackendType = "playwright"
)

// NewBackend creates a new rendering backend based on the specified type
func NewBackend(backendType BackendType, grafanaURL string, config model.RendererConfig) (Backend, error) {
	switch backendType {
	case BackendPlaywright:
		return NewPlaywrightRenderer(grafanaURL, config), nil
	case BackendChromium:
		return NewChromiumRenderer(grafanaURL, config), nil
	default:
		// Default to Chromium (rod) for better Alpine/Docker compatibility
		// Playwright requires Node.js driver which doesn't work well in Alpine
		return NewChromiumRenderer(grafanaURL, config), nil
	}
}
