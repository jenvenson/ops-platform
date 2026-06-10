// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect } from 'react'
import {
  Card, Form, Input, Button, Table, Modal, message, Space, Tag, Popconfirm
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined
} from '@ant-design/icons'
import { consulAPI, ConsulConfig } from '../../api/consul'

export default function ConfigManagementPage() {
  const [configs, setConfigs] = useState<ConsulConfig[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingConfig, setEditingConfig] = useState<ConsulConfig | null>(null)
  const [form] = Form.useForm()

  // 获取配置列表
  const fetchConfigs = async () => {
    setLoading(true)
    try {
      const data = await consulAPI.getConfigs()
      setConfigs(data)
    } catch (error) {
      message.error('获取配置列表失败')
    } finally {
      setLoading(false)
    }
  }

  // 保存配置
  const saveConfig = async (values: any) => {
    try {
      if (editingConfig) {
        // 更新现有配置
        await consulAPI.updateConfig(editingConfig.id, values)
        message.success('配置更新成功')
      } else {
        // 创建新配置
        await consulAPI.createConfig(values)
        message.success('配置创建成功')
      }
      setModalVisible(false)
      form.resetFields()
      setEditingConfig(null)
      fetchConfigs()
    } catch (error: any) {
      message.error(error.message || '操作失败')
    }
  }

  // 删除配置
  const deleteConfig = async (id: number) => {
    try {
      await consulAPI.deleteConfig(id)
      message.success('配置删除成功')
      fetchConfigs()
    } catch (error: any) {
      message.error(error.message || '删除失败')
    }
  }

  // 测试连接
  const testConnection = async (id: number) => {
    try {
      await consulAPI.testConnection(id)
      message.success('连接测试成功')
    } catch (error: any) {
      message.error(error.message || '连接测试失败')
    }
  }

  // 设置默认配置
  const setDefaultConfig = async (id: number) => {
    try {
      const config = configs.find(c => c.id === id)
      if (!config) return

      // 更新所有配置的is_default状态
      const updates = configs.map(async (c) => {
        if (c.id === id) {
          await consulAPI.updateConfig(c.id, { ...c, is_default: true })
        } else if (c.is_default) {
          await consulAPI.updateConfig(c.id, { ...c, is_default: false })
        }
      })

      await Promise.all(updates)
      message.success('默认配置设置成功')
      fetchConfigs()
    } catch (error: any) {
      message.error(error.message || '设置默认配置失败')
    }
  }

  useEffect(() => {
    fetchConfigs()
  }, [])

  const columns = [
    {
      title: '配置名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Consul地址',
      dataIndex: 'address',
      key: 'address',
    },
    {
      title: '数据中心',
      dataIndex: 'datacenter',
      key: 'datacenter',
    },
    {
      title: '默认配置',
      dataIndex: 'is_default',
      key: 'is_default',
      render: (isDefault: boolean) => (
        <Tag color={isDefault ? 'green' : 'default'}>
          {isDefault ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: ConsulConfig) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingConfig(record)
              form.setFieldsValue({
                name: record.name,
                address: record.address,
                datacenter: record.datacenter,
                token: record.token,
                username: record.username,
                password: record.password,
              })
              setModalVisible(true)
            }}
          >
            编辑
          </Button>
          <Button
            size="small"
            onClick={() => testConnection(record.id)}
          >
            测试连接
          </Button>
          {!record.is_default && (
            <Button
              size="small"
              onClick={() => setDefaultConfig(record.id)}
            >
              设为默认
            </Button>
          )}
          <Popconfirm
            title="确定要删除这个配置吗？"
            onConfirm={() => deleteConfig(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              danger
              size="small"
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <Card
        title="Consul配置"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingConfig(null)
              form.resetFields()
              setModalVisible(true)
            }}
          >
            添加配置
          </Button>
        }
        size="small"
      >
        <Table
          dataSource={configs}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* 配置编辑模态框 */}
      <Modal
        title={editingConfig ? '编辑配置' : '添加配置'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false)
          form.resetFields()
          setEditingConfig(null)
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={saveConfig}
          initialValues={{
            datacenter: 'dc1',
          }}
        >
          <Form.Item
            name="name"
            label="配置名称"
            rules={[{ required: true, message: '请输入配置名称' }]}
          >
            <Input placeholder="请输入配置名称，如：生产环境Consul" />
          </Form.Item>

          <Form.Item
            name="address"
            label="Consul地址"
            rules={[
              { required: true, message: '请输入Consul地址' },
              {
                pattern: /^https?:\/\/[\w.-]+(:\d+)?$/,
                message: '请输入有效的URL地址，如：http://127.0.0.1:8500'
              }
            ]}
          >
            <Input placeholder="请输入Consul地址，如：http://127.0.0.1:8500" />
          </Form.Item>

          <Form.Item
            name="datacenter"
            label="数据中心"
            rules={[{ required: true, message: '请输入数据中心名称' }]}
          >
            <Input placeholder="请输入数据中心名称，默认为dc1" />
          </Form.Item>

          <Form.Item
            name="token"
            label="ACL Token"
          >
            <Input.Password placeholder="可选：输入ACL Token用于认证" />
          </Form.Item>

          <Form.Item
            name="username"
            label="用户名"
          >
            <Input placeholder="可选：基本认证用户名" />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
          >
            <Input.Password placeholder="可选：基本认证密码" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存配置
              </Button>
              <Button onClick={() => {
                setModalVisible(false)
                form.resetFields()
                setEditingConfig(null)
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}