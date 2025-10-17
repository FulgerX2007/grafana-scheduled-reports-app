import { AppPlugin } from '@grafana/data';
import { AppConfig } from './components/AppConfig';
import { App } from './components/App';

export const plugin = new AppPlugin<{}>()
  .setRootPage(App)
  .addConfigPage({
    title: 'Configuration',
    icon: 'cog',
    body: AppConfig,
    id: 'configuration',
  });
