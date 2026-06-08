# Web 漏洞扫描重构实施清单

日期：2026-04-08

## 目标

将《参考绿盟思路的 Web 漏洞扫描重构设计》拆成可执行的实施清单，便于逐阶段推进、验收和交接。

关联文档：

- `docs/plans/2026-04-08-web-scan-rsas-refactor-design.md`
- `docs/plans/2026-04-08-rsas-inspired-scan-refactor-roadmap.md`
- `docs/plans/2026-04-08-security-scan-phase1-data-model.md`

## Phase 0：冻结旧链路

### 后端

- [ ] 在文档中明确旧 Web 扫描的能力边界
- [ ] 停止继续向旧 `engine.go` 添加复杂认证逻辑
- [ ] 标记旧 `BuildWebSession` 为兼容层

### 前端

- [ ] 页面说明统一为“正式结果 / 待验证 / 资产信息”
- [ ] 创建任务页增加“旧扫描链路能力边界”提示

### 验收

- [ ] 旧任务仍可正常创建与执行
- [ ] 无新增功能继续直接堆进旧链路

## Phase 1：Web dual-write

### 数据层

- [ ] 确认 Phase 1 基础迁移已在目标环境执行
- [ ] 为 Web 扫描补充 target kinds：
  - [ ] `url`
  - [ ] `page`
  - [ ] `api`
  - [ ] `form`
  - [ ] `auth`

### 后端

- [ ] 新增 `web_phase1.go`
- [ ] 扫描入口创建 `run` 阶段快照
- [ ] 发现入口写入 `security_scan_targets`
- [ ] 请求/响应/规则/模板结果写入 `security_scan_evidences`
- [ ] 结果命中写入 `security_scan_finding_occurrences`

### 验收

- [ ] 一次 Web 任务能查到完整 targets
- [ ] 能查到 HTTP 请求/响应或 discovery 证据
- [ ] occurrence 可回指 legacy vulnerability

## Phase 2：认证引擎

### 模型

- [ ] 新建 `AuthPlan`
- [ ] 新建 `SessionState`
- [ ] 定义登录动作模型：
  - [ ] request
  - [ ] extract
  - [ ] assert
  - [ ] store

### 后端

- [ ] 新建 `web_auth_plan.go`
- [ ] 新建 `web_session_state.go`
- [ ] 新建 `web_auth_executor.go`
- [ ] 兼容旧 `auth_mode/auth_flow`
- [ ] 支持 CookieJar
- [ ] 支持 CSRF/token 抽取
- [ ] 支持 localStorage/sessionStorage
- [ ] 支持 token 刷新

### 前端

- [ ] 创建任务页改为认证方案驱动
- [ ] 隐藏低层 `auth_flow` 概念

### 验收

- [ ] 登录后扫描能稳定复用会话
- [ ] 浏览器发现与 HTTP 扫描使用同一份会话状态

## Phase 3：发现引擎

### 后端

- [ ] 新建 `web_discovery_runner.go`
- [ ] 新建 `web_target_graph.go`
- [ ] 拆分匿名发现和登录后发现
- [ ] 支持：
  - [ ] HTML link
  - [ ] script src
  - [ ] JS URL extraction
  - [ ] XHR/fetch
  - [ ] form action
  - [ ] OpenAPI/Swagger
  - [ ] robots/sitemap
  - [ ] 目录猜测
  - [ ] 备份文件发现

### 目标建模

- [ ] 页面对象归一
- [ ] API 对象归一
- [ ] 表单对象归一
- [ ] 鉴权入口对象归一

### 验收

- [ ] 扫描详情可区分页面/API/表单
- [ ] 登录前和登录后新增发现有独立统计

## Phase 4：检测与验证

### 模板层

- [ ] 保留 Nuclei 扫描入口
- [ ] 模板扫描结果标准化写证据

### 规则层

- [ ] 新建 `web_rule_engine.go`
- [ ] 扩充规则：
  - [ ] 未授权访问
  - [ ] 鉴权绕过
  - [ ] 敏感信息泄露
  - [ ] 默认口令
  - [ ] 弱口令
  - [ ] 登录后页面暴露
  - [ ] 接口权限边界错误

### 验证层

- [ ] 新建 `web_authz_verifier.go`
- [ ] 为 Web finding 增加验证状态流转
- [ ] 为 URL/API/页面结果增加误报修正

### 验收

- [ ] Web 结果中能区分模板命中、规则命中、验证成功
- [ ] 已验证结果单独进入主风险口径

## Phase 5：前台与报告

### 前端

- [ ] 创建任务页切换为扫描策略模型：
  - [ ] 快速
  - [ ] 标准
  - [ ] 深度
  - [ ] 登录后
- [ ] 详情页新增攻击面视图
- [ ] 支持按页面/API/表单筛选
- [ ] 支持误报修正与例外范围

### 报告

- [ ] HTML 导出新增匿名/登录后覆盖说明
- [ ] JSON 导出新增目标树与攻击面统计
- [ ] CSV 导出补页面/API/表单分类

### 验收

- [ ] 用户看不到 `web-template`、`web-rule` 等内部术语
- [ ] 报告能解释覆盖范围、正式结果、待验证和资产信息

## Phase 6：切换读路径

### 后端

- [x] 新增基于新模型的 Web 详情查询接口
- [x] 新增目标树查询接口
- [x] 新增 evidence 查询接口

### 前端

- [x] 任务详情页优先读取新模型
- [x] 漏洞详情抽屉显示 occurrence + evidence
- [x] 攻击面页签改读新 targets

### 收尾

- [x] 评估 `security_web_discoveries` 的兼容保留期限
- [x] 清理旧 Web 扫描执行分支
- [x] 统一文档与测试入口

## 建议首批实施顺序

建议先做以下 6 项：

- [ ] 补 Web dual-write helper
- [ ] 补 `url/page/api/form/auth` targets
- [ ] 定义 `AuthPlan` 与 `SessionState`
- [ ] 拆分匿名发现与登录后发现阶段
- [ ] 落一版 `WebRuleEngine`
- [ ] 详情页增加攻击面目标树视图

## 每阶段最小验收命令

- [x] `cd .worktrees/phase1-framework/backend && GOCACHE=$(pwd)/.gocache go test ./internal/security/...`
- [x] `cd .worktrees/phase1-framework/frontend && npm run build`
- [x] 后端代码有变更时先执行 `cd .worktrees/phase1-framework && bash deploy/dev.sh backend`
- [x] 前端联调需要刷新时执行 `cd .worktrees/phase1-framework && bash deploy/dev.sh frontend`
- [x] `cd .worktrees/phase1-framework/frontend && npm run acceptance:smoke:security-web`
- [x] 核对 `GET /tasks/:id`、`/targets`、`/evidences`、`/occurrences` 返回完整

## 交接要求

每完成一个阶段，至少补：

- [ ] 设计文档更新
- [ ] 验证记录更新
- [ ] 开发环境回归记录
- [ ] 未完成项和风险说明

### 统一入口说明

- Web 扫描回归入口统一为 `frontend/package.json` 里的 `npm run acceptance:smoke:security-web`
- 脚本会自动完成：后台登录、生成 `auth_flow`、创建登录后 Web 扫描任务、轮询完成、校验 `latest_run/targets/evidences/occurrences/vulnerabilities`
- 开发环境是源码挂载到 `ops-backend-dev`/`ops-frontend-dev` 容器；后端代码变更后如果不重启 `ops-backend-dev`，会继续跑旧进程，导致新接口和新逻辑不会生效

## 2026-04-09 交接记录

### 当前能力边界

- Web 扫描已收敛为“只做登录后 Web 扫描”，匿名 Web 扫描已禁用。
- 认证链路、Phase 1 dual-write、会话复用已接通，任务可以稳定完成并写入 `run/target/evidence/occurrence`。
- 独立 `browser-helper` 容器已接入同一 Docker 网络，正式环境标准扫描默认走 browser discovery，不再依赖宿主机 helper。
- 当前正式环境 Web 扫描能力边界不是“能不能发现浏览器态目标”，而是“为了控制时长，部分低价值目标只做 `web-rule`，不再跑 full Nuclei”。

### 已验证结果

- 开发环境登录后 Web 扫描已回归通过，已确认 `security_scan_runs`、`security_scan_targets`、`security_scan_evidences`、`security_scan_finding_occurrences` 正常落库。
- 正式地址 `http://your-app/web_01` 已用账号 `web_01` 做真实登录后扫描验证。
- 任务 `18` 已验证“复用首轮登录态”有效，修复了逐目标重复登录导致的 `auth/oauth2/token` 超时失败。
- 任务 `19` 在当前网络结构下已完成，但由于 browser helper 不可达，只发现 `1` 个入口，说明链路可用、浏览器态发现未生效。
- 任务 `21` 已验证任务结果提示增强生效：当 browser discovery 回退为 HTTP discovery 时，最终 `message` 会显式提示“因 browser helper 不可达回退为 HTTP 发现”，便于直接从任务结果判断覆盖面边界。
- 后续已补独立 `browser-helper` 容器并恢复容器内可达性；正式地址任务 `22` 已验证 browser discovery 恢复生效，发现 `8` 个入口并完成 `8` 个目标扫描。
- 任务 `25` 与任务 `26` 已验证“定向分类扫描”端到端生效：
  - `task 25` 仅选择 `information-disclosure`，结果只命中“未授权敏感信息泄露”，未再混出“疑似未授权访问”。
  - `task 26` 仅选择 `broken-access`，结果只命中“疑似未授权访问”，未再混出“未授权敏感信息泄露”。
- 扫描任务详情页已补“漏洞标题”列，位置在“严重程度”后；同时为漏洞表增加固定宽度和横向滚动，避免标题列在窄宽度下被挤压隐藏。
- 任务 `27` 已验证超长响应保护生效：正式地址 `information-disclosure` 扫描正常完成，后端日志未再出现 `Data too long for column 'response'`。
- 任务 `30` 已验证第一轮正式环境预算优化生效：
  - `page`、`agreement/get-term` 这类长文本或低价值目标已降级为 `rule-only`
  - 最终结果为 `8` 个入口、`8` 个目标、`4` 个低价值目标仅执行规则检测
  - 总耗时约 `189s`
- 任务 `31` 已验证第二轮预算优化生效：
  - `base/open/plugin/installed/list`、`base/user/verify/type/*` 也已降级为 `rule-only`
  - 最终结果为 `8` 个入口、`8` 个目标、`6` 个低价值目标仅执行规则检测
  - 总耗时约 `99s`
- 任务 `32` 已验证第三轮预算优化生效：
  - `base/common/get-func`、`base/custom/get` 保留 full Nuclei，但单目标命令超时已从 `45s` 收紧到 `20s`
  - 后端日志明确出现 `timed out after 20s`
  - 最终结果为 `8` 个入口、`8` 个目标、`6` 个低价值目标仅执行规则检测
  - 总耗时约 `60s`
- 任务 `33` 已验证“深度扫描”能力生效：
  - 同样是正式地址 `8` 个入口、`8` 个目标
  - `agreement/get`、`get-term`、`plugin/installed/list`、`verify/type` 不再走 `rule-only`
- 任务 `37` 已在删除 `security_web_discoveries` 表后重新验证：
  - 目标 `http://your-app/web_01/`
  - 任务完成后 `latest_run` 摘要正常返回
  - `targets=10`、`evidences=18`、`occurrences=9`、`vulnerabilities=9`
  - 说明详情页和新读接口已不依赖旧 discoveries 表
- 新增统一回归入口 `npm run acceptance:smoke:security-web`，脚本位置为 `frontend/scripts/security-web-regression.cjs`
- 任务 `38` 已通过统一回归脚本验证：
  - 自动完成后台登录、`auth_flow` 生成、任务创建、状态轮询与结果校验
  - 最终结果为 `targets=10`、`evidences=20`、`occurrences=11`、`vulnerabilities=11`
  - 说明“删旧表 + 重启 dev 容器 + 新读接口校验”已形成固定入口
  - 最终结果为 `2 high / 7 medium / 0 low`
- 任务 `35` 已验证“定向分类扫描 + 全类别”仍会受到标准预算约束：
  - 最终结果为 `8` 个入口、`8` 个目标、`2 high / 8 medium / 0 low`
  - 其中 `6` 个低价值目标仅执行规则检测
- 页面入口已从 `标准/深度/定向分类` 收敛为 `标准扫描 / 专项扫描` 两档：
  - `标准扫描` 保持当前保守预算，适合首轮排查
  - `专项扫描` 默认全选全部类别，且自动使用更深预算
- 任务 `36` 已验证新的“专项扫描”端到端生效：
  - 正式地址 `http://your-app/web_01`
  - 默认全选全部 `9` 类漏洞类别
  - 最终结果为 `8` 个入口、`8` 个目标、`2 high / 8 medium / 0 low`
  - 最终消息不再出现“低价值目标仅执行规则检测”，说明专项扫描已使用深预算而非标准预算

### 当前实现约束

- Web 发现阶段已增加低价值资源过滤，默认跳过 `logo/background/avatar/watermark` 等资源型 URL。
- Web 验证阶段已增加目标优先级与预算控制，默认只扫描前 `8` 个高优先级 discovered targets，避免正式环境任务时间失控。
- 当前正式环境标准扫描的验证预算已分层：
  - `page`、`agreement/get-term`、`plugin installed list`、`verify/type` 走 `rule-only`
  - `base/common/get-func`、`base/custom/get` 保留 full Nuclei，但单目标预算为 `20s`
- 当前页面仅保留两类 Web 扫描入口：
  - `标准扫描` = 推荐模板集 + 保守预算
  - `专项扫描` = 类别收敛 + 深预算
- 漏洞持久化阶段已增加 `payload/request/response` 安全截断，避免单条超长响应导致整条漏洞保存失败。
- 以上优化只在 browser discovery 或 HTTP discovery 已拿到目标的前提下生效，不能替代浏览器态发现能力本身。

### 未完成项与风险

- browser helper 已迁移为独立容器并恢复到 Docker 网络内可达；当前主要风险已从“helper 不可达”转为“预算继续压缩可能伤到高价值 API 的覆盖率”。
- 漏洞库里的 `request/response` 目前通过保存前截断规避超长内容问题，后续如需保留更完整原文，可再评估是否把旧表字段升级为 `MEDIUMTEXT/LONGTEXT` 并进一步弱化对旧表原文的依赖。

### 后续建议

- 先按当前正式环境预算继续使用，不建议在这一轮里继续压缩时长；如果后续需要进一步压缩，应单开专项重新评估哪些高价值 API 仍值得保留 full Nuclei。
- 如需提升可解释性，可在任务详情页补充“哪些目标只做了规则检测”的展示，避免把预算策略误判成漏扫。
- 如需进一步提升结果完整性，可评估把旧 `security_vulnerabilities.request/response` 升级到更大文本类型，但短期内不影响扫描结果保存和展示。
