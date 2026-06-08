import { monitorAPI } from '../../api/monitor.js'

// 从 Prometheus 获取的服务器状态（与 Grafana 服务器资源总览表一致）
export interface PrometheusServer {
  instance: string
  name: string         // 名称 (name label)
  ip: string           // IP (instance 去掉端口)
  uptimeDays?: number  // 启动(天)
  memTotal?: number    // 总内存 (bytes)
  cpuCores?: number    // CPU 核数
  load5?: number       // 5分钟负载
  cpuUsage?: number    // CPU 使用率 (%)
  memUsage?: number    // 内存使用率 (%)
  ioUtil?: number      // IOutil 使用率 (%)
  diskUsage?: number   // 分区使用率 (%)
  diskRead?: number    // 磁盘读取 (bytes/s)
  diskWrite?: number   // 磁盘写入 (bytes/s)
  netDown?: number     // 下载带宽 (bits/s)
  netUp?: number       // 上传带宽 (bits/s)
}

// ===== 格式化工具函数 =====

// 格式化字节 (bytes → KB/MB/GB/TB)
export const formatBytes = (bytes: number, decimals = 1): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(decimals)} ${sizes[i]}`
}

// 格式化字节速率 (bytes/s → KB/s, MB/s, GB/s)
export const formatBytesRate = (bytesPerSec: number): string => {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(1)} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  if (bytesPerSec < 1024 * 1024 * 1024) return `${(bytesPerSec / 1024 / 1024).toFixed(1)} MB/s`
  return `${(bytesPerSec / 1024 / 1024 / 1024).toFixed(1)} GB/s`
}

// 格式化比特速率 (bits/s → Kbps, Mbps, Gbps)
export const formatBitsRate = (bitsPerSec: number): string => {
  if (bitsPerSec < 1000) return `${bitsPerSec.toFixed(1)} bps`
  if (bitsPerSec < 1000000) return `${(bitsPerSec / 1000).toFixed(1)} Kbps`
  if (bitsPerSec < 1000000000) return `${(bitsPerSec / 1000000).toFixed(1)} Mbps`
  return `${(bitsPerSec / 1000000000).toFixed(1)} Gbps`
}

// 百分比颜色（绿 → 黄 → 红渐变）
export const getPercentColor = (value: number): string => {
  if (value >= 90) return '#ff4d4f'
  if (value >= 80) return '#fa8c16'
  if (value >= 70) return '#faad14'
  if (value >= 60) return '#fadb14'
  if (value >= 40) return '#a0d911'
  return '#52c41a'
}

// IO/磁盘读写颜色阈值
export const getDiskRateColor = (bytesPerSec: number): string => {
  if (bytesPerSec > 20 * 1024 * 1024) return 'rgba(245, 54, 54, 0.9)'
  if (bytesPerSec > 10 * 1024 * 1024) return 'rgba(237, 129, 40, 0.89)'
  return 'rgba(50, 172, 45, 0.97)'
}

// 带宽颜色阈值
export const getBandwidthColor = (bitsPerSec: number): string => {
  if (bitsPerSec > 100 * 1024 * 1024) return 'rgba(245, 54, 54, 0.9)'
  if (bitsPerSec > 30 * 1024 * 1024) return 'rgba(237, 129, 40, 0.89)'
  return 'rgba(50, 172, 45, 0.97)'
}

// 从 Prometheus 查询结果中提取指定 instance 的值
export const getMetricValue = (
  result: { data?: { result?: Array<{ metric: Record<string, string>; value?: [number, string] }> } } | null,
  instance: string
): number | undefined => {
  if (!result?.data?.result) return undefined
  const item = result.data.result.find(r => r.metric.instance === instance)
  if (item?.value?.[1]) {
    const v = parseFloat(item.value[1])
    return isNaN(v) ? undefined : v
  }
  return undefined
}

// 获取 Grafana 代理地址（通过 nginx 反向代理，自动适配当前访问端口）
export const getGrafanaProxyBase = () => {
  return `${window.location.origin}/grafana-proxy`
}

// 打开 Grafana（使用代理地址）
export const openGrafana = (path?: string) => {
  window.open(`${getGrafanaProxyBase()}${path || ''}`, '_blank')
}

// 获取服务器状态数据（从 Prometheus/Grafana）
export const fetchServerStatusesData = async (): Promise<PrometheusServer[]> => {
  // 并行查询所有指标（与 Grafana 面板查询一致）
  const [
    nodeInfoResult,
    uptimeResult,
    memTotalResult,
    cpuCoresResult,
    loadResult,
    cpuUsageResult,
    memUsageResult,
    diskUsageResult,
    diskReadResult,
    diskWriteResult,
    netDownResult,
    netUpResult,
    ioUtilResult,
  ] = await Promise.all([
    monitorAPI.grafana.query('node_uname_info - 0').catch(() => null),
    monitorAPI.grafana.query('sum(time() - node_boot_time_seconds)by(instance)/86400').catch(() => null),
    monitorAPI.grafana.query('node_memory_MemTotal_bytes - 0').catch(() => null),
    monitorAPI.grafana.query('count(node_cpu_seconds_total{mode="system"}) by (instance)').catch(() => null),
    monitorAPI.grafana.query('node_load5').catch(() => null),
    monitorAPI.grafana.query('(1 - avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) by (instance)) * 100').catch(() => null),
    monitorAPI.grafana.query('(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100').catch(() => null),
    monitorAPI.grafana.query('max((node_filesystem_size_bytes{fstype=~"ext.?|xfs"}-node_filesystem_free_bytes{fstype=~"ext.?|xfs"}) *100/(node_filesystem_avail_bytes{fstype=~"ext.?|xfs"}+(node_filesystem_size_bytes{fstype=~"ext.?|xfs"}-node_filesystem_free_bytes{fstype=~"ext.?|xfs"})))by(instance)').catch(() => null),
    monitorAPI.grafana.query('max(irate(node_disk_read_bytes_total[5m])) by (instance)').catch(() => null),
    monitorAPI.grafana.query('max(irate(node_disk_written_bytes_total[5m])) by (instance)').catch(() => null),
    monitorAPI.grafana.query('max(irate(node_network_receive_bytes_total[5m])*8) by (instance)').catch(() => null),
    monitorAPI.grafana.query('max(irate(node_network_transmit_bytes_total[5m])*8) by (instance)').catch(() => null),
    monitorAPI.grafana.query('max(irate(node_disk_io_time_seconds_total[5m])) by (instance) *100').catch(() => null),
  ])

  // 从 node_uname_info 发现所有服务器实例
  const nodeMap = new Map<string, { name: string; group: string; nodename: string }>()
  if (nodeInfoResult?.data?.result) {
    for (const item of nodeInfoResult.data.result) {
      const inst = item.metric.instance
      if (inst) {
        nodeMap.set(inst, {
          name: item.metric.name || item.metric.instance || '',
          group: item.metric.group || item.metric.job || '',
          nodename: item.metric.nodename || '',
        })
      }
    }
  }
  // 备选：从 CPU 使用率结果发现实例
  if (nodeMap.size === 0 && cpuUsageResult?.data?.result) {
    for (const item of cpuUsageResult.data.result) {
      const inst = item.metric.instance
      if (inst && !nodeMap.has(inst)) {
        nodeMap.set(inst, { name: inst, group: '', nodename: '' })
      }
    }
  }

  // 为每个实例构建服务器状态
  const servers: PrometheusServer[] = []
  for (const [instance, info] of nodeMap) {
    const displayName = info.name || instance
    const ip = instance.includes(':') ? instance.split(':')[0] : instance

    servers.push({
      instance,
      name: displayName,
      ip,
      uptimeDays: getMetricValue(uptimeResult, instance),
      memTotal: getMetricValue(memTotalResult, instance),
      cpuCores: getMetricValue(cpuCoresResult, instance),
      load5: getMetricValue(loadResult, instance),
      cpuUsage: getMetricValue(cpuUsageResult, instance),
      memUsage: getMetricValue(memUsageResult, instance),
      ioUtil: getMetricValue(ioUtilResult, instance),
      diskUsage: getMetricValue(diskUsageResult, instance),
      diskRead: getMetricValue(diskReadResult, instance),
      diskWrite: getMetricValue(diskWriteResult, instance),
      netDown: getMetricValue(netDownResult, instance),
      netUp: getMetricValue(netUpResult, instance),
    })
  }

  // 默认按名称排序
  servers.sort((a, b) => a.name.localeCompare(b.name))
  return servers
}
