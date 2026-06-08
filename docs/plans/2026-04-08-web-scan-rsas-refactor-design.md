# 参考绿盟思路的 Web 漏洞扫描重构设计

日期：2026-04-08

## 目标

在保留现有 Web 漏扫可运行能力的前提下，将当前“URL 发现 + Nuclei + 少量规则”的实现，重构为更接近成熟扫描产品的 `认证 -> 发现 -> 检测 -> 验证 -> 误报修正 -> 报告` 链路。

本设计明确以下边界：

- 现有 Web 漏扫必须重构，不能继续在旧链路上堆功能。
- 本次重构以“对齐产品能力层次”为目标，不追求一次性完全复制商业产品。
- 现有页面和接口需要保持可用，迁移采用“新链路先落、旧链路兼容”的方式推进。
- 数据底座沿用已启动的 Phase 1 扫描模型迁移，即 `run / target / evidence / occurrence`。

## 当前实现现状

当前 Web 漏扫主路径位于：

- `backend/internal/security/handler.go`
- `backend/internal/security/session.go`
- `backend/internal/security/web_discovery.go`
- `backend/internal/security/browser_discovery.go`
- `backend/internal/security/engine.go`

当前实现的主要特征：

1. 任务创建默认使用 `browser` 发现模式，但默认发现参数较浅：
   - `DiscoveryMaxDepth = 1`
   - `DiscoveryMaxURLs = 25`

2. 认证模型以请求头注入为主：
   - `cookie`
   - `bearer`
   - `basic`
   - `header`
   - `advanced`
   - 登录表单和登录接口能力最终仍会落回会话头拼装

3. 发现模型以入口 URL 为中心：
   - HTML 链接
   - script URL
   - JS 中提取到的链接
   - robots/sitemap
   - 浏览器发现依赖外部 helper

4. 检测模型以 `Nuclei -tags` 为主，规则层较薄：
   - SQL 注入
   - XSS
   - SSRF
   - CSRF
   - RCE
   - 信息泄露
   - 越权访问
   - 文件包含

5. 当前自定义规则有限，主要集中在：
   - 未授权敏感信息泄露
   - 疑似未授权访问

## 与绿盟思路的主要差距

结合绿盟 RSAS/WVSS 公开资料，可以将当前差距归纳为 6 类。

### 1. 认证能力差距

当前问题：

- 会话模型过于依赖 header 注入
- Cookie、CSRF、localStorage、sessionStorage 没有统一状态管理
- 动态签名、登录流程跳转、多阶段认证仍然脆弱
- 浏览器发现和 HTTP 扫描没有共享稳定会话

目标能力：

- 统一认证计划 `AuthPlan`
- 统一会话状态 `SessionState`
- 支持匿名、预录制登录、表单登录、接口登录、多头会话、动态签名、代理登录
- 支持登录态续期和失败重试

### 2. 攻击面发现差距

当前问题：

- 默认深度和 URL 数过低
- 对 SPA、接口型应用、动态渲染页面支持有限
- 表单、API、页面三类对象没有统一建模
- 浏览器 helper 不可用时会明显降级

目标能力：

- 匿名发现 + 登录后继续发现
- 页面、API、表单、鉴权入口统一建模
- 支持 SPA 路由、XHR/fetch、JS 动态链接提取
- 支持目录猜测、备份文件、OpenAPI/Swagger、robots/sitemap、多入口扩展

### 3. 检测能力差距

当前问题：

- 主要依赖模板扫描
- 自定义规则数量少，覆盖面有限
- 缺少登录后专项验证
- 缺少认证/授权类的强验证逻辑

目标能力：

- 模板检测层
- 规则检测层
- 登录后检测层
- 认证/授权专项验证层
- 弱口令/默认口令、敏感目录、鉴权绕过、业务接口暴露等专项能力

### 4. 验证与误报修正差距

当前问题：

- Web 结果缺少原生“验证成功/验证失败/误报/例外”流程
- 现在的 `verification` 更偏主机版本匹配场景
- URL 级、接口级、页面级的误报修正范围还没有形成

目标能力：

- Web finding 的独立验证状态
- URL / API / 页面维度误报修正
- 任务级、目标级、全局级例外范围
- 报告与统计口径严格区分已验证与待验证

### 5. 产品抽象差距

当前问题：

- 前台更像在选漏洞标签
- 用户需要理解内部技术词，如 `web-template`、`web-rule`
- 缺少扫描策略模型

目标能力：

- 以扫描策略而不是漏洞标签驱动任务创建
- 至少支持：
  - `快速扫描`
  - `标准扫描`
  - `深度扫描`
  - `登录后扫描`
- 内部检测引擎细节不直接暴露给用户

### 6. 证据回溯差距

当前问题：

- 页面/接口/表单没有稳定目标树
- 请求响应证据散落在旧字段中
- 运行级命中与稳定 finding 的对应关系仍较弱

目标能力：

- 运行级目标树
- 证据标准化落库
- occurrence 绑定 target 和 evidence
- 同一 finding 可回看每次 run 的实际命中证据

## 目标架构

Web 漏扫重构后建议统一为以下 6 层。

### 1. 任务定义层

任务只描述：

- 目标 URL / 目标组
- 扫描策略
- 认证方式
- 速率与边界
- 登录后扫描范围

不再直接暴露过多底层引擎参数。

### 2. 运行快照层

每次扫描生成独立 `run`，保存：

- 实际配置快照
- 目标快照
- 认证快照
- 风险摘要
- 执行阶段

### 3. 目标展开层

统一建模以下目标对象：

- `url`
- `page`
- `api`
- `form`
- `auth`

并通过 `parent_target_id` 形成目标层级。

### 4. 认证与会话层

统一认证输入为 `AuthPlan`：

- `anonymous`
- `preset-cookie`
- `basic`
- `bearer`
- `form-login`
- `api-login`
- `prerecorded`
- `dynamic-sign`
- `multi-step`

统一会话输出为 `SessionState`：

- CookieJar
- default headers
- localStorage/sessionStorage
- csrf token
- token extractors
- refresh strategy

### 5. 扫描执行层

统一执行流建议为：

```text
目标 URL
  -> 基础探测
  -> 匿名发现
  -> 登录态获取
  -> 登录后继续发现
  -> 页面/API/表单归一
  -> 模板扫描
  -> 规则扫描
  -> 认证/授权专项验证
  -> 证据落库
  -> finding 归并
```

### 6. 结果与修正层

前台默认只展示：

- `正式结果`
- `待验证`
- `资产信息`

并提供：

- 验证状态
- 误报修正
- 例外范围
- 导出报告

## 分阶段重构方案

### Phase 0：冻结旧链路，明确兼容边界

目标：

- 不再继续往旧 Web 扫描链路叠加复杂新特性
- 明确旧链路只做兼容运行

工作项：

- 保持旧接口兼容
- 收口前台术语
- 补齐 Web 扫描的现状说明和能力边界

交付：

- 路线文档
- 前台能力边界提示

### Phase 1：接入新数据模型

目标：

- 将 Web 扫描结果开始双写到新表

工作项：

- `run` 增加 Web 阶段信息
- 新增 `url/page/api/form/auth` 目标写入
- 新增 `browser-discovery/http-request/http-response/nuclei-result/rule-match/auth-login` 证据写入
- 新增 Web finding occurrence 写入

交付：

- Web 扫描 dual-write helper
- 运行级目标树与证据链

验收：

- 一次 Web 扫描能在新表中查到完整目标树、证据和 occurrence
- 旧页面与报表不受影响

### Phase 2：认证引擎重构

目标：

- 建立稳定的登录后扫描基础能力

工作项：

- 新建 `web_auth_plan.go`
- 新建 `web_session_state.go`
- 拆分旧 `BuildWebSession`
- 支持登录流程动作定义
- 支持 token/csrf/cookie/storage 抽取
- 支持浏览器态与 HTTP 态共享会话

交付：

- `AuthPlan` 模型
- `SessionState` 模型
- 登录态执行器

验收：

- 登录一次后，发现、规则、模板三类扫描都能复用同一会话
- 浏览器发现和 HTTP 请求不再各自维护独立认证逻辑

### Phase 3：发现引擎重构

目标：

- 形成稳定的 Web 攻击面发现体系

工作项：

- 浏览器发现器内建化，不再只依赖临时 helper
- 匿名发现与登录后发现拆分阶段
- 补齐 SPA 路由、XHR/fetch、form action、OpenAPI/Swagger 解析
- 加目录猜测、备份文件、常见后台入口发现
- 对页面/API/表单做统一归一

交付：

- `web_discovery_runner.go`
- `web_target_graph.go`
- 扩展 discovery evidence

验收：

- 扫描结果能区分入口页、页面、接口、表单
- 登录前后发现结果有独立统计

### Phase 4：检测与验证层重构

目标：

- 形成模板、规则、专项验证并行的检测结构

工作项：

- 保留 `Nuclei` 作为模板层
- 抽出 `WebRuleEngine`
- 增加认证/授权专项验证器
- 增加敏感信息暴露、未授权访问、默认口令、弱口令、越权、登录后页面专项规则
- 将“已验证风险”与“待验证线索”严格分层

交付：

- `web_rule_engine.go`
- `web_authz_verifier.go`
- 规则证据写入

验收：

- Web 结果中出现可被明确标记为“验证成功”的风险
- 模板命中、规则命中、验证命中的证据可追溯

### Phase 5：产品交互与报告重构

目标：

- 前台交互更接近成熟扫描产品

工作项：

- 创建任务页改为“扫描策略”驱动
- 详情页改为：
  - 攻击面
  - 正式结果
  - 待验证
  - 资产信息
- 支持 URL / API / 页面粒度的误报修正
- 报告增加：
  - 匿名覆盖范围
  - 登录后覆盖范围
  - 页面/API/表单统计
  - 已验证结果与待验证结果分节展示

交付：

- 新任务创建交互
- 新详情页与导出模板

验收：

- 用户不需要理解 `web-template` 和 `web-rule`
- 报告能看出本次扫描覆盖到了哪些页面、接口和登录后区域

### Phase 6：切换读路径并收尾

目标：

- 逐步从旧读模型迁移到新读模型

工作项：

- 先切详情页的攻击面读取
- 再切 occurrence 查询
- 最后收口旧 `security_web_discoveries` 的职责

交付：

- 新查询接口
- 旧接口兼容层

验收：

- 新旧结果口径一致
- 页面性能和查询可控

## 推荐文件拆分

建议新增或重构以下模块。

### 后端

- `backend/internal/security/web_auth_plan.go`
- `backend/internal/security/web_session_state.go`
- `backend/internal/security/web_auth_executor.go`
- `backend/internal/security/web_discovery_runner.go`
- `backend/internal/security/web_target_graph.go`
- `backend/internal/security/web_rule_engine.go`
- `backend/internal/security/web_authz_verifier.go`
- `backend/internal/security/web_phase1.go`

### 前端

- `frontend/src/pages/security/TaskCreate.tsx`
- `frontend/src/pages/security/TaskDetail.tsx`
- `frontend/src/pages/security/VulnerabilityList.tsx`
- `frontend/src/api/security.ts`

### 迁移

- 扩展 `security_scan_runs`
- Web dual-write 需要使用：
  - `security_scan_targets`
  - `security_scan_evidences`
  - `security_scan_finding_occurrences`

## 优先顺序

建议按以下顺序推进：

1. `Phase 1`
2. `Phase 2`
3. `Phase 3`
4. `Phase 4`
5. `Phase 5`
6. `Phase 6`

原因：

- Web 最大瓶颈不是规则不够，而是没有稳定的会话和目标树
- 在认证与发现不稳的前提下继续堆模板收益很低
- 先补底座，后补检测，最后再改交互，风险最低

## 风险与约束

1. 浏览器能力引入后，运行成本会显著上升
2. 登录后扫描需要更严格的速率限制和范围控制
3. 动态签名和复杂会话能力需要额外测试环境验证
4. 新旧双写期间，必须持续校验新表与旧表的结果一致性

## 结论

要按绿盟思路重构 Web 漏扫，关键不是“再多加几个 Web 规则”，而是先把：

- 认证
- 会话
- 发现
- 目标树
- 证据链

这几层重做出来。

只有这样，后续再追加 Web 专项检测、登录后扫描、验证与误报修正，才会变成可维护的产品能力，而不是继续把旧 `engine.go` 变成更大的耦合点。
