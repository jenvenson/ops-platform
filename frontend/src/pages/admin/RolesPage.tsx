// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import { Card, Table, Tag, Space, Button, Modal, Input, Form, Select, message, Popconfirm, Checkbox, Tree } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminAPI, Role, Menu } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'
import { formatDateTime } from '../../utils/dateFormat'

const roleDisplayKeyMap: Record<string, string> = {
  admin: 'roleDisplayNameAdmin',
  ops: 'roleDisplayNameOps',
  dev: 'roleDisplayNameDev',
  qa: 'roleDisplayNameQa',
  user: 'roleDisplayNameUser',
}

const roleDescriptionKeyMap: Record<string, string> = {
  admin: 'roleDescriptionAdmin',
  ops: 'roleDescriptionOps',
  dev: 'roleDescriptionDev',
  user: 'roleDescriptionUser',
}

interface MenuNode {
  id: number
  title: string
  key: string
  children?: MenuNode[]
}

export default function RolesPage() {
  const { t } = useTranslation('admin')
  const { t: tc } = useTranslation('common')

  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingRole, setEditingRole] = useState<Role | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const [menuModalVisible, setMenuModalVisible] = useState(false)
  const [currentRole, setCurrentRole] = useState<Role | null>(null)
  const [allMenus, setAllMenus] = useState<Menu[]>([])
  const [selectedMenuIds, setSelectedMenuIds] = useState<number[]>([])

  const fetchRoles = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await adminAPI.getRoles()
      setRoles(resp)
    } catch (error) {
      console.error('获取角色列表失败:', error)
      message.error(t('getRolesFailed', '获取角色列表失败'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchRoles()
  }, [fetchRoles])

  const handleAdd = () => {
    setEditingRole(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (role: Role) => {
    setEditingRole(role)
    form.setFieldsValue({
      name: role.name,
      code: role.code,
      description: role.description,
      status: role.status === 1 ? 'active' : 'disabled',
    })
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await adminAPI.deleteRole(id)
      message.success(t('deleteRoleSuccess', '删除成功'))
      fetchRoles()
    } catch (error) {
      console.error('删除角色失败:', error)
      message.error(error instanceof Error ? error.message : t('deleteRoleFailed', '删除角色失败'))
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)

      if (editingRole) {
        await adminAPI.updateRole(editingRole.id, {
          name: values.name,
          code: values.code,
          description: values.description,
          status: values.status === 'active' ? 1 : 0,
        })
        message.success(t('updateRoleSuccess', '更新成功'))
      } else {
        await adminAPI.createRole({
          name: values.name,
          code: values.code,
          description: values.description,
          status: values.status === 'active' ? 1 : 0,
        })
        message.success(t('createRoleSuccess', '创建成功'))
      }

      setModalVisible(false)
      fetchRoles()
    } catch (error) {
      console.error('提交失败:', error)
      message.error(error instanceof Error ? error.message : t('submitFailed', '提交失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleConfigMenus = async (role: Role) => {
    setCurrentRole(role)
    try {
      const menusResp = await adminAPI.getMenus()
      setAllMenus(menusResp)

      const roleMenusResp = await adminAPI.getRoleMenus(role.id)
      setSelectedMenuIds(roleMenusResp.menu_ids)

      setMenuModalVisible(true)
    } catch (error) {
      console.error('获取数据失败:', error)
      message.error(t('getDataFailed', '获取数据失败'))
    }
  }

  const buildTreeData = (): MenuNode[] => {
    const menuMap = new Map<number, MenuNode>()
    const roots: MenuNode[] = []

    allMenus.forEach(menu => {
      menuMap.set(menu.id, { id: menu.id, title: menu.title, key: String(menu.id) })
    })

    allMenus.forEach(menu => {
      const node = menuMap.get(menu.id)!
      if (menu.parent_id === 0) {
        roots.push(node)
      } else {
        const parent = menuMap.get(menu.parent_id)
        if (parent) {
          parent.children = parent.children || []
          parent.children.push(node)
        }
      }
    })

    return roots
  }

  const handleMenuCheck = (checkedKeys: React.Key[] | { checked: React.Key[]; halfChecked: React.Key[] }) => {
    const keys = Array.isArray(checkedKeys) ? checkedKeys : checkedKeys.checked
    setSelectedMenuIds(keys.map(k => Number(k)))
  }

  const handleSaveMenus = async () => {
    if (!currentRole) return

    try {
      await adminAPI.updateRoleMenus(currentRole.id, selectedMenuIds)
      message.success(t('menuPermissionSaved', '菜单权限配置保存成功'))
      setMenuModalVisible(false)
    } catch (error) {
      console.error('保存失败:', error)
      message.error(t('saveFailed', '保存失败'))
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: t('roleName', '角色名称'),
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (name: string, record: Role) => {
        const key = roleDisplayKeyMap[record.code]
        return key ? t(key, name) : name
      },
    },
    {
      title: t('roleCode', '角色编码'),
      dataIndex: 'code',
      key: 'code',
      width: 150,
    },
    {
      title: t('description', '描述'),
      dataIndex: 'description',
      key: 'description',
      width: 300,
      render: (desc: string, record: Role) => {
        if (!desc) return '-'
        const key = roleDescriptionKeyMap[record.code]
        return key ? t(key, desc) : desc
      },
    },
    {
      title: t('status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={status === 1 ? 'success' : 'default'}>
          {status === 1 ? t('enable', '启用') : t('disable', '禁用')}
        </Tag>
      ),
    },
    {
      title: t('createdAt', '创建时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? formatDateTime(time) : '-',
    },
    {
      title: t('action', '操作'),
      key: 'action',
      width: 200,
      render: (_: unknown, record: Role) => {
        if (!canEdit()) return '-'
        return (
          <Space>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              {t('edit', '编辑')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<SettingOutlined />}
              onClick={() => handleConfigMenus(record)}
            >
              {t('configMenus', '配置菜单')}
            </Button>
            {record.code !== 'admin' && (
              <Popconfirm
                title={t('confirmDeleteRole', '确认删除')}
                description={t('confirmDeleteRoleDesc', '确定要删除这个角色吗？')}
                onConfirm={() => handleDelete(record.id)}
                okText={t('confirm', '确认')}
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
        )
      },
    },
  ]

  return (
    <div>
      <Card
        title={t('roleManagement', '角色管理')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchRoles}>
              {t('refresh', '刷新')}
            </Button>
            {canEdit() && (
              <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                {t('addRole', '添加角色')}
              </Button>
            )}
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={roles}
          rowKey="id"
          loading={loading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showTotal: (total: number) => tc('total', '共 {{count}} 条', { count: total }), showQuickJumper: true }}
          scroll={{ x: 1000 }}
          locale={{ emptyText: t('noRolesData', '暂无角色数据') }}
        />
      </Card>

      <Modal
        title={editingRole ? t('editRole', '编辑角色') : t('addNewRole', '新增角色')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('roleName', '角色名称')}
            rules={[
              { required: true, message: t('roleNameRequired', '请输入角色名称') },
              { min: 2, max: 50, message: t('roleNameLength', '角色名称长度必须在2-50之间') },
            ]}
          >
            <Input placeholder={t('roleNamePlaceholder', '请输入角色名称')} />
          </Form.Item>

          <Form.Item
            name="code"
            label={t('roleCode', '角色编码')}
            rules={[
              { required: true, message: t('roleCodeRequired', '请输入角色编码') },
              { pattern: /^[a-z][a-z0-9_]*$/, message: t('roleCodePattern', '角色编码必须以小写字母开头，只能包含小写字母、数字和下划线') },
            ]}
          >
            <Input placeholder={t('roleCodePlaceholder', '请输入角色编码')} disabled={!!editingRole} />
          </Form.Item>

          <Form.Item name="description" label={t('description', '描述')}>
            <Input.TextArea
              placeholder={t('descriptionPlaceholder', '请输入角色描述（可选）')}
              rows={3}
            />
          </Form.Item>

          <Form.Item name="status" label={t('status', '状态')}>
            <Select
              options={[
                { label: t('enable', '启用'), value: 'active' },
                { label: t('disable', '禁用'), value: 'disabled' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`${t('configMenuPermission', '配置菜单权限')} - ${currentRole?.name}`}
        open={menuModalVisible}
        onCancel={() => setMenuModalVisible(false)}
        onOk={handleSaveMenus}
        width={700}
      >
        <div style={{ marginBottom: 16 }}>
          <Checkbox
            checked={selectedMenuIds.length === allMenus.length}
            indeterminate={selectedMenuIds.length > 0 && selectedMenuIds.length < allMenus.length}
            onChange={(e) => {
              if (e.target.checked) {
                setSelectedMenuIds(allMenus.map(m => m.id))
              } else {
                setSelectedMenuIds([])
              }
            }}
          >
            {t('selectAll', '全选/取消全选')}
          </Checkbox>
        </div>
        <Tree
          checkable
          checkedKeys={selectedMenuIds.map(id => String(id))}
          onCheck={handleMenuCheck}
          treeData={buildTreeData()}
          height={400}
          blockNode
        />
      </Modal>
    </div>
  )
}
