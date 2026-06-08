import React, { useState, useEffect, useCallback } from 'react'
import {
  Table, Tag, Button, Modal, Form, Input, Select, Space,
  message, Popconfirm, Switch, Card, Row, Col, Typography, Tooltip,
  Divider, Alert,
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined,
  StarOutlined, StarFilled, CopyOutlined,
  MessageOutlined, ApiOutlined, MailOutlined,
} from '@ant-design/icons'
import { alertAPI, AlertTemplate } from '../../api/alert'
import { canEdit } from '../../utils/menuAccess'

const { TextArea } = Input
const { Text, Paragraph } = Typography

// 可用模板变量说明
const templateVars = [
  { var: '{{.RuleName}}', desc: '规则名称', example: 'DiskUsageHigh' },
  { var: '{{.Content}}', desc: '告警内容（触发=原始描述，恢复=恢复内容+当前值）', example: '磁盘使用率达到 95%' },
  { var: '{{.CurrentValue}}', desc: '当前指标值（仅恢复告警时有效）', example: '3.72%' },
  { var: '{{.Source}}', desc: '告警来源', example: '10.99.99.100:9100' },
  { var: '{{.Severity}}', desc: '级别(英文)', example: 'critical' },
  { var: '{{.SeverityLabel}}', desc: '级别(中文)', example: '严重' },
  { var: '{{.Status}}', desc: '状态(英文)', example: 'firing' },
  { var: '{{.StatusLabel}}', desc: '状态(中文)', example: '告警中' },
  { var: '{{.Category}}', desc: '分类(英文)', example: 'disk' },
  { var: '{{.CategoryLabel}}', desc: '分类(中文)', example: '磁盘' },
  { var: '{{.Time}}', desc: '触发/恢复时间', example: '2026-02-09 12:00:00' },
  { var: '{{.Emoji}}', desc: '级别Emoji', example: '🔴' },
]

const typeConfig: Record<string, { icon: React.ReactNode; label: string; color: string }> = {
  dingtalk: { icon: <MessageOutlined />, label: '钉钉', color: '#1890ff' },
  wechat:   { icon: <ApiOutlined />,     label: '企微', color: '#52c41a' },
  email:    { icon: <MailOutlined />,     label: '邮件', color: '#faad14' },
}

const sceneConfig: Record<string, { label: string; color: string }> = {
  firing:   { label: '告警触发', color: 'red' },
  resolved: { label: '告警恢复', color: 'green' },
}

export default function AlertTemplatesPage() {
  const [templates, setTemplates] = useState<AlertTemplate[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [editingTpl, setEditingTpl] = useState<AlertTemplate | undefined>()
  const [previewResult, setPreviewResult] = useState<{ title: string; content: string } | null>(null)
  const [previewType, setPreviewType] = useState('dingtalk')
  const [form] = Form.useForm()

  const fetchTemplates = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await alertAPI.templates.list()
      setTemplates(resp.data || [])
    } catch { message.error('加载失败') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchTemplates() }, [fetchTemplates])

  const openModal = (tpl?: AlertTemplate) => {
    setEditingTpl(tpl)
    if (tpl) {
      form.setFieldsValue(tpl)
    } else {
      form.resetFields()
      form.setFieldsValue({ type: 'dingtalk', scene: 'firing', enabled: true })
    }
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      if (editingTpl) {
        await alertAPI.templates.update(editingTpl.id, values)
        message.success('更新成功')
      } else {
        await alertAPI.templates.create(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchTemplates()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      if (error?.response?.data?.error) {
        message.error(error.response.data.error)
      } else {
        message.error('操作失败')
      }
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await alertAPI.templates.delete(id)
      message.success('删除成功')
      fetchTemplates()
    } catch { message.error('删除失败') }
  }

  const handleSetDefault = async (id: number) => {
    try {
      await alertAPI.templates.setDefault(id)
      message.success('已设为默认模板')
      fetchTemplates()
    } catch { message.error('操作失败') }
  }

  const handleCopy = (tpl: AlertTemplate) => {
    form.resetFields()
    form.setFieldsValue({
      ...tpl,
      name: tpl.name + '_副本',
      is_default: false,
    })
    setEditingTpl(undefined)
    setModalOpen(true)
  }

  const handlePreview = async () => {
    try {
      const values = form.getFieldsValue()
      const resp = await alertAPI.templates.preview({
        title_tpl: values.title_tpl || '',
        content_tpl: values.content_tpl || '',
        type: values.type || 'dingtalk',
      })
      setPreviewResult({ title: resp.title, content: resp.content })
      setPreviewType(values.type || 'dingtalk')
      setPreviewOpen(true)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      if (error?.response?.data?.error) {
        message.error(error.response.data.error)
      } else {
        message.error('预览失败')
      }
    }
  }

  const columns = [
    {
      title: '渠道类型', dataIndex: 'type', key: 'type', width: 100,
      render: (v: string) => {
        const cfg = typeConfig[v]
        return cfg ? <Space><span style={{ color: cfg.color }}>{cfg.icon}</span>{cfg.label}</Space> : v
      },
      filters: [
        { text: '钉钉', value: 'dingtalk' },
        { text: '企微', value: 'wechat' },
        { text: '邮件', value: 'email' },
      ],
      onFilter: (value: unknown, record: AlertTemplate) => record.type === value,
    },
    {
      title: '场景', dataIndex: 'scene', key: 'scene', width: 110,
      render: (v: string) => {
        const cfg = sceneConfig[v]
        return cfg ? <Tag color={cfg.color}>{cfg.label}</Tag> : v
      },
      filters: [
        { text: '告警触发', value: 'firing' },
        { text: '告警恢复', value: 'resolved' },
      ],
      onFilter: (value: unknown, record: AlertTemplate) => record.scene === value,
    },
    {
      title: '模板名称', dataIndex: 'name', key: 'name', width: 200,
      render: (v: string, r: AlertTemplate) => (
        <Space>
          {r.is_default && <StarFilled style={{ color: '#faad14' }} />}
          {v}
        </Space>
      ),
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '默认', dataIndex: 'is_default', key: 'is_default', width: 70,
      render: (v: boolean) => v ? <Tag color="gold">默认</Tag> : <Tag>否</Tag>,
    },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 70,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag>,
    },
    {
      title: '操作', key: 'action', width: 260,
      render: (_: unknown, r: AlertTemplate) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(r)}>编辑</Button>}
          {canEdit() && <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => handleCopy(r)}>复制</Button>}
          {!r.is_default && (
            <Tooltip title="设为默认模板">
              <Button type="link" size="small" icon={<StarOutlined />} onClick={() => handleSetDefault(r.id)}>默认</Button>
            </Tooltip>
          )}
          {canEdit() && <Popconfirm title="确定删除此模板？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        {canEdit() && <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新建模板</Button>}
        <Text type="secondary">告警通知发送时会自动使用对应渠道和场景的默认模板</Text>
      </div>

      <Table
        columns={columns}
        dataSource={templates}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20, showTotal: t => `共 ${t} 条` }}
      />

      {/* 编辑/新建模板弹窗 */}
      <Modal
        title={editingTpl ? '编辑告警模板' : '新建告警模板'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={800}
        footer={[
          <Button key="cancel" onClick={() => setModalOpen(false)}>取消</Button>,
          <Button key="preview" icon={<EyeOutlined />} onClick={handlePreview}>预览</Button>,
          <Button key="save" type="primary" onClick={handleSave}>保存</Button>,
        ]}
      >
        <Row gutter={24}>
          <Col span={16}>
            <Form form={form} layout="vertical">
              <Row gutter={16}>
                <Col span={8}>
                  <Form.Item name="type" label="渠道类型" rules={[{ required: true }]}>
                    <Select options={[
                      { label: '钉钉', value: 'dingtalk' },
                      { label: '企业微信', value: 'wechat' },
                      { label: '邮件', value: 'email' },
                    ]} />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="scene" label="使用场景" rules={[{ required: true }]}>
                    <Select options={[
                      { label: '告警触发', value: 'firing' },
                      { label: '告警恢复', value: 'resolved' },
                    ]} />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
                    <Switch />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
                <Input placeholder="如：钉钉-磁盘告警模板" />
              </Form.Item>

              <Form.Item name="description" label="描述">
                <Input placeholder="模板用途描述" />
              </Form.Item>

              <Form.Item name="title_tpl" label="标题模板">
                <Input placeholder="如：{{.Emoji}} 【告警】{{.RuleName}}" />
              </Form.Item>

              <Form.Item name="content_tpl" label="内容模板" rules={[{ required: true, message: '请输入内容模板' }]}>
                <TextArea
                  rows={12}
                  placeholder={'### {{.Emoji}} 【告警】{{.RuleName}}\n\n> **规则名称**：{{.RuleName}}\n\n> **告警内容**：{{.Content}}\n\n> **来源**：{{.Source}}\n\n> **级别**：{{.SeverityLabel}}\n\n> **状态**：{{.StatusLabel}}\n\n> **触发时间**：{{.Time}}'}
                  style={{ fontFamily: 'Monaco, Menlo, Consolas, monospace', fontSize: 13 }}
                />
              </Form.Item>
            </Form>
          </Col>

          <Col span={8}>
            <Card size="small" title="可用变量" style={{ marginTop: 30 }}>
              <div style={{ maxHeight: 420, overflowY: 'auto' }}>
                {templateVars.map(v => (
                  <div key={v.var} style={{ marginBottom: 8, fontSize: 12 }}>
                    <Text code copyable={{ text: v.var }} style={{ fontSize: 12 }}>{v.var}</Text>
                    <br />
                    <Text type="secondary" style={{ fontSize: 11 }}>{v.desc}：{v.example}</Text>
                  </div>
                ))}
              </div>
            </Card>
          </Col>
        </Row>
      </Modal>

      {/* 预览弹窗 */}
      <Modal
        title="模板预览"
        open={previewOpen}
        onCancel={() => setPreviewOpen(false)}
        footer={<Button onClick={() => setPreviewOpen(false)}>关闭</Button>}
        width={650}
      >
        {previewResult && (
          <div>
            <Alert
              message="以下为使用示例数据渲染的预览效果"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            {previewResult.title && (
              <div style={{ marginBottom: 16 }}>
                <Text strong>标题：</Text>
                <Paragraph style={{ marginTop: 4, padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
                  {previewResult.title}
                </Paragraph>
              </div>
            )}

            <Divider style={{ margin: '12px 0' }} />

            <Text strong>内容：</Text>
            {previewType === 'email' ? (
              <div
                style={{
                  marginTop: 8,
                  padding: 16,
                  border: '1px solid #f0f0f0',
                  borderRadius: 8,
                  background: '#fff',
                }}
                dangerouslySetInnerHTML={{ __html: previewResult.content }}
              />
            ) : (
              <div
                style={{
                  marginTop: 8,
                  padding: 16,
                  background: '#f5f5f5',
                  borderRadius: 8,
                  whiteSpace: 'pre-wrap',
                  fontFamily: 'Monaco, Menlo, Consolas, monospace',
                  fontSize: 13,
                  lineHeight: 1.8,
                }}
              >
                {previewResult.content}
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}
