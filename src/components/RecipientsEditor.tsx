import React from 'react';
import { Field, Input, Button, Icon } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import { Recipients } from '../types/types';

interface RecipientsEditorProps {
  value: Recipients;
  onChange: (value: Recipients) => void;
}

export const RecipientsEditor: React.FC<RecipientsEditorProps> = ({ value, onChange }) => {
  const styles = useStyles2(getStyles);

  const addRecipient = (field: 'to' | 'cc' | 'bcc') => {
    const updated = { ...value };
    if (!updated[field]) {
      updated[field] = [];
    }
    updated[field]!.push('');
    onChange(updated);
  };

  const updateRecipient = (field: 'to' | 'cc' | 'bcc', index: number, email: string) => {
    const updated = { ...value };
    updated[field]![index] = email;
    onChange(updated);
  };

  const removeRecipient = (field: 'to' | 'cc' | 'bcc', index: number) => {
    const updated = { ...value };
    updated[field]!.splice(index, 1);
    onChange(updated);
  };

  const renderField = (field: 'to' | 'cc' | 'bcc', label: string) => (
    <div className={styles.fieldGroup}>
      <label>{label}</label>
      {(value[field] || []).map((email, index) => (
        <div key={index} className={styles.recipientRow}>
          <Input
            value={email}
            onChange={(e) => updateRecipient(field, index, e.currentTarget.value)}
            placeholder="email@example.com"
          />
          {/* @ts-ignore */}
          <Button
            size="sm"
            variant="secondary"
            icon="trash-alt"
            onClick={() => removeRecipient(field, index)}
          />
        </div>
      ))}
      <Button size="sm" variant="secondary" icon="plus" onClick={() => addRecipient(field)}>
        Add {label}
      </Button>
    </div>
  );

  return (
    <div>
      {renderField('to', 'To')}
      {renderField('cc', 'CC')}
      {renderField('bcc', 'BCC')}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  fieldGroup: css`
    margin-bottom: ${theme.spacing(2)};
  `,
  recipientRow: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
  `,
});
