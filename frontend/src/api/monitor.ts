import apiClient from './client.js'

// ========== 类型定义 ==========

export interface GrafanaDashboard {
  id: number
  uid: string
  orgId: number
  title: string
  uri: string
  url: string
  type: string
  tags: string[]
  isStarred: boolean
  sortMeta: number
}

export interface GrafanaHealth {
  database: string
  version: string
  commit: string
}

export interface GrafanaDatasource {
  id: number
  uid: string
  name: string
  type: string
  url: string
  isDefault: boolean
}

export interface GrafanaURL {
  url: string
  username: string
}

export interface ServerStatus {
  server_id: number
  hostname: string
  ip: string
  ssh_port: number
  online: boolean
  latency: number
  last_check?: string
  project?: {
    id: number
    name: string
  }
  cluster?: {
    id: number
    name: string
  }
}

export interface ServerStatusResponse {
  data: ServerStatus[]
}

// Prometheus 查询结果
export interface PrometheusResult {
  status: string
  data: {
    resultType: string
    result: Array<{
      metric: Record<string, string>
      value?: [number, string]
      values?: Array<[number, string]>
    }>
  }
}

// ========== API ==========

export const monitorAPI = {
  // Grafana 代理 API（通过后端转发，无 CORS 问题）
  grafana: {
    // 获取 Grafana 健康状态
    getHealth: (): Promise<GrafanaHealth> =>
      apiClient.get<GrafanaHealth>('/grafana/health'),

    // 获取仪表盘列表
    getDashboards: (): Promise<GrafanaDashboard[]> =>
      apiClient.get<GrafanaDashboard[]>('/grafana/dashboards'),

    // 获取仪表盘详情
    getDashboard: (uid: string): Promise<{ meta: unknown; dashboard: unknown }> =>
      apiClient.get<{ meta: unknown; dashboard: unknown }>(`/grafana/dashboards/${uid}`),

    // 获取数据源列表
    getDatasources: (): Promise<GrafanaDatasource[]> =>
      apiClient.get<GrafanaDatasource[]>('/grafana/datasources'),

    // 获取告警（Grafana Unified Alerting）
    getAlerts: (): Promise<unknown[]> =>
      apiClient.get<unknown[]>('/grafana/alerts'),

    // 获取 Prometheus 告警规则（通过 Grafana datasource proxy）
    getPrometheusRules: (): Promise<{
      status: string
      data: {
        groups: Array<{
          name: string
          file: string
          rules: Array<{
            state: string
            name: string
            query: string
            duration: number
            labels: Record<string, string>
            annotations: Record<string, string>
            health: string
            type: string
          }>
          interval: number
        }>
      }
    }> => apiClient.get('/grafana/prometheus-rules'),

    // 获取 Grafana 地址信息
    getURL: (): Promise<GrafanaURL> =>
      apiClient.get<GrafanaURL>('/grafana/url'),

    // Prometheus 即时查询
    query: (expr: string, datasource?: string): Promise<PrometheusResult> => {
      const params = new URLSearchParams({ expr })
      if (datasource) params.append('datasource', datasource)
      return apiClient.get<PrometheusResult>(`/grafana/query?${params.toString()}`)
    },

    // Prometheus 范围查询
    queryRange: (expr: string, start: string, end: string, step?: string): Promise<PrometheusResult> => {
      const params = new URLSearchParams({ expr, start, end, step: step || '60' })
      return apiClient.get<PrometheusResult>(`/grafana/query_range?${params.toString()}`)
    },
  },

  // 服务器监控 API
  servers: {
    // 获取服务器状态（缓存）
    getServerStatus: (): Promise<ServerStatusResponse> =>
      apiClient.get<ServerStatusResponse>('/monitor/servers'),

    // 实时 ping 检测所有服务器
    pingAllServers: (): Promise<ServerStatusResponse> =>
      apiClient.get<ServerStatusResponse>('/monitor/servers/ping'),

    // 实时 ping 检测单个服务器
    pingServer: (id: number): Promise<ServerStatus> =>
      apiClient.get<ServerStatus>(`/monitor/servers/${id}/ping`),

    // 手动触发健康检查
    triggerCheck: (): Promise<{ success: boolean; message: string }> =>
      apiClient.post<{ success: boolean; message: string }>('/monitor/check'),
  },
}
