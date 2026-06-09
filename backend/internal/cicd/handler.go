package cicd

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
)

// PipelineRequest 流水线创建/更新请求
type PipelineRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	Repository     string `json:"repository" binding:"required"`
	TriggerMode    string `json:"trigger_mode" binding:"required,oneof=manual scheduled webhook"`
	CronExpression string `json:"cron_expression"`
	Branch         string `json:"branch"`
	YAMLConfig     string `json:"yaml_config"`
}

// PipelineUpdate 流水线更新请求
type PipelineUpdate struct {
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	Repository     string `json:"repository,omitempty"`
	TriggerMode    string `json:"trigger_mode,omitempty"`
	CronExpression string `json:"cron_expression,omitempty"`
	Branch         string `json:"branch,omitempty"`
	YAMLConfig     string `json:"yaml_config,omitempty"`
}

// TriggerRequest 触发流水线执行请求
type TriggerRequest struct {
	Branch        string `json:"branch"`
	CommitID      string `json:"commit_id"`
	CommitMessage string `json:"commit_message"`
}

// RegisterRoutes 注册 CI/CD 路由
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	api := r.Group("/api/cicd")
	api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		// 流水线 CRUD
		api.GET("/pipelines", GetPipelines)
		api.GET("/pipelines/:id", GetPipeline)
		api.POST("/pipelines", CreatePipeline)
		api.PUT("/pipelines/:id", UpdatePipeline)
		api.DELETE("/pipelines/:id", DeletePipeline)

		// 触发流水线执行
		api.POST("/pipelines/:id/trigger", TriggerPipeline)

		// 执行历史
		api.GET("/pipelines/:id/executions", GetPipelineExecutions)
		api.GET("/executions/:id", GetExecution)
		api.GET("/executions/:id/stages", GetExecutionStages)
		api.POST("/executions/:id/cancel", CancelExecution)

		// 执行日志
		api.GET("/executions/:id/logs", GetExecutionLogs)
	}
}

// ========== 流水线 CRUD ==========

// GetPipelines 获取流水线列表
func GetPipelines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	triggerMode := c.Query("trigger_mode")

	var pipelines []models.Pipeline
	query := database.DB.Where("deleted_at IS NULL")

	if triggerMode != "" {
		query = query.Where("trigger_mode = ?", triggerMode)
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pipelines"})
		return
	}

	var total int64
	database.DB.Model(&models.Pipeline{}).Where("deleted_at IS NULL").Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  pipelines,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetPipeline 获取单个流水线
func GetPipeline(c *gin.Context) {
	id := c.Param("id")

	var pipeline models.Pipeline
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&pipeline).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// CreatePipeline 创建流水线
func CreatePipeline(c *gin.Context) {
	var req PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	pipeline := models.Pipeline{
		Name:           req.Name,
		Description:    req.Description,
		Repository:     req.Repository,
		TriggerMode:    models.PipelineTriggerMode(req.TriggerMode),
		CronExpression: req.CronExpression,
		Branch:         req.Branch,
		YAMLConfig:     req.YAMLConfig,
	}

	if err := database.DB.Create(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create pipeline"})
		return
	}

	c.JSON(http.StatusCreated, pipeline)
}

// UpdatePipeline 更新流水线
func UpdatePipeline(c *gin.Context) {
	id := c.Param("id")

	var req PipelineUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var pipeline models.Pipeline
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&pipeline).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	// 更新字段
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Repository != "" {
		updates["repository"] = req.Repository
	}
	if req.TriggerMode != "" {
		updates["trigger_mode"] = models.PipelineTriggerMode(req.TriggerMode)
	}
	if req.CronExpression != "" {
		updates["cron_expression"] = req.CronExpression
	}
	if req.Branch != "" {
		updates["branch"] = req.Branch
	}
	if req.YAMLConfig != "" {
		updates["yaml_config"] = req.YAMLConfig
	}

	if err := database.DB.Model(&pipeline).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pipeline"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// DeletePipeline 删除流水线
func DeletePipeline(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Pipeline{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete pipeline"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pipeline deleted"})
}

// ========== 触发流水线执行 ==========

// TriggerPipeline 触发流水线执行
func TriggerPipeline(c *gin.Context) {
	pipelineID := c.Param("id")

	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供请求体，使用默认值
		req = TriggerRequest{
			Branch: "main",
		}
	}

	var pipeline models.Pipeline
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", pipelineID).First(&pipeline).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	// 创建执行记录
	now := time.Now()
	execution := models.PipelineExecution{
		PipelineID:    pipeline.ID,
		TriggerMode:   pipeline.TriggerMode,
		Branch:        req.Branch,
		CommitID:      req.CommitID,
		CommitMessage: req.CommitMessage,
		Status:        models.PipelineStatusRunning,
		StartedAt:     &now,
		ExecutorName:  "Manual", // TODO: 从认证上下文中获取
	}

	if err := database.DB.Create(&execution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create execution"})
		return
	}

	// TODO: 异步执行流水线任务

	// 更新流水线的最近执行信息
	pipeline.LastExecutionID = &execution.ID
	nowPtr := time.Now()
	pipeline.LastExecutionTime = &nowPtr
	database.DB.Model(&pipeline).Updates(map[string]interface{}{
		"last_execution_id":     execution.ID,
		"last_execution_time":   nowPtr,
		"last_execution_status": models.PipelineStatusRunning,
	})

	c.JSON(http.StatusCreated, execution)
}

// GetPipelineExecutions 获取流水线执行历史
func GetPipelineExecutions(c *gin.Context) {
	pipelineID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	var executions []models.PipelineExecution
	query := database.DB.Where("pipeline_id = ?", pipelineID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch executions"})
		return
	}

	var total int64
	database.DB.Model(&models.PipelineExecution{}).Where("pipeline_id = ?", pipelineID).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  executions,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetExecution 获取执行详情
func GetExecution(c *gin.Context) {
	id := c.Param("id")

	var execution models.PipelineExecution
	if err := database.DB.Where("id = ?", id).First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetExecutionStages 获取执行的阶段列表
func GetExecutionStages(c *gin.Context) {
	id := c.Param("id")

	var stages []models.PipelineStage
	if err := database.DB.Where("execution_id = ?", id).Order("stage_order ASC").Find(&stages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stages"})
		return
	}

	c.JSON(http.StatusOK, stages)
}

// CancelExecution 取消执行
func CancelExecution(c *gin.Context) {
	id := c.Param("id")

	var execution models.PipelineExecution
	if err := database.DB.Where("id = ?", id).First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	if execution.Status != models.PipelineStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution is not running"})
		return
	}

	now := time.Now()
	execution.Status = models.PipelineStatusCancelled
	execution.FinishedAt = &now
	execution.Duration = int64(now.Unix() - execution.StartedAt.Unix())

	database.DB.Save(&execution)

	c.JSON(http.StatusOK, gin.H{"message": "execution cancelled"})
}

// GetExecutionLogs 获取执行日志
func GetExecutionLogs(c *gin.Context) {
	id := c.Param("id")
	stageID := c.Query("stage_id")

	var logs string
	var err error

	if stageID != "" {
		var stage models.PipelineStage
		err = database.DB.Model(&models.PipelineStage{}).Where("id = ? AND execution_id = ?", stageID, id).First(&stage).Error
		if err == nil {
			logs = stage.Logs
		}
	} else {
		// 返回所有阶段的日志
		var stages []models.PipelineStage
		err = database.DB.Where("execution_id = ?", id).Order("stage_order ASC").Find(&stages).Error
		if err == nil {
			for _, stage := range stages {
				logs += "=== Stage: " + stage.Name + " ===\n"
				logs += stage.Logs + "\n\n"
			}
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution or stage not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
