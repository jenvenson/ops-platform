// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState, useEffect, useCallback } from 'react'
import {
  ReloadOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { PrometheusServer, fetchServerStatusesData, formatBytesRate, formatBitsRate, getDiskRateColor, getBandwidthColor } from './monitorShared'

// 深色主题 - 与安全概览一致
const theme = {
  bg: '#0a0e17',
  card: '#111827',
  border: '#1e293b',
  text: '#e2e8f0',
  textSecondary: '#94a3b8',
  primary: '#3b82f6',
}

export default function MonitorBigScreenPage() {
  const { t } = useTranslation('monitor')
  const { t: tc } = useTranslation('common')
  const [serverStatuses, setServerStatuses] = useState<PrometheusServer[]>([])
  const [serversLoading, setServersLoading] = useState(false)
  const [bigScreenFull, setBigScreenFull] = useState(false)

  const fetchServerStatuses = useCallback(async () => {
    setServersLoading(true)
    try {
      const servers = await fetchServerStatusesData()
      setServerStatuses(servers)
    } catch (err) {
      console.error('获取服务器状态失败:', err)
    } finally {
      setServersLoading(false)
    }
  }, [])

  // 初始加载
  useEffect(() => {
    fetchServerStatuses()
  }, [fetchServerStatuses])

  // 每 60 秒自动刷新
  useEffect(() => {
    const timer = setInterval(() => {
      fetchServerStatuses()
    }, 60000)
    return () => clearInterval(timer)
  }, [fetchServerStatuses])

  // ESC 键退出全屏
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && bigScreenFull) setBigScreenFull(false)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [bigScreenFull])

  const total = serverStatuses.length
  const onlineCount = total
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
  const diskTop5 = [...serverStatuses].filter(s => s.diskUsage !== undefined).sort((a, b) => (b.diskUsage ?? 0) - (a.diskUsage ?? 0)).slice(0, 5)
  const loadTop5 = [...serverStatuses].filter(s => s.load5 !== undefined).sort((a, b) => (b.load5 ?? 0) - (a.load5 ?? 0)).slice(0, 5)
  const diskReadTop5 = [...serverStatuses].filter(s => s.diskRead !== undefined).sort((a, b) => (b.diskRead ?? 0) - (a.diskRead ?? 0)).slice(0, 5)
  const diskWriteTop5 = [...serverStatuses].filter(s => s.diskWrite !== undefined).sort((a, b) => (b.diskWrite ?? 0) - (a.diskWrite ?? 0)).slice(0, 5)
  const netDownTop5 = [...serverStatuses].filter(s => s.netDown !== undefined).sort((a, b) => (b.netDown ?? 0) - (a.netDown ?? 0)).slice(0, 5)
  const netUpTop5 = [...serverStatuses].filter(s => s.netUp !== undefined).sort((a, b) => (b.netUp ?? 0) - (a.netUp ?? 0)).slice(0, 5)

  // 警告统计
  const cpuWarnCount = serverStatuses.filter(s => (s.cpuUsage ?? 0) > 90).length
  const memWarnCount = serverStatuses.filter(s => (s.memUsage ?? 0) > 90).length
  const diskAvg = avg(serverStatuses.map(s => s.diskUsage))

  // 颜色等级阈值标准
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
          <circle cx="60" cy="60" r={radius} fill="none" stroke={theme.border} strokeWidth="10" />
          <circle cx="60" cy="60" r={radius} fill="none" stroke={color} strokeWidth="10"
            strokeLinecap="round" strokeDasharray={circumference} strokeDashoffset={dashOffset}
            style={{ transition: 'stroke-dashoffset 0.6s ease' }} />
        </svg>
        <div style={{
          position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)',
          textAlign: 'center',
        }}>
          <div style={{ fontSize: 26, fontWeight: 700, color }}>{clampedVal.toFixed(1)}%</div>
          <div style={{ fontSize: 11, color: theme.textSecondary }}>{label}</div>
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
          <span style={{ fontSize: 13, color: theme.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '60%' }}>{name}</span>
          <span style={{ fontSize: 14, fontWeight: 600, color }}>{value.toFixed(1)}{unit}</span>
        </div>
        <div style={{ height: 6, background: theme.border, borderRadius: 3, overflow: 'hidden' }}>
          <div style={{ height: '100%', borderRadius: 3, width: `${pct}%`, background: `linear-gradient(90deg, ${color}cc, ${color})`, transition: 'width 0.4s ease' }} />
        </div>
      </div>
    )
  }

  // 磁盘读写/网络带宽进度条项
  const renderRateItem = (
    name: string,
    value: number | undefined,
    formatFn: (v: number) => string,
    colorFn: (v: number) => string,
    isBytes = true
  ) => {
    const val = value ?? 0
    const color = colorFn(val)
    const maxRef = isBytes ? 50 * 1024 * 1024 : 100 * 1024 * 1024
    const pct = Math.min((val / maxRef) * 100, 100)
    return (
      <div key={name} style={{ marginBottom: 10 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
          <span style={{ fontSize: 13, color: theme.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '60%' }}>{name}</span>
          <span style={{ fontSize: 14, fontWeight: 600, color }}>{formatFn(val)}</span>
        </div>
        <div style={{ height: 6, background: theme.border, borderRadius: 3, overflow: 'hidden' }}>
          <div style={{ height: '100%', borderRadius: 3, width: `${pct}%`, background: `linear-gradient(90deg, ${color}cc, ${color})`, transition: 'width 0.4s ease' }} />
        </div>
      </div>
    )
  }

  // 大屏卡片样式 - 深色主题
  const cardStyle: React.CSSProperties = { background: theme.card, borderRadius: 12, padding: 20, border: `1px solid ${theme.border}` }
  const titleStyle: React.CSSProperties = { fontSize: 15, fontWeight: 600, marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8, color: theme.text }

  return (
    <div style={{
      background: theme.bg,
      minHeight: bigScreenFull ? '100vh' : '100vh',
      padding: bigScreenFull ? 0 : 24,
      margin: bigScreenFull ? 0 : 0,
      borderRadius: bigScreenFull ? 0 : 8,
      ...(bigScreenFull ? {
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 1000,
        overflow: 'auto'
      } : {}),
    }}>
      {/* 顶部工具栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div style={{ fontSize: 18, fontWeight: 600, color: theme.text }}>{t('bigScreenTitle', '监控大屏')}</div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            onClick={() => { fetchServerStatuses() }}
            style={{
              background: theme.primary, border: 'none',
              color: '#fff', padding: '8px 16px', borderRadius: 6, cursor: 'pointer',
              display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, transition: 'all 0.2s',
            }}
            onMouseEnter={e => { e.currentTarget.style.opacity = '0.9' }}
            onMouseLeave={e => { e.currentTarget.style.opacity = '1' }}
          >
            <ReloadOutlined spin={serversLoading} /> {t('refreshData', '刷新数据')}
          </button>
          <button
            onClick={() => setBigScreenFull(!bigScreenFull)}
            style={{
              background: 'rgba(255,255,255,0.1)', border: '1px solid rgba(255,255,255,0.15)',
              color: theme.text, padding: '8px 16px', borderRadius: 6, cursor: 'pointer',
              display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, transition: 'all 0.2s',
            }}
            onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.2)' }}
            onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.1)' }}
          >
            {bigScreenFull ? <><FullscreenExitOutlined /> {t('exitFullscreen', '退出全屏')}</> : <><FullscreenOutlined /> {t('maximize', '最大化')}</>}
          </button>
        </div>
      </div>

      {/* 顶部：主机状态 */}
      <div style={{ marginBottom: 24 }}>
        <div style={{ fontSize: 13, color: theme.textSecondary, marginBottom: 12, textTransform: 'uppercase', letterSpacing: 1 }}>{t('hostStatus', '主机状态')}</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16 }}>
          {/* 在线主机 */}
          <div style={{ ...cardStyle }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <span style={{ fontSize: 13, color: theme.textSecondary }}>{t('onlineUp', '在线 (Up)')}</span>
              <span style={{ width: 10, height: 10, borderRadius: '50%', background: '#22c55e', boxShadow: '0 0 8px #22c55e', animation: 'pulse 1.5s ease-in-out infinite' }} />
            </div>
            <div style={{ fontSize: 42, fontWeight: 700, color: '#22c55e', lineHeight: 1 }}>{onlineCount}</div>
            <div style={{ fontSize: 12, color: theme.textSecondary, marginTop: 4 }}>{t('totalHosts', '总主机')}: {total}</div>
          </div>
          {/* 离线主机 */}
          <div style={{
            ...cardStyle,
            ...(offlineCount > 0 ? { background: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.3)' } : {}),
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <span style={{ fontSize: 13, color: theme.textSecondary }}>{t('offlineDown', '离线 (Down)')}</span>
              <span style={{ width: 10, height: 10, borderRadius: '50%', background: offlineCount > 0 ? '#ef4444' : '#444', boxShadow: offlineCount > 0 ? '0 0 8px #ef4444' : 'none' }} />
            </div>
            <div style={{ fontSize: 42, fontWeight: 700, color: offlineCount > 0 ? '#ef4444' : '#666', lineHeight: 1 }}>{offlineCount}</div>
            <div style={{ fontSize: 12, color: theme.textSecondary, marginTop: 4 }}>{offlineCount > 0 ? t('needsAttention', '需要关注!') : t('allNormal', '全部正常')}</div>
          </div>
          {/* CPU 均值 */}
          <div style={cardStyle}>
            <div style={{ fontSize: 13, color: theme.textSecondary, marginBottom: 8 }}>{t('cpuAverage', 'CPU 均值')}</div>
            <div style={{ fontSize: 42, fontWeight: 700, color: badgeLabel(cpuAvg).color, lineHeight: 1 }}>{cpuAvg.toFixed(1)}%</div>
            <div style={{ fontSize: 12, color: theme.textSecondary, marginTop: 4 }}>{cpuWarnCount > 0 ? t('machinesOverThreshold', '{{count}} 台超过 90%', { count: cpuWarnCount }) : t('allNormal', '全部正常')}</div>
          </div>
          {/* 内存均值 */}
          <div style={cardStyle}>
            <div style={{ fontSize: 13, color: theme.textSecondary, marginBottom: 8 }}>{t('memoryAverage', '内存均值')}</div>
            <div style={{ fontSize: 42, fontWeight: 700, color: badgeLabel(memAvg).color, lineHeight: 1 }}>{memAvg.toFixed(1)}%</div>
            <div style={{ fontSize: 12, color: theme.textSecondary, marginTop: 4 }}>{memWarnCount > 0 ? t('machinesOverThreshold', '{{count}} 台超过 90%', { count: memWarnCount }) : t('allNormal', '全部正常')}</div>
          </div>
        </div>
      </div>

      {/* 中部：性能预警 2x2 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* CPU 使用率 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('cpuUsage', 'CPU 使用率')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: `${badgeLabel(cpuAvg).color}33`, color: badgeLabel(cpuAvg).color,
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
            {renderGauge(cpuAvg, t('clusterAverage', '集群均值'))}
            <div style={{ flex: 1 }}>
              {cpuTop5.map(s => renderTopItem(s.name, s.cpuUsage ?? 0))}
              {cpuTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
          </div>
        </div>

        {/* 内存使用率 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('memoryUsage', '内存使用率')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: `${badgeLabel(memAvg).color}33`, color: badgeLabel(memAvg).color,
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
            {renderGauge(memAvg, t('clusterAverage', '集群均值'))}
            <div style={{ flex: 1 }}>
              {memTop5.map(s => renderTopItem(s.name, s.memUsage ?? 0))}
              {memTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
          </div>
        </div>

        {/* 磁盘使用率 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('diskUsage', '磁盘使用率')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: `${badgeLabel(diskAvg).color}33`, color: badgeLabel(diskAvg).color,
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
            {renderGauge(diskAvg, t('clusterAverage', '集群均值'))}
            <div style={{ flex: 1 }}>
              {diskTop5.map(s => renderTopItem(s.name, s.diskUsage ?? 0))}
              {diskTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
          </div>
        </div>

        {/* 系统负载 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('systemLoad5min', '系统负载（5分钟）')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: 'rgba(59, 130, 246, 0.2)', color: theme.primary,
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
            {/* 负载仪表盘 - 用相对于核数的百分比 */}
            {(() => {
              const avgCores = avg(serverStatuses.map(s => s.cpuCores))
              const loadPct = avgCores > 0 ? Math.min((loadAvg / avgCores) * 100, 100) : 0
              return renderGauge(loadPct, t('loadPerCore', '负载/核数'))
            })()}
            <div style={{ flex: 1 }}>
              {loadTop5.map(s => renderTopItem(s.name, s.load5 ?? 0, ''))}
              {loadTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
          </div>
        </div>

        {/* 磁盘读写 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('diskIO', '磁盘读写')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: 'rgba(139, 92, 246, 0.2)', color: '#8b5cf6',
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', gap: 20 }}>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, color: theme.textSecondary, marginBottom: 12 }}>{t('read', '读取')}</div>
              {diskReadTop5.map(s => renderRateItem(s.name, s.diskRead, formatBytesRate, getDiskRateColor, true))}
              {diskReadTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
            <div style={{ width: 1, background: theme.border }} />
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, color: theme.textSecondary, marginBottom: 12 }}>{t('write', '写入')}</div>
              {diskWriteTop5.map(s => renderRateItem(s.name, s.diskWrite, formatBytesRate, getDiskRateColor, true))}
              {diskWriteTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
          </div>
        </div>

        {/* 网络带宽 */}
        <div style={cardStyle}>
          <div style={titleStyle}>
            {t('networkBandwidth', '网络带宽')}
            <span style={{
              fontSize: 11, padding: '2px 8px', borderRadius: 10,
              background: 'rgba(16, 185, 129, 0.2)', color: '#10b981',
            }}>{t('top5', 'Top 5')}</span>
          </div>
          <div style={{ display: 'flex', gap: 20 }}>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, color: theme.textSecondary, marginBottom: 12 }}>{t('download', '下载')}</div>
              {netDownTop5.map(s => renderRateItem(s.name, s.netDown, formatBitsRate, getBandwidthColor, false))}
              {netDownTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
            </div>
            <div style={{ width: 1, background: theme.border }} />
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, color: theme.textSecondary, marginBottom: 12 }}>{t('upload', '上传')}</div>
              {netUpTop5.map(s => renderRateItem(s.name, s.netUp, formatBitsRate, getBandwidthColor, false))}
              {netUpTop5.length === 0 && <div style={{ color: theme.textSecondary, fontSize: 13 }}>{tc('noData', '暂无数据')}</div>}
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
