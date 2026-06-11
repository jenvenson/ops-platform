// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

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
import { useTranslation } from 'react-i18next'
import { alertAPI, AlertTemplate } from '../../api/alert'
import { canEdit } from '../../utils/menuAccess'

const { TextArea } = Input
const { Text, Paragraph } = Typography

export default function AlertTemplatesPage() {
  const { t } = useTranslation('alarm')
  const { t: tc } = useTranslation('common')
  const [templates, setTemplates] = useState<AlertTemplate[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [editingTpl, setEditingTpl] = useState<AlertTemplate | undefined>()
  const [previewResult, setPreviewResult] = useState<{ title: string; content: string } | null>(null)
  const [previewType, setPreviewType] = useState('dingtalk')
  const [form] = Form.useForm()

  const typeConfig: Record<string, { icon: React.ReactNode; label: string; color: string }> = {
    dingtalk: { icon: <MessageOutlined />, label: t('dingtalk', '钉钉'), color: '#1890ff' },
    wechat:   { icon: <ApiOutlined />,     label: t('wechatWork', '企微'), color: '#52c41a' },
    email:    { icon: <MailOutlined />,     label: t('emailType', '邮件'), color: '#faad14' },
  }

  const sceneConfig: Record<string, { label: string; color: string }> = {
    firing:   { label: t('alertFiring', '告警触发'), color: 'red' },
    resolved: { label: t('alertRecovered', '告警恢复'), color: 'green' },
  }

  // 内置默认模板的名称/描述按界面语言显示，自定义模板保持原文
  const defaultTplText: Record<string, string> = {
    '钉钉-告警触发': t('defaultTplDingtalkFiring', '钉钉-告警触发'),
    '钉钉-告警恢复': t('defaultTplDingtalkResolved', '钉钉-告警恢复'),
    '企微-告警触发': t('defaultTplWechatFiring', '企微-告警触发'),
    '企微-告警恢复': t('defaultTplWechatResolved', '企微-告警恢复'),
    '邮件-告警触发': t('defaultTplEmailFiring', '邮件-告警触发'),
    '邮件-告警恢复': t('defaultTplEmailResolved', '邮件-告警恢复'),
    '钉钉机器人默认告警触发模板': t('defaultTplDingtalkFiringDesc', '钉钉机器人默认告警触发模板'),
    '钉钉机器人默认告警恢复模板': t('defaultTplDingtalkResolvedDesc', '钉钉机器人默认告警恢复模板'),
    '企业微信默认告警触发模板': t('defaultTplWechatFiringDesc', '企业微信默认告警触发模板'),
    '企业微信默认告警恢复模板': t('defaultTplWechatResolvedDesc', '企业微信默认告警恢复模板'),
    '邮件默认告警触发模板': t('defaultTplEmailFiringDesc', '邮件默认告警触发模板'),
    '邮件默认告警恢复模板': t('defaultTplEmailResolvedDesc', '邮件默认告警恢复模板'),
  }
  const displayTpl = (s?: string) => (s && defaultTplText[s]) || s

  const fetchTemplates = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await alertAPI.templates.list()
      setTemplates(resp.data || [])
    } catch { message.error(tc('loadFailed', '加载失败')) }
    finally { setLoading(false) }
  }, [tc])

  useEffect(() => { fetchTemplates() }, [fetchTemplates])

  // 可用模板变量说明
  const templateVars = [
    { var: '{{.RuleName}}', desc: t('varRuleName', '规则名称'), example: 'DiskUsageHigh' },
    { var: '{{.Content}}', desc: t('varContent', '告警内容（触发=原始描述，恢复=恢复内容+当前值）'), example: '磁盘使用率达到 95%' },
    { var: '{{.CurrentValue}}', desc: t('varCurrentValue', '当前指标值（仅恢复告警时有效）'), example: '3.72%' },
    { var: '{{.Source}}', desc: t('varSource', '告警来源'), example: 'node-exporter:9100' },
    { var: '{{.Severity}}', desc: t('varSeverity', '级别(英文)'), example: 'critical' },
    { var: '{{.SeverityLabel}}', desc: t('varSeverityLabel', '级别(中文)'), example: '严重' },
    { var: '{{.Status}}', desc: t('varStatus', '状态(英文)'), example: 'firing' },
    { var: '{{.StatusLabel}}', desc: t('varStatusLabel', '状态(中文)'), example: '告警中' },
    { var: '{{.Category}}', desc: t('varCategory', '分类(英文)'), example: 'disk' },
    { var: '{{.CategoryLabel}}', desc: t('varCategoryLabel', '分类(中文)'), example: '磁盘' },
    { var: '{{.Time}}', desc: t('varTime', '触发/恢复时间'), example: '2026-02-09 12:00:00' },
    { var: '{{.Emoji}}', desc: t('varEmoji', '级别Emoji'), example: '🔴' },
  ]

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
        message.success(tc('updateSuccess', '更新成功'))
      } else {
        await alertAPI.templates.create(values)
        message.success(tc('createSuccess', '创建成功'))
      }
      setModalOpen(false)
      fetchTemplates()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      if (error?.response?.data?.error) {
        message.error(error.response.data.error)
      } else {
        message.error(tc('operationFailed', '操作失败'))
      }
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await alertAPI.templates.delete(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchTemplates()
    } catch { message.error(tc('deleteFailed', '删除失败')) }
  }

  const handleSetDefault = async (id: number) => {
    try {
      await alertAPI.templates.setDefault(id)
      message.success(t('setDefaultSuccess', '已设为默认模板'))
      fetchTemplates()
    } catch { message.error(tc('operationFailed', '操作失败')) }
  }

  const handleCopy = (tpl: AlertTemplate) => {
    form.resetFields()
    form.setFieldsValue({
      ...tpl,
      name: tpl.name + t('copySuffix', '_副本'),
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
        message.error(t('previewFailed', '预览失败'))
      }
    }
  }

  const typeFilters = [
    { text: t('dingtalk', '钉钉'), value: 'dingtalk' },
    { text: t('wechatWork', '企微'), value: 'wechat' },
    { text: t('emailType', '邮件'), value: 'email' },
  ]

  const sceneFilters = [
    { text: t('alertFiring', '告警触发'), value: 'firing' },
    { text: t('alertRecovered', '告警恢复'), value: 'resolved' },
  ]

  const columns = [
    {
      title: t('channelType', '渠道类型'), dataIndex: 'type', key: 'type', width: 100,
      render: (v: string) => {
        const cfg = typeConfig[v]
        return cfg ? <Space><span style={{ color: cfg.color }}>{cfg.icon}</span>{cfg.label}</Space> : v
      },
      filters: typeFilters,
      onFilter: (value: unknown, record: AlertTemplate) => record.type === value,
    },
    {
      title: t('scene', '场景'), dataIndex: 'scene', key: 'scene', width: 110,
      render: (v: string) => {
        const cfg = sceneConfig[v]
        return cfg ? <Tag color={cfg.color}>{cfg.label}</Tag> : v
      },
      filters: sceneFilters,
      onFilter: (value: unknown, record: AlertTemplate) => record.scene === value,
    },
    {
      title: t('templateName', '模板名称'), dataIndex: 'name', key: 'name', width: 200,
      render: (v: string, r: AlertTemplate) => (
        <Space>
          {r.is_default && <StarFilled style={{ color: '#faad14' }} />}
          {displayTpl(v)}
        </Space>
      ),
    },
    {
      title: t('descriptionLabel', '描述'), dataIndex: 'description', key: 'description', ellipsis: true,
      render: (v: string) => displayTpl(v),
    },
    {
      title: t('default', '默认'), dataIndex: 'is_default', key: 'is_default', width: 70,
      render: (v: boolean) => v ? <Tag color="gold">{t('default', '默认')}</Tag> : <Tag>{tc('no', '否')}</Tag>,
    },
    {
      title: tc('status', '状态'), dataIndex: 'enabled', key: 'enabled', width: 70,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? tc('enable', '启用') : tc('disable', '禁用')}</Tag>,
    },
    {
      title: tc('action', '操作'), key: 'action', width: 260,
      render: (_: unknown, r: AlertTemplate) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(r)}>{tc('edit', '编辑')}</Button>}
          {canEdit() && <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => handleCopy(r)}>{t('copy', '复制')}</Button>}
          {!r.is_default && (
            <Tooltip title={t('setAsDefault', '设为默认模板')}>
              <Button type="link" size="small" icon={<StarOutlined />} onClick={() => handleSetDefault(r.id)}>{t('default', '默认')}</Button>
            </Tooltip>
          )}
          {canEdit() && <Popconfirm title={t('confirmDeleteTemplate', '确定删除此模板？')} onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{tc('delete', '删除')}</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        {canEdit() && <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>{t('newTemplate', '新建模板')}</Button>}
        <Text type="secondary">{t('autoUseDefaultTemplateDesc', '告警通知发送时会自动使用对应渠道和场景的默认模板')}</Text>
      </div>

      <Table
        columns={columns}
        dataSource={templates}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20, showTotal: tCount => tc('total', '共 {{count}} 条', { count: tCount }) }}
      />

      {/* 编辑/新建模板弹窗 */}
      <Modal
        title={editingTpl ? t('editAlertTemplate', '编辑告警模板') : t('newAlertTemplate', '新建告警模板')}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        width={800}
        footer={[
          <Button key="cancel" onClick={() => setModalOpen(false)}>{tc('cancel', '取消')}</Button>,
          <Button key="preview" icon={<EyeOutlined />} onClick={handlePreview}>{t('preview', '预览')}</Button>,
          <Button key="save" type="primary" onClick={handleSave}>{tc('save', '保存')}</Button>,
        ]}
      >
        <Row gutter={24}>
          <Col span={16}>
            <Form form={form} layout="vertical">
              <Row gutter={16}>
                <Col span={8}>
                  <Form.Item name="type" label={t('channelType', '渠道类型')} rules={[{ required: true }]}>
                    <Select options={[
                      { label: t('dingtalk', '钉钉'), value: 'dingtalk' },
                      { label: t('wechatWork', '企业微信'), value: 'wechat' },
                      { label: t('emailType', '邮件'), value: 'email' },
                    ]} />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="scene" label={t('usageScene', '使用场景')} rules={[{ required: true }]}>
                    <Select options={[
                      { label: t('alertFiring', '告警触发'), value: 'firing' },
                      { label: t('alertRecovered', '告警恢复'), value: 'resolved' },
                    ]} />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="enabled" label={t('enableLabel', '启用')} valuePropName="checked" initialValue={true}>
                    <Switch />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item name="name" label={t('templateName', '模板名称')} rules={[{ required: true, message: t('inputTemplateName', '请输入模板名称') }]}>
                <Input placeholder={t('templateNameExample', '如：钉钉-磁盘告警模板')} />
              </Form.Item>

              <Form.Item name="description" label={t('descriptionLabel', '描述')}>
                <Input placeholder={t('templateUsageDescription', '模板用途描述')} />
              </Form.Item>

              <Form.Item name="title_tpl" label={t('titleTemplate', '标题模板')}>
                <Input placeholder={t('titleTplPlaceholder', '如：{{.Emoji}} 【告警】{{.RuleName}}')} />
              </Form.Item>

              <Form.Item name="content_tpl" label={t('contentTemplate', '内容模板')} rules={[{ required: true, message: t('inputContentTemplate', '请输入内容模板') }]}>
                <TextArea
                  rows={12}
                  placeholder={t('contentTplPlaceholder', '### {{.Emoji}} 【告警】{{.RuleName}}\n\n> **规则名称**：{{.RuleName}}\n\n> **告警内容**：{{.Content}}\n\n> **来源**：{{.Source}}\n\n> **级别**：{{.SeverityLabel}}\n\n> **状态**：{{.StatusLabel}}\n\n> **触发时间**：{{.Time}}')}
                  style={{ fontFamily: 'Monaco, Menlo, Consolas, monospace', fontSize: 13 }}
                />
              </Form.Item>
            </Form>
          </Col>

          <Col span={8}>
            <Card size="small" title={t('availableVariables', '可用变量')} style={{ marginTop: 30 }}>
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
        title={t('templatePreview', '模板预览')}
        open={previewOpen}
        onCancel={() => setPreviewOpen(false)}
        footer={<Button onClick={() => setPreviewOpen(false)}>{tc('close', '关闭')}</Button>}
        width={650}
      >
        {previewResult && (
          <div>
            <Alert
              message={t('previewInfo', '以下为使用示例数据渲染的预览效果')}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            {previewResult.title && (
              <div style={{ marginBottom: 16 }}>
                <Text strong>{t('title', '标题：')}</Text>
                <Paragraph style={{ marginTop: 4, padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
                  {previewResult.title}
                </Paragraph>
              </div>
            )}

            <Divider style={{ margin: '12px 0' }} />

            <Text strong>{t('content', '内容：')}</Text>
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
