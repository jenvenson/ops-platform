# Web 漏扫支持 Token + 签名系统的改造方案

日期：2026-03-18

## 背景

当前“安全管理 -> 扫描任务 -> Web漏洞扫描”对认证站点的支持模型偏简单，主要适配以下几类：

- `cookie`
- `bearer`
- `basic`
- `login-form`
- `login-token`

现有实现位于：

- [backend/internal/security/handler.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/handler.go)
- [backend/internal/security/engine.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/engine.go)

当前认证链路的核心限制：

1. 任务参数是平铺字段，表达能力有限。
2. `resolveWebAuth()` 最终只返回单个认证头。
3. `ExecuteNuclei()` 当前也只给 `nuclei` 注入一个 `-H` 请求头。
4. `login-form` 只适配“提交表单后拿 Cookie”的站点。
5. `login-token` 只适配“登录接口返回 JSON token，再按单头复用”的站点。

因此，当前实现不适合“多头 + 自定义登录请求 + 动态签名”的系统。

## 已定位的目标系统特征

针对 `http://your-app/web_01/` 的抓取和前端包分析，已经确认以下事实：

1. 这是一个 SPA，不是传统后端表单登录页。
2. `http://your-app/web_01/login` 返回 `404`，说明不存在可直接复用的传统登录页地址。
3. 前端使用 `localStorage` 保存 `token`，不是依赖登录响应 Cookie 维持会话。
4. 业务请求会带以下请求头：
   - `token`
   - `language`
   - `tenantkey`
   - `x-request-serial`
   - `x-signature`
   - `x-timestamp`
5. 登录接口为：
   - `POST /auth/oauth2/token`
6. 登录请求还依赖：
   - 客户端 `Authorization: Basic ...`
   - 表单提交
   - 密码按前端规则变换后再发送

进一步验证结果（2026-03-19）：

1. 真实可用凭据为：
   - `tenantKey=web_01`
   - `username=web_01`
   - `password=Web_01`
2. `grant_type=password` 是必填参数。
3. 登录签名规则已经确认：
   - `x-request-serial`: UUID
   - `x-timestamp`: 毫秒时间戳
   - `password`: `base64(serial + 明文密码 + timestamp)`
   - `x-signature`: `md5(base64(upper(serial-timestamp-path-method-secret)))`
4. 其中固定密钥为：
   - `your-login-secret`
5. 验证接口结果：
   - `GET /base/user/verify/type/web_01` 返回 `is_identity=false`
   - `GET /base/user/captcha/login/web_01?...` 返回 `data=null`
   - 当前租户登录不要求额外身份验证，也没有触发验证码
6. 登录成功后，至少以下接口仅靠 `token + tenantkey + language` 即可访问：
   - `GET /base/account`
   - `GET /base/user/use_permission_app_list`

这说明当前目标的最小可用扫描态是：

- 登录请求阶段使用自定义签名头
- 扫描请求阶段至少先带：
  - `token`
  - `tenantkey`
  - `language`

对于已验证的接口，不需要在每个请求上重新计算 `x-signature`。

结论：

- 当前 `login-form` 一定不适配该系统。
- 仅补一个简单 `bearer` 也不够，因为扫描请求不止一个认证头。
- 当前 `login-token` 也不够，因为它不支持自定义登录头、密码变换和动态签名。

## 目标能力

要支持这一类站点，认证系统至少要支持：

1. 自定义登录请求
   - URL
   - 方法
   - Content-Type
   - 多个请求头
   - JSON 或表单 body
2. 登录前变量生成
   - 时间戳
   - 随机串
   - Base64
   - 拼接
   - 哈希
3. 响应提取
   - 从 JSON 中提取 token
   - 从 Header 中提取值
4. 多请求头会话注入
   - 不是单个 `Authorization`
5. 可插拔签名器
   - 支持站点自定义签名逻辑
6. 必要时支持代理模式
   - 如果签名和每个请求的路径、方法、body 绑定，静态 `-H` 不足

## 建议架构

### 1. 将认证结果从“单头”升级为“会话对象”

现状：

- `resolveWebAuth()` 返回 `(mode, credential, header)`

建议改为：

```go
type HeaderKV struct {
    Name  string
    Value string
}

type ResolvedAuth struct {
    Headers   []HeaderKV
    Cookies   []HeaderKV
    Session   map[string]string
    ProxyURL  string
    DebugMeta map[string]string
}
```

这样 `ExecuteNuclei()` 就可以：

- 支持多个 `-H`
- 同时带多个静态头
- 未来可接代理模式

### 2. 新增高级认证配置模型

不再继续只堆平铺字段，新增 `auth_flow` JSON 配置。

建议在 `CreateTaskRequest` 中增加：

```go
AuthFlow json.RawMessage `json:"auth_flow"`
```

旧字段如 `auth_mode`、`login_url`、`username`、`password` 继续保留，用于兼容旧前端。

后端统一归一为内部结构：

```go
type AuthPlan struct {
    Type        string            `json:"type"`
    Login       *LoginStep        `json:"login,omitempty"`
    Session     *SessionTemplate  `json:"session,omitempty"`
    Signer      *SignerConfig     `json:"signer,omitempty"`
    Variables   map[string]string `json:"variables,omitempty"`
}
```

### 3. 抽独立认证引擎

建议新增文件：

- `backend/internal/security/auth_types.go`
- `backend/internal/security/auth_flow.go`
- `backend/internal/security/request_signer.go`

职责划分：

- `BuildAuthPlan()`：将旧参数或新 JSON 转换为统一内部配置
- `ExecuteLoginFlow()`：发起登录请求并提取变量
- `ResolveSessionHeaders()`：生成扫描阶段的头集合
- `RequestSigner`：对登录请求或扫描请求补充签名头

### 4. 将签名能力设计成插件接口

建议接口：

```go
type RequestSigner interface {
    Name() string
    Sign(req *http.Request, body []byte, session map[string]string) (map[string]string, error)
}
```

先实现：

- `none`
- `fscr`

这样像 `x-request-serial`、`x-timestamp`、`x-signature` 这类头可以由 signer 统一生成。

## 分阶段实施方案

### V1：先支持“高级 token 登录 + 多头注入”

目标：覆盖绝大多数“登录后拿 token，再带多个头扫描”的系统。

改造点：

1. `ExecuteNuclei()` 支持多个 `-H`
2. `resolveWebAuth()` 改为返回 `ResolvedAuth`
3. `login-token` 升级为“自定义登录请求”
4. 支持登录请求附加多个静态头
5. 支持变量模板：
   - `{{username}}`
   - `{{password}}`
   - `{{now_unix}}`
   - `{{now_unix_ms}}`
   - `{{rand_hex}}`
   - `{{base64(...)}}`
6. 支持从 JSON 中按路径提取：
   - `token`
   - `tenant`
   - 其他会话变量

这一步可支持：

- 登录接口需要 Basic 客户端认证
- 登录请求需要附加额外头
- 扫描请求需要多个静态头

但还不能完整解决“每个请求都要重新签名”的场景。

### V2：加入签名器

目标：支持“登录和扫描过程中需要额外签名头”的系统。

改造点：

1. 新增 `RequestSigner`
2. 登录请求支持 signer
3. 扫描前会话头支持 signer
4. 任务详情增加调试信息
   - 实际生效的头名
   - 提取到的变量名
   - 不回显敏感值

这一步能支持：

- 类似 `fscr` 这种需要动态生成若干签名头的系统

## 当前可直接使用的 auth_flow 示例

基于当前已补充的模板能力，`web_01` 可以先使用如下认证流：

```json
{
  "variables": [
    { "name": "tenant_key", "value": "web_01" },
    { "name": "lang", "value": "zh" },
    { "name": "client_basic", "value": "Basic {{base64:your-client-id:your-client-secret}}" },
    { "name": "request_serial", "value": "{{uuid}}" },
    { "name": "request_ts", "value": "{{now_unix_ms}}" },
    { "name": "login_secret", "value": "your-login-secret" },
    { "name": "sign_source", "value": "{{upper:${request_serial}-${request_ts}-/auth/oauth2/token-POST-${login_secret}}}" },
    { "name": "sign_payload", "value": "{{base64:${sign_source}}}" },
    { "name": "login_signature", "value": "{{md5:${sign_payload}}}" },
    { "name": "encoded_password", "value": "{{base64:${request_serial}${password}${request_ts}}}" }
  ],
  "login": {
    "url": "http://localhost:8080/auth/oauth2/token",
    "method": "POST",
    "content_type": "form",
    "headers": [
      { "name": "Authorization", "value": "${client_basic}" },
      { "name": "language", "value": "${lang}" },
      { "name": "tenantkey", "value": "${tenant_key}" },
      { "name": "x-request-serial", "value": "${request_serial}" },
      { "name": "x-timestamp", "value": "${request_ts}" },
      { "name": "x-signature", "value": "${login_signature}" }
    ],
    "body": {
      "grant_type": "password",
      "tenantKey": "${tenant_key}",
      "username": "${username}",
      "password": "${encoded_password}"
    },
    "extracts": [
      { "name": "token", "source": "json", "path": "data.accessToken.tokenValue" }
    ]
  },
  "session_headers": [
    { "name": "token", "value": "${token}" },
    { "name": "tenantkey", "value": "${tenant_key}" },
    { "name": "language", "value": "${lang}" }
  ]
}
```

任务接口中仍然需要传入：

- `auth_mode=advanced`
- `username=web_01`
- `password=Web_01`

当前已验证：

- 该 `auth_flow` 的登录请求字段和签名规则与前端真实实现一致
- 登录后的部分业务接口仅靠 `session_headers` 就可以访问

残留风险：

- 尚未证明所有业务接口都不依赖请求级签名
- 如果后续发现某些接口必须带 `x-signature`，仍需要继续演进为“请求级动态签名”或代理模式

前提：

- 这些签名头在一次扫描内可以静态复用，或与每个模板请求无强绑定

### V3：代理模式

目标：支持“签名与具体请求路径/方法/body 强绑定”的系统。

如果签名必须随每个请求动态重算，单靠 `nuclei -H` 不够。建议引入本地签名代理：

```text
nuclei -> local auth proxy -> target app
```

代理职责：

1. 拦截 `nuclei` 发出的请求
2. 注入 `token`
3. 注入 `tenantkey`
4. 动态计算 `signature`
5. 再转发到目标站点

优点：

- 不需要改写 `nuclei` 模板执行核心
- 复杂认证逻辑集中在平台内部

缺点：

- 实现复杂度更高
- 需要额外处理代理稳定性和调试日志

## 与当前代码的映射关系

### 后端

重点改造文件：

- [backend/internal/security/handler.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/handler.go)
- [backend/internal/security/engine.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/engine.go)

建议新增文件：

- [backend/internal/security/auth_types.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/auth_types.go)
- [backend/internal/security/auth_flow.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/auth_flow.go)
- [backend/internal/security/request_signer.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/security/request_signer.go)

建议的函数级改造：

1. `CreateTaskRequest`
   - 增加 `auth_flow`
2. `WebScanConfig`
   - 增加高级认证配置承载字段
3. `resolveWebAuth()`
   - 返回 `ResolvedAuth`
4. `ExecuteNuclei()`
   - 支持多个 `-H`
   - 未来支持 `-proxy`
5. `performLoginTokenAuth()`
   - 升级为通用 `ExecuteLoginFlow()`

### 前端

重点改造文件：

- [frontend/src/api/security.ts](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/api/security.ts)
- [frontend/src/pages/security/TaskList.tsx](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/frontend/src/pages/security/TaskList.tsx)

建议：

1. 保留现有简单模式：
   - none
   - cookie
   - bearer
   - basic
   - login-form
2. 新增“高级认证”模式
3. 先用 JSON 文本框承载 `auth_flow`
4. 后续再做可视化编排表单

## 数据与安全建议

当前任务表 [backend/internal/models/security.go](/Users/edy/Data/code/claude/ops-platform/.worktrees/phase1-framework/backend/internal/models/security.go) 没有认证配置快照字段。

建议补充：

- `auth_profile_type`
- `auth_profile_snapshot`

要求：

1. 任务详情可看见认证模型类型
2. 可看见脱敏后的配置快照
3. 不回显明文密码
4. token 只保存掩码值
5. 如需支持任务重跑，敏感值单独密文存储或仅保留运行期内存

## 明天继续的建议顺序

1. 先改 `ExecuteNuclei()`，让它支持多个 `-H`
2. 再抽 `ResolvedAuth` 和 `BuildAuthPlan()`
3. 再把 `performLoginTokenAuth()` 升级成通用登录流
4. 再补 `RequestSigner` 接口
5. 最后判断这个目标是否必须上代理模式

## 结论

当前 Web 漏扫无法支持该目标，不是参数没填对，而是认证能力模型不够。

真正要支持这类系统，最少需要：

- 高级登录流
- 多头会话注入
- 可插拔签名器

如果目标签名和每个请求强绑定，则最终还需要：

- 本地签名代理
