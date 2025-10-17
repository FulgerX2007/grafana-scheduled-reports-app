export interface VariableOption {
  text: string;
  value: string;
}

export interface Variable {
  name: string;
  value: string;
  options?: VariableOption[]; // Available options for select dropdown
  is_original?: boolean; // True for variables loaded from dashboard, false for user-created duplicates
}

export interface Schedule {
  id: number;
  org_id: number;
  name: string;
  dashboard_uid: string;
  dashboard_title?: string;
  panel_ids?: number[];
  range_from: string;
  range_to: string;
  interval_type: 'cron' | 'daily' | 'weekly' | 'monthly';
  cron_expr?: string;
  timezone: string;
  variables?: Variable[];
  recipients: Recipients;
  email_subject: string;
  email_body: string;
  template_id?: number;
  enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  owner_user_id: number;
  created_at: string;
  updated_at: string;
}

export interface Recipients {
  to: string[];
  cc?: string[];
  bcc?: string[];
}

export interface Run {
  id: number;
  schedule_id: number;
  org_id: number;
  started_at: string;
  finished_at?: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  error_text?: string;
  artifact_path?: string;
  rendered_pages: number;
  bytes: number;
  checksum?: string;
  created_at: string;
}

export interface Template {
  id: number;
  org_id: number;
  name: string;
  kind: 'pdf' | 'html' | 'email';
  config: TemplateConfig;
  created_at: string;
  updated_at: string;
}

export interface TemplateConfig {
  header?: string;
  footer?: string;
  logo_url?: string;
  watermark?: string;
  page_size?: 'A4' | 'Letter';
  orientation?: 'portrait' | 'landscape';
  margins?: {
    top: number;
    bottom: number;
    left: number;
    right: number;
  };
}

export interface Settings {
  id: number;
  org_id: number;
  smtp_config?: SMTPConfig;
  renderer_config: RendererConfig;
  limits: Limits;
  created_at: string;
  updated_at: string;
}

export interface SMTPConfig {
  host: string;
  port: number;
  username: string;
  password: string;
  from: string;
  use_tls: boolean;
  skip_tls_verify: boolean;
}

export interface RendererConfig {
  grafana_url?: string; // Grafana base URL (e.g., https://127.0.0.1:3000/dna)
  url?: string; // DEPRECATED - kept for backward compatibility
  timeout_ms: number;
  delay_ms: number;
  viewport_width: number;
  viewport_height: number;
  device_scale_factor?: number;
  skip_tls_verify?: boolean;

  // Chromium settings
  chromium_path?: string;
  headless?: boolean;
  disable_gpu?: boolean;
  no_sandbox?: boolean;
}

export interface Limits {
  max_recipients: number;
  max_attachment_size_mb: number;
  max_concurrent_renders: number;
  retention_days: number;
  allowed_domains?: string[]; // If empty, all domains are allowed
}

export interface ScheduleFormData {
  name: string;
  dashboard_uid: string;
  dashboard_title?: string;
  panel_ids?: number[];
  range_from: string;
  range_to: string;
  interval_type: 'cron' | 'daily' | 'weekly' | 'monthly';
  cron_expr?: string;
  timezone: string;
  variables?: Variable[];
  recipients: Recipients;
  email_subject: string;
  email_body: string;
  template_id?: number;
  enabled: boolean;
}
