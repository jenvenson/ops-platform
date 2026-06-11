// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { Card, Button, Space, Table, Tag } from 'antd'
import {
  DashboardOutlined,
  ExpandOutlined,
  ArrowLeftOutlined,
  LinkOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { monitorAPI, GrafanaDashboard } from '../../api/monitor.js'
import { getGrafanaProxyBase, openGrafana } from './monitorShared'

export default function MonitorDashboardsPage() {
  const { t } = useTranslation('monitor')
  const { t: tc } = useTranslation('common')
  const [loading, setLoading] = useState(true)
  const [dashboards, setDashboards] = useState<GrafanaDashboard[]>([])
  const [selectedDashboard, setSelectedDashboard] = useState<string>('')

  const fetchGrafanaData = useCallback(async () => {
    try {
      const dashboardsData = await monitorAPI.grafana.getDashboards().catch(() => [])
      setDashboards(dashboardsData)
    } catch (err) {
      console.error('获取仪表盘数据失败:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchGrafanaData()
  }, [fetchGrafanaData])

  const handleDashboardClick = (uid: string) => {
    setSelectedDashboard(uid)
  }

  // 仪表盘详情视图（嵌入 iframe）
  if (selectedDashboard) {
    const dashboard = dashboards.find(d => d.uid === selectedDashboard)
    const proxyBase = getGrafanaProxyBase()

    return (
      <div>
        <Space style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => setSelectedDashboard('')}>
            {t('backToList', '返回列表')}
          </Button>
          <Button
            type="primary"
            icon={<ExpandOutlined />}
            onClick={() => openGrafana(`/d/${selectedDashboard}`)}
          >
            {t('openInGrafana', '在 Grafana 中打开')}
          </Button>
        </Space>
        <Card title={dashboard?.title || t('dashboard', '仪表盘')} size="small">
          <iframe
            src={`${proxyBase}/d/${selectedDashboard}?orgId=1&kiosk`}
            width="100%"
            height={750}
            style={{ border: 'none', borderRadius: 4 }}
            title="Grafana Dashboard"
          />
        </Card>
      </div>
    )
  }

  // 仪表盘列表视图
  return (
    <div>
      <Card title={t('grafanaDashboards', 'Grafana 仪表盘')} size="small">
        <div style={{ marginBottom: 16 }}>
          <Button
            type="primary"
            icon={<LinkOutlined />}
            onClick={() => openGrafana('/dashboards')}
          >
            {t('viewAllInGrafana', '在 Grafana 中查看全部')}
          </Button>
        </div>
        <Table
          columns={[
            {
              title: t('dashboardName', '仪表盘名称'),
              dataIndex: 'title',
              key: 'title',
              render: (title: string, record: GrafanaDashboard) => (
                <a onClick={() => handleDashboardClick(record.uid)} style={{ fontWeight: 500 }}>
                  <DashboardOutlined style={{ marginRight: 8 }} />
                  {title}
                </a>
              ),
            },
            {
              title: t('tags', '标签'),
              dataIndex: 'tags',
              key: 'tags',
              width: 250,
              render: (tags: string[]) => (
                <Space wrap>
                  {(tags || []).slice(0, 4).map(tag => <Tag key={tag}>{tag}</Tag>)}
                  {tags && tags.length > 4 && <Tag>+{tags.length - 4}</Tag>}
                </Space>
              ),
            },
            {
              title: tc('action', '操作'),
              key: 'action',
              width: 180,
              render: (_: unknown, record: GrafanaDashboard) => (
                <Space>
                  <Button
                    type="link"
                    size="small"
                    onClick={() => handleDashboardClick(record.uid)}
                  >
                    {t('embedView', '嵌入查看')}
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<ExpandOutlined />}
                    onClick={() => openGrafana(record.url)}
                  >
                    {t('newWindow', '新窗口')}
                  </Button>
                </Space>
              ),
            },
          ]}
          dataSource={dashboards}
          rowKey="id"
          loading={loading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
        />
      </Card>
    </div>
  )
}
