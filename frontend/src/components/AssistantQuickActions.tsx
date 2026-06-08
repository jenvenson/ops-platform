import { Button, Card, Space } from 'antd'
import { RobotOutlined } from '@ant-design/icons'

type AssistantQuickAction = {
  label: string
  query: string
}

type AssistantQuickActionsProps = {
  description: string
  actions: AssistantQuickAction[]
}

const triggerAssistantPrompt = (query: string) => {
  window.dispatchEvent(new CustomEvent('ops-assistant:prompt', {
    detail: { query },
  }))
}

export default function AssistantQuickActions({ description, actions }: AssistantQuickActionsProps) {
  return (
    <Card
      size="small"
      style={{ marginBottom: 16 }}
      title={(
        <Space size={8}>
          <RobotOutlined />
          <span>智能分析</span>
        </Space>
      )}
      extra={<span style={{ color: '#8c8c8c', fontSize: 12 }}>{description}</span>}
    >
      <Space wrap>
        {actions.map((action) => (
          <Button key={`${action.label}-${action.query}`} onClick={() => triggerAssistantPrompt(action.query)}>
            {action.label}
          </Button>
        ))}
      </Space>
    </Card>
  )
}
