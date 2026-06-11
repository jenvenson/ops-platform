// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useMemo } from 'react'
import { Table, Button, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { cmdbAPI, Environment } from '../../api/cmdb'
import SearchBar, { SearchField } from '../../components/SearchBar'
import { canEdit } from '../../utils/menuAccess'

export default function EnvironmentsPage() {
  const { t } = useTranslation('cmdb')
  const { t: tc } = useTranslation('common')

  const [environments, setEnvironments] = useState<Environment[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingEnvironment, setEditingEnvironment] = useState<Environment | null>(null)
  const [searchValues, setSearchValues] = useState<Record<string, string>>({})
  const [form] = Form.useForm()

  // 搜索字段配置
  const searchFields: SearchField[] = [
    { name: 'name', label: t('envName', '环境名称'), type: 'text' },
    { name: 'type', label: t('colType', '类型'), type: 'select', options: [
      { value: 'dev', label: t('envTypeDev', '开发环境') },
      { value: 'test', label: t('envTypeTest', '测试环境') },
      { value: 'prod', label: t('envTypeProd', '生产环境') },
    ]},
  ]

  const fetchEnvironments = async () => {
    setLoading(true)
    try {
      const resp = await cmdbAPI.getEnvironments({ limit: 1000 })
      setEnvironments(resp.data)
    } catch (error) {
      message.error(t('loadEnvsFailed', '加载环境失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchEnvironments()
  }, [])

  // 搜索过滤后的数据
  const filteredEnvironments = useMemo(() => {
    return environments.filter((env) => {
      if (searchValues.name && !env.name?.toLowerCase().includes(searchValues.name.toLowerCase())) {
        return false
      }
      if (searchValues.type && env.type !== searchValues.type) {
        return false
      }
      return true
    })
  }, [environments, searchValues])

  const handleSearch = (values: Record<string, string>) => {
    setSearchValues(values)
  }

  const handleResetSearch = () => {
    setSearchValues({})
  }

  const handleAdd = () => {
    setEditingEnvironment(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: Environment) => {
    setEditingEnvironment(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await cmdbAPI.deleteEnvironment(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchEnvironments()
    } catch (error) {
      message.error(tc('deleteFailed', '删除失败'))
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingEnvironment) {
        await cmdbAPI.updateEnvironment(editingEnvironment.id, values)
        message.success(tc('updateSuccess', '更新成功'))
      } else {
        await cmdbAPI.createEnvironment(values)
        message.success(tc('createSuccess', '创建成功'))
      }
      setModalVisible(false)
      fetchEnvironments()
    } catch (error) {
      message.error(tc('operationFailed', '操作失败'))
    }
  }

  const columns = [
    { title: t('envName', '环境名称'), dataIndex: 'name', key: 'name' },
    {
      title: t('colType', '类型'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => {
        const color = type === 'prod' ? 'red' : type === 'test' ? 'orange' : 'green'
        return <span style={{ color, fontWeight: 'bold' }}>{type.toUpperCase()}</span>
      },
    },
    { title: t('description', '描述'), dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: tc('action', '操作'),
      key: 'action',
      width: 160,
      fixed: 'right' as 'right',
      render: (_: unknown, record: Environment) => {
        if (!canEdit()) return '-'
        return (
          <div style={{ whiteSpace: 'nowrap' }}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} style={{ padding: '4px 8px' }}>{tc('edit', '编辑')}</Button>
            <Popconfirm title={t('deleteEnvConfirm', '确定要删除此环境吗？')} onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} style={{ padding: '4px 8px' }}>{tc('delete', '删除')}</Button>
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
        fields={searchFields}
        onSearch={handleSearch}
        onReset={handleResetSearch}
      />

      <div style={{ marginBottom: 16 }}>
        {canEdit() && (
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            {t('addEnvironment', '新增环境')}
          </Button>
        )}
      </div>
      <Table
        columns={columns}
        dataSource={filteredEnvironments}
        rowKey="id"
        loading={loading}
        scroll={{ x: 800 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total) => tc('total', '共 {{count}} 条', { count: total }), showQuickJumper: true }}
      />
      <Modal
        title={editingEnvironment ? t('editEnvironment', '编辑环境') : t('addEnvironment', '新增环境')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('envName', '环境名称')}
            rules={[{ required: true, message: t('envNameRequired', '请输入环境名称') }]}
          >
            <Input placeholder={t('envNamePlaceholder', '请输入环境名称')} />
          </Form.Item>
          <Form.Item
            name="type"
            label={t('envType', '环境类型')}
            rules={[{ required: true, message: t('envTypeRequired', '请选择环境类型') }]}
          >
            <Select placeholder={t('envTypePlaceholder', '请选择环境类型')}>
              <Select.Option value="dev">{t('envTypeDev', '开发环境')}</Select.Option>
              <Select.Option value="test">{t('envTypeTest', '测试环境')}</Select.Option>
              <Select.Option value="prod">{t('envTypeProd', '生产环境')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="description" label={t('description', '描述')}>
            <Input.TextArea rows={4} placeholder={t('envDescPlaceholder', '请输入环境描述')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
