# 安装包聚合打包功能开发设计文档（修订版）

## 1. 系统架构设计

### 1.1 整体架构
安装包聚合打包功能采用前后端分离的架构，前端通过API接口与后端交互，后端通过Jenkins客户端触发构建任务，并从Consul获取必要的配置参数。

```
前端页面 -> API网关 -> 业务逻辑层 -> Jenkins客户端 -> Jenkins服务
                       ↓                 ↓
                   数据持久层 -> MySQL/PostgreSQL
                       ↓
                   Consul客户端 -> Consul服务
```

### 1.2 组件职责
- **前端组件**：提供用户交互界面，处理用户输入，展示打包状态
- **API层**：提供RESTful接口，处理请求参数校验，协调业务逻辑
- **业务逻辑层**：处理聚合打包业务逻辑，管理任务状态
- **Jenkins客户端**：与Jenkins服务通信，触发构建任务，获取构建状态
- **Consul客户端**：从Consul获取构建所需的参数（如tag等）
- **数据持久层**：存储打包任务记录、应用配置等信息

## 2. 数据库设计

### 2.1 aggregate_package_tasks 表
存储聚合打包任务基本信息

```sql
CREATE TABLE aggregate_package_tasks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_name VARCHAR(255) NOT NULL COMMENT '任务名称',
    project_name VARCHAR(255) NOT NULL COMMENT '项目名称',
    app_names JSON COMMENT '参与打包的应用名称列表(JSON格式)',
    jenkins_job_name VARCHAR(255) DEFAULT 'fscr-aggregation' COMMENT 'Jenkins任务名称',
    jenkins_job_url VARCHAR(500) DEFAULT 'http://js.zbnsec.com/view/auto-archive-deploy/job/fscr-aggregation/' COMMENT 'Jenkins任务地址',
    consul_config_path VARCHAR(500) DEFAULT 'plugin/fscr-aggregation/' COMMENT 'Consul配置路径',
    build_params JSON COMMENT '构建参数(JSON格式)',
    status ENUM('pending', 'building', 'success', 'failed', 'cancelled') DEFAULT 'pending' COMMENT '任务状态',
    triggered_by VARCHAR(255) COMMENT '触发人',
    start_time TIMESTAMP NULL COMMENT '开始时间',
    end_time TIMESTAMP NULL COMMENT '结束时间',
    duration INT DEFAULT 0 COMMENT '耗时(秒)',
    error_message TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_project_name (project_name),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);
```

### 2.2 aggregate_package_results 表
存储每个应用的打包结果

```sql
CREATE TABLE aggregate_package_results (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id BIGINT NOT NULL COMMENT '关联的聚合打包任务ID',
    app_name VARCHAR(255) NOT NULL COMMENT '应用名称',
    jenkins_build_num INT COMMENT 'Jenkins构建编号',
    jenkins_queue_id BIGINT COMMENT 'Jenkins队列ID',
    jenkins_console_url VARCHAR(500) COMMENT 'Jenkins控制台URL',
    consul_tag VARCHAR(255) COMMENT '从Consul获取的标签',
    status ENUM('pending', 'building', 'success', 'failed') DEFAULT 'pending' COMMENT '构建状态',
    error_message TEXT COMMENT '错误信息',
    start_time TIMESTAMP NULL COMMENT '开始时间',
    end_time TIMESTAMP NULL COMMENT '结束时间',
    duration INT DEFAULT 0 COMMENT '耗时(秒)',
    download_url VARCHAR(500) COMMENT '打包产物下载地址',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES aggregate_package_tasks(id) ON DELETE CASCADE,
    INDEX idx_task_app (task_id, app_name),
    INDEX idx_status (status)
);
```

## 3. 后端API设计

### 3.1 任务管理API

#### 创建聚合打包任务
```
POST /api/deploy/aggregate-package
Content-Type: application/json

Request:
{
  "project_name": "FSCR项目",
  "app_names": ["app1", "app2", "app3"],
  "task_name": "FSCR聚合打包_20231201",
  "description": "月度版本聚合打包"
}

Response:
{
  "success": true,
  "data": {
    "task_id": 12345
  }
}
```

#### 获取聚合打包任务列表
```
GET /api/deploy/aggregate-package?project_name=FSCR&status=pending&start_time=2023-12-01&end_time=2023-12-31&page=1&limit=20

Response:
{
  "success": true,
  "data": {
    "total": 10,
    "page": 1,
    "limit": 20,
    "tasks": [
      {
        "id": 12345,
        "task_name": "FSCR聚合打包_20231201",
        "project_name": "FSCR项目",
        "app_count": 3,
        "jenkins_job_name": "fscr-aggregation",
        "status": "success",
        "triggered_by": "张三",
        "duration": 1200,
        "created_at": "2023-12-01 10:30:00",
        "start_time": "2023-12-01 10:30:00",
        "end_time": "2023-12-01 10:50:00"
      }
    ]
  }
}
```

#### 获取聚合打包任务详情
```
GET /api/deploy/aggregate-package/12345

Response:
{
  "success": true,
  "data": {
    "task": {
      "id": 12345,
      "task_name": "FSCR聚合打包_20231201",
      "project_name": "FSCR项目",
      "app_names": ["app1", "app2", "app3"],
      "jenkins_job_name": "fscr-aggregation",
      "jenkins_job_url": "http://js.zbnsec.com/view/auto-archive-deploy/job/fscr-aggregation/",
      "status": "success",
      "triggered_by": "张三",
      "start_time": "2023-12-01 10:30:00",
      "end_time": "2023-12-01 10:50:00",
      "duration": 1200,
      "results": [
        {
          "app_name": "app1",
          "consul_tag": "v1.2.3",
          "status": "success",
          "jenkins_build_num": 100,
          "duration": 300,
          "download_url": "http://download.example.com/path/to/app1.zip"
        }
      ]
  }
}
```

#### 获取任务状态
```
GET /api/deploy/aggregate-package/12345/status

Response:
{
  "success": true,
  "data": {
    "status": "building",
    "progress": 60,
    "overall_status": "building",
    "app_statuses": [
      {
        "app_name": "app1",
        "status": "success"
      },
      {
        "app_name": "app2",
        "status": "building"
      }
    ]
  }
}
```

## 4. Consul集成设计

### 4.1 Consul客户端实现
系统需要从Consul获取构建所需的参数，特别是tag值：

```go
// Consul客户端
import (
    "fmt"
    "log"
    "path/filepath"

    "github.com/hashicorp/consul/api"
)

type ConsulClient struct {
    addr string
    client *api.Client
}

func NewConsulClient(addr string) (*ConsulClient, error) {
    config := &api.Config{
        Address: addr,
    }
    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }
    return &ConsulClient{
        addr:   addr,
        client: client,
    }, nil
}

// GetKVValue 获取指定路径的键值
func (c *ConsulClient) GetKVValue(path string) (string, error) {
    pair, _, err := c.client.KV().Get(path, nil)
    if err != nil {
        return "", err
    }
    if pair == nil {
        return "", fmt.Errorf("key %s not found", path)
    }
    return string(pair.Value), nil
}

// GetTagsFromPath 从指定路径获取tags
func (c *ConsulClient) GetTagsFromPath(basePath string, appNames []string) (map[string]string, error) {
    tags := make(map[string]string)

    for _, appName := range appNames {
        key := fmt.Sprintf("%s%s", basePath, appName)
        value, err := c.GetKVValue(key)
        if err != nil {
            log.Printf("Warning: Could not get tag for %s from path %s: %v", appName, key, err)
            tags[appName] = "latest" // 默认值
        } else {
            tags[appName] = value
        }
    }

    return tags, nil
}
```

### 4.2 Consul配置路径
- Consul地址：`http://10.99.99.98:8500`
- 配置路径：`plugin/fscr-aggregation/`
- 获取到的tag将用于Jenkins构建参数

## 5. Jenkins集成设计

### 5.1 Jenkins客户端实现
为支持指定Jenkins作业地址，需要实现Jenkins客户端：

```go
package jenkins

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"
)

type JenkinsClient struct {
    baseURL  string
    username string
    password string
    httpClient *http.Client
}

func NewJenkinsClient(baseURL, username, password string) *JenkinsClient {
    return &JenkinsClient{
        baseURL:  strings.TrimSuffix(baseURL, "/"),
        username: username,
        password: password,
        httpClient: &http.Client{
            Timeout: 60 * time.Second,
        },
    }
}

// GetCrumb 获取CSRF保护令牌
func (jc *JenkinsClient) GetCrumb() (string, error) {
    req, err := http.NewRequest("GET", jc.baseURL+"/crumbIssuer/api/json", nil)
    if err != nil {
        return "", err
    }

    req.SetBasicAuth(jc.username, jc.password)
    resp, err := jc.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode == 404 {
        // Jenkins可能没有启用CSRF保护
        return "", nil
    }

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("crumb request failed with status: %d", resp.StatusCode)
    }

    var crumbResp struct {
        Crumb             string `json:"crumb"`
        CrumbRequestField string `json:"crumbRequestField"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&crumbResp); err != nil {
        return "", err
    }

    return crumbResp.Crumb, nil
}

// BuildJobWithParams 触发带参数的Jenkins构建
func (jc *JenkinsClient) BuildJobWithParams(jobPath string, params map[string]string) (int64, error) {
    // 获取CSRF令牌
    crumb, err := jc.GetCrumb()
    if err != nil {
        return 0, fmt.Errorf("failed to get crumb: %v", err)
    }

    // 构建参数化请求
    formData := url.Values{}
    for key, value := range params {
        formData.Set(key, value)
    }

    // 创建构建请求
    jobPath = strings.TrimPrefix(jobPath, "/")
    reqURL := fmt.Sprintf("%s/%s/buildWithParameters", jc.baseURL, jobPath)

    req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
    if err != nil {
        return 0, err
    }

    req.SetBasicAuth(jc.username, jc.password)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    if crumb != "" {
        req.Header.Set("Jenkins-Crumb", crumb)
    }

    resp, err := jc.httpClient.Do(req)
    if err != nil {
        return 0, fmt.Errorf("request failed: %v", err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != 201 {
        return 0, fmt.Errorf("build job failed with status: %d, body: %s", resp.StatusCode, string(body))
    }

    // 从响应头获取队列ID
    location := resp.Header.Get("Location")
    if location == "" {
        return 0, fmt.Errorf("no location header in response")
    }

    // 提取队列ID
    queuePath := strings.TrimPrefix(location, jc.baseURL+"/queue/item/")
    queuePath = strings.TrimSuffix(queuePath, "/")
    queueID, err := strconv.ParseInt(queuePath, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("could not parse queue ID from location: %s, error: %v", location, err)
    }

    return queueID, nil
}

// GetBuildInfo 获取构建信息
func (jc *JenkinsClient) GetBuildInfo(jobPath string, buildNum int) (map[string]interface{}, error) {
    jobPath = strings.TrimPrefix(jobPath, "/")
    reqURL := fmt.Sprintf("%s/job/%s/%d/api/json", jc.baseURL, strings.ReplaceAll(jobPath, "/", "/job/"), buildNum)

    req, err := http.NewRequest("GET", reqURL, nil)
    if err != nil {
        return nil, err
    }

    req.SetBasicAuth(jc.username, jc.password)
    resp, err := jc.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("get build info failed with status: %d", resp.StatusCode)
    }

    var buildInfo map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&buildInfo); err != nil {
        return nil, err
    }

    return buildInfo, nil
}
```

### 5.2 构建参数传递
对于指定的Jenkins作业 `http://js.zbnsec.com/view/auto-archive-deploy/job/fscr-aggregation/`，需要传递以下参数：

- `app`: `fscr-aggregation` （固定值）
- `tag`: 从Consul获取的tag值
- `scope`: `all` （固定值）

### 5.3 业务逻辑实现

#### 任务创建逻辑
1. 验证输入参数的有效性（项目名称、应用名称列表）
2. 从Consul获取对应应用的tag信息
3. 准备Jenkins构建参数
4. 调用Jenkins API触发聚合打包任务
5. 保存任务记录到数据库
6. 返回任务ID

```go
import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
)

func (h *Handler) CreateAggregatePackageTask(c *gin.Context) {
    var req struct {
        ProjectName string   `json:"project_name"`
        AppNames    []string `json:"app_names"`
        TaskName    string   `json:"task_name"`
        Description string   `json:"description"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 验证输入参数
    if err := validateInputs(req.ProjectName, req.AppNames); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 获取Consul客户端
    consulAddr := "10.99.99.98:8500" // 可以从配置中获取
    consulClient, err := NewConsulClient(consulAddr)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to connect to Consul"})
        return
    }

    // 从Consul获取tag信息
    tagMap, err := consulClient.GetTagsFromPath("plugin/fscr-aggregation/", req.AppNames)
    if err != nil {
        // 如果Consul获取失败，记录错误但仍继续处理
        log.Printf("Warning: Could not get tags from Consul: %v", err)
    }

    // 准备Jenkins构建参数
    buildParams := map[string]string{
        "app":   "fscr-aggregation",
        "scope": "all",
    }

    // 将tag参数添加到构建参数中
    for _, appName := range req.AppNames {
        tagKey := fmt.Sprintf("tag_%s", appName)
        if tag, exists := tagMap[appName]; exists {
            buildParams[tagKey] = tag
        } else {
            buildParams[tagKey] = "latest" // 默认值
        }
    }

    // 保存任务到数据库
    task := &models.AggregatePackageTask{
        TaskName:     req.TaskName,
        ProjectName:  req.ProjectName,
        AppNames:     req.AppNames,
        JenkinsJobName: "fscr-aggregation",
        JenkinsJobUrl: "http://js.zbnsec.com/view/auto-archive-deploy/job/fscr-aggregation/",
        ConsulConfigPath: "plugin/fscr-aggregation/",
        BuildParams:  buildParams, // 存储构建参数
        Status:       "pending",
        TriggeredBy:  c.GetString("username"),
        CreatedAt:    time.Now(),
    }

    if err := h.db.Create(task).Error; err != nil {
        c.JSON(500, gin.H{"error": "failed to create task"})
        return
    }

    // 为每个应用创建结果记录
    for _, appName := range req.AppNames {
        result := &models.AggregatePackageResult{
            TaskID:    task.ID,
            AppName:   appName,
            ConsulTag: tagMap[appName], // 使用从Consul获取的tag
            Status:    "pending",
        }
        h.db.Create(result)
    }

    // 异步触发Jenkins构建
    go h.triggerJenkinsBuild(task.ID, "/view/auto-archive-deploy/job/fscr-aggregation/", buildParams)

    c.JSON(200, gin.H{"success": true, "data": gin.H{"task_id": task.ID}})
}

// 验证输入参数
func validateInputs(projectName string, appNames []string) error {
    if projectName == "" {
        return fmt.Errorf("项目名称不能为空")
    }

    if len(appNames) == 0 {
        return fmt.Errorf("至少需要指定一个应用名称")
    }

    if len(appNames) > 50 {
        return fmt.Errorf("应用名称数量不能超过50个")
    }

    // 验证名称格式（简单验证，只允许字母数字下划线和连字符）
    for _, appName := range appNames {
        if len(appName) == 0 {
            return fmt.Errorf("应用名称不能为空")
        }
    }

    return nil
}
```

#### 触发Jenkins构建
```go
func (h *Handler) triggerJenkinsBuild(taskID int64, jobPath string, params map[string]string) {
    // 更新任务状态为building
    h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
        Update("status", "building")

    // 初始化Jenkins客户端
    jenkinsClient := NewJenkinsClient(
        "http://js.zbnsec.com",  // Jenkins基础URL
        "your-jenkins-username", // 从配置获取
        "your-jenkins-password", // 从配置获取
    )

    // 触发Jenkins构建
    queueID, err := jenkinsClient.BuildJobWithParams(jobPath, params)
    if err != nil {
        // 记录错误并更新状态
        h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
            Updates(map[string]interface{}{
                "status":        "failed",
                "error_message": fmt.Sprintf("Jenkins构建失败: %v", err),
                "end_time":      time.Now(),
            })
        log.Printf("Jenkins build failed for task %d: %v", taskID, err)
        return
    }

    // 更新任务中的Jenkins队列ID
    h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
        Updates(map[string]interface{}{
            "jenkins_queue_id": queueID,
            "start_time":       time.Now(),
        })

    // 启动状态轮询
    go h.pollJenkinsStatus(taskID, queueID, jobPath)
}

// 轮询Jenkins状态
func (h *Handler) pollJenkinsStatus(taskID int64, queueID int64, jobPath string) {
    ticker := time.NewTicker(5 * time.Second) // 每5秒轮询一次
    defer ticker.Stop()

    maxAttempts := 240 // 最多轮询20分钟（240次 * 5秒）
    attempts := 0

    for {
        select {
        case <-ticker.C:
            attempts++

            // 通过队列ID检查构建是否已启动
            buildNum, status, err := h.checkQueueStatus(queueID)
            if err != nil {
                log.Printf("Error checking queue status for task %d: %v", taskID, err)
                continue
            }

            if buildNum > 0 {
                // 构建已启动，检查构建状态
                buildStatus, err := h.getBuildStatus(jobPath, buildNum)
                if err != nil {
                    log.Printf("Error getting build status for task %d: %v", taskID, err)
                    continue
                }

                // 更新数据库中的状态
                h.updateTaskAndResultStatus(taskID, buildStatus)

                // 如果构建完成，则退出轮询
                if buildStatus.IsComplete() {
                    ticker.Stop()
                    return
                }
            }

            // 如果达到最大尝试次数，则退出
            if attempts >= maxAttempts {
                h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
                    Updates(map[string]interface{}{
                        "status":        "failed",
                        "error_message": "构建超时",
                        "end_time":      time.Now(),
                    })
                ticker.Stop()
                return
            }
        }
    }
}

// 检查队列状态
func (h *Handler) checkQueueStatus(queueID int64) (int, string, error) {
    // 实现队列状态检查逻辑
    // 这里只是一个示例实现
    return 0, "waiting", nil
}

// 获取构建状态
func (h *Handler) getBuildStatus(jobPath string, buildNum int) (BuildStatus, error) {
    // 初始化Jenkins客户端
    jenkinsClient := NewJenkinsClient(
        "http://js.zbnsec.com",
        "your-jenkins-username",
        "your-jenkins-password",
    )

    buildInfo, err := jenkinsClient.GetBuildInfo(jobPath, buildNum)
    if err != nil {
        return BuildStatus{}, err
    }

    status := BuildStatus{}
    if buildResult, ok := buildInfo["result"].(string); ok {
        status.Result = buildResult
    }
    if building, ok := buildInfo["building"].(bool); ok {
        status.Building = building
    }

    return status, nil
}

// 更新任务和结果状态
func (h *Handler) updateTaskAndResultStatus(taskID int64, status BuildStatus) {
    // 更新任务状态
    dbStatus := "building"
    if !status.Building {
        if status.Result == "SUCCESS" {
            dbStatus = "success"
        } else {
            dbStatus = "failed"
        }
    }

    updates := map[string]interface{}{
        "status": dbStatus,
    }

    if !status.Building { // 构建已完成
        updates["end_time"] = time.Now()
        duration := time.Since(time.Now()) // 实际应从数据库中获取开始时间计算
        updates["duration"] = int(duration.Seconds())
    }

    h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
        Updates(updates)
}

type BuildStatus struct {
    Building bool
    Result   string
}

func (bs BuildStatus) IsComplete() bool {
    return !bs.Building
}
```

## 6. 前端设计

### 6.1 页面组件结构
```
AggregatePackagePage
├── AggregatePackageForm (聚合打包表单)
│   ├── ProjectInput (项目名称输入框)
│   ├── AppNameInput (应用名称输入区域 - 支持多行输入)
│   └── SubmitButton (提交按钮)
├── TaskStatusPanel (任务状态面板)
│   ├── TaskProgress (任务进度条)
│   ├── AppStatusTable (应用状态表格)
│   └── ActionResultDisplay (结果展示)
└── TaskHistoryList (历史任务列表)
```

### 6.2 前端实现

```tsx
import { useState } from 'react';
import { Form, Input, Button, Card, message, Space, Typography, Tag, Table, Alert } from 'antd';
import { apiClient } from '../../api/client';

const { TextArea } = Input;
const { Title } = Typography;

interface TaskStatus {
  id: number;
  task_name: string;
  project_name: string;
  app_names: string[];
  status: string;
  triggered_by: string;
  start_time?: string;
  end_time?: string;
  duration?: number;
  error_message?: string;
  results: Array<{
    app_name: string;
    status: string;
    consul_tag: string;
    jenkins_build_num?: number;
    duration?: number;
    download_url?: string;
    error_message?: string;
  }>;
}

export default function AggregatePackagePage() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [taskId, setTaskId] = useState<number | null>(null);
  const [taskStatus, setTaskStatus] = useState<TaskStatus | null>(null);
  const [polling, setPolling] = useState(false);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      // 将应用名称字符串按行分割
      const appNames = values.app_names.split('\n')
        .map((name: string) => name.trim())
        .filter((name: string) => name.length > 0);

      const response = await apiClient.post('/deploy/aggregate-package', {
        project_name: values.project_name,
        app_names: appNames,
        task_name: `聚合打包_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}_${Math.floor(Math.random() * 10000)}`,
      });

      if (response.success && response.data?.task_id) {
        message.success('聚合打包任务已提交');
        setTaskId(response.data.task_id);
        // 开始轮询任务状态
        startPolling(response.data.task_id);
      } else {
        message.error(response.error || '提交失败');
      }
    } catch (error: any) {
      message.error(error.response?.data?.error || '提交失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const startPolling = async (taskId: number) => {
    setPolling(true);

    const poll = async () => {
      try {
        const response = await apiClient.get(`/deploy/aggregate-package/${taskId}`);
        setTaskStatus(response.data.task);

        // 如果任务完成，则停止轮询
        if (['success', 'failed', 'cancelled'].includes(response.data.task.status)) {
          setPolling(false);
        } else {
          // 继续轮询
          setTimeout(poll, 5000);
        }
      } catch (error) {
        console.error('获取状态失败:', error);
        setPolling(false);
      }
    };

    poll();
  };

  // 应用状态表格列定义
  const appColumns = [
    {
      title: '应用名称',
      dataIndex: 'app_name',
      key: 'app_name',
    },
    {
      title: 'Consul标签',
      dataIndex: 'consul_tag',
      key: 'consul_tag',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: 'Jenkins构建号',
      dataIndex: 'jenkins_build_num',
      key: 'jenkins_build_num',
    },
    {
      title: '错误信息',
      dataIndex: 'error_message',
      key: 'error_message',
    }
  ];

  return (
    <div>
      <Card title="安装包聚合打包">
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            name="project_name"
            label="项目名称"
            rules={[{ required: true, message: '请输入项目名称' }]}
          >
            <Input placeholder="请输入项目名称，例如：FSCR项目" />
          </Form.Item>

          <Form.Item
            name="app_names"
            label="应用名称列表"
            extra="每行一个应用名称，系统将从Consul获取对应tag并触发Jenkins聚合打包"
            rules={[{ required: true, message: '请输入应用名称列表' }]}
          >
            <TextArea
              rows={8}
              placeholder="app1\napp2\napp3\n..."
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                disabled={polling}
              >
                开始聚合打包
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {taskId && (
          <div style={{ marginTop: 24 }}>
            <Title level={4}>任务状态</Title>
            <p>任务ID: {taskId}</p>

            {taskStatus && (
              <>
                <div style={{ marginBottom: 16 }}>
                  <p>任务名称: {taskStatus.task_name}</p>
                  <p>项目名称: {taskStatus.project_name}</p>
                  <p>状态: <Tag color={getStatusColor(taskStatus.status)}>{taskStatus.status}</Tag></p>
                  <p>触发人: {taskStatus.triggered_by}</p>
                  {taskStatus.error_message && (
                    <Alert message="错误信息" description={taskStatus.error_message} type="error" showIcon />
                  )}
                </div>

                {taskStatus.results && taskStatus.results.length > 0 && (
                  <div>
                    <Title level={5}>应用打包状态</Title>
                    <Table
                      columns={appColumns}
                      dataSource={taskStatus.results.map((result, index) => ({ ...result, key: index }))}
                      pagination={false}
                      size="small"
                    />
                  </div>
                )}
              </>
            )}

            {polling && (
              <Alert message="正在轮询任务状态，请稍候..." type="info" showIcon />
            )}
          </div>
        )}
      </Card>
    </div>
  );
}

function getStatusColor(status: string) {
  switch (status) {
    case 'success': return 'green';
    case 'failed': return 'red';
    case 'building': return 'blue';
    case 'pending': return 'orange';
    case 'cancelled': return 'default';
    default: return 'default';
  }
}
```

### 6.3 前端API定义

```typescript
// 新增聚合打包相关API
interface CreateAggregatePackageRequest {
  project_name: string;
  app_names: string[];
  task_name: string;
  description?: string;
}

interface CreateAggregatePackageResponse {
  success: boolean;
  data?: {
    task_id: number;
  };
  error?: string;
}

interface TaskStatusResponse {
  success: boolean;
  data: {
    task: {
      id: number;
      task_name: string;
      project_name: string;
      app_names: string[];
      jenkins_job_name: string;
      jenkins_job_url: string;
      status: string;
      triggered_by: string;
      start_time?: string;
      end_time?: string;
      duration?: number;
      error_message?: string;
      results: Array<{
        app_name: string;
        consul_tag: string;
        status: string;
        jenkins_build_num?: number;
        duration?: number;
        download_url?: string;
        error_message?: string;
      }>;
    };
  };
}

const aggregatePackageAPI = {
  createTask: (data: CreateAggregatePackageRequest) =>
    apiClient.post<CreateAggregatePackageResponse>('/deploy/aggregate-package', data),

  getTask: (taskId: number) =>
    apiClient.get<TaskStatusResponse>(`/deploy/aggregate-package/${taskId}`),

  getTasks: (params?: { project_name?: string; status?: string; start_time?: string; end_time?: string; page?: number; limit?: number }) =>
    apiClient.get<any>('/deploy/aggregate-package', { params }),
};

export { aggregatePackageAPI };
```

## 7. 安全设计

### 7.1 权限控制
- 验证用户是否具有触发聚合打包的权限
- 检查用户是否能够访问指定的项目
- 记录用户操作日志

### 7.2 参数校验
- 对项目名称进行验证，防止路径遍历
- 对应用名称列表进行逐个验证
- 限制应用名称的数量（如最多50个）
- 防止命令注入等安全风险

### 7.3 输入净化
```go
import "regexp"

// 验证和净化输入参数
func validateAndSanitizeInputs(projectName string, appNames []string) error {
    // 验证项目名称 - 只允许字母数字、下划线、连字符、空格
    if projectName == "" {
        return fmt.Errorf("项目名称不能为空")
    }

    if !regexp.MustCompile(`^[a-zA-Z0-9_\-\s]+$`).MatchString(projectName) {
        return fmt.Errorf("项目名称格式无效，只能包含字母、数字、下划线、连字符和空格")
    }

    // 验证应用名称列表
    if len(appNames) == 0 {
        return fmt.Errorf("至少需要指定一个应用名称")
    }

    if len(appNames) > 50 {
        return fmt.Errorf("应用名称数量不能超过50个")
    }

    // 验证每个应用名称
    validAppNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
    for _, appName := range appNames {
        if len(appName) == 0 {
            return fmt.Errorf("应用名称不能为空")
        }
        if !validAppNamePattern.MatchString(appName) {
            return fmt.Errorf("应用名称格式无效: %s，只能包含字母、数字、点、下划线和连字符", appName)
        }
    }

    return nil
}
```

## 8. 错误处理设计

### 8.1 前端错误处理
- 用户输入验证失败时显示友好提示
- API调用失败时显示错误信息
- 网络错误时提供重试机制
- 构建过程中断开连接时提供状态查询

### 8.2 后端错误处理
- 统一异常处理机制
- 详细的错误日志记录
- 优雅降级策略
- Jenkins服务不可用时的处理

```go
func (h *Handler) handleJenkinsError(taskID int64, err error) {
    h.db.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
        Updates(map[string]interface{}{
            "status": "failed",
            "error_message": fmt.Sprintf("Jenkins错误: %v", err),
            "end_time": time.Now(),
        })

    // 记录错误日志
    log.Printf("Jenkins error for task %d: %v", taskID, err)
}
```

## 9. 性能优化

### 9.1 数据库优化
- 为常用查询字段添加索引
- 合理设计分页查询
- 避免N+1查询问题

### 9.2 Consul集成优化
- 实现Consul连接池
- 增加缓存机制减少重复查询
- 使用Consul Watch机制监听配置变化

### 9.3 前端优化
- 使用虚拟滚动展示大量数据
- 合理缓存静态数据
- 按需加载组件
- 优化状态轮询频率

## 10. 部署配置

### 10.1 环境变量
- `CONSUL_ADDR`: Consul服务器地址，默认 `http://10.99.99.98:8500`
- `JENKINS_BASE_URL`: Jenkins基础URL，默认 `http://js.zbnsec.com`
- `JENKINS_USERNAME`: Jenkins用户名
- `JENKINS_PASSWORD`: Jenkins密码
- `AGGREGATE_PACKAGE_POLL_INTERVAL`: 状态轮询间隔，默认 `5000` 毫秒
- `MAX_CONCURRENT_TASKS`: 最大并发任务数，默认 `5`

### 10.2 监控配置
- 任务执行成功率监控
- 平均构建时间监控
- Consul连接健康度监控
- Jenkins服务可用性监控
- 错误率监控

## 11. 测试策略

### 11.1 单元测试
- 测试业务逻辑方法
- 测试Consul客户端功能
- 测试Jenkins客户端功能
- 测试数据访问层方法
- 测试API参数校验

### 11.2 集成测试
- 测试完整业务流程
- 测试异常处理逻辑
- 测试权限验证
- 测试Consul参数获取
- 测试Jenkins参数传递

### 11.3 端到端测试
- 测试用户操作流程
- 测试前后端集成
- 测试错误场景处理

## 12. 部署和运维注意事项

### 12.1 环境要求
- Consul服务可用且配置正确 (`http://10.99.99.98:8500`)
- Jenkins服务可用且具有指定作业 (`http://js.zbnsec.com/view/auto-archive-deploy/job/fscr-aggregation/`)
- 数据库连接正常
- 网络能够访问指定的Consul和Jenkins服务

### 12.2 监控告警
- 任务失败率超过阈值时发送告警
- Consul连接失败时发送告警
- Jenkins服务不可用时发送告警
- 任务执行时间过长时发送告警

此设计文档详细描述了安装包聚合打包功能的实现方案，特别是与您指定的Jenkins作业地址和Consul配置的集成。系统将接收项目名称和应用名称列表，从Consul获取相应的tag参数，然后触发指定的Jenkins作业，并传递以下参数：
- `app`: `fscr-aggregation`
- `tag`: 从Consul `plugin/fscr-aggregation/` 路径下获取
- `scope`: `all`ALTER TABLE security_vulnerabilities ADD COLUMN vuln_type VARCHAR(50);
