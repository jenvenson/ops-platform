import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Card, Col, Descriptions, Drawer, Input, Row, Select, Space, Statistic, Table, Tag, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons'
import { securityFIMAPI, type FIMPolicy, type FIMSnapshot } from '../../api/security-fim'
import { cmdbAPI, type Server } from '../../api/cmdb'
import { getFIMErrorMessage } from '../../utils/httpError'
import { canEdit } from '../../utils/menuAccess'

const { Paragraph, Text, Title } = Typography

const snapshotTypeLabelMap: Record<string, string> = {
  baseline: '构建基线',
  manual: '手动扫描',
  scheduled: '自动扫描',
}

const snapshotTypeColorMap: Record<string, string> = {
  baseline: 'blue',
  manual: 'gold',
  scheduled: 'purple',
}

const statusLabelMap: Record<string, string> = {
  running: '执行中',
  success: '成功',
  failed: '失败',
}

const statusColorMap: Record<string, string> = {
  running: 'processing',
  success: 'success',
  failed: 'error',
}

const baselineStateColorMap: Record<string, string> = {
  active: 'success',
  historical: 'gold',
  none: 'default',
}

const baselineStateLabelMap: Record<string, string> = {
  active: '当前生效基线',
  historical: '历史基线',
  none: '普通扫描',
}

export default function FIMExecutionsPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [items, setItems] = useState<FIMSnapshot[]>([])
  const [policies, setPolicies] = useState<FIMPolicy[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null)
  const [detail, setDetail] = useState<FIMSnapshot | null>(null)
  const [filters, setFilters] = useState({
    policyId: parseQueryNumber(searchParams.get('policy_id')),
    serverId: parseQueryNumber(searchParams.get('server_id')),
    snapshotType: searchParams.get('snapshot_type') || undefined as string | undefined,
    baselineState: searchParams.get('baseline_state') || undefined as string | undefined,
    status: searchParams.get('status') || undefined as string | undefined,
    keyword: searchParams.get('keyword') || '',
  })

  useEffect(() => {
    setFilters({
      policyId: parseQueryNumber(searchParams.get('policy_id')),
      serverId: parseQueryNumber(searchParams.get('server_id')),
      snapshotType: searchParams.get('snapshot_type') || undefined,
      baselineState: searchParams.get('baseline_state') || undefined,
      status: searchParams.get('status') || undefined,
      keyword: searchParams.get('keyword') || '',
    })
  }, [searchParams])

  useEffect(() => {
    const nextParams = new URLSearchParams()
    if (filters.policyId) nextParams.set('policy_id', String(filters.policyId))
    if (filters.serverId) nextParams.set('server_id', String(filters.serverId))
    if (filters.snapshotType) nextParams.set('snapshot_type', filters.snapshotType)
    if (filters.baselineState) nextParams.set('baseline_state', filters.baselineState)
    if (filters.status) nextParams.set('status', filters.status)
    if (filters.keyword.trim()) nextParams.set('keyword', filters.keyword.trim())
    setSearchParams(nextParams, { replace: true })
  }, [filters, setSearchParams])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [snapshotsResp, policiesResp, serversResp] = await Promise.all([
        securityFIMAPI.getSnapshots({ page: 1, page_size: 200 }),
        securityFIMAPI.getPolicies({ page: 1, page_size: 200 }),
        cmdbAPI.getServers({ limit: 1000 }),
      ])
      setItems(snapshotsResp.data ?? [])
      setPolicies(policiesResp.data ?? [])
      setServers(serversResp.data ?? [])
    } catch {
      setItems([])
      setPolicies([])
      setServers([])
      message.error('加载执行记录失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchData()
  }, [])

  const policyMap = useMemo(() => new Map(policies.map((policy) => [policy.id, policy])), [policies])
  const serverMap = useMemo(() => new Map(servers.map((server) => [server.id, server])), [servers])

  const filteredItems = useMemo(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    return items.filter((item) => {
      if (filters.policyId && item.policy_id !== filters.policyId) {
        return false
      }
      if (filters.serverId && item.server_id !== filters.serverId) {
        return false
      }
      if (filters.snapshotType && item.snapshot_type !== filters.snapshotType) {
        return false
      }
      if (filters.baselineState && getBaselineState(item) !== filters.baselineState) {
        return false
      }
      if (filters.status && item.status !== filters.status) {
        return false
      }
      if (keyword) {
        const policyName = item.policy_name ?? policyMap.get(item.policy_id)?.name ?? ''
        const serverName = item.server_name ?? serverMap.get(item.server_id)?.hostname ?? ''
        const serverIP = item.server_ip ?? serverMap.get(item.server_id)?.ip ?? ''
        const haystack = [policyName, serverName, serverIP, item.operator ?? '', item.error_message ?? ''].join('\n').toLowerCase()
        if (!haystack.includes(keyword)) {
          return false
        }
      }
      return true
    })
  }, [filters, items, policyMap, serverMap])

  const openRelatedEvents = (record: FIMSnapshot) => {
    const params = new URLSearchParams()
    params.set('policy_id', String(record.policy_id))
    params.set('server_id', String(record.server_id))
    navigate(`/security/fim/events?${params.toString()}`)
  }

  const openRelatedAlerts = (record: FIMSnapshot) => {
    const params = new URLSearchParams()
    params.set('policy_id', String(record.policy_id))
    params.set('server_id', String(record.server_id))
    navigate(`/security/fim/alerts?${params.toString()}`)
  }

  const validateRetryExecution = async (record: FIMSnapshot) => {
    const policy = policyMap.get(record.policy_id)
    if (!policy) {
      throw new Error('当前策略已不存在，无法重试')
    }
    if (!policy.enabled) {
      throw new Error('当前策略已停用，请先启用后再重试')
    }

    const [targetsResp, pathsResp] = await Promise.all([
      securityFIMAPI.getTargets(record.policy_id),
      securityFIMAPI.getWatchPaths(record.policy_id),
    ])

    const targets = targetsResp.data ?? []
    const matchedTarget = targets.find((item) => item.server_id === record.server_id)
    if (!matchedTarget) {
      throw new Error('当前主机已不在策略绑定范围内，无法重试')
    }
    if (!matchedTarget.enabled) {
      throw new Error('当前主机绑定已停用，请先启用后再重试')
    }

    const watchPaths = pathsResp.data ?? []
    if (watchPaths.length === 0) {
      throw new Error('当前策略还没有配置监控目录，无法重试')
    }
  }

  const handleRetryExecution = async (record: FIMSnapshot) => {
    try {
      setActionLoadingId(record.id)
      await validateRetryExecution(record)
      if (getSnapshotOriginType(record) === 'baseline') {
        await securityFIMAPI.buildBaseline(record.policy_id, record.server_id)
      } else {
        const originType = getSnapshotOriginType(record)
        const scanType = originType === 'scheduled' ? 'scheduled' : 'manual'
        await securityFIMAPI.runScan(record.policy_id, record.server_id, scanType)
      }
      message.success('已重新触发执行')
      await fetchData()
    } catch (error) {
      message.error(getFIMErrorMessage(error, '重试执行失败'))
    } finally {
      setActionLoadingId(null)
    }
  }

  const columns: ColumnsType<FIMSnapshot> = [
    {
      title: '执行方式',
      key: 'snapshot_type',
      width: 180,
      render: (_value, record) => (
        <Space direction="vertical" size={0}>
          <Tag color={snapshotTypeColorMap[getSnapshotOriginType(record)] || 'default'}>
            {snapshotTypeLabelMap[getSnapshotOriginType(record)] || getSnapshotOriginType(record)}
          </Tag>
          <Text type="secondary">{getSnapshotExecutionHint(record)}</Text>
        </Space>
      ),
    },
    {
      title: '基线状态',
      key: 'baseline_state',
      width: 140,
      render: (_value, record) => {
        const baselineState = getBaselineState(record)
        return <Tag color={baselineStateColorMap[baselineState] || 'default'}>{baselineStateLabelMap[baselineState] || baselineState}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{statusLabelMap[value] || value}</Tag>,
    },
    {
      title: '策略',
      key: 'policy',
      width: 200,
      render: (_value, record) => record.policy_name || policyMap.get(record.policy_id)?.name || `策略 #${record.policy_id}`,
    },
    {
      title: '主机',
      key: 'server',
      width: 220,
      render: (_value, record) => {
        const server = serverMap.get(record.server_id)
        return record.server_name && record.server_ip
          ? `${record.server_name} (${record.server_ip})`
          : server
            ? `${server.hostname} (${server.ip})`
            : `主机 #${record.server_id}`
      },
    },
    {
      title: '执行人',
      dataIndex: 'operator',
      key: 'operator',
      width: 120,
      render: (value?: string) => value || 'system',
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: '结束时间',
      dataIndex: 'finished_at',
      key: 'finished_at',
      width: 180,
      render: (value?: string) => formatDateTime(value),
    },
    {
      title: '耗时',
      key: 'duration',
      width: 120,
      render: (_value, record) => formatDuration(record.started_at, record.finished_at),
    },
    {
      title: '采集条目',
      dataIndex: 'entry_count',
      key: 'entry_count',
      width: 100,
    },
    {
      title: '失败原因',
      dataIndex: 'error_message',
      key: 'error_message',
      width: 260,
      render: (value?: string) => {
        if (!value) {
          return '-'
        }
        return (
          <Text
            style={{ maxWidth: 240, display: 'inline-block' }}
            ellipsis={{ tooltip: value }}
            copyable={{ text: value, tooltips: ['复制失败原因', '已复制'] }}
          >
            {value}
          </Text>
        )
      },
    },
    {
      title: '处理',
      key: 'actions',
      width: 280,
      render: (_value, record) => (
        <Space size={0} wrap>
          <Button size="small" type="link" onClick={() => setDetail(record)}>
            详情
          </Button>
          <Button size="small" type="link" onClick={() => openRelatedEvents(record)}>
            事件
          </Button>
          <Button size="small" type="link" onClick={() => openRelatedAlerts(record)}>
            告警
          </Button>
          {canEdit() && (
            <Button
              size="small"
              type="link"
              disabled={record.status !== 'failed'}
              loading={actionLoadingId === record.id}
              onClick={() => void handleRetryExecution(record)}
            >
              重试
            </Button>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Space direction="vertical" size={4}>
          <Title level={4} style={{ margin: 0 }}>执行记录</Title>
          <Paragraph type="secondary" style={{ margin: 0 }}>
            统一查看基线构建、手动扫描和自动扫描的执行结果，重点定位是否执行、执行了谁、结果如何以及失败原因。
          </Paragraph>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title="成功执行" value={filteredItems.filter((item) => item.status === 'success').length} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title="执行中" value={filteredItems.filter((item) => item.status === 'running').length} prefix={<SyncOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title="执行失败" value={filteredItems.filter((item) => item.status === 'failed').length} prefix={<CloseCircleOutlined />} /></Card>
        </Col>
      </Row>

      <Card title="筛选条件">
        <Space wrap size={12}>
          <Select
            allowClear
            placeholder="按策略筛选"
            style={{ width: 220 }}
            value={filters.policyId}
            onChange={(value) => setFilters((current) => ({ ...current, policyId: value }))}
            options={policies.map((policy) => ({ value: policy.id, label: policy.name }))}
          />
          <Select
            allowClear
            placeholder="按主机筛选"
            style={{ width: 240 }}
            value={filters.serverId}
            onChange={(value) => setFilters((current) => ({ ...current, serverId: value }))}
            options={servers.map((server) => ({ value: server.id, label: `${server.hostname} (${server.ip})` }))}
          />
          <Select
            allowClear
            placeholder="按执行方式筛选"
            style={{ width: 180 }}
            value={filters.snapshotType}
            onChange={(value) => setFilters((current) => ({ ...current, snapshotType: value }))}
            options={['baseline', 'manual', 'scheduled'].map((value) => ({ value, label: snapshotTypeLabelMap[value] || value }))}
          />
          <Select
            allowClear
            placeholder="按基线状态筛选"
            style={{ width: 180 }}
            value={filters.baselineState}
            onChange={(value) => setFilters((current) => ({ ...current, baselineState: value }))}
            options={['active', 'historical', 'none'].map((value) => ({ value, label: baselineStateLabelMap[value] || value }))}
          />
          <Select
            allowClear
            placeholder="按状态筛选"
            style={{ width: 160 }}
            value={filters.status}
            onChange={(value) => setFilters((current) => ({ ...current, status: value }))}
            options={['running', 'success', 'failed'].map((value) => ({ value, label: statusLabelMap[value] || value }))}
          />
          <Input
            allowClear
            placeholder="搜索策略 / 主机 / 执行人 / 失败原因"
            style={{ width: 280 }}
            value={filters.keyword}
            onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
          />
        </Space>
      </Card>

      <Card title={`执行记录 (${filteredItems.length})`}>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={filteredItems}
          pagination={{ pageSize: 20, showSizeChanger: false }}
          locale={{ emptyText: '当前没有执行记录' }}
          scroll={{ x: 2050 }}
        />
      </Card>

      <Drawer
        title="执行记录详情"
        open={detail !== null}
        onClose={() => setDetail(null)}
        width={620}
      >
        {detail && (
          <>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="执行方式">
                <Tag color={snapshotTypeColorMap[getSnapshotOriginType(detail)] || 'default'}>
                  {snapshotTypeLabelMap[getSnapshotOriginType(detail)] || getSnapshotOriginType(detail)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="基线状态">
                <Tag color={baselineStateColorMap[getBaselineState(detail)] || 'default'}>
                  {baselineStateLabelMap[getBaselineState(detail)] || getBaselineState(detail)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColorMap[detail.status] || 'default'}>{statusLabelMap[detail.status] || detail.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="策略">
                {detail.policy_name || policyMap.get(detail.policy_id)?.name || `策略 #${detail.policy_id}`}
              </Descriptions.Item>
              <Descriptions.Item label="主机">
                {detail.server_name && detail.server_ip
                  ? `${detail.server_name} (${detail.server_ip})`
                  : (() => {
                    const server = serverMap.get(detail.server_id)
                    return server ? `${server.hostname} (${server.ip})` : `主机 #${detail.server_id}`
                  })()}
              </Descriptions.Item>
              <Descriptions.Item label="执行人">{detail.operator || 'system'}</Descriptions.Item>
              <Descriptions.Item label="开始时间">{formatDateTime(detail.started_at)}</Descriptions.Item>
              <Descriptions.Item label="结束时间">{formatDateTime(detail.finished_at)}</Descriptions.Item>
              <Descriptions.Item label="耗时">{formatDuration(detail.started_at, detail.finished_at)}</Descriptions.Item>
              <Descriptions.Item label="采集条目">{detail.entry_count}</Descriptions.Item>
              <Descriptions.Item label="失败原因">
                {detail.error_message ? (
                  <Paragraph copyable={{ text: detail.error_message, tooltips: ['复制失败原因', '已复制'] }} style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>
                    {detail.error_message}
                  </Paragraph>
                ) : '-'}
              </Descriptions.Item>
            </Descriptions>
            <Space style={{ marginTop: 16 }}>
              <Button onClick={() => openRelatedEvents(detail)}>查看关联事件</Button>
              <Button type="primary" onClick={() => openRelatedAlerts(detail)}>查看关联告警</Button>
              {canEdit() && (
                <Button
                  disabled={detail.status !== 'failed'}
                  loading={actionLoadingId === detail.id}
                  onClick={() => void handleRetryExecution(detail)}
                >
                  重试执行
                </Button>
              )}
            </Space>
          </>
        )}
      </Drawer>
    </div>
  )
}

function formatDateTime(value?: string): string {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

function parseQueryNumber(value: string | null): number | undefined {
  if (!value) {
    return undefined
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined
  }
  return parsed
}

function getSnapshotOriginType(snapshot: FIMSnapshot): string {
  return snapshot.origin_type || snapshot.snapshot_type || 'manual'
}

function getBaselineState(snapshot: FIMSnapshot): 'active' | 'historical' | 'none' {
  if (snapshot.snapshot_type === 'baseline') {
    return 'active'
  }
  if (getSnapshotOriginType(snapshot) === 'baseline') {
    return 'historical'
  }
  return 'none'
}

function getSnapshotExecutionHint(snapshot: FIMSnapshot): string {
  if (snapshot.snapshot_type === 'baseline') {
    return '当前用于比对的参考线'
  }
  if (getSnapshotOriginType(snapshot) === 'baseline') {
    return '曾作为基线，后续已被新基线替代'
  }
  return '普通巡检执行记录'
}

function formatDuration(startedAt?: string, finishedAt?: string): string {
  if (!startedAt || !finishedAt) {
    return '-'
  }
  const start = new Date(startedAt).getTime()
  const end = new Date(finishedAt).getTime()
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) {
    return '-'
  }
  const durationMs = end - start
  if (durationMs < 1000) {
    return `${durationMs} ms`
  }
  const seconds = durationMs / 1000
  if (seconds < 60) {
    return `${seconds.toFixed(1)} s`
  }
  const minutes = Math.floor(seconds / 60)
  const remainSeconds = Math.round(seconds % 60)
  return `${minutes} 分 ${remainSeconds} 秒`
}
