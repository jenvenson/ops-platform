// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useState } from 'react'
import { Alert, Button, Card, Divider, Form, Input, InputNumber, Select, Space, Switch, Tabs, Tag, Typography, message } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { adminAPI, type AuditLogSetting, type FIMSSHSetting, type AssistantModelSetting, type SystemGeneralSetting } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'
import { useLocale } from '../../contexts/LocaleContext'

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
  const [modelForm] = Form.useForm()
  const [modelLoading, setModelLoading] = useState(false)
  const [modelSetting, setModelSetting] = useState<AssistantModelSetting | null>(null)
  const [generalLoading, setGeneralLoading] = useState(false)
  const { setLang } = useLocale()

  const loadGeneralSetting = async () => {
    setGeneralLoading(true)
    try {
      const setting = await adminAPI.getSystemGeneralSetting()
      form.setFieldsValue({
        siteName: setting.site_name,
        timezone: setting.timezone,
        language: setting.language,
      })
      if (setting.language) {
        setLang(setting.language)
      }
    } catch {
      message.error('加载通用配置失败')
    } finally {
      setGeneralLoading(false)
    }
  }

  const handleSaveGeneralSetting = async () => {
    try {
      const values = await form.validateFields()
      setGeneralLoading(true)
      const payload: SystemGeneralSetting = {
        site_name: values.siteName,
        timezone: values.timezone,
        language: values.language,
      }
      const result = await adminAPI.updateSystemGeneralSetting(payload)
      form.setFieldsValue({
        siteName: result.site_name,
        timezone: result.timezone,
        language: result.language,
      })
      document.title = result.site_name || '运维管理平台'
      if (result.language) {
        setLang(result.language)
      }
      message.success('通用设置已保存')
    } catch (error: unknown) {
      const msg =
        (error as { message?: string }).message ||
        (error as { errorFields?: Array<{ errors: string[] }> }).errorFields?.[0]?.errors?.[0] ||
        '保存通用配置失败'
      message.error(msg)
    } finally {
      setGeneralLoading(false)
    }
  }

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

  const loadModelSetting = async () => {
    setModelLoading(true)
    try {
      const setting = await adminAPI.getAssistantModelSetting()
      setModelSetting(setting)
      modelForm.setFieldsValue({
        provider: setting.provider || 'ollama',
        enabled: setting.enabled,
        api_key: '',
        base_url: setting.base_url || '',
        chat_model: setting.chat_model || '',
        embed_model: setting.embed_model || '',
        temperature: setting.temperature,
        timeout_sec: setting.timeout_sec,
      })
    } catch {
      message.error('加载 AI 模型配置失败')
    } finally {
      setModelLoading(false)
    }
  }

  useEffect(() => {
    void loadGeneralSetting()
    void loadFIMSetting()
    void loadAuditSetting()
    void loadModelSetting()
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

  const handleSaveModelSetting = async () => {
    try {
      const values = await modelForm.validateFields()
      setModelLoading(true)
      const payload = {
        provider: values.provider,
        enabled: !!values.enabled,
        api_key: values.api_key?.trim() || undefined,
        base_url: values.base_url?.trim() || '',
        chat_model: values.chat_model?.trim() || '',
        embed_model: values.embed_model?.trim() || '',
        temperature: values.temperature ?? 0.2,
        timeout_sec: values.timeout_sec ?? 20,
      }
      await adminAPI.updateAssistantModelSetting(payload)
      modelForm.setFieldsValue({ api_key: '' })
      message.success('AI 模型配置已保存，将在下一次请求时生效')
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '保存 AI 模型配置失败')
      }
    } finally {
      setModelLoading(false)
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
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={generalLoading} onClick={() => void handleSaveGeneralSetting()}>
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
            <Input placeholder="例如 192.168.1.100" />
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
      key: 'ai-model',
      label: 'AI 模型',
      children: (
        <Form
          form={modelForm}
          layout="vertical"
          initialValues={{
            provider: 'ollama',
            enabled: false,
            temperature: 0.2,
            timeout_sec: 20,
          }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="配置 AI 运维小助手使用的模型提供商。支持本地 Ollama 和第三方云模型。修改后保存将在下一次对话请求中生效，无需重启服务。"
          />
          <Form.Item label="启用 AI 助手" name="enabled" valuePropName="checked">
            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
          </Form.Item>
          <Form.Item
            label="模型提供商"
            name="provider"
            rules={[{ required: true, message: '请选择模型提供商' }]}
          >
            <Select
              showSearch
              options={[
                { label: 'Ollama (本地)', value: 'ollama' },
                { label: 'OpenAI', value: 'openai' },
                { label: 'DeepSeek', value: 'deepseek' },
                { label: '通义千问 (Qwen)', value: 'qwen' },
                { label: '智谱 GLM', value: 'zhipu' },
                { label: 'Kimi / Moonshot', value: 'moonshot' },
                { label: 'MiniMax', value: 'minimax' },
                { label: '豆包 (Doubao)', value: 'doubao' },
                { label: '百川 (Baichuan)', value: 'baichuan' },
                { label: '混元 (Hunyuan)', value: 'hunyuan' },
                { label: '文心一言 (Ernie)', value: 'ernie' },
                { label: '自定义中转', value: 'custom' },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, current) => prev.provider !== current.provider}
          >
            {({ getFieldValue }) => {
              const provider = getFieldValue('provider')
              if (provider && provider !== 'ollama') {
                return (
                  <Form.Item
                    label="API Key"
                    name="api_key"
                    extra={modelSetting?.api_key_configured ? <Tag color="green">已保存 API Key，留空表示继续使用当前 Key</Tag> : undefined}
                    rules={[
                      {
                        validator: async (_rule, value) => {
                          if (value?.trim() || modelSetting?.api_key_configured) {
                            return
                          }
                          throw new Error('请输入 API Key')
                        },
                      },
                    ]}
                  >
                    <Input.Password placeholder="sk-..." />
                  </Form.Item>
                )
              }
              return null
            }}
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, current) => prev.provider !== current.provider}
          >
            {({ getFieldValue }) => {
              const provider = getFieldValue('provider')
              if (!provider || provider === 'ollama') {
                return null
              }
              return (
                <Form.Item
                  label="API 地址"
                  name="base_url"
                  extra="选填。使用第三方中转地址时在此填写完整 API Base URL，例如 https://your-relay.example.com/v1"
                >
                  <Input placeholder="留空使用提供商默认地址" />
                </Form.Item>
              )
            }}
          </Form.Item>
          <Form.Item label="聊天模型" name="chat_model" extra="选填。留空使用该提供商的默认模型">
            <Input placeholder="例如 deepseek-chat 或 gpt-4o" />
          </Form.Item>
          <Form.Item label="嵌入模型" name="embed_model" extra="选填。用于知识库语义搜索，留空则降级为词法搜索">
            <Input placeholder="例如 text-embedding-3-small" />
          </Form.Item>
          <Form.Item label="温度" name="temperature" extra="控制回答的随机性，0-2 之间，越低越确定">
            <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="请求超时（秒）" name="timeout_sec" rules={[{ required: true, message: '请输入超时秒数' }]}>
            <InputNumber min={5} max={300} style={{ width: '100%' }} />
          </Form.Item>
          <Space size={12} style={{ marginBottom: 16 }}>
            <Tag color={modelSetting?.enabled ? 'green' : 'default'}>状态: {modelSetting?.enabled ? '已启用' : '已禁用'}</Tag>
            <Tag color={modelSetting?.api_key_configured ? 'green' : 'default'}>API Key: {modelSetting?.api_key_configured ? '已配置' : '未配置'}</Tag>
            <Tag>{modelSetting?.provider || 'ollama'}</Tag>
          </Space>
          <Form.Item>
            {canEdit() && <Button type="primary" icon={<SaveOutlined />} loading={modelLoading} onClick={() => void handleSaveModelSetting()}>
              保存 AI 模型配置
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
        <Tabs destroyInactiveTabPane={false} items={tabItems} />
      </Card>
    </div>
  )
}