// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import apiClient from './client.js'

// ========== 类型定义 ==========

export interface AlertRule {
  id: number
  grafana_uid: string
  name: string
  rule_group: string
  folder_title: string
  severity: string       // critical/warning/info
  category: string       // disk/memory/cpu/instance/network/load/other
  description: string
  expression: string
  condition: string
  enabled: boolean
  alert_group_id?: number
  notify_channels: string
  synced_at?: string
  grafana_state: string
  created_at: string
  updated_at: string
  group?: AlertNotifyGroup
}

export interface AlertContact {
  id: number
  name: string
  email: string
  phone: string
  dingtalk: string
  wechat: string
  created_at: string
  updated_at: string
  groups?: AlertNotifyGroup[]
}

export interface AlertNotifyGroup {
  id: number
  name: string
  description: string
  enabled: boolean
  created_at: string
  updated_at: string
  contacts?: AlertContact[]
}

export interface NotifyChannel {
  id: number
  name: string
  type: string            // dingtalk/wechat/email
  webhook_url: string
  secret: string
  smtp_host: string
  smtp_port: number
  smtp_user: string
  smtp_pass: string
  email_from: string
  enabled: boolean
  description: string
  created_at: string
  updated_at: string
}

export interface AlertEvent {
  id: number
  rule_id?: number
  rule_name: string
  severity: string
  category: string
  content: string
  source: string
  status: string          // firing/acknowledged/resolved/closed
  fired_at: string
  acked_at?: string
  resolved_at?: string
  closed_at?: string
  acked_by?: string
  closed_by?: string
  handle_type?: string    // ticket/auto/manual
  handle_note?: string
  labels?: string
  fingerprint?: string
  notify_status: string
  created_at: string
  updated_at: string
  rule?: AlertRule
  logs?: AlertEventLog[]
}

export interface AlertEventLog {
  id: number
  event_id: number
  action: string          // created/acked/resolved/closed/notified/note
  operator: string
  content: string
  created_at: string
}

export interface AlertTemplate {
  id: number
  name: string
  type: string            // dingtalk/wechat/email
  scene: string           // firing/resolved/test
  title_tpl: string
  content_tpl: string
  is_default: boolean
  enabled: boolean
  description: string
  created_at: string
  updated_at: string
}

export interface EventStats {
  status_stats: Array<{ status: string; count: number }>
  severity_stats: Array<{ severity: string; count: number }>
  today_count: number
  firing_count: number
}

// ========== API ==========

export const alertAPI = {
  // 告警规则
  rules: {
    list: (params?: Record<string, string>): Promise<{ data: AlertRule[] }> => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return apiClient.get(`/alert/rules${query}`)
    },
    create: (data: Partial<AlertRule>): Promise<{ data: AlertRule }> =>
      apiClient.post('/alert/rules', data),
    update: (id: number, data: Partial<AlertRule>): Promise<{ data: AlertRule }> =>
      apiClient.put(`/alert/rules/${id}`, data),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/rules/${id}`),
    sync: (rules: Array<{ grafana_uid: string; name: string; rule_group: string; folder_title: string; state: string; expression: string }>): Promise<{ message: string; created: number; synced: number }> =>
      apiClient.post('/alert/rules/sync', { rules }),
  },

  // 联系人
  contacts: {
    list: (): Promise<{ data: AlertContact[] }> =>
      apiClient.get('/alert/contacts'),
    create: (data: Partial<AlertContact>): Promise<{ data: AlertContact }> =>
      apiClient.post('/alert/contacts', data),
    update: (id: number, data: Partial<AlertContact>): Promise<{ data: AlertContact }> =>
      apiClient.put(`/alert/contacts/${id}`, data),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/contacts/${id}`),
  },

  // 报警组
  groups: {
    list: (): Promise<{ data: AlertNotifyGroup[] }> =>
      apiClient.get('/alert/groups'),
    create: (data: { name: string; description: string; enabled: boolean; contact_ids: number[] }): Promise<{ data: AlertNotifyGroup }> =>
      apiClient.post('/alert/groups', data),
    update: (id: number, data: { name: string; description: string; enabled: boolean; contact_ids: number[] }): Promise<{ data: AlertNotifyGroup }> =>
      apiClient.put(`/alert/groups/${id}`, data),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/groups/${id}`),
  },

  // 通知渠道
  channels: {
    list: (params?: Record<string, string>): Promise<{ data: NotifyChannel[] }> => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return apiClient.get(`/alert/channels${query}`)
    },
    create: (data: Partial<NotifyChannel>): Promise<{ data: NotifyChannel }> =>
      apiClient.post('/alert/channels', data),
    update: (id: number, data: Partial<NotifyChannel>): Promise<{ data: NotifyChannel }> =>
      apiClient.put(`/alert/channels/${id}`, data),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/channels/${id}`),
    test: (id: number): Promise<{ message: string }> =>
      apiClient.post(`/alert/channels/${id}/test`),
  },

  // 告警模板
  templates: {
    list: (params?: Record<string, string>): Promise<{ data: AlertTemplate[] }> => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return apiClient.get(`/alert/templates${query}`)
    },
    create: (data: Partial<AlertTemplate>): Promise<{ data: AlertTemplate }> =>
      apiClient.post('/alert/templates', data),
    update: (id: number, data: Partial<AlertTemplate>): Promise<{ data: AlertTemplate }> =>
      apiClient.put(`/alert/templates/${id}`, data),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/templates/${id}`),
    preview: (data: { title_tpl: string; content_tpl: string; type: string }): Promise<{ title: string; content: string; data: Record<string, string> }> =>
      apiClient.post('/alert/templates/preview', data),
    setDefault: (id: number): Promise<{ message: string }> =>
      apiClient.post(`/alert/templates/${id}/default`),
  },

  // 告警事件
  events: {
    list: (params?: Record<string, string>): Promise<{ data: AlertEvent[]; total: number; page: number; page_size: number }> => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return apiClient.get(`/alert/events${query}`)
    },
    get: (id: number): Promise<{ data: AlertEvent }> =>
      apiClient.get(`/alert/events/${id}`),
    ack: (id: number, data: { handle_type?: string; handle_note?: string }): Promise<void> =>
      apiClient.put(`/alert/events/${id}/ack`, data),
    close: (id: number, data: { note?: string }): Promise<void> =>
      apiClient.put(`/alert/events/${id}/close`, data),
    addNote: (id: number, content: string): Promise<{ data: AlertEventLog }> =>
      apiClient.post(`/alert/events/${id}/note`, { content }),
    getLogs: (id: number): Promise<{ data: AlertEventLog[] }> =>
      apiClient.get(`/alert/events/${id}/logs`),
    getStats: (): Promise<EventStats> =>
      apiClient.get('/alert/events/stats'),
    delete: (id: number): Promise<void> =>
      apiClient.delete(`/alert/events/${id}`),
  },
}