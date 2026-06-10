// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Tabs } from 'antd'
import FIMPoliciesPage from './FIMPoliciesPage'
import FIMEventsPage from './FIMEventsPage'
import FIMAlertsPage from './FIMAlertsPage'

export default function FIMConsolePage() {
  return (
    <Tabs
      defaultActiveKey="policies"
      items={[
        {
          key: 'policies',
          label: '巡检策略',
          children: <FIMPoliciesPage />,
        },
        {
          key: 'events',
          label: '文件变更事件',
          children: <FIMEventsPage />,
        },
        {
          key: 'alerts',
          label: '完整性告警',
          children: <FIMAlertsPage />,
        },
      ]}
    />
  )
}