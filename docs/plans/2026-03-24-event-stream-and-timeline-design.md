# 统一事件流与时间线设计

## 1. 文档目标

本文档用于定义 OPS Platform 的统一事件流与时间线模型，解决以下问题：

- 平台中不同模块产生的事件如何统一接入
- 哪些事件需要进入统一事件流
- assistant、分析引擎、审计系统如何消费事件流
- 时间线如何从事件流中生成与索引

本文档承接以下设计文档：

- [`2026-03-24-intelligent-ops-platform-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-intelligent-ops-platform-design.md)
- [`2026-03-24-unified-object-model-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-unified-object-model-design.md)
- [`2026-03-24-platform-ai-integration-design.md`](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/docs/plans/2026-03-24-platform-ai-integration-design.md)

## 2. 设计原则

### 2.1 事件统一，不改业务事实

统一事件流的目标不是替代原始业务表，而是在原始业务表之外增加一层统一抽象。

原始业务表继续负责：

- 业务事实落库
- 页面查询
- 模块内部逻辑

统一事件流负责：

- 跨模块归一化
- assistant 上下文组装
- 时间线生成
- 分析与审计

### 2.2 写入优先于实时计算

对智能化系统来说，最重要的是“事件被稳定记录”，而不是“所有分析都现算”。

因此优先级是：

1. 业务动作完成后写事件
2. 事件写入后生成时间线
3. assistant 和分析系统消费事件流

### 2.3 事件必须可回放

统一事件流必须具备回放能力，保证：

- assistant 能回溯上下文
- 页面能查看对象时间线
- 运维人员能审计关键动作

## 3. 统一事件流总览

建议平台增加一层统一事件流：

```text
业务模块写入原表
  -> Event Mapper
    -> platform_event_stream
      -> platform_timeline_entries
        -> Assistant / 页面分析 / 审计 / 统计
```

事件源包括：

- CMDB
- Deploy
- Jenkins
- Consul
- Alert
- Security
- Assistant
- Admin / Auth

## 4. 事件标准模型

### 4.1 核心结构

建议统一事件记录至少包含以下字段：

| 字段 | 说明 |
| --- | --- |
| `event_id` | 全局唯一事件 ID |
| `event_type` | 事件类型 |
| `event_category` | 事件分类 |
| `source_system` | 来源模块 |
| `source_table` | 来源表 |
| `source_id` | 来源主键 |
| `object_type` | 主对象类型 |
| `object_id` | 主对象逻辑 ID |
| `title` | 事件标题 |
| `summary` | 事件摘要 |
| `status` | 事件状态 |
| `severity` | 风险或重要级别 |
| `operator_id` | 操作人 |
| `operator_name` | 操作人名称 |
| `trigger_mode` | user/system/schedule/assistant |
| `started_at` | 开始时间 |
| `finished_at` | 结束时间 |
| `created_at` | 事件生成时间 |
| `metadata_json` | 补充上下文 |

### 4.2 事件分类建议

建议统一定义 `event_category`：

```text
asset
deploy
archive
aggregate
config
alert
security
assistant
auth
admin
execution
analysis
```

### 4.3 事件状态建议

建议统一状态字段：

```text
pending
running
success
failed
cancelled
resolved
closed
archived
unknown
```

## 5. 事件来源映射

### 5.1 CMDB 模块

原始对象：

- `projects`
- `environments`
- `servers`
- `applications`

建议写入事件：

- `project_created`
- `project_updated`
- `server_registered`
- `server_offline`
- `application_created`
- `application_updated`

### 5.2 Deploy 模块

原始对象：

- `deploy_records`
- `archive_records`
- `aggregate_package_tasks`
- `aggregated_histories`

建议写入事件：

- `deployed`
- `deploy_failed`
- `archived`
- `archive_failed`
- `aggregate_started`
- `aggregate_completed`
- `aggregate_failed`

### 5.3 Alert 模块

原始对象：

- `alert_events`
- `alert_event_logs`
- `alert_rules`

建议写入事件：

- `alert_fired`
- `alert_acked`
- `alert_resolved`
- `alert_closed`
- `alert_rule_created`
- `alert_rule_updated`

### 5.4 Security 模块

原始对象：

- `security_scan_tasks`
- `security_vulnerabilities`
- `security_vuln_tickets`
- `security_ticket_history`

建议写入事件：

- `scan_started`
- `scan_completed`
- `vuln_detected`
- `vuln_fixed`
- `ticket_created`
- `ticket_assigned`
- `ticket_closed`

### 5.5 Assistant 模块

原始对象：

- `assistant_sessions`
- `assistant_messages`
- `assistant_citations`

建议写入事件：

- `session_created`
- `session_archived`
- `session_deleted`
- `assistant_answered`
- `analysis_generated`
- `action_confirmed`

assistant 的分析结果进入事件流后，后续可被时间线与页面分析复用。

## 6. platform_event_stream 设计

### 6.1 表结构建议

建议新增统一事件流表：

```text
platform_event_stream
```

字段建议：

```text
id
event_id
event_type
event_category
source_system
source_table
source_id
object_type
object_id
title
summary
status
severity
operator_id
operator_name
trigger_mode
started_at
finished_at
created_at
metadata_json
raw_payload_json
```

### 6.2 索引建议

建议建立以下索引：

- `idx_object(object_type, object_id, created_at desc)`
- `idx_event_type(event_type, created_at desc)`
- `idx_category(event_category, created_at desc)`
- `idx_source(source_system, source_id)`
- `idx_operator(operator_id, created_at desc)`

### 6.3 写入策略

当前阶段建议采用“业务动作成功后同步写入 + 失败容错”策略：

- 主业务先落原始表
- 成功后尝试写入 `platform_event_stream`
- 写事件失败不回滚主业务，但必须记录错误日志

后续可演进为：

- outbox 模式
- 异步事件总线

## 7. platform_timeline_entries 设计

### 7.1 时间线定位

时间线不是独立事实源，而是事件流的对象视图索引。

建议增加时间线索引表：

```text
platform_timeline_entries
```

### 7.2 表结构建议

字段建议：

```text
id
timeline_id
object_type
object_id
event_id
event_type
title
summary
severity
operator_id
operator_name
source_system
related_object_ids_json
created_at
metadata_json
```

### 7.3 生成规则

每条事件流记录可生成 1 到 N 条时间线条目：

- 主对象一条
- 关联对象若干条

例如：

一次部署失败事件：

- 写入应用时间线
- 写入项目时间线
- 写入环境时间线

一次漏洞发现事件：

- 写入漏洞时间线
- 写入安全资产时间线
- 若已能映射到应用，也写入应用时间线

### 7.4 时间线用途

统一时间线支撑以下能力：

- assistant 回答“最近发生了什么”
- 页面展示“对象近期变更”
- 智能分析回溯上下文
- 审计查看关键动作链路

## 8. 关系回写设计

事件流不仅要写主对象，还要记录对象之间的关联。

建议在 `metadata_json` 或单独关系表中保存：

- `related_object_ids`
- `caused_by_event_id`
- `parent_event_id`

典型关系示例：

- `deploy_record -> application`
- `deploy_record -> environment`
- `alert_event -> server`
- `vulnerability -> security_asset`
- `assistant_message -> alert_event`

这些关系会决定时间线扩散范围和 assistant 的上下文组装精度。

## 9. assistant 如何消费事件流

assistant 消费事件流主要有 3 种方式。

### 9.1 最近事件查询

示例：

- “最近有哪些部署失败”
- “最近有哪些高危漏洞”
- “这个应用最近发生过什么”

### 9.2 时间线组装

示例：

- “这次部署失败前后发生了什么”
- “这个漏洞发现之后处理到哪一步了”

### 9.3 多模块关联分析

示例：

- “这条告警可能和哪个发布有关”
- “这个项目最近的不稳定和哪些变更有关”

## 10. 事件流与时间线生成顺序

建议按以下顺序实施：

### 10.1 第一阶段：统一事件定义

输出：

- 事件类型枚举
- 事件分类枚举
- 统一事件字段协议
- 各模块映射表

### 10.2 第二阶段：新增事件流表

输出：

- `platform_event_stream`
- 业务模块最小写入适配器

### 10.3 第三阶段：新增时间线索引表

输出：

- `platform_timeline_entries`
- 从事件流到时间线的生成器

### 10.4 第四阶段：assistant 接入事件流

输出：

- 基于事件流的只读查询工具
- 基于时间线的上下文组装能力

## 11. 与现有模块的最小集成建议

如果只做一版可落地最小改造，建议先接这 5 类事件：

1. `deploy_record`
2. `archive_record`
3. `alert_event`
4. `security_vulnerability`
5. `assistant_message`

原因：

- 这些事件价值最高
- 与智能分析关系最直接
- 现有平台已有稳定表结构

## 12. 审计与回放设计

统一事件流天然也是智能系统的审计底座。

至少应支持：

- 按对象回放事件
- 按用户回放动作
- 按 assistant 会话回放引用与分析
- 按时间窗口查看关键变更

建议保留：

- `raw_payload_json`
- `metadata_json`

以便后续追查：

- assistant 为什么给出这个建议
- 当时用了哪些数据和上下文
- 哪个模块写入了这条事件

## 13. 风险与约束

### 13.1 不能一次性覆盖所有模块

统一事件流必须分阶段推进，否则会拖慢主线业务开发。

### 13.2 事件要避免重复写入

需要定义幂等键，例如：

```text
{source_system}:{source_table}:{source_id}:{event_type}
```

### 13.3 事件摘要不能替代原始事实

`summary` 只是智能消费入口，真正审计仍需回到原始业务表和 `raw_payload_json`。

## 14. 结论

统一事件流与时间线是 OPS Platform 智能化落地的关键中间层。

对象模型解决“平台里有什么”，事件流解决“发生了什么”，时间线解决“这些事在上下文里如何串起来”。

只有这三层打通以后，assistant、页面内智能分析、风险判断和受控执行才有稳定的数据基础。
