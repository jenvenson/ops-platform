import { useState, useEffect } from 'react'
import {
  Card, Form, Input, Button, Select, Switch, Tag, Space, message,
  Typography, Row, Col, Alert, Divider, Table, Modal
} from 'antd'
import {
  CheckCircleOutlined, CloseCircleOutlined, PlayCircleOutlined, EyeOutlined, QuestionCircleOutlined, PlusOutlined, MinusCircleOutlined, DeleteOutlined, SearchOutlined, ExclamationCircleOutlined
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { consulAPI, ConsulConfig, BatchCopyAllProjectsRequest, BatchCopyResult, RuleItem, ReplacePair } from '../../api/consul'

const { Text, Paragraph } = Typography

export default function BatchCopyAllPage() {
  const navigate = useNavigate()
  const [allForm] = Form.useForm()
  const [configs, setConfigs] = useState<ConsulConfig[]>([])
  const [projects, setProjects] = useState<string[]>([])
  const [selectedConfigId, setSelectedConfigId] = useState<number | undefined>()
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<BatchCopyResult | null>(null)

  // 删除功能状态
  const [deleteSuffix, setDeleteSuffix] = useState('')
  const [deleteConfigId, setDeleteConfigId] = useState<number | undefined>()
  const [deleteKeys, setDeleteKeys] = useState<string[]>([])
  const [selectedDeleteKeys, setSelectedDeleteKeys] = useState<string[]>([])
  const [deleteKeysQueried, setDeleteKeysQueried] = useState(false)
  const [deleteQueryLoading, setDeleteQueryLoading] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [deleteResult, setDeleteResult] = useState<{ deleted: number; failed: number } | null>(null)

  const fetchProjects = async (configId?: number) => {
    if (!configId) {
      setProjects([])
      return
    }
    try {
      const resp = await consulAPI.getProjects(configId)
      setProjects(resp.projects || [])
    } catch {
      message.error('获取项目列表失败')
      setProjects([])
    }
  }

  // 查询指定后缀的 Key
  const handleQueryDeleteKeys = async () => {
    if (!deleteSuffix.trim()) {
      message.warning('请输入后缀')
      return
    }
    const cfgId = deleteConfigId || selectedConfigId
    if (!cfgId) {
      message.warning('请先选择 Consul 配置')
      return
    }
    try {
      setDeleteQueryLoading(true)
      setDeleteResult(null)
      setSelectedDeleteKeys([])
      const resp = await consulAPI.querySuffixKeys({ config_id: cfgId, suffix: deleteSuffix.trim() })
      setDeleteKeys(resp.keys || [])
      setDeleteKeysQueried(true)
      if (!resp.keys || resp.keys.length === 0) {
        message.info('未找到匹配的 Key')
      }
    } catch (error: any) {
      message.error(`查询失败: ${error.message || '未知错误'}`)
      setDeleteKeys([])
      setDeleteKeysQueried(false)
    } finally {
      setDeleteQueryLoading(false)
    }
  }

  // 执行批量删除
  const doDelete = async (keys: string[]) => {
    const cfgId = deleteConfigId || selectedConfigId
    if (!cfgId) return
    try {
      setDeleteLoading(true)
      const resp = await consulAPI.batchDeleteKeys({ config_id: cfgId, keys })
      setDeleteResult({ deleted: resp.deleted, failed: resp.failed })
      message.success(`删除完成：成功 ${resp.deleted} 个，失败 ${resp.failed} 个`)
      setSelectedDeleteKeys([])
      // 刷新列表
      if (resp.deleted > 0) {
        const remaining = deleteKeys.filter(k => !resp.deleted_keys.includes(k))
        setDeleteKeys(remaining)
        if (remaining.length === 0) setDeleteKeysQueried(false)
      }
    } catch (error: any) {
      message.error(`删除失败: ${error.message || '未知错误'}`)
    } finally {
      setDeleteLoading(false)
    }
  }

  // 删除选中
  const handleDeleteSelectedKeys = () => {
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除选中的 ${selectedDeleteKeys.length} 个 Key 吗？此操作不可恢复！`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => doDelete(selectedDeleteKeys),
    })
  }

  // 删除全部
  const handleDeleteAllKeys = () => {
    Modal.confirm({
      title: '确认删除全部',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>确定要删除后缀为 <strong>"{deleteSuffix}"</strong> 的全部 <strong style={{ color: '#ff4d4f' }}>{deleteKeys.length}</strong> 个 Key 吗？</p>
          <p style={{ color: '#ff4d4f' }}>此操作不可恢复！</p>
        </div>
      ),
      okText: '确认删除全部',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => doDelete(deleteKeys),
    })
  }

  // 获取配置列表
  const fetchConfigs = async () => {
    try {
      const data = await consulAPI.getConfigs()
      setConfigs(data)
      if (data.length > 0 && !selectedConfigId) {
        const defaultConfig = data.find(c => c.is_default) || data[0]
        setSelectedConfigId(defaultConfig.id)
        allForm.setFieldsValue({ config_id: defaultConfig.id }) // 初始化表单中的配置ID
        fetchProjects(defaultConfig.id)
      }
    } catch (error) {
      message.error('获取配置列表失败')
    }
  }

  // 初始加载配置
  useEffect(() => {
    fetchConfigs()
  }, [])

  // 当configs更新时，设置默认选择
  useEffect(() => {
    if (configs.length > 0 && !selectedConfigId) {
      const defaultConfig = configs.find(c => c.is_default) || configs[0]
      setSelectedConfigId(defaultConfig.id)
      allForm.setFieldsValue({ config_id: defaultConfig.id })
    }
    if (configs.length > 0 && !deleteConfigId) {
      const defaultConfig = configs.find(c => c.is_default) || configs[0]
      setDeleteConfigId(defaultConfig.id)
    }
  }, [configs, selectedConfigId, deleteConfigId, allForm])

  useEffect(() => {
    if (selectedConfigId) {
      fetchProjects(selectedConfigId)
    }
  }, [selectedConfigId])

  // 执行一键复制所有项目
  const handleBatchCopyAll = async (values: any) => {
    if (!values.config_id) {
      message.warning('请先选择 Consul 配置')
      return
    }

    const sourceSuffix = String(values.source_suffix || '').trim()
    const targetSuffix = String(values.target_suffix || '').trim()
    const replaceInPlace = Boolean(values.replace_in_place)

    if (sourceSuffix === targetSuffix && !replaceInPlace) {
      message.warning('源后缀和目标后缀相同时，请开启“原后缀内替换”')
      return
    }

    setLoading(true)
    setResult(null)

    try {
      // 构建高级替换规则
      const replaceRules: RuleItem[] = [];
      if (values.special_replace_rules) {
        values.special_replace_rules.forEach((rule: any) => {
          if (rule.type && rule.old_value && rule.new_value) {
            replaceRules.push({ type: rule.type, old_value: rule.old_value, new_value: rule.new_value });
          }
        });
      }

      // 构建各类替换对
      const buildPairs = (list: any[] | undefined): ReplacePair[] => {
        if (!list) return [];
        return list.filter((p: any) => p.old_pattern && p.new_pattern)
          .map((p: any) => ({ old_pattern: p.old_pattern, new_pattern: p.new_pattern }));
      };

      const request: BatchCopyAllProjectsRequest = {
        config_id: values.config_id,
        source_suffix: sourceSuffix,
        target_suffix: targetSuffix,
        replace_in_place: replaceInPlace,
        projects: values.projects && values.projects.length > 0 ? values.projects : undefined,
        recursive: values.recursive !== false,
        replace_rules: replaceRules,
        tag_replacements: buildPairs(values.tag_replacements),
        server_replacements: buildPairs(values.server_replacements),
        branch_replacements: buildPairs(values.branch_replacements),
        submodule_branch_replacements: buildPairs(values.submodule_branch_replacements),
      }

      const data = await consulAPI.batchCopyAllProjects(request)
      setResult(data)

      if (data.failed === 0) {
        message.success(`${replaceInPlace ? '原地替换' : '批量复制'}完成：成功 ${data.success} 个，耗时 ${data.elapsed_time}`)
      } else {
        message.warning(`${replaceInPlace ? '原地替换' : '批量复制'}完成：成功 ${data.success} 个，失败 ${data.failed} 个`)
      }
    } catch (error: any) {
      message.error(`批量复制失败：${error.message || '未知错误'}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ padding: 24 }}>
      {/* 使用说明卡片 */}
      <Card
        title={
          <Space>
            <QuestionCircleOutlined />
            <span>使用说明</span>
          </Space>
        }
        style={{ marginBottom: 24 }}
        size="small"
      >
        <Paragraph>
          <Text strong>批量配置下发功能使用指南:</Text>
        </Paragraph>
        <Paragraph>
          1. 选择 Consul 配置 - 从下拉菜单中选择要操作的 Consul 实例<br/>
          2. 设置源后缀和目标后缀 - 指定要从哪个环境复制到哪个环境 (如从 V2.5.1 复制到 147)<br/>
          2.1 如果源后缀和目标后缀相同，请开启“原后缀内替换”，此时不会创建新 Key，而是直接更新原 Key 内容<br/>
          3. 选择是否递归复制 - 选择是否复制所有子键<br/>
          4. （可选）配置替换规则 -
          支持Tag、Server、Branch、SubmoduleBranch等特殊替换，
          系统会精确匹配键值对格式，如 tag: "V2.5.0" → tag: "V2.5.1"，
          支持有引号和无引号的值格式：<br/>
          &nbsp;&nbsp;- Tag替换：替换版本标签，如 V2.5.0 → test10<br/>
          &nbsp;&nbsp;- Server替换：替换服务器地址，如 192.168.1.231 → 192.168.1.111<br/>
          &nbsp;&nbsp;- Branch替换：替换分支名称<br/>
          &nbsp;&nbsp;- SubmoduleBranch替换：替换子模块分支<br/>
          5. （可选）配置高级替换规则 - 使用文本替换或正则替换进行更复杂的替换操作<br/>
          6. 点击 "执行批量配置下发" - 开始执行批量配置操作<br/>
          7. 查看复制结果 - 成功/失败的键数量将在结果区域显示
        </Paragraph>
        <Paragraph style={{ marginTop: 16 }}>
          <Text strong>示例:</Text> 将所有项目的 V2.5.1 环境配置复制到 147 环境：
          <Text code>plugin/*/V2.5.1 → plugin/*/147</Text>
        </Paragraph>
        <Paragraph style={{ marginTop: 8 }}>
          <Text type="secondary">注意：系统会精确匹配键值对格式，不会误替换其他内容。</Text>
        </Paragraph>
      </Card>

      <Card
        title={
          <Space>
            <PlayCircleOutlined style={{ color: '#1890ff' }} />
            <span>批量配置下发</span>
          </Space>
        }
        size="small"
        style={{ marginBottom: 16 }}
      >
        <Form
          form={allForm}
          layout="vertical"
          onFinish={handleBatchCopyAll}
          initialValues={{
            source_suffix: 'V2.5.1',
            target_suffix: '',
            replace_in_place: false,
            recursive: true,
            tag_replacements: [],
            server_replacements: [],
            branch_replacements: [],
            submodule_branch_replacements: [],
            special_replace_rules: [],
          }}
        >
          <Row gutter={24}>
            <Col span={8}>
              <Form.Item name="config_id" label="Consul配置" rules={[{ required: true, message: '请选择Consul配置' }]}>
                <Select
                  placeholder="请选择Consul配置"
                  onChange={setSelectedConfigId}
                  options={configs.map(config => ({
                    label: `${config.name} (${config.address})`,
                    value: config.id,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="source_suffix" label="源后缀" rules={[{ required: true, message: '请输入源后缀' }]}>
                <Input placeholder="V2.5.1" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="target_suffix" label="目标后缀" rules={[{ required: true, message: '请输入目标后缀' }]}>
                <Input placeholder="147" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="projects" label="项目选择">
            <Select
              mode="multiple"
              allowClear
              placeholder="默认全部项目，可按需选择部分项目"
              options={projects.map(project => ({ label: project, value: project }))}
            />
          </Form.Item>

          <Row gutter={24}>
            <Col span={8}>
              <Form.Item name="recursive" label="递归复制" valuePropName="checked" style={{ marginTop: 8 }}>
                <Switch defaultChecked />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="replace_in_place" label="原后缀内替换" valuePropName="checked" style={{ marginTop: 8 }}>
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item noStyle shouldUpdate={(prev, curr) =>
            prev.source_suffix !== curr.source_suffix ||
            prev.target_suffix !== curr.target_suffix ||
            prev.replace_in_place !== curr.replace_in_place
          }>
            {({ getFieldValue }) => {
              const sameSuffix = String(getFieldValue('source_suffix') || '').trim() !== '' &&
                String(getFieldValue('source_suffix') || '').trim() === String(getFieldValue('target_suffix') || '').trim()
              const enabled = Boolean(getFieldValue('replace_in_place'))

              if (!sameSuffix && !enabled) {
                return null
              }

              return (
                <Alert
                  style={{ marginBottom: 16 }}
                  type={sameSuffix ? 'warning' : 'info'}
                  showIcon
                  message={sameSuffix ? '将执行原后缀内替换' : '已开启原后缀内替换'}
                  description={sameSuffix
                    ? '源后缀和目标后缀相同。执行后不会创建新 Key，而是直接覆盖原后缀下匹配 Key 的内容，请确认替换规则无误。'
                    : '开启后可支持同后缀场景的原地更新；不同后缀场景下通常无需开启。'}
                />
              )
            }}
          </Form.Item>

          <Divider orientation="left">替换规则（原模式 → 新模式）</Divider>

          {/* Tag 替换 */}
          <Form.Item label="Tag 替换">
            <Form.List name="tag_replacements">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item {...restField} name={[name, 'old_pattern']} rules={[{ required: true, message: '原模式' }]}>
                        <Input placeholder="原模式，如: V2.5.1" style={{ width: 200 }} />
                      </Form.Item>
                      <span>→</span>
                      <Form.Item {...restField} name={[name, 'new_pattern']} rules={[{ required: true, message: '新模式' }]}>
                        <Input placeholder="新模式，如: V2.5.1" style={{ width: 200 }} />
                      </Form.Item>
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} icon={<PlusOutlined />} style={{ width: 440 }}>
                    添加 Tag 替换
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Server 替换 */}
          <Form.Item label="Server 替换">
            <Form.List name="server_replacements">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item {...restField} name={[name, 'old_pattern']} rules={[{ required: true, message: '原模式' }]}>
                        <Input placeholder="原模式，如: dev-server" style={{ width: 200 }} />
                      </Form.Item>
                      <span>→</span>
                      <Form.Item {...restField} name={[name, 'new_pattern']} rules={[{ required: true, message: '新模式' }]}>
                        <Input placeholder="新模式，如: prod-server" style={{ width: 200 }} />
                      </Form.Item>
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} icon={<PlusOutlined />} style={{ width: 440 }}>
                    添加 Server 替换
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Branch 替换 */}
          <Form.Item label="Branch 替换">
            <Form.List name="branch_replacements">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item {...restField} name={[name, 'old_pattern']} rules={[{ required: true, message: '原模式' }]}>
                        <Input placeholder="原模式，如: develop" style={{ width: 200 }} />
                      </Form.Item>
                      <span>→</span>
                      <Form.Item {...restField} name={[name, 'new_pattern']} rules={[{ required: true, message: '新模式' }]}>
                        <Input placeholder="新模式，如: main" style={{ width: 200 }} />
                      </Form.Item>
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} icon={<PlusOutlined />} style={{ width: 440 }}>
                    添加 Branch 替换
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* SubmoduleBranch 替换 */}
          <Form.Item label="SubmoduleBranch 替换">
            <Form.List name="submodule_branch_replacements">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item {...restField} name={[name, 'old_pattern']} rules={[{ required: true, message: '原模式' }]}>
                        <Input placeholder="原模式，如: feature-branch" style={{ width: 200 }} />
                      </Form.Item>
                      <span>→</span>
                      <Form.Item {...restField} name={[name, 'new_pattern']} rules={[{ required: true, message: '新模式' }]}>
                        <Input placeholder="新模式，如: release-v1.0" style={{ width: 200 }} />
                      </Form.Item>
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} icon={<PlusOutlined />} style={{ width: 440 }}>
                    添加 SubmoduleBranch 替换
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          <Divider orientation="left">高级替换规则</Divider>

          {/* 特殊替换规则（支持branch、server等） */}
          <Form.Item label="高级替换规则">
            <Form.List name="special_replace_rules">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item
                        {...restField}
                        name={[name, 'type']}
                        rules={[{ required: true, message: '请选择替换类型' }]}
                      >
                        <Select placeholder="类型" style={{ width: 120 }}>
                          <Select.Option value="text">文本替换</Select.Option>
                          <Select.Option value="regex">正则替换</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'old_value']}
                        rules={[{ required: true, message: '请输入原值' }]}
                      >
                        <Input placeholder="原值" style={{ width: 200 }} />
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'new_value']}
                        rules={[{ required: true, message: '请输入新值' }]}
                      >
                        <Input placeholder="新值" style={{ width: 200 }} />
                      </Form.Item>
                      <MinusCircleOutlined onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Form.Item>
                    <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                      添加替换规则
                    </Button>
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              icon={<PlayCircleOutlined />}
              loading={loading}
              size="large"
              block
            >
              执行批量配置下发
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* 复制结果 */}
      {result && (
        <Card
          title="复制结果"
          size="small"
          style={{ marginTop: 16 }}
          extra={
            <Space>
              {result.failed === 0 ? (
                <Tag icon={<CheckCircleOutlined />} color="success">
                  全部成功
                </Tag>
              ) : (
                <Tag icon={<CloseCircleOutlined />} color="error">
                  部分失败
                </Tag>
              )}
              <Button
                icon={<EyeOutlined />}
                onClick={() => navigate('/consul/config')}
              >
                查看 Consul 配置
              </Button>
              <Button
                icon={<EyeOutlined />}
                onClick={() => navigate('/consul/operations')}
              >
                查看配置操作记录
              </Button>
            </Space>
          }
        >
          <Row gutter={16}>
            <Col span={6}>
              <Alert
                type="success"
                message="成功"
                description={<Text strong style={{ fontSize: 24 }}>{result.success}</Text>}
                showIcon
              />
            </Col>
            <Col span={6}>
              <Alert
                type={result.failed > 0 ? 'warning' : 'success'}
                message="失败"
                description={<Text strong style={{ fontSize: 24 }}>{result.failed}</Text>}
                showIcon
              />
            </Col>
            <Col span={6}>
              <Alert
                type="info"
                message="总计"
                description={<Text strong style={{ fontSize: 24 }}>{result.total}</Text>}
                showIcon
              />
            </Col>
            <Col span={6}>
              <Alert
                type="info"
                message="耗时"
                description={<Text strong style={{ fontSize: 16 }}>{result.elapsed_time}</Text>}
                showIcon
              />
            </Col>
          </Row>
        </Card>
      )}

      {/* 批量删除功能区 */}
      <Card
        title={
          <Space>
            <DeleteOutlined style={{ color: '#ff4d4f' }} />
            <span>批量删除配置</span>
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Row gutter={16} align="middle" style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Select
              placeholder="选择 Consul 配置"
              value={deleteConfigId}
              onChange={setDeleteConfigId}
              style={{ width: '100%' }}
              options={configs.map(config => ({
                label: `${config.name} (${config.address})`,
                value: config.id,
              }))}
            />
          </Col>
          <Col span={6}>
            <Input
              placeholder="输入后缀，如: V2.5.1"
              value={deleteSuffix}
              onChange={e => setDeleteSuffix(e.target.value)}
              onPressEnter={handleQueryDeleteKeys}
              allowClear
            />
          </Col>
          <Col>
            <Button type="primary" icon={<SearchOutlined />} loading={deleteQueryLoading} onClick={handleQueryDeleteKeys}>
              查询 Key
            </Button>
          </Col>
        </Row>

        {deleteKeysQueried && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Text>匹配到 <Text strong>{deleteKeys.length}</Text> 个 Key</Text>
                {selectedDeleteKeys.length > 0 && <Text type="secondary">（已选 {selectedDeleteKeys.length} 个）</Text>}
              </Space>
              <Space style={{ float: 'right' }}>
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  disabled={selectedDeleteKeys.length === 0}
                  onClick={handleDeleteSelectedKeys}
                  loading={deleteLoading}
                >
                  删除选中 ({selectedDeleteKeys.length})
                </Button>
                <Button
                  danger
                  type="primary"
                  icon={<DeleteOutlined />}
                  disabled={deleteKeys.length === 0}
                  onClick={handleDeleteAllKeys}
                  loading={deleteLoading}
                >
                  删除全部 ({deleteKeys.length})
                </Button>
              </Space>
            </div>

            <Table
              rowKey="key"
              columns={[
                { title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true },
              ]}
              dataSource={deleteKeys.map(k => ({ key: k }))}
              size="small"
              pagination={{ pageSize: 20 }}
              rowSelection={{
                selectedRowKeys: selectedDeleteKeys,
                onChange: (keys) => setSelectedDeleteKeys(keys as string[]),
              }}
              locale={{ emptyText: '未找到匹配的 Key' }}
            />
          </>
        )}

        {deleteResult && (
          <Alert
            style={{ marginTop: 16 }}
            message={`删除完成：成功 ${deleteResult.deleted} 个，失败 ${deleteResult.failed} 个`}
            type={deleteResult.failed > 0 ? 'warning' : 'success'}
            showIcon
            closable
            onClose={() => setDeleteResult(null)}
          />
        )}
      </Card>
    </div>
  )
}
