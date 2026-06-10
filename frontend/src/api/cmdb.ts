// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import apiClient from './client.js'

export interface Server {
  id: number
  hostname: string
  ip: string
  os?: string
  arch?: string
  status: 'online' | 'offline' | 'maintenance'
  ssh_port: number
  env_ids: string // 逗号分隔的环境ID
  // 健康检查状态字段
  last_heartbeat?: string
  projects?: Project[]
  project_ids?: number[] // 用于提交
  environment?: Environment
  created_at: string
  updated_at: string
}

export interface ServerCreateData {
  hostname: string
  ip: string
  os?: string
  arch?: string
  ssh_port: number
  project_ids: number[]
  env_ids: number[]
}

export interface ServerUpdateData {
  hostname?: string
  ip?: string
  os?: string
  arch?: string
  ssh_port?: number
  project_ids?: number[]
  env_ids?: number[]
  status?: string
}

export interface Environment {
  id: number
  name: string
  type: 'dev' | 'test' | 'prod'
  description?: string
  created_at: string
  updated_at: string
}

export interface Project {
  id: number
  name: string
  code: string
  description?: string
  created_at: string
  updated_at: string
}

export interface Application {
  id: number
  name: string
  code_repo?: string
  deploy_path?: string
  jenkins_job?: string
  jenkins_archive_job?: string
  env_id?: number
  project_id?: number  // 可能为 undefined，如果应用未设置所属项目
  environment?: Environment
  project?: Project
  created_at: string
  updated_at: string
}

export interface JenkinsViewJob {
  name: string
  app_name: string
  url: string
  job_url: string
  color: string
  exists: boolean
}

export interface PaginatedResponse<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export const cmdbAPI = {
  // 服务器
  getServers: (params?: { project_id?: number; env_id?: number; page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Server>>('/cmdb/servers', { params }),

  getServer: (id: number) =>
    apiClient.get<Server>(`/cmdb/servers/${id}`),

  createServer: (data: ServerCreateData) =>
    apiClient.post<Server>('/cmdb/servers', data),

  updateServer: (id: number, data: ServerUpdateData) =>
    apiClient.put<Server>(`/cmdb/servers/${id}`, data),

  deleteServer: (id: number) =>
    apiClient.delete<Server>(`/cmdb/servers/${id}`),

  // 环境
  getEnvironments: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Environment>>('/cmdb/environments', { params }),

  getEnvironment: (id: number) =>
    apiClient.get<Environment>(`/cmdb/environments/${id}`),

  createEnvironment: (data: Omit<Environment, 'id' | 'created_at' | 'updated_at'>) =>
    apiClient.post<Environment>('/cmdb/environments', data),

  updateEnvironment: (id: number, data: Partial<Environment>) =>
    apiClient.put<Environment>(`/cmdb/environments/${id}`, data),

  deleteEnvironment: (id: number) =>
    apiClient.delete<Environment>(`/cmdb/environments/${id}`),

  // 项目
  getProjects: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Project>>('/cmdb/projects', { params }),

  getProject: (id: number) =>
    apiClient.get<Project>(`/cmdb/projects/${id}`),

  createProject: (data: Omit<Project, 'id' | 'created_at' | 'updated_at'>) =>
    apiClient.post<Project>('/cmdb/projects', data),

  updateProject: (id: number, data: Partial<Project>) =>
    apiClient.put<Project>(`/cmdb/projects/${id}`, data),

  deleteProject: (id: number) =>
    apiClient.delete<Project>(`/cmdb/projects/${id}`),

  // 应用
  getApplications: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Application>>('/cmdb/applications', { params }),

  getApplication: (id: number) =>
    apiClient.get<Application>(`/cmdb/applications/${id}`),

  createApplication: (data: Omit<Application, 'id' | 'created_at' | 'updated_at'>) =>
    apiClient.post<Application>('/cmdb/applications', data),

  updateApplication: (id: number, data: Partial<Application>) =>
    apiClient.put<Application>(`/cmdb/applications/${id}`, data),

  deleteApplication: (id: number) =>
    apiClient.delete<Application>(`/cmdb/applications/${id}`),

  // Jenkins 集成
  getJenkinsViewJobs: (viewName: string, appNamePrefix?: string) =>
    apiClient.get<{ view_name: string; jobs: JenkinsViewJob[]; total: number; app_name_prefix?: string }>(
      `/cmdb/jenkins/views/${encodeURIComponent(viewName)}/jobs`,
      { params: appNamePrefix ? { app_name_prefix: appNamePrefix } : undefined }
    ),

  importJenkinsJobs: (data: { view_name: string; project_id: number; env_id: number; job_names: string[]; archive_job?: string; app_name_prefix?: string }) =>
    apiClient.post<{ created: number; skipped: number; errors: string[]; message: string }>('/cmdb/jenkins/import', data),

  // Jenkins 视图复制
  copyJenkinsView: (data: { source_view: string; target_view: string; tag_value?: string }) =>
    apiClient.post<{ success: boolean; message: string; copied_jobs: string[]; failed_jobs: string[] }>('/cmdb/jenkins/copy-view', data),
}

// 部署相关接口
export interface DeployTriggerRequest {
  app_id: number
  env_id: number
  deploy_type: 'all' | 'frontend' | 'backend'
}

export interface DeployTriggerResponse {
  success: boolean
  message: string
  deploy_id?: number
  task_id?: number
}

// 部署记录
export interface DeployRecord {
  id: number
  app_id: number
  app_name: string
  env_id: number
  env_name: string
  env_type: string
  project_code: string
  deploy_type: string
  jenkins_job: string
  jenkins_build_num: number
  jenkins_queue_id: number
  jenkins_console_url?: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'queued'
  error_message?: string
  start_time?: string
  end_time?: string
  duration: number
  triggered_by: string
  created_at: string
}

export const deployAPI = {
  // 触发应用部署
  triggerDeploy: (data: DeployTriggerRequest & { app_id: number }) =>
    apiClient.post<DeployTriggerResponse>(`/cmdb/applications/${data.app_id}/deploy`, data),

  // 获取部署历史列表
  getDeployRecords: (params?: { app_id?: number; app_name?: string; env_id?: number; status?: string; triggered_by?: string; start_time?: string; end_time?: string; page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<DeployRecord>>('/cmdb/deploy-records', { params }),

  // 获取单个部署记录
  getDeployRecord: (id: number) =>
    apiClient.get<DeployRecord>(`/cmdb/deploy-records/${id}`),

  // 获取部署状态（从 Jenkins 更新）
  getDeployStatus: (id: number) =>
    apiClient.get<{
      status: string
      build_number: number
      phase: string
      jenkins_url: string
      console_url: string
      timestamp: number
    }>(`/cmdb/deploy-records/${id}/status`),

  // 删除部署记录
  deleteDeployRecord: (id: number) =>
    apiClient.delete<{ message: string }>(`/cmdb/deploy-records/${id}`),

  // 触发应用归档
  triggerArchive: (data: { app_id: number; env_id: number; deploy_type: 'all' | 'frontend' | 'backend' }) =>
    apiClient.post<DeployTriggerResponse>(`/cmdb/applications/${data.app_id}/archive`, data),

  // 获取归档记录列表
  getArchiveRecords: (params?: { app_id?: number; app_name?: string; env_id?: number; page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<ArchiveRecord>>('/cmdb/archive-records', { params }),

  // 获取归档下载地址
  getArchiveDownloadUrl: (id: number) =>
    apiClient.get<{ download_url: string }>(`/cmdb/archive-records/${id}/download`),

  // 获取归档状态（从 Jenkins 更新）
  getArchiveStatus: (id: number) =>
    apiClient.get<{
      status: string
      build_number: number
      phase: string
      jenkins_url: string
      console_url: string
      download_url: string
      timestamp: number
    }>(`/cmdb/archive-records/${id}/status`),

  // 删除归档记录
  deleteArchiveRecord: (id: number) =>
    apiClient.delete<{ message: string }>(`/cmdb/archive-records/${id}`),

  // 获取归档文件列表
  getArchiveFiles: (id: number) =>
    apiClient.get<{
      base_url: string
      timestamp: string
      files: Array<{
        name: string
        url: string
        size: number
        timestamp: string
      }>
    }>(`/cmdb/archive-records/${id}/files`),
}

// 归档记录
export interface ArchiveRecord {
  id: number
  app_id: number
  app_name: string
  env_id: number
  env_name: string
  env_type: string
  deploy_type: string
  jenkins_job: string
  jenkins_build_num: number
  jenkins_queue_id: number
  jenkins_console_url?: string
  download_url?: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'queued'
  error_message?: string
  start_time?: string
  end_time?: string
  operator?: string
  created_at: string
}