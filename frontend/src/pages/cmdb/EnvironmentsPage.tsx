// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useMemo } from 'react'
import { Table, Button, Modal, Form, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { cmdbAPI, Environment } from '../../api/cmdb'
import SearchBar, { SearchField } from '../../components/SearchBar'
import { canEdit } from '../../utils/menuAccess'

// 搜索字段配置
const searchFields: SearchField[] = [
  { name: 'name', label: '环境名称', type: 'text' },
  { name: 'type', label: '类型', type: 'select', options: [
    { value: 'dev', label: '开发环境' },
    { value: 'test', label: '测试环境' },
    { value: 'prod', label: '生产环境' },
  ]},
]

export default function EnvironmentsPage() {
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingEnvironment, setEditingEnvironment] = useState<Environment | null>(null)
  const [searchValues, setSearchValues] = useState<Record<string, string>>({})
  const [form] = Form.useForm()

  const fetchEnvironments = async () => {
    setLoading(true)
    try {
      const resp = await cmdbAPI.getEnvironments({ limit: 1000 })
      setEnvironments(resp.data)
    } catch (error) {
      message.error('加载环境失败')
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
      message.success('删除成功')
      fetchEnvironments()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingEnvironment) {
        await cmdbAPI.updateEnvironment(editingEnvironment.id, values)
        message.success('更新成功')
      } else {
        await cmdbAPI.createEnvironment(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchEnvironments()
    } catch (error) {
      message.error('操作失败')
    }
  }

  const columns = [
    { title: '环境名称', dataIndex: 'name', key: 'name' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => {
        const color = type === 'prod' ? 'red' : type === 'test' ? 'orange' : 'green'
        return <span style={{ color, fontWeight: 'bold' }}>{type.toUpperCase()}</span>
      },
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 160,
      fixed: 'right' as 'right',
      render: (_: unknown, record: Environment) => {
        if (!canEdit()) return '-'
        return (
          <div style={{ whiteSpace: 'nowrap' }}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} style={{ padding: '4px 8px' }}>编辑</Button>
            <Popconfirm title="确定要删除此环境吗？" onConfirm={() => handleDelete(record.id)}>
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
        fields={searchFields}
        onSearch={handleSearch}
        onReset={handleResetSearch}
      />

      <div style={{ marginBottom: 16 }}>
        {canEdit() && (
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增环境
          </Button>
        )}
      </div>
      <Table
        columns={columns}
        dataSource={filteredEnvironments}
        rowKey="id"
        loading={loading}
        scroll={{ x: 800 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total) => `共 ${total} 条`, showQuickJumper: true }}
      />
      <Modal
        title={editingEnvironment ? '编辑环境' : '新增环境'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="环境名称"
            rules={[{ required: true, message: '请输入环境名称' }]}
          >
            <Input placeholder="请输入环境名称" />
          </Form.Item>
          <Form.Item
            name="type"
            label="环境类型"
            rules={[{ required: true, message: '请选择环境类型' }]}
          >
            <Select placeholder="请选择环境类型">
              <Select.Option value="dev">开发环境</Select.Option>
              <Select.Option value="test">测试环境</Select.Option>
              <Select.Option value="prod">生产环境</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} placeholder="请输入环境描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}