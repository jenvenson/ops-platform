import { useEffect, useState } from 'react'
import { Table, Card, Button, Modal, Form, Input, Select, Tag, Space, message, Popconfirm, Typography, Radio, Checkbox, Row, Col, Alert } from 'antd'
import {
  PlusOutlined,
  DeleteOutlined,
  EyeOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { securityAPI, SecurityScanTask, PaginatedResponse } from '../../api/security'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import TaskDetail from './TaskDetail'
import { canEdit } from '../../utils/menuAccess'
import { useSearchParams } from 'react-router-dom'
import WebScanPrecheckCard from './WebScanPrecheck'

const { Title, Text } = Typography

// 预设目标（仅用于提示参考）
const PRESET_TARGETS = [
  { value: '192.168.1.100', label: '192.168.1.100', type: 'ip_list', description: '单台主机' },
  { value: '10.0.0.10,10.0.0.20', label: '10.0.0.10,10.0.0.20', type: 'ip_list', description: '多台服务器' },
]

const WEB_PRESET_TARGETS = [
  { value: 'http://192.168.1.100:8080', label: 'http://192.168.1.100:8080', description: '单个 Web 站点' },
  { value: 'http://10.0.0.10:8080\nhttp://10.0.0.20:8080', label: 'http://10.0.0.10:8080 / http://10.0.0.20:8080', description: '多个 Web 入口' },
]

const webScanOptionConfigs = [
  { value: 'sql-injection', label: 'SQL 注入' },
  { value: 'xss', label: 'XSS 跨站脚本' },
  { value: 'ssrf', label: 'SSRF 服务端请求伪造' },
  { value: 'csrf', label: 'CSRF 跨站请求伪造' },
  { value: 'rce', label: '远程代码执行' },
  { value: 'information-disclosure', label: '信息泄露' },
  { value: 'broken-access', label: '越权访问' },
  { value: 'file-inclusion', label: '文件包含' },
  { value: 'header-injection', label: '响应头注入' },
]

const webScanModeConfigs = [
  {
    value: 'standard',
    title: '标准扫描',
    desc: '适合首轮排查，使用推荐的 Web 通用模板与规则集。',
  },
  {
    value: 'special',
    title: '专项扫描',
    desc: '按选中类别做更积极的专项验证，默认使用更深预算。',
  },
]

function getScanTypeLabel(scanType: 'port' | 'host-vuln' | 'web' | 'all') {
  switch (scanType) {
    case 'web':
      return 'Web漏洞扫描'
    case 'host-vuln':
      return '主机漏洞扫描'
    case 'port':
      return '资产发现'
    default:
      return '安全扫描'
  }
}

function buildDefaultTaskName(scanType: 'port' | 'host-vuln' | 'web' | 'all') {
  const timestamp = new Date().toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).replace(/\//g, '-')
  return `${getScanTypeLabel(scanType)}-${timestamp}`
}

export default function TaskList() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [tasks, setTasks] = useState<SecurityScanTask[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [detailTask, setDetailTask] = useState<SecurityScanTask | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [form] = Form.useForm()
  const taskGroupFilter = searchParams.get('task_group') || 'vuln'
  const statusFilter = searchParams.get('status') || ''

  const updateTaskFilters = (next: { task_group?: string; status?: string }) => {
    const params = new URLSearchParams(searchParams)
    if (next.task_group !== undefined) {
      if (next.task_group) {
        params.set('task_group', next.task_group)
      } else {
        params.delete('task_group')
      }
    }
    if (next.status !== undefined) {
      if (next.status) {
        params.set('status', next.status)
      } else {
        params.delete('status')
      }
    }
    setPage(1)
    setSearchParams(params)
  }

  const fetchTasks = async () => {
    setLoading(true)
    try {
      const response = await securityAPI.getTasks({
        page,
        page_size: pageSize,
        status: statusFilter || undefined,
        task_group: taskGroupFilter === 'all' ? undefined : (taskGroupFilter as 'vuln' | 'discovery'),
      })
      const data = response as PaginatedResponse<SecurityScanTask>
      setTasks(data.data || [])
      setTotal(data.total || 0)
    } catch (error) {
      message.error('获取任务列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTasks()
  }, [page, pageSize, taskGroupFilter, statusFilter])

  const handleCreateTask = async (values: {
    name?: string
    scan_type?: 'port' | 'host-vuln' | 'web' | 'all'
    target_type?: 'ip_list' | 'url'
    target: string
    web_scan_mode?: 'standard' | 'special'
    web_scan_options?: string[]
    login_url?: string
    username?: string
    password?: string
  }) => {
    try {
      const scanType = values.scan_type || 'web'
      let target = values.target
      let targetType = values.target_type || 'ip_list'
      let authFlow: Record<string, unknown> | undefined
      const taskName = values.name?.trim() || buildDefaultTaskName(scanType)

      // Web 漏洞扫描：目标类型为 URL
      if (scanType === 'web') {
        targetType = 'url'
        // 解析多个 URL（支持逗号或换行分隔）
        target = values.target
          .split(/[,\n]/)
          .map(url => url.trim())
          .filter(url => url.length > 0)
          .join(',')
      } else {
        // 端口/主机漏洞：处理 IP 列表
        if (values.target_type === 'ip_list') {
          target = values.target
            .split(/[,\n]/)
            .map(ip => ip.trim())
            .filter(ip => ip.length > 0)
            .join(',')
        }
      }

      // 构建请求数据
      const requestData: Record<string, unknown> = {
        name: taskName,
        scan_type: scanType,
        target_type: targetType,
        target,
      }

      // Web 扫描附加参数
      if (scanType === 'web') {
        requestData.discovery_mode = 'browser'
        requestData.web_scan_profile = values.web_scan_mode === 'special' ? 'deep' : 'standard'

        if (values.web_scan_mode === 'special' && values.web_scan_options && values.web_scan_options.length > 0) {
          requestData.web_scan_options = values.web_scan_options.join(',')
        }

        const entryTarget = target.split(',').map(item => item.trim()).find(Boolean)
        if (!entryTarget) {
          message.error('请先填写有效的 Web 目标地址')
          return
        }
        const generated = await securityAPI.generateAuthFlow({
          preset: 'auto',
          target_url: entryTarget,
          login_url: values.login_url?.trim() || undefined,
        })
        authFlow = generated.auth_flow
        requestData.auth_mode = 'advanced'
        requestData.login_url = values.login_url?.trim() || undefined
        requestData.username = values.username?.trim()
        requestData.password = values.password
        requestData.auth_flow = authFlow
      }

      await securityAPI.createTask({
        name: taskName,
        scan_type: scanType,
        target_type: targetType,
        target,
        ...requestData,
      } as Parameters<typeof securityAPI.createTask>[0])
      message.success('任务创建成功，扫描已开始')
      setCreateModalOpen(false)
      form.resetFields()
      setPage(1)
      fetchTasks()
    } catch (error) {
      message.error('创建任务失败')
    }
  }

  const handleDeleteTask = async (id: number) => {
    try {
      await securityAPI.deleteTask(id)
      message.success('任务删除成功')
      fetchTasks()
    } catch (error) {
      message.error('删除任务失败')
    }
  }

  const statusColors: Record<string, string> = {
    pending: 'default',
    running: 'processing',
    paused: 'warning',
    cancelled: 'default',
    completed: 'success',
    failed: 'error',
  }

  const statusLabels: Record<string, string> = {
    pending: '等待中',
    running: '运行中',
    paused: '已请求暂停',
    cancelled: '已请求取消',
    completed: '已完成',
    failed: '失败',
  }

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '目标类型',
      dataIndex: 'target_type',
      key: 'target_type',
      width: 100,
      render: (type: string) => (
        <Tag>{type === 'url' ? 'URL' : 'IP列表'}</Tag>
      ),
    },
    {
      title: '扫描目标',
      dataIndex: 'target',
      key: 'target',
      ellipsis: true,
    },
    {
      title: '扫描类型',
      dataIndex: 'scan_type',
      key: 'scan_type',
      width: 120,
      render: (type: string) => {
        const typeConfig: Record<string, { color: string; label: string }> = {
          'port': { color: 'blue', label: '资产发现' },
          'host-vuln': { color: 'orange', label: '主机漏洞' },
          'web': { color: 'green', label: 'Web漏洞' },
          'host': { color: 'blue', label: '资产发现' }, // 兼容旧数据
        }
        const config = typeConfig[type] || { color: 'default', label: type }
        return <Tag color={config.color}>{config.label}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status]}>
          {statusLabels[status] || status}
        </Tag>
      ),
    },
    {
      title: '已确认高/中/低',
      key: 'risk',
      width: 150,
      render: (_: unknown, record: SecurityScanTask) => (
        <Space size="small">
          <Tag color="red">{record.high_risk}</Tag>
          <Tag color="orange">{record.medium_risk}</Tag>
          <Tag color="blue">{record.low_risk}</Tag>
        </Space>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (time: string) => (time ? new Date(time).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '完成时间',
      dataIndex: 'completed_at',
      key: 'completed_at',
      width: 160,
      render: (time: string | undefined) => (time ? new Date(time).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: SecurityScanTask) => (
        <Space size="small">
          {(record.status === 'completed' || record.status === 'running') && (
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setDetailTask(record)}
            >
              详情
            </Button>
          )}
          {canEdit() && <Popconfirm
            title="确定删除此任务？"
            onConfirm={() => handleDeleteTask(record.id)}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>扫描任务</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => fetchTasks()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            新建漏洞扫描
          </Button>
        </Space>
      </div>

      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前安全任务列表发起查询"
        actions={[
          { label: '最近有哪些失败扫描任务', query: '最近有哪些失败扫描任务' },
          { label: '当前还有哪些扫描任务在运行', query: '当前还有哪些扫描任务在运行' },
          { label: '最近扫描异常集中在哪些目标', query: '最近扫描异常集中在哪些目标' },
        ]}
      />

      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, marginBottom: 16, flexWrap: 'wrap' }}>
          <Space wrap>
            <Select
              value={taskGroupFilter}
              style={{ width: 160 }}
              onChange={(value: string) => updateTaskFilters({ task_group: value })}
              options={[
                { value: 'vuln', label: '漏洞扫描' },
                { value: 'discovery', label: '资产发现' },
                { value: 'all', label: '全部任务' },
              ]}
            />
            <Select
              placeholder="状态"
              allowClear
              value={statusFilter || undefined}
              style={{ width: 140 }}
              onChange={(value?: string) => updateTaskFilters({ status: value || '' })}
              options={[
                { value: 'pending', label: '等待中' },
                { value: 'running', label: '运行中' },
                { value: 'paused', label: '已请求暂停' },
                { value: 'cancelled', label: '已请求取消' },
                { value: 'completed', label: '已完成' },
                { value: 'failed', label: '失败' },
              ]}
            />
          </Space>
          <Text type="secondary">
            {taskGroupFilter === 'discovery'
              ? '当前仅展示资产发现任务'
              : taskGroupFilter === 'all'
                ? '当前展示全部任务'
              : '当前仅展示漏洞扫描任务'}
          </Text>
        </div>
        {(taskGroupFilter === 'vuln' || taskGroupFilter === 'all') && (
          <Alert
            style={{ marginBottom: 16 }}
            type="info"
            showIcon
            message="漏洞计数口径已收紧"
            description="扫描任务的高/中/低危汇总会排除资产识别和全部待验证结果。待验证结果主要来自目标服务版本与漏洞知识库自动比对后的风险线索。若某环境尚未执行历史回填，旧任务短期内可能仍保留旧口径；详情页会按明细结果拆分展示正式结果、待验证和资产信息。"
          />
        )}
        <Table
          columns={columns}
          dataSource={tasks}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showQuickJumper: true,
            onChange: (nextPage, nextPageSize) => {
              if (nextPageSize !== pageSize) {
                setPageSize(nextPageSize)
                setPage(1)
                return
              }
              setPage(nextPage)
            },
          }}
          locale={{ emptyText: '暂无扫描任务' }}
        />
      </Card>

      {/* 创建任务弹窗 */}
      <Modal
        title="新建漏洞扫描"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        footer={null}
        width={550}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateTask}
          initialValues={{
            scan_type: 'web',
            target_type: 'ip_list',
            web_scan_mode: 'standard',
            web_scan_options: webScanOptionConfigs.map(option => option.value),
          }}
        >
          <Form.Item
            name="name"
            label="任务名称"
            tooltip="可选，不填会自动生成任务名"
          >
            <Input placeholder="例如：核心系统周扫（可留空自动生成）" />
          </Form.Item>

          <Form.Item
            name="scan_type"
            label="扫描类型"
            rules={[{ required: true, message: '请选择扫描类型' }]}
            initialValue="web"
            tooltip="新建扫描任务只提供网站漏洞和主机漏洞两类入口"
          >
            <Radio.Group>
              <Radio.Button value="web">网站漏洞</Radio.Button>
              <Radio.Button value="host-vuln">主机漏洞</Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.scan_type !== currentValues.scan_type ||
              prevValues.target_type !== currentValues.target_type
            }
          >
            {({ getFieldValue }) => {
              const scanType = getFieldValue('scan_type')

              // Web 漏洞扫描：显示 URL 输入、扫描选项、认证选项
              if (scanType === 'web') {
                return (
                  <>
                    <Form.Item label="推荐方式">
                      <Alert
                        message="Web 扫描只支持登录后扫描"
                        description="请填写业务入口 URL 和可登录测试账号。默认使用标准扫描；切到专项扫描时，会按你保留的类别同时约束模板和内置规则，并自动使用更深预算。"
                        type="info"
                        showIcon
                      />
                    </Form.Item>

                    <Form.Item
                      name="target"
                      label="扫描目标"
                      rules={[
                        { required: true, message: '请输入扫描 URL' },
                        {
                          validator: (_, value) => {
                            const raw = String(value || '').trim()
                            if (!raw) {
                              return Promise.resolve()
                            }

                            const urls = raw
                              .split(/[,\n]/)
                              .map((item) => item.trim())
                              .filter(Boolean)

                            const invalid = urls.find((item) => {
                              try {
                                const parsed = new URL(item)
                                return parsed.protocol !== 'http:' && parsed.protocol !== 'https:'
                              } catch {
                                return true
                              }
                            })

                            if (invalid) {
                              return Promise.reject(new Error(`URL 格式无效：${invalid}`))
                            }

                            return Promise.resolve()
                          },
                        },
                      ]}
                      extra="支持 http:// 或 https:// 的 URL 地址，每行一个"
                    >
                      <Input.TextArea
                        rows={3}
                        placeholder="例如：http://192.168.1.1:8080&#10;https://www.example.com&#10;http://10.0.0.5:9000"
                      />
                    </Form.Item>

                    <Form.Item
                      name="web_scan_mode"
                      label="模板策略"
                      initialValue="standard"
                    >
                      <Radio.Group>
                        {webScanModeConfigs.map((option) => (
                          <Radio.Button key={option.value} value={option.value}>
                            {option.title}
                          </Radio.Button>
                        ))}
                      </Radio.Group>
                    </Form.Item>

                    <Form.Item
                      noStyle
                      shouldUpdate={(prevValues, currentValues) =>
                        prevValues.web_scan_mode !== currentValues.web_scan_mode
                      }
                    >
                      {({ getFieldValue }) => (
                        <Alert
                          style={{ marginBottom: 16 }}
                          message={
                            getFieldValue('web_scan_mode') === 'special'
                              ? '专项扫描'
                              : '标准扫描'
                          }
                          description={
                            getFieldValue('web_scan_mode') === 'special'
                              ? '会按你勾选的漏洞类别执行模板和内置规则，并自动使用更积极的验证预算。'
                              : '使用推荐的通用模板与规则集，适合首轮扫描和常规巡检。'
                          }
                          type={getFieldValue('web_scan_mode') === 'standard' ? 'info' : 'warning'}
                          showIcon
                        />
                      )}
                    </Form.Item>

                    <Form.Item
                      noStyle
                      shouldUpdate={(prevValues, currentValues) =>
                        prevValues.web_scan_mode !== currentValues.web_scan_mode
                      }
                    >
                      {({ getFieldValue }) => (
                        <Form.Item
                          name="web_scan_options"
                          label="漏洞类别"
                          tooltip="专项扫描只会执行选中的漏洞类别，模板和对应内置规则会一起收敛"
                          hidden={getFieldValue('web_scan_mode') !== 'special'}
                          extra="默认已全选。通常只需要取消你暂时不关注的类别。"
                        >
                          <Checkbox.Group>
                            <Row gutter={[12, 12]}>
                              {webScanOptionConfigs.map((option) => (
                                <Col span={8} key={option.value}>
                                  <Card size="small">
                                    <Checkbox value={option.value}>{option.label}</Checkbox>
                                  </Card>
                                </Col>
                              ))}
                            </Row>
                          </Checkbox.Group>
                        </Form.Item>
                      )}
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="username" label="登录账号" rules={[{ required: true, message: '请输入账号' }]}>
                          <Input placeholder="例如：admin" />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="password" label="登录密码" rules={[{ required: true, message: '请输入密码' }]}>
                          <Input.Password placeholder="请输入密码" />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item
                      name="login_url"
                      label="登录接口 URL（可选）"
                      extra="不填则默认使用目标 URL 推断登录入口"
                    >
                      <Input placeholder="例如：https://example.com/api/login" />
                    </Form.Item>

                    <WebScanPrecheckCard form={form} compact />

                    <Alert
                      message="使用建议"
                      description="常规场景只需：选择扫描类型 -> 填目标 -> 填登录账号密码 -> 创建任务。模板和认证流程都会自动按默认策略处理。"
                      type="info"
                      showIcon
                    />
                  </>
                )
              }

              // 主机：显示 IP 列表输入
              return (
                <>
                  {scanType === 'port' && (
                    <Form.Item label="用途说明">
                      <Alert
                        message="资产发现只做端口和服务识别"
                        description="适合用于主机盘点、开放端口摸底和服务版本识别，不会执行漏洞检测。"
                        type="info"
                        showIcon
                      />
                    </Form.Item>
                  )}

                  <Form.Item
                    name="target_type"
                    label="目标类型"
                    rules={[{ required: true, message: '请选择目标类型' }]}
                    initialValue="ip_list"
                  >
                    <Radio.Group>
                      <Radio.Button value="ip_list">IP 列表</Radio.Button>
                    </Radio.Group>
                  </Form.Item>

                  <Form.Item
                    name="target"
                    label="IP 地址"
                    rules={[{ required: true, message: '请输入 IP 地址' }]}
                    extra={<Text type="secondary">支持多个 IP，逗号或换行分隔</Text>}
                  >
                    <Input.TextArea
                      rows={4}
                      placeholder="例如：192.168.1.10&#10;192.168.1.11&#10;10.0.0.5"
                    />
                  </Form.Item>
                </>
              )
            }}
          </Form.Item>

          {/* 预设目标参考 */}
          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.scan_type !== currentValues.scan_type}
          >
            {({ getFieldValue }) => (
              <Form.Item label="预设参考">
                <div style={{ fontSize: 12, color: '#999' }}>
                  <ul style={{ margin: 0, paddingLeft: 16 }}>
                    {(getFieldValue('scan_type') === 'web' ? WEB_PRESET_TARGETS : PRESET_TARGETS).map(t => (
                      <li key={t.value}>{t.label} - {t.description}</li>
                    ))}
                  </ul>
                </div>
              </Form.Item>
            )}
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Button onClick={() => setCreateModalOpen(false)} style={{ marginRight: 8 }}>
              取消
            </Button>
            <Button type="primary" htmlType="submit">
              创建并开始扫描
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      {/* 任务详情弹窗 */}
      {detailTask && (
        <TaskDetail
          task={detailTask}
          onClose={() => setDetailTask(null)}
          onRefresh={() => fetchTasks()}
        />
      )}
    </div>
  )
}
