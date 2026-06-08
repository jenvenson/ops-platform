import { useState, useEffect } from 'react'
import { Card, Descriptions, Button, Modal, Form, Input, message, Tag, Divider, Space } from 'antd'
import { KeyOutlined, UserOutlined, MailOutlined, SafetyOutlined, CalendarOutlined, IdcardOutlined } from '@ant-design/icons'
import { adminAPI, User } from '../api/admin'

// 角色名称映射
const roleNames: Record<string, string> = {
  admin: '超级管理员',
  ops: '运维人员',
  dev: '开发人员',
  qa: '测试人员',
  user: '普通用户',
}

// 角色颜色映射
const roleColors: Record<string, string> = {
  admin: 'red',
  ops: 'green',
  dev: 'blue',
  qa: 'orange',
  user: 'default',
}

export default function ProfilePage() {
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
      message.error('获取用户信息失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUserInfo()
  }, [])

  // 修改密码
  const handleChangePassword = async () => {
    try {
      const values = await passwordForm.validateFields()
      setSubmitting(true)
      await adminAPI.changePassword({
        old_password: values.oldPassword,
        new_password: values.newPassword,
      })
      message.success('密码修改成功，下次登录请使用新密码')
      setPasswordModalVisible(false)
      passwordForm.resetFields()
    } catch (error: any) {
      const errMsg = error?.response?.data?.error || '密码修改失败'
      message.error(errMsg)
    } finally {
      setSubmitting(false)
    }
  }

  const formatTime = (time: string) => {
    return time ? new Date(time).toLocaleString('zh-CN') : '-'
  }

  return (
    <div>
      <Card title="个人信息" loading={loading}>
        {userInfo && (
          <Descriptions
            bordered
            column={{ xs: 1, sm: 1, md: 2 }}
            labelStyle={{ width: 140, fontWeight: 500 }}
          >
            <Descriptions.Item label={<><UserOutlined style={{ marginRight: 8 }} />用户名</>}>
              {userInfo.username}
            </Descriptions.Item>
            <Descriptions.Item label={<><IdcardOutlined style={{ marginRight: 8 }} />姓名</>}>
              {userInfo.real_name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={<><SafetyOutlined style={{ marginRight: 8 }} />角色</>}>
              <Tag color={roleColors[userInfo.role] || 'default'}>
                {roleNames[userInfo.role] || userInfo.role}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={<><MailOutlined style={{ marginRight: 8 }} />邮箱</>}>
              {userInfo.email || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={<><CalendarOutlined style={{ marginRight: 8 }} />创建时间</>}>
              {formatTime(userInfo.created_at)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Card>

      <Divider />

      <Card title="安全设置">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <div style={{ fontWeight: 500, marginBottom: 4 }}>登录密码</div>
              <div style={{ color: '#999', fontSize: 13 }}>
                定期修改密码可以提高账户安全性
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
              修改密码
            </Button>
          </div>
        </Space>
      </Card>

      {/* 修改密码模态框 */}
      <Modal
        title="修改密码"
        open={passwordModalVisible}
        onCancel={() => setPasswordModalVisible(false)}
        onOk={handleChangePassword}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="oldPassword"
            label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password placeholder="请输入当前密码" />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度至少6位' },
            ]}
          >
            <Input.Password placeholder="请输入新密码（至少6位）" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认新密码"
            dependencies={['newPassword']}
            rules={[
              { required: true, message: '请再次输入新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
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
