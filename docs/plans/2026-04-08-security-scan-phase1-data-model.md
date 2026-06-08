# 安全扫描 Phase 1 数据模型迁移设计

日期：2026-04-08

## 目标

为主机漏洞扫描和网站漏洞扫描的后续重构，先落一版稳定的数据模型基础，使平台从“任务直接写结果”升级为“任务定义 + 运行快照 + 目标单元 + 证据 + occurrence”的结构。

本阶段不要求一次性替换所有现有读写路径，但要求：

- 新表先建起来
- 新旧模型可并行
- 现有页面和接口继续可用
- 后续主机/Web 扫描可逐步迁移到新表

## 当前模型问题

当前安全扫描模型主要有：

- `security_scan_tasks`
- `security_scan_runs`
- `security_assets`
- `security_web_discoveries`
- `security_vulnerabilities`

问题主要在：

1. `security_scan_tasks` 仍然过度承担运行态语义
2. `security_scan_runs` 只有最小统计字段，缺少配置快照和阶段信息
3. 主机与 Web 的目标单元没有统一模型
4. 原始证据散落在 `security_vulnerabilities.request/response/payload` 等字段中
5. “某次运行中的一次具体命中”没有独立对象

这会直接限制：

- 历史回溯
- 误报修正
- 运行对比
- 登录后扫描
- 攻击面图沉淀

## Phase 1 范围

Phase 1 只做基础模型，不改业务口径：

1. 扩展 `security_scan_runs`
2. 新增 `security_scan_targets`
3. 新增 `security_scan_evidences`
4. 新增 `security_scan_finding_occurrences`

暂不在本阶段新建 `security_scan_findings`，原因是：

- 当前 `security_vulnerabilities` 仍承担稳定 finding 的职责
- 先用 `occurrence -> legacy vulnerability` 的桥接方式过渡更稳

后续在 Phase 2 / Phase 3 再决定是否将 `security_vulnerabilities` 彻底升级为 `security_scan_findings`

## 设计原则

### 1. 运行与结果分离

- `task` 是长期定义
- `run` 是一次执行
- `occurrence` 是一次运行中的一次具体命中

### 2. 目标与证据解耦

- 目标单元独立建模
- 证据独立存储
- 一个 finding occurrence 可以关联多个证据

### 3. 兼容现有读模型

现有页面、报表和列表仍从以下表读取：

- `security_assets`
- `security_web_discoveries`
- `security_vulnerabilities`

新表先作为底座与过渡层，等执行器改造完成后再逐步切主读路径

## 目标模型

### 一、`security_scan_runs` 扩展

当前表已存在，建议新增以下字段：

- `run_no`
  - 同一任务下的运行序号
- `phase`
  - 当前阶段
  - 例如 `prepare / discovery / verification / authenticated / reporting / completed`
- `config_snapshot`
  - 本次实际配置快照
- `target_snapshot`
  - 本次目标展开快照
- `summary_snapshot`
  - 风险统计和运行摘要快照
- `started_by`
  - 与 `triggered_by` 统一收口时使用
- `cancelled_at`
  - 取消时间

本阶段不强制删旧字段，只增不减。

### 二、`security_scan_targets`

统一表示本次运行中的目标单元。

目标种类建议包括：

- `ip`
- `service`
- `url`
- `page`
- `api`
- `form`
- `auth`

关键字段：

- `id`
- `run_id`
- `task_id`
- `parent_target_id`
- `target_kind`
- `normalized_target`
- `host`
- `port`
- `scheme`
- `path`
- `service_name`
- `product_name`
- `version`
- `status`
- `discovery_source`
- `started_at`
- `completed_at`
- `metadata_json`

说明：

- `parent_target_id` 用于表示目标层级
  - 例如 `url -> page -> form`
  - 或 `ip -> service`
- `normalized_target` 用于做稳定展示和去重

### 三、`security_scan_evidences`

统一表示扫描产生的原始证据。

证据类型建议包括：

- `nmap-service`
- `http-request`
- `http-response`
- `banner`
- `nuclei-result`
- `rule-match`
- `browser-discovery`
- `auth-login`
- `config-check`
- `screenshot`

关键字段：

- `id`
- `run_id`
- `task_id`
- `target_id`
- `evidence_type`
- `source_engine`
- `digest`
- `request_excerpt`
- `response_excerpt`
- `payload_excerpt`
- `metadata_json`
- `raw_json`
- `storage_ref`

说明：

- `digest` 用于做证据去重和引用
- `storage_ref` 预留给后续对象存储
- `metadata_json` 用于扩展头信息、状态码、模板信息等

### 四、`security_scan_finding_occurrences`

统一表示某次运行中的一次具体命中。

关键字段：

- `id`
- `run_id`
- `task_id`
- `target_id`
- `legacy_vulnerability_id`
- `finding_key`
- `finding_family`
- `finding_source`
- `severity`
- `confidence`
- `match_mode`
- `primary_cve_id`
- `vuln_db_id`
- `title`
- `status`
- `verification_status`
- `evidence_count`
- `first_seen_at`
- `last_seen_at`
- `metadata_json`

说明：

- `legacy_vulnerability_id` 指向当前 `security_vulnerabilities.id`
- `finding_key` 用于表达跨运行的稳定识别键
- `status` 表示处置状态
- `verification_status` 表示验证状态

## 新旧表映射关系

### 当前读模型保留

- `security_assets`
  - 继续作为资产和端口发现的现有展示表
- `security_web_discoveries`
  - 继续作为 Web 发现页签数据来源
- `security_vulnerabilities`
  - 继续作为正式结果/待验证/资产信息的现有展示表

### 新表作用

- `security_scan_targets`
  - 作为统一目标树底座
- `security_scan_evidences`
  - 作为原始证据底座
- `security_scan_finding_occurrences`
  - 作为运行级命中底座

### 过渡策略

执行器在后续重构时按以下顺序迁移：

1. 先写新表
2. 再同步写旧表
3. 页面继续读旧表
4. 稳定后再逐步让页面切到新表

## 主机扫描如何接新模型

主机扫描迁移后建议这样落：

- `security_scan_targets`
  - `ip`
  - `service`
- `security_scan_evidences`
  - `nmap-service`
  - `banner`
  - `nuclei-result`
  - `config-check`
- `security_scan_finding_occurrences`
  - 每次版本线索命中一条
  - 每次模板验证命中一条
  - 每次配置核查命中一条

## 网站扫描如何接新模型

网站扫描迁移后建议这样落：

- `security_scan_targets`
  - `url`
  - `page`
  - `api`
  - `form`
  - `auth`
- `security_scan_evidences`
  - `browser-discovery`
  - `http-request`
  - `http-response`
  - `rule-match`
  - `auth-login`
- `security_scan_finding_occurrences`
  - 页面级命中
  - 接口级命中
  - 表单级命中

## 索引建议

### `security_scan_targets`

- `idx_scan_targets_run_id`
- `idx_scan_targets_task_id`
- `idx_scan_targets_parent_target_id`
- `idx_scan_targets_kind_status`
- `idx_scan_targets_normalized_target`

### `security_scan_evidences`

- `idx_scan_evidences_run_id`
- `idx_scan_evidences_task_id`
- `idx_scan_evidences_target_id`
- `idx_scan_evidences_type`
- `idx_scan_evidences_digest`

### `security_scan_finding_occurrences`

- `idx_scan_occurrences_run_id`
- `idx_scan_occurrences_task_id`
- `idx_scan_occurrences_target_id`
- `idx_scan_occurrences_legacy_vuln_id`
- `idx_scan_occurrences_source`
- `idx_scan_occurrences_status`
- `idx_scan_occurrences_verification_status`
- `idx_scan_occurrences_primary_cve_id`

## 迁移实施顺序

### Step 1

扩展 `security_scan_runs`

### Step 2

新增 `security_scan_targets`

### Step 3

新增 `security_scan_evidences`

### Step 4

新增 `security_scan_finding_occurrences`

### Step 5

在执行器里补“新旧双写”

## 风险与边界

1. 不建议在 Phase 1 直接废弃 `security_vulnerabilities`
   - 页面和导出仍强依赖它

2. 不建议在 Phase 1 直接切换查询口径
   - 先完成双写和回归

3. 不建议把 Web 与主机的所有高级字段一次性都写死到表结构
   - 扩展性优先，非关键字段先放 `metadata_json`

## 下一步建议

在本设计基础上，建议直接进入下面两个动作：

1. 生成正式 SQL 迁移
2. 从主机扫描执行器开始做第一批双写落库

配套 DDL 草案见：

- [2026-04-08-security-scan-phase1-ddl-draft.sql](docs/plans/2026-04-08-security-scan-phase1-ddl-draft.sql)
- 正式迁移草案见：

- [2026-04-09-security-scan-phase1-foundation.sql](deploy/migrations/2026-04-09-security-scan-phase1-foundation.sql)
