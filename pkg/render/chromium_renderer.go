package render

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/yourusername/scheduled-reports-app/pkg/model"
)

// ChromiumRenderer handles dashboard rendering using Chromium
type ChromiumRenderer struct {
	grafanaURL string
	config     model.RendererConfig
	browser    *rod.Browser
	instanceID string // Unique ID for this renderer instance
	profileDir string // Unique profile directory for this instance
}

// findChromeBinary tries to locate Chrome binary in common locations
func (r *ChromiumRenderer) findChromeBinary() string {
	// Get current working directory for debugging
	cwd, _ := os.Getwd()
	log.Printf("DEBUG: Searching for Chrome binary. Current working directory: %s", cwd)

	// List of common Chrome binary paths to check (in order of preference)
	candidatePaths := []string{
		// Bundled Chrome (relative to plugin directory)
		"./chrome-linux64/chrome",
		"chrome-linux64/chrome",
		"../chrome-linux64/chrome", // In case cwd is inside plugin dir

		// Try absolute path relative to plugin installation
		"/var/lib/grafana/plugins/scheduled-reports-app/chrome-linux64/chrome",

		// System Chrome installations
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/snap/bin/chromium",

		// macOS
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}

	for _, path := range candidatePaths {
		log.Printf("DEBUG: Checking Chrome path: %s", path)
		if info, err := os.Stat(path); err == nil {
			// Check if file is executable
			if info.Mode()&0111 != 0 {
				log.Printf("DEBUG: Found executable Chrome binary at: %s", path)
				return path
			} else {
				log.Printf("DEBUG: File exists but is not executable: %s", path)
			}
		}
	}

	log.Printf("DEBUG: No Chrome binary found in any candidate paths")
	return ""
}

// generateInstanceID creates a unique identifier for this renderer instance
func generateInstanceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewChromiumRenderer creates a new Chromium renderer instance
func NewChromiumRenderer(grafanaURL string, config model.RendererConfig) *ChromiumRenderer {
	// Set defaults
	if config.ViewportWidth == 0 {
		config.ViewportWidth = 1920
	}
	if config.ViewportHeight == 0 {
		config.ViewportHeight = 1080
	}
	if config.TimeoutMS == 0 {
		config.TimeoutMS = 30000
	}
	if config.DeviceScaleFactor == 0 {
		config.DeviceScaleFactor = 2.0
	}
	// Enable headless by default
	if !config.Headless {
		config.Headless = true
	}

	// Generate unique instance ID and profile directory
	instanceID := generateInstanceID()
	profileDir := fmt.Sprintf("/tmp/.chromium-profile-%s", instanceID)

	log.Printf("DEBUG: Created new ChromiumRenderer instance: %s, profile dir: %s", instanceID, profileDir)

	return &ChromiumRenderer{
		grafanaURL: grafanaURL,
		config:     config,
		browser:    nil, // Lazy initialization
		instanceID: instanceID,
		profileDir: profileDir,
	}
}

// getBrowser initializes or returns existing browser instance
func (r *ChromiumRenderer) getBrowser() (*rod.Browser, error) {
	if r.browser != nil {
		return r.browser, nil
	}

	// Set environment variables for Chrome crashpad handler
	// These directories must be writable for Chrome's crash reporting system
	os.Setenv("XDG_CONFIG_HOME", "/tmp/.chromium-config")
	os.Setenv("XDG_CACHE_HOME", "/tmp/.chromium-cache")

	// Ensure directories exist and are writable
	os.MkdirAll("/tmp/.chromium-config", 0755)
	os.MkdirAll("/tmp/.chromium-cache", 0755)
	os.MkdirAll("/tmp/chrome-crashes", 0755)
	os.MkdirAll(r.profileDir, 0755)

	log.Printf("DEBUG: Created writable directories for Chrome crashpad handler (instance: %s)", r.instanceID)

	// Configure launcher
	l := launcher.New()

	// Determine Chrome binary path
	chromePath := r.config.ChromiumPath

	// If not configured, try to find Chrome automatically
	if chromePath == "" {
		chromePath = r.findChromeBinary()
		if chromePath != "" {
			log.Printf("Auto-detected Chrome binary at: %s", chromePath)
		}
	}

	// Set Chrome binary path
	if chromePath != "" {
		l = l.Bin(chromePath)
		log.Printf("Using Chrome binary: %s", chromePath)
	} else {
		log.Printf("WARNING: No Chrome binary specified. Attempting to use system default or auto-download.")
		log.Printf("To avoid this, configure 'Chromium Path' in plugin Settings to: ./chrome-linux64/chrome")
	}

	// Essential Chrome flags for server environments
	// These are always enabled regardless of config
	l = l.Set("no-sandbox")               // Required for running as root or in Docker
	l = l.Set("disable-setuid-sandbox")   // Required for running as root or in Docker
	l = l.Set("disable-dev-shm-usage")    // Use /tmp instead of /dev/shm (prevents crashes in Docker)
	l = l.Set("disable-gpu")              // Disable GPU (not available in headless)
	l = l.Set("no-first-run")             // Skip first-run wizards
	l = l.Set("no-default-browser-check") // Don't check if Chrome is default browser
	l = l.Set("no-proxy-server")          // Avoid proxy issues

	// Crashpad handler configuration - fixes "chrome_crashpad_handler: --database is required" error
	l = l.Set("crash-dumps-dir", "/tmp/chrome-crashes") // Specify writable crash dump directory
	l = l.Set("disable-breakpad")                       // Disable breakpad crash reporter

	// User data directory - must be writable and UNIQUE per instance to avoid SingletonLock errors
	l = l.Set("user-data-dir", r.profileDir)

	// Use new headless mode for better PDF generation
	l = l.Headless(true)
	l = l.Set("headless", "new")

	// Skip TLS verification if configured
	if r.config.SkipTLSVerify {
		l = l.Set("ignore-certificate-errors")
		log.Printf("WARNING: TLS certificate verification disabled for renderer")
	}

	log.Printf("Chrome flags: no-sandbox, disable-setuid-sandbox, disable-dev-shm-usage, disable-gpu, crash-dumps-dir=/tmp/chrome-crashes, user-data-dir=%s, headless=new", r.profileDir)
	log.Printf("Environment: XDG_CONFIG_HOME=/tmp/.chromium-config, XDG_CACHE_HOME=/tmp/.chromium-cache")
	log.Printf("Instance ID: %s (unique profile prevents SingletonLock conflicts)", r.instanceID)

	// Launch browser
	log.Printf("DEBUG: Launching Chrome browser...")
	launchURL, err := l.Launch()
	if err != nil {
		log.Printf("ERROR: Failed to launch Chrome: %v", err)

		// Check common issues
		if chromePath != "" {
			if _, statErr := os.Stat(chromePath); statErr != nil {
				log.Printf("ERROR: Chrome binary not accessible: %v", statErr)
			}
		}

		// Provide helpful error message if Chrome not found
		if chromePath == "" {
			return nil, fmt.Errorf("failed to launch browser: %w\n\nChrome/Chromium not found. Please configure 'Chromium Path' in plugin Settings.\n\nOptions:\n  1. If you have bundled Chrome: set path to './chrome-linux64/chrome'\n  2. If using system Chrome: install via 'apt-get install chromium-browser' or 'yum install chromium'\n  3. Download Chrome for Testing from: https://googlechromelabs.github.io/chrome-for-testing/", err)
		}
		return nil, fmt.Errorf("failed to launch browser at '%s': %w\n\nPlease verify:\n  1. Chrome binary exists and is executable: chmod +x %s\n  2. Required system dependencies are installed\n  3. Sufficient disk space in /tmp for Chrome profile\n  4. If in Docker: ensure --security-opt seccomp=unconfined or use --no-sandbox", chromePath, err, chromePath)
	}

	log.Printf("DEBUG: Chrome launched successfully, debug URL: %s", launchURL)

	browser := rod.New().ControlURL(launchURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	r.browser = browser
	log.Printf("Chromium browser initialized successfully")
	return browser, nil
}

// getServiceAccountToken retrieves the service account token from Grafana's managed service accounts
func (r *ChromiumRenderer) getServiceAccountToken(ctx context.Context) (string, error) {
	log.Printf("DEBUG: ========== TOKEN RETRIEVAL START ==========")

	// Priority 1: Try to get token from Grafana's managed service account (preferred method)
	// Grafana 10.3+ automatically creates a service account for the plugin based on plugin.json IAM configuration
	cfg := backend.GrafanaConfigFromContext(ctx)
	if cfg != nil {
		log.Printf("DEBUG: Grafana config available in context - trying cfg.PluginAppClientSecret()")
		token, err := cfg.PluginAppClientSecret()
		if err != nil {
			log.Printf("ERROR: cfg.PluginAppClientSecret() returned error: %v", err)
		}
		if token != "" {
			log.Printf("SUCCESS: Retrieved token from Grafana SDK (length: %d, preview: %s...)", len(token), token[:min(20, len(token))])
			log.Printf("DEBUG: ========== TOKEN RETRIEVAL END (via SDK) ==========")
			return token, nil
		}
		log.Printf("WARNING: cfg.PluginAppClientSecret() returned empty token")
	} else {
		log.Printf("DEBUG: Grafana config NOT available in context (expected for background jobs)")
	}

	// Priority 2: Check environment variable GF_PLUGIN_APP_CLIENT_SECRET
	// This is set by Grafana when the plugin starts if managed service accounts are enabled
	log.Printf("DEBUG: Checking environment variable GF_PLUGIN_APP_CLIENT_SECRET...")
	token := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
	if token != "" {
		log.Printf("SUCCESS: Retrieved token from GF_PLUGIN_APP_CLIENT_SECRET env var (length: %d, preview: %s...)", len(token), token[:min(20, len(token))])
		log.Printf("DEBUG: ========== TOKEN RETRIEVAL END (via env var) ==========")
		return token, nil
	}
	log.Printf("WARNING: GF_PLUGIN_APP_CLIENT_SECRET environment variable is not set or empty")

	// No token available - managed service accounts not working
	log.Printf("ERROR: ========== TOKEN RETRIEVAL FAILED - NO TOKEN FOUND ==========")
	return "", fmt.Errorf(
		"no service account token available\n\n" +
			"Grafana managed service accounts are not configured correctly.\n\n" +
			"Requirements:\n" +
			"- Grafana 10.3 or later\n" +
			"- Feature toggle enabled: [feature_toggles] enable = externalServiceAccounts\n" +
			"- Plugin must be restarted after installation\n\n" +
			"Steps to fix:\n" +
			"1. Add to grafana.ini: [feature_toggles] enable = externalServiceAccounts\n" +
			"2. Restart Grafana: sudo systemctl restart grafana-server\n" +
			"3. Check Settings page (Apps → Scheduled Reports → Settings) for service account status\n\n" +
			"The plugin.json already has IAM permissions configured, so Grafana will automatically\n" +
			"create a service account when the feature toggle is enabled.",
	)
}

// RenderDashboard renders a dashboard to PDF using Chromium
// RenderDashboard renders a dashboard to PDF using Chromium (rod).
func (r *ChromiumRenderer) RenderDashboard(ctx context.Context, schedule *model.Schedule) ([]byte, error) {
	saToken, err := r.getServiceAccountToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("no service account token available: %w", err)
	}
	if saToken == "" {
		return nil, fmt.Errorf("service account token is empty; configure it in plugin settings or enable managed service accounts")
	}

	// Build final URL and *force* orgId to avoid redirects that drop Authorization
	dashboardURL, err := r.buildDashboardURL(schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to build dashboard URL: %w", err)
	}
	{
		u, err := url.Parse(dashboardURL)
		if err != nil {
			return nil, fmt.Errorf("invalid dashboard URL: %w", err)
		}
		q := u.Query()
		if q.Get("kiosk") == "" {
			q.Set("kiosk", "true")
		}
		if q.Get("theme") == "" {
			q.Set("theme", "light")
		}
		u.RawQuery = q.Encode()
		dashboardURL = u.String()
	}

	browser, err := r.getBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set global headers BEFORE any navigation. Key/value pairs, flat slice.
	kv := []string{"Authorization", "Bearer " + saToken}
	cleanup, err := page.SetExtraHeaders(kv)
	if err != nil {
		return nil, fmt.Errorf("failed to set global headers: %w", err)
	}
	defer cleanup()

	// Set viewport to configured dimensions
	if err := page.SetViewport(
		&proto.EmulationSetDeviceMetricsOverride{
			Width:             r.config.ViewportWidth,
			Height:            r.config.ViewportHeight,
			DeviceScaleFactor: r.config.DeviceScaleFactor,
			Mobile:            false,
		},
	); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	// Timeout wrapper
	page = page.Timeout(time.Duration(r.config.TimeoutMS) * time.Millisecond)

	// Navigate
	if err := page.Navigate(dashboardURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to dashboard: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Wait for panels to exist (not fatal if it races)
	_, _ = page.Timeout(30 * time.Second).Race().
		Element("div[class*='panel-container']").
		Element("div[data-panelid]").
		Element("div.react-grid-item").
		Do()

	// STEP 1: Force all lazy-loaded content to load eagerly
	log.Printf("DEBUG: Forcing lazy-loaded content to load eagerly...")
	_, _ = page.Eval(`() => {
		// Make all lazy images and iframes load immediately
		document.querySelectorAll('img[loading="lazy"]').forEach(img => img.loading = 'eager');
		document.querySelectorAll('iframe[loading="lazy"]').forEach(f => f.loading = 'eager');

		// Common lazy loading patterns - force load data-src attributes
		document.querySelectorAll('[data-src]').forEach(el => {
			if (!el.src && el.dataset.src) {
				el.src = el.dataset.src;
			}
		});
		document.querySelectorAll('source[data-srcset]').forEach(s => {
			if (s.dataset.srcset) {
				s.srcset = s.dataset.srcset;
			}
		});

		// Force Grafana panels to render by marking all as visible
		document.querySelectorAll('[data-panelid]').forEach(panel => {
			// Trigger intersection observer by making panel "visible"
			panel.style.visibility = 'visible';
		});
	}`)

	// STEP 2: Wait for panels to render with the tall viewport
	time.Sleep(time.Duration(r.config.DelayMS) * time.Millisecond)
	log.Printf("DEBUG: Waited %dms for panels to render in tall viewport", r.config.DelayMS)

	// STEP 3: Wait for network idle and all panel queries to complete
	log.Printf("DEBUG: Waiting for network to settle and panels to finish loading...")
	page.WaitIdle(5 * time.Second) // Wait for initial network idle

	// Additional wait for panel queries - pragmatic timeout
	// Note: Some panels may never finish loading (misconfigured datasources, etc.)
	// We wait a reasonable time, then proceed to avoid indefinite hangs
	maxWaitTime := 30 * time.Second  // Pragmatic timeout: 30 seconds
	checkInterval := 1 * time.Second
	elapsed := time.Duration(0)
	stableCount := 0                  // Count how many times we see 0 loading indicators
	requiredStableChecks := 3         // Require 3 consecutive checks with 0 indicators
	unchangedCount := 0               // Track if loading count stops changing
	lastLoadingCount := -1

	for elapsed < maxWaitTime {
		// Enhanced check for loading indicators
		loadingResult, err := page.Eval(`() => {
			// Count Grafana-specific loading indicators
			const spinners = document.querySelectorAll('.panel-loading, .fa-spinner, .fa-spin, [class*="loading"]');
			const skeletons = document.querySelectorAll('[class*="skeleton"]');
			const loadingPanels = document.querySelectorAll('[data-testid*="loading"]');

			// Grafana-specific: panels with "loading" in aria-label
			const ariaLoading = document.querySelectorAll('[aria-label*="loading" i]');

			// Check for empty panels that might still be loading
			const emptyPanels = Array.from(document.querySelectorAll('[data-panelid]')).filter(panel => {
				// Check if panel is visible but has no rendered content
				const hasCanvas = panel.querySelector('canvas');
				const hasText = panel.textContent.trim().length > 0;
				const hasImage = panel.querySelector('img');
				const hasSvg = panel.querySelector('svg');
				return panel.offsetParent !== null && !hasCanvas && !hasText && !hasImage && !hasSvg;
			});

			// Check for "Loading" or "Waiting" text in visible elements
			const loadingText = Array.from(document.querySelectorAll('*')).filter(el => {
				const text = el.textContent.toLowerCase();
				return (text.includes('loading') || text.includes('waiting') || text.includes('querying')) &&
				       el.offsetParent !== null &&
				       el.children.length === 0; // Only leaf nodes
			});

			return {
				spinners: spinners.length,
				skeletons: skeletons.length,
				loadingPanels: loadingPanels.length,
				ariaLoading: ariaLoading.length,
				emptyPanels: emptyPanels.length,
				loadingText: loadingText.length,
				total: spinners.length + skeletons.length + loadingPanels.length + ariaLoading.length + emptyPanels.length + loadingText.length
			};
		}`)

		if err == nil {
			result := loadingResult.Value.Get("total")
			loadingCount := int(result.Num())

			if loadingCount == 0 {
				stableCount++
				log.Printf("DEBUG: No loading indicators found (stable check %d/%d)", stableCount, requiredStableChecks)

				if stableCount >= requiredStableChecks {
					log.Printf("DEBUG: All panels finished loading (verified with %d stable checks)", requiredStableChecks)
					break
				}
			} else {
				stableCount = 0 // Reset if we see loading indicators again

				// Check if loading count is stuck (not changing)
				if loadingCount == lastLoadingCount {
					unchangedCount++
					if unchangedCount >= 5 {
						// Loading count hasn't changed for 5 seconds - likely stuck panels
						log.Printf("DEBUG: Loading count unchanged for %d checks (%d indicators) - likely stuck panels, proceeding anyway",
							unchangedCount, loadingCount)
						break
					}
				} else {
					unchangedCount = 0
					lastLoadingCount = loadingCount
				}

				// Log detailed breakdown
				spinners := int(loadingResult.Value.Get("spinners").Num())
				skeletons := int(loadingResult.Value.Get("skeletons").Num())
				panels := int(loadingResult.Value.Get("loadingPanels").Num())
				aria := int(loadingResult.Value.Get("ariaLoading").Num())
				empty := int(loadingResult.Value.Get("emptyPanels").Num())
				text := int(loadingResult.Value.Get("loadingText").Num())

				log.Printf("DEBUG: Still loading - spinners:%d skeletons:%d panels:%d aria:%d empty:%d text:%d (total:%d)",
					spinners, skeletons, panels, aria, empty, text, loadingCount)
			}
		}

		time.Sleep(checkInterval)
		elapsed += checkInterval
	}

	if elapsed >= maxWaitTime {
		log.Printf("WARNING: Timeout waiting for all panels to load (waited %v, %d stable checks achieved)", maxWaitTime, stableCount)
		log.Printf("WARNING: Some panels may be incomplete in the PDF")
	}

	// STEP 5: Extra delay if configured
	if r.config.DelayMS > 0 {
		log.Printf("DEBUG: Applying configured delay: %dms", r.config.DelayMS)
		time.Sleep(time.Duration(r.config.DelayMS) * time.Millisecond)
	}

	// STEP 6: Get final content dimensions
	log.Printf("DEBUG: Calculating final content dimensions...")

	// Get actual rendered content size using JavaScript
	contentWidthPx := float64(r.config.ViewportWidth)   // Fallback
	contentHeightPx := float64(r.config.ViewportHeight) // Fallback

	widthResult, err := page.Eval(`() => {
		// Get the maximum of scroll width, offset width, and client width
		const body = document.body;
		const html = document.documentElement;
		return Math.max(
			body.scrollWidth, body.offsetWidth,
			html.clientWidth, html.scrollWidth, html.offsetWidth
		);
	}`)
	if err == nil {
		contentWidthPx = widthResult.Value.Num()
	} else {
		log.Printf("WARNING: Failed to get content width: %v", err)
	}

	finalHeightResult, err := page.Eval(`() => {
		// Get the maximum of scroll height, offset height, and client height
		const body = document.body;
		const html = document.documentElement;
		return Math.max(
			body.scrollHeight, body.offsetHeight,
			html.clientHeight, html.scrollHeight, html.offsetHeight
		);
	}`)
	if err == nil {
		contentHeightPx = finalHeightResult.Value.Num()
	} else {
		log.Printf("WARNING: Failed to get content height: %v", err)
	}

	log.Printf("DEBUG: Final content dimensions: %.0fpx x %.0fpx", contentWidthPx, contentHeightPx)

	// Convert actual content dimensions to inches (Chrome uses 96 DPI)
	paperWidthInches := contentWidthPx / 96.0
	paperHeightInches := contentHeightPx / 96.0

	// Apply minimum dimensions (prevent tiny PDFs)
	if paperWidthInches < 8.0 {
		log.Printf("DEBUG: Content width %.2f\" too small, setting to 8\"", paperWidthInches)
		paperWidthInches = 8.0
	}
	if paperHeightInches < 6.0 {
		log.Printf("DEBUG: Content height %.2f\" too small, setting to 6\"", paperHeightInches)
		paperHeightInches = 6.0
	}

	// Apply maximum dimensions (Chrome PDF has a limit of ~200 inches)
	if paperHeightInches > 200.0 {
		log.Printf("WARNING: Content height %.2f inches exceeds Chrome limit (200\"), capping at 200 inches", paperHeightInches)
		log.Printf("WARNING: Some content may be cut off. Consider reducing viewport height or dashboard height.")
		paperHeightInches = 200.0
	}
	if paperWidthInches > 200.0 {
		log.Printf("WARNING: Content width %.2f inches exceeds Chrome limit (200\"), capping at 200 inches", paperWidthInches)
		paperWidthInches = 200.0
	}

	log.Printf("DEBUG: PDF dimensions: %.2f\" x %.2f\" (%.0fpx x %.0fpx @ 96 DPI)",
		paperWidthInches, paperHeightInches,
		paperWidthInches*96, paperHeightInches*96)

	// STEP 7: Generate PDF with full content capture
	// Use zero margins to capture exact content dimensions
	f := func(x float64) *float64 { return &x }
	stream, err := page.PDF(
		&proto.PagePrintToPDF{
			PrintBackground:     true,
			PreferCSSPageSize:   false, // Use our calculated dimensions
			PaperWidth:          f(paperWidthInches),
			PaperHeight:         f(paperHeightInches),
			MarginTop:           f(0.0), // Zero margins for full content
			MarginBottom:        f(0.0),
			MarginLeft:          f(0.0),
			MarginRight:         f(0.0),
			DisplayHeaderFooter: false,
			Scale:               f(1.0),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	log.Printf("DEBUG: PDF generated successfully")

	pdf, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF stream: %w", err)
	}
	if len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
		return nil, fmt.Errorf("output is not a PDF (got %d bytes)", len(pdf))
	}
	return pdf, nil
}

// Close closes the browser instance
func (r *ChromiumRenderer) Close() error {
	if r.browser != nil {
		log.Printf("Closing Chromium browser (instance: %s)", r.instanceID)
		err := r.browser.Close()

		// Clean up profile directory to free disk space
		if r.profileDir != "" {
			log.Printf("DEBUG: Cleaning up profile directory: %s", r.profileDir)
			os.RemoveAll(r.profileDir)
		}

		return err
	}
	return nil
}

// Name returns the backend name
func (r *ChromiumRenderer) Name() string {
	return "chromium"
}

// buildDashboardURL constructs the Grafana dashboard URL
func (r *ChromiumRenderer) buildDashboardURL(schedule *model.Schedule) (string, error) {
	// Use configured grafanaURL
	baseURL := r.grafanaURL

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Only convert localhost to grafana hostname if explicitly configured to do so
	// This is needed for Docker deployments where the plugin runs in a separate container
	// For non-Docker deployments, use the actual configured hostname
	// Note: This conversion should only happen if GRAFANA_HOSTNAME env var is set
	if targetHost := os.Getenv("GRAFANA_HOSTNAME"); targetHost != "" {
		if u.Host == "localhost:3000" || u.Host == "127.0.0.1:3000" || u.Host == "localhost" || u.Host == "127.0.0.1" {
			// Parse target to preserve protocol
			if u.Port() != "" {
				u.Host = fmt.Sprintf("%s:%s", targetHost, u.Port())
			} else {
				u.Host = targetHost
			}
			log.Printf("DEBUG: Converted localhost to %s for Docker deployment", u.Host)
		}
	}

	// Preserve any subpath from base URL (e.g., /dna from root_url)
	basePath := u.Path
	if basePath == "" || basePath == "/" {
		basePath = ""
	}

	u.Path = fmt.Sprintf("%s/d/%s", basePath, schedule.DashboardUID)

	q := u.Query()
	q.Set("from", schedule.RangeFrom)
	q.Set("to", schedule.RangeTo)
	q.Set("kiosk", "1") // Hide menu, header, and time picker
	//q.Set("orgId", strconv.FormatInt(schedule.OrgID, 10))
	q.Set("tz", schedule.Timezone)

	// Add dashboard variables (using Add to support duplicate variable names)
	for _, variable := range schedule.Variables {
		q.Add("var-"+variable.Name, variable.Value)
	}

	u.RawQuery = q.Encode()

	return u.String(), nil
}
