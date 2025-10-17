import React, { useState, useEffect } from 'react';
import { Select } from '@grafana/ui';
import { SelectableValue } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';

interface DashboardPickerProps {
  value: string;
  onChange: (uid: string, title?: string) => void;
}

export const DashboardPicker: React.FC<DashboardPickerProps> = ({ value, onChange }) => {
  const [dashboards, setDashboards] = useState<Array<SelectableValue<string>>>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadDashboards();
  }, []);

  const loadDashboards = async (query = '') => {
    setLoading(true);
    try {
      const response = await getBackendSrv().get('/api/search', { type: 'dash-db', query });
      const options = response.map((d: any) => ({
        label: d.title,
        value: d.uid,
        description: d.folderTitle,
      }));
      setDashboards(options);
    } catch (error) {
      console.error('Failed to load dashboards:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Select
      options={dashboards}
      value={value}
      onChange={(v) => onChange(v.value!, v.label)}
      onOpenMenu={() => loadDashboards()}
      isLoading={loading}
      placeholder="Select dashboard"
      allowCustomValue={false}
    />
  );
};
