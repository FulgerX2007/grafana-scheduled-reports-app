package api

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/cron"
    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/store"
    "github.com/grafana/grafana-plugin-sdk-go/backend"
    "github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
    "gopkg.in/gomail.v2"
)

// Handler handles HTTP API requests
type Handler struct {
    store         *store.Store
    scheduler     *cron.Scheduler
    mux           *http.ServeMux
    contextCached bool
}

// NewHandler creates a new API handler
func NewHandler(st *store.Store, scheduler *cron.Scheduler) *Handler {
    h := &Handler{
        store:         st,
        scheduler:     scheduler,
        mux:           http.NewServeMux(),
        contextCached: false,
    }

    h.registerRoutes()
    return h
}

// registerRoutes registers all HTTP routes
func (h *Handler) registerRoutes() {
    h.mux.HandleFunc("/api/schedules", h.handleSchedules)
    h.mux.HandleFunc("/api/schedules/", h.handleSchedule)
    h.mux.HandleFunc("/api/runs/", h.handleRun)
    h.mux.HandleFunc("/api/settings", h.handleSettings)
    h.mux.HandleFunc("/api/service-account/status", h.handleServiceAccountStatus)
    h.mux.HandleFunc("/api/service-account/test-token", h.handleTestToken)
    h.mux.HandleFunc("/api/smtp/test", h.handleSMTPTest)
}

// CallResource implements backend.CallResourceHandler
func (h *Handler) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    // Cache the Grafana config context for background scheduler jobs
    // Only do this once on first request
    if !h.contextCached {
        h.scheduler.SetContext(ctx)
        h.contextCached = true
        log.Println("Cached Grafana config context for scheduler")
    }

    adapter := httpadapter.New(h.mux)
    return adapter.CallResource(ctx, req, sender)
}

// handleSchedules handles GET /api/schedules and POST /api/schedules
func (h *Handler) handleSchedules(w http.ResponseWriter, r *http.Request) {
    orgID := getOrgID(r)

    switch r.Method {
    case http.MethodGet:
        schedules, err := h.store.ListSchedules(orgID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        respondJSON(w, map[string]interface{}{"schedules": schedules})

    case http.MethodPost:
        var schedule model.Schedule
        if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        schedule.OrgID = orgID
        schedule.OwnerUserID = getUserID(r)

        // Validate recipient email domains against whitelist
        settings, err := h.store.GetSettings(orgID)
        if err != nil {
            http.Error(w, fmt.Sprintf("Failed to get settings: %v", err), http.StatusInternalServerError)
            return
        }
        if settings != nil {
            if err := model.ValidateRecipientDomains(schedule.Recipients, settings.Limits.AllowedDomains); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        // Validate CRON expression if provided
        if schedule.CronExpr != "" {
            if err := model.ValidateCronExpression(schedule.CronExpr); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        // Calculate and set next run time only if schedule is enabled
        if schedule.Enabled {
            nextRun := h.scheduler.CalculateNextRun(&schedule)
            schedule.NextRunAt = &nextRun
        } else {
            schedule.NextRunAt = nil
        }

        if err := h.store.CreateSchedule(&schedule); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        respondJSON(w, schedule)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// handleSchedule handles operations on a specific schedule
func (h *Handler) handleSchedule(w http.ResponseWriter, r *http.Request) {
    orgID := getOrgID(r)
    path := r.URL.Path

    // Parse schedule ID and action from path
    var scheduleID int64
    var action string

    // Path format: /api/schedules/{id} or /api/schedules/{id}/runs or /api/schedules/{id}/run
    if _, err := fmt.Sscanf(path, "/api/schedules/%d/%s", &scheduleID, &action); err != nil {
        // Try without action
        if _, err := fmt.Sscanf(path, "/api/schedules/%d", &scheduleID); err != nil {
            http.Error(w, "Invalid path", http.StatusBadRequest)
            return
        }
    }

    // Handle actions
    if action == "run" && r.Method == http.MethodPost {
        schedule, err := h.store.GetSchedule(orgID, scheduleID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }

        h.scheduler.ExecuteSchedule(schedule)
        respondJSON(w, map[string]string{"status": "started"})
        return
    }

    if action == "runs" && r.Method == http.MethodGet {
        runs, err := h.store.ListRuns(orgID, scheduleID)
        if err != nil {
            fmt.Printf("Error loading runs for schedule %d, org %d: %v\n", scheduleID, orgID, err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        respondJSON(w, map[string]interface{}{"runs": runs})
        return
    }

    // Handle CRUD operations
    switch r.Method {
    case http.MethodGet:
        schedule, err := h.store.GetSchedule(orgID, scheduleID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }
        respondJSON(w, schedule)

    case http.MethodPut:
        var schedule model.Schedule
        if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        schedule.ID = scheduleID
        schedule.OrgID = orgID

        // Validate recipient email domains against whitelist
        settings, err := h.store.GetSettings(orgID)
        if err != nil {
            http.Error(w, fmt.Sprintf("Failed to get settings: %v", err), http.StatusInternalServerError)
            return
        }
        if settings != nil {
            if err := model.ValidateRecipientDomains(schedule.Recipients, settings.Limits.AllowedDomains); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        // Validate CRON expression if provided
        if schedule.CronExpr != "" {
            if err := model.ValidateCronExpression(schedule.CronExpr); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
        }

        // Recalculate next run time only if schedule is enabled
        if schedule.Enabled {
            nextRun := h.scheduler.CalculateNextRun(&schedule)
            schedule.NextRunAt = &nextRun
        } else {
            schedule.NextRunAt = nil
        }

        if err := h.store.UpdateSchedule(&schedule); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        respondJSON(w, schedule)

    case http.MethodDelete:
        if err := h.store.DeleteSchedule(orgID, scheduleID); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// handleRun handles run-related operations
func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request) {
    orgID := getOrgID(r)
    path := r.URL.Path

    var runID int64
    var action string

    // Path format: /api/runs/{id}/artifact
    if _, err := fmt.Sscanf(path, "/api/runs/%d/%s", &runID, &action); err != nil {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }

    if action == "artifact" && r.Method == http.MethodGet {
        run, err := h.store.GetRun(orgID, runID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }

        // Serve artifact from database
        if len(run.ArtifactData) == 0 {
            http.Error(w, "Artifact not found", http.StatusNotFound)
            return
        }

        // Get schedule to retrieve the name for filename
        schedule, err := h.store.GetSchedule(orgID, run.ScheduleID)
        if err != nil {
            http.Error(w, "Schedule not found", http.StatusNotFound)
            return
        }

        // Generate filename from schedule name and timestamp
        timestamp := run.StartedAt.Format("2006-01-02-150405")
        filename := fmt.Sprintf("%s-%s.pdf", strings.ReplaceAll(schedule.Name, " ", "_"), timestamp)

        w.Header().Set("Content-Type", "application/pdf")
        w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
        w.Header().Set("Content-Length", fmt.Sprintf("%d", len(run.ArtifactData)))
        w.Write(run.ArtifactData)
        log.Printf("Served artifact from database: schedule_id=%d, run_id=%d, size=%d bytes", run.ScheduleID, run.ID, len(run.ArtifactData))
        return
    }

    http.Error(w, "Invalid action", http.StatusBadRequest)
}

// handleSettings handles settings operations
func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
    orgID := getOrgID(r)

    switch r.Method {
    case http.MethodGet:
        settings, err := h.store.GetSettings(orgID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        if settings == nil {
            // Return default settings with all configurations
            settings = &model.Settings{
                OrgID: orgID,
                SMTPConfig: &model.SMTPConfig{
                    Host:          "",
                    Port:          587,
                    Username:      "",
                    Password:      "",
                    From:          "",
                    UseTLS:        true,
                    SkipTLSVerify: false,
                },
                RendererConfig: model.RendererConfig{
                    GrafanaURL:        "", // User must configure this
                    TimeoutMS:         60000,
                    DelayMS:           5000, // 5 seconds to allow dashboards to load
                    ViewportWidth:     1920,
                    ViewportHeight:    1080,
                    DeviceScaleFactor: 2.0,
                    Headless:          true,
                    NoSandbox:         true,
                    DisableGPU:        true,
                    SkipTLSVerify:     true, // Allow self-signed certificates
                },
                Limits: model.Limits{
                    MaxRecipients:        50,
                    MaxAttachmentSizeMB:  25,
                    MaxConcurrentRenders: 5,
                    RetentionDays:        30,
                },
            }
        } else {
            // Ensure smtp_config has defaults if it's nil
            if settings.SMTPConfig == nil {
                settings.SMTPConfig = &model.SMTPConfig{
                    Host:          "",
                    Port:          587,
                    Username:      "",
                    Password:      "",
                    From:          "",
                    UseTLS:        true,
                    SkipTLSVerify: false,
                }
            }
        }
        respondJSON(w, settings)

    case http.MethodPost:
        var settings model.Settings
        if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        settings.OrgID = orgID

        if err := h.store.UpsertSettings(&settings); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        // Clear renderer cache to force recreation with new settings
        if err := h.scheduler.ClearRendererCache(orgID); err != nil {
            log.Printf("Warning: Failed to clear renderer cache for org %d: %v", orgID, err)
        }

        respondJSON(w, settings)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// Helper functions

func getOrgID(r *http.Request) int64 {
    // In a real plugin, this would come from the Grafana request context
    // For now, we'll try to get it from a header or default to 1
    orgIDStr := r.Header.Get("X-Grafana-Org-Id")
    if orgIDStr == "" {
        return 1
    }
    orgID, _ := strconv.ParseInt(orgIDStr, 10, 64)
    return orgID
}

func getUserID(r *http.Request) int64 {
    // In a real plugin, this would come from the Grafana request context
    userIDStr := r.Header.Get("X-Grafana-User-Id")
    if userIDStr == "" {
        return 1
    }
    userID, _ := strconv.ParseInt(userIDStr, 10, 64)
    return userID
}

func respondJSON(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

// handleServiceAccountStatus handles GET /api/service-account/status
func (h *Handler) handleServiceAccountStatus(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    ctx := r.Context()

    // Check environment variable first
    envToken := os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
    log.Printf("DEBUG: Environment variable GF_PLUGIN_APP_CLIENT_SECRET is set: %v (length: %d)", envToken != "", len(envToken))

    // Try to get the managed service account token from Grafana
    cfg := backend.GrafanaConfigFromContext(ctx)
    if cfg == nil {
        log.Printf("DEBUG: GrafanaConfigFromContext returned nil - context doesn't have Grafana config")
        respondJSON(
            w, map[string]interface{}{
                "status":           "unavailable",
                "has_token":        false,
                "env_token_set":    envToken != "",
                "env_token_length": len(envToken),
                "error":            "Grafana configuration not available in context",
                "requirements":     "Grafana 10.3+ with externalServiceAccounts feature toggle enabled",
            },
        )
        return
    }

    // Try to retrieve the service account token
    saToken, err := cfg.PluginAppClientSecret()
    if err != nil {
        log.Printf("ERROR: Failed to get service account token from cfg.PluginAppClientSecret(): %v", err)
        respondJSON(
            w, map[string]interface{}{
                "status":           "error",
                "has_token":        false,
                "env_token_set":    envToken != "",
                "env_token_length": len(envToken),
                "error":            fmt.Sprintf("Failed to retrieve token: %v", err),
                "requirements":     "Grafana 10.3+ with externalServiceAccounts feature toggle enabled",
                "solution":         "Enable feature toggle in grafana.ini: [feature_toggles] enable = externalServiceAccounts",
            },
        )
        return
    }

    if saToken == "" {
        log.Printf("WARNING: cfg.PluginAppClientSecret() returned empty token")
        respondJSON(
            w, map[string]interface{}{
                "status":           "not_configured",
                "has_token":        false,
                "env_token_set":    envToken != "",
                "env_token_length": len(envToken),
                "error":            "Service account token is empty",
                "requirements":     "Grafana 10.3+ with externalServiceAccounts feature toggle enabled",
                "solution":         "Restart Grafana to allow automatic service account creation",
            },
        )
        return
    }

    // Token is available!
    log.Printf("SUCCESS: Service account token retrieved successfully (length: %d)", len(saToken))
    respondJSON(
        w, map[string]interface{}{
            "status":           "active",
            "has_token":        true,
            "token_length":     len(saToken),
            "token_preview":    saToken[:min(20, len(saToken))] + "...",
            "env_token_set":    envToken != "",
            "env_token_length": len(envToken),
            "tokens_match":     envToken == saToken,
            "message":          "Service account token is configured and ready",
            "info":             "Grafana automatically manages this service account based on plugin.json IAM permissions",
        },
    )
}

// handleTestToken tests if the token can actually access Grafana API
func (h *Handler) handleTestToken(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    ctx := r.Context()

    // Get token
    var token string
    cfg := backend.GrafanaConfigFromContext(ctx)
    if cfg != nil {
        token, _ = cfg.PluginAppClientSecret()
    }
    if token == "" {
        token = os.Getenv("GF_PLUGIN_APP_CLIENT_SECRET")
    }

    if token == "" {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   "No token available to test",
            },
        )
        return
    }

    // Test the token by calling Grafana API
    grafanaURL := os.Getenv("GF_URL")
    if grafanaURL == "" {
        grafanaURL = "http://localhost:3000"
    }

    // Try to get current user info with the token
    apiURL := fmt.Sprintf("%s/api/user", grafanaURL)
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   fmt.Sprintf("Failed to create request: %v", err),
            },
        )
        return
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        respondJSON(
            w, map[string]interface{}{
                "success":       false,
                "error":         fmt.Sprintf("Failed to call Grafana API: %v", err),
                "token_length":  len(token),
                "token_preview": token[:min(20, len(token))] + "...",
            },
        )
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    respondJSON(
        w, map[string]interface{}{
            "success":       resp.StatusCode == http.StatusOK,
            "status_code":   resp.StatusCode,
            "response_body": string(body),
            "token_length":  len(token),
            "token_preview": token[:min(20, len(token))] + "...",
            "message":       "Token was sent to Grafana API /api/user endpoint",
        },
    )
}

// min returns the minimum of two integers
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// handleSMTPTest handles POST /api/smtp/test
func (h *Handler) handleSMTPTest(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse request body to get SMTP config
    var smtpConfig model.SMTPConfig
    if err := json.NewDecoder(r.Body).Decode(&smtpConfig); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate required fields
    if smtpConfig.Host == "" {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   "SMTP host is required",
            },
        )
        return
    }

    if smtpConfig.Port == 0 {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   "SMTP port is required",
            },
        )
        return
    }

    if smtpConfig.From == "" {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   "From address is required",
            },
        )
        return
    }

    // Create dialer
    dialer := gomail.NewDialer(smtpConfig.Host, smtpConfig.Port, smtpConfig.Username, smtpConfig.Password)

    // Configure TLS
    if smtpConfig.UseTLS {
        dialer.TLSConfig = &tls.Config{
            InsecureSkipVerify: smtpConfig.SkipTLSVerify,
            ServerName:         smtpConfig.Host,
        }
    } else {
        dialer.TLSConfig = &tls.Config{
            InsecureSkipVerify: true,
        }
        dialer.SSL = false
    }

    // Try to connect to SMTP server
    closer, err := dialer.Dial()
    if err != nil {
        respondJSON(
            w, map[string]interface{}{
                "success": false,
                "error":   fmt.Sprintf("Failed to connect to SMTP server: %v", err),
                "host":    smtpConfig.Host,
                "port":    smtpConfig.Port,
            },
        )
        return
    }
    defer closer.Close()

    respondJSON(
        w, map[string]interface{}{
            "success": true,
            "message": "Successfully connected to SMTP server",
            "host":    smtpConfig.Host,
            "port":    smtpConfig.Port,
            "tls":     smtpConfig.UseTLS,
        },
    )
}
