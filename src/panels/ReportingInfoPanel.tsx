import React, { useEffect, useState } from 'react';
import { PanelProps } from '@grafana/data';
import { useStyles2, Icon } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { getBackendSrv, locationService } from '@grafana/runtime';

interface Schedule {
  id: number;
  name: string;
  interval_type: string;
  cron_expr?: string;
  enabled: boolean;
  next_run_at?: string;
}

export const ReportingInfoPanel: React.FC<PanelProps> = ({ width, height }) => {
  const styles = useStyles2(getStyles);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [dashboardUID, setDashboardUID] = useState<string>('');

  useEffect(() => {
    // Get dashboard UID from URL
    const location = locationService.getLocation();
    const pathParts = location.pathname.split('/');
    const dIndex = pathParts.indexOf('d');
    if (dIndex >= 0 && pathParts.length > dIndex + 1) {
      const uid = pathParts[dIndex + 1];
      setDashboardUID(uid);
      loadSchedules(uid);
    } else {
      setLoading(false);
    }
  }, []);

  const loadSchedules = async (uid: string) => {
    try {
      const response = await getBackendSrv().get(
        `/api/plugins/sheduled-reports-app/resources/api/dashboards/${uid}/schedules`
      );
      const scheduleList = response.schedules || [];
      setSchedules(scheduleList);
    } catch (error) {
      console.error('Failed to load schedules:', error);
      setSchedules([]);
    } finally {
      setLoading(false);
    }
  };

  const handleManageReports = () => {
    locationService.push('/a/sheduled-reports-app');
  };

  if (loading) {
    return (
      <div className={styles.container} style={{ width, height }}>
        <div className={styles.loading}>Loading...</div>
      </div>
    );
  }

  if (!dashboardUID) {
    return (
      <div className={styles.container} style={{ width, height }}>
        <div className={styles.error}>Unable to detect dashboard</div>
      </div>
    );
  }

  const activeSchedules = schedules.filter((s) => s.enabled);

  return (
    <div className={styles.container} style={{ width, height }}>
      {schedules.length === 0 ? (
        <div className={styles.noSchedules}>
          <Icon name="calendar-alt" size="xl" />
          <p>No scheduled reports for this dashboard</p>
          <button className={styles.link} onClick={handleManageReports}>
            Create a schedule
          </button>
        </div>
      ) : (
        <div className={styles.schedules}>
          <div className={styles.header}>
            <Icon name="calendar-alt" size="lg" />
            <span className={styles.title}>
              {activeSchedules.length} Active Report{activeSchedules.length !== 1 ? 's' : ''}
            </span>
          </div>
          <ul className={styles.list}>
            {schedules.map((schedule) => (
              <li key={schedule.id} className={styles.item}>
                <div className={styles.scheduleName}>
                  <Icon
                    name={schedule.enabled ? 'check-circle' : 'pause-circle'}
                    className={schedule.enabled ? styles.iconEnabled : styles.iconDisabled}
                  />
                  {schedule.name}
                </div>
                <div className={styles.scheduleInfo}>
                  {schedule.interval_type === 'cron'
                    ? schedule.cron_expr
                    : schedule.interval_type}
                  {schedule.next_run_at && schedule.enabled && (
                    <span className={styles.nextRun}>
                      Next: {new Date(schedule.next_run_at).toLocaleString()}
                    </span>
                  )}
                </div>
              </li>
            ))}
          </ul>
          <button className={styles.link} onClick={handleManageReports}>
            Manage reports →
          </button>
        </div>
      )}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
    overflow: auto;
  `,
  loading: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: ${theme.colors.text.secondary};
  `,
  error: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: ${theme.colors.error.text};
  `,
  noSchedules: css`
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: ${theme.colors.text.secondary};
    gap: ${theme.spacing(2)};

    p {
      margin: 0;
    }
  `,
  schedules: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
  `,
  header: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    padding-bottom: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  title: css`
    font-size: ${theme.typography.h5.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  list: css`
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  item: css`
    padding: ${theme.spacing(1)};
    background: ${theme.colors.background.primary};
    border-radius: ${theme.shape.radius.default};
    border: 1px solid ${theme.colors.border.weak};
  `,
  scheduleName: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    font-weight: ${theme.typography.fontWeightMedium};
    margin-bottom: ${theme.spacing(0.5)};
  `,
  scheduleInfo: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    margin-left: ${theme.spacing(3)};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(0.5)};
  `,
  nextRun: css`
    color: ${theme.colors.text.disabled};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  iconEnabled: css`
    color: ${theme.colors.success.text};
  `,
  iconDisabled: css`
    color: ${theme.colors.text.secondary};
  `,
  link: css`
    background: none;
    border: none;
    color: ${theme.colors.primary.text};
    cursor: pointer;
    padding: ${theme.spacing(1)} 0;
    font-size: ${theme.typography.body.fontSize};
    text-align: left;

    &:hover {
      text-decoration: underline;
    }
  `,
});
