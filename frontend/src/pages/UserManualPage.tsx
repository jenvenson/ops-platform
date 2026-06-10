// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Card, Col, Row, Space, Tag, Typography } from 'antd'
import manualData from '../data/user_manual.json'

const { Paragraph, Title } = Typography

type ManualSection = {
  title: string
  content: string
  keywords?: string[]
}

type ManualData = {
  manual_contents: Record<string, ManualSection>
}

const data = manualData as ManualData
const sections = Object.values(data.manual_contents)

export default function UserManualPage() {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card>
        <Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>
          用户手册
        </Title>
        <Paragraph type="secondary" style={{ marginBottom: 0 }}>
          这里汇总了平台主要模块的使用说明、常见操作路径和故障排查信息，内容与当前菜单命名保持一致。
        </Paragraph>
      </Card>

      <Row gutter={[16, 16]}>
        {sections.map((section) => (
          <Col xs={24} lg={12} key={section.title}>
            <Card title={section.title} style={{ height: '100%' }}>
              <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 16 }}>
                {section.content}
              </Paragraph>
              {section.keywords && section.keywords.length > 0 ? (
                <Space size={[8, 8]} wrap>
                  {section.keywords.map((keyword) => (
                    <Tag key={keyword}>{keyword}</Tag>
                  ))}
                </Space>
              ) : null}
            </Card>
          </Col>
        ))}
      </Row>
    </Space>
  )
}