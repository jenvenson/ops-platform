// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import React, { useState, useEffect, useCallback } from 'react'
import {
  Table, Tag, Button, Modal, Form, Input, Select, Space,
  message, Popconfirm, Switch, Tooltip, InputNumber,
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined,
  ApiOutlined, MailOutlined, MessageOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { alertAPI, NotifyChannel } from '../../api/alert'
import { canEdit } from '../../utils/menuAccess'

export default function AlertChannelsPage() {
  const { t } = useTranslation('alarm')
  const { t: tc } = useTranslation('common')
  const [channels, setChannels] = useState<NotifyChannel[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<NotifyChannel | undefined>()
  const [form] = Form.useForm()
  const [channelType, setChannelType] = useState('dingtalk')

  const fetchChannels = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await alertAPI.channels.list()
      setChannels(resp.data || [])
    } catch { message.error(tc('loadFailed', '加载失败')) }
    finally { setLoading(false) }
  }, [tc])

  useEffect(() => { fetchChannels() }, [fetchChannels])

  const openModal = (channel?: NotifyChannel) => {
    setEditingChannel(channel)
    if (channel) {
      form.setFieldsValue(channel)
      setChannelType(channel.type)
    } else {
      form.resetFields()
      form.setFieldsValue({ type: 'dingtalk', enabled: true })
      setChannelType('dingtalk')
    }
    setModalOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      if (editingChannel) {
        await alertAPI.channels.update(editingChannel.id, values)
        message.success(tc('updateSuccess', '更新成功'))
      } else {
        await alertAPI.channels.create(values)
        message.success(tc('createSuccess', '创建成功'))
      }
      setModalOpen(false)
      fetchChannels()
    } catch {
      message.error(tc('operationFailed', '操作失败'))
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await alertAPI.channels.delete(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchChannels()
    } catch {
      message.error(tc('deleteFailed', '删除失败'))
    }
  }

  const handleTest = async (id: number) => {
    try {
      const resp = await alertAPI.channels.test(id)
      message.success(resp.message || t('testMessageSent', '测试消息已发送'))
    } catch {
      message.error(t('testSendFailed', '测试发送失败'))
    }
  }

  const typeIcon: Record<string, React.ReactNode> = {
    dingtalk: <MessageOutlined style={{ color: '#1890ff' }} />,
    wechat: <ApiOutlined style={{ color: '#52c41a' }} />,
    email: <MailOutlined style={{ color: '#faad14' }} />,
  }

  const typeLabel: Record<string, string> = {
    dingtalk: t('dingtalkRobot', '钉钉机器人'),
    wechat: t('wechatWorkRobot', '企业微信机器人'),
    email: t('emailType', '邮件'),
  }

  const columns = [
    {
      title: t('type', '类型'), dataIndex: 'type', key: 'type', width: 140,
      render: (v: string) => <Space>{typeIcon[v]}{typeLabel[v] || v}</Space>,
    },
    { title: t('nameLabel', '名称'), dataIndex: 'name', key: 'name', width: 180 },
    { title: t('descriptionLabel', '描述'), dataIndex: 'description', key: 'description', width: 250, ellipsis: true },
    {
      title: tc('status', '状态'), dataIndex: 'enabled', key: 'enabled', width: 80,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? tc('enable', '启用') : tc('disable', '禁用')}</Tag>,
    },
    {
      title: tc('action', '操作'), key: 'action', width: 220,
      render: (_: unknown, r: NotifyChannel) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(r)}>{tc('edit', '编辑')}</Button>}
          <Tooltip title={t('sendTestMessage', '发送测试消息')}>
            <Button type="link" size="small" icon={<SendOutlined />} onClick={() => handleTest(r.id)}>{t('test', '测试')}</Button>
          </Tooltip>
          {canEdit() && <Popconfirm title={t('confirmDelete', '确定删除？')} onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{tc('delete', '删除')}</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>{t('addNotifyChannel', '添加通知渠道')}</Button>
      </div>
      <Table columns={columns} dataSource={channels} rowKey="id" loading={loading}
        pagination={{ pageSize: 20, showTotal: tCount => tc('total', '共 {{count}} 条', { count: tCount }) }} />

      <Modal title={editingChannel ? t('editNotifyChannel', '编辑通知渠道') : t('addNotifyChannel', '添加通知渠道')} open={modalOpen}
        onOk={handleSave} onCancel={() => setModalOpen(false)} width={550}>
        <Form form={form} layout="vertical">
          <Form.Item name="type" label={t('channelType', '渠道类型')} rules={[{ required: true }]}>
            <Select onChange={v => setChannelType(v)} options={[
              { label: t('dingtalkNotifyRobot', '钉钉通知机器人'), value: 'dingtalk' },
              { label: t('wechatWorkNotifyRobot', '企业微信通知机器人'), value: 'wechat' },
              { label: t('emailNotify', '邮件'), value: 'email' },
            ]} />
          </Form.Item>
          <Form.Item name="name" label={t('channelName', '渠道名称')} rules={[{ required: true, message: t('inputName', '请输入名称') }]}>
            <Input placeholder={t('channelNameExample', '如：运维告警群')} />
          </Form.Item>
          <Form.Item name="description" label={t('descriptionLabel', '描述')}>
            <Input placeholder={t('channelDescription', '渠道描述')} />
          </Form.Item>

          {(channelType === 'dingtalk' || channelType === 'wechat') && (
            <>
              <Form.Item name="webhook_url" label={t('webhookAddress', 'Webhook 地址')} rules={[{ required: true, message: t('inputWebhookUrl', '请输入 Webhook URL') }]}>
                <Input.TextArea rows={2} placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
              </Form.Item>
              {channelType === 'dingtalk' && (
                <Form.Item name="secret" label={t('signSecret', '签名密钥（可选）')}>
                  <Input placeholder="SEC..." />
                </Form.Item>
              )}
            </>
          )}

          {channelType === 'email' && (
            <>
              <Form.Item name="smtp_host" label={t('smtpServer', 'SMTP 服务器')} rules={[{ required: true }]}>
                <Input placeholder="smtp.example.com" />
              </Form.Item>
              <Form.Item name="smtp_port" label={t('smtpPort', 'SMTP 端口')} rules={[{ required: true }]}>
                <InputNumber style={{ width: '100%' }} placeholder="465" />
              </Form.Item>
              <Form.Item name="smtp_user" label={t('smtpUser', 'SMTP 用户名')} rules={[{ required: true }]}>
                <Input placeholder="user@example.com" />
              </Form.Item>
              <Form.Item name="smtp_pass" label={t('smtpPassword', 'SMTP 密码')} rules={[{ required: true }]}>
                <Input.Password placeholder={t('smtpPasswordPlaceholder', '密码')} />
              </Form.Item>
              <Form.Item name="email_from" label={t('senderAddress', '发件人地址')}>
                <Input placeholder={t('senderAddressPlaceholder', 'ops-alert@example.com（默认使用 SMTP 用户名）')} />
              </Form.Item>
            </>
          )}

          <Form.Item name="enabled" label={t('enableLabel', '启用')} valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
