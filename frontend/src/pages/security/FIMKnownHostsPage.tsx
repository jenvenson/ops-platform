// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

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
import { formatDateTime } from '../../utils/dateFormat';
import { useTranslation } from 'react-i18next';

const { TextArea } = Input;
const { TabPane } = Tabs;
const { Text, Title } = Typography;

export default function FIMKnownHostsPage() {
  const { t } = useTranslation('security')
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
      message.error(t('fimKnownHosts.loadHostsFailed', '获取主机列表失败'));
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
      message.success(t('fimKnownHosts.addHostSuccess', '添加成功'));
      setAddModalVisible(false);
      form.resetFields();
      fetchHosts();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('fimKnownHosts.addHostFailed', '添加失败'));
    }
  };

  const handleEdit = async (values: any) => {
    if (!selectedHost) return;
    try {
      await fimKnownHostsAPI.updateHost(selectedHost.id, values);
      message.success(t('fimKnownHosts.updateHostSuccess', '更新成功'));
      fetchHosts();
      if (detailDrawerVisible) {
        showDetail(selectedHost);
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || t('fimKnownHosts.updateHostFailed', '更新失败'));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await fimKnownHostsAPI.deleteHost(id);
      message.success(t('fimKnownHosts.deleteHostSuccess', '删除成功'));
      fetchHosts();
      if (detailDrawerVisible && selectedHost?.id === id) {
        setDetailDrawerVisible(false);
      }
    } catch (error) {
      message.error(t('fimKnownHosts.deleteHostFailed', '删除失败'));
    }
  };

  const handleImport = async (values: any) => {
    try {
      const response = await fimKnownHostsAPI.importHosts(values.content);
      message.success(t('fimKnownHosts.importSuccess', '成功导入 {{imported}} 个主机密钥，跳过 {{skipped}} 个', { imported: response.imported, skipped: response.skipped }));
      if (response.errors?.length > 0) {
        console.warn('导入错误:', response.errors);
      }
      setImportModalVisible(false);
      importForm.resetFields();
      fetchHosts();
    } catch (error: any) {
      message.error(error.response?.data?.error || t('fimKnownHosts.importFailed', '导入失败'));
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
      message.success(t('fimKnownHosts.exportSuccess', '导出成功'));
    } catch (error) {
      message.error(t('fimKnownHosts.exportFailed', '导出失败'));
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
      message.error(t('fimKnownHosts.loadDetailFailed', '获取详情失败'));
    }
  };

  const getResultTag = (result: string) => {
    const config: Record<string, { color: string; text: string }> = {
      success: { color: 'success', text: t('status.success', '成功') },
      key_not_found: { color: 'error', text: t('status.keyNotFound', '密钥不存在') },
      key_mismatch: { color: 'warning', text: t('status.keyMismatch', '密钥不匹配') },
      connection_failed: { color: 'default', text: t('status.connectionFailed', '连接失败') },
    };
    const { color, text } = config[result] || { color: 'default', text: result };
    return <Tag color={color}>{text}</Tag>;
  };

  const columns: ColumnsType<KnownHost> = [
    {
      title: t('fimKnownHosts.host', '主机'),
      key: 'host',
      width: 200,
      render: (record: KnownHost) => (
        <Space direction="vertical" size={0}>
          <Text strong>{record.hostname}</Text>
          {record.port !== 22 && <Tag color="blue">{t('fimKnownHosts.portLabel', '端口')}: {record.port}</Tag>}
        </Space>
      ),
    },
    {
      title: t('fimKnownHosts.keyType', '密钥类型'),
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
      title: t('fimKnownHosts.fingerprintSHA256', '指纹 (SHA256)'),
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
      title: t('fimKnownHosts.description', '描述'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('fimKnownHosts.status', '状态'),
      key: 'status',
      width: 120,
      render: (record: KnownHost) => (
        <Space>
          <Badge
            status={record.is_enabled ? 'success' : 'default'}
            text={record.is_enabled ? t('fimKnownHosts.statusEnabled', '已启用') : t('fimKnownHosts.statusDisabled', '已禁用')}
          />
        </Space>
      ),
    },
    {
      title: t('fimKnownHosts.usageStats', '使用统计'),
      key: 'stats',
      width: 150,
      render: (record: KnownHost) => (
        <Space direction="vertical" size={0}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t('fimKnownHosts.useCount', '次数')}: <Text strong>{record.use_count}</Text>
          </Text>
          {record.last_used_at && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('fimKnownHosts.lastUsed', '最后')}: {formatDateTime(record.last_used_at)}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: t('fimKnownHosts.addedBy', '添加人'),
      dataIndex: 'added_by',
      key: 'added_by',
      width: 100,
    },
    {
      title: t('fimKnownHosts.action', '操作'),
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
            {t('detail', '详情')}
          </Button>
          {canEdit() && (
            <Popconfirm
              title={t('fimKnownHosts.confirmDeleteHost', '确定要删除此主机密钥吗？')}
              description={t('fimKnownHosts.confirmDeleteHostDesc', '删除后，FIM扫描将无法连接该主机。请确保不再监控此服务器。')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('confirm', '确定')}
              cancelText={t('cancel', '取消')}
            >
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
              >
                {t('delete', '删除')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('fimKnownHosts.totalHosts', '总主机数')}
              value={stats.total}
              prefix={<SafetyOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('fimKnownHosts.enabledHosts', '已启用')}
              value={stats.active}
              valueStyle={{ color: '#3f8600' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('fimKnownHosts.usedHosts', '已使用')}
              value={stats.used}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title={t('fimKnownHosts.unusedHosts', '未使用')}
              value={stats.total - stats.used}
              valueStyle={{ color: '#cf1322' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <Space>
            <KeyOutlined />
            <span>{t('fimKnownHosts.title', 'FIM主机密钥白名单')}</span>
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
                {t('fimKnownHosts.addHost', '添加主机')}
              </Button>
            )}
            <Button
              icon={<UploadOutlined />}
              onClick={() => setImportModalVisible(true)}
            >
              {t('fimKnownHosts.import', '导入')}
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={handleExport}
            >
              {t('fimKnownHosts.export', '导出')}
            </Button>
            <Button
              icon={<SyncOutlined />}
              onClick={fetchHosts}
            >
              {t('fimKnownHosts.refresh', '刷新')}
            </Button>
          </Space>
        }
      >
        <Alert
          message={t('fimKnownHosts.strictModeNotice', '严格模式：只有在白名单中的主机密钥才能连接，密钥需要在系统设置-FIM SSH中配置。')}
          description={
            <div>
              <p>{t('fimKnownHosts.addFIMTargetNotice', '添加FIM目标服务器前，请先在此处添加其SSH主机公钥。这可以防止中间人攻击。')}</p>
              <p>{t('fimKnownHosts.getPublicKeyMethod', '获取主机公钥的方法：')}</p>
              <ol style={{ margin: 0, paddingLeft: 20 }}>
                <li>{t('fimKnownHosts.getPublicKeyStep1', '在目标服务器执行: cat /etc/ssh/ssh_host_*_key.pub')}</li>
                <li>{t('fimKnownHosts.getPublicKeyStep2', '或使用本页面的"导入"功能从known_hosts文件导入')}</li>
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
            showTotal: (total) => `${t('total', '共 {{count}} 条', { count: total })}`,
            defaultPageSize: 20,
          }}
        />
      </Card>

      <Modal
        title={t('fimKnownHosts.addHostKey', '添加主机密钥')}
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
            label={t('fimKnownHosts.selectFromCMDB', '从主机管理选择')}
            name="server_id"
          >
            <Select
              placeholder={t('fimKnownHosts.selectFromCMDBPlaceholder', '从 CMDB 主机列表选择（可选）')}
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

          <Divider style={{ margin: '12px 0' }} orientation="left">{t('fimKnownHosts.orManualInput', '或手动输入')}</Divider>

          <Form.Item
            label={t('fimKnownHosts.hostnameIP', '主机名/IP')}
            name="hostname"
            rules={[{ required: true, message: t('fimKnownHosts.hostnameIPRequired', '请输入主机名或IP') }]}
          >
            <Input placeholder={t('fimKnownHosts.hostnameIPPlaceholder', '例如：192.168.1.100 或 server.example.com')} />
          </Form.Item>

          <Form.Item
            label={t('fimKnownHosts.sshPort', 'SSH端口')}
            name="port"
            rules={[{ required: true, message: t('fimKnownHosts.sshPortRequired', '请输入端口') }]}
          >
            <Input type="number" placeholder={t('fimKnownHosts.sshPortPlaceholder', '默认22')} />
          </Form.Item>

          <Form.Item
            label={t('fimKnownHosts.keyType', '密钥类型')}
            name="key_type"
            rules={[{ required: true, message: t('fimKnownHosts.keyTypeRequired', '请选择密钥类型') }]}
          >
            <Select placeholder={t('fimKnownHosts.selectKeyType', '选择密钥类型')}>
              <Select.Option value="ecdsa-sha2-nistp256">
                <Space>
                  ecdsa-sha2-nistp256
                  <Tag color="green">{t('fimKnownHosts.recommended', '推荐')}</Tag>
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
            label={t('fimKnownHosts.publicKey', '公钥内容')}
            name="public_key"
            rules={[{ required: true, message: t('fimKnownHosts.publicKeyRequired', '请输入公钥') }]}
            extra={
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">{t('fimKnownHosts.publicKeyHint', '获取方式：')}</Text>
                <ol style={{ margin: 4, paddingLeft: 20 }}>
                  <li>{t('fimKnownHosts.publicKeyStep1', '在目标服务器执行: cat /etc/ssh/ssh_host_*_key.pub')}</li>
                  <li>{t('fimKnownHosts.publicKeyStep2', '或使用本页面的"导入"功能从known_hosts文件导入')}</li>
                  <li>{t('fimKnownHosts.publicKeyStep3', '粘贴完整的公钥内容（包括密钥类型前缀）')}</li>
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
            label={t('fimKnownHosts.description', '描述')}
            name="description"
          >
            <Input placeholder={t('fimKnownHosts.hostDescription', '主机描述或备注，例如：生产服务器-应用01')} />
          </Form.Item>

          <Form.Item
            label={t('tags', '标签')}
            name="tags"
          >
            <Select mode="tags" placeholder={t('fimKnownHosts.addTags', '添加标签，如：production, critical')}>
              <Select.Option value="production">production</Select.Option>
              <Select.Option value="staging">staging</Select.Option>
              <Select.Option value="critical">critical</Select.Option>
              <Select.Option value="database">database</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('add', '添加')}
              </Button>
              <Button onClick={() => setAddModalVisible(false)}>
                {t('cancel', '取消')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('fimKnownHosts.import', '导入主机密钥')}
        open={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
      >
        <Alert
          message={t('fimKnownHosts.importFromKnownHosts', '从 known_hosts 文件导入')}
          description={t('fimKnownHosts.importFromKnownHostsDesc', '粘贴或上传 known_hosts 格式的内容。重复的密钥会被自动跳过。')}
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
            label={t('fimKnownHosts.knownHostsContent', 'known_hosts 内容')}
            name="content"
            rules={[{ required: true, message: t('fimKnownHosts.knownHostsContentRequired', '请输入内容') }]}
            extra={
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">{t('fimKnownHosts.importFormatExample', '格式示例：')}</Text>
                <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, fontSize: 12 }}>
                  {t('fimKnownHosts.importFormatExampleDesc', '格式1: 默认端口\nserver.example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...\n\n格式2: 指定端口\n[server.example.com]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...')}
                </pre>
              </div>
            }
          >
            <TextArea
              rows={12}
              placeholder={t('fimKnownHosts.pasteKnownHostsContent', '粘贴 known_hosts 内容...')}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('fimKnownHosts.import', '导入')}
              </Button>
              <Button onClick={() => setImportModalVisible(false)}>
                {t('cancel', '取消')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={
          <Space>
            <KeyOutlined />
            <span>{t('fimKnownHosts.hostKeyDetail', '主机密钥详情')}</span>
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
            <TabPane tab={t('basicInfo', '基本信息')} key="info">
              <Descriptions column={1} bordered size="small">
                <Descriptions.Item label={t('fimKnownHosts.host', '主机')}>
                  <Text strong style={{ fontSize: 16 }}>
                    {selectedHost.hostname}:{selectedHost.port}
                  </Text>
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.keyType', '密钥类型')}>
                  <Tag color="blue">{selectedHost.key_type}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.fingerprint', '指纹 (SHA256)')}>
                  <Space>
                    <Text code copyable>
                      {selectedHost.fingerprint_sha256}
                    </Text>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.description', '描述')}>
                  {selectedHost.description || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('tags', '标签')}>
                  {selectedHost.tags && selectedHost.tags !== '' ? (
                    <Space wrap>
                      {(JSON.parse(selectedHost.tags) as string[]).map((tag, i) => (
                        <Tag key={i}>{tag}</Tag>
                      ))}
                    </Space>
                  ) : '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('common:status', '状态')}>
                  <Badge
                    status={selectedHost.is_enabled ? 'success' : 'default'}
                    text={selectedHost.is_enabled ? t('fimKnownHosts.statusEnabled', '已启用') : t('fimKnownHosts.statusDisabled', '已禁用')}
                  />
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.verificationStatus', '验证状态')}>
                  <Tag color={selectedHost.verification_status === 'verified' ? 'green' : 'orange'}>
                    {selectedHost.verification_status}
                  </Tag>
                  {selectedHost.verified_by && (
                    <Text type="secondary" style={{ marginLeft: 8 }}>
                      {t('fimKnownHosts.verifiedBy', '由 {{user}} 于 {{time}} 验证', { user: selectedHost.verified_by, time: formatDateTime(selectedHost.verified_at) })}
                    </Text>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.usageStats', '使用统计')}>
                  <Space direction="vertical">
                    <Text>{t('fimKnownHosts.useCountLabel', '使用次数')}: <Text strong>{selectedHost.use_count}</Text></Text>
                    <Text>{t('fimKnownHosts.lastUsedLabel', '最后使用')}: {formatDateTime(selectedHost.last_used_at)}</Text>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label={t('fimKnownHosts.addInfo', '添加信息')}>
                  <Space direction="vertical">
                    <Text>{t('fimKnownHosts.addedBy', '添加人')}: {selectedHost.added_by}</Text>
                    <Text>{t('fimKnownHosts.addTime', '添加时间')}: {formatDateTime(selectedHost.added_at)}</Text>
                  </Space>
                </Descriptions.Item>
              </Descriptions>

              <Divider />

              <Title level={5}>{t('fimKnownHosts.editInfo', '编辑信息')}</Title>
              <Form
                form={editForm}
                layout="vertical"
                onFinish={handleEdit}
              >
                <Form.Item label={t('fimKnownHosts.description', '描述')} name="description">
                  <Input placeholder={t('fimKnownHosts.hostDescription', '主机描述或备注')} />
                </Form.Item>

                <Form.Item label={t('tags', '标签')} name="tags">
                  <Select mode="tags" placeholder={t('fimKnownHosts.addTags', '添加标签')}>
                    <Select.Option value="production">production</Select.Option>
                    <Select.Option value="staging">staging</Select.Option>
                    <Select.Option value="critical">critical</Select.Option>
                  </Select>
                </Form.Item>

                <Form.Item label={t('common:status', '状态')} name="is_enabled" valuePropName="checked">
                  <Switch checkedChildren={t('fimKnownHosts.statusEnabled', '已启用')} unCheckedChildren={t('fimKnownHosts.statusDisabled', '已禁用')} />
                </Form.Item>

                <Form.Item>
                  <Space>
                    <Button type="primary" htmlType="submit">
                      {t('fimKnownHosts.saveChanges', '保存修改')}
                    </Button>
                    {canEdit() && (
                      <Popconfirm
                        title={t('fimKnownHosts.confirmDeleteHostShort', '确定要删除此主机密钥吗？')}
                        description={t('fimKnownHosts.confirmDeleteHostShortDesc', '删除后，FIM扫描将无法连接该主机。')}
                        onConfirm={() => handleDelete(selectedHost.id)}
                        okText={t('confirm', '确定')}
                        cancelText={t('cancel', '取消')}
                      >
                        <Button danger>{t('fimKnownHosts.deleteHost', '删除主机')}</Button>
                      </Popconfirm>
                    )}
                  </Space>
                </Form.Item>
              </Form>
            </TabPane>

            <TabPane tab={t('fimKnownHosts.publicKeyTab', '公钥')} key="key">
              <Space direction="vertical" style={{ width: '100%' }}>
                <Space>
                  <Button
                    icon={showPublicKey ? <EyeInvisibleOutlined /> : <EyeOutlined />}
                    onClick={() => setShowPublicKey(!showPublicKey)}
                  >
                    {showPublicKey ? t('fimKnownHosts.hide', '隐藏') : t('fimKnownHosts.show', '显示')}
                  </Button>
                  <Button
                    icon={<CopyOutlined />}
                    onClick={() => { navigator.clipboard.writeText(selectedHost.public_key); message.success(t('fimKnownHosts.copied', '已复制')); }}
                  >
                    {t('fimKnownHosts.copyPublicKey', '复制公钥')}
                  </Button>
                  <Button
                    icon={<CopyOutlined />}
                    onClick={() => { navigator.clipboard.writeText(selectedHost.fingerprint_sha256); message.success(t('fimKnownHosts.copied', '已复制')); }}
                  >
                    {t('fimKnownHosts.copyFingerprint', '复制指纹')}
                  </Button>
                </Space>
                <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, overflow: 'auto', maxHeight: 400 }}>
                  {showPublicKey ? selectedHost.public_key : '••••••••••••••••••••••••••••••••••••••••••••••••••'}
                </pre>
              </Space>
            </TabPane>

            <TabPane tab={t('fimKnownHosts.connectionHistory', '连接历史')} key="logs">
              {hostDetail?.logs?.length > 0 ? (
                <Timeline>
                  {hostDetail.logs.map((log: ConnectionLog, i: number) => (
                    <Timeline.Item key={i}>
                      <div>
                        <Space>
                          {getResultTag(log.result)}
                          <Text type="secondary">{formatDateTime(log.attempted_at)}</Text>
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
                              {t('fimKnownHosts.presentedFingerprint', '提供指纹')}: {log.presented_fingerprint}
                            </Text>
                          </div>
                        )}
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              ) : (
                <Empty description={t('fimKnownHosts.noConnectionLogs', '暂无连接记录')} />
              )}
            </TabPane>

            <TabPane tab={t('fimKnownHosts.changeHistory', '变更历史')} key="history">
              {hostDetail?.history?.length > 0 ? (
                <Timeline>
                  {hostDetail.history.map((h: HostHistory, i: number) => {
                    const changedFromLabels: Record<string, string> = {
                      added: t('fimKnownHosts.changedFrom.added', '添加'),
                      deleted: t('fimKnownHosts.changedFrom.deleted', '删除'),
                      updated: t('fimKnownHosts.changedFrom.updated', '更新'),
                      key_changed: t('fimKnownHosts.changedFrom.keyChange', '密钥变更'),
                    }
                    const actionLabel = changedFromLabels[h.action] || h.action
                    return (
                    <Timeline.Item key={i}>
                      <div>
                        <Space>
                          <Tag color={h.action === 'added' ? 'green' : h.action === 'deleted' ? 'red' : 'blue'}>
                            {actionLabel}
                          </Tag>
                          <Text type="secondary">{formatDateTime(h.operated_at)}</Text>
                        </Space>
                        <div style={{ marginTop: 4 }}>
                          <Text style={{ fontSize: 12 }}>{t('fimKnownHosts.operatorInfo', '操作人')}: {h.operated_by}</Text>
                        </div>
                        {h.reason && (
                          <div style={{ marginTop: 4 }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {t('fimKnownHosts.reason', '原因')}: {h.reason}
                            </Text>
                          </div>
                        )}
                        {h.old_fingerprint && h.new_fingerprint && (
                          <div style={{ marginTop: 4, padding: 8, background: '#fff2f0', borderRadius: 4 }}>
                            <Text type="danger" style={{ fontSize: 12, display: 'block' }}>
                              {t('fimKnownHosts.oldFingerprint', '原指纹')}: {h.old_fingerprint}
                            </Text>
                            <Text type="success" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
                              {t('fimKnownHosts.newFingerprint', '新指纹')}: {h.new_fingerprint}
                            </Text>
                          </div>
                        )}
                      </div>
                    </Timeline.Item>
                    )
                  })}
                </Timeline>
              ) : (
                <Empty description={t('fimKnownHosts.noChangeLogs', '暂无变更记录')} />
              )}
            </TabPane>
          </Tabs>
        )}
      </Drawer>
    </div>
  );
}