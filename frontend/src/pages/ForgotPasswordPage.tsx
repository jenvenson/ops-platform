import { useState } from 'react'
import { Form, Input, Button, message, Typography, Card, Result } from 'antd'
import { Link } from 'react-router-dom'
import { UserOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import apiClient from '../api/client'

const { Title, Text } = Typography

export default function ForgotPasswordPage() {
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ token: string; expires_at: string } | null>(null)

  const onSubmit = async (values: { username: string }) => {
    setLoading(true)
    try {
      const res = await apiClient.post<{ message: string; token?: string; expires_at?: string }>(
        '/auth/forgot-password',
        values
      )
      if (res.token) {
        setResult({ token: res.token, expires_at: res.expires_at || '' })
        message.success('重置令牌已生成')
      } else {
        message.info(res.message || '如果用户存在，重置令牌已生成')
      }
    } catch {
      message.error('请求失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (result) {
    return (
      <div style={styles.container}>
        <Card style={styles.card} styles={{ body: { padding: '48px 40px' } }}>
          <Result
            status="success"
            title="重置令牌已生成"
            subTitle={
              <div style={{ textAlign: 'left', marginTop: 16 }}>
                <Text>请复制以下令牌，用于重置密码（有效期 1 小时）：</Text>
                <div style={{
                  background: '#f6f8fa',
                  border: '1px solid #e1e4e8',
                  borderRadius: 6,
                  padding: '12px 16px',
                  margin: '12px 0',
                  fontFamily: 'monospace',
                  fontSize: 14,
                  wordBreak: 'break-all',
                }}>
                  {result.token}
                </div>
                <Link to={`/reset-password?token=${result.token}`}>
                  <Button type="primary" block size="large">
                    前往重置密码
                  </Button>
                </Link>
              </div>
            }
          />
        </Card>
      </div>
    )
  }

  return (
    <div style={styles.container}>
      <Card style={styles.card} styles={{ body: { padding: '48px 40px' } }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={styles.logo}>
            <span style={styles.logoText}>OPS</span>
          </div>
          <Title level={3} style={{ marginTop: 16 }}>忘记密码</Title>
          <Text type="secondary">输入用户名获取重置令牌</Text>
        </div>
        <Form onFinish={onSubmit}>
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              placeholder="用户名"
              prefix={<UserOutlined />}
              size="large"
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block size="large">
              获取重置令牌
            </Button>
          </Form.Item>
        </Form>
        <div style={{ textAlign: 'center' }}>
          <Link to="/login">
            <ArrowLeftOutlined /> 返回登录
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
    width: 420,
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
