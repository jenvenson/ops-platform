// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useMemo } from 'react'
import { Table, Button, Modal, Form, Input, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { cmdbAPI, Project } from '../../api/cmdb'
import SearchBar, { SearchField } from '../../components/SearchBar'
import { canEdit } from '../../utils/menuAccess'

// 搜索字段配置
const searchFields: SearchField[] = [
  { name: 'name', label: '项目名称', type: 'text' },
  { name: 'code', label: '项目编号', type: 'text' },
  { name: 'description', label: '描述', type: 'text' },
]

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)
  const [searchValues, setSearchValues] = useState<Record<string, string>>({})
  const [form] = Form.useForm()

  const fetchProjects = async () => {
    setLoading(true)
    try {
      const resp = await cmdbAPI.getProjects({ limit: 1000 })
      setProjects(resp.data)
    } catch (error) {
      message.error('加载项目失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProjects()
  }, [])

  // 搜索过滤后的数据
  const filteredProjects = useMemo(() => {
    return projects.filter((project) => {
      if (searchValues.name && !project.name?.toLowerCase().includes(searchValues.name.toLowerCase())) {
        return false
      }
      if (searchValues.code && !project.code?.toLowerCase().includes(searchValues.code.toLowerCase())) {
        return false
      }
      if (searchValues.description && !project.description?.toLowerCase().includes(searchValues.description.toLowerCase())) {
        return false
      }
      return true
    })
  }, [projects, searchValues])

  const handleSearch = (values: Record<string, string>) => {
    setSearchValues(values)
  }

  const handleResetSearch = () => {
    setSearchValues({})
  }

  const handleAdd = () => {
    setEditingProject(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: Project) => {
    setEditingProject(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await cmdbAPI.deleteProject(id)
      message.success('删除成功')
      fetchProjects()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingProject) {
        await cmdbAPI.updateProject(editingProject.id, values)
        message.success('更新成功')
      } else {
        await cmdbAPI.createProject(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      fetchProjects()
    } catch (error) {
      message.error('操作失败')
    }
  }

  const columns = [
    { title: '项目名称', dataIndex: 'name', key: 'name' },
    { title: '项目编号', dataIndex: 'code', key: 'code', width: 120 },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 160,
      fixed: 'right' as 'right',
      render: (_: unknown, record: Project) => {
        if (!canEdit()) return '-'
        return (
          <div style={{ whiteSpace: 'nowrap' }}>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} style={{ padding: '4px 8px' }}>编辑</Button>
            <Popconfirm title="确定要删除此项目吗？" onConfirm={() => handleDelete(record.id)}>
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
              新增项目
          </Button>
        )}
      </div>
      <Table
        columns={columns}
        dataSource={filteredProjects}
        rowKey="id"
        loading={loading}
        scroll={{ x: 600 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total) => `共 ${total} 条`, showQuickJumper: true }}
      />
      <Modal
        title={editingProject ? '编辑项目' : '新增项目'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="项目名称"
            rules={[{ required: true, message: '请输入项目名称' }]}
          >
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="项目编号"
            rules={[{ required: true, message: '请输入项目编号' }]}
          >
            <Input placeholder="请输入项目编号" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} placeholder="请输入项目描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}