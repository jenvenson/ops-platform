# 智能化一期首批任务拆解

## 1. 文档目标

本文档用于将《智能化一期实施拆解》中建议优先启动的 8 项任务，进一步拆成可直接排期、分配和实施的开发任务单。

本文档重点解决以下问题：

- 每项任务具体要交付什么
- 每项任务改哪些模块
- 每项任务依赖什么前置条件
- 每项任务如何验收
- 哪些任务可以并行，哪些必须串行

本文档承接：

- [`2026-03-24-intelligent-phase1-implementation-breakdown.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-intelligent-phase1-implementation-breakdown.md)

## 2. 首批任务总览

建议将首批实施任务定义为 8 个开发任务：

1. assistant 统一意图结构与决策输出结构
2. assistant 页面上下文输入字段
3. assistant 工具注册与调用接口
4. 只读工具第一批接入
5. `platform_object_index` 落地
6. `platform_event_stream` 落地
7. deploy 与 alert 页面内智能入口试点
8. 回归测试与浏览器冒烟补齐

## 3. 任务拆解

### T1：assistant 统一意图结构与决策输出结构

任务目标：

- 将当前 assistant 的分散意图识别和返回结构，收口成统一的内部协议，作为后续工具规划和上下文组装的基础。

建议产出：

- `AssistantIntent`
- `AssistantContext`
- `AssistantDecision`
- `AssistantAction`
- `AssistantResultCard`
- `AssistantExecutionPlan`

主要改动范围：

- [`backend/internal/assistant/types.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/types.go)
- [`backend/internal/assistant/service.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/service.go)
- [`backend/internal/models/assistant.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/models/assistant.go)

实施要点：

- 保留当前前端消费字段，先做兼容扩展，不做一次性替换
- 意图至少覆盖 `knowledge_qa`、`page_navigation`、`readonly_query`、`analysis`、`execution`
- 决策结构必须兼容引用、动作、结果卡片、风险级别和确认态

前置依赖：

- 无

验收标准：

- 现有 assistant API 不回退
- 现有“如何归档打包”“如何查看归档历史”“如何删除会话”测试继续通过
- 新结构可用于后续工具规划层

建议优先级：

- `P0`

### T2：assistant 页面上下文输入字段

任务目标：

- 让 assistant 请求能够携带页面路径、对象 ID、选中记录 ID 等上下文，减少用户重复描述当前页面状态。

建议产出：

- assistant 请求体新增页面上下文结构
- 前端发送页面上下文
- 后端读取并挂入会话上下文

主要改动范围：

- [`frontend/src/components/AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)
- [`backend/internal/assistant/handler.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/handler.go)
- [`backend/internal/assistant/types.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/types.go)

建议字段：

- `pagePath`
- `moduleKey`
- `objectType`
- `objectId`
- `selectedRecordIds`
- `pageTitle`

前置依赖：

- T1

验收标准：

- 页面上下文为可选字段，不影响现有调用方
- assistant 可在日志或调试输出中读取到页面上下文
- 页面内问答无需手工重复页面名称也能命中正确场景

建议优先级：

- `P0`

### T3：assistant 工具注册与调用接口

任务目标：

- 将 assistant 当前散落在 `service.go` 的只读查询逻辑，抽象成可注册、可调用、可审计的工具接口。

建议产出：

- 工具接口定义
- 工具注册表
- 工具执行结果结构
- 工具统一错误结构
- 工具调用审计字段

主要改动范围：

- [`backend/internal/assistant/service.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/service.go)
- 可能新增 `backend/internal/assistant/tools.go`
- 可能新增 `backend/internal/assistant/tool_registry.go`

建议接口能力：

- 根据意图决定是否调用工具
- 工具执行时继承用户身份
- 工具可返回结构化结果而不是只返回文本
- 工具超时和错误有统一包装

前置依赖：

- T1

验收标准：

- 至少支持注册 2 个真实只读工具
- 工具失败不会导致 assistant 整体崩溃
- 工具调用结果可回写到 assistant 决策结构

建议优先级：

- `P0`

### T4：只读工具第一批接入

任务目标：

- 将 deploy、alert 相关的第一批只读能力正式接入 assistant 工具层。

建议首批工具：

- `query_release_history`
- `query_archive_history`
- `query_alert_events`

第二优先工具：

- `query_vulnerability_summary`
- `query_object_summary`

主要改动范围：

- [`backend/internal/cicd`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/cicd)
- [`backend/internal/alert`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/alert)
- [`backend/internal/assistant`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)

实施要点：

- 尽量复用现有业务查询逻辑，不在 assistant 里直接查底表
- 工具接口只暴露只读信息，不暴露执行操作
- 对工具输出做统一摘要字段抽取

前置依赖：

- T3

验收标准：

- assistant 可基于工具结果回答最近发布、归档历史、告警列表类问题
- 工具返回结果结构稳定
- 权限继承不绕过原业务限制

建议优先级：

- `P0`

### T5：`platform_object_index` 落地

任务目标：

- 为核心对象建立统一索引表，提供稳定对象主键和跨模块摘要。

一期建议纳入对象：

- `Project`
- `Application`
- `DeployRecord`

第二阶段再补：

- `AlertEvent`
- `Vulnerability`

主要改动范围：

- 新增迁移脚本
- 可能新增 `backend/internal/platformobject`
- 对应业务模块的对象同步写入逻辑

建议表字段：

- `object_uid`
- `object_type`
- `source_module`
- `source_pk`
- `title`
- `summary`
- `status`
- `owner_id`
- `metadata`
- `created_at`
- `updated_at`

前置依赖：

- 无

验收标准：

- 至少 3 类对象可写入并查询
- assistant 能通过对象索引查到对象摘要
- 对象索引支持幂等更新

建议优先级：

- `P1`

### T6：`platform_event_stream` 落地

任务目标：

- 为关键事件建立统一事件流，作为时间线和后续智能分析的数据底座。

一期建议接入事件：

- deploy 事件
- alert 事件
- assistant 会话事件

主要改动范围：

- 新增迁移脚本
- 可能新增 `backend/internal/platformevent`
- `backend/internal/cicd`
- `backend/internal/alert`
- [`backend/internal/assistant`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)

建议表字段：

- `event_uid`
- `event_type`
- `object_uid`
- `related_object_uids`
- `operator_id`
- `status`
- `summary`
- `metadata`
- `occurred_at`
- `source_module`

前置依赖：

- T5 最好先完成对象 UID 规则，但不强制阻塞表结构建设

验收标准：

- 至少 3 类事件写入成功
- 按对象查最近事件可用
- assistant 可读取“最近相关事件”摘要

建议优先级：

- `P1`

### T7：deploy 与 alert 页面内智能入口试点

任务目标：

- 让用户在 deploy、alert 页面内直接触发上下文智能问答和只读分析，而不是只能从全局聊天入口进入。

建议试点页面：

- 发布历史页
- 归档历史页
- 告警列表页或告警详情页

主要改动范围：

- [`frontend/src/components/AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)
- `frontend/src/pages/deploy/*`
- `frontend/src/pages/alert/*`

建议交互：

- “智能分析”
- “解释一下”
- “查看相关记录”

前置依赖：

- T2
- T4

验收标准：

- 页面能自动传入上下文
- assistant 响应能针对当前对象或记录
- 页面入口不破坏现有页面操作流

建议优先级：

- `P1`

### T8：回归测试与浏览器冒烟补齐

任务目标：

- 为首批智能化实施改造补齐回归保护，避免 assistant 重构和页面接入造成能力回退。

建议覆盖：

- assistant 意图识别单测
- assistant 工具规划单测
- 工具调用集成测试
- 对象索引与事件流单测
- 页面内智能入口浏览器冒烟

主要改动范围：

- [`backend/internal/assistant`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)
- 新增对象/事件模块测试
- [`frontend/scripts/ui-acceptance-smoke.cjs`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/scripts/ui-acceptance-smoke.cjs)

前置依赖：

- T1 至 T7 按实现进度逐步接入

验收标准：

- 关键问法测试不回退
- 核心 API 集成测试可跑
- `npm run acceptance:smoke:core` 持续通过
- 页面内试点入口有最小冒烟覆盖

建议优先级：

- `P0`

## 4. 依赖关系

推荐依赖顺序如下：

1. T1
2. T2 与 T3
3. T4
4. T8 第一轮回归补齐
5. T5
6. T6
7. T7
8. T8 第二轮页面与底座回归补齐

说明：

- T1 是 assistant 主链路重构的基础
- T2、T3 可以并行
- T4 必须建立在工具接口抽象之后
- T5、T6 是数据底座，可与 T4 局部并行，但建议晚于 assistant 主链路结构化
- T7 必须等上下文协议和只读工具具备可用性
- T8 不是最后才做，而是分轮次插入

## 5. 建议迭代排期

建议分成 3 个迭代包：

### Iteration A：assistant 主链路结构化

包含：

- T1
- T2
- T3
- T8 第一轮

目标：

- assistant 从单点功能演进为可扩展编排骨架

### Iteration B：只读工具与最小底座

包含：

- T4
- T5
- T6

目标：

- assistant 具备真实业务数据接入能力

### Iteration C：页面内智能入口试点

包含：

- T7
- T8 第二轮

目标：

- 智能能力从全局助手扩展到页面内上下文入口

## 6. 建议的人力分工

如果按最小团队实施，建议按 3 条线分工：

- 后端编排线：负责 T1、T3、T4
- 后端数据线：负责 T5、T6
- 前端交互线：负责 T2、T7、T8 前端部分

如果只有 1 到 2 人实施，则优先顺序不要变，宁可减少试点页面，也不要跳过 T1 到 T4。

## 7. 第一周建议落地内容

如果从明天开始实施，第一周最值得完成的是：

1. 完成 T1 的结构定义与兼容改造
2. 完成 T2 的页面上下文协议
3. 完成 T3 的工具接口抽象
4. 为 T1 到 T3 补最小单测和回归测试

原因：

- 这一步完成后，平台智能化的实施主骨架才成立
- 后续对象索引、事件流和页面内入口都能直接挂接到这套骨架上
