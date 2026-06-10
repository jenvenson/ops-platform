// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Card, Col, DatePicker, Drawer, Empty, Input, Row, Select, Space, Statistic, Table, Tag, Timeline, Tooltip, Typography, message } from 'antd'
import { AppstoreOutlined, ArrowRightOutlined, BellOutlined, ClockCircleOutlined, DeploymentUnitOutlined, FieldTimeOutlined, FilterOutlined, RobotOutlined, SearchOutlined, SyncOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import useAssistantPageContext from '../../components/useAssistantPageContext'
import { platformEventsAPI, type PlatformEventListItem } from '../../api/platform-events'
import { hasMenuAccess, MENU_CHANGED_EVENT, readAllowedPaths } from '../../utils/menuAccess'

const { RangePicker } = DatePicker
const { Paragraph, Text, Title } = Typography

type EventFilterState = {
  q?: string
  event_category?: string
  source_system?: string
  object_type?: string
  object_id?: string
  status?: string
  severity?: string
  trigger_mode?: string
  occurred_from?: string
  occurred_to?: string
}

type QuickFilterKey = 'all' | 'attention' | 'failed' | 'highRisk' | 'today' | 'deploy' | 'alert' | 'assistant'

const categoryConfig: Record<string, { color: string; text: string }> = {
  deploy: { color: 'blue', text: '部署' },
  archive: { color: 'purple', text: '归档' },
  alert: { color: 'red', text: '告警' },
  assistant: { color: 'gold', text: '助手' },
}

const severityConfig: Record<string, { color: string; text: string }> = {
  critical: { color: 'red', text: '严重' },
  high: { color: 'volcano', text: '高' },
  warning: { color: 'orange', text: '警告' },
  medium: { color: 'gold', text: '中' },
  low: { color: 'blue', text: '低' },
  info: { color: 'default', text: '信息' },
}

const statusConfig: Record<string, { color: string; text: string }> = {
  pending: { color: 'default', text: '待执行' },
  queued: { color: 'processing', text: '排队中' },
  running: { color: 'processing', text: '进行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  firing: { color: 'error', text: '告警中' },
  acknowledged: { color: 'warning', text: '已介入' },
  resolved: { color: 'success', text: '已恢复' },
  closed: { color: 'default', text: '已关闭' },
  archived: { color: 'default', text: '已归档' },
}

const sourceIconMap: Record<string, JSX.Element> = {
  deploy: <DeploymentUnitOutlined style={{ color: '#1677ff' }} />,
  alert: <BellOutlined style={{ color: '#ff4d4f' }} />,
  assistant: <RobotOutlined style={{ color: '#faad14' }} />,
}

const currentDateValue = new Date().toISOString().slice(0, 10)

const defaultFilters: EventFilterState = {
  occurred_from: currentDateValue,
}

const quickFilterPresets: Record<QuickFilterKey, EventFilterState> = {
  all: defaultFilters,
  attention: { ...defaultFilters, status: 'failed' },
  failed: { ...defaultFilters, status: 'failed' },
  highRisk: { ...defaultFilters, severity: 'high' },
  today: { occurred_from: currentDateValue, occurred_to: currentDateValue },
  deploy: { ...defaultFilters, event_category: 'deploy' },
  alert: { ...defaultFilters, event_category: 'alert' },
  assistant: { ...defaultFilters, event_category: 'assistant' },
}

const getEventTarget = (event: PlatformEventListItem): { path: string; label: string } | null => {
  if (event.object_type === 'deploy_record' || event.event_category === 'deploy') {
    return { path: '/deploy/history', label: '查看原记录' }
  }
  if (event.object_type === 'archive_record' || event.event_category === 'archive') {
    return { path: '/deploy/archived', label: '查看原记录' }
  }
  if (event.object_type === 'alert_event' || event.event_category === 'alert') {
    return { path: '/alarm/events', label: '查看原记录' }
  }
  return null
}

export default function PlatformEventsPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [events, setEvents] = useState<PlatformEventListItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [filters, setFilters] = useState<EventFilterState>(defaultFilters)
  const [activeQuickFilter, setActiveQuickFilter] = useState<QuickFilterKey>('today')
  const [allowedPaths, setAllowedPaths] = useState<Set<string>>(() => readAllowedPaths())
  const [timelineState, setTimelineState] = useState<{
    open: boolean
    loading: boolean
    objectType?: string
    objectID?: string
    title?: string
    items: PlatformEventListItem[]
  }>({ open: false, loading: false, items: [] })

  useEffect(() => {
    const syncAllowedPaths = () => setAllowedPaths(readAllowedPaths())
    syncAllowedPaths()
    window.addEventListener('storage', syncAllowedPaths)
    window.addEventListener(MENU_CHANGED_EVENT, syncAllowedPaths)
    return () => {
      window.removeEventListener('storage', syncAllowedPaths)
      window.removeEventListener(MENU_CHANGED_EVENT, syncAllowedPaths)
    }
  }, [])

  useAssistantPageContext({
    filters: {
      eventCategory: filters.event_category,
      sourceSystem: filters.source_system,
      status: filters.status,
      severity: filters.severity,
      triggerMode: filters.trigger_mode,
      occurredFrom: filters.occurred_from,
      occurredTo: filters.occurred_to,
      keyword: filters.q,
    },
  })

  const fetchEvents = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await platformEventsAPI.getEvents({
        page,
        limit: pageSize,
        ...filters,
      })
      setEvents(resp.data || [])
      setTotal(resp.total || 0)
    } catch (error) {
      console.error('加载统一事件失败:', error)
      message.error('加载平台事件中心失败')
    } finally {
      setLoading(false)
    }
  }, [filters, page, pageSize])

  useEffect(() => {
    fetchEvents()
  }, [fetchEvents])

  useEffect(() => {
    const objectType = searchParams.get('object_type') || undefined
    const objectID = searchParams.get('object_id') || undefined

    if (!objectType || !objectID) {
      return
    }

    setFilters((current) => ({
      ...current,
      object_type: objectType,
      object_id: objectID,
    }))
    setPage(1)
  }, [searchParams])

  const summary = useMemo(() => {
    return events.reduce(
      (acc, item) => {
        acc.total += 1
        if (item.status === 'failed' || item.status === 'firing' || item.status === 'acknowledged') acc.attention += 1
        if (item.status === 'failed') acc.failed += 1
        if (item.event_category === 'alert' && (item.status === 'firing' || item.status === 'acknowledged')) acc.unresolvedAlert += 1
        if (item.event_category === 'assistant') acc.assistant += 1
        if (item.severity === 'critical' || item.severity === 'high') acc.highRisk += 1
        return acc
      },
      { total: 0, attention: 0, failed: 0, unresolvedAlert: 0, assistant: 0, highRisk: 0 }
    )
  }, [events])

  const eventPriority = useCallback((item: PlatformEventListItem) => {
    if (item.status === 'failed' || item.status === 'firing') return 0
    if (item.severity === 'critical' || item.severity === 'high') return 1
    if (item.status === 'acknowledged') return 2
    if (item.event_category === 'assistant') return 3
    return 4
  }, [])

  const prioritizedEvents = useMemo(() => {
    return [...events].sort((a, b) => {
      const priorityDiff = eventPriority(a) - eventPriority(b)
      if (priorityDiff !== 0) return priorityDiff
      const occurredAtDiff = new Date(b.occurred_at).getTime() - new Date(a.occurred_at).getTime()
      if (occurredAtDiff !== 0) return occurredAtDiff
      return b.id - a.id
    })
  }, [eventPriority, events])

  const applyQuickFilter = (key: QuickFilterKey) => {
    setActiveQuickFilter(key)
    setPage(1)
    setFilters((current) => ({
      ...current,
      ...quickFilterPresets[key],
      object_type: current.object_type,
      object_id: current.object_id,
      q: key === 'all' ? current.q : undefined,
    }))
  }

  const renderTargetButton = (path: string, label: string) => {
    const canAccess = hasMenuAccess(path, allowedPaths)
    const button = (
      <Button
        type="link"
        size="small"
        icon={<ArrowRightOutlined />}
        disabled={!canAccess}
        onClick={() => navigate(path)}
      >
        {label}
      </Button>
    )

    if (canAccess) {
      return button
    }

    return <Tooltip title="当前账号没有原页面权限">{button}</Tooltip>
  }

  const explainEventWithAssistant = (record: PlatformEventListItem) => {
    const query = `请解释这条平台事件需要关注什么：${record.title}，状态${record.status || '-'}，级别${record.severity || '-'}，摘要${record.summary || '-'}`
    window.dispatchEvent(new CustomEvent('ops-assistant:prompt', {
      detail: { query },
    }))
  }

  const openTimeline = async (record: PlatformEventListItem) => {
    setTimelineState({
      open: true,
      loading: true,
      objectType: record.object_type,
      objectID: record.object_id,
      title: record.title || record.object_id,
      items: [],
    })

    try {
      const resp = await platformEventsAPI.getTimeline({
        object_type: record.object_type,
        object_id: record.object_id,
        limit: 20,
      })
      setTimelineState((current) => ({
        ...current,
        loading: false,
        items: resp.data || [],
      }))
    } catch (error) {
      console.error('加载对象时间线失败:', error)
      message.error('加载对象时间线失败')
      setTimelineState((current) => ({ ...current, loading: false, items: [] }))
    }
  }

  useEffect(() => {
    const objectType = searchParams.get('object_type') || undefined
    const objectID = searchParams.get('object_id') || undefined
    const autoTimeline = searchParams.get('timeline')

    if (!autoTimeline || autoTimeline !== '1' || !objectType || !objectID || events.length === 0) {
      return
    }

    const matched = events.find((item) => item.object_type === objectType && item.object_id === objectID)
    if (matched) {
      void openTimeline(matched)
    }
  }, [events, searchParams])

  const columns: ColumnsType<PlatformEventListItem> = [
    {
      title: '来源模块',
      dataIndex: 'source_system',
      key: 'source_system',
      width: 130,
      render: (value: string) => (
        <Space size={8}>
          {sourceIconMap[value] || <AppstoreOutlined style={{ color: '#8c8c8c' }} />}
          <span>{value || '-'}</span>
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'event_category',
      key: 'event_category',
      width: 100,
      render: (value: string) => {
        const config = categoryConfig[value] || { color: 'default', text: value || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '最近动态',
      dataIndex: 'title',
      key: 'title',
      width: 380,
      render: (_: string, record) => (
        <div>
          <div style={{ fontWeight: 600, color: '#262626' }}>{record.title || record.event_type || '-'}</div>
          <div style={{ color: '#8c8c8c', marginTop: 4, lineHeight: 1.6 }}>{record.summary || '-'}</div>
          <Space size={8} wrap style={{ marginTop: 8 }}>
            <Tag color={(statusConfig[record.status] || { color: 'default' }).color}>{(statusConfig[record.status] || { text: record.status || '-' }).text}</Tag>
            <Tag color={(severityConfig[record.severity] || { color: 'default' }).color}>{(severityConfig[record.severity] || { text: record.severity || '-' }).text}</Tag>
            <Text type="secondary">{record.object_type || '-'}</Text>
          </Space>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (value: string) => {
        const config = statusConfig[value] || { color: 'default', text: value || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '风险',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (value: string) => {
        const config = severityConfig[value] || { color: 'default', text: value || '-' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '操作人',
      dataIndex: 'operator_name',
      key: 'operator_name',
      width: 120,
      render: (value: string, record) => value || record.operator_id || '-',
    },
    {
      title: '发生时间',
      dataIndex: 'occurred_at',
      key: 'occurred_at',
      width: 180,
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '处理',
      key: 'action',
      width: 300,
      fixed: 'right',
      render: (_: unknown, record) => {
        const target = getEventTarget(record)
        const timelineEnabled = Boolean(record.object_type && record.object_id)
        return (
          <Space size={4} wrap>
            {timelineEnabled && (
              <Button
                type="link"
                size="small"
                icon={<ClockCircleOutlined />}
                onClick={() => openTimeline(record)}
              >
                查看全过程
              </Button>
            )}
            <Button
              type="link"
              size="small"
              icon={<RobotOutlined />}
              onClick={() => explainEventWithAssistant(record)}
            >
              让助手解释
            </Button>
            {target ? renderTargetButton(target.path, target.label) : <span style={{ color: '#bfbfbf' }}>暂无原记录</span>}
          </Space>
        )
      },
    },
  ]

  const resetFilters = () => {
    setPage(1)
    setActiveQuickFilter('today')
    setFilters(defaultFilters)
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Space direction="vertical" size={4}>
          <Title level={4} style={{ margin: 0 }}>平台事件中心</Title>
          <Paragraph type="secondary" style={{ margin: 0 }}>
            统一查看部署、归档、告警和助手相关动态，优先识别失败、异常和高风险事件。
          </Paragraph>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small">
            <Statistic title="当前已加载事件" value={summary.total} prefix={<FieldTimeOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small">
            <Statistic title="需要关注" value={summary.attention} valueStyle={{ color: summary.attention > 0 ? '#cf1322' : undefined }} prefix={<FilterOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small">
            <Statistic title="未恢复告警" value={summary.unresolvedAlert} valueStyle={{ color: summary.unresolvedAlert > 0 ? '#ff4d4f' : undefined }} prefix={<BellOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small">
            <Statistic title="高风险事件" value={summary.highRisk} valueStyle={{ color: summary.highRisk > 0 ? '#ff4d4f' : undefined }} prefix={<DeploymentUnitOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card size="small" title="快捷查看" extra={<Text type="secondary">先看异常和风险，再下钻原记录或全过程</Text>}>
        <Space wrap>
          <Button type={activeQuickFilter === 'today' ? 'primary' : 'default'} onClick={() => applyQuickFilter('today')}>仅看今天</Button>
          <Button type={activeQuickFilter === 'attention' ? 'primary' : 'default'} danger={activeQuickFilter === 'attention'} onClick={() => applyQuickFilter('attention')}>仅看异常</Button>
          <Button type={activeQuickFilter === 'failed' ? 'primary' : 'default'} danger={activeQuickFilter === 'failed'} onClick={() => applyQuickFilter('failed')}>仅看失败</Button>
          <Button type={activeQuickFilter === 'highRisk' ? 'primary' : 'default'} onClick={() => applyQuickFilter('highRisk')}>仅看高风险</Button>
          <Button type={activeQuickFilter === 'deploy' ? 'primary' : 'default'} onClick={() => applyQuickFilter('deploy')}>部署相关</Button>
          <Button type={activeQuickFilter === 'alert' ? 'primary' : 'default'} onClick={() => applyQuickFilter('alert')}>告警相关</Button>
          <Button type={activeQuickFilter === 'assistant' ? 'primary' : 'default'} onClick={() => applyQuickFilter('assistant')}>助手相关</Button>
          <Button type={activeQuickFilter === 'all' ? 'primary' : 'default'} onClick={() => applyQuickFilter('all')}>查看全部</Button>
        </Space>
      </Card>

      <Card size="small" title="高级筛选">
        <Space wrap size={[12, 12]}>
          <Input.Search
            allowClear
            placeholder="搜索标题、摘要、对象或操作人"
            style={{ width: 280 }}
            enterButton={<SearchOutlined />}
            onSearch={(value) => {
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({ ...current, q: value || undefined }))
            }}
          />
          <Select
            allowClear
            placeholder="按模块查看"
            style={{ width: 140 }}
            value={filters.event_category}
            onChange={(value) => {
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({ ...current, event_category: value || undefined }))
            }}
            options={[
              { label: '部署', value: 'deploy' },
              { label: '归档', value: 'archive' },
              { label: '告警', value: 'alert' },
              { label: '助手', value: 'assistant' },
            ]}
          />
          <Select
            allowClear
            placeholder="来源系统"
            style={{ width: 140 }}
            value={filters.source_system}
            onChange={(value) => {
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({ ...current, source_system: value || undefined }))
            }}
            options={[
              { label: 'deploy', value: 'deploy' },
              { label: 'alert', value: 'alert' },
              { label: 'assistant', value: 'assistant' },
            ]}
          />
          <Select
            allowClear
            placeholder="按状态查看"
            style={{ width: 140 }}
            value={filters.status}
            onChange={(value) => {
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({ ...current, status: value || undefined }))
            }}
            options={[
              { label: '成功', value: 'success' },
              { label: '失败', value: 'failed' },
              { label: '告警中', value: 'firing' },
              { label: '已介入', value: 'acknowledged' },
              { label: '已恢复', value: 'resolved' },
              { label: '已关闭', value: 'closed' },
            ]}
          />
          <Select
            allowClear
            placeholder="按风险查看"
            style={{ width: 120 }}
            value={filters.severity}
            onChange={(value) => {
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({ ...current, severity: value || undefined }))
            }}
            options={[
              { label: '严重', value: 'critical' },
              { label: '高', value: 'high' },
              { label: '警告', value: 'warning' },
              { label: '中', value: 'medium' },
              { label: '低', value: 'low' },
              { label: '信息', value: 'info' },
            ]}
          />
          <RangePicker
            onChange={(_, dateStrings) => {
              const [from, to] = dateStrings as [string, string]
              setPage(1)
              setActiveQuickFilter('all')
              setFilters((current) => ({
                ...current,
                occurred_from: from || undefined,
                occurred_to: to || undefined,
              }))
            }}
          />
          <Button icon={<SyncOutlined />} onClick={fetchEvents}>
            刷新
          </Button>
          <Button onClick={resetFilters}>重置</Button>
        </Space>
      </Card>

      <AssistantQuickActions
        description="基于当前筛选结果，直接让运维小助手汇总最近异常、失败和高风险事件。"
        actions={[
          { label: '最近跨模块事件', query: '最近跨模块事件' },
          { label: '最近失败事件', query: '最近失败事件' },
          { label: '最近高风险事件', query: '最近高风险事件' },
        ]}
      />

      <Card
        title="需要关注的最近动态"
        extra={<span style={{ color: '#8c8c8c' }}>默认优先展示失败、异常和高风险事件，再按时间排序</span>}
      >
        <Table
          rowKey="event_id"
          loading={loading}
          columns={columns}
          dataSource={prioritizedEvents}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="当前筛选范围内没有需要关注的事件"
              />
            ),
          }}
          scroll={{ x: 1580 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (value) => `共 ${value} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage)
              setPageSize(nextPageSize)
            },
          }}
        />
      </Card>

      <Drawer
        title={timelineState.title ? `对象全过程: ${timelineState.title}` : '对象全过程'}
        width={520}
        open={timelineState.open}
        onClose={() => setTimelineState({ open: false, loading: false, items: [] })}
      >
        {timelineState.loading ? (
          <Card loading />
        ) : timelineState.items.length > 0 ? (
          <Timeline
            items={timelineState.items.map((item) => ({
              color: item.severity === 'critical' || item.severity === 'high' ? 'red' : item.status === 'failed' ? 'red' : item.status === 'success' ? 'green' : 'blue',
              children: (
                <div>
                  <div style={{ fontWeight: 600, color: '#262626' }}>{item.title || item.event_type}</div>
                  <div style={{ color: '#8c8c8c', marginTop: 4 }}>{item.summary || '-'}</div>
                  <Space size={8} wrap style={{ marginTop: 8 }}>
                    <Tag color={(categoryConfig[item.event_category] || { color: 'default' }).color}>{(categoryConfig[item.event_category] || { text: item.event_category || '-' }).text}</Tag>
                    <Tag color={(statusConfig[item.status] || { color: 'default' }).color}>{(statusConfig[item.status] || { text: item.status || '-' }).text}</Tag>
                    <span style={{ color: '#8c8c8c' }}>{item.occurred_at ? new Date(item.occurred_at).toLocaleString('zh-CN') : '-'}</span>
                  </Space>
                </div>
              ),
            }))}
          />
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当前对象没有更多事件" />
        )}
      </Drawer>
    </div>
  )
}