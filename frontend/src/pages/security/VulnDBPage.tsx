// @ts-nocheck
import { useEffect, useState } from 'react'
import { Card, Table, Input, Button, Select, Tag, Modal, Typography, Row, Col, Statistic, Alert, Upload, message, Space } from 'antd'
import {
  SearchOutlined,
  DatabaseOutlined,
  BugOutlined,
  SafetyOutlined,
  ClockCircleOutlined,
  UploadOutlined,
  InboxOutlined,
} from '@ant-design/icons'
import { securityAPI } from '../../api/security'
import VulnDetail from './VulnDetail'

const { Title, Text, Paragraph } = Typography
const { Search } = Input

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

const { Option } = Select

interface VulnDBRecord {
  id: number
  cve_id: string
  cnvd_id?: string
  cnnvd_id?: string
  title: string
  description?: string
  vuln_type?: string
  severity?: string
  cvss_score?: number
  cvss_vector?: string
  affected_product?: string
  affected_version?: string
  solution?: string
  source?: string
  last_updated?: string
}

export default function VulnDBPage() {
  const [loading, setLoading] = useState(true)
  const [vulns, setVulns] = useState<VulnDBRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const [severity, setSeverity] = useState<string>('')
  const [vulnType, setVulnType] = useState<string>('')
  const [stats, setStats] = useState<any>({})
  const [detailVisible, setDetailVisible] = useState(false)
  const [selectedVuln, setSelectedVuln] = useState<VulnDBRecord | null>(null)
  const [importVisible, setImportVisible] = useState(false)
  const [importing, setImporting] = useState(false)
  const [importSource, setImportSource] = useState<'nvd' | 'cnvd' | 'cnnvd'>('nvd')
  const [importContent, setImportContent] = useState('')
  const [importFileName, setImportFileName] = useState('')
  const [lastImportSummary, setLastImportSummary] = useState<string>('')

  const fetchData = async () => {
    setLoading(true)
    try {
      const [vulnsRes, statsRes] = await Promise.all([
        securityAPI.searchVulnDB({ keyword, severity, vuln_type: vulnType, page, page_size: pageSize }),
        securityAPI.getVulnDBStats(),
      ])

      setVulns(vulnsRes.data || [])
      setTotal(vulnsRes.total || 0)

      if (statsRes) {
        setStats(statsRes)
      }
    } catch (error) {
      console.error('获取漏洞库数据失败:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize])

  const handleSearch = () => {
    setPage(1)
    fetchData()
  }

  const handleViewDetail = (record: VulnDBRecord) => {
    setSelectedVuln(record)
    setDetailVisible(true)
  }

  const resetImportForm = () => {
    setImportSource('nvd')
    setImportContent('')
    setImportFileName('')
  }

  const openImportModal = () => {
    resetImportForm()
    setImportVisible(true)
  }

  const handleImport = async () => {
    const payload = importContent.trim()
    if (!payload) {
      message.warning('请先上传或粘贴漏洞数据内容')
      return
    }

    setImporting(true)
    try {
      let response: any
      if (importSource === 'cnvd') {
        response = await securityAPI.importCNVD(payload)
      } else if (importSource === 'cnnvd') {
        response = await securityAPI.importCNNVD(payload)
      } else {
        response = await securityAPI.importVulnerabilities(payload)
      }

      const summaryParts = [
        response?.message || '导入完成',
        typeof response?.inserted === 'number' ? `新增 ${response.inserted} 条` : '',
        typeof response?.updated === 'number' ? `更新 ${response.updated} 条` : '',
      ].filter(Boolean)

      const summary = summaryParts.join('，')
      setLastImportSummary(summary)
      message.success(summary || '导入完成')
      setImportVisible(false)
      await fetchData()
    } catch (error) {
      message.error((error as any)?.response?.data?.error || '导入失败')
    } finally {
      setImporting(false)
    }
  }

  const readLocalFile = async (file: File) => {
    const text = await file.text()
    setImportContent(text)
    setImportFileName(file.name)
    message.success(`已载入 ${file.name}`)
    return false
  }

  const importTemplateHint = {
    nvd: 'NVD 导入格式：cve_id,cnvd_id,cnnvd_id,title,severity,description,vuln_type,cvss_score,solution',
    cnvd: 'CNVD 导入格式：cnvd_id,cve_id,title,severity,cvss_score,description,solution',
    cnnvd: 'CNNVD 导入格式：cnnvd_id,cve_id,title,severity,cvss_score,description,solution',
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

  const columns = [
    {
      title: 'CVE ID',
      dataIndex: 'cve_id',
      key: 'cve_id',
      width: 150,
      render: (text: string) => (
        <a style={{ color: theme.primary, fontWeight: 600 }} onClick={() => handleViewDetail({ cve_id: text } as VulnDBRecord)}>
          {text}
        </a>
      ),
    },
    {
      title: '漏洞标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      render: (text: string, record: VulnDBRecord) => (
        <div>
          <div style={{ color: '#0f172a', fontWeight: 700, lineHeight: 1.5 }}>{text}</div>
          {record.affected_product && (
            <Text style={{ color: '#475569', fontSize: 12, fontWeight: 500 }}>
              影响产品: {record.affected_product}
            </Text>
          )}
        </div>
      ),
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)} style={{ textTransform: 'uppercase' }}>
          {severity || 'unknown'}
        </Tag>
      ),
    },
    {
      title: 'CVSS',
      dataIndex: 'cvss_score',
      key: 'cvss_score',
      width: 80,
      render: (score: number) => score ? (
        <span style={{ color: getSeverityColor(score >= 9 ? 'critical' : score >= 7 ? 'high' : score >= 4 ? 'medium' : 'low'), fontWeight: 600 }}>
          {score.toFixed(1)}
        </span>
      ) : '-',
    },
    {
      title: '漏洞类型',
      dataIndex: 'vuln_type',
      key: 'vuln_type',
      width: 120,
      render: (type: string) => type ? <Tag>{type}</Tag> : '-',
    },
    {
      title: '数据来源',
      dataIndex: 'source',
      key: 'source',
      width: 120,
      render: (source: string) => source?.toUpperCase(),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: VulnDBRecord) => (
        <Button type="link" size="small" onClick={() => handleViewDetail(record)}>
          详情
        </Button>
      ),
    },
  ]

  return (
    <div style={{ background: theme.bg, minHeight: '100vh', padding: '24px' }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
        <div>
          <Title level={3} style={{ margin: 0, color: theme.text, display: 'flex', alignItems: 'center', gap: 12 }}>
            <DatabaseOutlined style={{ color: theme.primary }} />
            漏洞知识库
          </Title>
          <Text style={{ color: theme.textSecondary }}>本地漏洞数据浏览与检索</Text>
        </div>
        <Button type="primary" icon={<UploadOutlined />} style={{ background: theme.primary, borderColor: theme.primary }} onClick={openImportModal}>
          导入数据包
        </Button>
      </div>

      <Alert
        style={{ marginBottom: 16, borderColor: theme.border, background: theme.card }}
        type="info"
        showIcon
        message={<span style={{ color: theme.text }}>平台外同步，平台内导入</span>}
        description={(
          <div style={{ color: theme.textSecondary }}>
            平台内已停用在线同步。请在平台外完成 NVD / CNVD / CNNVD 数据整理，再通过本页导入 CSV 数据包。
            {lastImportSummary ? <div style={{ marginTop: 8, color: theme.text }}>最近导入：{lastImportSummary}</div> : null}
          </div>
        )}
      />

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <div style={{ background: theme.card, borderRadius: 12, padding: 16, border: `1px solid ${theme.border}` }}>
            <Statistic
              title={<span style={{ color: theme.textSecondary }}>漏洞总数</span>}
              value={stats.total || 0}
              valueStyle={{ color: theme.text }}
              prefix={<DatabaseOutlined style={{ color: theme.primary }} />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: theme.card, borderRadius: 12, padding: 16, border: `1px solid ${theme.border}` }}>
            <Statistic
              title={<span style={{ color: theme.textSecondary }}>严重漏洞</span>}
              value={stats.critical || 0}
              valueStyle={{ color: theme.critical }}
              prefix={<BugOutlined />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: theme.card, borderRadius: 12, padding: 16, border: `1px solid ${theme.border}` }}>
            <Statistic
              title={<span style={{ color: theme.textSecondary }}>高危漏洞</span>}
              value={stats.high || 0}
              valueStyle={{ color: theme.high }}
              prefix={<SafetyOutlined />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: theme.card, borderRadius: 12, padding: 16, border: `1px solid ${theme.border}` }}>
            <Statistic
              title={<span style={{ color: theme.textSecondary }}>本周新增</span>}
              value={stats.this_week || 0}
              valueStyle={{ color: theme.accent }}
              prefix={<ClockCircleOutlined />}
            />
          </div>
        </Col>
      </Row>

      {/* 搜索和筛选 */}
      <Card style={{ background: theme.card, borderColor: theme.border, marginBottom: 16 }}>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={8}>
            <Search
              placeholder="搜索 CVE ID、漏洞标题、产品名"
              allowClear
              enterButton={<SearchOutlined />}
              onSearch={handleSearch}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </Col>
          <Col xs={12} sm={4}>
            <Select
              placeholder="严重程度"
              allowClear
              value={severity || undefined}
              onChange={(v) => { setSeverity(v || ''); setPage(1); fetchData(); }}
              style={{ width: '100%' }}
            >
              <Option value="critical">严重</Option>
              <Option value="high">高危</Option>
              <Option value="medium">中危</Option>
              <Option value="low">低危</Option>
            </Select>
          </Col>
          <Col xs={12} sm={4}>
            <Select
              placeholder="漏洞类型"
              allowClear
              value={vulnType || undefined}
              onChange={(v) => { setVulnType(v || ''); setPage(1); fetchData(); }}
              style={{ width: '100%' }}
            >
              <Option value="rce">RCE</Option>
              <Option value="sql-injection">SQL注入</Option>
              <Option value="xss">XSS</Option>
              <Option value="ssrf">SSRF</Option>
              <Option value="lfi">文件包含</Option>
              <Option value="auth-bypass">认证绕过</Option>
            </Select>
          </Col>
          <Col xs={24} sm={8} style={{ textAlign: 'right' }}>
            <Text style={{ color: theme.textSecondary }}>
              同步功能已停用
            </Text>
          </Col>
        </Row>
      </Card>

      {/* 漏洞列表 */}
      <Card style={{ background: theme.card, borderColor: theme.border }}>
        <Table
          columns={columns}
          dataSource={vulns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (p, ps) => {
              setPage(p)
              setPageSize(ps)
            },
            showTotal: (t) => `共 ${t} 条`,
          }}
          locale={{ emptyText: '暂无漏洞数据' }}
        />
      </Card>

      {/* 漏洞详情弹窗 */}
      <Modal
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={800}
        title="漏洞详情"
        centered
      >
        {selectedVuln && (
          <VulnDetail vuln={selectedVuln} />
        )}
      </Modal>

      <Modal
        open={importVisible}
        onCancel={() => setImportVisible(false)}
        onOk={handleImport}
        confirmLoading={importing}
        okText="开始导入"
        cancelText="取消"
        width={760}
        title="导入漏洞数据包"
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Alert
            type="info"
            showIcon
            message="导入说明"
            description="请先在平台外完成漏洞数据同步与清洗，再将生成的 CSV 文件上传到这里。平台内不会再主动发起在线同步。"
          />

          <div>
            <Text style={{ display: 'block', marginBottom: 8 }}>数据来源</Text>
            <Select value={importSource} onChange={setImportSource} style={{ width: 220 }}>
              <Option value="nvd">NVD / 通用 CVE 数据</Option>
              <Option value="cnvd">CNVD 关联数据</Option>
              <Option value="cnnvd">CNNVD 关联数据</Option>
            </Select>
          </div>

          <Alert
            type="warning"
            showIcon
            message="格式要求"
            description={importTemplateHint[importSource]}
          />

          <Upload.Dragger
            accept=".csv,.txt"
            multiple={false}
            beforeUpload={readLocalFile}
            showUploadList={false}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined style={{ color: theme.primary }} />
            </p>
            <p className="ant-upload-text">{importFileName ? `已选择文件：${importFileName}` : '点击或拖拽上传 CSV 数据包'}</p>
            <p className="ant-upload-hint">上传后会自动读取内容，你也可以直接在下方文本框粘贴。</p>
          </Upload.Dragger>

          <Input.TextArea
            rows={12}
            value={importContent}
            placeholder="在这里粘贴 CSV 内容，或先上传本地 CSV 文件。"
            onChange={(event) => setImportContent(event.target.value)}
          />
        </Space>
      </Modal>
    </div>
  )
}
