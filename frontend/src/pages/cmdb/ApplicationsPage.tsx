import { useState, useEffect, useMemo } from 'react'
import { Table, Button, Modal, Form, Input, Select, Tag, message, Popconfirm, Space, Alert, Checkbox, Tooltip } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined, CloudDownloadOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import { cmdbAPI, Application, Environment, Project, JenkinsViewJob } from '../../api/cmdb'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import SearchBar, { SearchField } from '../../components/SearchBar'
import { canEdit } from '../../utils/menuAccess'

// 从 Jenkins URL 提取简短显示名称
// http://js.zbnsec.com/view/6f_dev-187/job/6f_dev-fscr_on-site_update → 6f_dev-187/fscr_on-site_update
const formatJenkinsUrl = (url: string, envName?: string) => {
  if (!url) return '-'
  const viewMatch = url.match(/\/view\/([^/]+)\/job\/([^/]+)/)
  if (viewMatch) {
    const viewName = viewMatch[1]
    const jobName = viewMatch[2]
    // 去掉 job 名称中与 view 相同的前缀
    const prefixMatch = viewName.match(/^(.+?)-\d+$/)
    const prefix = prefixMatch ? prefixMatch[1] + '-' : ''
    const shortJob = prefix && jobName.startsWith(prefix) ? jobName.slice(prefix.length) : jobName
    return envName ? `${envName}-${shortJob}` : `${viewName}/${shortJob}`
  }
  return url
}

const searchFields: SearchField[] = [
  { name: 'name', label: '应用名称', type: 'text' },
  { name: 'project_id', label: '所属项目', type: 'select' },
  { name: 'env_id', label: '项目环境', type: 'select' },
]

const extractJobPrefixFromView = (viewName: string) => {
  const match = viewName.match(/-(\d+|[Vv]\d+(?:\.\d+)*)$/)
  if (!match) return ''
  return `${viewName.slice(0, -match[0].length)}-`
}

const trimDerivedJobPrefix = (jobName: string) => jobName.replace(/^\d+-/, '')

const applyImportAppNamePrefix = (job: JenkinsViewJob, viewName: string, prefix: string) => {
  const normalizedPrefix = prefix.trim()

  const trimExplicitPrefix = (name: string) => {
    if (normalizedPrefix && name.startsWith(normalizedPrefix)) {
      return name.slice(normalizedPrefix.length)
    }
    return name
  }

  let appName = trimExplicitPrefix(job.name)
  if (appName === job.name) {
    const derivedPrefix = extractJobPrefixFromView(viewName)
    if (derivedPrefix && job.name.startsWith(derivedPrefix)) {
      appName = trimDerivedJobPrefix(job.name.slice(derivedPrefix.length))
      appName = trimExplicitPrefix(appName)
    }
  }

  return {
    ...job,
    app_name: appName,
  }
}

export default function ApplicationsPage() {
  const [apps, setApps] = useState<Application[]>([])
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingApp, setEditingApp] = useState<Application | null>(null)
  const [isCopying, setIsCopying] = useState(false)
  const [searchValues, setSearchValues] = useState<Record<string, string>>({})
  const [form] = Form.useForm()
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])

  // Jenkins 导入相关状态
  const [importModalVisible, setImportModalVisible] = useState(false)
  const [importStep, setImportStep] = useState<'input' | 'preview' | 'result'>('input')
  const [importViewName, setImportViewName] = useState('')
  const [importProjectId, setImportProjectId] = useState<number | undefined>()
  const [importEnvId, setImportEnvId] = useState<number | undefined>()
  const [importJobs, setImportJobs] = useState<JenkinsViewJob[]>([])
  const [importSelectedJobs, setImportSelectedJobs] = useState<string[]>([])
  const [importArchiveJob, setImportArchiveJob] = useState<string | undefined>()
  const [importAppNamePrefix, setImportAppNamePrefix] = useState('')
  const [importLoading, setImportLoading] = useState(false)
  const [importResult, setImportResult] = useState<{ created: number; skipped: number; errors: string[]; message: string } | null>(null)

  const previewImportJobs = useMemo(
    () => importJobs.map(job => applyImportAppNamePrefix(job, importViewName, importAppNamePrefix)),
    [importJobs, importViewName, importAppNamePrefix]
  )

  const fetchData = async () => {
    setLoading(true)
    try {
      const [appsResp, envsResp, projectsResp] = await Promise.all([
        cmdbAPI.getApplications({ limit: 1000 }),
        cmdbAPI.getEnvironments({ limit: 1000 }),
        cmdbAPI.getProjects({ limit: 1000 }),
      ])
      setApps(appsResp.data)
      setEnvironments(envsResp.data)
      setProjects(projectsResp.data)
    } catch {
      message.error('加载数据失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const filteredApps = useMemo(() => {
    return apps.filter((app) => {
      if (searchValues.name && !app.name?.toLowerCase().includes(searchValues.name.toLowerCase())) return false
      if (searchValues.project_id && app.project_id !== Number(searchValues.project_id)) return false
      if (searchValues.env_id && app.env_id !== Number(searchValues.env_id)) return false
      return true
    })
  }, [apps, searchValues])

  const handleSearch = (values: Record<string, string>) => setSearchValues(values)
  const handleResetSearch = () => setSearchValues({})

  const handleAdd = () => {
    setEditingApp(null)
    setIsCopying(false)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: Application) => {
    setEditingApp(record)
    setIsCopying(false)
    form.setFieldsValue({
      ...record,
      project_id: record.project_id || undefined,
      env_id: record.env_id || undefined,
      jenkins_job: record.jenkins_job || '',
      jenkins_archive_job: record.jenkins_archive_job || '',
    })
    setModalVisible(true)
  }

  const handleCopy = (record: Application) => {
    setEditingApp(null)
    setIsCopying(true)
    form.setFieldsValue({
      name: `${record.name}_copy`,
      project_id: record.project_id || undefined,
      env_id: record.env_id || undefined,
      jenkins_job: record.jenkins_job || '',
      jenkins_archive_job: record.jenkins_archive_job || '',
      code_repo: record.code_repo || '',
    })
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await cmdbAPI.deleteApplication(id)
      message.success('删除成功')
      fetchData()
    } catch {
      message.error('删除失败')
    }
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) return
    Modal.confirm({
      title: '批量删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除选中的 ${selectedRowKeys.length} 条流水线吗？此操作不可恢复。`,
      okText: '确定删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        let successCount = 0
        let failCount = 0
        for (const id of selectedRowKeys) {
          try {
            await cmdbAPI.deleteApplication(id)
            successCount++
          } catch {
            failCount++
          }
        }
        setSelectedRowKeys([])
        fetchData()
        if (failCount === 0) {
          message.success(`成功删除 ${successCount} 条`)
        } else {
          message.warning(`删除完成：${successCount} 成功，${failCount} 失败`)
        }
      },
    })
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingApp) {
        await cmdbAPI.updateApplication(editingApp.id, values)
        message.success('更新成功')
      } else {
        await cmdbAPI.createApplication(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchData()
    } catch {
      message.error('操作失败')
    }
  }

  // Jenkins 导入
  const handleOpenImport = () => {
    setImportStep('input')
    setImportViewName('')
    setImportProjectId(undefined)
    setImportEnvId(undefined)
    setImportJobs([])
    setImportSelectedJobs([])
    setImportArchiveJob(undefined)
    setImportAppNamePrefix('')
    setImportResult(null)
    setImportModalVisible(true)
  }

  const handleFetchJobs = async () => {
    if (!importViewName.trim()) {
      message.warning('请输入 Jenkins View 名称')
      return
    }
    setImportLoading(true)
    try {
      const resp = await cmdbAPI.getJenkinsViewJobs(importViewName.trim(), importAppNamePrefix.trim() || undefined)
      const jobs = resp.jobs || []
      setImportJobs(jobs)
      const newJobs = jobs.filter(j => !j.exists).map(j => j.name)
      setImportSelectedJobs(newJobs)
      // 自动识别归档 Job（包含 on-site_update / on_site / archive 的 Job）
      const archiveJob = jobs.find(j => /on[-_]site|archive|update/i.test(j.name))
      if (archiveJob) {
        setImportArchiveJob(archiveJob.name)
      }
      setImportStep('preview')
      if (jobs.length === 0) {
        message.info('该 View 下没有 Job')
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } }
      message.error(err.response?.data?.error || '获取 Jenkins View Jobs 失败')
    } finally {
      setImportLoading(false)
    }
  }

  const handleImportSubmit = async () => {
    if (!importProjectId) { message.warning('请选择所属项目'); return }
    if (!importEnvId) { message.warning('请选择项目环境'); return }
    if (importSelectedJobs.length === 0) { message.warning('请选择要导入的 Job'); return }
    setImportLoading(true)
    try {
      const resp = await cmdbAPI.importJenkinsJobs({
        view_name: importViewName,
        project_id: importProjectId,
        env_id: importEnvId,
        job_names: importSelectedJobs,
        archive_job: importArchiveJob,
        app_name_prefix: importAppNamePrefix.trim() || undefined,
      })
      setImportResult(resp)
      setImportStep('result')
      if (resp.created > 0) {
        message.success(resp.message)
        fetchData()
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } }
      message.error(err.response?.data?.error || '导入失败')
    } finally {
      setImportLoading(false)
    }
  }

  const columns = [
    { title: '应用名称', dataIndex: 'name', key: 'name', width: 200 },
    {
      title: '所属项目',
      dataIndex: 'project',
      key: 'project',
      width: 150,
      render: (project: Project) => project ? <Tag color="green">{project.name}</Tag> : '-',
    },
    {
      title: '项目环境',
      dataIndex: 'environment',
      key: 'environment',
      width: 120,
      render: (env: Environment) => env ? <Tag color="blue">{env.name}</Tag> : '-',
    },
    {
      title: 'Jenkins发布流水线',
      dataIndex: 'jenkins_job',
      key: 'jenkins_job',
      width: 260,
      ellipsis: true,
      render: (url: string, record: Application) => {
        if (!url) return '-'
        const display = formatJenkinsUrl(url, record.environment?.name)
        return (
          <Tooltip title={url}>
            <a href={url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12 }}>{display}</a>
          </Tooltip>
        )
      },
    },
    {
      title: 'Jenkins归档流水线',
      dataIndex: 'jenkins_archive_job',
      key: 'jenkins_archive_job',
      width: 260,
      ellipsis: true,
      render: (url: string, record: Application) => {
        if (!url) return '-'
        const display = formatJenkinsUrl(url, record.environment?.name)
        return (
          <Tooltip title={url}>
            <a href={url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12 }}>{display}</a>
          </Tooltip>
        )
      },
    },
    { title: '描述', dataIndex: 'code_repo', key: 'code_repo', width: 200, ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 220,
      fixed: 'right' as 'right',
      render: (_: unknown, record: Application) => {
        if (!canEdit()) return '-'
        return (
          <div style={{ whiteSpace: 'nowrap' }}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} style={{ padding: '4px 8px' }}>编辑</Button>
            <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => handleCopy(record)} style={{ padding: '4px 8px' }}>复制</Button>
            <Popconfirm title="确定要删除此流水线吗？" onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} style={{ padding: '4px 8px' }}>删除</Button>
            </Popconfirm>
          </div>
        )
      },
    },
  ]

  return (
    <div>
      <SearchBar
        fields={searchFields.map(f => {
          if (f.name === 'env_id') return { ...f, options: environments.map(e => ({ value: e.id, label: e.name })) }
          if (f.name === 'project_id') return { ...f, options: projects.map(p => ({ value: p.id, label: p.name })) }
          return f
        })}
        onSearch={handleSearch}
        onReset={handleResetSearch}
      />

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        {canEdit() && (
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增应用流水线
            </Button>
            <Button icon={<CloudDownloadOutlined />} onClick={handleOpenImport}>
              从 Jenkins 导入
            </Button>
            {selectedRowKeys.length > 0 && (
              <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
          </Space>
        )}
        {!canEdit() && <div />}
      </div>
      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前应用流水线页面上下文发起查询"
        actions={[
          { label: '最近哪些应用发布最频繁', query: '最近哪些应用发布最频繁' },
          { label: '哪些应用最近部署失败较多', query: '哪些应用最近部署失败较多' },
          { label: '哪些应用缺少关键信息配置', query: '哪些应用缺少关键信息配置' },
        ]}
      />
      <Table
        columns={columns}
        dataSource={filteredApps}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1400 }}
        rowSelection={canEdit() ? {
          selectedRowKeys,
          onChange: keys => setSelectedRowKeys(keys as number[]),
        } : undefined}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total) => `共 ${total} 条`, showQuickJumper: true }}
      />

      <Modal
        title={editingApp ? '编辑应用流水线' : isCopying ? '复制新增应用流水线' : '新增应用流水线'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="应用名称" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="请输入应用名称" />
          </Form.Item>
          <Form.Item name="project_id" label="所属项目" rules={[{ required: true, message: '请选择所属项目' }]}>
            <Select placeholder="请选择所属项目" showSearch optionFilterProp="children">
              {projects.map((p) => (
                <Select.Option key={p.id} value={p.id}>{p.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="env_id" label="项目环境" rules={[{ required: true, message: '请选择项目环境' }]}>
            <Select placeholder="请选择项目环境" showSearch optionFilterProp="children">
              {environments.map((e) => (
                <Select.Option key={e.id} value={e.id}>{e.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="jenkins_job" label="Jenkins发布流水线">
            <Input placeholder="请输入Jenkins发布流水线地址" />
          </Form.Item>
          <Form.Item name="jenkins_archive_job" label="Jenkins归档流水线">
            <Input placeholder="请输入Jenkins归档流水线地址" />
          </Form.Item>
          <Form.Item name="code_repo" label="描述">
            <Input placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Jenkins 导入弹窗 */}
      <Modal
        title="从 Jenkins 导入流水线"
        open={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        width={780}
        footer={
          importStep === 'input' ? [
            <Button key="cancel" onClick={() => setImportModalVisible(false)}>取消</Button>,
            <Button key="fetch" type="primary" loading={importLoading} onClick={handleFetchJobs}>获取 Jobs</Button>,
          ] : importStep === 'preview' ? [
            <Button key="back" onClick={() => setImportStep('input')}>上一步</Button>,
            <Button key="import" type="primary" loading={importLoading} onClick={handleImportSubmit}
              disabled={importSelectedJobs.length === 0 || !importProjectId || !importEnvId}>
              导入 ({importSelectedJobs.length} 个)
            </Button>,
          ] : [
            <Button key="close" type="primary" onClick={() => setImportModalVisible(false)}>关闭</Button>,
          ]
        }
      >
        {importStep === 'input' && (
          <div>
            <Alert
              message="Jenkins View 必须存在！ 输入 Jenkins View 名称，自动获取该 View 下的所有 Job 并批量导入为流水线"
              description={<span>例如：输入 <code>6f_dev-187</code> 将获取 <code>http://js.zbnsec.com/view/6f_dev-187/</code> 下的所有 Job</span>}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
            <Form layout="vertical">
              <Form.Item label="Jenkins View 名称" required>
                <Input
                  placeholder="例如: 6f_dev-187"
                  value={importViewName}
                  onChange={e => setImportViewName(e.target.value)}
                  onPressEnter={handleFetchJobs}
                  size="large"
                />
              </Form.Item>
              <Form.Item label="所属项目" required>
                <Select placeholder="请选择所属项目" value={importProjectId} onChange={v => setImportProjectId(v)} showSearch optionFilterProp="children">
                  {projects.map(p => (<Select.Option key={p.id} value={p.id}>{p.name}</Select.Option>))}
                </Select>
              </Form.Item>
              <Form.Item label="项目环境" required>
                <Select placeholder="请选择项目环境" value={importEnvId} onChange={v => setImportEnvId(v)} showSearch optionFilterProp="children">
                  {environments.map(e => (<Select.Option key={e.id} value={e.id}>{e.name}</Select.Option>))}
                </Select>
              </Form.Item>
              <Form.Item label="应用名前缀清理规则">
                <Input
                  placeholder="可选，输入要从 Job 名开头去掉的前缀，例如: 190-"
                  value={importAppNamePrefix}
                  onChange={e => setImportAppNamePrefix(e.target.value)}
                />
              </Form.Item>
            </Form>
          </div>
        )}

        {importStep === 'preview' && (
          <div>
            <Alert
              message={`View "${importViewName}" 共 ${importJobs.length} 个 Job，已选择 ${importSelectedJobs.length} 个`}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
            {importAppNamePrefix.trim() && (
              <Alert
                message={`已启用应用名前缀清理规则：${importAppNamePrefix.trim()}`}
                description="预览中的应用名称会先去掉这个 Job 名前缀，导入时也会按相同规则落库。"
                type="success"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
            <Form layout="vertical" style={{ marginBottom: 16 }}>
              <Form.Item label="应用名前缀清理规则">
                <Input
                  placeholder="可选，输入要从 Job 名开头去掉的前缀，例如: 190-"
                  value={importAppNamePrefix}
                  onChange={e => setImportAppNamePrefix(e.target.value)}
                />
              </Form.Item>
            </Form>
            <div style={{ marginBottom: 12, padding: '8px 12px', background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 4 }}>
              <span style={{ marginRight: 8, fontWeight: 500 }}>
                <span style={{ color: '#ff4d4f' }}>*</span> 归档流水线：
              </span>
              <Select
                placeholder="选择归档流水线 Job（所有流水线共用）"
                value={importArchiveJob}
                onChange={v => setImportArchiveJob(v)}
                showSearch
                optionFilterProp="children"
                style={{ width: 420 }}
                status={!importArchiveJob ? 'warning' : undefined}
              >
                {previewImportJobs.map(j => (
                  <Select.Option key={j.name} value={j.name}>{j.app_name || j.name}</Select.Option>
                ))}
              </Select>
              {importArchiveJob && (
                <div style={{ marginTop: 6, fontSize: 12, color: '#52c41a' }}>
                  归档流水线主要用于现场迭代更新的部署包归档。
                  归档地址：{`${window.location.protocol}//js.zbnsec.com/view/${importViewName}/job/${importArchiveJob}`}
                </div>
              )}
            </div>
            <div style={{ marginBottom: 12 }}>
              <Space>
                <Checkbox
                  indeterminate={importSelectedJobs.length > 0 && importSelectedJobs.length < importJobs.filter(j => !j.exists).length}
                  checked={importSelectedJobs.length === importJobs.filter(j => !j.exists).length && importJobs.filter(j => !j.exists).length > 0}
                  onChange={e => {
                    if (e.target.checked) {
                      setImportSelectedJobs(previewImportJobs.filter(j => !j.exists).map(j => j.name))
                    } else {
                      setImportSelectedJobs([])
                    }
                  }}
                >
                  全选新增
                </Checkbox>
                <span style={{ color: '#999', fontSize: 12 }}>
                  <Tag color="green">新增</Tag> 可导入 &nbsp;
                  <Tag>已存在</Tag> 自动跳过
                </span>
              </Space>
            </div>
            <Table
              dataSource={previewImportJobs}
              rowKey="name"
              size="small"
              pagination={false}
              scroll={{ y: 340 }}
              rowSelection={{
                selectedRowKeys: importSelectedJobs,
                onChange: keys => setImportSelectedJobs(keys as string[]),
                getCheckboxProps: record => ({ disabled: record.exists }),
              }}
              columns={[
                {
                  title: '应用名称',
                  dataIndex: 'app_name',
                  key: 'app_name',
                  width: 200,
                  render: (appName: string, record: JenkinsViewJob) => (
                    <span>
                      {appName}
                      {record.exists && <Tag style={{ marginLeft: 8 }}>已存在</Tag>}
                    </span>
                  ),
                },
                {
                  title: 'Jenkins 地址',
                  dataIndex: 'job_url',
                  key: 'job_url',
                  ellipsis: true,
                  render: (url: string) => (
                    <Tooltip title={url}>
                      <a href={url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12 }}>{url}</a>
                    </Tooltip>
                  ),
                },
                {
                  title: '状态',
                  dataIndex: 'color',
                  key: 'color',
                  width: 80,
                  render: (color: string) => {
                    const colorMap: Record<string, { tag: string; label: string }> = {
                      blue: { tag: 'processing', label: '正常' },
                      blue_anime: { tag: 'processing', label: '构建中' },
                      red: { tag: 'error', label: '失败' },
                      red_anime: { tag: 'error', label: '构建中' },
                      yellow: { tag: 'warning', label: '不稳定' },
                      disabled: { tag: 'default', label: '禁用' },
                      notbuilt: { tag: 'default', label: '未构建' },
                    }
                    const c = colorMap[color] || { tag: 'default', label: color }
                    return <Tag color={c.tag}>{c.label}</Tag>
                  },
                },
              ]}
            />
          </div>
        )}

        {importStep === 'result' && importResult && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
            <h3>{importResult.message}</h3>
            <div style={{ marginTop: 16, textAlign: 'left', maxWidth: 400, margin: '16px auto' }}>
              <p>新增应用流水线: <strong>{importResult.created}</strong> 个</p>
              <p>跳过已存在: <strong>{importResult.skipped}</strong> 个</p>
              {importResult.errors?.length > 0 && (
                <Alert type="warning" message="部分导入失败" description={importResult.errors.join('\n')} style={{ marginTop: 8 }} />
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
