// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { Card, Button, Table } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import {
  PrometheusServer,
  fetchServerStatusesData,
  formatBytes,
  formatBytesRate,
  formatBitsRate,
  getPercentColor,
  getDiskRateColor,
  getBandwidthColor,
} from './monitorShared'

export default function MonitorOverviewPage() {
  const [serverStatuses, setServerStatuses] = useState<PrometheusServer[]>([])
  const [serversLoading, setServersLoading] = useState(false)

  const fetchServerStatuses = useCallback(async () => {
    setServersLoading(true)
    try {
      const servers = await fetchServerStatusesData()
      setServerStatuses(servers)
    } catch (err) {
      console.error('获取服务器状态失败:', err)
    } finally {
      setServersLoading(false)
    }
  }, [])

  // 初始加载
  useEffect(() => {
    fetchServerStatuses()
  }, [fetchServerStatuses])

  // 每 60 秒自动刷新
  useEffect(() => {
    const timer = setInterval(() => {
      fetchServerStatuses()
    }, 60000)
    return () => clearInterval(timer)
  }, [fetchServerStatuses])

  return (
    <div>
      <Card
        title={`服务器资源总览【主机总数：${serverStatuses.length}】`}
        size="small"
        style={{ marginBottom: 16 }}
        extra={
          <Button
            size="small"
            icon={<ReloadOutlined spin={serversLoading} />}
            onClick={fetchServerStatuses}
          >
            刷新
          </Button>
        }
      >
        <Table
          className="monitor-overview-table"
          columns={[
            {
              title: '名称',
              dataIndex: 'name',
              key: 'name',
              width: 145,
              fixed: 'left',
              sorter: (a: PrometheusServer, b: PrometheusServer) => a.name.localeCompare(b.name),
              render: (name: string) => <span style={{ fontWeight: 500 }}>{name}</span>,
            },
            {
              title: 'IP',
              dataIndex: 'ip',
              key: 'ip',
              width: 130,
              sorter: (a: PrometheusServer, b: PrometheusServer) => {
                const toNum = (ip: string) => ip.split('.').reduce((acc, oct) => acc * 256 + parseInt(oct, 10), 0)
                return toNum(a.ip) - toNum(b.ip)
              },
            },
            {
              title: '启动(天)',
              dataIndex: 'uptimeDays',
              key: 'uptimeDays',
              width: 80,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.uptimeDays ?? 0) - (b.uptimeDays ?? 0),
              render: (days: number | undefined) => days !== undefined ? days.toFixed(1) : '-',
            },
            {
              title: '内存',
              dataIndex: 'memTotal',
              key: 'memTotal',
              width: 80,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.memTotal ?? 0) - (b.memTotal ?? 0),
              render: (bytes: number | undefined) => {
                if (bytes === undefined) return '-'
                return <span style={{ color: '#1890ff', fontWeight: 500 }}>{formatBytes(bytes, 0)}</span>
              },
            },
            {
              title: 'CPU',
              dataIndex: 'cpuCores',
              key: 'cpuCores',
              width: 55,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.cpuCores ?? 0) - (b.cpuCores ?? 0),
              render: (cores: number | undefined) => {
                if (cores === undefined) return '-'
                return <span style={{ color: '#1890ff', fontWeight: 500 }}>{Math.round(cores)}</span>
              },
            },
            {
              title: '负载',
              dataIndex: 'load5',
              key: 'load5',
              width: 70,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.load5 ?? 0) - (b.load5 ?? 0),
              render: (load: number | undefined) => load !== undefined ? load.toFixed(2) : '-',
            },
            {
              title: 'CPU使用率',
              dataIndex: 'cpuUsage',
              key: 'cpuUsage',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.cpuUsage ?? 0) - (b.cpuUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '内存使用率',
              dataIndex: 'memUsage',
              key: 'memUsage',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.memUsage ?? 0) - (b.memUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: 'IOutil使用率*',
              dataIndex: 'ioUtil',
              key: 'ioUtil',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.ioUtil ?? 0) - (b.ioUtil ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '分区使用率*',
              dataIndex: 'diskUsage',
              key: 'diskUsage',
              width: 120,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskUsage ?? 0) - (b.diskUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '磁盘读取*',
              dataIndex: 'diskRead',
              key: 'diskRead',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskRead ?? 0) - (b.diskRead ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getDiskRateColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBytesRate(val)}
                  </span>
                )
              },
            },
            {
              title: '磁盘写入*',
              dataIndex: 'diskWrite',
              key: 'diskWrite',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskWrite ?? 0) - (b.diskWrite ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getDiskRateColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBytesRate(val)}
                  </span>
                )
              },
            },
            {
              title: '下载带宽*',
              dataIndex: 'netDown',
              key: 'netDown',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.netDown ?? 0) - (b.netDown ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getBandwidthColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBitsRate(val)}
                  </span>
                )
              },
            },
            {
              title: '上传带宽*',
              dataIndex: 'netUp',
              key: 'netUp',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.netUp ?? 0) - (b.netUp ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getBandwidthColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBitsRate(val)}
                  </span>
                )
              },
            },
          ]}
          dataSource={serverStatuses}
          rowKey="instance"
          loading={serversLoading}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total) => `共 ${total} 台`,
            showQuickJumper: true,
          }}
          size="small"
          scroll={{ x: 1600 }}
        />
      </Card>
    </div>
  )
}