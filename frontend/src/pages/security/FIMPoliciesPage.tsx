import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  DatabaseOutlined,
  FolderOpenOutlined,
  PlayCircleOutlined,
  SafetyCertificateOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { securityFIMAPI, type FIMPolicy, type FIMPolicyTarget, type FIMSnapshot, type FIMWatchPath } from '../../api/security-fim'
import { alertAPI, type NotifyChannel } from '../../api/alert'
import { cmdbAPI, type Server } from '../../api/cmdb'
import { getFIMErrorMessage } from '../../utils/httpError'
import { canEdit } from '../../utils/menuAccess'

const { Paragraph, Text, Title } = Typography

const severityLabelMap: Record<string, string> = {
  critical: '严重',
  high: '高',
  warning: '警告',
  medium: '中',
  low: '低',
  info: '提示',
}

type PolicyModalState = {
  open: boolean
  policy: FIMPolicy | null
}

type TargetsModalState = {
  open: boolean
  policy: FIMPolicy | null
  items: FIMPolicyTarget[]
  loading: boolean
}

type PathsModalState = {
  open: boolean
  policy: FIMPolicy | null
  items: FIMWatchPath[]
  loading: boolean
  editingItem: FIMWatchPath | null
}

type ScanModalState = {
  open: boolean
  policy: FIMPolicy | null
  targets: FIMPolicyTarget[]
  loading: boolean
}

type BaselineModalState = {
  open: boolean
  policy: FIMPolicy | null
  targets: FIMPolicyTarget[]
  snapshots: FIMSnapshot[]
  loading: boolean
}

export default function FIMPoliciesPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<FIMPolicy[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [channels, setChannels] = useState<NotifyChannel[]>([])
  const [baselineSnapshots, setBaselineSnapshots] = useState<FIMSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [policyModal, setPolicyModal] = useState<PolicyModalState>({ open: false, policy: null })
  const [targetsModal, setTargetsModal] = useState<TargetsModalState>({ open: false, policy: null, items: [], loading: false })
  const [pathsModal, setPathsModal] = useState<PathsModalState>({ open: false, policy: null, items: [], loading: false, editingItem: null })
  const [scanModal, setScanModal] = useState<ScanModalState>({ open: false, policy: null, targets: [], loading: false })
  const [baselineModal, setBaselineModal] = useState<BaselineModalState>({ open: false, policy: null, targets: [], snapshots: [], loading: false })
  const [policyForm] = Form.useForm()
  const [targetsForm] = Form.useForm()
  const [pathForm] = Form.useForm()
  const [scanForm] = Form.useForm()
  const selectedScanAction = Form.useWatch('action', scanForm)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [policiesResp, serversResp, snapshotsResp] = await Promise.all([
        securityFIMAPI.getPolicies({ page: 1, page_size: 50 }),
        cmdbAPI.getServers({ limit: 1000 }),
        securityFIMAPI.getSnapshots({ page: 1, page_size: 500, snapshot_type: 'baseline', status: 'success' }),
      ])
      setItems(policiesResp.data ?? [])
      setServers(serversResp.data ?? [])
      setBaselineSnapshots(snapshotsResp.data ?? [])
    } catch {
      setItems([])
      setBaselineSnapshots([])
      message.error('加载文件完整性巡检数据失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchData()
  }, [])

  useEffect(() => {
    const loadChannels = async () => {
      try {
        const resp = await alertAPI.channels.list()
        setChannels((resp.data ?? []).filter((item) => item.type === 'dingtalk' && item.enabled))
      } catch {
        setChannels([])
      }
    }
    void loadChannels()
  }, [])

  const serverMap = new Map(servers.map((server) => [server.id, server]))

  const openCreatePolicy = () => {
    policyForm.setFieldsValue({
      enabled: true,
      severity: 'high',
      notify_channels: [],
      scan_interval_sec: 300,
      hash_mode: 'changed_only',
      compare_mode: 'baseline',
    })
    setPolicyModal({ open: true, policy: null })
  }

  const openEditPolicy = (policy: FIMPolicy) => {
    policyForm.setFieldsValue({
      ...policy,
      notify_channels: parseNotifyChannelValues(policy.notify_channels),
    })
    setPolicyModal({ open: true, policy })
  }

  const handleSavePolicy = async () => {
    try {
      const values = await policyForm.validateFields()
      const payload = {
        ...values,
        notify_channels: Array.isArray(values.notify_channels) ? values.notify_channels.join(',') : '',
      }
      setSubmitting(true)
      if (policyModal.policy) {
        await securityFIMAPI.updatePolicy(policyModal.policy.id, payload)
        message.success('策略已更新')
      } else {
        await securityFIMAPI.createPolicy(payload)
        message.success('策略已创建')
      }
      setPolicyModal({ open: false, policy: null })
      policyForm.resetFields()
      await fetchData()
    } catch (error) {
      message.error(getFIMErrorMessage(error, '保存策略失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const loadTargets = async (policy: FIMPolicy) => {
    setTargetsModal({ open: true, policy, items: [], loading: true })
    try {
      const resp = await securityFIMAPI.getTargets(policy.id)
      const targetItems = resp.data ?? []
      setTargetsModal({ open: true, policy, items: targetItems, loading: false })
      targetsForm.setFieldsValue({ server_ids: targetItems.map((item) => item.server_id) })
    } catch {
      setTargetsModal({ open: true, policy, items: [], loading: false })
      message.error('加载绑定主机失败')
    }
  }

  const handleSaveTargets = async () => {
    const policy = targetsModal.policy
    if (!policy) {
      return
    }
    try {
      const values = await targetsForm.validateFields()
      const selectedServerIds: number[] = values.server_ids ?? []
      const currentServerIds = new Set(targetsModal.items.map((item) => item.server_id))
      const removeTargets = targetsModal.items.filter((item) => !selectedServerIds.includes(item.server_id))
      const addServerIds = selectedServerIds.filter((serverId) => !currentServerIds.has(serverId))

      setSubmitting(true)
      await Promise.all([
        addServerIds.length > 0 ? securityFIMAPI.addTargets(policy.id, addServerIds) : Promise.resolve(),
        ...removeTargets.map((target) => securityFIMAPI.deleteTarget(policy.id, target.id)),
      ])
      message.success('主机绑定已更新')
      setTargetsModal({ open: false, policy: null, items: [], loading: false })
      targetsForm.resetFields()
    } catch (error) {
      message.error(getFIMErrorMessage(error, '更新主机绑定失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const loadWatchPaths = async (policy: FIMPolicy) => {
    setPathsModal({ open: true, policy, items: [], loading: true, editingItem: null })
    pathForm.resetFields()
    pathForm.setFieldsValue({ scan_mode: 'full_hash', recursive: true, max_depth: 0, hash_on_match_only: true })
    try {
      const resp = await securityFIMAPI.getWatchPaths(policy.id)
      setPathsModal({ open: true, policy, items: resp.data ?? [], loading: false, editingItem: null })
    } catch {
      setPathsModal({ open: true, policy, items: [], loading: false, editingItem: null })
      message.error('加载目录配置失败')
    }
  }

  const refreshWatchPaths = async (policy: FIMPolicy) => {
    const resp = await securityFIMAPI.getWatchPaths(policy.id)
    setPathsModal((current) => ({ ...current, items: resp.data ?? [], loading: false }))
  }

  const openEditWatchPath = (item: FIMWatchPath) => {
    pathForm.setFieldsValue({
      path: item.path,
      scan_mode: item.scan_mode,
      recursive: item.recursive,
      max_depth: item.max_depth,
      file_glob: item.file_glob,
      exclude_glob: item.exclude_glob,
      hash_on_match_only: item.hash_on_match_only,
    })
    setPathsModal((current) => ({ ...current, editingItem: item }))
  }

  const resetWatchPathForm = () => {
    pathForm.resetFields()
    pathForm.setFieldsValue({ scan_mode: 'full_hash', recursive: true, max_depth: 0, hash_on_match_only: true })
    setPathsModal((current) => ({ ...current, editingItem: null }))
  }

  const handleAddWatchPath = async () => {
    const policy = pathsModal.policy
    if (!policy) {
      return
    }
    try {
      const values = await pathForm.validateFields()
      setSubmitting(true)
      if (pathsModal.editingItem) {
        await securityFIMAPI.updateWatchPath(pathsModal.editingItem.id, values)
        message.success('目录配置已更新')
      } else {
        await securityFIMAPI.createWatchPath(policy.id, values)
        message.success('目录配置已添加')
      }
      resetWatchPathForm()
      await refreshWatchPaths(policy)
    } catch (error) {
      message.error(getFIMErrorMessage(error, pathsModal.editingItem ? '更新目录失败' : '添加目录失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteWatchPath = async (watchPathId: number) => {
    const policy = pathsModal.policy
    if (!policy) {
      return
    }
    try {
      setSubmitting(true)
      await securityFIMAPI.deleteWatchPath(watchPathId)
      message.success('目录配置已删除')
      await refreshWatchPaths(policy)
    } catch (error) {
      message.error(getFIMErrorMessage(error, '删除目录失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const loadScanTargets = async (policy: FIMPolicy) => {
    setScanModal({ open: true, policy, targets: [], loading: true })
    scanForm.resetFields()
    scanForm.setFieldsValue({ action: 'baseline' })
    try {
      const resp = await securityFIMAPI.getTargets(policy.id)
      const targetItems = resp.data ?? []
      setScanModal({ open: true, policy, targets: targetItems, loading: false })
      if (targetItems.length > 0) {
        scanForm.setFieldsValue({
          action: 'baseline',
          server_id: targetItems[0].server_id,
        })
      }
    } catch {
      setScanModal({ open: true, policy, targets: [], loading: false })
      message.error('加载可执行主机失败')
    }
  }

  const handleRunScan = async () => {
    const policy = scanModal.policy
    if (!policy) {
      return
    }
    try {
      const values = await scanForm.validateFields()
      setSubmitting(true)
      if (values.action === 'baseline') {
        await securityFIMAPI.buildBaseline(policy.id, values.server_id)
        message.success('已触发基线构建')
      } else {
        await securityFIMAPI.runScan(policy.id, values.server_id, 'manual')
        message.success('已触发手动扫描')
      }
      setScanModal({ open: false, policy: null, targets: [], loading: false })
      scanForm.resetFields()
    } catch (error) {
      message.error(getFIMErrorMessage(error, '执行巡检失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleTogglePolicy = async (policy: FIMPolicy, enabled: boolean) => {
    try {
      if (enabled) {
        await securityFIMAPI.enablePolicy(policy.id)
      } else {
        await securityFIMAPI.disablePolicy(policy.id)
      }
      message.success(enabled ? '策略已启用' : '策略已停用')
      await fetchData()
    } catch (error) {
      message.error(getFIMErrorMessage(error, enabled ? '启用策略失败' : '停用策略失败'))
    }
  }

  const handleDeletePolicy = async (policy: FIMPolicy) => {
    try {
      setSubmitting(true)
      await securityFIMAPI.deletePolicy(policy.id)
      message.success('策略已删除')
      await fetchData()
    } catch (error) {
      message.error(getFIMErrorMessage(error, '删除策略失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const openExecutions = (policy: FIMPolicy) => {
    navigate(`/security/fim/executions?policy_id=${policy.id}`)
  }

  const openBaselineManager = async (policy: FIMPolicy) => {
    setBaselineModal({ open: true, policy, targets: [], snapshots: [], loading: true })
    try {
      const [targetsResp, snapshotsResp] = await Promise.all([
        securityFIMAPI.getTargets(policy.id),
        securityFIMAPI.getSnapshots({ page: 1, page_size: 200, policy_id: policy.id, snapshot_type: 'baseline', status: 'success' }),
      ])
      setBaselineModal({
        open: true,
        policy,
        targets: targetsResp.data ?? [],
        snapshots: snapshotsResp.data ?? [],
        loading: false,
      })
    } catch {
      setBaselineModal({ open: true, policy, targets: [], snapshots: [], loading: false })
      message.error('加载基线管理数据失败')
    }
  }

  const handleRebuildBaseline = async (policy: FIMPolicy, target: FIMPolicyTarget) => {
    try {
      setSubmitting(true)
      await securityFIMAPI.buildBaseline(policy.id, target.server_id)
      message.success('已触发基线重建，当前主机状态会作为新的比对参考')
      await Promise.all([fetchData(), openBaselineManager(policy)])
    } catch (error) {
      message.error(getFIMErrorMessage(error, '重建基线失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const getActiveBaselineSnapshots = (policyId: number) => {
    return baselineSnapshots.filter((item) => item.policy_id === policyId)
  }

  const getPolicyBaselineSummary = (policyId: number) => {
    const snapshots = getActiveBaselineSnapshots(policyId)
    if (snapshots.length === 0) {
      return '当前未建立生效基线'
    }
    const latest = snapshots.reduce((current, item) => {
      if (!current) {
        return item
      }
      return new Date(item.started_at).getTime() > new Date(current.started_at).getTime() ? item : current
    }, snapshots[0])
    return `当前生效基线：${snapshots.length} 台主机 | 最近基线：${formatDateTime(latest.started_at)}`
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Space direction="vertical" size={4}>
          <Title level={4} style={{ margin: 0 }}>文件完整性巡检</Title>
          <Paragraph type="secondary" style={{ margin: 0 }}>
            基于 SSH 定时巡检关键目录，对比基线，发现删除、修改等异常变化。
          </Paragraph>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={8}>
          <Card size="small">
            <Statistic title="巡检策略" value={items.length} prefix={<SafetyCertificateOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={8}>
          <Card size="small">
            <Statistic title="已启用策略" value={items.filter((item) => item.enabled).length} prefix={<SyncOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={8}>
          <Card size="small">
            <Statistic title="关键目录巡检" value={items.filter((item) => item.hash_mode !== 'off').length} prefix={<FolderOpenOutlined />} />
          </Card>
        </Col>
      </Row>

      <Card title="策略列表" extra={<Button type="primary" onClick={openCreatePolicy}>新增策略</Button>}>
        <List
          loading={loading}
          dataSource={items}
          locale={{ emptyText: '当前还没有文件完整性巡检策略' }}
          renderItem={(item) => (
            <List.Item
              actions={[
                canEdit() && <Button key="edit" type="link" onClick={() => openEditPolicy(item)}>编辑</Button>,
                canEdit() && <Button key="targets" type="link" onClick={() => void loadTargets(item)}>绑定主机</Button>,
                canEdit() && <Button key="paths" type="link" onClick={() => void loadWatchPaths(item)}>配置目录</Button>,
                <Button key="baseline" type="link" onClick={() => void openBaselineManager(item)}>基线管理</Button>,
                <Button key="executions" type="link" onClick={() => openExecutions(item)}>执行记录</Button>,
                canEdit() && <Button key="scan" type="link" icon={<PlayCircleOutlined />} onClick={() => void loadScanTargets(item)}>构建基线 / 扫描</Button>,
                canEdit() && <Popconfirm
                  key="delete"
                  title={`确认删除策略“${item.name}”？`}
                  description="会同时移除策略关联的目标主机、目录配置、快照、事件和告警记录。"
                  okText="删除"
                  cancelText="取消"
                  okButtonProps={{ danger: true, loading: submitting }}
                  onConfirm={() => void handleDeletePolicy(item)}
                >
                  <Button key="delete-button" type="link" danger>
                    删除
                  </Button>
                </Popconfirm>,
              ].filter(Boolean)}
            >
              <List.Item.Meta
                avatar={<DatabaseOutlined style={{ fontSize: 18, color: '#1677ff' }} />}
                title={(
                  <Space wrap>
                    <span>{item.name}</span>
                    <Tag color={item.enabled ? 'success' : 'default'}>{item.enabled ? '启用中' : '已停用'}</Tag>
                    <Tag color="blue">{severityLabelMap[item.severity] || item.severity}</Tag>
                    <Switch
                      size="small"
                      checked={item.enabled}
                      checkedChildren="启用"
                      unCheckedChildren="停用"
                      onChange={(checked) => void handleTogglePolicy(item, checked)}
                    />
                  </Space>
                )}
                description={(
                  <Space direction="vertical" size={2}>
                    <Text type="secondary">{item.description || '暂无描述'}</Text>
                    <Text type="secondary">扫描周期：{item.scan_interval_sec} 秒 | Hash 模式：{item.hash_mode} | 比对模式：{item.compare_mode}</Text>
                    <Text type="secondary">{getPolicyBaselineSummary(item.id)}</Text>
                    <Text type="secondary">通知渠道：{formatNotifyChannelNames(item.notify_channels, channels)}</Text>
                  </Space>
                )}
              />
            </List.Item>
          )}
        />
      </Card>

      <Modal
        title={policyModal.policy ? '编辑巡检策略' : '新增巡检策略'}
        open={policyModal.open}
        onOk={() => void handleSavePolicy()}
        onCancel={() => {
          setPolicyModal({ open: false, policy: null })
          policyForm.resetFields()
        }}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={policyForm} layout="vertical">
          <Form.Item name="name" label="策略名称" rules={[{ required: true, message: '请输入策略名称' }]}>
            <Input placeholder="例如：生产配置目录巡检" />
          </Form.Item>
          <Form.Item name="description" label="策略描述">
            <Input.TextArea rows={3} placeholder="描述巡检对象和目标" />
          </Form.Item>
          <Form.Item name="enabled" label="启用状态" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="severity" label="默认告警级别" rules={[{ required: true, message: '请选择默认告警级别' }]}>
                <Select options={[
                  { value: 'critical', label: '严重' },
                  { value: 'high', label: '高' },
                  { value: 'warning', label: '警告' },
                  { value: 'medium', label: '中' },
                  { value: 'low', label: '低' },
                  { value: 'info', label: '提示' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="scan_interval_sec" label="扫描周期（秒）" rules={[{ required: true, message: '请输入扫描周期' }]}>
                <InputNumber min={30} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="hash_mode" label="Hash 模式" rules={[{ required: true, message: '请选择 Hash 模式' }]}>
                <Select options={[
                  { value: 'off', label: 'off' },
                  { value: 'changed_only', label: 'changed_only' },
                  { value: 'full', label: 'full' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="compare_mode" label="比对模式" rules={[{ required: true, message: '请选择比对模式' }]}>
                <Select options={[
                  { value: 'baseline', label: 'baseline' },
                  { value: 'last_snapshot', label: 'last_snapshot' },
                ]} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="notify_channels" label="钉钉通知渠道">
            <Select
              mode="multiple"
              allowClear
              placeholder="选择告警中心里已配置的钉钉机器人"
              options={channels.map((item) => ({ value: String(item.id), label: item.name }))}
            />
          </Form.Item>
          <Text type="secondary">
            这里复用“告警中心 / 通知渠道”中的钉钉机器人。FIM 产生新告警时会自动推送到所选渠道。
          </Text>
        </Form>
      </Modal>

      <Modal
        title={targetsModal.policy ? `绑定主机: ${targetsModal.policy.name}` : '绑定主机'}
        open={targetsModal.open}
        onOk={() => void handleSaveTargets()}
        onCancel={() => {
          setTargetsModal({ open: false, policy: null, items: [], loading: false })
          targetsForm.resetFields()
        }}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Alert
          message="提示"
          description="请确保所选主机已在「安全 - FIM - SSH主机密钥」中添加主机公钥，否则扫描会失败。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form form={targetsForm} layout="vertical">
          <Form.Item name="server_ids" label="目标主机" rules={[{ required: true, message: '请至少选择一台主机' }]}>
            <Select
              mode="multiple"
              placeholder="请选择需要巡检的主机"
              loading={targetsModal.loading}
              options={servers.map((server) => ({
                value: server.id,
                label: `${server.hostname} (${server.ip})`,
              }))}
            />
          </Form.Item>
        </Form>
        <Divider />
        <List
          size="small"
          loading={targetsModal.loading}
          dataSource={targetsModal.items}
          locale={{ emptyText: '当前还没有绑定主机' }}
          renderItem={(item) => (
            <List.Item>
              <List.Item.Meta
                title={item.server_name && item.server_ip ? `${item.server_name} (${item.server_ip})` : `主机 #${item.server_id}`}
                description={`最近扫描：${formatDateTime(item.last_scan_at)} | 状态：${item.last_scan_status || '-'}`}
              />
            </List.Item>
          )}
        />
      </Modal>

      <Modal
        title={pathsModal.policy ? `配置目录: ${pathsModal.policy.name}` : '配置目录'}
        open={pathsModal.open}
        onCancel={() => {
          setPathsModal({ open: false, policy: null, items: [], loading: false, editingItem: null })
          pathForm.resetFields()
        }}
        footer={null}
        width={760}
        destroyOnHidden
      >
        <Form form={pathForm} layout="vertical">
          <Row gutter={12}>
            <Col span={14}>
              <Form.Item name="path" label="巡检目录" rules={[{ required: true, message: '请输入巡检目录' }]}>
                <Input placeholder="/etc/app" />
              </Form.Item>
            </Col>
            <Col span={10}>
              <Form.Item name="file_glob" label="文件匹配">
                <Input placeholder="*.conf" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="scan_mode" label="扫描模式" rules={[{ required: true, message: '请选择扫描模式' }]}>
                <Select
                  options={[
                    { value: 'full_hash', label: '完整校验' },
                    { value: 'presence_only', label: '仅删除监测' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="recursive" label="递归扫描" valuePropName="checked">
                <Switch checkedChildren="递归" unCheckedChildren="单层" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="max_depth" label="最大深度">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="hash_on_match_only" label="仅匹配文件计算 Hash" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
            </Col>
          </Row>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="扫描模式说明"
            description="完整校验会记录文件大小、时间和内容摘要，适合配置文件和小文件；仅删除监测只保留文件清单，适合大归档包目录，当前只检测文件是否被删除。"
          />
          <Form.Item name="exclude_glob" label="排除规则">
            <Input placeholder="*.log 或 /etc/app/cache/*" />
          </Form.Item>
          <Space>
            <Button type="primary" onClick={() => void handleAddWatchPath()} loading={submitting}>
              {pathsModal.editingItem ? '保存修改' : '添加目录'}
            </Button>
            {pathsModal.editingItem && (
              <Button onClick={resetWatchPathForm}>
                取消编辑
              </Button>
            )}
          </Space>
        </Form>

        <Divider />

        <List
          loading={pathsModal.loading}
          dataSource={pathsModal.items}
          locale={{ emptyText: '当前还没有配置巡检目录' }}
          renderItem={(item) => (
            <List.Item
              actions={[
                canEdit() && <Button key="edit" type="link" onClick={() => openEditWatchPath(item)}>编辑</Button>,
                canEdit() && <Popconfirm
                  key="delete"
                  title="确认删除该目录配置？"
                  onConfirm={() => void handleDeleteWatchPath(item.id)}
                >
                  <Button type="link" danger>删除</Button>
                </Popconfirm>,
              ].filter(Boolean)}
            >
              <List.Item.Meta
                title={item.path}
                description={`模式：${item.scan_mode === 'presence_only' ? '仅删除监测' : '完整校验'} | 递归：${item.recursive ? '是' : '否'} | 深度：${item.max_depth} | 匹配：${item.file_glob || '-'} | 排除：${item.exclude_glob || '-'}`}
              />
            </List.Item>
          )}
        />
      </Modal>

      <Modal
        title={baselineModal.policy ? `基线管理: ${baselineModal.policy.name}` : '基线管理'}
        open={baselineModal.open}
        onCancel={() => setBaselineModal({ open: false, policy: null, targets: [], snapshots: [], loading: false })}
        footer={null}
        width={860}
        destroyOnHidden
      >
        <Alert
          type="info"
          showIcon
          message="当前生效基线就是后续比对的参考线。重新构建基线后，当前主机状态会被视为新的正常状态。"
          style={{ marginBottom: 16 }}
        />
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={8}>
            <Card size="small"><Statistic title="绑定主机" value={baselineModal.targets.length} /></Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card size="small"><Statistic title="已建基线主机" value={baselineModal.snapshots.length} /></Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card size="small"><Statistic title="最近基线时间" value={getLatestBaselineTime(baselineModal.snapshots)} valueStyle={{ fontSize: 16 }} /></Card>
          </Col>
        </Row>
        <List
          loading={baselineModal.loading}
          dataSource={baselineModal.targets}
          locale={{ emptyText: '当前没有绑定主机' }}
          renderItem={(item) => {
            const snapshot = baselineModal.snapshots.find((entry) => entry.server_id === item.server_id)
            const title = item.server_name && item.server_ip ? `${item.server_name} (${item.server_ip})` : `主机 #${item.server_id}`
            return (
              <List.Item
                actions={[
                  <Button
                    key="rebuild"
                    type="link"
                    loading={submitting}
                    onClick={() => baselineModal.policy && void handleRebuildBaseline(baselineModal.policy, item)}
                  >
                    {snapshot ? '重建基线' : '构建基线'}
                  </Button>,
                  <Button
                    key="history"
                    type="link"
                    onClick={() => navigate(`/security/fim/executions?policy_id=${item.policy_id}&server_id=${item.server_id}&snapshot_type=baseline`)}
                  >
                    查看基线记录
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={title}
                  description={`当前生效基线：${snapshot ? formatDateTime(snapshot.started_at) : '未建立'} | 最近扫描：${formatDateTime(item.last_scan_at)} | 状态：${item.last_scan_status || '-'}`}
                />
              </List.Item>
            )
          }}
        />
      </Modal>

      <Modal
        title={scanModal.policy ? `执行巡检: ${scanModal.policy.name}` : '执行巡检'}
        open={scanModal.open}
        onOk={() => void handleRunScan()}
        onCancel={() => {
          setScanModal({ open: false, policy: null, targets: [], loading: false })
          scanForm.resetFields()
        }}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={scanForm} layout="vertical">
          <Form.Item name="server_id" label="目标主机" rules={[{ required: true, message: '请选择主机' }]}>
            <Select
              placeholder="请选择执行主机"
              loading={scanModal.loading}
              options={scanModal.targets.map((target) => {
                const server = serverMap.get(target.server_id)
                return {
                  value: target.server_id,
                  label: server ? `${server.hostname} (${server.ip})` : `主机 #${target.server_id}`,
                }
              })}
            />
          </Form.Item>
          <List
            size="small"
            loading={scanModal.loading}
            dataSource={scanModal.targets}
            locale={{ emptyText: '当前没有可执行主机' }}
            style={{ marginBottom: 16 }}
            renderItem={(item) => {
              const server = serverMap.get(item.server_id)
              const title = server ? `${server.hostname} (${server.ip})` : `主机 #${item.server_id}`
              return (
                <List.Item>
                  <List.Item.Meta
                    title={title}
                    description={`最近扫描：${formatDateTime(item.last_scan_at)} | 状态：${item.last_scan_status || '-'}`}
                  />
                </List.Item>
              )
            }}
          />
          <Form.Item name="action" label="执行动作" rules={[{ required: true, message: '请选择动作' }]}>
            <Select
              options={[
                { value: 'baseline', label: '构建基线' },
                { value: 'scan', label: '手动扫描' },
              ]}
            />
          </Form.Item>
          {selectedScanAction === 'baseline' ? (
            <Alert
              type="warning"
              showIcon
              message="重建基线会把当前主机状态认定为新的正常参考线。若当前目录存在异常变更，请勿直接重建。"
            />
          ) : (
            <Text type="secondary">
              首次使用建议先构建基线，后续再执行手动扫描生成差异事件和完整性告警。
            </Text>
          )}
        </Form>
      </Modal>
    </div>
  )
}

function formatDateTime(value?: string): string {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

function parseNotifyChannelValues(value?: string): string[] {
  if (!value) {
    return []
  }
  return value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function formatNotifyChannelNames(value: string | undefined, channels: NotifyChannel[]): string {
  const ids = parseNotifyChannelValues(value)
  if (ids.length === 0) {
    return '未配置'
  }
  const channelMap = new Map(channels.map((item) => [String(item.id), item.name]))
  return ids.map((id) => channelMap.get(id) || `渠道 #${id}`).join('、')
}

function getLatestBaselineTime(snapshots: FIMSnapshot[]): string {
  if (snapshots.length === 0) {
    return '-'
  }
  const latest = snapshots.reduce((current, item) => {
    if (!current) {
      return item
    }
    return new Date(item.started_at).getTime() > new Date(current.started_at).getTime() ? item : current
  }, snapshots[0])
  return formatDateTime(latest.started_at)
}
