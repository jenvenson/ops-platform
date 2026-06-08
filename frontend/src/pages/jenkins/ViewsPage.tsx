import { useState } from 'react';
import {
  Card, Form, Input, Button, Space, message, Typography, Row, Col, Alert, Table, Modal, Tag, Divider
} from 'antd';
import {
  CopyOutlined, PlayCircleOutlined, QuestionCircleOutlined, DeleteOutlined, SearchOutlined, ExclamationCircleOutlined, KeyOutlined
} from '@ant-design/icons';
import { jenkinsAPI, JenkinsViewJob, ViewCopyResult } from '../../api/jenkins';

const { Text, Paragraph } = Typography;

export default function ViewsPage() {
  const [copyForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any | null>(null);

  // 删除功能状态
  const [deleteViewName, setDeleteViewName] = useState('');
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [viewJobs, setViewJobs] = useState<JenkinsViewJob[]>([]);
  const [selectedJobKeys, setSelectedJobKeys] = useState<string[]>([]);
  const [viewQueried, setViewQueried] = useState(false);
  const [deleteResult, setDeleteResult] = useState<any | null>(null);

  // 凭据功能状态
  const [credForm] = Form.useForm();
  const [credLoading, setCredLoading] = useState(false);
  const [credResult, setCredResult] = useState<{ success: boolean; message: string } | null>(null);

  const [copyProgress, setCopyProgress] = useState<{ progress: number; total: number } | null>(null);

  // 创建 Jenkins SSH 凭据
  const handleCreateCredential = async (values: any) => {
    try {
      setCredLoading(true);
      setCredResult(null);
      const resp = await jenkinsAPI.createCredential({
        id: values.id.trim(),
        username: values.username.trim(),
        private_key: values.private_key,
        description: values.description?.trim() || '',
      });
      setCredResult({ success: true, message: resp.message });
      message.success(resp.message);
      credForm.resetFields();
      credForm.setFieldsValue({ username: 'root' });
    } catch (error: any) {
      const errMsg = error?.response?.data?.error || error.message || '未知错误';
      setCredResult({ success: false, message: errMsg });
      message.error(`创建凭据失败: ${errMsg}`);
    } finally {
      setCredLoading(false);
    }
  };

  // 轮询任务状态
  const pollTaskStatus = async (taskId: string): Promise<ViewCopyResult> => {
    return new Promise((resolve, reject) => {
      const poll = async () => {
        try {
          const task = await jenkinsAPI.getTaskStatus(taskId);
          setCopyProgress({ progress: task.progress, total: task.total });

          if (task.status === 'completed' && task.result) {
            resolve(task.result);
          } else if (task.status === 'failed') {
            reject(new Error(task.result?.message || '任务执行失败'));
          } else {
            setTimeout(poll, 2000);
          }
        } catch (err: any) {
          reject(new Error(err?.response?.data?.error || err.message || '查询任务状态失败'));
        }
      };
      poll();
    });
  };

  // 执行视图复制（异步模式）
  const handleCopyView = async (values: any) => {
    if (!values.source_view || !values.target_view) {
      message.warning('请先输入源视图和目标视图名称');
      return;
    }

    if (!values.jenkins_url) {
      message.warning('请输入Jenkins地址');
      return;
    }

    try {
      setLoading(true);
      setResult(null);
      setCopyProgress(null);

      const jobNameReplacements = [];
      if (values.source_view && values.target_view) {
        jobNameReplacements.push({
          old_pattern: values.source_view,
          new_pattern: values.target_view
        });
      }
      if (values.job_name_old_pattern && values.job_name_new_pattern) {
        jobNameReplacements.push({
          old_pattern: values.job_name_old_pattern,
          new_pattern: values.job_name_new_pattern
        });
      }

      const tagReplacements = [];
      if (values.pipeline_tag_old_pattern && values.pipeline_tag_new_pattern) {
        tagReplacements.push({
          old_pattern: values.pipeline_tag_old_pattern,
          new_pattern: values.pipeline_tag_new_pattern
        });
      }

      // 提交异步任务
      const asyncResp = await jenkinsAPI.copyView({
        source_view: values.source_view,
        target_view: values.target_view,
        jenkins_url: values.jenkins_url,
        tag_replacements: tagReplacements,
        job_name_replacements: jobNameReplacements
      });

      message.info('复制任务已提交，正在执行中...');

      // 轮询任务状态直到完成
      const taskResult = await pollTaskStatus(asyncResp.task_id);

      setResult(taskResult);
      setCopyProgress(null);
      if (taskResult.success) {
        message.success('视图复制成功！');
      } else {
        message.warning('视图复制部分失败');
      }
    } catch (error: any) {
      setCopyProgress(null);
      const errMsg = error?.response?.data?.error || error.message || '未知错误';
      message.error(`视图复制失败: ${errMsg}`);
    } finally {
      setLoading(false);
    }
  };

  // 查询视图下的 Jobs
  const handleQueryView = async () => {
    if (!deleteViewName.trim()) {
      message.warning('请输入视图名称');
      return;
    }
    try {
      setDeleteLoading(true);
      setDeleteResult(null);
      setSelectedJobKeys([]);
      const resp = await jenkinsAPI.getViewJobs(deleteViewName.trim());
      setViewJobs(resp.jobs || []);
      setViewQueried(true);
      if (!resp.jobs || resp.jobs.length === 0) {
        message.info('该视图下没有 Job');
      }
    } catch (error: any) {
      const errMsg = error?.response?.data?.error || error.message || '未知错误';
      if (error?.response?.status === 404 || errMsg.includes('不存在')) {
        message.warning(`视图 "${deleteViewName.trim()}" 不存在`);
      } else {
        message.error(`查询视图失败: ${errMsg}`);
      }
      setViewJobs([]);
      setViewQueried(false);
    } finally {
      setDeleteLoading(false);
    }
  };

  const [deleteProgress, setDeleteProgress] = useState<{ progress: number; total: number } | null>(null);

  // 轮询删除任务状态
  const pollDeleteTask = async (taskId: string): Promise<any> => {
    return new Promise((resolve, reject) => {
      const poll = async () => {
        try {
          const task = await jenkinsAPI.getTaskStatus(taskId);
          setDeleteProgress({ progress: task.progress, total: task.total });
          if (task.status === 'completed' && task.result) {
            resolve(task.result);
          } else if (task.status === 'failed') {
            reject(new Error(task.result?.message || '删除任务执行失败'));
          } else {
            setTimeout(poll, 1500);
          }
        } catch (err: any) {
          reject(new Error(err?.response?.data?.error || err.message || '查询任务状态失败'));
        }
      };
      poll();
    });
  };

  // 删除选中的 Jobs
  const handleDeleteSelectedJobs = () => {
    if (selectedJobKeys.length === 0) {
      message.warning('请先选择要删除的 Job');
      return;
    }
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除选中的 ${selectedJobKeys.length} 个 Job 吗？此操作不可恢复！`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          setDeleteLoading(true);
          setDeleteProgress(null);
          const asyncResp = await jenkinsAPI.batchDeleteJobs({
            view_name: deleteViewName.trim(),
            job_names: selectedJobKeys,
            delete_view: false,
          });
          message.info('删除任务已提交，正在执行中...');
          const result = await pollDeleteTask(asyncResp.task_id);
          setDeleteResult({ message: result.message, deleted_jobs: result.copied_jobs, failed_jobs: result.failed_jobs });
          setDeleteProgress(null);
          message.success(result.message);
          setSelectedJobKeys([]);
          handleQueryView();
        } catch (error: any) {
          setDeleteProgress(null);
          const errMsg = error?.response?.data?.error || error.message || '未知错误';
          message.error(`删除失败: ${errMsg}`);
        } finally {
          setDeleteLoading(false);
        }
      },
    });
  };

  // 删除整个视图（含所有 Jobs）
  const handleDeleteViewWithJobs = () => {
    const jobNames = viewJobs.map(j => j.name);
    Modal.confirm({
      title: '确认删除整个视图',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>确定要删除视图 <Text strong>"{deleteViewName}"</Text> 及其下的 <Text strong type="danger">{jobNames.length}</Text> 个 Job 吗？</p>
          <p style={{ color: '#ff4d4f' }}>此操作不可恢复！所有 Job 和视图都将被永久删除。</p>
        </div>
      ),
      okText: '确认删除全部',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          setDeleteLoading(true);
          setDeleteProgress(null);
          const asyncResp = await jenkinsAPI.batchDeleteJobs({
            view_name: deleteViewName.trim(),
            job_names: jobNames,
            delete_view: true,
          });
          message.info('删除任务已提交，正在执行中...');
          const result = await pollDeleteTask(asyncResp.task_id);
          setDeleteResult({ message: result.message, deleted_jobs: result.copied_jobs, failed_jobs: result.failed_jobs });
          setDeleteProgress(null);
          message.success(result.message);
          setViewJobs([]);
          setSelectedJobKeys([]);
          setViewQueried(false);
        } catch (error: any) {
          setDeleteProgress(null);
          const errMsg = error?.response?.data?.error || error.message || '未知错误';
          message.error(`删除失败: ${errMsg}`);
        } finally {
          setDeleteLoading(false);
        }
      },
    });
  };

  // 删除单个 Job
  const handleDeleteSingleJob = (jobName: string) => {
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除 Job "${jobName}" 吗？此操作不可恢复！`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await jenkinsAPI.deleteJob(jobName);
          message.success(`Job "${jobName}" 已删除`);
          handleQueryView();
        } catch (error: any) {
          message.error(`删除失败: ${error.message || '未知错误'}`);
        }
      },
    });
  };

  // 仅删除视图（不删除 Jobs）
  const handleDeleteViewOnly = () => {
    Modal.confirm({
      title: '确认删除视图',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>确定要删除视图 <Text strong>"{deleteViewName}"</Text> 吗？</p>
          <p>视图下的 Job 不会被删除，仅移除视图本身。</p>
        </div>
      ),
      okText: '确认删除视图',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await jenkinsAPI.deleteView(deleteViewName.trim());
          message.success(`视图 "${deleteViewName}" 已删除`);
          setViewJobs([]);
          setViewQueried(false);
          setSelectedJobKeys([]);
        } catch (error: any) {
          message.error(`删除视图失败: ${error.message || '未知错误'}`);
        }
      },
    });
  };

  const jobColumns = [
    {
      title: 'Job 名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'color',
      key: 'color',
      width: 100,
      render: (color: string) => {
        const colorMap: Record<string, { color: string; text: string }> = {
          blue: { color: 'blue', text: '成功' },
          red: { color: 'red', text: '失败' },
          yellow: { color: 'orange', text: '不稳定' },
          grey: { color: 'default', text: '未构建' },
          disabled: { color: 'default', text: '已禁用' },
          aborted: { color: 'default', text: '已中止' },
          notbuilt: { color: 'default', text: '未构建' },
        };
        const baseColor = color?.replace(/_anime$/, '') || 'grey';
        const info = colorMap[baseColor] || { color: 'default', text: color || '未知' };
        return <Tag color={info.color}>{info.text}{color?.endsWith('_anime') ? '(构建中)' : ''}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: any, record: JenkinsViewJob) => (
        <Button
          type="link"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => handleDeleteSingleJob(record.name)}
        >
          删除
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* 使用说明 */}
      <Card
        title={<Space><QuestionCircleOutlined /><span>使用说明</span></Space>}
        style={{ marginBottom: 24 }}
        size="small"
      >
        <Paragraph>
          <Text strong>Jenkins视图管理功能:</Text>
        </Paragraph>
        <Paragraph>
          <Text strong>视图复制：</Text>将源视图中所有 Job 复制到新视图，支持 Job 名称和 Tag 值自动替换。<br/>
          <Text strong>视图删除：</Text>输入视图名称查询后，可选择单个 Job 删除、批量选择删除、或删除整个视图（含所有 Job）。
        </Paragraph>
      </Card>

      {/* 视图复制 */}
      <Card
        title={<Space><CopyOutlined style={{ color: '#1890ff' }} /><span>Jenkins视图批量复制</span></Space>}
        size="small"
        style={{ marginBottom: 16 }}
      >
        <Form
          form={copyForm}
          layout="vertical"
          onFinish={handleCopyView}
          initialValues={{
            source_view: '', target_view: '', jenkins_url: 'http://js.zbnsec.com',
            pipeline_tag_old_pattern: '', pipeline_tag_new_pattern: '',
            job_name_old_pattern: '', job_name_new_pattern: ''
          }}
        >
          <Row gutter={24}>
            <Col span={8}>
              <Form.Item name="jenkins_url" label="Jenkins地址" rules={[{ required: true, message: '请输入Jenkins地址' }]}>
                <Input placeholder="如: http://js.zbnsec.com" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="source_view" label="源视图名称" rules={[{ required: true, message: '请输入源视图名称' }]}>
                <Input placeholder="如: demo" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="target_view" label="目标视图名称" rules={[{ required: true, message: '请输入目标视图名称' }]}>
                <Input placeholder="如: 147" />
              </Form.Item>
            </Col>
          </Row>

          <Card title="Job名称替换" size="small" style={{ marginTop: 16 }}>
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item name="job_name_old_pattern" label="原Job名称模式">
                  <Input placeholder="如: demo, old-project, etc." />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="job_name_new_pattern" label="新Job名称模式">
                  <Input placeholder="如: test, new-project, etc." />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Card title="流水线Tag值替换" size="small" style={{ marginTop: 16 }}>
            <Row gutter={24}>
              <Col span={12}>
                <Form.Item name="pipeline_tag_old_pattern" label="原Tag值模式">
                  <Input placeholder="如: demo, v1.0.0, dev, etc." />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="pipeline_tag_new_pattern" label="新Tag值模式">
                  <Input placeholder="如: test, v2.0.0, prod, etc." />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Form.Item style={{ marginTop: 24 }}>
            <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading} size="large" block>
              {loading && copyProgress ? `执行中 (${copyProgress.progress}/${copyProgress.total})...` : '执行视图复制 (自动复制所有Jobs)'}
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* 复制结果 */}
      {result && (
        <Card title="复制结果" size="small" style={{ marginTop: 16, marginBottom: 16 }}>
          <Alert
            message={result.success ? "复制成功" : "复制部分失败"}
            description={result.message}
            type={result.success ? "success" : "warning"}
            showIcon
            style={{ marginBottom: 16 }}
          />
          <Row gutter={16}>
            <Col span={6}>
              <Alert type="success" message="成功复制" description={<Text strong style={{ fontSize: 16 }}>{result.copied_jobs?.length || 0}</Text>} showIcon />
            </Col>
            <Col span={6}>
              <Alert type={result.failed_jobs?.length > 0 ? "error" : "info"} message="复制失败" description={<Text strong style={{ fontSize: 16 }}>{result.failed_jobs?.length || 0}</Text>} showIcon />
            </Col>
            <Col span={6}>
              <Alert type="info" message="跳过" description={<Text strong style={{ fontSize: 16 }}>{result.skipped_jobs?.length || 0}</Text>} showIcon />
            </Col>
            <Col span={6}>
              <Alert type={result.approved_count > 0 ? "success" : "info"} message="脚本自动审批" description={<Text strong style={{ fontSize: 16 }}>{result.approved_count || 0}</Text>} showIcon />
            </Col>
          </Row>
          {result.approval_note && (
            <Alert
              type={result.approved_count > 0 ? "info" : "warning"}
              message="审批说明"
              description={result.approval_note}
              showIcon
              style={{ marginTop: 16 }}
            />
          )}
          {result.copied_jobs?.length > 0 && (
            <div style={{ marginTop: 16 }}><h4>成功复制的Jobs:</h4><ul>{result.copied_jobs.map((job: string, i: number) => <li key={i}>{job}</li>)}</ul></div>
          )}
          {result.failed_jobs?.length > 0 && (
            <div style={{ marginTop: 16 }}><h4>复制失败的Jobs:</h4><ul>{result.failed_jobs.map((job: string, i: number) => <li key={i}>{job}</li>)}</ul></div>
          )}
        </Card>
      )}

      <Divider />

      {/* 视图删除 */}
      <Card
        title={<Space><DeleteOutlined style={{ color: '#ff4d4f' }} /><span>Jenkins视图删除</span></Space>}
        size="small"
      >
        <Row gutter={16} align="middle" style={{ marginBottom: 16 }}>
          <Col flex="auto">
            <Input
              placeholder="输入要删除的视图名称，如: 6f_dev-147"
              value={deleteViewName}
              onChange={(e) => setDeleteViewName(e.target.value)}
              onPressEnter={handleQueryView}
              allowClear
            />
          </Col>
          <Col>
            <Button type="primary" icon={<SearchOutlined />} loading={deleteLoading} onClick={handleQueryView}>
              查询视图
            </Button>
          </Col>
        </Row>

        {viewQueried && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Text>视图 <Text strong>"{deleteViewName}"</Text> 共 {viewJobs.length} 个 Job</Text>
                {selectedJobKeys.length > 0 && <Text type="secondary">（已选 {selectedJobKeys.length} 个）</Text>}
              </Space>
              <Space style={{ float: 'right' }}>
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  disabled={selectedJobKeys.length === 0}
                  onClick={handleDeleteSelectedJobs}
                  loading={deleteLoading}
                >
                  {deleteLoading && deleteProgress ? `删除中 (${deleteProgress.progress}/${deleteProgress.total})` : `删除选中 (${selectedJobKeys.length})`}
                </Button>
                <Button
                  onClick={handleDeleteViewOnly}
                  loading={deleteLoading}
                >
                  仅删除视图
                </Button>
                <Button
                  danger
                  type="primary"
                  icon={<DeleteOutlined />}
                  disabled={viewJobs.length === 0}
                  onClick={handleDeleteViewWithJobs}
                  loading={deleteLoading}
                >
                  {deleteLoading && deleteProgress ? `删除中 (${deleteProgress.progress}/${deleteProgress.total})` : `删除视图及全部Job (${viewJobs.length})`}
                </Button>
              </Space>
            </div>

            <Table
              rowKey="name"
              columns={jobColumns}
              dataSource={viewJobs}
              size="small"
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedJobKeys,
                onChange: (keys) => setSelectedJobKeys(keys as string[]),
              }}
              locale={{ emptyText: '该视图下没有 Job' }}
            />
          </>
        )}

        {/* 删除结果 */}
        {deleteResult && (
          <Alert
            style={{ marginTop: 16 }}
            message={deleteResult.message}
            type={deleteResult.failed_jobs?.length > 0 ? 'warning' : 'success'}
            showIcon
            closable
            onClose={() => setDeleteResult(null)}
          />
        )}
      </Card>

      {/* 新增 Jenkins 凭据 */}
      <Card
        title={<Space><KeyOutlined /> 新增 Jenkins 凭据</Space>}
        style={{ marginBottom: 16 }}
      >
        <Form
          form={credForm}
          layout="vertical"
          onFinish={handleCreateCredential}
          initialValues={{ username: 'root' }}
        >
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="凭据 ID（主机IP）" name="id" rules={[{ required: true, message: '请输入凭据ID' }]}>
                <Input placeholder="例如: 10.99.99.100" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="Username" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                <Input placeholder="默认: root" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="描述（可选）" name="description">
                <Input placeholder="凭据描述" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item label="Private Key" name="private_key" rules={[{ required: true, message: '请输入私钥内容' }]}>
            <Input.TextArea
              rows={8}
              placeholder={"-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"}
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" icon={<KeyOutlined />} loading={credLoading}>
              创建凭据
            </Button>
          </Form.Item>
        </Form>
        {credResult && (
          <Alert
            message={credResult.message}
            type={credResult.success ? 'success' : 'error'}
            showIcon
            closable
            onClose={() => setCredResult(null)}
          />
        )}
      </Card>
    </div>
  );
}
