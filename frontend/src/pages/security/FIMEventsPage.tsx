// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Button, Card, Input, Popconfirm, Select, Space, Table, Tag, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { securityFIMAPI, type FIMDiffEvent, type FIMPolicy } from '../../api/security-fim'
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

const eventTypeColorMap: Record<string, string> = {
  create: 'green',
  delete: 'red',
  modify: 'gold',
  chmod: 'orange',
  chown: 'purple',
  rename: 'blue',
}

export default function FIMEventsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation('security')
  const [items, setItems] = useState<FIMDiffEvent[]>([])
  const [policies, setPolicies] = useState<FIMPolicy[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null)
  const [filters, setFilters] = useState({
    policyId: parseOptionalNumber(searchParams.get('policy_id')),
    serverId: parseOptionalNumber(searchParams.get('server_id')),
    severity: searchParams.get('severity') || undefined as string | undefined,
    eventType: searchParams.get('event_type') || undefined as string | undefined,
    keyword: searchParams.get('path') || searchParams.get('keyword') || '',
  })

  useEffect(() => {
    const nextParams = new URLSearchParams()
    if (filters.policyId) nextParams.set('policy_id', String(filters.policyId))
    if (filters.serverId) nextParams.set('server_id', String(filters.serverId))
    if (filters.severity) nextParams.set('severity', filters.severity)
    if (filters.eventType) nextParams.set('event_type', filters.eventType)
    if (filters.keyword.trim()) nextParams.set('keyword', filters.keyword.trim())
    setSearchParams(nextParams, { replace: true })
  }, [filters, setSearchParams])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [eventsResp, policiesResp, serversResp] = await Promise.all([
        securityFIMAPI.getEvents({ page: 1, page_size: 200 }),
        securityFIMAPI.getPolicies({ page: 1, page_size: 200 }),
        cmdbAPI.getServers({ limit: 1000 }),
      ])
      setItems(eventsResp.data ?? [])
      setPolicies(policiesResp.data ?? [])
      setServers(serversResp.data ?? [])
    } catch {
      setItems([])
      setPolicies([])
      setServers([])
      message.error(t('fimEvents.loadEventsFailed', '加载文件变更事件失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchData()
  }, [])

  const handleDeleteEvent = async (id: number) => {
    try {
      setActionLoadingId(id)
      await securityFIMAPI.deleteEvent(id)
      message.success(t('fimEvents.eventDeleted', '事件已删除'))
      await fetchData()
    } catch {
      message.error(t('fimEvents.eventDeleteFailed', '删除事件失败'))
    } finally {
      setActionLoadingId(null)
    }
  }

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
      if (filters.severity && item.severity !== filters.severity) {
        return false
      }
      if (filters.eventType && item.event_type !== filters.eventType) {
        return false
      }
      if (keyword) {
        const policyName = item.policy_name ?? policyMap.get(item.policy_id)?.name ?? ''
        const serverName = item.server_name ?? serverMap.get(item.server_id)?.hostname ?? ''
        const serverIP = item.server_ip ?? serverMap.get(item.server_id)?.ip ?? ''
        const haystack = [item.path, policyName, serverName, serverIP].join('\n').toLowerCase()
        if (!haystack.includes(keyword)) {
          return false
        }
      }
      return true
    })
  }, [filters, items, policyMap, serverMap])

  const columns: ColumnsType<FIMDiffEvent> = [
    {
      title: `${t('fim.policyName', '策略')} / ${t('fim.targetHost', '主机')}`,
      key: 'policy_server',
      width: 220,
      render: (_value, record) => {
        const policy = policyMap.get(record.policy_id)
        const server = serverMap.get(record.server_id)
        return (
          <Space direction="vertical" size={0}>
            <Text>{record.policy_name || policy?.name || `${t('fim.policyName', '策略')} #${record.policy_id}`}</Text>
            <Text type="secondary">
              {record.server_name && record.server_ip
                ? `${record.server_name} (${record.server_ip})`
                : server ? `${server.hostname} (${server.ip})` : `${t('fim.targetHost', '主机')} #${record.server_id}`}
            </Text>
          </Space>
        )
      },
    },
    {
      title: t('fim.inspectionDir', '路径'),
      dataIndex: 'path',
      key: 'path',
      width: 420,
      render: (value: string) => <Text code>{value}</Text>,
    },
    {
      title: t('fimEvents.eventType.create', '事件类型'),
      dataIndex: 'event_type',
      key: 'event_type',
      width: 120,
      render: (value: string) => <Tag color={eventTypeColorMap[value] || 'default'}>{t(`fimEvents.eventType.${value}`, value)}</Tag>,
    },
    {
      title: t('severityLevel', '级别'),
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (value: string) => <Tag color={severityColorMap[value] || 'default'}>{t(`severity.${value}`, value)}</Tag>,
    },
    {
      title: t('discoveryTime', '发生时间'),
      dataIndex: 'occurred_at',
      key: 'occurred_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: t('action', '操作'),
      key: 'actions',
      width: 120,
      render: (_value, record) => (
        canEdit() ? (
          <Popconfirm
            title={t('fimEvents.confirmDeleteEvent', '确认删除该事件？')}
            description={t('fimEvents.confirmDeleteEventDesc', '删除事件时会同步删除其关联告警。')}
            onConfirm={() => void handleDeleteEvent(record.id)}
            okText={t('delete', '删除')}
            cancelText={t('cancel', '取消')}
            okButtonProps={{ danger: true, loading: actionLoadingId === record.id }}
          >
            <Button type="link" danger loading={actionLoadingId === record.id}>
              {t('delete', '删除')}
            </Button>
          </Popconfirm>
        ) : '-'
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Space direction="vertical" size={4}>
          <Title level={4} style={{ margin: 0 }}>{t('fimEvents.title', '文件变更事件')}</Title>
          <Paragraph type="secondary" style={{ margin: 0 }}>
            {t('fimEvents.description', '展示基线与当前巡检快照的差异结果，便于排查具体发生了什么变化。')}
          </Paragraph>
        </Space>
      </Card>

      <Card title={t('fim.executions.filterCondition', '筛选条件')}>
        <Space wrap size={12}>
          <Select
            allowClear
            placeholder={t('fimEvents.filterByPolicy', '按策略筛选')}
            style={{ width: 220 }}
            value={filters.policyId}
            onChange={(value) => setFilters((current) => ({ ...current, policyId: value }))}
            options={policies.map((policy) => ({ value: policy.id, label: policy.name }))}
          />
          <Select
            allowClear
            placeholder={t('fimEvents.filterByHost', '按主机筛选')}
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
            placeholder={t('fimEvents.filterByEventType', '按事件类型筛选')}
            style={{ width: 180 }}
            value={filters.eventType}
            onChange={(value) => setFilters((current) => ({ ...current, eventType: value }))}
            options={['create', 'delete', 'modify', 'chmod', 'chown', 'rename'].map((value) => ({
              value,
              label: t(`fimEvents.eventType.${value}`, value),
            }))}
          />
          <Select
            allowClear
            placeholder={t('fimEvents.filterBySeverity', '按级别筛选')}
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
            placeholder={t('fimEvents.searchHint', '搜索路径 / 策略 / 主机')}
            style={{ width: 260 }}
            value={filters.keyword}
            onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
          />
        </Space>
      </Card>

      <Card title={t('fimEvents.recentDiffEvents', '最近差异事件 ({{count}})', { count: filteredItems.length })}>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={filteredItems}
          pagination={{ pageSize: 20, showSizeChanger: false }}
          locale={{ emptyText: t('fimEvents.noEvents', '当前没有文件变更事件') }}
          scroll={{ x: 1240 }}
        />
      </Card>
    </div>
  )
}

function parseOptionalNumber(value: string | null): number | undefined {
  if (!value) {
    return undefined
  }
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
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