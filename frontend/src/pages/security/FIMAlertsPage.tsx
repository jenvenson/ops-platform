// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Card, Col, Descriptions, Drawer, Input, Popconfirm, Row, Select, Space, Statistic, Table, Tag, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { AlertOutlined, CheckCircleOutlined, WarningOutlined } from '@ant-design/icons'
import { securityFIMAPI, type FIMAlert, type FIMDiffEvent, type FIMPolicy } from '../../api/security-fim'
import { cmdbAPI, type Server } from '../../api/cmdb'
import { canEdit } from '../../utils/menuAccess'
import { useTranslation } from 'react-i18next'

const { Paragraph, Title, Text } = Typography

const severityColorMap: Record<string, string> = {
  critical: 'volcano',
  high: 'red',
  warning: 'gold',
  medium: 'orange',
  low: 'green',
  info: 'blue',
}

const statusColorMap: Record<string, string> = {
  open: 'red',
  acknowledged: 'gold',
  resolved: 'green',
  closed: 'default',
}

const eventTypeColorMap: Record<string, string> = {
  create: 'green',
  delete: 'red',
  modify: 'gold',
  chmod: 'orange',
  chown: 'purple',
  rename: 'blue',
}

export default function FIMAlertsPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation('security')
  const [items, setItems] = useState<FIMAlert[]>([])
  const [events, setEvents] = useState<FIMDiffEvent[]>([])
  const [policies, setPolicies] = useState<FIMPolicy[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null)
  const [detailAlert, setDetailAlert] = useState<FIMAlert | null>(null)
  const [filters, setFilters] = useState({
    policyId: parseOptionalNumber(searchParams.get('policy_id')),
    serverId: parseOptionalNumber(searchParams.get('server_id')),
    severity: searchParams.get('severity') || undefined as string | undefined,
    status: searchParams.get('status') || undefined as string | undefined,
    keyword: searchParams.get('keyword') || '',
  })

  useEffect(() => {
    const nextParams = new URLSearchParams()
    if (filters.policyId) nextParams.set('policy_id', String(filters.policyId))
    if (filters.serverId) nextParams.set('server_id', String(filters.serverId))
    if (filters.severity) nextParams.set('severity', filters.severity)
    if (filters.status) nextParams.set('status', filters.status)
    if (filters.keyword.trim()) nextParams.set('keyword', filters.keyword.trim())
    setSearchParams(nextParams, { replace: true })
  }, [filters, setSearchParams])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [alertsResp, eventsResp, policiesResp, serversResp] = await Promise.all([
        securityFIMAPI.getAlerts({ page: 1, page_size: 200 }),
        securityFIMAPI.getEvents({ page: 1, page_size: 500 }),
        securityFIMAPI.getPolicies({ page: 1, page_size: 200 }),
        cmdbAPI.getServers({ limit: 1000 }),
      ])
      setItems(alertsResp.data ?? [])
      setEvents(eventsResp.data ?? [])
      setPolicies(policiesResp.data ?? [])
      setServers(serversResp.data ?? [])
    } catch {
      setItems([])
      setEvents([])
      setPolicies([])
      setServers([])
      message.error(t('fimAlerts.loadAlertsFailed', '加载完整性告警失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchData()
  }, [])

  const policyMap = useMemo(() => new Map(policies.map((policy) => [policy.id, policy])), [policies])
  const serverMap = useMemo(() => new Map(servers.map((server) => [server.id, server])), [servers])
  const eventMap = useMemo(() => new Map(events.map((event) => [event.id, event])), [events])

  const filteredItems = useMemo(() => {
    const keyword = filters.keyword.trim().toLowerCase()
    return items.filter((item) => {
      if (filters.policyId && item.policy_id !== filters.policyId) {
        return false
      }
      if (filters.serverId && item.server_id !== filters.serverId) {
        return false
      }
      if (filters.severity && item.severity !== filters.severity) {
        return false
      }
      if (filters.status && item.status !== filters.status) {
        return false
      }
      if (keyword) {
        const policyName = item.policy_name ?? policyMap.get(item.policy_id)?.name ?? ''
        const serverName = item.server_name ?? serverMap.get(item.server_id)?.hostname ?? ''
        const serverIP = item.server_ip ?? serverMap.get(item.server_id)?.ip ?? ''
        const haystack = [item.title, item.summary ?? '', policyName, serverName, serverIP].join('\n').toLowerCase()
        if (!haystack.includes(keyword)) {
          return false
        }
      }
      return true
    })
  }, [filters, items, policyMap, serverMap])

  const handleUpdateStatus = async (id: number, action: 'ack' | 'resolve' | 'close') => {
    try {
      setActionLoadingId(id)
      if (action === 'ack') {
        await securityFIMAPI.ackAlert(id)
      } else if (action === 'resolve') {
        await securityFIMAPI.resolveAlert(id)
      } else {
        await securityFIMAPI.closeAlert(id)
      }
      message.success(t('fimAlerts.alertStatusUpdated', '告警状态已更新'))
      await fetchData()
    } catch {
      message.error(t('fimAlerts.alertStatusUpdateFailed', '更新告警状态失败'))
    } finally {
      setActionLoadingId(null)
    }
  }

  const handleDeleteAlert = async (id: number) => {
    try {
      setActionLoadingId(id)
      await securityFIMAPI.deleteAlert(id)
      if (detailAlert?.id === id) {
        setDetailAlert(null)
      }
      message.success(t('fimAlerts.alertDeleted', '告警已删除'))
      await fetchData()
    } catch {
      message.error(t('fimAlerts.alertDeleteFailed', '删除告警失败'))
    } finally {
      setActionLoadingId(null)
    }
  }

  const parseJSONValue = (value?: string) => {
    if (!value) {
      return null
    }
    try {
      return JSON.parse(value) as Record<string, unknown>
    } catch {
      return null
    }
  }

  const renderDiffValue = (value: Record<string, unknown> | null) => {
    if (!value) {
      return <Text type="secondary">-</Text>
    }
    return (
      <Space direction="vertical" size={2}>
        {Object.entries(value).map(([key, entryValue]) => (
          <Text key={key}>
            <Text strong>{key}:</Text> {String(entryValue)}
          </Text>
        ))}
      </Space>
    )
  }

  const handleOpenRelatedEvents = () => {
    if (!detailAlert) {
      return
    }
    const diffEvent = eventMap.get(detailAlert.diff_event_id)
    const params = new URLSearchParams()
    params.set('policy_id', String(detailAlert.policy_id))
    params.set('server_id', String(detailAlert.server_id))
    if (diffEvent?.path) {
      params.set('path', diffEvent.path)
    }
    navigate(`/security/fim/events?${params.toString()}`)
    setDetailAlert(null)
  }

  const columns: ColumnsType<FIMAlert> = [
    {
      title: t('fimAlerts.titleLabel', '标题'),
      dataIndex: 'title',
      key: 'title',
      width: 280,
      render: (value: string, record) => (
        <Space direction="vertical" size={0}>
          <Space size={8} wrap>
            <Text>{value}</Text>
            {(record.occurrence_count ?? 1) > 1 ? <Tag color="orange">{t('fimAlerts.duplicateLabel', '重复出现')}</Tag> : null}
          </Space>
          <Text type="secondary">{record.policy_name || policyMap.get(record.policy_id)?.name || `${t('fim.policyName', '策略')} #${record.policy_id}`}</Text>
        </Space>
      ),
    },
    {
      title: t('fim.targetHost', '主机'),
      key: 'server',
      width: 220,
      render: (_value, record) => {
        const server = serverMap.get(record.server_id)
        return <Text>{record.server_name && record.server_ip ? `${record.server_name} (${record.server_ip})` : server ? `${server.hostname} (${server.ip})` : `${t('fim.targetHost', '主机')} #${record.server_id}`}</Text>
      },
    },
    {
      title: t('fimAlerts.summary', '摘要'),
      dataIndex: 'summary',
      key: 'summary',
      width: 360,
      render: (value: string) => value || '-',
    },
    {
      title: t('severityLevel', '级别'),
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (value: string) => <Tag color={severityColorMap[value] || 'default'}>{t(`severity.${value}`, value)}</Tag>,
    },
    {
      title: t('status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{t(`status.${value}`, value)}</Tag>,
    },
    {
      title: t('fimAlerts.firstSeen', '首次发现'),
      dataIndex: 'first_seen_at',
      key: 'first_seen_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('fimAlerts.lastSeen', '最近发现'),
      dataIndex: 'last_seen_at',
      key: 'last_seen_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('fimAlerts.occurrenceCount', '出现次数'),
      dataIndex: 'occurrence_count',
      key: 'occurrence_count',
      width: 100,
      render: (value?: number) => value ?? 1,
    },
    {
      title: t('fim.executions.handle', '处理'),
      key: 'actions',
      width: 220,
      render: (_value, record) => (
        <Space wrap>
          <Button size="small" type="link" onClick={() => setDetailAlert(record)}>
            {t('detail', '详情')}
          </Button>
          {canEdit() && (
            <>
              <Button
                size="small"
                onClick={() => void handleUpdateStatus(record.id, 'ack')}
                loading={actionLoadingId === record.id}
                disabled={record.status !== 'open'}
              >
                {t('fimAlerts.acknowledge', '确认')}
              </Button>
              <Button
                size="small"
                onClick={() => void handleUpdateStatus(record.id, 'resolve')}
                loading={actionLoadingId === record.id}
                disabled={record.status === 'resolved' || record.status === 'closed'}
              >
                {t('fimAlerts.resolve', '解决')}
              </Button>
              <Button
                size="small"
                onClick={() => void handleUpdateStatus(record.id, 'close')}
                loading={actionLoadingId === record.id}
                disabled={record.status === 'closed'}
              >
                {t('fimAlerts.closeAlert', '关闭')}
              </Button>
              <Popconfirm
                title={t('fimAlerts.confirmDeleteAlert', '确认删除该告警？')}
                onConfirm={() => void handleDeleteAlert(record.id)}
                okText={t('delete', '删除')}
                cancelText={t('cancel', '取消')}
                okButtonProps={{ danger: true, loading: actionLoadingId === record.id }}
              >
                <Button
                  size="small"
                  danger
                  loading={actionLoadingId === record.id}
                >
                  {t('delete', '删除')}
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Space direction="vertical" size={4}>
          <Title level={4} style={{ margin: 0 }}>{t('fimAlerts.title', '完整性告警')}</Title>
          <Paragraph type="secondary" style={{ margin: 0 }}>
            {t('fimAlerts.description', '聚焦需要处理的完整性异常，当前已支持主机和策略维度查看，以及基础状态流转。')}
          </Paragraph>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title={t('fimAlerts.unprocessedAlerts', '未处理告警')} value={filteredItems.filter((item) => item.status === 'open').length} prefix={<AlertOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title={t('fimAlerts.highRiskAlerts', '高危告警')} value={filteredItems.filter((item) => item.severity === 'high' || item.severity === 'critical').length} prefix={<WarningOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card size="small"><Statistic title={t('fimAlerts.recoveredClosed', '已恢复/已关闭')} value={filteredItems.filter((item) => item.status === 'resolved' || item.status === 'closed').length} prefix={<CheckCircleOutlined />} /></Card>
        </Col>
      </Row>

      <Card title={t('fim.executions.filterCondition', '筛选条件')}>
        <Space wrap size={12}>
          <Select
            allowClear
            placeholder={t('fimAlerts.filterByPolicy', '按策略筛选')}
            style={{ width: 220 }}
            value={filters.policyId}
            onChange={(value) => setFilters((current) => ({ ...current, policyId: value }))}
            options={policies.map((policy) => ({ value: policy.id, label: policy.name }))}
          />
          <Select
            allowClear
            placeholder={t('fimAlerts.filterByHost', '按主机筛选')}
            style={{ width: 240 }}
            value={filters.serverId}
            onChange={(value) => setFilters((current) => ({ ...current, serverId: value }))}
            options={servers.map((server) => ({
              value: server.id,
              label: `${server.hostname} (${server.ip})`,
            }))}
          />
          <Select
            allowClear
            placeholder={t('fimAlerts.filterByStatus', '按状态筛选')}
            style={{ width: 180 }}
            value={filters.status}
            onChange={(value) => setFilters((current) => ({ ...current, status: value }))}
            options={['open', 'acknowledged', 'resolved', 'closed'].map((value) => ({
              value,
              label: t(`status.${value}`, value),
            }))}
          />
          <Select
            allowClear
            placeholder={t('fimAlerts.filterBySeverity', '按级别筛选')}
            style={{ width: 160 }}
            value={filters.severity}
            onChange={(value) => setFilters((current) => ({ ...current, severity: value }))}
            options={['critical', 'high', 'warning', 'medium', 'low', 'info'].map((value) => ({
              value,
              label: t(`severity.${value}`, value),
            }))}
          />
          <Input
            allowClear
            placeholder={t('fimAlerts.searchHint', '搜索标题 / 摘要 / 主机')}
            style={{ width: 260 }}
            value={filters.keyword}
            onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
          />
        </Space>
      </Card>

      <Card title={t('fimAlerts.alertList', '告警列表 ({{count}})', { count: filteredItems.length })}>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={filteredItems}
          pagination={{ pageSize: 20, showSizeChanger: false }}
          locale={{ emptyText: t('fimAlerts.noAlerts', '当前没有完整性告警') }}
          scroll={{ x: 1800 }}
        />
      </Card>

      <Drawer
        title={t('fimAlerts.alertDetail', '完整性告警详情')}
        open={detailAlert !== null}
        onClose={() => setDetailAlert(null)}
        width={620}
      >
        {detailAlert && (
          <>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label={t('fimAlerts.titleLabel', '标题')}>{detailAlert.title}</Descriptions.Item>
              <Descriptions.Item label={t('fim.policyName', '策略')}>
                {detailAlert.policy_name || policyMap.get(detailAlert.policy_id)?.name || `${t('fim.policyName', '策略')} #${detailAlert.policy_id}`}
              </Descriptions.Item>
              <Descriptions.Item label={t('fim.targetHost', '主机')}>
                {detailAlert.server_name && detailAlert.server_ip
                  ? `${detailAlert.server_name} (${detailAlert.server_ip})`
                  : (() => {
                    const server = serverMap.get(detailAlert.server_id)
                    return server ? `${server.hostname} (${server.ip})` : `${t('fim.targetHost', '主机')} #${detailAlert.server_id}`
                  })()}
              </Descriptions.Item>
              <Descriptions.Item label={t('severityLevel', '级别')}>
                <Tag color={severityColorMap[detailAlert.severity] || 'default'}>{t(`severity.${detailAlert.severity}`, detailAlert.severity)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('status', '状态')}>
                <Tag color={statusColorMap[detailAlert.status] || 'default'}>{t(`status.${detailAlert.status}`, detailAlert.status)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.summary', '摘要')}>{detailAlert.summary || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.firstSeen', '首次发现')}>{formatDateTime(detailAlert.first_seen_at)}</Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.lastSeen', '最近发现')}>{formatDateTime(detailAlert.last_seen_at)}</Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.totalOccurrenceCount', '累计发现次数')}>{detailAlert.occurrence_count ?? 1}</Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.duplicateStatus', '重复状态')}>
                {(detailAlert.occurrence_count ?? 1) > 1 ? <Tag color="orange">{t('fimAlerts.duplicateLabel', '重复出现')}</Tag> : <Tag>{t('fimAlerts.firstDiscovery', '首次发现')}</Tag>}
              </Descriptions.Item>
              <Descriptions.Item label={t('fimAlerts.assignee', '处理人')}>{detailAlert.assignee || '-'}</Descriptions.Item>
            </Descriptions>

            <Card size="small" title={t('fimAlerts.diffDetail', '差异详情')} style={{ marginTop: 16 }}>
              {(() => {
                const diffEvent = eventMap.get(detailAlert.diff_event_id)
                if (!diffEvent) {
                  return <Text type="secondary">{t('noDiffEventDetail', '当前未找到差异事件详情。')}</Text>
                }
                const oldValue = parseJSONValue(diffEvent.old_value_json)
                const newValue = parseJSONValue(diffEvent.new_value_json)
                return (
                  <Descriptions column={1} size="small" bordered>
                    <Descriptions.Item label={t('fim.inspectionDir', '路径')}>
                      <Text code>{diffEvent.path}</Text>
                    </Descriptions.Item>
                    <Descriptions.Item label={t('fimEvents.eventType.create', '事件类型')}>
                      <Tag color={eventTypeColorMap[diffEvent.event_type] || 'default'}>{t(`fimEvents.eventType.${diffEvent.event_type}`, diffEvent.event_type)}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label={t('discoveryTime', '发生时间')}>{formatDateTime(diffEvent.occurred_at)}</Descriptions.Item>
                    <Descriptions.Item label={t('fimAlerts.beforeChange', '变更前')}>{renderDiffValue(oldValue)}</Descriptions.Item>
                    <Descriptions.Item label={t('fimAlerts.afterChange', '变更后')}>{renderDiffValue(newValue)}</Descriptions.Item>
                  </Descriptions>
                )
              })()}
            </Card>

            <Space style={{ marginTop: 16 }}>
              <Button type="primary" onClick={handleOpenRelatedEvents}>
                {t('fimAlerts.viewRelatedEvents', '查看关联事件')}
              </Button>
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

function parseOptionalNumber(value: string | null): number | undefined {
  if (!value) {
    return undefined
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined
  }
  return parsed
}