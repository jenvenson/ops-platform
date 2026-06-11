// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import i18next from '../i18n'
import apiClient from './client'

export interface AuditListResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export interface AuditRetentionResult {
  retention_days: number
  access_affected: number
  operation_affected: number
  login_affected: number
}

export interface PlatformArchiveStats {
  access_total: number
  operation_total: number
  login_total: number
  total: number
  latest_archived?: string
}

export interface PlatformArchivedLog {
  id: number
  archive_type: 'access' | 'operation' | 'login'
  username: string
  real_name?: string
  role?: string
  title: string
  request_path: string
  request_method: string
  request_ip: string
  operation_status: 'success' | 'failed'
  status_code: number
  duration_ms: number
  error_message?: string
  occurred_at: string
  archived_at: string
}

export interface PlatformAccessLog {
  id: number
  trace_id: string
  user_id: number
  username: string
  real_name?: string
  role?: string
  menu_key?: string
  menu_title?: string
  page_path?: string
  request_path: string
  request_method: string
  request_ip: string
  user_agent?: string
  referer?: string
  status_code: number
  operation_status: 'success' | 'failed'
  duration_ms: number
  error_message?: string
  accessed_at: string
  created_at: string
}

export interface PlatformAuditLog {
  id: number
  trace_id: string
  user_id: number
  username: string
  real_name?: string
  role?: string
  module?: string
  resource_type?: string
  resource_id?: string
  resource_name?: string
  action?: string
  action_label?: string
  request_path: string
  request_method: string
  request_ip: string
  status_code: number
  operation_status: 'success' | 'failed'
  request_params_json?: string
  before_data_json?: string
  after_data_json?: string
  change_summary?: string
  error_message?: string
  duration_ms: number
  operated_at: string
  created_at: string
}

export interface PlatformLoginLog {
  id: number
  trace_id: string
  user_id: number
  username: string
  real_name?: string
  role?: string
  request_ip: string
  user_agent?: string
  request_path: string
  request_method: string
  status_code: number
  operation_status: 'success' | 'failed'
  login_type?: string
  error_message?: string
  duration_ms: number
  logged_in_at: string
  created_at: string
}

export const auditAPI = {
  getAccessLogs: (params?: Record<string, unknown>) =>
    apiClient.get<AuditListResponse<PlatformAccessLog>>('/admin/audit/access-logs', { params }),
  deleteAccessLog: (id: number) =>
    apiClient.delete(`/admin/audit/access-logs/${id}`),
  getOperationLogs: (params?: Record<string, unknown>) =>
    apiClient.get<AuditListResponse<PlatformAuditLog>>('/admin/audit/operation-logs', { params }),
  deleteOperationLog: (id: number) =>
    apiClient.delete(`/admin/audit/operation-logs/${id}`),
  getLoginLogs: (params?: Record<string, unknown>) =>
    apiClient.get<AuditListResponse<PlatformLoginLog>>('/admin/audit/login-logs', { params }),
  deleteLoginLog: (id: number) =>
    apiClient.delete(`/admin/audit/login-logs/${id}`),
  getArchiveStats: () =>
    apiClient.get<PlatformArchiveStats>('/admin/audit/archive-stats'),
  getArchivedLogs: (params?: Record<string, unknown>) =>
    apiClient.get<AuditListResponse<PlatformArchivedLog>>('/admin/audit/archive-logs', { params }),
  deleteArchivedLog: (archiveType: 'access' | 'operation' | 'login', id: number) =>
    apiClient.delete(`/admin/audit/archive-logs/${archiveType}/${id}`),
  archiveLogs: (retentionDays: number) =>
    apiClient.post<AuditRetentionResult>('/admin/audit/archive', { retention_days: retentionDays }),
  cleanupOnlineLogs: (retentionDays: number) =>
    apiClient.post<AuditRetentionResult>('/admin/audit/cleanup-online', { retention_days: retentionDays }),
  cleanupLogs: (retentionDays: number) =>
    apiClient.post<AuditRetentionResult>('/admin/audit/cleanup', { retention_days: retentionDays }),
  exportLogs: async (params: Record<string, string>) => {
    const query = new URLSearchParams(params)
    const token = localStorage.getItem('token')
    const response = await fetch(`/api/admin/audit/export?${query.toString()}`, {
      method: 'GET',
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    })
    if (!response.ok) {
      let errorMessage = i18next.t('common:exportLogFailed', '导出日志失败')
      const contentType = response.headers.get('content-type') || ''
      if (contentType.includes('application/json')) {
        try {
          const payload = await response.json() as { error?: string; message?: string }
          if (payload.error) {
            errorMessage = payload.error
          } else if (payload.message) {
            errorMessage = payload.message
          }
        } catch {
          // ignore parse error and fallback to default message
        }
      }
      throw new Error(errorMessage)
    }
    const disposition = response.headers.get('content-disposition') || ''
    const matched = disposition.match(/filename=\"?([^\";]+)\"?/)
    const filename = matched?.[1] || `platform-audit-${Date.now()}.csv`
    const blob = await response.blob()
    return { blob, filename }
  },
}