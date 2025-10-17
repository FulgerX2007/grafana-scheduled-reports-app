package render

import (
	"context"

	"github.com/yourusername/sheduled-reports-app/pkg/model"
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
	// BackendChromium is the Chromium-based renderer (only supported backend)
	BackendChromium BackendType = "chromium"
)

// NewBackend creates a new Chromium rendering backend
func NewBackend(backendType BackendType, grafanaURL string, config model.RendererConfig) (Backend, error) {
	// Only Chromium backend is supported
	return NewChromiumRenderer(grafanaURL, config), nil
}
