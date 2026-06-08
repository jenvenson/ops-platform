# 统一对象模型设计

## 1. 文档目标

本文档用于定义 OPS Platform 在智能化演进过程中的统一对象模型，解决以下问题：

- 现有平台各模块中的业务对象如何统一抽象
- 哪些对象属于核心资产对象，哪些属于事件对象，哪些属于关系对象
- assistant、分析引擎、审计系统和时间线系统如何基于统一模型工作
- 现有数据库表如何映射到统一对象模型

本文档是以下设计的基础支撑：

- [`2026-03-24-intelligent-ops-platform-design.md`](docs/plans/2026-03-24-intelligent-ops-platform-design.md)
- [`2026-03-24-platform-ai-integration-design.md`](docs/plans/2026-03-24-platform-ai-integration-design.md)

## 2. 设计原则

### 2.1 统一抽象，不强行重构

统一对象模型首先是“逻辑统一”，不是要求一次性重构现有所有数据库表。

当前阶段目标：

- 先统一对象语义
- 再统一对象标识
- 最后逐步统一数据存储和事件流

### 2.2 对象与事件分离

必须严格区分：

- `对象`
  平台中长期存在、可被引用、可被关联的实体
- `事件`
  某个时间点发生在对象上的动作、变化或结果

示例：

- 应用、主机、项目、漏洞、告警规则 是对象
- 部署成功、告警触发、漏洞修复、会话删除 是事件

### 2.3 主对象优先

对象模型不追求一开始覆盖所有细节，而是优先定义能支撑智能分析的主对象。

优先级：

1. 资产对象
2. 运行事件对象
3. 风险对象
4. 关系对象
5. 时间线对象

## 3. 统一对象模型总览

建议平台统一定义 5 大类对象。

### 3.1 资产对象

用于描述平台管理范围内的基础实体：

- 项目
- 环境
- 主机
- 应用
- Jenkins 流水线
- Consul 配置
- 安全资产

### 3.2 运行事件对象

用于描述资产对象上的运行行为：

- 部署事件
- 归档事件
- 聚合打包事件
- 构建事件
- 配置变更事件
- assistant 会话事件

### 3.3 风险对象

用于描述需要分析、处置或治理的风险实体：

- 告警规则
- 告警事件
- 漏洞
- 漏洞工单

### 3.4 关系对象

用于描述对象之间的关联：

- 项目与应用关系
- 环境与应用关系
- 应用与主机关系
- 告警与资产关系
- 漏洞与资产关系
- 发布与应用关系

### 3.5 时间线对象

用于描述某对象在时间维度上的变化轨迹。

时间线本身不是业务对象，而是统一视图：

- 项目时间线
- 应用时间线
- 主机时间线
- 告警时间线
- 漏洞时间线
- 会话时间线

## 4. 对象标识规范

### 4.1 逻辑对象 ID

建议为统一对象模型引入逻辑对象 ID，而不是直接暴露底层表主键。

格式建议：

```text
{object_type}:{source}:{id}
```

示例：

- `project:cmdb:12`
- `application:cmdb:37`
- `deploy_record:cmdb:205`
- `alert_event:alert:98`
- `vulnerability:security:501`
- `assistant_session:assistant:sess_20260324_xxx`

### 4.2 对象类型枚举

建议统一定义 `object_type` 枚举：

```text
project
environment
server
application
pipeline
consul_config
security_asset
deploy_record
archive_record
aggregate_task
aggregate_history
alert_rule
alert_event
vulnerability
vuln_ticket
assistant_session
assistant_message
user
menu
```

### 4.3 事件类型枚举

建议统一定义 `event_type`：

```text
created
updated
deleted
deployed
deploy_failed
archived
archive_failed
aggregate_started
aggregate_completed
alert_fired
alert_acked
alert_resolved
vuln_detected
vuln_fixed
ticket_created
ticket_closed
session_created
session_archived
session_deleted
assistant_answered
analysis_generated
action_confirmed
action_executed
```

## 5. 核心资产对象定义

### 5.1 Project

语义：

- 业务项目
- 平台内大多数资产归属的顶层容器

现有来源：

- [`cmdb.Project`](backend/internal/cmdb/models.go)

关键属性：

- `id`
- `name`
- `code`
- `description`
- `status`
- `created_at`
- `updated_at`

建议扩展：

- `owner_id`
- `owner_name`
- `department`
- `lifecycle_status`

### 5.2 Environment

语义：

- 应用运行环境，如 dev/test/prod

现有来源：

- [`cmdb.Environment`](backend/internal/cmdb/models.go)

关键属性：

- `id`
- `name`
- `type`
- `description`

### 5.3 Server

语义：

- 被纳入平台管理的主机节点

现有来源：

- [`cmdb.Server`](backend/internal/cmdb/models.go)

关键属性：

- `hostname`
- `ip`
- `os`
- `arch`
- `status`
- `agent_status`
- `last_heartbeat`
- `cpu_usage`
- `memory_usage`
- `disk_usage`

建议扩展：

- `importance`
- `owner`
- `cluster`
- `labels`

### 5.4 Application

语义：

- 业务应用或应用流水线实体

现有来源：

- [`cmdb.Application`](backend/internal/cmdb/models.go)

关键属性：

- `name`
- `project_id`
- `env_id`
- `code_repo`
- `deploy_path`
- `jenkins_job`
- `jenkins_archive_job`

### 5.5 SecurityAsset

语义：

- 安全扫描体系中的独立资产对象

现有来源：

- [`models.Asset`](backend/internal/models/security.go)
- [`models.SecurityAsset`](backend/internal/models/security.go)

说明：

- `SecurityAsset` 偏向扫描结果明细
- `Asset` 更适合作为统一安全资产对象

建议统一保留逻辑对象：

- `security_asset`

## 6. 核心风险对象定义

### 6.1 AlertRule

现有来源：

- [`alert.AlertRule`](backend/internal/alert/models.go)

关键属性：

- `grafana_uid`
- `name`
- `severity`
- `category`
- `expression`
- `enabled`

### 6.2 AlertEvent

现有来源：

- [`alert.AlertEvent`](backend/internal/alert/models.go)

关键属性：

- `rule_id`
- `rule_name`
- `severity`
- `category`
- `content`
- `source`
- `status`
- `fired_at`
- `acked_at`
- `resolved_at`

### 6.3 Vulnerability

现有来源：

- [`models.SecurityVulnerability`](backend/internal/models/security.go)

关键属性：

- `asset_id`
- `severity`
- `cvss_score`
- `cve_id`
- `title`
- `vuln_type`
- `solution`
- `status`
- `priority`

### 6.4 VulnTicket

现有来源：

- [`models.VulnTicket`](backend/internal/models/security.go)

关键属性：

- `vuln_id`
- `assignee`
- `status`
- `priority`
- `due_date`
- `resolved_at`

## 7. 核心运行事件对象定义

### 7.1 DeployRecord

现有来源：

- [`cmdb.DeployRecord`](backend/internal/cmdb/models.go)

关键属性：

- `app_id`
- `env_id`
- `project_code`
- `deploy_type`
- `jenkins_job`
- `jenkins_build_num`
- `status`
- `error_message`
- `start_time`
- `end_time`
- `triggered_by`

### 7.2 ArchiveRecord

现有来源：

- [`cmdb.ArchiveRecord`](backend/internal/cmdb/models.go)

关键属性：

- `app_id`
- `env_id`
- `project_code`
- `deploy_type`
- `status`
- `download_url`
- `jenkins_console_url`
- `operator`

### 7.3 AggregateTask / AggregateHistory

现有来源：

- [`models.AggregatePackageTask`](backend/internal/models/aggregate_package.go)
- [`models.AggregatedHistory`](backend/internal/models/aggregated_history.go)

逻辑定义：

- `aggregate_task`
- `aggregate_history`

### 7.4 AssistantSession / AssistantMessage

现有来源：

- [`models.AssistantSession`](backend/internal/models/assistant.go)
- [`models.AssistantMessage`](backend/internal/models/assistant.go)

关键属性：

- `session_id`
- `user_id`
- `scene`
- `status`
- `title`
- `summary`
- `intent`
- `content`
- `model_name`

说明：

- assistant 会话本身也是对象
- assistant 回答、工具调用、引用可被视为事件流的一部分

## 8. 关系模型设计

### 8.1 关系抽象

建议统一定义逻辑关系结构：

```text
subject_object_id
relation_type
target_object_id
source
metadata
created_at
```

### 8.2 关系类型建议

```text
belongs_to
deployed_to
runs_on
triggers
depends_on
causes
affects
related_to
generated_by
handled_by
references
```

### 8.3 关键关系示例

- `application -> belongs_to -> project`
- `application -> deployed_to -> environment`
- `application -> runs_on -> server`
- `deploy_record -> affects -> application`
- `archive_record -> affects -> application`
- `alert_event -> affects -> server`
- `vulnerability -> affects -> security_asset`
- `vuln_ticket -> handled_by -> user`
- `assistant_message -> references -> deploy_record`

## 9. 时间线模型设计

### 9.1 时间线统一结构

建议统一定义时间线条目：

```text
timeline_id
object_type
object_id
event_type
title
summary
severity
operator_id
operator_name
source
related_object_ids
created_at
metadata
```

### 9.2 时间线来源

时间线条目可来自：

- 部署记录
- 归档记录
- 聚合历史
- 告警事件
- 漏洞发现
- 工单状态变化
- assistant 分析记录

### 9.3 时间线用途

统一时间线将为以下能力提供底座：

- assistant 上下文组装
- 根因分析
- 影响范围分析
- 页面内“最近发生了什么”视图
- 审计回放

## 10. 与现有数据库表的映射建议

### 10.1 现有表直接映射

建议保留现有业务表不动，先做逻辑映射：

| 逻辑对象 | 当前来源 |
| --- | --- |
| `project` | `projects` |
| `environment` | `environments` |
| `server` | `servers` |
| `application` | `applications` |
| `deploy_record` | `deploy_records` |
| `archive_record` | `archive_records` |
| `aggregate_history` | `aggregated_histories` |
| `alert_rule` | `alert_rules` |
| `alert_event` | `alert_events` |
| `vulnerability` | `security_vulnerabilities` |
| `vuln_ticket` | `security_vuln_tickets` |
| `assistant_session` | `assistant_sessions` |
| `assistant_message` | `assistant_messages` |

### 10.2 建议新增逻辑层

在不动原表的前提下，新增三张统一逻辑表：

- `platform_object_index`
- `platform_event_stream`
- `platform_object_relations`

#### platform_object_index

作用：

- 保存统一逻辑对象 ID
- 建立对象与原表主键映射
- 为检索、时间线、assistant 提供统一入口

#### platform_event_stream

作用：

- 保存跨模块统一事件
- 支持 assistant、分析、审计、推荐共享

#### platform_object_relations

作用：

- 保存统一对象关系
- 支撑影响分析与关联查询

## 11. assistant 与对象模型的结合方式

assistant 在统一对象模型上主要消费三类能力：

### 11.1 对象定位

示例：

- “查看某项目最近发布记录”
- “某漏洞影响了哪些资产”
- “某告警关联哪台主机”

### 11.2 时间线组装

示例：

- “这次部署失败前后发生了什么”
- “这个应用最近一周有哪些异常”

### 11.3 关系推理

示例：

- “这条告警可能影响哪些应用”
- “这个漏洞和哪个项目关系最大”

## 12. 实施顺序建议

### 12.1 第一阶段

先定义统一对象字典，不改底层存储。

输出物：

- 对象类型枚举
- 事件类型枚举
- 关系类型枚举
- 现有表映射表

### 12.2 第二阶段

新增 `platform_object_index` 和 `platform_event_stream`。

### 12.3 第三阶段

让 assistant 和页面内分析入口改为基于统一对象模型查询。

### 12.4 第四阶段

再逐步推动跨模块分析、风险评估和受控执行。

## 13. 结论

统一对象模型是 OPS Platform 智能化建设的数据底座。

它的核心价值不在于“把所有表统一成一张表”，而在于：

- 让不同模块说同一种对象语言
- 让 assistant 能跨模块理解上下文
- 让事件、关系和时间线能形成闭环

只有对象模型统一后，后续的事件流、决策编排、智能分析和受控执行才具备可实施基础。
