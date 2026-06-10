// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect } from 'react';
import { Card, Table, Tag, Space, Button, Select, DatePicker, message, Modal, Input, List, Typography, Progress, Popconfirm } from 'antd';
import { SearchOutlined, DownloadOutlined, FileZipOutlined, LinkOutlined, SyncOutlined, FileTextOutlined, DeleteOutlined } from '@ant-design/icons';
import { aggregatedHistoryAPI, AggregatedHistory, AggregatedHistoryFile } from '../../api/aggregated-history';
import AssistantQuickActions from '../../components/AssistantQuickActions';
import useAssistantPageContext from '../../components/useAssistantPageContext';

const { RangePicker } = DatePicker;

export default function AggregatedHistoryPage() {
  const [histories, setHistories] = useState<AggregatedHistory[]>([]);
  const [loading, setLoading] = useState(false);
  const [refreshingId, setRefreshingId] = useState<number | null>(null);
  const [filters, setFilters] = useState({
    projectName: undefined as string | undefined,
    environment: undefined as string | undefined,
    operator: undefined as string | undefined,
    status: undefined as string | undefined,
    startTime: undefined as string | undefined,
    endTime: undefined as string | undefined,
  });
  const [fileModalVisible, setFileModalVisible] = useState(false);
  const [currentFiles, setCurrentFiles] = useState<AggregatedHistoryFile[]>([]);
  const [currentTimestamp, setCurrentTimestamp] = useState('');
  const [loadingFiles, setLoadingFiles] = useState(false);
  const [fileEmptyMessage, setFileEmptyMessage] = useState('未找到聚合文件');
  const [logModalVisible, setLogModalVisible] = useState(false);
  const [consoleLog, setConsoleLog] = useState('');
  const [loadingLog, setLoadingLog] = useState(false);

  useAssistantPageContext({
    filters: {
      projectName: filters.projectName,
      environment: filters.environment,
      operator: filters.operator,
      status: filters.status,
      startTime: filters.startTime,
      endTime: filters.endTime,
    },
  });

  // 重置筛选条件
  const resetFilters = () => {
    setFilters({
      projectName: undefined,
      environment: undefined,
      operator: undefined,
      status: undefined,
      startTime: undefined,
      endTime: undefined,
    });
  };

  // 获取聚合历史数据
  const fetchHistories = async () => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { limit: 100 };
      if (filters.projectName) params.project_name = filters.projectName;
      if (filters.environment) params.environment = filters.environment;
      if (filters.operator) params.operator = filters.operator;
      if (filters.status) params.status = filters.status;
      if (filters.startTime) params.start_time = filters.startTime;
      if (filters.endTime) params.end_time = filters.endTime;

      const resp = await aggregatedHistoryAPI.getHistories(params);
      setHistories(resp.data);
    } catch (error) {
      console.error('获取聚合历史失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHistories();
  }, [filters.projectName, filters.environment, filters.operator, filters.status, filters.startTime, filters.endTime]);

  // 手动刷新单个聚合历史记录的状态
  const refreshRecordStatus = async (id: number) => {
    setRefreshingId(id);
    try {
      await aggregatedHistoryAPI.getStatus(id);
      await fetchHistories();
    } catch (error) {
      console.error('刷新状态失败:', error);
    } finally {
      setRefreshingId(null);
    }
  };

  // 复制下载链接
  const copyDownloadLink = (url: string) => {
    if (!url) {
      message.warning('下载链接不可用');
      return;
    }
    navigator.clipboard.writeText(url);
    message.success('下载链接已复制到剪贴板');
  };

  // 查看归档文件列表
  const viewArchiveFiles = async (record: AggregatedHistory) => {
    setFileModalVisible(true);
    setCurrentFiles([]);
    setLoadingFiles(true);
    setFileEmptyMessage('未找到聚合文件');

    try {
      const resp = await aggregatedHistoryAPI.getFiles(record.id);
      setCurrentFiles(resp.files || []);
      setCurrentTimestamp(resp.timestamp || record.end_time || record.start_time || record.created_at || '');
      setFileEmptyMessage(resp.message || '未找到聚合文件');
    } catch (error) {
      console.error('获取文件列表失败:', error);
      message.error('获取文件列表失败');
      setCurrentTimestamp(record.end_time || record.start_time || record.created_at || '');
    } finally {
      setLoadingFiles(false);
    }
  };

  // 自动轮询刷新状态（当有运行中或排队中的聚合任务时）
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const resp = await aggregatedHistoryAPI.getHistories({ limit: 100 });
        const currentHistories = resp.data || [];

        const activeHistories = currentHistories.filter(
          (h: AggregatedHistory) => h && h.status && (h.status === 'running' || h.status === 'queued' || h.status === 'pending' || h.status === 'archiving') && h.id != null
        );

        if (activeHistories.length === 0) {
          setHistories(currentHistories);
          return;
        }

        // 并行刷新所有活跃聚合历史的状态，但要确保id有效
        const validActiveHistories = activeHistories.filter(h => h.id != null && typeof h.id !== 'undefined');
        if (validActiveHistories.length > 0) {
          await Promise.all(
            validActiveHistories.map((h: AggregatedHistory) => aggregatedHistoryAPI.getStatus(h.id).catch(err => {
              console.error(`获取ID为${h.id}的聚合历史状态时出错:`, err);
            }))
          );
        }

        const updatedResp = await aggregatedHistoryAPI.getHistories({ limit: 100 });
        setHistories(updatedResp.data || []);
      } catch (error) {
        console.error('自动刷新失败:', error);
      }
    }, 10000); // 每10秒刷新一次

    return () => clearInterval(interval);
  }, []);

  // 状态映射：归档中、完成、归档失败
  const statusMap: Record<string, { color: string; text: string }> = {
    pending: { color: 'default', text: '待执行' },
    queued: { color: 'processing', text: '排队中' },
    running: { color: 'processing', text: '归档中' },
    archiving: { color: 'processing', text: '归档中' },
    success: { color: 'success', text: '完成' },
    completed: { color: 'success', text: '完成' },
    failed: { color: 'error', text: '归档失败' },
  };

  // 查看控制台日志
  const viewConsoleLog = async (record: AggregatedHistory) => {
    setLogModalVisible(true);
    setLoadingLog(true);
    setConsoleLog('');

    try {
      const resp = await aggregatedHistoryAPI.getConsoleLog(record.id);
      setConsoleLog(resp.console_log || '暂无日志');
    } catch (error) {
      console.error('获取控制台日志失败:', error);
      message.error('获取控制台日志失败');
      setConsoleLog('获取日志失败');
    } finally {
      setLoadingLog(false);
    }
  };

  // 删除聚合历史记录
  const deleteHistory = async (id: number) => {
    try {
      await aggregatedHistoryAPI.deleteHistory(id);
      message.success('删除成功');
      fetchHistories();
    } catch (error) {
      console.error('删除失败:', error);
      message.error('删除失败');
    }
  };

  // 操作列的菜单
  const getOperationMenu = (record: AggregatedHistory) => {
    const isActive = ['running', 'queued', 'pending', 'archiving'].includes(record.status);
    return (
      <Space size="small">
        {isActive && (
          <Button
            type="link"
            size="small"
            onClick={() => refreshRecordStatus(record.id)}
            loading={refreshingId === record.id}
            icon={<SyncOutlined />}
          >
            刷新
          </Button>
        )}
        <Button
          type="link"
          size="small"
          icon={<FileTextOutlined />}
          onClick={() => {
            // 如果存在Jenkins控制台URL，则直接跳转到Jenkins页面
            if (record.jenkins_console_url) {
              window.open(record.jenkins_console_url, '_blank');
            } else {
              // 否则查看本地缓存的日志
              viewConsoleLog(record);
            }
          }}
        >
          查看日志
        </Button>
        <Popconfirm
          title="确定要删除此记录吗？"
          onConfirm={() => deleteHistory(record.id)}
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
      </Space>
    );
  };

  const columns = [
    {
      title: '项目名称',
      dataIndex: 'project_name',
      key: 'project_name',
      width: 200,
    },
    {
      title: 'Tag名称',
      dataIndex: 'environment',
      key: 'environment',
      width: 150,
      render: (name: string) => (
        <Tag color="blue">
          {name || '-'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const config = statusMap[status] || { color: 'default', text: status || 'unknown' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      width: 150,
      render: (progress: number, record: AggregatedHistory) => {
        const isActive = ['archiving', 'running', 'queued', 'pending'].includes(record.status);
        const percent = progress || 0;
        let status: 'success' | 'exception' | 'normal' | 'active' = 'normal';
        
        if (record.status === 'completed') {
          status = 'success';
        } else if (record.status === 'failed') {
          status = 'exception';
        } else if (isActive) {
          status = 'active';
        }
        
        return (
          <Progress 
            percent={percent} 
            size="small" 
            status={status}
            format={(p) => `${p}%`}
          />
        );
      },
    },
    {
      title: '归档开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 160,
      render: (time: string) => formatDateTime(time),
    },
    {
      title: '归档结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      width: 160,
      render: (time: string) => formatDateTime(time),
    },
    {
      title: '下载地址',
      dataIndex: 'download_url',
      key: 'download_url',
      width: 300,
      render: (url: string, record: AggregatedHistory) => {
        if (!url) {
          return '-';
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
                  icon={<FileZipOutlined />}
                  onClick={() => viewArchiveFiles(record)}
                >
                  查看文件
                </Button>
                <Button
                  size="small"
                  icon={<LinkOutlined />}
                  onClick={() => copyDownloadLink(url)}
                >
                  复制链接
                </Button>
              </Space>
            )}
          </Space>
        );
      },
    },
    {
      title: '操作人',
      dataIndex: 'operator_name',
      key: 'operator_name',
      width: 100,
      render: (name: string) => name || '系统',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: unknown, record: AggregatedHistory) => getOperationMenu(record),
    },
  ];

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        {/* 第一行：搜索条件 */}
        <div style={{ marginBottom: 8 }}>
          <Space wrap>
            <Input
              placeholder="项目名称"
              value={filters.projectName}
              onChange={e => setFilters({ ...filters, projectName: e.target.value })}
              allowClear
              style={{ width: 180 }}
              onPressEnter={fetchHistories}
            />
            <Input
              placeholder="Tag名称"
              value={filters.environment}
              onChange={e => setFilters({ ...filters, environment: e.target.value })}
              allowClear
              style={{ width: 150 }}
              onPressEnter={fetchHistories}
            />
            <Select
              placeholder="状态"
              value={filters.status}
              onChange={val => setFilters({ ...filters, status: val })}
              allowClear
              style={{ width: 120 }}
              options={[
                { label: '待执行', value: 'pending' },
                { label: '排队中', value: 'queued' },
                { label: '归档中', value: 'archiving' },
                { label: '完成', value: 'completed' },
                { label: '归档失败', value: 'failed' },
              ]}
            />
            <Input
              placeholder="操作人"
              value={filters.operator}
              onChange={e => setFilters({ ...filters, operator: e.target.value })}
              allowClear
              style={{ width: 120 }}
              onPressEnter={fetchHistories}
            />
            <RangePicker
              showTime
              onChange={(dates) => {
                if (dates && dates[0] && dates[1]) {
                  const start = dates[0].format('YYYY-MM-DD HH:mm:ss');
                  const end = dates[1].format('YYYY-MM-DD HH:mm:ss');
                  setFilters({
                    ...filters,
                    startTime: start,
                    endTime: end,
                  });
                } else {
                  setFilters({
                    ...filters,
                    startTime: undefined,
                    endTime: undefined,
                  });
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
            <Button type="primary" icon={<SearchOutlined />} onClick={fetchHistories}>
              搜索
            </Button>
            <Button icon={<SyncOutlined />} onClick={resetFilters}>
              重置
            </Button>
          </Space>
        </div>
      </Card>

      <AssistantQuickActions
        description="复用右侧运维小助手，基于当前聚合历史页面上下文发起查询"
        actions={[
          { label: '最近有哪些聚合失败', query: '最近有哪些聚合失败' },
          { label: '最近聚合是否正常完成', query: '最近聚合是否正常完成' },
          { label: '如何从下载地址下载聚合包', query: '如何从下载地址下载聚合包' },
        ]}
      />

      <Card>
        <Table
          columns={columns}
          dataSource={histories}
          rowKey="id"
          loading={loading}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total: number) => `共 ${total} 条`,
            showQuickJumper: true
          }}
          scroll={{ x: 1400 }}
          locale={{ emptyText: '暂无聚合历史记录' }}
        />
      </Card>

      {/* 文件列表弹窗 */}
      <Modal
        title={currentTimestamp ? `聚合文件列表 - ${formatTimestampToLocalTime(currentTimestamp)}` : '聚合文件列表'}
        open={fileModalVisible}
        onCancel={() => setFileModalVisible(false)}
        footer={null}
        width={600}
      >
        {loadingFiles ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : currentFiles.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            {fileEmptyMessage}
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

      {/* 控制台日志弹窗 */}
      <Modal
        title="Jenkins 构建日志"
        open={logModalVisible}
        onCancel={() => setLogModalVisible(false)}
        footer={null}
        width={900}
      >
        {loadingLog ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : (
          <pre style={{
            backgroundColor: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 4,
            maxHeight: 500,
            overflow: 'auto',
            fontSize: 12,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all'
          }}>
            {consoleLog}
          </pre>
        )}
      </Modal>
    </div>
  );
}

// 格式化日期时间，格式：2026/3/10 13:33:08
function formatDateTime(time: string | undefined | null): string {
  if (!time) return '-';
  try {
    const date = new Date(time);
    if (isNaN(date.getTime())) return '-';
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    const hour = date.getHours().toString().padStart(2, '0');
    const minute = date.getMinutes().toString().padStart(2, '0');
    const second = date.getSeconds().toString().padStart(2, '0');
    return `${year}/${month}/${day} ${hour}:${minute}:${second}`;
  } catch {
    return '-';
  }
}

// 格式化文件大小
function formatFileSize(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null || bytes <= 0) return '未知';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 格式化时间戳为北京时间（UTC+8）- 支持多种格式
// 格式化时间戳为北京时间（UTC+8）- 支持多种格式
function formatTimestampToLocalTime(timestamp: string): string {
  if (!timestamp) return '-';

  try {
    // 如果是 14 位数字格式 (20260129104425)
    if (timestamp.length === 14 && /^\d{14}$/.test(timestamp)) {
      // 按 UTC 时间解析，然后加 8 小时转为北京时间
      const year = parseInt(timestamp.slice(0, 4));
      const month = parseInt(timestamp.slice(4, 6)) - 1; // month is 0-indexed
      const day = parseInt(timestamp.slice(6, 8));
      const hour = parseInt(timestamp.slice(8, 10));
      const minute = parseInt(timestamp.slice(10, 12));
      const second = parseInt(timestamp.slice(12, 14));

      // 验证日期有效性
      if (isNaN(year) || isNaN(month) || isNaN(day) || isNaN(hour) || isNaN(minute) || isNaN(second)) {
        return timestamp;
      }

      const date = new Date(Date.UTC(year, month, day, hour, minute, second));

      // 检查日期是否有效
      if (isNaN(date.getTime())) {
        return timestamp;
      }

      // 加8小时转换为北京时间
      date.setTime(date.getTime() + 8 * 60 * 60 * 1000);
      return date.toLocaleString('zh-CN');
    }

    // 如果是 "YYYY-MM-DD HH:mm:ss" 格式（后端返回的 UTC 时间），加 8 小时转为北京时间
    if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(timestamp)) {
      // 后端返回的时间是 UTC 时间，需要加 8 小时转为北京时间
      const date = new Date(timestamp.replace(/-/g, '/'));
      if (isNaN(date.getTime())) {
        return timestamp;
      }
      date.setTime(date.getTime() + 8 * 60 * 60 * 1000);
      return date.toLocaleString('zh-CN');
    }

    // 如果是 ISO 格式（带时区），转换为北京时间
    if (timestamp.includes('T') || timestamp.includes('-')) {
      const date = new Date(timestamp);
      if (!isNaN(date.getTime())) {
        // 转换为北京时间
        date.setTime(date.getTime() + 8 * 60 * 60 * 1000);
        return date.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
      }
    }

    return timestamp;
  } catch (error) {
    console.error('Error formatting timestamp:', error);
    return timestamp;
  }
}