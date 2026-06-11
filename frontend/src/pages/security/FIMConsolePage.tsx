// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { Tabs } from 'antd'
import { useTranslation } from 'react-i18next'
import FIMPoliciesPage from './FIMPoliciesPage'
import FIMEventsPage from './FIMEventsPage'
import FIMAlertsPage from './FIMAlertsPage'

export default function FIMConsolePage() {
  const { t } = useTranslation('security')
  return (
    <Tabs
      defaultActiveKey="policies"
      items={[
        {
          key: 'policies',
          label: t('fim.policies', '巡检策略'),
          children: <FIMPoliciesPage />,
        },
        {
          key: 'events',
          label: t('fimEvents.title', '文件变更事件'),
          children: <FIMEventsPage />,
        },
        {
          key: 'alerts',
          label: t('fimAlerts.title', '完整性告警'),
          children: <FIMAlertsPage />,
        },
      ]}
    />
  )
}