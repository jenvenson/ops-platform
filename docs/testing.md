# 安装包聚合功能测试文档

## 功能测试清单

### 1. 安装包聚合功能测试

#### 1.1 操作人姓名显示测试
- [x] 验证操作人显示为真实姓名而非用户名
- [x] 验证OperatorName字段在数据库中正确存储
- [x] 验证前端界面正确显示OperatorName字段

#### 1.2 下载地址格式测试
- [x] 验证下载地址格式为 `http://10.99.99.65:8888/aggregation/{timestamp}.tar`
- [x] 验证timestamp为Unix时间戳格式
- [x] 验证下载地址在构建成功后正确生成
- [x] 验证多个构建任务生成不同的下载地址

#### 1.3 Jenkins日志查看测试
- [x] 验证"查看日志"按钮正常工作
- [x] 验证日志内容正确从Jenkins获取
- [x] 验证日志弹窗正确显示
- [x] 验证日志加载过程有适当的状态提示

### 2. Jenkins视图批量复制功能测试

#### 2.1 智能替换推理测试
- [x] 验证从fat-150-V2.5.1到fat-160-V2.5.1的视图复制
- [x] 验证fat150自动替换为fat160的配置参数
- [x] 验证不同命名模式的正确识别

#### 2.2 参数格式支持测试
- [x] 验证Jenkins Pipeline groovy脚本参数格式
- [x] 验证多种key-value格式的支持
- [x] 验证tag参数的正确替换

### 3. Consul流水线配置功能测试

#### 3.1 Tag替换功能测试
- [x] 验证多种key-value格式匹配
- [x] 验证Tag替换功能正常工作
- [x] 验证引号/无引号、冒号/等号分隔的支持

#### 3.2 Server替换功能测试
- [x] 验证Server替换功能正常工作
- [x] 验证各种格式的Server值匹配

## 性能测试

### 1. 并发测试
- [x] 验证多用户同时使用功能的稳定性
- [x] 验证大量历史记录的查询性能

### 2. 响应时间测试
- [x] API响应时间 < 2秒
- [x] 页面加载时间 < 3秒
- [x] 状态更新延迟 < 5秒

## 兼容性测试

### 1. 浏览器兼容性
- [x] Chrome最新版
- [x] Firefox最新版
- [x] Safari最新版

### 2. 设备兼容性
- [x] 桌面端显示
- [ ] 移动端显示（待优化）

## 安全测试

### 1. 输入验证测试
- [x] 验证所有用户输入都有适当的验证
- [x] 验证SQL注入防护
- [x] 验证XSS防护

### 2. 权限验证测试
- [x] 验证未授权用户无法执行操作
- [x] 验证API访问权限控制

## 回归测试

### 1. 现有功能验证
- [x] 验证其他Jenkins功能未受影响
- [x] 验证其他CMDB功能未受影响
- [x] 验证Consul功能未受影响

### 2. Web 扫描回归入口
- [x] 后端代码变更后执行 `cd .worktrees/phase1-framework && bash deploy/dev.sh backend`
- [x] 前端需要刷新开发容器时执行 `cd .worktrees/phase1-framework && bash deploy/dev.sh frontend`
- [x] 安全扫描后端单测：`cd .worktrees/phase1-framework/backend && GOCACHE=$(pwd)/.gocache go test ./internal/security/...`
- [x] 前端构建：`cd .worktrees/phase1-framework/frontend && npm run build`
- [x] 登录后 Web 扫描回归：`cd .worktrees/phase1-framework/frontend && npm run acceptance:smoke:security-web`
- [x] 回归脚本验证口径：创建真实任务并校验 `latest_run`、`targets`、`evidences`、`occurrences`、`vulnerabilities`

## 测试结果总结

### 已通过测试
- 操作人姓名显示功能
- 下载地址格式生成
- Jenkins日志查看功能
- 智能替换推理功能
- 参数格式支持功能
- Consul配置替换功能
- 基础性能指标
- 安全性验证

### 待优化项目
- 移动端UI适配
- 高并发场景优化
- 错误处理机制增强

## 测试环境
- 后端：Golang 1.24
- 前端：React 18, TypeScript
- 数据库：MySQL 8.0
- 容器：Docker
- Jenkins：最新版本
