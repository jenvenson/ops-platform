// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

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
import { useTranslation } from 'react-i18next'
import i18next from '../../i18n'
import { getDateLocale, formatDateTime } from '../../utils/dateFormat'

const { Title, Text } = Typography

// 预设目标（仅用于提示参考）
const PRESET_TARGETS = [
  { value: '192.168.1.100', label: '192.168.1.100', type: 'ip_list', description: i18next.t('security:presetSingleHost', { defaultValue: '单台主机' }) },
  { value: '10.0.0.10,10.0.0.20', label: '10.0.0.10,10.0.0.20', type: 'ip_list', description: i18next.t('security:presetMultipleHosts', { defaultValue: '多台服务器' }) },
]

const WEB_PRESET_TARGETS = [
  { value: 'http://192.168.1.100:8080', label: 'http://192.168.1.100:8080', description: i18next.t('security:presetSingleSite', { defaultValue: '单个 Web 站点' }) },
  { value: 'http://10.0.0.10:8080\nhttp://10.0.0.20:8080', label: 'http://10.0.0.10:8080 / http://10.0.0.20:8080', description: i18next.t('security:presetMultipleSites', { defaultValue: '多个 Web 入口' }) },
]

function getScanTypeLabelKey(scanType: 'port' | 'host-vuln' | 'web' | 'all') {
  switch (scanType) {
    case 'web':
      return 'webVulnScan'
    case 'host-vuln':
      return 'hostVulnScan'
    case 'port':
      return 'portScan'
    default:
      return 'securityScan'
  }
}

function buildDefaultTaskName(scanType: 'port' | 'host-vuln' | 'web' | 'all', t: (key: string, fallback: string) => string) {
  const timestamp = new Date().toLocaleString(getDateLocale(), {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).replace(/\//g, '-')
  return `${t(getScanTypeLabelKey(scanType), '安全扫描')}-${timestamp}`
}

export default function TaskList() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation('security')
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
      message.error(t('taskListLoadFailed', '获取任务列表失败'))
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
      const taskName = values.name?.trim() || buildDefaultTaskName(scanType, t)

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
          message.error(t('emptyWebTargetError', '请先填写有效的 Web 目标地址'))
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
      message.success(t('taskCreated', '任务创建成功，扫描已开始'))
      setCreateModalOpen(false)
      form.resetFields()
      setPage(1)
      fetchTasks()
    } catch (error) {
      message.error(t('createTaskFailed', '创建任务失败'))
    }
  }

  const handleDeleteTask = async (id: number) => {
    try {
      await securityAPI.deleteTask(id)
      message.success(t('taskDeleteSuccess', '任务删除成功'))
      fetchTasks()
    } catch (error) {
      message.error(t('taskDeleteFailed', '删除任务失败'))
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

  const webScanOptionConfigs = [
    { value: 'sql-injection', label: t('sqlInjection', 'SQL 注入') },
    { value: 'xss', label: t('xss', 'XSS 跨站脚本') },
    { value: 'ssrf', label: t('ssrf', 'SSRF 服务端请求伪造') },
    { value: 'csrf', label: t('csrf', 'CSRF 跨站请求伪造') },
    { value: 'rce', label: t('rce', '远程代码执行') },
    { value: 'information-disclosure', label: t('informationDisclosure', '信息泄露') },
    { value: 'broken-access', label: t('brokenAccess', '越权访问') },
    { value: 'file-inclusion', label: t('fileInclusion', '文件包含') },
    { value: 'header-injection', label: t('headerInjection', '响应头注入') },
  ]

  const webScanModeConfigs = [
    {
      value: 'standard',
      title: t('standardScan', '标准扫描'),
      desc: t('standardScanDesc', '适合首轮排查，使用推荐的 Web 通用模板与规则集。'),
    },
    {
      value: 'special',
      title: t('specialScan', '专项扫描'),
      desc: t('specialScanDesc', '按选中类别做更积极的专项验证，默认使用更深预算。'),
    },
  ]

  const columns = [
    {
      title: t('taskName', '任务名称'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: t('targetType', '目标类型'),
      dataIndex: 'target_type',
      key: 'target_type',
      width: 100,
      render: (type: string) => (
        <Tag>{type === 'url' ? 'URL' : t('ipList', 'IP列表')}</Tag>
      ),
    },
    {
      title: t('scanTarget', '扫描目标'),
      dataIndex: 'target',
      key: 'target',
      ellipsis: true,
    },
    {
      title: t('scanType', '扫描类型'),
      dataIndex: 'scan_type',
      key: 'scan_type',
      width: 120,
      render: (type: string) => {
        const typeConfig: Record<string, { color: string; label: string }> = {
          'port': { color: 'blue', label: t('portScan', '资产发现') },
          'host-vuln': { color: 'orange', label: t('hostVulnScan', '主机漏洞') },
          'web': { color: 'green', label: t('webScan', 'Web漏洞') },
          'host': { color: 'blue', label: t('portScan', '资产发现') },
        }
        const config = typeConfig[type] || { color: 'default', label: type }
        return <Tag color={config.color}>{config.label}</Tag>
      },
    },
    {
      title: t('common:status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status]}>
          {t(`status.${status}`, status)}
        </Tag>
      ),
    },
    {
      title: t('confirmedHighMediumLow', '已确认高/中/低'),
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
      title: t('createTime', '创建时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (time: string) => (time ? formatDateTime(time) : '-'),
    },
    {
      title: t('completeTime', '完成时间'),
      dataIndex: 'completed_at',
      key: 'completed_at',
      width: 160,
      render: (time: string | undefined) => (time ? formatDateTime(time) : '-'),
    },
    {
      title: t('action', '操作'),
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
              {t('detail', '详情')}
            </Button>
          )}
          {canEdit() && <Popconfirm
            title={t('confirmDeleteTask', '确定删除此任务？')}
            onConfirm={() => handleDeleteTask(record.id)}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              {t('delete', '删除')}
            </Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{t('scanTaskTitle', '扫描任务')}</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => fetchTasks()}>
            {t('refresh', '刷新')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            {t('newVulnScan', '新建漏洞扫描')}
          </Button>
        </Space>
      </div>

      <AssistantQuickActions
        description={t('scanTaskAssistantQuickActionsDesc', '复用右侧运维小助手，基于当前安全任务列表发起查询')}
        actions={[
          { label: t('scanHelp.recentFailedScans', '最近有哪些失败扫描任务'), query: '最近有哪些失败扫描任务' },
          { label: t('scanHelp.runningScans', '当前还有哪些扫描任务在运行'), query: '当前还有哪些扫描任务在运行' },
          { label: t('scanHelp.scanAbnormalTarget', '最近扫描异常集中在哪些目标'), query: '最近扫描异常集中在哪些目标' },
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
                { value: 'vuln', label: t('taskGroup.vuln', '漏洞扫描') },
                { value: 'discovery', label: t('taskGroup.discovery', '资产发现') },
                { value: 'all', label: t('taskGroup.all', '全部任务') },
              ]}
            />
            <Select
              placeholder={t('common:status', '状态')}
              allowClear
              value={statusFilter || undefined}
              style={{ width: 140 }}
              onChange={(value?: string) => updateTaskFilters({ status: value || '' })}
              options={[
                { value: 'pending', label: t('status.pending', '等待中') },
                { value: 'running', label: t('status.running', '运行中') },
                { value: 'paused', label: t('status.paused', '已请求暂停') },
                { value: 'cancelled', label: t('status.cancelled', '已请求取消') },
                { value: 'completed', label: t('status.completed', '已完成') },
                { value: 'failed', label: t('status.failed', '失败') },
              ]}
            />
          </Space>
          <Text type="secondary">
            {taskGroupFilter === 'discovery'
              ? t('currentlyShowingDiscovery', '当前仅展示资产发现任务')
              : taskGroupFilter === 'all'
                ? t('currentlyShowingAll', '当前展示全部任务')
              : t('currentlyShowingVuln', '当前仅展示漏洞扫描任务')}
          </Text>
        </div>
        {(taskGroupFilter === 'vuln' || taskGroupFilter === 'all') && (
          <Alert
            style={{ marginBottom: 16 }}
            type="info"
            showIcon
            message={t('vulnCountNarrowed', '漏洞计数口径已收紧')}
            description={t('vulnCountNarrowedDesc', '扫描任务的高/中/低危汇总会排除资产识别和全部待验证结果。待验证结果主要来自目标服务版本与漏洞知识库自动比对后的风险线索。若某环境尚未执行历史回填，旧任务短期内可能仍保留旧口径；详情页会按明细结果拆分展示正式结果、待验证和资产信息。')}
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
          locale={{ emptyText: t('noScanTasks', '暂无扫描任务') }}
        />
      </Card>

      {/* 创建任务弹窗 */}
      <Modal
        title={t('newVulnScanModalTitle', '新建漏洞扫描')}
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
            label={t('taskName', '任务名称')}
            tooltip={t('taskNameTooltip', '可选，不填会自动生成任务名')}
          >
            <Input placeholder={t('taskNamePlaceholder', '例如：核心系统周扫（可留空自动生成）')} />
          </Form.Item>

          <Form.Item
            name="scan_type"
            label={t('scanType', '扫描类型')}
            rules={[{ required: true, message: t('scanTypeRequired', '请选择扫描类型') }]}
            initialValue="web"
            tooltip={t('scanTypeTooltip', '新建扫描任务只提供网站漏洞和主机漏洞两类入口')}
          >
            <Radio.Group>
              <Radio.Button value="web">{t('webVuln', '网站漏洞')}</Radio.Button>
              <Radio.Button value="host-vuln">{t('hostVuln', '主机漏洞')}</Radio.Button>
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
                    <Form.Item label={t('recommendedMethod', '推荐方式')}>
                      <Alert
                        message={t('webScanOnlyLogin', 'Web 扫描只支持登录后扫描')}
                        description={t('webScanOnlyLoginDesc', '请填写业务入口 URL 和可登录测试账号。默认使用标准扫描；切到专项扫描时，会按你保留的类别同时约束模板和内置规则，并自动使用更深预算。')}
                        type="info"
                        showIcon
                      />
                    </Form.Item>

                    <Form.Item
                      name="target"
                      label={t('scanTarget', '扫描目标')}
                      rules={[
                        { required: true, message: t('scanTargetRequired', '请输入扫描 URL') },
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
                              return Promise.reject(new Error(t('urlFormatInvalid', 'URL 格式无效：{{invalid}}', { invalid })))
                            }

                            return Promise.resolve()
                          },
                        },
                      ]}
                      extra={t('scanTargetHint', '支持 http:// 或 https:// 的 URL 地址，每行一个')}
                    >
                      <Input.TextArea
                        rows={3}
                        placeholder={t('targetUrlPlaceholder', 'https://example.com\\nhttps://api.example.com')}
                      />
                    </Form.Item>

                    <Form.Item
                      name="web_scan_mode"
                      label={t('templateStrategy', '模板策略')}
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
                              ? t('specialScan', '专项扫描')
                              : t('standardScan', '标准扫描')
                          }
                          description={
                            getFieldValue('web_scan_mode') === 'special'
                              ? t('specialScanDesc', '会按你勾选的漏洞类别执行模板和内置规则，并自动使用更积极的验证预算。')
                              : t('standardScanDesc', '使用推荐的通用模板与规则集，适合首轮扫描和常规巡检。')
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
                          label={t('vulnCategory', '漏洞类别')}
                          tooltip={t('vulnCategoryTooltip', '专项扫描只会执行选中的漏洞类别，模板和对应内置规则会一起收敛')}
                          hidden={getFieldValue('web_scan_mode') !== 'special'}
                          extra={t('vulnCategoryExtra', '默认已全选。通常只需要取消你暂时不关注的类别。')}
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
                        <Form.Item name="username" label={t('loginAccount', '登录账号')} rules={[{ required: true, message: t('loginAccountRequired', '请输入账号') }]}>
                          <Input placeholder={t('loginAccountPlaceholder', '例如：default')} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="password" label={t('loginPassword', '登录密码')} rules={[{ required: true, message: t('loginPasswordRequired', '请输入密码') }]}>
                          <Input.Password placeholder={t('samplePlaceholder', '请输入密码')} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item
                      name="login_url"
                      label={t('loginUrlOptional', '登录接口 URL（可选）')}
                      extra={t('loginUrlHint', '不填则默认使用目标 URL 推断登录入口')}
                    >
                      <Input placeholder={t('loginUrlPlaceholder', '例如：http://target/api/login')} />
                    </Form.Item>

                    <WebScanPrecheckCard form={form} compact />

                    <Alert
                      message={t('usageTip', '使用建议')}
                      description={t('usageTipDesc', '常规场景只需：选择扫描类型 -> 填目标 -> 填登录账号密码 -> 创建任务。模板和认证流程都会自动按默认策略处理。')}
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
                    <Form.Item label={t('usageDescription', '用途说明')}>
                      <Alert
                        message={t('assetDiscoveryOnly', '资产发现只做端口和服务识别')}
                        description={t('assetDiscoveryOnlyDesc', '适合用于主机盘点、开放端口摸底和服务版本识别，不会执行漏洞检测。')}
                        type="info"
                        showIcon
                      />
                    </Form.Item>
                  )}

                  <Form.Item
                    name="target_type"
                    label={t('targetType', '目标类型')}
                    rules={[{ required: true, message: t('targetTypeRequired', '请选择目标类型') }]}
                    initialValue="ip_list"
                  >
                    <Radio.Group>
                      <Radio.Button value="ip_list">{t('ipList', 'IP 列表')}</Radio.Button>
                    </Radio.Group>
                  </Form.Item>

                  <Form.Item
                    name="target"
                    label={t('ipAddressField', 'IP 地址')}
                    rules={[{ required: true, message: t('ipAddressRequired', '请输入 IP 地址') }]}
                    extra={<Text type="secondary">{t('ipAddressHint', '支持多个 IP，逗号或换行分隔')}</Text>}
                  >
                    <Input.TextArea
                      rows={4}
                      placeholder={t('targetIpPlaceholder', '192.168.1.1\\n192.168.1.2\\n192.168.1.10-192.168.1.20')}
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
              <Form.Item label={t('presetReference', '预设参考')}>
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
              {t('cancel', '取消')}
            </Button>
            <Button type="primary" htmlType="submit">
              {t('createAndStartScan', '创建并开始扫描')}
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