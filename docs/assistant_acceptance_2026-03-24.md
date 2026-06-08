# 运维小助手验收记录

## 日期
2026-03-24

## 范围
- assistant 会话管理问答收口
- “归档打包”与“归档历史”问答准确性
- “删除会话”真实操作说明

## 验收方式
- 后端单测：`go test ./internal/assistant/...`
- 前端构建：`npm run build`
- 真实接口验收：在 `ops-backend-dev` 容器内调用 `/api/assistant/messages`
- 浏览器核心冒烟：`npm run acceptance:smoke:core`

## 已验证项

### 1. 如何归档打包
- 问法：`如何归档打包`
- 结果：通过
- 验证点：
  - 主回答返回 `/deploy/archive`
  - 引用命中 `user_manual / 归档应用`
  - 快捷操作为“打开归档打包”

### 2. 如何查看归档历史
- 问法：`如何查看归档历史`
- 结果：通过
- 验证点：
  - 主回答返回 `/deploy/archived`
  - 引用仅保留 `user_manual / 查看归档历史`
  - 快捷操作仅保留“打开归档历史”

### 3. 如何删除会话
- 问法：`如何删除会话`
- 结果：通过
- 验证点：
  - 主回答返回删除会话的真实前端步骤
  - 引用命中 `user_manual / 删除会话`
  - 不再返回无关页面导航或误导性快捷操作

## 本轮知识源修正
- `docs/user_manual.md` 新增“9.4 会话管理”
- `docs/user_manual.md` 新增“3.3 归档历史 (/deploy/archived)”
- `docs/user_manual.html` 同步修正
- `frontend/src/data/user_manual.json` 同步修正

## 环境说明
- assistant 运行结果受 `ops-backend-dev` 当前进程影响
- 本轮多次验证前均重启了 `ops-backend-dev`
- 若问答结果与源码不一致，优先确认容器是否已加载最新代码

## 2026-03-26 浏览器冒烟修复补充

### 问题现象
- Playwright 在当前 macOS 受限执行环境下启动浏览器时，系统 Chrome 会触发 `SIGABRT`
- 旧版 smoke 脚本在 Chrome 启动失败后会自动回退到 bundled Chromium，进一步触发 `SIGTRAP`
- 运维小助手相关断言过度依赖固定可见文案，容易被折叠引用、摘要波动和快捷操作文案调整误伤

### 脚本调整
- `frontend/scripts/ui-acceptance-smoke.cjs` 增加 `SMOKE_BROWSER`、`SMOKE_CHROME_EXECUTABLE`、`SMOKE_ALLOW_BUNDLED_CHROMIUM`
- macOS 默认优先使用系统 Chrome，不再默认回退到 bundled Chromium
- 运维小助手知识问答校验改为优先校验稳定信号：
  - 快捷操作按钮
  - 引用路径 `docs/user_manual.md`
  - 引用内容节点文本
- 页面内快捷提问按钮同步对齐当前页面真实文案，避免历史按钮名导致误报

### 2026-03-26 验证结果
- 语法检查：`node -c frontend/scripts/ui-acceptance-smoke.cjs`
- 核心冒烟：`npm run acceptance:smoke:core`
- 页面冒烟：`npm run acceptance:smoke:pages`
- 全量冒烟：`npm run acceptance:smoke`
- 结果：以上 3 组浏览器冒烟均已通过

## 当前遗留风险
- 运维小助手知识问答摘要仍可能随检索排序和模型输出发生轻微波动，因此 smoke 继续以“引用命中 + 正确快捷操作 + 正确页面行为”为主，不再强依赖固定摘要句式
- 当前浏览器冒烟通过依赖本机正常 GUI Chrome 启动环境；若在无 GUI 或更严格沙箱环境运行，仍需按环境变量显式调整浏览器启动策略
