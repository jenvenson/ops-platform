// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useMemo, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Steps, Select, Button, Table, Tag, message, Space, Empty, Result } from 'antd'
import { RocketOutlined, HistoryOutlined, SyncOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { cmdbAPI, Application, Project, deployAPI } from '../../api/cmdb'

type DeployType = 'all' | 'frontend' | 'backend'

export default function AppReleasePage() {
  const navigate = useNavigate()
  const { t } = useTranslation('deploy')
  const { t: tc } = useTranslation('common')
  const [currentStep, setCurrentStep] = useState(0)
  const [projects, setProjects] = useState<Project[]>([])
  const [applications, setApplications] = useState<Application[]>([])
  const [selectedProjectId, setSelectedProjectId] = useState<number | undefined>()
  const [selectedApps, setSelectedApps] = useState<Application[]>([])
  const [deployType, setDeployType] = useState<DeployType>('all')
  const [deploying, setDeploying] = useState(false)
  const [deployResults, setDeployResults] = useState<{ app: Application; triggered: boolean; message: string; deployRecordId?: number; deployType: string }[]>([])
  const [loading, setLoading] = useState(false)

  const getDeployBlockReason = (app: Application) => {
    if (!app.env_id) {
      return t('notConfiguredDeployEnv', '未配置部署环境')
    }
    if (!app.jenkins_job) {
      return t('notConfiguredJenkinsPipeline', '未配置 Jenkins 发布流水线')
    }
    return ''
  }

  // 使用 useMemo 自动过滤应用，当 selectedProjectId 或 applications 改变时自动更新
  const projectApps = useMemo(() => {
    if (!selectedProjectId) {
      return []
    }

    return applications.filter(app => {
      const appProjectId = app.project_id

      if (!appProjectId || appProjectId === 0) {
        return false
      }

      return Number(appProjectId) === Number(selectedProjectId)
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
        message.error(tc('loadFailed', '加载数据失败'))
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
    setSelectedApps([]) // 清空已选应用
  }

  // 下一步
  const handleNext = () => {
    if (currentStep === 0 && !selectedProjectId) {
      message.warning(t('selectProjectPlaceholder', '请选择项目'))
      return
    }
    if (currentStep === 1 && selectedApps.length === 0) {
      message.warning(t('selectProjectPlaceholder', '请选择要部署的应用'))
      return
    }
    if (currentStep === 1 && selectedApps.length > 3) {
      message.warning(t('maxConcurrentDeploy', '最多支持3个应用并发部署'))
      return
    }
    setCurrentStep(currentStep + 1)
  }

  // 上一步
  const handlePrev = () => {
    setCurrentStep(currentStep - 1)
  }

  // 执行部署
  const handleDeploy = async () => {
    setDeploying(true)
    const results: { app: Application; triggered: boolean; message: string; deployRecordId?: number; deployType: string }[] = []

    for (const app of selectedApps) {
      const blockReason = getDeployBlockReason(app)
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
        const resp = await deployAPI.triggerDeploy({
          app_id: app.id,
          env_id: app.env_id || 0,
          deploy_type: deployType,
        })
        // API 客户端拦截器已返回 response.data，resp 就是数据本身
        results.push({
          app,
          triggered: resp.success,
          message: resp.success ? t('deploying', '部署中') : (resp.message || t('triggerFailed', '触发失败')),
          deployRecordId: resp.deploy_id,
          deployType,
        })
      } catch (error: unknown) {
        const err = error as { response?: { data?: { error?: string } } }
        results.push({
          app,
          triggered: false,
          message: err.response?.data?.error || t('triggerFailed', '触发失败'),
          deployType,
        })
      }
    }

    setDeployResults(results)
    setDeploying(false)
    setCurrentStep(3) // 完成步骤
  }

  // 跳转到部署历史
  const handleViewHistory = () => {
    navigate('/deploy/history')
  }

  // 重新发布
  const handleRestart = () => {
    setCurrentStep(0)
    setSelectedProjectId(undefined)
    setSelectedApps([])
    setDeployResults([])
  }

  // 使用 ref 获取最新的 deployResults，避免闭包问题
  const deployResultsRef = useRef(deployResults)
  deployResultsRef.current = deployResults

  // 部署完成后自动刷新状态
  useEffect(() => {
    if (currentStep !== 3 || deployResults.length === 0) return

    // 立即检查一次
    const checkStatus = async () => {
      // 每次都从 ref 获取最新值
      const currentResults = deployResultsRef.current
      if (currentResults.every(r => !r.triggered)) {
        return // 所有任务都已完成，停止检查
      }

      const updatedResults = [...currentResults]
      let hasUpdate = false

      for (let i = 0; i < updatedResults.length; i++) {
        const result = updatedResults[i]
        if (!result.triggered || !result.deployRecordId) continue

        try {
          const resp = await deployAPI.getDeployStatus(result.deployRecordId)
          // 根据返回状态更新 UI
          if (resp.status === 'success') {
            updatedResults[i] = {
              ...result,
              triggered: false,
              message: t('deploySuccess', '部署成功'),
            }
            hasUpdate = true
          } else if (resp.status === 'failed') {
            updatedResults[i] = {
              ...result,
              triggered: false,
              message: t('deployFailed', '部署失败'),
            }
            hasUpdate = true
          }
        } catch (e) {
          console.error('检查部署状态失败:', e)
        }
      }

      if (hasUpdate) {
        setDeployResults(updatedResults)
      }
    }

    // 立即执行一次
    checkStatus()

    // 每3秒轮询（加快频率便于测试）
    const interval = setInterval(checkStatus, 3000)

    return () => clearInterval(interval)
  }, [currentStep])

  // 应用选择表格列（不含部署类型，用于步骤2和步骤3）
  const appColumns = [
    {
      title: t('colAppName', '应用名称'),
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: t('colProjectEnv', '项目环境'),
      dataIndex: 'environment',
      key: 'environment',
      render: (env: Application['environment']) => env?.name ? <Tag color="blue">{env.name}</Tag> : '-',
    },
    {
      title: t('colJenkinsPipeline', 'Jenkins发布流水线'),
      dataIndex: 'jenkins_job',
      key: 'jenkins_job',
      ellipsis: true,
      render: (value: string) => value || <Tag color="error">{t('notConfigured', '未配置')}</Tag>,
    },
    {
      title: t('colDeployReady', '可部署状态'),
      key: 'deploy_ready',
      width: 180,
      render: (_: unknown, record: Application) => {
        const reason = getDeployBlockReason(record)
        if (!reason) {
          return <Tag color="success">{t('deployReady', '可部署')}</Tag>
        }
        return <Tag color="warning">{reason}</Tag>
      },
    },
  ]

  // 部署结果表格列（包含部署类型）
  const resultColumns = [
    {
      title: t('colAppName', '应用名称'),
      dataIndex: ['app', 'name'],
      key: 'name',
    },
    {
      title: t('colProjectEnv', '项目环境'),
      key: 'environment',
      render: (_: unknown, record: { app: Application }) =>
        record.app.environment?.name ? <Tag color="blue">{record.app.environment.name}</Tag> : '-',
    },
    {
      title: t('colDeployType', '部署类型'),
      key: 'deployType',
      width: 120,
      render: (_: unknown, record: { app: Application; triggered: boolean; message: string; deployRecordId?: number; deployType: string }) => {
        const typeMap: Record<string, { color: string; text: string }> = {
          all: { color: 'purple', text: t('allDeploy', '全部') },
          frontend: { color: 'cyan', text: t('frontendDeploy', '前端') },
          backend: { color: 'blue', text: t('backendDeploy', '后端') },
        }
        const config = typeMap[record.deployType] || { color: 'default', text: record.deployType }
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: tc('status', '状态'),
      key: 'status',
      render: (_: unknown, record: { app: Application; triggered: boolean; message: string; deployRecordId?: number; deployType: string }) => {
        if (record.triggered) {
          return <Tag icon={<SyncOutlined spin />} color="processing">{t('deploying', '部署中')}</Tag>
        }
        if (record.message === t('deploySuccess', '部署成功')) {
          return <Tag color="success">{tc('success', '成功')}</Tag>
        }
        if (record.message === t('deployFailed', '部署失败')) {
          return <Tag color="error">{tc('failed', '失败')}</Tag>
        }
        return <Tag color="error">{record.message}</Tag>
      },
    },
    {
      title: t('colMessage', '消息'),
      dataIndex: 'message',
      key: 'message',
    },
  ]

  const steps = [
    { title: t('stepSelectProject', '选择项目'), description: t('stepDescSelectDeployProject', '选择要部署的项目') },
    { title: t('stepSelectApp', '选择应用'), description: t('stepDescSelectDeployApp', '选择要部署的应用') },
    { title: t('stepConfirmDeploy', '确认部署'), description: t('stepDescConfirmDeploy', '确认并执行部署') },
    { title: t('stepComplete', '完成'), description: t('stepDescDeployComplete', '部署结果') },
  ]

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Steps current={currentStep} items={steps} />
      </Card>

      <Card title={t('iterativeDeploy', '迭代部署')}>
        {/* 步骤 1: 选择项目 */}
        {currentStep === 0 && (
          <div style={{ padding: '40px 0', textAlign: 'center' }}>
            <h3 style={{ marginBottom: 8 }}>{t('selectDeployProjectHeading', '请选择要部署的项目')}</h3>
            <div style={{ marginBottom: 24, fontSize: '12px', color: '#999' }}>
              {t('selectDeployProjectHint', '选择项目后，将显示该项目的所有应用（应用管理中"所属项目"为该项目的应用）')}
            </div>
            <Select
              placeholder={t('selectProjectPlaceholder', '选择项目')}
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
                {t('nextStep', '下一步')}
              </Button>
            </div>
          </div>
        )}

        {/* 步骤 2: 选择应用 */}
        {currentStep === 1 && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <span style={{ marginRight: 16 }}>{t('selectedProject', '已选择项目:')} <Tag color="blue">{projects.find(p => p.id === selectedProjectId)?.name}</Tag></span>
            </div>
            <div style={{ marginBottom: 16, fontSize: '12px', color: '#666', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
              {t('deployHint', '提示：仅可选择已配置部署环境和 Jenkins 发布流水线的应用。')}
            </div>
            {projectApps.length === 0 ? (
              <Empty
                description={
                  <div style={{ textAlign: 'center' }}>
                    <div style={{ fontSize: '16px', marginBottom: '8px' }}>{t('noAppsInProject', '该项目下没有关联的应用')}</div>
                    <div style={{ fontSize: '12px', color: '#999', marginTop: '8px' }}>
                      <div>{t('noAppsExplanation', '说明：迭代部署根据"应用管理"中的"所属项目"字段来匹配应用')}</div>
                      <div style={{ marginTop: '8px' }}>{t('possibleReasons', '可能的原因：')}</div>
                      <div style={{ marginTop: '4px' }}>{t('reason1NoProject', '1. 应用尚未在"应用管理"中设置"所属项目"')}</div>
                      <div style={{ marginTop: '4px' }}>{t('reason2MismatchProject', '2. 应用的"所属项目"与当前选择的项目不匹配')}</div>
                      <div style={{ marginTop: '12px', color: '#1890ff' }}>
                        {t('solutionGoToAppMgmt', '解决方法：请前往"应用管理"页面，编辑应用，设置"所属项目"为当前选择的项目')}
                      </div>
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
                    disabled: Boolean(getDeployBlockReason(record)),
                  }),
                  onChange: (_, selectedRows) => {
                    if (selectedRows.length > 3) {
                      message.warning(t('maxConcurrentDeployLimit', '最多支持3个应用并发部署，已自动限制为前3个'))
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
              <div style={{ marginTop: 16, padding: '8px 12px', background: selectedApps.length > 3 ? '#fff2f0' : '#f6ffed', borderRadius: 4 }}>
                {t('selectedCount', '已选择')} <strong>{selectedApps.length}</strong> {t('appsUnit', '个应用')}
                {selectedApps.length > 3 && (
                  <span style={{ color: '#ff4d4f', marginLeft: 8 }}>{t('maxConcurrentWarning', '（最多支持3个并发部署）')}</span>
                )}
              </div>
            )}
            <div style={{ marginTop: 24, textAlign: 'center' }}>
              <Space size="large">
                <Button size="large" onClick={handlePrev}>{t('prevStep', '上一步')}</Button>
                <Button type="primary" size="large" onClick={handleNext} disabled={selectedApps.length === 0 || selectedApps.length > 3}>
                  {t('nextStepCount', '下一步 ({{count}} 个应用)', { count: selectedApps.length })}
                </Button>
              </Space>
            </div>
          </div>
        )}

        {/* 步骤 3: 确认部署 */}
        {currentStep === 2 && (
          <div style={{ padding: '20px 0' }}>
            <h3 style={{ marginBottom: 16 }}>{t('confirmDeployInfo', '确认发布信息')}</h3>
            <div style={{ marginBottom: 24 }}>
              <p><strong>{t('projectLabel', '项目:')}</strong> {projects.find(p => p.id === selectedProjectId)?.name}</p>
              <p><strong>{t('appCountLabel', '应用数量:')}</strong> {selectedApps.length}{t('geUnit', '个')}</p>
              <p>
                <strong>{t('deployTypeLabel', '部署类型:')}</strong>
                <Select
                  value={deployType}
                  onChange={setDeployType}
                  style={{ width: 200, marginLeft: 8 }}
                >
                  <Select.Option value="all">{t('allDeployOption', '全部部署（前端+后端）')}</Select.Option>
                  <Select.Option value="frontend">{t('frontendDeployOption', '前端部署')}</Select.Option>
                  <Select.Option value="backend">{t('backendDeployOption', '后端部署')}</Select.Option>
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
                <Button size="large" onClick={handlePrev}>{t('prevStep', '上一步')}</Button>
                <Button
                  type="primary"
                  size="large"
                  icon={<RocketOutlined />}
                  onClick={handleDeploy}
                  loading={deploying}
                >
                  {t('startDeploy', '开始部署')}
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
                deployResults.every(r => r.message === t('deploySuccess', '部署成功')) ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <SyncOutlined spin style={{ color: '#1890ff' }} />
                )
              }
              title={
                deployResults.every(r => r.message === t('deploySuccess', '部署成功')) ? t('deployComplete', '部署完成') :
                deployResults.every(r => !r.triggered) ? t('deployEnded', '部署结束') : t('deploySubmitted', '部署任务已提交')
              }
              subTitle={
                (() => {
                  const successCount = deployResults.filter(r => r.message === t('deploySuccess', '部署成功')).length
                  const failedCount = deployResults.filter(r => r.message === t('deployFailed', '部署失败')).length
                  const runningCount = deployResults.filter(r => r.triggered).length
                  if (runningCount > 0) {
                    return t('deployResultSummary', '共 {{total}} 个应用，{{running}} 个正在部署中', { total: deployResults.length, running: runningCount })
                  }
                  return t('deployResultFinal', '共 {{total}} 个应用，{{success}} 个成功，{{failed}} 个失败', { total: deployResults.length, success: successCount, failed: failedCount })
                })()
              }
              extra={
                deployResults.every(r => !r.triggered) ? (
                  <Button type="primary" onClick={handleRestart}>
                    {tc('close', '关闭')}
                  </Button>
                ) : (
                  <Button
                    type="primary"
                    icon={<HistoryOutlined />}
                    onClick={handleViewHistory}
                  >
                    {t('viewDeployRecords', '查看部署记录')}
                  </Button>
                )
              }
            />
            <Table
              columns={resultColumns}
              dataSource={deployResults}
              rowKey={(r) => r.app.id}
              pagination={false}
              size="small"
            />
            {deployResults.every(r => !r.triggered) && (
              <div style={{ marginTop: 24, textAlign: 'center' }}>
                <Space size="large">
                  <Button size="large" onClick={handleRestart}>
                    {t('continueDeploy', '继续部署')}
                  </Button>
                  <Button type="primary" size="large" icon={<HistoryOutlined />} onClick={handleViewHistory}>
                    {t('viewDeployRecords', '查看部署记录')}
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
