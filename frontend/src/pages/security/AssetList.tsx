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
import { useNavigate } from 'react-router-dom'
import TaskDetail from './TaskDetail'

const { Title } = Typography
const { Option } = Select

// 资产类型配置
const ASSET_TYPE_CONFIG: Record<string, { color: string; icon: React.ReactNode }> = {
  server: { color: 'blue', icon: <DesktopOutlined /> },
  network: { color: 'purple', icon: <CloudOutlined /> },
  web: { color: 'green', icon: <GlobalOutlined /> },
  database: { color: 'orange', icon: <DatabaseOutlined /> },
  other: { color: 'default', icon: <QuestionCircleOutlined /> },
}

// 状态配置
const STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  online: { color: 'success', text: '在线' },
  offline: { color: 'error', text: '离线' },
  unknown: { color: 'default', text: '未知' },
}

// 重要性配置
const IMPORTANCE_CONFIG: Record<string, { color: string; text: string }> = {
  critical: { color: 'red', text: '严重' },
  high: { color: 'orange', text: '高' },
  medium: { color: 'yellow', text: '中' },
  low: { color: 'green', text: '低' },
}

const DISCOVERY_TASK_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  pending: { color: 'default', text: '等待中' },
  running: { color: 'processing', text: '运行中' },
  paused: { color: 'warning', text: '已请求暂停' },
  cancelled: { color: 'default', text: '已请求取消' },
  completed: { color: 'success', text: '已完成' },
  failed: { color: 'error', text: '失败' },
}

function getServiceStatusLabel(name: string) {
  if (!name.endsWith('?')) {
    return null
  }
  const normalized = name.slice(0, -1).toLowerCase()
  if (normalized === 'https') {
    return '开放但握手失败'
  }
  return '协议未确认'
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

  // 分页状态
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  // 筛选状态
  const [filters, setFilters] = useState({
    asset_type: '',
    status: '',
    importance: '',
    keyword: '',
  })

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
      message.error('获取资产列表失败')
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
      message.success('资产创建成功')
      setCreateModalOpen(false)
      form.resetFields()
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error('创建失败')
    }
  }

  const handleEdit = async (values: CreateAssetRequest) => {
    if (!selectedAsset) return
    try {
      await securityAPI.updateAsset(selectedAsset.id, values)
      message.success('资产更新成功')
      setEditModalOpen(false)
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error('更新失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await securityAPI.deleteAsset(id)
      message.success('资产删除成功')
      fetchAssets()
      fetchStats()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const buildDefaultDiscoveryTaskName = () => {
    const timestamp = new Date().toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).replace(/\//g, '-')
    return `资产发现-${timestamp}`
  }

  const handleCreateDiscovery = async (values: { name?: string; target: string }) => {
    const normalizedTarget = String(values.target || '')
      .split(/[,\n]/)
      .map((item) => item.trim())
      .filter(Boolean)
      .join(',')

    if (!normalizedTarget) {
      discoveryForm.setFields([{ name: 'target', errors: ['请输入至少一个目标 IP'] }])
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
      message.success('资产发现任务已创建，可在下方查看最近任务状态')
      setDiscoveryModalOpen(false)
      discoveryForm.resetFields()
      fetchRecentDiscoveryTasks()
    } catch (error) {
      message.error('创建资产发现任务失败')
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

      // 获取资产详情和漏洞统计
      const res = await securityAPI.getAsset(record.id)
      setSelectedAsset(res.asset)
      setSelectedAssetVulnCount(res.vuln_count)

      // 获取该 IP 的关联漏洞列表
      const vulnRes = await securityAPI.getVulnerabilities({ ip: record.ip })
      // 处理分页响应
      const vulnData = vulnRes as PaginatedResponse<SecurityVulnerability>
      setSelectedAssetVulns(vulnData.data || [])
    } catch (error) {
      message.error('获取详情失败')
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
      title: '端口',
      dataIndex: 'port',
      key: 'port',
      width: 80,
      render: (port: number) => port || '-',
    },
    {
      title: '服务',
      dataIndex: 'service_name',
      key: 'service_name',
      width: 120,
      render: (name: string) => {
        return renderServiceWithStatus(name)
      },
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      width: 100,
      render: (v: string) => v || '-',
    },
    {
      title: '类型',
      dataIndex: 'asset_type',
      key: 'asset_type',
      width: 100,
      render: (type: string) => {
        const config = ASSET_TYPE_CONFIG[type] || ASSET_TYPE_CONFIG.other
        return <Tag color={config.color}>{config.icon} {type}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const config = STATUS_CONFIG[status] || STATUS_CONFIG.unknown
        return <Badge status={config.color as 'success' | 'error' | 'default'} text={config.text} />
      },
    },
    {
      title: '重要性',
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
      title: '负责人',
      dataIndex: 'owner',
      key: 'owner',
      width: 100,
      render: (owner: string) => owner || '-',
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
      width: 100,
      render: (dept: string) => dept || '-',
    },
    {
      title: '最近发现',
      dataIndex: 'last_seen',
      key: 'last_seen',
      width: 170,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, record: Asset) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleViewDetail(record)}>
            详情
          </Button>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditClick(record)}>
            编辑
          </Button>}
          {canEdit() && <Popconfirm title="确定删除此资产?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: '24px' }}>
      <Title level={3}>安全资产</Title>

      {/* 统计卡片 */}
      {stats && (
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={6}>
            <Card>
              <Statistic title="资产总数" value={stats.total} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="在线" value={stats.online} valueStyle={{ color: '#3f8600' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="离线" value={stats.offline} valueStyle={{ color: '#cf1322' }} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="未知" value={stats.unknown} />
            </Card>
          </Col>
        </Row>
      )}

      <Card
        title="最近资产发现任务"
        style={{ marginBottom: 24 }}
        extra={(
          <Button type="link" onClick={() => navigate('/security/tasks?task_group=discovery')}>
            查看全部任务
          </Button>
        )}
      >
        <List
          loading={recentDiscoveryLoading}
          dataSource={recentDiscoveryTasks}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无资产发现任务" /> }}
          renderItem={(task) => {
            const status = DISCOVERY_TASK_STATUS_CONFIG[task.status] || { color: 'default', text: task.status }
            const targetCount = formatTargetCount(task.target)
            return (
              <List.Item
                actions={[
                  <Button key="detail" type="link" size="small" onClick={() => handleViewDiscoveryTask(task)}>
                    查看详情
                  </Button>,
                  <Button key="list" type="link" size="small" onClick={() => navigate('/security/tasks?task_group=discovery')}>
                    任务列表
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
                      <span>{targetCount} 个目标</span>
                      <span>进度 {task.progress || 0}%</span>
                      <span>{task.created_at ? new Date(task.created_at).toLocaleString('zh-CN') : '-'}</span>
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
        {/* 工具栏 */}
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col flex="auto">
            <Space wrap>
              <Input.Search
                placeholder="搜索 IP/服务/Banner"
                style={{ width: 250 }}
                onSearch={(value) => setFilters({ ...filters, keyword: value })}
                allowClear
              />
              <Select
                placeholder="资产类型"
                style={{ width: 120 }}
                allowClear
                value={filters.asset_type || undefined}
                onChange={(value) => setFilters({ ...filters, asset_type: value || '' })}
              >
                <Option value="server">服务器</Option>
                <Option value="network">网络设备</Option>
                <Option value="web">Web服务</Option>
                <Option value="database">数据库</Option>
                <Option value="other">其他</Option>
              </Select>
              <Select
                placeholder="状态"
                style={{ width: 100 }}
                allowClear
                value={filters.status || undefined}
                onChange={(value) => setFilters({ ...filters, status: value || '' })}
              >
                <Option value="online">在线</Option>
                <Option value="offline">离线</Option>
                <Option value="unknown">未知</Option>
              </Select>
              <Select
                placeholder="重要性"
                style={{ width: 100 }}
                allowClear
                value={filters.importance || undefined}
                onChange={(value) => setFilters({ ...filters, importance: value || '' })}
              >
                <Option value="critical">严重</Option>
                <Option value="high">高</Option>
                <Option value="medium">中</Option>
                <Option value="low">低</Option>
              </Select>
            </Space>
          </Col>
          <Col>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => { fetchAssets(); fetchRecentDiscoveryTasks() }}>刷新</Button>
              <Button icon={<RadarChartOutlined />} onClick={() => setDiscoveryModalOpen(true)}>
                发起资产发现
              </Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
                新增资产
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
            showTotal={(t) => `共 ${t} 条`}
          />
        </div>
      </Card>

      <Modal
        title="发起资产发现"
        open={discoveryModalOpen}
        onCancel={() => { setDiscoveryModalOpen(false); discoveryForm.resetFields() }}
        footer={null}
        width={560}
      >
        <Form form={discoveryForm} layout="vertical" onFinish={handleCreateDiscovery}>
          <Form.Item label="用途说明">
            <Card size="small" style={{ background: '#fafafa' }}>
              资产发现会对目标 IP 执行开放端口探测和服务识别，用于资产盘点，不进行漏洞检测。
            </Card>
          </Form.Item>

          <Form.Item
            name="name"
            label="任务名称"
            tooltip="可选，不填会自动生成任务名"
          >
            <Input placeholder="例如：办公网段资产摸底（可留空自动生成）" />
          </Form.Item>

          <Form.Item
            name="target"
            label="目标 IP"
            rules={[{ required: true, message: '请输入目标 IP 地址' }]}
            extra="支持多个 IP，逗号或换行分隔"
          >
            <Input.TextArea
              rows={5}
              placeholder="例如：192.168.1.10&#10;192.168.1.11&#10;10.0.0.10,10.0.0.20"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setDiscoveryModalOpen(false); discoveryForm.resetFields() }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={discoverySubmitting}>
                创建任务
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

      {/* 创建资产弹窗 */}
      <Modal
        title="新增资产"
        open={createModalOpen}
        onCancel={() => { setCreateModalOpen(false); form.resetFields() }}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ip" label="IP地址" rules={[{ required: true, message: '请输入IP地址' }]}>
                <Input placeholder="192.168.1.1" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="port" label="端口">
                <Input type="number" placeholder="80" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="protocol" label="协议">
                <Select placeholder="选择协议" allowClear>
                  <Option value="TCP">TCP</Option>
                  <Option value="UDP">UDP</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="service_name" label="服务名称">
                <Input placeholder="nginx" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="version" label="版本">
                <Input placeholder="1.20.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="asset_type" label="资产类型">
                <Select placeholder="选择类型" allowClear>
                  <Option value="server">服务器</Option>
                  <Option value="network">网络设备</Option>
                  <Option value="web">Web服务</Option>
                  <Option value="database">数据库</Option>
                  <Option value="other">其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="status" label="状态">
                <Select placeholder="选择状态" allowClear>
                  <Option value="online">在线</Option>
                  <Option value="offline">离线</Option>
                  <Option value="unknown">未知</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="importance" label="重要性">
                <Select placeholder="选择重要性" allowClear>
                  <Option value="critical">严重</Option>
                  <Option value="high">高</Option>
                  <Option value="medium">中</Option>
                  <Option value="low">低</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="owner" label="负责人">
                <Input placeholder="张三" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="department" label="部门">
                <Input placeholder="运维部" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="asset_group" label="资产分组">
            <Input placeholder="生产环境" />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Input placeholder="web,nginx,生产" />
          </Form.Item>
          <Form.Item name="os_info" label="操作系统信息">
            <Input placeholder="Linux 5.4.0" />
          </Form.Item>
          <Form.Item name="banner" label="Banner">
            <Input.TextArea rows={2} placeholder="服务指纹信息" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">创建</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑资产弹窗 */}
      <Modal
        title="编辑资产"
        open={editModalOpen}
        onCancel={() => { setEditModalOpen(false); editForm.resetFields() }}
        footer={null}
        width={600}
      >
        <Form form={editForm} layout="vertical" onFinish={handleEdit}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ip" label="IP地址" rules={[{ required: true, message: '请输入IP地址' }]}>
                <Input placeholder="192.168.1.1" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="port" label="端口">
                <Input type="number" placeholder="80" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="protocol" label="协议">
                <Select placeholder="选择协议" allowClear>
                  <Option value="TCP">TCP</Option>
                  <Option value="UDP">UDP</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="service_name" label="服务名称">
                <Input placeholder="nginx" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="version" label="版本">
                <Input placeholder="1.20.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="asset_type" label="资产类型">
                <Select placeholder="选择类型" allowClear>
                  <Option value="server">服务器</Option>
                  <Option value="network">网络设备</Option>
                  <Option value="web">Web服务</Option>
                  <Option value="database">数据库</Option>
                  <Option value="other">其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="status" label="状态">
                <Select placeholder="选择状态" allowClear>
                  <Option value="online">在线</Option>
                  <Option value="offline">离线</Option>
                  <Option value="unknown">未知</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="importance" label="重要性">
                <Select placeholder="选择重要性" allowClear>
                  <Option value="critical">严重</Option>
                  <Option value="high">高</Option>
                  <Option value="medium">中</Option>
                  <Option value="low">低</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="owner" label="负责人">
                <Input placeholder="张三" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="department" label="部门">
                <Input placeholder="运维部" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="asset_group" label="资产分组">
            <Input placeholder="生产环境" />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Input placeholder="web,nginx,生产" />
          </Form.Item>
          <Form.Item name="os_info" label="操作系统信息">
            <Input placeholder="Linux 5.4.0" />
          </Form.Item>
          <Form.Item name="banner" label="Banner">
            <Input.TextArea rows={2} placeholder="服务指纹信息" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">保存</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 资产详情弹窗 */}
      <Modal
        title="资产详情"
        open={detailModalOpen}
        onCancel={() => { setDetailModalOpen(false); setSelectedAsset(null); setSelectedAssetVulns([]); setSelectedAssetVulnCount(null) }}
        footer={[
          <Button key="close" onClick={() => setDetailModalOpen(false)}>关闭</Button>
        ]}
        width={900}
      >
        {selectedAsset && (
          <Tabs
            defaultActiveKey="info"
            items={[
              {
                key: 'info',
                label: '基本信息',
                icon: <DesktopOutlined />,
                children: (
                  <Descriptions column={2} bordered size="small">
                    <Descriptions.Item label="IP地址">{selectedAsset.ip}</Descriptions.Item>
                    <Descriptions.Item label="端口">{selectedAsset.port || '-'}</Descriptions.Item>
                    <Descriptions.Item label="协议">{selectedAsset.protocol || '-'}</Descriptions.Item>
                    <Descriptions.Item label="服务">{renderServiceWithStatus(selectedAsset.service_name)}</Descriptions.Item>
                    <Descriptions.Item label="版本">{selectedAsset.version || '-'}</Descriptions.Item>
                    <Descriptions.Item label="资产类型">{selectedAsset.asset_type || '-'}</Descriptions.Item>
                    <Descriptions.Item label="状态">
                      {(() => {
                        const config = STATUS_CONFIG[selectedAsset.status] || STATUS_CONFIG.unknown
                        return <Badge status={config.color as 'success' | 'error' | 'default'} text={config.text} />
                      })()}
                    </Descriptions.Item>
                    <Descriptions.Item label="重要性">
                      {selectedAsset.importance ? (
                        (() => {
                          const config = IMPORTANCE_CONFIG[selectedAsset.importance]
                          return <Tag color={config?.color}>{config?.text}</Tag>
                        })()
                      ) : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="负责人">{selectedAsset.owner || '-'}</Descriptions.Item>
                    <Descriptions.Item label="部门">{selectedAsset.department || '-'}</Descriptions.Item>
                    <Descriptions.Item label="资产分组">{selectedAsset.asset_group || '-'}</Descriptions.Item>
                    <Descriptions.Item label="标签">{selectedAsset.tags || '-'}</Descriptions.Item>
                    <Descriptions.Item label="操作系统" span={2}>{selectedAsset.os_info || '-'}</Descriptions.Item>
                    <Descriptions.Item label="前端服务" span={2}>
                      {selectedAssetBanner.frontend || selectedAsset.banner || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="后端应用" span={2}>
                      {selectedAssetBanner.upstream || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="Banner" span={2}>{selectedAsset.banner || '-'}</Descriptions.Item>
                    <Descriptions.Item label="首次发现">{selectedAsset.first_seen ? new Date(selectedAsset.first_seen).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
                    <Descriptions.Item label="最近发现">{selectedAsset.last_seen ? new Date(selectedAsset.last_seen).toLocaleString('zh-CN') : '-'}</Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'vulns',
                label: '关联漏洞',
                icon: <BugOutlined />,
                children: (
                  <div>
                    {/* 漏洞统计 */}
                    {selectedAssetVulnCount && (
                      <Row gutter={16} style={{ marginBottom: 16 }}>
                        <Col span={4}>
                          <Card size="small">
                            <Statistic
                              title="漏洞总数"
                              value={selectedAssetVulnCount.total}
                              valueStyle={{ color: selectedAssetVulnCount.total > 0 ? '#cf1322' : '#3f8600' }}
                              prefix={<BugOutlined />}
                            />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title="严重" value={selectedAssetVulnCount.critical} valueStyle={{ color: '#722ed1' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title="高危" value={selectedAssetVulnCount.high} valueStyle={{ color: '#cf1322' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title="中危" value={selectedAssetVulnCount.medium} valueStyle={{ color: '#fa8c16' }} />
                          </Card>
                        </Col>
                        <Col span={5}>
                          <Card size="small">
                            <Statistic title="低危" value={selectedAssetVulnCount.low} valueStyle={{ color: '#52c41a' }} />
                          </Card>
                        </Col>
                      </Row>
                    )}

                    {/* 风险评估进度条 */}
                    {selectedAssetVulnCount && selectedAssetVulnCount.total > 0 && (
                      <Card size="small" title="风险分布" style={{ marginBottom: 16 }}>
                        <Progress
                          percent={100}
                          success={{ percent: (selectedAssetVulnCount.low / selectedAssetVulnCount.total) * 100, strokeColor: '#52c41a' }}
                          strokeColor="#fa8c16"
                          trailColor="#cf1322"
                          format={() => `严重/高危: ${selectedAssetVulnCount.critical + selectedAssetVulnCount.high} | 中危: ${selectedAssetVulnCount.medium} | 低危: ${selectedAssetVulnCount.low}`}
                        />
                      </Card>
                    )}

                    {/* 漏洞列表 */}
                    <Table
                      dataSource={selectedAssetVulns}
                      loading={vulnsLoading}
                      rowKey="id"
                      size="small"
                      pagination={{ pageSize: 5 }}
                      locale={{ emptyText: <Empty description="暂无关联漏洞" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
                      columns={[
                        {
                          title: '严重程度',
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
                              critical: '严重',
                              high: '高危',
                              medium: '中危',
                              low: '低危',
                              info: '信息',
                            }
                            return <Tag color={colorMap[severity] || 'default'}>{textMap[severity] || severity}</Tag>
                          },
                        },
                        {
                          title: '漏洞名称',
                          dataIndex: 'title',
                          ellipsis: true,
                        },
                        {
                          title: 'CVE',
                          dataIndex: 'cve_id',
                          width: 140,
                          render: (cve: string) => cve || '-',
                        },
                        {
                          title: '端口',
                          dataIndex: 'port',
                          width: 70,
                        },
                        {
                          title: '状态',
                          dataIndex: 'status',
                          width: 90,
                          render: (status: string) => {
                            const statusMap: Record<string, { color: string; text: string }> = {
                              open: { color: 'processing', text: '待处理' },
                              acknowledged: { color: 'warning', text: '已确认' },
                              fixed: { color: 'success', text: '已修复' },
                              ignored: { color: 'default', text: '已忽略' },
                            }
                            const config = statusMap[status] || { color: 'default', text: status }
                            return <Tag color={config.color}>{config.text}</Tag>
                          },
                        },
                        {
                          title: '发现时间',
                          dataIndex: 'created_at',
                          width: 160,
                          render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
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