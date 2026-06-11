// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Layout, Menu } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

const { Header, Content, Sider } = Layout

export default function CmdbPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const { t } = useTranslation('cmdb')
  const { t: tm } = useTranslation('menu')

  const getSelectedKey = () => {
    if (location.pathname === '/cmdb/projects') return 'projects'
    if (location.pathname === '/cmdb/clusters') return 'clusters'
    if (location.pathname === '/cmdb/servers') return 'servers'
    if (location.pathname === '/cmdb/applications') return 'applications'
    return 'projects'
  }

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(`/cmdb/${key}`)
  }

  useEffect(() => {
    if (location.pathname === '/cmdb') {
      navigate('/cmdb/projects')
    }
  }, [location.pathname, navigate])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={200} theme="light">
        <div style={{ height: 48, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '16px', fontWeight: 'bold', borderBottom: '1px solid #f0f0f0' }}>
          {t('sidebarTitle', '运维管理平台')}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[getSelectedKey()]}
          onClick={handleMenuClick}
          theme="light"
          items={[
            { key: 'projects', label: tm('cmdb-projects', '项目管理') },
            { key: 'clusters', label: t('clusters', '集群管理') },
            { key: 'servers', label: tm('cmdb-servers', '主机管理') },
            { key: 'applications', label: tm('cmdb-applications', '应用流水线管理') },
          ]}
        />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 24px', borderBottom: '1px solid #f0f0f0' }}>
          <span style={{ fontSize: '18px', fontWeight: 'bold' }}>{t('pageTitle', 'CMDB 资产管理')}</span>
        </Header>
        <Content style={{ padding: '24px', background: '#f0f2f5' }}>
          <div style={{ padding: '24px', background: '#fff', borderRadius: '8px', minHeight: '400px' }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}