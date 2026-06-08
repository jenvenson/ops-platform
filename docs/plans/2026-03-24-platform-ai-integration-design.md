# 现有平台功能与智能能力集成设计

## 1. 文档目标

本文档用于说明 OPS Platform 现有功能模块如何与智能能力集成，重点解决以下问题：

- 智能能力如何接入现有前端页面和后端模块
- assistant 如何复用现有平台业务能力，而不是重复造业务系统
- 页面入口、聊天入口、工具层和事件回写如何打通
- 集成顺序和最小改造范围如何定义

本文档是 [`2026-03-24-intelligent-ops-platform-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-intelligent-ops-platform-design.md) 的配套实现设计，聚焦“如何接到现有平台里”。

## 2. 集成原则

### 2.1 不新建第二个平台

智能能力不是独立产品，而是现有 OPS Platform 的认知层、分析层和编排层。

现有平台继续负责：

- 业务对象管理
- 权限控制
- 页面交互
- 数据存储
- 具体执行逻辑

智能层负责：

- 理解问题
- 组装上下文
- 调用受控工具
- 生成建议
- 返回结构化结果

### 2.2 业务归业务，智能归编排

现有后端模块仍然是业务 owner：

- `backend/internal/cmdb`
- `backend/internal/cicd`
- `backend/internal/consul`
- `backend/internal/alert`
- `backend/internal/monitor`
- `backend/internal/security`
- `backend/internal/auth`

`backend/internal/assistant` 不接管业务逻辑，只做：

- 会话管理
- 意图识别
- 工具编排
- 知识检索
- 回答生成
- 审计回写

### 2.3 前端双入口

智能能力在前端采用双入口结构：

- `全局入口`
  继续使用 [`AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)
- `页面内入口`
  在业务页面增加“智能分析”“智能建议”“解释一下”入口

## 3. 与现有前端结构的集成

### 3.1 当前前端基础

当前前端页面已按业务域划分：

- `frontend/src/pages/cmdb/*`
- `frontend/src/pages/deploy/*`
- `frontend/src/pages/consul/*`
- `frontend/src/pages/alarm/*`
- `frontend/src/pages/monitor/*`
- `frontend/src/pages/security/*`
- `frontend/src/pages/admin/*`

全局聊天入口已存在：

- [`AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)

### 3.2 前端集成方式

建议新增统一的页面内智能入口组件：

```text
frontend/src/components/
  ├── AIChatbot.tsx
  ├── AIInsightPanel.tsx
  ├── AIActionCard.tsx
  └── AIFloatingEntry.tsx
```

职责建议：

- `AIChatbot.tsx`
  全局跨模块提问
- `AIInsightPanel.tsx`
  页面内展示“智能分析”结果
- `AIActionCard.tsx`
  展示建议、风险等级、引用和候选动作
- `AIFloatingEntry.tsx`
  页面级悬浮触发入口

### 3.3 页面内智能入口落点

#### 变更发布

页面：

- [`AppReleasePage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/deploy/AppReleasePage.tsx)
- [`DeployHistoryPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/deploy/DeployHistoryPage.tsx)
- [`ArchivePage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/deploy/ArchivePage.tsx)
- [`ArchiveHistoryPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/deploy/ArchiveHistoryPage.tsx)

建议增加：

- `分析失败原因`
- `解释发布结果`
- `推荐回滚路径`
- `解释归档记录`

#### 告警事件

页面：

- [`AlertEventsPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/alarm/AlertEventsPage.tsx)
- [`AlertRulesPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/alarm/AlertRulesPage.tsx)

建议增加：

- `生成处置建议`
- `解释告警含义`
- `查看关联资产`

#### 安全中心

页面：

- [`TaskList.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/security/TaskList.tsx)
- [`VulnerabilityList.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/security/VulnerabilityList.tsx)
- [`VulnDetail.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/security/VulnDetail.tsx)

建议增加：

- `判断修复优先级`
- `分析影响范围`
- `生成修复建议`

#### 资产中心

页面：

- [`ProjectsPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/cmdb/ProjectsPage.tsx)
- [`ApplicationsPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/cmdb/ApplicationsPage.tsx)
- [`ServersPage.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/cmdb/ServersPage.tsx)

建议增加：

- `查看关联关系`
- `查看影响面`
- `生成资产摘要`

### 3.4 前端结果结构统一

页面入口和聊天入口都应消费统一响应结构：

```json
{
  "summary": "结论",
  "intent": "knowledge_qa|readonly_query|analysis|action_plan",
  "citations": [],
  "actions": [],
  "resultCards": [],
  "riskLevel": "low|medium|high|critical",
  "needConfirmation": false
}
```

这样页面和聊天入口可以共用 assistant 返回结构，不需要各自再定义一套 AI 数据协议。

## 4. 与现有后端结构的集成

### 4.1 当前后端基础

现有智能主链路位于：

- [`backend/internal/assistant/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)

现有业务模块位于：

- [`backend/internal/cmdb/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/cmdb)
- [`backend/internal/cicd/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/cicd)
- [`backend/internal/consul/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/consul)
- [`backend/internal/alert/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/alert)
- [`backend/internal/monitor/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/monitor)
- [`backend/internal/security/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security)

### 4.2 后端集成原则

assistant 不应：

- 直接访问各领域底层表
- 直接拼接 SQL
- 直接请求 Jenkins、Consul、Grafana
- 直接决定高风险动作执行

assistant 应：

- 调用各领域模块暴露的受控工具
- 复用现有权限体系
- 统一返回结构化结果
- 将结果回写到会话和审计日志

### 4.3 工具层拆分建议

建议在 assistant 内部引入标准工具注册层：

```text
backend/internal/assistant/tools/
  ├── navigation/
  ├── cmdb/
  ├── deploy/
  ├── alert/
  ├── monitor/
  ├── security/
  └── execution/
```

每类工具只封装调用入口，不重写原业务逻辑。

#### CMDB 工具

建议抽取：

- `query_projects`
- `query_project_detail`
- `query_applications`
- `query_application_relations`
- `query_servers`

#### 发布工具

建议抽取：

- `query_release_history`
- `query_archive_history`
- `query_aggregate_history`
- `query_release_detail`
- `query_release_failures`

#### 告警工具

建议抽取：

- `query_alert_events`
- `query_alert_detail`
- `query_alert_rules`
- `query_alert_relations`

#### 安全工具

建议抽取：

- `query_vulnerabilities`
- `query_vulnerability_detail`
- `query_vulnerability_assets`
- `query_tickets`

#### 监控工具

建议抽取：

- `query_monitor_summary`
- `query_host_metrics`
- `query_dashboard_entry`

### 4.4 现有 assistant 模块扩展方向

建议在现有 assistant 主链路基础上继续增强：

- `handler.go`
  继续承接 API 协议
- `service.go`
  继续承接意图识别、动作过滤、引用过滤、工具编排
- `knowledge.go`
  继续承接静态知识和路由知识
- 新增 `tools/`
  承接业务工具集成
- 新增 `analysis/`
  承接多模块结果融合与风险分析

## 5. 现有平台功能与智能能力映射

### 5.1 资产中心

现有功能：

- 项目管理
- 环境管理
- 主机管理
- 应用流水线管理

适合集成的智能能力：

- 资产问答
- 关系查询
- 影响范围分析
- 资产摘要生成

典型输出：

- 某应用关联了哪些环境和主机
- 某主机最近影响了哪些发布
- 某项目的资产概览摘要

### 5.2 变更发布

现有功能：

- 迭代部署
- 部署记录
- 归档打包
- 归档历史
- 聚合打包

适合集成的智能能力：

- 如何做某项发布操作
- 发布失败原因分析
- 发布记录摘要
- 回滚建议
- 归档结果解释

典型输出：

- 某次部署失败的可能原因
- 某项目最近 10 次发布概况
- 某条归档记录的结果解释

### 5.3 Consul 配置

现有功能：

- 配置管理
- 批量配置下发
- 配置操作记录

适合集成的智能能力：

- 配置说明
- 配置变更影响分析
- 批量下发前风险提示

### 5.4 告警事件

现有功能：

- 事件中心
- 告警规则
- 联系人管理
- 通知渠道

适合集成的智能能力：

- 告警摘要
- 告警归因
- 处置建议
- 告警收敛建议

### 5.5 监控中心

现有功能：

- 监控大屏
- 监控概览
- Grafana 仪表盘

适合集成的智能能力：

- 指标解释
- 异常摘要
- 趋势分析

### 5.6 安全中心

现有功能：

- 扫描任务
- 安全资产
- 漏洞管理
- 漏洞工单
- 漏洞知识库

适合集成的智能能力：

- 漏洞优先级判断
- 影响资产分析
- 修复建议
- 风险汇总

## 6. 数据与审计回写

智能能力不能只返回文本，必须回写到现有平台数据流中。

建议回写位置：

- assistant 会话记录
- 统一智能审计表
- 对象时间线
- 高价值页面分析历史

建议回写字段：

- `request_id`
- `user_id`
- `scene`
- `object_type`
- `object_id`
- `input`
- `used_tools`
- `used_citations`
- `summary`
- `risk_level`
- `actions`
- `status`
- `created_at`

## 7. 交互设计建议

### 7.1 全局入口

继续保留右侧运维小助手，适合：

- 跨模块问答
- 页面定位
- 快速只读查询
- 简单分析

### 7.2 页面入口

在关键业务页面增加：

- `智能分析`
- `生成建议`
- `解释结果`
- `查看影响面`

页面入口应携带页面上下文，例如：

- 当前项目
- 当前应用
- 当前部署记录 ID
- 当前告警 ID
- 当前漏洞 ID

这样 assistant 不需要重复询问上下文。

## 8. 集成顺序建议

### 8.1 第一阶段

先完成“全局聊天入口 + 文档问答 + 页面导航 + 只读查询”。

当前已具备基础能力，可继续增强：

- 高频问法覆盖
- 引用准确性
- 页面内建议卡片

### 8.2 第二阶段

为 `deploy`、`alarm`、`security` 三大高价值模块增加页面内智能入口。

优先原因：

- 数据价值高
- 用户使用频率高
- 分析场景最明确

### 8.3 第三阶段

补统一事件流和对象时间线，为跨模块分析做底座。

### 8.4 第四阶段

再引入受控执行能力：

- 半自动发布
- 半自动回滚
- 半自动配置下发

## 9. 最小改造方案

如果要在现有项目上最小成本起步，建议只做以下改造：

### 前端

- 保留现有 [`AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)
- 在 `deploy`、`alarm`、`security` 页面增加一个 `AIInsightPanel`

### 后端

- 保留现有 [`backend/internal/assistant/`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)
- 从 `cmdb`、`cicd`、`alert`、`security` 抽只读工具
- 增加统一审计写入

### 数据

- 不先重构所有业务表
- 先在 assistant 层补一张统一分析/审计记录表
- 再逐步推进统一事件流

## 10. 结论

现有平台功能与智能能力集成的关键，不是再造一个“AI 子系统”，而是让智能层嵌入现有模块：

- 前端嵌入页面入口和全局入口
- 后端复用既有业务模块能力
- assistant 做统一编排
- 审计和事件回写做统一闭环

这样平台才能在保持现有功能稳定的前提下，逐步走向智能化，而不是把智能能力做成一个旁路功能。
