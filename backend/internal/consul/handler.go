// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package consul

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler Consul API 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建 Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	consul := r.Group("/consul")
	{
		// 配置管理
		consul.GET("/configs", h.GetConfigs)
		consul.GET("/projects", h.ListProjects)
		consul.GET("/configs/:id", h.GetConfig)
		consul.POST("/configs", h.CreateConfig)
		consul.PUT("/configs/:id", h.UpdateConfig)
		consul.DELETE("/configs/:id", h.DeleteConfig)
		consul.POST("/configs/:id/test", h.TestConnection)

		// KV 操作
		consul.GET("/kv", h.ListKeys)
		consul.GET("/kv/:key", h.GetKeyValue)
		consul.PUT("/kv/:key", h.PutKeyValue)
		consul.DELETE("/kv/:key", h.DeleteKey)
		consul.POST("/kv/copy", h.CopyKey)

		// 批量复制（类似参考脚本功能）
		consul.POST("/kv/batch-copy", h.BatchCopyKeys)

		// 批量复制所有项目（一键复制）
		consul.POST("/kv/batch-copy-all", h.BatchCopyAllProjects)

		// 批量删除
		consul.POST("/kv/query-suffix-keys", h.QuerySuffixKeys)
		consul.POST("/kv/batch-delete", h.BatchDeleteKeys)

		// 替换规则
		consul.GET("/rules", h.GetReplaceRules)
		consul.POST("/rules", h.CreateReplaceRule)
		consul.PUT("/rules/:id", h.UpdateReplaceRule)
		consul.DELETE("/rules/:id", h.DeleteReplaceRule)

		// 操作历史
		consul.GET("/operations", h.GetOperations)
		consul.DELETE("/operations/:id", h.DeleteOperation)
	}
}

func (h *Handler) ListProjects(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置 ID"})
		return
	}

	if configID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		configID = uint64(config.ID)
	}

	projects, err := h.service.ListProjects(uint(configID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

// GetConfigs 获取所有配置
func (h *Handler) GetConfigs(c *gin.Context) {
	configs, err := h.service.GetConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetConfig 获取单个配置
func (h *Handler) GetConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	config, err := h.service.GetConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateConfig 创建配置
func (h *Handler) CreateConfig(c *gin.Context) {
	var config ConsulConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateConfig 更新配置
func (h *Handler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var config ConsulConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.ID = uint(id)
	if err := h.service.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteConfig 删除配置
func (h *Handler) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := h.service.DeleteConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestConnection 测试连接
func (h *Handler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	config, err := h.service.GetConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	if err := h.service.TestConnection(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "连接成功"})
}

// ListKeys 列出 KV 键
func (h *Handler) ListKeys(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置 ID"})
		return
	}

	// 如果没有指定配置 ID，使用默认配置
	if configID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		configID = uint64(config.ID)
	}

	prefix := c.DefaultQuery("prefix", "")
	recurse := c.Query("recurse") == "true"

	keys, err := h.service.ListKeys(uint(configID), prefix, recurse)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果请求树形结构
	if c.Query("tree") == "true" {
		tree := h.service.BuildKVTree(keys, prefix)
		c.JSON(http.StatusOK, gin.H{
			"keys": keys,
			"tree": tree,
		})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// GetKeyValue 获取键值
func (h *Handler) GetKeyValue(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置 ID"})
		return
	}

	if configID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		configID = uint64(config.ID)
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "键名不能为空"})
		return
	}

	kv, err := h.service.GetKeyValue(uint(configID), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, kv)
}

// PutKeyValue 设置键值
func (h *Handler) PutKeyValue(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置 ID"})
		return
	}

	if configID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		configID = uint64(config.ID)
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "键名不能为空"})
		return
	}

	var body struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.PutKeyValue(uint(configID), key, body.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteKey 删除键
func (h *Handler) DeleteKey(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, err := strconv.ParseUint(configIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置 ID"})
		return
	}

	if configID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		configID = uint64(config.ID)
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "键名不能为空"})
		return
	}

	if err := h.service.DeleteKey(uint(configID), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// CopyKey 复制键
func (h *Handler) CopyKey(c *gin.Context) {
	var req CopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)

	result, err := h.service.CopyKey(req.ConfigID, &req, operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetReplaceRules 获取替换规则
func (h *Handler) GetReplaceRules(c *gin.Context) {
	rules, err := h.service.GetReplaceRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

// CreateReplaceRule 创建替换规则
func (h *Handler) CreateReplaceRule(c *gin.Context) {
	var rule ReplaceRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateReplaceRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// UpdateReplaceRule 更新替换规则
func (h *Handler) UpdateReplaceRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var rule ReplaceRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = uint(id)
	if err := h.service.UpdateReplaceRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

// DeleteReplaceRule 删除替换规则
func (h *Handler) DeleteReplaceRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := h.service.DeleteReplaceRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetOperations 获取操作历史
func (h *Handler) GetOperations(c *gin.Context) {
	configIDStr := c.DefaultQuery("config_id", "0")
	configID, _ := strconv.ParseUint(configIDStr, 10, 32)

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	operations, err := h.service.GetOperations(uint(configID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, operations)
}

// BatchCopyKeys 批量复制键（支持目录级复制）
func (h *Handler) BatchCopyKeys(c *gin.Context) {
	var req BatchCopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)

	if req.ConfigID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		req.ConfigID = config.ID
	}

	result, err := h.service.BatchCopyKeys(req.ConfigID, &req, operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// BatchCopyAllProjects 批量复制所有项目（一键复制）
func (h *Handler) BatchCopyAllProjects(c *gin.Context) {
	var req BatchCopyAllProjectsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operator := getOperatorName(c)

	if req.ConfigID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		req.ConfigID = config.ID
	}

	result, err := h.service.BatchCopyAllProjects(req.ConfigID, &req, operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// QuerySuffixKeys 查询指定后缀的所有项目 Key
func (h *Handler) QuerySuffixKeys(c *gin.Context) {
	var req struct {
		ConfigID uint   `json:"config_id"`
		Suffix   string `json:"suffix" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConfigID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		req.ConfigID = config.ID
	}

	keys, err := h.service.ListProjectSuffixKeys(req.ConfigID, req.Suffix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":  keys,
		"total": len(keys),
	})
}

// BatchDeleteKeys 批量删除 KV 键
func (h *Handler) BatchDeleteKeys(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConfigID == 0 {
		config, err := h.service.GetDefaultConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 Consul 连接"})
			return
		}
		req.ConfigID = config.ID
	}

	result, err := h.service.BatchDeleteKeys(req.ConfigID, req.Keys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getOperatorName 从上下文获取操作人姓名
func getOperatorName(c *gin.Context) string {
	if realName, exists := c.Get("real_name"); exists {
		if name, ok := realName.(string); ok && name != "" {
			return name
		}
	}
	if username, exists := c.Get("username"); exists {
		if name, ok := username.(string); ok && name != "" {
			return name
		}
	}
	return ""
}

// DeleteOperation 删除操作记录
func (h *Handler) DeleteOperation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	if err := h.service.DeleteOperation(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// init 小写函数名修复
func init() {
	_ = strings.Replace
}