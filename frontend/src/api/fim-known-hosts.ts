// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import client from './client';

export interface KnownHost {
  id: number;
  hostname: string;
  port: number;
  key_type: string;
  public_key: string;
  fingerprint_sha256: string;
  server_id?: number;
  description?: string;
  tags?: string; // JSON string in database
  verification_status: string;
  verified_by?: string;
  verified_at?: string;
  added_by: string;
  added_at: string;
  last_used_at?: string;
  use_count: number;
  is_enabled: boolean;
}

export interface ConnectionLog {
  id: number;
  hostname: string;
  port: number;
  result: 'success' | 'key_not_found' | 'key_mismatch' | 'connection_failed';
  error_message?: string;
  presented_key_type?: string;
  presented_fingerprint?: string;
  expected_fingerprint?: string;
  attempted_at: string;
}

export interface HostHistory {
  id: number;
  action: 'added' | 'updated' | 'deleted' | 'key_changed';
  old_key_type?: string;
  old_fingerprint?: string;
  new_key_type?: string;
  new_fingerprint?: string;
  operated_by: string;
  operated_at: string;
  reason?: string;
}

export interface AddKnownHostRequest {
  hostname: string;
  port: number;
  key_type: string;
  public_key: string;
  server_id?: number;
  description?: string;
  tags?: string[];
}

export interface UpdateKnownHostRequest {
  description?: string;
  tags?: string[];
  is_enabled?: boolean;
  public_key?: string;
}

export const fimKnownHostsAPI = {
  // 获取已知主机列表
  getHosts: async (params?: { hostname?: string; status?: string; key_type?: string }): Promise<{ data: KnownHost[]; total: number }> => {
    const response: any = await client.get('/security/fim/known-hosts', { params });
    return response;
  },

  // 获取单个主机详情
  getHost: async (id: number): Promise<{ host: KnownHost; logs: ConnectionLog[]; history: HostHistory[] }> => {
    const response: any = await client.get(`/security/fim/known-hosts/${id}`);
    return response.data;
  },

  // 添加已知主机
  addHost: async (data: AddKnownHostRequest): Promise<KnownHost> => {
    const response: any = await client.post('/security/fim/known-hosts', data);
    return response.data;
  },

  // 更新已知主机
  updateHost: async (id: number, data: UpdateKnownHostRequest): Promise<KnownHost> => {
    const response: any = await client.put(`/security/fim/known-hosts/${id}`, data);
    return response.data;
  },

  // 删除已知主机
  deleteHost: async (id: number): Promise<{ success: boolean }> => {
    const response: any = await client.delete(`/security/fim/known-hosts/${id}`);
    return response.data;
  },

  // 批量添加主机
  batchAddHosts: async (hosts: AddKnownHostRequest[]): Promise<{ success: boolean; added: number; skipped: number; errors: string[] }> => {
    const response: any = await client.post('/security/fim/known-hosts/batch', { hosts });
    return response.data;
  },

  // 导入known_hosts文件
  importHosts: async (content: string): Promise<{ success: boolean; imported: number; skipped: number; errors: string[] }> => {
    const response: any = await client.post('/security/fim/known-hosts/import', { content });
    return response.data;
  },

  // 导出known_hosts文件
  exportHosts: async (): Promise<Blob> => {
    const response: any = await client.get('/security/fim/known-hosts/export', {
      responseType: 'blob',
    });
    return response.data;
  },

  // 获取连接日志
  getConnectionLogs: async (params?: { hostname?: string; result?: string; limit?: number }): Promise<{ data: ConnectionLog[]; total: number }> => {
    const response: any = await client.get('/security/fim/connection-logs', { params });
    return response.data;
  },
};

export default fimKnownHostsAPI;