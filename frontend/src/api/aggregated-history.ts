// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import apiClient from './client';

// 聚合历史相关API
export interface AggregatedHistory {
  id: number;
  project_name: string;
  environment: string;  // Tag名称
  status: string;
  progress: number;
  start_time?: string;  // 归档开始时间
  end_time?: string;    // 归档结束时间
  download_url?: string;
  operator: string;
  operator_name: string;  // 操作人姓名
  jenkins_job_name: string;
  jenkins_build_num?: number;
  jenkins_queue_id?: number;
  jenkins_console_url?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface AggregatedHistoryRequest {
  project_name: string;
  environment: string;
}

export interface AggregatedHistoryFile {
  name: string;
  url: string;
  size: number;
  timestamp: string;
}

// 聚合历史API
export const aggregatedHistoryAPI = {
  // 获取聚合历史列表
  getHistories: (params: {
    page?: number;
    limit?: number;
    project_name?: string;
    environment?: string;
    operator?: string;
    status?: string;
  }) => {
    return apiClient.get<{ data: AggregatedHistory[]; page: number; limit: number; total: number }>(
      '/cmdb/aggregated-histories',
      { params }
    );
  },

  // 获取单个聚合历史记录
  getHistory: (id: number) => {
    return apiClient.get<AggregatedHistory>(`/cmdb/aggregated-histories/${id}`);
  },

  // 删除聚合历史记录
  deleteHistory: (id: number) => {
    return apiClient.delete<{ success: boolean; message: string }>(`/cmdb/aggregated-histories/${id}`);
  },

  // 获取聚合历史状态（从Jenkins更新）
  getStatus: (id: number) => {
    return apiClient.get<{
      id: number;
      status: string;
      build_number?: number;
      phase?: string;
      jenkins_url?: string;
      console_url?: string;
      timestamp?: string;
      queue_id?: number;
      message?: string;
    }>(`/cmdb/aggregated-histories/${id}/status`);
  },

  // 获取控制台日志
  getConsoleLog: (id: number) => {
    return apiClient.get<{
      success: boolean;
      console_log: string;
      build_num: number;
      job_name: string;
      console_url: string;
    }>(`/cmdb/aggregated-histories/${id}/console-log`);
  },

  // 获取聚合文件列表
  getFiles: (id: number) => {
    return apiClient.get<{
      base_url: string;
      timestamp: string;
      files: AggregatedHistoryFile[];
      message?: string;
    }>(`/cmdb/aggregated-histories/${id}/files`);
  },

  // 获取所有不同的项目名称
  getUniqueProjectNames: () => {
    return apiClient.get<{ project_names: string[] }>(
      '/cmdb/aggregated-histories',
      { params: { distinct_projects: true } }
    );
  },

  // 获取所有不同的环境
  getUniqueEnvironments: () => {
    return apiClient.get<{ environments: string[] }>(
      '/cmdb/aggregated-histories',
      { params: { distinct_environments: true } }
    );
  },
};