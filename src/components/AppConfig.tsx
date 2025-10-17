import React, { useState } from 'react';
import { AppPluginMeta, PluginConfigPageProps } from '@grafana/data';
import { Button, Alert } from '@grafana/ui';
import { getBackendSrv } from '@grafana/runtime';

interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta> {}

export const AppConfig: React.FC<AppConfigProps> = ({ plugin }) => {
  const { enabled } = plugin.meta;
  const [errorMessage, setErrorMessage] = useState('');

  const handleEnablePlugin = async () => {
    try {
      await getBackendSrv().post(`/api/plugins/${plugin.meta.id}/settings`, {
        enabled: true,
        pinned: true,
      });
      window.location.reload();
    } catch (error) {
      console.error('Failed to enable plugin:', error);
      setErrorMessage('Failed to enable plugin');
    }
  };

  return (
    <div>
      <h2>Scheduled Reports</h2>
      <p>
        Schedule and email dashboard reports as PDF or HTML files. This plugin uses Grafana's managed service accounts for authentication (requires Grafana 10.3+).
      </p>

      <Alert title="Plugin Status" severity={enabled ? 'success' : 'info'}>
        {enabled ? (
          <>
            Plugin is enabled and ready to use. Go to the <strong>Schedules</strong> page to create report schedules,
            or visit <strong>Settings</strong> to configure SMTP and renderer options.
          </>
        ) : (
          'Enable the plugin to start creating report schedules'
        )}
      </Alert>

      {errorMessage && (
        <Alert title="Error" severity="error">
          {errorMessage}
        </Alert>
      )}

      {!enabled && (
        <div style={{ marginTop: '16px' }}>
          {/* @ts-ignore */}
          <Button variant="primary" onClick={handleEnablePlugin}>
            Enable Plugin
          </Button>
        </div>
      )}

      {enabled && (
        <Alert title="Authentication" severity="info" style={{ marginTop: '16px' }}>
          This plugin uses Grafana's managed service accounts. Authentication is handled automatically - no manual token configuration required.
        </Alert>
      )}
    </div>
  );
};
