# 问题修复跟踪

## 高优先级问题

### 1. 前端类型安全缺失 ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `frontend/src/pages/deploy/AggregatedHistoryPage.tsx:1` |
| **问题** | 使用 `// @ts-nocheck` 禁用 TypeScript 类型检查 |
| **修复** | 移除该注释，正确处理 API 返回类型 |
| **验证** | TypeScript 编译通过 |

---

### 2. 硬编码默认 Jenkins URL ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `backend/internal/cmdb/aggregate_handler.go` |
| **问题** | 多处硬编码 `http://your-jenkins-server` |
| **修复** | 添加 `getJenkinsURL()` 方法，使用配置值，移除 fallback |
| **验证** | 代码中无 `js.zbnsec.com` 字符串 |

---

### 3. 后端轮询机制效率低下 ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `backend/internal/cmdb/aggregated_history_handler.go` |
| **问题** | 固定 10 秒间隔，无 context 控制，无法优雅退出，无 WaitGroup 跟踪 |
| **修复** | 添加 `pollInterval` 参数支持可配置间隔；添加 `stopChan` 和 `sync.WaitGroup` 支持优雅关闭；新增 `StopAggregatedHistoryRefresh()` 函数；`server.go` 关闭时调用停止函数 |
| **验证** | 服务关闭时轮询 goroutine 能正确退出 |

---

### 4. 魔法数字和硬编码字符串 ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `backend/internal/cmdb/aggregate_handler.go` |
| **问题** | `plugin/fscr-aggregation/` 在 6 处重复硬编码 |
| **修复** | 提取为常量 `consulAggregationPath` |
| **验证** | 所有引用改为使用常量 |

---

## 中优先级问题

### 5. 代码重复 📋 待处理

| 项目 | 内容 |
|------|------|
| **位置** | `aggregate_handler.go` vs `aggregated_history_handler.go` |
| **问题** | Jenkins 客户端初始化、状态检查逻辑重复 |
| **建议** | 抽取公共 service 层 |

---

### 6. 命令注入风险 ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `backend/internal/security/fim_service.go:1229-1310` |
| **问题** | Path/Glob 输入未验证，可能包含危险字符；`shellQuote` 仅处理单引号 |
| **修复** | 添加 `validateWatchPathInput()` 函数进行安全验证；Path 只允许字母、数字、`_/.-`，必须绝对路径，禁止 `..` 遍历；Glob 只允许字母、数字、`_.*?-`；简化命令构造 |
| **验证** | 非法路径输入被正确拒绝 |

> 注意：`shellQuote` 函数变为未使用，可后续清理

---

### 7. 敏感信息日志 ✅ 已修复

| 项目 | 内容 |
|------|------|
| **位置** | `aggregate_handler.go:649, 661, 763` |
| **问题** | 日志输出包含用户名、URL 等敏感信息 |
| **修复** | 移除日志中的敏感字段（URL、Username），只保留任务相关信息 |
| **验证** | 代码中无敏感信息日志输出 |

---

## 低优先级问题

### 8. 职责不清晰 📋 待处理

| 项目 | 内容 |
|------|------|
| **位置** | `aggregate_handler.go` vs `aggregated_history_handler.go` |
| **问题** | 两个文件都处理聚合打包，边界模糊 |

---

### 9. 紧耦合 📋 待处理

| 项目 | 内容 |
|------|------|
| **位置** | `aggregate_handler.go:713-779` |
| **问题** | 业务逻辑与数据访问层耦合 |

---

### 10. 异常处理不一致 📋 待处理

| 项目 | 内容 |
|------|------|
| **位置** | 多处 |
| **问题** | 错误处理策略不统一 |

---

### 11. 代码复杂度过高 📋 待处理

| 项目 | 内容 |
|------|------|
| **位置** | `fim_service.go:715-819` |
| **问题** | 函数 104 行，职责过多 |

---

## 修复进度汇总

| 优先级 | 总数 | 已修复 | 待处理 |
|--------|------|--------|--------|
| 高 | 4 | 4 | 0 |
| 中 | 3 | 2 | 1 |
| 低 | 4 | 0 | 4 |
| **总计** | **11** | **6** | **5** |
