// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import client from './client';

// Jenkins 视图作业类型
export interface JenkinsViewJob {
  name: string;
  app_name: string;
  url: string;
  job_url: string;
  color: string;
  exists: boolean;
}

// 视图复制请求类型
export interface ViewCopyRequest {
  source_view: string;
  target_view: string;
  jenkins_url?: string; // Jenkins地址
  tag_replacements?: Array<{
    old_pattern: string;
    new_pattern: string;
  }>; // Tag值替换规则
  job_name_replacements?: Array<{
    old_pattern: string;
    new_pattern: string;
  }>; // Job名称替换
}

// 视图复制结果类型
export interface ViewCopyResult {
  success: boolean;
  message: string;
  copied_jobs: string[];
  failed_jobs: string[];
  skipped_jobs: string[];
  approved_count: number;
  approval_note?: string;
}

// Jenkins API 接口
export const jenkinsAPI = {
  // 获取 Jenkins 视图下的 Jobs
  getViewJobs: (viewName: string, jenkinsUrl?: string) =>
    client.get<{ view_name: string; jobs: JenkinsViewJob[]; total: number }>(
      `/cmdb/jenkins/views/${encodeURIComponent(viewName)}/jobs`,
      { params: { jenkins_url: jenkinsUrl } }
    ),

  // 复制 Jenkins 视图（异步模式）
  copyView: (data: ViewCopyRequest) =>
    client.post<{ task_id: string; status: string; message: string }>('/cmdb/jenkins/copy-view?async=true', data),

  // 查询异步任务状态
  getTaskStatus: (taskId: string) =>
    client.get<{
      id: string;
      type: string;
      status: string;
      progress: number;
      total: number;
      result?: ViewCopyResult;
    }>(`/cmdb/tasks/${taskId}`),

  // 导入 Jenkins Jobs
  importJobs: (data: {
    view_name: string;
    project_id: number;
    env_id: number;
    job_names: string[];
    jenkins_url?: string;
    archive_job?: string
  }) =>
    client.post<{ created: number; skipped: number; errors: string[]; message: string }>('/cmdb/jenkins/import', data),

  // 删除单个 Job
  deleteJob: (jobName: string) =>
    client.delete<{ message: string }>(`/cmdb/jenkins/jobs/${encodeURIComponent(jobName)}`),

  // 批量删除 Jobs（可选同时删除视图，异步模式）
  batchDeleteJobs: (data: { view_name: string; job_names: string[]; delete_view: boolean }) =>
    client.post<{ task_id: string; status: string; message: string }>('/cmdb/jenkins/delete-jobs', data),

  // 删除视图（仅视图）
  deleteView: (viewName: string) =>
    client.delete<{ message: string }>(`/cmdb/jenkins/views/${encodeURIComponent(viewName)}`),

  // 创建 SSH 凭据
  createCredential: (data: { id: string; username: string; private_key: string; description?: string; passphrase?: string }) =>
    client.post<{ message: string }>('/cmdb/jenkins/credentials', data),
};