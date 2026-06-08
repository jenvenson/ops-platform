import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Table, Tag, Button, Modal, Form, Input, Select, Card, Space,
  message, Popconfirm, Badge, Statistic, Row, Col, Timeline, Drawer,
} from 'antd'
import {
  AlertOutlined, SyncOutlined, CheckCircleOutlined, CloseCircleOutlined,
  InfoCircleOutlined, FileTextOutlined, FieldTimeOutlined, DeleteOutlined,
} from '@ant-design/icons'
import { alertAPI, AlertEvent, AlertEventLog, EventStats } from '../../api/alert'
import { severityConfig, categoryConfig, eventStatusConfig } from './alarmConfig'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import useAssistantPageContext from '../../components/useAssistantPageContext'
import { canEdit } from '../../utils/menuAccess'

export default function AlertEventsPage() {
  const navigate = useNavigate()
  const [events, setEvents] = useState<AlertEvent[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [stats, setStats] = useState<EventStats | null>(null)
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; event?: AlertEvent }>({ open: false })
  const [eventLogs, setEventLogs] = useState<AlertEventLog[]>([])
  const [ackModal, setAckModal] = useState<{ open: boolean; eventId?: number }>({ open: false })
  const [ackForm] = Form.useForm()

  useAssistantPageContext({
    objectType: detailDrawer.event ? 'alert_event' : undefined,
    objectId: detailDrawer.event?.id,
    selectedRecordIds: detailDrawer.event ? [detailDrawer.event.id] : [],
    filters: {
      severity: filters.severity,
      category: filters.category,
      status: filters.status,
      source: filters.source,
    },
  })

  const fetchEvents = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = { page: String(page), page_size: String(pageSize), ...filters }
      const resp = await alertAPI.events.list(params)
      setEvents(resp.data || [])
      setTotal(resp.total || 0)
    } catch {
      message.error('加载告警中心数据失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, filters])

  const fetchStats = useCallback(async () => {
    try {
      const resp = await alertAPI.events.getStats()
      setStats(resp)
    } catch { /* ignore */ }
  }, [])

  useEffect(() => { fetchEvents() }, [fetchEvents])
  useEffect(() => { fetchStats() }, [fetchStats])

  const openDetail = async (event: AlertEvent) => {
    setDetailDrawer({ open: true, event })
    try {
      const resp = await alertAPI.events.getLogs(event.id)
      setEventLogs(resp.data || [])
    } catch { /* ignore */ }
  }

  const handleAck = async () => {
    if (!ackModal.eventId) return
    try {
      const values = await ackForm.validateFields()
      await alertAPI.events.ack(ackModal.eventId, values)
      message.success('已介入处理')
      setAckModal({ open: false })
      ackForm.resetFields()
      fetchEvents()
      fetchStats()
    } catch {
      message.error('操作失败')
    }
  }

  const handleClose = async (id: number) => {
    try {
      await alertAPI.events.close(id, { note: '手动关闭' })
      message.success('已关闭')
      fetchEvents()
      fetchStats()
    } catch {
      message.error('操作失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await alertAPI.events.delete(id)
      message.success('已删除')
      fetchEvents()
      fetchStats()
    } catch {
      message.error('删除失败')
    }
  }

  const actionLogMap: Record<string, { color: string; label: string }> = {
    created:  { color: 'red', label: '告警触发' },
    acked:    { color: 'orange', label: '介入处理' },
    resolved: { color: 'green', label: '自动恢复' },
    closed:   { color: 'gray', label: '关闭' },
    notified: { color: 'blue', label: '通知发送' },
    note:     { color: 'purple', label: '备注' },
  }

  const columns = [
    {
      title: '级别', dataIndex: 'severity', key: 'severity', width: 80,
      render: (v: string) => {
        const cfg = severityConfig[v] || { color: 'default', text: v }
        return <Tag color={cfg.color}>{cfg.text}</Tag>
      },
    },
    {
      title: '分类', dataIndex: 'category', key: 'category', width: 90,
      render: (v: string) => {
        const cfg = categoryConfig[v] || { color: 'default', text: v || '-' }
        return <Tag color={cfg.color}>{cfg.text}</Tag>
      },
    },
    { title: '规则名称', dataIndex: 'rule_name', key: 'rule_name', width: 220, ellipsis: true },
    { title: '告警内容', dataIndex: 'content', key: 'content', width: 280, ellipsis: true },
    { title: '来源', dataIndex: 'source', key: 'source', width: 150, ellipsis: true },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (v: string) => {
        const cfg = eventStatusConfig[v] || { color: 'default', text: v }
        return <Badge status={cfg.color as 'error' | 'warning' | 'success' | 'default'} text={cfg.text} />
      },
    },
    {
      title: '处理方式', dataIndex: 'handle_type', key: 'handle_type', width: 100,
      render: (v: string) => {
        if (!v) return '-'
        const map: Record<string, string> = { ticket: '工单', auto: '自动化', manual: '手动标注' }
        return map[v] || v
      },
    },
    { title: '触发时间', dataIndex: 'fired_at', key: 'fired_at', width: 170 },
    {
      title: '操作', key: 'action', width: 240, fixed: 'right' as const,
      render: (_: unknown, record: AlertEvent) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<FileTextOutlined />} onClick={() => openDetail(record)}>详情</Button>
          <Button
            type="link"
            size="small"
            onClick={() => navigate(`/platform/events?object_type=alert_event&object_id=alert_event:alert:${record.id}&timeline=1`)}
          >
            事件流
          </Button>
          {record.status === 'firing' && (
            <Button type="link" size="small" icon={<CheckCircleOutlined />} onClick={() => { setAckModal({ open: true, eventId: record.id }); ackForm.resetFields() }}>介入</Button>
          )}
          {(record.status === 'firing' || record.status === 'acknowledged') && (
            <Popconfirm title="确定关闭此告警？" onConfirm={() => handleClose(record.id)}>
              <Button type="link" size="small" danger icon={<CloseCircleOutlined />}>关闭</Button>
            </Popconfirm>
          )}
          {canEdit() && (
            <Popconfirm title="确定删除此告警？" onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const firingCount = stats?.firing_count || 0
  const todayCount = stats?.today_count || 0
  const statusCounts = (stats?.status_stats || []).reduce((acc, s) => ({ ...acc, [s.status]: s.count }), {} as Record<string, number>)

  return (
    <div>
      {/* 统计概览 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic title="告警中" value={firingCount} valueStyle={{ color: firingCount > 0 ? '#ff4d4f' : '#52c41a' }} prefix={<AlertOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="今日新增" value={todayCount} prefix={<FieldTimeOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="已介入" value={statusCounts['acknowledged'] || 0} valueStyle={{ color: '#faad14' }} prefix={<CheckCircleOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="已恢复" value={statusCounts['resolved'] || 0} valueStyle={{ color: '#52c41a' }} prefix={<InfoCircleOutlined />} />
          </Card>
        </Col>
      </Row>

      {/* 筛选 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select placeholder="级别" allowClear style={{ width: 100 }} onChange={v => setFilters(f => ({ ...f, severity: v || '' }))}
            options={[{ label: '严重', value: 'critical' }, { label: '警告', value: 'warning' }, { label: '提醒', value: 'info' }]} />
          <Select placeholder="分类" allowClear style={{ width: 120 }} onChange={v => setFilters(f => ({ ...f, category: v || '' }))}
            options={Object.entries(categoryConfig).map(([k, v]) => ({ label: v.text, value: k }))} />
          <Select placeholder="状态" allowClear style={{ width: 120 }} onChange={v => setFilters(f => ({ ...f, status: v || '' }))}
            options={Object.entries(eventStatusConfig).map(([k, v]) => ({ label: v.text, value: k }))} />
          <Input.Search placeholder="来源" allowClear style={{ width: 200 }} onSearch={v => setFilters(f => ({ ...f, source: v }))} />
          <Button icon={<SyncOutlined />} onClick={() => { fetchEvents(); fetchStats() }}>刷新</Button>
        </Space>
      </Card>

      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前告警页面上下文发起查询"
        actions={[
          { label: '最新告警动作', query: '最新告警动作' },
          { label: '查看未恢复告警', query: '查看未恢复告警' },
          { label: '查看已恢复告警', query: '查看已恢复告警' },
        ]}
      />

      <Table columns={columns} dataSource={events} rowKey="id" loading={loading} scroll={{ x: 1500 }}
        pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: t => `共 ${t} 条`,
          pageSizeOptions: ['10', '20', '50'], onChange: (p, ps) => { setPage(p); setPageSize(ps) } }} />

      {/* 介入处理弹窗 */}
      <Modal title="介入处理" open={ackModal.open} onOk={handleAck} onCancel={() => setAckModal({ open: false })} width={500}>
        <Form form={ackForm} layout="vertical">
          <Form.Item name="handle_type" label="处理方式" rules={[{ required: true, message: '请选择处理方式' }]}>
            <Select options={[{ label: '创建工单', value: 'ticket' }, { label: '自动化处理', value: 'auto' }, { label: '手动标注', value: 'manual' }]} />
          </Form.Item>
          <Form.Item name="handle_note" label="处理备注">
            <Input.TextArea rows={3} placeholder="请输入处理备注（可选）" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 告警详情抽屉 */}
      <Drawer title="告警详情" open={detailDrawer.open} onClose={() => setDetailDrawer({ open: false })} width={520}>
        {detailDrawer.event && (
          <div>
            <Card size="small" style={{ marginBottom: 16 }}>
              <p><b>规则名称：</b>{detailDrawer.event.rule_name}</p>
              <p><b>告警内容：</b>{detailDrawer.event.content}</p>
              <p><b>来源：</b>{detailDrawer.event.source}</p>
              <p><b>级别：</b><Tag color={severityConfig[detailDrawer.event.severity]?.color}>{severityConfig[detailDrawer.event.severity]?.text}</Tag></p>
              <p><b>状态：</b>{eventStatusConfig[detailDrawer.event.status]?.text}</p>
              <p><b>触发时间：</b>{detailDrawer.event.fired_at}</p>
              {detailDrawer.event.acked_at && <p><b>介入时间：</b>{detailDrawer.event.acked_at}（{detailDrawer.event.acked_by}）</p>}
              {detailDrawer.event.closed_at && <p><b>关闭时间：</b>{detailDrawer.event.closed_at}（{detailDrawer.event.closed_by}）</p>}
              {detailDrawer.event.handle_type && <p><b>处理方式：</b>{{ ticket: '工单', auto: '自动化', manual: '手动标注' }[detailDrawer.event.handle_type] || detailDrawer.event.handle_type}</p>}
              {detailDrawer.event.handle_note && <p><b>处理备注：</b>{detailDrawer.event.handle_note}</p>}
            </Card>
            <h4><FieldTimeOutlined /> 生命周期</h4>
            <Timeline items={eventLogs.map(log => ({
              color: actionLogMap[log.action]?.color || 'gray',
              children: (
                <div>
                  <b>{actionLogMap[log.action]?.label || log.action}</b>
                  {log.operator && <span style={{ color: '#888' }}> by {log.operator}</span>}
                  <br /><span style={{ color: '#999', fontSize: 12 }}>{log.created_at}</span>
                  {log.content && <div style={{ marginTop: 4, color: '#555' }}>{log.content}</div>}
                </div>
              ),
            }))} />
          </div>
        )}
      </Drawer>
    </div>
  )
}
