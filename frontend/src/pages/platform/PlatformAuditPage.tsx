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
      message.error('加载平台审计数据失败')
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
        message.success(`归档完成：访问 ${result.access_affected} 条，操作 ${result.operation_affected} 条，登录 ${result.login_affected} 条`)
      } else if (retentionAction === 'cleanup-online') {
        message.success(`在线清理完成：访问 ${result.access_affected} 条，操作 ${result.operation_affected} 条，登录 ${result.login_affected} 条`)
      } else {
        message.success(`归档清理完成：访问 ${result.access_affected} 条，操作 ${result.operation_affected} 条，登录 ${result.login_affected} 条`)
      }
      setRetentionAction(null)
      await refreshCurrentView()
    } catch (error) {
      if (retentionAction === 'archive') {
        message.error(getErrorMessage(error, '归档日志失败'))
      } else if (retentionAction === 'cleanup-online') {
        message.error(getErrorMessage(error, '清理在线日志失败'))
      } else {
        message.error(getErrorMessage(error, '清理归档日志失败'))
      }
    } finally {
      setRetentionLoading(false)
    }
  }

  const handleExportLogs = () => {
    if (activeTab === 'overview') {
      message.info('总览页不支持直接导出，请切换到具体日志页后再导出。')
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
      message.success('日志导出成功')
    } catch (error) {
      message.error(getErrorMessage(error, '导出日志失败'))
    } finally {
      setExportLoading(false)
    }
  }

  const handleDeleteArchivedLog = async (record: PlatformArchivedLog) => {
    try {
      await auditAPI.deleteArchivedLog(record.archive_type, record.id)
      message.success('归档日志已删除')
      await refreshCurrentView()
    } catch {
      message.error('删除归档日志失败')
    }
  }

  const handleDeleteAccessLog = async (record: PlatformAccessLog) => {
    try {
      await auditAPI.deleteAccessLog(record.id)
      message.success('访问日志已删除')
      await refreshCurrentView()
    } catch {
      message.error('删除访问日志失败')
    }
  }

  const handleDeleteOperationLog = async (record: PlatformAuditLog) => {
    try {
      await auditAPI.deleteOperationLog(record.id)
      message.success('操作审计已删除')
      await refreshCurrentView()
    } catch {
      message.error('删除操作审计失败')
    }
  }

  const handleDeleteLoginLog = async (record: PlatformLoginLog) => {
    try {
      await auditAPI.deleteLoginLog(record.id)
      message.success('登录日志已删除')
      await refreshCurrentView()
    } catch {
      message.error('删除登录日志失败')
    }
  }

  const accessColumns: ColumnsType<PlatformAccessLog> = [
    { title: '访问时间', dataIndex: 'accessed_at', key: 'accessed_at', width: 176, render: formatDateTime },
    { title: '操作人', dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: '菜单', dataIndex: 'menu_title', key: 'menu_title', width: 180, render: (value?: string) => value || '-' },
    { title: '请求路径', dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: renderStatusTag,
    },
    { title: '请求IP', dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: '耗时', dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: '操作',
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapAccessDetail(record))}>详情</Button>
          {canEdit() && (
            <Popconfirm title="确定删除这条访问日志？" onConfirm={() => void handleDeleteAccessLog(record)}>
              <Button type="link" danger>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const operationColumns: ColumnsType<PlatformAuditLog> = [
    { title: '操作时间', dataIndex: 'operated_at', key: 'operated_at', width: 176, render: formatDateTime },
    { title: '操作人', dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: '模块', dataIndex: 'module', key: 'module', width: 120, render: (value?: string) => value || '-' },
    { title: '动作', dataIndex: 'action_label', key: 'action_label', width: 120, render: (value?: string) => <Tag color="blue">{value || '操作'}</Tag> },
    { title: '资源名称', dataIndex: 'resource_name', key: 'resource_name', ellipsis: true, render: (value?: string) => value || '-' },
    {
      title: '状态',
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: renderStatusTag,
    },
    { title: '请求IP', dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: '耗时', dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: '操作',
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapOperationDetail(record))}>详情</Button>
          {canEdit() && (
            <Popconfirm title="确定删除这条操作审计？" onConfirm={() => void handleDeleteOperationLog(record)}>
              <Button type="link" danger>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const loginColumns: ColumnsType<PlatformLoginLog> = [
    { title: '登录时间', dataIndex: 'logged_in_at', key: 'logged_in_at', width: 176, render: formatDateTime },
    { title: '账号', dataIndex: 'username', key: 'username', width: 120 },
    { title: '角色', dataIndex: 'role', key: 'role', width: 120, render: (value?: string) => value || '-' },
    { title: '登录类型', dataIndex: 'login_type', key: 'login_type', width: 120, render: (value?: string) => value || 'password' },
    { title: '请求路径', dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'operation_status',
      key: 'operation_status',
      width: 92,
      render: renderStatusTag,
    },
    { title: '请求IP', dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: '耗时', dataIndex: 'duration_ms', key: 'duration_ms', width: 96, render: (value: number) => `${value}ms` },
    {
      title: '操作',
      key: 'op',
      width: 124,
      render: (_: unknown, record) => (
        <Space size={0}>
          <Button type="link" onClick={() => setDetail(mapLoginDetail(record))}>详情</Button>
          {canEdit() && (
            <Popconfirm title="确定删除这条登录日志？" onConfirm={() => void handleDeleteLoginLog(record)}>
              <Button type="link" danger>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const archivedColumns: ColumnsType<PlatformArchivedLog> = [
    { title: '归档时间', dataIndex: 'archived_at', key: 'archived_at', width: 176, render: formatDateTime },
    { title: '原始时间', dataIndex: 'occurred_at', key: 'occurred_at', width: 176, render: formatDateTime },
    { title: '类型', dataIndex: 'archive_type', key: 'archive_type', width: 100, render: renderArchiveTypeTag },
    { title: '操作人', dataIndex: 'username', key: 'username', width: 140, render: (_: string, record) => displayOperator(record.real_name, record.username) },
    { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
    { title: '请求路径', dataIndex: 'request_path', key: 'request_path', ellipsis: true },
    { title: '请求IP', dataIndex: 'request_ip', key: 'request_ip', width: 132 },
    { title: '状态', dataIndex: 'operation_status', key: 'operation_status', width: 92, render: renderStatusTag },
    {
      title: '操作',
      key: 'op',
      width: 96,
      render: (_: unknown, record) => (
        canEdit() ? (
          <Popconfirm title="确定删除这条归档日志？" onConfirm={() => void handleDeleteArchivedLog(record)}>
            <Button type="link" danger>删除</Button>
          </Popconfirm>
        ) : '-'
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, flexWrap: 'wrap' }}>
          <div>
            <Title level={4} style={{ margin: 0 }}>平台审计</Title>
            <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
              一个入口统一查看访问日志、操作审计与登录日志，符合当前平台的后台结构和使用习惯。
            </Paragraph>
          </div>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void loadData(activeTab)}>刷新</Button>
          </Space>
        </div>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title="总审计记录" value={stats.total} prefix={<FileSearchOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title="访问日志" value={stats.access} prefix={<FileSearchOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title="操作审计" value={stats.operation} prefix={<SafetyCertificateOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title="登录日志" value={stats.login} prefix={<LoginOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small"><Statistic title="归档日志" value={archiveStats.total} prefix={<InboxOutlined />} /></Card>
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
              label: '总览',
              children: (
                <AuditOverview
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
              label: '访问日志',
              children: (
                <AuditTablePanel
                  tab="access"
                  loading={loading}
                  title="访问日志"
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
              label: '操作审计',
              children: (
                <AuditTablePanel
                  tab="operation"
                  loading={loading}
                  title="操作审计"
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
              label: '登录日志',
              children: (
                <AuditTablePanel
                  tab="login"
                  loading={loading}
                  title="登录日志"
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
              label: '归档日志',
              children: (
                <div style={{ padding: '0 16px 16px' }}>
                  <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
                    <Col xs={24} lg={8}>
                      <Card size="small">
                        <Statistic title="归档总量" value={archiveStats.total} />
                        <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                          最近归档：{archiveStats.latest_archived ? formatDateTime(archiveStats.latest_archived) : '-'}
                        </Paragraph>
                      </Card>
                    </Col>
                    <Col xs={24} lg={16}>
                      <Card size="small">
                        <Row gutter={[16, 16]}>
                          <Col span={8}><Statistic title="访问归档" value={archiveStats.access_total} /></Col>
                          <Col span={8}><Statistic title="操作归档" value={archiveStats.operation_total} /></Col>
                          <Col span={8}><Statistic title="登录归档" value={archiveStats.login_total} /></Col>
                        </Row>
                      </Card>
                    </Col>
                  </Row>
                  <Card
                    size="small"
                    title="归档日志"
                    extra={(
                      <Space>
                        <Select
                          value={retentionDays}
                          style={{ width: 150 }}
                          onChange={setRetentionDays}
                          options={[
                            { value: 7, label: '保留 1 周' },
                            { value: 30, label: '保留 1 个月' },
                            { value: 180, label: '保留 6 个月' },
                          ]}
                        />
                        <Button icon={<InboxOutlined />} loading={retentionLoading} onClick={handleArchiveLogs}>归档旧日志</Button>
                        <Button danger loading={retentionLoading} onClick={handleCleanupOnlineLogs}>清理在线日志</Button>
                        <Button danger icon={<DeleteOutlined />} loading={retentionLoading} onClick={handleCleanupLogs}>清理归档日志</Button>
                        <Divider type="vertical" />
                        <Button type="primary" icon={<DownloadOutlined />} onClick={handleExportLogs}>导出日志</Button>
                        <Button onClick={() => resetFilters('archive', setFilters, setDetail, () => void loadData('archive', createEmptyFilters()))}>重置</Button>
                        <Button type="primary" onClick={() => void loadData('archive')}>查询</Button>
                        <Select
                          value={archiveType}
                          style={{ width: 140 }}
                          onChange={setArchiveType}
                          options={[
                            { value: 'all', label: '全部类型' },
                            { value: 'access', label: '访问日志' },
                            { value: 'operation', label: '操作审计' },
                            { value: 'login', label: '登录日志' },
                          ]}
                        />
                        <Button onClick={() => void loadData('archive')}>刷新</Button>
                      </Space>
                    )}
                  >
                    <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Input
                          placeholder="操作人"
                          value={filters.archive.username}
                          onChange={(event) => updateFilters('archive', { ...filters.archive, username: event.target.value }, setFilters)}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Select
                          placeholder="状态"
                          style={{ width: '100%' }}
                          allowClear
                          value={filters.archive.status}
                          onChange={(value) => updateFilters('archive', { ...filters.archive, status: value }, setFilters)}
                          options={[
                            { value: 'success', label: '成功' },
                            { value: 'failed', label: '失败' },
                          ]}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={4}>
                        <Input
                          placeholder="请求IP"
                          value={filters.archive.requestIP}
                          onChange={(event) => updateFilters('archive', { ...filters.archive, requestIP: event.target.value }, setFilters)}
                        />
                      </Col>
                      <Col xs={24} sm={12} md={8} xl={6}>
                        <Input
                          placeholder="请求路径"
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
                      pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
                      scroll={{ x: 1200 }}
                    />
                  </Card>
                </div>
              ),
            },
          ]}
        />
      </Card>

      <Drawer title="审计详情" placement="right" width={520} open={!!detail} onClose={() => setDetail(null)}>
        {detail && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="操作人">{detail.operator}</Descriptions.Item>
              <Descriptions.Item label="角色">{detail.role}</Descriptions.Item>
              <Descriptions.Item label="模块">{detail.module}</Descriptions.Item>
              <Descriptions.Item label="动作">{detail.action}</Descriptions.Item>
              <Descriptions.Item label="状态">{renderStatusTag(detail.status)}</Descriptions.Item>
              <Descriptions.Item label="请求IP">{detail.requestIP}</Descriptions.Item>
              <Descriptions.Item label="请求路径" span={2}>{detail.requestPath}</Descriptions.Item>
              <Descriptions.Item label="请求方法">{detail.requestMethod}</Descriptions.Item>
              <Descriptions.Item label="操作时间">{detail.time}</Descriptions.Item>
              <Descriptions.Item label="耗时" span={2}>{detail.durationMS}ms</Descriptions.Item>
            </Descriptions>
            <Card size="small" title="摘要">
              <Paragraph style={{ marginBottom: 0 }}>{detail.summary}</Paragraph>
            </Card>
            <Card size="small" title="请求参数">
              <pre style={preStyle}>{detail.params || '-'}</pre>
            </Card>
            <Card size="small" title="变更前">
              <pre style={preStyle}>{detail.beforeData || '-'}</pre>
            </Card>
            <Card size="small" title="变更后">
              <pre style={preStyle}>{detail.afterData || '-'}</pre>
            </Card>
            <Card size="small" title="错误信息">
              <Paragraph style={{ marginBottom: 0 }}>{detail.errorMessage || '-'}</Paragraph>
            </Card>
          </Space>
        )}
      </Drawer>
      <Modal
        title={getRetentionModalTitle(retentionAction)}
        open={!!retentionAction}
        onCancel={() => setRetentionAction(null)}
        onOk={() => void handleConfirmRetentionAction()}
        confirmLoading={retentionLoading}
        okText={retentionAction === 'archive' ? '确认归档' : '确认清理'}
        okButtonProps={retentionAction === 'archive' ? undefined : { danger: true }}
        cancelText="取消"
      >
        <Paragraph style={{ marginBottom: 0 }}>
          {getRetentionModalDescription(retentionAction, retentionDays)}
        </Paragraph>
      </Modal>
      <Modal
        title="导出日志"
        open={exportOpen}
        onCancel={() => setExportOpen(false)}
        onOk={() => void handleConfirmExport()}
        okText="确认导出"
        cancelText="取消"
        confirmLoading={exportLoading}
      >
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Paragraph style={{ marginBottom: 0 }}>
            当前将导出“{getExportTabLabel(activeTab, archiveType)}”在当前筛选条件下的全部结果。
          </Paragraph>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            导出格式为 CSV，由后端直接生成，适合留存和二次分析。
          </Paragraph>
        </Space>
      </Modal>
    </div>
  )
}

function AuditOverview({
  loading,
  stats,
  failedTotals,
  archiveStats,
  accessLogs,
  operationLogs,
  loginLogs,
}: {
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
            <Card title="审计态势" size="small">
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <Card size="small" bordered={false} style={{ background: '#fafafa' }}>
                    <Statistic title="累计审计记录" value={stats.total} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      统一汇总访问日志、操作审计和登录日志。
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={12}>
                  <Card size="small" bordered={false} style={{ background: stats.failed > 0 ? '#fff7e6' : '#f6ffed' }}>
                    <Statistic title="失败记录" value={stats.failed} valueStyle={{ color: stats.failed > 0 ? '#d46b08' : '#389e0d' }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      {stats.failed > 0 ? '建议优先查看登录失败和执行失败操作。' : '当前未发现失败审计记录。'}
                    </Paragraph>
                  </Card>
                </Col>
              </Row>
              <Row gutter={[16, 16]} style={{ marginTop: 4 }}>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title="访问日志" value={stats.access} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      失败 {failedTotals.access} 条
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title="操作审计" value={stats.operation} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      失败 {failedTotals.operation} 条
                    </Paragraph>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small" bordered={false}>
                    <Statistic title="登录日志" value={stats.login} valueStyle={{ fontSize: 22 }} />
                    <Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                      失败 {failedTotals.login} 条
                    </Paragraph>
                  </Card>
                </Col>
              </Row>
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title="审计覆盖范围" size="small">
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <AuditBullet
                  title="登录链路"
                  description="记录后台登录请求、账号、IP、结果和耗时。"
                />
                <AuditBullet
                  title="访问链路"
                  description="记录页面与关键 GET 接口访问，适合追溯用户看过什么。"
                />
                <AuditBullet
                  title="操作链路"
                  description="记录非 GET 关键操作，适合排查谁在什么时候改了什么。"
                />
                <AuditBullet
                  title="归档链路"
                  description={`已归档 ${archiveStats.total} 条历史日志，支持按保留周期做归档与清理。`}
                />
              </Space>
            </Card>
          </Col>

          <Col xs={24} xl={8}>
            <Card title="最近操作" size="small">
              <AuditRecentItem
                title={recentOperation?.action_label || recentOperation?.action || '暂无数据'}
                subtitle={recentOperation ? `${recentOperation.module || '未分类模块'} / ${recentOperation.resource_name || '未命名资源'}` : '当前还没有操作审计记录'}
                extra={recentOperation ? formatDateTime(recentOperation.operated_at) : '-'}
                status={recentOperation?.operation_status}
              />
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title="最近登录" size="small">
              <AuditRecentItem
                title={displayOperator(recentLogin?.real_name, recentLogin?.username) || '暂无数据'}
                subtitle={recentLogin ? `${recentLogin.request_ip} / ${recentLogin.login_type || 'password'}` : '当前还没有登录日志'}
                extra={recentLogin ? formatDateTime(recentLogin.logged_in_at) : '-'}
                status={recentLogin?.operation_status}
              />
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card title="最近访问" size="small">
              <AuditRecentItem
                title={recentAccess?.menu_title || recentAccess?.request_path || '暂无数据'}
                subtitle={recentAccess ? `${displayOperator(recentAccess.real_name, recentAccess.username)} / ${recentAccess.request_ip}` : '当前还没有访问日志'}
                extra={recentAccess ? formatDateTime(recentAccess.accessed_at) : '-'}
                status={recentAccess?.operation_status}
              />
            </Card>
          </Col>
          <Col xs={24}>
            <Card title="最近归档维护" size="small">
              <Row gutter={[16, 16]}>
                {recentArchiveOps.length > 0 ? recentArchiveOps.map((item) => (
                  <Col xs={24} md={12} key={item.id}>
                    <AuditRecentItem
                      title={item.action_label || item.action || '日志维护'}
                      subtitle={`${displayOperator(item.real_name, item.username)} / ${item.change_summary || item.request_path}`}
                      extra={formatDateTime(item.operated_at)}
                      status={item.operation_status}
                    />
                  </Col>
                )) : (
                  <Col xs={24}>
                    <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                      当前还没有归档或清理操作记录。
                    </Paragraph>
                  </Col>
                )}
              </Row>
            </Card>
          </Col>

          <Col xs={24}>
            <Card title="审计使用建议" size="small">
              <Row gutter={[16, 16]}>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>先看登录日志</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    适合判断是谁、从哪个 IP 登录，以及是否存在异常登录失败。
                  </Paragraph>
                </Col>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>再看访问日志</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    适合确认登录后访问了哪些页面和关键查询接口。
                  </Paragraph>
                </Col>
                <Col xs={24} md={8}>
                  <Paragraph style={{ marginBottom: 8 }}><strong>最后看操作审计</strong></Paragraph>
                  <Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    适合还原最终执行了哪些变更、测试或处置动作。
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

function AuditBullet({ title, description }: { title: string; description: string }) {
  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
      <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#1677ff', marginTop: 8, flexShrink: 0 }} />
      <div>
        <Paragraph style={{ marginBottom: 4, fontWeight: 600 }}>{title}</Paragraph>
        <Paragraph type="secondary" style={{ marginBottom: 0 }}>{description}</Paragraph>
      </div>
    </div>
  )
}

function AuditRecentItem({
  title,
  subtitle,
  extra,
  status,
}: {
  title: string
  subtitle: string
  extra: string
  status?: 'success' | 'failed'
}) {
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'flex-start' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Paragraph ellipsis={{ rows: 2 }} style={{ marginBottom: 4, fontWeight: 600 }}>
            {title}
          </Paragraph>
          <Paragraph ellipsis={{ rows: 2 }} type="secondary" style={{ marginBottom: 0 }}>
            {subtitle}
          </Paragraph>
        </div>
        {status ? renderStatusTag(status) : null}
      </div>
      <Paragraph type="secondary" style={{ marginBottom: 0 }}>
        时间：{extra}
      </Paragraph>
    </Space>
  )
}

function AuditTablePanel<T extends { id: number }>({
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
            <Button onClick={onReset}>重置</Button>
            <Button type="primary" onClick={onSearch}>查询</Button>
          </Space>
        )}
      >
        <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={12} md={8} xl={4}>
            <Input
              placeholder="操作人"
              value={filters.username}
              onChange={(event) => onFiltersChange({ ...filters, username: event.target.value })}
            />
          </Col>
          {tab === 'operation' && (
            <>
              <Col xs={24} sm={12} md={8} xl={4}>
                <Select
                  placeholder="模块"
                  style={{ width: '100%' }}
                  allowClear
                  value={filters.module}
                  onChange={(value) => onFiltersChange({ ...filters, module: value })}
                  options={[
                    { value: '安全中心', label: '安全中心' },
                    { value: '系统管理', label: '系统管理' },
                    { value: '告警中心', label: '告警中心' },
                    { value: '平台审计', label: '平台审计' },
                  ]}
                />
              </Col>
              <Col xs={24} sm={12} md={8} xl={4}>
                <Select
                  placeholder="动作"
                  style={{ width: '100%' }}
                  allowClear
                  value={filters.action}
                  onChange={(value) => onFiltersChange({ ...filters, action: value })}
                  options={[
                    { value: 'execute', label: '执行操作' },
                    { value: 'update', label: '更新配置' },
                    { value: 'delete', label: '删除' },
                    { value: 'create', label: '新增' },
                  ]}
                />
              </Col>
            </>
          )}
          <Col xs={24} sm={12} md={8} xl={4}>
            <Select
              placeholder="状态"
              style={{ width: '100%' }}
              allowClear
              value={filters.status}
              onChange={(value) => onFiltersChange({ ...filters, status: value })}
              options={[
                { value: 'success', label: '成功' },
                { value: 'failed', label: '失败' },
              ]}
            />
          </Col>
          <Col xs={24} sm={12} md={8} xl={4}>
            <Input
              placeholder="请求IP"
              value={filters.requestIP}
              onChange={(event) => onFiltersChange({ ...filters, requestIP: event.target.value })}
            />
          </Col>
          <Col xs={24} sm={12} md={8} xl={tab === 'operation' ? 8 : 4}>
            <Input
              placeholder="请求路径"
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
            pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>
    </div>
  )
}

function renderStatusTag(value: 'success' | 'failed') {
  return <Tag color={value === 'success' ? 'green' : 'red'}>{value === 'success' ? '成功' : '失败'}</Tag>
}

function renderArchiveTypeTag(value: 'access' | 'operation' | 'login') {
  if (value === 'access') return <Tag color="blue">访问日志</Tag>
  if (value === 'operation') return <Tag color="purple">操作审计</Tag>
  return <Tag color="gold">登录日志</Tag>
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

function displayOperator(realName?: string, username?: string) {
  return realName || username || '-'
}

function mapAccessDetail(record: PlatformAccessLog): DetailRecord {
  return {
    title: record.menu_title || '访问日志',
    operator: displayOperator(record.real_name, record.username),
    role: record.role || '-',
    module: record.menu_title || '访问日志',
    action: '访问页面',
    requestPath: record.request_path,
    requestMethod: record.request_method,
    requestIP: record.request_ip,
    status: record.operation_status,
    time: formatDateTime(record.accessed_at),
    durationMS: record.duration_ms,
    summary: `访问了 ${record.menu_title || record.request_path}。`,
    params: '{}',
    beforeData: '',
    afterData: '',
    errorMessage: record.error_message,
  }
}

function mapOperationDetail(record: PlatformAuditLog): DetailRecord {
  return {
    title: record.resource_name || '操作审计',
    operator: displayOperator(record.real_name, record.username),
    role: record.role || '-',
    module: record.module || '-',
    action: record.action_label || record.action || '操作',
    requestPath: record.request_path,
    requestMethod: record.request_method,
    requestIP: record.request_ip,
    status: record.operation_status,
    time: formatDateTime(record.operated_at),
    durationMS: record.duration_ms,
    summary: record.change_summary || '-',
    params: record.request_params_json || '{}',
    beforeData: record.before_data_json || '',
    afterData: record.after_data_json || '',
    errorMessage: record.error_message,
  }
}

function mapLoginDetail(record: PlatformLoginLog): DetailRecord {
  return {
    title: '登录日志',
    operator: displayOperator(record.real_name, record.username),
    role: record.role || '-',
    module: '认证中心',
    action: record.operation_status === 'success' ? '登录成功' : '登录失败',
    requestPath: record.request_path,
    requestMethod: record.request_method,
    requestIP: record.request_ip,
    status: record.operation_status,
    time: formatDateTime(record.logged_in_at),
    durationMS: record.duration_ms,
    summary: record.error_message ? `登录失败：${record.error_message}` : '使用账号密码登录后台平台。',
    params: '{}',
    beforeData: '',
    afterData: '',
    errorMessage: record.error_message,
  }
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

function getRetentionModalTitle(action: 'archive' | 'cleanup-online' | 'cleanup-archive' | null) {
  if (action === 'archive') return '归档旧日志'
  if (action === 'cleanup-online') return '清理在线日志'
  if (action === 'cleanup-archive') return '清理归档日志'
  return '日志维护'
}

function getRetentionModalDescription(action: 'archive' | 'cleanup-online' | 'cleanup-archive' | null, retentionDays: number) {
  if (action === 'archive') {
    return `将归档 ${retentionDays} 天前的平台审计日志，并从在线日志中移除。`
  }
  if (action === 'cleanup-online') {
    return `将永久删除在线表中 ${retentionDays} 天前的平台审计日志，不会归档到归档表。此操作不可恢复。`
  }
  if (action === 'cleanup-archive') {
    return `将永久删除归档表中 ${retentionDays} 天前的平台审计日志。此操作不可恢复。`
  }
  return '请确认执行日志维护操作。'
}

function getExportTabLabel(activeTab: AuditTabKey, archiveType: 'all' | 'access' | 'operation' | 'login') {
  if (activeTab === 'archive') {
    if (archiveType === 'access') return '归档访问日志'
    if (archiveType === 'operation') return '归档操作审计'
    if (archiveType === 'login') return '归档登录日志'
    return '归档日志'
  }
  if (activeTab === 'access') return '访问日志'
  if (activeTab === 'operation') return '操作审计'
  if (activeTab === 'login') return '登录日志'
  return '平台审计'
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
