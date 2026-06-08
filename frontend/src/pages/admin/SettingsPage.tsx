import { useEffect, useState } from 'react'
import { Alert, Button, Card, Divider, Form, Input, InputNumber, Select, Space, Switch, Tabs, Tag, Typography, message } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { adminAPI, type AuditLogSetting, type FIMSSHSetting } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'

const { Text } = Typography

export default function SettingsPage() {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [fimForm] = Form.useForm()
  const [fimLoading, setFIMLoading] = useState(false)
  const [fimTesting, setFIMTesting] = useState(false)
  const [fimSetting, setFIMSetting] = useState<FIMSSHSetting | null>(null)
  const [fimTestResult, setFIMTestResult] = useState<{ success: boolean; message: string; output?: string } | null>(null)
  const [auditForm] = Form.useForm()
  const [auditLoading, setAuditLoading] = useState(false)

  const handleSubmit = () => {
    setLoading(true)
    setTimeout(() => {
      message.success('保存成功')
      setLoading(false)
    }, 500)
  }

  const loadFIMSetting = async () => {
    setFIMLoading(true)
    try {
      const setting = await adminAPI.getFIMSSHSetting()
      setFIMSetting(setting)
      fimForm.setFieldsValue({
        auth_mode: setting.auth_mode,
        ssh_user: setting.ssh_user,
        timeout_sec: setting.timeout_sec,
        password: '',
        private_key: '',
        test_host: '',
        test_port: 22,
      })
    } catch {
      message.error('加载 FIM SSH 配置失败')
    } finally {
      setFIMLoading(false)
    }
  }

  const loadAuditSetting = async () => {
    setAuditLoading(true)
    try {
      const setting = await adminAPI.getAuditLogSetting()
      auditForm.setFieldsValue(setting)
    } catch {
      message.error('加载平台审计配置失败')
    } finally {
      setAuditLoading(false)
    }
  }

  useEffect(() => {
    void loadFIMSetting()
    void loadAuditSetting()
  }, [])

  const handleSaveAuditSetting = async () => {
    try {
      const values = await auditForm.validateFields()
      setAuditLoading(true)
      const payload: AuditLogSetting = {
        access_log_enabled: !!values.access_log_enabled,
        operation_log_enabled: !!values.operation_log_enabled,
        login_log_enabled: !!values.login_log_enabled,
      }
      await adminAPI.updateAuditLogSetting(payload)
      message.success('平台审计开关已保存')
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '保存平台审计配置失败')
      }
    } finally {
      setAuditLoading(false)
    }
  }

  const handleSaveFIMSetting = async () => {
    try {
      const values = await fimForm.validateFields()
      setFIMLoading(true)
      const payload = {
        auth_mode: values.auth_mode,
        ssh_user: values.ssh_user,
        timeout_sec: values.timeout_sec,
        password: values.password?.trim() || undefined,
        private_key: values.private_key?.trim() || undefined,
      }
      const setting = await adminAPI.updateFIMSSHSetting(payload)
      setFIMSetting(setting)
      fimForm.setFieldsValue({
        auth_mode: setting.auth_mode,
        ssh_user: setting.ssh_user,
        timeout_sec: setting.timeout_sec,
        password: '',
        private_key: '',
      })
      setFIMTestResult(null)
      message.success('FIM SSH 配置已保存')
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '保存 FIM SSH 配置失败')
      }
    } finally {
      setFIMLoading(false)
    }
  }

  const handleTestFIMConnection = async () => {
    try {
      const values = await fimForm.validateFields(['test_host', 'test_port'])
      setFIMTesting(true)
      setFIMTestResult(null)
      const result = await adminAPI.testFIMSSHConnection({
        host: values.test_host.trim(),
        port: values.test_port || 22,
      })
      setFIMTestResult(result)
      message[result.success ? 'success' : 'warning'](result.message)
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '测试 SSH 连接失败')
      }
    } finally {
      setFIMTesting(false)
    }
  }

  const tabItems = [
    {
      key: 'general',
      label: '通用设置',
      children: (
        <Form form={form} layout="vertical" initialValues={{ siteName: '运维管理平台', timezone: 'Asia/Shanghai', language: 'zh-CN' }}>
          <Form.Item label="系统名称" name="siteName">
            <Input placeholder="请输入系统名称" />
          </Form.Item>
          <Form.Item label="时区" name="timezone">
            <Select
              options={[
                { label: 'Asia/Shanghai (UTC+8)', value: 'Asia/Shanghai' },
                { label: 'UTC', value: 'UTC' },
              ]}
            />
          </Form.Item>
          <Form.Item label="语言" name="language">
            <Select
              options={[
                { label: '简体中文', value: 'zh-CN' },
                { label: 'English', value: 'en-US' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSubmit}>
              保存设置
            </Button>}
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'security',
      label: '安全设置',
      children: (
        <Form layout="vertical" initialValues={{ tokenExpiry: 24, loginRetry: 5 }}>
          <Form.Item label="Token 有效期（小时）" name="tokenExpiry">
            <Input type="number" min={1} max={72} />
          </Form.Item>
          <Form.Item label="登录失败锁定次数" name="loginRetry">
            <Input type="number" min={3} max={10} />
          </Form.Item>
          <Form.Item label="强制启用双因素认证" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSubmit}>
              保存设置
            </Button>}
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'audit',
      label: '平台审计',
      children: (
        <Form
          form={auditForm}
          layout="vertical"
          initialValues={{
            access_log_enabled: true,
            operation_log_enabled: true,
            login_log_enabled: true,
          }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="平台审计开关会影响后端是否继续写入访问日志、操作审计和登录日志。关闭后，已有历史日志不会删除，只是不再新增。"
          />
          <Card size="small" title="日志采集开关" loading={auditLoading}>
            <Space direction="vertical" size={20} style={{ width: '100%' }}>
              <Form.Item label="访问日志" name="access_log_enabled" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
              <Form.Item label="操作审计" name="operation_log_enabled" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
              <Form.Item label="登录日志" name="login_log_enabled" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
            </Space>
          </Card>
          <Form.Item style={{ marginTop: 16, marginBottom: 0 }}>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={auditLoading} onClick={() => void handleSaveAuditSetting()}>
              保存平台审计配置
            </Button>}
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'fim',
      label: 'FIM SSH',
      children: (
        <Form
          form={fimForm}
          layout="vertical"
          initialValues={{ auth_mode: 'password', timeout_sec: 15 }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="文件完整性巡检通过后端发起 SSH 连接。私钥方式只需要在平台保存私钥，目标主机上需提前部署对应公钥。"
          />
          <Form.Item label="SSH 用户" name="ssh_user" rules={[{ required: true, message: '请输入 SSH 用户' }]}>
            <Input placeholder="例如 root 或 ubuntu" />
          </Form.Item>
          <Form.Item label="认证方式" name="auth_mode" rules={[{ required: true, message: '请选择认证方式' }]}>
            <Select
              options={[
                { label: '账号密码', value: 'password' },
                { label: '公私钥', value: 'private_key' },
              ]}
            />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, current) => prev.auth_mode !== current.auth_mode}>
            {({ getFieldValue }) => {
              const authMode = getFieldValue('auth_mode')
              if (authMode === 'private_key') {
                return (
                  <Form.Item
                    label="私钥内容"
                    name="private_key"
                    extra={fimSetting?.private_key_configured ? <Tag color="green">已保存私钥，留空表示继续使用当前私钥</Tag> : '支持 OpenSSH 私钥内容'}
                    rules={[
                      {
                        validator: async (_rule, value) => {
                          if (value?.trim() || fimSetting?.private_key_configured) {
                            return
                          }
                          throw new Error('请输入私钥内容')
                        },
                      },
                    ]}
                  >
                    <Input.TextArea rows={8} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
                  </Form.Item>
                )
              }
              return (
                <Form.Item
                  label="SSH 密码"
                  name="password"
                  extra={fimSetting?.password_configured ? <Tag color="green">已保存密码，留空表示继续使用当前密码</Tag> : undefined}
                  rules={[
                    {
                      validator: async (_rule, value) => {
                        if (value?.trim() || fimSetting?.password_configured) {
                          return
                        }
                        throw new Error('请输入 SSH 密码')
                      },
                    },
                  ]}
                >
                  <Input.Password placeholder="请输入 SSH 密码" />
                </Form.Item>
              )
            }}
          </Form.Item>
          <Form.Item label="SSH 超时（秒）" name="timeout_sec" rules={[{ required: true, message: '请输入超时秒数' }]}>
            <InputNumber min={3} max={120} style={{ width: '100%' }} />
          </Form.Item>
          <Space size={12} style={{ marginBottom: 16 }}>
            <Tag color={fimSetting?.password_configured ? 'green' : 'default'}>密码: {fimSetting?.password_configured ? '已配置' : '未配置'}</Tag>
            <Tag color={fimSetting?.private_key_configured ? 'green' : 'default'}>私钥: {fimSetting?.private_key_configured ? '已配置' : '未配置'}</Tag>
          </Space>
          <Divider style={{ marginTop: 8 }}>连通性测试</Divider>
          <Form.Item label="测试主机" name="test_host" rules={[{ required: true, message: '请输入测试主机 IP 或域名' }]}>
            <Input placeholder="例如 10.99.99.187" />
          </Form.Item>
          <Form.Item label="测试端口" name="test_port" initialValue={22}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Space size={12} style={{ marginBottom: 16 }}>
            {canEdit() && <Button onClick={() => void handleTestFIMConnection()} loading={fimTesting}>
              测试 SSH 连接
            </Button>}
            {fimTestResult && (
              <Text type={fimTestResult.success ? 'success' : 'danger'}>
                {fimTestResult.message}
                {fimTestResult.output ? ` | 远端主机: ${fimTestResult.output}` : ''}
              </Text>
            )}
          </Space>
          <Form.Item>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={fimLoading} onClick={() => void handleSaveFIMSetting()}>
              保存 FIM SSH 配置
            </Button>}
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'notification',
      label: '通知设置',
      children: (
        <Form layout="vertical" initialValues={{ emailEnabled: true, dingTalkEnabled: false }}>
          <Divider>邮件通知</Divider>
          <Form.Item label="启用邮件通知" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="SMTP 服务器">
            <Input placeholder="smtp.example.com" />
          </Form.Item>
          <Form.Item label="端口">
            <Input type="number" placeholder="465" />
          </Form.Item>
          <Form.Item label="发件邮箱">
            <Input placeholder="noreply@example.com" />
          </Form.Item>
          <Divider>钉钉通知</Divider>
          <Form.Item label="启用钉钉通知" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="钉钉 Access Token">
            <Input.Password placeholder="请输入 Access Token" />
          </Form.Item>
          <Form.Item>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSubmit}>
              保存设置
            </Button>}
          </Form.Item>
        </Form>
      ),
    },
  ]

  return (
    <div>
      <Card>
        <Tabs items={tabItems} />
      </Card>
    </div>
  )
}
