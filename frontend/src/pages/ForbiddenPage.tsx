// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Button, Card, Result, Space, Typography } from 'antd'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

const { Text } = Typography

export default function ForbiddenPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const { t } = useTranslation('admin')
  const fromState = (location.state as { from?: string } | null)?.from || ''
  const fromQuery = searchParams.get('from') || ''
  const fromPath = fromState || fromQuery

  return (
    <Card>
      <Result
        status="403"
        title={t('forbidden', '没有访问权限')}
        subTitle={t('forbiddenHint', '当前账号没有访问该页面的权限，请联系管理员分配菜单或角色权限。')}
        extra={
          <Space wrap>
            <Button type="primary" onClick={() => navigate('/')}>
              {t('returnToDashboard', '返回工作台')}
            </Button>
            <Button onClick={() => navigate('/profile')}>
              {t('viewMyProfile', '查看我的资料')}
            </Button>
          </Space>
        }
      />
      {fromPath ? (
        <div style={{ textAlign: 'center', marginTop: 8 }}>
          <Text type="secondary">{t('blockedPage', '被拦截页面')}：{fromPath}</Text>
        </div>
      ) : null}
    </Card>
  )
}
