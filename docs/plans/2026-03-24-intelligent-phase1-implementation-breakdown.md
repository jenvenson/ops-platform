# 智能化一期实施拆解

## 1. 文档目标

本文档用于将智能化相关设计从“架构方案”转为“实施任务”，明确一期建设范围、实施顺序、模块拆分、交付物、验收标准和风险控制。

本文档不再讨论“要不要做”，而是回答 5 个实施问题：

- 一期到底做哪些能力，不做哪些能力
- 哪些改造先做，哪些改造后做
- 每个阶段需要改哪些模块
- 每个阶段的交付物和验收口径是什么
- 哪些风险需要提前规避

本文档承接以下设计：

- [`2026-03-24-intelligent-ops-platform-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-intelligent-ops-platform-design.md)
- [`2026-03-24-platform-ai-integration-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-platform-ai-integration-design.md)
- [`2026-03-24-unified-object-model-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-unified-object-model-design.md)
- [`2026-03-24-event-stream-and-timeline-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-event-stream-and-timeline-design.md)
- [`2026-03-24-assistant-orchestration-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-assistant-orchestration-design.md)

## 2. 一期目标与边界

### 2.1 一期目标

一期只做“辅助智能”最小闭环，不做自动执行闭环。

一期完成后，平台应具备以下能力：

- assistant 能识别高频问答、导航和部分只读查询
- assistant 能消费统一知识源和有限的对象上下文
- deploy、alert、security、cmdb 页面能逐步接入页面内智能入口
- assistant 的回答、查询和分析过程具备基础审计能力
- 平台具备统一对象索引和最小事件流骨架，为二期分析能力铺底

### 2.2 一期明确不做

一期不做以下内容：

- 高风险动作自动执行
- 通用 Agent 自主规划多轮执行
- 全量历史数据回灌
- 全模块深度关联分析
- 复杂审批流与编排中心
- 模型直连底层业务系统

### 2.3 一期成功标准

一期完成的最低成功标准：

- 至少 10 个高频问法有稳定答案和回归测试
- 至少 4 个业务模块支持只读工具接入 assistant
- 至少 3 类核心对象能建立统一对象索引
- 至少 3 类关键事件能进入统一事件流
- assistant 的核心浏览器冒烟和 API 回归可稳定通过

## 3. 一期实施原则

### 3.1 先底座，后场景

先完成对象索引、事件流骨架、工具层接口，再扩页面场景和问答覆盖。

### 3.2 先只读，后建议

先把只读查询和知识问答做好，再扩“分析建议”能力；避免在数据底座不稳定时直接做复杂分析。

### 3.3 先骨架接入，后深度治理

一期先把审计、确认、回写结构做出来，不要求一开始就覆盖全部智能场景。

## 4. 一期实施范围拆分

建议拆成 5 个工作流并行推进，但实施顺序仍以依赖关系为准。

### 4.1 工作流 A：assistant 编排重构

目标：

- 将当前 `service.go` 的固定链路升级为“意图 -> 上下文 -> 工具计划 -> 决策输出”的结构化链路

核心改造点：

- 拆分意图识别层
- 增加上下文组装器
- 定义工具计划结构
- 统一决策输出结构
- 补审计和回写埋点

主要涉及模块：

- [`backend/internal/assistant/service.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/service.go)
- [`backend/internal/assistant/handler.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/handler.go)
- [`backend/internal/assistant/types.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/types.go)
- [`backend/internal/models/assistant.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/models/assistant.go)

一期交付要求：

- 新增统一意图结构
- 新增统一工具计划结构
- 新增统一决策结构
- 保持现有问答与会话能力不回退

### 4.2 工作流 B：统一对象索引

目标：

- 为 assistant 和后续分析能力提供最小统一对象视图

一期对象范围：

- `Project`
- `Application`
- `DeployRecord`
- `AlertEvent`
- `Vulnerability`

建议建设内容：

- 设计 `platform_object_index`
- 建对象标准 ID 规则
- 增量同步现有业务表到对象索引
- 提供对象查询接口

主要涉及模块：

- `backend/internal/cmdb`
- `backend/internal/cicd`
- `backend/internal/alert`
- `backend/internal/security`
- 可能新增 `backend/internal/platformobject`

一期交付要求：

- 至少 3 类对象完成索引接入
- assistant 能按对象 ID 拉取基础摘要信息

### 4.3 工作流 C：统一事件流与时间线骨架

目标：

- 让 deploy、alert、assistant 至少 3 类事件进入统一事件流

一期事件范围：

- 发布事件
- 告警事件
- assistant 会话事件

建议建设内容：

- 设计 `platform_event_stream`
- 设计 `platform_timeline_entries`
- 编写事件归一化写入逻辑
- 建立对象时间线查询接口

主要涉及模块：

- `backend/internal/cicd`
- `backend/internal/alert`
- `backend/internal/assistant`
- 可能新增 `backend/internal/platformevent`

一期交付要求：

- 至少能查到某应用最近发布与告警事件
- assistant 可消费“最近事件”作为上下文补充

### 4.4 工作流 D：业务工具层接入

目标：

- 将现有业务模块能力以只读工具形式开放给 assistant

一期优先工具：

- `query_release_history`
- `query_archive_history`
- `query_alert_events`
- `query_vulnerability_summary`
- `query_object_summary`
- `navigate_to_page`

主要涉及模块：

- [`backend/internal/cicd`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/cicd)
- [`backend/internal/alert`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/alert)
- [`backend/internal/security`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security)
- [`backend/internal/cmdb`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/cmdb)
- [`backend/internal/assistant`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant)

一期交付要求：

- 每个工具都有明确输入输出结构
- 工具调用继承当前用户权限
- 工具错误能回到 assistant 统一错误结构

### 4.5 工作流 E：前端页面内智能入口

目标：

- 除全局助手外，在重点页面接入上下文化智能入口

一期优先页面：

- 发布历史页
- 归档打包页
- 归档历史页
- 告警列表或详情页
- 漏洞列表或详情页

建议建设内容：

- 定义页面上下文协议
- 在页面中注入对象 ID、页面路径、当前记录 ID
- 提供“智能分析”“解释一下”“查看相关记录”等入口
- 统一结果卡片展示

主要涉及模块：

- [`frontend/src/components/AIChatbot.tsx`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/components/AIChatbot.tsx)
- `frontend/src/pages/deploy/*`
- `frontend/src/pages/alert/*`
- `frontend/src/pages/security/*`
- `frontend/src/pages/cmdb/*`

一期交付要求：

- 至少 2 个业务页面支持页面上下文问答
- 页面上下文问题不需要用户重复输入对象名称

## 5. 里程碑拆解

建议按 4 个阶段落地。

### M1：assistant 编排收口

目标：

- 在不引入复杂底座的前提下，先把 assistant 主链路结构化

交付物：

- 意图结构、工具计划结构、决策结构
- 高频问法回归测试扩充
- assistant 审计字段补齐

完成标志：

- 现有高频问法不回退
- 新结构已落地但仍兼容当前前端展示

### M2：对象索引与工具层落地

目标：

- 打通统一对象视图和只读工具骨架

交付物：

- `platform_object_index`
- 对象查询接口
- 4 到 6 个只读工具

完成标志：

- assistant 能回答“这个应用最近有什么部署/告警/漏洞”

### M3：事件流与时间线接入

目标：

- 打通最小事件流和时间线查询

交付物：

- `platform_event_stream`
- `platform_timeline_entries`
- 发布、告警、assistant 事件接入

完成标志：

- assistant 可输出“最近相关事件”摘要

### M4：页面内智能入口试点

目标：

- 将智能能力从聊天框扩到页面上下文

交付物：

- 2 到 3 个页面内智能入口
- 页面上下文协议
- 结果卡片统一展示

完成标志：

- 用户在页面内可直接发起上下文相关提问和只读分析

## 6. 任务清单

### 6.1 后端任务

#### assistant 编排

- 重构 `classifyIntent()` 输出为统一意图结构
- 新增上下文组装器
- 新增工具计划器
- 新增统一决策输出结构
- 为工具调用、引用、决策补审计字段

#### 对象索引

- 新增对象索引模型和迁移脚本
- 为 deploy、alert、security 建索引写入器
- 提供对象详情与对象搜索接口

#### 事件流

- 新增事件流模型和迁移脚本
- 为 deploy、alert、assistant 建事件适配器
- 提供按对象查时间线接口

#### 工具层

- 抽象工具注册与调用接口
- 为优先业务模块补只读工具
- 建立统一错误码和超时策略

### 6.2 前端任务

- 为 assistant 请求增加页面上下文字段
- 新增结果卡片渲染骨架
- 为重点页面注入“智能分析”入口
- 为引用、对象卡片、事件卡片统一 UI
- 为只读工具结果补前端空态和错误态

### 6.3 数据与迁移任务

- 新增对象索引表迁移
- 新增事件流表迁移
- 制定索引回填策略
- 制定旧数据增量同步策略

### 6.4 测试任务

- 增加 assistant 意图与工具规划单测
- 增加对象索引和事件归一化单测
- 增加关键 API 集成测试
- 增加页面内智能入口浏览器冒烟

## 7. 验收标准

### 7.1 功能验收

- 高风险动作仍不可自动执行
- assistant 能稳定回答一期定义的高频问题
- assistant 能触发至少 4 个只读工具
- assistant 能基于对象上下文补充回答
- 页面内智能入口可带上下文提问

### 7.2 数据验收

- 对象索引数据可查、可追溯、可增量更新
- 事件流写入成功率可监控
- 时间线查询结果和原业务记录可对上

### 7.3 治理验收

- assistant 继承当前用户权限
- 工具调用有审计记录
- 模型失败、工具失败、超时有明确降级
- 结果引用可追溯到文档或业务数据源

### 7.4 测试验收

- `go test ./internal/assistant/...` 持续通过
- 新增对象与事件模块单测持续通过
- `npm run acceptance:smoke:core` 持续通过
- 页面内智能入口最小冒烟可通过

## 8. 风险与控制

### 8.1 数据口径不统一

风险：

- 不同模块对象 ID 和状态字段不一致，导致对象索引和时间线难以合并

控制措施：

- 先统一对象 ID 规则
- 为一期只选最关键字段做归一化

### 8.2 assistant 编排重构导致现有能力回退

风险：

- 重构 `service.go` 时容易把已收口的问答链路打散

控制措施：

- 先补足回归测试
- 分阶段切换，先兼容旧逻辑再替换

### 8.3 工具层权限穿透

风险：

- assistant 调工具时绕过原有用户权限

控制措施：

- 工具层必须强制接收当前用户上下文
- 禁止 assistant 直接查业务表

### 8.4 页面内智能入口扩散过快

风险：

- 过早在太多页面接入口，导致前端改造面过大

控制措施：

- 一期只做 2 到 3 个页面试点
- 优先选 deploy 和 alert 页面

## 9. 推荐实施顺序

建议实际执行顺序如下：

1. assistant 编排结构化改造
2. 只读工具接口抽象
3. 对象索引最小落地
4. 事件流与时间线骨架
5. 页面内智能入口试点
6. 回归测试与浏览器验收补齐

原因：

- 先把 assistant 主链路稳住，后续对象和事件能力才能顺滑接入
- 先有工具层，才能避免把对象和事件能力直接耦合进 `service.go`
- 页面入口放在最后，可避免前端过早承担不稳定后端接口

## 10. 建议的第一批实施任务

如果马上进入开发，建议从以下 8 项开始排期：

1. 定义 assistant 统一意图结构与决策输出结构
2. 为 assistant 增加页面上下文输入字段
3. 抽象工具注册与调用接口
4. 落地 `query_release_history` 和 `query_alert_events`
5. 设计并创建 `platform_object_index`
6. 设计并创建 `platform_event_stream`
7. 为 deploy 和 alert 接入页面内智能入口
8. 补一轮新的单测与浏览器冒烟

这 8 项完成后，平台就会从“助手功能点”进入“智能化底座 + 场景试点”的真实实施阶段。
