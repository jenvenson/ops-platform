import { useState, useEffect, useMemo } from 'react'
import { Table, Button, Modal, Form, Input, Select, message, Tag, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { cmdbAPI, Server, Project, Environment } from '../../api/cmdb'
import { monitorAPI, ServerStatus } from '../../api/monitor'
import AssistantQuickActions from '../../components/AssistantQuickActions'
import SearchBar, { SearchField } from '../../components/SearchBar'
import { canEdit } from '../../utils/menuAccess'

// 搜索字段配置
const searchFields: SearchField[] = [
  { name: 'hostname', label: '主机名', type: 'text' },
  { name: 'ip', label: 'IP地址', type: 'text' },
  { name: 'os', label: '操作系统', type: 'text' },
  { name: 'arch', label: '架构', type: 'select', options: [
    { value: 'x86_64', label: 'x86_64' },
    { value: 'arm64', label: 'arm64' },
  ]},
  { name: 'env_id', label: '环境', type: 'select' },
]

export default function ServersPage() {
  const [servers, setServers] = useState<Server[]>([])
  const [serverStatus, setServerStatus] = useState<Map<number, ServerStatus>>(new Map())
  const [projects, setProjects] = useState<Project[]>([])
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingServer, setEditingServer] = useState<Server | null>(null)
  const [checking, setChecking] = useState(false)
  const [searchValues, setSearchValues] = useState<Record<string, string>>({})
  const [form] = Form.useForm()

  const fetchData = async () => {
    setLoading(true)
    try {
      const [serversResp, projectsResp, environmentsResp] = await Promise.all([
        cmdbAPI.getServers({ limit: 1000 }),
        cmdbAPI.getProjects({ limit: 1000 }),
        cmdbAPI.getEnvironments({ limit: 1000 }),
      ])
      setServers(serversResp.data)
      setProjects(projectsResp.data)
      setEnvironments(environmentsResp.data)

      // 获取服务器状态
      try {
        const statusResp = await monitorAPI.servers.getServerStatus()
        const statusMap = new Map<number, ServerStatus>()
        for (const s of statusResp.data || []) {
          statusMap.set(s.server_id, s)
        }
        setServerStatus(statusMap)
      } catch {
        // 忽略状态获取错误
      }
    } catch (error) {
      message.error('加载数据失败')
    } finally {
      setLoading(false)
    }
  }

  // 手动触发健康检查（实时 ping）
  const handleCheck = async () => {
    setChecking(true)
    try {
      const statusResp = await monitorAPI.servers.pingAllServers()
      const statusMap = new Map<number, ServerStatus>()
      for (const s of statusResp.data || []) {
        statusMap.set(s.server_id, s)
      }
      setServerStatus(statusMap)
      message.success(`检测完成，在线 ${statusResp.data?.filter(s => s.online).length || 0}/${statusResp.data?.length || 0} 台服务器`)
    } catch (error) {
      message.error('检测失败')
    } finally {
      setChecking(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  // 搜索过滤后的数据
  const filteredServers = useMemo(() => {
    return servers.filter((server) => {
      // 主机名搜索
      if (searchValues.hostname && !server.hostname?.toLowerCase().includes(searchValues.hostname.toLowerCase())) {
        return false
      }
      // IP地址搜索
      if (searchValues.ip && !server.ip?.includes(searchValues.ip)) {
        return false
      }
      // 操作系统搜索
      if (searchValues.os && !server.os?.toLowerCase().includes(searchValues.os.toLowerCase())) {
        return false
      }
      // 架构搜索
      if (searchValues.arch && server.arch !== searchValues.arch) {
        return false
      }
      // 环境搜索（支持多选）
      if (searchValues.env_id) {
        const searchEnvId = Number(searchValues.env_id)
        const serverEnvIds = server.env_ids ? server.env_ids.split(',').map(Number) : []
        if (!serverEnvIds.includes(searchEnvId)) {
          return false
        }
      }
      return true
    })
  }, [servers, searchValues])

  const handleSearch = (values: Record<string, string>) => {
    setSearchValues(values)
  }

  const handleResetSearch = () => {
    setSearchValues({})
  }

  const handleAdd = () => {
    setEditingServer(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: Server) => {
    setEditingServer(record)
    // 将 env_ids 字符串转换为数组
    const envIdsArray = record.env_ids ? record.env_ids.split(',').map(Number).filter(Boolean) : []
    form.setFieldsValue({
      ...record,
      project_ids: record.projects?.map((p) => p.id) || [],
      env_ids: envIdsArray,
    })
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await cmdbAPI.deleteServer(id)
      message.success('删除成功')
      fetchData()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingServer) {
        await cmdbAPI.updateServer(editingServer.id, {
          hostname: values.hostname,
          ip: values.ip,
          os: values.os,
          arch: values.arch,
          ssh_port: values.ssh_port ? parseInt(values.ssh_port, 10) : 22,
          project_ids: values.project_ids,
          env_ids: values.env_ids,
        })
        message.success('更新成功')
      } else {
        await cmdbAPI.createServer({
          hostname: values.hostname,
          ip: values.ip,
          os: values.os,
          arch: values.arch,
          ssh_port: values.ssh_port ? parseInt(values.ssh_port, 10) : 22,
          project_ids: values.project_ids,
          env_ids: values.env_ids,
        })
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchData()
    } catch (error) {
      message.error('操作失败')
    }
  }

  const columns = [
    { title: '主机名', dataIndex: 'hostname', key: 'hostname' },
    { title: 'IP 地址', dataIndex: 'ip', key: 'ip' },
    { title: '操作系统', dataIndex: 'os', key: 'os' },
    { title: '架构', dataIndex: 'arch', key: 'arch', width: 120 },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: Server) => {
        const status = serverStatus.get(record.id)
        const isOnline = status?.online ?? (record.status === 'online')
        return <Tag color={isOnline ? 'green' : 'red'}>{isOnline ? '在线' : '离线'}</Tag>
      },
    },
    {
      title: '延迟',
      key: 'latency',
      width: 80,
      render: (_: unknown, record: Server) => {
        const status = serverStatus.get(record.id)
        const latency = status?.latency
        if (latency && latency > 0) {
          return <span style={{ color: latency > 100 ? '#faad14' : '#52c41a' }}>{latency}ms</span>
        }
        return '-'
      },
    },
    {
      title: '项目',
      dataIndex: 'projects',
      key: 'projects',
      width: 180,
      render: (projects: Project[]) => (
        <div>
          {projects && projects.length > 0 ? (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
              {projects.map((p) => (
                <Tag key={p.id} color="blue">{p.name}</Tag>
              ))}
            </div>
          ) : (
            '-'
          )}
        </div>
      ),
    },
    {
      title: '环境',
      dataIndex: 'env_ids',
      key: 'env_ids',
      width: 180,
      render: (envIds: string) => {
        if (!envIds) return '-'
        const ids = envIds.split(',').map(Number)
        const envNames = ids.map(id => {
          const env = environments.find(e => e.id === id)
          return env?.name
        }).filter(Boolean)
        return envNames.length > 0 ? envNames.map(name => <Tag key={name} color="blue">{name}</Tag>) : '-'
      },
    },
    {
      title: '最后心跳',
      dataIndex: 'last_heartbeat',
      key: 'last_heartbeat',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    { title: 'SSH 端口', dataIndex: 'ssh_port', key: 'ssh_port', width: 100 },
    {
      title: '操作',
      key: 'action',
      width: 160,
      fixed: 'right' as 'right',
      render: (_: unknown, record: Server) => {
        if (!canEdit()) return '-'
        return (
          <div style={{ whiteSpace: 'nowrap' }}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} style={{ padding: '4px 8px' }}>编辑</Button>
            <Popconfirm title="确定要删除此服务器吗？" onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} style={{ padding: '4px 8px' }}>删除</Button>
            </Popconfirm>
          </div>
        )
      },
    },
  ]

  return (
    <div>
      {/* 搜索栏 */}
      <SearchBar
        fields={searchFields.map(f => f.name === 'env_id' ? {
          ...f,
          options: environments.map(e => ({ value: e.id, label: e.name }))
        } : f)}
        onSearch={handleSearch}
        onReset={handleResetSearch}
      />

      <div style={{ marginBottom: 16, display: 'flex', gap: 8 }}>
        {canEdit() && (
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增服务器
          </Button>
        )}
        <Button icon={<ReloadOutlined />} onClick={handleCheck} loading={checking}>
          刷新状态
        </Button>
      </div>
      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前主机管理页面上下文发起查询"
        actions={[
          { label: '当前主机分布情况', query: '当前主机分布情况' },
          { label: '当前离线主机有哪些', query: '当前离线主机有哪些' },
          { label: '主机异常主要集中在哪个环境', query: '主机异常主要集中在哪个环境' },
        ]}
      />
      <Table
        columns={columns}
        dataSource={filteredServers}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1500 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total) => `共 ${total} 条`, showQuickJumper: true }}
      />
      <Modal
        title={editingServer ? '编辑服务器' : '新增服务器'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="hostname"
            label="主机名"
            rules={[{ required: true, message: '请输入主机名' }]}
          >
            <Input placeholder="请输入主机名" />
          </Form.Item>
          <Form.Item
            name="ip"
            label="IP 地址"
            rules={[{ required: true, message: '请输入IP地址' }]}
          >
            <Input placeholder="请输入IP地址" />
          </Form.Item>
          <Form.Item name="os" label="操作系统">
            <Input placeholder="例如: Ubuntu 22.04" />
          </Form.Item>
          <Form.Item name="arch" label="架构">
            <Select placeholder="请选择架构">
              <Select.Option value="x86_64">x86_64</Select.Option>
              <Select.Option value="arm64">arm64</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="ssh_port"
            label="SSH 端口"
            rules={[{ required: true, message: '请输入SSH端口' }]}
          >
            <Input type="number" placeholder="默认22" />
          </Form.Item>
          <Form.Item
            name="project_ids"
            label="所属项目"
            rules={[{ required: true, message: '请选择项目' }]}
          >
            <Select mode="multiple" placeholder="请选择项目">
              {projects.map((p) => (
                <Select.Option key={p.id} value={p.id}>{p.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="env_ids"
            label="所属环境"
            rules={[{ required: true, message: '请选择环境' }]}
          >
            <Select mode="multiple" placeholder="请选择环境">
              {environments.map((e) => (
                <Select.Option key={e.id} value={e.id}>{e.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
