// SPDX-License-Identifier: MIT
// Copyright (c) 2026 OPS Platform Contributors

import { Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import MainLayout from './components/MainLayout'
import { ThemeProvider, useTheme } from './contexts/ThemeContext'

const LoginPage = lazy(() => import('./pages/LoginPage'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const ProjectsPage = lazy(() => import('./pages/cmdb/ProjectsPage'))
const EnvironmentsPage = lazy(() => import('./pages/cmdb/EnvironmentsPage'))
const ServersPage = lazy(() => import('./pages/cmdb/ServersPage'))
const ApplicationsPage = lazy(() => import('./pages/cmdb/ApplicationsPage'))
const AppReleasePage = lazy(() => import('./pages/deploy/AppReleasePage'))
const DeployHistoryPage = lazy(() => import('./pages/deploy/DeployHistoryPage'))
const ArchivePage = lazy(() => import('./pages/deploy/ArchivePage'))
const ArchiveHistoryPage = lazy(() => import('./pages/deploy/ArchiveHistoryPage'))
const MonitorLayout = lazy(() => import('./pages/monitor/MonitorLayout'))
const MonitorBigScreenPage = lazy(() => import('./pages/monitor/MonitorBigScreenPage'))
const MonitorOverviewPage = lazy(() => import('./pages/monitor/MonitorOverviewPage'))
const MonitorDashboardsPage = lazy(() => import('./pages/monitor/MonitorDashboardsPage'))
const AlarmLayout = lazy(() => import('./pages/alarm/AlarmLayout'))
const AlertEventsPage = lazy(() => import('./pages/alarm/AlertEventsPage'))
const AlertRulesPage = lazy(() => import('./pages/alarm/AlertRulesPage'))
const AlertContactsPage = lazy(() => import('./pages/alarm/AlertContactsPage'))
const AlertChannelsPage = lazy(() => import('./pages/alarm/AlertChannelsPage'))
const AlertTemplatesPage = lazy(() => import('./pages/alarm/AlertTemplatesPage'))
const SecurityOverview = lazy(() => import('./pages/security/Overview'))
const FIMPoliciesPage = lazy(() => import('./pages/security/FIMPoliciesPage'))
const FIMExecutionsPage = lazy(() => import('./pages/security/FIMExecutionsPage'))
const FIMEventsPage = lazy(() => import('./pages/security/FIMEventsPage'))
const FIMAlertsPage = lazy(() => import('./pages/security/FIMAlertsPage'))
const FIMKnownHostsPage = lazy(() => import('./pages/security/FIMKnownHostsPage'))
const TaskList = lazy(() => import('./pages/security/TaskList'))
const VulnerabilityList = lazy(() => import('./pages/security/VulnerabilityList'))
const VulnDBPage = lazy(() => import('./pages/security/VulnDBPage'))
const AssetList = lazy(() => import('./pages/security/AssetList'))
const BatchCopyAllPage = lazy(() => import('./pages/consul/BatchCopyAllPage'))
const ConfigManagementPage = lazy(() => import('./pages/consul/ConfigManagementPage'))
const OperationHistoryPage = lazy(() => import('./pages/consul/OperationHistoryPage'))
const ViewsPage = lazy(() => import('./pages/jenkins/ViewsPage'))
const AggregatePackagePage = lazy(() => import('./pages/deploy/AggregatePackagePage'))
const AggregatedHistoryPage = lazy(() => import('./pages/deploy/AggregatedHistoryPage'))
const RolesPage = lazy(() => import('./pages/admin/RolesPage'))
const MenusPage = lazy(() => import('./pages/admin/MenusPage'))
const UsersPage = lazy(() => import('./pages/admin/UsersPage'))
const SettingsPage = lazy(() => import('./pages/admin/SettingsPage'))
const ProfilePage = lazy(() => import('./pages/ProfilePage'))
const UserManualPage = lazy(() => import('./pages/UserManualPage'))
const ForbiddenPage = lazy(() => import('./pages/ForbiddenPage'))
const PlatformEventsPage = lazy(() => import('./pages/platform/PlatformEventsPage'))
const PlatformAuditPage = lazy(() => import('./pages/platform/PlatformAuditPage'))

function withPageLoader(element: JSX.Element) {
  return (
    <Suspense fallback={<div style={{ padding: 24 }}>加载中...</div>}>
      {element}
    </Suspense>
  )
}

function ThemedApp() {
  const { antdTheme } = useTheme()

  return (
    <ConfigProvider
      theme={antdTheme}
      locale={zhCN}
    >
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={withPageLoader(<LoginPage />)} />
        <Route path="/" element={<MainLayout />}>
          <Route index element={withPageLoader(<Dashboard />)} />
          <Route path="cmdb/projects" element={withPageLoader(<ProjectsPage />)} />
          <Route path="cmdb/environments" element={withPageLoader(<EnvironmentsPage />)} />
          <Route path="cmdb/servers" element={withPageLoader(<ServersPage />)} />
          <Route path="cmdb/applications" element={withPageLoader(<ApplicationsPage />)} />
          <Route path="deploy/release" element={withPageLoader(<AppReleasePage />)} />
          <Route path="deploy/history" element={withPageLoader(<DeployHistoryPage />)} />
          <Route path="deploy/archive" element={withPageLoader(<ArchivePage />)} />
          <Route path="deploy/archived" element={withPageLoader(<ArchiveHistoryPage />)} />
          <Route path="deploy/aggregate-package" element={withPageLoader(<AggregatePackagePage />)} />
          <Route path="deploy/aggregated-history" element={withPageLoader(<AggregatedHistoryPage />)} />
          <Route path="monitor" element={withPageLoader(<MonitorLayout />)}>
            <Route index element={withPageLoader(<MonitorBigScreenPage />)} />
            <Route path="bigscreen" element={withPageLoader(<MonitorBigScreenPage />)} />
            <Route path="overview" element={withPageLoader(<MonitorOverviewPage />)} />
            <Route path="dashboards" element={withPageLoader(<MonitorDashboardsPage />)} />
          </Route>
          <Route path="alarm" element={withPageLoader(<AlarmLayout />)}>
            <Route index element={withPageLoader(<AlertEventsPage />)} />
            <Route path="events" element={withPageLoader(<AlertEventsPage />)} />
            <Route path="rules" element={withPageLoader(<AlertRulesPage />)} />
            <Route path="contacts" element={withPageLoader(<AlertContactsPage />)} />
            <Route path="channels" element={withPageLoader(<AlertChannelsPage />)} />
            <Route path="templates" element={withPageLoader(<AlertTemplatesPage />)} />
          </Route>
          <Route path="security">
            <Route index element={withPageLoader(<SecurityOverview />)} />
            <Route path="overview" element={withPageLoader(<SecurityOverview />)} />
            <Route path="fim" element={<Navigate to="/security/fim/policies" replace />} />
            <Route path="fim/policies" element={withPageLoader(<FIMPoliciesPage />)} />
            <Route path="fim/executions" element={withPageLoader(<FIMExecutionsPage />)} />
            <Route path="fim/events" element={withPageLoader(<FIMEventsPage />)} />
            <Route path="fim/alerts" element={withPageLoader(<FIMAlertsPage />)} />
            <Route path="fim/known-hosts" element={withPageLoader(<FIMKnownHostsPage />)} />
            <Route path="tasks" element={withPageLoader(<TaskList />)} />
            <Route path="assets" element={withPageLoader(<AssetList />)} />
            <Route path="vulnerabilities" element={withPageLoader(<VulnerabilityList />)} />
            <Route path="vuln-db" element={withPageLoader(<VulnDBPage />)} />
          </Route>
          <Route path="consul">
            <Route path="batch-all" element={withPageLoader(<BatchCopyAllPage />)} />
            <Route path="config" element={withPageLoader(<ConfigManagementPage />)} />
            <Route path="operations" element={withPageLoader(<OperationHistoryPage />)} />
          </Route>
          <Route path="jenkins">
            <Route index element={withPageLoader(<ViewsPage />)} />
            <Route path="views" element={withPageLoader(<ViewsPage />)} />
          </Route>
          <Route path="admin/roles" element={withPageLoader(<RolesPage />)} />
          <Route path="admin/menus" element={withPageLoader(<MenusPage />)} />
          <Route path="admin/users" element={withPageLoader(<UsersPage />)} />
          <Route path="admin/settings" element={withPageLoader(<SettingsPage />)} />
          <Route path="profile" element={withPageLoader(<ProfilePage />)} />
          <Route path="user-manual" element={withPageLoader(<UserManualPage />)} />
          <Route path="platform/audit" element={withPageLoader(<PlatformAuditPage />)} />
          <Route path="platform/events" element={withPageLoader(<PlatformEventsPage />)} />
          <Route path="forbidden" element={withPageLoader(<ForbiddenPage />)} />
        </Route>
      </Routes>
    </BrowserRouter>
    </ConfigProvider>
  )
}

function App() {
  return (
    <ThemeProvider>
      <ThemedApp />
    </ThemeProvider>
  )
}

export default App
