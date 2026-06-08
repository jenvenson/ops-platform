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
            {
              key: 'jobs',
              label: '作业管理',
              children: (
                <Card>
                  <p>作业管理功能正在开发中...</p>
                </Card>
              ),
            },
            {
              key: 'script',
              label: '脚本执行',
              children: (
                <Card>
                  <h3>Jenkins Script Console 脚本执行</h3>
                  <p>可以直接在Jenkins Script Console中执行Groovy脚本来管理视图和作业</p>
                  <pre style={{ background: '#f6f6f6', padding: '12px', borderRadius: '4px', marginTop: '12px' }}>
                    {`// Groovy脚本示例：复制视图并替换Job名称和Tag值
def sourceViewName = "${'${SOURCE_VIEW}'}"
def targetViewName = "${'${TARGET_VIEW}'}"
def newTagValue = "${'${NEW_TAG_VALUE}'}"

def sourceView = Jenkins.instance.getView(sourceViewName)
def targetView = Jenkins.instance.getView(targetViewName)

if (!targetView) {
    targetView = new ListView(targetViewName)
    Jenkins.instance.addView(targetView)
}

// 遍历源视图中的所有Job并创建副本
sourceView.getItems().each { job ->
    def newJobName = job.name.replace(sourceViewName, targetViewName)
    // 在Job配置中查找并替换Tag值
    // 复制Job逻辑...
}`}
                  </pre>
                </Card>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}