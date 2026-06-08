# Codex Best Practices 与上下文优化建议

按 OpenAI 官方这篇最佳实践来看，优化上下文使用的核心不是“塞更多信息”，而是“只给对的上下文，并把可复用信息沉淀出去”。

官方文章里最关键的几条是：

- 提示里尽量固定 4 个部分：`Goal`、`Context`、`Constraints`、`Done when`
- 把长期有效的规则放进 `AGENTS.md`
- 复杂任务先 `plan`
- 仓库外、会变动的数据优先走 MCP/工具，不要反复手贴
- 长任务按“一个任务一个线程”管理，线程变长时用 compact，而不是一直累加历史

来源：<https://developers.openai.com/codex/learn/best-practices/?utm_source=chatgpt.com>

## 结合当前仓库的落地建议

### 1. 请求时只给“最小必要上下文”

不要一上来贴大段历史。优先给：

- 目标：要改什么
- 相关文件：精确到 2 到 5 个路径
- 限制：不能动什么
- 完成标准：改完看什么结果

例如比起“帮我看看运维小助手”，更好的是：

- 目标：把开发环境默认模型切到 `qwen3:4b`
- 上下文：`backend/pkg/config/config.go`、`deploy/docker-compose.dev.yml`、`deploy/.env.example`
- 约束：线上部署方式不要改
- 完成标准：开发环境默认走宿主机 Ollama 的 `4B`

### 2. 把稳定规则继续收口到 `AGENTS.md`

你现在已经在做这件事了，这是对的。接下来重点不是写更长，而是写更“硬”：

- 仓库根目录到底在哪
- 开发/生产的 Ollama 差异
- 哪些脚本是实际入口
- 哪些文件是历史文档，不要误当现状

官方也明确说，`AGENTS.md` 要“短、准、基于真实摩擦”更有效。

来源：同页 `Make guidance reusable with AGENTS.md`

### 3. 一类任务一个线程，不要一个线程包所有事情

你现在这个项目里，线上排障、菜单修复、Jenkins、Ollama、文档更新如果都堆在一个线程里，后面上下文会越来越脏。

更好的分法：

- 线程 A：线上聚合打包故障
- 线程 B：Ollama 部署与模型切换
- 线程 C：AGENTS.md / 文档治理

官方明确提到，不要“一个项目一个线程”，而要“一个任务一个线程”。

来源：同页 `Organize long-running work with session controls`

### 4. 长日志和长文档不要整段贴，先摘要再引用文件

你这类仓库最容易浪费上下文的是：

- `docker logs`
- 大段 SQL 输出
- 大篇部署文档
- 大段 diff

更好的做法是：

- 先给 3 到 5 行结论
- 再给文件路径或命令
- 只有定位不到时再展开原文

### 5. 把“重复解释”的内容变成技能或模板

如果你经常做这些事：

- 线上排障
- 菜单修复
- Ollama 联调
- Jenkins 触发链路检查

就可以沉淀成固定模板，甚至 skill。

官方建议：重复工作不要一直靠长 prompt，应该沉淀成 `AGENTS.md`、skills 或工具集成。

来源：同页 `Turn repeatable work into skills`

### 6. 仓库外动态信息不要靠聊天记忆

像这些内容不适合靠上下文硬记：

- 线上 `.env`
- Jenkins token
- 数据库实时状态
- Ollama 已安装模型
- 容器网络现状

这类信息应该实时查，或接 MCP/工具，而不是在会话里反复复制。

官方建议也是：上下文在仓库外且变化频繁时，用 MCP/工具。

来源：同页 `Use MCPs for external context`

## 一句话原则

- 短期任务信息放 prompt
- 长期规则放 `AGENTS.md`
- 重复流程做成模板/skill
- 动态数据靠工具实时查
- 一个任务一个线程，长了就 compact

## 参考来源

- OpenAI Codex Best Practices: <https://developers.openai.com/codex/learn/best-practices/?utm_source=chatgpt.com>
