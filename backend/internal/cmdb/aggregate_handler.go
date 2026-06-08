package cmdb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/auth"
	"github.com/edy/ops-platform/internal/consul"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"github.com/edy/ops-platform/pkg/config"
	"github.com/edy/ops-platform/pkg/jenkins"
)

// Handler 结构体
type Handler struct {
	Cfg *config.Config
}

// consulAggregationPath Consul 路径常量
const consulAggregationPath = "plugin/fscr-aggregation/"

// getJenkinsURL 构建 Jenkins URL，基础 URL 末尾不带斜杠
func (h *Handler) getJenkinsURL() string {
	url := strings.TrimRight(h.Cfg.Jenkins.URL, "/")
	return url
}

// getDefaultConsulConfig 获取默认的 Consul 配置地址
func getDefaultConsulConfig() (string, error) {
	cs := consul.NewService(database.DB)
	config, err := cs.GetDefaultConfig()
	if err != nil {
		return "", fmt.Errorf("未找到可用的 Consul 配置: %w", err)
	}
	return config.Address, nil
}

// getConsulConfigByID 根据ID获取 Consul 配置地址
func getConsulConfigByID(id uint) (string, error) {
	cs := consul.NewService(database.DB)
	config, err := cs.GetConfig(id)
	if err != nil {
		return "", fmt.Errorf("未找到指定的 Consul 配置: %w", err)
	}
	return config.Address, nil
}

// 将时间转换为上海时区（UTC+8）
func toShanghaiTime(t time.Time) time.Time {
	shanghaiLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果无法加载时区，则返回原始时间
		return t
	}
	return t.In(shanghaiLoc)
}

// ConsulClient 封装Consul客户端
type ConsulClient struct {
	client *api.Client
}

// NewConsulClient 创建新的Consul客户端
func NewConsulClient(addr string) (*ConsulClient, error) {
	config := &api.Config{
		Address: addr,
	}
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ConsulClient{
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
		return "", fmt.Errorf("key not found: %s", path)
	}
	return string(pair.Value), nil
}

// ListKeys 列出指定路径下的所有键
func (c *ConsulClient) ListKeys(prefix string) ([]string, error) {
	keys, _, err := c.client.KV().Keys(prefix, "", nil)
	if err != nil {
		return nil, err
	}
	return keys, nil
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

// GetKeysByPrefix 从指定前缀获取所有键值对
func (c *ConsulClient) GetKeysByPrefix(prefix string) (map[string]string, error) {
	pairs, _, err := c.client.KV().List(prefix, nil)
	if err != nil {
		return nil, err
	}

	kvMap := make(map[string]string)
	for _, pair := range pairs {
		// 获取键名（去掉前缀部分作为key）
		key := strings.TrimPrefix(pair.Key, prefix)
		kvMap[key] = string(pair.Value)
	}

	return kvMap, nil
}

// ListConsulConfigs 获取所有 Consul 配置列表
func (h *Handler) ListConsulConfigs(c *gin.Context) {
	cs := consul.NewService(database.DB)
	configs, err := cs.GetConfigs()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get Consul configs: " + err.Error()})
		return
	}

	// 转换为前端需要的格式
	type ConsulConfigItem struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Address  string `json:"address"`
		IsDefault bool   `json:"is_default"`
	}

	items := make([]ConsulConfigItem, 0, len(configs))
	for _, cfg := range configs {
		items = append(items, ConsulConfigItem{
			ID:       cfg.ID,
			Name:     cfg.Name,
			Address:  cfg.Address,
			IsDefault: cfg.IsDefault,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    items,
	})
}

// QueryConsulKv 查询Consul KV数据
func (h *Handler) QueryConsulKv(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "path参数不能为空"})
		return
	}

	// 获取Consul地址（优先使用指定的配置，否则使用默认配置）
	var consulAddr string
	var err error

	consulConfigIDStr := c.Query("consul_config_id")
	if consulConfigIDStr != "" {
		consulConfigID, parseErr := strconv.ParseUint(consulConfigIDStr, 10, 32)
		if parseErr == nil {
			consulAddr, err = getConsulConfigByID(uint(consulConfigID))
		} else {
			consulAddr, err = getDefaultConsulConfig()
		}
	} else {
		consulAddr, err = getDefaultConsulConfig()
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get Consul config: " + err.Error()})
		return
	}
	consulClient, err := NewConsulClient(consulAddr)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to connect to Consul"})
		return
	}

	// 查询Consul KV数据
	kvMap, err := consulClient.GetKeysByPrefix(path)
	if err != nil {
		log.Printf("Error querying Consul KV: %v", err)
		c.JSON(500, gin.H{"error": "failed to query Consul KV"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    kvMap,
	})
}

// QueryAggregateTags 查询聚合打包可用的 tag 列表
func (h *Handler) QueryAggregateTags(c *gin.Context) {
	// 获取Consul地址（优先使用指定的配置，否则使用默认配置）
	var consulAddr string
	var err error

	consulConfigIDStr := c.Query("consul_config_id")
	if consulConfigIDStr != "" {
		consulConfigID, parseErr := strconv.ParseUint(consulConfigIDStr, 10, 32)
		if parseErr == nil {
			consulAddr, err = getConsulConfigByID(uint(consulConfigID))
		} else {
			consulAddr, err = getDefaultConsulConfig()
		}
	} else {
		consulAddr, err = getDefaultConsulConfig()
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get Consul config: " + err.Error()})
		return
	}

	// 创建 Consul 客户端
	consulClient, err := NewConsulClient(consulAddr)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to connect to Consul: " + err.Error()})
		return
	}

	// 查询 plugin/fscr-aggregation/ 下的所有键
	keys, err := consulClient.ListKeys(consulAggregationPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to list tags from Consul: " + err.Error()})
		return
	}

	// 提取 tag 值（键名的最后一部分）
	var tags []map[string]interface{}
	for _, key := range keys {
		// 去掉前缀 plugin/fscr-aggregation/
		tag := strings.TrimPrefix(key, consulAggregationPath)
		if tag != "" && tag != key {
			// 获取该 tag 的详细信息（可选）
			value, err := consulClient.GetKVValue(key)
			if err == nil {
				tags = append(tags, map[string]interface{}{
					"tag":   tag,
					"value": value,
				})
			} else {
				tags = append(tags, map[string]interface{}{
					"tag":   tag,
					"value": "",
				})
			}
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    tags,
	})
}

// TriggerAggregatePackage 创建聚合打包任务
func (h *Handler) TriggerAggregatePackage(c *gin.Context) {
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
	if err := validateAggregatePackageInputs(req.ProjectName, req.AppNames); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 准备Jenkins构建参数
	// Jenkins job fscr-aggregation 需要的参数：
	// - app: "fscr-aggregation"（固定值）
	// - tag: 用户选择的 tag 值（如 "V2.5.1", "6f_dev" 等）
	// - scope: "all"（固定值）
	buildParams := map[string]string{
		"app":   "fscr-aggregation",
		"scope": "all",
	}

	// 设置 tag 参数
	// 用户输入的应用名称列表，第一个作为 tag
	// 注意：用户应该输入 tag 值（如 V2.5.1），而不是应用名称
	tagName := "latest"
	if len(req.AppNames) > 0 {
		tagValue := req.AppNames[0]
		// 验证 tag 是否在 Consul 中存在
		tagName = getTagFromConsul(tagValue)
		buildParams["tag"] = tagName
	} else {
		buildParams["tag"] = "latest"
	}

	// 保存任务到数据库
	task := &models.AggregatePackageTask{
		TaskName:         req.TaskName,
		ProjectName:      req.ProjectName,
		AppNames:         req.AppNames,
		JenkinsJobName:   "fscr-aggregation",
		JenkinsJobUrl:    fmt.Sprintf("%s/view/auto-archive-deploy/job/fscr-aggregation/", h.getJenkinsURL()),
		ConsulConfigPath: consulAggregationPath,
		BuildParams:      buildParams, // 存储构建参数
		Status:           "pending",
		TriggeredBy:      c.GetString("username"),
	}

	if err := database.DB.Create(task).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create task"})
		return
	}

	// 立即创建聚合历史记录，状态为"归档中"
	jenkinsConsoleURL := fmt.Sprintf("%s/view/auto-archive-deploy/job/fscr-aggregation/", h.getJenkinsURL())
	now := time.Now()

	// 获取有效的操作人姓名
	operatorName := getEffectiveOperatorName(c.GetString("username"), c.GetString("real_name"))

	history := &models.AggregatedHistory{
		ProjectName:       task.ProjectName,
		Environment:       tagName,     // 使用 Tag 名称
		Status:            "archiving", // 归档中
		Progress:          0,           // 初始进度为0
		StartTime:         &now,        // 设置开始时间
		Operator:          operatorName,
		OperatorName:      operatorName,
		JenkinsJobName:    task.JenkinsJobName,
		JenkinsConsoleURL: &jenkinsConsoleURL,
		TaskID:            &task.ID, // 关联任务ID
	}
	if err := database.DB.Create(history).Error; err != nil {
		log.Printf("Failed to create aggregated history record: %v", err)
	}

	// 为每个应用创建结果记录
	for _, appName := range req.AppNames {
		consulTag := getTagFromConsul(appName)
		result := &models.AggregatePackageResult{
			TaskID:      uint(task.ID),
			AppName:     appName,
			ConsulTag:   consulTag, // 使用从Consul获取的tag
			Status:      "pending",
		}
		database.DB.Create(result)
	}

	// 异步触发Jenkins构建
	go h.triggerJenkinsBuild(uint(task.ID), "/view/auto-archive-deploy/job/fscr-aggregation/", buildParams)

	c.JSON(200, gin.H{"success": true, "data": gin.H{"task_id": task.ID, "history_id": history.ID}})
}

// getEffectiveOperatorName 获取有效的操作人姓名
func getEffectiveOperatorName(username string, realNameFromContext string) string {
	// 如果从上下文获得了有效的实名，直接使用
	if realNameFromContext != "" && realNameFromContext != "系统" && realNameFromContext != "未知用户" && realNameFromContext != "系统用户" {
		return realNameFromContext
	}

	// 从数据库查询用户的真实姓名
	var user models.User
	if err := database.DB.Select("real_name, username").Where("username = ?", username).First(&user).Error; err == nil {
		// 检查数据库中的真实姓名是否有效
		if user.RealName != "" && user.RealName != "系统" && user.RealName != "未知用户" && user.RealName != "系统用户" {
			return user.RealName
		} else {
			// 如果数据库中没有有效的实名，则使用用户名
			if user.Username != "" {
				return user.Username
			} else {
				return "系统用户"
			}
		}
	}

	// 如果数据库查询失败，回退到用户名
	if username != "" {
		return username
	}

	return "系统用户"
}

// getTagFromConsul 从Consul获取指定应用的标签
// 注意：Consul 中的键名就是 tag 值（如 V2.5.1, 6f_dev 等）
// 这里直接返回用户输入的 tag 值
func getTagFromConsul(tagValue string) string {
	// 验证 tag 值是否在 Consul 中存在
	consulAddr, err := getDefaultConsulConfig()
	if err != nil {
		log.Printf("Warning: Could not get Consul config: %v", err)
		// 如果无法获取 Consul 配置，仍然返回用户输入的值
		return tagValue
	}
	consulClient, err := NewConsulClient(consulAddr)
	if err != nil {
		log.Printf("Warning: Could not connect to Consul: %v", err)
		// 如果无法连接 Consul，仍然返回用户输入的值
		return tagValue
	}

	// 检查 Consul 中是否存在该 tag
	key := consulAggregationPath + tagValue
	_, err = consulClient.GetKVValue(key)
	if err != nil {
		log.Printf("Warning: Tag %s not found in Consul: %v", tagValue, err)
		// 如果 tag 不存在，仍然返回用户输入的值（让 Jenkins 处理错误）
		return tagValue
	}

	return tagValue
}

// Validate inputs for aggregate package
func validateAggregatePackageInputs(projectName string, appNames []string) error {
	if projectName == "" {
		return fmt.Errorf("项目名称不能为空")
	}

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

// QueryJenkinsJobs 查询Jenkins作业列表
func (h *Handler) QueryJenkinsJobs(c *gin.Context) {
	// 使用配置中的 Jenkins 认证信息初始化客户端
	jenkinsClient := jenkins.NewClient(
		h.Cfg.Jenkins.URL,      // Jenkins基础URL，从配置获取
		h.Cfg.Jenkins.Username, // 用户名
		h.Cfg.Jenkins.Token,    // API Token
	)

	// 先获取所有视图
	viewsURL := jenkinsClient.BaseURL + "/api/json?tree=views[name,url]"
	req, err := http.NewRequest("GET", viewsURL, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create request: " + err.Error()})
		return
	}
	req.SetBasicAuth(jenkinsClient.Username, jenkinsClient.Password)

	resp, err := jenkinsClient.Client.Do(req)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get Jenkins views: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(500, gin.H{"error": "failed to get Jenkins views, status: " + resp.Status})
		return
	}

	var viewsResponse struct {
		Views []map[string]interface{} `json:"views"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&viewsResponse); err != nil {
		c.JSON(500, gin.H{"error": "failed to decode Jenkins views response: " + err.Error()})
		return
	}

	allJobs := []map[string]interface{}{}

	for _, view := range viewsResponse.Views {
		viewName, ok := view["name"].(string)
		if !ok {
			continue
		}

		// 跳过某些系统视图
		if viewName == "all" || viewName == "All" {
			continue
		}

		// 获取视图中的作业
		viewJobs, err := jenkinsClient.GetViewJobs(viewName)
		if err != nil {
			log.Printf("Warning: Could not get view %s jobs: %v", viewName, err)
			continue
		}

		for _, job := range viewJobs.Jobs {
			// 只保留符合应用命名规范的作业
			if isValidAppName(job.Name) {
				jobInfo := map[string]interface{}{
					"name":      job.Name,
					"view":      viewName,
					"url":       job.URL,
					"color":     job.Color,
					"job_name":  job.Name,
				}
				allJobs = append(allJobs, jobInfo)
			}
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    allJobs,
		"total":   len(allJobs),
	})
}

// QueryAppTags 查询应用的标签
func (h *Handler) QueryAppTags(c *gin.Context) {
	var req struct {
		AppNames []string `json:"app_names" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 获取Consul客户端
	consulAddr, err := getDefaultConsulConfig()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get Consul config: " + err.Error()})
		return
	}
	consulClient, err := NewConsulClient(consulAddr)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to connect to Consul"})
		return
	}

	// 从Consul获取tag信息
	// 使用标准路径 plugin/fscr-aggregation/ 作为基础路径
	basePath := consulAggregationPath
	tagMap, err := consulClient.GetTagsFromPath(basePath, req.AppNames)
	if err != nil {
		// 如果Consul获取失败，记录错误但仍返回部分结果
		log.Printf("Warning: Could not get tags from Consul: %v", err)
		// 即使出错也继续，因为可能有些应用存在标签而有些不存在
	}

	// 确保所有请求的应用名称都在结果中（不存在的使用默认值）
	for _, appName := range req.AppNames {
		if _, exists := tagMap[appName]; !exists {
			tagMap[appName] = "latest" // 默认值
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    tagMap,
	})
}

// isValidAppName 检查应用名称是否符合规范
func isValidAppName(name string) bool {
	// 假设应用名称应该符合某种规范，这里简单检查是否包含常见的应用标识
	// 可以根据实际需求调整此函数
	if len(name) == 0 {
		return false
	}

	// 不是文件夹类型
	if strings.Contains(name, "/") {
		return false
	}

	// 过滤掉某些系统作业
	blacklist := []string{"_template", "template", "backup"}
	for _, black := range blacklist {
		if strings.Contains(strings.ToLower(name), black) {
			return false
		}
	}

	return true
}

// triggerJenkinsBuild 触发Jenkins构建
func (h *Handler) triggerJenkinsBuild(taskID uint, jobPath string, params map[string]string) {
	// 更新任务状态为building
	database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
		Update("status", "building")

	// 使用配置中的Jenkins信息初始化Jenkins客户端
	jenkinsURL := h.Cfg.Jenkins.URL
	if jenkinsURL == "" {
		// 记录错误并更新状态
		log.Printf("Jenkins URL is not configured for task %d", taskID)
		database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": "Jenkins URL 未配置",
				"end_time":      toShanghaiTime(time.Now()),
			})
		return
	}

	log.Printf("Initializing Jenkins client with URL: %s, Username: %s", jenkinsURL, h.Cfg.Jenkins.Username)

	jenkinsClient := jenkins.NewClient(
		jenkinsURL,                 // Jenkins基础URL
		h.Cfg.Jenkins.Username,     // 用户名
		h.Cfg.Jenkins.Token,        // API Token
	)

	// 触发Jenkins构建
	queueID, err := jenkinsClient.BuildJobWithParams(jobPath, params)
	if err != nil {
		// 记录错误并更新状态
		log.Printf("Jenkins build failed for task %d: %v. URL: %s, Username: %s", taskID, err, jenkinsURL, h.Cfg.Jenkins.Username)
		database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
			Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": fmt.Sprintf("Jenkins构建失败: %v", err),
				"end_time":      toShanghaiTime(time.Now()), // 记录结束时间为上海时区
			})
		return
	}

	// 更新任务中的Jenkins队列ID和开始时间
	database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"jenkins_queue_id": queueID,
			"start_time":       toShanghaiTime(time.Now()), // 记录开始时间为上海时区
		})

	// 同时更新聚合历史记录中的Jenkins队列ID
	database.DB.Model(&models.AggregatedHistory{}).Where("task_id = ?", taskID).
		Update("jenkins_queue_id", queueID)

	// 启动状态轮询
	go h.pollJenkinsStatus(taskID, queueID, jobPath)
}

// pollJenkinsStatus 轮询Jenkins状态
func (h *Handler) pollJenkinsStatus(taskID uint, queueID int64, jobPath string) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒轮询一次
	defer ticker.Stop()

	maxAttempts := 1440 // 最多轮询2小时（1440次 * 5秒）
	attempts := 0

	for {
		select {
		case <-ticker.C:
			attempts++

			// 通过队列ID检查构建是否已启动
			buildNum, _, err := h.checkQueueStatus(queueID)
			if err != nil {
				log.Printf("Error checking queue status for task %d: %v", taskID, err)
				continue
			}

			if buildNum > 0 {
				// 更新历史记录的构建号
				consoleURL := fmt.Sprintf("%s/view/auto-archive-deploy/job/fscr-aggregation/%d/console", h.getJenkinsURL(), buildNum)
				database.DB.Model(&models.AggregatedHistory{}).Where("task_id = ?", taskID).
					Updates(map[string]interface{}{
						"jenkins_build_num":  buildNum,
						"jenkins_console_url": consoleURL,
					})

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
				database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
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

// checkQueueStatus 检查队列状态
func (h *Handler) checkQueueStatus(queueID int64) (int, string, error) {
	// 实现队列状态检查逻辑
	// 这里只是一个示例实现
	return 0, "waiting", nil
}

// getBuildStatus 获取构建状态
func (h *Handler) getBuildStatus(jobPath string, buildNum int) (BuildStatus, error) {
	// 使用配置中的Jenkins信息初始化Jenkins客户端
	jenkinsURL := h.Cfg.Jenkins.URL
	if jenkinsURL == "" {
		log.Printf("Jenkins URL is not configured for build status check")
		return BuildStatus{}, fmt.Errorf("Jenkins URL 未配置")
	}

	log.Printf("Initializing Jenkins client for status check with URL: %s, Username: %s", jenkinsURL, h.Cfg.Jenkins.Username)

	jenkinsClient := jenkins.NewClient(
		jenkinsURL,
		h.Cfg.Jenkins.Username, // 用户名
		h.Cfg.Jenkins.Token,    // API Token
	)

	buildInfo, err := jenkinsClient.GetBuildInfo(jobPath, buildNum)
	if err != nil {
		log.Printf("Error getting build info for job %s #%d: %v", jobPath, buildNum, err)
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

// updateTaskAndResultStatus 更新任务和结果状态
func (h *Handler) updateTaskAndResultStatus(taskID uint, status BuildStatus) {
	// 更新任务状态
	dbStatus := "building"
	historyStatus := "archiving" // 聚合历史状态：归档中
	if !status.Building {
		if status.Result == "SUCCESS" {
			dbStatus = "success"
			historyStatus = "completed" // 完成
		} else {
			dbStatus = "failed"
			historyStatus = "failed" // 归档失败
		}
	}

	updates := map[string]interface{}{
		"status": dbStatus,
	}

	var task models.AggregatePackageTask
	if database.DB.Where("id = ?", taskID).First(&task).Error != nil {
		log.Printf("Task %d not found", taskID)
		return
	}

	if !status.Building { // 构建已完成
		updates["end_time"] = toShanghaiTime(time.Now()) // 记录结束时间为上海时区
		// 计算持续时间
		if !task.StartTime.IsZero() {
			duration := time.Since(*task.StartTime)
			updates["duration"] = int(duration.Seconds())
		}

		// 更新已存在的聚合历史记录
		h.updateAggregatedHistoryByTaskID(taskID, historyStatus, task)
	}

	database.DB.Model(&models.AggregatePackageTask{}).Where("id = ?", taskID).
		Updates(updates)
}

// updateAggregatedHistoryByTaskID 根据任务ID更新聚合历史记录
func (h *Handler) updateAggregatedHistoryByTaskID(taskID uint, historyStatus string, task models.AggregatePackageTask) {
	var history models.AggregatedHistory
	if err := database.DB.Where("task_id = ?", taskID).First(&history).Error; err != nil {
		// 如果没有找到关联的历史记录，创建一个新的
		log.Printf("No aggregated history found for task %d, creating new one", taskID)
		h.createAggregatedHistoryRecord(task, historyStatus)
		return
	}

	// 获取当前操作人姓名（保留原来的操作人姓名，避免被覆盖为"系统"）
	originalOperatorName := history.OperatorName
	originalOperator := history.Operator

	// 更新历史记录状态
	now := time.Now()
	updates := map[string]interface{}{
		"status": historyStatus,
	}

	// 更新 Jenkins QueueID
	if task.JenkinsQueueID != nil {
		updates["jenkins_queue_id"] = *task.JenkinsQueueID
	}

	// 根据状态更新其他字段
	if historyStatus == "completed" {
		// 成功：生成下载链接，进度100%
		downloadURL := getLatestAggregatedPackageURLFromHistory()
		if downloadURL != "" {
			updates["download_url"] = downloadURL
		}
		updates["end_time"] = now
		updates["progress"] = 100
	} else if historyStatus == "failed" {
		// 失败：记录错误信息，进度100%
		if task.ErrorMessage != nil {
			updates["error_message"] = *task.ErrorMessage
		} else {
			updates["error_message"] = "聚合打包失败"
		}
		updates["end_time"] = now
		updates["progress"] = 100
	}

	// 尝试从第一个结果记录中获取构建号，更新 Jenkins 控制台 URL
	var firstResult models.AggregatePackageResult
	if err := database.DB.Where("task_id = ?", taskID).First(&firstResult).Error; err == nil {
		if firstResult.JenkinsBuildNum != nil {
			updates["jenkins_build_num"] = *firstResult.JenkinsBuildNum
			consoleURL := fmt.Sprintf("%s/view/auto-archive-deploy/job/fscr-aggregation/%d/console", h.getJenkinsURL(), *firstResult.JenkinsBuildNum)
			updates["jenkins_console_url"] = consoleURL
		}
	}

	// 确保不覆盖原有的操作人和操作人姓名
	if originalOperatorName != "" {
		updates["operator_name"] = originalOperatorName
	}
	if originalOperator != "" {
		updates["operator"] = originalOperator
	}

	if err := database.DB.Model(&history).Updates(updates).Error; err != nil {
		log.Printf("Failed to update aggregated history for task %d: %v", taskID, err)
	} else {
		log.Printf("Updated aggregated history for task %d, status: %s", taskID, historyStatus)
	}
}

// BuildStatus 构建状态
type BuildStatus struct {
	Building bool
	Result   string
}

// IsComplete 是否完成
func (bs BuildStatus) IsComplete() bool {
	return !bs.Building
}

// getLatestAggregatedPackageURLFromHistory 获取最新的聚合包下载地址
func getLatestAggregatedPackageURLFromHistory() string {
	return findLatestAggregatedPackageURL()
}

// createAggregatedHistoryRecord 创建聚合历史记录
func (h *Handler) createAggregatedHistoryRecord(task models.AggregatePackageTask, finalStatus string) {
	// 获取有效的操作人姓名
	operatorName := getEffectiveOperatorName(task.TriggeredBy, "")

	// 创建聚合历史记录
	history := &models.AggregatedHistory{
		ProjectName:      task.ProjectName,
		Environment:      "production", // 默认环境，可根据实际需要调整
		Status:           finalStatus,
		Operator:         operatorName,
		OperatorName:     operatorName,
		JenkinsJobName:   task.JenkinsJobName,
		JenkinsQueueID:   task.JenkinsQueueID, // 使用任务中的队列ID
		ErrorMessage:     task.ErrorMessage, // 如果有错误信息
	}

	// 如果有Jenkins控制台URL，添加它
	if task.JenkinsJobUrl != "" {
		history.JenkinsConsoleURL = &task.JenkinsJobUrl
	}

	// 如果是成功状态，生成下载链接
	now := time.Now()
	if finalStatus == "success" {
		// 获取最新的聚合包下载地址
		downloadURL := getLatestAggregatedPackageURLFromHistory()
		if downloadURL != "" {
			history.DownloadURL = &downloadURL
		}
		history.EndTime = &now
	} else if finalStatus == "failed" && history.ErrorMessage == nil {
		// 如果失败且没有错误信息，使用默认错误信息
		errorMsg := "聚合打包失败"
		history.ErrorMessage = &errorMsg
		history.EndTime = &now
	}

	// 保存到数据库
	if err := database.DB.Create(history).Error; err != nil {
		log.Printf("Failed to create aggregated history record for task %d: %v", task.ID, err)
	} else {
		log.Printf("Successfully created aggregated history record for task %d", task.ID)
	}
}

// GetAggregatePackageRecords 获取聚合打包任务列表
func (h *Handler) GetAggregatePackageRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	projectName := c.Query("project_name")
	status := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	offset := (page - 1) * limit

	query := database.DB.Model(&models.AggregatePackageTask{})

	if projectName != "" {
		query = query.Where("project_name LIKE ?", "%"+projectName+"%")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}

	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	query.Count(&total)

	var tasks []models.AggregatePackageTask
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tasks).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch tasks"})
		return
	}

	// 为每个任务获取结果数量
	for i := range tasks {
		var count int64
		database.DB.Model(&models.AggregatePackageResult{}).Where("task_id = ?", tasks[i].ID).Count(&count)
		tasks[i].AppNames = append(tasks[i].AppNames[:0:0], tasks[i].AppNames...) // 修正切片长度以确保JSON序列化正确
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"tasks": tasks,
		},
	})
}

// GetAggregatePackageRecord 获取聚合打包任务详情
func (h *Handler) GetAggregatePackageRecord(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid task ID"})
		return
	}

	var task models.AggregatePackageTask
	if err := database.DB.First(&task, uint(taskID)).Error; err != nil {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}

	var results []models.AggregatePackageResult
	database.DB.Where("task_id = ?", task.ID).Find(&results)

	taskData := map[string]interface{}{
		"id":                task.ID,
		"task_name":         task.TaskName,
		"project_name":      task.ProjectName,
		"app_names":         task.AppNames,
		"jenkins_job_name":  task.JenkinsJobName,
		"jenkins_job_url":   task.JenkinsJobUrl,
		"status":            task.Status,
		"triggered_by":      task.TriggeredBy,
		"start_time":        task.StartTime,
		"end_time":          task.EndTime,
		"duration":          task.Duration,
		"error_message":     task.ErrorMessage,
		"results":           results,
		"created_at":        task.CreatedAt,
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"task": taskData,
		},
	})
}

// GetAggregatePackageStatus 获取聚合打包任务状态
func (h *Handler) GetAggregatePackageStatus(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid task ID"})
		return
	}

	var task models.AggregatePackageTask
	if err := database.DB.First(&task, uint(taskID)).Error; err != nil {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}

	var results []models.AggregatePackageResult
	database.DB.Where("task_id = ?", task.ID).Find(&results)

	appStatuses := make([]map[string]string, len(results))
	for i, result := range results {
		appStatuses[i] = map[string]string{
			"app_name": result.AppName,
			"status":   result.Status,
		}
	}

	var totalProgress float64
	if len(results) > 0 {
		successful := 0
		for _, result := range results {
			if result.Status == "success" {
				successful++
			}
		}
		totalProgress = float64(successful) / float64(len(results)) * 100
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"status":        task.Status,
			"progress":      totalProgress,
			"overall_status": task.Status,
			"app_statuses":  appStatuses,
		},
	})
}

// RegisterAggregatePackageRoutes 注册聚合打包相关路由
func (h *Handler) RegisterAggregatePackageRoutes(r *gin.Engine, cfg *config.Config) {
	// 聚合打包相关API
	aggregateGroup := r.Group("/api/deploy")
	aggregateGroup.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		aggregateGroup.POST("/aggregate-package", h.TriggerAggregatePackage)
		aggregateGroup.GET("/aggregate-package", h.GetAggregatePackageRecords)
		aggregateGroup.GET("/aggregate-package/:taskId", h.GetAggregatePackageRecord)
		aggregateGroup.GET("/aggregate-package/:taskId/status", h.GetAggregatePackageStatus)
		aggregateGroup.POST("/query-app-tags", h.QueryAppTags)        // 查询应用标签
		aggregateGroup.GET("/query-jenkins-jobs", h.QueryJenkinsJobs)  // 查询Jenkins作业
		aggregateGroup.GET("/query-consul-kv", h.QueryConsulKv)        // 查询Consul KV数据
		aggregateGroup.GET("/aggregate-tags", h.QueryAggregateTags)    // 查询聚合打包可用的 tag 列表
		aggregateGroup.GET("/consul-configs", h.ListConsulConfigs)     // 获取Consul配置列表
	}
}
