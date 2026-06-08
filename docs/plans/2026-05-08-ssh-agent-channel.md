# SSH 统一智能体通道 — 实现方案

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 去掉每台机器的 Agent 进程，改为平台通过 SSH 统一管控所有服务器，实现命令执行、状态检测和凭据管理。

**Architecture:** 平台后端作为 SSH 客户端（类 Ansible 模式），通过 `golang.org/x/crypto/ssh` 连接被管服务器。每台服务器的 SSH 凭据存在 CMDB Server 表中。Agent 管理页面从"部署 Agent"改为"配置 SSH 连接"，在线状态通过 SSH 连接测试判断。

**Tech Stack:** Go + `golang.org/x/crypto/ssh` + Ant Design + React + TypeScript

---

### Task 1: Server 模型增加 SSH 凭据字段

**Files:**
- Modify: `backend/internal/cmdb/models.go:33-63`
- Create: `backend/migrations/000013_add_ssh_credentials.sql`

**Step 1: 添加 SSH 凭据字段到 Server 模型**

在 `Server` struct 的 `SSHPort` 字段后添加：

```go
SSHUser        string `json:"ssh_user" gorm:"size:50"`       // SSH 用户名
SSHAuthMethod  string `json:"ssh_auth_method" gorm:"size:20;default:'password'"` // password 或 key
SSHPassword    string `json:"-" gorm:"size:500"`              // SSH 密码（加密存储，json 不输出）
SSHKey         string `json:"-" gorm:"size:4096"`             // SSH 私钥（加密存储，json 不输出）
SSHKeyPath     string `json:"ssh_key_path" gorm:"size:255"`   // 或使用服务器上已有的私钥路径
LastSSHTest    *time.Time `json:"last_ssh_test"`              // 最后一次 SSH 连接测试时间
SSHTestResult  string `json:"ssh_test_result" gorm:"size:20"` // success / failed
```

**Step 2: 创建数据库迁移**

```sql
ALTER TABLE servers
  ADD COLUMN ssh_user VARCHAR(50) DEFAULT '',
  ADD COLUMN ssh_auth_method VARCHAR(20) DEFAULT 'password',
  ADD COLUMN ssh_password VARCHAR(500) DEFAULT '',
  ADD COLUMN ssh_key VARCHAR(4096) DEFAULT '',
  ADD COLUMN ssh_key_path VARCHAR(255) DEFAULT '',
  ADD COLUMN last_ssh_test DATETIME NULL,
  ADD COLUMN ssh_test_result VARCHAR(20) DEFAULT '';
```

**Step 3: 运行迁移并重启后端**

```bash
docker exec ops-mysql mysql -uroot -p... ops_platform < backend/migrations/000013_add_ssh_credentials.sql
docker compose -f deploy/docker-compose.dev.yml rm -sf backend && up -d backend
```

---

### Task 2: SSH 执行引擎

**Files:**
- Create: `backend/pkg/ssh/client.go` — SSH 客户端封装
- Create: `backend/pkg/ssh/client_test.go`

**Step 1: 实现 SSH 客户端**

```go
package ssh

import (
    "fmt"
    "net"
    "time"
    "golang.org/x/crypto/ssh"
)

type Client struct {
    conn *ssh.Client
}

type Config struct {
    Host       string
    Port       int
    User       string
    AuthMethod string // "password" 或 "key"
    Password   string
    PrivateKey string
    Timeout    time.Duration
}

func NewClient(cfg Config) (*Client, error) {
    // 构建 ssh.ClientConfig，超时 10s
    // 支持密码和私钥两种认证
    // 不验证主机密钥（生产可加 known_hosts）
}

func (c *Client) Execute(cmd string) (string, error) {
    // 创建 session，执行命令，返回组合输出
    // 超时 60s
}

func (c *Client) Test() error {
    // 执行 echo ok 测试连接
}

func (c *Client) Close() error {
    return c.conn.Close()
}
```

**Step 2: 单元测试**

用 `sshd` 测试库或 mock。

---

### Task 3: 重构 Agent Handler — SSH 命令执行 + 连接测试

**Files:**
- Modify: `backend/internal/agent/handler.go` — 新增 SSH 测试和命令执行
- Modify: `backend/internal/agent/routes.go` — 新增路由
- Modify: `backend/internal/agent/deploy.go` — 简化/移除 Agent 二进制相关代码

**Step 1: 实现 SSH 连接测试 Handler**

```go
// SSHPingHandler POST /api/agent/ssh-ping
// 输入: { server_id: N }
// 逻辑: 从 DB 查 server → 用 SSH 凭据连接 → 执行 echo → 更新 last_ssh_test / ssh_test_result / status
func SSHPingHandler(c *gin.Context) { ... }
```

**Step 2: 实现 SSH 命令执行 Handler（替换空壳 SSHCommandHandler）**

```go
// SSHCommandHandler POST /api/agent/ssh
// 输入: { server_id, command }
// 逻辑: 从 DB 查询凭据 → SSH 连接 → 执行命令 → 返回输出
func SSHCommandHandler(c *gin.Context) { ... }
```

**Step 3: 简化 deploy.go**

移除 `DownloadHandler`、`DeployHandler`、`InstallLogHandler`。保留 `GetServerStatus`（去掉 metrics 字段）。

**Step 4: 更新路由**

```go
func RegisterRoutes(r *gin.Engine) {
    agent := r.Group("/api/agent")
    {
        agent.POST("/ssh", SSHCommandHandler)     // 执行 SSH 命令
        agent.POST("/ssh-ping", SSHPingHandler)   // SSH 连接测试
        agent.GET("/status", GetServerStatus)      // 状态列表
        agent.GET("/status/:id", InstallLogHandler) // 单台详情（改名）
    }
}
```

---

### Task 4: 前端 API 更新

**Files:**
- Modify: `frontend/src/api/agent.ts`

```typescript
// 去除 deploy interface，新增 SSH 相关

export interface SSHPingRequest {
  server_id: number
}

export interface SSHCommandRequest {
  server_id: number
  command: string
}

export interface SSHCommandResponse {
  success: boolean
  output: string
  error?: string
}

export const agentAPI = {
  status: () => apiClient.get('/agent/status'),
  sshPing: (data: SSHPingRequest) => apiClient.post('/agent/ssh-ping', data),
  sshExec: (data: SSHCommandRequest) => apiClient.post('/agent/ssh', data),
}
```

---

### Task 5: Agent 管理页面改造

**Files:**
- Modify: `frontend/src/pages/agent/AgentManagementPage.tsx`

**改动：**

1. 表列从"部署/已部署"改为"测试连接 / 连接成功/失败"
2. 新增操作：点击「测试连接」→ 调 `/agent/ssh-ping` → 更新在线状态
3. 新增「执行命令」按钮 → 弹出 Modal → 输入命令 → 调 `/agent/ssh` → 显示输出
4. 在线/离线判断改为基于 `last_ssh_test` 和 `ssh_test_result`
5. 状态列改为：绿色=SSH 连接成功 / 红色=连接失败 / 灰色=未配置
6. 点击主机展开 SSH 凭据配置表单（用户、认证方式、密码/私钥）

**伪代码：**
```tsx
// 操作列
{
  title: '操作',
  render: (record) => (
    <Space>
      <Button onClick={() => handleSSHPing(record)}>测试连接</Button>
      {record.ssh_test_result === 'success' && (
        <Button onClick={() => handleCommand(record)}>执行命令</Button>
      )}
    </Space>
  ),
}
```

---

### Task 6: 清理 Agent 二进制相关代码

**Files:**
- Delete: `backend/cmd/agent/main.go`
- Delete: `backend/scripts/install-agent.sh`
- Delete: `backend/internal/agent/handler.go` 中的 `RegisterHandler`、`HeartbeatHandler`、`CheckOfflineServers`
- Delete: `backend/internal/server/server.go` 中的 `CheckOfflineServers` goroutine

**CMDB Server 模型处理：**
- 保留 Agent 字段（已上线数据库，删字段有风险）
- 不新增迁移删字段，标记为 deprecated

---

### 架构变更总结

```
之前                                之后
┌──────────┐                      ┌──────────────┐
│ OPS 平台  │                      │   OPS 平台    │
│ Agent页面 │                      │  SSH Manager  │
│ 注册/心跳 │                      │  ssh ping/exec│
└──┬──┬────┘                      └──┬──┬──┬─────┘
   │  │                               │  │  │
  Agent Agent                       SSH SSH SSH
  进程  进程                       (系统自带sshd)

每台机器需装 Agent ❌              零安装，利用 SSH ✅
```

---

### 影响范围

| 模块 | 变更 |
|------|------|
| cmdb/models.go | +7 SSH 凭据字段 |
| migrations/ | +1 迁移脚本 |
| pkg/ssh/ | 新建 SSH 客户端 |
| agent/handler.go | 删除注册/心跳，新增 SSH ping/exec |
| agent/routes.go | 路由调整 |
| agent/deploy.go | 删除下载/部署，保留状态查询 |
| cmd/agent/ | 删除 |
| server.go | 删除 CheckOfflineServers |
| frontend agent page | SSH 管理模式 |
| frontend api/agent.ts | 新接口 |
