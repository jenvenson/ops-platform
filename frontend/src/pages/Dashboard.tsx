// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Button, Space, Empty, Tabs } from 'antd'
import {
  ProjectOutlined,
  CloudOutlined,
  DesktopOutlined,
  AppstoreOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { cmdbAPI } from '../api/cmdb'
import { deployAPI, DeployRecord, ArchiveRecord } from '../api/cmdb.js'
import { aggregatedHistoryAPI, AggregatedHistory } from '../api/aggregated-history'
import { alertAPI, AlertEvent } from '../api/alert'
import { platformEventsAPI } from '../api/platform-events'
import { securityAPI, SecurityScanTask, SecurityVulnerability } from '../api/security'
import { MENU_CHANGED_EVENT, hasMenuAccess, readAllowedPaths, readStoredMenus } from '../utils/menuAccess'
import { formatDateTime } from '../utils/dateFormat'

import '../styles/dashboard.css'

const MORE_ACTIVITY_TAB_STORAGE_KEY = 'dashboard_more_activity_tab'
const currentDateValue = new Date().toISOString().slice(0, 10)

export default function Dashboard() {
  const navigate = useNavigate()
  const { t } = useTranslation('dashboard')
  const { t: tc } = useTranslation('common')
  const [allowedPaths, setAllowedPaths] = useState<Set<string>>(() => readAllowedPaths())
  const [menusReady, setMenusReady] = useState(() => readStoredMenus().length > 0)
  const hasAccess = (path: string) => hasMenuAccess(path, allowedPaths)
  const [stats, setStats] = useState({
    projectCount: 0,
    applicationCount: 0,
    environmentCount: 0,
    serverCount: 0,
    onlineServerCount: 0,
    deployCount: 0,
    archiveCount: 0,
    aggregateCount: 0,
    envByType: {
      dev: 0,
      test: 0,
      prod: 0,
    },
  })
  const [recentDeploys, setRecentDeploys] = useState<DeployRecord[]>([])
  const [recentArchives, setRecentArchives] = useState<ArchiveRecord[]>([])
  const [recentAggregates, setRecentAggregates] = useState<AggregatedHistory[]>([])
  const [recentAlerts, setRecentAlerts] = useState<AlertEvent[]>([])
  const [recentScanTasks, setRecentScanTasks] = useState<SecurityScanTask[]>([])
  const [recentVulnerabilities, setRecentVulnerabilities] = useState<SecurityVulnerability[]>([])
  const [platformEventSummary, setPlatformEventSummary] = useState({
    todayTotal: 0,
    attention: 0,
    failed: 0,
    highRisk: 0,
  })
  const [loading, setLoading] = useState(true)
  const [activeMoreTab, setActiveMoreTab] = useState(() => localStorage.getItem(MORE_ACTIVITY_TAB_STORAGE_KEY) || '')
  const allowedPathSignature = Array.from(allowedPaths).sort().join('|')

  useEffect(() => {
    const syncMenus = () => {
      const menus = readStoredMenus()
      setAllowedPaths(readAllowedPaths())
      setMenusReady(menus.length > 0)
    }

    syncMenus()
    window.addEventListener('storage', syncMenus)
    window.addEventListener(MENU_CHANGED_EVENT, syncMenus)

    return () => {
      window.removeEventListener('storage', syncMenus)
      window.removeEventListener(MENU_CHANGED_EVENT, syncMenus)
    }
  }, [])

  useEffect(() => {
    if (!menusReady) {
      setLoading(true)
      return
    }

    const fetchData = async () => {
      const canViewProjects = hasAccess('/cmdb/projects')
      const canViewApplications = hasAccess('/cmdb/applications')
      const canViewEnvironments = hasAccess('/cmdb/environments')
      const canViewServers = hasAccess('/cmdb/servers')
      const canViewDeploys = hasAccess('/deploy/history')
      const canViewArchives = hasAccess('/deploy/archived')
      const canViewAggregates = hasAccess('/deploy/aggregated-history')
      const canViewAlerts = hasAccess('/alarm/events')
      const canViewPlatformEvents = hasAccess('/platform/events')
      const canViewScanTasks = hasAccess('/security/tasks')
      const canViewVulnerabilities = hasAccess('/security/vulnerabilities')

      try {
        const [
          projectsResp,
          appsResp,
          envsResp,
          serversResp,
          deploysResp,
          archivesResp,
          aggregatesResp,
          alertsResp,
          platformEventsResp,
          failedPlatformEventsResp,
          firingPlatformEventsResp,
          acknowledgedPlatformEventsResp,
          criticalPlatformEventsResp,
          highPlatformEventsResp,
          scanTasksResp,
          vulnerabilitiesResp,
        ] = await Promise.all([
          canViewProjects ? cmdbAPI.getProjects({ limit: 1000 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewApplications ? cmdbAPI.getApplications({ limit: 1000 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewEnvironments ? cmdbAPI.getEnvironments({ limit: 1000 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewServers ? cmdbAPI.getServers({ limit: 1000 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewDeploys ? deployAPI.getDeployRecords({ limit: 3 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewArchives ? deployAPI.getArchiveRecords({ limit: 5 }).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
          canViewAggregates ? aggregatedHistoryAPI.getHistories({ limit: 5 }).catch(() => ({ data: [], total: 0, page: 1, limit: 5 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 5 }),
          canViewAlerts ? alertAPI.events.list({ page: '1', page_size: '3' }).catch(() => ({ data: [], total: 0, page: 1, page_size: 3 })) : Promise.resolve({ data: [], total: 0, page: 1, page_size: 3 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 5, occurred_from: currentDateValue }).catch(() => ({ data: [], total: 0, page: 1, limit: 5 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 5 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 1, occurred_from: currentDateValue, status: 'failed' }).catch(() => ({ data: [], total: 0, page: 1, limit: 1 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 1 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 1, occurred_from: currentDateValue, status: 'firing' }).catch(() => ({ data: [], total: 0, page: 1, limit: 1 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 1 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 1, occurred_from: currentDateValue, status: 'acknowledged' }).catch(() => ({ data: [], total: 0, page: 1, limit: 1 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 1 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 1, occurred_from: currentDateValue, severity: 'critical' }).catch(() => ({ data: [], total: 0, page: 1, limit: 1 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 1 }),
          canViewPlatformEvents ? platformEventsAPI.getEvents({ page: 1, limit: 1, occurred_from: currentDateValue, severity: 'high' }).catch(() => ({ data: [], total: 0, page: 1, limit: 1 })) : Promise.resolve({ data: [], total: 0, page: 1, limit: 1 }),
          canViewScanTasks ? securityAPI.getTasks({ page: 1, page_size: 5 }).catch(() => ({ data: [], total: 0, page: 1, page_size: 5, total_pages: 0 })) : Promise.resolve({ data: [], total: 0, page: 1, page_size: 5, total_pages: 0 }),
          canViewVulnerabilities ? securityAPI.getVulnerabilities({ page: 1, page_size: 5 }).catch(() => ({ data: [], total: 0, page: 1, page_size: 5, total_pages: 0 })) : Promise.resolve({ data: [], total: 0, page: 1, page_size: 5, total_pages: 0 }),
        ])

        const environments = envsResp.data || []
        const servers = serversResp.data || []
        const onlineCount = servers.filter((s) => s.status === 'online').length
        const envByType = environments.reduce(
          (acc, env) => {
            if (env.type === 'dev' || env.type === 'test' || env.type === 'prod') {
              acc[env.type] += 1
            }
            return acc
          },
          { dev: 0, test: 0, prod: 0 }
        )

        setStats({
          projectCount: projectsResp.total || 0,
          applicationCount: appsResp.total || 0,
          environmentCount: envsResp.total || 0,
          serverCount: serversResp.total || 0,
          onlineServerCount: onlineCount,
          deployCount: deploysResp.total || 0,
          archiveCount: archivesResp.total || 0,
          aggregateCount: aggregatesResp.total || 0,
          envByType,
        })

        setRecentDeploys(deploysResp.data || [])
        setRecentArchives(archivesResp.data || [])
        setRecentAggregates(aggregatesResp.data || [])
        setRecentAlerts(alertsResp.data || [])
        setPlatformEventSummary({
          todayTotal: platformEventsResp.total || 0,
          attention: (failedPlatformEventsResp.total || 0) + (firingPlatformEventsResp.total || 0) + (acknowledgedPlatformEventsResp.total || 0),
          failed: failedPlatformEventsResp.total || 0,
          highRisk: (criticalPlatformEventsResp.total || 0) + (highPlatformEventsResp.total || 0),
        })
        setRecentScanTasks(scanTasksResp.data || [])
        setRecentVulnerabilities(vulnerabilitiesResp.data || [])
      } catch (error) {
        console.error('获取仪表盘数据失败:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [allowedPathSignature, menusReady])

  const statusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: t('statusPending', '待执行') },
    queued: { color: 'processing', text: t('statusQueued', '排队中') },
    running: { color: 'processing', text: t('statusDeploying', '部署中') },
    success: { color: 'success', text: t('statusSuccess', '成功') },
    failed: { color: 'error', text: t('statusFailed', '失败') },
  }

  const aggregateStatusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: t('statusPending', '待执行') },
    queued: { color: 'processing', text: t('statusQueued', '排队中') },
    running: { color: 'processing', text: t('statusArchiving', '归档中') },
    archiving: { color: 'processing', text: t('statusArchiving', '归档中') },
    success: { color: 'success', text: t('statusComplete', '完成') },
    completed: { color: 'success', text: t('statusComplete', '完成') },
    failed: { color: 'error', text: t('statusFailed', '失败') },
  }

  const alertSeverityMap: Record<string, { color: string; text: string }> = {
    critical: { color: 'red', text: t('severityCritical', '严重') },
    warning: { color: 'orange', text: t('severityWarning', '警告') },
    info: { color: 'blue', text: t('severityInfo', '提醒') },
  }

  const alertEventStatusMap: Record<string, { color: string; text: string }> = {
    firing: { color: 'error', text: t('alertFiring', '告警中') },
    acknowledged: { color: 'warning', text: t('alertAcknowledged', '已介入') },
    resolved: { color: 'success', text: t('alertResolved', '已恢复') },
    closed: { color: 'default', text: t('alertClosed', '已关闭') },
  }

  const vulnerabilitySeverityMap: Record<string, string> = {
    critical: 'red',
    high: 'volcano',
    medium: 'orange',
    low: 'blue',
    info: 'default',
  }

  const vulnerabilityStatusMap: Record<string, { color: string; text: string }> = {
    open: { color: 'red', text: t('vulnOpen', '待处理') },
    acknowledged: { color: 'orange', text: t('vulnAcknowledged', '已确认') },
    fixed: { color: 'green', text: t('vulnFixed', '已修复') },
    ignored: { color: 'default', text: t('vulnIgnored', '已忽略') },
  }

  const scanTaskStatusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: t('statusPending', '待执行') },
    running: { color: 'processing', text: t('statusScanning', '扫描中') },
    completed: { color: 'success', text: t('statusComplete', '已完成') },
    failed: { color: 'error', text: t('statusFailed', '失败') },
  }

  const onlineRate = stats.serverCount > 0
    ? Math.round((stats.onlineServerCount / stats.serverCount) * 100)
    : 0

  const statCardsData = [
    {
      key: 'projects',
      path: '/cmdb/projects',
      icon: <ProjectOutlined />,
      iconClass: 'blue',
      value: stats.projectCount,
      label: t('projects', '项目'),
      trend: null,
      footer: t('projectsDesc', '业务范围和归属'),
    },
    {
      key: 'applications',
      path: '/cmdb/applications',
      icon: <AppstoreOutlined />,
      iconClass: 'blue',
      value: stats.applicationCount,
      label: t('applications', '应用'),
      trend: null,
      footer: t('applicationsDesc', '流水线和发布配置'),
    },
    {
      key: 'environments',
      path: '/cmdb/environments',
      icon: <CloudOutlined />,
      iconClass: 'green',
      value: stats.environmentCount,
      label: t('environments', '环境'),
      trend: null,
      footer: (
        <Space size={[4, 4]} wrap>
          <Tag color="blue" className="dashboard-tag">{t('environmentsDev', '开发')} {stats.envByType.dev}</Tag>
          <Tag color="orange" className="dashboard-tag">{t('environmentsTest', '测试')} {stats.envByType.test}</Tag>
          <Tag color="red" className="dashboard-tag">{t('environmentsProd', '生产')} {stats.envByType.prod}</Tag>
        </Space>
      ),
    },
    {
      key: 'servers',
      path: '/cmdb/servers',
      icon: <DesktopOutlined />,
      iconClass: 'purple',
      value: stats.serverCount,
      label: t('servers', '主机'),
      trend: {
        value: onlineRate,
        suffix: '%',
        type: onlineRate >= 80 ? 'up' : onlineRate >= 50 ? 'down' : 'down',
      },
      footer: (
        <Space size={[4, 4]} wrap>
          <Tag color="green" className="dashboard-tag">{t('serversOnline', '在线')} {stats.onlineServerCount} {t('serversUnit', '台')}</Tag>
          {stats.serverCount - stats.onlineServerCount > 0 ? (
            <Tag color="red" className="dashboard-tag">{t('serversOffline', '离线')} {stats.serverCount - stats.onlineServerCount} {t('serversUnit', '台')}</Tag>
          ) : null}
        </Space>
      ),
    },
  ].filter((item) => hasAccess(item.path))


  const canViewPlatformEvents = hasAccess('/platform/events')

  const hasVisibleContent = statCardsData.length > 0
    || hasAccess('/deploy/history')
    || hasAccess('/alarm/events')
    || hasAccess('/deploy/archived')
    || hasAccess('/deploy/aggregated-history')
    || hasAccess('/security/tasks')
    || hasAccess('/security/vulnerabilities')
    || canViewPlatformEvents

  const deployColumns = [
    {
      title: t('colApp', '应用'),
      dataIndex: 'app_name',
      key: 'app_name',
      width: 120,
    },
    {
      title: t('colEnv', '环境'),
      dataIndex: 'env_name',
      key: 'env_name',
      width: 80,
      render: (name: string, record: DeployRecord) => (
        <Tag color={record.env_type === 'prod' ? 'red' : record.env_type === 'test' ? 'orange' : 'blue'}>
          {name || '-'}
        </Tag>
      ),
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const config = statusMap[status] || { color: 'default', text: status }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colTriggeredBy', '发起人'),
      dataIndex: 'triggered_by',
      key: 'triggered_by',
      width: 110,
      render: (value: string) => value || '-',
    },
    {
      title: t('colTime', '时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 140,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const archiveColumns = [
    {
      title: t('colApp', '应用'),
      dataIndex: 'app_name',
      key: 'app_name',
      width: 120,
    },
    {
      title: t('colEnv', '环境'),
      dataIndex: 'env_name',
      key: 'env_name',
      width: 80,
      render: (name: string, record: ArchiveRecord) => (
        <Tag color={record.env_type === 'prod' ? 'red' : record.env_type === 'test' ? 'orange' : 'blue'}>
          {name || '-'}
        </Tag>
      ),
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const config = statusMap[status] || { color: 'default', text: status }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colOperator', '操作人'),
      dataIndex: 'operator',
      key: 'operator',
      width: 110,
      render: (value: string) => value || '-',
    },
    {
      title: t('colArchiveTime', '归档时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 140,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const aggregateColumns = [
    {
      title: t('colProject', '项目'),
      dataIndex: 'project_name',
      key: 'project_name',
      width: 120,
      render: (value: string) => value || '-',
    },
    {
      title: t('colTag', '标签'),
      dataIndex: 'environment',
      key: 'environment',
      width: 100,
      render: (value: string) => <Tag color="blue">{value || '-'}</Tag>,
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const config = aggregateStatusMap[status] || { color: 'default', text: status }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colOperator', '操作人'),
      dataIndex: 'operator_name',
      key: 'operator_name',
      width: 110,
      render: (value: string, record: AggregatedHistory) => value || record.operator || '-',
    },
    {
      title: t('colTime', '时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 140,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const alertColumns = [
    {
      title: t('colSeverity', '级别'),
      dataIndex: 'severity',
      key: 'severity',
      width: 80,
      render: (severity: string) => {
        const config = alertSeverityMap[severity] || { color: 'default', text: severity || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colRule', '规则'),
      dataIndex: 'rule_name',
      key: 'rule_name',
      width: 170,
      ellipsis: true,
      render: (value: string) => value || '-',
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const config = alertEventStatusMap[status] || { color: 'default', text: status || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colTime', '时间'),
      dataIndex: 'fired_at',
      key: 'fired_at',
      width: 150,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const vulnerabilityColumns = [
    {
      title: t('colSeverity', '级别'),
      dataIndex: 'severity',
      key: 'severity',
      width: 80,
      render: (severity: string) => <Tag color={vulnerabilitySeverityMap[severity] || 'default'}>{severity || '-'}</Tag>,
    },
    {
      title: t('colVulnTitle', '漏洞标题'),
      dataIndex: 'title',
      key: 'title',
      width: 180,
      ellipsis: true,
      render: (value: string) => value || '-',
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const config = vulnerabilityStatusMap[status] || { color: 'default', text: status || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colTime', '时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const scanTaskColumns = [
    {
      title: t('colTaskName', '任务名称'),
      dataIndex: 'name',
      key: 'name',
      width: 170,
      ellipsis: true,
      render: (value: string) => value || '-',
    },
    {
      title: t('colType', '类型'),
      dataIndex: 'scan_type',
      key: 'scan_type',
      width: 90,
      render: (value: string) => {
        const labelMap: Record<string, string> = {
          port: t('scanTypePort', '端口'),
          'host-vuln': t('scanTypeHostVuln', '主机漏洞'),
          web: t('scanTypeWeb', 'Web'),
          all: t('scanTypeAll', '全量'),
        }
        return <Tag color="blue">{labelMap[value] || value || '-'}</Tag>
      },
    },
    {
      title: t('colStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const config = scanTaskStatusMap[status] || { color: 'default', text: status || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: t('colTime', '时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
  ]

  const moreActivityTabs = [
    hasAccess('/deploy/archived') ? {
      key: 'archives',
      label: t('tabArchives', '归档'),
      path: '/deploy/archived',
      children: (
        <Table
          columns={archiveColumns}
          dataSource={recentArchives}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 520 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noArchiveRecords', '还没有归档记录。')} /> }}
        />
      ),
    } : null,
    hasAccess('/deploy/aggregated-history') ? {
      key: 'aggregates',
      label: t('tabAggregates', '聚合'),
      path: '/deploy/aggregated-history',
      children: (
        <Table
          columns={aggregateColumns}
          dataSource={recentAggregates}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 560 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noAggregateRecords', '还没有聚合记录。')} /> }}
        />
      ),
    } : null,
    hasAccess('/security/tasks') ? {
      key: 'scanTasks',
      label: t('tabScanTasks', '扫描任务'),
      path: '/security/tasks',
      children: (
        <Table
          columns={scanTaskColumns}
          dataSource={recentScanTasks}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 520 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noScanTaskRecords', '还没有扫描任务记录。')} /> }}
        />
      ),
    } : null,
    hasAccess('/security/vulnerabilities') ? {
      key: 'vulnerabilities',
      label: t('tabVulnerabilities', '扫描漏洞'),
      path: '/security/vulnerabilities',
      children: (
        <Table
          columns={vulnerabilityColumns}
          dataSource={recentVulnerabilities}
          rowKey="id"
          pagination={false}
          size="small"
          scroll={{ x: 520 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noVulnerabilityRecords', '还没有扫描漏洞记录。')} /> }}
        />
      ),
    } : null,
  ].filter(Boolean) as Array<{ key: string; label: string; path: string; children: ReactNode }>

  useEffect(() => {
    if (moreActivityTabs.length === 0) {
      if (activeMoreTab !== '') {
        setActiveMoreTab('')
      }
      return
    }

    const exists = moreActivityTabs.some((item) => item.key === activeMoreTab)
    if (exists) {
      return
    }

    const fallbackKey = moreActivityTabs[0].key
    setActiveMoreTab(fallbackKey)
    localStorage.setItem(MORE_ACTIVITY_TAB_STORAGE_KEY, fallbackKey)
  }, [activeMoreTab, moreActivityTabs])

  return (
    <div className="dashboard-container">
      {!menusReady && (
        <Card loading={loading} style={{ minHeight: 240 }} />
      )}

      {menusReady && !hasVisibleContent && (
        <Card>
          <Empty
            description={t('noPermission', '当前账号暂未分配工作台可见模块，请联系管理员分配菜单权限。')}
          />
        </Card>
      )}

      {/* 统计卡片网格 */}
      {statCardsData.length > 0 && (
        <div className="dashboard-stats-grid">
          {statCardsData.map((item) => (
            <div
              key={item.key}
              className={`dashboard-stat-card ${item.iconClass}`}
              onClick={() => navigate(item.path)}
            >
              <div className="stat-header">
                <div className={`stat-icon ${item.iconClass}`}>
                  {item.icon}
                </div>
                {item.trend && (
                  <div className={`stat-trend ${item.trend.type}`}>
                    {item.trend.type === 'up' ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
                    {item.trend.value}{item.trend.suffix}
                  </div>
                )}
              </div>
              <div className="stat-value">{item.value.toLocaleString()}</div>
              <div className="stat-label">{item.label}</div>
              <div className="stat-footer">{item.footer}</div>
            </div>
          ))}
        </div>
      )}

      {/* 平台事件中心 */}
      {canViewPlatformEvents && (
        <div className="dashboard-events-card">
          <div className="card-header">
            <div>
              <div className="card-title">{t('platformEvents', '平台事件中心')}</div>
              <div className="card-subtitle">{t('platformEventsSubtitle', '只展示今日统计，详情在事件中心查看')}</div>
            </div>
            <Button type="link" size="small" onClick={() => navigate('/platform/events')} style={{ color: 'var(--primary)', fontWeight: 500, padding: '4px 0' }}>
              {tc('viewAll', '查看全部')}
            </Button>
          </div>
          <div className="dashboard-events-grid">
            <div className="dashboard-event-stat">
              <div className={`event-value ${platformEventSummary.todayTotal === 0 ? 'muted' : ''}`}>
                {platformEventSummary.todayTotal}
              </div>
              <div className="event-label">{t('todayEvents', '今日事件')}</div>
            </div>
            <div className="dashboard-event-stat">
              <div className={`event-value ${platformEventSummary.attention > 0 ? 'warning' : 'muted'}`}>
                {platformEventSummary.attention}
              </div>
              <div className="event-label">{t('pendingAttention', '待关注')}</div>
            </div>
            <div className="dashboard-event-stat">
              <div className={`event-value ${platformEventSummary.failed > 0 ? 'danger' : 'muted'}`}>
                {platformEventSummary.failed}
              </div>
              <div className="event-label">{t('failedEvents', '失败事件')}</div>
            </div>
            <div className="dashboard-event-stat">
              <div className={`event-value ${platformEventSummary.highRisk > 0 ? 'danger' : 'muted'}`}>
                {platformEventSummary.highRisk}
              </div>
              <div className="event-label">{t('highRisk', '高风险')}</div>
            </div>
          </div>
        </div>
      )}

      {/* 双列卡片 */}
      <div className="dashboard-dual-grid">
        {hasAccess('/deploy/history') && (
          <div className="dashboard-data-card">
            <div className="card-header">
              <span className="card-title">{t('recentDeploys', '最近部署')}</span>
              <Button type="link" size="small" onClick={() => navigate('/deploy/history')} style={{ color: 'var(--primary)', fontWeight: 500, padding: '4px 0' }}>
                {tc('viewAll', '查看全部')}
              </Button>
            </div>
            <div className="card-body">
              {recentDeploys.length > 0 ? (
                <Table
                  columns={deployColumns}
                  dataSource={recentDeploys}
                  rowKey="id"
                  pagination={false}
                  size="small"
                  scroll={{ x: 520 }}
                />
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={t('noDeployRecords', '还没有部署记录，可以从【应用发布】开始第一条变更。')}
                />
              )}
            </div>
          </div>
        )}

        {hasAccess('/alarm/events') && (
          <div className="dashboard-data-card">
            <div className="card-header">
              <span className="card-title">{t('recentAlerts', '最近告警')}</span>
              <Button type="link" size="small" onClick={() => navigate('/alarm/events')} style={{ color: 'var(--primary)', fontWeight: 500, padding: '4px 0' }}>
                {tc('viewAll', '查看全部')}
              </Button>
            </div>
            <div className="card-body">
              {recentAlerts.length > 0 ? (
                <Table
                  columns={alertColumns}
                  dataSource={recentAlerts}
                  rowKey="id"
                  pagination={false}
                  size="small"
                  scroll={{ x: 490 }}
                />
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={t('noAlertRecords', '还没有告警记录。')}
                />
              )}
            </div>
          </div>
        )}
      </div>

      {/* 更多动态标签页 */}
      {moreActivityTabs.length > 0 && (
        <div className="dashboard-more-card">
          <div className="card-header">
            <div>
              <div className="card-title">{t('moreActivity', '更多动态')}</div>
              <div className="card-subtitle">{t('moreActivitySubtitle', '次级模块改为分页签展示，减少首页并列信息量')}</div>
            </div>
          </div>
          <Tabs
            activeKey={activeMoreTab || moreActivityTabs[0]?.key}
            onChange={(key) => {
              setActiveMoreTab(key)
              localStorage.setItem(MORE_ACTIVITY_TAB_STORAGE_KEY, key)
            }}
            items={moreActivityTabs.map((item) => ({
              key: item.key,
              label: item.label,
              children: item.children,
            }))}
          />
        </div>
      )}
    </div>
  )
}
