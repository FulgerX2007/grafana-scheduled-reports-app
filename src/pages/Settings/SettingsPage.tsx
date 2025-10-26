import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Button, Field, Input, Switch, FieldSet, Form, TextArea } from '@grafana/ui';
import { Settings, SMTPConfig, RendererConfig, Limits } from '../../types/types';
import { getBackendSrv, getAppEvents } from '@grafana/runtime';
import { AppEvents } from '@grafana/data';

interface SettingsPageProps {
  onNavigate: (page: string) => void;
}

// Default configurations to ensure all fields have valid values
const DEFAULT_SMTP_CONFIG: SMTPConfig = {
  host: '',
  port: 587,
  username: '',
  password: '',
  from: '',
  use_tls: true,
  skip_tls_verify: false,
};

const DEFAULT_RENDERER_CONFIG: RendererConfig = {
  url: '',
  timeout_ms: 60000,
  delay_ms: 5000,
  viewport_width: 1920,
  viewport_height: 1080,
  device_scale_factor: 2.0,
  headless: true,
  no_sandbox: true,
  disable_gpu: true,
  skip_tls_verify: true,
};

const DEFAULT_LIMITS: Limits = {
  max_recipients: 50,
  max_attachment_size_mb: 25,
  max_concurrent_renders: 5,
  retention_days: 30,
};

export const SettingsPage: React.FC<SettingsPageProps> = ({ onNavigate }) => {
  const styles = useStyles2(getStyles);
  const [settings, setSettings] = useState<Partial<Settings>>({
    smtp_config: DEFAULT_SMTP_CONFIG,
    renderer_config: DEFAULT_RENDERER_CONFIG,
    limits: DEFAULT_LIMITS,
  });
  const [serviceAccountStatus, setServiceAccountStatus] = useState<any>(null);
  const [chromiumCheckResult, setChromiumCheckResult] = useState<any>(null);
  const [isCheckingChromium, setIsCheckingChromium] = useState(false);
  const [smtpTestResult, setSmtpTestResult] = useState<any>(null);
  const [isTestingSMTP, setIsTestingSMTP] = useState(false);

  useEffect(() => {
    loadSettings();
    loadServiceAccountStatus();
  }, []);

  const loadSettings = async () => {
    try {
      const response = await getBackendSrv().get('/api/plugins/fulgerx2007-scheduledreports-app/resources/api/settings');
      if (response) {
        // Merge loaded settings with defaults to ensure all fields are present
        setSettings({
          ...response,
          smtp_config: { ...DEFAULT_SMTP_CONFIG, ...response.smtp_config },
          renderer_config: { ...DEFAULT_RENDERER_CONFIG, ...response.renderer_config },
          limits: { ...DEFAULT_LIMITS, ...response.limits },
        });
      }
    } catch (error) {
      console.error('Failed to load settings:', error);
    }
  };

  const loadServiceAccountStatus = async () => {
    try {
      const response = await getBackendSrv().get('/api/plugins/fulgerx2007-scheduledreports-app/resources/api/service-account/status');
      setServiceAccountStatus(response);
    } catch (error) {
      console.error('Failed to load service account status:', error);
      setServiceAccountStatus({
        status: 'error',
        error: 'Failed to load status',
        has_token: false
      });
    }
  };

  const handleSubmit = async () => {
    const appEvents = getAppEvents();
    try {
      await getBackendSrv().post('/api/plugins/fulgerx2007-scheduledreports-app/resources/api/settings', settings);
      appEvents.publish({
        type: AppEvents.alertSuccess.name,
        payload: ['Settings saved successfully'],
      });
    } catch (error) {
      console.error('Failed to save settings:', error);
      appEvents.publish({
        type: AppEvents.alertError.name,
        payload: ['Failed to save settings'],
      });
    }
  };

  const updateSMTP = (field: keyof SMTPConfig, value: any) => {
    setSettings({
      ...settings,
      smtp_config: {
        ...DEFAULT_SMTP_CONFIG,
        ...settings.smtp_config,
        [field]: value
      },
    });
  };

  const updateRenderer = (field: keyof RendererConfig, value: any) => {
    setSettings({
      ...settings,
      renderer_config: {
        ...DEFAULT_RENDERER_CONFIG,
        ...settings.renderer_config,
        [field]: value
      },
    });
  };

  const updateLimits = (field: keyof Limits, value: any) => {
    setSettings({
      ...settings,
      limits: {
        ...DEFAULT_LIMITS,
        ...settings.limits,
        [field]: value
      },
    });
  };

  const handleCheckChromium = async () => {
    const appEvents = getAppEvents();
    setIsCheckingChromium(true);
    setChromiumCheckResult(null);

    try {
      const response = await getBackendSrv().post(
        '/api/plugins/fulgerx2007-scheduledreports-app/resources/api/chromium/check-version',
        { chromium_path: settings.renderer_config?.chromium_path || '' }
      );

      setChromiumCheckResult(response);

      if (response.success) {
        appEvents.publish({
          type: AppEvents.alertSuccess.name,
          payload: ['Chromium check successful: ' + response.version],
        });
      } else {
        appEvents.publish({
          type: AppEvents.alertError.name,
          payload: ['Chromium check failed: ' + response.error],
        });
      }
    } catch (error) {
      console.error('Failed to check Chromium:', error);
      setChromiumCheckResult({
        success: false,
        error: 'Failed to check Chromium: ' + error,
      });
      appEvents.publish({
        type: AppEvents.alertError.name,
        payload: ['Failed to check Chromium'],
      });
    } finally {
      setIsCheckingChromium(false);
    }
  };

  const handleTestSMTP = async () => {
    const appEvents = getAppEvents();
    setIsTestingSMTP(true);
    setSmtpTestResult(null);

    try {
      const response = await getBackendSrv().post(
        '/api/plugins/fulgerx2007-scheduledreports-app/resources/api/smtp/test',
        settings.smtp_config
      );

      setSmtpTestResult(response);

      if (response.success) {
        appEvents.publish({
          type: AppEvents.alertSuccess.name,
          payload: ['SMTP connection successful'],
        });
      } else {
        appEvents.publish({
          type: AppEvents.alertError.name,
          payload: ['SMTP connection failed: ' + response.error],
        });
      }
    } catch (error) {
      console.error('Failed to test SMTP:', error);
      setSmtpTestResult({
        success: false,
        error: 'Failed to test SMTP: ' + error,
      });
      appEvents.publish({
        type: AppEvents.alertError.name,
        payload: ['Failed to test SMTP connection'],
      });
    } finally {
      setIsTestingSMTP(false);
    }
  };

  return (
    <div className={styles.container}>
      <h2>Settings</h2>

      <Form onSubmit={handleSubmit}>
        {() => (
          <>
            <FieldSet label="Service Account Authentication">
              <div style={{ marginBottom: '16px' }}>
                <p style={{ marginBottom: '8px' }}>
                  This plugin uses Grafana's <strong>managed service accounts</strong> to authenticate when rendering dashboards.
                  The service account is automatically created by Grafana when the plugin starts (requires Grafana 10.3+).
                </p>
                {serviceAccountStatus && (
                  <div style={{
                    padding: '12px',
                    background: serviceAccountStatus.status === 'active' ? '#e7f5e7' :
                              serviceAccountStatus.status === 'error' ? '#ffebee' : '#fff4e5',
                    border: `1px solid ${
                      serviceAccountStatus.status === 'active' ? '#4caf50' :
                      serviceAccountStatus.status === 'error' ? '#f44336' : '#ff9800'
                    }`,
                    borderRadius: '4px',
                    marginBottom: '12px'
                  }}>
                    {serviceAccountStatus.status === 'active' ? (
                      <>
                        <div style={{ fontSize: '16px', marginBottom: '8px' }}>
                          <strong>✓ Service Account: Active</strong>
                        </div>
                        <div><strong>Token Status:</strong> Configured ({serviceAccountStatus.token_length} characters)</div>
                        <div style={{ marginTop: '8px', fontSize: '13px', opacity: 0.8 }}>
                          {serviceAccountStatus.info}
                        </div>
                      </>
                    ) : serviceAccountStatus.status === 'error' ? (
                      <>
                        <div style={{ fontSize: '16px', marginBottom: '8px' }}>
                          <strong>✗ Service Account: Error</strong>
                        </div>
                        <div><strong>Issue:</strong> {serviceAccountStatus.error}</div>
                        <div style={{ marginTop: '8px' }}>
                          <strong>Requirements:</strong> {serviceAccountStatus.requirements}
                        </div>
                        {serviceAccountStatus.solution && (
                          <div style={{ marginTop: '8px', padding: '8px', background: 'rgba(0,0,0,0.05)', borderRadius: '4px' }}>
                            <strong>Solution:</strong> {serviceAccountStatus.solution}
                          </div>
                        )}
                      </>
                    ) : (
                      <>
                        <div style={{ fontSize: '16px', marginBottom: '8px' }}>
                          <strong>⚠ Service Account: Not Configured</strong>
                        </div>
                        <div><strong>Status:</strong> {serviceAccountStatus.error || 'Token not available'}</div>
                        {serviceAccountStatus.requirements && (
                          <div style={{ marginTop: '8px' }}>
                            <strong>Requirements:</strong> {serviceAccountStatus.requirements}
                          </div>
                        )}
                        {serviceAccountStatus.solution && (
                          <div style={{ marginTop: '8px', padding: '8px', background: 'rgba(0,0,0,0.05)', borderRadius: '4px' }}>
                            <strong>Solution:</strong> {serviceAccountStatus.solution}
                          </div>
                        )}
                      </>
                    )}
                  </div>
                )}
                <div style={{ marginTop: '12px', padding: '12px', background: '#f5f5f5', borderRadius: '4px', fontSize: '13px' }}>
                  <strong>ℹ️ How it works:</strong>
                  <ul style={{ marginTop: '8px', marginBottom: '0', paddingLeft: '20px' }}>
                    <li>Grafana automatically creates a service account for this plugin</li>
                    <li>The service account permissions are defined in <code>plugin.json</code> IAM section</li>
                    <li>No manual token creation needed - Grafana manages everything</li>
                    <li>If status shows error, enable the feature toggle in <code>grafana.ini</code></li>
                  </ul>
                </div>
              </div>
            </FieldSet>

            <FieldSet label="SMTP Configuration">
              <Field
                label="Host"
                description="SMTP server hostname (e.g., smtp.gmail.com)"
              >
                <Input
                  value={settings.smtp_config?.host || ''}
                  onChange={(e) => updateSMTP('host', e.currentTarget.value)}
                  placeholder="smtp.gmail.com"
                  required
                />
              </Field>
              <Field label="Port">
                <Input
                  type="number"
                  value={settings.smtp_config?.port ?? 587}
                  onChange={(e) => {
                    const portValue = parseInt(e.currentTarget.value, 10);
                    // Only update if it's a valid number
                    if (!isNaN(portValue)) {
                      updateSMTP('port', portValue);
                    }
                  }}
                  required
                />
              </Field>
              <Field label="Username" description="SMTP authentication username (optional for anonymous)">
                <Input
                  value={settings.smtp_config?.username || ''}
                  onChange={(e) => updateSMTP('username', e.currentTarget.value)}
                />
              </Field>
              <Field label="Password" description="SMTP authentication password">
                <Input
                  type="password"
                  value={settings.smtp_config?.password || ''}
                  onChange={(e) => updateSMTP('password', e.currentTarget.value)}
                />
              </Field>
              <Field label="From Address" description="Email address to send reports from">
                <Input
                  value={settings.smtp_config?.from || ''}
                  onChange={(e) => updateSMTP('from', e.currentTarget.value)}
                  placeholder="noreply@example.com"
                  required
                />
              </Field>
              <Field label="Use TLS" description="Enable TLS/STARTTLS encryption">
                <Switch
                  value={settings.smtp_config?.use_tls}
                  onChange={(e) => updateSMTP('use_tls', e.currentTarget.checked)}
                />
              </Field>
              <Field label="Skip TLS Verification" description="Skip TLS certificate verification (use for self-signed certificates)">
                <Switch
                  value={settings.smtp_config?.skip_tls_verify}
                  onChange={(e) => updateSMTP('skip_tls_verify', e.currentTarget.checked)}
                />
              </Field>

              {/* SMTP Test Button */}
              <div style={{ marginTop: '16px' }}>
                <Button
                  variant="secondary"
                  onClick={handleTestSMTP}
                  disabled={isTestingSMTP || !settings.smtp_config?.host}
                >
                  {isTestingSMTP ? 'Testing...' : 'Test SMTP Connection'}
                </Button>
                {smtpTestResult && (
                  <div style={{
                    marginTop: '8px',
                    padding: '8px 12px',
                    background: smtpTestResult.success ? '#e7f5e7' : '#ffebee',
                    border: `1px solid ${smtpTestResult.success ? '#4caf50' : '#f44336'}`,
                    borderRadius: '4px',
                    fontSize: '13px'
                  }}>
                    {smtpTestResult.success ? (
                      <>
                        <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                          ✓ {smtpTestResult.message}
                        </div>
                        <div style={{ opacity: 0.8 }}>
                          Host: {smtpTestResult.host}:{smtpTestResult.port} (TLS: {smtpTestResult.tls ? 'Yes' : 'No'})
                        </div>
                      </>
                    ) : (
                      <>
                        <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                          ✗ {smtpTestResult.error}
                        </div>
                        {smtpTestResult.host && (
                          <div style={{ opacity: 0.8 }}>
                            Host: {smtpTestResult.host}:{smtpTestResult.port}
                          </div>
                        )}
                      </>
                    )}
                  </div>
                )}
              </div>
            </FieldSet>

            <FieldSet label="Renderer Configuration">
              <Field
                label="Grafana URL"
                description="Full Grafana URL including protocol and subpath (e.g., https://127.0.0.1:3000/dna)"
              >
                <Input
                  value={settings.renderer_config?.grafana_url || ''}
                  onChange={(e) => updateRenderer('grafana_url', e.currentTarget.value)}
                  placeholder="https://127.0.0.1:3000/dna"
                />
              </Field>

              {/* Rendering settings */}
              <Field label="Timeout (ms)" description="Maximum time to wait for dashboard rendering">
                <Input
                  type="number"
                  value={settings.renderer_config?.timeout_ms ?? 30000}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateRenderer('timeout_ms', value);
                    }
                  }}
                />
              </Field>
              <Field label="Render Delay (ms)" description="Wait time after page load to allow queries to finish">
                <Input
                  type="number"
                  value={settings.renderer_config?.delay_ms ?? 2000}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateRenderer('delay_ms', value);
                    }
                  }}
                />
              </Field>
              <Field label="Viewport Width">
                <Input
                  type="number"
                  value={settings.renderer_config?.viewport_width ?? 1920}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateRenderer('viewport_width', value);
                    }
                  }}
                />
              </Field>
              <Field label="Viewport Height">
                <Input
                  type="number"
                  value={settings.renderer_config?.viewport_height ?? 1080}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateRenderer('viewport_height', value);
                    }
                  }}
                />
              </Field>
              <Field label="Device Scale Factor" description="Higher values (2-4) increase image quality">
                <Input
                  type="number"
                  step="0.1"
                  value={settings.renderer_config?.device_scale_factor ?? 2.0}
                  onChange={(e) => {
                    const value = parseFloat(e.currentTarget.value);
                    if (!isNaN(value)) {
                      updateRenderer('device_scale_factor', value);
                    }
                  }}
                />
              </Field>
              <Field label="Skip TLS Verification" description="Disable TLS certificate verification (use for self-signed certificates)">
                <Switch
                  value={settings.renderer_config?.skip_tls_verify || false}
                  onChange={(e) => updateRenderer('skip_tls_verify', e.currentTarget.checked)}
                />
              </Field>

              {/* Chromium settings */}
              <Field label="Chromium Path" description="Path to Chrome/Chromium binary (leave empty for auto-detection)">
                <>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                    <div style={{ flex: 1 }}>
                      <Input
                        value={settings.renderer_config?.chromium_path || ''}
                        onChange={(e) => updateRenderer('chromium_path', e.currentTarget.value)}
                        placeholder="./chrome-linux64/chrome (or leave empty)"
                      />
                    </div>
                    <Button
                      variant="secondary"
                      onClick={handleCheckChromium}
                      disabled={isCheckingChromium}
                    >
                      {isCheckingChromium ? 'Checking...' : 'Check'}
                    </Button>
                  </div>
                  {chromiumCheckResult && (
                    <div style={{
                      marginTop: '8px',
                      padding: '8px 12px',
                      background: chromiumCheckResult.success ? '#e7f5e7' : '#ffebee',
                      border: `1px solid ${chromiumCheckResult.success ? '#4caf50' : '#f44336'}`,
                      borderRadius: '4px',
                      fontSize: '13px'
                    }}>
                      {chromiumCheckResult.success ? (
                        <>
                          <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                            ✓ {chromiumCheckResult.version}
                          </div>
                          <div style={{ opacity: 0.8 }}>
                            Path: {chromiumCheckResult.path}
                          </div>
                        </>
                      ) : (
                        <>
                          <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                            ✗ {chromiumCheckResult.error}
                          </div>
                          {chromiumCheckResult.path && (
                            <div style={{ opacity: 0.8 }}>
                              Path: {chromiumCheckResult.path}
                            </div>
                          )}
                          {chromiumCheckResult.message && (
                            <div style={{ marginTop: '4px', fontStyle: 'italic' }}>
                              {chromiumCheckResult.message}
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  )}
                </>
              </Field>
              <Field
                label="Chrome Flags"
                description="The following flags are always enabled for stability: --no-sandbox, --disable-gpu, --disable-dev-shm-usage, --disable-crash-reporter, --headless=new"
              >
                <Input
                  value="✓ Always enabled (no-sandbox, disable-gpu, disable-dev-shm-usage, disable-crash-reporter)"
                  disabled
                />
              </Field>
            </FieldSet>

            <FieldSet label="Limits">
              <Field label="Max Recipients">
                <Input
                  type="number"
                  value={settings.limits?.max_recipients ?? 50}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateLimits('max_recipients', value);
                    }
                  }}
                />
              </Field>
              <Field label="Max Attachment Size (MB)">
                <Input
                  type="number"
                  value={settings.limits?.max_attachment_size_mb ?? 25}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateLimits('max_attachment_size_mb', value);
                    }
                  }}
                />
              </Field>
              <Field label="Max Concurrent Renders">
                <Input
                  type="number"
                  value={settings.limits?.max_concurrent_renders ?? 5}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateLimits('max_concurrent_renders', value);
                    }
                  }}
                />
              </Field>
              <Field label="Retention Days">
                <Input
                  type="number"
                  value={settings.limits?.retention_days ?? 30}
                  onChange={(e) => {
                    const value = parseInt(e.currentTarget.value, 10);
                    if (!isNaN(value)) {
                      updateLimits('retention_days', value);
                    }
                  }}
                />
              </Field>
              <Field
                label="Allowed Email Domains"
                description="Whitelist of allowed email domains for report recipients. Enter one domain per line. Leave empty to allow all domains. Supports wildcards (e.g., *.example.com)"
              >
                <TextArea
                  rows={5}
                  value={(settings.limits?.allowed_domains || []).join('\n')}
                  onChange={(e) => {
                    const domains = e.currentTarget.value
                      .split('\n')
                      .map(d => d.trim())
                      .filter(d => d !== '');
                    updateLimits('allowed_domains', domains);
                  }}
                  placeholder="example.com&#10;*.example.org&#10;company.net"
                />
              </Field>
            </FieldSet>

            {/* @ts-ignore */}
            <Button type="submit" variant="primary">
              Save Settings
            </Button>
          </>
        )}
      </Form>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
    max-width: 1200px;
  `,
});
