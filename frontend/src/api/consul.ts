// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import client from './client'

// Consul 配置类型
export interface ConsulConfig {
  id: number
  name: string
  address: string
  datacenter: string
  token?: string
  username?: string
  password?: string
  is_default: boolean
  created_at: string
  updated_at: string
}

// KV 键值项
export interface KVItem {
  key: string
  value?: string
  flags?: number
  create_index?: number
  modify_index?: number
}

// KV 树节点
export interface KVNode {
  key: string
  name: string
  is_dir: boolean
  children?: KVNode[]
}

// 替换规则
export interface ReplaceRule {
  id: number
  name: string
  description?: string
  source_type: 'text' | 'regex'
  old_value: string
  new_value: string
  enabled: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

// 复制请求
export interface CopyRequest {
  config_id: number
  source_key: string
  target_key: string
  tag_replacements?: ReplacePair[]
  server_replacements?: ReplacePair[]
  branch_replacements?: ReplacePair[]
  submodule_branch_replacements?: ReplacePair[]
  replace_rules?: RuleItem[]
  recursive?: boolean
}

// 规则项
export interface RuleItem {
  type: 'text' | 'regex'
  old_value: string
  new_value: string
}

// 替换对（原模式 → 新模式）
export interface ReplacePair {
  old_pattern: string
  new_pattern: string
}

// 复制结果
export interface CopyResult {
  success: number
  failed: number
  total: number
  copied_keys: string[]
  failed_keys: string[]
  errors: string[]
}

// 批量复制请求
export interface BatchCopyRequest {
  config_id?: number
  source_prefix: string
  target_prefix: string
  recursive?: boolean
  replace_rules?: RuleItem[]
  tag_replacements?: ReplacePair[]
  server_replacements?: ReplacePair[]
  branch_replacements?: ReplacePair[]
  submodule_branch_replacements?: ReplacePair[]
}

// 批量复制所有项目请求
export interface BatchCopyAllProjectsRequest {
  config_id?: number
  source_suffix: string
  target_suffix: string
  replace_in_place?: boolean
  projects?: string[]
  replace_rules?: RuleItem[]
  tag_replacements?: ReplacePair[]
  server_replacements?: ReplacePair[]
  branch_replacements?: ReplacePair[]
  submodule_branch_replacements?: ReplacePair[]
  recursive?: boolean
}

// 批量复制结果
export interface BatchCopyResult {
  success: number
  failed: number
  total: number
  copied_keys: string[]
  failed_keys: string[]
  errors: string[]
  elapsed_time: string
}

// 操作记录
export interface CopyOperation {
  id: number
  config_id: number
  source_key: string
  target_key: string
  rules_applied: string
  status: string
  message: string
  operator: string
  created_at: string
}

// Consul API
export const consulAPI = {
  // 配置管理
  getConfigs: () =>
    client.get<ConsulConfig[]>('/consul/configs'),

  getProjects: (configId?: number) =>
    client.get<{ projects: string[]; total: number }>('/consul/projects', { params: { config_id: configId } }),

  getConfig: (id: number) =>
    client.get<ConsulConfig>(`/consul/configs/${id}`),

  createConfig: (data: Partial<ConsulConfig>) =>
    client.post<ConsulConfig>('/consul/configs', data),

  updateConfig: (id: number, data: Partial<ConsulConfig>) =>
    client.put<ConsulConfig>(`/consul/configs/${id}`, data),

  deleteConfig: (id: number) =>
    client.delete(`/consul/configs/${id}`),

  testConnection: (id: number) =>
    client.post<{ message: string }>(`/consul/configs/${id}/test`),

  // KV 操作
  listKeys: (params: { config_id?: number; prefix?: string; recurse?: boolean; tree?: boolean }) =>
    client.get<string[] | { keys: string[]; tree: KVNode[] }>('/consul/kv', { params }),

  getKeyValue: (key: string, configId?: number) =>
    client.get<KVItem>(`/consul/kv/${encodeURIComponent(key)}`, { params: { config_id: configId } }),

  putKeyValue: (key: string, value: string, configId?: number) =>
    client.put<{ message: string }>(`/consul/kv/${encodeURIComponent(key)}`, { value }, { params: { config_id: configId } }),

  deleteKey: (key: string, configId?: number) =>
    client.delete(`/consul/kv/${encodeURIComponent(key)}`, { params: { config_id: configId } }),

  copyKey: (data: CopyRequest) =>
    client.post<CopyResult>('/consul/kv/copy', data),

  // 批量复制
  batchCopyKeys: (data: BatchCopyRequest) =>
    client.post<BatchCopyResult>('/consul/kv/batch-copy', data),

  // 批量复制所有项目（一键复制）
  batchCopyAllProjects: (data: BatchCopyAllProjectsRequest) =>
    client.post<BatchCopyResult>('/consul/kv/batch-copy-all', data),

  // 替换规则
  getReplaceRules: () =>
    client.get<ReplaceRule[]>('/consul/rules'),

  createReplaceRule: (data: Partial<ReplaceRule>) =>
    client.post<ReplaceRule>('/consul/rules', data),

  updateReplaceRule: (id: number, data: Partial<ReplaceRule>) =>
    client.put<ReplaceRule>(`/consul/rules/${id}`, data),

  deleteReplaceRule: (id: number) =>
    client.delete(`/consul/rules/${id}`),

  // 操作历史
  getOperations: (params?: { config_id?: number; limit?: number }) =>
    client.get<CopyOperation[]>('/consul/operations', { params }),

  deleteOperation: (id: number) =>
    client.delete(`/consul/operations/${id}`),

  // 查询指定后缀的所有项目 Key
  querySuffixKeys: (data: { config_id?: number; suffix: string }) =>
    client.post<{ keys: string[]; total: number }>('/consul/kv/query-suffix-keys', data),

  // 批量删除 KV 键
  batchDeleteKeys: (data: { config_id?: number; keys: string[] }) =>
    client.post<{ deleted: number; failed: number; total: number; deleted_keys: string[]; failed_keys: string[]; errors: string[] }>('/consul/kv/batch-delete', data),
}