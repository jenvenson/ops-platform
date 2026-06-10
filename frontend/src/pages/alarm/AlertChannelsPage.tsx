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
import { alertAPI, NotifyChannel } from '../../api/alert'
import { canEdit } from '../../utils/menuAccess'

export default function AlertChannelsPage() {
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
    } catch { message.error('加载失败') }
    finally { setLoading(false) }
  }, [])

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
        message.success('更新成功')
      } else {
        await alertAPI.channels.create(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      fetchChannels()
    } catch {
      message.error('操作失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await alertAPI.channels.delete(id)
      message.success('删除成功')
      fetchChannels()
    } catch {
      message.error('删除失败')
    }
  }

  const handleTest = async (id: number) => {
    try {
      const resp = await alertAPI.channels.test(id)
      message.success(resp.message || '测试消息已发送')
    } catch {
      message.error('测试发送失败')
    }
  }

  const typeIcon: Record<string, React.ReactNode> = {
    dingtalk: <MessageOutlined style={{ color: '#1890ff' }} />,
    wechat: <ApiOutlined style={{ color: '#52c41a' }} />,
    email: <MailOutlined style={{ color: '#faad14' }} />,
  }

  const typeLabel: Record<string, string> = {
    dingtalk: '钉钉机器人',
    wechat: '企业微信机器人',
    email: '邮件',
  }

  const columns = [
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 140,
      render: (v: string) => <Space>{typeIcon[v]}{typeLabel[v] || v}</Space>,
    },
    { title: '名称', dataIndex: 'name', key: 'name', width: 180 },
    { title: '描述', dataIndex: 'description', key: 'description', width: 250, ellipsis: true },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 80,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag>,
    },
    {
      title: '操作', key: 'action', width: 220,
      render: (_: unknown, r: NotifyChannel) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(r)}>编辑</Button>}
          <Tooltip title="发送测试消息">
            <Button type="link" size="small" icon={<SendOutlined />} onClick={() => handleTest(r.id)}>测试</Button>
          </Tooltip>
          {canEdit() && <Popconfirm title="确定删除？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>添加通知渠道</Button>
      </div>
      <Table columns={columns} dataSource={channels} rowKey="id" loading={loading}
        pagination={{ pageSize: 20, showTotal: t => `共 ${t} 条` }} />

      <Modal title={editingChannel ? '编辑通知渠道' : '添加通知渠道'} open={modalOpen}
        onOk={handleSave} onCancel={() => setModalOpen(false)} width={550}>
        <Form form={form} layout="vertical">
          <Form.Item name="type" label="渠道类型" rules={[{ required: true }]}>
            <Select onChange={v => setChannelType(v)} options={[
              { label: '钉钉通知机器人', value: 'dingtalk' },
              { label: '企业微信通知机器人', value: 'wechat' },
              { label: '邮件', value: 'email' },
            ]} />
          </Form.Item>
          <Form.Item name="name" label="渠道名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：运维告警群" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="渠道描述" />
          </Form.Item>

          {(channelType === 'dingtalk' || channelType === 'wechat') && (
            <>
              <Form.Item name="webhook_url" label="Webhook 地址" rules={[{ required: true, message: '请输入 Webhook URL' }]}>
                <Input.TextArea rows={2} placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
              </Form.Item>
              {channelType === 'dingtalk' && (
                <Form.Item name="secret" label="签名密钥（可选）">
                  <Input placeholder="SEC..." />
                </Form.Item>
              )}
            </>
          )}

          {channelType === 'email' && (
            <>
              <Form.Item name="smtp_host" label="SMTP 服务器" rules={[{ required: true }]}>
                <Input placeholder="smtp.example.com" />
              </Form.Item>
              <Form.Item name="smtp_port" label="SMTP 端口" rules={[{ required: true }]}>
                <InputNumber style={{ width: '100%' }} placeholder="465" />
              </Form.Item>
              <Form.Item name="smtp_user" label="SMTP 用户名" rules={[{ required: true }]}>
                <Input placeholder="user@example.com" />
              </Form.Item>
              <Form.Item name="smtp_pass" label="SMTP 密码" rules={[{ required: true }]}>
                <Input.Password placeholder="密码" />
              </Form.Item>
              <Form.Item name="email_from" label="发件人地址">
                <Input placeholder="ops-alert@example.com（默认使用 SMTP 用户名）" />
              </Form.Item>
            </>
          )}

          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}