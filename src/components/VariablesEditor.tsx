import React from 'react';
import { Input, Button, Select } from '@grafana/ui';
import { css } from '@emotion/css';
import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';
import { Variable } from '../types/types';

interface VariablesEditorProps {
  value: Variable[];
  onChange: (value: Variable[]) => void;
  readOnlyKeys?: boolean; // When true, variable names are read-only (but duplicates can still be added/removed)
}

export const VariablesEditor: React.FC<VariablesEditorProps> = ({ value, onChange, readOnlyKeys = false }) => {
  const styles = useStyles2(getStyles);
  const variables = value || [];

  const addVariable = () => {
    onChange([...variables, { name: '', value: '' }]);
  };

  const updateKey = (index: number, newName: string) => {
    const updated = [...variables];
    updated[index] = { ...updated[index], name: newName };
    onChange(updated);
  };

  const updateValue = (index: number, newValue: string) => {
    const updated = [...variables];
    updated[index] = { ...updated[index], value: newValue };
    onChange(updated);
  };

  const removeVariable = (index: number) => {
    const updated = variables.filter((_, i) => i !== index);
    onChange(updated);
  };

  const duplicateVariable = (index: number) => {
    const varToDuplicate = variables[index];
    const updated = [...variables];
    // Insert the duplicate right after the original, mark as non-original (user-created duplicate)
    // Preserve options from the original variable
    updated.splice(index + 1, 0, {
      name: varToDuplicate.name,
      value: varToDuplicate.value,
      options: varToDuplicate.options,
      is_original: false
    });
    onChange(updated);
  };

  return (
    <div>
      <div className={readOnlyKeys ? styles.headerReadOnly : styles.header}>
        <span>Variable Name</span>
        <span>Value</span>
        {readOnlyKeys && <span></span>} {/* Duplicate button column */}
        {readOnlyKeys && <span></span>} {/* Delete button column */}
        {!readOnlyKeys && <span></span>} {/* Delete button column when not read-only */}
      </div>
      {variables.map((variable, index) => (
        <div key={index} className={readOnlyKeys ? styles.rowReadOnly : styles.row}>
          <Input
            value={variable.name}
            onChange={(e) => updateKey(index, e.currentTarget.value)}
            placeholder="variable_name"
            disabled={readOnlyKeys}
          />
          {/* Use Select if options are available, otherwise use Input */}
          {variable.options && variable.options.length > 0 ? (
            <Select
              options={variable.options.map(opt => ({ label: opt.text, value: opt.value }))}
              value={variable.value}
              onChange={(selected: SelectableValue<string>) => updateValue(index, selected.value || '')}
              placeholder="Select value"
              isClearable={false}
            />
          ) : (
            <Input
              value={variable.value}
              onChange={(e) => updateValue(index, e.currentTarget.value)}
              placeholder="value"
            />
          )}
          {/* Show duplicate button only for variables without options (non-select types) */}
          {readOnlyKeys && (!variable.options || variable.options.length === 0) && (
            // @ts-ignore
            <Button
              size="sm"
              variant="secondary"
              icon="copy"
              onClick={() => duplicateVariable(index)}
              tooltip="Duplicate variable"
            />
          )}
          {/* Show placeholder for select-type variables to maintain grid alignment */}
          {readOnlyKeys && variable.options && variable.options.length > 0 && (
            <div></div>
          )}
          {/* Show delete button in read-only mode only for duplicates (non-original variables) */}
          {readOnlyKeys && !variable.is_original && (
            // @ts-ignore
            <Button
              size="sm"
              variant="secondary"
              icon="trash-alt"
              onClick={() => removeVariable(index)}
              tooltip="Remove duplicate"
            />
          )}
          {/* Show placeholder for original variables to maintain grid alignment */}
          {readOnlyKeys && variable.is_original && (
            <div></div>
          )}
          {/* Show delete button in edit mode */}
          {!readOnlyKeys && (
            // @ts-ignore
            <Button size="sm" variant="secondary" icon="trash-alt" onClick={() => removeVariable(index)} />
          )}
        </div>
      ))}
      {!readOnlyKeys && (
        // @ts-ignore
        <Button size="sm" variant="secondary" icon="plus" onClick={addVariable}>
          Add Variable
        </Button>
      )}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  header: css`
    display: grid;
    grid-template-columns: 1fr 1fr 40px;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  headerReadOnly: css`
    display: grid;
    grid-template-columns: 1fr 1fr 40px 40px;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  row: css`
    display: grid;
    grid-template-columns: 1fr 1fr 40px;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
  `,
  rowReadOnly: css`
    display: grid;
    grid-template-columns: 1fr 1fr 40px 40px;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
  `,
});
