// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

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
import { useTranslation } from 'react-i18next'
import { alertAPI, AlertContact, AlertNotifyGroup } from '../../api/alert'
import { adminAPI, type User } from '../../api/admin'
import { canEdit } from '../../utils/menuAccess'

export default function AlertContactsPage() {
  const { t } = useTranslation('alarm')
  const { t: tc } = useTranslation('common')
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
    } catch { message.error(t('loadContactsFailed', '加载联系人失败')) }
    finally { setLoading(false) }
  }, [t])

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
      message.warning(t('selectUsers', '请选择用户'))
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
      message.success(t('createdContactsCount', '已创建 {{count}} 个联系人', { count: selectedUsers.length }))
      setUserSelectModal({ open: false })
      fetchContacts()
    } catch {
      message.error(t('createContactFailed', '创建联系人失败'))
    }
  }

  // -- 联系人 CRUD --
  const handleSaveContact = async () => {
    try {
      const values = await contactForm.validateFields()
      if (contactModal.contact) {
        await alertAPI.contacts.update(contactModal.contact.id, values)
        message.success(tc('updateSuccess', '更新成功'))
      } else {
        await alertAPI.contacts.create(values)
        message.success(tc('createSuccess', '创建成功'))
      }
      setContactModal({ open: false })
      fetchContacts()
    } catch {
      message.error(tc('operationFailed', '操作失败'))
    }
  }

  const handleDeleteContact = async (id: number) => {
    try {
      await alertAPI.contacts.delete(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchContacts()
    } catch {
      message.error(tc('deleteFailed', '删除失败'))
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
        message.success(tc('updateSuccess', '更新成功'))
      } else {
        await alertAPI.groups.create(data)
        message.success(tc('createSuccess', '创建成功'))
      }
      setGroupModal({ open: false })
      fetchGroups()
    } catch {
      message.error(tc('operationFailed', '操作失败'))
    }
  }

  const handleDeleteGroup = async (id: number) => {
    try {
      await alertAPI.groups.delete(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchGroups()
    } catch {
      message.error(tc('deleteFailed', '删除失败'))
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
    { title: t('nameLabel', '姓名'), dataIndex: 'name', key: 'name', width: 120 },
    { title: t('email', '邮箱'), dataIndex: 'email', key: 'email', width: 200 },
    { title: t('phone', '手机'), dataIndex: 'phone', key: 'phone', width: 140 },
    { title: t('dingtalk', '钉钉'), dataIndex: 'dingtalk', key: 'dingtalk', width: 150, render: (v: string) => v || '-' },
    { title: t('wechat', '企微'), dataIndex: 'wechat', key: 'wechat', width: 150, render: (v: string) => v || '-' },
    {
      title: t('belongingGroup', '所属组'), key: 'groups', width: 200,
      render: (_: unknown, r: AlertContact) => (r.groups || []).map(g => <Tag key={g.id} color="blue">{g.name}</Tag>),
    },
    {
      title: tc('action', '操作'), key: 'action', width: 140,
      render: (_: unknown, r: AlertContact) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openContactModal(r)}>{tc('edit', '编辑')}</Button>}
          {canEdit() && <Popconfirm title={t('confirmDelete', '确定删除？')} onConfirm={() => handleDeleteContact(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{tc('delete', '删除')}</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  const groupColumns = [
    { title: t('groupName', '组名'), dataIndex: 'name', key: 'name', width: 150 },
    { title: t('descriptionLabel', '描述'), dataIndex: 'description', key: 'description', width: 250, ellipsis: true },
    {
      title: tc('status', '状态'), dataIndex: 'enabled', key: 'enabled', width: 80,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? tc('enable', '启用') : tc('disable', '禁用')}</Tag>,
    },
    {
      title: t('members', '成员'), key: 'contacts', width: 300,
      render: (_: unknown, r: AlertNotifyGroup) => {
        const cs = r.contacts || []
        if (cs.length === 0) return <span style={{ color: '#999' }}>{t('noMembers', '暂无成员')}</span>
        return cs.map(c => <Tag key={c.id}>{c.name}</Tag>)
      },
    },
    {
      title: tc('action', '操作'), key: 'action', width: 140,
      render: (_: unknown, r: AlertNotifyGroup) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openGroupModal(r)}>{tc('edit', '编辑')}</Button>}
          {canEdit() && <Popconfirm title={t('confirmDelete', '确定删除？')} onConfirm={() => handleDeleteGroup(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{tc('delete', '删除')}</Button>
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
          key: 'contacts', label: t('contacts', '联系人'),
          children: (
            <div>
              <div style={{ marginBottom: 16 }}>
                <Space>
                  {canEdit() && <Button type="primary" icon={<PlusOutlined />} onClick={() => openContactModal()}>{t('addContact', '添加联系人')}</Button>}
                  {canEdit() && <Button icon={<TeamOutlined />} onClick={() => openUserSelectModal()}>{t('addFromUser', '从用户添加')}</Button>}
                </Space>
              </div>
              <Table columns={contactColumns} dataSource={contacts} rowKey="id" loading={loading} scroll={{ x: 1100 }}
                pagination={{ pageSize: 20, showTotal: tCount => tc('total', '共 {{count}} 条', { count: tCount }) }} />
            </div>
          ),
        },
        {
          key: 'groups', label: t('alertGroups', '报警组'),
          children: (
            <div>
              <div style={{ marginBottom: 16 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={() => openGroupModal()}>{t('createAlertGroup', '创建报警组')}</Button>
              </div>
              <Table columns={groupColumns} dataSource={groups} rowKey="id" scroll={{ x: 900 }}
                pagination={{ pageSize: 20, showTotal: tCount => tc('total', '共 {{count}} 条', { count: tCount }) }} />
            </div>
          ),
        },
      ]} />

      {/* 联系人弹窗 */}
      <Modal title={contactModal.contact ? t('editContact', '编辑联系人') : t('addContact', '添加联系人')} open={contactModal.open}
        onOk={handleSaveContact} onCancel={() => setContactModal({ open: false })} width={500}>
        <Form form={contactForm} layout="vertical">
          <Form.Item name="name" label={t('nameLabel', '姓名')} rules={[{ required: true, message: t('inputName', '请输入姓名') }]}>
            <Input placeholder={t('inputName', '请输入姓名')} />
          </Form.Item>
          <Form.Item name="email" label={t('email', '邮箱')}>
            <Input placeholder={t('inputEmail', '请输入邮箱')} prefix={<MailOutlined />} />
          </Form.Item>
          <Form.Item name="phone" label={t('phone', '手机号')}>
            <Input placeholder={t('inputPhone', '请输入手机号')} />
          </Form.Item>
          <Form.Item name="dingtalk" label={t('dingtalkLabel', '钉钉（UserID 或手机号）')}>
            <Input placeholder={t('dingtalkPlaceholder', '用于钉钉通知')} />
          </Form.Item>
          <Form.Item name="wechat" label={t('wechatWorkLabel', '企业微信（UserID）')}>
            <Input placeholder={t('wechatPlaceholder', '用于企微通知')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 报警组弹窗 */}
      <Modal title={groupModal.group ? t('editAlertGroup', '编辑报警组') : t('createAlertGroup', '创建报警组')} open={groupModal.open}
        onOk={handleSaveGroup} onCancel={() => setGroupModal({ open: false })} width={650}>
        <Form form={groupForm} layout="vertical">
          <Form.Item name="name" label={t('groupName', '组名')} rules={[{ required: true, message: t('inputGroupName', '请输入组名') }]}>
            <Input placeholder={t('inputAlertGroupName', '请输入报警组名称')} />
          </Form.Item>
          <Form.Item name="description" label={t('descriptionLabel', '描述')}>
            <Input.TextArea rows={2} placeholder={t('alertGroupDescription', '报警组描述')} />
          </Form.Item>
          <Form.Item name="enabled" label={t('enableLabel', '启用')} valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Form.Item label={t('selectMembers', '选择成员')}>
            <Transfer dataSource={contactTransferData} targetKeys={selectedContactKeys} onChange={handleTransferChange}
              render={item => `${item.title} ${item.description ? `(${item.description})` : ''}`}
              titles={[t('availableContacts', '可选联系人'), t('selectedMembers', '已选成员')]} listStyle={{ width: 260, height: 250 }} showSearch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 从用户选择弹窗 */}
      <Modal
        title={t('addContactFromUser', '从用户添加联系人')}
        open={userSelectModal.open}
        onOk={handleCreateContactsFromUsers}
        onCancel={() => setUserSelectModal({ open: false })}
        width={700}
        okText={tc('add', '添加')}
        destroyOnClose
      >
        <Transfer
          dataSource={userTransferData}
          targetKeys={selectedUserIds.map(String)}
          onChange={handleUserTransferChange}
          render={item => `${item.title} ${item.description ? `(${item.description})` : ''}`}
          titles={[t('availableUsers', '可选用户'), t('selectedUsers', '已选用户')]}
          listStyle={{ width: 280, height: 300 }}
          showSearch
        />
      </Modal>
    </>
  )
}
