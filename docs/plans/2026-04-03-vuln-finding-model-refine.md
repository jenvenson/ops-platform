# 漏洞扫描结果模型收敛方案

日期：2026-04-03

## 目标

在保留现有扫描引擎主链路的前提下，先把结果模型收敛清楚，解决以下问题：

- Web 规则结果、模板结果、CVE 结果混在一起
- 主机版本匹配结果和服务模板结果没有清晰分层
- 扫描结果与漏洞库只有“字段拷贝”，没有稳定关联
- 前端展示只能靠 `vuln_type` / `scan_method` 猜结果类型

本方案只覆盖 3 件事：

1. 数据库字段设计
2. 接口返回结构
3. 页面展示改法

不在本阶段解决：

- `scan_task` / `scan_run` 拆分
- 历史快照与差异对比
- 权限模型重构

## 当前实现问题

现有 `security_vulnerabilities` 主要有这些来源：

- `scanner = nuclei`
  - Web 模板结果
  - 主机服务标签模板结果
- `scanner = vuln-matcher`
  - 主机版本匹配结果
- `scanner = nuclei` 但 `template_id = web-rule-*`
  - Web 规则合成结果

这些来源的置信度和语义不同，但现在没有显式字段区分，只能靠：

- `scanner`
- `template_id`
- `scan_method`
- `cve_id`
- `vuln_type`

前端只能二次推断，长期不可维护。

## 一、数据库字段设计

### 1. 保留现有主表

继续使用当前表：

- `security_vulnerabilities`

在它上面补语义字段，而不是现在就拆成新表。

### 2. 为 `security_vulnerabilities` 新增字段

建议新增以下字段：

#### `finding_source`

结果来源，明确这条结果是怎么来的。

推荐枚举：

- `web-template`
- `web-rule`
- `host-template`
- `host-version-match`
- `asset-inventory`

说明：

- `web-template`：Web 站点上的 Nuclei 模板命中
- `web-rule`：未授权访问、敏感信息等平台内置规则
- `host-template`：主机服务标签模板命中
- `host-version-match`：基于服务指纹/CPE/版本的本地漏洞库匹配
- `asset-inventory`：服务识别或资产枚举型结果，不是正式漏洞

#### `confidence`

结果置信度。

推荐枚举：

- `high`
- `medium`
- `low`

默认赋值策略：

- `host-version-match`
  - 有明确 `CPE + version range`：`high`
  - 只有产品/版本模糊命中：`medium`
- `web-template`
  - 有 `CVE` 或高危模板：`high`
  - 其他模板：`medium`
- `web-rule`
  - 默认 `medium`
- `asset-inventory`
  - 默认 `low`

#### `primary_cve_id`

主关联 CVE。

说明：

- 当前 `cve_id` 仍保留，兼容逗号分隔的多个 CVE
- `primary_cve_id` 用于界面展示、知识库跳转和稳定过滤

#### `vuln_db_id`

主漏洞库引用。

说明：

- 指向 `vulnerability_database.id`
- 仅在能确定主关联知识库记录时写入
- Web 模板无 CVE 的结果可为空

#### `match_mode`

匹配模式，主要服务主机漏洞。

推荐枚举：

- `exact`
- `version-range`
- `fuzzy-product`
- `rule`
- `template`
- `inventory`

说明：

- 让前端不必再解析 `matched_on`
- 也方便后端按精度统计

#### `finding_family`

统一的大类，用于页面分组。

推荐枚举：

- `vulnerability`
- `inventory`

说明：

- 先只分两类，保持简单
- `资产识别` 统一归到 `inventory`

### 3. 新增多对多关联表

建议增加：

- `security_vulnerability_links`

字段：

- `id`
- `vulnerability_id`
- `vuln_db_id`
- `link_type`
- `created_at`

其中 `link_type` 推荐枚举：

- `primary`
- `related`
- `variant`

作用：

- 支持一条扫描结果关联多个 CVE
- 兼容 `Nuclei` 一条结果带多个 CVE 的场景

### 4. 现有字段保留但职责收口

- `scanner`
  - 保留原始引擎名，如 `nuclei`、`vuln-matcher`
- `template_id`
  - 保留模板标识
- `scan_method`
  - 逐步弱化，不再承担分类职责
- `risk_category`
  - 继续作为接口输出字段
  - 但后续由 `finding_source/finding_family` 计算得出

## 二、接口返回结构

### 1. 列表接口统一补语义字段

以下接口都补充相同结构：

- `GET /api/security/vulnerabilities`
- `GET /api/security/tasks/:id/vulnerabilities`
- `GET /api/security/vulnerabilities/:id`

新增返回字段：

- `finding_source`
- `finding_family`
- `confidence`
- `primary_cve_id`
- `vuln_db_id`
- `match_mode`
- `knowledge`
- `display_group`

### 2. `knowledge` 子对象

用于前端直接展示与漏洞库关联的信息。

示例：

```json
{
  "primary_cve_id": "CVE-2024-12345",
  "vuln_db_id": 18273,
  "knowledge": {
    "title": "Example RCE",
    "severity": "high",
    "cvss_score": 8.8,
    "cnvd_id": "CNVD-2024-12345",
    "cnnvd_id": "CNNVD-202401-0001",
    "has_reference": true
  }
}
```

说明：

- 没有关联时返回 `null`
- 前端不再自己拼 `CNVD/CNNVD/CVSS`

### 3. `display_group`

后端直接给页面分组值，减少 UI 推断。

推荐值：

- `网站漏洞`
- `主机漏洞`
- `资产识别`

映射规则：

- `finding_family = inventory` -> `资产识别`
- `finding_source in (web-template, web-rule)` -> `网站漏洞`
- `finding_source in (host-template, host-version-match)` -> `主机漏洞`

### 4. 新增筛选参数

漏洞列表和任务详情列表支持：

- `finding_source`
- `finding_family`
- `confidence`
- `has_knowledge`
- `match_mode`

建议保留现有：

- `risk_category`
- `severity`
- `status`

### 5. 兼容策略

第一阶段兼容旧前端：

- 原字段不删
- 新字段只新增
- 老页面继续可用

## 三、页面展示改法

### 1. 扫描任务详情页

文件：

- `frontend/src/pages/security/TaskDetail.tsx`

改法：

- 顶部统计拆成：
  - `真实漏洞`
  - `资产识别`
  - `高置信度`
  - `待确认`
- 主标签页按 `display_group` 分为：
  - `网站漏洞`
  - `主机漏洞`
  - `资产识别`
- 每条记录新增两个标签：
  - 来源标签：`模板命中` / `规则命中` / `版本匹配`
  - 置信度标签：`高` / `中` / `低`

详情抽屉里新增“知识库关联”区块：

- 主 CVE
- 漏洞库标题
- CVSS
- CNVD/CNNVD
- 匹配模式

### 2. 漏洞管理页

文件：

- `frontend/src/pages/security/VulnerabilityList.tsx`

改法：

- 顶部筛选补 3 个：
  - `结果来源`
  - `置信度`
  - `是否已关联漏洞库`
- 默认视图只显示：
  - `finding_family = vulnerability`
  - `confidence != low`
- 将 `资产识别` 收到次级视图，不和真实漏洞默认混排

表格列建议新增：

- `来源`
- `置信度`
- `主 CVE`
- `漏洞库`

其中“漏洞库”列展示：

- 已关联：`CVE / CNVD`
- 未关联：`未关联`

### 3. 资产页

文件：

- `frontend/src/pages/security/AssetList.tsx`

改法：

- 资产详情中的漏洞列表默认只展示：
  - `finding_family = vulnerability`
- 增加开关：
  - `显示资产识别结果`

这样资产页默认聚焦风险，不把版本枚举型结果混进来。

## 四、后端写入规则

### 1. Web 模板结果

写入时：

- `finding_source = web-template`
- `finding_family = vulnerability`
- `match_mode = template`
- 有 `CVE` 时：
  - 设置 `primary_cve_id`
  - 查 `vulnerability_database`
  - 命中后写 `vuln_db_id`

### 2. Web 规则结果

写入时：

- `finding_source = web-rule`
- `finding_family = vulnerability`
- `match_mode = rule`
- `confidence = medium`
- `vuln_db_id = null`

### 3. 主机版本匹配结果

写入时：

- `finding_source = host-version-match`
- `finding_family = vulnerability`
- `scanner = vuln-matcher`
- 有明确版本范围命中：
  - `match_mode = version-range`
  - `confidence = high`
- 只有模糊产品命中：
  - `match_mode = fuzzy-product`
  - `confidence = medium`

### 4. 主机模板结果

写入时：

- `finding_source = host-template`
- `finding_family = vulnerability`
- `match_mode = template`

### 5. 资产识别结果

写入时：

- `finding_source = asset-inventory`
- `finding_family = inventory`
- `match_mode = inventory`
- `confidence = low`

## 五、迁移顺序

### 第一阶段：只加字段

1. 给 `security_vulnerabilities` 增加新字段
2. 后端查询接口返回新字段
3. 前端开始消费新字段，但保留旧兜底逻辑

目标：

- 不影响已有任务创建和扫描
- 先把语义统一起来

### 第二阶段：补写入逻辑

1. `SaveScanResult()` 写 `finding_source/confidence/...`
2. `saveVersionMatchResults()` 写 `match_mode/vuln_db_id`
3. 多 CVE 结果写入关联表

目标：

- 新扫描结果具备稳定语义

### 第三阶段：清理旧推断逻辑

1. 前端移除对 `scan_method/vuln_type` 的大量猜测
2. 后端 `risk_category` 改为基于新字段统一计算
3. 再考虑补 `run` 层与历史快照

## 六、最小可实施版本

如果只做最小集，我建议这次先落这 6 项：

1. `security_vulnerabilities` 新增：
   - `finding_source`
   - `finding_family`
   - `confidence`
   - `primary_cve_id`
   - `vuln_db_id`
   - `match_mode`
2. `GET /api/security/tasks/:id/vulnerabilities` 返回这些字段
3. `GET /api/security/vulnerabilities` 支持 `finding_family` 和 `confidence`
4. `SaveScanResult()` 给 Web / 主机模板结果写 `finding_source`
5. `saveVersionMatchResults()` 给版本匹配结果写 `finding_source=host-version-match`
6. 前端两个页面只用新字段做来源和置信度展示

这样能先把“结果到底是什么”讲清楚，再继续做更大的模型重构。
