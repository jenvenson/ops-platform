// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Steps, Select, Button, Table, Tag, message, Space, Empty, Result } from 'antd'
import { HistoryOutlined, SyncOutlined, CheckCircleOutlined, FileZipOutlined } from '@ant-design/icons'
import { cmdbAPI, Application, Project, deployAPI } from '../../api/cmdb'

type DeployType = 'all' | 'frontend' | 'backend'

export default function ArchivePage() {
  const navigate = useNavigate()
  const [currentStep, setCurrentStep] = useState(0)
  const [projects, setProjects] = useState<Project[]>([])
  const [applications, setApplications] = useState<Application[]>([])
  const [selectedProjectId, setSelectedProjectId] = useState<number | undefined>()
  const [selectedApps, setSelectedApps] = useState<Application[]>([])
  const [deployType, setDeployType] = useState<DeployType>('all')
  const [archiving, setArchiving] = useState(false)
  const [archiveResults, setArchiveResults] = useState<{ app: Application; triggered: boolean; message: string; archiveRecordId?: number; deployType: string }[]>([])
  const [loading, setLoading] = useState(false)

  const getArchiveBlockReason = (app: Application) => {
    if (!app.env_id) {
      return '未配置归档环境'
    }
    if (!app.jenkins_archive_job) {
      return '未配置 Jenkins 归档流水线'
    }
    return ''
  }

  // 使用 useMemo 自动过滤应用
  const projectApps = useMemo(() => {
    if (!selectedProjectId) {
      return []
    }
    return applications.filter(app => {
      if (!app.project_id || app.project_id === 0) {
        return false
      }
      return Number(app.project_id) === Number(selectedProjectId)
    })
  }, [selectedProjectId, applications])

  // 加载项目和应用数据
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const [projectsResp, appsResp] = await Promise.all([
          cmdbAPI.getProjects({ limit: 1000 }),
          cmdbAPI.getApplications({ limit: 1000 }),
        ])
        setProjects(projectsResp.data)
        setApplications(appsResp.data)
      } catch (error) {
        console.error('加载数据失败:', error)
        message.error('加载数据失败')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  // 选择项目
  const handleProjectSelect = (projectId: number | string) => {
    const pid = Number(projectId)
    setSelectedProjectId(pid)
    setSelectedApps([])
  }

  // 下一步
  const handleNext = () => {
    if (currentStep === 0 && !selectedProjectId) {
      message.warning('请选择项目')
      return
    }
    if (currentStep === 1 && selectedApps.length === 0) {
      message.warning('请选择要归档的应用')
      return
    }
    if (currentStep === 1 && selectedApps.length > 3) {
      message.warning('最多支持3个应用并发归档')
      return
    }
    setCurrentStep(currentStep + 1)
  }

  // 上一步
  const handlePrev = () => {
    setCurrentStep(currentStep - 1)
  }

  // 执行归档
  const handleArchive = async () => {
    setArchiving(true)
    const results: { app: Application; triggered: boolean; message: string; archiveRecordId?: number; deployType: string }[] = []

    for (const app of selectedApps) {
      const blockReason = getArchiveBlockReason(app)
      if (blockReason) {
        results.push({
          app,
          triggered: false,
          message: blockReason,
          deployType,
        })
        continue
      }

      try {
        const resp = await deployAPI.triggerArchive({
          app_id: app.id,
          env_id: app.env_id || 0,
          deploy_type: deployType,
        })
        results.push({
          app,
          triggered: resp.success,
          message: resp.success ? '归档中' : (resp.message || '触发失败'),
          archiveRecordId: resp.deploy_id,
          deployType,
        })
      } catch (error: unknown) {
        const err = error as { response?: { data?: { error?: string } } }
        results.push({
          app,
          triggered: false,
          message: err.response?.data?.error || '触发失败',
          deployType,
        })
      }
    }

    setArchiveResults(results)
    setArchiving(false)
    setCurrentStep(3)
  }

  // 跳转到归档历史
  const handleViewHistory = () => {
    navigate('/deploy/archived')
  }

  // 重新归档
  const handleRestart = () => {
    setCurrentStep(0)
    setSelectedProjectId(undefined)
    setSelectedApps([])
    setArchiveResults([])
  }

  // 归档完成后自动刷新状态
  useEffect(() => {
    if (currentStep !== 3 || archiveResults.length === 0) return

    const checkStatus = async () => {
      const currentResults = [...archiveResults]
      let hasUpdate = false

      for (let i = 0; i < currentResults.length; i++) {
        const result = currentResults[i]
        if (!result.triggered || !result.archiveRecordId) continue

        try {
          const resp = await deployAPI.getArchiveStatus(result.archiveRecordId)
          if (resp.status === 'success') {
            currentResults[i] = {
              ...result,
              triggered: false,
              message: '归档成功',
            }
            hasUpdate = true
          } else if (resp.status === 'failed') {
            currentResults[i] = {
              ...result,
              triggered: false,
              message: '归档失败',
            }
            hasUpdate = true
          }
        } catch (e) {
          console.error('检查归档状态失败:', e)
        }
      }

      if (hasUpdate) {
        setArchiveResults(currentResults)
      }
    }

    // 立即执行一次
    checkStatus()

    // 每3秒轮询
    const interval = setInterval(checkStatus, 3000)

    return () => clearInterval(interval)
  }, [currentStep])

  // 应用选择表格列
  const appColumns = [
    {
      title: '应用名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '项目环境',
      dataIndex: 'environment',
      key: 'environment',
      render: (env: Application['environment']) => env?.name ? <Tag color="blue">{env.name}</Tag> : '-',
    },
    {
      title: 'Jenkins归档流水线',
      dataIndex: 'jenkins_archive_job',
      key: 'jenkins_archive_job',
      ellipsis: true,
      render: (value: string) => value || <Tag color="error">未配置</Tag>,
    },
    {
      title: '可归档状态',
      key: 'archive_ready',
      width: 180,
      render: (_: unknown, record: Application) => {
        const reason = getArchiveBlockReason(record)
        if (!reason) {
          return <Tag color="success">可归档</Tag>
        }
        return <Tag color="warning">{reason}</Tag>
      },
    },
  ]

  // 归档结果表格列
  const resultColumns = [
    {
      title: '应用名称',
      dataIndex: ['app', 'name'],
      key: 'name',
    },
    {
      title: '项目环境',
      key: 'environment',
      render: (_: unknown, record: { app: Application }) =>
        record.app.environment?.name ? <Tag color="blue">{record.app.environment.name}</Tag> : '-',
    },
    {
      title: '归档类型',
      key: 'deployType',
      width: 120,
      render: (_: unknown, record: { app: Application; triggered: boolean; message: string; archiveRecordId?: number; deployType: string }) => {
        const typeMap: Record<string, { color: string; text: string }> = {
          all: { color: 'purple', text: '全部' },
          frontend: { color: 'cyan', text: '前端' },
          backend: { color: 'blue', text: '后端' },
        }
        const config = typeMap[record.deployType] || { color: 'default', text: record.deployType }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '状态',
      key: 'status',
      render: (_: unknown, record: { app: Application; triggered: boolean; message: string; archiveRecordId?: number; deployType: string }) => {
        if (record.triggered) {
          return <Tag icon={<SyncOutlined spin />} color="processing">归档中</Tag>
        }
        if (record.message === '归档成功') {
          return <Tag color="success">成功</Tag>
        }
        if (record.message === '归档失败') {
          return <Tag color="error">失败</Tag>
        }
        return <Tag color="error">{record.message}</Tag>
      },
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
    },
  ]

  const steps = [
    { title: '选择项目', description: '选择要归档的项目' },
    { title: '选择应用', description: '选择要归档的应用' },
    { title: '确认归档', description: '确认并执行归档' },
    { title: '完成', description: '归档结果' },
  ]

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Steps current={currentStep} items={steps} />
      </Card>

      <Card>
        {/* 步骤 1: 选择项目 */}
        {currentStep === 0 && (
          <div style={{ padding: '40px 0', textAlign: 'center' }}>
            <h3 style={{ marginBottom: 8 }}>请选择要归档的项目</h3>
            <div style={{ marginBottom: 24, fontSize: '12px', color: '#999' }}>
              选择项目后，将显示该项目的所有应用
            </div>
            <Select
              placeholder="选择项目"
              value={selectedProjectId}
              onChange={handleProjectSelect}
              style={{ width: 400 }}
              size="large"
              showSearch
              optionFilterProp="children"
              loading={loading}
            >
              {projects.map(p => (
                <Select.Option key={p.id} value={p.id}>
                  {p.name} ({p.code})
                </Select.Option>
              ))}
            </Select>
            <div style={{ marginTop: 40 }}>
              <Button type="primary" size="large" onClick={handleNext} disabled={!selectedProjectId}>
                下一步
              </Button>
            </div>
          </div>
        )}

        {/* 步骤 2: 选择应用 */}
        {currentStep === 1 && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <span style={{ marginRight: 16 }}>已选择项目: <Tag color="blue">{projects.find(p => p.id === selectedProjectId)?.name}</Tag></span>
            </div>
            <div style={{ marginBottom: 16, fontSize: '12px', color: '#666', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
              提示：仅可选择已配置归档环境和 Jenkins 归档流水线的应用。
            </div>
            {projectApps.length === 0 ? (
              <Empty
                description={
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: '16px', marginBottom: '8px' }}>该项目下没有关联的应用</div>
                    <div style={{ fontSize: '12px', color: '#999' }}>
                      请在"应用管理"页面中为应用设置"所属项目"
                    </div>
                  </div>
                }
              />
            ) : (
              <Table
                rowSelection={{
                  type: 'checkbox',
                  selectedRowKeys: selectedApps.map(a => a.id),
                  getCheckboxProps: (record: Application) => ({
                    disabled: Boolean(getArchiveBlockReason(record)),
                  }),
                  onChange: (_, selectedRows) => {
                    if (selectedRows.length > 3) {
                      message.warning('最多支持3个应用并发归档，已自动限制为前3个')
                      setSelectedApps(selectedRows.slice(0, 3) as Application[])
                    } else {
                      setSelectedApps(selectedRows)
                    }
                  },
                }}
                columns={appColumns}
                dataSource={projectApps}
                rowKey="id"
                pagination={false}
              />
            )}
            {selectedApps.length > 0 && (
              <div style={{ marginTop: 16, padding: '8px 12px', background: '#f6ffed', borderRadius: 4 }}>
                已选择 <strong>{selectedApps.length}</strong> 个应用
              </div>
            )}
            <div style={{ marginTop: 24, textAlign: 'center' }}>
              <Space size="large">
                <Button size="large" onClick={handlePrev}>上一步</Button>
                <Button type="primary" size="large" onClick={handleNext} disabled={selectedApps.length === 0 || selectedApps.length > 3}>
                  下一步 ({selectedApps.length} 个应用)
                </Button>
              </Space>
            </div>
          </div>
        )}

        {/* 步骤 3: 确认归档 */}
        {currentStep === 2 && (
          <div style={{ padding: '20px 0' }}>
            <h3 style={{ marginBottom: 16 }}>确认归档信息</h3>
            <div style={{ marginBottom: 24 }}>
              <p><strong>项目:</strong> {projects.find(p => p.id === selectedProjectId)?.name}</p>
              <p><strong>应用数量:</strong> {selectedApps.length} 个</p>
              <p>
                <strong>归档类型:</strong>
                <Select
                  value={deployType}
                  onChange={setDeployType}
                  style={{ width: 200, marginLeft: 8 }}
                >
                  <Select.Option value="all">全部归档（前端+后端）</Select.Option>
                  <Select.Option value="frontend">前端归档</Select.Option>
                  <Select.Option value="backend">后端归档</Select.Option>
                </Select>
              </p>
            </div>
            <Table
              columns={appColumns}
              dataSource={selectedApps}
              rowKey="id"
              pagination={false}
              size="small"
            />
            <div style={{ marginTop: 24, textAlign: 'center' }}>
              <Space size="large">
                <Button size="large" onClick={handlePrev}>上一步</Button>
                <Button
                  type="primary"
                  size="large"
                  icon={<FileZipOutlined />}
                  onClick={handleArchive}
                  loading={archiving}
                >
                  开始归档
                </Button>
              </Space>
            </div>
          </div>
        )}

        {/* 步骤 4: 完成 */}
        {currentStep === 3 && (
          <div style={{ padding: '20px 0' }}>
            <Result
              icon={
                archiveResults.every(r => r.message === '归档成功') ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <SyncOutlined spin style={{ color: '#1890ff' }} />
                )
              }
              title={
                archiveResults.every(r => r.message === '归档成功') ? '归档完成' :
                archiveResults.every(r => !r.triggered) ? '归档结束' : '归档任务已提交'
              }
              subTitle={
                (() => {
                  const successCount = archiveResults.filter(r => r.message === '归档成功').length
                  const failedCount = archiveResults.filter(r => r.message === '归档失败').length
                  const runningCount = archiveResults.filter(r => r.triggered).length
                  if (runningCount > 0) {
                    return `共 ${archiveResults.length} 个应用，${runningCount} 个正在归档中`
                  }
                  return `共 ${archiveResults.length} 个应用，${successCount} 个成功，${failedCount} 个失败`
                })()
              }
              extra={
                archiveResults.every(r => !r.triggered) ? (
                  <Button type="primary" onClick={handleRestart}>
                    关闭
                  </Button>
                ) : (
                  <Button
                    type="primary"
                    icon={<HistoryOutlined />}
                    onClick={handleViewHistory}
                  >
                    查看归档历史
                  </Button>
                )
              }
            />
            <Table
              columns={resultColumns}
              dataSource={archiveResults}
              rowKey={(r) => r.app.id}
              pagination={false}
              size="small"
            />
            {archiveResults.every(r => !r.triggered) && (
              <div style={{ marginTop: 24, textAlign: 'center' }}>
                <Space size="large">
                  <Button size="large" onClick={handleRestart}>
                    继续归档
                  </Button>
                  <Button type="primary" size="large" icon={<HistoryOutlined />} onClick={handleViewHistory}>
                    查看归档历史
                  </Button>
                </Space>
              </div>
            )}
          </div>
        )}
      </Card>
    </div>
  )
}