# 项目代码结构

本文档用于说明 `phase1-framework` worktree 的当前代码结构，便于快速定位后端、前端、部署和历史归档内容。

## 1. 根目录结构

```text
phase1-framework/
├── backend/                  # Go 后端服务
├── frontend/                 # React + Vite 前端应用
├── deploy/                   # Docker / Nginx / 部署脚本
├── docs/                     # 项目文档、设计与计划
├── migrations/               # 历史补充迁移
├── _archive/                 # 已归档的历史目录
├── CLAUDE.md                 # 项目协作说明
└── README.md                 # 项目总体说明
```

## 2. 后端结构

后端位于 `backend/`，采用 Go Modules 组织，入口、业务模块、配置和迁移分别拆分。

```text
backend/
├── cmd/
│   ├── server/
│   │   ├── main.go           # 主服务入口
│   │   ├── backend_service.log
│   │   └── ops-server
│   └── jenkins_test/
│       └── main.go           # Jenkins 相关测试入口
├── internal/
│   ├── agent/                # Agent 相关能力
│   ├── alert/                # 告警管理
│   ├── api/                  # API 路由/处理
│   ├── auth/                 # 认证与授权
│   ├── assistant/            # 运维小助手能力
│   ├── cicd/                 # 流水线/发布相关逻辑
│   ├── cmdb/                 # CMDB 领域逻辑
│   ├── consul/               # Consul 集成
│   ├── database/             # 数据库初始化与连接
│   ├── models/               # 数据模型定义
│   ├── monitor/              # 监控相关逻辑
│   ├── security/             # 安全扫描/漏洞相关逻辑
│   ├── server/               # 服务启动与装配
│   └── tasks/                # 后台任务
├── pkg/                      # 可复用公共包
├── configs/                  # 配置文件
├── migrations/               # 主数据库迁移
├── scripts/                  # 初始化与辅助脚本
├── deploy/                   # 后端相关部署遗留内容
├── go.mod
├── go.sum
├── start.sh
└── add_aggregate_menu.sql
```

说明：

- `backend/cmd/server/main.go` 是后端主程序入口。
- `backend/internal/` 是主要业务代码所在位置，按领域拆分较清晰。
- `backend/cmd/server/backend_service.log` 和 `backend/cmd/server/ops-server` 看起来是运行产物或调试残留，后续可视情况清理或加入忽略规则。

## 3. 前端结构

前端位于 `frontend/`，基于 React + Vite，按页面、组件、接口封装分层。

```text
frontend/
├── src/
│   ├── api/                  # 前端接口请求封装
│   ├── assets/               # 前端资源
│   ├── components/           # 通用组件
│   ├── data/                 # 静态数据或配置
│   ├── layouts/              # 布局组件
│   └── pages/
│       ├── admin/            # 系统管理
│       ├── alarm/            # 告警页面
│       ├── cmdb/             # CMDB 页面
│       ├── consul/           # Consul 页面
│       ├── deploy/           # 部署相关页面
│       ├── jenkins/          # Jenkins 页面
│       ├── monitor/          # 监控页面
│       └── security/         # 安全页面
├── public/                   # 静态资源
├── assets/                   # 项目级资源目录
├── deploy/                   # 前端相关部署遗留内容
├── index.html
├── package.json
├── pnpm-lock.yaml
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
└── CLAUDE.md
```

说明：

- `frontend/src/pages/` 基本对应平台功能模块。
- `frontend/src/api/` 与后端接口契约关系较紧，排查联调问题时应与 `backend/internal/api/` 一起看。

## 4. 部署与运行结构

部署相关内容位于根目录 `deploy/`。

```text
deploy/
├── docker-compose.dev.yml    # 开发环境编排
├── docker-compose.yml        # 生产环境编排
├── Dockerfile.backend        # 后端镜像构建
├── Dockerfile.frontend       # 前端镜像构建
├── nginx.conf                # Nginx 配置
├── nginx.prod.conf           # 生产 Nginx 配置
├── frontend-nginx.conf       # 前端 Nginx 配置
├── deploy-init.sh            # 初始化脚本
├── deploy-prod.sh            # 生产部署脚本
├── deploy-update.sh          # 更新脚本
├── dev.sh                    # 本地开发辅助脚本
├── DEPLOY.md                 # 部署说明
├── migrations/              # 部署相关迁移
├── nginx/                   # Nginx 资源目录
└── server
```

说明：

- 本地开发和生产部署的统一入口都在 `deploy/`。
- `docker-compose.dev.yml` 用于开发环境，`docker-compose.yml` 用于生产环境。

## 5. 文档与历史目录

```text
docs/
├── README.md
├── deploy.md
├── design.md
├── testing.md
├── user_manual.md
├── user_manual.html
├── completed-fixes.md
├── pending-fixes.md
├── pending_fix_resolution.md
├── history_fixes.md
├── aggregate-package-feature-design.md
├── aggregate-package-feature-requirements.md
├── consul_pipeline_config_guide.md
├── archive/
└── plans/

migrations/
├── 000007_add_aggregated_history_table.sql
└── 000008_add_aggregated_history_menu_item.sql

_archive/
├── duplicate-legacy/
└── root-legacy/
```

说明：

- `docs/` 保存设计、部署、测试和修复记录。
- 根目录 `migrations/` 属于历史补充迁移，新增迁移更适合放入 `backend/migrations/`。
- `_archive/` 不参与当前主工程运行，主要用于保留旧内容。

## 6. 快速定位建议

- 查后端入口：`backend/cmd/server/main.go`
- 查后端业务：`backend/internal/*`
- 查前端页面：`frontend/src/pages/*`
- 查前后端接口联调：`frontend/src/api/*` 与 `backend/internal/api/*`
- 查开发环境：`deploy/docker-compose.dev.yml`
- 查生产部署：`deploy/docker-compose.yml`、`deploy/deploy-prod.sh`
- 查数据库迁移：`backend/migrations/`、`migrations/`

## 7. 机器人代码入口

当前项目里的“运维小助手”分为前端聊天组件和后端聊天接口两部分。

### 7.1 前端入口

前端挂载链路如下：

```text
frontend/src/main.tsx
  -> frontend/src/App.tsx
    -> frontend/src/components/MainLayout.tsx
      -> frontend/src/components/AIChatbot.tsx
```

关键位置：

- `frontend/src/main.tsx`：前端应用启动入口
- `frontend/src/App.tsx`：路由总入口
- `frontend/src/components/MainLayout.tsx`：主布局中通过 `<AIChatbot />` 挂载机器人浮窗
- `frontend/src/components/AIChatbot.tsx`：机器人主组件，负责创建会话、发送消息、渲染聊天窗口

前端调用接口：

- 查询会话：`GET /api/assistant/sessions`
- 创建会话：`POST /api/assistant/sessions`
- 查询历史：`GET /api/assistant/sessions/:sessionId/messages`
- 发送消息：`POST /api/assistant/messages`

补充说明：

- `AIChatbot.tsx` 当前统一请求同站点 `/api/assistant/*`
- 开发环境下该请求会通过 Vite 代理转发到后端服务

### 7.2 后端入口

后端接入链路如下：

```text
backend/cmd/server/main.go
  -> backend/internal/server/server.go
    -> assistant.RegisterRoutes(r, cfg)
    -> assistant.Init()
```

关键位置：

- `backend/cmd/server/main.go`：后端服务进程入口
- `backend/internal/server/server.go`：统一注册系统各模块路由，assistant 模块在这里接入
- `backend/internal/assistant/routes.go`：声明 `/api/assistant` 路由
- `backend/internal/assistant/handler.go`：处理会话列表、创建会话、读取历史和发送消息

当前后端 assistant 模块行为：

- `Init()` 会自动迁移 assistant 会话、消息和引用表
- `CreateSession()` 会绑定当前登录用户并复用最近活跃会话
- `SendMessage()` 会做限流、超时、RAG 检索和只读工具调用
- 当前主链路优先走 `Ollama + qwen2.5:1.5b`，不可用时回退到受控 fallback

### 7.3 当前结论

当前机器人主链路只有一套：

- 前端 `AIChatbot.tsx`
- 后端 `backend/internal/assistant/`

旧 `/api/chatbot/*` 接口和对应后端实现已从活跃代码中清理，当前唯一保留的聊天主链路是 `/api/assistant/*`。
