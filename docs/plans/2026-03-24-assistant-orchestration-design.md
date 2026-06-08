# assistant 决策编排设计

## 1. 文档目标

本文档用于定义 OPS Platform 中 assistant 的决策编排设计，说明 assistant 如何从“问一句答一句”的回复逻辑，演进到“具备意图识别、上下文组装、工具编排、规则判断、模型融合和结果回写”的统一智能编排层。

本文档重点回答以下问题：

- assistant 的主流程应该如何拆层
- 哪些环节必须先走规则，哪些环节适合引入模型
- assistant 如何消费对象模型、事件流和工具层
- assistant 如何支持只读查询、分析建议和受控执行

本文档承接以下设计：

- [`2026-03-24-intelligent-ops-platform-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-intelligent-ops-platform-design.md)
- [`2026-03-24-platform-ai-integration-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-platform-ai-integration-design.md)
- [`2026-03-24-unified-object-model-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-unified-object-model-design.md)
- [`2026-03-24-event-stream-and-timeline-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-event-stream-and-timeline-design.md)

## 2. 当前实现现状

当前 assistant 主链路位于：

- [`backend/internal/assistant/service.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/service.go)
- [`backend/internal/assistant/handler.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/handler.go)

现有流程大致为：

```text
接收消息
  -> classifyIntent()
  -> buildActions()
  -> buildCitations()
  -> runReadonlyTools()
  -> generateModelAnswer()
  -> fallbackAnswer()
```

当前优点：

- 已有统一入口 `/api/assistant/*`
- 已有会话、消息、引用和基础历史能力
- 已有知识检索、页面导航和只读查询骨架
- 已有 fallback 逻辑，模型不可用时不会完全失效

当前不足：

- 意图识别仍偏关键词驱动
- 工具编排是固定链路，不是标准规划器
- 缺少统一的执行计划结构
- 缺少对象上下文和事件上下文注入
- 缺少风险分级、确认态和回写态的统一状态机

## 3. 编排设计原则

### 3.1 规则优先，模型增强

assistant 的关键控制逻辑不能完全依赖模型。

必须由规则控制的环节：

- 权限判断
- 工具可用性判断
- 风险分级
- 执行确认
- 回写格式

适合模型增强的环节：

- 用户意图理解
- 自然语言摘要
- 多数据源结果归纳
- 候选建议生成

### 3.2 工具先于答案

assistant 不是“先答再想”，而应该是“先确定要用什么信息，再组织答案”。

主流程必须优先决定：

- 当前问题属于哪类场景
- 是否需要工具调用
- 是否需要引用静态知识
- 是否需要补充上下文

### 3.3 结果必须结构化

assistant 最终不能只返回纯文本，而应产出统一决策结构。

至少包括：

- `intent`
- `context`
- `citations`
- `actions`
- `resultCards`
- `riskLevel`
- `needConfirmation`
- `executionPlan`

## 4. 总体编排架构

建议将 assistant 主链路拆成 7 层。

```text
消息接入层
  -> 会话上下文层
    -> 意图识别层
      -> 上下文组装层
        -> 工具规划层
          -> 决策生成层
            -> 回写与审计层
```

### 4.1 消息接入层

负责：

- 接收用户输入
- 绑定用户身份
- 绑定会话
- 基础限流

现有落点：

- [`handler.go`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/assistant/handler.go)

### 4.2 会话上下文层

负责：

- 读取有限轮历史消息
- 读取当前页面上下文
- 读取会话元信息
- 控制上下文长度

未来建议新增输入：

- 当前页面路径
- 当前对象 ID
- 当前页面选中的记录 ID

### 4.3 意图识别层

负责：

- 问答类识别
- 页面导航类识别
- 只读查询类识别
- 分析类识别
- 执行类识别

建议输出统一意图结构：

```json
{
  "intent": "knowledge_qa|page_navigation|readonly_query|analysis|execution",
  "sub_intent": "deploy_history|archive_history|vuln_detail|alert_analysis",
  "confidence": 0.0,
  "need_tools": true
}
```

### 4.4 上下文组装层

负责组装 4 类上下文：

- `会话上下文`
- `对象上下文`
- `事件上下文`
- `知识上下文`

示例：

- 用户在部署记录页问“这次为什么失败”
- 页面上下文应自动带上 `deploy_record_id`
- assistant 再去查对象、事件流和相关文档

### 4.5 工具规划层

负责决定：

- 调哪些工具
- 按什么顺序调
- 哪些工具结果进入摘要
- 哪些工具结果只用于判断

建议输出统一工具计划：

```json
{
  "plan_id": "plan_xxx",
  "steps": [
    {"tool": "query_release_history", "purpose": "load_release_record"},
    {"tool": "query_alert_events", "purpose": "load_related_alerts"},
    {"tool": "query_timeline", "purpose": "load_recent_context"}
  ]
}
```

### 4.6 决策生成层

负责将工具结果、规则结果和知识引用融合成最终输出。

建议统一决策结构：

```json
{
  "summary": "结论",
  "intent": "analysis",
  "evidence": [],
  "citations": [],
  "actions": [],
  "resultCards": [],
  "riskLevel": "medium",
  "needConfirmation": false,
  "executionPlan": null
}
```

### 4.7 回写与审计层

负责：

- 将 assistant 回答写入会话
- 将分析结果写入事件流
- 将工具调用写入审计
- 将执行确认写入时间线

## 5. 意图模型设计

### 5.1 一级意图

assistant 建议统一到 5 个一级意图：

- `knowledge_qa`
- `page_navigation`
- `readonly_query`
- `analysis`
- `execution`

### 5.2 二级意图

二级意图用于绑定具体领域与场景：

```text
deploy_history_query
archive_history_query
aggregate_history_query
alert_event_query
vuln_query
release_failure_analysis
alert_root_cause_analysis
vuln_priority_analysis
release_execution
consul_sync_execution
```

### 5.3 意图识别策略

当前阶段建议采用“规则优先 + 模型补充”的双通道：

#### 第一阶段

- 关键词规则
- 页面上下文规则
- 路由词典
- 对象词典

#### 第二阶段

- 轻量模型做补充分流
- 输出置信度
- 低置信度时回退规则路径

## 6. 上下文组装设计

### 6.1 上下文来源

assistant 组装上下文时，优先级建议如下：

1. 当前页面上下文
2. 当前会话历史
3. 对象模型
4. 统一事件流
5. 文档知识库

### 6.2 页面上下文协议

前端未来建议在调用 assistant 时可附带：

```json
{
  "pagePath": "/deploy/history",
  "objectType": "deploy_record",
  "objectId": "deploy_record:cmdb:205",
  "selectedIds": ["deploy_record:cmdb:205"]
}
```

这样 assistant 不需要反复通过自然语言猜用户当前在看什么。

### 6.3 对象上下文

从统一对象模型读取：

- 对象基本信息
- 所属关系
- 关联资产
- 当前状态

### 6.4 事件上下文

从统一事件流和时间线读取：

- 最近事件
- 失败事件
- 同一窗口内关联事件
- 上一条相关分析结果

## 7. 工具规划设计

### 7.1 工具分类

建议 assistant 工具按职责分 4 类：

#### 导航工具

- `navigate_to_page`

#### 知识工具

- `search_documents`
- `search_manual_entries`

#### 查询工具

- `query_projects`
- `query_release_history`
- `query_alert_events`
- `query_vulnerabilities`
- `query_object_timeline`

#### 执行工具

- `trigger_release`
- `retry_build`
- `sync_consul_config`

### 7.2 规划策略

当前阶段建议采用“模板化工具计划”，而不是让模型自由生成任意工具链。

示例：

- `knowledge_qa`
  默认先知识检索，再考虑页面导航
- `readonly_query`
  默认先领域查询，再做摘要
- `analysis`
  默认先对象查询，再查事件流，再查相关知识
- `execution`
  默认先权限校验，再生成执行计划，再等待确认

### 7.3 工具结果标准

每个工具统一输出：

```json
{
  "toolName": "query_release_history",
  "success": true,
  "summary": "工具摘要",
  "cards": [],
  "rawData": {},
  "relatedObjects": [],
  "events": []
}
```

## 8. 规则与模型融合

### 8.1 融合原则

assistant 的最终输出应由以下三部分共同决定：

- 规则系统
- 工具结果
- 模型归纳

### 8.2 规则负责什么

- 意图兜底
- 工具可选范围
- 风险等级
- 确认开关
- 输出格式约束

### 8.3 模型负责什么

- 用户问题理解
- 多来源信息压缩
- 解释性回答
- 候选建议排序

### 8.4 失败回退

若模型不可用或结果为空，应允许回退到：

- 静态规则回答
- 文档摘要回答
- 工具结果直出回答

## 9. 分场景编排模板

### 9.1 知识问答

```text
识别知识问答
  -> 查页面路径
  -> 查知识文档
  -> 过滤引用
  -> 生成步骤型回答
```

### 9.2 只读查询

```text
识别领域查询
  -> 解析对象或过滤条件
  -> 调只读工具
  -> 生成结果卡片
  -> 补充建议页面入口
```

### 9.3 分析场景

```text
识别分析问题
  -> 读取对象
  -> 读取事件流/时间线
  -> 查相关知识
  -> 生成结论 + 依据 + 候选动作
```

### 9.4 执行场景

```text
识别执行请求
  -> 权限校验
  -> 风险分级
  -> 生成执行计划
  -> 用户确认
  -> 调执行工具
  -> 结果回写
```

## 10. 确认与状态机设计

### 10.1 状态建议

对执行类 assistant 请求建议引入状态机：

```text
received
planned
waiting_confirmation
approved
executing
completed
failed
cancelled
```

### 10.2 确认对象

建议新增逻辑确认对象：

```json
{
  "confirmationId": "confirm_xxx",
  "riskLevel": "high",
  "actionType": "trigger_release",
  "summary": "将触发某应用重新发布",
  "expiresAt": "..."
}
```

## 11. 回写设计

assistant 每次编排都建议回写三类记录：

### 11.1 会话回写

- 用户消息
- assistant 回答
- 引用
- 快捷动作
- 结果卡片

### 11.2 审计回写

- 识别到的意图
- 使用的工具
- 使用的对象上下文
- 风险等级
- 是否发生确认

### 11.3 事件流回写

当满足以下条件时，建议写入统一事件流：

- 生成分析结论
- 触发确认流程
- 执行受控动作

## 12. 与当前代码的演进路径

### 12.1 当前代码保持不动的部分

- `/api/assistant/*` 路由
- 会话与消息存储
- 基础知识检索
- fallback 回答机制

### 12.2 第一阶段改造

在现有 `service.go` 上逐步引入：

- 意图识别结果对象
- 工具计划对象
- 决策输出对象
- 审计回写接口

### 12.3 第二阶段改造

新增：

- `context_builder.go`
- `planner.go`
- `decision_engine.go`
- `audit_writer.go`

### 12.4 第三阶段改造

接入：

- 统一对象模型
- 统一事件流
- 页面上下文输入
- 执行确认状态机

## 13. 推荐文件结构

建议 assistant 最终演进到如下结构：

```text
backend/internal/assistant/
├── handler.go
├── service.go
├── intent.go
├── context_builder.go
├── planner.go
├── decision_engine.go
├── knowledge.go
├── tools/
├── audit/
└── confirmation/
```

## 14. 结论

assistant 的核心价值不在于“接了模型”，而在于它成为现有平台的统一决策编排层。

它必须做到：

- 能理解问题
- 能定位对象
- 能组装上下文
- 能调对工具
- 能给出有依据的结论
- 能把关键动作纳入确认、审计和回写闭环

当这套编排层稳定后，assistant 才能从“聊天入口”真正升级为“智能运维编排入口”。
