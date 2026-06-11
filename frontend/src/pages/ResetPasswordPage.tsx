// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState } from 'react'
import { Form, Input, Button, message, Typography, Card } from 'antd'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { LockOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'

const { Title, Text } = Typography

export default function ResetPasswordPage() {
  const { t } = useTranslation('login')
  const [loading, setLoading] = useState(false)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const tokenFromUrl = searchParams.get('token') || ''

  const onSubmit = async (values: { token: string; new_password: string; confirm_password: string }) => {
    if (values.new_password !== values.confirm_password) {
      message.error(t('passwordMismatch', '两次输入的密码不一致'))
      return
    }
    setLoading(true)
    try {
      await apiClient.post('/auth/reset-password', {
        token: values.token,
        new_password: values.new_password,
      })
      message.success(t('resetSuccess', '密码重置成功，请使用新密码登录'))
      navigate('/login')
    } catch (error: unknown) {
      const errMsg =
        (error as { response?: { data?: { error?: string } } }).response?.data?.error ||
        t('resetFailed', '密码重置失败，请重试')
      message.error(errMsg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={styles.container}>
      <Card style={styles.card} styles={{ body: { padding: '48px 40px' } }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={styles.logo}>
            <span style={styles.logoText}>OPS</span>
          </div>
          <Title level={3} style={{ marginTop: 16 }}>{t('resetPasswordTitle', '重置密码')}</Title>
          <Text type="secondary">{t('resetPasswordSubtitle', '输入重置令牌和新密码')}</Text>
        </div>
        <Form
          onFinish={onSubmit}
          initialValues={{ token: tokenFromUrl }}
        >
          <Form.Item
            name="token"
            rules={[{ required: true, message: t('resetTokenRequired', '请输入重置令牌') }]}
          >
            <Input.TextArea
              placeholder={t('resetToken', '重置令牌')}
              size="large"
              autoSize={{ minRows: 2, maxRows: 3 }}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
          <Form.Item
            name="new_password"
            rules={[
              { required: true, message: t('newPasswordRequired', '请输入新密码') },
              { min: 6, message: t('passwordMinLength', '密码长度至少 6 位') },
            ]}
          >
            <Input.Password
              placeholder={t('newPasswordPlaceholder', '新密码')}
              prefix={<LockOutlined />}
              size="large"
            />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            rules={[{ required: true, message: t('confirmPasswordRequired', '请再次输入新密码') }]}
          >
            <Input.Password
              placeholder={t('confirmPasswordPlaceholder', '确认新密码')}
              prefix={<LockOutlined />}
              size="large"
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block size="large">
              {t('resetPasswordButton', '重置密码')}
            </Button>
          </Form.Item>
        </Form>
        <div style={{ textAlign: 'center' }}>
          <Link to="/login">
            <ArrowLeftOutlined /> {t('backToLogin', '返回登录')}
          </Link>
        </div>
      </Card>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: '100vh',
    background: '#f0f4f8',
  },
  card: {
    width: 440,
    borderRadius: 24,
    background: 'rgba(255, 255, 255, 0.85)',
    backdropFilter: 'blur(20px)',
    boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.15)',
  },
  logo: {
    width: 64,
    height: 64,
    borderRadius: 16,
    background: 'linear-gradient(135deg, #40a9ff 0%, #096dd9 100%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '0 auto',
  },
  logoText: {
    fontSize: 22,
    fontWeight: 700,
    color: '#fff',
    letterSpacing: 2,
  },
}
