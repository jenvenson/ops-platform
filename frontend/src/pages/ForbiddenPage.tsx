// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Button, Card, Result, Space, Typography } from 'antd'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'

const { Text } = Typography

export default function ForbiddenPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const fromState = (location.state as { from?: string } | null)?.from || ''
  const fromQuery = searchParams.get('from') || ''
  const fromPath = fromState || fromQuery

  return (
    <Card>
      <Result
        status="403"
        title="没有访问权限"
        subTitle="当前账号没有访问该页面的权限，请联系管理员分配菜单或角色权限。"
        extra={
          <Space wrap>
            <Button type="primary" onClick={() => navigate('/')}>
              返回工作台
            </Button>
            <Button onClick={() => navigate('/profile')}>
              查看我的资料
            </Button>
          </Space>
        }
      />
      {fromPath ? (
        <div style={{ textAlign: 'center', marginTop: 8 }}>
          <Text type="secondary">被拦截页面：{fromPath}</Text>
        </div>
      ) : null}
    </Card>
  )
}