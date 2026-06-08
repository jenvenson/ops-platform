# 基于 qwen3:8b 的运维小助手详细落地设计（历史方案）

## 1. 目标

本文档用于定义 OPS Platform 在本地模型方案下的“运维小助手”详细落地设计。

说明：

- 当前落地默认对话模型已调整为 `qwen2.5:1.5b`
- 本文档保留 `qwen3:8b` 方案，作为历史设计和大模型档位参考

当前默认配置：

- 模型运行底座：`Ollama`
- 默认对话模型：`qwen2.5:1.5b`
- 向量模型：`qwen3-embedding:4b`

历史方案说明：

- `qwen3:8b` 曾作为一期默认对话模型方案
- 当前保留其设计思路，用于质量优先场景评估

设计目标：

- 在当前项目中快速落地一期可用版本
- 优先支持知识问答、页面导航、只读查询
- 保证模型与业务系统之间有清晰安全边界
- 为后续执行类能力预留架构

## 2. 一期范围

一期只做以下能力：

- 文档问答
- 页面导航
- 平台只读查询
- 引用来源展示
- 基础会话存储

一期不做：

- 自动发布
- 自动回滚
- 自动修改配置
- 自动执行 shell
- 多 Agent 自主决策

## 3. 总体架构

```text
浏览器
-> AIChatbot
-> /api/assistant/messages
-> assistant api handler
-> assistant service
   -> intent classifier
   -> retriever
   -> tools
   -> ollama client
-> response assembler
-> 前端渲染 answer / citations / actions
```

核心原则：

- 当前默认由 `qwen2.5:1.5b` 负责理解、路由、总结
- `qwen3:8b` 作为历史高档位方案保留
- 平台数据由 `tools` 提供
- 文档知识由 `retriever` 提供
- 安全控制由后端保证，不交给模型

## 4. 目录设计

建议新增以下目录：

```text
backend/internal/assistant/
├── api/
│   ├── handler.go
│   └── routes.go
├── service/
│   ├── assistant.go
│   ├── classifier.go
│   ├── planner.go
│   └── responder.go
├── llm/
│   ├── client.go
│   ├── ollama.go
│   └── types.go
├── retriever/
│   ├── chunker.go
│   ├── indexer.go
│   ├── search.go
│   └── types.go
├── tools/
│   ├── registry.go
│   ├── navigate.go
│   ├── cmdb.go
│   ├── deploy.go
│   ├── alert.go
│   ├── security.go
│   └── types.go
├── memory/
│   └── session.go
├── audit/
│   └── logger.go
├── prompt/
│   ├── system.txt
│   ├── classify.txt
│   └── answer.txt
└── models/
    ├── session.go
    ├── message.go
    ├── tool_call.go
    └── citation.go
```

## 5. 模块职责

### 5.1 api

负责：

- 暴露 HTTP 接口
- 获取当前登录用户
- 参数校验
- 返回统一 JSON 结构

### 5.2 service

负责：

- 串联完整对话流程
- 调用分类、检索、工具和模型
- 组装最终回答

### 5.3 llm

负责：

- 封装 Ollama API
- 控制模型参数
- 屏蔽模型实现细节

### 5.4 retriever

负责：

- 文档切片
- 向量索引
- 相似度检索
- 返回引用内容

### 5.5 tools

负责：

- 将业务能力封装为受控工具
- 做权限校验
- 以结构化结果返回查询结果

### 5.6 memory

负责：

- 管理最近几轮上下文
- 支持摘要化历史

### 5.7 audit

负责：

- 记录用户问题
- 记录工具调用
- 记录模型耗时和结果状态

## 6. 会话处理流程

### 6.1 创建会话

```text
前端初始化
-> POST /api/assistant/sessions
-> 创建 assistant_session
-> 返回 session_id
```

### 6.2 消息处理主链路

```text
用户发送消息
-> 读取用户身份
-> 加载最近 N 轮会话
-> 分类意图
-> 判断是否需要:
   - 直接回答
   - 检索文档
   - 调用工具
-> 汇总上下文
-> 调用当前默认模型（`qwen2.5:1.5b`，历史方案为 `qwen3:8b`）
-> 返回 answer / citations / actions
-> 落库消息和日志
```

### 6.3 一期推荐路由策略

- `knowledge_qa`
  走 `retriever + llm`

- `page_navigation`
  走 `navigate tool + llm`

- `readonly_query`
  走 `tool + llm`

- `fallback`
  走 `llm` 基础回答

## 7. 接口设计

### 7.1 创建会话

```http
POST /api/assistant/sessions
```

请求：

```json
{
  "scene": "web",
  "user_agent": "Mozilla/5.0"
}
```

响应：

```json
{
  "session_id": "asst_20260312_xxx",
  "created_at": "2026-03-12T10:00:00Z"
}
```

### 7.2 发送消息

```http
POST /api/assistant/messages
```

请求：

```json
{
  "session_id": "asst_20260312_xxx",
  "message": "如何查看部署失败原因？"
}
```

响应：

```json
{
  "message_id": "msg_xxx",
  "intent": "knowledge_qa",
  "answer": "可以在发布历史中查看失败记录，并进入对应日志页面。",
  "citations": [
    {
      "title": "部署说明",
      "path": "docs/deploy.md",
      "snippet": "可通过发布历史查看失败记录"
    }
  ],
  "actions": [
    {
      "type": "navigate",
      "label": "打开发布历史",
      "path": "/deploy/history"
    }
  ]
}
```

### 7.3 查询会话历史

```http
GET /api/assistant/sessions/:session_id/messages
```

一期可选实现。

## 8. 数据表设计

### 8.1 assistant_sessions

建议字段：

- `id`
- `session_id`
- `user_id`
- `scene`
- `status`
- `summary`
- `created_at`
- `updated_at`

### 8.2 assistant_messages

建议字段：

- `id`
- `session_id`
- `role`
- `content`
- `intent`
- `model_name`
- `prompt_tokens`
- `completion_tokens`
- `latency_ms`
- `created_at`

### 8.3 assistant_tool_calls

建议字段：

- `id`
- `session_id`
- `message_id`
- `tool_name`
- `tool_input_json`
- `tool_output_json`
- `status`
- `duration_ms`
- `error_message`
- `created_at`

### 8.4 assistant_citations

建议字段：

- `id`
- `message_id`
- `source_type`
- `source_title`
- `source_path`
- `snippet`

## 9. Ollama 接入设计

### 9.1 配置项

建议配置：

```env
ASSISTANT_PROVIDER=ollama
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_CHAT_MODEL=qwen3:8b
OLLAMA_EMBED_MODEL=qwen3-embedding:4b
ASSISTANT_MAX_CONTEXT_MESSAGES=12
ASSISTANT_TOP_K=4
ASSISTANT_TEMPERATURE=0.2
```

当前默认建议改为：

```env
ASSISTANT_PROVIDER=ollama
OLLAMA_BASE_URL=http://127.0.0.1:11434
OLLAMA_CHAT_MODEL=qwen2.5:1.5b
OLLAMA_EMBED_MODEL=qwen3-embedding:4b
ASSISTANT_MAX_CONTEXT_MESSAGES=12
ASSISTANT_TOP_K=4
ASSISTANT_TEMPERATURE=0.2
```

### 9.2 Chat 请求封装

后端建议统一封装：

```go
type ChatRequest struct {
    Model       string
    System      string
    Messages    []Message
    Temperature float64
}
```

### 9.3 Ollama 调用建议

- 文档问答：`temperature=0.2`
- 查询总结：`temperature=0.1`
- 普通闲聊兜底：`temperature=0.5`

避免高温度，防止回答发散。

## 10. Prompt 设计

### 10.1 System Prompt

建议原则：

- 明确角色是“OPS Platform 运维小助手”
- 优先回答平台相关问题
- 不确定时明确说明不知道
- 只能根据提供的文档和工具结果回答
- 不得虚构页面路径和操作结果

建议模板：

```text
你是 OPS Platform 的运维小助手。
你的职责是帮助用户理解平台功能、定位页面入口、总结查询结果。
你只能根据给定的上下文、引用资料和工具结果回答。
如果信息不足，直接说明信息不足，不要编造。
回答使用中文，简洁、明确、可执行。
```

### 10.2 分类 Prompt

分类结果建议为以下枚举：

- `knowledge_qa`
- `page_navigation`
- `readonly_query`
- `fallback`

### 10.3 回答 Prompt

输入上下文包含：

- 用户问题
- 最近几轮消息摘要
- 检索结果
- 工具结果

输出要求：

- 先给结论
- 再给步骤或解释
- 如有页面入口，给出路径
- 如有引用，尽量与回答一致

## 11. RAG 设计

### 11.1 首批知识源

- [docs/deploy.md](docs/deploy.md)
- [docs/design.md](docs/design.md)
- [docs/testing.md](docs/testing.md)
- [docs/user_manual.md](docs/user_manual.md)
- [docs/project-structure.md](docs/project-structure.md)
- [docs/运维小助手技术方案.md](docs/运维小助手技术方案.md)

### 11.2 分片规则

建议：

- 按标题切片
- 每片 300 到 800 中文字
- 保留文件路径、标题、模块标签

### 11.3 检索策略

- 召回前 `top_k=4`
- 只将最相关片段注入模型
- 回答时附带引用来源

## 12. 工具设计

### 12.1 工具注册接口

```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx *ToolContext, input any) (*ToolResult, error)
}
```

### 12.2 一期工具清单

#### 页面导航

- `navigate_to_page`

输入：

- `keyword`

输出：

- `title`
- `path`

#### CMDB 查询

- `query_projects`
- `query_environments`
- `query_servers`

#### 部署查询

- `query_release_history`
- `query_aggregate_history`

#### 告警查询

- `query_alert_events`

#### 安全查询

- `query_vulnerabilities`
- `query_assets`

### 12.3 工具实现原则

- 工具输入必须结构化
- 工具层负责参数校验
- 工具层负责权限判断
- 模型只看工具结果摘要，不看敏感原始数据

## 13. 前端改造建议

现有 [AIChatbot.tsx](frontend/src/components/AIChatbot.tsx) 可以继续使用，但建议扩展响应结构支持：

- `answer`
- `intent`
- `citations`
- `actions`

推荐消息结构：

```ts
type AssistantResponse = {
  message_id: string
  intent: string
  answer: string
  citations?: Array<{
    title: string
    path: string
    snippet?: string
  }>
  actions?: Array<{
    type: string
    label: string
    path?: string
  }>
}
```

前端建议增加：

- 引用来源展示
- 一键跳转按钮
- 错误状态提示
- 加载中状态

## 14. 安全设计

### 14.1 基本原则

- 所有请求绑定登录用户
- 一期仅开放只读能力
- 不允许模型生成可执行命令并直接运行

### 14.2 数据控制

- 工具返回内容做脱敏
- 不在 prompt 中注入 Token、密码、密钥
- 工具结果只保留回答所需字段

### 14.3 审计

必须记录：

- 用户问题
- 分类结果
- 检索命中文档
- 调用的工具
- 模型名称
- 响应耗时

## 15. 开发任务拆分

### Phase 1: 基础框架

- 新建 `assistant` 目录结构
- 增加路由和 handler
- 增加 session/message 表
- 增加 Ollama 客户端

### Phase 2: 纯模型问答

- 跑通当前默认 `qwen2.5:1.5b` 调用
- 跑通基础会话
- 跑通简单问答

### Phase 3: RAG 问答

- 实现文档分片
- 实现索引构建
- 实现检索与引用返回

### Phase 4: 工具接入

- 接页面导航工具
- 接 CMDB 查询工具
- 接部署查询工具
- 接告警/安全查询工具

### Phase 5: 前端联调

- 前端切换 `/api/assistant/messages`
- 渲染引用与导航动作
- 处理错误与加载态

## 16. 一期验收标准

- 能回答常见平台使用问题
- 能根据问题返回正确页面入口
- 能查询基础只读业务信息
- 能引用文档来源
- 响应时间可接受
- 不涉及高风险执行操作

## 17. 推荐实施顺序

建议实际执行顺序：

1. 先接 `Ollama + qwen2.5:1.5b`
2. 再做文档问答
3. 再做页面导航
4. 再做只读查询工具
5. 最后再优化多轮上下文和回答样式

这样可以最快做出一个“能答、能查、能带路”的本地版运维小助手。

补充说明：

- 如果后续硬件资源充足、且更看重回答质量，可以回切或并行评估 `qwen3:8b`
