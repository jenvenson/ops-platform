// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import client from './client'

// 扫描任务类型
export interface SecurityScanTask {
  id: number
  name: string
  target_type: 'ip_list' | 'url'
  target: string
  scan_type: 'port' | 'host-vuln' | 'web' | 'all'
  status: 'pending' | 'running' | 'paused' | 'cancelled' | 'completed' | 'failed'
  progress: number
  total_ips: number
  scanned_ips: number
  message?: string
  high_risk: number
  medium_risk: number
  low_risk: number
  current_run_id?: number
  latest_run_id?: number
  nuclei_version?: string
  template_version?: string
  started_at?: string
  completed_at?: string
  created_at: string
  current_run?: SecurityScanRunDetail
  latest_run?: SecurityScanRunDetail
}

export interface SecurityScanRunDetail {
  id: number
  task_id: number
  status: string
  progress: number
  total_targets: number
  scanned_targets: number
  message?: string
  high_risk: number
  medium_risk: number
  low_risk: number
  started_at?: string
  completed_at?: string
  phase?: string
  config_snapshot?: Record<string, unknown>
  target_snapshot?: Record<string, unknown>
  summary_snapshot?: Record<string, unknown>
}

// 资产类型
export interface SecurityAsset {
  id: number
  task_id: number
  ip: string
  port: number
  protocol: string
  service_name: string
  version?: string
  os_info?: string
  banner?: string
}

export interface SecurityScanTarget {
  id: number
  run_id: number
  task_id: number
  parent_target_id?: number
  target_kind: string
  normalized_target: string
  target_url: string
  host: string
  port?: number
  scheme: string
  path: string
  service_name: string
  product_name: string
  version: string
  status: string
  discovery_source: string
  started_at?: string
  completed_at?: string
  metadata?: Record<string, unknown>
}

export interface SecurityScanEvidence {
  id: number
  run_id: number
  task_id: number
  target_id?: number
  evidence_type: string
  source_engine: string
  digest: string
  request_excerpt?: string
  response_excerpt?: string
  payload_excerpt?: string
  storage_ref?: string
  created_at: string
  metadata?: Record<string, unknown>
}

export interface SecurityScanFindingOccurrence {
  id: number
  run_id: number
  task_id: number
  target_id?: number
  finding_key: string
  finding_family: string
  finding_source: string
  severity: string
  confidence: string
  match_mode: string
  primary_cve_id: string
  title: string
  status: string
  verification_status: string
  evidence_count: number
  first_seen_at?: string
  last_seen_at?: string
  metadata?: Record<string, unknown>
  evidence_id?: number
  target?: SecurityScanTarget
}

export interface SecurityVulnerabilityDetailResponse {
  vulnerability: SecurityVulnerability
  occurrences: SecurityScanFindingOccurrence[]
  evidences: SecurityScanEvidence[]
}

// 漏洞类型
export interface SecurityVulnerability {
  id: number
  task_id: number
  first_task_id?: number
  last_task_id?: number
  source_vuln_id?: number
  confirmed_vuln_id?: number
  asset_id?: number
  ip: string
  port: number
  protocol?: string
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info'
  cve_id?: string
  cnvd_id?: string
  cnnvd_id?: string
  cncve_id?: string
  vuln_type?: string
  title: string
  description?: string
  solution?: string
  matched_on?: string
  exploit_prereq?: string
  cvss_score: number
  cvss_vector?: string
  payload?: string
  request?: string
  response?: string
  reference_url?: string
  vuln_url?: string
  scanner?: string
  template_id?: string
  scan_method?: string
  finding_source?: 'web-template' | 'web-rule' | 'host-template' | 'host-version-match' | 'host-manual-confirmed' | 'asset-inventory' | string
  finding_family?: 'vulnerability' | 'inventory' | string
  confidence?: 'high' | 'medium' | 'low' | string
  primary_cve_id?: string
  vuln_db_id?: number
  match_mode?: 'template' | 'rule' | 'version-range' | 'fuzzy-product' | 'inventory' | 'manual-review' | string
  risk_category?: 'inventory' | 'cve_risk' | 'config_risk' | 'generic_risk'
  display_group?: 'web_vuln' | 'host_vuln' | 'inventory' | 'generic_discovery' | string
  knowledge?: {
    id: number
    title: string
    severity: string
    cvss_score: number
    cnvd_id?: string
    cnnvd_id?: string
    cncve_id?: string
    has_reference?: boolean
  } | null
  priority?: string
  false_positive?: boolean
  verification_status?: 'pending' | 'needs-test' | 'confirmed' | 'rejected'
  verification_note?: string
  verified_by?: number
  verified_at?: string
  review_status?: 'pending' | 'needs-test' | 'confirmed' | 'rejected'
  review_note?: string
  reviewed_by?: number
  reviewed_at?: string
  status: 'open' | 'acknowledged' | 'fixed' | 'ignored'
  created_at: string
}

export interface UpdateVulnerabilityVerificationRequest {
  verification_status: 'pending' | 'needs-test' | 'confirmed' | 'rejected'
  verification_note?: string
}

export type ReviewVulnerabilityCandidateRequest = UpdateVulnerabilityVerificationRequest

export interface VulnDBSyncTask {
  id: number
  source: string
  status: 'running' | 'completed' | 'failed' | string
  total_count: number
  success_count: number
  fail_count: number
  start_time: string
  end_time?: string
  error_message?: string
  created_at?: string
}

export interface SecurityVulnerabilityFilters {
  severity?: string
  status?: string
  ip?: string
  risk_category?: string
  finding_source?: SecurityVulnerability['finding_source']
  finding_family?: SecurityVulnerability['finding_family']
  confidence?: SecurityVulnerability['confidence']
  has_knowledge?: 'true' | 'false'
  match_mode?: SecurityVulnerability['match_mode']
  page?: number
  page_size?: number
}

// 统计信息
export interface SecurityStatistics {
  total_tasks: number
  running_tasks: number
  completed_tasks: number
  total_assets: number
  total_vulnerabilities: number
  high_risk_count: number
  medium_risk_count: number
  low_risk_count: number
}

// 漏洞知识库记录（用于 VulnDB 页面）
export interface VulnDBRecord {
  id: number
  cve_id: string
  cnvd_id: string
  cnnvd_id: string
  cncve_id?: string
  title: string
  description?: string
  vuln_type?: string
  severity: string
  cvss_score: number
  cvss_vector?: string
  solution?: string
  patch_url?: string
  workaround?: string
  references?: string
  cwe_id?: string
  tags?: string
  source?: string
  last_updated: string
  created_at: string
}

// 漏洞知识库统计
export interface VulnDBStats {
  total: number
  critical: number
  high: number
  medium: number
  low: number
}

// 创建任务请求
export interface CreateTaskRequest {
  name: string
  target_type: 'ip_list' | 'url'
  target: string
  scan_type?: 'port' | 'host-vuln' | 'web' | 'all'
  web_scan_profile?: 'standard' | 'deep'
  web_scan_options?: string
  discovery_mode?: 'none' | 'browser'
  auth_mode?: 'none' | 'cookie' | 'bearer' | 'basic' | 'login-form' | 'login-token' | 'advanced'
  auth_credential?: string
  auth_header?: string
  auth_flow?: Record<string, unknown>
  login_url?: string
  login_method?: 'POST' | 'GET'
  login_content_type?: 'form' | 'json'
  username?: string
  password?: string
  username_field?: string
  password_field?: string
  token_field?: string
}

export interface GenerateAuthFlowRequest {
  preset?: 'auto' | 'token-json' | 'token-form' | 'custom-oauth2'
  target_url: string
  login_url?: string
  content_type?: 'form' | 'json'
  token_path?: string
  session_header?: string
  session_prefix?: string
  extra_headers?: Array<{ name: string; value: string }>
}

export interface GenerateAuthFlowResponse {
  preset: string
  auth_flow: Record<string, unknown>
  preview: string
}

// ==================== 资产中心类型 ====================

// 资产类型（新增独立资产库）
export interface Asset {
  id: number
  ip: string
  port: number
  protocol: string
  service_name: string
  version?: string
  os_info?: string
  banner?: string
  asset_type: 'server' | 'network' | 'web' | 'database' | 'other'
  asset_group?: string
  tags?: string
  importance?: 'critical' | 'high' | 'medium' | 'low'
  owner?: string
  department?: string
  status: 'online' | 'offline' | 'unknown'
  first_seen?: string
  last_seen?: string
  created_at: string
  updated_at?: string
}

// 资产统计
export interface AssetStats {
  total: number
  online: number
  offline: number
  unknown: number
  by_type: { type: string; count: number }[]
  by_importance: { importance: string; count: number }[]
}

// 资产关联漏洞统计
export interface AssetVulnCount {
  critical: number
  high: number
  medium: number
  low: number
  total: number
}

// 创建资产请求
export interface CreateAssetRequest {
  ip: string
  port?: number
  protocol?: string
  service_name?: string
  version?: string
  os_info?: string
  banner?: string
  asset_type?: string
  asset_group?: string
  tags?: string
  importance?: string
  owner?: string
  department?: string
  status?: string
}

// ==================== 漏洞工单类型 ====================

// 漏洞工单类型
export interface VulnTicket {
  id: number
  vuln_id: number
  vuln_title: string
  assignee: number
  assignee_name?: string
  department?: string
  status: 'open' | 'processing' | 'fixed' | 'closed' | 'rejected'
  priority?: 'high' | 'medium' | 'low'
  due_date?: string
  notes?: string
  comments?: string
  created_by: number
  created_by_name?: string
  resolved_at?: string
  created_at: string
  updated_at?: string
}

// 创建工单请求
export interface CreateTicketRequest {
  vuln_id: number
  assignee?: number
  priority?: string
  due_date?: string
  notes?: string
}

// 更新工单请求
export interface UpdateTicketRequest {
  assignee?: number
  assignee_name?: string
  priority?: string
  status?: string
  due_date?: string
  notes?: string
  comments?: string
}

// 指派工单请求
export interface AssignTicketRequest {
  assignee: number
  assignee_name: string
  department?: string
  priority?: string
}

// 分页响应类型
export interface PaginatedResponse<T> {
  total: number
  page: number
  page_size: number
  total_pages: number
  data: T[]
}

// 安全扫描 API
export const securityAPI = {
  // 获取统计信息
  getStatistics: () =>
    client.get<SecurityStatistics>('/security/statistics'),

  // 获取任务列表（分页）
  getTasks: (params?: {
    status?: string
    scan_type?: 'port' | 'host-vuln' | 'web' | 'all'
    task_group?: 'vuln' | 'discovery'
    page?: number
    page_size?: number
  }) =>
    client.get<PaginatedResponse<SecurityScanTask>>('/security/tasks', { params }),

  // 获取任务详情
  getTask: (id: number) =>
    client.get<SecurityScanTask>(`/security/tasks/${id}`),

  // 创建扫描任务
  createTask: (data: CreateTaskRequest) =>
    client.post<SecurityScanTask>('/security/tasks', data),

  generateAuthFlow: (data: GenerateAuthFlowRequest) =>
    client.post<GenerateAuthFlowResponse>('/security/auth-flow/generate', data),

  // 删除任务
  deleteTask: (id: number) =>
    client.delete(`/security/tasks/${id}`),

  // 获取任务的资产列表
  getTaskAssets: (id: number) =>
    client.get<SecurityAsset[]>(`/security/tasks/${id}/assets`),

  // 获取任务的新模型目标列表
  getTaskTargets: (id: number) =>
    client.get<SecurityScanTarget[]>(`/security/tasks/${id}/targets`),

  getTaskOccurrences: (id: number) =>
    client.get<SecurityScanFindingOccurrence[]>(`/security/tasks/${id}/occurrences`),

  getTaskEvidences: (id: number) =>
    client.get<SecurityScanEvidence[]>(`/security/tasks/${id}/evidences`),

  // 获取任务的漏洞列表
  getTaskVulnerabilities: (id: number, params?: SecurityVulnerabilityFilters) =>
    client.get<SecurityVulnerability[]>(`/security/tasks/${id}/vulnerabilities`, { params }),

  // 获取漏洞列表（分页）
  getVulnerabilities: (params?: SecurityVulnerabilityFilters) =>
    client.get<PaginatedResponse<SecurityVulnerability>>('/security/vulnerabilities', { params }),

  // 获取漏洞详情
  getVulnerability: (id: number) =>
    client.get<SecurityVulnerability>(`/security/vulnerabilities/${id}`),

  getVulnerabilityDetail: (id: number) =>
    client.get<SecurityVulnerabilityDetailResponse>(`/security/vulnerabilities/${id}/detail`),

  // 更新漏洞状态
  updateVulnerabilityStatus: (id: number, status: string) =>
    client.put(`/security/vulnerabilities/${id}/status`, { status }),

  // 更新待验证结果的验证状态
  updateVulnerabilityVerification: (id: number, data: UpdateVulnerabilityVerificationRequest) =>
    client.put<SecurityVulnerability>(`/security/vulnerabilities/${id}/review`, data),

  // 兼容旧调用
  reviewVulnerabilityCandidate: (id: number, data: ReviewVulnerabilityCandidateRequest) =>
    client.put<SecurityVulnerability>(`/security/vulnerabilities/${id}/review`, data),

  // 删除漏洞
  deleteVulnerability: (id: number) =>
    client.delete(`/security/vulnerabilities/${id}`),

  // 导出报告（支持 HTML/JSON/CSV）
  exportReport: (id: number, format: 'html' | 'json' | 'csv' = 'html') =>
    client.get(`/security/tasks/${id}/export`, {
      params: { format },
      responseType: 'blob',
    }),

  // 获取漏洞库列表（分页）
  getVulnDBList: (params?: { keyword?: string; severity?: string; vuln_type?: string; page?: number; page_size?: number; limit?: number }) =>
    client.get<PaginatedResponse<VulnDBRecord>>('/security/vuln-db/list', { params }),

  // 获取漏洞库统计
  getVulnStats: () =>
    client.get<VulnDBStats>('/security/vuln-db/stats'),

  // 获取漏洞库统计（新接口）
  getVulnDBStats: () =>
    client.get<{ total: number; critical: number; high: number; medium: number; low: number; this_week: number }>('/security/vuln-db/stats'),

  // 搜索漏洞库
  searchVulnDB: (params: { keyword?: string; severity?: string; vuln_type?: string; page?: number; page_size?: number }) =>
    client.get<{ data: VulnDBRecord[]; total: number; page: number; page_size: number }>('/security/vuln-db/search', { params }),

  // 根据 CVE ID 获取漏洞详情
  getVulnByCVE: (cveId: string) =>
    client.get<VulnDBRecord>(`/security/vuln-db/${cveId}`),

  // 服务漏洞匹配
  matchServiceVulns: (data: { service_name: string; product_name?: string; version?: string }) =>
    client.post<{ service: string; version: string; total: number; vulns: any[] }>('/security/vuln-db/match-service', data),

  // 获取同步任务历史
  getSyncTasks: (limit?: number) =>
    client.get<{ tasks: VulnDBSyncTask[]; last_sync_time: string; stats: any }>('/security/vuln-db/sync-tasks', { params: { limit } }),

  // 触发全量 NVD 同步
  syncNVDFull: () =>
    client.post<{ message: string }>('/security/vuln-db/sync-nvd-full', {}),

  // 导入漏洞数据
  importVulnerabilities: (csvData: string) =>
    client.post<{ inserted: number }>('/security/vuln-db/import', { csv_data: csvData }),

  // 从 NVD API 同步漏洞数据
  syncNVD: () =>
    client.post<{ message: string; mode: string; poll_hint: string }>('/security/vuln-db/sync-nvd', {}),

  // 导入 CNVD 数据
  importCNVD: (csvData: string) =>
    client.post<{ inserted: number; updated: number }>('/security/vuln-db/import-cnvd', { csv_data: csvData }),

  // 导入 CNNVD 数据
  importCNNVD: (csvData: string) =>
    client.post<{ inserted: number; updated: number }>('/security/vuln-db/import-cnnvd', { csv_data: csvData }),

  // ==================== 资产中心 API ====================

  getAssets: (params?: {
    page?: number
    page_size?: number
    ip?: string
    asset_type?: string
    status?: string
    importance?: string
    asset_group?: string
    owner?: string
    keyword?: string
  }) => client.get<{ total: number; page: number; page_size: number; total_pages: number; data: Asset[] }>('/security/assets', { params }),

  getAsset: (id: number) =>
    client.get<{ asset: Asset; vuln_count: AssetVulnCount }>(`/security/assets/${id}`),

  createAsset: (data: CreateAssetRequest) =>
    client.post<Asset>('/security/assets', data),

  updateAsset: (id: number, data: CreateAssetRequest) =>
    client.put<Asset>(`/security/assets/${id}`, data),

  deleteAsset: (id: number) =>
    client.delete(`/security/assets/${id}`),

  getAssetStats: () =>
    client.get<AssetStats>('/security/assets/stats'),

  // ==================== 漏洞工单 API ====================

  getTickets: (params?: {
    page?: number
    page_size?: number
    status?: string
    priority?: string
    assignee?: number
    department?: string
  }) => client.get<{ total: number; page: number; page_size: number; total_pages: number; data: VulnTicket[] }>('/security/tickets', { params }),

  getTicket: (id: number) =>
    client.get<{ ticket: VulnTicket; vuln?: SecurityVulnerability }>(`/security/tickets/${id}`),

  createTicket: (data: CreateTicketRequest) =>
    client.post<VulnTicket>('/security/tickets', data),

  updateTicket: (id: number, data: UpdateTicketRequest) =>
    client.put<VulnTicket>(`/security/tickets/${id}`, data),

  deleteTicket: (id: number) =>
    client.delete(`/security/tickets/${id}`),

  assignTicket: (id: number, data: AssignTicketRequest) =>
    client.post<VulnTicket>(`/security/tickets/${id}/assign`, data),

  closeTicket: (id: number, status: string, comments?: string) =>
    client.post<VulnTicket>(`/security/tickets/${id}/close`, { status, comments }),
}