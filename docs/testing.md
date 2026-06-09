# 测试说明

## 自动化测试

### CI 流水线

每次提交和 PR 都会通过 GitHub Actions 自动运行：

- **后端**: `go vet` · `go build` · `go test ./... -short`
- **前端**: `npx tsc --noEmit` · `npm run build`

详见 [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)

### 本地运行测试

```bash
# 后端单元测试
cd backend
go test ./... -short

# 指定模块测试
go test ./internal/security/...
go test ./internal/assistant/...

# 前端类型检查
cd frontend
npx tsc --noEmit

# 前端构建
npm run build
```

## 端到端验收测试

项目包含基于 Playwright 的 UI 冒烟测试脚本：

```bash
cd frontend

# 全量冒烟测试
npm run acceptance:smoke:full

# 按模块测试
npm run acceptance:smoke:core       # 核心功能
npm run acceptance:smoke:navigation # 导航验证
npm run acceptance:smoke:pages      # 页面加载
npm run acceptance:smoke:security-web  # 安全 Web 扫描
npm run acceptance:smoke:fim-error  # FIM 错误场景
```

> 注意：端到端测试需要运行中的开发环境（`docker compose -f deploy/docker-compose.dev.yml up -d`）

## 测试环境

| 组件 | 版本 |
|------|------|
| Go | 1.24 |
| Node.js | 20 |
| MySQL | 8.0 |
| Redis | 7.4 |
| Playwright | 1.58 |
