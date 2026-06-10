// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/jenvenson/ops-platform/pkg/jenkins"
	"github.com/gin-gonic/gin"
)

// 聚合历史轮询全局状态
var (
	aggregatedHistoryStopChan chan struct{}
	aggregatedHistoryWg       sync.WaitGroup
	aggregatedHistoryRunning  bool
	aggregatedHistoryMu       sync.Mutex
)

// 为了方便使用，创建一个别名
type AggregatedHistory = models.AggregatedHistory

// AggregatedHistoryRequest 聚合历史请求结构
type AggregatedHistoryRequest struct {
	ProjectName string `json:"project_name" binding:"required"`
	Environment string `json:"environment" binding:"required"`
}

// AggregatedHistoryResponse 聚合历史响应结构
type AggregatedHistoryResponse struct {
	ID                uint       `json:"id"`
	ProjectName       string     `json:"project_name"`
	Environment       string     `json:"environment"`
	Status            string     `json:"status"`
	Progress          int        `json:"progress"`
	StartTime         *time.Time `json:"start_time,omitempty"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	DownloadURL       *string    `json:"download_url,omitempty"`
	Operator          string     `json:"operator"`
	OperatorName      string     `json:"operator_name"`
	JenkinsJobName    string     `json:"jenkins_job_name"`
	JenkinsBuildNum   *int       `json:"jenkins_build_num,omitempty"`
	JenkinsQueueID    *int64     `json:"jenkins_queue_id,omitempty"`
	JenkinsConsoleURL *string    `json:"jenkins_console_url,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// GetAggregatedHistories 获取聚合历史列表
func GetAggregatedHistories(c *gin.Context) {
	projectName := c.Query("project_name")
	environment := c.Query("environment")
	operator := c.Query("operator")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := database.DB.Model(&AggregatedHistory{})

	if projectName != "" {
		query = query.Where("project_name LIKE ?", "%"+projectName+"%")
	}
	if environment != "" {
		query = query.Where("environment = ?", environment)
	}
	if operator != "" {
		query = query.Where("operator LIKE ?", "%"+operator+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var histories []AggregatedHistory
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch aggregated histories"})
		return
	}

	// 转换为响应格式
	responses := make([]*AggregatedHistoryResponse, len(histories))
	for i, history := range histories {
		responses[i] = toAggregatedHistoryResponse(&history)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetAggregatedHistory 获取单个聚合历史记录
func GetAggregatedHistory(c *gin.Context) {
	id := c.Param("id")

	var history AggregatedHistory
	if err := database.DB.Where("id = ?", id).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "aggregated history not found"})
		return
	}

	c.JSON(http.StatusOK, toAggregatedHistoryResponse(&history))
}

// DeleteAggregatedHistory 删除聚合历史记录
func DeleteAggregatedHistory(c *gin.Context) {
	id := c.Param("id")

	var history AggregatedHistory
	if err := database.DB.Where("id = ?", id).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "aggregated history not found"})
		return
	}

	if err := database.DB.Delete(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete aggregated history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// toAggregatedHistoryResponse 转换为API响应格式
func toAggregatedHistoryResponse(history *AggregatedHistory) *AggregatedHistoryResponse {
	if history == nil {
		return nil
	}

	return &AggregatedHistoryResponse{
		ID:                history.ID,
		ProjectName:       history.ProjectName,
		Environment:       history.Environment,
		Status:            history.Status,
		Progress:          history.Progress,
		StartTime:         history.StartTime,
		EndTime:           history.EndTime,
		DownloadURL:       history.DownloadURL,
		Operator:          history.Operator,
		OperatorName:      history.OperatorName,
		JenkinsJobName:    history.JenkinsJobName,
		JenkinsBuildNum:   history.JenkinsBuildNum,
		JenkinsQueueID:    history.JenkinsQueueID,
		JenkinsConsoleURL: history.JenkinsConsoleURL,
		ErrorMessage:      history.ErrorMessage,
		CreatedAt:         history.CreatedAt,
		UpdatedAt:         history.UpdatedAt,
	}
}

// GetAggregatedHistoryStatus 获取聚合历史状态（从Jenkins更新）
func GetAggregatedHistoryStatus(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var history AggregatedHistory
	if err := database.DB.Where("id = ?", id).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "aggregated history not found"})
		return
	}

	// 如果Jenkins配置为空，直接返回当前状态
	if cfg.Jenkins.URL == "" || cfg.Jenkins.Username == "" || cfg.Jenkins.Token == "" {
		c.JSON(http.StatusOK, gin.H{
			"id":     history.ID,
			"status": history.Status,
		})
		return
	}

	// 创建Jenkins客户端
	jenkinsClient := jenkins.NewClient(
		cfg.Jenkins.URL,
		cfg.Jenkins.Username,
		cfg.Jenkins.Token,
		time.Duration(cfg.Jenkins.Timeout)*time.Second,
	)

	buildNum := 0
	if history.JenkinsBuildNum != nil {
		buildNum = *history.JenkinsBuildNum
	}

	queueID := int64(0)
	if history.JenkinsQueueID != nil {
		queueID = *history.JenkinsQueueID
	}

	// 如果没有构建号但有队列ID，尝试通过队列ID获取构建号
	if buildNum == 0 && queueID > 0 {
		queueInfo, err := jenkinsClient.GetQueueItemInfo(queueID)
		if err == nil && queueInfo != nil && queueInfo.Executable != nil {
			buildNum = queueInfo.Executable.Number
			history.JenkinsBuildNum = &buildNum
			database.DB.Model(&history).Update("jenkins_build_num", buildNum)
		} else if err == nil && queueInfo == nil {
			// 队列项已消失，从Job获取最新的构建
			jobInfo, err := jenkinsClient.GetJobInfo(history.JenkinsJobName)
			if err == nil && jobInfo.LastBuild != nil {
				buildNum = jobInfo.LastBuild.Number
				history.JenkinsBuildNum = &buildNum
				database.DB.Model(&history).Update("jenkins_build_num", buildNum)
			}
		}
	}

	// 如果仍然没有构建号，检查是否在队列中
	if buildNum == 0 {
		c.JSON(http.StatusOK, gin.H{
			"id":       history.ID,
			"status":   "queued",
			"queue_id": queueID,
			"message":  "构建任务已在队列中等待",
		})
		return
	}

	// 从Jenkins获取构建状态
	status, err := jenkinsClient.GetBuildStatus(history.JenkinsJobName, buildNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取 Jenkins 状态失败: %v", err)})
		return
	}

	// 更新本地记录状态
	now := time.Now()
	updated := false

	// 转换为大写以兼容 Jenkins 返回的大小写
	phase := strings.ToUpper(status.Phase)
	result := strings.ToUpper(status.Result)

	// 解析 Jenkins 状态
	switch phase {
	case "QUEUED":
		history.Status = "queued"
	case "STARTED", "RUNNING":
		history.Status = "running"
	case "COMPLETED", "FINALIZED":
		if result == "SUCCESS" {
			history.Status = "success"
			downloadURL := getLatestAggregatedPackageURL()
			if downloadURL != "" {
				history.DownloadURL = &downloadURL
			}
		} else {
			history.Status = "failed"
		}
		history.EndTime = &now
		updated = true
	case "":
		// 构建已完成，phase 为空
		if result == "SUCCESS" {
			history.Status = "success"
			downloadURL := getLatestAggregatedPackageURL()
			if downloadURL != "" {
				history.DownloadURL = &downloadURL
			}
		} else if result == "FAILURE" {
			history.Status = "failed"
			history.EndTime = &now
		} else {
			history.Status = "running"
		}
		if history.Status != "running" {
			history.EndTime = &now
			updated = true
		}
	}

	// 保存原始的OperatorName以避免在更新时丢失
	originalOperatorName := history.OperatorName
	originalOperator := history.Operator

	if updated {
		if err := database.DB.Save(&history).Error; err != nil {
			fmt.Printf("[AggregatedHistoryStatus] 更新数据库失败: %v\n", err)
		}
	}

	// 如果OperatorName被覆盖，恢复原始值
	if updated && originalOperatorName != "" && history.OperatorName != originalOperatorName {
		database.DB.Model(&history).Update("operator_name", originalOperatorName)
	}
	if updated && originalOperator != "" && history.Operator != originalOperator {
		database.DB.Model(&history).Update("operator", originalOperator)
	}

	// 记录日志用于调试
	fmt.Printf("[AggregatedHistoryStatus] id=%d, phase=%s, result=%s, updated=%v\n",
		history.ID, phase, result, updated)

	// 更新Jenkins控制台URL
	consoleURL := fmt.Sprintf("%s/view/auto-archive-deploy/job/%s/%d/console", cfg.Jenkins.URL, history.JenkinsJobName, buildNum)
	history.JenkinsConsoleURL = &consoleURL

	// 使用 Updates 而不是 Update 来避免覆盖其他字段
	updateFields := map[string]interface{}{
		"jenkins_console_url": consoleURL,
	}
	database.DB.Model(&history).Updates(updateFields)

	c.JSON(http.StatusOK, gin.H{
		"id":           history.ID,
		"status":       history.Status,
		"build_number": buildNum,
		"phase":        status.Phase,
		"jenkins_url":  status.URL,
		"console_url":  consoleURL,
		"download_url": history.DownloadURL,
		"timestamp":    status.Timestamp,
	})
}

// GetAggregatedHistoryFiles 获取聚合历史关联文件列表
func GetAggregatedHistoryFiles(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var history AggregatedHistory
	if err := database.DB.Where("id = ?", id).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "aggregated history not found"})
		return
	}

	if history.Status != "success" && history.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "聚合任务尚未完成，无法获取文件列表"})
		return
	}

	files, baseURL, timestamp, err := resolveAggregatedHistoryFiles(&history, cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"base_url":  baseURL,
			"timestamp": timestamp,
			"files":     []ArchiveFile{},
			"message":   "未找到聚合文件",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"base_url":  baseURL,
		"timestamp": timestamp,
		"files":     files,
	})
}

// RefreshAggregatedHistoryStatus 刷新单个聚合历史记录的状态
func RefreshAggregatedHistoryStatus(history *AggregatedHistory, client *jenkins.Client) {
	buildNum := 0
	if history.JenkinsBuildNum != nil {
		buildNum = *history.JenkinsBuildNum
	}

	queueID := int64(0)
	if history.JenkinsQueueID != nil {
		queueID = *history.JenkinsQueueID
	}

	// 如果没有构建号但有队列ID，尝试通过队列ID获取构建号
	if buildNum == 0 && queueID > 0 {
		queueInfo, err := client.GetQueueItemInfo(queueID)
		if err == nil && queueInfo != nil && queueInfo.Executable != nil {
			buildNum = queueInfo.Executable.Number
			history.JenkinsBuildNum = &buildNum
			database.DB.Model(history).Update("jenkins_build_num", buildNum)
		} else if err == nil && queueInfo == nil {
			// 队列项已消失，从Job获取最新的构建
			jobInfo, err := client.GetJobInfo(history.JenkinsJobName)
			if err == nil && jobInfo.LastBuild != nil {
				buildNum = jobInfo.LastBuild.Number
				history.JenkinsBuildNum = &buildNum
				database.DB.Model(history).Update("jenkins_build_num", buildNum)
			}
		}
	}

	if buildNum == 0 {
		// 仍在排队，设置进度为5%
		database.DB.Model(history).Update("progress", 5)
		return
	}

	// 获取构建状态
	buildStatus, err := client.GetBuildStatus(history.JenkinsJobName, buildNum)
	if err != nil || buildStatus == nil {
		return
	}

	now := time.Now()
	updates := map[string]interface{}{}

	// 计算进度
	if buildStatus.Phase == "BUILDING" && buildStatus.EstimatedDuration > 0 {
		// 计算已运行时间（毫秒）
		elapsed := time.Now().UnixMilli() - buildStatus.BuildTimestamp
		// 进度 = 已运行时间 / 预估时间 * 100
		progress := int(float64(elapsed) / float64(buildStatus.EstimatedDuration) * 100)
		// 限制进度在 0-100 之间（构建中）
		if progress < 0 {
			progress = 0
		} else if progress > 100 {
			progress = 100
		}
		updates["progress"] = progress
		history.Progress = progress
	}

	// 根据 Jenkins 结果更新状态
	switch buildStatus.Phase {
	case "COMPLETED", "FINALIZED":
		if buildStatus.Result == "SUCCESS" {
			history.Status = "completed"
			updates["status"] = "completed"
			// 获取最新的聚合包下载地址
			downloadURL := getLatestAggregatedPackageURL()
			if downloadURL != "" {
				history.DownloadURL = &downloadURL
				updates["download_url"] = downloadURL
			}
		} else {
			history.Status = "failed"
			updates["status"] = "failed"
		}
		updates["end_time"] = &now
		updates["progress"] = 100
		history.Progress = 100
	case "":
		// 构建已完成，phase 为空
		if buildStatus.Result == "SUCCESS" {
			history.Status = "completed"
			updates["status"] = "completed"
			// 获取最新的聚合包下载地址
			downloadURL := getLatestAggregatedPackageURL()
			if downloadURL != "" {
				history.DownloadURL = &downloadURL
				updates["download_url"] = downloadURL
			}
		} else if buildStatus.Result == "FAILURE" {
			history.Status = "failed"
			updates["status"] = "failed"
		}
		if history.Status != "running" && history.Status != "archiving" {
			updates["end_time"] = &now
			updates["progress"] = 100
			history.Progress = 100
		}
	}

	// 设置Jenkins控制台URL
	consoleURL := fmt.Sprintf("%s/view/auto-archive-deploy/job/%s/%d/console", client.BaseURL, history.JenkinsJobName, buildNum)
	history.JenkinsConsoleURL = &consoleURL
	updates["jenkins_console_url"] = consoleURL

	// 保存原始的OperatorName以避免在更新时丢失
	originalOperatorName := history.OperatorName
	originalOperator := history.Operator

	if len(updates) > 0 {
		database.DB.Model(history).Updates(updates)
	}

	// 如果OperatorName被覆盖，恢复原始值
	if len(updates) > 0 && originalOperatorName != "" && history.OperatorName != originalOperatorName {
		database.DB.Model(history).Update("operator_name", originalOperatorName)
	}
	if len(updates) > 0 && originalOperator != "" && history.Operator != originalOperator {
		database.DB.Model(history).Update("operator", originalOperator)
	}
}

// ScheduleAggregatedHistoryRefresh 启动定时刷新聚合历史记录的Jenkins状态
func ScheduleAggregatedHistoryRefresh(client *jenkins.Client, pollInterval time.Duration, stopChan chan struct{}) {
	aggregatedHistoryMu.Lock()
	if aggregatedHistoryRunning {
		aggregatedHistoryMu.Unlock()
		return
	}
	aggregatedHistoryRunning = true
	aggregatedHistoryStopChan = stopChan
	aggregatedHistoryMu.Unlock()

	// 启动一个 goroutine 每隔一定时间刷新状态
	aggregatedHistoryWg.Add(1)
	go func() {
		defer aggregatedHistoryWg.Done()

		// 如果没有提供停止通道，创建一个内部的
		localStopChan := stopChan
		if localStopChan == nil {
			localStopChan = make(chan struct{})
		}

		// 使用配置的间隔，默认10秒
		interval := pollInterval
		if interval == 0 {
			interval = 10 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 查询所有状态为 "running" 或 "queued" 或 "archiving" 的聚合历史记录
				var histories []AggregatedHistory
				database.DB.Where("status IN ?", []string{"running", "queued", "pending", "archiving"}).Find(&histories)

				// 逐个更新它们的状态
				for i := range histories {
					RefreshAggregatedHistoryStatus(&histories[i], client)
				}
			case <-localStopChan:
				return
			}
		}
	}()
}

// StopAggregatedHistoryRefresh 停止聚合历史记录的定时刷新
func StopAggregatedHistoryRefresh() {
	aggregatedHistoryMu.Lock()
	defer aggregatedHistoryMu.Unlock()

	if aggregatedHistoryRunning && aggregatedHistoryStopChan != nil {
		close(aggregatedHistoryStopChan)
		aggregatedHistoryStopChan = nil
	}
	aggregatedHistoryWg.Wait()
	aggregatedHistoryRunning = false
}

// GetAggregatedHistoryConsoleLog 获取聚合历史的Jenkins控制台日志
func GetAggregatedHistoryConsoleLog(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var history AggregatedHistory
	if err := database.DB.Where("id = ?", id).First(&history).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "aggregated history not found"})
		return
	}

	// 检查是否有构建号
	buildNum := 0
	if history.JenkinsBuildNum != nil {
		buildNum = *history.JenkinsBuildNum
	}

	// 如果没有构建号但有队列ID，尝试获取构建号
	if buildNum == 0 && history.JenkinsQueueID != nil {
		jenkinsClient := jenkins.NewClient(
			cfg.Jenkins.URL,
			cfg.Jenkins.Username,
			cfg.Jenkins.Token,
			time.Duration(cfg.Jenkins.Timeout)*time.Second,
		)
		queueInfo, err := jenkinsClient.GetQueueItemInfo(*history.JenkinsQueueID)
		if err == nil && queueInfo != nil && queueInfo.Executable != nil {
			buildNum = queueInfo.Executable.Number
		}
	}

	if buildNum == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "构建尚未开始或构建号不可用"})
		return
	}

	// 创建Jenkins客户端获取日志
	jenkinsClient := jenkins.NewClient(
		cfg.Jenkins.URL,
		cfg.Jenkins.Username,
		cfg.Jenkins.Token,
		time.Duration(cfg.Jenkins.Timeout)*time.Second,
	)

	// 获取控制台日志
	consoleLog, err := jenkinsClient.GetConsoleLog(history.JenkinsJobName, buildNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取控制台日志失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"console_log": consoleLog,
		"build_num":   buildNum,
		"job_name":    history.JenkinsJobName,
		"console_url": fmt.Sprintf("%s/job/%s/%d/console", cfg.Jenkins.URL, history.JenkinsJobName, buildNum),
	})
}

// getLatestAggregatedPackageURL 获取最新的聚合包下载地址
func getLatestAggregatedPackageURL() string {
	return findLatestAggregatedPackageURL()
}

func resolveAggregatedHistoryFiles(history *AggregatedHistory, cfg *config.Config) ([]ArchiveFile, string, string, error) {
	if history == nil || history.DownloadURL == nil || strings.TrimSpace(*history.DownloadURL) == "" {
		return nil, "", "", fmt.Errorf("下载地址为空")
	}

	downloadURL := strings.TrimSpace(*history.DownloadURL)
	baseURL, fileName := splitAggregatedDownloadURL(downloadURL)
	timestamp := extractTimestampFromFilename(fileName)
	if timestamp == "" {
		timestamp = history.CreatedAt.Format("20060102150405")
	}

	files, err := fetchAggregatedFilesFromDirectory(baseURL, fileName, timestamp)
	if err == nil && len(files) > 0 {
		return files, baseURL, aggregatedResponseTimestamp(files, timestamp), nil
	}

	files = buildAggregatedFilesFromDownloadURL(downloadURL, fileName, timestamp)
	if len(files) > 0 {
		return files, baseURL, aggregatedResponseTimestamp(files, timestamp), nil
	}

	files = generateAggregatedFilesFromJenkins(history, cfg, baseURL, timestamp)
	return files, baseURL, aggregatedResponseTimestamp(files, timestamp), nil
}

func splitAggregatedDownloadURL(downloadURL string) (string, string) {
	parsed, err := neturl.Parse(downloadURL)
	if err != nil {
		return strings.TrimSuffix(downloadURL, "/"), ""
	}

	cleanPath := strings.TrimSuffix(parsed.Path, "/")
	if cleanPath == "" || cleanPath == "." || cleanPath == "/" {
		base := *parsed
		base.RawQuery = ""
		base.Fragment = ""
		return strings.TrimSuffix(base.String(), "/") + "/", ""
	}

	fileName := path.Base(cleanPath)
	dirPath := path.Dir(cleanPath)
	if dirPath == "." {
		dirPath = "/"
	}

	base := *parsed
	base.Path = strings.TrimSuffix(dirPath, "/") + "/"
	base.RawQuery = ""
	base.Fragment = ""
	return base.String(), fileName
}

func fetchAggregatedFilesFromDirectory(baseURL, targetName, timestamp string) ([]ArchiveFile, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("下载目录为空")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("无法访问目录: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("不是 HTML 目录列表")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)
	pattern := regexp.MustCompile(`(?s)<a\s+href="([^"]+)">[^<]+</a>\s+(\d{2}-[A-Za-z]{3}-\d{4})\s+(\d{2}:\d{2}(?::\d{2})?)\s+([\d.]+[KMGT]?[BBytes]?)`)
	matches := pattern.FindAllStringSubmatch(html, -1)

	files := make([]ArchiveFile, 0)
	seen := make(map[string]bool)
	targetStem := aggregatedFileStem(targetName)

	appendFile := func(name, sizeStr string) {
		if name == "" || name == "../" || strings.HasSuffix(name, "/") {
			return
		}
		if !isSupportedAggregatedFile(name) || seen[name] {
			return
		}
		if !matchesAggregatedFile(name, targetName, targetStem, timestamp) {
			return
		}

		seen[name] = true
		files = append(files, ArchiveFile{
			Name:      name,
			URL:       strings.TrimSuffix(baseURL, "/") + "/" + name,
			Timestamp: firstNonEmpty(extractTimestampFromFilename(name), timestamp),
			Size:      parseFileSize(sizeStr),
		})
	}

	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		appendFile(match[1], match[4])
	}

	if len(files) == 0 {
		simplePattern := regexp.MustCompile(`<a href="([^"]+\.(?:tar|tgz|tar\.gz|json))"`)
		simpleMatches := simplePattern.FindAllStringSubmatch(html, -1)
		for _, match := range simpleMatches {
			if len(match) < 2 {
				continue
			}
			appendFile(match[1], "")
		}
	}

	sortAggregatedFiles(files)
	return files, nil
}

func buildAggregatedFilesFromDownloadURL(downloadURL, fileName, timestamp string) []ArchiveFile {
	if downloadURL == "" {
		return nil
	}

	files := make([]ArchiveFile, 0, 2)
	seen := make(map[string]bool)
	appendURL := func(name, url string) {
		if name == "" || url == "" || seen[name] {
			return
		}
		seen[name] = true
		files = append(files, ArchiveFile{
			Name:      name,
			URL:       url,
			Timestamp: firstNonEmpty(extractTimestampFromFilename(name), timestamp),
			Size:      getFileSize(url),
		})
	}

	appendURL(firstNonEmpty(fileName, path.Base(downloadURL)), downloadURL)
	if companionName, companionURL := aggregatedCompanionJSON(downloadURL, fileName); companionName != "" {
		if size := getFileSize(companionURL); size > 0 {
			appendURL(companionName, companionURL)
		}
	}

	sortAggregatedFiles(files)
	return files
}

func generateAggregatedFilesFromJenkins(history *AggregatedHistory, cfg *config.Config, baseURL, fallbackTimestamp string) []ArchiveFile {
	if history == nil || cfg == nil || baseURL == "" {
		return nil
	}

	buildNum := 0
	if history.JenkinsBuildNum != nil {
		buildNum = *history.JenkinsBuildNum
	}
	if history.JenkinsJobName == "" || buildNum == 0 {
		return nil
	}

	jenkinsClient := jenkins.NewClient(
		cfg.Jenkins.URL,
		cfg.Jenkins.Username,
		cfg.Jenkins.Token,
		time.Duration(cfg.Jenkins.Timeout)*time.Second,
	)

	artifacts, err := jenkinsClient.GetBuildArtifacts(history.JenkinsJobName, buildNum)
	if err != nil || len(artifacts) == 0 {
		return nil
	}

	files := make([]ArchiveFile, 0, len(artifacts))
	seen := make(map[string]bool)
	for _, artifact := range artifacts {
		if !isSupportedAggregatedFile(artifact.Name) || seen[artifact.Name] {
			continue
		}
		seen[artifact.Name] = true
		fileURL := strings.TrimSuffix(baseURL, "/") + "/" + artifact.Name
		files = append(files, ArchiveFile{
			Name:      artifact.Name,
			URL:       fileURL,
			Timestamp: firstNonEmpty(extractTimestampFromFilename(artifact.Name), fallbackTimestamp),
			Size:      getFileSize(fileURL),
		})
	}

	sortAggregatedFiles(files)
	return files
}

func aggregatedCompanionJSON(downloadURL, fileName string) (string, string) {
	name := fileName
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		base := strings.TrimSuffix(name, ".tar.gz")
		return base + ".json", strings.TrimSuffix(downloadURL, ".tar.gz") + ".json"
	case strings.HasSuffix(name, ".tgz"):
		base := strings.TrimSuffix(name, ".tgz")
		return base + ".json", strings.TrimSuffix(downloadURL, ".tgz") + ".json"
	case strings.HasSuffix(name, ".tar"):
		base := strings.TrimSuffix(name, ".tar")
		return base + ".json", strings.TrimSuffix(downloadURL, ".tar") + ".json"
	default:
		return "", ""
	}
}

func matchesAggregatedFile(name, targetName, targetStem, timestamp string) bool {
	if targetName != "" && name == targetName {
		return true
	}

	if targetStem != "" && aggregatedFileStem(name) == targetStem {
		return true
	}

	if timestamp != "" && extractTimestampFromFilename(name) == timestamp {
		return true
	}

	return false
}

func aggregatedFileStem(name string) string {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		return strings.TrimSuffix(name, ".tar.gz")
	case strings.HasSuffix(name, ".tgz"):
		return strings.TrimSuffix(name, ".tgz")
	case strings.HasSuffix(name, ".tar"):
		return strings.TrimSuffix(name, ".tar")
	case strings.HasSuffix(name, ".json"):
		return strings.TrimSuffix(name, ".json")
	default:
		return name
	}
}

func isSupportedAggregatedFile(name string) bool {
	return strings.HasSuffix(name, ".tar") ||
		strings.HasSuffix(name, ".tgz") ||
		strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".json")
}

func aggregatedResponseTimestamp(files []ArchiveFile, fallback string) string {
	for _, file := range files {
		if file.Timestamp != "" {
			return file.Timestamp
		}
	}
	return fallback
}

func sortAggregatedFiles(files []ArchiveFile) {
	sort.SliceStable(files, func(i, j int) bool {
		iExt := aggregatedFilePriority(files[i].Name)
		jExt := aggregatedFilePriority(files[j].Name)
		if iExt != jExt {
			return iExt < jExt
		}
		return files[i].Name < files[j].Name
	})
}

func aggregatedFilePriority(name string) int {
	switch {
	case strings.HasSuffix(name, ".tar"), strings.HasSuffix(name, ".tgz"), strings.HasSuffix(name, ".tar.gz"):
		return 0
	case strings.HasSuffix(name, ".json"):
		return 1
	default:
		return 2
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}