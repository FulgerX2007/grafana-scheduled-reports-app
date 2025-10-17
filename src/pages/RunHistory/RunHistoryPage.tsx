import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, Button, LoadingPlaceholder } from '@grafana/ui';
import { Run } from '../../types/types';
import { getBackendSrv, config } from '@grafana/runtime';

interface RunWithSchedule extends Run {
  schedule_name?: string;
}

interface RunHistoryPageProps {
  onNavigate: (page: string) => void;
  scheduleId: number | null | undefined;
}

export const RunHistoryPage: React.FC<RunHistoryPageProps> = ({ onNavigate, scheduleId }) => {
  const styles = useStyles2(getStyles);
  const [runs, setRuns] = useState<RunWithSchedule[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadRuns();
  }, [scheduleId]);

  useEffect(() => {
    const interval = setInterval(() => {
      loadRuns();
    }, 5000);

    return () => clearInterval(interval);
  }, [scheduleId]);

  const loadRuns = async () => {
    try {
      let response;
      if (scheduleId) {
        // Load runs for specific schedule
        response = await getBackendSrv().get(
          `/api/plugins/sheduled-reports-app/resources/api/schedules/${scheduleId}/runs`
        );
        setRuns(response.runs || []);
      } else {
        // Load all runs from all schedules
        const schedulesResponse = await getBackendSrv().get('/api/plugins/sheduled-reports-app/resources/api/schedules');
        const schedules = schedulesResponse.schedules || schedulesResponse || [];

        const allRuns: RunWithSchedule[] = [];
        for (const schedule of schedules) {
          try {
            const runsResponse = await getBackendSrv().get(
              `/api/plugins/sheduled-reports-app/resources/api/schedules/${schedule.id}/runs`
            );
            const scheduleRuns = runsResponse.runs || [];
            // Add schedule name to each run
            const runsWithSchedule = scheduleRuns.map((run: Run) => ({
              ...run,
              schedule_name: schedule.name,
            }));
            allRuns.push(...runsWithSchedule);
          } catch (err) {
            console.error(`Failed to load runs for schedule ${schedule.id}:`, err);
          }
        }
        // Sort by started_at descending (most recent first)
        allRuns.sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime());
        setRuns(allRuns);
      }
    } catch (error) {
      console.error('Failed to load runs:', error);
    } finally {
      setLoading(false);
    }
  };

  const downloadArtifact = async (runId: number) => {
    const appSubUrl = config.appSubUrl || '';
    window.open(`${appSubUrl}/api/plugins/sheduled-reports-app/resources/api/runs/${runId}/artifact`, '_blank');
  };

  if (loading) {
    return <LoadingPlaceholder text="Loading run history..." />;
  }

  const safeRuns = Array.isArray(runs) ? runs : [];

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>{scheduleId ? 'Run History' : 'All Run History'}</h2>
        {scheduleId && (
          // @ts-ignore
          <Button
            variant="secondary"
            icon="arrow-left"
            onClick={() => onNavigate('schedules')}
          >
            Back to Schedules
          </Button>
        )}
      </div>
      {safeRuns.length === 0 ? (
        <p>No runs yet</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              {!scheduleId && (
                <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Schedule</th>
              )}
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Started</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Status</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Duration</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Pages</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Size</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Error</th>
              <th style={{ textAlign: 'left', padding: '8px', borderBottom: '2px solid #ddd' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {safeRuns.map((run) => {
              const status = run.status;
              const statusClass =
                status === 'completed'
                  ? styles.statusSuccess
                  : status === 'failed'
                  ? styles.statusError
                  : styles.statusPending;

              const duration = run.finished_at
                ? `${((new Date(run.finished_at).getTime() - new Date(run.started_at).getTime()) / 1000).toFixed(1)}s`
                : 'Running...';

              const mb = run.bytes / 1024 / 1024;
              const size = mb > 1 ? `${mb.toFixed(2)} MB` : `${(run.bytes / 1024).toFixed(2)} KB`;

              return (
                <tr key={run.id}>
                  {!scheduleId && (
                    <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                      {run.schedule_name || `Schedule #${run.schedule_id}`}
                    </td>
                  )}
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                    {new Date(run.started_at).toLocaleString()}
                  </td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                    <span className={statusClass}>{status}</span>
                  </td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>{duration}</td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>{run.rendered_pages}</td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>{size}</td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>{run.error_text || '-'}</td>
                  <td style={{ padding: '8px', borderBottom: '1px solid #eee' }}>
                    {run.status === 'completed' && run.artifact_path ? (
                      // @ts-ignore
                      <Button size="sm" variant="secondary" icon="download-alt" onClick={() => downloadArtifact(run.id)}>
                        Download
                      </Button>
                    ) : null}
                  </td>
                </tr>
              );
            })}
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
    margin-bottom: ${theme.spacing(2)};
  `,
  statusSuccess: css`
    color: ${theme.colors.success.text};
  `,
  statusError: css`
    color: ${theme.colors.error.text};
  `,
  statusPending: css`
    color: ${theme.colors.warning.text};
  `,
});
