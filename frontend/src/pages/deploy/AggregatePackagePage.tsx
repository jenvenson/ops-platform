// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

// @ts-nocheck
import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Button, Card, message, Space, Alert, Row, Col, Select } from 'antd';
import { aggregatePackageAPI, ConsulConfig } from '../../api/aggregate-package';
import { cmdbAPI, Project } from '../../api/cmdb';

export default function AggregatePackagePage() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(false);
  const [tagOptions, setTagOptions] = useState<string[]>([]); // 可用的Tag选项
  const [loadingTags, setLoadingTags] = useState(false); // 加载Tags的loading状态
  const [consulConfigs, setConsulConfigs] = useState<ConsulConfig[]>([]); // Consul配置列表
  const [selectedConsulConfig, setSelectedConsulConfig] = useState<number | undefined>(); // 选中的Consul配置

  // 加载项目列表
  useEffect(() => {
    loadProjects();
    loadConsulConfigs(); // 加载Consul配置列表
    loadTagOptions(); // 加载Tag选项
  }, []);

  // 加载Consul配置列表
  const loadConsulConfigs = async () => {
    try {
      const response = await aggregatePackageAPI.getConsulConfigs();
      if (response.success) {
        setConsulConfigs(response.data);
        // 默认选中第一个或标记为默认的配置
        if (response.data.length > 0) {
          const defaultConfig = response.data.find((c: ConsulConfig) => c.is_default);
          if (defaultConfig) {
            setSelectedConsulConfig(defaultConfig.id);
          } else {
            setSelectedConsulConfig(response.data[0].id);
          }
        }
      }
    } catch (error) {
      console.error('加载Consul配置列表失败:', error);
    }
  };

  const loadProjects = async () => {
    setLoadingProjects(true);
    try {
      const response = await cmdbAPI.getProjects({ limit: 100 }); // 获取所有项目
      setProjects(response.data); // 直接使用 response.data，因为它已经是 Project[] 数组
    } catch (error) {
      console.error('加载项目列表失败:', error);
      message.error('加载项目列表失败');
    } finally {
      setLoadingProjects(false);
    }
  };

  // 加载Tag选项列表，从Consul获取
  const loadTagOptions = async () => {
    setLoadingTags(true);
    try {
      // 从Consul的 plugin/aggregation/ 路径获取Tag列表
      const response = await aggregatePackageAPI.queryConsulKv('plugin/aggregation/', {
        consul_config_id: selectedConsulConfig,
      });

      if (response.success) {
        // 从响应数据中提取键名作为Tag选项
        const tags = Object.keys(response.data);
        setTagOptions(tags);
      } else {
        message.error('加载Tag列表失败: ' + response.error);
      }
    } catch (error) {
      console.error('加载Tag列表失败:', error);
      message.error('加载Tag列表失败');
    } finally {
      setLoadingTags(false);
    }
  };

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      // 获取Tag名称
      const tagName = values.tag_name || '';

      if (!tagName) {
        message.error('请输入Tag名称');
        return;
      }

      // 验证Tag是否存在
      if (!tagOptions.includes(tagName)) {
        message.error(`Tag "${tagName}" 不存在，请输入有效的Tag名称`);
        return;
      }

      // 获取项目名称
      const project = projects.find(p => p.id === values.project_id);
      if (!project?.name) {
        message.error('未找到对应项目，请重新选择项目');
        return;
      }
      const projectName = project.name;

      // 使用Tag名称作为应用名称
      const appNames = [tagName]; // 使用Tag名称作为应用名称

      const response = await aggregatePackageAPI.createTask({
        project_name: projectName,
        app_names: appNames,
        task_name: `聚合打包_${tagName}_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}_${Math.floor(Math.random() * 10000)}`,
      });

      if (response.success && response.data?.task_id) {
        message.success('聚合打包任务已提交，正在跳转到聚合历史页面...');
        setTimeout(() => {
          navigate('/deploy/aggregated-history');
        }, 1000);
      } else {
        message.error(response.error || '提交失败');
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || '提交失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="安装包聚合打包">
        <Alert
          message="安装包聚合打包说明"
          description={
            <div>
              <p>使用前请按以下步骤操作：</p>
              <ol>
                <li>先提交静态文件修改和聚合打包的代码仓库代码</li>
                <li>然后在打包机器拉取静态文件的代码</li>
                <li>最后在此界面操作开始聚合打包</li>
              </ol>
              <p style={{ marginTop: 12 }}>
                <strong>说明：</strong>Consul地址从配置管理中的 Consul配置 获取
              </p>
            </div>
          }
          type="info"
          showIcon
          style={{ marginBottom: 24 }}
        />

        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="consul_config_id"
                label="Consul配置"
                rules={[{ required: true, message: '请选择Consul配置' }]}
                initialValue={selectedConsulConfig}
              >
                <Select
                  placeholder="请选择Consul配置"
                  showSearch
                  optionFilterProp="children"
                  value={selectedConsulConfig}
                  onChange={(value) => setSelectedConsulConfig(value)}
                  filterOption={(input, option) =>
                    (typeof option?.children === 'string' ? option.children : '').toLowerCase().includes(input.toLowerCase())
                  }
                >
                  {consulConfigs.map((config) => (
                    <Select.Option key={config.id} value={config.id}>
                      {config.name} ({config.address}){config.is_default ? ' [默认]' : ''}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="project_id"
                label="项目名称"
                rules={[{ required: true, message: '请选择项目名称' }]}
              >
                <Select
                  placeholder="请选择项目"
                  loading={loadingProjects}
                  showSearch
                  optionFilterProp="children"
                  filterOption={(input, option) =>
                    (typeof option?.children === 'string' ? option.children : '').toLowerCase().includes(input.toLowerCase())
                  }
                >
                  {projects.map(project => (
                    <Select.Option key={project.id} value={project.id}>
                      {project.name}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="tag_name"
            label="Tag名称"
            rules={[{ required: true, message: '请输入Tag名称' }]}
            extra="从Consul获取的Tag名称，如V2.5.1"
          >
            <Select
              placeholder="请输入或选择Tag名称"
              loading={loadingTags}
              showSearch
              optionFilterProp="children"
              disabled={loadingTags}
              allowClear
            >
              {tagOptions.map((tag, index) => (
                <Select.Option key={index} value={tag}>
                  {tag}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <div style={{ marginBottom: 16 }}>
            <Button
              type="default"
              onClick={() => loadTagOptions()}
              loading={loadingTags}
              style={{ marginRight: 8 }}
            >
              刷新Tag列表
            </Button>
          </div>

          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
              >
                开始聚合打包
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}