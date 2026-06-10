// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package cmdb

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/internal/platformevent"
	"github.com/jenvenson/ops-platform/internal/tasks" // 新增导入
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/jenvenson/ops-platform/pkg/jenkins"
	"github.com/gin-gonic/gin"
)

// updateBaseURL 返回更新包下载的基础 URL（从环境变量读取）
var updateBaseURL = func() string {
	if u := os.Getenv("UPDATE_BASE_URL"); u != "" {
		return u
	}
	return "http://localhost:8888/update"
}()

// 辅助函数：将 uint 切片转换为逗号分隔的字符串
func intSliceToString(nums []uint) string {
	if len(nums) == 0 {
		return ""
	}
	strs := make([]string, len(nums))
	for i, n := range nums {
		strs[i] = strconv.FormatUint(uint64(n), 10)
	}
	return strings.Join(strs, ",")
}

// 辅助函数：将逗号分隔的字符串转换为 uint 切片
func stringToIntSlice(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.ParseUint(p, 10, 32); err == nil {
			result = append(result, uint(n))
		}
	}
	return result
}

func normalizeServerEnvIDs(envIDs []uint) ([]uint, uint) {
	normalized := make([]uint, 0, len(envIDs))
	seen := make(map[uint]struct{}, len(envIDs))
	for _, id := range envIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return nil, 0
	}
	return normalized, normalized[0]
}

type ServerRequest struct {
	Hostname   string `json:"hostname" binding:"required"`
	IP         string `json:"ip" binding:"required"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	SSHPort    int    `json:"ssh_port"`
	ProjectIDs []uint `json:"project_ids" binding:"required"`
	EnvIDs     []uint `json:"env_ids" binding:"required"`
}

type ServerUpdate struct {
	Hostname   string `json:"hostname,omitempty"`
	IP         string `json:"ip,omitempty"`
	OS         string `json:"os,omitempty"`
	Arch       string `json:"arch,omitempty"`
	Status     string `json:"status,omitempty"`
	SSHPort    int    `json:"ssh_port,omitempty"`
	ProjectIDs []uint `json:"project_ids,omitempty"`
	EnvIDs     []uint `json:"env_ids,omitempty"`
}

type EnvironmentRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=dev test prod"`
	Description string `json:"description"`
}

type ProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type ApplicationRequest struct {
	Name              string `json:"name" binding:"required"`
	CodeRepo          string `json:"code_repo"`
	DeployPath        string `json:"deploy_path"`
	JenkinsJob        string `json:"jenkins_job"`
	JenkinsArchiveJob string `json:"jenkins_archive_job"`
	EnvID             uint   `json:"env_id"`
	ProjectID         uint   `json:"project_id"`
}

// DeployTriggerRequest 应用部署触发请求
type DeployTriggerRequest struct {
	EnvID      uint   `json:"env_id" binding:"required"`
	DeployType string `json:"deploy_type" binding:"required,oneof=all frontend backend"`
}

// DeployTriggerResponse 应用部署触发响应
type DeployTriggerResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	DeployID uint   `json:"deploy_id,omitempty"`
	TaskID   int64  `json:"task_id,omitempty"`
}

// DeployRecordResponse 部署记录响应
type DeployRecordResponse struct {
	ID                uint   `json:"id"`
	AppID             uint   `json:"app_id"`
	AppName           string `json:"app_name"`
	EnvName           string `json:"env_name"`
	EnvType           string `json:"env_type"`
	ProjectCode       string `json:"project_code"`
	DeployType        string `json:"deploy_type"`
	JenkinsJob        string `json:"jenkins_job"`
	JenkinsBuildNum   int    `json:"jenkins_build_num"`
	JenkinsQueueID    int64  `json:"jenkins_queue_id"`
	JenkinsConsoleURL string `json:"jenkins_console_url,omitempty"`
	Status            string `json:"status"`
	ErrorMessage      string `json:"error_message,omitempty"`
	StartTime         string `json:"start_time,omitempty"`
	EndTime           string `json:"end_time,omitempty"`
	Duration          int    `json:"duration"`
	TriggeredBy       string `json:"triggered_by"`
	TriggeredByName   string `json:"triggered_by_name"`
	CreatedAt         string `json:"created_at"`
}

// DeployStatusRequest 部署状态查询请求
type DeployStatusRequest struct {
	BuildNumber int `json:"build_number" binding:"required"`
}

// ArchiveTriggerRequest 归档触发请求
type ArchiveTriggerRequest struct {
	EnvID      uint   `json:"env_id" binding:"required"`
	DeployType string `json:"deploy_type" binding:"required,oneof=all frontend backend"`
}

// ArchiveTriggerResponse 归档触发响应
type ArchiveTriggerResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	DeployID uint   `json:"deploy_id,omitempty"`
	TaskID   int64  `json:"task_id,omitempty"`
}

// ArchiveRecordResponse 归档记录响应
type ArchiveRecordResponse struct {
	ID                uint   `json:"id"`
	AppID             uint   `json:"app_id"`
	AppName           string `json:"app_name"`
	EnvName           string `json:"env_name"`
	EnvType           string `json:"env_type"`
	DeployType        string `json:"deploy_type"`
	JenkinsJob        string `json:"jenkins_job"`
	JenkinsBuildNum   int    `json:"jenkins_build_num"`
	JenkinsQueueID    int64  `json:"jenkins_queue_id"`
	JenkinsConsoleURL string `json:"jenkins_console_url,omitempty"`
	DownloadURL       string `json:"download_url,omitempty"`
	Status            string `json:"status"`
	ErrorMessage      string `json:"error_message,omitempty"`
	StartTime         string `json:"start_time,omitempty"`
	EndTime           string `json:"end_time,omitempty"`
	CreatedAt         string `json:"created_at"`
	Operator          string `json:"operator,omitempty"`
}

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	api := r.Group("/api/cmdb")
	api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		// 服务器
		api.GET("/servers", GetServers)
		api.GET("/servers/:id", GetServer)
		api.POST("/servers", CreateServer)
		api.PUT("/servers/:id", UpdateServer)
		api.DELETE("/servers/:id", DeleteServer)

		// 环境
		api.GET("/environments", GetEnvironments)
		api.GET("/environments/:id", GetEnvironment)
		api.POST("/environments", CreateEnvironment)
		api.PUT("/environments/:id", UpdateEnvironment)
		api.DELETE("/environments/:id", DeleteEnvironment)

		// 项目
		api.GET("/projects", GetProjects)
		api.GET("/projects/:id", GetProject)
		api.POST("/projects", CreateProject)
		api.PUT("/projects/:id", UpdateProject)
		api.DELETE("/projects/:id", DeleteProject)

		// 应用
		api.GET("/applications", GetApplications)
		api.GET("/applications/:id", GetApplication)
		api.POST("/applications", CreateApplication)
		api.PUT("/applications/:id", UpdateApplication)
		api.DELETE("/applications/:id", DeleteApplication)
		api.POST("/applications/:id/deploy", func(c *gin.Context) {
			TriggerApplicationDeploy(c, cfg)
		})

		// Jenkins 集成
		api.GET("/jenkins/views/:name/jobs", func(c *gin.Context) {
			GetJenkinsViewJobs(c, cfg)
		})
		api.POST("/jenkins/import", func(c *gin.Context) {
			ImportJenkinsJobs(c, cfg)
		})
		api.POST("/jenkins/copy-view", func(c *gin.Context) {
			CopyJenkinsView(c, cfg)
		})
		api.DELETE("/jenkins/jobs/:name", func(c *gin.Context) {
			DeleteJenkinsJob(c, cfg)
		})
		api.POST("/jenkins/delete-jobs", func(c *gin.Context) {
			BatchDeleteJenkinsJobs(c, cfg)
		})
		api.DELETE("/jenkins/views/:name", func(c *gin.Context) {
			DeleteJenkinsView(c, cfg)
		})
		api.POST("/jenkins/credentials", func(c *gin.Context) {
			CreateJenkinsCredential(c, cfg)
		})

		// 任务状态API
		api.GET("/tasks/:id", GetTaskStatus)

		// 部署历史
		api.GET("/deploy-records", func(c *gin.Context) {
			GetDeployRecords(c, cfg)
		})
		api.GET("/deploy-records/:id", func(c *gin.Context) {
			GetDeployRecord(c, cfg)
		})
		api.GET("/deploy-records/:id/status", func(c *gin.Context) {
			GetDeployStatus(c, cfg)
		})
		api.DELETE("/deploy-records/:id", func(c *gin.Context) {
			DeleteDeployRecord(c)
		})

		// 应用归档
		api.POST("/applications/:id/archive", func(c *gin.Context) {
			TriggerApplicationArchive(c, cfg)
		})

		// 归档历史
		api.GET("/archive-records", func(c *gin.Context) {
			GetArchiveRecords(c, cfg)
		})
		api.GET("/archive-records/:id/status", func(c *gin.Context) {
			GetArchiveStatus(c, cfg)
		})
		api.GET("/archive-records/:id/download", func(c *gin.Context) {
			GetArchiveDownloadURL(c, cfg)
		})
		api.GET("/archive-records/:id/files", func(c *gin.Context) {
			GetArchiveFiles(c, cfg)
		})
		api.DELETE("/archive-records/:id", func(c *gin.Context) {
			DeleteArchiveRecord(c)
		})

		// 聚合历史路由
		api.GET("/aggregated-histories", GetAggregatedHistories)
		api.GET("/aggregated-histories/:id", GetAggregatedHistory)
		api.DELETE("/aggregated-histories/:id", DeleteAggregatedHistory)
		api.GET("/aggregated-histories/:id/status", func(c *gin.Context) {
			GetAggregatedHistoryStatus(c, cfg)
		})
		api.GET("/aggregated-histories/:id/files", func(c *gin.Context) {
			GetAggregatedHistoryFiles(c, cfg)
		})
		api.GET("/aggregated-histories/:id/console-log", func(c *gin.Context) {
			GetAggregatedHistoryConsoleLog(c, cfg)
		})
	}

	// 添加额外的路由，支持前端习惯使用的路径
	extraApi := r.Group("/api")
	extraApi.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		extraApi.GET("/jenkins/views/:name/jobs", func(c *gin.Context) {
			GetJenkinsViewJobs(c, cfg)
		})
		// 添加任务状态API到额外路由组
		extraApi.GET("/tasks/:id", GetTaskStatus)
	}

	// 聚合打包路由（暂未实现）
	// api.POST("/deploy/aggregate-package", func(c *gin.Context) {
	// 	TriggerAggregatePackage(c, cfg)
	// })
	// api.GET("/deploy/aggregate-packages", func(c *gin.Context) {
	// 	GetAggregatePackageRecords(c, cfg)
	// })
	// api.GET("/deploy/aggregate-packages/:id", func(c *gin.Context) {
	// 	GetAggregatePackageRecord(c, cfg)
	// })
	// api.GET("/deploy/aggregate-packages/:id/status", func(c *gin.Context) {
	// 	GetAggregatePackageStatus(c, cfg)
	// })
}

// ========== 服务器 CRUD ==========

func GetServers(c *gin.Context) {
	envID, _ := strconv.Atoi(c.Query("env_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	countQuery := database.DB.Model(&Server{}).Where("deleted_at IS NULL")
	query := database.DB.Where("deleted_at IS NULL")
	if envID > 0 {
		countQuery = countQuery.Where("env_id = ?", envID)
		query = query.Where("env_id = ?", envID)
	}

	var total int64
	countQuery.Count(&total)

	var servers []Server
	offset := (page - 1) * limit
	if err := query.Preload("Projects").
		Offset(offset).Limit(limit).Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch servers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  servers,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func GetServer(c *gin.Context) {
	id := c.Param("id")

	var server Server
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Projects").First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	c.JSON(http.StatusOK, server)
}

func CreateServer(c *gin.Context) {
	var req ServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	envIDs, primaryEnvID := normalizeServerEnvIDs(req.EnvIDs)
	if len(envIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "env_ids is required"})
		return
	}

	var envCount int64
	if err := database.DB.Model(&Environment{}).Where("id IN ?", envIDs).Count(&envCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate env_ids"})
		return
	}
	if envCount != int64(len(envIDs)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid env_ids"})
		return
	}

	// 获取项目对象
	var projects []Project
	if err := database.DB.Where("id IN ?", req.ProjectIDs).Find(&projects).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_ids"})
		return
	}

	// 将环境ID数组转换为逗号分隔的字符串
	envIDsStr := intSliceToString(envIDs)

	server := Server{
		Hostname: req.Hostname,
		IP:       req.IP,
		OS:       req.OS,
		Arch:     req.Arch,
		SSHPort:  req.SSHPort,
		EnvID:    primaryEnvID,
		EnvIDs:   envIDsStr,
		Status:   "offline",
		Projects: projects,
	}

	if err := database.DB.Create(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create server"})
		return
	}

	c.JSON(http.StatusCreated, server)
}

func UpdateServer(c *gin.Context) {
	id := c.Param("id")

	var req ServerUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var server Server
	if err := database.DB.Preload("Projects").Where("id = ? AND deleted_at IS NULL", id).First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	if req.Hostname != "" {
		server.Hostname = req.Hostname
	}
	if req.IP != "" {
		server.IP = req.IP
	}
	if req.OS != "" {
		server.OS = req.OS
	}
	if req.Arch != "" {
		server.Arch = req.Arch
	}
	if req.Status != "" {
		server.Status = req.Status
	}
	if req.SSHPort != 0 {
		server.SSHPort = req.SSHPort
	}
	if len(req.EnvIDs) > 0 {
		envIDs, primaryEnvID := normalizeServerEnvIDs(req.EnvIDs)
		if len(envIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "env_ids is required"})
			return
		}
		var envCount int64
		if err := database.DB.Model(&Environment{}).Where("id IN ?", envIDs).Count(&envCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate env_ids"})
			return
		}
		if envCount != int64(len(envIDs)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid env_ids"})
			return
		}
		server.EnvID = primaryEnvID
		server.EnvIDs = intSliceToString(envIDs)
	}

	// 更新项目关联
	if len(req.ProjectIDs) > 0 {
		var projects []Project
		if err := database.DB.Where("id IN ?", req.ProjectIDs).Find(&projects).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_ids"})
			return
		}
		if err := database.DB.Model(&server).Association("Projects").Replace(projects); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update projects"})
			return
		}
	}

	if err := database.DB.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update server"})
		return
	}

	// 重新加载以返回完整数据
	database.DB.Preload("Projects").First(&server, server.ID)

	c.JSON(http.StatusOK, server)
}

func DeleteServer(c *gin.Context) {
	id := c.Param("id")

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	if err := tx.Exec("DELETE FROM server_projects WHERE server_id = ?", id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	if err := tx.Exec("DELETE FROM server_apps WHERE server_id = ?", id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	if err := tx.Delete(&Server{}, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "server deleted"})
}

// ========== 环境 CRUD ==========

func GetEnvironments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var total int64
	database.DB.Model(&Environment{}).Where("deleted_at IS NULL").Count(&total)

	var environments []Environment
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Offset(offset).Limit(limit).Find(&environments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch environments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  environments,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func GetEnvironment(c *gin.Context) {
	id := c.Param("id")

	var environment Environment
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&environment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}

	c.JSON(http.StatusOK, environment)
}

func CreateEnvironment(c *gin.Context) {
	var req EnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	environment := Environment{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
	}

	if err := database.DB.Create(&environment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create environment"})
		return
	}

	c.JSON(http.StatusCreated, environment)
}

func UpdateEnvironment(c *gin.Context) {
	id := c.Param("id")

	var req EnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var environment Environment
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&environment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}

	if req.Name != "" {
		environment.Name = req.Name
	}
	if req.Type != "" {
		environment.Type = req.Type
	}
	if req.Description != "" {
		environment.Description = req.Description
	}

	if err := database.DB.Save(&environment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update environment"})
		return
	}

	c.JSON(http.StatusOK, environment)
}

func DeleteEnvironment(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&Environment{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete environment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "environment deleted"})
}

// ========== 项目 CRUD ==========

func GetProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var total int64
	database.DB.Model(&Project{}).Where("deleted_at IS NULL").Count(&total)

	var projects []Project
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  projects,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func GetProject(c *gin.Context) {
	id := c.Param("id")

	var project Project
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func CreateProject(c *gin.Context) {
	var req ProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	project := Project{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func UpdateProject(c *gin.Context) {
	id := c.Param("id")

	var req ProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var project Project
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Code != "" {
		project.Code = req.Code
	}
	if req.Description != "" {
		project.Description = req.Description
	}

	if err := database.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func DeleteProject(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&Project{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "project deleted"})
}

// ========== 应用 CRUD ==========

func GetApplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var total int64
	database.DB.Model(&Application{}).Where("deleted_at IS NULL").Count(&total)

	var apps []Application
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Preload("Environment").
		Preload("Project").
		Offset(offset).Limit(limit).Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  apps,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func GetApplication(c *gin.Context) {
	id := c.Param("id")

	var app Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Environment").
		Preload("Project").
		First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	c.JSON(http.StatusOK, app)
}

func CreateApplication(c *gin.Context) {
	var req ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	app := Application{
		Name:              req.Name,
		CodeRepo:          req.CodeRepo,
		DeployPath:        req.DeployPath,
		JenkinsJob:        req.JenkinsJob,
		JenkinsArchiveJob: req.JenkinsArchiveJob,
		EnvID:             req.EnvID,
		ProjectID:         req.ProjectID,
	}

	if err := database.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create application"})
		return
	}

	// 重新加载以返回完整数据
	database.DB.Preload("Environment").Preload("Project").First(&app, app.ID)

	c.JSON(http.StatusCreated, app)
}

func UpdateApplication(c *gin.Context) {
	id := c.Param("id")

	var req ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var app Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// 直接更新所有字段
	if req.Name != "" {
		app.Name = req.Name
	}
	if req.CodeRepo != "" {
		app.CodeRepo = req.CodeRepo
	}
	app.DeployPath = req.DeployPath
	app.JenkinsJob = req.JenkinsJob
	app.JenkinsArchiveJob = req.JenkinsArchiveJob
	app.EnvID = req.EnvID
	app.ProjectID = req.ProjectID

	if err := database.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update application"})
		return
	}

	// 重新加载以返回完整数据
	database.DB.Preload("Environment").Preload("Project").First(&app, app.ID)

	c.JSON(http.StatusOK, app)
}

func DeleteApplication(c *gin.Context) {
	id := c.Param("id")

	// 先删除关联的部署记录和归档记录，解除外键约束
	database.DB.Exec("DELETE FROM deploy_records WHERE app_id = ?", id)
	database.DB.Exec("DELETE FROM archive_records WHERE app_id = ?", id)

	if err := database.DB.Delete(&Application{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application deleted"})
}

// TriggerApplicationDeploy 触发应用部署
func TriggerApplicationDeploy(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var req DeployTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var app Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Environment").Preload("Project").First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// 检查是否配置了 Jenkins Job
	if app.JenkinsJob == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用未配置 Jenkins Job"})
		return
	}

	// 获取 Jenkins 配置
	jenkinsURL := cfg.Jenkins.URL
	jenkinsUser := cfg.Jenkins.Username
	jenkinsToken := cfg.Jenkins.Token

	if jenkinsURL == "" || jenkinsUser == "" || jenkinsToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	// 创建 Jenkins 客户端
	jenkinsClient := jenkins.NewClient(jenkinsURL, jenkinsUser, jenkinsToken, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	// 从完整 URL 中提取 job 名称
	jobName := app.JenkinsJob
	if strings.Contains(jobName, "/job/") {
		// 从 URL 中提取最后的 job 名称
		parts := strings.Split(jobName, "/")
		jobName = parts[len(parts)-1]
	}

	// 构建环境名称: 直接使用应用关联的环境名称字段
	envName := ""
	if app.Environment != nil {
		envName = app.Environment.Name
	}

	// 构建参数
	params := jenkins.BuildParams{
		APP:   app.Name,
		TAG:   envName,
		SCOPE: req.DeployType,
	}

	// 打印部署参数用于调试
	fmt.Printf("[Deploy] 应用: %s, JenkinsJob: %s, 传递的TAG: %s\n",
		app.Name, jobName, envName)

	// 触发 Jenkins 构建
	queueID, _, err := jenkinsClient.TriggerBuild(jobName, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("触发 Jenkins 构建失败: %v", err)})
		return
	}

	// 保存部署记录
	now := time.Now()
	projectCode := ""
	if app.Project != nil {
		projectCode = app.Project.Code
	}
	envNameStr := ""
	envType := ""
	if app.Environment != nil {
		envNameStr = app.Environment.Name
		envType = app.Environment.Type
	}

	deployRecord := DeployRecord{
		AppID:          app.ID,
		AppName:        app.Name,
		EnvID:          app.EnvID,
		EnvName:        envNameStr,
		EnvType:        envType,
		ProjectCode:    projectCode,
		DeployType:     req.DeployType,
		JenkinsJob:     jobName,
		JenkinsQueueID: queueID,
		Status:         "running",
		StartTime:      &now,
		TriggeredBy:    c.GetString("username"),
	}

	if err := database.DB.Create(&deployRecord).Error; err != nil {
		// 记录保存失败但部署已触发，返回部署信息
		fmt.Printf("[Deploy] 保存部署记录失败: %v\n", err)
	}
	_ = platformevent.RecordDeployRecord(toDeployEventPayload(deployRecord))

	fmt.Printf("[Deploy] 部署记录已创建，ID: %d, QueueID: %d\n", deployRecord.ID, queueID)

	c.JSON(http.StatusOK, DeployTriggerResponse{
		Success:  true,
		Message:  fmt.Sprintf("部署任务已提交，Jenkins Queue ID: %d", queueID),
		DeployID: deployRecord.ID,
		TaskID:   queueID,
	})
}

// ========== 部署历史 API ==========

// formatTime 格式化时间
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func toDeployEventPayload(record DeployRecord) platformevent.DeployRecordPayload {
	return platformevent.DeployRecordPayload{
		ID:           record.ID,
		AppID:        record.AppID,
		AppName:      record.AppName,
		EnvID:        record.EnvID,
		EnvName:      record.EnvName,
		ProjectCode:  record.ProjectCode,
		DeployType:   record.DeployType,
		Status:       record.Status,
		ErrorMessage: record.ErrorMessage,
		StartTime:    record.StartTime,
		EndTime:      record.EndTime,
		TriggeredBy:  record.TriggeredBy,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toArchiveEventPayload(record ArchiveRecord) platformevent.ArchiveRecordPayload {
	return platformevent.ArchiveRecordPayload{
		ID:           record.ID,
		AppID:        record.AppID,
		AppName:      record.AppName,
		EnvID:        record.EnvID,
		EnvName:      record.EnvName,
		ProjectCode:  record.ProjectCode,
		DeployType:   record.DeployType,
		Status:       record.Status,
		ErrorMessage: record.ErrorMessage,
		StartTime:    record.StartTime,
		EndTime:      record.EndTime,
		Operator:     record.Operator,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

// resolveRealName 根据用户名查询真实姓名
func resolveRealName(username string) string {
	if username == "" || username == "system" {
		return username
	}
	var user models.User
	if err := database.DB.Select("real_name").Where("username = ?", username).First(&user).Error; err == nil && user.RealName != "" {
		return user.RealName
	}
	return username
}

// toDeployRecordResponse 转换为 API 响应格式
func toDeployRecordResponse(record *DeployRecord, jenkinsBaseURL string) *DeployRecordResponse {
	if record == nil {
		return nil
	}

	// 生成 Jenkins 控制台日志链接
	consoleURL := ""
	if record.JenkinsJob != "" && record.JenkinsBuildNum > 0 {
		consoleURL = fmt.Sprintf("%s/job/%s/%d/console", jenkinsBaseURL, record.JenkinsJob, record.JenkinsBuildNum)
	}

	return &DeployRecordResponse{
		ID:                record.ID,
		AppID:             record.AppID,
		AppName:           record.AppName,
		EnvName:           record.EnvName,
		EnvType:           record.EnvType,
		ProjectCode:       record.ProjectCode,
		DeployType:        record.DeployType,
		JenkinsJob:        record.JenkinsJob,
		JenkinsBuildNum:   record.JenkinsBuildNum,
		JenkinsQueueID:    record.JenkinsQueueID,
		JenkinsConsoleURL: consoleURL,
		Status:            record.Status,
		ErrorMessage:      record.ErrorMessage,
		StartTime:         formatTime(record.StartTime),
		EndTime:           formatTime(record.EndTime),
		Duration:          record.Duration,
		TriggeredBy:       record.TriggeredBy,
		TriggeredByName:   resolveRealName(record.TriggeredBy),
		CreatedAt:         formatTime(&record.CreatedAt),
	}
}

// GetDeployRecords 获取部署历史列表
func GetDeployRecords(c *gin.Context, cfg *config.Config) {
	appID, _ := strconv.Atoi(c.Query("app_id"))
	appName := c.Query("app_name") // 支持按应用名称查询
	envID, _ := strconv.Atoi(c.Query("env_id"))
	status := c.Query("status")
	triggeredBy := c.Query("triggered_by")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := database.DB.Model(&DeployRecord{})

	if appID > 0 {
		query = query.Where("app_id = ?", appID)
	}
	if appName != "" {
		query = query.Where("app_name LIKE ?", "%"+appName+"%")
	}
	if envID > 0 {
		query = query.Where("env_id = ?", envID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if triggeredBy != "" {
		query = query.Where("triggered_by LIKE ?", "%"+triggeredBy+"%")
	}
	// 时间范围筛选
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	query.Count(&total)

	var records []DeployRecord
	offset := (page - 1) * limit
	if err := query.Preload("App").Preload("Environment").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch deploy records"})
		return
	}

	// 转换为响应格式
	responses := make([]*DeployRecordResponse, len(records))
	for i, record := range records {
		responses[i] = toDeployRecordResponse(&record, cfg.Jenkins.URL)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// refreshDeployRecordStatus 刷新单个部署记录的状态
func refreshDeployRecordStatus(record *DeployRecord, client *jenkins.Client) {
	buildNum := record.JenkinsBuildNum

	// 如果没有 build number，尝试通过 queue ID 获取
	if buildNum == 0 && record.JenkinsQueueID > 0 {
		queueInfo, err := client.GetQueueItemInfo(record.JenkinsQueueID)
		if err == nil && queueInfo != nil && queueInfo.Executable != nil {
			buildNum = queueInfo.Executable.Number
			record.JenkinsBuildNum = buildNum
			database.DB.Model(record).Update("jenkins_build_num", buildNum)
		} else if err == nil && queueInfo == nil {
			// 队列项已消失，从 Job 获取
			jobInfo, err := client.GetJobInfo(record.JenkinsJob)
			if err == nil && jobInfo.LastBuild != nil {
				buildNum = jobInfo.LastBuild.Number
				record.JenkinsBuildNum = buildNum
				database.DB.Model(record).Update("jenkins_build_num", buildNum)
			}
		}
	}

	if buildNum == 0 {
		return // 仍在排队
	}

	// 获取构建状态
	buildStatus, err := client.GetBuildStatus(record.JenkinsJob, buildNum)
	if err != nil || buildStatus == nil {
		return
	}

	now := time.Now()
	updated := false

	// 根据 Jenkins 结果更新状态
	switch buildStatus.Phase {
	case "COMPLETED", "FINALIZED":
		if buildStatus.Result == "SUCCESS" {
			record.Status = "success"
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		// 使用 Jenkins 返回的构建耗时（毫秒转秒）
		if buildStatus.Duration > 0 {
			record.Duration = int(buildStatus.Duration / 1000)
		}
		updated = true
	case "": // 构建已完成，phase 为空
		if buildStatus.Result == "SUCCESS" {
			record.Status = "success"
		} else if buildStatus.Result == "FAILURE" {
			record.Status = "failed"
		}
		if record.Status != "running" {
			record.EndTime = &now
			// 使用 Jenkins 返回的构建耗时（毫秒转秒）
			if buildStatus.Duration > 0 {
				record.Duration = int(buildStatus.Duration / 1000)
			}
			updated = true
		}
	}

	if updated {
		database.DB.Model(record).Updates(gin.H{
			"status":   record.Status,
			"end_time": record.EndTime,
			"duration": record.Duration,
		})
	}
}

// GetDeployRecord 获取单个部署记录
func GetDeployRecord(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var record DeployRecord
	if err := database.DB.Where("id = ?", id).
		Preload("App").
		Preload("Environment").
		First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deploy record not found"})
		return
	}

	c.JSON(http.StatusOK, toDeployRecordResponse(&record, cfg.Jenkins.URL))
}

// DeleteDeployRecord 删除部署记录
func DeleteDeployRecord(c *gin.Context) {
	id := c.Param("id")

	var record DeployRecord
	if err := database.DB.Where("id = ?", id).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deploy record not found"})
		return
	}

	// 软删除
	if err := database.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete deploy record"})
		return
	}
	_ = platformevent.RecordDeployRecordDeleted(toDeployEventPayload(record))

	c.JSON(http.StatusOK, gin.H{"message": "deploy record deleted successfully"})
}

// GetDeployStatus 获取部署状态（从 Jenkins 更新）
func GetDeployStatus(c *gin.Context, cfgParam *config.Config) {
	id := c.Param("id")

	var record DeployRecord
	if err := database.DB.Where("id = ?", id).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deploy record not found"})
		return
	}

	// 创建 Jenkins 客户端
	jenkinsClient := jenkins.NewClient(
		cfgParam.Jenkins.URL,
		cfgParam.Jenkins.Username,
		cfgParam.Jenkins.Token,
		time.Duration(cfgParam.Jenkins.Timeout)*time.Second,
	)

	// 如果构建号为空但有 queue ID，先尝试通过 queue ID 获取 build number
	buildNum := record.JenkinsBuildNum
	if buildNum == 0 && record.JenkinsQueueID > 0 {
		queueInfo, err := jenkinsClient.GetQueueItemInfo(record.JenkinsQueueID)
		if err == nil && queueInfo != nil && queueInfo.Executable != nil {
			// 获取到 build number，更新记录
			buildNum = queueInfo.Executable.Number
			database.DB.Model(&record).Update("jenkins_build_num", buildNum)
			record.JenkinsBuildNum = buildNum
		} else if err == nil && queueInfo == nil {
			// 队列项已消失，构建已开始，尝试从 Job 获取最新的 build
			jobInfo, err := jenkinsClient.GetJobInfo(record.JenkinsJob)
			if err == nil && jobInfo.LastBuild != nil {
				buildNum = jobInfo.LastBuild.Number
				record.JenkinsBuildNum = buildNum
				database.DB.Model(&record).Update("jenkins_build_num", buildNum)
			}
		}
	}

	// 如果仍然没有 build number，返回排队状态
	if buildNum == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":   "queued",
			"queue_id": record.JenkinsQueueID,
			"message":  "构建任务已在队列中等待",
		})
		return
	}

	// 从 Jenkins 获取构建状态
	status, err := jenkinsClient.GetBuildStatus(record.JenkinsJob, buildNum)
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
		record.Status = "queued"
	case "STARTED":
		record.Status = "running"
	case "COMPLETED":
		if result == "SUCCESS" {
			record.Status = "success"
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		// 使用 Jenkins 返回的构建耗时（毫秒转秒）
		if status.Duration > 0 {
			record.Duration = int(status.Duration / 1000)
		}
		updated = true
	case "FINALIZED":
		if result == "SUCCESS" {
			record.Status = "success"
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		if status.Duration > 0 {
			record.Duration = int(status.Duration / 1000)
		}
		updated = true
	case "": // 构建已完成，phase 可能为空
		fallthrough
	default:
		// 根据 result 字段判断结果
		if result == "SUCCESS" {
			record.Status = "success"
			record.EndTime = &now
			if status.Duration > 0 {
				record.Duration = int(status.Duration / 1000)
			}
			updated = true
		} else if result == "FAILURE" {
			record.Status = "failed"
			record.EndTime = &now
			if status.Duration > 0 {
				record.Duration = int(status.Duration / 1000)
			}
			updated = true
		} else if phase == "STARTED" || phase == "RUNNING" {
			record.Status = "running"
		}
	}

	if updated {
		// 使用 Save 方法确保更新到数据库
		if err := database.DB.Save(&record).Error; err != nil {
			fmt.Printf("[DeployStatus] 更新数据库失败: %v\n", err)
		} else {
			fmt.Printf("[DeployStatus] 数据库更新成功: id=%d, status=%s, duration=%d\n",
				record.ID, record.Status, record.Duration)
			_ = platformevent.RecordDeployRecord(toDeployEventPayload(record))
		}
	}

	// 记录日志用于调试
	fmt.Printf("[DeployStatus] id=%d, phase=%s, result=%s, duration=%d, updated=%v\n",
		record.ID, phase, result, status.Duration, updated)

	// 返回从 Jenkins 获取的最新状态，而不是数据库中的旧状态
	c.JSON(http.StatusOK, gin.H{
		"id":           record.ID,
		"status":       record.Status,
		"build_number": buildNum,
		"phase":        status.Phase,
		"jenkins_url":  status.URL,
		"console_url":  fmt.Sprintf("%s/job/%s/%d/console", cfgParam.Jenkins.URL, record.JenkinsJob, buildNum),
		"timestamp":    status.Timestamp,
	})
}

// ========== 归档 API ==========

// TriggerApplicationArchive 触发应用归档
func TriggerApplicationArchive(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var req ArchiveTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var app Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Environment").Preload("Project").First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// 检查是否配置了 Jenkins Job
	if app.JenkinsJob == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用未配置 Jenkins Job"})
		return
	}

	// 获取 Jenkins 配置
	jenkinsURL := cfg.Jenkins.URL
	jenkinsUser := cfg.Jenkins.Username
	jenkinsToken := cfg.Jenkins.Token

	if jenkinsURL == "" || jenkinsUser == "" || jenkinsToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	// 创建 Jenkins 客户端
	jenkinsClient := jenkins.NewClient(jenkinsURL, jenkinsUser, jenkinsToken, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	// 从完整 URL 中提取 job 名称（使用归档流水线）
	jobName := app.JenkinsArchiveJob
	if jobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用未配置 Jenkins 归档流水线"})
		return
	}
	if strings.Contains(jobName, "/job/") {
		parts := strings.Split(jobName, "/")
		jobName = parts[len(parts)-1]
	}

	// 构建环境名称
	envName := ""
	if app.Environment != nil {
		envName = app.Environment.Name
	}

	// 查询 Consul KV 路径替换规则
	var consulPrefix string
	var replaceRule models.ConsulKVReplaceRule
	ruleErr := database.DB.Where("app_id = ?", app.ID).First(&replaceRule).Error
	if ruleErr == nil {
		// 使用自定义路径前缀
		consulPrefix = replaceRule.ConsulPathPrefix
		fmt.Printf("[Archive] 使用自定义 Consul 路径前缀: %s\n", consulPrefix)
	} else {
		// 使用默认路径前缀
		consulPrefix = fmt.Sprintf("plugin/%s", app.Name)
		fmt.Printf("[Archive] 使用默认 Consul 路径前缀: %s\n", consulPrefix)
	}

	// 构建参数
	params := jenkins.BuildParams{
		APP:   app.Name,
		TAG:   envName,
		SCOPE: req.DeployType,
	}

	// 验证参数不为空
	if params.APP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用名称不能为空"})
		return
	}
	if params.TAG == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "环境名称不能为空"})
		return
	}

	fmt.Printf("[Archive] 应用: %s, JenkinsArchiveJob: %s, 传递的参数: APP=%s, TAG=%s, SCOPE=%s, ConsulPrefix=%s\n",
		app.Name, jobName, params.APP, params.TAG, params.SCOPE, consulPrefix)

	// 触发 Jenkins 构建
	queueID, _, err := jenkinsClient.TriggerBuild(jobName, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("触发 Jenkins 构建失败: %v", err)})
		return
	}

	// 保存归档记录
	now := time.Now()
	envNameStr := ""
	envType := ""
	if app.Environment != nil {
		envNameStr = app.Environment.Name
		envType = app.Environment.Type
	}

	// 获取项目代码用于下载地址
	projectCode := ""
	if app.Project != nil {
		projectCode = app.Project.Code
	}
	if projectCode == "" {
		projectCode = "unknown"
	}

	archiveRecord := ArchiveRecord{
		AppID:          app.ID,
		AppName:        app.Name,
		EnvID:          app.EnvID,
		EnvName:        envNameStr,
		EnvType:        envType,
		DeployType:     req.DeployType,
		ProjectCode:    projectCode,
		JenkinsJob:     jobName,
		JenkinsQueueID: queueID,
		Status:         "running",
		StartTime:      &now,
		Operator:       c.GetString("real_name"),
	}

	if err := database.DB.Create(&archiveRecord).Error; err != nil {
		fmt.Printf("[Archive] 保存归档记录失败: %v\n", err)
	}
	_ = platformevent.RecordArchiveRecord(toArchiveEventPayload(archiveRecord))

	fmt.Printf("[Archive] 归档记录已创建，ID: %d, QueueID: %d\n", archiveRecord.ID, queueID)

	c.JSON(http.StatusOK, ArchiveTriggerResponse{
		Success:  true,
		Message:  fmt.Sprintf("归档任务已提交，Jenkins Queue ID: %d", queueID),
		DeployID: archiveRecord.ID,
		TaskID:   queueID,
	})
}

// toArchiveRecordResponse 转换为归档记录响应格式
func toArchiveRecordResponse(record *ArchiveRecord, jenkinsBaseURL string) *ArchiveRecordResponse {
	if record == nil {
		return nil
	}

	// 生成 Jenkins 控制台日志链接
	consoleURL := ""
	if record.JenkinsJob != "" && record.JenkinsBuildNum > 0 {
		consoleURL = fmt.Sprintf("%s/job/%s/%d/console", jenkinsBaseURL, record.JenkinsJob, record.JenkinsBuildNum)
	}

	return &ArchiveRecordResponse{
		ID:                record.ID,
		AppID:             record.AppID,
		AppName:           record.AppName,
		EnvName:           record.EnvName,
		EnvType:           record.EnvType,
		DeployType:        record.DeployType,
		JenkinsJob:        record.JenkinsJob,
		JenkinsBuildNum:   record.JenkinsBuildNum,
		JenkinsQueueID:    record.JenkinsQueueID,
		JenkinsConsoleURL: consoleURL,
		DownloadURL:       record.DownloadURL,
		Status:            record.Status,
		ErrorMessage:      record.ErrorMessage,
		StartTime:         formatTime(record.StartTime),
		EndTime:           formatTime(record.EndTime),
		CreatedAt:         formatTime(&record.CreatedAt),
		Operator:          record.Operator,
	}
}

// GetArchiveRecords 获取归档历史列表
func GetArchiveRecords(c *gin.Context, cfg *config.Config) {
	appID, _ := strconv.Atoi(c.Query("app_id"))
	appName := c.Query("app_name")
	envID, _ := strconv.Atoi(c.Query("env_id"))
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := database.DB.Model(&ArchiveRecord{})

	if appID > 0 {
		query = query.Where("app_id = ?", appID)
	}
	if appName != "" {
		query = query.Where("app_name LIKE ?", "%"+appName+"%")
	}
	if envID > 0 {
		query = query.Where("env_id = ?", envID)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	query.Count(&total)

	var records []ArchiveRecord
	offset := (page - 1) * limit
	if err := query.Preload("App").Preload("Environment").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch archive records"})
		return
	}

	// 转换为响应格式
	responses := make([]*ArchiveRecordResponse, len(records))
	for i, record := range records {
		responses[i] = toArchiveRecordResponse(&record, cfg.Jenkins.URL)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// refreshArchiveRecordStatus 刷新单个归档记录的状态
func refreshArchiveRecordStatus(record *ArchiveRecord, client *jenkins.Client) {
	buildNum := resolveArchiveBuildNumber(record, client)

	if buildNum == 0 {
		return
	}

	buildStatus, err := client.GetBuildStatus(record.JenkinsJob, buildNum)
	if err != nil || buildStatus == nil {
		return
	}

	now := time.Now()
	updated := false

	switch buildStatus.Phase {
	case "COMPLETED", "FINALIZED":
		if buildStatus.Result == "SUCCESS" {
			record.Status = "success"
			// 生成下载地址: http://your-update-server/update/example/20260129/
			envName := record.EnvName
			if envName == "" {
				envName = "unknown"
			}
			dateStr := now.Format("20060102")
			record.DownloadURL = fmt.Sprintf("%s/%s/%s/", updateBaseURL, envName, dateStr)
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		updated = true
	case "":
		if buildStatus.Result == "SUCCESS" {
			record.Status = "success"
			envName := record.EnvName
			if envName == "" {
				envName = "unknown"
			}
			dateStr := now.Format("20060102")
			record.DownloadURL = fmt.Sprintf("%s/%s/%s/", updateBaseURL, envName, dateStr)
		} else if buildStatus.Result == "FAILURE" {
			record.Status = "failed"
		}
		if record.Status != "running" {
			record.EndTime = &now
			updated = true
		}
	}

	if updated {
		database.DB.Model(record).Updates(gin.H{
			"status":       record.Status,
			"end_time":     record.EndTime,
			"download_url": record.DownloadURL,
		})
	}
}

const archiveBuildLookupLimit = 50

func resolveArchiveBuildNumber(record *ArchiveRecord, client *jenkins.Client) int {
	if record.JenkinsBuildNum > 0 {
		return record.JenkinsBuildNum
	}

	buildNum := archiveBuildNumberFromQueue(record, client)
	if buildNum == 0 {
		buildNum = archiveBuildNumberFromRecentBuilds(record, client)
	}
	if buildNum > 0 {
		record.JenkinsBuildNum = buildNum
		database.DB.Model(record).Update("jenkins_build_num", buildNum)
	}
	return buildNum
}

func archiveBuildNumberFromQueue(record *ArchiveRecord, client *jenkins.Client) int {
	if record.JenkinsQueueID <= 0 {
		return 0
	}
	queueInfo, err := client.GetQueueItemInfo(record.JenkinsQueueID)
	if err != nil || queueInfo == nil || queueInfo.Executable == nil {
		return 0
	}
	return queueInfo.Executable.Number
}

func archiveBuildNumberFromRecentBuilds(record *ArchiveRecord, client *jenkins.Client) int {
	if strings.TrimSpace(record.JenkinsJob) == "" || strings.TrimSpace(record.AppName) == "" {
		return 0
	}

	jobInfo, err := client.GetJobInfo(record.JenkinsJob)
	if err != nil || jobInfo == nil {
		return 0
	}

	for idx, build := range jobInfo.Builds {
		if idx >= archiveBuildLookupLimit {
			break
		}
		buildInfo, err := client.GetBuildInfo(record.JenkinsJob, build.Number)
		if err != nil {
			continue
		}
		if archiveBuildMatchesRecord(record, buildInfo) {
			return build.Number
		}
	}
	return 0
}

func archiveBuildMatchesRecord(record *ArchiveRecord, buildInfo map[string]interface{}) bool {
	if !archiveBuildStartedAfterRecord(record, buildInfo) {
		return false
	}

	params := archiveBuildParameters(buildInfo)
	return params["app"] == record.AppName &&
		params["tag"] == record.EnvName &&
		params["scope"] == record.DeployType
}

func archiveBuildStartedAfterRecord(record *ArchiveRecord, buildInfo map[string]interface{}) bool {
	if record.StartTime == nil {
		return true
	}
	rawTimestamp, ok := buildInfo["timestamp"]
	if !ok {
		return true
	}
	timestampMs, ok := rawTimestamp.(float64)
	if !ok || timestampMs <= 0 {
		return true
	}

	buildStartedAt := time.UnixMilli(int64(timestampMs))
	return !buildStartedAt.Before(record.StartTime.Add(-5 * time.Minute))
}

func archiveBuildParameters(buildInfo map[string]interface{}) map[string]string {
	params := make(map[string]string)

	actions, ok := buildInfo["actions"].([]interface{})
	if !ok {
		return params
	}
	for _, action := range actions {
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			continue
		}
		rawParams, ok := actionMap["parameters"].([]interface{})
		if !ok {
			continue
		}
		for _, rawParam := range rawParams {
			paramMap, ok := rawParam.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := paramMap["name"].(string)
			value, _ := paramMap["value"].(string)
			if name != "" {
				params[strings.ToLower(name)] = value
			}
		}
	}
	return params
}

// GetArchiveStatus 获取归档状态（从 Jenkins 更新）
func GetArchiveStatus(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var record ArchiveRecord
	if err := database.DB.Where("id = ?", id).Preload("App").Preload("App.Project").Preload("Environment").First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive record not found"})
		return
	}

	jenkinsClient := jenkins.NewClient(
		cfg.Jenkins.URL,
		cfg.Jenkins.Username,
		cfg.Jenkins.Token,
		time.Duration(cfg.Jenkins.Timeout)*time.Second,
	)

	buildNum := resolveArchiveBuildNumber(&record, jenkinsClient)

	if buildNum == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":   "queued",
			"queue_id": record.JenkinsQueueID,
			"message":  "构建任务已在队列中等待",
		})
		return
	}

	status, err := jenkinsClient.GetBuildStatus(record.JenkinsJob, buildNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取 Jenkins 状态失败: %v", err)})
		return
	}

	now := time.Now()
	updated := false

	phase := strings.ToUpper(status.Phase)
	result := strings.ToUpper(status.Result)

	// 使用环境名称生成下载地址（如 6f_dev）
	envName := record.EnvName
	if envName == "" {
		envName = "unknown"
	}

	// 使用日期格式 YYYYMMDD
	dateStr := now.Format("20060102")

	// 生成下载地址基础路径
	baseURL := fmt.Sprintf("%s/%s/%s/", updateBaseURL, envName, dateStr)

	switch phase {
	case "QUEUED":
		record.Status = "queued"
	case "STARTED":
		record.Status = "running"
	case "COMPLETED":
		if result == "SUCCESS" {
			record.Status = "success"
			record.DownloadURL = baseURL
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		updated = true
	case "FINALIZED":
		if result == "SUCCESS" {
			record.Status = "success"
			record.DownloadURL = baseURL
		} else {
			record.Status = "failed"
		}
		record.EndTime = &now
		updated = true
	case "":
		fallthrough
	default:
		if result == "SUCCESS" {
			record.Status = "success"
			record.DownloadURL = baseURL
			record.EndTime = &now
			updated = true
		} else if result == "FAILURE" {
			record.Status = "failed"
			record.EndTime = &now
			updated = true
		} else if phase == "STARTED" || phase == "RUNNING" {
			record.Status = "running"
		}
	}

	if updated {
		if err := database.DB.Save(&record).Error; err != nil {
			fmt.Printf("[ArchiveStatus] 更新数据库失败: %v\n", err)
		} else {
			_ = platformevent.RecordArchiveRecord(toArchiveEventPayload(record))
		}
	}

	fmt.Printf("[ArchiveStatus] id=%d, phase=%s, result=%s, updated=%v\n",
		record.ID, phase, result, updated)

	c.JSON(http.StatusOK, gin.H{
		"id":           record.ID,
		"status":       record.Status,
		"build_number": buildNum,
		"phase":        status.Phase,
		"jenkins_url":  status.URL,
		"console_url":  fmt.Sprintf("%s/job/%s/%d/console", cfg.Jenkins.URL, record.JenkinsJob, buildNum),
		"download_url": record.DownloadURL,
		"timestamp":    status.Timestamp,
	})
}

// GetArchiveDownloadURL 获取归档下载地址
func GetArchiveDownloadURL(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var record ArchiveRecord
	if err := database.DB.Where("id = ?", id).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive record not found"})
		return
	}

	// 如果归档未成功，返回错误
	if record.Status != "success" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "归档尚未完成，无法获取下载地址"})
		return
	}

	// 如果下载地址为空，尝试生成
	if record.DownloadURL == "" {
		now := time.Now()
		envName := record.EnvName
		if envName == "" {
			envName = "unknown"
		}
		dateStr := now.Format("20060102")
		downloadURL := fmt.Sprintf("%s/%s/%s/", updateBaseURL, envName, dateStr)
		record.DownloadURL = downloadURL
		database.DB.Model(&record).Update("download_url", downloadURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"download_url": record.DownloadURL,
	})
}

// DeleteArchiveRecord 删除归档记录
func DeleteArchiveRecord(c *gin.Context) {
	id := c.Param("id")

	var record ArchiveRecord
	if err := database.DB.Where("id = ?", id).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive record not found"})
		return
	}

	if err := database.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete archive record"})
		return
	}
	_ = platformevent.RecordArchiveRecordDeleted(toArchiveEventPayload(record))

	c.JSON(http.StatusOK, gin.H{"message": "archive record deleted successfully"})
}

// ArchiveFile 归档文件信息
type ArchiveFile struct {
	Name      string `json:"name"`      // 文件名
	URL       string `json:"url"`       // 下载链接
	Size      int64  `json:"size"`      // 文件大小（字节）
	Timestamp string `json:"timestamp"` // 时间戳（如 20260129104425）
}

// GetArchiveFiles 获取归档文件列表
func GetArchiveFiles(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")

	var record ArchiveRecord
	if err := database.DB.Where("id = ?", id).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive record not found"})
		return
	}

	// 如果归档未完成，返回错误
	if record.Status != "success" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "归档尚未完成，无法获取文件列表"})
		return
	}

	// 获取基础下载 URL
	baseURL := strings.TrimSuffix(record.DownloadURL, "/")
	if baseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "下载地址为空"})
		return
	}

	// 从归档记录创建时间获取日期部分（年月日）用于过滤
	createdDate := record.CreatedAt.Format("20060102")

	// 从下载目录获取文件列表
	files, err := fetchArchiveFilesFromDirectory(baseURL, createdDate, record.AppName)
	if err != nil {
		// 如果获取失败，尝试从 Jenkins 获取
		files = generateFilesFromJenkins(record, cfg)
	}

	// 如果仍然没有文件，返回错误
	if len(files) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"base_url":  baseURL,
			"timestamp": createdDate + "000000",
			"files":     []ArchiveFile{},
			"message":   "未找到归档文件",
		})
		return
	}

	// 返回的文件中第一个文件的时间戳作为响应的时间戳
	responseTimestamp := createdDate + "000000"
	if len(files) > 0 && files[0].Timestamp != "" {
		responseTimestamp = files[0].Timestamp
	}

	c.JSON(http.StatusOK, gin.H{
		"base_url":  baseURL,
		"timestamp": responseTimestamp,
		"files":     files,
	})
}

// fetchArchiveFilesFromDirectory 从 nginx 目录列表 HTML 中解析真实文件
// 只要文件名包含应用名称就匹配
func fetchArchiveFilesFromDirectory(baseURL string, createdDate string, appName string) ([]ArchiveFile, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(baseURL)
	if err != nil {
		return nil, fmt.Errorf("无法访问目录: %v", err)
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

	// 解析 nginx 目录列表中的文件
	// 格式: <a href="filename">filename</a>  日期  时间  大小
	filePattern := regexp.MustCompile(`(?s)<a\s+href="([^"]+)">[^<]+</a>\s+(\d{2}-[A-Za-z]{3}-\d{4})\s+(\d{2}:\d{2}(?::\d{2})?)\s+([\d.]+[KMGT]?[BBytes]?)`)
	matches := filePattern.FindAllStringSubmatch(html, -1)

	var files []ArchiveFile
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		filename := match[1]
		sizeStr := match[4]

		// 跳过目录和父目录链接
		if filename == "../" || strings.HasSuffix(filename, "/") {
			continue
		}

		// 只保留 tar 和 json 文件
		if !strings.HasSuffix(filename, ".tar") && !strings.HasSuffix(filename, ".tgz") &&
			!strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".json") {
			continue
		}

		// 过滤：文件名必须包含应用名称（不区分大小写）
		if appName != "" && !strings.Contains(strings.ToLower(filename), strings.ToLower(appName)) {
			continue
		}

		// 过滤：时间戳日期部分必须匹配
		fileTS := extractTimestampFromFilename(filename)
		if fileTS != "" && len(fileTS) >= 8 && fileTS[:8] != createdDate {
			continue
		}

		// 去重
		if seen[filename] {
			continue
		}
		seen[filename] = true

		size := parseFileSize(sizeStr)
		files = append(files, ArchiveFile{
			Name:      filename,
			URL:       fmt.Sprintf("%s/%s", baseURL, filename),
			Timestamp: fileTS,
			Size:      size,
		})
	}

	// 如果没有找到文件，尝试简单匹配（没有大小信息的文件名）
	if len(files) == 0 {
		simplePattern := regexp.MustCompile(`<a href="([^"]+\.(?:tar|tgz|tar\.gz|json))"`)
		simpleMatches := simplePattern.FindAllStringSubmatch(html, -1)
		for _, match := range simpleMatches {
			if len(match) < 2 {
				continue
			}
			filename := match[1]
			if seen[filename] {
				continue
			}
			// 过滤：文件名必须包含应用名称
			if appName != "" && !strings.Contains(strings.ToLower(filename), strings.ToLower(appName)) {
				continue
			}
			seen[filename] = true
			files = append(files, ArchiveFile{
				Name:      filename,
				URL:       fmt.Sprintf("%s/%s", baseURL, filename),
				Timestamp: extractTimestampFromFilename(filename),
			})
		}
	}

	return files, nil
}

// generateFilesFromJenkins 从 Jenkins 获取产物名称生成文件列表
func generateFilesFromJenkins(record ArchiveRecord, cfg *config.Config) []ArchiveFile {
	baseURL := strings.TrimSuffix(record.DownloadURL, "/")
	if baseURL == "" {
		return nil
	}

	var artifactNames []string
	timestamp := record.CreatedAt.Format("20060102150405")

	if record.JenkinsJob != "" && record.JenkinsBuildNum > 0 {
		jenkinsClient := jenkins.NewClient(
			cfg.Jenkins.URL,
			cfg.Jenkins.Username,
			cfg.Jenkins.Token,
			time.Duration(cfg.Jenkins.Timeout)*time.Second,
		)

		// 尝试从 Jenkins API 获取产物
		artifacts, err := jenkinsClient.GetBuildArtifacts(record.JenkinsJob, record.JenkinsBuildNum)
		if err == nil && len(artifacts) > 0 {
			for _, a := range artifacts {
				if strings.HasSuffix(a.Name, ".tar") || strings.HasSuffix(a.Name, ".tgz") ||
					strings.HasSuffix(a.Name, ".tar.gz") || strings.HasSuffix(a.Name, ".json") {
					artifactNames = append(artifactNames, a.Name)
				}
			}
		}
	}

	files := []ArchiveFile{}
	seen := make(map[string]bool)

	for _, name := range artifactNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		files = append(files, ArchiveFile{
			Name:      name,
			URL:       fmt.Sprintf("%s/%s", baseURL, name),
			Timestamp: extractTimestampFromFilename(name),
			Size:      getFileSize(fmt.Sprintf("%s/%s", baseURL, name)),
		})
	}

	// 如果没有产物，生成候选文件名
	if len(files) == 0 && record.AppName != "" {
		versions := []string{"2.0.3", "2.0.2", "2.0.1", "2.0.0", "1.0.0"}
		extensions := []string{".tar", ".json"}

		for _, ext := range extensions {
			for _, version := range versions {
				pattern := fmt.Sprintf("%s-V%s-build%s%s", record.AppName, version, timestamp, ext)
				files = append(files, ArchiveFile{
					Name:      pattern,
					URL:       fmt.Sprintf("%s/%s", baseURL, pattern),
					Timestamp: timestamp,
				})
			}
		}
	}

	return files
}

// fetchFilesFromDirectory 从 nginx 目录列表 HTML 中解析真实文件
// 只返回匹配指定应用名称和日期的文件
func fetchFilesFromDirectory(baseURL string, createdDate string, appName string) ([]ArchiveFile, error) {
	client := &http.Client{Timeout: 10 * time.Second}

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

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)

	// 解析 nginx 目录列表中的文件
	// nginx autoindex 格式: <a href="filename">filename</a>  日期  时间[秒]  大小
	filePattern := regexp.MustCompile(`(?s)<a\s+href="([^"]+)">[^<]+</a>\s+(\d{2}-[A-Za-z]{3}-\d{4})\s+(\d{2}:\d{2}(?::\d{2})?)\s+([\d.]+[KMGT]?[BBytes]?)`)
	matches := filePattern.FindAllStringSubmatch(html, -1)

	var files []ArchiveFile
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		filename := match[1]
		sizeStr := match[4]

		// 跳过目录链接
		if filename == "../" || strings.HasSuffix(filename, "/") {
			continue
		}

		// 只保留 tar 和 json 文件
		if !strings.HasSuffix(filename, ".tar") && !strings.HasSuffix(filename, ".tgz") &&
			!strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".json") {
			continue
		}

		// 提取文件名中的应用名称和时间戳
		fileAppName := extractAppNameFromFilename(filename)
		fileTS := extractTimestampFromFilename(filename)

		// 过滤：应用名称必须匹配
		if appName != "" && fileAppName != "" && fileAppName != appName {
			continue // 应用名称不匹配，跳过
		}

		// 过滤：时间戳的日期部分必须匹配
		if fileTS != "" && createdDate != "" && len(fileTS) >= 8 && fileTS[:8] != createdDate {
			continue // 日期不匹配，跳过
		}

		// 去重
		if seen[filename] {
			continue
		}
		seen[filename] = true

		// 解析文件大小
		size := parseFileSize(sizeStr)

		files = append(files, ArchiveFile{
			Name:      filename,
			URL:       fmt.Sprintf("%s/%s", baseURL, filename),
			Timestamp: fileTS,
			Size:      size,
		})
	}

	// 如果没有找到文件，尝试从 HTML 中直接提取文件名（没有大小信息）
	if len(files) == 0 {
		simplePattern := regexp.MustCompile(`<a href="([^"]+\.(?:tar|tgz|tar\.gz|json))"`)
		simpleMatches := simplePattern.FindAllStringSubmatch(html, -1)
		for _, match := range simpleMatches {
			if len(match) < 2 {
				continue
			}
			filename := match[1]
			if seen[filename] {
				continue
			}

			// 提取文件名中的应用名称和时间戳进行过滤
			fileAppName := extractAppNameFromFilename(filename)
			fileTS := extractTimestampFromFilename(filename)

			// 过滤：应用名称必须匹配
			if appName != "" && fileAppName != "" && fileAppName != appName {
				continue
			}

			// 过滤：时间戳的日期部分必须匹配
			if fileTS != "" && createdDate != "" && len(fileTS) >= 8 && fileTS[:8] != createdDate {
				continue
			}

			seen[filename] = true

			files = append(files, ArchiveFile{
				Name:      filename,
				URL:       fmt.Sprintf("%s/%s", baseURL, filename),
				Timestamp: fileTS,
			})
		}
	}

	return files, nil
}

// extractTimestampFromFilename 从文件名中提取时间戳
// 优先匹配: xxx-Vx.x.x-buildxxxxxxxxxxxxx 格式
// 也支持: xxx-buildxxxxxxxxxxxxx 或直接的时间戳格式
func extractTimestampFromFilename(filename string) string {
	// 优先匹配标准的产物文件名格式：xxx-Vx.x.x-buildxxxxxxxxxxxxx.tar 或 .json
	pattern := regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_-]*-[vV]\d+\.\d+\.\d+-build(\d{14})\.(?:tar|tgz|tar\.gz|json)$`)
	match := pattern.FindStringSubmatch(filename)
	if len(match) >= 2 {
		return match[1]
	}

	// 也支持 xxx-buildxxxxxxxxxxxxx 格式（无版本号）
	pattern2 := regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_-]*-build(\d{14})\.(?:tar|tgz|tar\.gz|json)$`)
	match2 := pattern2.FindStringSubmatch(filename)
	if len(match2) >= 2 {
		return match2[1]
	}

	// 也支持 buildxxxxxxxxxxxxx 格式
	pattern3 := regexp.MustCompile(`build(\d{14})\.(?:tar|tgz|tar\.gz|json)$`)
	match3 := pattern3.FindStringSubmatch(filename)
	if len(match3) >= 2 {
		return match3[1]
	}

	return ""
}

// extractAppNameFromFilename 从文件名中提取应用名称
// 格式: xxx-Vx.x.x-buildxxxxxxxxxxxxx.tar -> 返回 xxx
// 也支持: xxx-buildxxxxxxxxxxxxx 格式
func extractAppNameFromFilename(filename string) string {
	// 标准格式: xxx-Vx.x.x-buildxxxxxxxxxxxxx.tar
	pattern := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)-[vV]\d+\.\d+\.\d+-build\d{14}\.(?:tar|tgz|tar\.gz|json)$`)
	match := pattern.FindStringSubmatch(filename)
	if len(match) >= 1 {
		return match[1]
	}

	// 无版本号格式: xxx-buildxxxxxxxxxxxxx.tar
	pattern2 := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)-build\d{14}\.(?:tar|tgz|tar\.gz|json)$`)
	match2 := pattern2.FindStringSubmatch(filename)
	if len(match2) >= 1 {
		return match2[1]
	}

	return ""
}

// isValidArchiveFilename 检查文件名是否为有效的归档文件名
func isValidArchiveFilename(filename string) bool {
	// tar/tgz/tar.gz/json 文件，且符合 xxx-Vx.x.x-buildxxxxxxxxxxxxx 格式
	if !strings.HasSuffix(filename, ".tar") && !strings.HasSuffix(filename, ".tgz") &&
		!strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".json") {
		return false
	}
	return extractTimestampFromFilename(filename) != ""
}

// parseFileSize 解析文件大小字符串（如 "203M", "2.5K", "1024"）
func parseFileSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	var size float64
	var unit string
	_, err := fmt.Sscanf(sizeStr, "%f%s", &size, &unit)
	if err != nil && sizeStr != "" {
		// 如果没有单位，尝试直接解析
		size, _ = strconv.ParseFloat(sizeStr, 64)
		return int64(size)
	}

	// 转换为字节
	switch strings.ToUpper(unit) {
	case "K", "KB", "KIB":
		return int64(size * 1024)
	case "M", "MB", "MIB":
		return int64(size * 1024 * 1024)
	case "G", "GB", "GIB":
		return int64(size * 1024 * 1024 * 1024)
	case "T", "TB", "TIB":
		return int64(size * 1024 * 1024 * 1024 * 1024)
	case "B", "BYTES":
		return int64(size)
	default:
		// 如果没有识别到单位但有数值，返回数值（假设是字节）
		if size > 0 {
			return int64(size)
		}
		return 0
	}
}

// getFileSize 通过 HTTP HEAD 请求获取文件大小
func getFileSize(url string) int64 {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	return resp.ContentLength
}

// generateFileCandidates 生成候选文件列表（当无法访问目录时使用）
func generateFileCandidates(record ArchiveRecord, cfg *config.Config) ([]ArchiveFile, string) {
	var timestamps []string
	var artifactNames []string

	if record.JenkinsJob != "" && record.JenkinsBuildNum > 0 {
		jenkinsClient := jenkins.NewClient(
			cfg.Jenkins.URL,
			cfg.Jenkins.Username,
			cfg.Jenkins.Token,
			time.Duration(cfg.Jenkins.Timeout)*time.Second,
		)

		consoleLog, err := jenkinsClient.GetConsoleLog(record.JenkinsJob, record.JenkinsBuildNum)
		if err == nil && consoleLog != "" {
			timestamps = jenkins.ParseBuildTimestampFromLog(consoleLog)
			artifactNames = jenkins.ParseArtifactNameFromLog(consoleLog)
		}

		if len(artifactNames) == 0 {
			artifacts, err := jenkinsClient.GetBuildArtifacts(record.JenkinsJob, record.JenkinsBuildNum)
			if err == nil && len(artifacts) > 0 {
				for _, a := range artifacts {
					if strings.HasSuffix(a.Name, ".tar") || strings.HasSuffix(a.Name, ".tgz") ||
						strings.HasSuffix(a.Name, ".tar.gz") || strings.HasSuffix(a.Name, ".json") {
						artifactNames = append(artifactNames, a.Name)
					}
				}
			}
		}
	}

	if len(timestamps) == 0 {
		ts := record.CreatedAt.Format("20060102150405")
		timestamps = []string{ts}
	}

	baseURL := strings.TrimSuffix(record.DownloadURL, "/")
	files := []ArchiveFile{}
	seen := make(map[string]bool)
	timestamp := timestamps[0]

	// 优先使用 Jenkins 产物名称
	for _, name := range artifactNames {
		if seen[name] {
			continue
		}
		// 只保留符合产物文件名格式的文件（xxx-Vx.x.x-buildxxxxxxxxxxxxx.{tar,tgz,tar.gz,json,json}）
		if !isValidArchiveFilename(name) {
			continue
		}
		seen[name] = true
		fileURL := fmt.Sprintf("%s/%s", baseURL, name)
		files = append(files, ArchiveFile{
			Name:      name,
			URL:       fileURL,
			Timestamp: timestamp,
			Size:      getFileSize(fileURL),
		})

		// 如果是 tar 文件，自动添加对应的 json 文件（如果存在）
		if strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar.gz") {
			baseName := strings.TrimSuffix(name, ".tar")
			baseName = strings.TrimSuffix(baseName, ".tgz")
			baseName = strings.TrimSuffix(baseName, ".tar.gz")
			jsonName := baseName + ".json"
			if !seen[jsonName] {
				seen[jsonName] = true
				jsonURL := fmt.Sprintf("%s/%s", baseURL, jsonName)
				files = append(files, ArchiveFile{
					Name:      jsonName,
					URL:       jsonURL,
					Timestamp: timestamp,
					Size:      getFileSize(jsonURL),
				})
			}
		}
	}

	// 如果没有产物名称，使用时间戳生成
	if len(artifactNames) == 0 {
		patterns := []string{
			fmt.Sprintf("*-build%s.tar", timestamp),
			fmt.Sprintf("*-build%s.tgz", timestamp),
			fmt.Sprintf("*-build%s.tar.gz", timestamp),
			fmt.Sprintf("*-build%s.json", timestamp),
			fmt.Sprintf("build%s.tar", timestamp),
			fmt.Sprintf("build%s.json", timestamp),
		}
		for _, pattern := range patterns {
			if seen[pattern] {
				continue
			}
			seen[pattern] = true
			patternURL := fmt.Sprintf("%s/%s", baseURL, pattern)
			files = append(files, ArchiveFile{
				Name:      pattern,
				URL:       patternURL,
				Timestamp: timestamp,
				Size:      getFileSize(patternURL),
			})
		}
	}

	return files, timestamp
}

// ========== Jenkins 集成 ==========

// GetJenkinsViewJobs 获取 Jenkins View 下的所有 Jobs
func GetJenkinsViewJobs(c *gin.Context, cfg *config.Config) {
	viewName := c.Param("name")
	appNamePrefix := strings.TrimSpace(c.Query("app_name_prefix"))
	if viewName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "view name is required"})
		return
	}

	// 优先使用请求参数中的Jenkins URL，如果没有则使用配置中的默认值
	jenkinsURL := c.Query("jenkins_url")
	if jenkinsURL == "" {
		jenkinsURL = cfg.Jenkins.URL
	}

	// 使用配置中的认证信息
	jenkinsClient := jenkins.NewClient(jenkinsURL, cfg.Jenkins.Username, cfg.Jenkins.Token, time.Duration(cfg.Jenkins.Timeout)*time.Second)
	viewInfo, err := jenkinsClient.GetViewJobs(viewName)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("视图 '%s' 不存在", viewName)})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("获取 Jenkins View 失败: %v", err)})
		}
		return
	}

	// 查询已存在的应用（按 jenkins_job 匹配，支持 job name 或完整 URL）
	var existingApps []Application
	database.DB.Where("deleted_at IS NULL").Select("jenkins_job").Find(&existingApps)
	existingJobs := make(map[string]bool)
	for _, app := range existingApps {
		if app.JenkinsJob != "" {
			existingJobs[app.JenkinsJob] = true
			// 从 URL 中提取 job 名称也标记为已存在
			if strings.Contains(app.JenkinsJob, "/job/") {
				parts := strings.Split(app.JenkinsJob, "/job/")
				if len(parts) > 1 {
					existingJobs[strings.TrimRight(parts[len(parts)-1], "/")] = true
				}
			}
		}
	}

	jenkinsBaseURL := strings.TrimRight(cfg.Jenkins.URL, "/")
	prefix := extractJobPrefix(viewName)
	type JobItem struct {
		Name    string `json:"name"`
		AppName string `json:"app_name"`
		URL     string `json:"url"`
		JobURL  string `json:"job_url"`
		Color   string `json:"color"`
		Exists  bool   `json:"exists"`
	}

	jobs := make([]JobItem, 0, len(viewInfo.Jobs))
	for _, j := range viewInfo.Jobs {
		appName := extractAppNameFromJobWithPrefix(viewName, j.Name, appNamePrefix)
		jobURL := fmt.Sprintf("%s/view/%s/job/%s", jenkinsBaseURL, viewName, j.Name)
		jobs = append(jobs, JobItem{
			Name:    j.Name,
			AppName: appName,
			URL:     j.URL,
			JobURL:  jobURL,
			Color:   j.Color,
			Exists:  existingJobs[j.Name],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"view_name":       viewInfo.Name,
		"jobs":            jobs,
		"total":           len(jobs),
		"prefix":          prefix,
		"app_name_prefix": appNamePrefix,
	})
}

// extractJobPrefix 从 View 名称提取 Job 公共前缀
// 例如 "6f_dev-187" / "jlsf_dev-V2.4.0" → "6f_dev-" / "jlsf_dev-"，用于去掉 job 名称中的环境前缀
func extractJobPrefix(viewName string) string {
	re := regexp.MustCompile(`-(\d+|[Vv]\d+(?:\.\d+)*)$`)
	base := re.ReplaceAllString(viewName, "")
	if base != viewName && base != "" {
		return base + "-"
	}
	return ""
}

// ImportJenkinsJobs 从 Jenkins View 批量导入 Jobs 为应用
func ImportJenkinsJobs(c *gin.Context, cfg *config.Config) {
	var req struct {
		ViewName      string   `json:"view_name" binding:"required"`
		ProjectID     uint     `json:"project_id" binding:"required"`
		EnvID         uint     `json:"env_id" binding:"required"`
		JobNames      []string `json:"job_names" binding:"required"`
		ArchiveJob    string   `json:"archive_job"`
		JenkinsURL    string   `json:"jenkins_url"` // 支持动态Jenkins URL
		AppNamePrefix string   `json:"app_name_prefix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证项目和环境是否存在
	var project Project
	if err := database.DB.First(&project, req.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目不存在"})
		return
	}
	var env Environment
	if err := database.DB.First(&env, req.EnvID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "环境不存在"})
		return
	}

	// 优先使用请求中的Jenkins URL，如果为空则使用配置中的默认值
	jenkinsURL := req.JenkinsURL
	if jenkinsURL == "" {
		jenkinsURL = cfg.Jenkins.URL
	}

	jenkinsBaseURL := strings.TrimRight(jenkinsURL, "/")

	// 构建归档流水线 URL
	archiveJobURL := ""
	if req.ArchiveJob != "" {
		archiveJobURL = fmt.Sprintf("%s/view/%s/job/%s", jenkinsBaseURL, req.ViewName, req.ArchiveJob)
	}
	fmt.Printf("[Jenkins Import] view=%s, archive_job=%q, archiveURL=%q, jobs=%d\n",
		req.ViewName, req.ArchiveJob, archiveJobURL, len(req.JobNames))

	created := 0
	skipped := 0
	var errors []string

	for _, jobName := range req.JobNames {
		// 构建完整 Jenkins URL
		jobURL := fmt.Sprintf("%s/view/%s/job/%s", jenkinsBaseURL, req.ViewName, jobName)

		// 检查是否已存在（按完整 URL 或 job 名称判断）
		var count int64
		database.DB.Model(&Application{}).Where(
			"(jenkins_job = ? OR jenkins_job = ?) AND deleted_at IS NULL", jobName, jobURL,
		).Count(&count)
		if count > 0 {
			skipped++
			continue
		}

		// 去掉视图相关前缀得到简洁的应用名称
		appName := extractAppNameFromJobWithPrefix(req.ViewName, jobName, req.AppNamePrefix)

		app := Application{
			Name:              appName,
			ProjectID:         req.ProjectID,
			EnvID:             req.EnvID,
			JenkinsJob:        jobURL,
			JenkinsArchiveJob: archiveJobURL,
		}

		if err := database.DB.Create(&app).Error; err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", jobName, err))
			continue
		}
		created++
	}

	c.JSON(http.StatusOK, gin.H{
		"created": created,
		"skipped": skipped,
		"errors":  errors,
		"message": fmt.Sprintf("导入完成：新增 %d 个，跳过 %d 个已存在", created, skipped),
	})
}

// ViewCopyRequest 视图复制请求结构
type ViewCopyRequest struct {
	SourceView          string                   `json:"source_view" binding:"required"`
	TargetView          string                   `json:"target_view" binding:"required"`
	JenkinsURL          string                   `json:"jenkins_url"`
	TagReplacements     []TagReplacementRule     `json:"tag_replacements"`
	JobNameReplacements []JobNameReplacementRule `json:"job_name_replacements"`
}

// JobNameReplacementRule Job名称替换
type JobNameReplacementRule struct {
	OldPattern string `json:"old_pattern"`
	NewPattern string `json:"new_pattern"`
}

// TagReplacementRule Tag替换规则
type TagReplacementRule struct {
	OldPattern string `json:"old_pattern"`
	NewPattern string `json:"new_pattern"`
}

// URLReplacementRule URL替换规则
type URLReplacementRule struct {
	OldPattern string `json:"old_pattern"`
	NewPattern string `json:"new_pattern"`
}

// ViewCopyResult 视图复制结果结构
type ViewCopyResult struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	CopiedJobs    []string `json:"copied_jobs"`
	FailedJobs    []string `json:"failed_jobs"`
	SkippedJobs   []string `json:"skipped_jobs"`
	ApprovedCount int      `json:"approved_count"`
	ApprovalNote  string   `json:"approval_note,omitempty"`
}

// CopyJenkinsView 复制 Jenkins View 及其 Jobs
func CopyJenkinsView(c *gin.Context, cfg *config.Config) {
	var req ViewCopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查是否是异步请求
	async := c.Query("async") == "true"

	if async {
		// 异步处理模式
		handleAsyncCopyJenkinsView(c, cfg, req)
	} else {
		// 同步处理模式
		handleSyncCopyJenkinsView(c, cfg, req)
	}
}

// 异步处理函数
func handleAsyncCopyJenkinsView(c *gin.Context, cfg *config.Config, req ViewCopyRequest) {
	// 验证Jenkins URL的安全性，防止SSRF攻击
	if req.JenkinsURL != "" && req.JenkinsURL != cfg.Jenkins.URL {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不允许使用自定义Jenkins URL"})
		return
	}

	// 验证Jenkins配置是否完整
	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	// 预检查源视图是否存在
	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)
	_, err := jenkinsClient.GetViewJobs(req.SourceView)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("源 Jenkins View 不存在: %v", err)})
		return
	}

	// 创建异步任务
	taskManager := tasks.GetDefaultTaskManager()
	taskInfo := taskManager.CreateTask("jenkins-copy-view")
	taskInfo.Status = tasks.TaskRunning
	taskManager.UpdateTaskStatus(taskInfo.ID, tasks.TaskRunning, nil)

	// 在后台运行复制任务
	go runJenkinsCopyTask(taskInfo.ID, cfg, req)

	// 返回任务ID供前端轮询
	c.JSON(http.StatusAccepted, gin.H{
		"task_id": taskInfo.ID,
		"status":  "accepted",
		"message": "复制任务已提交，请使用任务ID轮询进度",
	})
}

// 运行实际的复制任务
func runJenkinsCopyTask(taskID string, cfg *config.Config, req ViewCopyRequest) {
	taskManager := tasks.GetDefaultTaskManager()

	// 创建Jenkins客户端
	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	// 获取源视图的Jobs
	sourceViewInfo, err := jenkinsClient.GetViewJobs(req.SourceView)
	if err != nil {
		taskManager.UpdateTaskStatus(taskID, tasks.TaskFailed, &tasks.TaskResult{
			Success: false,
			Message: fmt.Sprintf("获取源 Jenkins View 失败: %v", err),
		})
		return
	}

	totalJobs := len(sourceViewInfo.Jobs)
	taskManager.UpdateTaskProgress(taskID, 0, totalJobs)

	// 创建目标视图 - 使用完整的XML配置
	viewConfig := fmt.Sprintf(`<hudson.model.ListView>
  <name>%s</name>
  <filterExecutors>false</filterExecutors>
  <filterQueue>false</filterQueue>
  <properties class="hudson.model.View$PropertyList"/>
  <jobNames>
    <comparator class="hudson.util.CaseInsensitiveComparator"/>
  </jobNames>
  <jobFilters/>
  <columns>
    <hudson.views.StatusColumn/>
    <hudson.views.WeatherColumn/>
    <hudson.views.JobColumn/>
    <hudson.views.LastSuccessColumn/>
    <hudson.views.LastFailureColumn/>
    <hudson.views.LastDurationColumn/>
    <hudson.views.BuildButtonColumn/>
  </columns>
  <recurse>false</recurse>
</hudson.model.ListView>`, req.TargetView)

	err = jenkinsClient.CreateView(req.TargetView, viewConfig)
	if err != nil {
		// 如果视图已存在，继续处理
		if !strings.Contains(err.Error(), "400") && !strings.Contains(err.Error(), "already exists") {
			taskManager.UpdateTaskStatus(taskID, tasks.TaskFailed, &tasks.TaskResult{
				Success: false,
				Message: fmt.Sprintf("创建目标 Jenkins View 失败: %v", err),
			})
			return
		}
	}

	// 初始化结果统计
	var copiedJobs, failedJobs, skippedJobs []string

	jobNameReplacements := normalizeJobNameReplacements(req.JobNameReplacements)

	// 遍历源视图中的所有Jobs并复制
	for i, job := range sourceViewInfo.Jobs {
		newJobName := job.Name

		// 应用Job名称替换规则
		for _, rule := range jobNameReplacements {
			if rule.OldPattern != "" && rule.NewPattern != "" {
				re, err := regexp.Compile(regexp.QuoteMeta(rule.OldPattern))
				if err != nil {
					// 如果正则编译失败，尝试直接字符串替换
					newJobName = strings.ReplaceAll(newJobName, rule.OldPattern, rule.NewPattern)
				} else {
					newJobName = re.ReplaceAllString(newJobName, rule.NewPattern)
				}
			}
		}

		// 如果新名称与原名称相同，跳过
		if newJobName == job.Name {
			// 检查是否已经是目标视图的命名方式（即已经存在于目标视图中）
			if strings.Contains(job.Name, req.TargetView) {
				skippedJobs = append(skippedJobs, job.Name)
				taskManager.UpdateTaskProgress(taskID, i+1, totalJobs)
				continue
			}
		}

		// 获取源Job的配置
		jobConfig, err := jenkinsClient.GetJobConfigXML(job.Name)
		if err != nil {
			failedJobs = append(failedJobs, fmt.Sprintf("%s (获取配置失败)", job.Name))
			taskManager.UpdateTaskProgress(taskID, i+1, totalJobs)
			continue
		}

		// 基于视图名称变化推断的额外替换规则
		// 如果源视图名是 fat-150-V2.5.1，目标是 fat-160-V2.5.1，我们也应该替换 fat150 -> fat160
		if req.SourceView != req.TargetView {
			jobConfig = applyInferredReplacements(jobConfig, req.SourceView, req.TargetView)
		}

		// 用户显式填写的 Tag 替换规则优先级更高，最后执行，避免被推断替换覆盖。
		jobConfig = applyTagReplacements(jobConfig, req.TagReplacements)

		// 检查 Job 是否已存在
		if jenkinsClient.JobExists(newJobName) {
			skippedJobs = append(skippedJobs, fmt.Sprintf("%s (已存在)", newJobName))
			taskManager.UpdateTaskProgress(taskID, i+1, totalJobs)
			continue
		}

		// 创建新Job
		err = jenkinsClient.CreateJob(newJobName, jobConfig)
		if err != nil {
			failedJobs = append(failedJobs, fmt.Sprintf("%s (创建失败: %v)", newJobName, err))
			taskManager.UpdateTaskProgress(taskID, i+1, totalJobs)
			continue
		}

		// 将新Job添加到目标视图
		err = jenkinsClient.AddJobToView(req.TargetView, newJobName)
		if err != nil {
			// 即使添加到视图失败，也认为Job创建成功
			copiedJobs = append(copiedJobs, fmt.Sprintf("%s (添加到视图失败: %v)", newJobName, err))
		} else {
			copiedJobs = append(copiedJobs, newJobName)
		}

		// 更新进度
		taskManager.UpdateTaskProgress(taskID, i+1, totalJobs)
	}

	// 自动审批 Jenkins Script Approval 中的待处理项
	approvedCount := 0
	approvalNote := ""
	if len(copiedJobs) > 0 {
		if approvalResult, err := jenkinsClient.ApprovePendingScriptApprovalItems(); err == nil {
			approvedCount = approvalResult.ApprovedCount()
			if approvalResult.ApprovedSignatures > 0 {
				approvalNote = fmt.Sprintf("自动审批脚本 %d 个，签名 %d 个", approvalResult.ApprovedScripts, approvalResult.ApprovedSignatures)
			}
		} else {
			approvalNote = fmt.Sprintf("Jenkins Script Approval 自动审批未完成，仍需人工在 Jenkins 页面审核：%v", err)
		}
	}

	// 准备返回结果
	approvalMsg := ""
	if approvedCount > 0 {
		approvalMsg = fmt.Sprintf("，自动审批脚本 %d 个", approvedCount)
	}
	if approvalNote != "" && approvedCount == 0 {
		approvalMsg = "，" + approvalNote
	}

	result := &tasks.TaskResult{
		Success:       len(failedJobs) == 0 && len(copiedJobs) > 0,
		Message:       fmt.Sprintf("复制完成：成功 %d 个，失败 %d 个，跳过 %d 个%s", len(copiedJobs), len(failedJobs), len(skippedJobs), approvalMsg),
		CopiedJobs:    copiedJobs,
		FailedJobs:    failedJobs,
		SkippedJobs:   skippedJobs,
		ApprovedCount: approvedCount,
		ApprovalNote:  approvalNote,
	}

	// 设置最终状态
	finalStatus := tasks.TaskCompleted
	if len(copiedJobs) == 0 && len(failedJobs) > 0 {
		finalStatus = tasks.TaskFailed
		result.Success = false
		result.Message = fmt.Sprintf("复制失败：所有Job都复制失败。成功 %d 个，失败 %d 个", len(copiedJobs), len(failedJobs))
	} else if len(copiedJobs) == 0 && len(failedJobs) == 0 && len(skippedJobs) == 0 {
		finalStatus = tasks.TaskFailed
		result.Success = false
		result.Message = "没有找到任何需要复制的Jobs"
	} else if len(copiedJobs) > 0 && len(failedJobs) > 0 {
		result.Message = fmt.Sprintf("部分复制成功：成功 %d 个，失败 %d 个，跳过 %d 个%s", len(copiedJobs), len(failedJobs), len(skippedJobs), approvalMsg)
	}

	taskManager.UpdateTaskStatus(taskID, finalStatus, result)
}

// 同步处理函数（原有逻辑，稍作优化）
func handleSyncCopyJenkinsView(c *gin.Context, cfg *config.Config, req ViewCopyRequest) {
	// 验证Jenkins URL的安全性，防止SSRF攻击
	if req.JenkinsURL != "" && req.JenkinsURL != cfg.Jenkins.URL {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不允许使用自定义Jenkins URL"})
		return
	}

	// 验证Jenkins配置是否完整
	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	// 创建Jenkins客户端
	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	// 简单测试Jenkins连接
	testReq, testErr := http.NewRequest("GET", fmt.Sprintf("%s/api/json", cfg.Jenkins.URL), nil)
	if testErr == nil {
		testReq.SetBasicAuth(username, token)
		testResp, testRespErr := jenkinsClient.Client.Do(testReq) // 使用Jenkins客户端的HTTP客户端，包含超时设置
		if testRespErr != nil || testResp.StatusCode != http.StatusOK {
			if testRespErr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("无法连接到 Jenkins 服务器: %v", testRespErr)})
			} else {
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Jenkins 连接失败，状态码: %d", testResp.StatusCode)})
			}
			return
		}
		if testResp != nil {
			testResp.Body.Close()
		}
	}

	// 获取源视图的Jobs
	sourceViewInfo, err := jenkinsClient.GetViewJobs(req.SourceView)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("获取源 Jenkins View 失败: %v", err)})
		return
	}

	// 检查是否有太多Jobs（为了避免超时）
	if len(sourceViewInfo.Jobs) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "源视图包含过多Jobs(" + fmt.Sprintf("%d", len(sourceViewInfo.Jobs)) + ")，建议使用异步模式",
			"suggestion": "添加 query 参数 ?async=true 来使用异步处理",
		})
		return
	}

	// 创建目标视图 - 使用完整的XML配置
	viewConfig := fmt.Sprintf(`<hudson.model.ListView>
  <name>%s</name>
  <filterExecutors>false</filterExecutors>
  <filterQueue>false</filterQueue>
  <properties class="hudson.model.View$PropertyList"/>
  <jobNames>
    <comparator class="hudson.util.CaseInsensitiveComparator"/>
  </jobNames>
  <jobFilters/>
  <columns>
    <hudson.views.StatusColumn/>
    <hudson.views.WeatherColumn/>
    <hudson.views.JobColumn/>
    <hudson.views.LastSuccessColumn/>
    <hudson.views.LastFailureColumn/>
    <hudson.views.LastDurationColumn/>
    <hudson.views.BuildButtonColumn/>
  </columns>
  <recurse>false</recurse>
</hudson.model.ListView>`, req.TargetView)

	err = jenkinsClient.CreateView(req.TargetView, viewConfig)
	if err != nil {
		// 如果视图已存在，继续处理
		if !strings.Contains(err.Error(), "400") && !strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("创建目标 Jenkins View 失败: %v", err)})
			return
		}
	}

	// 初始化结果统计
	var copiedJobs, failedJobs, skippedJobs []string

	jobNameReplacements := normalizeJobNameReplacements(req.JobNameReplacements)

	// 遍历源视图中的所有Jobs并复制
	for _, job := range sourceViewInfo.Jobs {
		newJobName := job.Name

		// 应用Job名称替换规则
		for _, rule := range jobNameReplacements {
			if rule.OldPattern != "" && rule.NewPattern != "" {
				re, err := regexp.Compile(regexp.QuoteMeta(rule.OldPattern))
				if err != nil {
					// 如果正则编译失败，尝试直接字符串替换
					newJobName = strings.ReplaceAll(newJobName, rule.OldPattern, rule.NewPattern)
				} else {
					newJobName = re.ReplaceAllString(newJobName, rule.NewPattern)
				}
			}
		}

		// 如果新名称与原名称相同，跳过
		if newJobName == job.Name {
			// 检查是否已经是目标视图的命名方式（即已经存在于目标视图中）
			if strings.Contains(job.Name, req.TargetView) {
				skippedJobs = append(skippedJobs, job.Name)
				continue
			}
		}

		// 获取源Job的配置
		jobConfig, err := jenkinsClient.GetJobConfigXML(job.Name)
		if err != nil {
			failedJobs = append(failedJobs, fmt.Sprintf("%s (获取配置失败)", job.Name))
			continue
		}

		// 先做基于视图名的推断，再做用户显式的 Tag 替换，确保显式规则优先。
		if req.SourceView != req.TargetView {
			jobConfig = applyInferredReplacements(jobConfig, req.SourceView, req.TargetView)
		}
		jobConfig = applyTagReplacements(jobConfig, req.TagReplacements)

		// 检查 Job 是否已存在
		if jenkinsClient.JobExists(newJobName) {
			skippedJobs = append(skippedJobs, fmt.Sprintf("%s (已存在)", newJobName))
			continue
		}

		// 创建新Job
		err = jenkinsClient.CreateJob(newJobName, jobConfig)
		if err != nil {
			failedJobs = append(failedJobs, fmt.Sprintf("%s (创建失败: %v)", newJobName, err))
			continue
		}

		// 将新Job添加到目标视图
		err = jenkinsClient.AddJobToView(req.TargetView, newJobName)
		if err != nil {
			// 即使添加到视图失败，也认为Job创建成功
			copiedJobs = append(copiedJobs, fmt.Sprintf("%s (添加到视图失败: %v)", newJobName, err))
		} else {
			copiedJobs = append(copiedJobs, newJobName)
		}
	}

	// 自动审批 Jenkins Script Approval 中的待处理项
	approvedCount := 0
	approvalNote := ""
	if len(copiedJobs) > 0 {
		if approvalResult, err := jenkinsClient.ApprovePendingScriptApprovalItems(); err == nil {
			approvedCount = approvalResult.ApprovedCount()
			if approvalResult.ApprovedSignatures > 0 {
				approvalNote = fmt.Sprintf("自动审批脚本 %d 个，签名 %d 个", approvalResult.ApprovedScripts, approvalResult.ApprovedSignatures)
			}
		} else {
			approvalNote = fmt.Sprintf("Jenkins Script Approval 自动审批未完成，仍需人工在 Jenkins 页面审核：%v", err)
		}
	}

	approvalMsg := ""
	if approvedCount > 0 {
		approvalMsg = fmt.Sprintf("，自动审批脚本 %d 个", approvedCount)
	}
	if approvalNote != "" && approvedCount == 0 {
		approvalMsg = "，" + approvalNote
	}

	// 准备返回结果
	result := ViewCopyResult{
		Success:       len(failedJobs) == 0,
		Message:       fmt.Sprintf("复制完成：成功 %d 个，失败 %d 个，跳过 %d 个%s", len(copiedJobs), len(failedJobs), len(skippedJobs), approvalMsg),
		CopiedJobs:    copiedJobs,
		FailedJobs:    failedJobs,
		SkippedJobs:   skippedJobs,
		ApprovedCount: approvedCount,
		ApprovalNote:  approvalNote,
	}

	if len(copiedJobs) > 0 && len(failedJobs) > 0 {
		result.Success = false
		result.Message = fmt.Sprintf("部分复制成功：成功 %d 个，失败 %d 个，跳过 %d 个%s", len(copiedJobs), len(failedJobs), len(skippedJobs), approvalMsg)
	} else if len(copiedJobs) == 0 && len(failedJobs) == 0 && len(skippedJobs) == 0 {
		result.Success = false
		result.Message = "没有找到任何需要复制的Jobs"
	}

	c.JSON(http.StatusOK, result)
}

// 获取任务状态的API
func GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	taskManager := tasks.GetDefaultTaskManager()
	taskInfo, exists := taskManager.GetTask(taskID)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	response := gin.H{
		"id":         taskInfo.ID,
		"type":       taskInfo.Type,
		"status":     taskInfo.Status,
		"progress":   taskInfo.Progress,
		"total":      taskInfo.Total,
		"created_at": taskInfo.CreatedAt.Format(time.RFC3339),
		"updated_at": taskInfo.UpdatedAt.Format(time.RFC3339),
	}

	if taskInfo.Result != nil {
		response["result"] = taskInfo.Result
	}

	c.JSON(http.StatusOK, response)
}

// DeleteJenkinsJob 删除单个 Jenkins Job
func DeleteJenkinsJob(c *gin.Context, cfg *config.Config) {
	jobName := c.Param("name")
	if jobName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job 名称不能为空"})
		return
	}

	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	if err := jenkinsClient.DeleteJob(jobName); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("删除 Job 失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Job '%s' 已删除", jobName)})
}

// BatchDeleteJenkinsJobs 批量删除 Jenkins Jobs，可选同时删除视图（异步模式）
func BatchDeleteJenkinsJobs(c *gin.Context, cfg *config.Config) {
	var req struct {
		ViewName   string   `json:"view_name"`
		JobNames   []string `json:"job_names" binding:"required"`
		DeleteView bool     `json:"delete_view"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	taskManager := tasks.GetDefaultTaskManager()
	taskInfo := taskManager.CreateTask("jenkins-batch-delete")
	taskInfo.Status = tasks.TaskRunning
	taskManager.UpdateTaskStatus(taskInfo.ID, tasks.TaskRunning, nil)
	taskManager.UpdateTaskProgress(taskInfo.ID, 0, len(req.JobNames))

	go runJenkinsBatchDeleteTask(taskInfo.ID, cfg, req.ViewName, req.JobNames, req.DeleteView)

	c.JSON(http.StatusAccepted, gin.H{
		"task_id": taskInfo.ID,
		"status":  "accepted",
		"message": "删除任务已提交",
	})
}

func runJenkinsBatchDeleteTask(taskID string, cfg *config.Config, viewName string, jobNames []string, deleteView bool) {
	taskManager := tasks.GetDefaultTaskManager()
	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, cfg.Jenkins.Username, cfg.Jenkins.Token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	var deletedJobs, failedJobs []string
	total := len(jobNames)

	for i, jobName := range jobNames {
		if err := jenkinsClient.DeleteJob(jobName); err != nil {
			failedJobs = append(failedJobs, fmt.Sprintf("%s (%v)", jobName, err))
		} else {
			deletedJobs = append(deletedJobs, jobName)
		}
		taskManager.UpdateTaskProgress(taskID, i+1, total)
	}

	viewDeleted := false
	if deleteView && viewName != "" {
		if err := jenkinsClient.DeleteView(viewName); err == nil {
			viewDeleted = true
		}
	}

	msg := fmt.Sprintf("删除完成：成功 %d 个，失败 %d 个", len(deletedJobs), len(failedJobs))
	if viewDeleted {
		msg += fmt.Sprintf("，视图 '%s' 已删除", viewName)
	}

	result := &tasks.TaskResult{
		Success:    len(failedJobs) == 0,
		Message:    msg,
		CopiedJobs: deletedJobs,
		FailedJobs: failedJobs,
	}

	status := tasks.TaskCompleted
	if len(deletedJobs) == 0 && len(failedJobs) > 0 {
		status = tasks.TaskFailed
	}
	taskManager.UpdateTaskStatus(taskID, status, result)
}

// DeleteJenkinsView 删除 Jenkins View（仅删除视图，不删除 Jobs）
func DeleteJenkinsView(c *gin.Context, cfg *config.Config) {
	viewName := c.Param("name")
	if viewName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视图名称不能为空"})
		return
	}

	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	if err := jenkinsClient.DeleteView(viewName); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("删除视图失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("视图 '%s' 已删除", viewName)})
}

// CreateJenkinsCredential 创建 Jenkins SSH 凭据
func CreateJenkinsCredential(c *gin.Context, cfg *config.Config) {
	var req jenkins.SSHCredentialRequest
	// 使用 ShouldBindJSON 来解析请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 打印所有字段的调试信息
	fmt.Printf("[Jenkins] 接收到凭证请求: ID=%s, Username=%s, PrivateKey长度=%d, Desc=%s\n",
		req.ID, req.Username, len(req.GetPrivateKey()), req.Description)

	if req.ID == "" || req.Username == "" || req.GetPrivateKey() == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ID、Username 和 PrivateKey 为必填项 (实际值: ID='%s', Username='%s', PrivateKey长度=%d)",
			req.ID, req.Username, len(req.GetPrivateKey()))})
		return
	}

	username := cfg.Jenkins.Username
	token := cfg.Jenkins.Token
	if cfg.Jenkins.URL == "" || username == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenkins 配置不完整"})
		return
	}

	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, username, token, time.Duration(cfg.Jenkins.Timeout)*time.Second)

	if err := jenkinsClient.CreateSSHCredential(req); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "400") {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("创建凭据失败 (格式错误): %v", err)})
		} else if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Jenkins 认证失败: %v", err)})
		} else if strings.Contains(errMsg, "502") || strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("无法连接到 Jenkins 服务器，请检查 Jenkins 地址和网络连接: %v", err)})
		} else {
			if strings.Contains(errMsg, "已存在") {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("创建凭据失败: %v", err)})
			}
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("凭据 '%s' 创建成功", req.ID)})
}

// 根据视图名称变化推断额外的替换规则并应用到配置
func applyInferredReplacements(config, sourceViewName, targetViewName string) string {
	replacements := inferReplacementsFromViewNames(sourceViewName, targetViewName)

	updatedConfig := config
	for oldVal, newVal := range replacements {
		// 避免无限循环，只替换一次（这里使用全局替换，但需要小心处理避免重复应用）
		updatedConfig = strings.ReplaceAll(updatedConfig, oldVal, newVal)
	}

	return updatedConfig
}

// 根据视图名称变化推断替换规则
func inferReplacementsFromViewNames(sourceViewName, targetViewName string) map[string]string {
	replacements := make(map[string]string)

	// 按分隔符分割视图名称，如 fat-150-V2.5.1 -> ["fat", "150", "V2.5.1"]
	sourceParts := strings.Split(sourceViewName, "-")
	targetParts := strings.Split(targetViewName, "-")

	// 对比各部分，找出变化的模式
	for i, srcPart := range sourceParts {
		if i < len(targetParts) {
			tgtPart := targetParts[i]

			// 如果这部分不同，检查是否只是数字变化
			if srcPart != tgtPart {
				// 尝试找出前缀相同的模式（如 "fat150" 和 "fat160"）
				srcPrefix := extractPrefix(srcPart)
				tgtPrefix := extractPrefix(tgtPart)

				if srcPrefix == tgtPrefix {
					// 看看是否可以从视图名称的变化推断配置中的其他变化
					// 例如：fat-150 到 fat-160 => fat150 到 fat160
					variation1 := strings.ReplaceAll(srcPart, "-", "")
					variation2 := strings.ReplaceAll(tgtPart, "-", "")

					if variation1 != srcPart && variation2 != tgtPart {
						replacements[variation1] = variation2
					}
				}
			}
		}
	}

	// 另一种模式：检查是否在视图名称中有类似 fat-150 的模式，在配置中可能表现为 fat150
	for i, srcPart := range sourceParts {
		if i < len(targetParts) {
			tgtPart := targetParts[i]
			if srcPart != tgtPart {
				// 检查是否是 fat-XXX 模式
				if strings.Contains(srcPart, "-") {
					subParts := strings.Split(srcPart, "-")
					for j := 0; j < len(subParts); j++ {
						if j < len(subParts)-1 && j < len(strings.Split(targetParts[i], "-"))-1 {
							// 检查是否有前缀相同的模式
							nextSrcPart := subParts[j+1]
							nextTgtParts := strings.Split(targetParts[i], "-")
							if j+1 < len(nextTgtParts) {
								nextTgtPart := nextTgtParts[j+1]

								if nextSrcPart != nextTgtPart {
									// 简单地推断替换，例如：150 -> 160
									replacements[srcPart] = tgtPart

									// 如果视图名是 fat-150，则在配置中可能有 fat150 模式
									srcCombined := strings.ReplaceAll(srcPart, "-", "")
									tgtCombined := strings.ReplaceAll(tgtPart, "-", "")
									if srcCombined != srcPart {
										replacements[srcCombined] = tgtCombined
									}
								}
							}
						}
					}
				} else {
					// 如果视图的一部分本身没有分隔符，看是否能匹配配置中的模式
					// 例如，源名是 "fat150"，目标是 "fat160"
					replacements[srcPart] = tgtPart
				}
			}
		}
	}

	return replacements
}

// 提取前缀（非数字部分）
func extractPrefix(s string) string {
	for i, r := range s {
		if r >= '0' && r <= '9' {
			return s[:i]
		}
	}
	return s
}