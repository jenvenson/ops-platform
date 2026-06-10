// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Space, Button, Select, DatePicker, message, Modal, Input, List, Typography, Popconfirm } from 'antd'
import { SearchOutlined, DeleteOutlined, ReloadOutlined, DownloadOutlined, FileZipOutlined, FolderOutlined, LinkOutlined } from '@ant-design/icons'
import { cmdbAPI, Application, deployAPI, ArchiveRecord } from '../../api/cmdb'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import useAssistantPageContext from '../../components/useAssistantPageContext'

const { RangePicker } = DatePicker

// 获取当前用户角色
const getCurrentUserRole = (): string => {
  const token = localStorage.getItem('token')
  if (token) {
    try {
      const parts = token.split('.')
      if (parts.length === 3) {
        const payload = JSON.parse(atob(parts[1]))
        return payload.role || 'user'
      }
    } catch {
      // ignore
    }
  }
  return 'user'
}

// 缓存键
const CACHE_KEYS = {
  APPLICATIONS: 'archive_history_apps',
  ENVIRONMENTS: 'archive_history_envs',
}

interface CachePayload<T> {
  data: T
  timestamp: number
}

// 从缓存读取数据
const getCachedData = <T,>(key: string): T | null => {
  try {
    const cached = localStorage.getItem(key)
    if (cached) {
      const parsed = JSON.parse(cached) as Partial<CachePayload<T>>
      if (typeof parsed?.timestamp !== 'number') {
        return null
      }
      const data = parsed.data as T
      const timestamp = parsed.timestamp
      if (Date.now() - timestamp < 5 * 60 * 1000) {
        return data
      }
    }
  } catch {
    // ignore
  }
  return null
}

// 写入缓存
const setCachedData = (key: string, data: unknown) => {
  try {
    localStorage.setItem(key, JSON.stringify({ data, timestamp: Date.now() }))
  } catch {
    // ignore
  }
}

export default function ArchiveHistoryPage() {
  const navigate = useNavigate()
  const [archives, setArchives] = useState<ArchiveRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [refreshingId, setRefreshingId] = useState<number | null>(null)
  const [selectedArchive, setSelectedArchive] = useState<ArchiveRecord | null>(null)
  const [applications, setApplications] = useState<Application[]>(() => getCachedData(CACHE_KEYS.APPLICATIONS) || [])
  const [environments, setEnvironments] = useState<{ id: number; name: string }[]>(() => getCachedData(CACHE_KEYS.ENVIRONMENTS) || [])
  const [filters, setFilters] = useState({
    appId: undefined as number | undefined,
    envId: undefined as number | undefined,
    startTime: undefined as string | undefined,
    endTime: undefined as string | undefined,
  })
  const [currentUserRole, setCurrentUserRole] = useState<string>('user')

  useAssistantPageContext({
    objectType: selectedArchive ? 'archive_record' : undefined,
    objectId: selectedArchive?.id,
    selectedRecordIds: selectedArchive ? [selectedArchive.id] : [],
    filters: {
      appId: filters.appId,
      appName: applications.find((application) => application.id === filters.appId)?.name,
      envId: filters.envId,
      envName: environments.find((environment) => environment.id === filters.envId)?.name,
      startTime: filters.startTime,
      endTime: filters.endTime,
    },
  })

  // 获取当前用户角色
  useEffect(() => {
    setCurrentUserRole(getCurrentUserRole())
  }, [])

  // 文件列表弹窗状态
  const [fileModalVisible, setFileModalVisible] = useState(false)
  const [currentFiles, setCurrentFiles] = useState<Array<{ name: string; url: string; size: number; timestamp: string }>>([])
  const [currentTimestamp, setCurrentTimestamp] = useState('')
  const [loadingFiles, setLoadingFiles] = useState(false)

  // 重置筛选条件
  const resetFilters = () => {
    setFilters({
      appId: undefined,
      envId: undefined,
      startTime: undefined,
      endTime: undefined,
    })
  }

  // 加载应用和环境数据用于下拉选择（带缓存）
  const loadAppsAndEnvs = useCallback(async () => {
    const cachedApps = getCachedData<Application[]>(CACHE_KEYS.APPLICATIONS)
    const cachedEnvs = getCachedData<{ id: number; name: string }[]>(CACHE_KEYS.ENVIRONMENTS)
    if (cachedApps && cachedEnvs && cachedApps.length > 0 && cachedEnvs.length > 0) {
      return
    }

    try {
      const [appsResp, envsResp] = await Promise.all([
        cmdbAPI.getApplications({ limit: 1000 }),
        cmdbAPI.getEnvironments({ limit: 1000 }),
      ])
      setApplications(appsResp.data)
      setEnvironments(envsResp.data.map(e => ({ id: e.id, name: e.name })))
      setCachedData(CACHE_KEYS.APPLICATIONS, appsResp.data)
      setCachedData(CACHE_KEYS.ENVIRONMENTS, envsResp.data.map(e => ({ id: e.id, name: e.name })))
    } catch (error) {
      console.error('获取数据失败:', error)
    }
  }, [])

  useEffect(() => {
    loadAppsAndEnvs()
  }, [loadAppsAndEnvs])

  // 获取归档历史数据
  const fetchArchives = async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = { limit: 100 }
      const appName = applications.find(a => a.id === filters.appId)?.name
      if (appName) {
        params.app_name = appName
      }
      if (filters.envId) params.env_id = filters.envId
      if (filters.startTime) params.start_time = filters.startTime
      if (filters.endTime) params.end_time = filters.endTime

      const resp = await deployAPI.getArchiveRecords(params)
      setArchives(resp.data)
    } catch (error) {
      console.error('获取归档历史失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchArchives()
  }, [filters.appId, filters.envId, filters.startTime, filters.endTime])

  useEffect(() => {
    if (selectedArchive && !archives.some((record) => record.id === selectedArchive.id)) {
      setSelectedArchive(null)
    }
  }, [archives, selectedArchive])

  // 手动刷新单个归档记录的状态
  const refreshRecordStatus = async (id: number) => {
    setRefreshingId(id)
    try {
      await deployAPI.getArchiveStatus(id)
      await fetchArchives()
    } catch (error) {
      console.error('刷新状态失败:', error)
    } finally {
      setRefreshingId(null)
    }
  }

  // 删除归档记录
  const deleteRecord = async (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这条归档记录吗？',
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deployAPI.deleteArchiveRecord(id)
          message.success('删除成功')
          fetchArchives()
        } catch (error) {
          console.error('删除失败:', error)
          message.error('删除失败')
        }
      },
    })
  }

  // 复制下载链接
  const copyDownloadLink = (url: string) => {
    navigator.clipboard.writeText(url)
    message.success('下载链接已复制到剪贴板')
  }

  // 查看归档文件列表
  const viewArchiveFiles = async (record: ArchiveRecord) => {
    setSelectedArchive(record)
    setFileModalVisible(true)
    setCurrentFiles([])
    setLoadingFiles(true)

    try {
      const resp = await deployAPI.getArchiveFiles(record.id)
      setCurrentFiles(resp.files || [])
      setCurrentTimestamp(resp.timestamp || '')
    } catch (error) {
      console.error('获取文件列表失败:', error)
      message.error('获取文件列表失败')
    } finally {
      setLoadingFiles(false)
    }
  }

  // 自动轮询刷新状态（当有运行中或排队中的归档时）
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const resp = await deployAPI.getArchiveRecords({ limit: 100 })
        const currentArchives = resp.data

        const activeArchives = currentArchives.filter(
          (d: ArchiveRecord) => d.status === 'running' || d.status === 'queued'
        )

        if (activeArchives.length === 0) {
          setArchives(currentArchives)
          return
        }

        // 并行刷新所有活跃归档的状态
        await Promise.all(
          activeArchives.map((d: ArchiveRecord) => deployAPI.getArchiveStatus(d.id))
        )

        const updatedResp = await deployAPI.getArchiveRecords({ limit: 100 })
        setArchives(updatedResp.data)
      } catch (error) {
        console.error('自动刷新失败:', error)
      }
    }, 5000)

    return () => clearInterval(interval)
  }, [])

  const statusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: '待执行' },
    queued: { color: 'processing', text: '排队中' },
    running: { color: 'processing', text: '归档中' },
    success: { color: 'success', text: '成功' },
    failed: { color: 'error', text: '失败' },
  }

  // 只有 admin 和 ops 角色才能看到删除按钮
  const canEdit = currentUserRole === 'admin' || currentUserRole === 'ops'

  const columns = [
    {
      title: '应用',
      dataIndex: 'app_name',
      key: 'app_name',
      width: 200,
    },
    {
      title: '环境',
      dataIndex: 'env_name',
      key: 'env_name',
      width: 120,
      render: (name: string, record: ArchiveRecord) => (
        <Tag color={record.env_type === 'prod' ? 'red' : record.env_type === 'test' ? 'orange' : 'blue'}>
          {name || '-'}
        </Tag>
      ),
    },
    {
      title: '归档类型',
      dataIndex: 'deploy_type',
      key: 'deploy_type',
      width: 100,
      render: (type: string) => {
        const typeMap: Record<string, string> = {
          all: '全部',
          frontend: '前端',
          backend: '后端',
        }
        return typeMap[type] || type
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config = statusMap[status] || { color: 'default', text: status || 'unknown' }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '归档时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '下载地址',
      dataIndex: 'download_url',
      key: 'download_url',
      width: 300,
      render: (url: string, record: ArchiveRecord) => {
        if (!url) {
          return '-'
        }
        return (
          <Space direction="vertical" size="small" style={{ width: '100%' }}>
            <div style={{ maxWidth: 280, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              <a href={url} target="_blank" rel="noopener noreferrer">{url}</a>
            </div>
            {record.status === 'success' && (
              <Space>
                <Button
                  type="primary"
                  size="small"
                  icon={<FolderOutlined />}
                  onClick={() => viewArchiveFiles(record)}
                >
                  查看文件
                </Button>
                <Button
                  size="small"
                  icon={<FileZipOutlined />}
                  onClick={() => copyDownloadLink(url)}
                >
                  复制链接
                </Button>
              </Space>
            )}
          </Space>
        )
      },
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
      width: 100,
      render: (name: string) => name || '系统',
    },
    {
      title: '操作',
      key: 'action',
      width: canEdit ? 220 : 180,
      render: (_: unknown, record: ArchiveRecord) => {
        const isActive = record.status === 'running' || record.status === 'queued'
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`/platform/events?object_type=archive_record&object_id=archive_record:deploy:${record.id}&timeline=1`)}
            >
              事件流
            </Button>
            {isActive && (
              <Button
                type="link"
                size="small"
                onClick={() => refreshRecordStatus(record.id)}
                loading={refreshingId === record.id}
              >
                刷新
              </Button>
            )}
            <a href="http://your-update-server/update/readme.html" target="_blank" rel="noopener noreferrer">
              更新说明
            </a>
            {record.jenkins_console_url && (
              <a href={record.jenkins_console_url} target="_blank" rel="noopener noreferrer">
                查看日志
              </a>
            )}
            {canEdit && (
              <Popconfirm
                title="确认删除"
                description="确定要删除这条归档记录吗？"
                onConfirm={() => deleteRecord(record.id)}
                okText="确认"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                >
                  删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        )
      },
    },
  ]

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        {/* 第一行：搜索条件 */}
        <div style={{ marginBottom: 8 }}>
          <Space wrap>
            <Input
              placeholder="应用名称"
              value={applications.find(a => a.id === filters.appId)?.name || ''}
              onChange={e => {
                const name = e.target.value.trim()
                if (!name) {
                  setFilters({ ...filters, appId: undefined })
                } else {
                  const found = applications.find(a => a.name === name)
                  if (found) {
                    setFilters({ ...filters, appId: found.id })
                  }
                }
              }}
              allowClear
              style={{ width: 180 }}
              onPressEnter={fetchArchives}
            />
            <Select
              placeholder="选择环境"
              value={filters.envId}
              onChange={val => setFilters({ ...filters, envId: val })}
              allowClear
              style={{ width: 150 }}
              options={environments.map(env => ({
                label: env.name,
                value: env.id,
              }))}
            />
            <RangePicker
              showTime
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  const start = dates[0].format('YYYY-MM-DD HH:mm:ss')
                  const end = dates[1].format('YYYY-MM-DD HH:mm:ss')
                  setFilters({
                    ...filters,
                    startTime: start,
                    endTime: end,
                  })
                } else {
                  setFilters({
                    ...filters,
                    startTime: undefined,
                    endTime: undefined,
                  })
                }
              }}
              style={{ width: 340 }}
              placeholder={['开始时间', '结束时间']}
            />
          </Space>
        </div>
        {/* 第二行：按钮 */}
        <div>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={fetchArchives}>
              搜索
            </Button>
            <Button icon={<ReloadOutlined />} onClick={resetFilters}>
              重置
            </Button>
          </Space>
        </div>
      </Card>

      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前归档历史页面上下文发起查询"
        actions={[
          { label: '最近有哪些归档失败', query: '最近有哪些归档失败' },
          { label: '最近归档是否正常完成', query: '最近归档是否正常完成' },
          { label: '如何从下载地址下载归档包', query: '如何从下载地址下载归档包' },
        ]}
      />

      <Card>
        <Table
          columns={columns}
          dataSource={archives}
          rowKey="id"
          onRow={(record) => ({
            onClick: () => setSelectedArchive(record),
          })}
          loading={loading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total: number) => `共 ${total} 条`, showQuickJumper: true }}
          scroll={{ x: 1400 }}
          locale={{ emptyText: '暂无归档记录' }}
        />
      </Card>

      {/* 文件列表弹窗 */}
      <Modal
        title={`归档文件列表 - ${formatTimestampToLocalTime(currentTimestamp)}`}
        open={fileModalVisible}
        onCancel={() => setFileModalVisible(false)}
        footer={null}
        width={600}
      >
        {loadingFiles ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : currentFiles.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            未找到归档文件
          </div>
        ) : (
          <List
            dataSource={currentFiles}
            renderItem={file => (
              <List.Item
                actions={[
                  <Button
                    type="link"
                    icon={<LinkOutlined />}
                    onClick={() => navigator.clipboard.writeText(file.url)}
                  >
                    复制链接
                  </Button>,
                  <Button
                    type="link"
                    icon={<DownloadOutlined />}
                    href={file.url}
                    target="_blank"
                  >
                    下载
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  avatar={<FileZipOutlined style={{ fontSize: 24, color: '#1890ff' }} />}
                  title={file.name}
                  description={
                    <Typography.Text type="secondary">
                      更新时间: {formatTimestampToLocalTime(file.timestamp)} | 大小: {formatFileSize(file.size)}
                    </Typography.Text>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Modal>
    </div>
  )
}

// 格式化文件大小
function formatFileSize(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null || bytes <= 0) return '未知'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 格式化时间戳为北京时间（UTC+8）- 支持多种格式
function formatTimestampToLocalTime(timestamp: string): string {
  if (!timestamp) return '-'

  // 如果是 14 位数字格式 (20260129104425)
  if (timestamp.length === 14 && /^\d{14}$/.test(timestamp)) {
    // 按 UTC 时间解析，然后加 8 小时转为北京时间
    const date = new Date(
      parseInt(timestamp.slice(0, 4)),      // year
      parseInt(timestamp.slice(4, 6)) - 1,  // month (0-indexed)
      parseInt(timestamp.slice(6, 8)),      // day
      parseInt(timestamp.slice(8, 10)),     // hour
      parseInt(timestamp.slice(10, 12)),    // minute
      parseInt(timestamp.slice(12, 14))     // second
    )
    date.setHours(date.getHours() + 8)
    return date.toLocaleString('zh-CN')
  }

  // 如果是 "YYYY-MM-DD HH:mm:ss" 格式（后端返回的 UTC 时间），加 8 小时转为北京时间
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(timestamp)) {
    // 后端返回的时间是 UTC 时间，需要加 8 小时转为北京时间
    const date = new Date(timestamp.replace(/-/g, '/'))
    date.setHours(date.getHours() + 8)
    return date.toLocaleString('zh-CN')
  }

  // 如果是 ISO 格式（带时区），转换为北京时间
  if (timestamp.includes('T') || timestamp.includes('-')) {
    const date = new Date(timestamp)
    if (!isNaN(date.getTime())) {
      // 转换为 UTC+8
      const beijingTime = new Date(date.getTime() + 8 * 60 * 60 * 1000)
      return beijingTime.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
    }
  }

  return timestamp
}