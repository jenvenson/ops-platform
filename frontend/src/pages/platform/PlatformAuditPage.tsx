// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Col,
  Divider,
  DatePicker,
  Descriptions,
  Drawer,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { Dayjs } from 'dayjs'
import {
  DeleteOutlined,
  DownloadOutlined,
  InboxOutlined,
  FileSearchOutlined,
  LoginOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import i18next from '../../i18n'
import type { TFunction } from 'i18next'
import {
  auditAPI,
  type PlatformAccessLog,
  type PlatformAuditLog,
  type PlatformArchivedLog,
  type PlatformArchiveStats,
  type PlatformLoginLog,
} from '../../api/audit'
import { getErrorMessage } from '../../utils/httpError'
import { canEdit } from '../../utils/menuAccess'
import { formatDateTime } from '../../utils/dateFormat'

const moduleLabelKeyMap: Record<string, string> = {
  // Old Chinese values (existing DB data)
  '安全中心': 'moduleSecurityCenter',
  '系统管理': 'moduleSystemManagement',
  '告警中心': 'moduleAlertCenter',
  '平台审计': 'modulePlatformAudit',
  'Jenkins管理': 'moduleJenkinsManagement',
  '视图管理': 'moduleViewManagement',
  '认证中心': 'moduleAuthCenter',
  '用户中心': 'moduleUserCenter',
  '工作台': 'moduleDashboard',
  '资产中心': 'moduleCmdb',
  '变更发布': 'moduleDeploy',
  '监控中心': 'moduleMonitor',
  '平台事件': 'moduleEvents',
  '平台通用': 'moduleGeneral',
  // New English codes (new data from updated backend)
  'security': 'moduleSecurityCenter',
  'system': 'moduleSystemManagement',
  'alert': 'moduleAlertCenter',
  'audit': 'modulePlatformAudit',
  'auth': 'moduleAuthCenter',
  'user': 'moduleUserCenter',
  'dashboard': 'moduleDashboard',
  'cmdb': 'moduleCmdb',
  'deploy': 'moduleDeploy',
  'monitor': 'moduleMonitor',
  'events': 'moduleEvents',
  'general': 'moduleGeneral',
}

const actionLabelKeyMap: Record<string, string> = {
  // Old Chinese values (existing DB data)
  '执行操作': 'actionLabelExecute',
  '更新配置': 'actionLabelUpdateConfig',
  '删除': 'actionLabelDelete',
  '新增': 'actionLabelCreate',
  '登录成功': 'actionLabelLoginSuccess',
  '登录失败': 'actionLabelLoginFailed',
  '访问页面': 'actionLabelVisitPage',
  '归档': 'actionLabelArchive',
  '清理': 'actionLabelCleanup',
  '登录': 'actionLabelLogin',
  '导出日志': 'actionLabelExportLogs',
  '归档日志': 'actionLabelArchiveLogs',
  '清理日志': 'actionLabelCleanupLogs',
  '确认': 'actionLabelAcknowledge',
  '解决': 'actionLabelResolve',
  '关闭': 'actionLabelClose',
  '启用': 'actionLabelEnable',
  '停用': 'actionLabelDisable',
  // New English codes (new data from updated backend)
  'login': 'actionLabelLogin',
  'export_logs': 'actionLabelExportLogs',
  'archive_logs': 'actionLabelArchiveLogs',
  'cleanup_logs': 'actionLabelCleanupLogs',
  'acknowledge': 'actionLabelAcknowledge',
  'resolve': 'actionLabelResolve',
  'close': 'actionLabelClose',
  'enable': 'actionLabelEnable',
  'disable': 'actionLabelDisable',
  'execute': 'actionLabelExecute',
  'create': 'actionLabelCreate',
  'update': 'actionLabelUpdateConfig',
  'delete': 'actionLabelDelete',
  'visit_page': 'actionLabelVisitPage',
}

const roleNameToCodeMap: Record<string, string> = {
  '超级管理员': 'admin',
  '运维人员': 'ops',
  '开发人员': 'dev',
  '测试人员': 'qa',
  '普通用户': 'user',
}

const roleCodeToKeyMap: Record<string, string> = {
  admin: 'roleDisplayNameAdmin',
  ops: 'roleDisplayNameOps',
  dev: 'roleDisplayNameDev',
  qa: 'roleDisplayNameQa',
  user: 'roleDisplayNameUser',
}

function getTranslatedModule(t: TFunction<'platform'>, module?: string): string {
  if (!module) return '-'
  const key = moduleLabelKeyMap[module]
  return key ? t(key, module) : module
}

function getTranslatedActionLabel(t: TFunction<'platform'>, actionLabel?: string): string {
  if (!actionLabel) return '-'
  const key = actionLabelKeyMap[actionLabel]
  return key ? t(key, actionLabel) : actionLabel
}

function getTranslatedRole(roleName?: string): string {
  if (!roleName) return '-'
  const code = roleNameToCodeMap[roleName]
  if (code) {
    const key = roleCodeToKeyMap[code]
    if (key) return i18next.t('admin:' + key, { defaultValue: roleName })
  }
  return roleName
}

// Maps old Chinese and new English menu titles to menu.json keys
const menuTitleToMenuKey: Record<string, string> = {
  // Old Chinese values (existing DB data)
  '工作台': 'dashboard',
  '项目管理': 'cmdb-projects',
  '环境管理': 'cmdb-environments',
  '主机管理': 'cmdb-servers',
  '应用流水线管理': 'cmdb-applications',
  '迭代部署': 'deploy-release',
  '部署记录': 'deploy-history',
  '归档历史': 'deploy-archived',
  '归档打包': 'deploy-archive',
  '聚合打包': 'deploy-aggregate-package',
  '聚合历史': 'aggregated-history',
  '配置管理': 'consul-config',
  '批量配置下发': 'consul-batch-all',
  '配置操作记录': 'consul-operations',
  '视图管理': 'jenkins-views',
  '监控大屏': 'monitor-bigscreen',
  '监控概览': 'monitor-overview',
  'Grafana仪表盘': 'monitor-dashboards',
  '告警事件': 'alarm-events',
  '告警规则': 'alarm-rules',
  '联系人管理': 'alarm-contacts',
  '通知渠道': 'alarm-channels',
  '通知模板': 'alarm-templates',
  '安全概览': 'security-overview',
  '文件完整性巡检': 'security-fim',
  '巡检策略': 'security-fim-policies',
  '执行记录': 'security-fim-executions',
  '文件变更事件': 'security-fim-events',
  '完整性告警': 'security-fim-alerts',
  '扫描任务': 'security-tasks',
  '安全资产': 'security-assets',
  '漏洞管理': 'security-vulnerabilities',
  '漏洞工单': 'security-tickets',
  '漏洞知识库': 'security-vuln-db',
  '用户管理': 'admin-users',
  '角色管理': 'admin-roles',
  '菜单管理': 'admin-menus',
  '系统设置': 'admin-settings',
  '系统管理': 'admin-settings',
  '平台审计': 'platform-audit',
  '平台事件中心': 'platform-events',
  '我的资料': 'profile',
  '用户手册': 'user-manual',
  '用户中心': 'platform-user',
  '认证中心': 'auth-center',
  '登录日志': 'auth-login',
  '接口访问': 'platform-api',
  '后台管理平台': 'platform-api',
  // New English values (from updated backend)
  'Dashboard': 'dashboard',
  'Projects': 'cmdb-projects',
  'Environments': 'cmdb-environments',
  'Servers': 'cmdb-servers',
  'Applications': 'cmdb-applications',
  'Release Deploy': 'deploy-release',
  'Deploy History': 'deploy-history',
  'Archive History': 'deploy-archived',
  'Archive Package': 'deploy-archive',
  'Aggregate Package': 'deploy-aggregate-package',
  'Aggregated History': 'aggregated-history',
  'Config Management': 'consul-config',
  'Batch Config Push': 'consul-batch-all',
  'Config Operations': 'consul-operations',
  'Jenkins Views': 'jenkins-views',
  'Monitor Dashboard': 'monitor-bigscreen',
  'Monitor Overview': 'monitor-overview',
  'Grafana Dashboards': 'monitor-dashboards',
  'Alert Events': 'alarm-events',
  'Alert Rules': 'alarm-rules',
  'Contacts': 'alarm-contacts',
  'Notify Channels': 'alarm-channels',
  'Notify Templates': 'alarm-templates',
  'Security Overview': 'security-overview',
  'File Integrity Monitoring': 'security-fim',
  'FIM Policies': 'security-fim-policies',
  'FIM Executions': 'security-fim-executions',
  'File Change Events': 'security-fim-events',
  'Integrity Alerts': 'security-fim-alerts',
  'Scan Tasks': 'security-tasks',
  'Security Assets': 'security-assets',
  'Vulnerabilities': 'security-vulnerabilities',
  'Vulnerability Tickets': 'security-tickets',
  'Vulnerability Knowledge DB': 'security-vuln-db',
  'Users': 'admin-users',
  'Roles': 'admin-roles',
  'Menus': 'admin-menus',
  'System Settings': 'admin-settings',
  'System Management': 'admin-settings',
  'Platform Audit': 'platform-audit',
  'Platform Events': 'platform-events',
  'My Profile': 'profile',
  'User Manual': 'user-manual',
  'User Center': 'platform-user',
  'Auth Center': 'auth-center',
  'Login Log': 'auth-login',
  'Login': 'auth-login',
  'API Access': 'platform-api',
  'Management Platform': 'platform-api',
  // Archive COALESCE defaults (fallback when menu_title/resource_name is NULL)
  '访问日志': 'accessLog',
  '操作审计': 'operationAudit',
  'Access Log': 'accessLog',
  'Operation Audit': 'operationAudit',
}

function getTranslatedMenuTitle(menuTitle?: string): string {
  if (!menuTitle) return '-'
  const key = menuTitleToMenuKey[menuTitle]
  if (!key) return menuTitle
  if (i18next.exists('menu:' + key)) return i18next.t('menu:' + key)
  if (i18next.exists('platform:' + key)) return i18next.t('platform:' + key)
  return menuTitle
}

// Maps old Chinese change_summary text to English equivalents.
// Handles template-based summaries from buildChangeSummary() and
// fixed handler-specific summaries from SetOperationAuditSummary().
const changeSummaryDirectMap: Record<string, string> = {
  // Old Chinese handler-specific summaries
  '创建了 FIM 巡检策略。': 'Created FIM policy.',
  '更新了 FIM 巡检策略。': 'Updated FIM policy.',
  '删除了 FIM 巡检策略。': 'Deleted FIM policy.',
  '新增了 FIM 监控目录配置。': 'Added FIM watch path config.',
  '更新了 FIM 监控目录配置。': 'Updated FIM watch path config.',
  '删除了 FIM 监控目录配置。': 'Deleted FIM watch path config.',
  '创建了用户。': 'Created user.',
  '更新了用户信息。': 'Updated user info.',
  '删除了用户。': 'Deleted user.',
  '创建了角色。': 'Created role.',
  '更新了角色配置。': 'Updated role config.',
  '删除了角色。': 'Deleted role.',
  '创建了菜单配置。': 'Created menu config.',
  '更新了菜单配置。': 'Updated menu config.',
  '删除了菜单配置。': 'Deleted menu config.',
  '更新了角色菜单授权。': 'Updated role menu authorization.',
  '更新了平台审计开关配置。': 'Updated platform audit log settings.',
  '更新了 FIM SSH 配置。': 'Updated FIM SSH config.',
  '创建了告警规则。': 'Created alert rule.',
  '更新了告警规则。': 'Updated alert rule.',
  '删除了告警规则。': 'Deleted alert rule.',
  '确认了告警。': 'Acknowledged alert.',
  '关闭了告警。': 'Closed alert.',
  '删除了告警。': 'Deleted alert.',
}

function translateChangeSummary(changeSummary?: string): string {
  if (!changeSummary) return '-'

  // Already English (new backend generates English text)
  if (/^[A-Z]/.test(changeSummary.trim())) return changeSummary

  // Try direct mapping for handler-specific summaries
  if (changeSummaryDirectMap[changeSummary]) return changeSummaryDirectMap[changeSummary]

  // Pattern 1: "访问了 X，查看当前列表或详情。"  (old visit_page template)
  const visitMatch = changeSummary.match(/^访问了 (.+)，查看当前列表或详情。$/)
  if (visitMatch) {
    const pageName = getTranslatedMenuTitle(visitMatch[1])
    return i18next.t('platform:accessedPage', '访问了 {{page}}。', { page: pageName })
  }

  // Pattern 2: "在 X 中执行了Y，请求路径 Z。"  (old operation template)
  const opMatch = changeSummary.match(/^在 (.+) 中执行了(.+)，请求路径 (.+)。$/)
  if (opMatch) {
    const modKey = moduleLabelKeyMap[opMatch[1]]
    const translatedModule = modKey ? i18next.t('platform:' + modKey, { defaultValue: opMatch[1] }) : opMatch[1]
    const actKey = actionLabelKeyMap[opMatch[2]]
    const translatedAction = actKey ? i18next.t('platform:' + actKey, { defaultValue: opMatch[2] }) : opMatch[2]
    return `Performed ${translatedAction} in ${translatedModule} at ${opMatch[3]}.`
  }

  return changeSummary
}

// Maps old Chinese and new English resource names to platform.json keys
const resourceNameKeyMap: Record<string, string> = {
  // Old Chinese values
  '工作台': 'resourceNameDashboard',
  '项目管理': 'resourceNameProjects',
  '环境管理': 'resourceNameEnvironments',
  '主机管理': 'resourceNameServers',
  '应用流水线管理': 'resourceNameApplications',
  '迭代部署': 'resourceNameReleaseDeploy',
  '部署记录': 'resourceNameDeployHistory',
  '归档历史': 'resourceNameArchiveHistory',
  '归档打包': 'resourceNameArchivePackage',
  '聚合打包': 'resourceNameAggregatePackage',
  '聚合历史': 'resourceNameAggregatedHistory',
  '配置管理': 'resourceNameConfigManagement',
  '批量配置下发': 'resourceNameBatchConfigPush',
  '配置操作记录': 'resourceNameConfigOperations',
  '视图管理': 'resourceNameJenkinsViews',
  '监控大屏': 'resourceNameMonitorDashboard',
  '监控概览': 'resourceNameMonitorOverview',
  'Grafana仪表盘': 'resourceNameGrafanaDashboards',
  '告警事件': 'resourceNameAlertEvents',
  '告警规则': 'resourceNameAlertRules',
  '联系人管理': 'resourceNameContacts',
  '通知渠道': 'resourceNameNotifyChannels',
  '通知模板': 'resourceNameNotifyTemplates',
  '安全概览': 'resourceNameSecurityOverview',
  '文件完整性巡检': 'resourceNameFIM',
  '巡检策略': 'resourceNameFIMPolicies',
  '执行记录': 'resourceNameFIMExecutions',
  '文件变更事件': 'resourceNameFileChangeEvents',
  '完整性告警': 'resourceNameIntegrityAlerts',
  '扫描任务': 'resourceNameScanTasks',
  '安全资产': 'resourceNameSecurityAssets',
  '漏洞管理': 'resourceNameVulnerabilities',
  '漏洞工单': 'resourceNameVulnerabilityTickets',
  '漏洞知识库': 'resourceNameVulnerabilityKnowledgeDB',
  '用户管理': 'resourceNameUsers',
  '角色管理': 'resourceNameRoles',
  '菜单管理': 'resourceNameMenus',
  '系统设置': 'resourceNameSystemSettings',
  '平台审计': 'resourceNamePlatformAudit',
  '平台事件中心': 'resourceNamePlatformEvents',
  '我的资料': 'resourceNameMyProfile',
  '用户手册': 'resourceNameUserManual',
  '个人资料': 'resourceNameMyProfile',
  '后台管理平台': 'resourceNameManagementPlatform',
  '菜单访问': 'resourceNameMenuAccess',
  'FIM 扫描任务': 'resourceNameFIMScanTask',
  '登录接口': 'resourceNameLogin',
  '接口访问': 'resourceNameAPIAccess',
  // New English values
  'Dashboard': 'resourceNameDashboard',
  'Projects': 'resourceNameProjects',
  'Environments': 'resourceNameEnvironments',
  'Servers': 'resourceNameServers',
  'Applications': 'resourceNameApplications',
  'Release Deploy': 'resourceNameReleaseDeploy',
  'Deploy History': 'resourceNameDeployHistory',
  'Archive History': 'resourceNameArchiveHistory',
  'Archive Package': 'resourceNameArchivePackage',
  'Aggregate Package': 'resourceNameAggregatePackage',
  'Aggregated History': 'resourceNameAggregatedHistory',
  'Config Management': 'resourceNameConfigManagement',
  'Batch Config Push': 'resourceNameBatchConfigPush',
  'Config Operations': 'resourceNameConfigOperations',
  'Jenkins Views': 'resourceNameJenkinsViews',
  'Monitor Dashboard': 'resourceNameMonitorDashboard',
  'Monitor Overview': 'resourceNameMonitorOverview',
  'Grafana Dashboards': 'resourceNameGrafanaDashboards',
  'Alert Events': 'resourceNameAlertEvents',
  'Alert Rules': 'resourceNameAlertRules',
  'Contacts': 'resourceNameContacts',
  'Notify Channels': 'resourceNameNotifyChannels',
  'Notify Templates': 'resourceNameNotifyTemplates',
  'Security Overview': 'resourceNameSecurityOverview',
  'File Integrity Monitoring': 'resourceNameFIM',
  'FIM Policies': 'resourceNameFIMPolicies',
  'FIM Executions': 'resourceNameFIMExecutions',
  'File Change Events': 'resourceNameFileChangeEvents',
  'Integrity Alerts': 'resourceNameIntegrityAlerts',
  'Scan Tasks': 'resourceNameScanTasks',
  'Security Assets': 'resourceNameSecurityAssets',
  'Vulnerabilities': 'resourceNameVulnerabilities',
  'Vulnerability Tickets': 'resourceNameVulnerabilityTickets',
  'Vulnerability Knowledge DB': 'resourceNameVulnerabilityKnowledgeDB',
  'Users': 'resourceNameUsers',
  'Roles': 'resourceNameRoles',
  'Menus': 'resourceNameMenus',
  'System Settings': 'resourceNameSystemSettings',
  'Platform Audit': 'resourceNamePlatformAudit',
  'Platform Events': 'resourceNamePlatformEvents',
  'My Profile': 'resourceNameMyProfile',
  'User Manual': 'resourceNameUserManual',
  'Profile': 'resourceNameMyProfile',
  'Menu Access': 'resourceNameMenuAccess',
  'FIM Scan Task': 'resourceNameFIMScanTask',
  'Integrity Alert': 'resourceNameIntegrityAlerts',
  'File Change Event': 'resourceNameFileChangeEvents',
  'Watch Path Config': 'resourceNameFIMPolicies',
  'Login': 'resourceNameLogin',
  'API Access': 'resourceNameAPIAccess',
  'Management Platform': 'resourceNameManagementPlatform',
}

function getTranslatedResourceName(t: TFunction<'platform'>, resourceName?: string): string {
  if (!resourceName) return '-'
  const key = resourceNameKeyMap[resourceName]
  return key ? t(key, resourceName) : resourceName
}

const { RangePicker } = DatePicker
const { Paragraph, Title } = Typography

type AuditTabKey = 'overview' | 'access' | 'operation' | 'login' | 'archive'

type DetailRecord = {
  title: string
  operator: string
  role: string
  module: string
  action: string
  requestPath: string
  requestMethod: string
  requestIP: string
  status: 'success' | 'failed'
  time: string
  durationMS: number
  summary: string
  params: string
  beforeData?: string
  afterData?: string
  errorMessage?: string
}

type AuditFilters = {
  username: string
  module?: string
  action?: string
  status?: 'success' | 'failed'
  requestIP: string
  requestPath: string
  range: [Dayjs | null, Dayjs | null] | null
}

export default function PlatformAuditPage() {
  const { t } = useTranslation('platform')
  const [activeTab, setActiveTab] = useState<AuditTabKey>('operation')
  const [loading, setLoading] = useState(false)
  const [overviewLoading, setOverviewLoading] = useState(false)
  const [accessLogs, setAccessLogs] = useState<PlatformAccessLog[]>([])
  const [operationLogs, setOperationLogs] = useState<PlatformAuditLog[]>([])
  const [loginLogs, setLoginLogs] = useState<PlatformLoginLog[]>([])
  const [archivedLogs, setArchivedLogs] = useState<PlatformArchivedLog[]>([])
  const [detail, setDetail] = useState<DetailRecord | null>(null)
  const [totals, setTotals] = useState({ access: 0, operation: 0, login: 0 })
  const [failedTotals, setFailedTotals] = useState({ access: 0, operation: 0, login: 0 })
  const [archiveStats, setArchiveStats] = useState<PlatformArchiveStats>({ access_total: 0, operation_total: 0, login_total: 0, total: 0 })
  const [retentionDays, setRetentionDays] = useState<number>(30)
  const [retentionLoading, setRetentionLoading] = useState(false)
  const [archiveType, setArchiveType] = useState<'all' | 'access' | 'operation' | 'login'>('all')
  const [retentionAction, setRetentionAction] = useState<'archive' | 'cleanup-online' | 'cleanup-archive' | null>(null)
  const [exportOpen, setExportOpen] = useState(false)
  const [exportLoading, setExportLoading] = useState(false)
  const [filters, setFilters] = useState<Record<'access' | 'operation' | 'login' | 'archive', AuditFilters>>({
    access: createEmptyFilters(),
    operation: createEmptyFilters(),
    login: createEmptyFilters(),
    archive: createEmptyFilters(),
  })

  const loadData = async (tab: AuditTabKey, currentFilters?: AuditFilters) => {
    if (tab === 'overview') {
      return
    }
    setLoading(true)
    try {
      if (tab === 'access') {
        const params = buildAuditParams(currentFilters || filters.access)
        const resp = await auditAPI.getAccessLogs({ page: 1, page_size: 50, ...params })
        setAccessLogs(resp.data ?? [])
        setTotals((prev) => ({ ...prev, access: resp.total ?? 0 }))
      } else if (tab === 'login') {
        const params = buildAuditParams(currentFilters || filters.login)
        const resp = await auditAPI.getLoginLogs({ page: 1, page_size: 50, ...params })
        setLoginLogs(resp.data ?? [])
        setTotals((prev) => ({ ...prev, login: resp.total ?? 0 }))
      } else if (tab === 'archive') {
        const params = buildAuditParams(currentFilters || filters.archive)
        const resp = await auditAPI.getArchivedLogs({ page: 1, page_size: 50, archive_type: archiveType, ...params })
        setArchivedLogs(resp.data ?? [])
      } else {
        const params = buildAuditParams(currentFilters || filters.operation)
        const resp = await auditAPI.getOperationLogs({ page: 1, page_size: 50, ...params })
        setOperationLogs(resp.data ?? [])
        setTotals((prev) => ({ ...prev, operation: resp.total ?? 0 }))
      }
    } catch {
      message.error(t('loadAuditDataFailed', '加载平台审计数据失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const loadOverview = async () => {
      setOverviewLoading(true)
      try {
        const [accessResp, operationResp, loginResp, failedAccessResp, failedOperationResp, failedLoginResp, archiveStatsResp] = await Promise.all([
          auditAPI.getAccessLogs({ page: 1, page_size: 1 }),
          auditAPI.getOperationLogs({ page: 1, page_size: 10 }),
          auditAPI.getLoginLogs({ page: 1, page_size: 1 }),
          auditAPI.getAccessLogs({ page: 1, page_size: 1, status: 'failed' }),
          auditAPI.getOperationLogs({ page: 1, page_size: 1, status: 'failed' }),
          auditAPI.getLoginLogs({ page: 1, page_size: 5, status: 'failed' }),
          auditAPI.getArchiveStats(),
        ])
        setTotals({
          access: accessResp.total ?? 0,
          operation: operationResp.total ?? 0,
          login: loginResp.total ?? 0,
        })
        setFailedTotals({
          access: failedAccessResp.total ?? 0,
          operation: failedOperationResp.total ?? 0,
          login: failedLoginResp.total ?? 0,
        })
        setAccessLogs(accessResp.data ?? [])
        setOperationLogs(operationResp.data ?? [])
        setLoginLogs(loginResp.data ?? [])
        setArchiveStats(archiveStatsResp)
      } catch {
        // keep zero values and let tab loads show more specific errors
      } finally {
        setOverviewLoading(false)
      }
    }
    void loadOverview()
  }, [])

  useEffect(() => {
    void loadData(activeTab)
  }, [activeTab, archiveType])

  const stats = useMemo(() => ({
    total: totals.access + totals.operation + totals.login,
    access: totals.access,
    operation: totals.operation,
    login: totals.login,
    failed: failedTotals.access + failedTotals.operation + failedTotals.login,
  }), [failedTotals, totals])

  const refreshCurrentView = async () => {
    await Promise.all([
      loadData(activeTab),
      (async () => {
        try {
          const [accessResp, operationResp, loginResp, failedAccessResp, failedOperationResp, failedLoginResp, archiveStatsResp] = await Promise.all([
            auditAPI.getAccessLogs({ page: 1, page_size: 1 }),
            auditAPI.getOperationLogs({ page: 1, page_size: 10 }),
            auditAPI.getLoginLogs({ page: 1, page_size: 1 }),
            auditAPI.getAccessLogs({ page: 1, page_size: 1, status: 'failed' }),
            auditAPI.getOperationLogs({ page: 1, page_size: 1, status: 'failed' }),
            auditAPI.getLoginLogs({ page: 1, page_size: 5, status: 'failed' }),
            auditAPI.getArchiveStats(),
          ])
          setTotals({
            access: accessResp.total ?? 0,
            operation: operationResp.total ?? 0,
            login: loginResp.total ?? 0,
          })
          setFailedTotals({
            access: failedAccessResp.total ?? 0,
            operation: failedOperationResp.total ?? 0,
            login: failedLoginResp.total ?? 0,
          })
          setArchiveStats(archiveStatsResp)
        } catch {
          // ignore follow-up refresh errors
        }
      })(),
    ])
  }

  const handleArchiveLogs = () => {
    setRetentionAction('archive')
  }

  const handleCleanupLogs = () => {
    setRetentionAction('cleanup-archive')
  }

  const handleCleanupOnlineLogs = () => {
    setRetentionAction('cleanup-online')
  }

  const handleConfirmRetentionAction = async () => {
    if (!retentionAction) {
      return
    }
    setRetentionLoading(true)
    try {
      const actionMap = {
        archive: auditAPI.archiveLogs,
        'cleanup-online': auditAPI.cleanupOnlineLogs,
        'cleanup-archive': auditAPI.cleanupLogs,
      } as const
      const result = await actionMap[retentionAction](retentionDays)
      if (retentionAction === 'archive') {
        message.success(t('archiveCompleted', '归档完成：访问 {{access}} 条，操作 {{operation}} 条，登录 {{login}} 条', { access: result.access_affected, operation: result.operation_affected, login: result.login_affected }))
      } else if (retentionAction === 'cleanup-online') {
        message.success(t('onlineCleanupCompleted', '在线清理完成：访问 {{access}} 条，操作 {{operation}} 条，登录 {{login}} 条', { access: result.access_affected, operation: result.operation_affected, login: result.login_affected }))
      } else {
        message.success(t('archiveCleanupCompleted', '归档清理完成：访问 {{access}} 条，操作 {{operation}} 条，登录 {{login}} 条', { access: result.access_affected, operation: result.operation_affected, login: result.login_affected }))
      }
      setRetentionAction(null)
      await refreshCurrentView()
    } catch (error) {
      if (retentionAction === 'archive') {
        message.error(getErrorMessage(error, t('archiveLogsFailed', '归档日志失败')))
      } else if (retentionAction === 'cleanup-online') {
        message.error(getErrorMessage(error, t('cleanupOnlineLogsFailed', '清理在线日志失败')))
      } else {
        message.error(getErrorMessage(error, t('cleanupArchiveLogsFailed', '清理归档日志失败')))
      }
    } finally {
      setRetentionLoading(false)
    }
  }

  const handleExportLogs = () => {
    if (activeTab === 'overview') {
      message.info(t('overviewPageNoExport', '总览页不支持直接导出，请切换到具体日志页后再导出。'))
      return
    }
    setExportOpen(true)
  }

  const handleConfirmExport = async () => {
    try {
      setExportLoading(true)
      const currentFilterKey: 'access' | 'operation' | 'login' | 'archive' = activeTab === 'archive' ? 'archive' : activeTab as 'access' | 'operation' | 'login'
      const params = buildAuditParams(filters[currentFilterKey])
      const result = await auditAPI.exportLogs({
        type: activeTab === 'archive' ? 'archive' : activeTab,
        ...(activeTab === 'archive' ? { archive_type: archiveType } : {}),
        ...params,
      })
      downloadBlob(result.filename, result.blob)
      setExportOpen(false)
      message.success(t('exportSuccess', '日志导出成功'))
    } catch (error) {
      message.error(getErrorMessage(error, t('exportFailed', '导出日志失败')))
    } finally {
      setExportLoading(false)
    }
  }

  const handleDeleteArchivedLog = async (record: PlatformArchivedLog) => {
    try {
      await auditAPI.deleteArchivedLog(record.archive_type, record.id)
      message.success(t('archivedLogDeleted', '归档日志已删除'))
      await refreshCurrentView()
    } catch {
      message.error(t('deleteArchivedLogFailed', '删除归档日志失败'))
    }
  }

  const handleDeleteAccessLog = async (record: PlatformAccessLog) => {
    try {
      await auditAPI.deleteAccessLog(record.id)
      message.success(t('accessLogDeleted', '访问日志已删除'))
      await refreshCurrentView()
    } catch {
      message.error(t('deleteAccessLogFailed', '删除访问日志失败'))
    }
  }

  const handleDeleteOperationLog = async (record: PlatformAuditLog) => {
    try {
      await auditAPI.deleteOperationLog(record.id)
      message.success(t('operationAuditDeleted', '操作审计已删除'))
      await refreshCurrentView()
    } catch {
      message.error(t('deleteOperationAuditFailed', '删除操作审计失败'))
    }
  }

  const handleDeleteLoginLog = async (record: PlatformLoginLog) => {
    try {
      await auditAPI.deleteLoginLog(record.id)
      message.success(t('loginLogDeleted', '登录日志已删除'))
      await refreshCurrentView()
    } catch {
      message.error(t('deleteLoginLogFailed', '删除登录日志失败'))
    }
  }

  const accessColumns: ColumnsType<PlatformAccessLog> = [
    { title: t('accessTime', '访问时间'), dataIndex: 'accessed_at', key: 'accessed_at', width: 176, render: formatDateTime },
    { title: t('operator', '操作人'), dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: t('menu', '菜单'), dataIndex: 'menu_title', key: 'menu_title', width: 180, render: (value?: string) => getTranslatedMenuTitle(value) },
    { title: t('role', '角色'), dataIndex: 'role', key: 'role', width: 100, render: (value?: string) => getTranslatedRole( value) },
    { title: t('requestPath', '请求路径'), dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    {
      title: t('status', '状态'),
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: (value: 'success' | 'failed') => <Tag color={value === 'success' ? 'green' : 'red'}>{value === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag>,
    },
    { title: t('requestIP', '请求IP'), dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: t('duration', '耗时'), dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: t('action', '操作'),
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapAccessDetail(record))}>{t('details', '详情')}</Button>
          {canEdit() && (
            <Popconfirm title={t('confirmDeleteAccessLog', '确定删除这条访问日志？')} onConfirm={() => void handleDeleteAccessLog(record)}>
              <Button type="link" danger>{t('delete', '删除')}</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const operationColumns: ColumnsType<PlatformAuditLog> = [
    { title: t('operationTime', '操作时间'), dataIndex: 'operated_at', key: 'operated_at', width: 176, render: formatDateTime },
    { title: t('operator', '操作人'), dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: t('module', '模块'), dataIndex: 'module', key: 'module', width: 120, render: (value?: string) => getTranslatedModule(t, value) },
    { title: t('actionLabel', '动作'), dataIndex: 'action_label', key: 'action_label', width: 120, render: (value?: string) => <Tag color="blue">{getTranslatedActionLabel(t, value) || t('executeOperation', '操作')}</Tag> },
    { title: t('resourceName', '资源名称'), dataIndex: 'resource_name', key: 'resource_name', ellipsis: true, render: (value?: string) => getTranslatedResourceName(t, value) },
    {
      title: t('status', '状态'),
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: (value: 'success' | 'failed') => <Tag color={value === 'success' ? 'green' : 'red'}>{value === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag>,
    },
    { title: t('requestIP', '请求IP'), dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: t('duration', '耗时'), dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: t('action', '操作'),
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapOperationDetail(record))}>{t('details', '详情')}</Button>
          {canEdit() && (
            <Popconfirm title={t('confirmDeleteOperationAudit', '确定删除这条操作审计？')} onConfirm={() => void handleDeleteOperationLog(record)}>
              <Button type="link" danger>{t('delete', '删除')}</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const loginColumns: ColumnsType<PlatformLoginLog> = [
    { title: t('loginTime', '登录时间'), dataIndex: 'logged_in_at', key: 'logged_in_at', width: 176, render: formatDateTime },
    { title: t('account', '账号'), dataIndex: 'username', key: 'username', width: 120 },
    { title: t('role', '角色'), dataIndex: 'role', key: 'role', width: 120, render: (value?: string) => getTranslatedRole( value) },
    { title: t('loginType', '登录类型'), dataIndex: 'login_type', key: 'login_type', width: 120, render: (value?: string) => value || 'password' },
    { title: t('requestPath', '请求路径'), dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    {
      title: t('status', '状态'),
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: (value: 'success' | 'failed') => <Tag color={value === 'success' ? 'green' : 'red'}>{value === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag>,
    },
    { title: t('requestIP', '请求IP'), dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: t('duration', '耗时'), dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: t('action', '操作'),
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapLoginDetail(record))}>{t('details', '详情')}</Button>
          {canEdit() && (
            <Popconfirm title={t('confirmDeleteLoginLog', '确定删除这条登录日志？')} onConfirm={() => void handleDeleteLoginLog(record)}>
              <Button type="link" danger>{t('delete', '删除')}</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const archivedColumns: ColumnsType<PlatformArchivedLog> = [
    { title: t('archiveTime', '归档时间'), dataIndex: 'archived_at', key: 'archived_at', width: 176, render: formatDateTime },
    { title: t('originalTime', '原始时间'), dataIndex: 'occurred_at', key: 'occurred_at', width: 176, render: formatDateTime },
    { title: t('type', '类型'), dataIndex: 'archive_type', key: 'archive_type', width: 100, render: renderArchiveTypeTag },
    { title: t('operator', '操作人'), dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: t('title', '标题'), dataIndex: 'title', key: 'title', ellipsis: true, render: (value?: string) => getTranslatedMenuTitle(value) },
    { title: t('requestPath', '请求路径'), dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    { title: t('requestIP', '请求IP'), dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: t('status', '状态'), dataIndex: 'operation_status', key: 'operation_status', width: 92, render: (value: 'success' | 'failed') => <Tag color={value === 'success' ? 'green' : 'red'}>{value === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag> },
    {
      title: t('action', '操作'),
      key: 'op',
      width: 96,
      render: (_: unknown, record) => (
        canEdit() ? (
          <Popconfirm title={t('confirmDeleteArchivedLog', '确定删除这条归档日志？')} onConfirm={() => void handleDeleteArchivedLog(record)}>
            <Button type="link" danger>{t('delete', '删除')}</Button>
          </Popconfirm>
        ) : '-'
      ),
    },
  ]

  function renderArchiveTypeTag(value: 'access' | 'operation' | 'login') {
    if (value === 'access') return <Tag color="blue">{t('accessLog', '访问日志')}</Tag>
    if (value === 'operation') return <Tag color="purple">{t('operationAudit', '操作审计')}</Tag>
    return <Tag color="gold">{t('loginLog', '登录日志')}</Tag>
  }

  function mapAccessDetail(record: PlatformAccessLog): DetailRecord {
    return {
      title: getTranslatedMenuTitle(record.menu_title) || t('accessLog', '访问日志'),
      operator: displayOperator(record.real_name, record.username),
      role: getTranslatedRole( record.role),
      module: getTranslatedMenuTitle(record.menu_title) || t('accessLog', '访问日志'),
      action: t('viewPage', '访问页面'),
      requestPath: record.request_path,
      requestMethod: record.request_method,
      requestIP: record.request_ip,
      status: record.operation_status,
      time: formatDateTime(record.accessed_at),
      durationMS: record.duration_ms,
      summary: t('accessedPage', '访问了 {{page}}。', { page: getTranslatedMenuTitle(record.menu_title) || record.request_path }),
      params: '{}',
      beforeData: '',
      afterData: '',
      errorMessage: record.error_message,
    }
  }

  function mapOperationDetail(record: PlatformAuditLog): DetailRecord {
    return {
      title: getTranslatedResourceName(t, record.resource_name) || t('operationAudit', '操作审计'),
      operator: displayOperator(record.real_name, record.username),
      role: getTranslatedRole( record.role),
      module: getTranslatedModule(t, record.module),
      action: getTranslatedActionLabel(t, record.action_label) || record.action || t('executeOperation', '操作'),
      requestPath: record.request_path,
      requestMethod: record.request_method,
      requestIP: record.request_ip,
      status: record.operation_status,
      time: formatDateTime(record.operated_at),
      durationMS: record.duration_ms,
      summary: translateChangeSummary(record.change_summary),
      params: record.request_params_json || '{}',
      beforeData: record.before_data_json || '',
      afterData: record.after_data_json || '',
      errorMessage: record.error_message,
    }
  }

  function mapLoginDetail(record: PlatformLoginLog): DetailRecord {
    return {
      title: t('loginLog', '登录日志'),
      operator: displayOperator(record.real_name, record.username),
      role: getTranslatedRole( record.role),
      module: t('authenticateCenter', '认证中心'),
      action: record.operation_status === 'success' ? t('loginSuccess', '登录成功') : t('loginFailed', '登录失败'),
      requestPath: record.request_path,
      requestMethod: record.request_method,
      requestIP: record.request_ip,
      status: record.operation_status,
      time: formatDateTime(record.logged_in_at),
      durationMS: record.duration_ms,
      summary: record.error_message ? t('loginFailedDesc', '登录失败：{{error}}', { error: record.error_message }) : t('loginSuccessDesc', '使用账号密码登录后台平台。'),
      params: '{}',
      beforeData: '',
      afterData: '',
      errorMessage: record.error_message,
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, flexWrap: 'wrap' }}>
          <div>
            <Title level={4} style={{ margin: 0 }}>{t('platformAudit', '平台审计')}</Title>
            <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
              {t('platformAuditDesc', '一个入口统一查看访问日志、操作审计与登录日志，符合当前平台的后台结构和使用习惯。')}
            </Paragraph>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void loadData(activeTab)}>{t('refresh', '刷新')}</Button>
          </Space>
        </div>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title={t('totalAuditRecords', '总审计记录')} value={stats.total} prefix={<FileSearchOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title={t('accessLog', '访问日志')} value={stats.access} prefix={<FileSearchOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title={t('operationAudit', '操作审计')} value={stats.operation} prefix={<SafetyCertificateOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title={t('loginLog', '登录日志')} value={stats.login} prefix={<LoginOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title={t('archivedLog', '归档日志')} value={archiveStats.total} prefix={<InboxOutlined />} /></Card>
        </Col>
      </Row>

      <Card bodyStyle={{ padding: 0 }}>
        <Tabs
          activeKey={activeTab}
          onChange={(key) => {
            setDetail(null)
            setActiveTab(key as AuditTabKey)
          }}
          style={{ padding: '0 16px' }}
          items={[
            {
              key: 'overview',
              label: t('overview', '总览'),
              children: (
                <AuditOverview
                  t={t}
                  loading={overviewLoading}
                  stats={stats}
                  failedTotals={failedTotals}
                  archiveStats={archiveStats}
                  accessLogs={accessLogs}
                  operationLogs={operationLogs}
                  loginLogs={loginLogs}
                />
              ),
            },
            {
              key: 'access',
              label: t('accessLog', '访问日志'),
              children: (
                <AuditTablePanel
                  t={t}
                  tab="access"
                  loading={loading}
                  title={t('accessLog', '访问日志')}
                  columns={accessColumns}
                  data={accessLogs}
                  filters={filters.access}
                  onFiltersChange={(next) => updateFilters('access', next, setFilters)}
                  onSearch={() => void loadData('access')}
                  onReset={() => resetFilters('access', setFilters, setDetail, () => void loadData('access', createEmptyFilters()))}
                />
              ),
            },
            {
              key: 'operation',
              label: t('operationAudit', '操作审计'),
              children: (
                <AuditTablePanel
                  t={t}
                  tab="operation"
                  loading={loading}
                  title={t('operationAudit', '操作审计')}
                  columns={operationColumns}
                  data={operationLogs}
                  filters={filters.operation}
                  onFiltersChange={(next) => updateFilters('operation', next, setFilters)}
                  onSearch={() => void loadData('operation')}
                  onReset={() => resetFilters('operation', setFilters, setDetail, () => void loadData('operation', createEmptyFilters()))}
                />
              ),
            },
            {
              key: 'login',
              label: t('loginLog', '登录日志'),
              children: (
                <AuditTablePanel
                  t={t}
                  tab="login"
                  loading={loading}
                  title={t('loginLog', '登录日志')}
                  columns={loginColumns}
                  data={loginLogs}
                  filters={filters.login}
                  onFiltersChange={(next) => updateFilters('login', next, setFilters)}
                  onSearch={() => void loadData('login')}
                  onReset={() => resetFilters('login', setFilters, setDetail, () => void loadData('login', createEmptyFilters()))}
                />
              ),
            },
            {
              key: 'archive',
              label: t('archivedLogs', '归档日志'),
              children: (
                <div style={{ padding: '0 16px 16px' }}>
                  <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
                    <Col xs={24} lg={8}>
                      <Card size="small">
                        <Statistic title={t('archiveTotal', '归档总量')} value={archiveStats.total} />
                        <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                          {t('recentArchived', '最近归档')}：{archiveStats.latest_archived ? formatDateTime(archiveStats.latest_archived) : '-'}
                        </Paragraph>
                      </Card>
                    </Col>
                    <Col xs={24} lg={16}>
                      <Card size="small">
                        <Row gutter={[16, 16]}>
                          <Col span={8}><Statistic title={t('accessArchive', '访问归档')} value={archiveStats.access_total} /></Col>
                          <Col span={8}><Statistic title={t('operationArchive', '操作归档')} value={archiveStats.operation_total} /></Col>
                          <Col span={8}><Statistic title={t('loginArchive', '登录归档')} value={archiveStats.login_total} /></Col>
                        </Row>
                      </Card>
                    </Col>
                  </Row>
                  <Card
                    size="small"
                    title={t('archiveLogTitle', '归档日志')}
                    extra={(
                      <Space>
                        <Select
                          value={retentionDays}
                          style={{ width: 150 }}
                          onChange={setRetentionDays}
                          options={[
                            { value: 7, label: t('retentionWeek1', '保留 1 周') },
                            { value: 30, label: t('retentionMonth1', '保留 1 个月') },
                            { value: 180, label: t('retentionMonth6', '保留 6 个月') },
                          ]}
                        />
                        <Button icon={<InboxOutlined />} loading={retentionLoading} onClick={handleArchiveLogs}>{t('archiveOldLogs', '归档旧日志')}</Button>
                        <Button danger loading={retentionLoading} onClick={handleCleanupOnlineLogs}>{t('cleanupOnlineLogs', '清理在线日志')}</Button>
                        <Button danger icon={<DeleteOutlined />} loading={retentionLoading} onClick={handleCleanupLogs}>{t('cleanupArchivedLogs', '清理归档日志')}</Button>
                        <Divider type="vertical" />
                        <Button type="primary" icon={<DownloadOutlined />} onClick={handleExportLogs}>{t('exportLogs', '导出日志')}</Button>
                        <Button onClick={() => resetFilters('archive', setFilters, setDetail, () => void loadData('archive', createEmptyFilters()))}>{t('reset', '重置')}</Button>
                        <Button type="primary" onClick={() => void loadData('archive')}>{t('query', '查询')}</Button>
                        <Select
                          value={archiveType}
                          style={{ width: 140 }}
                          onChange={setArchiveType}
                          options={[
                            { value: 'all', label: t('allTypes', '全部类型') },
                            { value: 'access', label: t('accessLog', '访问日志') },
                            { value: 'operation', label: t('operationAudit', '操作审计') },
                            { value: 'login', label: t('loginLog', '登录日志') },
                          ]}
                        />
                        <Button onClick={() => void loadData('archive')}>{t('refresh', '刷新')}</Button>
                      </Space>
                    )}
                  >
                    <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Input
                          placeholder={t('filterOperator', '操作人')}
                          value={filters.archive.username}
                          onChange={(event) => updateFilters('archive', { ...filters.archive, username: event.target.value }, setFilters)}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Select
                          placeholder={t('filterStatus', '状态')}
                          style={{ width: '100%' }}
                          allowClear
                          value={filters.archive.status}
                          onChange={(value) => updateFilters('archive', { ...filters.archive, status: value }, setFilters)}
                          options={[
                            { value: 'success', label: t('success', '成功') },
                            { value: 'failed', label: t('failed', '失败') },
                          ]}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Input
                          placeholder={t('filterRequestIP', '请求IP')}
                          value={filters.archive.requestIP}
                          onChange={(event) => updateFilters('archive', { ...filters.archive, requestIP: event.target.value }, setFilters)}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={6}>
                        <Input
                          placeholder={t('filterRequestPath', '请求路径')}
                          value={filters.archive.requestPath}
                          onChange={(event) => updateFilters('archive', { ...filters.archive, requestPath: event.target.value }, setFilters)}
                        />
                      </Col>
                      <Col xs={24} md={24} xl={6}>
                        <RangePicker
                          showTime
                          style={{ width: '100%' }}
                          value={filters.archive.range}
                          onChange={(value) => updateFilters('archive', { ...filters.archive, range: (value as [Dayjs | null, Dayjs | null] | null) ?? null }, setFilters)}
                        />
                      </Col>
                    </Row>
                    <Table
                      rowKey={(record) => `${record.archive_type}-${record.id}-${record.archived_at}`}
                      columns={archivedColumns}
                      dataSource={archivedLogs}
                      pagination={{ pageSize: 10, showTotal: (total) => `${t('total', '共')} ${total} ${t('items', '条')}` }}
                      scroll={{ x: 1200 }}
                    />
                  </Card>
                </div>
              ),
            },
          ]}
        />
      </Card>

      <Drawer title={t('auditDetail', '审计详情')} placement="right" width={520} open={!!detail} onClose={() => setDetail(null)}>
        {detail && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label={t('operator', '操作人')}>{detail.operator}</Descriptions.Item>
              <Descriptions.Item label={t('role', '角色')}>{detail.role}</Descriptions.Item>
              <Descriptions.Item label={t('module', '模块')}>{detail.module}</Descriptions.Item>
              <Descriptions.Item label={t('actionLabel', '动作')}>{detail.action}</Descriptions.Item>
              <Descriptions.Item label={t('status', '状态')}>{detail.status === 'success' ? <Tag color="green">{t('success', '成功')}</Tag> : <Tag color="red">{t('failed', '失败')}</Tag>}</Descriptions.Item>
              <Descriptions.Item label={t('requestIP', '请求IP')}>{detail.requestIP}</Descriptions.Item>
              <Descriptions.Item label={t('requestPath', '请求路径')} span={2}>{detail.requestPath}</Descriptions.Item>
              <Descriptions.Item label={t('requestMethod', '请求方法')}>{detail.requestMethod}</Descriptions.Item>
              <Descriptions.Item label={t('time', '操作时间')}>{detail.time}</Descriptions.Item>
              <Descriptions.Item label={t('duration', '耗时')} span={2}>{detail.durationMS}ms</Descriptions.Item>
            </Descriptions>
            <Card size="small" title={t('summary', '摘要')}>
              <Paragraph style={{ marginBottom: 0 }}>{detail.summary}</Paragraph>
            </Card>
            <Card size="small" title={t('requestParams', '请求参数')}>
              <pre style={preStyle}>{detail.params || '-'}</pre>
            </Card>
            <Card size="small" title={t('beforeChange', '变更前')}>
              <pre style={preStyle}>{detail.beforeData || '-'}</pre>
            </Card>
            <Card size="small" title={t('afterChange', '变更后')}>
              <pre style={preStyle}>{detail.afterData || '-'}</pre>
            </Card>
            <Card size="small" title={t('errorInfo', '错误信息')}>
              <Paragraph style={{ marginBottom: 0 }}>{detail.errorMessage || '-'}</Paragraph>
            </Card>
          </Space>
        )}
      </Drawer>
      <Modal
        title={getRetentionModalTitle(t, retentionAction)}
        open={!!retentionAction}
        onCancel={() => setRetentionAction(null)}
        onOk={() => void handleConfirmRetentionAction()}
        confirmLoading={retentionLoading}
        okText={retentionAction === 'archive' ? t('confirmArchive', '确认归档') : t('confirmCleanup', '确认清理')}
        okButtonProps={retentionAction === 'archive' ? undefined : { danger: true }}
        cancelText={t('cancel', '取消')}
      >
        <Paragraph style={{ marginBottom: 0 }}>
          {getRetentionModalDescription(t, retentionAction, retentionDays)}
        </Paragraph>
      </Modal>
      <Modal
        title={t('exportLogsTitle', '导出日志')}
        open={exportOpen}
        onCancel={() => setExportOpen(false)}
        onOk={() => void handleConfirmExport()}
        okText={t('confirmExport', '确认导出')}
        cancelText={t('cancel', '取消')}
        confirmLoading={exportLoading}
      >
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Paragraph style={{ marginBottom: 0 }}>
            {t('exportDesc', '当前将导出"{{type}}"在当前筛选条件下的全部结果。', { type: getExportTabLabel(t, activeTab, archiveType) })}
          </Paragraph>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t('exportHint', '导出格式为 CSV，由后端直接生成，适合留存和二次分析。')}
          </Paragraph>
        </Space>
      </Modal>
    </div>
  )
}

function AuditOverview({
  t,
  loading,
  stats,
  failedTotals,
  archiveStats,
  accessLogs,
  operationLogs,
  loginLogs,
}: {
  t: TFunction<'platform'>
  loading: boolean
  stats: { total: number; access: number; operation: number; login: number; failed: number }
  failedTotals: { access: number; operation: number; login: number }
  archiveStats: PlatformArchiveStats
  accessLogs: PlatformAccessLog[]
  operationLogs: PlatformAuditLog[]
  loginLogs: PlatformLoginLog[]
}) {
  const recentOperation = operationLogs[0]
  const recentArchiveOps = operationLogs.filter((item) => item.action === 'archive' || item.action === 'cleanup').slice(0, 2)
  const recentLogin = loginLogs[0]
  const recentAccess = accessLogs[0]

  return (
    <div style={{ padding: 16 }}>
      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <Card title={t('auditOverview', '审计态势')} size="small">
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <Card size="small" bordered={false} style={{ background: '#fafafa' }}>
                    <Statistic title={t('accumulatedAuditRecords', '累计审计记录')} value={stats.total} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {t('accumulatedAuditDesc', '统一汇总访问日志、操作审计和登录日志。')}
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={12}>
                  <Card size="small" bordered={false} style={{ background: stats.failed > 0 ? '#fff7e6' : '#f6ffed' }}>
                    <Statistic title={t('failedRecords', '失败记录')} value={stats.failed} valueStyle={{ color: stats.failed > 0 ? '#d46b08' : '#389e0d' }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {stats.failed > 0 ? t('failedRecordsTip1', '建议优先查看登录失败和执行失败操作。') : t('failedRecordsTip2', '当前未发现失败审计记录。')}
                    </Paragraph>
                  </Card>
                </Col>
              </Row>
              <Row gutter={[16, 16]} style={{ marginTop: 4 }}>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title={t('accessLog', '访问日志')} value={stats.access} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {t('failedCount', '失败 {{count}} 条', { count: failedTotals.access })}
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title={t('operationAudit', '操作审计')} value={stats.operation} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {t('failedCount', '失败 {{count}} 条', { count: failedTotals.operation })}
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title={t('loginLog', '登录日志')} value={stats.login} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {t('failedCount', '失败 {{count}} 条', { count: failedTotals.login })}
                    </Paragraph>
                  </Card>
                </Col>
              </Row>
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title={t('auditCoverage', '审计覆盖范围')} size="small">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#1677ff', marginTop: 8, flexShrink: 0 }} />
                  <div>
                    <Paragraph style={{ marginBottom: 4, fontWeight: 600 }}>{t('loginChain', '登录链路')}</Paragraph>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>{t('loginChainDesc', '记录后台登录请求、账号、IP、结果和耗时。')}</Paragraph>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#1677ff', marginTop: 8, flexShrink: 0 }} />
                  <div>
                    <Paragraph style={{ marginBottom: 4, fontWeight: 600 }}>{t('accessChain', '访问链路')}</Paragraph>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>{t('accessChainDesc', '记录页面与关键 GET 接口访问，适合追溯用户看过什么。')}</Paragraph>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#1677ff', marginTop: 8, flexShrink: 0 }} />
                  <div>
                    <Paragraph style={{ marginBottom: 4, fontWeight: 600 }}>{t('operationChain', '操作链路')}</Paragraph>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>{t('operationChainDesc', '记录非 GET 关键操作，适合排查谁在什么时候改了什么。')}</Paragraph>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#1677ff', marginTop: 8, flexShrink: 0 }} />
                  <div>
                    <Paragraph style={{ marginBottom: 4, fontWeight: 600 }}>{t('archiveChain', '归档链路')}</Paragraph>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>{t('archiveChainDesc', '已归档 {{count}} 条历史日志，支持按保留周期做归档与清理。', { count: archiveStats.total })}</Paragraph>
                  </div>
                </div>
              </Space>
            </Card>
          </Col>

          <Col xs={24} xl={8}>
            <Card title={t('recentOperation', '最近操作')} size="small">
              <div>
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 4, fontWeight: 600 }}>
                        {getTranslatedActionLabel(t, recentOperation?.action_label) || recentOperation?.action || t('tableNoData', '暂无数据')}
                      </Paragraph>
                      <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 0 }}>
                        {recentOperation ? `${getTranslatedModule(t, recentOperation.module)} / ${getTranslatedResourceName(t, recentOperation.resource_name) || t('unnamedResource', '未命名资源')}` : t('noOperationRecords', '当前还没有操作审计记录')}
                      </Paragraph>
                    </div>
                    {recentOperation?.operation_status ? <Tag color={recentOperation.operation_status === 'success' ? 'green' : 'red'}>{recentOperation.operation_status === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag> : null}
                  </div>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('timeLabel', '时间：')}{recentOperation ? formatDateTime(recentOperation.operated_at) : '-'}
                  </Paragraph>
                </Space>
              </div>
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title={t('recentLogin', '最近登录')} size="small">
              <div>
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 4, fontWeight: 600 }}>
                        {displayOperator(recentLogin?.real_name, recentLogin?.username) || t('tableNoData', '暂无数据')}
                      </Paragraph>
                      <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 0 }}>
                        {recentLogin ? `${recentLogin.request_ip} / ${recentLogin.login_type || 'password'}` : t('noLoginRecords', '当前还没有登录日志')}
                      </Paragraph>
                    </div>
                    {recentLogin?.operation_status ? <Tag color={recentLogin.operation_status === 'success' ? 'green' : 'red'}>{recentLogin.operation_status === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag> : null}
                  </div>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('timeLabel', '时间：')}{recentLogin ? formatDateTime(recentLogin.logged_in_at) : '-'}
                  </Paragraph>
                </Space>
              </div>
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title={t('recentAccess', '最近访问')} size="small">
              <div>
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 4, fontWeight: 600 }}>
                        {getTranslatedMenuTitle(recentAccess?.menu_title) || recentAccess?.request_path || t('tableNoData', '暂无数据')}
                      </Paragraph>
                      <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 0 }}>
                        {recentAccess ? `${displayOperator(recentAccess.real_name, recentAccess.username)} / ${recentAccess.request_ip}` : t('noAccessRecords', '当前还没有访问日志')}
                      </Paragraph>
                    </div>
                    {recentAccess?.operation_status ? <Tag color={recentAccess.operation_status === 'success' ? 'green' : 'red'}>{recentAccess.operation_status === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag> : null}
                  </div>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('timeLabel', '时间：')}{recentAccess ? formatDateTime(recentAccess.accessed_at) : '-'}
                  </Paragraph>
                </Space>
              </div>
            </Card>
          </Col>
          <Col xs={24}>
            <Card title={t('recentArchive', '最近归档维护')} size="small">
              <Row gutter={[16, 16]}>
                {recentArchiveOps.length > 0 ? recentArchiveOps.map((item) => (
                  <Col xs={24} md={12} key={item.id}>
                    <div>
                      <Space direction="vertical" size={12} style={{ width: '100%' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
                          <div style={{ flex: 1, minWidth: 0 }}>
                            <Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 4, fontWeight: 600 }}>
                              {getTranslatedActionLabel(t, item.action_label) || item.action || t('logMaintenance', '日志维护')}
                            </Paragraph>
                            <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 0 }}>
                              {`${displayOperator(item.real_name, item.username)} / ${translateChangeSummary(item.change_summary || item.request_path)}`}
                            </Paragraph>
                          </div>
                          {item.operation_status ? <Tag color={item.operation_status === 'success' ? 'green' : 'red'}>{item.operation_status === 'success' ? t('success', '成功') : t('failed', '失败')}</Tag> : null}
                        </div>
                        <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                          {t('timeLabel', '时间：')}{formatDateTime(item.operated_at)}
                        </Paragraph>
                      </Space>
                    </div>
                  </Col>
                )) : (
                  <Col xs={24}>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                      {t('noArchiveRecords', '当前还没有归档或清理操作记录。')}
                    </Paragraph>
                  </Col>
                )}
              </Row>
            </Card>
          </Col>

          <Col xs={24}>
            <Card title={t('auditSuggestions', '审计使用建议')} size="small">
              <Row gutter={[16, 16]}>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>{t('suggestionLogin', '先看登录日志')}</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('suggestionLoginDesc', '适合判断是谁、从哪个 IP 登录，以及是否存在异常登录失败。')}
                  </Paragraph>
                </Col>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>{t('suggestionAccess', '再看访问日志')}</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('suggestionAccessDesc', '适合确认登录后访问了哪些页面和关键查询接口。')}
                  </Paragraph>
                </Col>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>{t('suggestionOperation', '最后看操作审计')}</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    {t('suggestionOperationDesc', '适合还原最终执行了哪些变更、测试或处置动作。')}
                  </Paragraph>
                </Col>
              </Row>
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  )
}

function AuditTablePanel<T extends { id: number }>({
  t,
  tab,
  title,
  data,
  columns,
  loading,
  filters,
  onFiltersChange,
  onSearch,
  onReset,
}: {
  t: TFunction<'platform'>
  tab: 'access' | 'operation' | 'login'
  title: string
  data: T[]
  columns: ColumnsType<T>
  loading: boolean
  filters: AuditFilters
  onFiltersChange: (next: AuditFilters) => void
  onSearch: () => void
  onReset: () => void
}) {
  return (
    <div style={{ padding: '0 16px 16px' }}>
      <Card
        size="small"
        title={title}
        extra={(
          <Space>
            <Button onClick={onReset}>{t('reset', '重置')}</Button>
            <Button type="primary" onClick={onSearch}>{t('query', '查询')}</Button>
          </Space>
        )}
      >
        <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={12} md={8} xl={4}>
            <Input
              placeholder={t('filterOperator', '操作人')}
              value={filters.username}
              onChange={(event) => onFiltersChange({ ...filters, username: event.target.value })}
            />
          </Col>
          {tab === 'operation' && (
            <>
              <Col xs={24} sm={12} md={8} xl={4}>
                <Select
                  placeholder={t('filterModule', '模块')}
                  style={{ width: '100%' }}
                  allowClear
                  value={filters.module}
                  onChange={(value) => onFiltersChange({ ...filters, module: value })}
                  options={[
                    { value: 'security', label: t('moduleSecurityCenter', '安全中心') },
                    { value: 'system', label: t('moduleSystemManagement', '系统管理') },
                    { value: 'alert', label: t('moduleAlertCenter', '告警中心') },
                    { value: 'audit', label: t('modulePlatformAudit', '平台审计') },
                  ]}
                />
              </Col>
              <Col xs={24} sm={12} md={8} xl={4}>
                <Select
                  placeholder={t('filterAction', '动作')}
                  style={{ width: '100%' }}
                  allowClear
                  value={filters.action}
                  onChange={(value) => onFiltersChange({ ...filters, action: value })}
                  options={[
                    { value: 'execute', label: t('actionLabelExecute', '执行操作') },
                    { value: 'update', label: t('actionLabelUpdateConfig', '更新配置') },
                    { value: 'delete', label: t('actionLabelDelete', '删除') },
                    { value: 'create', label: t('actionLabelCreate', '新增') },
                  ]}
                />
              </Col>
            </>
          )}
          <Col xs={24} sm={12} md={8} xl={4}>
            <Select
              placeholder={t('filterStatus', '状态')}
              style={{ width: '100%' }}
              allowClear
              value={filters.status}
              onChange={(value) => onFiltersChange({ ...filters, status: value })}
              options={[
                { value: 'success', label: t('success', '成功') },
                { value: 'failed', label: t('failed', '失败') },
              ]}
            />
          </Col>
          <Col xs={24} sm={12} md={8} xl={4}>
            <Input
              placeholder={t('filterRequestIP', '请求IP')}
              value={filters.requestIP}
              onChange={(event) => onFiltersChange({ ...filters, requestIP: event.target.value })}
            />
          </Col>
          <Col xs={24} sm={12} md={8} xl={tab === 'operation' ? 8 : 4}>
            <Input
              placeholder={t('filterRequestPath', '请求路径')}
              value={filters.requestPath}
              onChange={(event) => onFiltersChange({ ...filters, requestPath: event.target.value })}
            />
          </Col>
          <Col xs={24} md={24} xl={tab === 'operation' ? 24 : 8}>
            <RangePicker
              showTime
              style={{ width: '100%' }}
              value={filters.range}
              onChange={(value) => onFiltersChange({ ...filters, range: (value as [Dayjs | null, Dayjs | null] | null) ?? null })}
            />
          </Col>
        </Row>

        <Spin spinning={loading}>
          <Table
            rowKey="id"
            columns={columns}
            dataSource={data}
            pagination={{ pageSize: 10, showTotal: (total) => `${t('total', '共')} ${total} ${t('items', '条')}` }}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>
    </div>
  )
}

function displayOperator(realName?: string, username?: string) {
  return realName || username || '-'
}

const preStyle: React.CSSProperties = {
  margin: 0,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontSize: 12,
  lineHeight: 1.7,
}

function createEmptyFilters(): AuditFilters {
  return {
    username: '',
    module: undefined,
    action: undefined,
    status: undefined,
    requestIP: '',
    requestPath: '',
    range: null,
  }
}

function buildAuditParams(filters: AuditFilters) {
  const params: Record<string, string> = {}
  if (filters.username.trim()) params.username = filters.username.trim()
  if (filters.module) params.module = filters.module
  if (filters.action) params.action = filters.action
  if (filters.status) params.status = filters.status
  if (filters.requestIP.trim()) params.request_ip = filters.requestIP.trim()
  if (filters.requestPath.trim()) params.request_path = filters.requestPath.trim()
  if (filters.range?.[0]) params.start = filters.range[0].format('YYYY-MM-DD HH:mm:ss')
  if (filters.range?.[1]) params.end = filters.range[1].format('YYYY-MM-DD HH:mm:ss')
  return params
}

function updateFilters(
  key: 'access' | 'operation' | 'login' | 'archive',
  next: AuditFilters,
  setFilters: React.Dispatch<React.SetStateAction<Record<'access' | 'operation' | 'login' | 'archive', AuditFilters>>>
) {
  setFilters((current) => ({ ...current, [key]: next }))
}

function resetFilters(
  key: 'access' | 'operation' | 'login' | 'archive',
  setFilters: React.Dispatch<React.SetStateAction<Record<'access' | 'operation' | 'login' | 'archive', AuditFilters>>>,
  setDetail: React.Dispatch<React.SetStateAction<DetailRecord | null>>,
  callback: () => void
) {
  setDetail(null)
  setFilters((current) => ({ ...current, [key]: createEmptyFilters() }))
  callback()
}

function getRetentionModalTitle(t: TFunction<'platform'>, action: 'archive' | 'cleanup-online' | 'cleanup-archive' | null) {
  if (action === 'archive') return t('archiveModalTitle', '归档旧日志')
  if (action === 'cleanup-online') return t('cleanupOnlineModalTitle', '清理在线日志')
  if (action === 'cleanup-archive') return t('cleanupArchiveModalTitle', '清理归档日志')
  return t('logMaintenance', '日志维护')
}

function getRetentionModalDescription(t: TFunction<'platform'>, action: 'archive' | 'cleanup-online' | 'cleanup-archive' | null, retentionDays: number) {
  if (action === 'archive') {
    return t('archiveModalDesc', '将归档 {{days}} 天前的平台审计日志，并从在线日志中移除。', { days: retentionDays })
  }
  if (action === 'cleanup-online') {
    return t('cleanupOnlineModalDesc', '将永久删除在线表中 {{days}} 天前的平台审计日志，不会归档到归档表。此操作不可恢复。', { days: retentionDays })
  }
  if (action === 'cleanup-archive') {
    return t('cleanupArchiveModalDesc', '将永久删除归档表中 {{days}} 天前的平台审计日志。此操作不可恢复。', { days: retentionDays })
  }
  return t('confirmLogMaintenance', '请确认执行日志维护操作。')
}

function getExportTabLabel(t: TFunction<'platform'>, activeTab: AuditTabKey, archiveType: 'all' | 'access' | 'operation' | 'login') {
  if (activeTab === 'archive') {
    if (archiveType === 'access') return t('archivedAccessLog', '归档访问日志')
    if (archiveType === 'operation') return t('archivedOperationAudit', '归档操作审计')
    if (archiveType === 'login') return t('archivedLoginLog', '归档登录日志')
    return t('archivedLogs', '归档日志')
  }
  if (activeTab === 'access') return t('accessLog', '访问日志')
  if (activeTab === 'operation') return t('operationAudit', '操作审计')
  if (activeTab === 'login') return t('loginLog', '登录日志')
  return t('platformAudit', '平台审计')
}

function downloadBlob(filename: string, blob: Blob) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', filename)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
