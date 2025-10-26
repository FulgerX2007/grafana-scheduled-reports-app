package render

import (
    "context"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
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

// NewBackend creates a new rendering backend
// Currently only supports Chromium-based rendering via go-rod
func NewBackend(grafanaURL string, config model.RendererConfig) (Backend, error) {
    return NewChromiumRenderer(grafanaURL, config), nil
}
