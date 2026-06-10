// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import client from './client'

export interface FIMPolicy {
  id: number
  name: string
  description?: string
  enabled: boolean
  severity: 'critical' | 'high' | 'warning' | 'medium' | 'low' | 'info'
  notify_channels?: string
  scan_interval_sec: number
  hash_mode: 'off' | 'changed_only' | 'full'
  compare_mode: 'baseline' | 'last_snapshot'
  created_by?: string
  updated_by?: string
  created_at: string
  updated_at: string
}

export interface FIMPolicyTarget {
  id: number
  policy_id: number
  server_id: number
  server_name?: string
  server_ip?: string
  enabled: boolean
  last_scan_at?: string
  last_scan_status?: string
}

export interface FIMWatchPath {
  id: number
  policy_id: number
  path: string
  scan_mode: 'full_hash' | 'presence_only'
  recursive: boolean
  max_depth: number
  file_glob?: string
  exclude_glob?: string
  hash_on_match_only: boolean
  created_at: string
  updated_at: string
}

export interface FIMSnapshot {
  id: number
  policy_id: number
  server_id: number
  policy_name?: string
  server_name?: string
  server_ip?: string
  origin_type?: 'baseline' | 'scheduled' | 'manual'
  snapshot_type: 'baseline' | 'scheduled' | 'manual'
  status: 'running' | 'success' | 'failed'
  operator?: string
  started_at: string
  finished_at?: string
  entry_count: number
  error_message?: string
  created_at: string
}

export interface FIMDiffEvent {
  id: number
  policy_id: number
  server_id: number
  policy_name?: string
  server_name?: string
  server_ip?: string
  path: string
  event_type: 'create' | 'delete' | 'modify' | 'chmod' | 'chown' | 'rename'
  severity: string
  occurred_at: string
  old_value_json?: string
  new_value_json?: string
}

export interface FIMAlert {
  id: number
  diff_event_id: number
  policy_id: number
  server_id: number
  path?: string
  event_type?: 'create' | 'delete' | 'modify' | 'chmod' | 'chown' | 'rename'
  policy_name?: string
  server_name?: string
  server_ip?: string
  title: string
  summary?: string
  severity: string
  status: 'open' | 'acknowledged' | 'resolved' | 'closed'
  occurrence_count?: number
  assignee?: string
  first_seen_at: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
  total_pages?: number
}

export const securityFIMAPI = {
  getPolicies: (params?: Record<string, unknown>) =>
    client.get<PaginatedResponse<FIMPolicy>>('/security/fim/policies', { params }),
  createPolicy: (data: Partial<FIMPolicy>) =>
    client.post<FIMPolicy>('/security/fim/policies', data),
  updatePolicy: (id: number, data: Partial<FIMPolicy>) =>
    client.put<FIMPolicy>(`/security/fim/policies/${id}`, data),
  deletePolicy: (id: number) =>
    client.delete(`/security/fim/policies/${id}`),
  clearPolicyHistory: (id: number) =>
    client.post(`/security/fim/policies/${id}/clear-history`, {}),
  enablePolicy: (id: number) =>
    client.post(`/security/fim/policies/${id}/enable`, {}),
  disablePolicy: (id: number) =>
    client.post(`/security/fim/policies/${id}/disable`, {}),
  getTargets: (policyId: number) =>
    client.get<{ data: FIMPolicyTarget[] }>(`/security/fim/policies/${policyId}/targets`),
  addTargets: (policyId: number, server_ids: number[]) =>
    client.post(`/security/fim/policies/${policyId}/targets`, { server_ids }),
  deleteTarget: (policyId: number, targetId: number) =>
    client.delete(`/security/fim/policies/${policyId}/targets/${targetId}`),
  getWatchPaths: (policyId: number) =>
    client.get<{ data: FIMWatchPath[] }>(`/security/fim/policies/${policyId}/watch-paths`),
  createWatchPath: (policyId: number, data: Partial<FIMWatchPath>) =>
    client.post<FIMWatchPath>(`/security/fim/policies/${policyId}/watch-paths`, data),
  updateWatchPath: (id: number, data: Partial<FIMWatchPath>) =>
    client.put<FIMWatchPath>(`/security/fim/watch-paths/${id}`, data),
  deleteWatchPath: (id: number) =>
    client.delete(`/security/fim/watch-paths/${id}`),
  buildBaseline: (policyId: number, server_id: number) =>
    client.post(`/security/fim/policies/${policyId}/baselines/build`, { server_id }),
  runScan: (policyId: number, server_id: number, scan_type = 'manual') =>
    client.post(`/security/fim/policies/${policyId}/scan`, { server_id, scan_type }),
  getSnapshots: (params?: Record<string, unknown>) =>
    client.get<PaginatedResponse<FIMSnapshot>>('/security/fim/snapshots', { params }),
  getSnapshotDetail: (id: number) =>
    client.get<FIMSnapshot>(`/security/fim/snapshots/${id}`),
  activateBaseline: (id: number) =>
    client.post(`/security/fim/snapshots/${id}/activate-baseline`, {}),
  getEvents: (params?: Record<string, unknown>) =>
    client.get<PaginatedResponse<FIMDiffEvent>>('/security/fim/events', { params }),
  deleteEvent: (id: number) =>
    client.delete(`/security/fim/events/${id}`),
  getAlerts: (params?: Record<string, unknown>) =>
    client.get<PaginatedResponse<FIMAlert>>('/security/fim/alerts', { params }),
  getAlertDetail: (id: number) =>
    client.get<FIMAlert>(`/security/fim/alerts/${id}`),
  deleteAlert: (id: number) =>
    client.delete(`/security/fim/alerts/${id}`),
  ackAlert: (id: number, comment = '') =>
    client.post(`/security/fim/alerts/${id}/ack`, { comment }),
  resolveAlert: (id: number, comment = '') =>
    client.post(`/security/fim/alerts/${id}/resolve`, { comment }),
  closeAlert: (id: number, comment = '') =>
    client.post(`/security/fim/alerts/${id}/close`, { comment }),
}