# Phase 2 Implementation Plan - CMDB 核心功能

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 完成 CMDB 资产管理系统核心功能，包括项目管理、集群管理、服务器管理、应用管理、资产关系展示。

**Architecture:**
- 后端：在 Phase 1 基础上添加 CMDB 模块（CRUD API）
- 前端：添加 CMDB 管理页面（列表、表单、搜索）
- 数据库：在现有 users 表基础上添加 CMDB 相关表

**Tech Stack:** Go (Gin + GORM), React (Ant Design), MySQL 8.0

---

## Task 1: 数据库模型迁移

**Files:**
- Create: \`backend/migrations/000002_add_cmdb_tables.sql\`

**Step 1: 创建迁移脚本**

\`\`\`sql
-- backend/migrations/000002_add_cmdb_tables.sql
USE ops_platform;

-- 项目表
CREATE TABLE IF NOT EXISTS projects (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_name (name, deleted_at),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';

-- 集群表
CREATE TABLE IF NOT EXISTS clusters (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type ENUM('dev', 'test', 'prod') NOT NULL DEFAULT 'dev',
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_name (name, deleted_at),
    KEY idx_type (type),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='集群表';

-- 服务器表
CREATE TABLE IF NOT EXISTS servers (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    hostname VARCHAR(100) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    os VARCHAR(50),
    arch VARCHAR(20),
    status ENUM('online', 'offline', 'maintenance') NOT NULL DEFAULT 'offline',
    ssh_port INT DEFAULT 22,
    project_id BIGINT UNSIGNED NOT NULL,
    cluster_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
    UNIQUE KEY uk_ip (ip, deleted_at),
    KEY idx_hostname (hostname),
    KEY idx_status (status),
    KEY idx_project (project_id),
    KEY idx_cluster (cluster_id),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器表';

-- 应用表
CREATE TABLE IF NOT EXISTS applications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code_repo VARCHAR(255),
    deploy_path VARCHAR(255),
    jenkins_job VARCHAR(100),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_atATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_name (name, deleted_at),
    KEY idx_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用表';

-- 服务器应用关联表
CREATE TABLE IF NOT EXISTS server_apps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    server_id BIGINT UNSIGNED NOT NULL,
    app_id BIGINT UNSIGNED NOT NULL,
    version VARCHAR(50),
    deployed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE,
    UNIQUE KEY uk_server_app (server_id, app_id),
    KEY idx_app (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器应用关联表';

-- 标签表
CREATE TABLE IF NOT EXISTS tags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    key VARCHAR(50) NOT NULL,
    value VARCHAR(100) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_key_value (key, value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签表';

-- 资产标签关联表
CREATE TABLE IF NOT EXISTS asset_tags (
    asset_type ENUM('server', 'cluster', 'application') NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (asset_type, asset_id, tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资产标签关联表';
\`\`\`

**Step 2: Commit**

\`\`\`bash
git add backend/migrations/
git commit -m "migrations: 添加 CMDB 核心表

- projects: 项目表
- clusters: 集群表
- servers: 服务器表
- applications: 应用表
- server_apps: 服务器应用关联表
- tags: 标签表
- asset_tags: 资产标签关联表"
\`\`\`

---

## Task 2: 后端 CMDB 模型

**Files:**
- Create: \`backend/internal/cmdb/models.go\`

**Step 1: 创建 CMDB 模型**

\`\`\`go
package cmdb

import "time"

type Project struct {
	ID          uint      \`json:"id" gorm:"primaryKey"\`
	Name        string    \`json:"name" gorm:"uniqueIndex:name,deleted_at;size:100;not null"\`
	Description string    \`json:"description"\`
	CreatedAt   time.Time \`json:"created_at"\`
	UpdatedAt   time.Time \`json:"updated_at"\`
	DeletedAt   *time.Time \`json:"deleted_at,omitempty" gorm:"index"\`
}

func (Project) TableName() string {
	return "projects"
}

type Cluster struct {
	ID          uint      \`json:"id" gorm:"primaryKey"\`
	Name        string    \`json:"name" gorm:"uniqueIndex:name,deleted_at;size:100;not null"\`
	Type        string    \`json:"type" gorm:"size:10;default:'dev';not null"\`
	Description string    \`json:"description"\`
	CreatedAt   time.Time \`json:"created_at"\`
	UpdatedAt   time.TimeTime \`json:"updated_at"\`
	DeletedAt   *time.Time \`json:"deleted_at,omitempty" gorm:"index"\`
}

func (Cluster) TableName() string {
	return "clusters"
}

type Server struct {
	ID        uint      \`json:"id" gorm:"primaryKey"\`
	Hostname  string    \`json:"hostname" gorm:"size:100;not null"\`
	IP        string    \`json:"ip" gorm:"uniqueIndex:ip,deleted_at;size:45;not null"\`
	OS        string    \`json:"os" gorm:"size:50"\`
	Arch      string    \`json:"arch" gorm:"size:20"\`
	Status    string    \`json:"status" gorm:"size:10;default:'offline';not null"\`
	SSHPort  int       \`json:"ssh_port" gorm:"default:22"\`
	ProjectID uint      \`json:"project_id" gorm:"not null;index"\`
	ClusterID uint      \`json:"cluster_id" gorm:"not null;index"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
	DeletedAt *time.Time \`json:"deleted_at,omitempty" gorm:"index"\`

	// 关联
	Project   *Project \`json:"project,omitempty" gorm:"foreignKey:ProjectID"\`
	Cluster   *Cluster \`json:"cluster,omitempty" gorm:"foreignKey:ClusterID"\`
}

func (Server) TableName() string {
	return "servers"
}

type Application struct {
	ID         uint      \`json:"id" gorm:"primaryKey"\`
	Name       string    \`json:"name" gorm:"uniqueIndex:name,deleted_at;size:100;not null"\`
	CodeRepo   string    \`json:"code_repo"\`
	DeployPath string    \`json:"deploy_path"\`
	JenkinsJob string    \`json:"jenkins_job"\`
	CreatedAt  time.Time \`json:"created_at"\`
	UpdatedAt  time.Time \`json:"updated_at"\`
	DeletedAt  *time.Time \`json:"deleted_at,omitempty" gorm:"index"\`
}

func (Application) TableName() string {
	return "applications"
}

type ServerApp struct {
	ID         uint      \`json:"id" gorm:"primaryKey"\`
	ServerID   uint      \`json:"server_id" gorm:"not null;uniqueIndex:server_app"\`
	AppID     uint      \`json:"app_id" gorm:"not null;index"\`
	Version    string    \`json:"version"\`
	DeployedAt *time.Time \`json:"deployed_at"\`
	CreatedAt  time.Time \`json:"created_at"\`
	UpdatedAt  time.Time \`json:"updated_at"\`

	// 关联
	Server   *Server      \`json:"server,omitempty" gorm:"foreignKey:ServerID"\`
	App      *Application \`json:"app,omitempty" gorm:"foreignKey:AppID"\`
}

func (ServerApp) TableName() string {
	return "server_apps"
}

type Tag struct {
	ID        uint      \`json:"id" gorm:"primaryKey"\`
	Key       string    \`json:"key" gorm:"size:50;not null;uniqueIndex:key_value"\`
	Value     string    \`json:"value" gorm:"size:100;not null"\`
	CreatedAt time.Time \`json:"created_at"\`
}

func (Tag) TableName() string {
	return "tags"
}

type AssetTag struct {
	AssetType string    \`json:"asset_type" gorm:"size:20;not null"\`
	AssetID   uint      \`json:"asset_id" gorm:"not null"\`
	TagID     uint      \`json:"tag_id" gorm:"not null"\`

	// 关联
	Tag       *Tag \`json:"tag,omitempty" gorm:"foreignKey:TagID"\`
}

func (AssetTag) TableName() string {
	return "asset_tags"
}
\`\`\`

**Step 2: Commit**

\`\`\`bash
git add backend/internal/cmdb/models.go
git commit -m "feat: CMDB 后端模型定义

- Project: 项目模型
- Cluster: 集群模型
- Server: 服务器模型
- Application: 应用模型
- ServerApp: 服务器应用关联模型
- Tag: 标签模型
- AssetTag: 资产标签关联模型"
\`\`\`

---

## Task 3: 后端 CMDB 处理器

**Files:**
- Create: \`backend/internal/cmdb/handler.go\`

**Step 1: 创建 CMDB 处理器**

\`\`\`go
package cmdb

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
)

type ServerRequest struct {
	Hostname string \`json:"hostname" binding:"required"\`
	IP       string \`json:"ip" binding:"required,ip"\`
	OS       string \`json:"os"\`
	Arch     string \`json:"arch"\`
	SSHPort  int    \`json:"ssh_port"\`
	ProjectID uint  \`json:"project_id" binding:"required"\`
	ClusterID uint  \`json:"cluster_id" binding:"required"\`
}

type ServerUpdate struct {
	Hostname string \`json:"hostname,omitempty"\`
	IP       string    \`json:"ip,omitempty"\`
	OS       string    \`json:"os,omitempty"\`
	Arch     string    \`json:"arch,omitempty"\`
	Status   string    \`json:"status,omitempty"\`
	SSHPort  int    \`json:"ssh_port,omitempty"\`
	ProjectID uint  \`json:"project_id,omitempty"\`
	ClusterID uint  \`json:"cluster_id,omitempty"\`
}

type ClusterRequest struct {
	Name        string \`json:"name" binding:"required"\`
	Type        string \`json:"type" binding:"required,oneof=dev test prod"\`
	Description string \`json:"description"\`
}

type ProjectRequest struct {
	Name        string \`json:"name" binding:"required"\`
	Description string \`json:"description"\`
}

type ApplicationRequest struct {
	Name       string \`json:"name" binding:"required"\`
	CodeRepo  string \`json:"code_repo"\`
	DeployPath fanc \`json:"deploy_path"\`
	JenkinsJob string \`json:"jenkins_job"\`
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

		// 集群
		api.GET("/clusters", GetClusters)
		api.GET("/clusters/:id", GetCluster)
		api.POST("/clusters", CreateCluster)
		api.PUT("/clusters/:id", UpdateCluster)
		api.DELETE("/clusters/:id", DeleteCluster)

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
	}
}

// ========== 服务器 CRUD ==========

func GetServers(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	clusterID, _ := strconv.Atoi(c(c.Query("cluster_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	query := database.DB.Where("deleted_at IS NULL")
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	var servers []models.Server
	offset := (page - 1) * limit
	if err := query.Preload("Project").Preload("Cluster").
		Offset(offset).Limit(limit).Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch servers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   servers,
		"page":   page,
		"limit":  limit,
		"total":  len(servers),
	})
}

func GetServer(c *gin.Context) {
	id := c.Param("id")

	var server models.Server
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Project").Preload("Cluster").First(&server).Error; err != nil {
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

	server := models.Server{
		Hostname:  req.Hostname,
		IP:        req.IP,
		OS:        req.OS,
		Arch:      req.Arch,
		SSHPort:   req.SSHPort,
		ProjectID:  req.ProjectID,
		ClusterID:  req.ClusterID,
		Status:    "offline",
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

	var server models.Server
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&server).Error; err != nil {
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
	if reqStatus != "" {
		server.Status = req.Status
	}
	if req.SSHPort != 0 {
		server.SSHPort = req.SSHPort
	}

	if err := database.DB.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update server"})
		return
	}

	c.JSON(http.StatusOK, server)
}

func DeleteServer(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Server{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "server deleted"})
}

// ========== 集群 CRUD ==========

func GetClusters(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var clusters []models.Cluster
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Offset(offset).Limit(limit).Find(&clusters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch clusters"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   clusters,
		"page":   page,
		"limit":  limit,
		"total":  len(clusters),
	})
}

func GetCluster(c *gin.Context) {
	id := c.Param("id")

	var cluster models.Cluster
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&cluster).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	c.JSON(http.StatusOK, cluster)
}

func CreateCluster(c *gin.Context) {
	var req ClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	cluster := models.Cluster{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
	}

	if err := database.DB.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cluster"})
		return
	}

	c.JSON(http.StatusCreated, cluster)
}

func UpdateCluster(c *gin.Context) {
	id := c.Param("id")

	var req ClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var cluster models.Cluster
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&cluster).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	if req.Name != "" {
		cluster.Name = req.Name
	}
	if req.Type != "" {
		cluster.Type = req.Type
	}
	if req.Description != "" {
		cluster.Description = req.Description
	}

	if err := database.DB.Save(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update cluster"})
		return
	}

	c.JSON(http.StatusOK, cluster)
}

func DeleteCluster(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Cluster{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete cluster"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster deleted"})
}

// ========== 项目 CRUD ==========

func GetProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var projects []models.Project
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   projects,
		"page":   page,
		"limit":  limit,
		"total":  len(projects),
	})
}

func GetProject(c *gin.Context) {
	id := c.Param("id")

	var project models.Project
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

	project := models.Project{
		Name:        req.Name,
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

	var project models.Project
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	if req.Name != "" {
		project.Name = req.Name
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

	if err := database.DB.Delete(&models.Project{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "project deleted"})
}

// ========== 应用 CRUD ==========

func GetApplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var apps []models.Application
	offset := (page - 1) * limit
	if err := database.DB.Where("deleted_at IS NULL").
		Offset(offset).Limit(limit).Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   apps,
		"page":   page,
		"limit":  limit,
		"total":  len(apps),
	})
}

func GetApplication(c *gin.Context) {
	id := c.Param("id")

	var app models.Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&app).Error; err != nil {
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

	app := models.Application{
		Name:       req.Name,
		CodeRepo:   req.CodeRepo,
		DeployPath: req.DeployPath,
		JenkinsJob: req.JenkinsJob,
	}

	if err := database.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create application"})
		return
	}

	c.JSON(http.StatusCreated, app)
}

func UpdateApplication(c *gin.Context) {
	id := c.Param("id")

	var req ApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var app models.Application
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&app).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.CodeRepo != "" {
		app.CodeRepo = req.CodeRepo
	}
	if req.DeployPath != "" {
		app.DeployPath = req.DeployPath
	}
	if req.JenkinsJob != "" {
		app.JenkinsJob = req.JenkinsJob
	}

	if err := database.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update application"})
		return
	}

	c.JSON(http.StatusOK, app)
}

func DeleteApplication(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Application{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application deleted"})
}
\`\`\`

**Step 2: 更新 server.go 注册 CMDB 路由**

\`\`\`go
// 添加 CMDB 路由注册
cmdb.RegisterRoutes(r, cfg)
\`\`\`

**Step 3: Commit**

\`\`\`bash
git add backend/internal/cmdb/handler.go backend/internal/server/server.go
git commit -m "feat: CMDB 后端 CRUD API

- 服务器 CRUD
- 集群 CRUD
- 项目 CRUD
- 应用 CRUD"
\`\`\`

---

## Task 4: 前端 CMDB API 客户端

**Files:**
- Create: \`frontend/src/api/cmdb.ts\`

**Step 1: 创建 CMDB API 客户端**

\`\`\`ts
import apiClient from './client'

export interface Server {
  id: number
  hostname: string
  ip: string
  os?: string
  arch?: string
  status: 'online' | 'offline' | 'maintenance'
  ssh_port: number
  project_id: number
  cluster_id: number
  project?: Project
  cluster?: Cluster
  created_at: string
  updated_at: string
}

export interface Cluster {
  id: number
  name: string
  type: 'dev' | 'test' | 'prod'
  description?: string
  created_at: string
  updated_at: string
}

export interface Project {
  id: number
  name: string
  description?: string
  created_at: string
  updated_at: string
}

export interface Application {
  id: number
  name: string
  code_repo?: string
  deploy_path?: string
  jenkins_job?: string
  created_at: string
  updated_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export const cmdbAPI = {
  // 服务器
  getServers: (params?: { project_id?: number; cluster_id?: number; page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Server>>('/cmdb/servers', { params }),

  getServer: (id: number) =>
    apiClient.get<Server>(\`/cmdb/servers/\${id}\`),

  createServer: (data: Omit<Server, 'id'>) =>
    apiClient.post<Server>('/cmdb/servers', data),

  updateServer: (id: number, data: Partial<Server>) =>
    apiClient.put<Server>(\`/cmdb/servers/\${id}\`, data),

  deleteServer: (id: number) =>
    apiClient.delete<Server>(\`/cmdb/servers/\${id}\`),

  // 集群
  getClusters: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Cluster>>('/cmdb/clusters', { params }),

  getCluster: (id: number) =>
    apiClient.get<Cluster>(\`/cmdb/clusters/\${id}\`),

  createCluster: (data: Omit<Cluster, 'id'>) =>
    apiClient.post<Cluster>('/cmdb/clusters', data),

  updateCluster: (id: number, data: Partial<Cluster>) =>
    apiClient.put<Cluster>(\`/cmdb/clusters/\${id}\`, data),

  deleteCluster: (id: number) =>
    apiClient.delete<Cluster>(\`/cmdb/clusters/\${id}\`),

  // 项目
  getProjects: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Project>>('/cmdb/projects', { params }),

  getProject: (id: number) =>
    apiClient.get<Project>(\`/cmdb/projects/\${id}\`),

  createProject: (data: Omit<Project, 'id'>) =>
    apiClient.post<Project>('/cmdb/projects', data),

  updateProject: (id: number, data: Partial<Project>) =>
    apiClient.put<Project>(\`/cmdb/projects/\${id}\`, data),

  deleteProject: (id: number) =>
    apiClient.delete<Project>(\`/cmdb/projects/\${id}\`),

  // 应用
  getApplications: (params?: { page?: number; limit?: number }) =>
    apiClient.get<PaginatedResponse<Application>>('/cmdb/applications', { params }),

  getApplication: (id: number) =>
    apiClient.get<Application>(\`/cmdb/applications/\${id}\`),

  createApplication: (data: Omit<Application, 'id'>) =>
    apiClient.post<Application>('/cmdb/applications', data),

  updateApplication: (id: number, data: Partial<Application>) =>
    apiClient.put<Application>(\`/cmdb/applications/\${id}\`, data),

  deleteApplication: (id: number) =>
    apiClient.delete<Application>(\`/cmdb/applications/\${id}\`),
}
\`\`\`

**Step 2: Commit**

\`\`\`bash
git add frontend/src/api/cmdb.ts
git commit -m "feat: CMDB API 客户端

- 服务器 API
- 集群 API
- 项目 API
- 应用 API"
\`\`\`

---

## 完成清单

- [ ] 数据库模型迁移
- [ ] 后端 CMDB 模型
- [ ] 后端 CMDB 处理器
- [ ] 前端 CMDB API 客户端
- [ ] 前端 CMDB 页面组件

---

## 预计时间：2 周

每个任务预计 2-4 天，留出缓冲时间处理意外问题。
