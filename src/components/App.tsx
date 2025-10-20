import React, { useState, useEffect } from 'react';
import { AppRootProps } from '@grafana/data';
import { TabsBar, Tab, TabContent } from '@grafana/ui';
import { config } from '@grafana/runtime';
import { SchedulesPage } from '../pages/Schedules/SchedulesPage';
import { ScheduleEditPage } from '../pages/Schedules/ScheduleEditPage';
import { RunHistoryPage } from '../pages/RunHistory/RunHistoryPage';
import { SettingsPage } from '../pages/Settings/SettingsPage';
import { DocumentationPage } from '../pages/Documentation/DocumentationPage';

type Page = 'schedules' | 'schedule-new' | 'schedule-edit' | 'run-history' | 'settings' | 'documentation';

export const App: React.FC<AppRootProps> = (props) => {
  const [currentPage, setCurrentPage] = useState<Page>('schedules');
  const [selectedScheduleId, setSelectedScheduleId] = useState<number | null>(null);

  // Check URL path to determine initial page
  useEffect(() => {
    const path = props.path || '';
    const urlParams = new URLSearchParams(window.location.search);
    const scheduleIdParam = urlParams.get('scheduleId');

    if (scheduleIdParam) {
      setSelectedScheduleId(parseInt(scheduleIdParam, 10));
    }

    if (path.includes('settings')) {
      setCurrentPage('settings');
    } else if (path.includes('documentation')) {
      setCurrentPage('documentation');
    } else if (path.includes('schedule/new')) {
      setCurrentPage('schedule-new');
    } else if (path.includes('schedule/edit')) {
      setCurrentPage('schedule-edit');
    } else if (path.includes('history')) {
      setCurrentPage('run-history');
    } else if (path.includes('documentation')) {
      setCurrentPage('documentation');
    } else {
      setCurrentPage('schedules');
    }
  }, [props.path]);

  const navigate = (page: Page, scheduleId?: number) => {
    // Navigate to the proper URL for each page
    const appSubUrl = config.appSubUrl || '';
    const baseUrl = `${appSubUrl}/a/scheduled-reports-app`;
    let url = baseUrl;

    switch (page) {
      case 'schedules':
        url = baseUrl;
        break;
      case 'schedule-new':
        url = `${baseUrl}/schedule/new`;
        break;
      case 'schedule-edit':
        url = `${baseUrl}/schedule/edit?scheduleId=${scheduleId}`;
        break;
      case 'run-history':
        url = `${baseUrl}/history?scheduleId=${scheduleId}`;
        break;
      case 'settings':
        url = `${baseUrl}/settings`;
        break;
      case 'documentation':
        url = `${baseUrl}/documentation`;
        break;
    }

    window.location.href = url;
  };

  const renderPage = () => {
    switch (currentPage) {
      case 'schedules':
        return <SchedulesPage onNavigate={navigate} />;
      case 'schedule-new':
        return <ScheduleEditPage onNavigate={navigate} isNew={true} />;
      case 'schedule-edit':
        return <ScheduleEditPage onNavigate={navigate} isNew={false} scheduleId={selectedScheduleId} />;
      case 'run-history':
        return <RunHistoryPage onNavigate={navigate} scheduleId={selectedScheduleId} />;
      case 'settings':
        return <SettingsPage onNavigate={navigate} />;
      case 'documentation':
        return <DocumentationPage />;
      default:
        return <SchedulesPage onNavigate={navigate} />;
    }
  };

  return (
    <div>
      <TabsBar>
        <Tab
          label="Schedules"
          active={currentPage === 'schedules'}
          onChangeTab={() => navigate('schedules')}
        />
        <Tab
          label="Run History"
          active={currentPage === 'run-history'}
          onChangeTab={() => navigate('run-history')}
        />
        <Tab
          label="Documentation"
          active={currentPage === 'documentation'}
          onChangeTab={() => navigate('documentation')}
        />
        <Tab
          label="Settings"
          active={currentPage === 'settings'}
          onChangeTab={() => navigate('settings')}
        />
      </TabsBar>
      {renderPage()}
    </div>
  );
};
