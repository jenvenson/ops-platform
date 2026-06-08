# AGENTS.md 编写落地版

本文档用于把 Codex 最佳实践收口成一套可直接用于编写 `AGENTS.md` 的规则，重点不是讲概念，而是指导如何写出“短、硬、有效”的仓库级约束。

## 1. AGENTS.md 的目标

`AGENTS.md` 的作用不是重复 README，也不是写一份完整项目说明书。

它更适合承载这三类信息：

- 代理最容易做错的事
- 这个仓库特有的事实
- 默认工作方式和验证要求

一句话说，`AGENTS.md` 要解决的是：

- 让代理少误判
- 让代理少走弯路
- 让代理优先按你的协作方式做事

## 2. 适合写进 AGENTS.md 的内容

优先写这几类内容。

### 2.1 仓库事实

只写真实有效、会影响执行结果的事实，例如：

- 真正的仓库根目录
- 主工程目录分别在哪里
- 后端/前端主入口文件
- 开发环境与生产环境的差异
- 哪些目录是历史归档，不能当主链路

示例：

```md
## Repo Facts
- 当前主仓库根目录是 `phase1-framework/`，不是外层容器目录。
- 主工程目录：`backend/`、`frontend/`、`deploy/`、`docs/`
- `_archive/` 不参与主运行链路。
- 后端主入口：`backend/cmd/server/main.go`
```

### 2.2 高风险约束

优先写“做错会出事”的规则，例如：

- 不要用破坏性 Git 命令
- 不要覆盖用户已有改动
- 不要误改线上数据库结构
- 涉及线上操作前先确认机器和目录
- 环境变量变更后不能只 `restart`，需要重建容器

示例：

```md
## Safety Rules
- 不要执行 `git reset --hard`、`git checkout --`
- 不要直接改线上数据库结构
- 涉及线上配置变更时，先记录原值
```

### 2.3 默认工作流

告诉代理默认应该按什么顺序做，而不是让它自由发挥。

示例：

```md
## Workflow
1. 先定位实际代码入口、调用链和配置。
2. 做最小必要修改。
3. 修改后执行与改动直接相关的验证。
4. 回复时先给结论，再给证据和剩余风险。
```

### 2.4 项目特有踩坑点

这是 `AGENTS.md` 价值最高的部分。

应优先记录：

- 只有这个仓库才会遇到的坑
- 最近几次真实踩过的坑
- 不是读代码几分钟就能立刻看出来的事实

示例：

- 开发环境默认使用宿主机 Ollama
- 线上环境 Ollama 通过 Docker 部署
- `deploy/ollama/` 是线上模板，不是开发入口
- `docker-compose restart` 不会重建容器
- `deploy-update.sh backend` 会把本地编译二进制覆盖到线上容器

## 3. 不适合写进 AGENTS.md 的内容

下面这些通常不值得放进去：

- 大段背景介绍
- 重复 README 的安装教程
- 一般性的“写高质量代码”“保持可读性”
- 会频繁过期的具体业务状态
- 过细的实现细节和大段样例代码

判断标准很简单：

- 如果这条信息不能明显减少误判，就不该进 `AGENTS.md`

## 4. 推荐结构

一个实用版本通常控制在 6 到 8 个小节内。

推荐结构：

```md
# AGENTS.md

## Purpose
## Repo Facts
## Workflow
## Safety Rules
## Project-Specific Notes
## Online Notes
## Validation Defaults
## Output Style
```

这套结构的优点是：

- 足够短
- 信息层次清楚
- 便于长期维护

## 5. 编写原则

### 5.1 写事实，不写口号

弱表达：

- 请保持良好代码质量
- 注意系统稳定性

强表达：

- 当前主仓库根目录是 `phase1-framework/`
- 线上 `.env` 位于 `/opt/ops-platform/deploy/.env`
- 环境变量变更后需要 `docker-compose up -d --no-deps --force-recreate`

### 5.2 写默认动作，不写空泛建议

弱表达：

- 修改后请适当验证

强表达：

- 后端改动后优先检查路由、日志和配置是否实际生效
- 涉及 assistant 改动时，至少核对模型配置和 `assistant_messages.model_name`

### 5.3 写仓库特性，不写通用废话

有价值：

- 开发环境默认使用宿主机 Ollama
- 前端 Axios 拦截器返回 `response.data`

价值低：

- 代码应尽量模块化
- 注意安全性和性能

### 5.4 控制篇幅

如果一份 `AGENTS.md` 太长，代理不会更听话，只会更难抓重点。

建议：

- 优先压缩到 60 行上下
- 真正高价值的规则尽量放前面
- 同类信息合并，不重复说

## 6. 维护方式

`AGENTS.md` 不应该一次写完就不动。

更好的维护方式是：

- 每次出现一次真实误判，就补一条规则
- 每次发现某段规则长期没价值，就删掉
- 每次环境切换或主链路变化，就更新相关事实

适合补进 `AGENTS.md` 的来源包括：

- 最近一次线上事故复盘
- 最近一次部署踩坑
- 最近一次代理误判
- 最近一次路径或入口识别错误

## 7. 面向当前仓库的建议

如果继续维护当前仓库的 `AGENTS.md`，建议长期保留这些内容：

- 主仓库实际根目录不是外层 `ops-platform/`
- 开发环境默认使用宿主机 Ollama
- 线上环境 Ollama 使用 Docker 部署
- `deploy/ollama/` 是线上部署模板
- 线上环境变量文件位置
- `docker-compose restart` 不会重建容器
- `deploy-update.sh backend` 与容器重建的差异
- assistant 主链路是 `/api/assistant/*`
- 聚合打包相关接口在 `/api/deploy/*`

## 8. 一版可复用模板

```md
# AGENTS.md

## Purpose
- 默认先理解现有代码和部署方式，再做修改。
- 优先解决实际问题，采用最小必要改动。
- 修改后必须验证；无法验证时明确说明缺口。

## Repo Facts
- 当前主仓库根目录是 `phase1-framework/`。
- 主工程目录：`backend/`、`frontend/`、`deploy/`、`docs/`
- `_archive/` 不参与主运行链路。

## Workflow
1. 先定位实际入口、调用链和配置。
2. 优先阅读主干文件，不从归档目录开始。
3. 做最小必要修改。
4. 修改后执行直接相关的验证。

## Safety Rules
- 不要执行破坏性 Git 命令。
- 不要覆盖用户已有改动。
- 不要直接改线上数据库结构。

## Project-Specific Notes
- 开发环境默认使用宿主机 Ollama。
- 线上环境 Ollama 使用 Docker 部署。
- `deploy/ollama/` 是线上部署模板目录。
- `docker-compose restart` 不会重建容器。

## Validation Defaults
- 后端改动优先检查路由、日志、配置是否生效。
- 涉及 assistant 改动时，至少核对模型配置和实际落库模型名。
```

## 9. 一句话原则

`AGENTS.md` 应该像“执行规则”而不是“项目介绍”。

优先写：

- 会影响判断的事实
- 会减少返工的约束
- 会直接改变代理行为的默认流程

少写：

- 背景故事
- 通用原则
- 可以从代码里几分钟就看出来的内容

## 参考来源

- OpenAI Codex Best Practices: <https://developers.openai.com/codex/learn/best-practices/?utm_source=chatgpt.com>
