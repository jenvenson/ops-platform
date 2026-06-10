// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

import { useState } from 'react';
import { Card, Tabs, Typography } from 'antd';
import ViewsPage from './ViewsPage';

const { Title } = Typography;

export default function JenkinsManagementPage() {
  const [activeKey, setActiveKey] = useState('views');

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <Title level={2} style={{ marginBottom: 24 }}>Jenkins管理</Title>

        <Tabs
          activeKey={activeKey}
          onChange={setActiveKey}
          items={[
            {
              key: 'views',
              label: '视图管理',
              children: <ViewsPage />,
            },
          ]}
        />
      </Card>
    </div>
  );
}