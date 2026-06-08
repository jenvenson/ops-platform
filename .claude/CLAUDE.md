# CLAUDE.md

## Purpose
- 默认先理解现有代码和部署方式，再做修改。
- 优先最小必要改动，避免顺手重构。
- 修改后必须验证；无法验证时明确说明缺口。

## Repo Facts
- 当前主仓库根目录是 `phase1-framework/`，不是外层容器目录。
- 主工程目录：`backend/`、`frontend/`、`deploy/`、`docs/`
- `_archive/` 不参与主运行链路。
- 后端入口：`backend/cmd/server/main.go`
- 前端 API 客户端：`frontend/src/api/client.ts`

## Workflow
1. 先定位入口、调用链和配置。
2. 优先阅读主干文件，不从归档目录开始。
3. 做最小必要修改。
4. 修改后执行直接相关的验证。
5. 回复时先给结论，再给证据和风险。

## Safety Rules
- 不要执行 `git reset --hard`、`git checkout --`。
- 不要覆盖用户已有改动，除非明确要求。
- 不要直接改线上数据库结构。
- 涉及线上操作时，先确认目标机器、目录和影响面。

## Project Notes
- 开发环境默认使用宿主机 Ollama。
- 线上环境 Ollama 使用 Docker 部署。
- `deploy/ollama/` 是线上部署模板，不是开发入口。
- 小助手主链路是 `/api/assistant/*`。
- `docker-compose restart` 不会重建容器。

## Validation
- 后端改动优先检查路由、日志、配置是否生效。
- 前端改动优先检查 API 结构和页面入口。
- 涉及 assistant 改动时，至少核对模型配置和实际落库模型名。

---

## 代码结构分析 (Code Structure)

### 1. 整体目录结构

```
phase1-framework/
├── backend/              # Go 后端服务 (Gin + GORM)
│   ├── cmd/server/       # 程序入口
│   ├── internal/         # 业务模块
│   ├── pkg/              # 公共工具包
│   ├── configs/          # 配置文件
│   ├── migrations/       # 主数据库迁移
│   └── scripts/          # 初始化脚本
├── frontend/             # React + TypeScript + Vite + Ant Design
│   ├── src/
│   │   ├── api/          # API 客户端
│   │   ├── components/   # 公共组件
│   │   ├── pages/        # 页面组件
│   │   ├── layouts/      # 布局组件
│   │   └── utils/        # 工具函数
│   └── public/           # 静态资源
├── deploy/               # Docker Compose + Nginx 部署配置
│   ├── docker-compose.dev.yml  # 开发环境编排
│   ├── docker-compose.yml      # 生产环境编排
│   ├── nginx.conf             # Nginx 配置
│   ├── deploy-*.sh           # 部署脚本
│   └── migrations/           # 部署相关迁移
├── docs/                 # 设计/测试/部署/用户手册
├── migrations/           # 历史补充迁移
└── _archive/             # 归档（不参与主链路）
```

### 2. 后端模块 (backend/internal/)

| 模块 | 路径 | 职责 |
|------|------|------|
| `agent` | `backend/internal/agent/` | Agent 部署能力 |
| `alert` | `backend/internal/alert/` | 告警管理（规则、联系人、渠道、模板） |
| `api` | `backend/internal/api/` | API 路由/处理入口 |
| `assistant` | `backend/internal/assistant/` | 运维小助手（AI 对话、RAG、Ollama 集成） |
| `audit` | `backend/internal/audit/` | 操作审计中间件 |
| `auth` | `backend/internal/auth/` | 用户认证、JWT、RBAC 角色权限 |
| `chatbot` | `backend/internal/chatbot/` | 旧版聊天机器人（已废弃） |
| `cicd` | `backend/internal/cicd/` | CI/CD 流水线管理 |
| `cmdb` | `backend/internal/cmdb/` | 资产管理（项目、环境、服务器、应用、部署） |
| `consul` | `backend/internal/consul/` | Consul 配置管理、批量下发 |
| `database` | `backend/internal/database/` | 数据库连接管理 |
| `models` | `backend/internal/models/` | 数据模型定义 |
| `monitor` | `backend/internal/monitor/` | 监控、健康检查、Prometheus 指标 |
| `platformevent` | `backend/internal/platformevent/` | 平台事件流 |
| `platformobject` | `backend/internal/platformobject/` | 统一对象索引 |
| `secureconfig` | `backend/internal/secureconfig/` | 安全配置 |
| `security` | `backend/internal/security/` | 安全扫描、FIM 漏洞管理 |
| `server` | `backend/internal/server/` | 服务启动与模块路由注册 |
| `tasks` | `backend/internal/tasks/` | 后台任务管理器 |

**后端公共包** (backend/pkg/):
- `config/` - Viper 配置加载
- `logger/` - Uber Zap 结构化日志
- `jenkins/` - Jenkins API 客户端
- `consul/` - Consul API 客户端

### 3. 前端结构 (frontend/src/)

**前端技术栈**: React 18 + TypeScript + Vite + Ant Design v5 + Axios + React Router v6

**入口文件**:
- `main.tsx` - 应用启动入口
- `App.tsx` - 路由总入口
- `components/MainLayout.tsx` - 主布局组件
- `components/AIChatbot.tsx` - AI 小助手组件

**前端页面** (pages/):

| 目录 | 页面 | 路径 |
|------|------|------|
| `admin/` | 角色/菜单/用户管理 | `/admin/*` |
| `alarm/` | 告警事件/规则/联系人/渠道/模板 | `/alarm/*` |
| `cmdb/` | 项目/环境/主机/应用管理 | `/cmdb/*` |
| `consul/` | Consul 配置管理 | `/consul/*` |
| `deploy/` | 部署发布/归档打包 | `/deploy/*` |
| `jenkins/` | Jenkins 视图 | `/jenkins/*` |
| `monitor/` | 监控大屏/概览/仪表盘 | `/monitor/*` |
| `platform/` | 平台事件/审计日志 | `/platform/*` |
| `security/` | 安全概览/FIM/漏洞/工单/资产 | `/security/*` |

**前端 API** (api/):

| 文件 | 职责 |
|------|------|
| `client.ts` | Axios 实例、拦截器、错误处理 |
| `admin.ts` | 用户、角色、菜单管理 |
| `cmdb.ts` | CMDB 资产管理 |
| `security.ts` | 安全扫描、漏洞 |
| `security-fim.ts` | FIM 文件完整性巡检 |
| `fim-known-hosts.ts` | FIM SSH 主机密钥 |
| `alert.ts` | 告警管理 |
| `monitor.ts` | 监控 |
| `consul.ts` | Consul 配置 |
| `jenkins.ts` | Jenkins |
| `audit.ts` | 审计日志 |
| `aggregate-package.ts` | 聚合打包 |
| `aggregated-history.ts` | 聚合历史 |
| `platform-events.ts` | 平台事件 |

### 4. 数据库迁移

**主迁移目录**: `backend/migrations/`

| 迁移文件 | 说明 |
|----------|------|
| `000002_add_cmdb_tables.sql` | CMDB 表（项目、环境、服务器、应用） |
| `000003_add_jenkins_fields.sql` | Jenkins 字段 |
| `000004_add_deploy_records.sql` | 部署记录 |
| `000005_add_archive_records.sql` | 归档记录 |
| `000006_add_archive_job_field.sql` | 归档 Job 字段 |
| `000007_add_project_code_to_archive_records.sql` | 归档记录项目代码 |
| `000008_add_security_module.sql` | 安全模块（资产、漏洞、工单） |
| `000009_add_operator_to_archive_records.sql` | 归档操作员字段 |
| `000010_add_real_name_to_users.sql` | 用户真实姓名 |
| `000011_add_security_fim_agentless.sql` | FIM 无代理扫描 |
| `000012_add_fim_known_hosts.sql` | FIM SSH 主机密钥 |

**历史补充迁移**: `migrations/` (不建议继续追加)

### 5. 部署架构 (deploy/)

**Docker Compose 服务**:
- `mysql` - MySQL 8.0 (开发: 13306, 生产: 3306)
- `redis` - Redis 7.4 (开发: 6379, 生产: 6379)
- `backend` - Go 后端服务 (8080)
- `nginx` - 反向代理 (开发: 8890, 生产: 80)

**Nginx 路由**:
- `/api/*` → 后端服务 (ops-backend:8080)
- `/auth/*` → 认证接口
- `/grafana-proxy/` → Grafana 仪表盘（带认证）
- `/` → 前端静态文件

**部署脚本**:
- `deploy-init.sh` - 初始化脚本
- `deploy-prod.sh` - 生产部署
- `deploy-update.sh` - 后端更新（远程）
- `dev.sh` - 开发辅助脚本

### 6. 关键 API 端点

| 路径 | 模块 | 功能 |
|------|------|------|
| `POST /api/auth/login` | auth | 用户登录 |
| `GET /api/user/me` | auth | 当前用户信息 |
| `GET /api/user/menus` | auth | 用户菜单权限 |
| `GET/POST /api/cmdb/servers` | cmdb | 服务器 CRUD |
| `GET/POST /api/cmdb/environments` | cmdb | 环境 CRUD |
| `GET/POST /api/cmdb/projects` | cmdb | 项目 CRUD |
| `GET/POST /api/cmdb/applications` | cmdb | 应用 + 部署触发 |
| `GET /api/cmdb/deploy-records` | cmdb | 部署记录 |
| `GET/POST /api/security/tasks` | security | 安全扫描任务 |
| `GET/POST /api/security/vulnerabilities` | security | 漏洞管理 |
| `GET/POST /api/security/fim/policies` | security | FIM 策略 |
| `GET/POST /api/security/assets` | security | 安全资产 |
| `GET/POST /api/security/tickets` | security | 漏洞工单 |
| `GET/POST /api/assistant/sessions` | assistant | AI 会话管理 |
| `POST /api/assistant/messages` | assistant | 发送 AI 消息 |
| `GET/POST /api/alert/rules` | alert | 告警规则 |
| `GET/POST /api/alert/contacts` | alert | 告警联系人 |
| `GET/POST /api/consul/configs` | consul | Consul 配置 |

### 7. 模块交互关系

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React)                       │
│  pages/  ←→  components/  ←→  api/  ←→  AIChatbot           │
└────────────────────────────┬────────────────────────────────┘
                             │ Axios (client.ts)
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Nginx Reverse Proxy                       │
│   /api/* → backend:8080   |   / → frontend static files     │
└────────────────────────────┬────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────────┐   ┌──────────────┐
│    Auth      │    │      CMDB        │   │   Security   │
│ /api/auth    │    │   /api/cmdb     │   │ /api/security│
└──────────────┘    └────────┬─────────┘   └──────────────┘
        │                    │                    │
        │            ┌───────┴───────┐          │
        │            ▼               ▼          ▼
        │     ┌────────────┐  ┌──────────────┐ ┌────────────┐
        │     │   MySQL    │  │   Jenkins    │ │   Ollama   │
        │     │   (GORM)   │  │ (部署/归档)  │ │ (AI 模型)  │
        │     └────────────┘  └──────────────┘ └────────────┘
        │                          │
        ▼                          ▼
   ┌────────┐               ┌────────────┐
   │ Redis  │               │    Git    │
   │(缓存)  │               │  (代码)   │
   └────────┘               └────────────┘
```

### 8. 快速定位索引

| 查找内容 | 文件路径 |
|----------|----------|
| 后端入口 | `backend/cmd/server/main.go` |
| 后端业务逻辑 | `backend/internal/<模块>/` |
| 前端页面 | `frontend/src/pages/<模块>/` |
| 前端 API 调用 | `frontend/src/api/<模块>.ts` |
| 数据库模型 | `backend/internal/models/` |
| API 路由注册 | `backend/internal/server/server.go` |
| 前端路由 | `frontend/src/App.tsx` |
| 前端主布局 | `frontend/src/components/MainLayout.tsx` |
| AI 小助手 | `frontend/src/components/AIChatbot.tsx` |
| 开发环境配置 | `deploy/docker-compose.dev.yml` |
| 生产环境配置 | `deploy/docker-compose.yml` |
| 数据库迁移 | `backend/migrations/*.sql` |
| Nginx 配置 | `deploy/nginx.conf` |


<claude-mem-context>
# Recent Activity

<!-- This section is auto-generated by claude-mem. Edit content outside the tags. -->

*No recent activity*
</claude-mem-context>