// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { Card, Table, Tag, Space, Button, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, KeyOutlined } from '@ant-design/icons'
import { adminAPI, User, Role } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()
  const [resetPasswordForm] = Form.useForm()
  const [resetPasswordVisible, setResetPasswordVisible] = useState(false)
  const [resetPasswordUser, setResetPasswordUser] = useState<User | null>(null)
  const [resetSubmitting, setResetSubmitting] = useState(false)

  // 加载用户列表
  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await adminAPI.getUsers()
      setUsers(resp)
    } catch (error) {
      console.error('获取用户列表失败:', error)
      message.error('获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }, [])

  // 加载角色列表
  const fetchRoles = useCallback(async () => {
    try {
      const resp = await adminAPI.getRoles()
      setRoles(resp.filter(r => r.status === 1)) // 只显示启用的角色
    } catch (error) {
      console.error('获取角色列表失败:', error)
    }
  }, [])

  useEffect(() => {
    fetchUsers()
    fetchRoles()
  }, [fetchUsers, fetchRoles])

  // 打开新增模态框
  const handleAdd = async () => {
    setEditingUser(null)
    // 确保角色列表已加载
    if (roles.length === 0) {
      await fetchRoles()
    }
    form.resetFields()
    setModalVisible(true)
  }

  // 打开编辑模态框
  const handleEdit = async (user: User) => {
    setEditingUser(user)
    // 确保角色列表已加载
    if (roles.length === 0) {
      await fetchRoles()
    }
    form.setFieldsValue({
      username: user.username,
      real_name: user.real_name,
      email: user.email,
      role: user.role,
      password: '', // 编辑时不显示密码
    })
    setModalVisible(true)
  }

  // 打开重置密码模态框
  const handleResetPassword = (user: User) => {
    setResetPasswordUser(user)
    resetPasswordForm.resetFields()
    setResetPasswordVisible(true)
  }

  // 提交重置密码
  const handleResetPasswordSubmit = async () => {
    try {
      const values = await resetPasswordForm.validateFields()
      setResetSubmitting(true)
      await adminAPI.resetUserPassword(resetPasswordUser!.id, values.password)
      message.success(`已成功重置用户 ${resetPasswordUser!.username} 的密码`)
      setResetPasswordVisible(false)
    } catch (error) {
      console.error('重置密码失败:', error)
      message.error('重置密码失败')
    } finally {
      setResetSubmitting(false)
    }
  }

  // 删除用户
  const handleDelete = async (id: number) => {
    try {
      await adminAPI.deleteUser(id)
      message.success('删除成功')
      fetchUsers()
    } catch (error) {
      console.error('删除用户失败:', error)
      message.error('删除用户失败')
    }
  }

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)

      if (editingUser) {
        // 更新用户
        const updateData: { username?: string; real_name?: string; email?: string; role?: 'admin' | 'user'; password?: string } = {}
        if (values.username && values.username !== editingUser.username) {
          updateData.username = values.username
        }
        if (values.real_name !== editingUser.real_name) {
          updateData.real_name = values.real_name
        }
        if (values.email && values.email !== editingUser.email) {
          updateData.email = values.email
        }
        if (values.role && values.role !== editingUser.role) {
          updateData.role = values.role as 'admin' | 'user'
        }
        if (values.password) {
          updateData.password = values.password
        }

        await adminAPI.updateUser(editingUser.id, updateData)
        message.success('更新成功')
      } else {
        // 创建用户
        await adminAPI.createUser({
          username: values.username,
          password: values.password,
          real_name: values.real_name,
          email: values.email,
          role: values.role || 'user',
        })
        message.success('创建成功')
      }

      setModalVisible(false)
      fetchUsers()
    } catch (error) {
      console.error('提交失败:', error)
      message.error('提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      width: 120,
    },
    {
      title: '姓名',
      dataIndex: 'real_name',
      key: 'real_name',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      width: 180,
      render: (email: string) => email || '-',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (roleCode: string) => {
        const role = roles.find(r => r.code === roleCode)
        const roleName = role?.name || roleCode
        const isAdmin = roleCode === 'admin'
        return (
          <Tag color={isAdmin ? 'green' : 'blue'}>
            {roleName}
          </Tag>
        )
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_: unknown, record: User) => {
        if (!canEdit()) return '-'
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              编辑
            </Button>
            <Button
              type="link"
              size="small"
              icon={<KeyOutlined />}
              onClick={() => handleResetPassword(record)}
            >
              重置密码
            </Button>
            <Popconfirm
              title="确认删除"
              description="确定要删除这个用户吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="确认"
              cancelText="取消"
            >
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        )
      },
    },
  ]

  return (
    <div>
      <Card
        title="用户管理"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchUsers}>
              刷新
            </Button>
            {canEdit() && (
              <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                新增用户
              </Button>
            )}
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total: number) => `共 ${total} 条`, showQuickJumper: true }}
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* 新增/编辑用户模态框 */}
      <Modal
        title={editingUser ? '编辑用户' : '新增用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, max: 50, message: '用户名长度必须在3-50之间' },
            ]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

          <Form.Item name="real_name" label="姓名">
            <Input placeholder="请输入姓名（可选）" />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: !editingUser, message: '请输入密码' },
              { min: 6, message: '密码长度至少6位' },
            ]}
          >
            <Input.Password placeholder={editingUser ? '留空则不修改密码' : '请输入密码'} />
          </Form.Item>

          <Form.Item name="email" label="邮箱">
            <Input placeholder="请输入邮箱（可选）" />
          </Form.Item>

          <Form.Item name="role" label="角色">
            <Select
              placeholder="选择角色"
              options={roles.map(role => ({
                label: role.name,
                value: role.code,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码模态框 */}
      <Modal
        title={`重置密码 - ${resetPasswordUser?.username || ''}`}
        open={resetPasswordVisible}
        onCancel={() => setResetPasswordVisible(false)}
        onOk={handleResetPasswordSubmit}
        confirmLoading={resetSubmitting}
        destroyOnClose
      >
        <Form form={resetPasswordForm} layout="vertical">
          <Form.Item
            name="password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度至少6位' },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请再次输入新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}