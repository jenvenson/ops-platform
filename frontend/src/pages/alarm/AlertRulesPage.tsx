// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import {
  Table, Tag, Button, Modal, Form, Input, Select, Space,
  message, Popconfirm, Switch,
} from 'antd'
import {
  SyncOutlined, EditOutlined, DeleteOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { alertAPI, AlertRule, AlertNotifyGroup } from '../../api/alert'
import { monitorAPI } from '../../api/monitor'
import { severityConfig, categoryConfig } from './alarmConfig'
import { canEdit } from '../../utils/menuAccess'

export default function AlertRulesPage() {
  const { t } = useTranslation('alarm')
  const { t: tc } = useTranslation('common')
  const [rules, setRules] = useState<AlertRule[]>([])
  const [groups, setGroups] = useState<AlertNotifyGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [editModal, setEditModal] = useState<{ open: boolean; rule?: AlertRule }>({ open: false })
  const [form] = Form.useForm()

  const fetchRules = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await alertAPI.rules.list()
      setRules(resp.data || [])
    } catch { message.error(t('loadRulesFailed', '加载规则失败')) }
    finally { setLoading(false) }
  }, [t])

  const fetchGroups = useCallback(async () => {
    try {
      const resp = await alertAPI.groups.list()
      setGroups(resp.data || [])
    } catch { /* ignore */ }
  }, [])

  useEffect(() => { fetchRules(); fetchGroups() }, [fetchRules, fetchGroups])

  const handleSync = async () => {
    setSyncing(true)
    try {
      // 从 Prometheus 获取告警规则（通过 Grafana datasource proxy）
      const result = await monitorAPI.grafana.getPrometheusRules()
      const groupsRes = result?.data?.groups || []

      // 将 Prometheus 规则格式展平并映射为同步格式
      const mapped: Array<{
        grafana_uid: string; name: string; rule_group: string;
        folder_title: string; state: string; expression: string
      }> = []

      for (const group of groupsRes) {
        for (const rule of group.rules || []) {
          if (rule.type !== 'alerting') continue
          mapped.push({
            // 用 "规则组名:规则名" 作为唯一标识（Prometheus 规则没有 UID）
            grafana_uid: `prom:${group.name}:${rule.name}`,
            name: rule.name,
            rule_group: group.name,
            folder_title: group.file || '',
            state: rule.state || '',
            expression: rule.query || '',
          })
        }
      }

      if (mapped.length === 0) {
        message.warning(t('noPrometheusRules', '未从 Prometheus 获取到告警规则'))
        return
      }

      const resp = await alertAPI.rules.sync(mapped)
      message.success(resp.message)
      fetchRules()
    } catch {
      message.error(t('syncPrometheusFailed', '同步 Prometheus 规则失败'))
    } finally {
      setSyncing(false)
    }
  }

  const handleEditRule = (rule: AlertRule) => {
    setEditModal({ open: true, rule })
    form.setFieldsValue({
      severity: rule.severity,
      category: rule.category,
      description: rule.description,
      condition: rule.condition,
      enabled: rule.enabled,
      alert_group_id: rule.alert_group_id || undefined,
    })
  }

  const handleSaveRule = async () => {
    if (!editModal.rule) return
    try {
      const values = await form.validateFields()
      await alertAPI.rules.update(editModal.rule.id, values)
      message.success(tc('updateSuccess', '更新成功'))
      setEditModal({ open: false })
      fetchRules()
    } catch {
      message.error(t('updateFailed', '更新失败'))
    }
  }

  const handleDeleteRule = async (id: number) => {
    try {
      await alertAPI.rules.delete(id)
      message.success(tc('deleteSuccess', '删除成功'))
      fetchRules()
    } catch {
      message.error(tc('deleteFailed', '删除失败'))
    }
  }

  const handleToggle = async (id: number, enabled: boolean) => {
    try {
      await alertAPI.rules.update(id, { enabled })
      message.success(enabled ? t('enabled', '已启用') : t('disabled', '已禁用'))
      fetchRules()
    } catch {
      message.error(tc('operationFailed', '操作失败'))
    }
  }

  const columns = [
    {
      title: tc('status', '状态'), key: 'enabled', width: 70,
      render: (_: unknown, r: AlertRule) => (
        <Switch size="small" checked={r.enabled} onChange={v => handleToggle(r.id, v)} />
      ),
    },
    { title: t('ruleName', '规则名称'), dataIndex: 'name', key: 'name', width: 250, ellipsis: true },
    {
      title: t('severity', '级别'), dataIndex: 'severity', key: 'severity', width: 80,
      render: (v: string) => <Tag color={severityConfig[v]?.color}>{t(severityConfig[v]?.key || v, severityConfig[v]?.text || v)}</Tag>,
    },
    {
      title: t('category', '分类'), dataIndex: 'category', key: 'category', width: 100,
      render: (v: string) => <Tag color={categoryConfig[v]?.color}>{t(categoryConfig[v]?.key || v, categoryConfig[v]?.text || v)}</Tag>,
    },
    { title: t('ruleGroup', '规则组'), dataIndex: 'rule_group', key: 'rule_group', width: 150, ellipsis: true },
    {
      title: t('alertStatus', '告警状态'), dataIndex: 'grafana_state', key: 'grafana_state', width: 120,
      render: (v: string) => {
        if (!v) return '-'
        const colorMap: Record<string, string> = { normal: 'green', alerting: 'red', pending: 'orange', inactive: 'default', 'no_data': 'default' }
        return <Tag color={colorMap[v] || 'default'}>{v}</Tag>
      },
    },
    {
      title: t('alertGroup', '报警组'), dataIndex: 'group', key: 'group', width: 120,
      render: (g: AlertNotifyGroup | undefined) => g ? <Tag color="blue">{g.name}</Tag> : <span style={{ color: '#999' }}>{t('notConfigured', '未配置')}</span>,
    },
    {
      title: t('lastSync', '最后同步'), dataIndex: 'synced_at', key: 'synced_at', width: 170,
      render: (v: string) => {
        if (!v) return '-'
        const d = new Date(v)
        if (isNaN(d.getTime())) return v
        const pad = (n: number) => n.toString().padStart(2, '0')
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
      },
    },
    {
      title: tc('action', '操作'), key: 'action', width: 140, fixed: 'right' as const,
      render: (_: unknown, r: AlertRule) => (
        <Space size={4}>
          {canEdit() && <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditRule(r)}>{t('configure', '配置')}</Button>}
          {canEdit() && <Popconfirm title={t('confirmDeleteRule', '确定删除此规则？')} onConfirm={() => handleDeleteRule(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>{tc('delete', '删除')}</Button>
          </Popconfirm>}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Button type="primary" icon={<SyncOutlined spin={syncing} />} onClick={handleSync} loading={syncing}>
            {t('syncPrometheusRules', '从 Prometheus 同步规则')}
          </Button>
        </Space>
        <span style={{ color: '#999', fontSize: 13 }}>{t('totalRules', '共 {{count}} 条规则', { count: rules.length })}</span>
      </div>

      <Table
        columns={columns}
        dataSource={rules}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1400 }}
        pagination={{ pageSize: 20, showTotal: tCount => tc('total', '共 {{count}} 条', { count: tCount }) }}
        expandable={{
          expandedRowRender: (record: AlertRule) => (
            <div style={{ padding: '8px 0' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '80px 1fr', gap: '8px 12px', fontSize: 13 }}>
                <span style={{ color: '#999', fontWeight: 500 }}>{t('triggerCondition', '触发条件:')}</span>
                <code style={{
                  background: '#f6f8fa', padding: '4px 8px', borderRadius: 4,
                  fontSize: 12, color: '#d63384', wordBreak: 'break-all',
                }}>
                  {record.expression || t('notConfigured', '未配置')}
                </code>
                {record.condition && (
                  <>
                    <span style={{ color: '#999', fontWeight: 500 }}>{t('conditionDescription', '条件说明:')}</span>
                    <span>{record.condition}</span>
                  </>
                )}
                {record.description && (
                  <>
                    <span style={{ color: '#999', fontWeight: 500 }}>{t('ruleDescriptionLabel', '规则描述:')}</span>
                    <span>{record.description}</span>
                  </>
                )}
                {record.folder_title && (
                  <>
                    <span style={{ color: '#999', fontWeight: 500 }}>{t('ruleSource', '规则来源:')}</span>
                    <span style={{ color: '#666' }}>{record.folder_title}</span>
                  </>
                )}
                {record.grafana_uid && (
                  <>
                    <span style={{ color: '#999', fontWeight: 500 }}>{t('ruleIdentifier', '规则标识:')}</span>
                    <span style={{ color: '#999', fontSize: 12 }}>{record.grafana_uid}</span>
                  </>
                )}
              </div>
            </div>
          ),
          rowExpandable: () => true,
        }}
      />

      <Modal title={t('configureAlertRule', '配置告警规则')} open={editModal.open} onOk={handleSaveRule} onCancel={() => setEditModal({ open: false })} width={550}>
        <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 6 }}>
          <b>{editModal.rule?.name}</b>
          {editModal.rule?.expression && <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>{t('expression', '表达式')}: {editModal.rule.expression}</div>}
        </div>
        <Form form={form} layout="vertical">
          <Form.Item name="severity" label={t('alertSeverity', '告警级别')} rules={[{ required: true }]}>
            <Select options={[{ label: t('severityCritical', '严重'), value: 'critical' }, { label: t('severityWarning', '警告'), value: 'warning' }, { label: t('severityInfo', '提醒'), value: 'info' }]} />
          </Form.Item>
          <Form.Item name="category" label={t('alertCategory', '告警分类')} rules={[{ required: true }]}>
            <Select options={Object.entries(categoryConfig).map(([k, v]) => ({ label: t(v.key, v.text), value: k }))} />
          </Form.Item>
          <Form.Item name="alert_group_id" label={t('relatedAlertGroup', '关联报警组')}>
            <Select allowClear placeholder={t('selectAlertGroup', '选择报警组')} options={groups.map(g => ({ label: g.name, value: g.id }))} />
          </Form.Item>
          <Form.Item name="condition" label={t('triggerConditionDesc', '触发条件说明')}>
            <Input.TextArea rows={2} placeholder={t('cpuConditionExample', '如：CPU 使用率持续 5 分钟超过 90%')} />
          </Form.Item>
          <Form.Item name="description" label={t('ruleDescription', '规则描述')}>
            <Input.TextArea rows={2} placeholder={t('ruleDescriptionPlaceholder', '规则描述信息')} />
          </Form.Item>
          <Form.Item name="enabled" label={tc('enable', '启用')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
