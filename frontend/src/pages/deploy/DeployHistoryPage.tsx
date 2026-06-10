// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Space, Button, Select, DatePicker, message, Modal, Input } from 'antd'
import { SearchOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { cmdbAPI, Application } from '../../api/cmdb'
import { deployAPI, DeployRecord } from '../../api/cmdb.js'
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
  APPLICATIONS: 'deploy_history_apps',
  ENVIRONMENTS: 'deploy_history_envs',
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
      // 缓存5分钟
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

export default function DeployHistoryPage() {
  const navigate = useNavigate()
  const [deployments, setDeployments] = useState<DeployRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [refreshingId, setRefreshingId] = useState<number | null>(null)
  const [selectedDeploy, setSelectedDeploy] = useState<DeployRecord | null>(null)
  const [applications, setApplications] = useState<Application[]>(() => getCachedData(CACHE_KEYS.APPLICATIONS) || [])
  const [environments, setEnvironments] = useState<{ id: number; name: string }[]>(() => getCachedData(CACHE_KEYS.ENVIRONMENTS) || [])
  const [filters, setFilters] = useState({
    appId: undefined as number | undefined,
    envId: undefined as number | undefined,
    status: undefined as string | undefined,
    triggeredBy: undefined as string | undefined,
    startTime: undefined as string | undefined,
    endTime: undefined as string | undefined,
  })
  const [currentUserRole, setCurrentUserRole] = useState<string>('user')

  useAssistantPageContext({
    objectType: selectedDeploy ? 'deploy_record' : undefined,
    objectId: selectedDeploy?.id,
    selectedRecordIds: selectedDeploy ? [selectedDeploy.id] : [],
    filters: {
      appId: filters.appId,
      appName: applications.find((application) => application.id === filters.appId)?.name,
      envId: filters.envId,
      envName: environments.find((environment) => environment.id === filters.envId)?.name,
      status: filters.status,
      triggeredBy: filters.triggeredBy,
      startTime: filters.startTime,
      endTime: filters.endTime,
    },
  })

  // 获取当前用户角色
  useEffect(() => {
    setCurrentUserRole(getCurrentUserRole())
  }, [])

  // 重置筛选条件
  const resetFilters = () => {
    setFilters({
      appId: undefined,
      envId: undefined,
      status: undefined,
      triggeredBy: undefined,
      startTime: undefined,
      endTime: undefined,
    })
  }

  // 加载应用和环境数据用于下拉选择（带缓存）
  const loadAppsAndEnvs = useCallback(async () => {
    // 如果缓存有效，直接返回
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
      // 写入缓存
      setCachedData(CACHE_KEYS.APPLICATIONS, appsResp.data)
      setCachedData(CACHE_KEYS.ENVIRONMENTS, envsResp.data.map(e => ({ id: e.id, name: e.name })))
    } catch (error) {
      console.error('获取数据失败:', error)
    }
  }, [])

  useEffect(() => {
    loadAppsAndEnvs()
  }, [loadAppsAndEnvs])

  // 获取部署历史数据
  const fetchDeployments = async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = { limit: 100 }
      // 应用名称模糊查询（用户输入应用名称）
      const appName = applications.find(a => a.id === filters.appId)?.name
      if (appName) {
        params.app_name = appName
      }
      if (filters.envId) params.env_id = filters.envId
      if (filters.status) params.status = filters.status
      if (filters.triggeredBy) params.triggered_by = filters.triggeredBy
      if (filters.startTime) params.start_time = filters.startTime
      if (filters.endTime) params.end_time = filters.endTime

      const resp = await deployAPI.getDeployRecords(params)
      setDeployments(resp.data)
    } catch (error) {
      console.error('获取部署历史失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDeployments()
  }, [filters.appId, filters.envId, filters.status, filters.triggeredBy, filters.startTime, filters.endTime])

  useEffect(() => {
    if (selectedDeploy && !deployments.some((record) => record.id === selectedDeploy.id)) {
      setSelectedDeploy(null)
    }
  }, [deployments, selectedDeploy])

  // 手动刷新单个部署记录的状态
  const refreshRecordStatus = async (id: number) => {
    setRefreshingId(id)
    try {
      await deployAPI.getDeployStatus(id)
      // 刷新后重新获取列表
      await fetchDeployments()
    } catch (error) {
      console.error('刷新状态失败:', error)
    } finally {
      setRefreshingId(null)
    }
  }

  // 删除部署记录
  const deleteRecord = async (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这条部署记录吗？',
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deployAPI.deleteDeployRecord(id)
          message.success('删除成功')
          fetchDeployments()
        } catch (error) {
          console.error('删除失败:', error)
          message.error('删除失败')
        }
      },
    })
  }

  // 自动轮询刷新状态（当有运行中或排队中的部署时）
  useEffect(() => {
    const interval = setInterval(async () => {
      // 每次从 API 获取最新列表，检查是否有活跃部署
      try {
        const resp = await deployAPI.getDeployRecords({ limit: 100 })
        const currentDeployments = resp.data

        const activeDeployments = currentDeployments.filter(
          (d: DeployRecord) => d.status === 'running' || d.status === 'queued'
        )

        if (activeDeployments.length === 0) {
          // 没有活跃部署，停止轮询
          setDeployments(currentDeployments)
          return
        }

        // 并行刷新所有活跃部署的状态
        await Promise.all(
          activeDeployments.map((d: DeployRecord) => deployAPI.getDeployStatus(d.id))
        )

        // 重新获取最新列表
        const updatedResp = await deployAPI.getDeployRecords({ limit: 100 })
        setDeployments(updatedResp.data)
      } catch (error) {
        console.error('自动刷新失败:', error)
      }
    }, 5000) // 每5秒刷新

    return () => clearInterval(interval)
  }, [])

  const statusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: '待执行' },
    queued: { color: 'processing', text: '排队中' },
    running: { color: 'processing', text: '部署中' },
    success: { color: 'success', text: '成功' },
    failed: { color: 'error', text: '失败' },
  }

  const columns = [
    {
      title: '应用',
      dataIndex: 'app_name',
      key: 'app_name',
      width: 250,
    },
    {
      title: '环境',
      dataIndex: 'env_name',
      key: 'env_name',
      width: 120,
      render: (name: string, record: DeployRecord) => (
        <Tag color={record.env_type === 'prod' ? 'red' : record.env_type === 'test' ? 'orange' : 'blue'}>
          {name || '-'}
        </Tag>
      ),
    },
    {
      title: '部署类型',
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
      title: '触发时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '耗时',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration: number) => {
        if (!duration) return '-'
        if (duration < 60) return `${duration}秒`
        if (duration < 3600) return `${Math.floor(duration / 60)}分${duration % 60}秒`
        return `${Math.floor(duration / 3600)}小时${Math.floor((duration % 3600) / 60)}分`
      },
    },
    {
      title: '控制台日志',
      key: 'console_url',
      width: 100,
      render: (_: unknown, record: DeployRecord) => {
        if (record.jenkins_console_url) {
          return (
            <a href={record.jenkins_console_url} target="_blank" rel="noopener noreferrer">
              查看日志
            </a>
          )
        }
        return '-'
      },
    },
    {
      title: '操作人',
      dataIndex: 'triggered_by_name',
      key: 'triggered_by_name',
      width: 100,
      render: (name: string, record: DeployRecord) => name || record.triggered_by || '系统',
    },
  ]

  // 只有 admin 和 ops 角色才能看到操作列
  const canEdit = currentUserRole === 'admin' || currentUserRole === 'ops'
  if (canEdit) {
    columns.push({
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, record: DeployRecord) => {
        const isActive = record.status === 'running' || record.status === 'queued'
        return (
          <Space size="small">
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`/platform/events?object_type=deploy_record&object_id=deploy_record:deploy:${record.id}&timeline=1`)}
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
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => deleteRecord(record.id)}
            >
              删除
            </Button>
          </Space>
        )
      },
    })
  }

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        {/* 搜索条件第一行 */}
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
              onPressEnter={fetchDeployments}
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
            <Select
              placeholder="状态"
              value={filters.status}
              onChange={val => setFilters({ ...filters, status: val })}
              allowClear
              style={{ width: 120 }}
              options={[
                { label: '排队中', value: 'queued' },
                { label: '部署中', value: 'running' },
                { label: '成功', value: 'success' },
                { label: '失败', value: 'failed' },
              ]}
            />
            <Input
              placeholder="操作人"
              value={filters.triggeredBy}
              onChange={e => setFilters({ ...filters, triggeredBy: e.target.value })}
              allowClear
              style={{ width: 120 }}
            />
            <RangePicker
              showTime
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  // 使用本地时区的日期时间格式
                  const start = dates[0].format('YYYY-MM-DD HH:mm:ss')
                  const end = dates[1].format('YYYY-MM-DD HH:mm:ss')
                  setFilters({
                    ...filters,
                    startTime: start,
                    endTime: end,
                  })
                } else {
                  setFilters({ ...filters, startTime: undefined, endTime: undefined })
                }
              }}
              placeholder={['开始时间', '结束时间']}
              style={{ width: 380 }}
            />
          </Space>
        </div>
        {/* 按钮第二行 */}
        <div>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={fetchDeployments}>
              搜索
            </Button>
            <Button icon={<ReloadOutlined />} onClick={resetFilters}>
              重置
            </Button>
          </Space>
        </div>
      </Card>

      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前部署记录页面上下文发起查询"
        actions={[
          { label: '最近有哪些失败部署', query: '最近有哪些失败部署' },
          { label: '当前还有哪些部署在执行中', query: '当前还有哪些部署在执行中' },
          { label: '最近部署异常集中在哪些应用', query: '最近部署异常集中在哪些应用' },
        ]}
      />

      <Card title="部署记录">
        <Table
          columns={columns}
          dataSource={deployments}
          rowKey="id"
          onRow={(record) => ({
            onClick: () => setSelectedDeploy(record),
          })}
          loading={loading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total: number) => `共 ${total} 条`, showQuickJumper: true }}
          scroll={{ x: 1400 }}
          locale={{ emptyText: '暂无部署记录' }}
        />
      </Card>
    </div>
  )
}