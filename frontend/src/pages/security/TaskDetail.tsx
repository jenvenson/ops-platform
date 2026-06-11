// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal, Card, Descriptions, Table, Tag, Space, Button, Progress, Typography, Tabs, message, Dropdown, Row, Col, Statistic, Drawer, Select, Alert, Empty, Input, Spin } from 'antd'
import { ExportOutlined, ReloadOutlined, FileTextOutlined, FileOutlined, DownOutlined, SafetyOutlined, CloudServerOutlined, WarningOutlined, InfoCircleOutlined, TableOutlined, EyeOutlined } from '@ant-design/icons'
import { securityAPI, SecurityScanTask, SecurityAsset, SecurityVulnerability, SecurityScanTarget, SecurityVulnerabilityDetailResponse, SecurityScanFindingOccurrence, SecurityScanEvidence } from '../../api/security'
import { getConfidenceTag, getConfidenceValue, getFindingSourceTag, getFindingSourceValue, getMatchModeValue, getPrimaryCVE, getRiskCategory, getScanMethodLabel, getVerificationNoteValue, getVerificationStatusTag, getVerificationStatusValue, getVerifiedAtValue, hasKnowledgeLink, isCandidateFinding, isInventoryFinding } from './vulnerabilityPresentation'
import i18next from '../../i18n'
import { formatDateTime } from '../../utils/dateFormat'

const { Text, Paragraph, Link } = Typography
const { Option } = Select
const { TextArea } = Input

interface TaskDetailProps {
  task: SecurityScanTask
  onClose: () => void
  onRefresh: () => void
}

function getServiceStatusLabel(name: string) {
  if (!name.endsWith('?')) {
    return null
  }
  const normalized = name.slice(0, -1).toLowerCase()
  if (normalized === 'https') {
    return i18next.t('security:openButHandshakeFailed', '开放但握手失败')
  }
  return i18next.t('security:protocolNotConfirmed', '协议未确认')
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function getSnapshotNumber(snapshot: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = snapshot?.[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function getSnapshotString(snapshot: Record<string, unknown> | undefined, key: string): string | undefined {
  const value = snapshot?.[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function getSnapshotWarnings(snapshot: Record<string, unknown> | undefined): Array<Record<string, unknown>> {
  const raw = snapshot?.discovery_warnings
  if (!Array.isArray(raw)) {
    return []
  }
  return raw.filter(isObjectRecord)
}

function formatDiscoveryMode(mode?: string) {
  switch (mode) {
    case 'browser':
      return i18next.t('security:browserDiscovery', '浏览器发现')
    case 'http':
      return i18next.t('security:httpDiscovery', 'HTTP 发现')
    case 'none':
      return i18next.t('security:noAutoDiscovery', '不自动发现')
    default:
      return mode || '-'
  }
}

function formatScanProfile(profile?: string) {
  switch (profile) {
    case 'deep':
      return i18next.t('security:specialScan', '专项扫描')
    case 'standard':
      return i18next.t('security:standardScan', '标准扫描')
    default:
      return profile || '-'
  }
}

function getMetadataNumber(metadata: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = metadata?.[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function getMetadataString(metadata: Record<string, unknown> | undefined, key: string): string | undefined {
  const value = metadata?.[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

export default function TaskDetail({ task, onClose, onRefresh }: TaskDetailProps) {
  const { t } = useTranslation('security')
  const { t: tc } = useTranslation('common')

  const [taskDetail, setTaskDetail] = useState<SecurityScanTask>(task)
  const [assets, setAssets] = useState<SecurityAsset[]>([])
  const [scanTargets, setScanTargets] = useState<SecurityScanTarget[]>([])
  const [vulnerabilities, setVulnerabilities] = useState<SecurityVulnerability[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('summary')
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false)
  const [selectedVuln, setSelectedVuln] = useState<SecurityVulnerability | null>(null)
  const [selectedVulnDetail, setSelectedVulnDetail] = useState<SecurityVulnerabilityDetailResponse | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [candidateReviewLoadingId, setCandidateReviewLoadingId] = useState<number | null>(null)
  const [candidateReviewNote, setCandidateReviewNote] = useState('')
  const [vulnFilters, setVulnFilters] = useState({
    riskCategory: '',
    findingSource: '',
    confidence: '',
    matchMode: '',
    hasKnowledge: '',
  })

  const fetchData = async () => {
    setLoading(true)
    try {
      const [taskData, assetsData, targetsData, vulnsData] = await Promise.all([
        securityAPI.getTask(task.id),
        securityAPI.getTaskAssets(task.id),
        securityAPI.getTaskTargets(task.id),
        securityAPI.getTaskVulnerabilities(task.id),
      ])
      setTaskDetail(taskData)
      setAssets(assetsData)
      setScanTargets(targetsData)
      setVulnerabilities(vulnsData)
    } catch (error) {
      message.error(t('loadTaskDetailFailed', '获取任务详情失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [task.id])

  useEffect(() => {
    setTaskDetail(task)
    setActiveTab('summary')
    setScanTargets([])
    setVulnFilters({
      riskCategory: '',
      findingSource: '',
      confidence: '',
      matchMode: '',
      hasKnowledge: '',
    })
    setDetailDrawerOpen(false)
    setSelectedVuln(null)
    setSelectedVulnDetail(null)
    setDetailLoading(false)
    setCandidateReviewNote('')
  }, [task.id])

  useEffect(() => {
    setCandidateReviewNote(selectedVuln ? getVerificationNoteValue(selectedVuln) : '')
  }, [selectedVuln?.id, selectedVuln?.verification_note, selectedVuln?.review_note])

  const handleCandidateReview = async (vuln: SecurityVulnerability, reviewStatus: 'pending' | 'needs-test' | 'confirmed' | 'rejected') => {
    setCandidateReviewLoadingId(vuln.id)
    try {
      const reviewNote = selectedVuln?.id === vuln.id ? candidateReviewNote : getVerificationNoteValue(vuln)
      const updated = await securityAPI.updateVulnerabilityVerification(vuln.id, {
        verification_status: reviewStatus,
        verification_note: reviewNote?.trim() || undefined,
      })
      setVulnerabilities((prev) => prev.map((item) => item.id === vuln.id ? updated : item))
      if (selectedVuln?.id === vuln.id) {
        setSelectedVuln(updated)
      }
      if (selectedVulnDetail?.vulnerability.id === vuln.id) {
        setSelectedVulnDetail({
          ...selectedVulnDetail,
          vulnerability: updated,
        })
      }
      await fetchData()
      message.success(reviewStatus === 'confirmed' ? t('verificationReviewSuccess', '验证成功，已转入正式结果') : t('verifyStatusUpdated', '验证状态已更新'))
    } catch {
      message.error(t('verificationReviewFailed', '更新验证状态失败'))
    } finally {
      setCandidateReviewLoadingId(null)
    }
  }

  const openVulnerabilityDetail = async (vuln: SecurityVulnerability) => {
    setSelectedVuln(vuln)
    setSelectedVulnDetail(null)
    setDetailDrawerOpen(true)
    setDetailLoading(true)
    try {
      const detail = await securityAPI.getVulnerabilityDetail(vuln.id)
      setSelectedVuln(detail.vulnerability)
      setSelectedVulnDetail(detail)
    } catch {
      message.error(t('vulnDetailLoadFailed', '获取漏洞详情失败'))
    } finally {
      setDetailLoading(false)
    }
  }

  const handleExportReport = async (format: 'html' | 'json' | 'csv' = 'html') => {
    try {
      const res = await securityAPI.exportReport(taskData.id, format)
      let contentType: string
      let extension: string

      switch (format) {
        case 'json':
          contentType = 'application/json'
          extension = 'json'
          break
        case 'csv':
          contentType = 'text/csv;charset=utf-8'
          extension = 'csv'
          break
        default:
          contentType = 'text/html'
          extension = 'html'
      }

      const blob = new Blob([res as BlobPart], { type: contentType })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `security_report_${taskData.name}_${new Date().toISOString().split('T')[0]}.${extension}`
      a.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      const formatLabels: Record<string, string> = {
        html: 'HTML',
        json: 'JSON',
        csv: 'CSV',
      }
      message.error(t('exportReportFailed', '导出{{format}}报告失败', { format: formatLabels[format] || format }))
    }
  }

  const exportItems = [
    {
      key: 'html',
      label: t('htmlReport', 'HTML 报告'),
      icon: <FileOutlined />,
      onClick: () => handleExportReport('html'),
    },
    {
      key: 'json',
      label: t('jsonData', 'JSON 数据'),
      icon: <FileTextOutlined />,
      onClick: () => handleExportReport('json'),
    },
    {
      key: 'csv',
      label: t('csvDetailReport', 'CSV 详细报告（漏洞明细）'),
      icon: <TableOutlined />,
      onClick: () => handleExportReport('csv'),
    },
  ]

  const statusColors: Record<string, string> = {
    pending: 'default',
    running: 'processing',
    paused: 'warning',
    cancelled: 'default',
    completed: 'success',
    failed: 'error',
  }

  const statusLabels: Record<string, string> = {
    pending: t('status.pending', '等待中'),
    running: t('status.running', '运行中'),
    paused: t('status.paused', '已请求暂停'),
    cancelled: t('status.cancelled', '已请求取消'),
    completed: t('status.completed', '已完成'),
    failed: t('status.failed', '失败'),
  }

  const inventoryFindings = vulnerabilities.filter(isInventoryFinding)
  const actionableVulnerabilities = vulnerabilities.filter((vuln) => !isInventoryFinding(vuln))
  const candidateVulnerabilities = actionableVulnerabilities.filter(isCandidateFinding)
  const confirmedVulnerabilities = actionableVulnerabilities.filter((vuln) => !isCandidateFinding(vuln))
  const taskData = taskDetail || task
  const activeRun = taskData.current_run || taskData.latest_run
  const configSnapshot = isObjectRecord(activeRun?.config_snapshot) ? activeRun.config_snapshot : undefined
  const targetSnapshot = isObjectRecord(activeRun?.target_snapshot) ? activeRun.target_snapshot : undefined
  const summarySnapshot = isObjectRecord(activeRun?.summary_snapshot) ? activeRun.summary_snapshot : undefined
  const authSnapshot = isObjectRecord(configSnapshot?.auth) ? configSnapshot.auth : undefined

  // 计算漏洞统计
  const vulnStats = {
    high: confirmedVulnerabilities.filter(v => v.severity === 'critical' || v.severity === 'high').length,
    medium: confirmedVulnerabilities.filter(v => v.severity === 'medium').length,
    low: confirmedVulnerabilities.filter(v => v.severity === 'low').length,
    info: confirmedVulnerabilities.filter(v => v.severity === 'info').length,
  }
  const totalVulns = vulnStats.high + vulnStats.medium + vulnStats.low + vulnStats.info
  const riskCategoryStats = confirmedVulnerabilities.reduce(
    (acc, vuln) => {
      const code = getRiskCategory(vuln).code
      if (code === 'cve_risk') acc.cve += 1
      else if (code === 'config_risk') acc.config += 1
      else acc.generic += 1
      return acc
    },
    { cve: 0, config: 0, generic: 0 },
  )
  const dispositionStats = confirmedVulnerabilities.reduce(
    (acc, vuln) => {
      if (vuln.status === 'open') acc.open += 1
      else if (vuln.status === 'acknowledged') acc.acknowledged += 1
      else if (vuln.status === 'fixed') acc.fixed += 1
      else if (vuln.status === 'ignored') acc.ignored += 1
      return acc
    },
    { open: 0, acknowledged: 0, fixed: 0, ignored: 0 },
  )
  const handledCount = dispositionStats.acknowledged + dispositionStats.fixed + dispositionStats.ignored

  // 计算风险评分 (高危权重最高)
  const riskScore = totalVulns > 0
    ? Math.round((vulnStats.high * 10 + vulnStats.medium * 5 + vulnStats.low * 2) / totalVulns * 10)
    : 0
  const matchesVulnerabilityFilters = (vuln: SecurityVulnerability) => {
    if (vulnFilters.riskCategory && getRiskCategory(vuln).code !== vulnFilters.riskCategory) {
      return false
    }
    if (vulnFilters.findingSource && getFindingSourceValue(vuln) !== vulnFilters.findingSource) {
      return false
    }
    if (vulnFilters.confidence && getConfidenceValue(vuln) !== vulnFilters.confidence) {
      return false
    }
    if (vulnFilters.matchMode && getMatchModeValue(vuln) !== vulnFilters.matchMode) {
      return false
    }
    if (vulnFilters.hasKnowledge === 'true' && !hasKnowledgeLink(vuln)) {
      return false
    }
    if (vulnFilters.hasKnowledge === 'false' && hasKnowledgeLink(vuln)) {
      return false
    }
    return true
  }
  const filteredVulnerabilities = actionableVulnerabilities.filter((vuln) => {
    if (isCandidateFinding(vuln)) {
      return false
    }
    return matchesVulnerabilityFilters(vuln)
  })
  const filteredCandidateVulnerabilities = candidateVulnerabilities.filter(matchesVulnerabilityFilters)
  const filteredInventoryFindings = inventoryFindings.filter(matchesVulnerabilityFilters)
  const resultComparisonRows = [
    {
      key: 'confirmed',
      category: t('confirmedResults', '正式结果'),
      count: confirmedVulnerabilities.length,
      metric: `${vulnStats.high} ${t('highLabel', '高')} / ${vulnStats.medium} ${t('mediumLabel', '中')} / ${vulnStats.low + vulnStats.info} ${t('lowLabel', '低')}`,
      guidance: t('confirmedGuidance', '进入风险统计、处置和导出主结论'),
    },
    {
      key: 'verification',
      category: t('pendingVerification', '待验证'),
      count: candidateVulnerabilities.length,
      metric: t('candidateMetric', '线索核验'),
      guidance: t('candidateGuidance', '默认不进入正式风险，建议结合版本、配置和人工验证确认'),
    },
    {
      key: 'inventory',
      category: t('assetInfo', '资产信息'),
      count: inventoryFindings.length,
      metric: t('inventoryMetric', '不计风险'),
      guidance: t('inventoryGuidance', '用于资产盘点和后续扫描路由'),
    },
  ]

  // 获取唯一IP数量
  const uniqueIPs = [...new Set(assets.map(a => a.ip))].length
  const uniqueServices = [...new Set(assets.map((asset) => (asset.service_name || '').replace(/\?$/, '').trim()).filter(Boolean))].length
  const isWebScan = taskData.scan_type === 'web'
  const isDiscoveryTask = taskData.scan_type === 'port' || String(taskData.scan_type) === 'host'
  const targetEntries = taskData.target
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  const webEntryCount = getSnapshotNumber(summarySnapshot, 'entry_count')
    ?? getSnapshotNumber(targetSnapshot, 'entry_count')
    ?? getSnapshotNumber(configSnapshot, 'entry_count')
    ?? targetEntries.length
  const webDiscoveredCount = getSnapshotNumber(summarySnapshot, 'discovered_count')
    ?? getSnapshotNumber(targetSnapshot, 'discovered_count')
    ?? (scanTargets.length > 0 ? scanTargets.length : targetEntries.length)
  const webScannedCount = getSnapshotNumber(summarySnapshot, 'scanned_targets')
    ?? activeRun?.scanned_targets
    ?? (scanTargets.length > 0 ? scanTargets.length : targetEntries.length)
  const webSkippedCount = getSnapshotNumber(summarySnapshot, 'skipped_target_count')
    ?? getSnapshotNumber(targetSnapshot, 'skipped_target_count')
    ?? 0
  const webRuleOnlyCount = getSnapshotNumber(summarySnapshot, 'rule_only_target_count')
    ?? getSnapshotNumber(targetSnapshot, 'rule_only_target_count')
    ?? 0
  const summaryWarnings = getSnapshotWarnings(summarySnapshot)
  const targetWarnings = getSnapshotWarnings(targetSnapshot)
  const discoveryWarnings = summaryWarnings.length > 0 ? summaryWarnings : targetWarnings
  const browserFallbackCount = getSnapshotNumber(summarySnapshot, 'browser_fallback_count')
    ?? discoveryWarnings.length
  const discoveryMode = getSnapshotString(summarySnapshot, 'discovery_mode')
    ?? getSnapshotString(configSnapshot, 'discovery_mode')
  const scanProfile = getSnapshotString(configSnapshot, 'scan_profile')
  const authMode = getSnapshotString(authSnapshot, 'auth_mode')
  const verificationTargetBudget = getSnapshotNumber(configSnapshot, 'verification_max_targets')
  const discoveryBudget = getSnapshotNumber(configSnapshot, 'discovery_max_urls')
  const webSummary = {
    entries: webEntryCount,
    discovered: webDiscoveredCount,
    scanned: webScannedCount,
    skipped: webSkippedCount,
    ruleOnly: webRuleOnlyCount,
    mode: taskData.status === 'completed' ? t('status.completed', '已完成') : statusLabels[taskData.status] || taskData.status,
    auth: authMode === 'advanced' ? t('autoAuthFlow', '自动生成认证流程') : (authMode || t('execByTaskConfig', '按任务配置执行')),
    strategy: formatScanProfile(scanProfile),
    discoveryMode: formatDiscoveryMode(discoveryMode),
    fallback: browserFallbackCount,
  }
  const hasTargetTree = isWebScan && scanTargets.length > 0
  const scanEntryRows = hasTargetTree
    ? (() => {
        type ScanEntryRow = {
          key: string
          entry: string
          kind: string
          source: string
          depth: number
          host: string
          port: number | string
          protocol: string
          service: string
          status: string
          children?: ScanEntryRow[]
        }

        const rowMap = new Map<number, ScanEntryRow>()
        const parentById = new Map<number, number | undefined>()
        const roots: ScanEntryRow[] = []

        const resolveDepth = (target: SecurityScanTarget, seen = new Set<number>()): number => {
          const metadataDepth = getMetadataNumber(target.metadata, 'depth')
          if (typeof metadataDepth === 'number') {
            return metadataDepth
          }
          if (!target.parent_target_id || seen.has(target.id)) {
            return 0
          }
          seen.add(target.id)
          const parent = scanTargets.find((item) => item.id === target.parent_target_id)
          if (!parent) {
            return 0
          }
          return resolveDepth(parent, seen) + 1
        }

        for (const target of scanTargets) {
          const source = getMetadataString(target.metadata, 'source') || target.discovery_source || '-'
          const entry = getMetadataString(target.metadata, 'url') || target.target_url || target.normalized_target
          const row: ScanEntryRow = {
            key: String(target.id),
            entry,
            kind: target.target_kind || '-',
            source,
            depth: resolveDepth(target),
            host: target.host || '-',
            port: typeof target.port === 'number' ? target.port : '-',
            protocol: target.scheme ? target.scheme.toUpperCase() : '-',
            service: target.service_name || '-',
            status: target.status || '-',
          }
          rowMap.set(target.id, row)
          parentById.set(target.id, target.parent_target_id)
        }

        for (const target of scanTargets) {
          const row = rowMap.get(target.id)
          if (!row) {
            continue
          }
          if (target.parent_target_id && rowMap.has(target.parent_target_id)) {
            const parent = rowMap.get(target.parent_target_id)
            if (parent) {
              parent.children = parent.children || []
              parent.children.push(row)
            }
            continue
          }
          roots.push(row)
        }

        const sortRows = (rows: ScanEntryRow[]) => {
          rows.sort((left, right) => {
            if (left.depth !== right.depth) {
              return left.depth - right.depth
            }
            return left.entry.localeCompare(right.entry, 'zh-CN')
          })
          for (const row of rows) {
            if (row.children && row.children.length > 0) {
              sortRows(row.children)
            }
          }
        }

        sortRows(roots)
        return roots
      })()
    : targetEntries.map((entry, index) => {
        const matchedAsset = assets.find((asset) => entry.includes(asset.ip) || asset.ip.includes(entry))
        return {
          key: `${entry}-${index}`,
          entry,
          kind: index === 0 ? 'url' : '-',
          source: 'manual',
          depth: 0,
          host: matchedAsset?.ip || '-',
          port: matchedAsset?.port || '-',
          protocol: matchedAsset?.protocol || '-',
          service: matchedAsset?.service_name || '-',
          status: '-',
        }
      })
  const scanEntryCount = hasTargetTree ? scanTargets.length : scanEntryRows.length
  const showWebFollowupHint = isWebScan && totalVulns === 0
  const webFollowupHint = webSummary.discovered > 1
    ? t('webFollowupMultiHint', '本次已自动发现并扫描多个入口，但未命中模板规则。可优先对当前入口继续做专项扫描，或针对重点接口复测认证、参数和权限差异。')
    : t('webFollowupSingleHint', '本次未命中模板规则。若当前目标只是单一入口，可改为自动发现后扫描，或补充更具体的业务 URL / API URL 再复测。')
  const discoverySummary = {
    targets: targetEntries.length,
    hosts: uniqueIPs,
    ports: assets.length,
    services: uniqueServices,
  }
  const progressUnit = isWebScan ? t('entryUnit', '入口') : isDiscoveryTask ? t('targetUnit', '目标') : 'IP'

  // 按IP统计漏洞数量
  const ipVulnCount: Record<string, { high: number; medium: number; low: number }> = {}
  confirmedVulnerabilities.forEach(v => {
    if (!ipVulnCount[v.ip]) {
      ipVulnCount[v.ip] = { high: 0, medium: 0, low: 0 }
    }
    if (v.severity === 'critical' || v.severity === 'high') ipVulnCount[v.ip].high++
    else if (v.severity === 'medium') ipVulnCount[v.ip].medium++
    else if (v.severity === 'low') ipVulnCount[v.ip].low++
  })
  const vulnOccurrences = selectedVulnDetail?.occurrences || []
  const vulnEvidences = selectedVulnDetail?.evidences || []

  const assetColumns = [
    {
      title: isWebScan ? t('resolvedHost', '解析主机') : t('ipAddress', 'IP'),
      dataIndex: 'ip',
      key: 'ip',
      width: 180,
    },
    {
      title: t('port', '端口'),
      dataIndex: 'port',
      key: 'port',
      width: 80,
      render: (port: number) => port || '-',
    },
    {
      title: t('protocol', '协议'),
      dataIndex: 'protocol',
      key: 'protocol',
      width: 80,
    },
    {
      title: t('service', '服务'),
      dataIndex: 'service_name',
      key: 'service',
      width: 100,
      render: (service: string) => {
        if (!service) return '-'
        const unverified = service.endsWith('?')
        const display = unverified ? service.slice(0, -1) : service
        const statusLabel = getServiceStatusLabel(service)
        return (
          <Space size={4}>
            <span>{display}</span>
            {unverified && statusLabel && <Tag color="cyan">{statusLabel}</Tag>}
          </Space>
        )
      },
    },
    {
      title: t('version', '版本'),
      dataIndex: 'version',
      key: 'version',
      width: 200,
      render: (version: string, record: SecurityAsset) => {
        // 版本信息可能在 version 或 banner 字段
        const ver = version || record.banner || '-'
        return ver
      },
    },
  ]

  const scanEntryColumns = [
    {
      title: t('scanEntry', '扫描入口'),
      dataIndex: 'entry',
      key: 'entry',
      ellipsis: true,
      render: (value: string) => <Text copyable>{value}</Text>,
    },
    {
      title: t('kindType', '类型'),
      dataIndex: 'kind',
      key: 'kind',
      width: 90,
      render: (value: string) => <Tag color={value === 'api' ? 'blue' : value === 'page' ? 'green' : 'default'}>{value || '-'}</Tag>,
    },
    {
      title: t('source', '来源'),
      dataIndex: 'source',
      key: 'source',
      width: 110,
      render: (value: string) => {
        const color = value === 'browser' || value === 'browser-request' || value === 'browser-dom'
          ? 'cyan'
          : value === 'http' || value === 'html' || value === 'script'
            ? 'gold'
            : value === 'manual' || value === 'entry'
              ? 'blue'
              : value === 'auth'
                ? 'purple'
                : 'default'
        return <Tag color={color}>{value || '-'}</Tag>
      },
    },
    {
      title: t('depth', '深度'),
      dataIndex: 'depth',
      key: 'depth',
      width: 70,
    },
    {
      title: t('host', '主机'),
      dataIndex: 'host',
      key: 'host',
      width: 140,
    },
    {
      title: t('port', '端口'),
      dataIndex: 'port',
      key: 'port',
      width: 80,
    },
    {
      title: t('protocol', '协议'),
      dataIndex: 'protocol',
      key: 'protocol',
      width: 80,
    },
    {
      title: t('service', '服务'),
      dataIndex: 'service',
      key: 'service',
      width: 100,
    },
    {
      title: tc('status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (value: string) => {
        const color = value === 'completed' ? 'green' : value === 'pending' ? 'default' : value === 'running' ? 'processing' : 'default'
        return <Tag color={color}>{value || '-'}</Tag>
      },
    },
  ]

  const occurrenceColumns = [
    {
      title: t('target', '目标'),
      key: 'target',
      render: (_: unknown, record: SecurityScanFindingOccurrence) => record.target?.target_url || getMetadataString(record.metadata, 'vuln_url') || '-',
    },
    {
      title: t('source', '来源'),
      dataIndex: 'finding_source',
      key: 'finding_source',
      width: 110,
      render: (value: string) => <Tag color={value === 'web-rule' ? 'gold' : value === 'web-template' ? 'blue' : 'default'}>{value || '-'}</Tag>,
    },
    {
      title: t('match', '匹配'),
      dataIndex: 'match_mode',
      key: 'match_mode',
      width: 100,
      render: (value: string) => value || '-',
    },
    {
      title: t('firstHit', '首次命中'),
      dataIndex: 'first_seen_at',
      key: 'first_seen_at',
      width: 160,
      render: (value?: string) => formatDateTime(value),
    },
    {
      title: t('lastHit', '最近命中'),
      dataIndex: 'last_seen_at',
      key: 'last_seen_at',
      width: 160,
      render: (value?: string) => formatDateTime(value),
    },
    {
      title: t('evidence', '证据'),
      dataIndex: 'evidence_count',
      key: 'evidence_count',
      width: 70,
      render: (value: number) => value ?? 0,
    },
  ]

  const vulnColumns = [
    {
      title: t('severityLevel', '严重程度'),
      dataIndex: 'severity',
      key: 'severity',
      width: 90,
      render: (sev: string) => (
        <Tag color={sev === 'high' ? 'red' : sev === 'medium' ? 'orange' : sev === 'low' ? 'blue' : 'default'}>
          {sev?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: t('vulnTitle', '漏洞标题'),
      dataIndex: 'title',
      key: 'title',
      width: 240,
      ellipsis: true,
      render: (title: string) => title || '-',
    },
    {
      title: t('cve', 'CVE'),
      dataIndex: 'primary_cve_id',
      key: 'cve_id',
      width: 140,
      render: (_: string, record: SecurityVulnerability) => getPrimaryCVE(record) || '-',
    },
    {
      title: t('target', '目标'),
      key: 'target',
      width: 140,
      render: (_: unknown, record: SecurityVulnerability) => `${record.ip}:${record.port}`,
    },
    {
      title: t('cvss', 'CVSS'),
      dataIndex: 'cvss_score',
      key: 'cvss_score',
      width: 70,
      render: (score: number) => score?.toFixed(1) || '-',
    },
    {
      title: t('riskCategory', '风险类别'),
      key: 'risk_category',
      width: 110,
      render: (_: unknown, record: SecurityVulnerability) => {
        const category = getRiskCategory(record)
        return <Tag color={category.color}>{category.text}</Tag>
      },
    },
    {
      title: t('findingSource', '结果来源'),
      key: 'finding_source',
      width: 110,
      render: (_: unknown, record: SecurityVulnerability) => {
        const source = getFindingSourceTag(record)
        return <Tag color={source.color}>{source.text}</Tag>
      },
    },
    {
      title: t('confidence', '置信度'),
      key: 'confidence',
      width: 90,
      render: (_: unknown, record: SecurityVulnerability) => {
        const confidence = getConfidenceTag(record)
        return <Tag color={confidence.color}>{confidence.text}</Tag>
      },
    },
    {
      title: tc('status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const colors: Record<string, string> = {
          open: 'red',
          acknowledged: 'orange',
          fixed: 'green',
          ignored: 'default',
        }
        const labels: Record<string, string> = {
          open: t('statusLabel.open', '待处理'),
          acknowledged: t('statusLabel.acknowledged', '已确认'),
          fixed: t('statusLabel.fixed', '已修复'),
          ignored: t('statusLabel.ignored', '已忽略'),
        }
        return <Tag color={colors[status]}>{labels[status] || status}</Tag>
      },
    },
    {
      title: t('verificationStatus', '验证状态'),
      key: 'verification_status',
      width: 110,
      render: (_: unknown, record: SecurityVulnerability) => {
        if (!isCandidateFinding(record)) {
          return '-'
        }
        const review = getVerificationStatusTag(record)
        return <Tag color={review.color}>{review.text}</Tag>
      },
    },
    {
      title: t('action', '操作'),
      key: 'action',
      width: 190,
      render: (_: unknown, record: SecurityVulnerability) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => {
              openVulnerabilityDetail(record)
            }}
          >
            {t('detail', '详情')}
          </Button>
          {isCandidateFinding(record) && (
            <Select
              size="small"
              value={getVerificationStatusValue(record)}
              style={{ width: 100 }}
              loading={candidateReviewLoadingId === record.id}
              onChange={(val) => handleCandidateReview(record, val as 'pending' | 'needs-test' | 'confirmed' | 'rejected')}
              options={[
                { value: 'pending', label: t('unverified', '未验证') },
                { value: 'needs-test', label: t('retesting', '复测中') },
                { value: 'confirmed', label: t('verified', '验证成功') },
                { value: 'rejected', label: t('verificationFailed', '验证失败') },
              ]}
            />
          )}
        </Space>
      ),
    },
  ]

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <SafetyOutlined />
          <span>{t('scanTaskDetail', '扫描任务详情')} - {taskData.name}</span>
          <Tag color={statusColors[taskData.status]}>{statusLabels[taskData.status]}</Tag>
        </div>
      }
      open={!!task}
      onCancel={onClose}
      width={1100}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button icon={<ReloadOutlined />} onClick={() => { fetchData(); onRefresh(); }}>
            {tc('refresh', '刷新')}
          </Button>
          <Space>
            <Button onClick={onClose}>{tc('close', '关闭')}</Button>
            {taskData.status === 'completed' && (
              <Dropdown menu={{ items: exportItems }} trigger={['click']}>
                <Button type="primary" icon={<ExportOutlined />}>
                  {t('exportReport', '导出报告')} <DownOutlined />
                </Button>
              </Dropdown>
            )}
          </Space>
        </div>
      }
    >
      {/* 任务基本信息 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Descriptions column={4} size="small">
          <Descriptions.Item label={t('taskName', '任务名称')}>{taskData.name}</Descriptions.Item>
          <Descriptions.Item label={t('taskType', '任务类型')}>
            {isDiscoveryTask ? <Tag color="blue">{t('portScan', '资产发现')}</Tag> : isWebScan ? <Tag color="green">{t('webVuln', '网站漏洞')}</Tag> : <Tag color="orange">{t('hostVuln', '主机漏洞')}</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label={t('scanTarget', '扫描目标')}>
            <Text copyable>{taskData.target}</Text>
          </Descriptions.Item>
          <Descriptions.Item label={t('createTime', '创建时间')}>
            {taskData.created_at ? formatDateTime(taskData.created_at) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('completeTime', '完成时间')}>
            {taskData.completed_at ? formatDateTime(taskData.completed_at) : '-'}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {isDiscoveryTask && (
        <Card size="small" style={{ marginBottom: 16 }} title={t('assetDiscoverySummary', '资产发现摘要')}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title={t('discoveryScanTargets', '扫描目标')} value={discoverySummary.targets} suffix={t('targetUnitSuffix', '个')} />
            </Col>
            <Col span={6}>
              <Statistic title={t('identifiedHosts', '识别主机')} value={discoverySummary.hosts} suffix={t('hostUnitSuffix', '台')} />
            </Col>
            <Col span={6}>
              <Statistic title={t('openPorts', '开放端口')} value={discoverySummary.ports} suffix={t('portUnitSuffix', '个')} />
            </Col>
            <Col span={6}>
              <Statistic title={t('serviceTypes', '服务类型')} value={discoverySummary.services} suffix={t('serviceTypeUnitSuffix', '种')} />
            </Col>
          </Row>
          <Alert
            style={{ marginTop: 16 }}
            type="info"
            showIcon
            message={t('taskNoteHeader', '任务说明')}
            description={t('discoverySummaryDesc', '资产发现任务仅执行开放端口探测和服务识别，用于资产盘点，不包含漏洞检测。可从结果中挑选重点主机，再发起主机漏洞或网站漏洞扫描。')}
          />
        </Card>
      )}

      {isWebScan && (
        <Card size="small" style={{ marginBottom: 16 }} title={t('webScanSummary', 'Web 扫描摘要')}>
          <Row gutter={16}>
            <Col span={4}>
              <Statistic title={t('inputEntries', '输入入口')} value={webSummary.entries} suffix={t('entryUnitSuffix', '个')} />
            </Col>
            <Col span={4}>
              <Statistic title={t('discoveredTargets', '发现目标')} value={webSummary.discovered} suffix={t('targetUnitSuffix', '个')} />
            </Col>
            <Col span={4}>
              <Statistic title={t('actualScanned', '实际扫描')} value={webSummary.scanned} suffix={t('targetUnitSuffix', '个')} />
            </Col>
            <Col span={4}>
              <Statistic title={t('ruleDetection', '规则检测')} value={webSummary.ruleOnly} suffix={t('targetUnitSuffix', '个')} />
            </Col>
            <Col span={4}>
              <Statistic title={t('budgetSkipped', '预算跳过')} value={webSummary.skipped} suffix={t('targetUnitSuffix', '个')} />
            </Col>
            <Col span={4}>
              <Statistic title={t('taskStatus', '任务状态')} value={webSummary.mode} />
            </Col>
          </Row>
          <div style={{ marginTop: 16, display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            <Tag color="blue">{webSummary.strategy}</Tag>
            <Tag color="cyan">{webSummary.discoveryMode}</Tag>
            <Tag color="geekblue">{webSummary.auth}</Tag>
            {typeof verificationTargetBudget === 'number' && <Tag color="gold">{t('verificationBudgetTargets', '验证预算 {{count}} 目标', { count: verificationTargetBudget })}</Tag>}
            {typeof discoveryBudget === 'number' && <Tag>{t('discoveryBudgetUrls', '发现预算 {{count}} URL', { count: discoveryBudget })}</Tag>}
            {webSummary.fallback > 0 && <Tag color="orange">{t('fallbackCount', '回退 {{count}} 次', { count: webSummary.fallback })}</Tag>}
          </div>
          <Alert
            style={{ marginTop: 16 }}
            type="info"
            showIcon
            message={t('scanModeExplanation', '扫描方式说明')}
            description={t('scanModeExplanationDesc', '当前详情优先展示运行期摘要。它会区分输入入口、发现目标、实际扫描目标、仅规则检测目标以及因预算跳过的目标，便于判断覆盖边界。')}
          />
          {webSummary.ruleOnly > 0 && (
            <Alert
              style={{ marginTop: 12 }}
              type="warning"
              showIcon
              message={t('partialTargetsOnlyRule', '部分目标只执行了规则检测')}
              description={t('partialTargetsOnlyRuleDesc', '当前任务有 {{ruleOnly}} 个低价值目标未跑 full Nuclei，只做了内置规则检测。这是标准扫描的预算控制，不等同于漏扫失败。', { ruleOnly: webSummary.ruleOnly })}
            />
          )}
          {webSummary.skipped > 0 && (
            <Alert
              style={{ marginTop: 12 }}
              type="warning"
              showIcon
              message={t('targetsSkippedByBudget', '部分发现目标因预算被跳过')}
              description={t('targetsSkippedByBudgetDesc', '当前任务有 {{skipped}} 个低优先级目标未进入验证阶段。若这些目标属于重点接口，建议改用专项扫描复测。', { skipped: webSummary.skipped })}
            />
          )}
          {webSummary.fallback > 0 && (
            <Alert
              style={{ marginTop: 12 }}
              type="warning"
              showIcon
              message={t('browserFallbackOccurred', '浏览器发现发生回退')}
              description={t('browserFallbackOccurredDesc', '本次有 {{fallback}} 个入口因 browser helper 不可达或失败回退为 HTTP discovery，扫描链路可用，但覆盖面通常会比浏览器态更窄。', { fallback: webSummary.fallback })}
            />
          )}
        </Card>
      )}

      {(taskData.status === 'paused' || taskData.status === 'cancelled') && (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message={taskData.status === 'paused' ? t('pauseRequestNote', '暂停请求为阶段性生效') : t('cancelRequestNote', '取消请求为阶段性生效')}
          description={taskData.message || t('taskStopNoteDesc', '当前任务会在现有步骤结束后停止。若需继续执行，请重新创建任务。')}
        />
      )}

      {/* 运行中显示进度 */}
      {taskData.status === 'running' && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Progress
              percent={taskData.progress}
              status="active"
              strokeColor={{
                '0%': '#108ee9',
                '100%': '#87d068',
              }}
              style={{ flex: 1 }}
            />
            <Text type="secondary">
              {taskData.message || t('scanning', '扫描中...')} ({taskData.scanned_ips}/{taskData.total_ips} {progressUnit})
            </Text>
          </div>
        </Card>
      )}

      {/* 已完成显示完整统计 */}
      {taskData.status === 'completed' && !isDiscoveryTask && (
        <>
          {/* 风险概览卡片 */}
          <Card size="small" style={{ marginBottom: 16 }} title={t('riskOverview', '风险概览')}>
            <Row gutter={16}>
              <Col span={3}>
                <Statistic
                  title={isWebScan ? t('actualScanLabel', '实际扫描') : t('scanHostLabel', '扫描主机')}
                  value={isWebScan ? webSummary.scanned : uniqueIPs}
                  prefix={<CloudServerOutlined style={{ color: '#1890ff' }} />}
                  suffix={isWebScan ? t('entryUnitSuffix', '个') : t('hostUnitSuffix', '台')}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={isWebScan ? t('scanAssetLabel', '扫描资产') : t('openPorts', '开放端口')}
                  value={assets.length}
                  prefix={<InfoCircleOutlined style={{ color: '#52c41a' }} />}
                  suffix={isWebScan ? t('assetUnitSuffix', '条') : t('portUnitSuffix', '个')}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('realVulnerabilities', '真实漏洞')}
                  value={totalVulns}
                  prefix={<WarningOutlined style={{ color: '#faad14' }} />}
                  suffix={t('vulnUnitSuffix', '个')}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('pendingVerification', '待验证')}
                  value={candidateVulnerabilities.length}
                  valueStyle={{ color: '#d48806' }}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('assetIdentification', '资产识别')}
                  value={inventoryFindings.length}
                  valueStyle={{ color: '#1677ff' }}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('pending', '待处理')}
                  value={dispositionStats.open}
                  valueStyle={{ color: '#cf1322' }}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('handled', '已处置')}
                  value={handledCount}
                  valueStyle={{ color: '#389e0d' }}
                />
              </Col>
              <Col span={3}>
                <Statistic
                  title={t('riskScore', '风险评分')}
                  value={riskScore}
                  suffix="/100"
                />
              </Col>
            </Row>
            <div style={{ marginTop: 16, display: 'flex', flexWrap: 'wrap', gap: 12 }}>
              <Tag color="red">{t('cveRisk', 'CVE 风险')} {riskCategoryStats.cve}</Tag>
              <Tag color="cyan">{t('configRisk', '配置风险')} {riskCategoryStats.config}</Tag>
              <Tag>{t('genericRisk', '通用风险')} {riskCategoryStats.generic}</Tag>
              <Tag color="gold">{t('acknowledgedLabel', '已确认')} {dispositionStats.acknowledged}</Tag>
              <Tag color="green">{t('fixedLabel', '已修复')} {dispositionStats.fixed}</Tag>
              <Tag color="default">{t('ignoredLabel', '已忽略')} {dispositionStats.ignored}</Tag>
            </div>
            <Card size="small" style={{ marginTop: 16, background: '#fafafa' }} title={t('resultLayering', '结果分层')}>
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic title={t('confirmedResults', '正式结果')} value={confirmedVulnerabilities.length} valueStyle={{ color: '#cf1322' }} />
                </Col>
                <Col span={6}>
                  <Statistic title={t('pendingVerification', '待验证')} value={candidateVulnerabilities.length} valueStyle={{ color: '#d48806' }} />
                </Col>
                <Col span={6}>
                  <Statistic title={t('assetInfo', '资产信息')} value={inventoryFindings.length} valueStyle={{ color: '#1677ff' }} />
                </Col>
              </Row>
              <Alert
                style={{ marginTop: 16, marginBottom: 16 }}
                type="info"
                showIcon
                message={t('resultInterpretation', '结果解读')}
                description={t('resultInterpretationDesc', '正式结果才进入风险评分和主报表结论；待验证结果只作为核验线索；资产信息用于盘点和后续扫描，不计入风险。')}
              />
              <Table
                rowKey="key"
                size="small"
                pagination={false}
                dataSource={resultComparisonRows}
                columns={[
                  { title: t('category', '分类'), dataIndex: 'category', key: 'category', width: 120 },
                  { title: t('quantity', '数量'), dataIndex: 'count', key: 'count', width: 80 },
                  { title: t('caliber', '口径'), dataIndex: 'metric', key: 'metric', width: 160 },
                  { title: t('suggestedAction', '建议动作'), dataIndex: 'guidance', key: 'guidance' },
                ]}
              />
            </Card>
          {candidateVulnerabilities.length > 0 && (
            <Alert
              style={{ marginTop: 16 }}
              type="warning"
              showIcon
              message={t('versionMatchGroupedCandidate', '版本匹配结果已单独归入待验证')}
              description={t('versionMatchGroupedCandidateDesc', '本次扫描中有 {{count}} 条结果是根据目标开放服务的产品名、版本号或 CPE 信息，与漏洞知识库自动比对后得到的风险线索。它们默认不计入风险评分和正式结果统计；验证成功后会额外派生正式结果。', { count: candidateVulnerabilities.length })}
            />
          )}
          {inventoryFindings.length > 0 && (
            <Alert
              style={{ marginTop: 16 }}
              type="info"
              showIcon
              message={t('inventorySeparated', '资产识别结果已单独展示')}
              description={t('inventorySeparatedDesc', '本次扫描还发现 {{count}} 条服务识别/版本枚举结果，这些结果会保留用于资产盘点，但不计入风险评分和真实漏洞统计。', { count: inventoryFindings.length })}
            />
          )}
        </Card>

          {/* 风险分布 */}
          <Card size="small" style={{ marginBottom: 16 }} title={t('vulnRiskDistribution', '漏洞风险分布')}>
            <Row gutter={24}>
              <Col span={16}>
                <div style={{ marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text>{t('severityLabel.high', '高危漏洞')}</Text>
                    <Text strong style={{ color: '#ff4d4f' }}>{vulnStats.high} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <Progress
                    percent={totalVulns > 0 ? Math.round(vulnStats.high / totalVulns * 100) : 0}
                    strokeColor="#ff4d4f"
                    showInfo={false}
                    size="small"
                  />
                </div>
                <div style={{ marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text>{t('severityLabel.medium', '中危漏洞')}</Text>
                    <Text strong style={{ color: '#faad14' }}>{vulnStats.medium} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <Progress
                    percent={totalVulns > 0 ? Math.round(vulnStats.medium / totalVulns * 100) : 0}
                    strokeColor="#faad14"
                    showInfo={false}
                    size="small"
                  />
                </div>
                <div style={{ marginBottom: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text>{t('severityLabel.low', '低危漏洞')}</Text>
                    <Text strong style={{ color: '#1890ff' }}>{vulnStats.low} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <Progress
                    percent={totalVulns > 0 ? Math.round(vulnStats.low / totalVulns * 100) : 0}
                    strokeColor="#1890ff"
                    showInfo={false}
                    size="small"
                  />
                </div>
                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <Text>{t('infoLevel', '信息级')}</Text>
                    <Text strong style={{ color: '#999' }}>{vulnStats.info} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <Progress
                    percent={totalVulns > 0 ? Math.round(vulnStats.info / totalVulns * 100) : 0}
                    strokeColor="#999"
                    showInfo={false}
                    size="small"
                  />
                </div>
              </Col>
              <Col span={8}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag color="red" style={{ margin: 0 }}>{t('severityLabel.high', '高危')}</Tag>
                    <Text>{vulnStats.high} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag color="orange" style={{ margin: 0 }}>{t('severityLabel.medium', '中危')}</Tag>
                    <Text>{vulnStats.medium} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag color="blue" style={{ margin: 0 }}>{t('severityLabel.low', '低危')}</Tag>
                    <Text>{vulnStats.low} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Tag color="default" style={{ margin: 0 }}>{t('vulnSeverityInfo', '信息')}</Tag>
                    <Text>{vulnStats.info} {t('vulnUnitSuffix', '个')}</Text>
                  </div>
                </div>
              </Col>
            </Row>
          </Card>

          {/* 主机风险列表 */}
          {Object.keys(ipVulnCount).length > 0 && (
            <Card size="small" style={{ marginBottom: 16 }} title={t('hostRiskList', '主机风险列表')}>
              <Table
                dataSource={Object.entries(ipVulnCount).map(([ip, counts]) => ({
                  ip,
                  high: counts.high,
                  medium: counts.medium,
                  low: counts.low,
                  total: counts.high + counts.medium + counts.low,
                }))}
                rowKey="ip"
                size="small"
                pagination={false}
                columns={[
                  { title: t('ipAddrLabel', 'IP 地址'), dataIndex: 'ip', key: 'ip' },
                  {
                    title: t('severityLabel.high', '高危'),
                    dataIndex: 'high',
                    key: 'high',
                    width: 80,
                    render: (v: number) => v > 0 ? <Text type="danger" strong>{v}</Text> : '-',
                  },
                  {
                    title: t('severityLabel.medium', '中危'),
                    dataIndex: 'medium',
                    key: 'medium',
                    width: 80,
                    render: (v: number) => v > 0 ? <Text type="warning">{v}</Text> : '-',
                  },
                  {
                    title: t('severityLabel.low', '低危'),
                    dataIndex: 'low',
                    key: 'low',
                    width: 80,
                    render: (v: number) => v > 0 ? <Text style={{ color: '#1890ff' }}>{v}</Text> : '-',
                  },
                  {
                    title: t('totalLabel', '总计'),
                    dataIndex: 'total',
                    key: 'total',
                    width: 80,
                    render: (v: number) => <Text strong>{v}</Text>,
                  },
                ]}
              />
            </Card>
          )}

          {/* 版本信息 */}
          {(taskData.nuclei_version || taskData.template_version) && (
            <Card size="small" style={{ marginBottom: 16 }} title={t('scanEngineInfo', '扫描引擎')}>
              <Space size="large">
                {taskData.nuclei_version && (
                  <div>
                    <Text type="secondary">Nuclei</Text>
                    <div><Text code>{taskData.nuclei_version}</Text></div>
                  </div>
                )}
                {taskData.template_version && (
                  <div>
                    <Text type="secondary">{t('templateVersion', '模板版本')}</Text>
                    <div><Text code>{taskData.template_version}</Text></div>
                  </div>
                )}
              </Space>
            </Card>
          )}
        </>
      )}

      {/* Tab 页签 */}
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          ...(isDiscoveryTask ? [{
            key: 'summary',
            label: `${t('discoveryResults', '发现结果')} (${assets.length})`,
            children: (
              <>
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">
                    {t('discoveryResultsSummary', '共识别到 {{assetCount}} 条端口和服务记录，覆盖 {{hostCount}} 台主机。', { assetCount: assets.length, hostCount: uniqueIPs })}
                  </Text>
                </div>
                <Table
                  columns={assetColumns}
                  dataSource={assets}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                  locale={{
                    emptyText: (
                      <Empty
                        description={t('noOpenPorts', '未发现开放端口')}
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      >
                        <Text type="secondary">
                          {t('noOpenPortsDesc', '当前目标未识别到开放端口或服务响应，可检查目标 IP 是否可达、网段是否正确或稍后重试。')}
                        </Text>
                      </Empty>
                    ),
                  }}
                />
              </>
            ),
          }] : [{
            key: 'summary',
            label: `${t('vulnerabilityList', '漏洞列表')} (${filteredVulnerabilities.length})`,
            children: (
              <>
                {candidateVulnerabilities.length > 0 && (
                <Alert
                  style={{ marginBottom: 12 }}
                  type="warning"
                  showIcon
                    message={t('candidateResultsCollapsed', '待验证结果已单独折叠')}
                    description={t('candidateResultsCollapsedDesc', '本次有 {{count}} 条基于目标服务版本与漏洞知识库自动比对得到的风险线索，已移到"待验证"页签，默认不和正式结果混排。', { count: candidateVulnerabilities.length })}
                  />
                )}
                <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
                  <Space wrap>
                    <Select
                      placeholder={t('filterByRiskCategory', '风险类别')}
                      allowClear
                      style={{ width: 140 }}
                      value={vulnFilters.riskCategory || undefined}
                      onChange={(val) => setVulnFilters({ ...vulnFilters, riskCategory: val || '' })}
                    >
                      <Option value="cve_risk">{t('cveRisk', 'CVE 风险')}</Option>
                      <Option value="config_risk">{t('configRisk', '配置风险')}</Option>
                      <Option value="generic_risk">{t('genericRisk', '通用风险')}</Option>
                    </Select>
                    <Select
                      placeholder={t('filterByFindingSource', '结果来源')}
                      allowClear
                      style={{ width: 140 }}
                      value={vulnFilters.findingSource || undefined}
                      onChange={(val) => setVulnFilters({ ...vulnFilters, findingSource: val || '' })}
                    >
                      <Option value="web-template">{t('webTemplate', 'Web 模板')}</Option>
                      <Option value="web-rule">{t('webRule', 'Web 规则')}</Option>
                      <Option value="host-template">{t('serviceTemplate', '服务模板')}</Option>
                      <Option value="host-version-match">{t('versionMatch', '版本匹配')}</Option>
                      <Option value="host-manual-confirmed">{t('manualConfirmed', '人工确认')}</Option>
                      <Option value="asset-inventory">{t('assetInventory', '资产识别')}</Option>
                    </Select>
                    <Select
                      placeholder={t('filterByConfidence', '置信度')}
                      allowClear
                      style={{ width: 120 }}
                      value={vulnFilters.confidence || undefined}
                      onChange={(val) => setVulnFilters({ ...vulnFilters, confidence: val || '' })}
                    >
                      <Option value="high">{t('severityLevel.high', '高')}</Option>
                      <Option value="medium">{t('severityLevel.medium', '中')}</Option>
                      <Option value="low">{t('severityLevel.low', '低')}</Option>
                    </Select>
                    <Select
                      placeholder={t('filterByMatchMode', '匹配模式')}
                      allowClear
                      style={{ width: 140 }}
                      value={vulnFilters.matchMode || undefined}
                      onChange={(val) => setVulnFilters({ ...vulnFilters, matchMode: val || '' })}
                    >
                      <Option value="template">{t('templateMatch', '模板命中')}</Option>
                      <Option value="rule">{t('ruleMatch', '规则匹配')}</Option>
                      <Option value="version-range">{t('versionRange', '版本区间')}</Option>
                      <Option value="fuzzy-product">{t('fuzzyProduct', '产品模糊匹配')}</Option>
                      <Option value="manual-review">{t('manualReview', '人工复核')}</Option>
                      <Option value="inventory">{t('inventory', '资产识别')}</Option>
                    </Select>
                    <Select
                      placeholder={t('filterByKnowledge', '知识库关联')}
                      allowClear
                      style={{ width: 140 }}
                      value={vulnFilters.hasKnowledge || undefined}
                      onChange={(val) => setVulnFilters({ ...vulnFilters, hasKnowledge: val || '' })}
                    >
                      <Option value="true">{t('hasKnowledge', '已关联')}</Option>
                      <Option value="false">{t('noKnowledge', '未关联')}</Option>
                    </Select>
                  </Space>
                </div>
                <Table
                  columns={vulnColumns}
                  dataSource={filteredVulnerabilities}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 1400 }}
                  pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                  locale={{
                    emptyText: isWebScan ? (
                      <Empty
                        description={t('noTemplateMatch', '本次未命中模板漏洞')}
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                      >
                        <Text type="secondary">
                          {t('noTemplateMatchDesc', '这通常表示当前目标 URL 未命中现有模板，不代表目标绝对安全。建议改扫真实业务 API、登录后入口，或切换为专项扫描。')}
                        </Text>
                      </Empty>
                    ) : t('noVulnerabilityData', '暂无漏洞数据'),
                  }}
                />
              </>
            ),
          }]),
          ...(candidateVulnerabilities.length > 0 ? [{
            key: 'candidates',
            label: `${t('candidateTab', '待验证')} (${filteredCandidateVulnerabilities.length})`,
            children: (
              <>
                <Alert
                  style={{ marginBottom: 12 }}
                  type="warning"
                  showIcon
                  message={t('resultsNeedVerification', '以下结果需要验证')}
                  description={t('resultsNeedVerificationDesc', '这些结果是根据目标开放服务的产品名、版本号或 CPE 信息，与漏洞知识库自动比对后得到的风险线索。它们适合作为排查依据，不建议直接当作正式漏洞下发。')}
                />
                <Table
                  columns={vulnColumns}
                  dataSource={filteredCandidateVulnerabilities}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 1400 }}
                  pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                  locale={{ emptyText: t('noCandidateResults', '暂无待验证结果') }}
                />
              </>
            ),
          }] : []),
          ...(inventoryFindings.length > 0 ? [{
            key: 'inventory',
            label: `${t('inventoryTab', '资产识别')} (${filteredInventoryFindings.length})`,
            children: (
              <>
                <Alert
                  style={{ marginBottom: 12 }}
                  type="info"
                  showIcon
                  message={t('resultsForInventory', '以下结果用于资产盘点')}
                  description={t('resultsForInventoryDesc', '这类结果通常来自服务识别、版本枚举或信息探测，不代表已确认可利用漏洞。')}
                />
                <Table
                  columns={vulnColumns}
                  dataSource={filteredInventoryFindings}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 1400 }}
                  pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                  locale={{ emptyText: t('noInventoryResults', '暂无资产识别结果') }}
                />
              </>
            ),
          }] : []),
          ...(!isDiscoveryTask ? [{
            key: 'assets',
            label: isWebScan ? `${t('scanEntriesCount', '扫描入口')} (${scanEntryCount})` : `${t('portListCount', '端口列表')} (${assets.length})`,
            children: (
              <>
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">
                    {isWebScan
                      ? t('scanEntriesSummary', '共展示 {{count}} 个扫描入口；结果区优先读取新 targets 模型，展示父子关系、入口类型、来源以及已解析到的主机和服务信息。', { count: scanEntryCount })
                      : t('openPortsSummary', '共发现 {{portCount}} 个开放端口，分布在 {{hostCount}} 个主机上', { portCount: assets.length, hostCount: uniqueIPs })}
                  </Text>
                </div>
                {isWebScan && hasTargetTree && (
                  <Alert
                    style={{ marginBottom: 12 }}
                    type="info"
                    showIcon
                    message={t('newTargetModel', '当前页签已切到新目标模型')}
                    description={t('newTargetModelDesc', '这里优先读取 security_scan_targets，而不是旧 discoveries。父子层级更接近真实发现与验证链路。')}
                  />
                )}
                {isWebScan && !hasTargetTree && (
                  <Alert
                    style={{ marginBottom: 12 }}
                    type="warning"
                    showIcon
                    message={t('noTargetModelData', '该任务没有可用的新目标模型数据')}
                    description={t('noTargetModelDataDesc', '当前页签只读取 security_scan_targets；如果这里为空，说明这条任务还没有完成新模型落库，或历史数据尚未迁移。')}
                  />
                )}
                {isWebScan && webSummary.ruleOnly > 0 && (
                  <Alert
                    style={{ marginBottom: 12 }}
                    type="warning"
                    showIcon
                    message={t('entryTableNotFullNuclei', '入口表并不代表全部都执行了 full Nuclei')}
                    description={t('entryTableNotFullNucleiDesc', '当前展示的入口里，有 {{ruleOnly}} 个目标只执行了规则检测；另有 {{skipped}} 个目标因预算被跳过。建议结合摘要卡判断真实覆盖面。', { ruleOnly: webSummary.ruleOnly, skipped: webSummary.skipped })}
                  />
                )}
                {isWebScan && webSummary.fallback > 0 && (
                  <Alert
                    style={{ marginBottom: 12 }}
                    type="warning"
                    showIcon
                    message={t('discoveryChainBrowserFallback', '本次发现链路存在浏览器回退')}
                    description={discoveryWarnings[0] && typeof discoveryWarnings[0].reason === 'string'
                      ? t('discoveryChainBrowserFallbackDesc1', '当前任务至少有一次 browser discovery 回退到 HTTP discovery。最近一次原因：{{reason}}', { reason: String(discoveryWarnings[0].reason) })
                      : t('discoveryChainBrowserFallbackDesc2', '当前任务至少有一次 browser discovery 回退到 HTTP discovery，发现结果可能偏保守。')}
                  />
                )}
                {isWebScan ? (
                  <Table
                    columns={scanEntryColumns}
                    dataSource={scanEntryRows}
                    rowKey="key"
                    loading={loading}
                    size="small"
                    pagination={false}
                    expandable={hasTargetTree ? { defaultExpandAllRows: true } : undefined}
                    locale={{ emptyText: t('noScanEntryData', '暂无扫描入口数据') }}
                  />
                ) : (
                  <Table
                    columns={assetColumns}
                    dataSource={assets}
                    rowKey="id"
                    loading={loading}
                    size="small"
                    pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                    locale={{ emptyText: t('noAssetData', '暂无资产数据') }}
                  />
                )}
                {isWebScan && (
                  showWebFollowupHint ? (
                    <Alert
                      style={{ marginTop: 16 }}
                      type="info"
                      showIcon
                      message={t('retestSuggestion', '复测建议')}
                      description={webFollowupHint}
                    />
                  ) : null
                )}
              </>
            ),
          }] : []),
          ...(isDiscoveryTask && filteredVulnerabilities.length > 0 ? [{
            key: 'vulns',
            label: `${t('relatedVulnerabilities', '关联漏洞')} (${filteredVulnerabilities.length})`,
            children: (
              <>
                {candidateVulnerabilities.length > 0 && (
                <Alert
                  style={{ marginBottom: 12 }}
                  type="warning"
                  showIcon
                  message={t('candidateResultsCollapsed', '待验证结果已单独折叠')}
                  description={t('associatedCandidateNote', '当前还有 {{count}} 条根据目标开放服务的产品名、版本号或 CPE 信息，与漏洞知识库自动比对后得到的风险线索，已移到"待验证"页签。', { count: candidateVulnerabilities.length })}
                />
                )}
                <Table
                  columns={vulnColumns}
                  dataSource={filteredVulnerabilities}
                  rowKey="id"
                  loading={loading}
                  size="small"
                  scroll={{ x: 1400 }}
                  pagination={{ defaultPageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50', '100'], showQuickJumper: true }}
                  locale={{ emptyText: t('noRelatedVulns', '暂无关联漏洞数据') }}
                />
              </>
            ),
          }] : []),
        ]}
      />

      {/* 漏洞详情抽屉 */}
      <Drawer
        title={t('vulnDetailTitle', '漏洞详情')}
        placement="right"
        width={600}
        open={detailDrawerOpen}
        onClose={() => {
          setDetailDrawerOpen(false)
          setSelectedVulnDetail(null)
        }}
      >
        {selectedVuln && (
          <>
            {detailLoading && (
              <div style={{ marginBottom: 16, textAlign: 'center' }}>
                <Spin />
              </div>
            )}
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label={t('vulnTitle', '漏洞标题')}>
                <Text strong>{selectedVuln.title}</Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('severityLevel', '严重程度')}>
                <Tag color={selectedVuln.severity === 'high' ? 'red' : selectedVuln.severity === 'medium' ? 'orange' : 'blue'}>
                  {selectedVuln.severity?.toUpperCase()}
                </Tag>
                {selectedVuln.cvss_score > 0 && (
                  <Text style={{ marginLeft: 8 }}>CVSS: {selectedVuln.cvss_score.toFixed(1)}</Text>
                )}
              </Descriptions.Item>
              <Descriptions.Item label={t('vulnUrl', '目标地址')}>{selectedVuln.vuln_url || `${selectedVuln.ip}:${selectedVuln.port}`}</Descriptions.Item>
              <Descriptions.Item label={t('vulnType', '漏洞类型')}>{selectedVuln.vuln_type || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('riskCategory', '风险类别')}>
                {(() => {
                  const category = getRiskCategory(selectedVuln)
                  return <Tag color={category.color}>{category.text}</Tag>
                })()}
              </Descriptions.Item>
              {isCandidateFinding(selectedVuln) && (
                <Descriptions.Item label={t('verificationStatus', '验证状态')}>
                  {(() => {
                    const review = getVerificationStatusTag(selectedVuln)
                    return <Tag color={review.color}>{review.text}</Tag>
                  })()}
                </Descriptions.Item>
              )}
              {isCandidateFinding(selectedVuln) && selectedVuln.confirmed_vuln_id && (
                <Descriptions.Item label={t('confirmedToFormal', '已转正式结果')}>#{selectedVuln.confirmed_vuln_id}</Descriptions.Item>
              )}
              {!isCandidateFinding(selectedVuln) && selectedVuln.source_vuln_id && (
                <Descriptions.Item label={t('sourceCandidate', '来源待验证')}>#{selectedVuln.source_vuln_id}</Descriptions.Item>
              )}
              {selectedVuln.cve_id && (
                <Descriptions.Item label={t('cveId', 'CVE 编号')}>{selectedVuln.cve_id}</Descriptions.Item>
              )}
              {getPrimaryCVE(selectedVuln) && (
                <Descriptions.Item label={t('primaryCve', '主 CVE')}>{getPrimaryCVE(selectedVuln)}</Descriptions.Item>
              )}
              {selectedVuln.cnvd_id && (
                <Descriptions.Item label={t('cnvdId', 'CNVD 编号')}>{selectedVuln.cnvd_id}</Descriptions.Item>
              )}
              {selectedVuln.cnnvd_id && (
                <Descriptions.Item label={t('cnnvdId', 'CNNVD 编号')}>{selectedVuln.cnnvd_id}</Descriptions.Item>
              )}
              {selectedVuln.cncve_id && (
                <Descriptions.Item label={t('cncveId', 'CNCVE 编号')}>{selectedVuln.cncve_id}</Descriptions.Item>
              )}
            </Descriptions>

            <Card size="small" title={t('vulnDescription', '漏洞描述')} style={{ marginTop: 16 }}>
              <Paragraph>
                {selectedVuln.description || t('noVulnDescription', '暂无漏洞描述信息')}
              </Paragraph>
            </Card>

            <Card size="small" title={t('fixSuggestion', '修复方案')} style={{ marginTop: 16 }}>
              <Paragraph>
                {selectedVuln.solution || t('noFixSuggestion', '暂无修复方案信息')}
              </Paragraph>
            </Card>

            {isCandidateFinding(selectedVuln) && (
              <Card size="small" title={t('verificationStatusSection', '验证状态')} style={{ marginTop: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }} size={12}>
                  <Alert
                    type="warning"
                    showIcon
                    message={t('candidateFindingInfo', '该结果属于待验证')}
                    description={selectedVuln.confirmed_vuln_id
                      ? t('candidateFindingDescDerived', '系统已识别目标服务版本，并命中漏洞知识库中的受影响范围。由于这类结果主要基于版本信息推断，仍需结合实际补丁情况、配置状态或人工验证后再确认是否成立。当前结果已派生正式结果 #{{id}}。', { id: selectedVuln.confirmed_vuln_id })
                      : t('candidateFindingDesc', '系统已识别目标服务版本，并命中漏洞知识库中的受影响范围。由于这类结果主要基于版本信息推断，仍需结合实际补丁情况、配置状态或人工验证后再确认是否成立。')}
                  />
                  <Descriptions column={2} bordered size="small">
                    <Descriptions.Item label={t('verificationStatus', '验证状态')}>
                      {(() => {
                        const review = getVerificationStatusTag(selectedVuln)
                        return <Tag color={review.color}>{review.text}</Tag>
                      })()}
                    </Descriptions.Item>
                    <Descriptions.Item label={t('verificationTime', '验证时间')}>
                      {getVerifiedAtValue(selectedVuln) ? formatDateTime(getVerifiedAtValue(selectedVuln)) : '-'}
                    </Descriptions.Item>
                  </Descriptions>
                  <Select
                    value={getVerificationStatusValue(selectedVuln)}
                    onChange={(val) => handleCandidateReview(selectedVuln, val as 'pending' | 'needs-test' | 'confirmed' | 'rejected')}
                    loading={candidateReviewLoadingId === selectedVuln.id}
                    options={[
                      { value: 'pending', label: t('unverified', '未验证') },
                      { value: 'needs-test', label: t('retesting', '复测中') },
                      { value: 'confirmed', label: t('verified', '验证成功') },
                      { value: 'rejected', label: t('verificationFailed', '验证失败') },
                    ]}
                  />
                  <TextArea
                    rows={4}
                    value={candidateReviewNote}
                    placeholder={t('verificationNotePlaceholder', '记录验证依据，例如补丁版本、人工验证结果、复测命令或失败原因。')}
                    onChange={(event) => setCandidateReviewNote(event.target.value)}
                  />
                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Button
                      type="primary"
                      loading={candidateReviewLoadingId === selectedVuln.id}
                      onClick={() => handleCandidateReview(selectedVuln, getVerificationStatusValue(selectedVuln) as 'pending' | 'needs-test' | 'confirmed' | 'rejected')}
                    >
                      {t('saveVerificationNote', '保存验证备注')}
                    </Button>
                  </div>
                </Space>
              </Card>
            )}

            {(selectedVuln.vuln_db_id || selectedVuln.knowledge || getPrimaryCVE(selectedVuln)) && (
              <Card size="small" title={t('knowledgeLink', '知识库关联')} style={{ marginTop: 16 }}>
                <Descriptions column={1} bordered size="small">
                  <Descriptions.Item label={t('vulnDBRecord', '漏洞库记录')}>
                    {selectedVuln.vuln_db_id ? `#${selectedVuln.vuln_db_id}` : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('findingSource', '结果来源')}>
                    {(() => {
                      const source = getFindingSourceTag(selectedVuln)
                      return <Tag color={source.color}>{source.text}</Tag>
                    })()}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('confidence', '置信度')}>
                    {(() => {
                      const confidence = getConfidenceTag(selectedVuln)
                      return <Tag color={confidence.color}>{confidence.text}</Tag>
                    })()}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('matchMode', '匹配模式')}>{getMatchModeValue(selectedVuln) || '-'}</Descriptions.Item>
                  <Descriptions.Item label={t('primaryCve', '主 CVE')}>{getPrimaryCVE(selectedVuln) || '-'}</Descriptions.Item>
                  <Descriptions.Item label={t('vulnDBTitle', '漏洞库标题')}>{selectedVuln.knowledge?.title || '-'}</Descriptions.Item>
                  <Descriptions.Item label={t('knowledgeSeverity', '知识库严重度')}>{selectedVuln.knowledge?.severity || '-'}</Descriptions.Item>
                  <Descriptions.Item label={t('knowledgeCVSS', '知识库 CVSS')}>
                    {selectedVuln.knowledge?.cvss_score ? selectedVuln.knowledge.cvss_score.toFixed(1) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('knowledgeCNVD', '知识库 CNVD')}>{selectedVuln.knowledge?.cnvd_id || selectedVuln.cnvd_id || '-'}</Descriptions.Item>
                  <Descriptions.Item label={t('knowledgeCNNVD', '知识库 CNNVD')}>{selectedVuln.knowledge?.cnnvd_id || selectedVuln.cnnvd_id || '-'}</Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            {!detailLoading && selectedVulnDetail && vulnOccurrences.length === 0 && vulnEvidences.length === 0 && (
              <Alert
                style={{ marginTop: 16 }}
                type="info"
                showIcon
                message={t('noNewModelDetail', '当前漏洞暂无新模型详情')}
                description={t('noNewModelDetailDesc', '这条结果还没有 occurrence/evidence 数据，抽屉目前继续保留旧漏洞字段展示。')}
              />
            )}

            {vulnOccurrences.length > 0 && (
              <Card size="small" title={t('hitRecords', '命中记录')} style={{ marginTop: 16 }}>
                <Table
                  rowKey="id"
                  size="small"
                  pagination={false}
                  scroll={{ x: 900 }}
                  dataSource={vulnOccurrences}
                  columns={occurrenceColumns}
                />
              </Card>
            )}

            {vulnEvidences.length > 0 && (
              <Card size="small" title={t('evidence', '证据')} style={{ marginTop: 16 }}>
                <Space direction="vertical" style={{ width: '100%' }} size={12}>
                  {vulnEvidences.map((evidence: SecurityScanEvidence) => (
                    <Card
                      key={evidence.id}
                      size="small"
                      title={`#${evidence.id} ${evidence.evidence_type || 'evidence'}`}
                      extra={<Tag color={evidence.source_engine === 'rule' ? 'gold' : evidence.source_engine === 'nuclei' ? 'blue' : 'default'}>{evidence.source_engine || '-'}</Tag>}
                    >
                      <Descriptions size="small" column={1}>
                        <Descriptions.Item label={t('payloadUrl', 'Payload / URL')}>{evidence.payload_excerpt || '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('invItem', '命中时间')}>{evidence.created_at ? formatDateTime(evidence.created_at) : '-'}</Descriptions.Item>
                        <Descriptions.Item label={t('target', '目标')}>
                          {getMetadataString(evidence.metadata, 'url')
                            || getMetadataString(evidence.metadata, 'matched_at')
                            || '-'}
                        </Descriptions.Item>
                      </Descriptions>
                      {evidence.request_excerpt && (
                        <Card size="small" title={t('requestSnippet', '请求片段')} style={{ marginTop: 12 }}>
                          <Paragraph copyable={{ text: evidence.request_excerpt }} style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>
                            {evidence.request_excerpt}
                          </Paragraph>
                        </Card>
                      )}
                      {evidence.response_excerpt && (
                        <Card size="small" title={t('responseSnippet', '响应片段')} style={{ marginTop: 12 }}>
                          <Paragraph copyable={{ text: evidence.response_excerpt }} style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>
                            {evidence.response_excerpt}
                          </Paragraph>
                        </Card>
                      )}
                    </Card>
                  ))}
                </Space>
              </Card>
            )}

            {selectedVuln.reference_url && (
              <Card size="small" title={t('referenceLinks', '参考链接')} style={{ marginTop: 16 }}>
                {selectedVuln.reference_url.split(',').map((url, index) => (
                  <div key={index} style={{ marginBottom: 8 }}>
                    <Link href={url.trim()} target="_blank">
                      {url.trim()}
                    </Link>
                  </div>
                ))}
              </Card>
            )}

            {selectedVuln.payload && (
              <Card size="small" title={t('testPayload', '测试 Payload')} style={{ marginTop: 16 }}>
                <Paragraph copyable>
                  <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: 4 }}>
                    {selectedVuln.payload}
                  </code>
                </Paragraph>
              </Card>
            )}

            <Card size="small" title={t('scanInfo', '扫描信息')} style={{ marginTop: 16 }}>
              <Descriptions column={1} size="small">
                <Descriptions.Item label={t('findingSource', '结果来源')}>
                  {(() => {
                    const source = getFindingSourceTag(selectedVuln)
                    return <Tag color={source.color}>{source.text}</Tag>
                  })()}
                </Descriptions.Item>
                <Descriptions.Item label={t('confidence', '置信度')}>
                  {(() => {
                    const confidence = getConfidenceTag(selectedVuln)
                    return <Tag color={confidence.color}>{confidence.text}</Tag>
                  })()}
                </Descriptions.Item>
                <Descriptions.Item label={t('scanner', '扫描引擎')}>{selectedVuln.scanner || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('scanMethod', '扫描方法')}>
                  {getMatchModeValue(selectedVuln) || getScanMethodLabel(selectedVuln.scan_method) || (selectedVuln.scanner === 'vuln-matcher' ? t('serviceVersionMatch', '服务版本匹配') : '-')}
                </Descriptions.Item>
                <Descriptions.Item label={t('matchedOn', '匹配依据')}>
                  {selectedVuln.matched_on || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('exploitPrereq', '利用前提')}>
                  {selectedVuln.exploit_prereq || '-'}
                </Descriptions.Item>
                <Descriptions.Item label={t('discoveryTime', '发现时间')}>
                  {selectedVuln.created_at ? formatDateTime(selectedVuln.created_at) : '-'}
                </Descriptions.Item>
              </Descriptions>
            </Card>
          </>
        )}
      </Drawer>
    </Modal>
  )
}
