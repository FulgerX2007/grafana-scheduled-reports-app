package render

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/playwright-community/playwright-go"
	"github.com/yourusername/scheduled-reports-app/pkg/model"
)

// PlaywrightRenderer handles dashboard rendering using Playwright
type PlaywrightRenderer struct {
	grafanaURL string
	config     model.RendererConfig
	pw         *playwright.Playwright
	browser    playwright.Browser
	instanceID string
}

// NewPlaywrightRenderer creates a new Playwright renderer instance
func NewPlaywrightRenderer(grafanaURL string, config model.RendererConfig) *PlaywrightRenderer {
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
	if !config.Headless {
		config.Headless = true
	}

	instanceID := generateInstanceID()
	log.Printf("DEBUG: Created new PlaywrightRenderer instance: %s", instanceID)

	return &PlaywrightRenderer{
		grafanaURL: grafanaURL,
		config:     config,
		pw:         nil, // Lazy initialization
		browser:    nil,
		instanceID: instanceID,
	}
}

// getBrowser initializes or returns existing browser instance
func (r *PlaywrightRenderer) getBrowser() (playwright.Browser, error) {
	if r.browser != nil {
		return r.browser, nil
	}

	log.Printf("DEBUG: Initializing Playwright (instance: %s)", r.instanceID)

	// Set writable cache directory for Playwright
	// This is critical for Docker/Grafana environments where home directory is read-only
	playwrightCache := os.Getenv("PLAYWRIGHT_BROWSERS_PATH")
	if playwrightCache == "" {
		playwrightCache = "/tmp/.playwright-cache"
		os.Setenv("PLAYWRIGHT_BROWSERS_PATH", playwrightCache)
		log.Printf("DEBUG: Set PLAYWRIGHT_BROWSERS_PATH to: %s", playwrightCache)
	}

	// Ensure cache directory exists and is writable
	if err := os.MkdirAll(playwrightCache, 0755); err != nil {
		log.Printf("WARNING: Failed to create Playwright cache directory: %v", err)
	}

	// Don't try to install Playwright browsers in Alpine - use system Chromium instead
	log.Printf("DEBUG: Skipping Playwright browser installation (will use system Chromium)")

	// Initialize Playwright with custom driver path
	driverPath := os.Getenv("PLAYWRIGHT_DRIVER_PATH")
	if driverPath == "" {
		driverPath = "/tmp/.playwright-driver"
		os.Setenv("PLAYWRIGHT_DRIVER_PATH", driverPath)
		log.Printf("DEBUG: Set PLAYWRIGHT_DRIVER_PATH to: %s", driverPath)
	}

	// Ensure driver directory exists
	if err := os.MkdirAll(driverPath, 0755); err != nil {
		log.Printf("WARNING: Failed to create Playwright driver directory: %v", err)
	}

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start Playwright: %w\n\nPlaywright requires Node.js driver which may not be compatible with Alpine Linux.\nConsider using the legacy 'chromium' backend instead by setting backend='chromium' in settings.", err)
	}
	r.pw = pw

	// Find system Chromium binary
	chromiumPath := ""
	chromiumPaths := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	}
	for _, path := range chromiumPaths {
		if _, err := os.Stat(path); err == nil {
			chromiumPath = path
			log.Printf("DEBUG: Found system Chromium at: %s", chromiumPath)
			break
		}
	}

	// Configure browser launch options
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(r.config.Headless),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--no-first-run",
			"--no-default-browser-check",
			"--no-proxy-server",
			"--disable-breakpad",
		},
	}

	// Use system Chromium if available
	if chromiumPath != "" {
		launchOptions.ExecutablePath = playwright.String(chromiumPath)
		log.Printf("DEBUG: Using system Chromium: %s", chromiumPath)
	} else {
		log.Printf("WARNING: No system Chromium found, will try Playwright's bundled version")
	}

	// Skip TLS verification if configured
	if r.config.SkipTLSVerify {
		launchOptions.Args = append(launchOptions.Args, "--ignore-certificate-errors")
		log.Printf("WARNING: TLS certificate verification disabled for renderer")
	}

	// Launch Chromium browser
	log.Printf("DEBUG: Launching Chromium browser with Playwright...")
	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to launch Chromium: %w", err)
	}

	r.browser = browser
	log.Printf("Playwright Chromium browser initialized successfully")
	return browser, nil
}

// getServiceAccountToken retrieves the service account token from Grafana
func (r *PlaywrightRenderer) getServiceAccountToken(ctx context.Context) (string, error) {
	log.Printf("DEBUG: ========== TOKEN RETRIEVAL START ==========")

	// Priority 1: Try to get token from Grafana's managed service account
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
	log.Printf("DEBUG: Checking environment variable GF_PLUGIN_APP_CLIENT_SECRET...")
	token := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
	if token != "" {
		log.Printf("SUCCESS: Retrieved token from GF_PLUGIN_APP_CLIENT_SECRET env var (length: %d, preview: %s...)", len(token), token[:min(20, len(token))])
		log.Printf("DEBUG: ========== TOKEN RETRIEVAL END (via env var) ==========")
		return token, nil
	}
	log.Printf("WARNING: GF_PLUGIN_APP_CLIENT_SECRET environment variable is not set or empty")

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
			"3. Check Settings page (Apps → Scheduled Reports → Settings) for service account status",
	)
}

// RenderDashboard renders a dashboard to PDF using Playwright
func (r *PlaywrightRenderer) RenderDashboard(ctx context.Context, schedule *model.Schedule) ([]byte, error) {
	saToken, err := r.getServiceAccountToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("no service account token available: %w", err)
	}
	if saToken == "" {
		return nil, fmt.Errorf("service account token is empty")
	}

	// Build dashboard URL
	dashboardURL, err := r.buildDashboardURL(schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to build dashboard URL: %w", err)
	}

	// Parse URL to add kiosk and theme parameters
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

	log.Printf("DEBUG: Rendering dashboard URL: %s", dashboardURL)

	// Get or create browser
	browser, err := r.getBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	// Create new browser context with authentication
	contextOptions := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  r.config.ViewportWidth,
			Height: r.config.ViewportHeight,
		},
		DeviceScaleFactor: playwright.Float(r.config.DeviceScaleFactor),
		ExtraHttpHeaders: map[string]string{
			"Authorization": "Bearer " + saToken,
		},
		IgnoreHttpsErrors: playwright.Bool(r.config.SkipTLSVerify),
	}

	browserContext, err := browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}
	defer browserContext.Close()

	// Create new page
	page, err := browserContext.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set timeout
	page.SetDefaultTimeout(float64(r.config.TimeoutMS))

	// Navigate to dashboard
	log.Printf("DEBUG: Navigating to dashboard...")
	_, err = page.Goto(dashboardURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to dashboard: %w", err)
	}

	log.Printf("DEBUG: Page loaded, waiting for panels...")

	// Wait for panels to appear (with timeout)
	panelSelectors := []string{
		"div[class*='panel-container']",
		"div[data-panelid]",
		"div.react-grid-item",
	}

	for _, selector := range panelSelectors {
		_, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(30000),
			State:   playwright.WaitForSelectorStateVisible,
		})
		if err != nil {
			log.Printf("WARNING: Panel selector '%s' not found (may be normal for some dashboards): %v", selector, err)
		} else {
			log.Printf("DEBUG: Found panels with selector: %s", selector)
			break
		}
	}

	// STEP 1: Force all lazy-loaded content to load eagerly
	log.Printf("DEBUG: Forcing lazy-loaded content to load...")
	_, err = page.Evaluate(`() => {
		// Make all lazy images and iframes load immediately
		document.querySelectorAll('img[loading="lazy"]').forEach(img => img.loading = 'eager');
		document.querySelectorAll('iframe[loading="lazy"]').forEach(f => f.loading = 'eager');

		// Common lazy loading patterns
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

		// Force Grafana panels to render
		document.querySelectorAll('[data-panelid]').forEach(panel => {
			panel.style.visibility = 'visible';
		});
	}`)
	if err != nil {
		log.Printf("WARNING: Failed to force lazy load: %v", err)
	}

	// STEP 2: Progressive scrolling to trigger all content
	log.Printf("DEBUG: Scrolling through page to trigger lazy-loaded panels...")
	maxScrolls := 100
	scrollCount := 0

	for i := 0; i < maxScrolls; i++ {
		// Get current scroll height
		prevHeightResult, err := page.Evaluate("() => document.scrollingElement.scrollHeight")
		if err != nil {
			log.Printf("WARNING: Failed to get scroll height: %v", err)
			break
		}
		prevHeight := int(prevHeightResult.(float64))

		// Scroll down by one viewport
		_, _ = page.Evaluate("() => window.scrollBy(0, window.innerHeight)")
		scrollCount++

		// Wait for content to load
		time.Sleep(300 * time.Millisecond)

		// Check if page height increased
		curHeightResult, err := page.Evaluate("() => document.scrollingElement.scrollHeight")
		if err != nil {
			log.Printf("WARNING: Failed to get new scroll height: %v", err)
			break
		}
		curHeight := int(curHeightResult.(float64))

		log.Printf("DEBUG: Scroll %d: height %dpx -> %dpx", scrollCount, prevHeight, curHeight)

		// If height didn't change, we've reached the bottom
		if curHeight <= prevHeight {
			log.Printf("DEBUG: Reached bottom of page after %d scrolls", scrollCount)
			break
		}
	}

	if scrollCount >= maxScrolls {
		log.Printf("WARNING: Hit maximum scroll limit (%d scrolls)", maxScrolls)
	}

	// STEP 3: Scroll back to top
	log.Printf("DEBUG: Scrolling back to top...")
	_, _ = page.Evaluate("() => window.scrollTo(0, 0)")
	time.Sleep(300 * time.Millisecond)

	// STEP 4: Wait for network idle and loading indicators
	log.Printf("DEBUG: Waiting for network idle and panels to finish loading...")

	// Wait for network idle
	_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// Wait for loading indicators to disappear
	maxWaitTime := 30 * time.Second
	checkInterval := 500 * time.Millisecond
	elapsed := time.Duration(0)

	for elapsed < maxWaitTime {
		loadingResult, err := page.Evaluate(`() => {
			const spinners = document.querySelectorAll('.panel-loading, .fa-spinner, .fa-spin, [class*="loading"]');
			const skeletons = document.querySelectorAll('[class*="skeleton"]');
			const loadingPanels = document.querySelectorAll('[data-testid*="loading"]');

			const loadingText = Array.from(document.querySelectorAll('*')).filter(el => {
				const text = el.textContent.toLowerCase();
				return (text.includes('loading') || text.includes('waiting')) &&
				       el.offsetParent !== null &&
				       el.children.length === 0;
			});

			return spinners.length + skeletons.length + loadingPanels.length + loadingText.length;
		}`)

		if err == nil {
			loadingCount := int(loadingResult.(float64))
			if loadingCount == 0 {
				log.Printf("DEBUG: All panels finished loading")
				break
			}
			log.Printf("DEBUG: Still waiting for %d loading indicators...", loadingCount)
		}

		time.Sleep(checkInterval)
		elapsed += checkInterval
	}

	if elapsed >= maxWaitTime {
		log.Printf("WARNING: Timeout waiting for panels to load (waited %v)", maxWaitTime)
	}

	// STEP 5: Extra delay if configured
	if r.config.DelayMS > 0 {
		log.Printf("DEBUG: Applying configured delay: %dms", r.config.DelayMS)
		time.Sleep(time.Duration(r.config.DelayMS) * time.Millisecond)
	}

	// STEP 6: Get final content dimensions
	log.Printf("DEBUG: Calculating final content dimensions...")

	widthResult, err := page.Evaluate(`() => {
		const body = document.body;
		const html = document.documentElement;
		return Math.max(
			body.scrollWidth, body.offsetWidth,
			html.clientWidth, html.scrollWidth, html.offsetWidth
		);
	}`)
	contentWidthPx := float64(r.config.ViewportWidth)
	if err == nil {
		contentWidthPx = widthResult.(float64)
	}

	heightResult, err := page.Evaluate(`() => {
		const body = document.body;
		const html = document.documentElement;
		return Math.max(
			body.scrollHeight, body.offsetHeight,
			html.clientHeight, html.scrollHeight, html.offsetHeight
		);
	}`)
	contentHeightPx := float64(r.config.ViewportHeight)
	if err == nil {
		contentHeightPx = heightResult.(float64)
	}

	log.Printf("DEBUG: Final content dimensions: %.0fpx x %.0fpx", contentWidthPx, contentHeightPx)

	// Convert to inches (96 DPI)
	paperWidthInches := contentWidthPx / 96.0
	paperHeightInches := contentHeightPx / 96.0

	// Apply minimum dimensions
	if paperWidthInches < 8.0 {
		paperWidthInches = 8.0
	}
	if paperHeightInches < 6.0 {
		paperHeightInches = 6.0
	}

	// Apply maximum dimensions (Chrome PDF limit)
	if paperHeightInches > 200.0 {
		log.Printf("WARNING: Content height %.2f inches exceeds limit, capping at 200 inches", paperHeightInches)
		paperHeightInches = 200.0
	}
	if paperWidthInches > 200.0 {
		log.Printf("WARNING: Content width %.2f inches exceeds limit, capping at 200 inches", paperWidthInches)
		paperWidthInches = 200.0
	}

	log.Printf("DEBUG: PDF dimensions: %.2f\" x %.2f\"", paperWidthInches, paperHeightInches)

	// STEP 7: Generate PDF
	log.Printf("DEBUG: Generating PDF...")

	pdfOptions := playwright.PagePdfOptions{
		PrintBackground: playwright.Bool(true),
		PreferCSSPageSize: playwright.Bool(false),
		Width:  playwright.String(fmt.Sprintf("%.2fin", paperWidthInches)),
		Height: playwright.String(fmt.Sprintf("%.2fin", paperHeightInches)),
		Margin: &playwright.Margin{
			Top:    playwright.String("0in"),
			Bottom: playwright.String("0in"),
			Left:   playwright.String("0in"),
			Right:  playwright.String("0in"),
		},
		DisplayHeaderFooter: playwright.Bool(false),
		Scale: playwright.Float(1.0),
	}

	pdf, err := page.PDF(pdfOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	log.Printf("DEBUG: PDF generated successfully (%d bytes)", len(pdf))

	// Verify PDF
	if len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
		return nil, fmt.Errorf("output is not a valid PDF (got %d bytes)", len(pdf))
	}

	return pdf, nil
}

// Close closes the browser instance
func (r *PlaywrightRenderer) Close() error {
	if r.browser != nil {
		log.Printf("Closing Playwright browser (instance: %s)", r.instanceID)
		err := r.browser.Close()
		if err != nil {
			return err
		}
	}
	if r.pw != nil {
		log.Printf("Stopping Playwright (instance: %s)", r.instanceID)
		err := r.pw.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

// Name returns the backend name
func (r *PlaywrightRenderer) Name() string {
	return "playwright"
}

// buildDashboardURL constructs the Grafana dashboard URL
func (r *PlaywrightRenderer) buildDashboardURL(schedule *model.Schedule) (string, error) {
	baseURL := r.grafanaURL

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Handle Docker deployments
	if targetHost := os.Getenv("GRAFANA_HOSTNAME"); targetHost != "" {
		if u.Host == "localhost:3000" || u.Host == "127.0.0.1:3000" || u.Host == "localhost" || u.Host == "127.0.0.1" {
			if u.Port() != "" {
				u.Host = fmt.Sprintf("%s:%s", targetHost, u.Port())
			} else {
				u.Host = targetHost
			}
			log.Printf("DEBUG: Converted localhost to %s for Docker deployment", u.Host)
		}
	}

	// Preserve any subpath from base URL
	basePath := u.Path
	if basePath == "" || basePath == "/" {
		basePath = ""
	}

	u.Path = fmt.Sprintf("%s/d/%s", basePath, schedule.DashboardUID)

	q := u.Query()
	q.Set("from", schedule.RangeFrom)
	q.Set("to", schedule.RangeTo)
	q.Set("kiosk", "1")
	q.Set("tz", schedule.Timezone)

	// Add dashboard variables
	for _, variable := range schedule.Variables {
		q.Add("var-"+variable.Name, variable.Value)
	}

	u.RawQuery = q.Encode()

	return u.String(), nil
}
