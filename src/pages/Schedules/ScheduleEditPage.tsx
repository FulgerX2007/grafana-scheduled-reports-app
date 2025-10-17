import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Button, Field, Input, Select, Switch, TextArea, Form, FieldSet } from '@grafana/ui';
import { ScheduleFormData } from '../../types/types';
import { getBackendSrv, getAppEvents } from '@grafana/runtime';
import { AppEvents } from '@grafana/data';
import { DashboardPicker } from '../../components/DashboardPicker';
import { CronEditor } from '../../components/CronEditor';
import { RecipientsEditor } from '../../components/RecipientsEditor';
import { VariablesEditor } from '../../components/VariablesEditor';

interface ScheduleEditPageProps {
  onNavigate: (page: string) => void;
  isNew: boolean;
  scheduleId?: number | null;
}

const intervalOptions = [
  { label: 'Daily', value: 'daily' },
  { label: 'Weekly', value: 'weekly' },
  { label: 'Monthly', value: 'monthly' },
  { label: 'Custom (Cron)', value: 'cron' },
];

export const ScheduleEditPage: React.FC<ScheduleEditPageProps> = ({ onNavigate, isNew, scheduleId }) => {
  const styles = useStyles2(getStyles);

  const [formData, setFormData] = useState<ScheduleFormData>({
    name: '',
    dashboard_uid: '',
    range_from: 'now-7d',
    range_to: 'now',
    interval_type: 'daily',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    recipients: { to: [] },
    email_subject: 'Grafana Report: {{dashboard.title}}',
    email_body: 'Please find attached the dashboard report for {{timerange}}.',
    enabled: true,
  });

  useEffect(() => {
    if (!isNew && scheduleId) {
      loadSchedule();
    }
  }, [scheduleId]);

  const loadSchedule = async () => {
    try {
      const response = await getBackendSrv().get(`/api/plugins/sheduled-reports-app/resources/api/schedules/${scheduleId}`);
      setFormData(response);
    } catch (error) {
      console.error('Failed to load schedule:', error);
    }
  };

  const loadDashboardVariables = async (dashboardUid: string) => {
    try {
      const dashboard = await getBackendSrv().get(`/api/dashboards/uid/${dashboardUid}`);
      const templateVars = dashboard.dashboard?.templating?.list || [];

      const variables: Array<{ name: string; value: string; options?: Array<{text: string; value: string}>; is_original: boolean }> = [];
      templateVars.forEach((v: any) => {
        if (v.name && v.type !== 'constant' && v.type !== 'datasource') {
          // Use current value if exists, otherwise use default or empty string
          let value = v.current?.value || v.default || '';
          // Handle multi-value variables (convert arrays to comma-separated strings)
          if (Array.isArray(value)) {
            value = value.join(',');
          }

          // Extract options if available
          let options: Array<{text: string; value: string}> | undefined;
          if (v.options && Array.isArray(v.options) && v.options.length > 0) {
            options = v.options.map((opt: any) => ({
              text: opt.text || String(opt.value || ''),
              value: String(opt.value || opt.text || '')
            }));
          }

          // Mark as original (loaded from dashboard, not user-created duplicate)
          variables.push({ name: v.name, value, options, is_original: true });
        }
      });

      setFormData((prev) => ({ ...prev, variables }));
    } catch (error) {
      console.error('Failed to load dashboard variables:', error);
    }
  };

  const handleSubmit = async () => {
    const appEvents = getAppEvents();
    try {
      if (isNew) {
        await getBackendSrv().post('/api/plugins/sheduled-reports-app/resources/api/schedules', formData);
      } else {
        await getBackendSrv().put(`/api/plugins/sheduled-reports-app/resources/api/schedules/${scheduleId}`, formData);
      }
      onNavigate('schedules');
    } catch (error) {
      console.error('Failed to save schedule:', error);
      appEvents.publish({
        type: AppEvents.alertError.name,
        payload: ['Failed to save schedule'],
      });
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>{isNew ? 'New Schedule' : 'Edit Schedule'}</h2>
        {/* @ts-ignore */}
        <Button
          variant="secondary"
          icon="arrow-left"
          onClick={() => onNavigate('schedules')}
        >
          Back to Schedules
        </Button>
      </div>

      <Form onSubmit={handleSubmit}>
        {() => (
          <>
            <FieldSet label="Basic Information">
              <Field label="Name" required>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.currentTarget.value })}
                  placeholder="Weekly Sales Report"
                />
              </Field>

              <Field label="Dashboard" required>
                <DashboardPicker
                  value={formData.dashboard_uid}
                  onChange={(uid, title) => {
                    setFormData({ ...formData, dashboard_uid: uid, dashboard_title: title });
                    loadDashboardVariables(uid);
                  }}
                />
              </Field>

              <Field label="Enabled">
                <Switch
                  value={formData.enabled}
                  onChange={(e) => setFormData({ ...formData, enabled: e.currentTarget.checked })}
                />
              </Field>
            </FieldSet>

            <FieldSet label="Time Range">
              <Field label="From">
                <Input
                  value={formData.range_from}
                  onChange={(e) => setFormData({ ...formData, range_from: e.currentTarget.value })}
                  placeholder="now-7d"
                />
              </Field>

              <Field label="To">
                <Input
                  value={formData.range_to}
                  onChange={(e) => setFormData({ ...formData, range_to: e.currentTarget.value })}
                  placeholder="now"
                />
              </Field>
            </FieldSet>

            <FieldSet label="Schedule">
              <Field
                label="Interval"
                description={
                  formData.interval_type === 'daily' ? 'Runs every day at 00:00' :
                  formData.interval_type === 'weekly' ? 'Runs every Monday at 00:00' :
                  formData.interval_type === 'monthly' ? 'Runs on the 1st of each month at 00:00' :
                  formData.interval_type === 'cron' ? 'Custom cron schedule' : ''
                }
              >
                <Select
                  options={intervalOptions}
                  value={formData.interval_type}
                  onChange={(v) => {
                    const intervalType = v.value as any;
                    let cronExpr = formData.cron_expr;

                    // Set appropriate cron expressions for presets
                    if (intervalType === 'daily') {
                      cronExpr = '0 0 * * *'; // Every day at 00:00
                    } else if (intervalType === 'weekly') {
                      cronExpr = '0 0 * * 1'; // Every Monday at 00:00
                    } else if (intervalType === 'monthly') {
                      cronExpr = '0 0 1 * *'; // First day of month at 00:00
                    }

                    setFormData({ ...formData, interval_type: intervalType, cron_expr: cronExpr });
                  }}
                />
              </Field>

              {formData.interval_type === 'cron' && (
                <Field label="Cron Expression">
                  <CronEditor
                    value={formData.cron_expr || ''}
                    onChange={(expr) => setFormData({ ...formData, cron_expr: expr })}
                  />
                </Field>
              )}

              <Field label="Timezone">
                <Input
                  value={formData.timezone}
                  onChange={(e) => setFormData({ ...formData, timezone: e.currentTarget.value })}
                />
              </Field>
            </FieldSet>

            <FieldSet label="Dashboard Variables">
              <VariablesEditor
                value={formData.variables || []}
                onChange={(vars) => setFormData({ ...formData, variables: vars })}
                readOnlyKeys={true}
              />
            </FieldSet>

            <FieldSet label="Email">
              <Field label="Recipients" required>
                <RecipientsEditor
                  value={formData.recipients}
                  onChange={(recipients) => setFormData({ ...formData, recipients })}
                />
              </Field>

              <Field label="Subject" description="Email subject line. You can use template variables like {{dashboard.title}}">
                <Input
                  value={formData.email_subject}
                  onChange={(e) => setFormData({ ...formData, email_subject: e.currentTarget.value })}
                />
              </Field>

              <Field label="Body">
                <>
                  <TextArea
                    value={formData.email_body}
                    onChange={(e) => setFormData({ ...formData, email_body: e.currentTarget.value })}
                    rows={5}
                  />
                  <div style={{
                    marginTop: '8px',
                    padding: '12px',
                    background: '#f5f5f5',
                    borderRadius: '4px',
                    fontSize: '13px'
                  }}>
                    <strong>Available template variables:</strong>
                    <ul style={{ marginTop: '8px', marginBottom: '0', paddingLeft: '20px' }}>
                      <li><code>{'{{schedule.name}}'}</code> - Schedule name</li>
                      <li><code>{'{{dashboard.title}}'}</code> - Dashboard title</li>
                      <li><code>{'{{timerange}}'}</code> - Time range (e.g., "now-7d to now")</li>
                      <li><code>{'{{run.started_at}}'}</code> - Report generation timestamp</li>
                    </ul>
                  </div>
                </>
              </Field>
            </FieldSet>

            <div className={styles.actions}>
              {/* @ts-ignore */}
              <Button type="submit" variant="primary">
                {isNew ? 'Create' : 'Save'}
              </Button>
              {/* @ts-ignore */}
              <Button variant="secondary" onClick={() => onNavigate('schedules')}>
                Cancel
              </Button>
            </div>
          </>
        )}
      </Form>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: ${theme.spacing(3)};
  `,
  actions: css`
    display: flex;
    gap: ${theme.spacing(2)};
    margin-top: ${theme.spacing(3)};
  `,
});
