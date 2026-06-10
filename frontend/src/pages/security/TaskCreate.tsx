// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState } from 'react'
import { Card, Steps, Form, Input, Button, Space, Typography, Radio, Alert, Tag, Row, Col, Checkbox } from 'antd'
import {
  AimOutlined,
  RadarChartOutlined,
  GlobalOutlined,
  CheckCircleOutlined,
  ArrowRightOutlined,
  ArrowLeftOutlined,
  ScanOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { securityAPI, CreateTaskRequest } from '../../api/security'
import { useNavigate } from 'react-router-dom'
import WebScanPrecheckCard from './WebScanPrecheck'

const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

// SOC 风格深色主题配置
const theme = {
  bg: '#0a0e17',
  card: '#111827',
  cardHover: '#1a2234',
  border: '#1e293b',
  primary: '#3b82f6',
  accent: '#06b6d4',
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
  text: '#e2e8f0',
  textSecondary: '#94a3b8',
}

interface TaskFormData {
  name: string
  target_type: 'ip_list' | 'url'
  target: string
  scan_type: 'port' | 'host-vuln' | 'web'
  web_scan_mode?: 'standard' | 'special'
  web_scan_options?: string[]
  username?: string
  password?: string
  login_url?: string
  precheck_access?: 'unknown' | 'public' | 'login-required'
  precheck_auth_complexity?: 'unknown' | 'standard' | 'custom' | 'captcha'
  precheck_session_carrier?: 'unknown' | 'cookie' | 'header' | 'mixed'
}

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
    desc: '适合首轮排查，使用推荐的 Web 通用模板与规则集，覆盖面更广。',
    tone: 'info' as const,
  },
  {
    value: 'special',
    title: '专项扫描',
    desc: '按选中类别做更积极的专项验证，默认使用更深预算，适合重点目标复测和深挖。',
    tone: 'warning' as const,
  },
]

const scanTypeOptions = [
  { value: 'host-vuln', label: '主机漏洞扫描', desc: '检测 SSH/数据库/Redis 等服务的安全漏洞', icon: <WarningOutlined /> },
  { value: 'web', label: 'Web漏洞扫描', desc: '使用 Nuclei 检测 Web 应用漏洞', icon: <GlobalOutlined /> },
]

const vulnTypes = webScanOptionConfigs.map(option => option.value)

export default function TaskCreate() {
  const navigate = useNavigate()
  const [current, setCurrent] = useState(0)
  const [form] = Form.useForm<TaskFormData>()
  const [loading, setLoading] = useState(false)
  const [taskId, setTaskId] = useState<number | null>(null)

  const steps = [
    { title: '目标', icon: <AimOutlined /> },
    { title: '扫描类型', icon: <RadarChartOutlined /> },
    { title: '配置', icon: <ScanOutlined /> },
    { title: '确认', icon: <CheckCircleOutlined /> },
  ]

  const handleNext = async () => {
    try {
      const values = await form.validateFields()
      const normalizedTargetType: TaskFormData['target_type'] =
        values.scan_type === 'web'
          ? 'url'
          : (values.target_type === 'url' ? 'ip_list' : (values.target_type || 'ip_list'))

      if (current < steps.length - 1) {
        if (current === 1 && normalizedTargetType !== values.target_type) {
          form.setFieldValue('target_type', normalizedTargetType)
        }
        setCurrent(current + 1)
      } else {
        // 创建任务
        setLoading(true)
        try {
          let authFlow: Record<string, unknown> | undefined
          if (values.scan_type === 'web') {
            const entryTarget = values.target
              .split(/[,\n]/)
              .map((item: string) => item.trim())
              .find((item: string) => item.length > 0)

            if (!entryTarget) {
              form.setFields([{ name: 'target', errors: ['请先输入扫描目标 URL'] }])
              setCurrent(2)
              return
            }

            const generated = await securityAPI.generateAuthFlow({
              preset: 'auto',
              target_url: entryTarget,
              login_url: values.login_url?.trim() || undefined,
            })
            authFlow = generated.auth_flow
          }
          const payload: CreateTaskRequest = {
            name: values.name,
            target: values.target,
            scan_type: values.scan_type,
            target_type: normalizedTargetType,
            web_scan_options: values.scan_type === 'web' && values.web_scan_mode === 'special'
              ? values.web_scan_options?.join(',')
              : undefined,
            web_scan_profile: values.scan_type === 'web' && values.web_scan_mode === 'special'
              ? 'deep'
              : (values.scan_type === 'web' ? 'standard' : undefined),
            discovery_mode: values.scan_type === 'web' ? 'browser' : undefined,
            auth_mode: values.scan_type === 'web' ? 'advanced' : undefined,
            username: values.scan_type === 'web' ? values.username?.trim() : undefined,
            password: values.scan_type === 'web' ? values.password : undefined,
            auth_flow: values.scan_type === 'web' ? authFlow : undefined,
            login_url: values.scan_type === 'web' ? values.login_url?.trim() : undefined,
          }
          const task = await securityAPI.createTask(payload)
          setTaskId(task.id)
          setCurrent(current + 1)
        } catch (error) {
          console.error('创建任务失败:', error)
        } finally {
          setLoading(false)
        }
      }
    } catch (error) {
      console.error('验证失败:', error)
    }
  }

  const handlePrev = () => {
    if (current > 0) {
      setCurrent(current - 1)
    }
  }

  const renderStepContent = () => {
    switch (current) {
      case 0:
        return (
          <div>
            <Title level={4} style={{ color: theme.text, marginBottom: 24 }}>
              <AimOutlined style={{ marginRight: 8 }} />
              输入扫描目标
            </Title>

            <Form.Item
              name="name"
              label="任务名称"
              rules={[{ required: true, message: '请输入任务名称' }]}
            >
              <Input placeholder="例如：生产服务器安全扫描" />
            </Form.Item>

            <Form.Item
              name="target_type"
              label="目标类型"
              rules={[{ required: true, message: '请选择目标类型' }]}
              initialValue="ip_list"
            >
              <Radio.Group buttonStyle="solid">
                <Radio.Button value="ip_list">IP 列表</Radio.Button>
                <Radio.Button value="url">URL 列表</Radio.Button>
              </Radio.Group>
            </Form.Item>

            <Form.Item
              name="target"
              label="目标地址"
              rules={[{ required: true, message: '请输入目标地址' }]}
            >
              <TextArea
                rows={4}
                placeholder={
                  form.getFieldValue('target_type') === 'url'
                    ? 'https://example.com\nhttps://api.example.com'
                    : '192.168.1.1\n192.168.1.2\n192.168.1.10-192.168.1.20'
                }
              />
            </Form.Item>

            <Alert
              message="提示"
              description={
                form.getFieldValue('target_type') === 'url'
                  ? '登录后 Web 扫描只支持 URL 列表；每行一个 URL，必须包含 http:// 或 https://'
                  : '每行一个 IP 或 IP 范围（如 192.168.1.1-10）'
              }
              type="info"
              showIcon
            />

            {form.getFieldValue('target_type') === 'url' && (
              <div style={{ marginTop: 16 }}>
                <WebScanPrecheckCard form={form} />
              </div>
            )}
          </div>
        )

      case 1:
        return (
          <div>
            <Title level={4} style={{ color: theme.text, marginBottom: 24 }}>
              <RadarChartOutlined style={{ marginRight: 8 }} />
              选择扫描类型
            </Title>

            <Form.Item
              name="scan_type"
              rules={[{ required: true, message: '请选择扫描类型' }]}
            >
              <Radio.Group style={{ width: '100%' }}>
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  {scanTypeOptions.map((option) => (
                    <Radio key={option.value} value={option.value} style={{ width: '100%' }}>
                      <Card
                        size="small"
                        style={{
                          background: form.getFieldValue('scan_type') === option.value ? theme.cardHover : theme.card,
                          borderColor: form.getFieldValue('scan_type') === option.value ? theme.primary : theme.border,
                          marginTop: 8,
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                          <div style={{ fontSize: 24, color: theme.primary }}>{option.icon}</div>
                          <div>
                            <div style={{ fontWeight: 600, color: theme.text }}>{option.label}</div>
                            <div style={{ fontSize: 12, color: theme.textSecondary }}>{option.desc}</div>
                          </div>
                        </div>
                      </Card>
                    </Radio>
                  ))}
                </Space>
              </Radio.Group>
            </Form.Item>
          </div>
        )

      case 2:
        const scanType = form.getFieldValue('scan_type')
        return (
          <div>
            <Title level={4} style={{ color: theme.text, marginBottom: 24 }}>
              <ScanOutlined style={{ marginRight: 8 }} />
              扫描配置
            </Title>

            {scanType === 'web' ? (
              <>
                <Alert
                  message="Web 漏扫建议"
                  description="优先直接录入业务 URL。默认会先自动登录并发现页面/API，再对发现到的入口执行 Nuclei 模板探测。"
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form.Item
                  name="web_scan_mode"
                  label="扫描策略"
                  initialValue="standard"
                >
                  <Radio.Group style={{ width: '100%' }}>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      {webScanModeConfigs.map((option) => (
                        <Radio key={option.value} value={option.value} style={{ width: '100%' }}>
                          <Card
                            size="small"
                            style={{
                              background: form.getFieldValue('web_scan_mode') === option.value ? theme.cardHover : theme.card,
                              borderColor: form.getFieldValue('web_scan_mode') === option.value ? theme.primary : theme.border,
                              marginTop: 8,
                            }}
                          >
                            <div style={{ fontWeight: 600, color: theme.text, marginBottom: 4 }}>{option.title}</div>
                            <div style={{ fontSize: 12, color: theme.textSecondary }}>{option.desc}</div>
                          </Card>
                        </Radio>
                      ))}
                    </Space>
                  </Radio.Group>
                </Form.Item>

                <Alert
                  message={
                    form.getFieldValue('web_scan_mode') === 'special'
                      ? '专项 Web 扫描'
                      : '标准 Web 扫描'
                  }
                  description={
                    form.getFieldValue('web_scan_mode') === 'special'
                      ? '会按你勾选的漏洞类别执行模板和内置规则，并自动使用更积极的验证预算，适合专项复测和重点目标深挖。'
                      : '默认使用推荐的 Web 通用模板与规则集，适合首轮扫描和常规普查。'
                  }
                  type={form.getFieldValue('web_scan_mode') === 'standard' ? 'info' : 'warning'}
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Alert
                  message="自动发现后扫描"
                  description="当前创建入口默认使用自动发现策略，更适合首轮排查 SPA、登录后系统和真实业务 API。"
                  type="success"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form.Item
                  noStyle
                  shouldUpdate={(prevValues, currentValues) => prevValues.web_scan_mode !== currentValues.web_scan_mode}
                >
                  {({ getFieldValue }) => (
                    <Form.Item
                      name="web_scan_options"
                      label="漏洞类别"
                      initialValue={vulnTypes}
                      hidden={getFieldValue('web_scan_mode') !== 'special'}
                      extra="默认已全选。专项扫描会只执行你保留的类别，并使用更深的验证预算。"
                    >
                      <Checkbox.Group style={{ width: '100%' }}>
                        <Row gutter={[12, 12]}>
                          {webScanOptionConfigs.map((option) => (
                            <Col span={8} key={option.value}>
                              <Card size="small" style={{ background: theme.card, borderColor: theme.border }}>
                                <Checkbox value={option.value}>{option.label}</Checkbox>
                              </Card>
                            </Col>
                          ))}
                        </Row>
                      </Checkbox.Group>
                    </Form.Item>
                  )}
                </Form.Item>

                <Title level={5} style={{ color: theme.text, marginBottom: 8 }}>认证配置</Title>
                <Paragraph style={{ color: theme.textSecondary, marginBottom: 16 }}>
                  Web 扫描已调整为只做登录后扫描。请提供测试账号、密码和可选登录接口，认证流程会自动生成。
                </Paragraph>

                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item
                      name="username"
                      label="登录账号"
                      rules={[{ required: true, message: '请输入账号' }]}
                    >
                      <Input placeholder="例如：default" />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      name="password"
                      label="登录密码"
                      rules={[{ required: true, message: '请输入密码' }]}
                    >
                      <Input.Password placeholder="请输入密码" />
                    </Form.Item>
                  </Col>
                </Row>

                <Form.Item
                  name="login_url"
                  label="登录接口 URL（可选）"
                  extra="不填则默认使用目标 URL 推断登录入口。"
                >
                  <Input placeholder="例如：http://target/api/login" />
                </Form.Item>

                <Alert
                  message="登录态处理"
                  description="系统会根据当前目标自动生成认证流程；匿名扫描已禁用。"
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <WebScanPrecheckCard form={form} />

                <Card size="small" style={{ background: theme.card, borderColor: theme.border }}>
                  <Space direction="vertical" size="small" style={{ width: '100%' }}>
                    <Text style={{ color: theme.text }}>推荐操作顺序</Text>
                    <Text type="secondary">1. 先填写业务入口 URL 和可登录测试账号</Text>
                    <Text type="secondary">2. 首轮排查选标准扫描，重点目标复测或深挖时切到专项扫描</Text>
                    <Text type="secondary">3. 专项扫描默认全选全部类别，你再按需要删减</Text>
                    <Text type="secondary">4. 登录接口可选，不填时系统会自动推断</Text>
                  </Space>
                </Card>
              </>
            ) : scanType === 'host-vuln' ? (
              <Alert
                message="主机漏洞扫描配置"
                description="主机漏洞扫描将自动检测 SSH、数据库、Redis 等服务的安全漏洞，无需额外配置"
                type="success"
                showIcon
              />
            ) : (
              <Alert
                message="扫描配置"
                description="请选择网站漏洞或主机漏洞扫描类型后继续配置。"
                type="info"
                showIcon
              />
            )}
          </div>
        )

      case 3:
        const values = form.getFieldsValue()
        const displayTargetType = values.scan_type === 'web' ? 'url' : (values.target_type || 'ip_list')
        return (
          <div>
            <Title level={4} style={{ color: theme.text, marginBottom: 24 }}>
              <CheckCircleOutlined style={{ marginRight: 8 }} />
              确认扫描任务
            </Title>

            <Card style={{ background: theme.card, borderColor: theme.border, marginBottom: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Text style={{ color: theme.textSecondary }}>任务名称</Text>
                  <div style={{ color: theme.text, fontWeight: 600 }}>{values.name}</div>
                </Col>
                <Col span={12}>
                  <Text style={{ color: theme.textSecondary }}>扫描类型</Text>
                  <div style={{ color: theme.primary, fontWeight: 600 }}>
                    {scanTypeOptions.find(s => s.value === values.scan_type)?.label}
                  </div>
                </Col>
                <Col span={12}>
                  <Text style={{ color: theme.textSecondary }}>目标类型</Text>
                  <div style={{ color: theme.text }}>{displayTargetType?.toUpperCase()}</div>
                </Col>
                <Col span={12}>
                  <Text style={{ color: theme.textSecondary }}>目标数量</Text>
                  <div style={{ color: theme.text }}>
                    {values.target?.split('\n').filter(Boolean).length || 1} 个目标
                  </div>
                </Col>
              </Row>
            </Card>

            {values.scan_type === 'web' && (
              <div style={{ marginBottom: 16 }}>
                <Text style={{ color: theme.textSecondary }}>Web 扫描方式：</Text>
                <div style={{ marginTop: 8 }}>
                  <Tag color="cyan" style={{ marginBottom: 4 }}>自动发现后扫描</Tag>
                  {values.web_scan_mode === 'special' ? (
                    <>
                      <Tag color="orange" style={{ marginBottom: 4 }}>专项扫描</Tag>
                      {(values.web_scan_options || []).map((type: string) => {
                        const option = webScanOptionConfigs.find(item => item.value === type)
                        return (
                          <Tag key={type} color="blue" style={{ marginBottom: 4 }}>
                            {option?.label || type}
                          </Tag>
                        )
                      })}
                    </>
                  ) : (
                    <Tag color="green" style={{ marginBottom: 4 }}>标准扫描（推荐模板集）</Tag>
                  )}
                </div>
              </div>
            )}

            {values.scan_type === 'web' && (
              <div style={{ marginBottom: 16 }}>
                <Text style={{ color: theme.textSecondary }}>认证方式：</Text>
                <div style={{ marginTop: 8 }}>
                  <Tag color="blue">登录后扫描</Tag>
                  <Tag color="cyan">自动生成认证流程</Tag>
                </div>
              </div>
            )}

            {(values.scan_type === 'host-vuln') && (
              <Alert
                message="漏洞检测范围"
                description={
                  values.scan_type === 'host-vuln'
                    ? '将检测 SSH 弱口令、数据库未授权访问、Redis 未授权等主机层面漏洞'
                    : '将检测 Web 漏洞 + 主机层面漏洞（SSH/数据库/Redis 等）'
                }
                type="info"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}

            <Alert
              message="开始扫描后，任务将立即执行"
              description="扫描过程可能需要较长时间，请耐心等待。您可以在任务列表中查看进度。"
              type="warning"
              showIcon
            />
          </div>
        )

      case 4:
        return (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <div style={{ fontSize: 64, color: theme.low, marginBottom: 24 }}>
              <CheckCircleOutlined />
            </div>
            <Title level={3} style={{ color: theme.text, marginBottom: 16 }}>
              任务创建成功！
            </Title>
            <Paragraph style={{ color: theme.textSecondary, marginBottom: 32 }}>
              扫描任务已提交，正在后台执行。您可以前往任务列表查看进度。
            </Paragraph>
            <Space>
              <Button type="primary" onClick={() => navigate(`/security/tasks/${taskId}`)}>
                查看任务详情
              </Button>
              <Button onClick={() => navigate('/security/tasks')}>
                返回任务列表
              </Button>
            </Space>
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div style={{ background: theme.bg, minHeight: '100vh', padding: '24px' }}>
      <Card
        style={{
          background: theme.card,
          borderColor: theme.border,
          maxWidth: 800,
          margin: '0 auto',
        }}
      >
        <Steps
          current={current}
          items={steps}
          style={{ marginBottom: 32 }}
        />

        <Form
          form={form}
          layout="vertical"
          initialValues={{
            target_type: 'ip_list',
            scan_type: 'web',
            web_scan_mode: 'standard',
            web_scan_options: vulnTypes,
          }}
        >
          {renderStepContent()}
        </Form>

        {current < 4 && (
          <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 24, paddingTop: 24, borderTop: `1px solid ${theme.border}` }}>
            <Button
              onClick={handlePrev}
              disabled={current === 0}
              icon={<ArrowLeftOutlined />}
            >
              上一步
            </Button>
            <Button
              type="primary"
              onClick={handleNext}
              loading={loading}
              icon={current === steps.length - 1 ? <ScanOutlined /> : <ArrowRightOutlined />}
            >
              {current === steps.length - 1 ? '开始扫描' : '下一步'}
            </Button>
          </div>
        )}
      </Card>
    </div>
  )
}