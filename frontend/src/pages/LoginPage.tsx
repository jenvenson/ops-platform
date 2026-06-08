import { useState } from 'react'
import { Form, Input, Button, message, Typography, Card } from 'antd'
import { useNavigate } from 'react-router-dom'
import { UserOutlined, LockOutlined, ArrowRightOutlined } from '@ant-design/icons'
import apiClient from '../api/client'
import { notifyMenusChanged } from '../utils/menuAccess'

const { Title, Text } = Typography

interface LoginValues {
  username: string
  password: string
}

interface MenuItem {
  key: string
  path: string
  title: string
  icon: string
  roles: string[]
  children?: MenuItem[]
}

interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
    real_name?: string
    email: string
    role: string
  }
  menus: MenuItem[]
}

export default function LoginPage() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const onFinish = async (values: LoginValues) => {
    setLoading(true)
    try {
      const result = await apiClient.post<LoginResponse>('/auth/login', values)
      localStorage.setItem('token', result.token)
      if (result.menus) {
        localStorage.setItem('user_menus', JSON.stringify(result.menus))
        notifyMenusChanged()
      }
      localStorage.setItem('user_info', JSON.stringify({
        username: result.user.username,
        real_name: result.user.real_name,
        role: result.user.role,
      }))
      message.success('登录成功')
      navigate('/')
    } catch (error: unknown) {
      const errorMessage =
        (error as { response?: { data?: { error?: string; message?: string } } }).response?.data?.error ||
        (error as { response?: { data?: { error?: string; message?: string } } }).response?.data?.message ||
        '登录失败，请稍后重试'
      message.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={styles.container}>
      {/* 背景装饰 */}
      <div style={styles.background}>
        <div style={styles.grid} />
        <div style={{ ...styles.orb, ...styles.orb1 }} />
        <div style={{ ...styles.orb, ...styles.orb2 }} />
        <div style={{ ...styles.orb, ...styles.orb3 }} />
      </div>

      {/* 登录卡片 */}
      <Card style={styles.card} styles={{ body: styles.cardBody }}>
        {/* Logo 区域 */}
        <div style={styles.header}>
          <div style={styles.logo}>
            <span style={styles.logoText}>OPS</span>
          </div>
          <Title level={2} style={styles.title}>运维管理平台</Title>
          <Text style={styles.subtitle}>Operations Management Platform</Text>
        </div>

        {/* 表单区域 */}
        <Form onFinish={onFinish} style={styles.form}>
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              placeholder="用户名"
              prefix={<UserOutlined style={styles.inputIcon} />}
              size="large"
              style={styles.input}
            />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码长度至少 6 位' },
            ]}
          >
            <Input.Password
              placeholder="密码"
              prefix={<LockOutlined style={styles.inputIcon} />}
              size="large"
              style={styles.input}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              size="large"
              block
              style={styles.submitBtn}
            >
              登 录 <ArrowRightOutlined />
            </Button>
          </Form.Item>
        </Form>
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
    position: 'relative',
    overflow: 'hidden',
    background: '#f0f4f8',
  },
  background: {
    position: 'absolute',
    inset: 0,
    overflow: 'hidden',
  },
  grid: {
    position: 'absolute',
    inset: 0,
    backgroundImage: `
      linear-gradient(rgba(64, 169, 255, 0.05) 1px, transparent 1px),
      linear-gradient(90deg, rgba(64, 169, 255, 0.05) 1px, transparent 1px)
    `,
    backgroundSize: '50px 50px',
  },
  orb: {
    position: 'absolute',
    borderRadius: '50%',
    filter: 'blur(100px)',
    opacity: 0.5,
  },
  orb1: {
    width: 500,
    height: 500,
    background: 'linear-gradient(135deg, #40a9ff 0%, #096dd9 100%)',
    top: -150,
    right: -150,
  },
  orb2: {
    width: 400,
    height: 400,
    background: 'linear-gradient(135deg, #73d13d 0%, #389e0d 100%)',
    bottom: -100,
    left: -100,
  },
  orb3: {
    width: 300,
    height: 300,
    background: 'linear-gradient(135deg, #69c0ff 0%, #40a9ff 100%)',
    top: 50,
    left: -100,
  },
  card: {
    width: 420,
    borderRadius: 24,
    background: 'rgba(255, 255, 255, 0.85)',
    backdropFilter: 'blur(20px)',
    boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.15)',
    border: '1px solid rgba(255, 255, 255, 0.8)',
    position: 'relative',
    zIndex: 1,
  },
  cardBody: {
    padding: '48px 40px',
  },
  header: {
    textAlign: 'center',
    marginBottom: 40,
  },
  logo: {
    width: 72,
    height: 72,
    borderRadius: 16,
    background: 'linear-gradient(135deg, #40a9ff 0%, #096dd9 100%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '0 auto 20px',
    boxShadow: '0 8px 32px rgba(64, 169, 255, 0.25)',
  },
  logoText: {
    fontSize: 26,
    fontWeight: 700,
    color: '#fff',
    letterSpacing: 2,
  },
  title: {
    margin: 0,
    fontSize: 24,
    fontWeight: 600,
    color: '#1a1a1a',
    letterSpacing: 4,
  },
  subtitle: {
    display: 'block',
    fontSize: 12,
    color: '#8c8c8c',
    marginTop: 8,
    letterSpacing: 1,
  },
  form: {
    marginBottom: 0,
  },
  inputIcon: {
    color: '#bfbfbf',
  },
  input: {
    background: '#fafafa',
    border: '1px solid #e8e8e8',
    borderRadius: 10,
    color: '#1a1a1a',
    height: 48,
  },
  submitBtn: {
    height: 48,
    borderRadius: 10,
    background: 'linear-gradient(135deg, #40a9ff 0%, #096dd9 100%)',
    border: 'none',
    fontSize: 16,
    fontWeight: 500,
    letterSpacing: 4,
    boxShadow: '0 4px 16px rgba(64, 169, 255, 0.35)',
  },
}
