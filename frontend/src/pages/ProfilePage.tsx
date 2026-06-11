// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect } from 'react'
import { Card, Descriptions, Button, Modal, Form, Input, message, Tag, Divider, Space } from 'antd'
import { KeyOutlined, UserOutlined, MailOutlined, SafetyOutlined, CalendarOutlined, IdcardOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminAPI, User } from '../api/admin'
import { formatDateTime } from '../utils/dateFormat'

export default function ProfilePage() {
  const { t } = useTranslation('admin')

  const [userInfo, setUserInfo] = useState<User | null>(null)
  const [loading, setLoading] = useState(false)
  const [passwordModalVisible, setPasswordModalVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [passwordForm] = Form.useForm()

  // 获取当前用户信息
  const fetchUserInfo = async () => {
    setLoading(true)
    try {
      const user = await adminAPI.getCurrentUser()
      setUserInfo(user)
    } catch (error) {
      console.error('获取用户信息失败:', error)
      message.error(t('loadFailed', '获取用户信息失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUserInfo()
  }, [])

  const handleChangePassword = async () => {
    try {
      const values = await passwordForm.validateFields()
      setSubmitting(true)
      await adminAPI.changePassword({
        old_password: values.oldPassword,
        new_password: values.newPassword,
      })
      message.success(t('changePasswordSuccess', '密码修改成功，下次登录请使用新密码'))
      setPasswordModalVisible(false)
      passwordForm.resetFields()
    } catch (error: any) {
      const errMsg = error?.response?.data?.error || t('changePasswordFailed', '密码修改失败')
      message.error(errMsg)
    } finally {
      setSubmitting(false)
    }
  }

  const roleNameMap: Record<string, string> = {
    admin: t('roleDisplayNameAdmin', '超级管理员'),
    ops: t('roleDisplayNameOps', '运维人员'),
    dev: t('roleDisplayNameDev', '开发人员'),
    qa: t('roleDisplayNameQa', '测试人员'),
    user: t('roleDisplayNameUser', '普通用户'),
  }

  const roleColors: Record<string, string> = {
    admin: 'red',
    ops: 'green',
    dev: 'blue',
    qa: 'orange',
    user: 'default',
  }

  return (
    <div>
      <Card title={t('profile', '个人信息')} loading={loading}>
        {userInfo && (
          <Descriptions
            bordered
            column={{ xs: 1, sm: 1, md: 2 }}
            labelStyle={{ width: 140, fontWeight: 500 }}
          >
            <Descriptions.Item label={<><UserOutlined style={{ marginRight: 8 }} />{t('username', '用户名')}</>}>
              {userInfo.username}
            </Descriptions.Item>
            <Descriptions.Item label={<><IdcardOutlined style={{ marginRight: 8 }} />{t('realName', '姓名')}</>}>
              {userInfo.real_name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={<><SafetyOutlined style={{ marginRight: 8 }} />{t('role', '角色')}</>}>
              <Tag color={roleColors[userInfo.role] || 'default'}>
                {roleNameMap[userInfo.role] || userInfo.role}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={<><MailOutlined style={{ marginRight: 8 }} />{t('email', '邮箱')}</>}>
              {userInfo.email || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={<><CalendarOutlined style={{ marginRight: 8 }} />{t('createdAt', '创建时间')}</>}>
              {formatDateTime(userInfo.created_at)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Card>

      <Divider />

      <Card title={t('securitySettings', '安全设置')}>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <div style={{ fontWeight: 500, marginBottom: 4 }}>{t('loginPassword', '登录密码')}</div>
              <div style={{ color: '#999', fontSize: 13 }}>
                {t('loginPasswordHint', '定期修改密码可以提高账户安全性')}
              </div>
            </div>
            <Button
              type="primary"
              icon={<KeyOutlined />}
              onClick={() => {
                passwordForm.resetFields()
                setPasswordModalVisible(true)
              }}
            >
              {t('changePassword', '修改密码')}
            </Button>
          </div>
        </Space>
      </Card>

      {/* 修改密码模态框 */}
      <Modal
        title={t('changePasswordTitle', '修改密码')}
        open={passwordModalVisible}
        onCancel={() => setPasswordModalVisible(false)}
        onOk={handleChangePassword}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="oldPassword"
            label={t('currentPassword', '当前密码')}
            rules={[{ required: true, message: t('currentPasswordRequired', '请输入当前密码') }]}
          >
            <Input.Password placeholder={t('currentPasswordPlaceholder', '请输入当前密码')} />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('newPassword', '新密码')}
            rules={[
              { required: true, message: t('newPasswordRequired', '请输入新密码') },
              { min: 6, message: t('passwordMinLength6', '密码长度至少6位') },
            ]}
          >
            <Input.Password placeholder={t('newPasswordHintPlaceholder', '请输入新密码（至少6位）')} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('confirmPasswordLabel', '确认新密码')}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('confirmPasswordRequired', '请再次输入新密码') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
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
