// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { Card, Table, Tag, Space, Button, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, KeyOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminAPI, User, Role } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'
import { formatDateTime } from '../../utils/dateFormat'

const roleDisplayKeyMap: Record<string, string> = {
  admin: 'roleDisplayNameAdmin',
  ops: 'roleDisplayNameOps',
  dev: 'roleDisplayNameDev',
  qa: 'roleDisplayNameQa',
  user: 'roleDisplayNameUser',
}

export default function UsersPage() {
  const { t } = useTranslation('admin')
  const { t: tc } = useTranslation('common')

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

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await adminAPI.getUsers()
      setUsers(resp)
    } catch (error) {
      console.error('获取用户列表失败:', error)
      message.error(t('getUsersFailed', '获取用户列表失败'))
    } finally {
      setLoading(false)
    }
  }, [t])

  const fetchRoles = useCallback(async () => {
    try {
      const resp = await adminAPI.getRoles()
      setRoles(resp.filter(r => r.status === 1))
    } catch (error) {
      console.error('获取角色列表失败:', error)
    }
  }, [])

  useEffect(() => {
    fetchUsers()
    fetchRoles()
  }, [fetchUsers, fetchRoles])

  const handleAdd = async () => {
    setEditingUser(null)
    if (roles.length === 0) {
      await fetchRoles()
    }
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = async (user: User) => {
    setEditingUser(user)
    if (roles.length === 0) {
      await fetchRoles()
    }
    form.setFieldsValue({
      username: user.username,
      real_name: user.real_name,
      email: user.email,
      role: user.role,
      password: '',
    })
    setModalVisible(true)
  }

  const handleResetPassword = (user: User) => {
    setResetPasswordUser(user)
    resetPasswordForm.resetFields()
    setResetPasswordVisible(true)
  }

  const handleResetPasswordSubmit = async () => {
    try {
      const values = await resetPasswordForm.validateFields()
      setResetSubmitting(true)
      await adminAPI.resetUserPassword(resetPasswordUser!.id, values.password)
      message.success(t('resetPasswordSuccess', '已成功重置用户的密码').replace('用户', resetPasswordUser!.username))
      setResetPasswordVisible(false)
    } catch (error) {
      console.error('重置密码失败:', error)
      message.error(t('resetPasswordFailed', '重置密码失败'))
    } finally {
      setResetSubmitting(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await adminAPI.deleteUser(id)
      message.success(t('deleteUserSuccess', '删除成功'))
      fetchUsers()
    } catch (error) {
      console.error('删除用户失败:', error)
      message.error(t('deleteUserFailed', '删除用户失败'))
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)

      if (editingUser) {
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
        message.success(t('updateUserSuccess', '更新成功'))
      } else {
        await adminAPI.createUser({
          username: values.username,
          password: values.password,
          real_name: values.real_name,
          email: values.email,
          role: values.role || 'user',
        })
        message.success(t('createUserSuccess', '创建成功'))
      }

      setModalVisible(false)
      fetchUsers()
    } catch (error) {
      console.error('提交失败:', error)
      message.error(t('submitFailed', '提交失败'))
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
      title: t('username', '用户名'),
      dataIndex: 'username',
      key: 'username',
      width: 120,
    },
    {
      title: t('realName', '姓名'),
      dataIndex: 'real_name',
      key: 'real_name',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: t('email', '邮箱'),
      dataIndex: 'email',
      key: 'email',
      width: 180,
      render: (email: string) => email || '-',
    },
    {
      title: t('role', '角色'),
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (roleCode: string) => {
        const role = roles.find(r => r.code === roleCode)
        const i18nKey = roleDisplayKeyMap[roleCode]
        const displayName = i18nKey ? t(i18nKey, role?.name || roleCode) : (role?.name || roleCode)
        const isAdmin = roleCode === 'admin'
        return (
          <Tag color={isAdmin ? 'green' : 'blue'}>
            {displayName}
          </Tag>
        )
      },
    },
    {
      title: t('createdAt', '创建时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? formatDateTime(time) : '-',
    },
    {
      title: t('action', '操作'),
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
              {t('edit', '编辑')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<KeyOutlined />}
              onClick={() => handleResetPassword(record)}
            >
              {t('resetPassword', '重置密码')}
            </Button>
            <Popconfirm
              title={t('confirmDeleteUser', '确认删除')}
              description={t('confirmDeleteUserDesc', '确定要删除这个用户吗？')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('confirm', '确认')}
              cancelText={t('cancel', '取消')}
            >
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
              >
                {t('delete', '删除')}
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
        title={t('userManagement', '用户管理')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchUsers}>
              {t('refresh', '刷新')}
            </Button>
            {canEdit() && (
              <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                {t('addUser', '新增用户')}
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
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total: number) => tc('total', '共 {{count}} 条', { count: total }), showQuickJumper: true }}
          scroll={{ x: 1000 }}
        />
      </Card>

      <Modal
        title={editingUser ? t('editUser', '编辑用户') : t('addNewUser', '新增用户')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="username"
            label={t('username', '用户名')}
            rules={[
              { required: true, message: t('usernameRequired', '请输入用户名') },
              { min: 3, max: 50, message: t('usernameLength', '用户名长度必须在3-50之间') },
            ]}
          >
            <Input placeholder={t('usernameRequired', '请输入用户名')} />
          </Form.Item>

          <Form.Item name="real_name" label={t('realName', '姓名')}>
            <Input placeholder={t('realNamePlaceholder', '请输入姓名（可选）')} />
          </Form.Item>

          <Form.Item
            name="password"
            label={t('password', '密码')}
            rules={[
              { required: !editingUser, message: t('passwordRequired', '请输入密码') },
              { min: 6, message: t('passwordMinLength6', '密码长度至少6位') },
            ]}
          >
            <Input.Password placeholder={editingUser ? t('passwordPlaceholderEdit', '留空则不修改密码') : t('passwordPlaceholderCreate', '请输入密码')} />
          </Form.Item>

          <Form.Item name="email" label={t('email', '邮箱')}>
            <Input placeholder={t('emailPlaceholder', '请输入邮箱（可选）')} />
          </Form.Item>

          <Form.Item name="role" label={t('role', '角色')}>
            <Select
              placeholder={t('selectPlaceholder', '选择角色')}
              options={roles.map(role => {
                const i18nKey = roleDisplayKeyMap[role.code]
                return {
                  label: i18nKey ? t(i18nKey, role.name) : role.name,
                  value: role.code,
                }
              })}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`${t('resetPasswordTitle', '重置密码')} - ${resetPasswordUser?.username || ''}`}
        open={resetPasswordVisible}
        onCancel={() => setResetPasswordVisible(false)}
        onOk={handleResetPasswordSubmit}
        confirmLoading={resetSubmitting}
        destroyOnClose
      >
        <Form form={resetPasswordForm} layout="vertical">
          <Form.Item
            name="password"
            label={t('newPasswordLabel', '新密码')}
            rules={[
              { required: true, message: t('newPasswordRequired', '请输入新密码') },
              { min: 6, message: t('passwordMinLength6', '密码长度至少6位') },
            ]}
          >
            <Input.Password placeholder={t('newPasswordPlaceholder', '请输入新密码')} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('confirmPasswordLabel', '确认密码')}
            dependencies={['password']}
            rules={[
              { required: true, message: t('confirmPasswordRequired', '请再次输入新密码') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error(t('passwordMismatch', '两次输入的密码不一致')))
                },
              }),
            ]}
          >
            <Input.Password placeholder={t('confirmPasswordPlaceholder', '请再次输入新密码')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
