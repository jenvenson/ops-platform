import { useState, useEffect, useCallback } from 'react'
import {
  Tabs, Table, Tag, Button, Modal, Form, Input, Space,
  message, Popconfirm, Switch, Transfer,
} from 'antd'
import type { TransferProps } from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, MailOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { alertAPI, AlertContact, AlertNotifyGroup } from '../../api/alert'
import { adminAPI, type User } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'

export default function AlertContactsPage() {
  const [contacts, setContacts] = useState<AlertContact[]>([])
  const [groups, setGroups] = useState<AlertNotifyGroup[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [activeSubTab, setActiveSubTab] = useState('contacts')
  const [contactModal, setContactModal] = useState<{ open: boolean; contact?: AlertContact }>({ open: false })
  const [groupModal, setGroupModal] = useState<{ open: boolean; group?: AlertNotifyGroup }>({ open: false })
  const [userSelectModal, setUserSelectModal] = useState<{ open: boolean }>({ open: false })
  const [contactForm] = Form.useForm()
  const [groupForm] = Form.useForm()
  const [selectedContactKeys, setSelectedContactKeys] = useState<string[]>([])
  const [selectedUserIds, setSelectedUserIds] = useState<number[]>([])

  const fetchContacts = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await alertAPI.contacts.list()
      setContacts(resp.data || [])
    } catch { message.error('加载联系人失败') }
    finally { setLoading(false) }
  }, [])

  const fetchUsers = useCallback(async () => {
    try {
      const resp = await adminAPI.getUsers()
      setUsers(resp)
    } catch { /* ignore */ }
  }, [])

  const fetchGroups = useCallback(async () => {
    try {
      const resp = await alertAPI.groups.list()
      setGroups(resp.data || [])
    } catch { /* ignore */ }
  }, [])

  useEffect(() => { fetchContacts(); fetchGroups() }, [fetchContacts, fetchGroups])

  // -- 从用户创建联系人 --
  const openUserSelectModal = () => {
    setSelectedUserIds([])
    fetchUsers()
    setUserSelectModal({ open: true })
  }

  const handleUserTransferChange: TransferProps['onChange'] = (keys) => {
    setSelectedUserIds((keys as string[]).map(Number))
  }

  const handleTransferChange: TransferProps['onChange'] = (keys) => {
    setSelectedContactKeys(keys as string[])
  }

  const handleCreateContactsFromUsers = async () => {
    if (selectedUserIds.length === 0) {
      message.warning('请选择用户')
      return
    }
    try {
      const selectedUsers = users.filter(u => selectedUserIds.includes(u.id))
      // 批量创建联系人
      for (const user of selectedUsers) {
        // 检查是否已存在同名联系人
        const existing = contacts.find(c => c.name === user.real_name || c.name === user.username)
        if (existing) continue
        await alertAPI.contacts.create({
          name: user.real_name || user.username,
          email: user.email || '',
          phone: '',
          dingtalk: '',
          wechat: '',
        })
      }
      message.success(`已创建 ${selectedUsers.length} 个联系人`)
      setUserSelectModal({ open: false })
      fetchContacts()
    } catch {
      message.error('创建联系人失败')
    }
  }

  // -- 联系人 CRUD --
  const handleSaveContact = async () => {
    try {
      const values = await contactForm.validateFields()
      if (contactModal.contact) {
        await alertAPI.contacts.update(contactModal.contact.id, values)
        message.success('更新成功')
      } else {
        await alertAPI.contacts.create(values)
        message.success('创建成功')
      }
      setContactModal({ open: false })
      fetchContacts()
    } catch {
      message.error('操作失败')
    }
  }

  const handleDeleteContact = async (id: number) => {
    try {
      await alertAPI.contacts.delete(id)
      message.success('删除成功')
      fetchContacts()
    } catch {
      message.error('删除失败')
    }
  }

  const openContactModal = (contact?: AlertContact) => {
    setContactModal({ open: true, contact })
    if (contact) {
      contactForm.setFieldsValue(contact)
    } else {
      contactForm.resetFields()
    }
  }

  // -- 报警组 CRUD --
  const handleSaveGroup = async () => {
    try {
      const values = await groupForm.validateFields()
      const data = {
        name: values.name,
        description: values.description || '',
        enabled: values.enabled ?? true,
        contact_ids: selectedContactKeys.map(Number),
      }
      if (groupModal.group) {
        await alertAPI.groups.update(groupModal.group.id, data)
        message.success('更新成功')
      } else {
        await alertAPI.groups.create(data)
        message.success('创建成功')
      }
      setGroupModal({ open: false })
      fetchGroups()
    } catch {
      message.error('操作失败')
    }
  }

  const handleDeleteGroup = async (id: number) => {
    try {
      await alertAPI.groups.delete(id)
      message.success('删除成功')
      fetchGroups()
    } catch {
      message.error('删除失败')
    }
  }

  const openGroupModal = (group?: AlertNotifyGroup) => {
    setGroupModal({ open: true, group })
    if (group) {
      groupForm.setFieldsValue(group)
      setSelectedContactKeys((group.contacts || []).map(c => String(c.id)))
    } else {
      groupForm.resetFields()
      setSelectedContactKeys([])
    }
  }

  const contactColumns = [
    { title: '姓名', dataIndex: 'name', key: 'name', width: 120 },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 200 },
    { title: '手机', dataIndex: 'phone', key: 'phone', width: 140 },
    { title: '钉钉', dataIndex: 'dingtalk', key: 'dingtalk', width: 150, render: (v: string) => v || '-' },
    { title: '企微', dataIndex: 'wechat', key: 'wechat', width: 150, render: (v: string) => v || '-' },
    {
      title: '所属组', key: 'groups', width: 200,
      render: (_: unknown, r: AlertContact) => (r.groups || []).map(g => <Tag key={g.id} color="blue">{g.name}</Tag>),
    },
    {
      title: '操作', key: 'action', width: 140,
      render: (_: unknown, r: AlertContact) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openContactModal(r)}>编辑</Button>}
          {canEdit() && <Popconfirm title="确定删除？" onConfirm={() => handleDeleteContact(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  const groupColumns = [
    { title: '组名', dataIndex: 'name', key: 'name', width: 150 },
    { title: '描述', dataIndex: 'description', key: 'description', width: 250, ellipsis: true },
    {
      title: '状态', dataIndex: 'enabled', key: 'enabled', width: 80,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag>,
    },
    {
      title: '成员', key: 'contacts', width: 300,
      render: (_: unknown, r: AlertNotifyGroup) => {
        const cs = r.contacts || []
        if (cs.length === 0) return <span style={{ color: '#999' }}>暂无成员</span>
        return cs.map(c => <Tag key={c.id}>{c.name}</Tag>)
      },
    },
    {
      title: '操作', key: 'action', width: 140,
      render: (_: unknown, r: AlertNotifyGroup) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openGroupModal(r)}>编辑</Button>}
          {canEdit() && <Popconfirm title="确定删除？" onConfirm={() => handleDeleteGroup(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  const contactTransferData = contacts.map(c => ({ key: String(c.id), title: c.name, description: c.email || c.phone || '' }))
  const userTransferData = users.map(u => ({ key: String(u.id), title: u.real_name || u.username, description: u.email || '' }))

  return (
    <>
      <Tabs activeKey={activeSubTab} onChange={setActiveSubTab} size="small" items={[
        {
          key: 'contacts', label: '联系人',
          children: (
            <div>
              <div style={{ marginBottom: 16 }}>
                <Space>
                  {canEdit() && <Button type="primary" icon={<PlusOutlined />} onClick={() => openContactModal()}>添加联系人</Button>}
                  {canEdit() && <Button icon={<TeamOutlined />} onClick={() => openUserSelectModal()}>从用户添加</Button>}
                </Space>
              </div>
              <Table columns={contactColumns} dataSource={contacts} rowKey="id" loading={loading} scroll={{ x: 1100 }}
                pagination={{ pageSize: 20, showTotal: t => `共 ${t} 条` }} />
            </div>
          ),
        },
        {
          key: 'groups', label: '报警组',
          children: (
            <div>
              <div style={{ marginBottom: 16 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openGroupModal()}>创建报警组</Button>
              </div>
              <Table columns={groupColumns} dataSource={groups} rowKey="id" scroll={{ x: 900 }}
                pagination={{ pageSize: 20, showTotal: t => `共 ${t} 条` }} />
            </div>
          ),
        },
      ]} />

      {/* 联系人弹窗 */}
      <Modal title={contactModal.contact ? '编辑联系人' : '添加联系人'} open={contactModal.open}
        onOk={handleSaveContact} onCancel={() => setContactModal({ open: false })} width={500}>
        <Form form={contactForm} layout="vertical">
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
            <Input placeholder="请输入姓名" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="请输入邮箱" prefix={<MailOutlined />} />
          </Form.Item>
          <Form.Item name="phone" label="手机号">
            <Input placeholder="请输入手机号" />
          </Form.Item>
          <Form.Item name="dingtalk" label="钉钉（UserID 或手机号）">
            <Input placeholder="用于钉钉通知" />
          </Form.Item>
          <Form.Item name="wechat" label="企业微信（UserID）">
            <Input placeholder="用于企微通知" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 报警组弹窗 */}
      <Modal title={groupModal.group ? '编辑报警组' : '创建报警组'} open={groupModal.open}
        onOk={handleSaveGroup} onCancel={() => setGroupModal({ open: false })} width={650}>
        <Form form={groupForm} layout="vertical">
          <Form.Item name="name" label="组名" rules={[{ required: true, message: '请输入组名' }]}>
            <Input placeholder="请输入报警组名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="报警组描述" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Form.Item label="选择成员">
            <Transfer dataSource={contactTransferData} targetKeys={selectedContactKeys} onChange={handleTransferChange}
              render={item => `${item.title} ${item.description ? `(${item.description})` : ''}`}
              titles={['可选联系人', '已选成员']} listStyle={{ width: 260, height: 250 }} showSearch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 从用户选择弹窗 */}
      <Modal
        title="从用户添加联系人"
        open={userSelectModal.open}
        onOk={handleCreateContactsFromUsers}
        onCancel={() => setUserSelectModal({ open: false })}
        width={700}
        okText="添加"
        destroyOnClose
      >
        <Transfer
          dataSource={userTransferData}
          targetKeys={selectedUserIds.map(String)}
          onChange={handleUserTransferChange}
          render={item => `${item.title} ${item.description ? `(${item.description})` : ''}`}
          titles={['可选用户', '已选用户']}
          listStyle={{ width: 280, height: 300 }}
          showSearch
        />
      </Modal>
    </>
  )
}
