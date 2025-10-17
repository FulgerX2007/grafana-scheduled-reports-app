import React from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2 } from '@grafana/ui';

export const TemplatesPage: React.FC = () => {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.container}>
      <h2>Templates</h2>
      <p>Template management coming soon...</p>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
  `,
});
