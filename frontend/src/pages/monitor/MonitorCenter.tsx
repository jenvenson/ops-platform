import { useState, useEffect, useCallback } from 'react'
import {
  Card,
  Button,
  Space,
  Tag,
  Table,
  Tabs,
  Alert,
} from 'antd'
import {
  ReloadOutlined,
  DashboardOutlined,
  LineChartOutlined,
  ExpandOutlined,
  ArrowLeftOutlined,
  LinkOutlined,
  FundProjectionScreenOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
} from '@ant-design/icons'
import { monitorAPI, GrafanaDashboard } from '../../api/monitor.js'

// 从 Prometheus 获取的服务器状态（与 Grafana 服务器资源总览表一致）
interface PrometheusServer {
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
const formatBytes = (bytes: number, decimals = 1): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(decimals)} ${sizes[i]}`
}

// 格式化字节速率 (bytes/s → KB/s, MB/s, GB/s)
const formatBytesRate = (bytesPerSec: number): string => {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(1)} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  if (bytesPerSec < 1024 * 1024 * 1024) return `${(bytesPerSec / 1024 / 1024).toFixed(1)} MB/s`
  return `${(bytesPerSec / 1024 / 1024 / 1024).toFixed(1)} GB/s`
}

// 格式化比特速率 (bits/s → Kbps, Mbps, Gbps)
const formatBitsRate = (bitsPerSec: number): string => {
  if (bitsPerSec < 1000) return `${bitsPerSec.toFixed(1)} bps`
  if (bitsPerSec < 1000000) return `${(bitsPerSec / 1000).toFixed(1)} Kbps`
  if (bitsPerSec < 1000000000) return `${(bitsPerSec / 1000000).toFixed(1)} Mbps`
  return `${(bitsPerSec / 1000000000).toFixed(1)} Gbps`
}

// 百分比颜色（绿 → 黄 → 红渐变）
const getPercentColor = (value: number): string => {
  if (value >= 90) return '#ff4d4f'
  if (value >= 80) return '#fa8c16'
  if (value >= 70) return '#faad14'
  if (value >= 60) return '#fadb14'
  if (value >= 40) return '#a0d911'
  return '#52c41a'
}

// IO/磁盘读写颜色阈值
const getDiskRateColor = (bytesPerSec: number): string => {
  if (bytesPerSec > 20 * 1024 * 1024) return 'rgba(245, 54, 54, 0.9)'
  if (bytesPerSec > 10 * 1024 * 1024) return 'rgba(237, 129, 40, 0.89)'
  return 'rgba(50, 172, 45, 0.97)'
}

// 带宽颜色阈值
const getBandwidthColor = (bitsPerSec: number): string => {
  if (bitsPerSec > 100 * 1024 * 1024) return 'rgba(245, 54, 54, 0.9)'
  if (bitsPerSec > 30 * 1024 * 1024) return 'rgba(237, 129, 40, 0.89)'
  return 'rgba(50, 172, 45, 0.97)'
}

export default function MonitorCenter() {
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('bigscreen')
  const [bigScreenFull, setBigScreenFull] = useState(false)
  const [selectedDashboard, setSelectedDashboard] = useState<string>('')

  // Grafana 数据
  const [dashboards, setDashboards] = useState<GrafanaDashboard[]>([])
  const [error, setError] = useState<string | null>(null)

  // 服务器状态（从 Prometheus 获取）
  const [serverStatuses, setServerStatuses] = useState<PrometheusServer[]>([])
  const [serversLoading, setServersLoading] = useState(false)

  

  

  // 获取 Grafana 基础数据（仪表盘、数据源列表）
  const fetchGrafanaData = useCallback(async () => {
    try {
      const dashboardsData = await monitorAPI.grafana.getDashboards().catch(() => [])
      setDashboards(dashboardsData)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取监控数据失败')
    } finally {
      setLoading(false)
    }
  }, [])

  // 从 Prometheus 查询结果中提取指定 instance 的值
  const getMetricValue = (result: { data?: { result?: Array<{ metric: Record<string, string>; value?: [number, string] }> } } | null, instance: string): number | undefined => {
    if (!result?.data?.result) return undefined
    const item = result.data.result.find(r => r.metric.instance === instance)
    if (item?.value?.[1]) {
      const v = parseFloat(item.value[1])
      return isNaN(v) ? undefined : v
    }
    return undefined
  }

  // 获取服务器状态（从 Prometheus/Grafana，与 Grafana 服务器资源总览表一致）
  const fetchServerStatuses = useCallback(async () => {
    setServersLoading(true)
    try {
      // 并行查询所有指标（与 Grafana 面板查询一致）
      const [
        nodeInfoResult,   // A: 主机名/名称
        uptimeResult,     // D: 启动天数
        memTotalResult,   // B: 总内存
        cpuCoresResult,   // C: CPU核数
        loadResult,       // L: 5分钟负载
        cpuUsageResult,   // F: CPU使用率
        memUsageResult,   // G: 内存使用率
        diskUsageResult,  // E: 分区使用率
        diskReadResult,   // H: 磁盘读取
        diskWriteResult,  // I: 磁盘写入
        netDownResult,    // J: 下载带宽
        netUpResult,      // K: 上传带宽
        ioUtilResult,     // P: IOutil使用率
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
        // 名称：优先使用 name 标签，否则用 instance
        const displayName = info.name || instance
        // IP: instance 标签（可能含端口则去掉端口，否则直接使用）
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
      setServerStatuses(servers)
    } catch (err) {
      console.error('获取服务器状态失败:', err)
    } finally {
      setServersLoading(false)
    }
  }, [])

  // 初始加载
  useEffect(() => {
    setLoading(true)
    fetchGrafanaData()
    fetchServerStatuses()
  }, [fetchGrafanaData, fetchServerStatuses])

  // 每 60 秒自动刷新服务器状态数据
  useEffect(() => {
    const timer = setInterval(() => {
      fetchServerStatuses()
    }, 60000)
    return () => clearInterval(timer)
  }, [fetchServerStatuses])

  // ESC 键退出大屏全屏
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && bigScreenFull) setBigScreenFull(false)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [bigScreenFull])

  // Grafana 代理地址（通过 nginx 反向代理，自动适配当前访问端口）
  const getGrafanaProxyBase = () => {
    return `${window.location.origin}/grafana-proxy`
  }

  // 打开 Grafana（使用代理地址）
  const openGrafana = (path?: string) => {
    window.open(`${getGrafanaProxyBase()}${path || ''}`, '_blank')
  }

  // Dashboard 点击
  const handleDashboardClick = (uid: string) => {
    setSelectedDashboard(uid)
    setActiveTab('dashboard-detail')
  }

  // ========== 渲染 ==========

  // 监控大屏 - 暗色主题仪表盘
  const renderBigScreen = () => {
    const total = serverStatuses.length
    const onlineCount = total // Prometheus 能查到的都是在线的
    const offlineCount = 0

    // 计算各项平均值
    const avg = (arr: (number | undefined)[]) => {
      const valid = arr.filter((v): v is number => v !== undefined)
      return valid.length > 0 ? valid.reduce((a, b) => a + b, 0) / valid.length : 0
    }
    const cpuAvg = avg(serverStatuses.map(s => s.cpuUsage))
    const memAvg = avg(serverStatuses.map(s => s.memUsage))
    const loadAvg = avg(serverStatuses.map(s => s.load5))

    // Top 5 排名
    const cpuTop5 = [...serverStatuses].filter(s => s.cpuUsage !== undefined).sort((a, b) => (b.cpuUsage ?? 0) - (a.cpuUsage ?? 0)).slice(0, 5)
    const memTop5 = [...serverStatuses].filter(s => s.memUsage !== undefined).sort((a, b) => (b.memUsage ?? 0) - (a.memUsage ?? 0)).slice(0, 5)
    const diskTop6 = [...serverStatuses].filter(s => s.diskUsage !== undefined).sort((a, b) => (b.diskUsage ?? 0) - (a.diskUsage ?? 0)).slice(0, 6)
    const loadTop5 = [...serverStatuses].filter(s => s.load5 !== undefined).sort((a, b) => (b.load5 ?? 0) - (a.load5 ?? 0)).slice(0, 5)

    // 警告统计
    const cpuWarnCount = serverStatuses.filter(s => (s.cpuUsage ?? 0) > 90).length
    const memWarnCount = serverStatuses.filter(s => (s.memUsage ?? 0) > 90).length
    const diskWarnCount = serverStatuses.filter(s => (s.diskUsage ?? 0) > 90).length

    // 颜色等级阈值标准
    // ┌──────────────┬──────────────┬──────────────┬──────────────┐
    // │ 指标         │ 绿色（健康） │ 黄色（警告） │ 红色（危险） │
    // ├──────────────┼──────────────┼──────────────┼──────────────┤
    // │ CPU 使用率   │ < 80%        │ 80% ~ 90%    │ >= 90%       │
    // │ 内存使用率   │ < 80%        │ 80% ~ 90%    │ >= 90%       │
    // │ 磁盘使用率   │ < 80%        │ 80% ~ 90%    │ >= 90%       │
    // │ 系统负载     │ < 20         │ 20 ~ 40      │ >= 40        │
    // └──────────────┴──────────────┴──────────────┴──────────────┘
    const getLevel = (v: number, thresholds = [80, 90]) => {
      if (v >= thresholds[1]) return 'danger'
      if (v >= thresholds[0]) return 'warning'
      return 'healthy'
    }
    const levelColors: Record<string, string> = { danger: '#ef4444', warning: '#f59e0b', healthy: '#22c55e' }
    const badgeLabel = (v: number) => {
      const level = getLevel(v)
      return { level, color: levelColors[level] }
    }

    // 仪表盘 SVG（环形图）
    const renderGauge = (value: number, label: string) => {
      const clampedVal = Math.min(Math.max(value, 0), 100)
      const { color } = badgeLabel(clampedVal)
      const radius = 52
      const circumference = 2 * Math.PI * radius
      const dashOffset = circumference * (1 - clampedVal / 100)
      return (
        <div style={{ position: 'relative', width: 130, height: 130, flexShrink: 0 }}>
          <svg viewBox="0 0 120 120" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
            <circle cx="60" cy="60" r={radius} fill="none" stroke="#2d3140" strokeWidth="10" />
            <circle cx="60" cy="60" r={radius} fill="none" stroke={color} strokeWidth="10"
              strokeLinecap="round" strokeDasharray={circumference} strokeDashoffset={dashOffset}
              style={{ transition: 'stroke-dashoffset 0.6s ease' }} />
          </svg>
          <div style={{
            position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)',
            textAlign: 'center',
          }}>
            <div style={{ fontSize: 26, fontWeight: 700, color }}>{clampedVal.toFixed(1)}%</div>
            <div style={{ fontSize: 11, color: '#666' }}>{label}</div>
          </div>
        </div>
      )
    }

    // 进度条项
    const renderTopItem = (name: string, value: number, unit = '%') => {
      const level = getLevel(value, unit === '%' ? [80, 90] : [20, 40])
      const color = levelColors[level]
      const pct = unit === '%' ? Math.min(value, 100) : Math.min(value * 10, 100)
      return (
        <div key={name} style={{ marginBottom: 10 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ fontSize: 13, color: '#ccc', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '60%' }}>{name}</span>
            <span style={{ fontSize: 14, fontWeight: 600, color }}>{value.toFixed(1)}{unit}</span>
          </div>
          <div style={{ height: 6, background: '#2d3140', borderRadius: 3, overflow: 'hidden' }}>
            <div style={{ height: '100%', borderRadius: 3, width: `${pct}%`, background: `linear-gradient(90deg, ${color}cc, ${color})`, transition: 'width 0.4s ease' }} />
          </div>
        </div>
      )
    }

    // 磁盘进度条
    const renderDiskItem = (s: PrometheusServer) => {
      const usage = s.diskUsage ?? 0
      const level = getLevel(usage, [80, 90])
      const color = levelColors[level]
      return (
        <div key={s.instance} style={{ marginBottom: 14 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ fontSize: 13, color: '#ccc' }}>{s.name}</span>
            <span style={{ fontSize: 12, color: '#888' }}>{usage.toFixed(1)}%</span>
          </div>
          <div style={{ height: 8, background: '#2d3140', borderRadius: 4, overflow: 'hidden' }}>
            <div style={{ height: '100%', borderRadius: 4, width: `${Math.min(usage, 100)}%`, background: `linear-gradient(90deg, ${color}cc, ${color})`, transition: 'width 0.4s ease' }} />
          </div>
        </div>
      )
    }

    // 大屏卡片样式
    const cardStyle: React.CSSProperties = { background: '#1a1d29', borderRadius: 12, padding: 20 }
    const titleStyle: React.CSSProperties = { fontSize: 15, fontWeight: 600, marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8, color: '#e0e0e0' }

    return (
      <div style={{
        background: '#0f1117', borderRadius: bigScreenFull ? 0 : 8, padding: 24,
        margin: bigScreenFull ? 0 : -16, minHeight: 600,
        ...(bigScreenFull ? { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 1000, overflow: 'auto' } : {}),
      }}>
        {/* 顶部工具栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <div style={{ fontSize: 18, fontWeight: 600, color: '#fff' }}>监控大屏</div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              onClick={() => { fetchServerStatuses() }}
              style={{
                background: 'rgba(255,255,255,0.1)', border: '1px solid rgba(255,255,255,0.15)',
                color: '#e0e0e0', padding: '6px 14px', borderRadius: 6, cursor: 'pointer',
                display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, transition: 'all 0.2s',
              }}
              onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.2)' }}
              onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.1)' }}
            >
              <ReloadOutlined spin={serversLoading} /> 刷新数据
            </button>
            <button
              onClick={() => setBigScreenFull(!bigScreenFull)}
              style={{
                background: 'rgba(255,255,255,0.1)', border: '1px solid rgba(255,255,255,0.15)',
                color: '#e0e0e0', padding: '6px 14px', borderRadius: 6, cursor: 'pointer',
                display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, transition: 'all 0.2s',
              }}
              onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.2)' }}
              onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.1)' }}
            >
              {bigScreenFull ? <><FullscreenExitOutlined /> 退出全屏</> : <><FullscreenOutlined /> 最大化</>}
            </button>
          </div>
        </div>

        {/* 顶部：主机状态 */}
        <div style={{ marginBottom: 24 }}>
          <div style={{ fontSize: 13, color: '#666', marginBottom: 12, textTransform: 'uppercase', letterSpacing: 1 }}>主机状态</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16 }}>
            {/* 在线主机 */}
            <div style={{ ...cardStyle }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontSize: 13, color: '#888' }}>在线 (Up)</span>
                <span style={{ width: 10, height: 10, borderRadius: '50%', background: '#22c55e', boxShadow: '0 0 8px #22c55e', animation: 'pulse 1.5s ease-in-out infinite' }} />
              </div>
              <div style={{ fontSize: 42, fontWeight: 700, color: '#22c55e', lineHeight: 1 }}>{onlineCount}</div>
              <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>总主机: {total}</div>
            </div>
            {/* 离线主机 */}
            <div style={{
              ...cardStyle,
              ...(offlineCount > 0 ? { background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)' } : {}),
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontSize: 13, color: '#888' }}>离线 (Down)</span>
                <span style={{ width: 10, height: 10, borderRadius: '50%', background: offlineCount > 0 ? '#ef4444' : '#444', boxShadow: offlineCount > 0 ? '0 0 8px #ef4444' : 'none' }} />
              </div>
              <div style={{ fontSize: 42, fontWeight: 700, color: offlineCount > 0 ? '#ef4444' : '#444', lineHeight: 1 }}>{offlineCount}</div>
              <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>{offlineCount > 0 ? '需要关注!' : '全部正常'}</div>
            </div>
            {/* CPU 均值 */}
            <div style={cardStyle}>
              <div style={{ fontSize: 13, color: '#888', marginBottom: 8 }}>CPU 均值</div>
              <div style={{ fontSize: 42, fontWeight: 700, color: badgeLabel(cpuAvg).color, lineHeight: 1 }}>{cpuAvg.toFixed(1)}%</div>
              <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>{cpuWarnCount > 0 ? `${cpuWarnCount} 台超过 90%` : '全部正常'}</div>
            </div>
            {/* 内存均值 */}
            <div style={cardStyle}>
              <div style={{ fontSize: 13, color: '#888', marginBottom: 8 }}>内存均值</div>
              <div style={{ fontSize: 42, fontWeight: 700, color: badgeLabel(memAvg).color, lineHeight: 1 }}>{memAvg.toFixed(1)}%</div>
              <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>{memWarnCount > 0 ? `${memWarnCount} 台超过 90%` : '全部正常'}</div>
            </div>
          </div>
        </div>

        {/* 中部：性能预警 2x2 */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
          {/* CPU 使用率 */}
          <div style={cardStyle}>
            <div style={titleStyle}>
              CPU 使用率
              <span style={{
                fontSize: 11, padding: '2px 8px', borderRadius: 10,
                background: `${badgeLabel(cpuAvg).color}33`, color: badgeLabel(cpuAvg).color,
              }}>均值 {cpuAvg.toFixed(1)}%</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
              {renderGauge(cpuAvg, '集群均值')}
              <div style={{ flex: 1 }}>
                {cpuTop5.map(s => renderTopItem(s.name, s.cpuUsage ?? 0))}
                {cpuTop5.length === 0 && <div style={{ color: '#666', fontSize: 13 }}>暂无数据</div>}
              </div>
            </div>
          </div>

          {/* 内存使用率 */}
          <div style={cardStyle}>
            <div style={titleStyle}>
              内存使用率
              <span style={{
                fontSize: 11, padding: '2px 8px', borderRadius: 10,
                background: `${badgeLabel(memAvg).color}33`, color: badgeLabel(memAvg).color,
              }}>均值 {memAvg.toFixed(1)}%</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
              {renderGauge(memAvg, '集群均值')}
              <div style={{ flex: 1 }}>
                {memTop5.map(s => renderTopItem(s.name, s.memUsage ?? 0))}
                {memTop5.length === 0 && <div style={{ color: '#666', fontSize: 13 }}>暂无数据</div>}
              </div>
            </div>
          </div>

          {/* 磁盘使用率 */}
          <div style={cardStyle}>
            <div style={titleStyle}>
              磁盘（分区）使用率
              <span style={{
                fontSize: 11, padding: '2px 8px', borderRadius: 10,
                background: diskWarnCount > 0 ? 'rgba(239,68,68,0.2)' : 'rgba(34,197,94,0.2)',
                color: diskWarnCount > 0 ? '#ef4444' : '#22c55e',
              }}>{diskWarnCount > 0 ? `${diskWarnCount} 台 > 90%` : '全部正常'}</span>
            </div>
            {diskTop6.map(s => renderDiskItem(s))}
            {diskTop6.length === 0 && <div style={{ color: '#666', fontSize: 13 }}>暂无数据</div>}
          </div>

          {/* 系统负载 */}
          <div style={cardStyle}>
            <div style={titleStyle}>
              系统负载（5分钟）
              <span style={{
                fontSize: 11, padding: '2px 8px', borderRadius: 10,
                background: 'rgba(24,144,255,0.2)', color: '#1890ff',
              }}>均值 {loadAvg.toFixed(2)}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
              {/* 负载仪表盘 - 用相对于核数的百分比 */}
              {(() => {
                const avgCores = avg(serverStatuses.map(s => s.cpuCores))
                const loadPct = avgCores > 0 ? Math.min((loadAvg / avgCores) * 100, 100) : 0
                return renderGauge(loadPct, '负载/核数')
              })()}
              <div style={{ flex: 1 }}>
                {loadTop5.map(s => renderTopItem(s.name, s.load5 ?? 0, ''))}
                {loadTop5.length === 0 && <div style={{ color: '#666', fontSize: 13 }}>暂无数据</div>}
              </div>
            </div>
          </div>
        </div>

        {/* 动画样式 */}
        <style>{`
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}</style>
      </div>
    )
  }

  // 概览标签页
  const renderOverview = () => (
    <div>
      {/* 服务器资源总览表（从 Prometheus 获取，与 Grafana 一致） */}
      <Card
        title={`服务器资源总览【主机总数：${serverStatuses.length}】`}
        size="small"
        style={{ marginBottom: 16 }}
        extra={
          <Button
            size="small"
            icon={<ReloadOutlined spin={serversLoading} />}
            onClick={fetchServerStatuses}
          >
            刷新
          </Button>
        }
      >
        <Table
          columns={[
            {
              title: '名称',
              dataIndex: 'name',
              key: 'name',
              width: 145,
              fixed: 'left',
              sorter: (a: PrometheusServer, b: PrometheusServer) => a.name.localeCompare(b.name),
              render: (name: string) => <span style={{ fontWeight: 500 }}>{name}</span>,
            },
            {
              title: 'IP',
              dataIndex: 'ip',
              key: 'ip',
              width: 130,
              sorter: (a: PrometheusServer, b: PrometheusServer) => {
                const toNum = (ip: string) => ip.split('.').reduce((acc, oct) => acc * 256 + parseInt(oct, 10), 0)
                return toNum(a.ip) - toNum(b.ip)
              },
            },
            {
              title: '启动(天)',
              dataIndex: 'uptimeDays',
              key: 'uptimeDays',
              width: 80,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.uptimeDays ?? 0) - (b.uptimeDays ?? 0),
              render: (days: number | undefined) => days !== undefined ? days.toFixed(1) : '-',
            },
            {
              title: '内存',
              dataIndex: 'memTotal',
              key: 'memTotal',
              width: 80,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.memTotal ?? 0) - (b.memTotal ?? 0),
              render: (bytes: number | undefined) => {
                if (bytes === undefined) return '-'
                return <span style={{ color: '#1890ff', fontWeight: 500 }}>{formatBytes(bytes, 0)}</span>
              },
            },
            {
              title: 'CPU',
              dataIndex: 'cpuCores',
              key: 'cpuCores',
              width: 55,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.cpuCores ?? 0) - (b.cpuCores ?? 0),
              render: (cores: number | undefined) => {
                if (cores === undefined) return '-'
                return <span style={{ color: '#1890ff', fontWeight: 500 }}>{Math.round(cores)}</span>
              },
            },
            {
              title: '负载',
              dataIndex: 'load5',
              key: 'load5',
              width: 70,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.load5 ?? 0) - (b.load5 ?? 0),
              render: (load: number | undefined) => load !== undefined ? load.toFixed(2) : '-',
            },
            {
              title: 'CPU使用率',
              dataIndex: 'cpuUsage',
              key: 'cpuUsage',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.cpuUsage ?? 0) - (b.cpuUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '内存使用率',
              dataIndex: 'memUsage',
              key: 'memUsage',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.memUsage ?? 0) - (b.memUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: 'IOutil使用率*',
              dataIndex: 'ioUtil',
              key: 'ioUtil',
              width: 110,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.ioUtil ?? 0) - (b.ioUtil ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '分区使用率*',
              dataIndex: 'diskUsage',
              key: 'diskUsage',
              width: 120,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskUsage ?? 0) - (b.diskUsage ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const pct = Math.min(val, 100)
                return (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <div style={{ flex: 1, height: 16, background: '#f0f0f0', borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: `linear-gradient(90deg, #52c41a, ${getPercentColor(pct)})`, borderRadius: 2, transition: 'width 0.3s' }} />
                    </div>
                    <span style={{ fontSize: 12, minWidth: 40, textAlign: 'right' }}>{pct.toFixed(1)}%</span>
                  </div>
                )
              },
            },
            {
              title: '磁盘读取*',
              dataIndex: 'diskRead',
              key: 'diskRead',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskRead ?? 0) - (b.diskRead ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getDiskRateColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBytesRate(val)}
                  </span>
                )
              },
            },
            {
              title: '磁盘写入*',
              dataIndex: 'diskWrite',
              key: 'diskWrite',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.diskWrite ?? 0) - (b.diskWrite ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getDiskRateColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBytesRate(val)}
                  </span>
                )
              },
            },
            {
              title: '下载带宽*',
              dataIndex: 'netDown',
              key: 'netDown',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.netDown ?? 0) - (b.netDown ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getBandwidthColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBitsRate(val)}
                  </span>
                )
              },
            },
            {
              title: '上传带宽*',
              dataIndex: 'netUp',
              key: 'netUp',
              width: 100,
              sorter: (a: PrometheusServer, b: PrometheusServer) => (a.netUp ?? 0) - (b.netUp ?? 0),
              render: (val: number | undefined) => {
                if (val === undefined) return '-'
                const color = getBandwidthColor(val)
                return (
                  <span style={{
                    background: color,
                    color: '#fff',
                    padding: '2px 6px',
                    borderRadius: 3,
                    fontSize: 12,
                    whiteSpace: 'nowrap',
                  }}>
                    {formatBitsRate(val)}
                  </span>
                )
              },
            },
          ]}
          dataSource={serverStatuses}
          rowKey="instance"
          loading={serversLoading}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total) => `共 ${total} 台`,
            showQuickJumper: true,
          }}
          size="small"
          scroll={{ x: 1600 }}
        />
      </Card>
    </div>
  )

  // 仪表盘列表标签页
  const renderDashboards = () => (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<LinkOutlined />}
          onClick={() => openGrafana('/dashboards')}
        >
          在 Grafana 中查看全部
        </Button>
      </div>
      <Table
        columns={[
          {
            title: '仪表盘名称',
            dataIndex: 'title',
            key: 'title',
            render: (title: string, record: GrafanaDashboard) => (
              <a onClick={() => handleDashboardClick(record.uid)} style={{ fontWeight: 500 }}>
                <DashboardOutlined style={{ marginRight: 8 }} />
                {title}
              </a>
            ),
          },
          {
            title: '标签',
            dataIndex: 'tags',
            key: 'tags',
            width: 250,
            render: (tags: string[]) => (
              <Space wrap>
                {(tags || []).slice(0, 4).map(tag => <Tag key={tag}>{tag}</Tag>)}
                {tags && tags.length > 4 && <Tag>+{tags.length - 4}</Tag>}
              </Space>
            ),
          },
          {
            title: '操作',
            key: 'action',
            width: 180,
            render: (_: unknown, record: GrafanaDashboard) => (
              <Space>
                <Button
                  type="link"
                  size="small"
                  onClick={() => handleDashboardClick(record.uid)}
                >
                  嵌入查看
                </Button>
                <Button
                  type="link"
                  size="small"
                  icon={<ExpandOutlined />}
                  onClick={() => openGrafana(record.url)}
                >
                  新窗口
                </Button>
              </Space>
            ),
          },
        ]}
        dataSource={dashboards}
        rowKey="id"
        loading={loading}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
      />
    </div>
  )

  // 仪表盘详情标签页（嵌入 iframe，通过 nginx 反向代理）
  const renderDashboardDetail = () => {
    const dashboard = dashboards.find(d => d.uid === selectedDashboard)
    const proxyBase = getGrafanaProxyBase()

    return (
      <div>
        <Space style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => setActiveTab('dashboards')}>
            返回列表
          </Button>
          <Button
            type="primary"
            icon={<ExpandOutlined />}
            onClick={() => openGrafana(`/d/${selectedDashboard}`)}
          >
            在 Grafana 中打开
          </Button>
        </Space>
        <Card title={dashboard?.title || '仪表盘'} size="small">
          <iframe
            src={`${proxyBase}/d/${selectedDashboard}?orgId=1&kiosk`}
            width="100%"
            height={750}
            style={{ border: 'none', borderRadius: 4 }}
            title="Grafana Dashboard"
          />
        </Card>
      </div>
    )
  }

  return (
    <div>
      {error && (
        <Alert
          message="监控数据获取失败"
          description={error}
          type="error"
          showIcon
          closable
          style={{ marginBottom: 16 }}
          action={<Button size="small" onClick={() => { fetchGrafanaData(); fetchServerStatuses(); }}>重试</Button>}
        />
      )}

      {/* 标签页 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'bigscreen',
              label: <span><FundProjectionScreenOutlined /> 监控大屏</span>,
              children: renderBigScreen(),
            },
            {
              key: 'overview',
              label: <span><LineChartOutlined /> 监控概览</span>,
              children: renderOverview(),
            },
            {
              key: 'dashboards',
              label: <span><DashboardOutlined /> Grafana仪表盘</span>,
              children: renderDashboards(),
            },
            
            ...(selectedDashboard ? [{
              key: 'dashboard-detail',
              label: <span><ExpandOutlined /> 仪表盘详情</span>,
              children: renderDashboardDetail(),
            }] : []),
          ]}
        />
      </Card>
    </div>
  )
}
