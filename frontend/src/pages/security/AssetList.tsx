// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useState } from 'react'
import {
  Table, Card, Button, Modal, Form, Input, Select, Tag, Space, message,
  Popconfirm, Typography, Row, Col, Statistic, Descriptions, Badge, Pagination,
  Tabs, Progress, Empty, List
} from 'antd'
import {
  PlusOutlined, DeleteOutlined, EditOutlined, EyeOutlined, ReloadOutlined,
  DesktopOutlined, CloudOutlined, GlobalOutlined, DatabaseOutlined, QuestionCircleOutlined,
  BugOutlined, RadarChartOutlined
} from '@ant-design/icons'
import { securityAPI, Asset, AssetStats, CreateAssetRequest, AssetVulnCount, SecurityScanTask, SecurityVulnerability, PaginatedResponse } from '../../api/security'
import { canEdit } from '../../utils/menuAccess'
import { getDateLocale, formatDateTime } from '../../utils/dateFormat'
import { useNavigate } from 'react-router-dom'
import TaskDetail from './TaskDetail'
import { useTranslation } from 'react-i18next'

const { Title } = Typography
const { Option } = Select

function parseCompositeBanner(banner?: string) {
  const raw = (banner || '').trim()
  if (!raw) {
    return {
      frontend: '',
      upstream: '',
    }
  }

  const parts = raw.split('| upstream=')
  return {
    frontend: parts[0]?.trim() || '',
    upstream: parts[1]?.trim() || '',
  }
}

export default function AssetList() {
  const { t } = useTranslation('security')
  const navigate = useNavigate()
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [stats, setStats] = useState<AssetStats | null>(null)
  const [recentDiscoveryTasks, setRecentDiscoveryTasks] = useState<SecurityScanTask[]>([])
  const [recentDiscoveryLoading, setRecentDiscoveryLoading] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [discoveryModalOpen, setDiscoveryModalOpen] = useState(false)
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [detailModalOpen, setDetailModalOpen] = useState(false)
  const [selectedDiscoveryTask, setSelectedDiscoveryTask] = useState<SecurityScanTask | null>(null)
  const [selectedAsset, setSelectedAsset] = useState<Asset | null>(null)
  const [selectedAssetVulnCount, setSelectedAssetVulnCount] = useState<AssetVulnCount | null>(null)
  const [selectedAssetVulns, setSelectedAssetVulns] = useState<SecurityVulnerability[]>([])
  const [vulnsLoading, setVulnsLoading] = useState(false)
  const [discoverySubmitting, setDiscoverySubmitting] = useState(false)
  const [form] = Form.useForm()
  const [discoveryForm] = Form.useForm()
  const [editForm] = Form.useForm()

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  const [filters, setFilters] = useState({
    asset_type: '',
    status: '',
    importance: '',
    keyword: '',
  })

  const ASSET_TYPE_CONFIG: Record<string, { color: string; icon: React.ReactNode }> = {
    server: { color: 'blue', icon: <DesktopOutlined /> },
    network: { color: 'purple', icon: <CloudOutlined /> },
    web: { color: 'green', icon: <GlobalOutlined /> },
    database: { color: 'orange', icon: <DatabaseOutlined /> },
    other: { color: 'default', icon: <QuestionCircleOutlined /> },
  }

  const assetTypeLabels: Record<string, string> = {
    server: t('server', '服务器'),
    network: t('networkDevice', '网络设备'),
    web: t('webService', 'Web服务'),
    database: t('database', '数据库'),
    other: t('other', '其他'),
  }

  const STATUS_CONFIG: Record<string, { color: string; text: string }> = {
    online: { color: 'success', text: t('status.online', '在线') },
    offline: { color: 'error', text: t('status.offline', '离线') },
    unknown: { color: 'default', text: t('status.unknown', '未知') },
  }

  const IMPORTANCE_CONFIG: Record<string, { color: string; text: string }> = {
    critical: { color: 'red', text: t('severityLabel.critical', '严重') },
    high: { color: 'orange', text: t('severityLabel.high', '高') },
    medium: { color: 'yellow', text: t('severityLabel.medium', '中') },
    low: { color: 'green', text: t('severityLabel.low', '低') },
  }

  const DISCOVERY_TASK_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: t('status.pending', '等待中') },
    running: { color: 'processing', text: t('status.running', '运行中') },
    paused: { color: 'warning', text: t('status.paused', '已请求暂停') },
    cancelled: { color: 'default', text: t('status.cancelled', '已请求取消') },
    completed: { color: 'success', text: t('status.completed', '已完成') },
    failed: { color: 'error', text: t('status.failed', '失败') },
  }

  function getServiceStatusLabel(name: string) {
    if (!name.endsWith('?')) {
      return null
    }
    const normalized = name.slice(0, -1).toLowerCase()
    if (normalized === 'https') {
      return t('openButHandshakeFailed', '开放但握手失败')
    }
    return t('protocolNotConfirmed', '协议未确认')
  }

  function renderServiceWithStatus(name?: string) {
    if (!name) return '-'
    const unverified = name.endsWith('?')
    const display = unverified ? name.slice(0, -1) : name
    const statusLabel = getServiceStatusLabel(name)
    return (
      <Space size={4}>
        <span>{display}</span>
        {unverified && statusLabel && <Tag color="cyan">{statusLabel}</Tag>}
      </Space>
    )
  }

  const fetchAssets = async () => {
    setLoading(true)
    try {
      const params: Record<string, string | number> = { page, page_size: pageSize }
      if (filters.asset_type) params.asset_type = filters.asset_type
      if (filters.status) params.status = filters.status
      if (filters.importance) params.importance = filters.importance
      if (filters.keyword) params.keyword = filters.keyword

      const res = await securityAPI.getAssets(params)
      setAssets(res.data)
      setTotal(res.total)
    } catch (error) {
      message.error(t('assetListLoadFailed', '获取资产列表失败'))
    } finally {
      setLoading(false)
    }
  }

  const fetchStats = async () => {
    try {
      const res = await securityAPI.getAssetStats()
      setStats(res)
    } catch (error) {
      console.error('获取统计失败', error)
    }
  }

  const fetchRecentDiscoveryTasks = async () => {
    setRecentDiscoveryLoading(true)
    try {
      const res = await securityAPI.getTasks({ page: 1, page_size: 5, task_group: 'discovery' })
      setRecentDiscoveryTasks(res.data || [])
    } catch (error) {
      console.error('获取最近资产发现任务失败', error)
    } finally {
      setRecentDiscoveryLoading(false)
    }
  }

  useEffect(() => {
    fetchAssets()
  }, [page, pageSize, filters])

  useEffect(() => {
    fetchStats()
    fetchRecentDiscoveryTasks()
  }, [])

  const handleCreate = async (values: CreateAssetRequest) => {
    try {
      await securityAPI.createAsset(values)
      message.success(t('assetCreateSuccess', '资产创建成功'))
      setCreateModalOpen(false)
      form.resetFields()
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error(t('assetCreateFailed', '创建失败'))
    }
  }

  const handleEdit = async (values: CreateAssetRequest) => {
    if (!selectedAsset) return
    try {
      await securityAPI.updateAsset(selectedAsset.id, values)
      message.success(t('assetUpdateSuccess', '资产更新成功'))
      setEditModalOpen(false)
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error(t('assetUpdateFailed', '更新失败'))
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await securityAPI.deleteAsset(id)
      message.success(t('assetDeleteSuccess', '资产删除成功'))
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error(t('assetDeleteFailed', '删除失败'))
    }
  }

  const buildDefaultDiscoveryTaskName = () => {
    const timestamp = new Date().toLocaleString(getDateLocale(), {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).replace(/\//g, '-')
    return `${t('assetDiscoveryTaskName', '资产发现')}-${timestamp}`
  }

  const handleCreateDiscovery = async (values: { name?: string; target: string }) => {
    const normalizedTarget = String(values.target || '')
      .split(/[,\n]/)
      .map((item) => item.trim())
      .filter(Boolean)
      .join(',')

    if (!normalizedTarget) {
      discoveryForm.setFields([{ name: 'target', errors: [t('pleaseEnterAtLeastOneIP', '请输入至少一个目标 IP')] }])
      return
    }

    setDiscoverySubmitting(true)
    try {
      await securityAPI.createTask({
        name: String(values.name || '').trim() || buildDefaultDiscoveryTaskName(),
        target_type: 'ip_list',
        target: normalizedTarget,
        scan_type: 'port',
      })
      message.success(t('assetDiscoveryTaskCreated', '资产发现任务已创建，可在下方查看最近任务状态'))
      setDiscoveryModalOpen(false)
      discoveryForm.resetFields()
      fetchRecentDiscoveryTasks()
    } catch (error) {
      message.error(t('createAssetDiscoveryFailed', '创建资产发现任务失败'))
    } finally {
      setDiscoverySubmitting(false)
    }
  }

  const formatTargetCount = (target: string) =>
    target
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean).length

  const handleViewDetail = async (record: Asset) => {
    try {
      setDetailModalOpen(true)
      setVulnsLoading(true)

      const res = await securityAPI.getAsset(record.id)
      setSelectedAsset(res.asset)
      setSelectedAssetVulnCount(res.vuln_count)

      const vulnRes = await securityAPI.getVulnerabilities({ ip: record.ip })
      const vulnData = vulnRes as PaginatedResponse<SecurityVulnerability>
      setSelectedAssetVulns(vulnData.data || [])
    } catch (error) {
      message.error(t('assetDetailLoadFailed', '获取详情失败'))
    } finally {
      setVulnsLoading(false)
    }
  }

  const handleViewDiscoveryTask = (task: SecurityScanTask) => {
    setSelectedDiscoveryTask(task)
  }

  const handleEditClick = (record: Asset) => {
    setSelectedAsset(record)
    editForm.setFieldsValue(record)
    setEditModalOpen(true)
  }

  const selectedAssetBanner = parseCompositeBanner(selectedAsset?.banner)

  const columns = [
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 140,
    },
    {
      title: t('port', '端口'),
      dataIndex: 'port',
      key: 'port',
      width: 80,
      render: (port: number) => port || '-',
    },
    {
      title: t('service', '服务'),
      dataIndex: 'service_name',
      key: 'service_name',
      width: 120,
      render: (name: string) => {
        return renderServiceWithStatus(name)
      },
    },
    {
      title: t('version', '版本'),
      dataIndex: 'version',
      key: 'version',
      width: 100,
      render: (v: string) => v || '-',
    },
    {
      title: t('assetType', '类型'),
      dataIndex: 'asset_type',
      key: 'asset_type',
      width: 100,
      render: (type: string) => {
        const config = ASSET_TYPE_CONFIG[type] || ASSET_TYPE_CONFIG.other
        return <Tag color={config.color}>{config.icon} {assetTypeLabels[type] || type}</Tag>
      },
    },
    {
      title: t('assetStatus', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const config = STATUS_CONFIG[status] || STATUS_CONFIG.unknown
        return <Badge status={config.color as 'success' | 'error' | 'default'} text={config.text} />
      },
    },
    {
      title: t('importance', '重要性'),
      dataIndex: 'importance',
      key: 'importance',
      width: 80,
      render: (imp: string) => {
        if (!imp) return '-'
        const config = IMPORTANCE_CONFIG[imp]
        return <Tag color={config?.color}>{config?.text}</Tag>
      },
    },
    {
      title: t('owner', '负责人'),
      dataIndex: 'owner',
      key: 'owner',
      width: 100,
      render: (owner: string) => owner || '-',
    },
    {
      title: t('department', '部门'),
      dataIndex: 'department',
      key: 'department',
      width: 100,
      render: (dept: string) => dept || '-',
    },
    {
      title: t('lastSeen', '最近发现'),
      dataIndex: 'last_seen',
      key: 'last_seen',
      width: 170,
      render: (time: string) => time ? formatDateTime(time) : '-',
    },
    {
      title: t('action', '操作'),
      key: 'action',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, record: Asset) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleViewDetail(record)}>
            {t('detail', '详情')}
          </Button>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditClick(record)}>
            {t('editAsset', '编辑')}
          </Button>}
          {canEdit() && <Popconfirm title={t('confirmDeleteAsset', '确定删除此资产?')} onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              {t('assetDeleteSuccess', '删除')}
            </Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: '24px' }}>
      <Title level={3}>{t('securityAssets', '安全资产')}</Title>

      {stats && (
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={6}>
            <Card>
              <Statistic title={t('totalAssets', '资产总数')} value={stats.total} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('online', '在线')} value={stats.online} valueStyle={{ color: '#3f8600' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('offline', '离线')} value={stats.offline} valueStyle={{ color: '#cf1322' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('status.unknown', '未知')} value={stats.unknown} />
            </Card>
          </Col>
        </Row>
      )}

      <Card
        title={t('recentAssetDiscoveryTasks', '最近资产发现任务')}
        style={{ marginBottom: 24 }}
        extra={(
          <Button type="link" onClick={() => navigate('/security/tasks?task_group=discovery')}>
            {t('viewAllTasks', '查看全部任务')}
          </Button>
        )}
      >
        <List
          loading={recentDiscoveryLoading}
          dataSource={recentDiscoveryTasks}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('noAssetDiscoveryTask', '暂无资产发现任务')} /> }}
          renderItem={(task) => {
            const status = DISCOVERY_TASK_STATUS_CONFIG[task.status] || { color: 'default', text: task.status }
            const targetCount = formatTargetCount(task.target)
            return (
              <List.Item
                actions={[
                  <Button key="detail" type="link" size="small" onClick={() => handleViewDiscoveryTask(task)}>
                    {t('viewTaskDetail', '查看详情')}
                  </Button>,
                  <Button key="list" type="link" size="small" onClick={() => navigate('/security/tasks?task_group=discovery')}>
                    {t('taskList', '任务列表')}
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={(
                    <Space wrap>
                      <span>{task.name}</span>
                      <Tag color={status.color}>{status.text}</Tag>
                    </Space>
                  )}
                  description={(
                    <Space size="middle" wrap>
                      <span>{targetCount} {t('targetCount', '个目标')}</span>
                      <span>{t('progress', '进度')} {task.progress || 0}%</span>
                      <span>{task.created_at ? formatDateTime(task.created_at) : '-'}</span>
                      {task.message ? <span>{task.message}</span> : null}
                    </Space>
                  )}
                />
              </List.Item>
            )
          }}
        />
      </Card>

      <Card>
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col flex="auto">
            <Space wrap>
              <Input.Search
                placeholder={t('searchIPServiceBanner', '搜索 IP/服务/Banner')}
                style={{ width: 250 }}
                onSearch={(value) => setFilters({ ...filters, keyword: value })}
                allowClear
              />
              <Select
                placeholder={t('assetType', '资产类型')}
                style={{ width: 120 }}
                allowClear
                value={filters.asset_type || undefined}
                onChange={(value) => setFilters({ ...filters, asset_type: value || '' })}
              >
                <Option value="server">{t('server', '服务器')}</Option>
                <Option value="network">{t('networkDevice', '网络设备')}</Option>
                <Option value="web">{t('webService', 'Web服务')}</Option>
                <Option value="database">{t('database', '数据库')}</Option>
                <Option value="other">{t('other', '其他')}</Option>
              </Select>
              <Select
                placeholder={t('assetStatus', '状态')}
                style={{ width: 100 }}
                allowClear
                value={filters.status || undefined}
                onChange={(value) => setFilters({ ...filters, status: value || '' })}
              >
                <Option value="online">{t('status.online', '在线')}</Option>
                <Option value="offline">{t('status.offline', '离线')}</Option>
                <Option value="unknown">{t('status.unknown', '未知')}</Option>
              </Select>
              <Select
                placeholder={t('importance', '重要性')}
                style={{ width: 100 }}
                allowClear
                value={filters.importance || undefined}
                onChange={(value) => setFilters({ ...filters, importance: value || '' })}
              >
                <Option value="critical">{t('severityLabel.critical', '严重')}</Option>
                <Option value="high">{t('severityLabel.high', '高')}</Option>
                <Option value="medium">{t('severityLabel.medium', '中')}</Option>
                <Option value="low">{t('severityLabel.low', '低')}</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => { fetchAssets(); fetchRecentDiscoveryTasks() }}>{t('refresh', '刷新')}</Button>
              <Button icon={<RadarChartOutlined />} onClick={() => setDiscoveryModalOpen(true)}>
                {t('startDiscovery', '发起资产发现')}
              </Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
                {t('addAsset', '新增资产')}
              </Button>
            </Space>
          </Col>
        </Row>

        <Table
          columns={columns}
          dataSource={assets}
          loading={loading}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1400 }}
        />

        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={(p, ps) => { setPage(p); setPageSize(ps) }}
            showSizeChanger
            showTotal={(tCount) => t('total', '共 {{count}} 条', { count: tCount })}
          />
        </div>
      </Card>

      <Modal
        title={t('startDiscovery', '发起资产发现')}
        open={discoveryModalOpen}
        onCancel={() => { setDiscoveryModalOpen(false); discoveryForm.resetFields() }}
        footer={null}
        width={560}
      >
        <Form form={discoveryForm} layout="vertical" onFinish={handleCreateDiscovery}>
          <Form.Item label={t('usageDescription', '用途说明')}>
            <Card size="small" style={{ background: '#fafafa' }}>
              {t('discoveryDescription', '资产发现会对目标 IP 执行开放端口探测和服务识别，用于资产盘点，不进行漏洞检测。')}
            </Card>
          </Form.Item>

          <Form.Item
            name="name"
            label={t('taskName', '任务名称')}
            tooltip={t('taskNameTooltip', '可选，不填会自动生成任务名')}
          >
            <Input placeholder={t('taskNameExample', '例如：办公网段资产摸底（可留空自动生成）')} />
          </Form.Item>

          <Form.Item
            name="target"
            label={t('targetIP', '目标 IP')}
            rules={[{ required: true, message: t('targetIPRequired', '请输入目标 IP 地址') }]}
            extra={t('targetIPHint', '支持多个 IP，逗号或换行分隔')}
          >
            <Input.TextArea
              rows={5}
              placeholder="192.168.1.10&#10;192.168.1.11&#10;10.0.0.10,10.0.0.20"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setDiscoveryModalOpen(false); discoveryForm.resetFields() }}>
                {t('cancel', '取消')}
              </Button>
              <Button type="primary" htmlType="submit" loading={discoverySubmitting}>
                {t('createTask', '创建任务')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {selectedDiscoveryTask && (
        <TaskDetail
          task={selectedDiscoveryTask}
          onClose={() => setSelectedDiscoveryTask(null)}
          onRefresh={() => {
            fetchAssets()
            fetchStats()
            fetchRecentDiscoveryTasks()
          }}
        />
      )}

      <Modal
        title={t('addAsset', '新增资产')}
        open={createModalOpen}
        onCancel={() => { setCreateModalOpen(false); form.resetFields() }}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ip" label={t('ipAddress', 'IP地址')} rules={[{ required: true, message: t('ipAddressRequired', '请输入IP地址') }]}>
                <Input placeholder="192.168.1.1" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="port" label={t('port', '端口')}>
                <Input type="number" placeholder="80" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="protocol" label={t('protocol', '协议')}>
                <Select placeholder={t('selectProtocol', '选择协议')} allowClear>
                  <Option value="TCP">TCP</Option>
                  <Option value="UDP">UDP</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="service_name" label={t('serviceName', '服务名称')}>
                <Input placeholder="nginx" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="version" label={t('version', '版本')}>
                <Input placeholder="1.20.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="asset_type" label={t('assetType', '资产类型')}>
                <Select placeholder={t('selectAssetType', '选择类型')} allowClear>
                  <Option value="server">{t('server', '服务器')}</Option>
                  <Option value="network">{t('networkDevice', '网络设备')}</Option>
                  <Option value="web">{t('webService', 'Web服务')}</Option>
                  <Option value="database">{t('database', '数据库')}</Option>
                  <Option value="other">{t('other', '其他')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="status" label={t('assetStatus', '状态')}>
                <Select placeholder={t('selectStatus', '选择状态')} allowClear>
                  <Option value="online">{t('status.online', '在线')}</Option>
                  <Option value="offline">{t('status.offline', '离线')}</Option>
                  <Option value="unknown">{t('status.unknown', '未知')}</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="importance" label={t('importance', '重要性')}>
                <Select placeholder={t('selectImportance', '选择重要性')} allowClear>
                  <Option value="critical">{t('severityLabel.critical', '严重')}</Option>
                  <Option value="high">{t('severityLabel.high', '高')}</Option>
                  <Option value="medium">{t('severityLabel.medium', '中')}</Option>
                  <Option value="low">{t('severityLabel.low', '低')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="owner" label={t('owner', '负责人')}>
                <Input placeholder={t('ownerPlaceholder', '张三')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="department" label={t('department', '部门')}>
                <Input placeholder={t('departmentPlaceholder', '运维部')} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="asset_group" label={t('assetGroup', '资产分组')}>
            <Input placeholder={t('environment', '生产环境')} />
          </Form.Item>
          <Form.Item name="tags" label={t('tags', '标签')}>
            <Input placeholder={t('tagsPlaceholder', 'web,nginx,生产')} />
          </Form.Item>
          <Form.Item name="os_info" label={t('osInfo', '操作系统信息')}>
            <Input placeholder="Linux 5.4.0" />
          </Form.Item>
          <Form.Item name="banner" label={t('banner', 'Banner')}>
            <Input.TextArea rows={2} placeholder={t('serviceFingerprint', '服务指纹信息')} />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalOpen(false)}>{t('cancel', '取消')}</Button>
              <Button type="primary" htmlType="submit">{t('createAsset', '创建')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('editAsset', '编辑资产')}
        open={editModalOpen}
        onCancel={() => { setEditModalOpen(false); editForm.resetFields() }}
        footer={null}
        width={600}
      >
        <Form form={editForm} layout="vertical" onFinish={handleEdit}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ip" label={t('ipAddress', 'IP地址')} rules={[{ required: true, message: t('ipAddressRequired', '请输入IP地址') }]}>
                <Input placeholder="192.168.1.1" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="port" label={t('port', '端口')}>
                <Input type="number" placeholder="80" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="protocol" label={t('protocol', '协议')}>
                <Select placeholder={t('selectProtocol', '选择协议')} allowClear>
                  <Option value="TCP">TCP</Option>
                  <Option value="UDP">UDP</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="service_name" label={t('serviceName', '服务名称')}>
                <Input placeholder="nginx" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="version" label={t('version', '版本')}>
                <Input placeholder="1.20.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="asset_type" label={t('assetType', '资产类型')}>
                <Select placeholder={t('selectAssetType', '选择类型')} allowClear>
                  <Option value="server">{t('server', '服务器')}</Option>
                  <Option value="network">{t('networkDevice', '网络设备')}</Option>
                  <Option value="web">{t('webService', 'Web服务')}</Option>
                  <Option value="database">{t('database', '数据库')}</Option>
                  <Option value="other">{t('other', '其他')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="status" label={t('assetStatus', '状态')}>
                <Select placeholder={t('selectStatus', '选择状态')} allowClear>
                  <Option value="online">{t('status.online', '在线')}</Option>
                  <Option value="offline">{t('status.offline', '离线')}</Option>
                  <Option value="unknown">{t('status.unknown', '未知')}</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="importance" label={t('importance', '重要性')}>
                <Select placeholder={t('selectImportance', '选择重要性')} allowClear>
                  <Option value="critical">{t('severityLabel.critical', '严重')}</Option>
                  <Option value="high">{t('severityLabel.high', '高')}</Option>
                  <Option value="medium">{t('severityLabel.medium', '中')}</Option>
                  <Option value="low">{t('severityLabel.low', '低')}</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="owner" label={t('owner', '负责人')}>
                <Input placeholder={t('ownerPlaceholder', '张三')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="department" label={t('department', '部门')}>
                <Input placeholder={t('departmentPlaceholder', '运维部')} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="asset_group" label={t('assetGroup', '资产分组')}>
            <Input placeholder={t('environment', '生产环境')} />
          </Form.Item>
          <Form.Item name="tags" label={t('tags', '标签')}>
            <Input placeholder={t('tagsPlaceholder', 'web,nginx,生产')} />
          </Form.Item>
          <Form.Item name="os_info" label={t('osInfo', '操作系统信息')}>
            <Input placeholder="Linux 5.4.0" />
          </Form.Item>
          <Form.Item name="banner" label={t('banner', 'Banner')}>
            <Input.TextArea rows={2} placeholder={t('serviceFingerprint', '服务指纹信息')} />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditModalOpen(false)}>{t('cancel', '取消')}</Button>
              <Button type="primary" htmlType="submit">{t('save', '保存')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('assetDetail', '资产详情')}
        open={detailModalOpen}
        onCancel={() => { setDetailModalOpen(false); setSelectedAsset(null); setSelectedAssetVulns([]); setSelectedAssetVulnCount(null) }}
        footer={[
          <Button key="close" onClick={() => setDetailModalOpen(false)}>{t('close', '关闭')}</Button>
        ]}
        width={900}
      >
        {selectedAsset && (
          <Tabs
            defaultActiveKey="info"
            items={[
              {
                key: 'info',
                label: t('basicInfo', '基本信息'),
                icon: <DesktopOutlined />,
                children: (
                  <Descriptions column={2} bordered size="small">
                    <Descriptions.Item label={t('ipAddress', 'IP地址')}>{selectedAsset.ip}</Descriptions.Item>
                    <Descriptions.Item label={t('port', '端口')}>{selectedAsset.port || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('protocol', '协议')}>{selectedAsset.protocol || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('service', '服务')}>{renderServiceWithStatus(selectedAsset.service_name)}</Descriptions.Item>
                    <Descriptions.Item label={t('version', '版本')}>{selectedAsset.version || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('assetType', '资产类型')}>{assetTypeLabels[selectedAsset.asset_type] || selectedAsset.asset_type || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('assetStatus', '状态')}>
                      {(() => {
                        const config = STATUS_CONFIG[selectedAsset.status] || STATUS_CONFIG.unknown
                        return <Badge status={config.color as 'success' | 'error' | 'default'} text={config.text} />
                      })()}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('importance', '重要性')}>
                      {selectedAsset.importance ? (
                        (() => {
                          const config = IMPORTANCE_CONFIG[selectedAsset.importance]
                          return <Tag color={config?.color}>{config?.text}</Tag>
                        })()
                      ) : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('owner', '负责人')}>{selectedAsset.owner || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('department', '部门')}>{selectedAsset.department || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('assetGroup', '资产分组')}>{selectedAsset.asset_group || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('tags', '标签')}>{selectedAsset.tags || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('operatingSystem', '操作系统')} span={2}>{selectedAsset.os_info || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('frontendService', '前端服务')} span={2}>
                      {selectedAssetBanner.frontend || selectedAsset.banner || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('backendApp', '后端应用')} span={2}>
                      {selectedAssetBanner.upstream || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('banner', 'Banner')} span={2}>{selectedAsset.banner || '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('firstSeen', '首次发现')}>{selectedAsset.first_seen ? formatDateTime(selectedAsset.first_seen) : '-'}</Descriptions.Item>
                    <Descriptions.Item label={t('lastSeen', '最近发现')}>{selectedAsset.last_seen ? formatDateTime(selectedAsset.last_seen) : '-'}</Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'vulns',
                label: t('relatedVulnerabilities', '关联漏洞'),
                icon: <BugOutlined />,
                children: (
                  <div>
                    {selectedAssetVulnCount && (
                      <Row gutter={16} style={{ marginBottom: 16 }}>
                        <Col span={4}>
                          <Card size="small">
                            <Statistic
                              title={t('totalVulnerabilities', '漏洞总数')}
                              value={selectedAssetVulnCount.total}
                              valueStyle={{ color: selectedAssetVulnCount.total > 0 ? '#cf1322' : '#3f8600' }}
                              prefix={<BugOutlined />}
                            />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title={t('severity.critical', '严重')} value={selectedAssetVulnCount.critical} valueStyle={{ color: '#722ed1' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title={t('severityLabel.high', '高危')} value={selectedAssetVulnCount.high} valueStyle={{ color: '#cf1322' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title={t('severityLabel.medium', '中危')} value={selectedAssetVulnCount.medium} valueStyle={{ color: '#fa8c16' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title={t('severityLabel.low', '低危')} value={selectedAssetVulnCount.low} valueStyle={{ color: '#52c41a' }} />
                          </Card>
                        </Col>
                      </Row>
                    )}

                    {selectedAssetVulnCount && selectedAssetVulnCount.total > 0 && (
                      <Card size="small" title={t('riskDistribution', '风险分布')} style={{ marginBottom: 16 }}>
                        <Progress
                          percent={100}
                          success={{ percent: (selectedAssetVulnCount.low / selectedAssetVulnCount.total) * 100, strokeColor: '#52c41a' }}
                          strokeColor="#fa8c16"
                          trailColor="#cf1322"
                          format={() => `${t('severityLabel.critical', '严重')}/${t('severityLabel.high', '高危')}: ${selectedAssetVulnCount.critical + selectedAssetVulnCount.high} | ${t('severityLabel.medium', '中危')}: ${selectedAssetVulnCount.medium} | ${t('severityLabel.low', '低危')}: ${selectedAssetVulnCount.low}`}
                        />
                      </Card>
                    )}

                    <Table
                      dataSource={selectedAssetVulns}
                      loading={vulnsLoading}
                      rowKey="id"
                      size="small"
                      pagination={{ pageSize: 5 }}
                      locale={{ emptyText: <Empty description={t('noRelatedVulnerabilities', '暂无关联漏洞')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
                      columns={[
                        {
                          title: t('severityLevel', '严重程度'),
                          dataIndex: 'severity',
                          width: 100,
                          render: (severity: string) => {
                            const colorMap: Record<string, string> = {
                              critical: 'purple',
                              high: 'red',
                              medium: 'orange',
                              low: 'green',
                              info: 'blue',
                            }
                            const textMap: Record<string, string> = {
                              critical: t('severityLabel.critical', '严重'),
                              high: t('severityLabel.high', '高危'),
                              medium: t('severityLabel.medium', '中危'),
                              low: t('severityLabel.low', '低危'),
                              info: t('severityLabel.info', '信息'),
                            }
                            return <Tag color={colorMap[severity] || 'default'}>{textMap[severity] || severity}</Tag>
                          },
                        },
                        {
                          title: t('vulnName', '漏洞名称'),
                          dataIndex: 'title',
                          ellipsis: true,
                        },
                        {
                          title: t('cve', 'CVE'),
                          dataIndex: 'cve_id',
                          width: 140,
                          render: (cve: string) => cve || '-',
                        },
                        {
                          title: t('port', '端口'),
                          dataIndex: 'port',
                          width: 70,
                        },
                        {
                          title: t('assetStatus', '状态'),
                          dataIndex: 'status',
                          width: 90,
                          render: (status: string) => {
                            const statusMap: Record<string, { color: string; text: string }> = {
                              open: { color: 'processing', text: t('statusLabel.open', '待处理') },
                              acknowledged: { color: 'warning', text: t('statusLabel.acknowledged', '已确认') },
                              fixed: { color: 'success', text: t('statusLabel.fixed', '已修复') },
                              ignored: { color: 'default', text: t('statusLabel.ignored', '已忽略') },
                            }
                            const config = statusMap[status] || { color: 'default', text: status }
                            return <Tag color={config.color}>{config.text}</Tag>
                          },
                        },
                        {
                          title: t('discoveryTime', '发现时间'),
                          dataIndex: 'created_at',
                          width: 160,
                          render: (time: string) => time ? formatDateTime(time) : '-',
                        },
                      ]}
                    />
                  </div>
                ),
              },
            ]}
          />
        )}
      </Modal>
    </div>
  )
}