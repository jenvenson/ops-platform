// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Alert, Button, Card, Col, Form, Radio, Row, Space, Tag, Typography } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import type { FormInstance } from 'antd/es/form'

const { Text, Paragraph } = Typography

type AccessExpectation = 'unknown' | 'public' | 'login-required'
type AuthComplexity = 'unknown' | 'standard' | 'custom' | 'captcha'
type SessionCarrier = 'unknown' | 'cookie' | 'header' | 'mixed'

type WebScanPrecheckValues = {
  target?: string
  login_url?: string
  username?: string
  precheck_access?: AccessExpectation
  precheck_auth_complexity?: AuthComplexity
  precheck_session_carrier?: SessionCarrier
}

type PrecheckLevel = 'direct' | 'custom' | 'blocked'

type PrecheckResult = {
  level: PrecheckLevel
  title: string
  description: string
  reasons: string[]
  assistantQuery: string
}

function getLevelMeta(t: (key: string, fallback: string) => string): Record<PrecheckLevel, { color: string; text: string }> {
  return {
    direct: { color: 'green', text: t('webScanPrecheck.direct', '可直接接入') },
    custom: { color: 'gold', text: t('webScanPrecheck.custom', '需要定制认证流') },
    blocked: { color: 'red', text: t('webScanPrecheck.blocked', '暂不建议自动扫描') },
  }
}

const triggerAssistantPrompt = (query: string) => {
  window.dispatchEvent(new CustomEvent('ops-assistant:prompt', {
    detail: { query },
  }))
}

function firstTarget(target?: string) {
  return target
    ?.split(/[,\n]/)
    .map((item) => item.trim())
    .find(Boolean) || ''
}

function containsKeyword(value: string, keywords: string[]) {
  const lower = value.toLowerCase()
  return keywords.some((keyword) => lower.includes(keyword))
}

function inferResult(values: WebScanPrecheckValues, t: (key: string, fallback: string) => string): PrecheckResult {
  const target = firstTarget(values.target)
  const loginURL = values.login_url?.trim() || ''
  const access = values.precheck_access || 'unknown'
  const authComplexity = values.precheck_auth_complexity || 'unknown'
  const sessionCarrier = values.precheck_session_carrier || 'unknown'
  const reasons: string[] = []

  let level: PrecheckLevel = 'direct'

  if (!target) {
    level = 'custom'
    reasons.push(t('webScanPrecheck.reason.noTarget', '还没有填写目标 URL，无法判断是否适合直接接入。'))
  }

  if (access === 'public') {
    level = 'custom'
    reasons.push(t('webScanPrecheck.reason.publicAccess', '该站点看起来允许匿名访问，当前 Web 漏扫链路更偏向登录后扫描。'))
  }

  if (authComplexity === 'custom') {
    level = 'custom'
    reasons.push(t('webScanPrecheck.reason.customAuth', '登录链路包含签名、加密密码、租户标识或动态参数，通常需要定制 auth_flow。'))
  }

  if (authComplexity === 'captcha') {
    level = 'blocked'
    reasons.push(t('webScanPrecheck.reason.captcha', '登录链路依赖验证码、短信、扫码或人机校验，当前不适合直接自动扫描。'))
  }

  if (sessionCarrier === 'mixed') {
    level = level === 'blocked' ? 'blocked' : 'custom'
    reasons.push(t('webScanPrecheck.reason.mixedSession', '登录后会话依赖 Cookie 和 Header 混合态，建议先人工确认会话复用方式。'))
  }

  if (sessionCarrier === 'unknown') {
    reasons.push(t('webScanPrecheck.reason.unknownSession', '会话载体还不明确，建议先确认登录后是 Cookie 还是 Header Token。'))
  }

  if (!values.username?.trim() && access !== 'public') {
    level = level === 'blocked' ? 'blocked' : 'custom'
    reasons.push(t('webScanPrecheck.reason.noAccount', '未提供测试账号，后续即使建任务也很难完成登录态验证。'))
  }

  if (containsKeyword(target, ['/open/', '/public/', '/anonymous/']) || containsKeyword(loginURL, ['oauth2', 'token'])) {
    if (authComplexity === 'unknown') {
      level = level === 'blocked' ? 'blocked' : 'custom'
      reasons.push(t('webScanPrecheck.reason.openEndpoint', '目标 URL 或登录接口看起来存在开放接口或 token 登录特征，建议先做接入预检查。'))
    }
  }

  if (reasons.length === 0) {
    reasons.push(t('webScanPrecheck.reason.standard', '当前信息更接近常规账号密码 + 可复用会话的网站，可以先按标准流程接入。'))
  }

  const title = level === 'direct'
    ? t('webScanPrecheck.title.direct', '该网站更像常规登录站点')
    : level === 'custom'
      ? t('webScanPrecheck.title.custom', '该网站可能需要额外认证适配')
      : t('webScanPrecheck.title.blocked', '该网站当前不适合直接自动扫描')

  const description = level === 'direct'
    ? t('webScanPrecheck.desc.direct', '建议先按标准登录后 Web 扫描创建任务；如果自动生成的 auth_flow 不稳定，再转定制认证。')
    : level === 'custom'
      ? t('webScanPrecheck.desc.custom', '建议先让运维小助手协助判断登录链路，再决定是否生成或调整 auth_flow。')
      : t('webScanPrecheck.desc.blocked', '建议先人工复盘登录流程，确认是否存在验证码、短信、扫码或设备校验，再决定是否走自动化扫描。')

  const levelMetaObj = getLevelMeta(t)
  const assistantQuery = [
    t('webScanPrecheck.assistant.query1', '请帮我判断这个网站能否直接接入登录后 Web 漏洞扫描。'),
    `${t('webScanPrecheck.assistant.target', '目标 URL')}: ${target || '-'}`,
    `${t('webScanPrecheck.assistant.loginUrl', '登录 URL')}: ${loginURL || '-'}`,
    `${t('webScanPrecheck.assistant.access', '访问要求')}: ${access}`,
    `${t('webScanPrecheck.assistant.authComplexity', '登录复杂度')}: ${authComplexity}`,
    `${t('webScanPrecheck.assistant.sessionCarrier', '会话方式')}: ${sessionCarrier}`,
    `${t('webScanPrecheck.assistant.testAccount', '测试账号')}: ${values.username?.trim() ? t('webScanPrecheck.assistant.provided', '已提供') : t('webScanPrecheck.assistant.notProvided', '未提供')}`,
    `${t('webScanPrecheck.assistant.pagePrediction', '页面预判')}: ${levelMetaObj[level].text}`,
    t('webScanPrecheck.assistant.query2', '如果不能直接接入，请只回答：需要补什么关键信息，或为什么暂时不建议自动扫描。'),
  ].join('\n')

  return { level, title, description, reasons, assistantQuery }
}

type WebScanPrecheckCardProps = {
  form: FormInstance
  compact?: boolean
}

export default function WebScanPrecheckCard({ form, compact = false }: WebScanPrecheckCardProps) {
  const { t } = useTranslation('security')
  const values = Form.useWatch([], form) as WebScanPrecheckValues | undefined
  const result = inferResult(values || {}, t)
  const levelMetaObj = getLevelMeta(t)
  const level = levelMetaObj[result.level]

  return (
    <Card
      size="small"
      title={t('webScanPrecheck.cardTitle', '接入预检查')}
      style={{ marginBottom: compact ? 16 : 24 }}
      extra={<Tag color={level.color}>{level.text}</Tag>}
    >
      <Space direction="vertical" style={{ width: '100%' }} size={12}>
        <Alert
          type={result.level === 'direct' ? 'success' : result.level === 'custom' ? 'warning' : 'error'}
          showIcon
          message={result.title}
          description={result.description}
        />

        <Row gutter={16}>
          <Col span={24}>
            <Form.Item
              name="precheck_access"
              label={t('webScanPrecheck.accessLabel', '该站点是否本来就要求登录后访问')}
              initialValue="login-required"
            >
              <Radio.Group optionType="button" buttonStyle="solid">
                <Radio.Button value="login-required">{t('webScanPrecheck.loginRequired', '要求登录')}</Radio.Button>
                <Radio.Button value="public">{t('webScanPrecheck.public', '允许匿名')}</Radio.Button>
                <Radio.Button value="unknown">{t('webScanPrecheck.unsure', '暂不确定')}</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Col>
          <Col span={24}>
            <Form.Item
              name="precheck_auth_complexity"
              label={t('webScanPrecheck.authLabel', '登录复杂度')}
              initialValue="unknown"
              extra={t('webScanPrecheck.authExtra', '如果登录要做签名、加密密码、租户参数或动态 token，选「定制认证」。')}
            >
              <Radio.Group optionType="button" buttonStyle="solid">
                <Radio.Button value="standard">{t('webScanPrecheck.standardForm', '常规表单/Token')}</Radio.Button>
                <Radio.Button value="custom">{t('webScanPrecheck.customAuth', '定制认证')}</Radio.Button>
                <Radio.Button value="captcha">{t('webScanPrecheck.captchaAuth', '验证码/短信/扫码')}</Radio.Button>
                <Radio.Button value="unknown">{t('webScanPrecheck.unsure', '暂不确定')}</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Col>
          <Col span={24}>
            <Form.Item
              name="precheck_session_carrier"
              label={t('webScanPrecheck.sessionLabel', '登录后会话载体')}
              initialValue="unknown"
              extra={t('webScanPrecheck.sessionExtra', '会话复用越明确，自动扫描越稳。')}
            >
              <Radio.Group optionType="button" buttonStyle="solid">
                <Radio.Button value="cookie">Cookie</Radio.Button>
                <Radio.Button value="header">Header Token</Radio.Button>
                <Radio.Button value="mixed">{t('webScanPrecheck.mixed', '混合态')}</Radio.Button>
                <Radio.Button value="unknown">{t('webScanPrecheck.unsure', '暂不确定')}</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Col>
        </Row>

        <div>
          <Text strong>{t('webScanPrecheck.judgment', '初步判断')}</Text>
          <div style={{ marginTop: 8 }}>
            {result.reasons.map((reason) => (
              <div key={reason} style={{ marginBottom: 4, color: 'rgba(0,0,0,0.72)' }}>
                - {reason}
              </div>
            ))}
          </div>
        </div>

        <div>
          <Paragraph type="secondary" style={{ marginBottom: 8 }}>
            {t('webScanPrecheck.assistantHint', '如果你对登录方式、会话复用或是否需要定制认证拿不准，再让运维小助手补一层判断。')}
          </Paragraph>
          <Button icon={<RobotOutlined />} onClick={() => triggerAssistantPrompt(result.assistantQuery)}>
            {t('webScanPrecheck.askAssistant', '不确定？问小助手')}
          </Button>
        </div>
      </Space>
    </Card>
  )
}