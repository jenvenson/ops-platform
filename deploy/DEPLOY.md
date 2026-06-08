# 生产环境部署指南

## 一、服务器要求

| 配置 | 最低要求 | 推荐配置 |
|------|----------|----------|
| CPU | 2 核 | 4 核 |
| 内存 | 4 GB | 8 GB |
| 磁盘 | 20 GB | 50 GB SSD |
| Docker | 20.10+ | 20.10+ |
| Docker Compose | 2.0+ | 2.0+ |

## 二、传输代码到服务器

### 本地打包

```bash
cd /path/to/ops-platform

# 打包
tar -czvf ops-platform.tar.gz backend frontend deploy \
  --exclude=backend/.git \
  --exclude=frontend/.git \
  --exclude=.git \
  --exclude=.worktrees \
  --exclude=node_modules \
  --exclude=backend/vendor
```

### 传输到服务器

```bash
scp ops-platform.tar.gz user@你的服务器IP:/opt/
```

## 三、服务器上部署

### 1. 解压

```bash
ssh user@你的服务器IP
mkdir -p /opt/ops-platform
tar -xzvf /opt/ops-platform.tar.gz -C /opt/ops-platform
cd /opt/ops-platform/deploy
```

### 2. 配置环境变量

```bash
cp .env.example .env
vim .env
```

**必须修改的配置：**

```bash
# 数据库密码（至少8位）
DB_PASSWORD=你的密码

# Redis 密码
REDIS_PASSWORD=你的密码

# JWT 密钥（至少32位）
JWT_SECRET=你的密钥至少32个字符
```

**建议同时确认 Grafana 代理配置：**

```bash
GRAFANA_URL=http://grafana.internal:3000
GRAFANA_USERNAME=admin
GRAFANA_PASSWORD=你的密码
GRAFANA_UPSTREAM=http://grafana.internal:3000/
GRAFANA_HOST_HEADER=grafana.internal:3000
GRAFANA_BASIC_AUTH=base64后的admin:password
GRAFANA_APP_URL=http://grafana.internal:3000/
```

如果 Grafana 不需要 Basic Auth，可将 `GRAFANA_BASIC_AUTH` 留空。

### 3. 执行首次部署

```bash
chmod +x deploy-init.sh deploy-update.sh
./deploy-init.sh
```

查看帮助：

```bash
./deploy-init.sh --help
```

如需指定环境变量文件或前端发布目录，可通过环境变量覆盖：

```bash
ENV_FILE=/opt/ops-platform/deploy/.env ./deploy-init.sh
NGINX_HTML_DIR=/opt/ops-platform/deploy/nginx/html ./deploy-init.sh
```

`deploy-init.sh` 默认会在检测到已有运行中的 compose 服务时直接退出，避免误覆盖现有环境。只有确认要重置当前 compose 项目时，才显式执行：

```bash
FORCE_INIT=1 ./deploy-init.sh
```

部署脚本会自动：
- 拉取镜像（MySQL、Redis、Nginx）
- 构建后端镜像（包含 Nmap、RustScan、Nuclei）
- 使用 Node 容器编译前端静态页面并发布到 `deploy/nginx/html`
- 初始化数据库
- 创建迁移状态表 `schema_migrations`
- 自动执行所有未应用迁移
- 启动所有服务

### 4. 验证部署

```bash
# 查看服务状态
docker compose ps

# 访问前端
curl http://localhost/

# 通过 Nginx 检查
curl http://localhost/api/health

# 直接检查后端
curl http://localhost:8080/health
```

## 四、初始化管理员账号

系统初始化时会创建管理员账号，但文档中不再公开默认用户名和密码。

请在首次登录后立即修改管理员密码，并通过受控渠道保管账号信息。

## 五、日常更新（迭代部署）

当前推荐方式不是“登录线上服务器手工更新代码”，而是**在本地执行 `deploy-update.sh`**，由脚本通过 `ssh/scp` 完成以下动作：

- 同步远端部署模板和迁移文件
- 校验线上 `.env`
- 在迁移前自动备份数据库
- 在远端执行数据库迁移
- 本地编译 Linux 后端二进制并替换远端后端程序
- 本地构建前端 `dist` 并刷新远端静态资源

### 本地执行的最终上线命令清单

在本地发布机执行：

```bash
cd /path/to/ops-platform/deploy
```

先确认前提条件：

- 线上已经执行过 `deploy-init.sh`
- 本地可以免密 SSH 到 `root@<your-server-ip>`
- 线上存在 `/opt/ops-platform/deploy/.env`
- 本地已安装 `go`、`npm`、`ssh`、`scp`、`tar`

发布前建议先检查线上 `.env` 是否包含：

- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `GRAFANA_UPSTREAM`
- `GRAFANA_HOST_HEADER`
- `GRAFANA_APP_URL`

先只同步并执行数据库迁移：

```bash
./deploy-update.sh migrate
```

`migrate` 会自动：

- 校验线上 `.env`
- 备份线上数据库到 `/opt/ops-platform/deploy/backups`
- 同步迁移文件和部署脚本
- 在远端执行所有待应用迁移

迁移完成后，建议立刻核对线上迁移记录：

```bash
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose exec mysql mysql -N -uroot -p\"\${DB_PASSWORD}\" ops_platform -e 'SELECT path, mode, applied_at FROM schema_migrations ORDER BY id DESC LIMIT 10;'"
```

确认无误后，再执行正式发布：

```bash
./deploy-update.sh
```

`deploy-update.sh` 现在会自动：

- 同步远端部署模板和迁移文件
- 校验线上 `.env`
- 自动备份线上数据库
- 执行所有待应用数据库迁移
- 再更新后端二进制和前端静态资源

如果这次发布包含以下变化，需要额外执行一次：

- 修改了 `deploy/nginx.prod.conf.template`
- 修改了线上 `.env` 中的 `GRAFANA_*`

执行命令：

```bash
./deploy-update.sh nginx
```

`nginx` 目标会：

- 同步最新生产 Nginx 模板
- 强制重建线上 Nginx 容器
- 重新渲染模板并校验页面与后端健康状态

目标差异总结：

- `all`
  - 同步部署文件
  - 备份数据库
  - 执行迁移
  - 发布后端
  - 发布前端
- `backend`
  - 同步部署文件
  - 备份数据库
  - 执行迁移
  - 只发布后端
- `frontend`
  - 只发布前端
- `nginx`
  - 同步模板并重建 Nginx
- `migrate`
  - 同步部署文件
  - 备份数据库
  - 只执行迁移

发布完成后，建议再做一次线上检查：

```bash
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose ps"
ssh root@<your-server-ip> "curl -sf http://localhost/api/health"
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose logs --tail 50 backend"
ssh root@<your-server-ip> "cd /opt/ops-platform/deploy && docker compose logs --tail 50 nginx"
```

最后手工打开并检查：

- 登录
- 工作台
- 监控大屏
- 监控概览
- Grafana 仪表盘
- FIM 策略
- FIM 已知主机

查看帮助：

```bash
./deploy-update.sh --help
```

如果远端主机、用户或部署目录和默认值不同，可以通过环境变量覆盖：

```bash
REMOTE_HOST=<your-server-ip> REMOTE_USER=deploy REMOTE_DEPLOY_DIR=/srv/ops-platform/deploy ./deploy-update.sh backend
REMOTE_HOST=<your-server-ip> REMOTE_USER=deploy REMOTE_DEPLOY_DIR=/srv/ops-platform/deploy ./deploy-update.sh migrate
```

## 六、常用命令

```bash
# 查看日志
docker compose logs -f

# 查看后端日志
docker compose logs -f backend

# 重启后端
docker compose restart backend

# 重启所有服务
docker compose restart

# 停止所有服务
docker compose down

# 启动所有服务
docker compose up -d
```

## 七、目录结构

```
ops-platform/
├── backend/              # Go 后端源码
│   ├── cmd/server/       # 入口文件
│   ├── internal/         # 业务逻辑
│   ├── configs/          # 配置文件
│   └── scripts/          # 数据库初始化脚本
├── frontend/             # 前端源码
└── deploy/               # Docker 部署配置
    ├── docker-compose.yml
    ├── Dockerfile.backend
    ├── nginx.prod.conf.template
    ├── nginx/html/       # 前端静态文件位置（deploy-init.sh 生成）
    ├── migrations/       # 数据库迁移
    ├── deploy-init.sh    # 首次部署脚本
    └── deploy-update.sh  # 更新部署脚本
```

## 八、端口说明

| 服务 | 端口 | 访问方式 |
|------|------|----------|
| Nginx | 80 | http://服务器IP |
| MySQL | 3306 | 内部访问 |
| Redis | 6379 | 内部访问 |
| Backend | 8080 | 内部访问 |

## 九、故障排查

### 1. 后端启动失败

```bash
# 查看错误日志
docker compose logs backend
```

### 2. 数据库连接失败

```bash
# 检查 MySQL 状态
docker compose ps

# 查看 MySQL 日志
docker compose logs mysql
```

### 3. 前端页面空白

```bash
# 检查静态文件
ls -la deploy/nginx/html/

# 检查 Nginx 错误日志
docker compose exec nginx cat /var/log/nginx/error.log

# 查看实际渲染后的 Nginx 配置
docker compose exec nginx cat /etc/nginx/nginx.conf
```

### 4. 安全扫描不工作

```bash
# 进入后端容器
docker compose exec backend sh

# 检查工具
which nmap
which nuclei
```

## 十、备份与恢复

### 备份数据库

```bash
docker compose exec mysql mysqldump -uroot -p"${DB_PASSWORD}" ops_platform > backup_$(date +%Y%m%d).sql
```

### 恢复数据库

```bash
docker compose exec -T mysql mysql -uroot -p"${DB_PASSWORD}" ops_platform < backup_20260212.sql
```

恢复完成后，如代码版本比备份更新，继续执行：

```bash
ENV_FILE=/opt/ops-platform/deploy/.env ./apply-migrations.sh
```

## 十一、安全加固（可选）

### 1. 修改默认密码

编辑 `.env` 文件后重启：

```bash
docker compose restart backend
```

### 2. 配置防火墙

```bash
# 只开放 80 端口
sudo ufw allow 80
sudo ufw enable
```

### 3. 启用 HTTPS（使用 Nginx Proxy Manager 或 Let's Encrypt）

## 十二、快速参考表

| 操作 | 命令 |
|------|------|
| 首次部署 | `./deploy-init.sh` |
| 更新部署 | `./deploy-update.sh` |
| 查看状态 | `docker compose ps` |
| 查看日志 | `docker compose logs -f` |
| 重启后端 | `docker compose restart backend` |
| 停止服务 | `docker compose down` |
| 启动服务 | `docker compose up -d` |

## 十三、联系信息

部署遇到问题请联系运维人员。
