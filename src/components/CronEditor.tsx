import React, { useState } from 'react';
import { Input, Field, Button, HorizontalGroup } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

interface CronEditorProps {
  value: string;
  onChange: (value: string) => void;
}

const presets = [
  { label: 'Every Monday 8 AM', value: '0 8 * * 1' },
  { label: 'Every Day 8 AM', value: '0 8 * * *' },
  { label: 'Every Hour', value: '0 * * * *' },
  { label: 'Every Week Sunday 8 AM', value: '0 8 * * 0' },
];

export const CronEditor: React.FC<CronEditorProps> = ({ value, onChange }) => {
  const styles = useStyles2(getStyles);

  return (
    <div>
      <Field label="Presets">
        <HorizontalGroup>
          {presets.map((preset) => (
            // @ts-ignore
            <Button
              key={preset.value}
              size="sm"
              variant="secondary"
              onClick={() => onChange(preset.value)}
            >
              {preset.label}
            </Button>
          ))}
        </HorizontalGroup>
      </Field>
      <Field label="Custom Expression" description="Format: minute hour day month weekday">
        <Input value={value} onChange={(e) => onChange(e.currentTarget.value)} placeholder="0 8 * * 1" />
      </Field>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({});
