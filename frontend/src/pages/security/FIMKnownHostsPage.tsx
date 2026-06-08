import { useState, useEffect } from 'react';
import type { ColumnsType } from 'antd/es/table';
import {
  Card, Table, Button, Space, Tag, Modal, Form, Input, Select,
  message, Popconfirm, Tooltip, Badge, Drawer, Descriptions,
  Timeline, Alert, Statistic, Row, Col, Typography, Tabs,
  Divider, Empty, Switch
} from 'antd';
import {
  PlusOutlined, DeleteOutlined,
  UploadOutlined, DownloadOutlined, KeyOutlined, SafetyOutlined,
  ExclamationCircleOutlined, CheckCircleOutlined, CopyOutlined,
  EyeOutlined, EyeInvisibleOutlined, SyncOutlined
} from '@ant-design/icons';
import fimKnownHostsAPI, { KnownHost, ConnectionLog, HostHistory } from '../../api/fim-known-hosts';
import { cmdbAPI, Server } from '../../api/cmdb';
import { canEdit } from '../../utils/menuAccess';

const { TextArea } = Input;
const { TabPane } = Tabs;
const { Text, Title } = Typography;

export default function FIMKnownHostsPage() {
  const [hosts, setHosts] = useState<KnownHost[]>([]);
  const [loading, setLoading] = useState(false);
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [importModalVisible, setImportModalVisible] = useState(false);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [selectedHost, setSelectedHost] = useState<KnownHost | null>(null);
  const [hostDetail, setHostDetail] = useState<any>(null);
  const [stats, setStats] = useState({ total: 0, active: 0, used: 0 });
  const [showPublicKey, setShowPublicKey] = useState(false);
  const [servers, setServers] = useState<Server[]>([]);
  const [serversLoading, setServersLoading] = useState(false);

  const [form] = Form.useForm();
  const [importForm] = Form.useForm();
  const [editForm] = Form.useForm();

  useEffect(() => {
    fetchHosts();
  }, []);

  const fetchHosts = async () => {
    setLoading(true);
    try {
      const response = await fimKnownHostsAPI.getHosts();
      setHosts(response.data || []);
      setStats({
        total: response.total || 0,
        active: (response.data || []).filter((h: KnownHost) => h.is_enabled).length,
        used: (response.data || []).filter((h: KnownHost) => h.use_count > 0).length,
      });
    } catch (error) {
      message.error('获取主机列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchServers = async () => {
    setServersLoading(true);
    try {
      const response = await cmdbAPI.getServers({ limit: 500 });
      setServers(response.data || []);
    } catch (error) {
      console.error('获取服务器列表失败', error);
    } finally {
      setServersLoading(false);
    }
  };

  const handleOpenAddModal = () => {
    fetchServers();
    setAddModalVisible(true);
  };

  const handleServerSelect = (serverId: number) => {
    const server = servers.find(s => s.id === serverId);
    if (server) {
      form.setFieldsValue({
        hostname: server.ip || server.hostname,
        port: server.ssh_port || 22,
      });
    }
  };



  const handleAdd = async (values: any) => {
    try {
      await fimKnownHostsAPI.addHost(values);
      message.success('添加成功');
      setAddModalVisible(false);
      form.resetFields();
      fetchHosts();
    } catch (error: any) {
      message.error(error.response?.data?.error || '添加失败');
    }
  };

  const handleEdit = async (values: any) => {
    if (!selectedHost) return;
    try {
      await fimKnownHostsAPI.updateHost(selectedHost.id, values);
      message.success('更新成功');
      fetchHosts();
      if (detailDrawerVisible) {
        showDetail(selectedHost);
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || '更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await fimKnownHostsAPI.deleteHost(id);
      message.success('删除成功');
      fetchHosts();
      if (detailDrawerVisible && selectedHost?.id === id) {
        setDetailDrawerVisible(false);
      }
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleImport = async (values: any) => {
    try {
      const response = await fimKnownHostsAPI.importHosts(values.content);
      message.success(`成功导入 ${response.imported} 个主机密钥，跳过 ${response.skipped} 个`);
      if (response.errors?.length > 0) {
        console.warn('导入错误:', response.errors);
      }
      setImportModalVisible(false);
      importForm.resetFields();
      fetchHosts();
    } catch (error: any) {
      message.error(error.response?.data?.error || '导入失败');
    }
  };

  const handleExport = async () => {
    try {
      const blob = await fimKnownHostsAPI.exportHosts();
      const url = window.URL.createObjectURL(new Blob([blob]));
      const a = document.createElement('a');
      a.href = url;
      a.download = 'known_hosts';
      a.click();
      window.URL.revokeObjectURL(url);
      message.success('导出成功');
    } catch (error) {
      message.error('导出失败');
    }
  };

  const showDetail = async (host: KnownHost) => {
    try {
      const data = await fimKnownHostsAPI.getHost(host.id);
      setSelectedHost(host);
      setHostDetail(data);
      setDetailDrawerVisible(true);
      editForm.setFieldsValue({
        description: host.description,
        is_enabled: host.is_enabled,
      });
    } catch (error) {
      message.error('获取详情失败');
    }
  };

  const formatTime = (time?: string) => {
    if (!time) return '-';
    return new Date(time).toLocaleString('zh-CN');
  };

  const getResultTag = (result: string) => {
    const config: Record<string, { color: string; text: string }> = {
      success: { color: 'success', text: '成功' },
      key_not_found: { color: 'error', text: '密钥不存在' },
      key_mismatch: { color: 'warning', text: '密钥不匹配' },
      connection_failed: { color: 'default', text: '连接失败' },
    };
    const { color, text } = config[result] || { color: 'default', text: result };
    return <Tag color={color}>{text}</Tag>;
  };

  const columns: ColumnsType<KnownHost> = [
    {
      title: '主机',
      key: 'host',
      width: 200,
      render: (record: KnownHost) => (
        <Space direction="vertical" size={0}>
          <Text strong>{record.hostname}</Text>
          {record.port !== 22 && <Tag color="blue">端口: {record.port}</Tag>}
        </Space>
      ),
    },
    {
      title: '密钥类型',
      dataIndex: 'key_type',
      key: 'key_type',
      width: 150,
      render: (type: string) => {
        const colors: Record<string, string> = {
          'ssh-rsa': 'blue',
          'ssh-ed25519': 'green',
          'ecdsa-sha2-nistp256': 'cyan',
          'ecdsa-sha2-nistp384': 'cyan',
          'ecdsa-sha2-nistp521': 'cyan',
        };
        return <Tag color={colors[type] || 'default'}>{type}</Tag>;
      },
    },
    {
      title: '指纹 (SHA256)',
      dataIndex: 'fingerprint_sha256',
      key: 'fingerprint',
      width: 180,
      render: (fp: string) => (
        <Tooltip title={fp}>
          <Text code style={{ fontSize: 11 }} copyable={{ text: fp }}>
            {fp.substring(0, 16)}...
          </Text>
        </Tooltip>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render: (record: KnownHost) => (
        <Space>
          <Badge
            status={record.is_enabled ? 'success' : 'default'}
            text={record.is_enabled ? '已启用' : '已禁用'}
          />
        </Space>
      ),
    },
    {
      title: '使用统计',
      key: 'stats',
      width: 150,
      render: (record: KnownHost) => (
        <Space direction="vertical" size={0}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            次数: <Text strong>{record.use_count}</Text>
          </Text>
          {record.last_used_at && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              最后: {formatTime(record.last_used_at)}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '添加人',
      dataIndex: 'added_by',
      key: 'added_by',
      width: 100,
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right' as const,
      render: (record: KnownHost) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => showDetail(record)}
          >
            详情
          </Button>
          {canEdit() && (
            <Popconfirm
              title="确定要删除此主机密钥吗？"
              description="删除后，FIM扫描将无法连接该主机。请确保不再监控此服务器。"
              onConfirm={() => handleDelete(record.id)}
              okText="确定"
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
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总主机数"
              value={stats.total}
              prefix={<SafetyOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已启用"
              value={stats.active}
              valueStyle={{ color: '#3f8600' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已使用"
              value={stats.used}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="未使用"
              value={stats.total - stats.used}
              valueStyle={{ color: '#cf1322' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 主卡片 */}
      <Card
        title={
          <Space>
            <KeyOutlined />
            <span>FIM主机密钥白名单</span>
          </Space>
        }
        extra={
          <Space>
            {canEdit() && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleOpenAddModal}
              >
                添加主机
              </Button>
            )}
            <Button
              icon={<UploadOutlined />}
              onClick={() => setImportModalVisible(true)}
            >
              导入
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={handleExport}
            >
              导出
            </Button>
            <Button
              icon={<SyncOutlined />}
              onClick={fetchHosts}
            >
              刷新
            </Button>
          </Space>
        }
      >
        {/* 提示信息 */}
        <Alert
          message="严格模式：只有在白名单中的主机密钥才能连接，密钥需要在系统设置-FIM SSH中配置。"
          description={
            <div>
              <p>添加FIM目标服务器前，请先在此处添加其SSH主机公钥。这可以防止中间人攻击。</p>
              <p>获取主机公钥的方法：</p>
              <ol style={{ margin: 0, paddingLeft: 20 }}>
                <li>在目标服务器执行: <code>cat /etc/ssh/ssh_host_*_key.pub</code></li>
                <li>或使用本页面的"导入"功能从known_hosts文件导入</li>
              </ol>
            </div>
          }
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Table
          columns={columns}
          dataSource={hosts}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            defaultPageSize: 20,
          }}
        />
      </Card>

      {/* 添加主机弹窗 */}
      <Modal
        title="添加主机密钥"
        open={addModalVisible}
        onCancel={() => setAddModalVisible(false)}
        footer={null}
        width={700}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleAdd}
          initialValues={{ port: 22, key_type: 'ecdsa-sha2-nistp256' }}
        >
          <Form.Item
            label="从主机管理选择"
            name="server_id"
          >
            <Select
              placeholder="从 CMDB 主机列表选择（可选）"
              showSearch
              allowClear
              loading={serversLoading}
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              onChange={handleServerSelect}
              options={servers.map(server => ({
                value: server.id,
                label: `${server.hostname} (${server.ip}) - ${server.ssh_port}`,
              }))}
            />
          </Form.Item>

          <Divider style={{ margin: '12px 0' }} orientation="left">或手动输入</Divider>

          <Form.Item
            label="主机名/IP"
            name="hostname"
            rules={[{ required: true, message: '请输入主机名或IP' }]}
          >
            <Input placeholder="例如：192.168.1.100 或 server.example.com" />
          </Form.Item>

          <Form.Item
            label="SSH端口"
            name="port"
            rules={[{ required: true, message: '请输入端口' }]}
          >
            <Input type="number" placeholder="默认22" />
          </Form.Item>

          <Form.Item
            label="密钥类型"
            name="key_type"
            rules={[{ required: true, message: '请选择密钥类型' }]}
          >
            <Select placeholder="选择密钥类型">
              <Select.Option value="ecdsa-sha2-nistp256">
                <Space>
                  ecdsa-sha2-nistp256
                  <Tag color="green">推荐</Tag>
                </Space>
              </Select.Option>
              <Select.Option value="ssh-ed25519">
                ssh-ed25519
              </Select.Option>
              <Select.Option value="ssh-rsa">ssh-rsa</Select.Option>
              <Select.Option value="ecdsa-sha2-nistp384">ecdsa-sha2-nistp384</Select.Option>
              <Select.Option value="ecdsa-sha2-nistp521">ecdsa-sha2-nistp521</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="公钥内容"
            name="public_key"
            rules={[{ required: true, message: '请输入公钥' }]}
            extra={
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">获取方式：</Text>
                <ol style={{ margin: 4, paddingLeft: 20 }}>
                  <li>在目标服务器执行: <code>cat /etc/ssh/ssh_host_*_key.pub</code></li>
                  <li>或使用本页面的"导入"功能从known_hosts文件导入</li>
                  <li>粘贴完整的公钥内容（包括密钥类型前缀）</li>
                </ol>
              </div>
            }
          >
            <TextArea
              rows={4}
              placeholder="ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBB..."
            />
          </Form.Item>

          <Form.Item
            label="描述"
            name="description"
          >
            <Input placeholder="主机描述或备注，例如：生产服务器-应用01" />
          </Form.Item>

          <Form.Item
            label="标签"
            name="tags"
          >
            <Select mode="tags" placeholder="添加标签，如：production, critical">
              <Select.Option value="production">production</Select.Option>
              <Select.Option value="staging">staging</Select.Option>
              <Select.Option value="critical">critical</Select.Option>
              <Select.Option value="database">database</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                添加
              </Button>
              <Button onClick={() => setAddModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 导入弹窗 */}
      <Modal
        title="导入主机密钥"
        open={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
      >
        <Alert
          message="从 known_hosts 文件导入"
          description="粘贴或上传 known_hosts 格式的内容。重复的密钥会被自动跳过。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form
          form={importForm}
          layout="vertical"
          onFinish={handleImport}
        >
          <Form.Item
            label="known_hosts 内容"
            name="content"
            rules={[{ required: true, message: '请输入内容' }]}
            extra={
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">格式示例：</Text>
                <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 12 }}>
{`# 格式1: 默认端口
server.example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...

# 格式2: 指定端口
[server.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...`}
                </pre>
              </div>
            }
          >
            <TextArea
              rows={12}
              placeholder="粘贴 known_hosts 内容..."
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                导入
              </Button>
              <Button onClick={() => setImportModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={
          <Space>
            <KeyOutlined />
            <span>主机密钥详情</span>
          </Space>
        }
        placement="right"
        width={700}
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
        destroyOnClose
      >
        {selectedHost && (
          <Tabs defaultActiveKey="info">
            <TabPane tab="基本信息" key="info">
              <Descriptions column={1} bordered size="small">
                <Descriptions.Item label="主机">
                  <Text strong style={{ fontSize: 16 }}>
                    {selectedHost.hostname}:{selectedHost.port}
                  </Text>
                </Descriptions.Item>
                <Descriptions.Item label="密钥类型">
                  <Tag color="blue">{selectedHost.key_type}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="指纹 (SHA256)">
                  <Space>
                    <Text code copyable>
                      {selectedHost.fingerprint_sha256}
                    </Text>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="描述">
                  {selectedHost.description || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="标签">
                  {selectedHost.tags && selectedHost.tags !== '' ? (
                    <Space wrap>
                      {(JSON.parse(selectedHost.tags) as string[]).map((tag, i) => (
                        <Tag key={i}>{tag}</Tag>
                      ))}
                    </Space>
                  ) : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Badge
                    status={selectedHost.is_enabled ? 'success' : 'default'}
                    text={selectedHost.is_enabled ? '已启用' : '已禁用'}
                  />
                </Descriptions.Item>
                <Descriptions.Item label="验证状态">
                  <Tag color={selectedHost.verification_status === 'verified' ? 'green' : 'orange'}>
                    {selectedHost.verification_status}
                  </Tag>
                  {selectedHost.verified_by && (
                    <Text type="secondary" style={{ marginLeft: 8 }}>
                      由 {selectedHost.verified_by} 于 {formatTime(selectedHost.verified_at)} 验证
                    </Text>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="使用统计">
                  <Space direction="vertical">
                    <Text>使用次数: <Text strong>{selectedHost.use_count}</Text></Text>
                    <Text>最后使用: {formatTime(selectedHost.last_used_at)}</Text>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="添加信息">
                  <Space direction="vertical">
                    <Text>添加人: {selectedHost.added_by}</Text>
                    <Text>添加时间: {formatTime(selectedHost.added_at)}</Text>
                  </Space>
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              {/* 编辑表单 */}
              <Title level={5}>编辑信息</Title>
              <Form
                form={editForm}
                layout="vertical"
                onFinish={handleEdit}
              >
                <Form.Item label="描述" name="description">
                  <Input placeholder="主机描述或备注" />
                </Form.Item>

                <Form.Item label="标签" name="tags">
                  <Select mode="tags" placeholder="添加标签">
                    <Select.Option value="production">production</Select.Option>
                    <Select.Option value="staging">staging</Select.Option>
                    <Select.Option value="critical">critical</Select.Option>
                  </Select>
                </Form.Item>

                <Form.Item label="状态" name="is_enabled" valuePropName="checked">
                  <Switch checkedChildren="已启用" unCheckedChildren="已禁用" />
                </Form.Item>

                <Form.Item>
                  <Space>
                    <Button type="primary" htmlType="submit">
                      保存修改
                    </Button>
                    {canEdit() && (
                      <Popconfirm
                        title="确定要删除此主机密钥吗？"
                        description="删除后，FIM扫描将无法连接该主机。"
                        onConfirm={() => handleDelete(selectedHost.id)}
                        okText="确定"
                        cancelText="取消"
                      >
                        <Button danger>删除主机</Button>
                      </Popconfirm>
                    )}
                  </Space>
                </Form.Item>
              </Form>
            </TabPane>

            <TabPane tab="公钥" key="key">
              <Space direction="vertical" style={{ width: '100%' }}>
                <Space>
                  <Button
                    icon={showPublicKey ? <EyeInvisibleOutlined /> : <EyeOutlined />}
                    onClick={() => setShowPublicKey(!showPublicKey)}
                  >
                    {showPublicKey ? '隐藏' : '显示'}
                  </Button>
                  <Button
                    icon={<CopyOutlined />}
                    onClick={() => { navigator.clipboard.writeText(selectedHost.public_key); message.success('已复制'); }}
                  >
                    复制公钥
                  </Button>
                  <Button
                    icon={<CopyOutlined />}
                    onClick={() => { navigator.clipboard.writeText(selectedHost.fingerprint_sha256); message.success('已复制'); }}
                  >
                    复制指纹
                  </Button>
                </Space>
                <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, overflow: 'auto', maxHeight: 400 }}>
                  {showPublicKey ? selectedHost.public_key : '••••••••••••••••••••••••••••••••••••••••••••••••••'}
                </pre>
              </Space>
            </TabPane>

            <TabPane tab="连接历史" key="logs">
              {hostDetail?.logs?.length > 0 ? (
                <Timeline>
                  {hostDetail.logs.map((log: ConnectionLog, i: number) => (
                    <Timeline.Item key={i}>
                      <div>
                        <Space>
                          {getResultTag(log.result)}
                          <Text type="secondary">{formatTime(log.attempted_at)}</Text>
                        </Space>
                        {log.error_message && (
                          <div style={{ marginTop: 4 }}>
                            <Text type="danger" style={{ fontSize: 12 }}>
                              {log.error_message}
                            </Text>
                          </div>
                        )}
                        {log.presented_fingerprint && (
                          <div style={{ marginTop: 4 }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              提供指纹: {log.presented_fingerprint}
                            </Text>
                          </div>
                        )}
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              ) : (
                <Empty description="暂无连接记录" />
              )}
            </TabPane>

            <TabPane tab="变更历史" key="history">
              {hostDetail?.history?.length > 0 ? (
                <Timeline>
                  {hostDetail.history.map((h: HostHistory, i: number) => (
                    <Timeline.Item key={i}>
                      <div>
                        <Space>
                          <Tag color={h.action === 'added' ? 'green' : h.action === 'deleted' ? 'red' : 'blue'}>
                            {h.action === 'added' ? '添加' : h.action === 'deleted' ? '删除' : h.action === 'updated' ? '更新' : '密钥变更'}
                          </Tag>
                          <Text type="secondary">{formatTime(h.operated_at)}</Text>
                        </Space>
                        <div style={{ marginTop: 4 }}>
                          <Text style={{ fontSize: 12 }}>操作人: {h.operated_by}</Text>
                        </div>
                        {h.reason && (
                          <div style={{ marginTop: 4 }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              原因: {h.reason}
                            </Text>
                          </div>
                        )}
                        {h.old_fingerprint && h.new_fingerprint && (
                          <div style={{ marginTop: 4, padding: 8, background: '#fff2f0', borderRadius: 4 }}>
                            <Text type="danger" style={{ fontSize: 12, display: 'block' }}>
                              原指纹: {h.old_fingerprint}
                            </Text>
                            <Text type="success" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
                              新指纹: {h.new_fingerprint}
                            </Text>
                          </div>
                        )}
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              ) : (
                <Empty description="暂无变更记录" />
              )}
            </TabPane>
          </Tabs>
        )}
      </Drawer>
    </div>
  );
}
