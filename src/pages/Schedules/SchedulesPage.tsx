import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Button, Icon, LoadingPlaceholder } from '@grafana/ui';
import { Schedule } from '../../types/types';
import { getBackendSrv, getAppEvents, config } from '@grafana/runtime';
import { AppEvents } from '@grafana/data';

interface SchedulesPageProps {
  onNavigate: (page: string, id?: number) => void;
}

export const SchedulesPage: React.FC<SchedulesPageProps> = ({ onNavigate }) => {
  const styles = useStyles2(getStyles);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [dashboardFolders, setDashboardFolders] = useState<Record<string, string>>({});
  const [latestRuns, setLatestRuns] = useState<Record<number, any>>({});

  useEffect(() => {
    loadSchedules();
  }, []);

  const loadSchedules = async () => {
    try {
      const response = await getBackendSrv().get('/api/plugins/scheduled-reports-app/resources/api/schedules');
      console.log('API Response:', response);

      let schedulesData: Schedule[] = [];
      // Handle different response formats
      if (Array.isArray(response)) {
        schedulesData = response;
      } else if (response && response.schedules && Array.isArray(response.schedules)) {
        schedulesData = response.schedules;
      } else if (response && response.schedules === null) {
        // Backend returned null instead of empty array
        schedulesData = [];
      } else {
        console.warn('Unexpected response format:', response);
        schedulesData = [];
      }

      setSchedules(schedulesData);

      // Fetch folder information and latest runs for all schedules
      const folders: Record<string, string> = {};
      const runs: Record<number, any> = {};

      for (const schedule of schedulesData) {
        // Fetch folder info
        if (schedule.dashboard_uid) {
          try {
            const dashboards = await getBackendSrv().get('/api/search', {
              type: 'dash-db',
              dashboardUIDs: schedule.dashboard_uid,
            });
            if (dashboards && dashboards.length > 0 && dashboards[0].folderTitle) {
              folders[schedule.dashboard_uid] = dashboards[0].folderTitle;
            }
          } catch (error) {
            console.error(`Failed to load folder for dashboard ${schedule.dashboard_uid}:`, error);
          }
        }

        // Fetch latest run
        try {
          const runsResponse = await getBackendSrv().get(
            `/api/plugins/scheduled-reports-app/resources/api/schedules/${schedule.id}/runs`
          );
          const scheduleRuns = runsResponse.runs || [];
          if (scheduleRuns.length > 0) {
            // Get the most recent completed run with an artifact
            const completedRun = scheduleRuns.find(
              (run: any) => run.status === 'completed' && run.artifact_path
            );
            if (completedRun) {
              runs[schedule.id] = completedRun;
            }
          }
        } catch (error) {
          console.error(`Failed to load runs for schedule ${schedule.id}:`, error);
        }
      }

      setDashboardFolders(folders);
      setLatestRuns(runs);
    } catch (error) {
      console.error('Failed to load schedules:', error);
      setSchedules([]);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this schedule?')) {
      return;
    }
    try {
      await getBackendSrv().delete(`/api/plugins/scheduled-reports-app/resources/api/schedules/${id}`);
      loadSchedules();
    } catch (error) {
      console.error('Failed to delete schedule:', error);
    }
  };

  const handleRunNow = async (id: number) => {
    const appEvents = getAppEvents();
    try {
      await getBackendSrv().post(`/api/plugins/scheduled-reports-app/resources/api/schedules/${id}/run`);
      appEvents.publish({
        type: AppEvents.alertSuccess.name,
        payload: ['Report generation started'],
      });
    } catch (error) {
      console.error('Failed to run schedule:', error);
      appEvents.publish({
        type: AppEvents.alertError.name,
        payload: ['Failed to run schedule'],
      });
    }
  };

  const handleToggle = async (schedule: Schedule) => {
    try {
      await getBackendSrv().put(`/api/plugins/scheduled-reports-app/resources/api/schedules/${schedule.id}`, {
        ...schedule,
        enabled: !schedule.enabled,
      });
      loadSchedules();
    } catch (error) {
      console.error('Failed to toggle schedule:', error);
    }
  };

  const downloadArtifact = (runId: number) => {
    const appSubUrl = config.appSubUrl || '';
    window.open(`${appSubUrl}/api/plugins/scheduled-reports-app/resources/api/runs/${runId}/artifact`, '_blank');
  };

  if (loading) {
    return <LoadingPlaceholder text="Loading schedules..." />;
  }

  // Ensure schedules is always an array
  const safeSchedules = Array.isArray(schedules) ? schedules : [];

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Report Schedules</h2>
          {/* @ts-ignore */}
        <Button
          icon="plus"
          onClick={() => onNavigate('schedule-new')}
        >
          New Schedule
        </Button>
      </div>

      {safeSchedules.length === 0 ? (
        <div className={styles.empty}>
          <Icon name="calendar-alt" size="xxxl" />
          <h3>No schedules yet</h3>
          <p>Create your first report schedule to get started</p>
          {/* @ts-ignore */}
          <Button
            icon="plus"
            onClick={() => onNavigate('schedule-new')}
          >
            Create Schedule
          </Button>
        </div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Name</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Dashboard</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Schedule</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Status</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Next Run</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Latest Report</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {safeSchedules.map((schedule) => (
              <tr key={schedule.id}>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>{schedule.name}</td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  <div>
                    <a
                      href={`${config.appSubUrl || ''}/d/${schedule.dashboard_uid}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={styles.dashboardLink}
                    >
                      {schedule.dashboard_title || schedule.dashboard_uid}
                    </a>
                    {dashboardFolders[schedule.dashboard_uid] && (
                      <div className={styles.folderName}>
                        <Icon name="folder" size="sm" />
                        {dashboardFolders[schedule.dashboard_uid]}
                      </div>
                    )}
                  </div>
                </td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  {schedule.interval_type === 'cron' ? schedule.cron_expr : schedule.interval_type}
                </td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  <span className={schedule.enabled ? styles.statusEnabled : styles.statusDisabled}>
                    {schedule.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  {schedule.next_run_at ? new Date(schedule.next_run_at).toLocaleString() : 'N/A'}
                </td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  {latestRuns[schedule.id] ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                      <span style={{ fontSize: '12px', color: '#888' }}>
                        {new Date(latestRuns[schedule.id].started_at).toLocaleString()}
                      </span>
                      {/* @ts-ignore */}
                      <Button
                        size="sm"
                        variant="secondary"
                        icon="download-alt"
                        onClick={() => downloadArtifact(latestRuns[schedule.id].id)}
                      >
                        Download
                      </Button>
                    </div>
                  ) : (
                    <span style={{ color: '#888' }}>No reports yet</span>
                  )}
                </td>
                <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                  <div className={styles.actions}>
                    {/* @ts-ignore */}
                    <Button
                      size="sm"
                      variant="secondary"
                      icon="play"
                      onClick={() => handleRunNow(schedule.id)}
                      title="Run now"
                    />
                    {/* @ts-ignore */}
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={schedule.enabled ? 'toggle-on' : 'toggle-off'}
                      onClick={() => handleToggle(schedule)}
                      title={schedule.enabled ? 'Disable' : 'Enable'}
                      className={schedule.enabled ? styles.toggleEnabled : styles.toggleDisabled}
                    />
                    {/* @ts-ignore */}
                    <Button
                      size="sm"
                      variant="secondary"
                      icon="edit"
                      onClick={() => onNavigate('schedule-edit', schedule.id)}
                      title="Edit"
                    />
                    {/* @ts-ignore */}
                    <Button
                      size="sm"
                      variant="secondary"
                      icon="history"
                      onClick={() => onNavigate('run-history', schedule.id)}
                      title="View history"
                    />
                    {/* @ts-ignore */}
                    <Button
                      size="sm"
                      variant="destructive"
                      icon="trash-alt"
                      onClick={() => handleDelete(schedule.id)}
                      title="Delete"
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
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
    gap: ${theme.spacing(1)};
  `,
  statusEnabled: css`
    color: ${theme.colors.success.text};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  statusDisabled: css`
    color: ${theme.colors.text.secondary};
  `,
  toggleEnabled: css`
    color: ${theme.colors.success.text};

    &:hover {
      color: ${theme.colors.success.text};
    }
  `,
  toggleDisabled: css`
    color: ${theme.colors.text.secondary};

    &:hover {
      color: ${theme.colors.text.secondary};
    }
  `,
  dashboardLink: css`
    color: ${theme.colors.primary.text};
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  `,
  folderName: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(0.5)};
    margin-top: ${theme.spacing(0.5)};
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
  `,
  empty: css`
    text-align: center;
    padding: ${theme.spacing(8)};
    color: ${theme.colors.text.secondary};

    h3 {
      margin: ${theme.spacing(2)} 0;
    }

    p {
      margin-bottom: ${theme.spacing(3)};
    }
  `,
});
