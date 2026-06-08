# 运维管理平台架构设计

> 设计日期：2026-01-26
> 目标用户：中小企业内部团队
> 部署方式：单机部署（Docker Compose）

---

## 1. 平台整体架构设计

### 架构层次

平台采用**单体架构 + 微服务模块化**设计，适合中小企业快速部署和运维。

```
┌─────────────────────────────────────────────────────────┐
│                        用户层                             │
│                    (浏览器 / CLI)                        │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                      接入层 (Nginx)                       │
│              静态资源 (React) + API 路由                  │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                   应用层 (Go 单体应用)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │   CMDB   │  │  监控    │  │  CI/CD   │               │
│  │  模块    │  │  模块    │  │  模块    │               │
│  └──────────┘  └──────────┘  └──────────┘               │
│         └──────────────┬───────────────┘                │
│                    统一 API 网关                          │
└─────────────────────────────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    MySQL     │  │  Prometheus  │  │    Redis     │
│  (业务数据)   │  │  (监控指标)   │  │   (缓存)     │
└──────────────┘  └──────────────┘  └──────────────┘
            │               │
            ▼               ▼
┌──────────────┐  ┌──────────────┐
│  目标服务器   │  │    Jenkins   │
│  (SSH 执行)   │  │  (构建任务)   │
└──────────────┘  └──────────────┘
```

### 技术栈

| 层次 | 技术 |
|------|------|
| 前端 | React + TypeScript + Ant Design |
| 后端 | Go (Gin 框架) |
| 数据库 | MySQL 8.0 |
| 监控 | Prometheus + Grafana |
| 缓存 | Redis 7.0 |
| CI/CD | Jenkins |
| 部署 | Docker Compose |
| 任务执行 | SSH (无需 Agent) |

### 关键设计决策

1. **Docker Compose 一键部署**：降低使用门槛
2. **Go 单体应用**：简化部署和调试，通过 `internal` 包隔离子系统边界
3. **前后端分离**：React + TypeScript + Ant Design
4. **SSH 直接连接**：无需 Agent 部署成本，适合中小企业环境

### 数据流向

```
前端请求 → Nginx → Go API → 业务逻辑 → MySQL/Prometheus/Redis/Jenkins/SSH
                                    ↓
                            响应返回
```

---

## 2. 三大子系统的职责与边界

### 2.1 CMDB 资产管理系统

**核心职责**：作为唯一数据源，管理 IT 资产全生命周期

- 资产录入、变更、下线
- 资产关系拓扑（项目、集群、服务器、应用）
- 资产标签、分组管理
- 资产查询与导出

**边界**：不负责执行具体操作（如监控采集、部署），只提供数据查询接口

**对外接口**：
- `GET /api/cmdb/servers` - 获取服务器列表
- `GET /api/cmdb/clusters` - 获取集群列表
- `GET /api/cmdb/applications` - 获取应用列表
- `POST /api/cmdb/changes` - 资产变更事件通知

### 2.2 基础监控管理系统

**核心职责**：监控数据采集配置、告警管理、可视化

- 管理 Prometheus 配置（基于 CMDB 数据自动生成）
- 告警规则配置与通知（邮件、钉钉、企微）
- 监控大盘展示
- 可用性监控

**边界**：依赖 CMDB 获取目标列表，指标存储和采集委托给 Prometheus

**对外接口**：
- `GET /api/monitoring/metrics` - 获取监控数据
- `GET /api/monitoring/alerts` - 获取告警列表
- `POST /api/monitoring/reload` - 触发 Prometheus 配置重载

### 2.3 CI/CD 自动化发布系统

**核心职责**：代码到生产的自动化流程编排

- Jenkins Pipeline 模板管理
- 发布任务编排
- 发布记录与回滚
- 发布状态查询

**边界**：依赖 CMDB 获取目标环境，实际构建委托给 Jenkins

**对外接口**：
- `POST /api/cicd/deploy` - 创建发布任务
- `GET /api/cicd/deployments` - 获取发布记录
- `POST /api/cicd/rollback` - 执行回滚

---

## 3. CMDB 核心资产模型

### 3.1 核心实体设计

```sql
-- 项目
CREATE TABLE projects (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '项目名称',
    description TEXT COMMENT '项目描述',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    KEY idx_name (name),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';

-- 集群/环境
CREATE TABLE clusters (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '集群名称',
    type ENUM('dev', 'test', 'prod') NOT NULL COMMENT '环境类型',
    description TEXT COMMENT '集群描述',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    KEY idx_name (name),
    KEY idx_type (type),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='集群表';

-- 服务器
CREATE TABLE servers (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    hostname VARCHAR(100) NOT NULL COMMENT '主机名',
    ip VARCHAR(45) NOT NULL COMMENT 'IP地址',
    os VARCHAR(50) COMMENT '操作系统',
    arch VARCHAR(20) COMMENT '架构',
    status ENUM('online', 'offline', 'maintenance') NOT NULL DEFAULT 'offline' COMMENT '状态',
    project_id BIGINT UNSIGNED NOT NULL COMMENT '所属项目',
    cluster_id BIGINT UNSIGNED NOT NULL COMMENT '所属集群',
    ssh_port INT DEFAULT 22 COMMENT 'SSH端口',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
    UNIQUE KEY uk_ip (ip),
    KEY idx_hostname (hostname),
    KEY idx_status (status),
    KEY idx_project (project_id),
    KEY idx_cluster (cluster_id),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器表';

-- 应用
CREATE TABLE applications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '应用名称',
    code_repo VARCHAR(255) COMMENT '代码仓库',
    deploy_path VARCHAR(255) COMMENT '部署路径',
    jenkins_job VARCHAR(100) COMMENT 'Jenkins任务名称',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_name (name),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用表';

-- 服务器-应用关联（部署关系）
CREATE TABLE server_apps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    server_id BIGINT UNSIGNED NOT NULL COMMENT '服务器ID',
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    version VARCHAR(50) COMMENT '部署版本',
    deployed_at DATETIME COMMENT '部署时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE,
    UNIQUE KEY uk_server_app (server_id, app_id),
    KEY idx_app (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器应用关联表';

-- 标签
CREATE TABLE tags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    key VARCHAR(50) NOT NULL COMMENT '标签键',
    value VARCHAR(100) NOT NULL COMMENT '标签值',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_key_value (key, value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签表';

-- 资产标签关联
CREATE TABLE asset_tags (
    asset_type ENUM('server', 'cluster', 'application') NOT NULL COMMENT '资产类型',
    asset_id BIGINT UNSIGNED NOT NULL COMMENT '资产ID',
    tag_id BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (asset_type, asset_id, tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资产标签关联表';
```

### 3.2 关键设计点

- **状态机**：服务器状态（online、offline、maintenance）
- **软删除**：所有实体使用 `deleted_at` 而非物理删除
- **审计字段**：`created_at`、`updated_at` 记录时间戳
- **关系拓扑**：通过外键和中间表支持多对多关系
- **项目维度**：所有服务器必须关联到项目，支持按项目管理资产

### 3.3 资产关系图

```
项目 (Projects)
    └── 集群 (Clusters)
            └── 服务器 (Servers)
                    └── 应用 (Applications) [N:M 关系]

标签 (Tags)
    └── 资产标签关联 (AssetTags)
```

---

## 4. 监控与 CMDB 的联动方案

### 4.1 核心机制：配置同步

```
┌──────────┐     变更事件      ┌──────────┐
│   CMDB   │ ────────────────▶ │  监控模块 │
│          │                   │          │
└──────────┘                   └──────────┘
                                    │
                                    ▼
                            生成 Prometheus 配置
                                    │
                                    ▼
                            HTTP API 热加载
```

**流程**：
1. **CMDB 触发**：当服务器/应用资产变更时，发送事件通知
2. **监控模块监听**：定时扫描 CMDB 变更（每分钟），生成 Prometheus 配置
3. **配置生成**：基于 CMDB 数据构建 `prometheus.yml` 和告警规则
4. **热加载**：通过 HTTP API 触发 Prometheus 重载配置

### 4.2 配置生成逻辑

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  # 服务器可用性监控
  - job_name: 'server_alive'
    static_configs:
      - targets:
          # 从 CMDB 查询所有 online 状态的服务器 IP:9100
          - '192.168.1.10:9100'
          - '192.168.1.11:9100'
        labels:
          project: 'project_name'  # 从 CMDB 关联获取
          cluster: 'prod'           # 从 CMDB 关联获取

  # 节点资源监控
  - job_name: 'node_metrics'
    static_configs:
      - targets:
          - '192.168.1.10:9100'
          - '192.168.1.11:9100'
        labels:
          project: 'project_name'
          cluster: 'prod'
```

### 4.3 可用性监控实现

```yaml
# 告警规则 alerts.yml
groups:
  - name: server_alive
    rules:
      - alert: ServerDown
        expr: up{job="server_alive"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务器 {{ $labels.instance }} 宕机"
          description: "服务器 {{ $labels.instance }} 已离线超过 1 分钟"

      - alert: ServerHighCPU
        expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "服务器 {{ $labels.instance }} CPU 使用率过高"
```

### 4.4 配置热加载

```go
// 触发 Prometheus 重载配置
func reloadPrometheusConfig(prometheusURL string) error {
    resp, err := http.Post(prometheusURL+"/-/reload", "text/plain", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}
```

---

## 5. CI/CD 自动化流程设计

### 5.1 发布流水线

```
开发提交代码
    │
    ▼
触发 Jenkins 构建 (Git Webhook)
    │
    ▼
质量检查（单测/代码扫描）
    │
    ▼
生成发布单
    │
    ▼
执行发布（SSH 部署）
    │
    ├─ 成功 → 更新 CMDB 记录
    │
    └─ 失败 → 自动回滚
```

### 5.2 核心组件

#### 5.2.1 Pipeline 模板
预置通用部署流程，支持多语言：

```yaml
# Java 应用部署模板
pipeline:
  stages:
    - name: build
      cmd: 'mvn clean package'
    - name: deploy
      cmd: 'scp target/app.jar user@{host}:{deploy_path}'
      cmd: 'ssh user@{host} "systemctl restart app"'
```

#### 5.2.2 发布单管理
```sql
CREATE TABLE deployments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    cluster_id BIGINT UNSIGNED NOT NULL COMMENT '集群ID',
    version VARCHAR(50) NOT NULL COMMENT '发布版本',
    status ENUM('pending', 'running', 'success', 'failed', 'rollback') NOT NULL,
    jenkins_build_id INT COMMENT 'Jenkins构建ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES applications(id),
    FOREIGN KEY (cluster_id) REFERENCES clusters(id)
);
```

#### 5.2.3 SSH 执行器
```go
// SSH 部署执行器
type SSHExecutor struct {
    Host     string
    Port     int
    Username string
    Key      string
}

func (e *SSHExecutor) Execute(cmd string) (string, error) {
    client, err := e.connect()
    if err != nil {
        return "", err
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return "", err
    }
    defer session.Close()

    return session.CombinedOutput(cmd)
}
```

### 5.3 CMDB 联动

- **发布时**：从 CMDB 获取目标环境的服务器列表
- **发布后**：更新 `server_apps` 表记录部署版本和状态
- **查询目标**：支持按集群或单机选择发布目标

### 5.4 回滚机制

```go
// 回滚到上一版本
func Rollback(deploymentID int64) error {
    // 1. 查询当前部署记录
    current, err := GetDeployment(deploymentID)
    if err != nil {
        return err
    }

    // 2. 查询上一成功版本
    prev, err := GetLastSuccessfulDeployment(current.AppID, current.ClusterID)
    if err != nil {
        return err
    }

    // 3. 执行上一版本的部署命令
    for _, server := range prev.Servers {
        executor := NewSSHExecutor(server.Host)
        _, err := executor.Execute(fmt.Sprintf("deploy %s", prev.Version))
        if err != nil {
            return err
        }
    }

    // 4. 记录回滚日志
    return CreateDeploymentLog(deploymentID, "rollback", "success")
}
```

---

## 6. MVP 功能拆解

### Phase 1：基础框架（2 周）

**目标**：完成项目脚手架和基础功能

- [ ] Go 项目脚手架搭建
  - [ ] 使用 Gin 框架
  - [ ] 配置文件管理（Viper）
  - [ ] 数据库连接（GORM）
- [ ] React 前端脚手架
  - [ ] 使用 Vite + TypeScript
  - [ ] 集成 Ant Design
  - [ ] API 请求封装
- [ ] Docker Compose 部署环境
  - [ ] Go 服务容器
  - [ ] React 前端容器
  - [ ] MySQL 容器
  - [ ] Nginx 反向代理
- [ ] 用户认证登录
  - [ ] JWT Token 认证
  - [ ] 用户管理接口
- [ ] MySQL 数据库初始化
  - [ ] 建表 SQL 脚本
  - [ ] 初始数据

**交付物**：
- 可运行的 Docker Compose 环境
- 用户登录功能

### Phase 2：CMDB 核心功能（2 周）

**目标**：完成 CMDB 资产管理核心功能

- [ ] 项目管理
  - [ ] 项目 CRUD 接口
  - [ ] 项目列表页面
- [ ] 集群管理
  - [ ] 集群 CRUD 接口
  - [ ] 集群列表页面
- [ ] 服务器管理
  - [ ] 服务器 CRUD 接口
  - [ ] 服务器状态管理
  - [ ] 服务器列表与搜索
  - [ ] SSH 连接测试
- [ ] 应用管理
  - [ ] 应用 CRUD 接口
  - [ ] 应用列表页面
- [ ] 资产关系展示
  - [ ] 项目-集群-服务器树形视图
  - [ ] 服务器应用部署关系图

**交付物**：
- 完整的 CMDB 管理界面
- 资产数据录入与查询

### Phase 3：监控集成（1 周）

**目标**：集成 Prometheus，实现基础监控

- [ ] Prometheus 集成
  - [ ] Prometheus 容器部署
  - [ ] 配置文件生成
  - [ ] CMDB 资产变更监听
- [ ] 服务器可用性监控
  - [ ] node_exporter 部署指南
  - [ ] 宕机告警规则配置
  - [ ] 可用性状态展示
  - [ ] 可用性趋势图表
- [ ] 基础资源监控大盘
  - [ ] CPU 使用率
  - [ ] 内存使用率
  - [ ] 磁盘使用率
- [ ] 告警规则配置
  - [ ] 告警规则 CRUD
  - [ ] 告警通知配置（邮件）

**交付物**：
- Prometheus 集成环境
- 服务器可用性监控
- 基础资源监控大盘

### Phase 4：CI/CD 核心（2 周）

**目标**：实现 CI/CD 核心发布功能

- [ ] Jenkins API 集成
  - [ ] Jenkins 任务触发
  - [ ] 构建状态查询
  - [ ] 构建日志获取
- [ ] Pipeline 模板管理
  - [ ] 模板 CRUD 接口
  - [ ] 预置模板（Java/Go/Node.js）
- [ ] 发布单管理
  - [ ] 发布单 CRUD 接口
  - [ ] 发布列表与详情页面
- [ ] SSH 部署执行
  - [ ] SSH 连接池
  - [ ] 远程命令执行
  - [ ] 文件传输
- [ ] 发布记录与回滚
  - [ ] 发布日志记录
  - [ ] 版本回滚功能

**交付物**：
- Jenkins 集成
- 发布任务管理
- 一键部署与回滚

### Phase 5：完善与优化（1 周）

**目标**：完善体验与文档

- [ ] 操作日志审计
  - [ ] 操作日志记录
  - [ ] 操作日志查询
- [ ] 前端体验优化
  - [ ] 加载状态优化
  - [ ] 错误提示优化
  - [ ] 响应式布局
- [ ] 文档补充
  - [ ] 部署文档
  - [ ] 使用手册
  - [ ] API 文档

**交付物**：
- 完整的使用文档
- 优化后的用户体验

---

## 7. 安全与可观测性

### 7.1 安全设计

- **密码加密**：用户密码使用 bcrypt 加密存储
- **SSH 密钥管理**：SSH 私钥加密存储在数据库
- **API 认证**：所有接口需 JWT Token 认证
- **操作审计**：所有写操作记录操作人和时间
- **SQL 防注入**：使用参数化查询（GORM）

### 7.2 可观测性设计

- **结构化日志**：使用 zap 记录结构化日志
- **请求追踪**：每个请求生成 trace_id，贯穿日志
- **健康检查**：提供 `/health` 接口用于探活
- **指标暴露**：暴露 Prometheus 指标（请求量、耗时、错误率）

---

## 8. 项目目录结构

```
ops-platform/
├── backend/                 # Go 后端
│   ├── cmd/                # 主程序入口
│   │   └── server/
│   │       └── main.go
│   ├── internal/           # 内部模块
│   │   ├── cmdb/          # CMDB 模块
│   │   ├── monitoring/    # 监控模块
│   │   ├── cicd/          # CI/CD 模块
│   │   ├── auth/          # 认证模块
│   │   └── ssh/           # SSH 执行器
│   ├── pkg/                # 公共包
│   ├── configs/            # 配置文件
│   └── scripts/            # SQL 脚本
├── frontend/                # React 前端
│   ├── src/
│   │   ├── pages/          # 页面
│   │   ├── components/     # 组件
│   │   ├── api/            # API 封装
│   │   └── utils/          # 工具函数
│   └── package.json
├── deploy/                  # 部署文件
│   ├── docker-compose.yml
│   ├── nginx.conf
│   └── prometheus.yml
└── docs/                    # 文档
    └── plans/
        └── 2026-01-26-ops-platform-design.md
```

---

## 9. 后续扩展方向

### 9.1 短期扩展（3-6 个月）

- 权限系统（RBAC）
- 消息通知（钉钉、企微）
- 告警规则模板化
- 更多语言支持

### 9.2 长期规划（6-12 个月）

- Agent 模式支持（应对复杂网络）
- Kubernetes 集成
- 灰度发布
- 成本分析
- 自动化运维（自愈）

---

**文档结束**
