// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

// @ts-nocheck
import { Card, Row, Col, Tag, Typography, Divider, Alert, Button, Space, Descriptions, Spin } from 'antd'
import {
  BugOutlined,
  WarningOutlined,
  SafetyOutlined,
  ToolOutlined,
  LinkOutlined,
  FileTextOutlined,
  CopyOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

const { Title, Text, Paragraph } = Typography

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

interface VulnDBRecord {
  id?: number
  cve_id?: string
  cnvd_id?: string
  cnnvd_id?: string
  title?: string
  description?: string
  vuln_type?: string
  severity?: string
  cvss_score?: number
  cvss_vector?: string
  affected_product?: string
  affected_version?: string
  solution?: string
  patch_url?: string
  workaround?: string
  references?: string
  cwe_id?: string
  source?: string
  tags?: string
}

interface VulnDetailProps {
  vuln: VulnDBRecord
}

export default function VulnDetail({ vuln }: VulnDetailProps) {
  const { t } = useTranslation('security')
  const sectionTitleStyle = {
    color: '#f8fafc',
    marginBottom: 12,
    fontWeight: 700,
    letterSpacing: '0.02em' as const,
  }

  const sectionIconStyle = {
    marginRight: 8,
    color: '#7dd3fc',
  }

  const getSeverityColor = (severity?: string) => {
    const colors: Record<string, string> = {
      critical: theme.critical,
      high: theme.high,
      medium: theme.medium,
      low: theme.low,
    }
    return colors[severity || ''] || theme.textSecondary
  }

  const getSeverityBgColor = (severity?: string) => {
    const colors: Record<string, string> = {
      critical: 'rgba(239, 68, 68, 0.1)',
      high: 'rgba(249, 115, 22, 0.1)',
      medium: 'rgba(234, 179, 8, 0.1)',
      low: 'rgba(34, 197, 94, 0.1)',
    }
    return colors[severity || ''] || 'transparent'
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  return (
    <div style={{ background: theme.card, padding: 16, borderRadius: 8 }}>
      {/* CVE ID 和严重程度 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0, color: theme.text, fontFamily: 'monospace' }}>
            {vuln.cve_id}
          </Title>
          {vuln.cnvd_id && (
            <Text style={{ color: '#cbd5e1', fontWeight: 500 }}>CNVD: {vuln.cnvd_id}</Text>
          )}
          {vuln.cnnvd_id && (
            <div>
              <Text style={{ color: '#cbd5e1', fontWeight: 500 }}>CNNVD: {vuln.cnnvd_id}</Text>
            </div>
          )}
        </div>
        <div style={{
          padding: '8px 16px',
          borderRadius: 8,
          background: getSeverityBgColor(vuln.severity),
          border: `1px solid ${getSeverityColor(vuln.severity)}`,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <BugOutlined style={{ color: getSeverityColor(vuln.severity), fontSize: 20 }} />
            <div>
              <div style={{ color: getSeverityColor(vuln.severity), fontWeight: 'bold', textTransform: 'uppercase' }}>
                {vuln.severity || 'Unknown'}
              </div>
              <div style={{ color: '#cbd5e1', fontSize: 12, fontWeight: 500 }}>
                CVSS: {vuln.cvss_score?.toFixed(1) || 'N/A'}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 漏洞标题 */}
      <div style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0, color: '#f8fafc', fontWeight: 700, lineHeight: 1.45 }}>
          {vuln.title || 'Unknown Vulnerability'}
        </Title>
      </div>

      {/* 标签 */}
      <Space style={{ marginBottom: 16 }}>
        {vuln.vuln_type && <Tag color="blue">{vuln.vuln_type}</Tag>}
        {vuln.cwe_id && <Tag>{vuln.cwe_id}</Tag>}
        {vuln.source && <Tag color="purple">{vuln.source.toUpperCase()}</Tag>}
        {vuln.tags?.split(',').map((tag: string) => (
          <Tag key={tag} style={{ margin: 0 }}>{tag.trim()}</Tag>
        ))}
      </Space>

      <Divider style={{ borderColor: theme.border }} />

      {/* 影响范围 */}
      <div style={{ marginBottom: 16 }}>
        <Title level={5} style={sectionTitleStyle}>
          <SafetyOutlined style={sectionIconStyle} />
          {t('vulnDetail.affectedScope', '影响范围')}
        </Title>
        <Card size="small" style={{ background: theme.cardHover, borderColor: theme.border }}>
          <Descriptions
            column={2}
            size="small"
            labelStyle={{ color: '#cbd5e1', fontWeight: 600 }}
            contentStyle={{ color: '#f8fafc', fontWeight: 500 }}
          >
            <Descriptions.Item label={t('vulnDetail.affectedProduct', '受影响产品')}>
              <Text style={{ color: '#f8fafc', fontWeight: 500 }}>{vuln.affected_product || 'Unknown'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('vulnDetail.affectedVersion', '受影响版本')}>
              <Text style={{ color: '#f8fafc', fontWeight: 500 }}>{vuln.affected_version || 'Unknown'}</Text>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      </div>

      {/* 漏洞描述 */}
      <div style={{ marginBottom: 16 }}>
        <Title level={5} style={sectionTitleStyle}>
          <FileTextOutlined style={sectionIconStyle} />
          {t('vulnDetail.description', '漏洞描述')}
        </Title>
        <Card size="small" style={{ background: theme.cardHover, borderColor: theme.border }}>
          <Paragraph style={{ color: '#f8fafc', margin: 0, whiteSpace: 'pre-wrap', lineHeight: 1.75, fontWeight: 500 }}>
            {vuln.description || t('vulnDetail.noDescription', '暂无详细描述')}
          </Paragraph>
        </Card>
      </div>

      {/* 修复建议 */}
      <div style={{ marginBottom: 16 }}>
        <Title level={5} style={sectionTitleStyle}>
          <ToolOutlined style={sectionIconStyle} />
          {t('vulnDetail.fixSuggestion', '修复建议')}
        </Title>
        <Card
          size="small"
          style={{
            background: 'linear-gradient(135deg, rgba(34, 197, 94, 0.1) 0%, rgba(6, 182, 212, 0.1) 100%)',
            borderColor: theme.low,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
            <CheckCircleOutlined style={{ color: theme.low, fontSize: 20, marginTop: 4 }} />
            <div style={{ flex: 1 }}>
              <Paragraph style={{ color: '#f8fafc', margin: 0, whiteSpace: 'pre-wrap', lineHeight: 1.75, fontWeight: 500 }}>
                {vuln.solution || t('vulnDetail.noSolution', '暂无修复建议')}
              </Paragraph>
              {vuln.patch_url && (
                <div style={{ marginTop: 8 }}>
                  <Button
                    type="link"
                    icon={<LinkOutlined />}
                    href={vuln.patch_url}
                    target="_blank"
                    style={{ padding: 0, color: '#7dd3fc', fontWeight: 600 }}
                  >
                  {t('vulnDetail.viewOfficialPatch', '查看官方补丁')}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </Card>
      </div>

      {/* 缓解措施 */}
      {vuln.workaround && (
        <div style={{ marginBottom: 16 }}>
          <Title level={5} style={sectionTitleStyle}>
            <WarningOutlined style={sectionIconStyle} />
            {t('vulnDetail.mitigation', '缓解措施')}
          </Title>
          <Card size="small" style={{ background: theme.cardHover, borderColor: theme.border }}>
            <Paragraph style={{ color: '#f8fafc', margin: 0, whiteSpace: 'pre-wrap', lineHeight: 1.75, fontWeight: 500 }}>
              {vuln.workaround}
            </Paragraph>
          </Card>
        </div>
      )}

      {/* CVSS 向量 */}
      {vuln.cvss_vector && (
        <div style={{ marginBottom: 16 }}>
          <Title level={5} style={sectionTitleStyle}>
            {t('vulnDetail.cvssVector', 'CVSS 向量')}
          </Title>
          <Card
            size="small"
            style={{ background: theme.cardHover, borderColor: theme.border }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ color: '#e2e8f0', fontFamily: 'monospace', fontSize: 12, fontWeight: 500 }}>
                {vuln.cvss_vector}
              </Text>
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(vuln.cvss_vector || '')}
                style={{ color: '#cbd5e1' }}
              />
            </div>
          </Card>
        </div>
      )}

      {/* 参考链接 */}
      {vuln.references && (
        <div>
          <Title level={5} style={sectionTitleStyle}>
            <LinkOutlined style={sectionIconStyle} />
            {t('vulnDetail.references', '参考链接')}
          </Title>
          <Card size="small" style={{ background: theme.cardHover, borderColor: theme.border }}>
            {vuln.references.split('\n').filter(Boolean).map((ref: string, idx: number) => (
              <div key={idx} style={{ marginBottom: 4 }}>
                <a
                  href={ref.trim()}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ color: '#93c5fd', fontSize: 13, fontWeight: 500, wordBreak: 'break-all', lineHeight: 1.6 }}
                >
                  {ref.trim()}
                </a>
              </div>
            ))}
          </Card>
        </div>
      )}
    </div>
  )
}