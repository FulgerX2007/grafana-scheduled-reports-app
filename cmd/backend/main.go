package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/yourusername/scheduled-reports-app/pkg/api"
	"github.com/yourusername/scheduled-reports-app/pkg/cron"
	"github.com/yourusername/scheduled-reports-app/pkg/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func run() error {
	// Get plugin data path
	dataPath := os.Getenv("GF_PLUGIN_APP_DATA_PATH")
	if dataPath == "" {
		// Fallback: use executable directory + data
		execPath, err := os.Executable()
		if err == nil {
			dataPath = filepath.Join(filepath.Dir(execPath), "data")
		} else {
			dataPath = "./data"
		}
	}

	log.Printf("Using data path: %s", dataPath)

	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		log.Printf("Failed to create data directory %s: %v", dataPath, err)
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize database
	dbPath := filepath.Join(dataPath, "reporting.db")
	log.Printf("Initializing database at: %s", dbPath)
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer st.Close()

	// Build Grafana URL from instance configuration
	// Grafana sets these environment variables for plugins
	protocol := os.Getenv("GF_INSTANCE_PROTOCOL")
	if protocol == "" {
		protocol = "http"
	}

	httpAddr := os.Getenv("GF_INSTANCE_HTTP_ADDR")
	if httpAddr == "" || httpAddr == "0.0.0.0" || httpAddr == "::" {
		// If binding to all interfaces, use localhost for internal requests
		httpAddr = "localhost"
	}

	httpPort := os.Getenv("GF_INSTANCE_HTTP_PORT")
	if httpPort == "" {
		httpPort = os.Getenv("GF_SERVER_HTTP_PORT")
		if httpPort == "" {
			httpPort = "3000"
		}
	}

	// Get root_url subpath if configured
	rootURL := os.Getenv("GF_SERVER_ROOT_URL")
	subPath := ""
	if rootURL != "" {
		// Parse root_url to extract subpath (e.g., http://localhost:3000/dna -> /dna)
		if u, err := url.Parse(rootURL); err == nil && u.Path != "" && u.Path != "/" {
			subPath = u.Path
			log.Printf("Detected root_url subpath: %s", subPath)
		}
	}

	grafanaURL := fmt.Sprintf("%s://%s:%s%s", protocol, httpAddr, httpPort, subPath)
	log.Printf("Using Grafana URL: %s", grafanaURL)
	log.Println("Using Grafana-managed service account for authentication")
	log.Println("Token will be retrieved automatically from plugin context")

	// Check if managed service account token is available at startup
	// This environment variable is set by Grafana when externalServiceAccounts feature toggle is enabled
	saToken := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
	if saToken != "" {
		log.Printf("✓ SUCCESS: Service account token found in environment variable GF_PLUGIN_APP_CLIENT_SECRET")
		log.Printf("✓ Token length: %d characters", len(saToken))
		log.Printf("✓ Token preview: %s...", saToken[:min(20, len(saToken))])
		log.Println("✓ This token will be used for authenticating dashboard rendering requests")
	} else {
		log.Println("⚠ WARNING: GF_PLUGIN_APP_CLIENT_SECRET environment variable is NOT set")
		log.Println("⚠ Managed service accounts may not be configured correctly")
		log.Println("⚠ Please ensure:")
		log.Println("  1. Grafana version is 10.3 or later")
		log.Println("  2. Feature toggle is enabled: [feature_toggles] enable = externalServiceAccounts")
		log.Println("  3. Grafana has been restarted after enabling the feature toggle")
		log.Println("  4. Check Settings page (Apps → Scheduled Reports → Settings) for status")
	}

	// Create artifacts directory
	artifactsPath := filepath.Join(dataPath, "artifacts")
	log.Printf("Creating artifacts directory: %s", artifactsPath)
	if err := os.MkdirAll(artifactsPath, 0755); err != nil {
		log.Printf("Failed to create artifacts directory: %v", err)
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Initialize scheduler (token will be retrieved from context on first API call)
	maxConcurrent := 5 // Default max concurrent renders
	log.Printf("Initializing scheduler (max concurrent: %d)", maxConcurrent)
	scheduler := cron.NewScheduler(st, grafanaURL, artifactsPath, maxConcurrent)

	// Start scheduler
	log.Println("Starting scheduler...")
	if err := scheduler.Start(); err != nil {
		log.Printf("Failed to start scheduler: %v", err)
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	defer scheduler.Stop()
	log.Println("Scheduler started")

	// Create API handler
	log.Println("Creating API handler...")
	handler := api.NewHandler(st, scheduler)

	// Serve plugin
	log.Println("Starting plugin server...")
	return backend.Serve(backend.ServeOpts{
		CallResourceHandler: handler,
	})
}
