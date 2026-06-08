# 安装包聚合功能设计文档

## 功能概述
安装包聚合功能是一个用于管理和追踪软件包聚合构建过程的系统。该功能允许用户创建聚合任务，监控构建状态，并提供构建产物的下载链接。

## 系统架构
```
前端界面 -> API网关 -> 后端服务 -> Jenkins -> 构建产物存储
```

## 核心功能模块

### 1. 任务管理模块
- 创建聚合任务
- 监控任务状态
- 任务历史记录

### 2. 状态同步模块
- 与Jenkins状态同步
- 实时状态更新
- 进度追踪

### 3. 下载管理模块
- 生成唯一的下载链接
- 下载链接有效期管理
- 构建产物版本管理

## 数据模型

### AggregatedHistory 模型
```go
type AggregatedHistory struct {
    ID              uint       // 主键
    ProjectName     string     // 项目名称
    Environment     string     // 环境/Tag名称
    Status          string     // 状态 (pending, queued, running, completed, failed)
    Progress        int        // 进度百分比
    StartTime       *time.Time // 开始时间
    EndTime         *time.Time // 结束时间
    DownloadURL     *string    // 下载地址
    Operator        string     // 操作人用户名
    OperatorName    string     // 操作人姓名
    JenkinsJobName  string     // Jenkins任务名称
    JenkinsBuildNum *int       // Jenkins构建编号
    JenkinsQueueID  *int64     // Jenkins队列ID
    JenkinsConsoleURL *string  // Jenkins控制台URL
    ErrorMessage    *string    // 错误信息
    CreatedAt       time.Time  // 创建时间
    UpdatedAt       time.Time  // 更新时间
}
```

## API接口设计

### 获取聚合历史列表
```
GET /cmdb/aggregated-histories
参数：page, limit, project_name, environment, operator, status
```

### 获取单个聚合历史
```
GET /cmdb/aggregated-histories/{id}
```

### 刷新聚合历史状态
```
GET /cmdb/aggregated-histories/{id}/status
```

### 获取控制台日志
```
GET /cmdb/aggregated-histories/{id}/console-log
```

## 业务流程

### 1. 任务创建流程
1. 用户提交聚合任务请求
2. 后端验证参数
3. 创建任务记录
4. 触发Jenkins构建
5. 返回任务ID

### 2. 状态同步流程
1. 前端轮询或后端定时同步
2. 从Jenkins获取构建状态
3. 更新本地数据库记录
4. 更新UI显示

### 3. 下载链接生成流程
1. 构建成功后生成时间戳
2. 创建标准格式的下载链接
3. 存储到数据库
4. 提供给用户

## 安全考虑
- 认证授权：确保只有授权用户才能操作
- 参数验证：防止恶意输入
- URL安全性：验证下载链接的合法性

## 性能优化
- 数据库索引优化
- 状态缓存机制
- 分页查询优化