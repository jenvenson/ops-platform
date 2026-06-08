import { useState, useEffect, useCallback } from 'react'
import { Card, Table, Tag, Space, Button, Modal, Form, Input, Select, InputNumber, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { adminAPI, Menu } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'

// 树形菜单类型
interface MenuTreeNode extends Menu {
  children?: MenuTreeNode[]
}

const flattenMenuTree = (nodes: MenuTreeNode[]): MenuTreeNode[] => {
  const result: MenuTreeNode[] = []

  const walk = (items: MenuTreeNode[]) => {
    items.forEach((item) => {
      result.push(item)
      if (item.children?.length) {
        walk(item.children)
      }
    })
  }

  walk(nodes)
  return result
}

const findSiblingNodes = (nodes: MenuTreeNode[], parentID: number): MenuTreeNode[] => {
  if (parentID === 0) {
    return [...nodes].sort((a, b) => (a.sort || 0) - (b.sort || 0))
  }

  const stack = [...nodes]
  while (stack.length > 0) {
    const current = stack.shift()
    if (!current) continue
    if (current.id === parentID) {
      return [...(current.children || [])].sort((a, b) => (a.sort || 0) - (b.sort || 0))
    }
    if (current.children?.length) {
      stack.push(...current.children)
    }
  }

  return []
}

// 将扁平菜单转换为树形结构
const buildMenuTree = (menus: Menu[]): MenuTreeNode[] => {
  const menuMap = new Map<number, MenuTreeNode>()
  const roots: MenuTreeNode[] = []

  // 创建所有节点
  menus.forEach(menu => {
    menuMap.set(menu.id, { ...menu, children: [] })
  })

  // 构建树
  menus.forEach(menu => {
    const node = menuMap.get(menu.id)!
    if (menu.parent_id === 0) {
      roots.push(node)
    } else {
      const parent = menuMap.get(menu.parent_id)
      if (parent) {
        parent.children!.push(node)
      }
    }
  })

  // 递归清理空 children 并排序
  const cleanAndSortTree = (nodes: MenuTreeNode[]): MenuTreeNode[] => {
    const sorted = [...nodes].sort((a, b) => (a.sort || 0) - (b.sort || 0))
    return sorted.map(node => {
      // 递归处理子节点
      if (node.children && node.children.length > 0) {
        node.children = cleanAndSortTree(node.children)
      } else {
        node.children = undefined
      }
      return node
    })
  }

  // 对根菜单排序
  const sortedRoots = [...roots].sort((a, b) => (a.sort || 0) - (b.sort || 0))

  return cleanAndSortTree(sortedRoots)
}

export default function MenusPage() {
  const [menus, setMenus] = useState<Menu[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingMenu, setEditingMenu] = useState<Menu | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  // 加载菜单列表
  const fetchMenus = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await adminAPI.getMenus()
      // 转换为树形数据
      const treeData = buildMenuTree(resp)
      setMenus(treeData as unknown as Menu[])
    } catch (error) {
      console.error('获取菜单列表失败:', error)
      message.error('获取菜单列表失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchMenus()
  }, [fetchMenus])

  // 打开新增模态框
  const handleAdd = async (parentId?: number) => {
    setEditingMenu(null)
    if (menus.length === 0) {
      await fetchMenus()
    }
    form.resetFields()
    form.setFieldsValue({ parent_id: parentId || 0, sort: 0, status: 'active' })
    setModalVisible(true)
  }

  // 打开编辑模态框
  const handleEdit = async (menu: Menu) => {
    setEditingMenu(menu)
    if (menus.length === 0) {
      await fetchMenus()
    }
    form.setFieldsValue({
      title: menu.title,
      key: menu.key,
      path: menu.path,
      icon: menu.icon,
      parent_id: menu.parent_id,
      sort: menu.sort,
      status: menu.status === 1 ? 'active' : 'disabled',
    })
    setModalVisible(true)
  }

  // 删除菜单
  const handleDelete = async (id: number) => {
    try {
      await adminAPI.deleteMenu(id)
      message.success('删除成功')
      fetchMenus()
      refreshUserMenus()
    } catch (error) {
      console.error('删除菜单失败:', error)
      message.error(error instanceof Error ? error.message : '删除菜单失败')
    }
  }

  const handleMove = async (menu: Menu, direction: 'up' | 'down') => {
    const siblingNodes = findSiblingNodes(menus as unknown as MenuTreeNode[], menu.parent_id)
    const currentIndex = siblingNodes.findIndex((item) => item.id === menu.id)
    if (currentIndex < 0) return

    const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
    if (targetIndex < 0 || targetIndex >= siblingNodes.length) return

    const reordered = [...siblingNodes]
    const [moved] = reordered.splice(currentIndex, 1)
    reordered.splice(targetIndex, 0, moved)

    try {
      setSubmitting(true)
      await Promise.all(
        reordered.map((item, index) => adminAPI.updateMenu(item.id, { sort: index }))
      )
      message.success('排序已更新')
      await fetchMenus()
      await refreshUserMenus()
    } catch (error) {
      console.error('更新菜单排序失败:', error)
      message.error(error instanceof Error ? error.message : '更新菜单排序失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 刷新用户菜单缓存
  const refreshUserMenus = async () => {
    try {
      // 通知主布局组件重新加载菜单
      // 方案1：清除缓存的菜单数据，让用户在下次访问时重新获取
      localStorage.removeItem('user_menus');

      // 方案2：触发全局事件，通知所有组件更新
      window.dispatchEvent(new CustomEvent('menuUpdated'));

      message.success('菜单已更新，请刷新页面以查看最新菜单');
    } catch (error) {
      console.error('刷新菜单缓存失败:', error)
    }
  }

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)

      if (editingMenu) {
        // 更新菜单
        await adminAPI.updateMenu(editingMenu.id, {
          title: values.title,
          key: values.key,
          path: values.path,
          icon: values.icon,
          parent_id: values.parent_id,
          sort: values.sort,
          status: values.status === 'active' ? 1 : 0,
        })
        message.success('更新成功')
      } else {
        // 创建菜单
        await adminAPI.createMenu({
          title: values.title,
          key: values.key,
          path: values.path,
          icon: values.icon,
          parent_id: values.parent_id || 0,
          sort: values.sort || 0,
          status: values.status === 'active' ? 1 : 0,
        })
        message.success('创建成功')
      }

      setModalVisible(false)
      fetchMenus()
      refreshUserMenus()
    } catch (error) {
      console.error('提交失败:', error)
      message.error(error instanceof Error ? error.message : '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 获取父级菜单选项
  const flatMenus = flattenMenuTree(menus as unknown as MenuTreeNode[])

  const parentMenuOptions = [
    { label: '一级菜单', value: 0 },
    ...flatMenus
      .filter(m => m.parent_id === 0)
      .map(m => ({
        label: m.title,
        value: m.id,
      })),
  ]

  // 表格列定义
  const columns = [
    {
      title: '菜单标题',
      dataIndex: 'title',
      key: 'title',
      width: 260,
      ellipsis: {
        showTitle: true,
      },
      render: (title: string) => (
        <span style={{ display: 'inline-block', whiteSpace: 'nowrap' }}>
          {title}
        </span>
      ),
    },
    {
      title: '菜单标识',
      dataIndex: 'key',
      key: 'key',
      width: 150,
    },
    {
      title: '路径',
      dataIndex: 'path',
      key: 'path',
      width: 200,
      render: (path: string) => path || '-',
    },
    {
      title: '图标',
      dataIndex: 'icon',
      key: 'icon',
      width: 120,
      render: (icon: string) => icon ? <Tag>{icon}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={status === 1 ? 'success' : 'default'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '排序',
      dataIndex: 'sort',
      key: 'sort',
      width: 180,
      render: (sort: number, record: Menu) => {
        const siblingNodes = findSiblingNodes(menus as unknown as MenuTreeNode[], record.parent_id)
        const currentIndex = siblingNodes.findIndex((item) => item.id === record.id)
        const canMoveUp = currentIndex > 0
        const canMoveDown = currentIndex >= 0 && currentIndex < siblingNodes.length - 1

        return (
          <Space size={4}>
            <span>{sort}</span>
            {canEdit() && (
              <>
                <Button
                  size="small"
                  onClick={() => handleMove(record, 'up')}
                  disabled={!canMoveUp || submitting}
                >
                  上移
                </Button>
                <Button
                  size="small"
                  onClick={() => handleMove(record, 'down')}
                  disabled={!canMoveDown || submitting}
                >
                  下移
                </Button>
              </>
            )}
          </Space>
        )
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_: unknown, record: Menu) => {
        if (!canEdit()) return '-'
        return (
          <Space>
            <Button
              type="link"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => handleAdd(record.id)}
            >
              新增
            </Button>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title="确认删除"
              description="确定要删除这个菜单吗？子菜单也会被删除。"
              onConfirm={() => handleDelete(record.id)}
              okText="确认"
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
        )
      },
    },
  ]

  return (
    <div>
      <Card
        title="菜单管理"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchMenus}>
              刷新
            </Button>
            {canEdit() && (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => handleAdd()}>
                添加菜单
              </Button>
            )}
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={menus}
          rowKey="id"
          loading={loading}
          pagination={false}
          scroll={{ x: 1320 }}
          locale={{ emptyText: '暂无菜单数据' }}
          expandable={{
            defaultExpandAllRows: true,
            indentSize: 20,
            childrenColumnName: 'children',
            expandIcon: ({ expanded, onExpand, record }: any) => {
              const hasChildren = record.children && record.children.length > 0
              if (!hasChildren) {
                return <span style={{ marginRight: 8 }} />
              }
              return (
                <span
                  onClick={(e) => onExpand(record, e)}
                  style={{
                    cursor: 'pointer',
                    marginRight: 8,
                    display: 'inline-block',
                    width: 0,
                    height: 0,
                    borderLeft: '6px solid transparent',
                    borderRight: '6px solid transparent',
                    borderTop: expanded ? '6px solid #333' : '6px solid #333',
                    transform: expanded ? 'rotate(0deg)' : 'rotate(-90deg)',
                    transition: 'transform 0.2s',
                    verticalAlign: 'middle',
                  }}
                />
              )
            },
          }}
        />
      </Card>

      {/* 新增/编辑菜单模态框 */}
      <Modal
        title={editingMenu ? '编辑菜单' : '新增菜单'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="title"
            label="菜单标题"
            rules={[
              { required: true, message: '请输入菜单标题' },
              { min: 1, max: 50, message: '菜单标题长度必须在1-50之间' },
            ]}
          >
            <Input placeholder="请输入菜单标题" />
          </Form.Item>

          <Form.Item
            name="key"
            label="菜单标识"
            rules={[
              { required: true, message: '请输入菜单标识' },
              { pattern: /^[a-z][a-z0-9-]*$/, message: '菜单标识必须以小写字母开头，只能包含小写字母、数字和连字符' },
            ]}
          >
            <Input placeholder="请输入菜单标识" disabled={!!editingMenu} />
          </Form.Item>

          <Form.Item name="path" label="菜单路径">
            <Input placeholder="请输入菜单路径（可选）" />
          </Form.Item>

          <Form.Item name="icon" label="图标">
            <Select
              placeholder="选择图标"
              options={[
                { label: 'DashboardOutlined', value: 'DashboardOutlined' },
                { label: 'SettingOutlined', value: 'SettingOutlined' },
                { label: 'RocketOutlined', value: 'RocketOutlined' },
                { label: 'MonitorOutlined', value: 'MonitorOutlined' },
                { label: 'UserOutlined', value: 'UserOutlined' },
                { label: 'TeamOutlined', value: 'TeamOutlined' },
                { label: 'MenuOutlined', value: 'MenuOutlined' },
                { label: 'ProjectOutlined', value: 'ProjectOutlined' },
                { label: 'CloudOutlined', value: 'CloudOutlined' },
                { label: 'DesktopOutlined', value: 'DesktopOutlined' },
                { label: 'AppstoreOutlined', value: 'AppstoreOutlined' },
                { label: 'HistoryOutlined', value: 'HistoryOutlined' },
                { label: 'InboxOutlined', value: 'InboxOutlined' },
                { label: 'BellOutlined', value: 'BellOutlined' },
                { label: 'SafetyOutlined', value: 'SafetyOutlined' },
                { label: 'ToolOutlined', value: 'ToolOutlined' },
                { label: 'ScanOutlined', value: 'ScanOutlined' },
                { label: 'BugOutlined', value: 'BugOutlined' },
              ]}
              allowClear
            />
          </Form.Item>

          <Form.Item name="parent_id" label="父级菜单">
            <Select
              placeholder="选择父级菜单"
              options={parentMenuOptions}
            />
          </Form.Item>

          <Form.Item name="sort" label="排序">
            <InputNumber min={0} placeholder="排序号" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="status" label="状态">
            <Select
              options={[
                { label: '启用', value: 'active' },
                { label: '禁用', value: 'disabled' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
