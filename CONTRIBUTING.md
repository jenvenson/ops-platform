# Contributing to OPS Platform

感谢你对 OPS Platform 的关注！

## 开发环境

```bash
git clone git@github.com:jenvenson/ops-platform.git
cd ops-platform/deploy
cp .env.example .env
# 编辑 .env，设置 DB_PASSWORD、REDIS_PASSWORD、JWT_SECRET
docker compose -f docker-compose.dev.yml up -d
```

## 提交规范

- 一个 commit 做一件事
- commit message 使用中文描述

## 代码风格

- **Go**: 遵循标准 Go 惯例，错误必须处理
- **TypeScript**: 优先 ESM，禁止 `any`（边界层例外）
- **前端组件**: 遵循 Ant Design 惯例，优先使用已有组件

## 提 Issue

- Bug 报告请附带复现步骤和日志
- 功能建议请描述使用场景

## Pull Request

1. Fork 仓库
2. 创建 feature 分支
3. 确保本地测试通过
4. 提交 PR，描述变更内容

## 项目结构

详见 [README.md](README.md)
