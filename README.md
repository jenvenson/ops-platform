# OPS Platform 项目说明

## 1. 项目概述

OPS Platform 是一套面向运维场景的管理平台，当前主工程包括以下核心能力：

- CMDB 管理
- 自动化部署与归档
- 聚合打包与聚合历史
- Jenkins 与 Consul 集成
- 监控与告警管理
- 安全扫描与漏洞管理
- 系统管理与权限控制
- 聊天机器人辅助能力

当前主工程目录如下：

- `backend/`：Go 后端服务
- `frontend/`：React + Vite 前端应用
- `deploy/`：开发环境与生产环境部署文件
- `docs/`：设计、测试、部署、用户手册等文档
- `migrations/`：历史补充迁移文件

归档目录：

- `_archive/`：历史残留、重复结构和已归档文件

## 2. 目录结构

```text
phase1-framework/
├── backend/
├── frontend/
├── deploy/
├── docs/
├── migrations/
├── _archive/
├── .gitignore
└── CLAUDE.md
```

### 2.1 后端目录说明

- `backend/cmd/`：程序入口
- `backend/internal/`：业务模块
- `backend/pkg/`：公共封装
- `backend/configs/`：运行配置
- `backend/migrations/`：主数据库迁移
- `backend/scripts/`：初始化与辅助脚本

### 2.2 前端目录说明

- `frontend/src/`：页面、组件、API 封装
- `frontend/public/`：静态资源
- `frontend/package.json`：前端依赖与脚本定义

### 2.3 部署目录说明

- `deploy/docker-compose.dev.yml`：开发环境编排
- `deploy/docker-compose.yml`：生产环境编排
- `deploy/Dockerfile.backend`：后端镜像构建
- `deploy/Dockerfile.frontend`：前端镜像构建
- `deploy/nginx.conf.template`：开发环境 Nginx 模板
- `deploy/nginx.prod.conf.template`：生产环境 Nginx 模板
- `deploy/*.sh`：部署与更新脚本

## 3. 开发环境搭建

### 3.1 前置条件

开发环境依赖以下基础软件：

- Docker
- Docker Compose

建议环境：

- macOS / Linux
- 至少 4GB 可用内存
- 至少 10GB 可用磁盘空间

### 3.2 开发环境启动

开发环境采用 Docker Compose 进行编排，后端与前端均以源码挂载方式运行。

执行命令：

```bash
cd deploy
docker compose -f docker-compose.dev.yml up -d
```

### 3.3 开发环境服务端口

- MySQL：`13306:3306`
- Redis：`6379:6379`
- Backend：`8080:8080`
- Frontend：`5174:5173`
- Nginx：`8890:80`

访问地址：

- 前端入口：`http://localhost:8890`
- 后端接口：`http://localhost:8080`

### 3.4 开发环境运行机制

开发环境的实际行为如下：

- 后端容器在启动时执行 `go run ./cmd/server/main.go`
- 后端启动前会自动执行 `backend/scripts/init.sql` 和 `deploy/apply-migrations.sh`，确保开发库结构追平代码
- 前端容器在启动时执行 `npm run dev`
- 前端支持热更新
- 后端代码修改后通常需要重启 `backend` 容器

### 3.5 开发环境常用命令

```bash
# 启动
docker compose -f docker-compose.dev.yml up -d

# 查看状态
docker compose -f docker-compose.dev.yml ps

# 查看后端日志
docker compose -f docker-compose.dev.yml logs -f backend

# 重启后端
docker compose -f docker-compose.dev.yml restart backend

# 重启前端
docker compose -f docker-compose.dev.yml restart frontend

# 停止环境
docker compose -f docker-compose.dev.yml down
```

### 3.6 前端 smoke 验收

前端内置了可重复执行的 smoke 验收脚本与报告产物。

执行 smoke：

```bash
cd frontend
npm run acceptance:smoke
```

查看最新报告摘要：

```bash
cd frontend
npm run acceptance:report
```

报告产物默认输出到：

- `frontend/artifacts/acceptance-smoke/report.html`
- `frontend/artifacts/acceptance-smoke/last-run.json`

当前 smoke 主要覆盖：

- 一级菜单与关键二级菜单
- 关键页面入口与页面标识
- 右上角入口
- 运维小助手页面导航主流程

## 4. 开发迭代说明

### 4.1 后端迭代

后端代码变更后，执行：

```bash
cd deploy
docker compose -f docker-compose.dev.yml restart backend
```

### 4.2 前端迭代

前端页面修改后通常会自动热更新。如前端容器异常，可执行：

```bash
cd deploy
docker compose -f docker-compose.dev.yml restart frontend
```

### 4.3 数据库变更

数据库迁移分为两类：

- 主迁移目录：`backend/migrations/`
- 历史补充迁移目录：`migrations/`
- 部署补充迁移目录：`deploy/migrations/`

维护原则：

- 新迁移优先写入 `backend/migrations/`
- `migrations/` 仅保留历史补充迁移，不建议继续追加
- 执行迁移前需确认当前数据库版本和历史执行顺序
- 标准执行入口统一为 `deploy/apply-migrations.sh`
- 迁移记录保存在数据库表 `schema_migrations`
- 现有数据库第一次接入该机制时，会在检测到“已有完整业务表但没有迁移记录”时执行一次基线登记，之后只执行新增迁移

## 5. 生产环境搭建

### 5.1 前置条件

生产环境建议满足以下条件：

- Linux 服务器
- Docker
- Docker Compose
- 至少 4GB 内存
- 至少 20GB 磁盘空间

推荐部署目录：

```bash
/opt/ops-platform
```

### 5.2 准备部署目录

首次部署前，生产服务器上需要先准备完整项目目录。常见方式：

- 通过代码仓库检出到 `/opt/ops-platform`
- 通过离线压缩包上传并解压到 `/opt/ops-platform`

进入部署目录：

```bash
cd /opt/ops-platform/deploy
cp .env.example .env
```

### 5.3 生产环境配置

请至少配置以下环境变量：

- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `JENKINS_URL`
- `JENKINS_USERNAME`
- `JENKINS_TOKEN`
- `GRAFANA_URL`
- `GRAFANA_USERNAME`
- `GRAFANA_PASSWORD`
- `GRAFANA_UPSTREAM`
- `GRAFANA_HOST_HEADER`
- `GRAFANA_BASIC_AUTH`（如 Grafana 仍使用 Basic Auth）
- `GRAFANA_APP_URL`

### 5.4 执行首次部署

首次部署统一使用 `deploy-init.sh`，不要再手工逐步执行数据库初始化和容器启动。

```bash
cd /opt/ops-platform/deploy
chmod +x deploy-init.sh deploy-update.sh
./deploy-init.sh
```

`deploy-init.sh` 会自动完成：

- 启动 MySQL 和 Redis
- 执行 `backend/scripts/init.sql`
- 创建 `schema_migrations`
- 自动执行所有待应用迁移
- 构建后端镜像
- 发布前端静态资源
- 启动完整生产服务

### 5.5 首次部署后验证

```bash
cd /opt/ops-platform/deploy
docker compose ps
curl -sf http://localhost/api/health
docker compose logs --tail 50 backend
docker compose logs --tail 50 nginx
```

默认端口：

- Nginx：`80`
- Backend：`8080`
- MySQL：`3306`
- Redis：`6379`

## 6. 生产环境部署与更新

### 6.1 标准更新方式

生产环境日常更新统一使用本地执行的 `deploy/deploy-update.sh`，不要登录线上服务器手工 `git pull`、手工编译前端或手工替换容器文件。

执行位置：

- 在本地发布机执行

前提条件：

- 线上已完成首次部署
- 本地可免密 SSH 到目标服务器
- 线上已有 `/opt/ops-platform/deploy/.env`
- 本地已安装 `go`、`npm`、`ssh`、`scp`、`tar`

推荐发布顺序：

```bash
cd /Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/deploy
./deploy-update.sh migrate
./deploy-update.sh
```

说明：

- `migrate` 和默认 `all` 会先校验线上 `.env`
- `migrate`、`backend`、`all` 会在迁移前自动备份线上数据库
- 如果修改了 `nginx.prod.conf.template` 或线上 `.env` 里的 `GRAFANA_*`，需要额外执行一次 `./deploy-update.sh nginx`

### 6.2 deploy-update.sh 实际行为

`deploy/deploy-update.sh` 会自动完成：

- 同步远端部署模板和迁移文件
- 校验线上 `.env` 是否满足当前目标所需变量
- 对 `migrate`、`backend`、`all` 自动备份线上数据库
- 在远端执行数据库迁移
- 本地编译 Linux 后端二进制并替换线上后端程序
- 本地构建前端 `dist` 并刷新线上静态资源

目标差异：

- `./deploy-update.sh`
  - 同步部署文件
  - 执行迁移
  - 发布后端
  - 发布前端
- `./deploy-update.sh migrate`
  - 只同步部署文件并执行迁移
- `./deploy-update.sh backend`
  - 同步部署文件
  - 执行迁移
  - 只发布后端
- `./deploy-update.sh frontend`
  - 只发布前端静态文件
- `./deploy-update.sh nginx`
  - 同步生产 Nginx 模板并强制重建线上 Nginx 容器

注意：

- 默认 `all` 会同步最新 `nginx.prod.conf.template`，但不会自动重建 Nginx 容器
- 模板变更和 `GRAFANA_*` 配置变更，必须单独执行 `./deploy-update.sh nginx`

如需只执行部分动作，可使用：

```bash
./deploy-update.sh backend
./deploy-update.sh frontend
./deploy-update.sh nginx
./deploy-update.sh migrate
```

### 6.3 发布后检查

```bash
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose ps"
ssh root@<your-server-ip> "curl -sf http://localhost/api/health"
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose logs --tail 50 backend"
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose logs --tail 50 nginx"
```

并手工检查：

- 登录
- 工作台
- 监控大屏
- 监控概览
- Grafana 仪表盘
- FIM 策略
- FIM 已知主机

## 7. 运维与维护建议

### 7.1 开发与部署统一入口

- 开发环境统一通过 `deploy/docker-compose.dev.yml`
- 生产环境统一通过 `deploy/docker-compose.yml`

### 7.2 代码维护边界

日常功能开发应只在以下目录进行：

- `backend/`
- `frontend/`
- `deploy/`
- `docs/`

### 7.3 归档目录说明

`_archive/` 目录用于保存以下内容：

- 历史重复目录
- 一次性脚本
- 旧根级配置文件
- 历史修复说明

该目录不参与当前主工程运行。

## 8. 相关文档

- `deploy/DEPLOY.md`：最新生产部署与上线清单
- `docs/deploy.md`：部署文档入口说明（不再维护完整流程）
- `docs/design.md`：设计文档
- `docs/testing.md`：测试说明
- `docs/user_manual.md`：用户手册
