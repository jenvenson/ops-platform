// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

// @ts-nocheck
import { useEffect, useState } from 'react'
import { Card, Row, Col, Progress, Spin, Typography, Tag, Button, Statistic, List, Space } from 'antd'
import {
  SafetyOutlined,
  AlertOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  ScanOutlined,
  RadarChartOutlined,
  BugOutlined,
  ThunderboltOutlined,
  ArrowRightOutlined,
} from '@ant-design/icons'
import { securityAPI, SecurityStatistics, SecurityVulnerability, PaginatedResponse } from '../../api/security'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

// SOC 风格深色主题配置
const theme = {
  bg: '#0a0e17',
  card: '#111827',
  cardHover: '#1a2234',
  border: '#1e293b',
  primary: '#3b82f6',
  accent: '#06b6d4',
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
  text: '#e2e8f0',
  textSecondary: '#94a3b8',
}

interface VulnDBStats {
  total: number
  critical: number
  high: number
  medium: number
  low: number
  this_week: number
}

interface SyncTask {
  id: number
}

export default function SecurityOverview() {
  const navigate = useNavigate()
  const { t } = useTranslation('security')
  const [loading, setLoading] = useState(true)
  const [stats, setStats] = useState<SecurityStatistics>({
    total_tasks: 0,
    running_tasks: 0,
    completed_tasks: 0,
    total_assets: 0,
    total_vulnerabilities: 0,
    high_risk_count: 0,
    medium_risk_count: 0,
    low_risk_count: 0,
  })
  const [vulnStats, setVulnStats] = useState<VulnDBStats>({
    total: 0,
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    this_week: 0,
  })
  const [recentVulns, setRecentVulns] = useState<SecurityVulnerability[]>([])

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const [statsData, vulnsResponse, vulnStatsData] = await Promise.all([
          securityAPI.getStatistics(),
          securityAPI.getVulnerabilities({ page: 1, page_size: 5 }),
          securityAPI.getVulnDBStats(),
        ])
        setStats(statsData)

        const vulnsData = vulnsResponse as PaginatedResponse<SecurityVulnerability>
        setRecentVulns(vulnsData.data || [])

        if (vulnStatsData) {
          setVulnStats(vulnStatsData as VulnDBStats)
        }
      } catch (error) {
        console.error('获取安全数据失败:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  // 计算安全评分 (模拟计算)
  const securityScore = Math.max(0, 100 - (vulnStats.critical * 5) - (vulnStats.high * 2) - (vulnStats.medium * 0.5))

  const getScoreColor = (score: number) => {
    if (score >= 80) return '#22c55e'
    if (score >= 60) return '#eab308'
    return '#ef4444'
  }

  const getSeverityColor = (severity: string) => {
    const colors: Record<string, string> = {
      critical: theme.critical,
      high: theme.high,
      medium: theme.medium,
      low: theme.low,
    }
    return colors[severity] || theme.textSecondary
  }

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px', background: theme.bg }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div style={{ background: theme.bg, minHeight: '100vh', padding: '24px' }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Title level={3} style={{ margin: 0, color: theme.text, display: 'flex', alignItems: 'center', gap: 12 }}>
            <SafetyOutlined style={{ color: theme.primary }} />
            {t('securityOperationsCenter', '安全运营中心')}
          </Title>
          <Text style={{ color: theme.textSecondary }}>{t('realTimeMonitorSecurityPosture', '实时监控企业安全态势')}</Text>
        </div>
        <Space>
          <Button
            icon={<RadarChartOutlined />}
            size="large"
            onClick={() => navigate('/security/assets')}
          >
            {t('assetDiscovery', '资产发现')}
          </Button>
          <Button
            type="primary"
            icon={<ScanOutlined />}
            size="large"
            onClick={() => navigate('/security/tasks')}
            style={{ background: theme.primary, borderColor: theme.primary }}
          >
            {t('immediateVulnerabilityScan', '立即漏洞扫描')}
          </Button>
        </Space>
      </div>

      {/* 安全评分 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={8}>
          <div style={{
            background: `linear-gradient(135deg, ${theme.card} 0%, #1e293b 100%)`,
            borderRadius: 16,
            padding: 24,
            border: `1px solid ${theme.border}`,
            height: '100%',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ position: 'relative', width: 120, height: 120 }}>
                <svg viewBox="0 0 100 100" style={{ transform: 'rotate(-90deg)' }}>
                  <circle cx="50" cy="50" r="45" fill="none" stroke={theme.border} strokeWidth="8" />
                  <circle
                    cx="50" cy="50" r="45" fill="none"
                    stroke={getScoreColor(securityScore)}
                    strokeWidth="8"
                    strokeDasharray={`${(securityScore / 100) * 283} 283`}
                    strokeLinecap="round"
                  />
                </svg>
                <div style={{
                  position: 'absolute',
                  top: '50%',
                  left: '50%',
                  transform: 'translate(-50%, -50%)',
                  textAlign: 'center',
                }}>
                  <div style={{ fontSize: 32, fontWeight: 'bold', color: getScoreColor(securityScore) }}>
                    {securityScore}
                  </div>
                  <div style={{ fontSize: 12, color: theme.textSecondary }}>{t('securityScore', '安全评分')}</div>
                </div>
              </div>
              <div>
                <div style={{ fontSize: 18, fontWeight: 600, color: theme.text, marginBottom: 8 }}>
                  {securityScore >= 80 ? t('securityStatusGood', '安全状态良好') : securityScore >= 60 ? t('needsAttention', '需要注意') : t('hasRisk', '存在风险')}
                </div>
                <div style={{ color: theme.textSecondary, fontSize: 13 }}>
                  {t('todayScan', '今日扫描')} {stats.completed_tasks} {t('scanTimes', '次')}
                </div>
                <div style={{ color: theme.textSecondary, fontSize: 13 }}>
                  {t('found', '发现')} {stats.total_vulnerabilities} {t('vulnerability', '个漏洞')}
                </div>
              </div>
            </div>
          </div>
        </Col>

        {/* 漏洞统计 */}
        <Col xs={24} lg={16}>
          <div style={{
            background: `linear-gradient(135deg, ${theme.card} 0%, #1e293b 100%)`,
            borderRadius: 16,
            padding: 24,
            border: `1px solid ${theme.border}`,
            height: '100%',
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <Title level={5} style={{ margin: 0, color: theme.text }}>
                <BugOutlined style={{ marginRight: 8, color: theme.primary }} />
                {t('vulnerabilityDistribution', '漏洞分布')}
              </Title>
              <Tag color="blue" style={{ cursor: 'pointer' }} onClick={() => navigate('/security/vuln-db')}>
                {t('vulnerabilityDatabase', '漏洞库')} <ArrowRightOutlined />
              </Tag>
            </div>
            <Row gutter={16}>
              <Col span={6}>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: 32, fontWeight: 'bold', color: theme.critical }}>{vulnStats.critical}</div>
                  <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('severity.critical', '严重')}</div>
                </div>
              </Col>
              <Col span={6}>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: 32, fontWeight: 'bold', color: theme.high }}>{vulnStats.high}</div>
                  <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('severity.high', '高危')}</div>
                </div>
              </Col>
              <Col span={6}>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: 32, fontWeight: 'bold', color: theme.medium }}>{vulnStats.medium}</div>
                  <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('severity.medium', '中危')}</div>
                </div>
              </Col>
              <Col span={6}>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ fontSize: 32, fontWeight: 'bold', color: theme.low }}>{vulnStats.low}</div>
                  <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('severity.low', '低危')}</div>
                </div>
              </Col>
            </Row>
            <div style={{ marginTop: 20 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                <Text style={{ color: theme.textSecondary }}>{t('vulnDBCoverage', '漏洞库覆盖率')}</Text>
                <Text style={{ color: theme.primary }}>{vulnStats.total.toLocaleString()} {t('cveCount', '条 CVE')}</Text>
              </div>
              <Progress
                percent={Math.min(100, (vulnStats.total / 200000) * 100)}
                strokeColor={theme.primary}
                trailColor={theme.border}
                showInfo={false}
              />
              <Text style={{ color: theme.textSecondary, fontSize: 12 }}>
                {t('newThisWeek', '本周新增')} {vulnStats.this_week} {t('vulnDataCount', '条漏洞数据')}
              </Text>
            </div>
          </div>
        </Col>
      </Row>

      {/* 资产和任务 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <div style={{
            background: theme.card,
            borderRadius: 12,
            padding: 20,
            border: `1px solid ${theme.border}`,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{
                width: 48, height: 48,
                borderRadius: 12,
                background: 'rgba(59, 130, 246, 0.1)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <ScanOutlined style={{ fontSize: 24, color: theme.primary }} />
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: theme.text }}>{stats.total_tasks}</div>
                <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('scanTasks', '扫描任务')}</div>
              </div>
            </div>
          </div>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <div style={{
            background: theme.card,
            borderRadius: 12,
            padding: 20,
            border: `1px solid ${theme.border}`,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{
                width: 48, height: 48,
                borderRadius: 12,
                background: 'rgba(34, 197, 94, 0.1)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <CheckCircleOutlined style={{ fontSize: 24, color: theme.low }} />
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: theme.text }}>{stats.completed_tasks}</div>
                <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('completed', '已完成')}</div>
              </div>
            </div>
          </div>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <div style={{
            background: theme.card,
            borderRadius: 12,
            padding: 20,
            border: `1px solid ${theme.border}`,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{
                width: 48, height: 48,
                borderRadius: 12,
                background: 'rgba(6, 182, 212, 0.1)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <SafetyOutlined style={{ fontSize: 24, color: theme.accent }} />
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: theme.text }}>{stats.total_assets}</div>
                <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('assetDiscovery', '资产发现')}</div>
              </div>
            </div>
          </div>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <div style={{
            background: theme.card,
            borderRadius: 12,
            padding: 20,
            border: `1px solid ${theme.border}`,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{
                width: 48, height: 48,
                borderRadius: 12,
                background: 'rgba(234, 179, 8, 0.1)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <ThunderboltOutlined style={{ fontSize: 24, color: theme.medium }} />
              </div>
              <div>
                <div style={{ fontSize: 24, fontWeight: 'bold', color: theme.text }}>{stats.running_tasks}</div>
                <div style={{ color: theme.textSecondary, fontSize: 12 }}>{t('inProgress', '进行中')}</div>
              </div>
            </div>
          </div>
        </Col>
      </Row>

      {/* 最新漏洞 */}
      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <div style={{
            background: theme.card,
            borderRadius: 16,
            padding: 24,
            border: `1px solid ${theme.border}`,
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <Title level={5} style={{ margin: 0, color: theme.text }}>
                <AlertOutlined style={{ marginRight: 8, color: theme.critical }} />
                {t('latestVulnerabilities', '最新漏洞')}
              </Title>
              <Button type="link" onClick={() => navigate('/security/vulnerabilities')} style={{ color: theme.primary }}>
                {t('viewAll', '查看全部')} <ArrowRightOutlined />
              </Button>
            </div>
            <List
              dataSource={recentVulns}
              renderItem={(item) => (
                <List.Item style={{ borderBottom: `1px solid ${theme.border}`, padding: '12px 0' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', alignItems: 'center' }}>
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                        <Tag color={getSeverityColor(item.severity)} style={{ margin: 0 }}>
                          {item.severity?.toUpperCase()}
                        </Tag>
                        <Text style={{ color: theme.text, fontWeight: 500 }}>{item.cve_id || item.title}</Text>
                      </div>
                      <Text style={{ color: theme.textSecondary, fontSize: 12 }}>
                        {item.ip}:{item.port} • {item.vuln_type}
                      </Text>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ color: theme.primary, fontWeight: 500 }}>{item.cvss_score?.toFixed(1)}</div>
                      <Text style={{ color: theme.textSecondary, fontSize: 12 }}>CVSS</Text>
                    </div>
                  </div>
                </List.Item>
              )}
              locale={{ emptyText: t('noVulnerabilityData', '暂无漏洞数据') }}
            />
          </div>
        </Col>
      </Row>
    </div>
  )
}