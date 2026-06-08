package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/edy/ops-platform/internal/audit"
	"github.com/edy/ops-platform/internal/auth"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/platformevent"
	"github.com/edy/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type auditAlertRuleRecord struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	GrafanaUID     string `json:"grafana_uid"`
	RuleGroup      string `json:"rule_group"`
	FolderTitle    string `json:"folder_title"`
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Enabled        bool   `json:"enabled"`
	NotifyChannels string `json:"notify_channels"`
}

type auditAlertEventRecord struct {
	ID         uint    `json:"id"`
	RuleName   string  `json:"rule_name"`
	Severity   string  `json:"severity"`
	Category   string  `json:"category"`
	Source     string  `json:"source"`
	Status     string  `json:"status"`
	HandleType string  `json:"handle_type"`
	HandleNote string  `json:"handle_note"`
	AckedBy    string  `json:"acked_by"`
	ClosedBy   string  `json:"closed_by"`
	AckedAt    *string `json:"acked_at,omitempty"`
	ClosedAt   *string `json:"closed_at,omitempty"`
}

func buildAuditAlertRuleRecord(rule AlertRule) auditAlertRuleRecord {
	return auditAlertRuleRecord{
		ID:             rule.ID,
		Name:           rule.Name,
		GrafanaUID:     rule.GrafanaUID,
		RuleGroup:      rule.RuleGroup,
		FolderTitle:    rule.FolderTitle,
		Severity:       rule.Severity,
		Category:       rule.Category,
		Description:    rule.Description,
		Enabled:        rule.Enabled,
		NotifyChannels: rule.NotifyChannels,
	}
}

func buildAuditAlertEventRecord(event AlertEvent) auditAlertEventRecord {
	record := auditAlertEventRecord{
		ID:         event.ID,
		RuleName:   event.RuleName,
		Severity:   event.Severity,
		Category:   event.Category,
		Source:     event.Source,
		Status:     event.Status,
		HandleType: event.HandleType,
		HandleNote: event.HandleNote,
		AckedBy:    event.AckedBy,
		ClosedBy:   event.ClosedBy,
	}
	if event.AckedAt != nil && !event.AckedAt.IsZero() {
		value := event.AckedAt.Format("2006-01-02 15:04:05")
		record.AckedAt = &value
	}
	if event.ClosedAt != nil && !event.ClosedAt.IsZero() {
		value := event.ClosedAt.Format("2006-01-02 15:04:05")
		record.ClosedAt = &value
	}
	return record
}

// Prometheus 配置
var prometheusURL = func() string {
	if u := os.Getenv("PROMETHEUS_URL"); u != "" {
		return u
	}
	return "http://localhost:9090"
}()

// queryCurrentValue 查询 Prometheus 获取当前指标值
// 根据 ruleName 确定正确的查询表达式
func queryCurrentValue(labels map[string]string, ruleName string) string {
	// 获取 instance 标签
	instance := labels["instance"]
	if instance == "" {
		return ""
	}

	// 根据告警名称确定查询表达式
	var baseExpr string
	ruleNameLower := strings.ToLower(ruleName)

	if strings.Contains(ruleNameLower, "cpu") {
		// CPU 使用率
		baseExpr = "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"
	} else if strings.Contains(ruleNameLower, "memory") || strings.Contains(ruleNameLower, "mem") || strings.Contains(ruleNameLower, "内存") {
		// 内存使用率
		baseExpr = "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100"
	} else if strings.Contains(ruleNameLower, "disk") || strings.Contains(ruleNameLower, "磁盘") {
		// 磁盘使用率
		baseExpr = "(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) * 100"
	} else if strings.Contains(ruleNameLower, "load") || strings.Contains(ruleNameLower, "负载") {
		// 系统负载
		baseExpr = "node_load1"
	} else if strings.Contains(ruleNameLower, "network") || strings.Contains(ruleNameLower, "网络") {
		// 网络流量（字节/秒）
		baseExpr = "rate(node_network_receive_bytes_total[5m])"
	} else if strings.Contains(ruleNameLower, "diskio") || strings.Contains(ruleNameLower, "磁盘IO") {
		// 磁盘 IO
		baseExpr = "rate(node_disk_io_time_seconds_total[5m])"
	} else {
		// 默认尝试 CPU 查询
		baseExpr = "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"
	}

	// 查询所有实例的 CPU 使用率
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, url.QueryEscape(baseExpr)))
	if err != nil {
		log.Printf("[alert] 查询 CPU 失败: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	json.Unmarshal(body, &result)

	// 查找匹配的 instance
	for _, r := range result.Data.Result {
		if r.Metric["instance"] == instance && len(r.Value) >= 2 {
			value := fmt.Sprintf("%v", r.Value[1])
			// 格式化数值为百分比
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				return fmt.Sprintf("%.2f%%", num)
			}
			return value
		}
	}

	return ""
}

// RegisterRoutes 注册告警路由
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	// Webhook 接口（无需认证，供 Alertmanager 调用）
	r.POST("/api/alert/webhook", AlertWebhook)

	api := r.Group("/api/alert")
	api.Use(auth.AuthMiddleware(cfg.JWT.Secret))
	{
		// 告警规则
		api.GET("/rules", GetRules)
		api.POST("/rules", CreateRule)
		api.PUT("/rules/:id", UpdateRule)
		api.DELETE("/rules/:id", DeleteRule)
		api.POST("/rules/sync", SyncGrafanaRules)

		// 联系人
		api.GET("/contacts", GetContacts)
		api.POST("/contacts", CreateContact)
		api.PUT("/contacts/:id", UpdateContact)
		api.DELETE("/contacts/:id", DeleteContact)

		// 报警组
		api.GET("/groups", GetGroups)
		api.POST("/groups", CreateGroup)
		api.PUT("/groups/:id", UpdateGroup)
		api.DELETE("/groups/:id", DeleteGroup)

		// 通知渠道
		api.GET("/channels", GetChannels)
		api.POST("/channels", CreateChannel)
		api.PUT("/channels/:id", UpdateChannel)
		api.DELETE("/channels/:id", DeleteChannel)
		api.POST("/channels/:id/test", TestChannel)

		// 告警模板
		api.GET("/templates", GetTemplates)
		api.POST("/templates", CreateTemplate)
		api.PUT("/templates/:id", UpdateTemplate)
		api.DELETE("/templates/:id", DeleteTemplate)
		api.POST("/templates/preview", PreviewTemplate)
		api.POST("/templates/:id/default", SetDefaultTemplate)

		// 告警事件
		api.GET("/events", GetEvents)
		api.GET("/events/:id", GetEvent)
		api.PUT("/events/:id/ack", AckEvent)
		api.PUT("/events/:id/close", CloseEvent)
		api.POST("/events/:id/note", AddEventNote)
		api.GET("/events/:id/logs", GetEventLogs)
		api.GET("/events/stats", GetEventStats)
		api.DELETE("/events/:id", DeleteEvent)
	}
}

// ==================== 告警规则 ====================

func GetRules(c *gin.Context) {
	var rules []AlertRule
	query := database.DB.Preload("Group")

	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if enabled := c.Query("enabled"); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true")
	}

	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

func CreateRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	audit.SetOperationAuditAfter(c, buildAuditAlertRuleRecord(rule))
	audit.SetOperationAuditSummary(c, "创建了告警规则。")
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule AlertRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	before := buildAuditAlertRuleRecord(rule)

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := database.DB.Model(&rule).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := database.DB.First(&rule, id).Error; err == nil {
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, buildAuditAlertRuleRecord(rule))
		audit.SetOperationAuditSummary(c, "更新了告警规则。")
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func DeleteRule(c *gin.Context) {
	id := c.Param("id")
	var rule AlertRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	if err := database.DB.Delete(&AlertRule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	audit.SetOperationAuditBefore(c, buildAuditAlertRuleRecord(rule))
	audit.SetOperationAuditSummary(c, "删除了告警规则。")
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// SyncGrafanaRules 从 Grafana 同步告警规则
func SyncGrafanaRules(c *gin.Context) {
	// 接收前端传来的 Grafana 规则数据并同步到数据库
	var input struct {
		Rules []struct {
			GrafanaUID  string `json:"grafana_uid"`
			Name        string `json:"name"`
			RuleGroup   string `json:"rule_group"`
			FolderTitle string `json:"folder_title"`
			State       string `json:"state"`
			Expression  string `json:"expression"`
		} `json:"rules"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	now := time.Now()
	synced := 0
	created := 0

	for _, r := range input.Rules {
		var existing AlertRule
		err := database.DB.Where("grafana_uid = ?", r.GrafanaUID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// 新规则，自动分类
			category := guessCategory(r.Name)
			newRule := AlertRule{
				GrafanaUID:   r.GrafanaUID,
				Name:         r.Name,
				RuleGroup:    r.RuleGroup,
				FolderTitle:  r.FolderTitle,
				Severity:     "warning",
				Category:     category,
				Expression:   r.Expression,
				Enabled:      true,
				GrafanaState: r.State,
				SyncedAt:     &now,
			}
			database.DB.Create(&newRule)
			created++
		} else if err == nil {
			// 更新已有规则
			database.DB.Model(&existing).Updates(map[string]interface{}{
				"name":          r.Name,
				"rule_group":    r.RuleGroup,
				"folder_title":  r.FolderTitle,
				"grafana_state": r.State,
				"expression":    r.Expression,
				"synced_at":     &now,
			})
			synced++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("同步完成：新增 %d 条，更新 %d 条", created, synced),
		"created": created,
		"synced":  synced,
	})
}

// guessCategory 根据规则名称自动推断分类
func guessCategory(name string) string {
	keywords := map[string][]string{
		"disk":     {"磁盘", "disk", "分区", "partition", "存储"},
		"memory":   {"内存", "memory", "mem", "swap"},
		"cpu":      {"CPU", "cpu", "处理器"},
		"instance": {"存活", "alive", "up", "down", "实例", "instance", "宕机"},
		"network":  {"网络", "network", "带宽", "bandwidth"},
		"load":     {"负载", "load"},
	}
	for cat, kws := range keywords {
		for _, kw := range kws {
			if containsIgnoreCase(name, kw) {
				return cat
			}
		}
	}
	return "other"
}

func containsIgnoreCase(s, substr string) bool {
	// 简单的中英文包含匹配
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== 联系人 ====================

func GetContacts(c *gin.Context) {
	var contacts []AlertContact
	if err := database.DB.Preload("Groups").Order("created_at DESC").Find(&contacts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": contacts})
}

func CreateContact(c *gin.Context) {
	var contact AlertContact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := database.DB.Create(&contact).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": contact})
}

func UpdateContact(c *gin.Context) {
	id := c.Param("id")
	var contact AlertContact
	if err := database.DB.First(&contact, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "联系人不存在"})
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := database.DB.Model(&contact).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": contact})
}

func DeleteContact(c *gin.Context) {
	id := c.Param("id")
	// 先删除关联
	database.DB.Exec("DELETE FROM alert_group_contacts WHERE alert_contact_id = ?", id)
	if err := database.DB.Delete(&AlertContact{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 报警组 ====================

func GetGroups(c *gin.Context) {
	var groups []AlertNotifyGroup
	if err := database.DB.Preload("Contacts").Order("created_at DESC").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func CreateGroup(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
		ContactIDs  []uint `json:"contact_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	group := AlertNotifyGroup{
		Name:        input.Name,
		Description: input.Description,
		Enabled:     input.Enabled,
	}

	if err := database.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	// 添加联系人关联
	if len(input.ContactIDs) > 0 {
		var contacts []AlertContact
		database.DB.Where("id IN ?", input.ContactIDs).Find(&contacts)
		database.DB.Model(&group).Association("Contacts").Replace(contacts)
	}

	// 重新查询带关联的数据
	database.DB.Preload("Contacts").First(&group, group.ID)
	c.JSON(http.StatusOK, gin.H{"data": group})
}

func UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var group AlertNotifyGroup
	if err := database.DB.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "报警组不存在"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
		ContactIDs  []uint `json:"contact_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	database.DB.Model(&group).Updates(map[string]interface{}{
		"name":        input.Name,
		"description": input.Description,
		"enabled":     input.Enabled,
	})

	// 更新联系人关联
	var contacts []AlertContact
	if len(input.ContactIDs) > 0 {
		database.DB.Where("id IN ?", input.ContactIDs).Find(&contacts)
	}
	database.DB.Model(&group).Association("Contacts").Replace(contacts)

	database.DB.Preload("Contacts").First(&group, group.ID)
	c.JSON(http.StatusOK, gin.H{"data": group})
}

func DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	database.DB.Exec("DELETE FROM alert_group_contacts WHERE alert_notify_group_id = ?", id)
	if err := database.DB.Delete(&AlertNotifyGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 通知渠道 ====================

func GetChannels(c *gin.Context) {
	var channels []NotifyChannel
	query := database.DB.Order("created_at DESC")
	if channelType := c.Query("type"); channelType != "" {
		query = query.Where("type = ?", channelType)
	}
	if err := query.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	// 脱敏：隐藏 SMTP 密码
	for i := range channels {
		if channels[i].SMTPPass != "" {
			channels[i].SMTPPass = "******"
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func CreateChannel(c *gin.Context) {
	var channel NotifyChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := database.DB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channel})
}

func UpdateChannel(c *gin.Context) {
	id := c.Param("id")
	var channel NotifyChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知渠道不存在"})
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	// 如果密码是脱敏的则不更新
	if pass, ok := input["smtp_pass"]; ok && pass == "******" {
		delete(input, "smtp_pass")
	}

	if err := database.DB.Model(&channel).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channel})
}

func DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&NotifyChannel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func TestChannel(c *gin.Context) {
	id := c.Param("id")
	var channel NotifyChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知渠道不存在"})
		return
	}

	err := SendTestNotification(&channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("测试失败: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试消息发送成功"})
}

// ==================== 告警事件 ====================

func GetEvents(c *gin.Context) {
	var events []AlertEvent
	query := database.DB.Preload("Rule")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source LIKE ?", "%"+source+"%")
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query.Model(&AlertEvent{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("fired_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 格式化时间字段为字符串
	formattedEvents := make([]map[string]interface{}, len(events))
	for i, e := range events {
		formattedEvents[i] = formatEventTime(e)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      formattedEvents,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetEvent(c *gin.Context) {
	id := c.Param("id")
	var event AlertEvent

	if err := database.DB.Preload("Rule").Preload("Logs", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": formatEventTime(event)})
}

// formatEventTime 将 AlertEvent 的时间字段格式化为字符串（东八区）
func formatEventTime(event AlertEvent) map[string]interface{} {
	timeFmt := "2006-01-02 15:04:05"
	data := map[string]interface{}{
		"id":            event.ID,
		"rule_id":       event.RuleID,
		"rule_name":     event.RuleName,
		"severity":      event.Severity,
		"category":      event.Category,
		"content":       event.Content,
		"source":        event.Source,
		"status":        event.Status,
		"fired_at":      formatTimeInShanghai(event.FiredAt, timeFmt),
		"acked_at":      formatTimePtrInShanghai(event.AckedAt, timeFmt),
		"resolved_at":   formatTimePtrInShanghai(event.ResolvedAt, timeFmt),
		"closed_at":     formatTimePtrInShanghai(event.ClosedAt, timeFmt),
		"acked_by":      event.AckedBy,
		"closed_by":     event.ClosedBy,
		"handle_type":   event.HandleType,
		"handle_note":   event.HandleNote,
		"labels":        event.Labels,
		"fingerprint":   event.Fingerprint,
		"notify_status": event.NotifyStatus,
		"created_at":    formatTimeInShanghai(event.CreatedAt, timeFmt),
		"updated_at":    formatTimeInShanghai(event.UpdatedAt, timeFmt),
	}
	// 添加关联字段（检查 Rule 是否为 nil）
	if event.Rule != nil && event.Rule.ID != 0 {
		data["rule"] = event.Rule
	}
	if len(event.Logs) > 0 {
		logs := make([]map[string]interface{}, len(event.Logs))
		for i, log := range event.Logs {
			logs[i] = map[string]interface{}{
				"id":         log.ID,
				"event_id":   log.EventID,
				"action":     log.Action,
				"operator":   log.Operator,
				"content":    log.Content,
				"created_at": formatTimeInShanghai(log.CreatedAt, timeFmt),
			}
		}
		data["logs"] = logs
	}
	return data
}

// formatTimeInShanghai 将 time.Time 格式化为东八区字符串
func formatTimeInShanghai(t time.Time, fmt string) string {
	if t.IsZero() {
		return ""
	}
	if locShanghai != nil {
		t = t.In(locShanghai)
	}
	return t.Format(fmt)
}

// formatTimePtrInShanghai 将 *time.Time 格式化为东八区字符串
func formatTimePtrInShanghai(t *time.Time, fmt string) string {
	if t == nil || t.IsZero() {
		return ""
	}
	if locShanghai != nil {
		return t.In(locShanghai).Format(fmt)
	}
	return t.Format(fmt)
}

func toAlertEventPayload(event AlertEvent) platformevent.AlertEventPayload {
	return platformevent.AlertEventPayload{
		ID:           event.ID,
		RuleName:     event.RuleName,
		Severity:     event.Severity,
		Category:     event.Category,
		Content:      event.Content,
		Source:       event.Source,
		Status:       event.Status,
		FiredAt:      event.FiredAt,
		AckedAt:      event.AckedAt,
		ResolvedAt:   event.ResolvedAt,
		ClosedAt:     event.ClosedAt,
		AckedBy:      event.AckedBy,
		ClosedBy:     event.ClosedBy,
		HandleType:   event.HandleType,
		HandleNote:   event.HandleNote,
		Labels:       event.Labels,
		Fingerprint:  event.Fingerprint,
		NotifyStatus: event.NotifyStatus,
		CreatedAt:    event.CreatedAt,
		UpdatedAt:    event.UpdatedAt,
	}
}

func AckEvent(c *gin.Context) {
	id := c.Param("id")
	var event AlertEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	var input struct {
		HandleType string `json:"handle_type"` // ticket/auto/manual
		HandleNote string `json:"handle_note"`
	}
	c.ShouldBindJSON(&input)

	username := c.GetString("username")
	now := time.Now()
	before := buildAuditAlertEventRecord(event)

	database.DB.Model(&event).Updates(map[string]interface{}{
		"status":      "acknowledged",
		"acked_at":    &now,
		"acked_by":    username,
		"handle_type": input.HandleType,
		"handle_note": input.HandleNote,
	})

	// 记录日志
	note := "介入处理"
	if input.HandleType == "ticket" {
		note = "创建工单处理"
	} else if input.HandleType == "auto" {
		note = "自动化处理"
	} else if input.HandleNote != "" {
		note = input.HandleNote
	}

	database.DB.Create(&AlertEventLog{
		EventID:  event.ID,
		Action:   "acked",
		Operator: username,
		Content:  note,
	})

	event.Status = "acknowledged"
	event.AckedAt = &now
	event.AckedBy = username
	event.HandleType = input.HandleType
	event.HandleNote = input.HandleNote
	audit.SetOperationAuditBefore(c, before)
	audit.SetOperationAuditAfter(c, buildAuditAlertEventRecord(event))
	audit.SetOperationAuditSummary(c, "确认了告警。")
	_ = platformevent.RecordAlertEvent(toAlertEventPayload(event))

	c.JSON(http.StatusOK, gin.H{"message": "已确认处理"})
}

func CloseEvent(c *gin.Context) {
	id := c.Param("id")
	var event AlertEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	var input struct {
		Note string `json:"note"`
	}
	c.ShouldBindJSON(&input)

	username := c.GetString("username")
	now := time.Now()
	before := buildAuditAlertEventRecord(event)

	database.DB.Model(&event).Updates(map[string]interface{}{
		"status":    "closed",
		"closed_at": &now,
		"closed_by": username,
	})

	database.DB.Create(&AlertEventLog{
		EventID:  event.ID,
		Action:   "closed",
		Operator: username,
		Content:  input.Note,
	})

	event.Status = "closed"
	event.ClosedAt = &now
	event.ClosedBy = username
	audit.SetOperationAuditBefore(c, before)
	audit.SetOperationAuditAfter(c, buildAuditAlertEventRecord(event))
	audit.SetOperationAuditSummary(c, "关闭了告警。")
	_ = platformevent.RecordAlertEvent(toAlertEventPayload(event))

	c.JSON(http.StatusOK, gin.H{"message": "已关闭"})
}

// DeleteEvent 删除告警事件
func DeleteEvent(c *gin.Context) {
	id := c.Param("id")
	var event AlertEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	// 删除关联的日志
	database.DB.Where("event_id = ?", id).Delete(&AlertEventLog{})

	// 删除事件
	database.DB.Delete(&event)
	audit.SetOperationAuditBefore(c, buildAuditAlertEventRecord(event))
	audit.SetOperationAuditSummary(c, "删除了告警。")
	_ = platformevent.RecordAlertEventDeleted(toAlertEventPayload(event))

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func AddEventNote(c *gin.Context) {
	id := c.Param("id")
	var event AlertEvent
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "事件不存在"})
		return
	}

	var input struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入备注内容"})
		return
	}

	username := c.GetString("username")
	log := AlertEventLog{
		EventID:  event.ID,
		Action:   "note",
		Operator: username,
		Content:  input.Content,
	}
	database.DB.Create(&log)

	c.JSON(http.StatusOK, gin.H{"data": log})
}

func GetEventLogs(c *gin.Context) {
	id := c.Param("id")
	var logs []AlertEventLog
	if err := database.DB.Where("event_id = ?", id).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// GetEventStats 告警事件统计
func GetEventStats(c *gin.Context) {
	type StatResult struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusStats []StatResult
	database.DB.Model(&AlertEvent{}).Select("status, count(*) as count").Group("status").Find(&statusStats)

	type SeverityResult struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var severityStats []SeverityResult
	database.DB.Model(&AlertEvent{}).Select("severity, count(*) as count").Group("severity").Find(&severityStats)

	// 今日新增
	today := time.Now().Format("2006-01-02")
	var todayCount int64
	database.DB.Model(&AlertEvent{}).Where("DATE(fired_at) = ?", today).Count(&todayCount)

	// 待处理数
	var firingCount int64
	database.DB.Model(&AlertEvent{}).Where("status = ?", "firing").Count(&firingCount)

	c.JSON(http.StatusOK, gin.H{
		"status_stats":   statusStats,
		"severity_stats": severityStats,
		"today_count":    todayCount,
		"firing_count":   firingCount,
	})
}

// ==================== Alertmanager Webhook ====================

// AlertmanagerPayload Alertmanager 推送的数据结构
type AlertmanagerPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"` // firing / resolved
	Receiver          string            `json:"receiver"`
	Alerts            []AlertmanagerMsg `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
}

// AlertmanagerMsg Alertmanager 单条告警
type AlertmanagerMsg struct {
	Status       string            `json:"status"` // firing / resolved
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// AlertWebhook 接收 Alertmanager 的 Webhook 推送
// POST /api/alert/webhook（无需登录认证）
func AlertWebhook(c *gin.Context) {
	var payload AlertmanagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[alert webhook] 解析失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析请求数据"})
		return
	}

	log.Printf("[alert webhook] 收到推送 status=%s alerts=%d", payload.Status, len(payload.Alerts))

	created := 0
	resolved := 0

	for _, alert := range payload.Alerts {
		fingerprint := alert.Fingerprint
		alertName := alert.Labels["alertname"]
		instance := alert.Labels["instance"]
		severity := alert.Labels["severity"]
		summary := alert.Annotations["summary"]
		description := alert.Annotations["description"]

		// 调试日志：查看原始数据
		log.Printf("[alert webhook] 原始数据: alertname=%s, summary=%s, description=%s", alertName, summary, description)

		// 构建告警内容
		content := description
		if content == "" {
			content = summary
		}
		if content == "" {
			content = fmt.Sprintf("告警: %s, 实例: %s", alertName, instance)
		}

		// 调试日志：查看构建结果
		log.Printf("[alert webhook] 构建内容: %s", content)

		// 来源：优先用 instance 标签
		source := instance
		if source == "" {
			source = alert.Labels["job"]
		}

		if alert.Status == "firing" {
			// ===== 触发告警 =====

			// 去重：根据 fingerprint 检查是否已有未关闭的事件
			var existing AlertEvent
			err := database.DB.Where("fingerprint = ? AND status IN ?", fingerprint, []string{"firing", "acknowledged"}).First(&existing).Error
			if err == nil {
				log.Printf("[alert webhook] 告警已存在 fingerprint=%s 跳过重复", fingerprint)
				continue
			}

			// 匹配本地告警规则
			var matchedRule *AlertRule
			var rules []AlertRule
			database.DB.Where("enabled = ? AND deleted_at IS NULL", true).Find(&rules)
			for i, rule := range rules {
				// 通过规则名称匹配（Prometheus alertname = 规则名称）
				if rule.Name == alertName {
					matchedRule = &rules[i]
					break
				}
			}

			// 确定告警级别
			eventSeverity := "warning"
			eventCategory := guessCategory(alertName)
			var ruleID *uint
			var alertGroupID *uint

			if matchedRule != nil {
				eventSeverity = matchedRule.Severity
				eventCategory = matchedRule.Category
				ruleID = &matchedRule.ID
				alertGroupID = matchedRule.AlertGroupID

				// 更新规则的 Grafana 状态
				database.DB.Model(matchedRule).Update("grafana_state", "firing")
			}

			// 如果 Prometheus 标签有 severity，也参考
			if severity == "critical" || severity == "Disaster" {
				eventSeverity = "critical"
			}

			// 创建告警事件
			// 转换时区：Grafana 发送的时间可能不包含时区信息，需要确保使用上海时间
			firedAt := alert.StartsAt
			if locShanghai != nil {
				// 如果时间不包含时区信息，假定为上海时间并转换
				if firedAt.Location() != locShanghai {
					firedAt = firedAt.In(locShanghai)
				}
			}
			event := AlertEvent{
				RuleID:      ruleID,
				RuleName:    alertName,
				Severity:    eventSeverity,
				Category:    eventCategory,
				Content:     content,
				Source:      source,
				Status:      "firing",
				FiredAt:     firedAt,
				Labels:      labelsToJSON(alert.Labels),
				Fingerprint: fingerprint,
			}
			if err := database.DB.Create(&event).Error; err != nil {
				log.Printf("[alert webhook] 创建事件失败 alertname=%s: %v", alertName, err)
				continue
			}
			_ = platformevent.RecordAlertEvent(toAlertEventPayload(event))

			log.Printf("[alert webhook] 已创建事件 id=%d alertname=%s 开始发送通知", event.ID, alertName)

			// 记录事件日志
			database.DB.Create(&AlertEventLog{
				EventID:   event.ID,
				Action:    "created",
				Content:   fmt.Sprintf("告警触发: %s [%s] - %s", alertName, eventSeverity, source),
				CreatedAt: time.Now(),
			})

			// ===== 发送通知 =====
			go sendAlertNotifications(event, alertGroupID)

			created++

		} else if alert.Status == "resolved" {
			// ===== 告警恢复 =====

			var event AlertEvent
			err := database.DB.Where("fingerprint = ? AND status IN ?", fingerprint, []string{"firing", "acknowledged"}).First(&event).Error
			if err != nil {
				continue // 找不到对应的事件
			}

			now := time.Now()
			database.DB.Model(&event).Updates(map[string]interface{}{
				"status":      "resolved",
				"resolved_at": &now,
			})
			event.Status = "resolved"
			event.ResolvedAt = &now
			_ = platformevent.RecordAlertEvent(toAlertEventPayload(event))

			// 记录恢复日志
			database.DB.Create(&AlertEventLog{
				EventID:   event.ID,
				Action:    "resolved",
				Content:   fmt.Sprintf("告警恢复: %s - %s", alertName, source),
				CreatedAt: now,
			})

			// 更新规则状态
			if event.RuleID != nil {
				database.DB.Model(&AlertRule{}).Where("id = ?", *event.RuleID).Update("grafana_state", "inactive")
			}

			// 发送恢复通知
			go sendResolvedNotifications(event)

			resolved++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("处理完成: 新增 %d 条告警, 恢复 %d 条", created, resolved),
		"created":  created,
		"resolved": resolved,
	})
}

// labelsToJSON 将标签 map 转为 JSON 字符串
func labelsToJSON(labels map[string]string) string {
	data, err := json.Marshal(labels)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// sendAlertNotifications 发送告警通知（异步）
func sendAlertNotifications(event AlertEvent, alertGroupID *uint) {
	channels := getNotifyChannels(alertGroupID)
	if len(channels) == 0 {
		log.Printf("[alert notify] 无可用通知渠道 event_id=%d rule=%s 请到报警中心-通知渠道添加并启用", event.ID, event.RuleName)
		return
	}

	log.Printf("[alert notify] event_id=%d 开始发送到 %d 个渠道", event.ID, len(channels))

	msg := &NotifyMessage{
		Title:    fmt.Sprintf("【告警】%s", event.RuleName),
		RuleName: event.RuleName,
		Content:  event.Content,
		Severity: event.Severity,
		Status:   event.Status,
		Source:   event.Source,
		Category: event.Category,
		Time:     FormatTimeLocal(event.FiredAt),
	}

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		err := SendNotification(&channel, msg)
		status := "sent"
		logContent := fmt.Sprintf("通知已发送至 [%s] %s", channel.Type, channel.Name)
		if err != nil {
			status = "failed"
			logContent = fmt.Sprintf("通知发送失败 [%s] %s: %v", channel.Type, channel.Name, err)
			log.Printf("[alert notify] 发送失败 event_id=%d channel=%s %s: %v", event.ID, channel.Name, channel.Type, err)
		}

		// 记录通知日志
		database.DB.Create(&AlertEventLog{
			EventID:   event.ID,
			Action:    "notified",
			Content:   logContent,
			CreatedAt: time.Now(),
		})

		// 更新事件通知状态
		database.DB.Model(&event).Update("notify_status", status)
	}
}

// sendResolvedNotifications 发送恢复通知（异步）
func sendResolvedNotifications(event AlertEvent) {
	// 查询当前指标值
	currentValue := ""
	if event.Labels != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(event.Labels), &labels); err == nil {
			log.Printf("[alert resolved] 解析 Labels 成功: %v", labels)
			currentValue = queryCurrentValue(labels, event.RuleName)
			log.Printf("[alert resolved] 查询结果 currentValue: %s", currentValue)
		} else {
			log.Printf("[alert resolved] 解析 Labels 失败: %v", err)
		}
	} else {
		log.Printf("[alert resolved] Labels 为空")
	}

	// 构建恢复内容
	content := fmt.Sprintf("告警已恢复 - %s", event.Source)
	if currentValue != "" {
		content = fmt.Sprintf("告警已恢复 - %s（当前值: %s）", event.Source, currentValue)
	}

	log.Printf("[alert resolved] 最终恢复内容: %s", content)

	msg := &NotifyMessage{
		Title:        fmt.Sprintf("【恢复】%s", event.RuleName),
		RuleName:     event.RuleName,
		Content:      content,
		CurrentValue: currentValue,
		Severity:     "info",
		Status:       "resolved",
		Source:       event.Source,
		Category:     event.Category,
		Time:         FormatTimeLocal(time.Now()),
	}

	var ruleGroupID *uint
	if event.RuleID != nil {
		var rule AlertRule
		if err := database.DB.First(&rule, *event.RuleID).Error; err == nil {
			ruleGroupID = rule.AlertGroupID
		}
	}

	channels := getNotifyChannels(ruleGroupID)
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		_ = SendNotification(&channel, msg)
	}
}

// getNotifyChannels 获取通知渠道
// 优先使用报警组关联的渠道，如果没有配置报警组则使用所有启用的渠道
func getNotifyChannels(alertGroupID *uint) []NotifyChannel {
	var channels []NotifyChannel

	// 如果有关联报警组，查找组配置的渠道（目前简化：使用所有启用的渠道）
	// TODO: 未来可以给报警组单独配置通知渠道
	database.DB.Where("enabled = ? AND deleted_at IS NULL", true).Find(&channels)

	return channels
}

// ==================== 告警模板管理 ====================

// TemplateData 模板渲染的数据上下文
// 前端模板编辑器中可使用的所有变量
type TemplateData struct {
	RuleName      string // 规则名称
	Content       string // 告警内容
	CurrentValue  string // 当前指标值（恢复告警时）
	Source        string // 来源 (IP:Port)
	Severity      string // 级别英文 (critical/warning/info)
	SeverityLabel string // 级别中文 (严重/警告/提醒)
	Status        string // 状态英文 (firing/resolved)
	StatusLabel   string // 状态中文 (告警中/已恢复)
	Category      string // 分类英文 (disk/memory/cpu等)
	CategoryLabel string // 分类中文 (磁盘/内存/CPU等)
	Time          string // 触发/恢复时间
	Emoji         string // 级别Emoji (🔴🟡🔵✅)
}

// buildTemplateData 从 NotifyMessage 构建模板数据
func buildTemplateData(msg *NotifyMessage) TemplateData {
	return TemplateData{
		RuleName:      msg.RuleName,
		Content:       msg.Content,
		CurrentValue:  msg.CurrentValue,
		Source:        msg.Source,
		Severity:      msg.Severity,
		SeverityLabel: getSeverityLabel(msg.Severity),
		Status:        msg.Status,
		StatusLabel:   getStatusLabel(msg.Status),
		Category:      msg.Category,
		CategoryLabel: getCategoryLabel(msg.Category),
		Time:          msg.Time,
		Emoji:         getSeverityEmoji(msg.Severity, msg.Status),
	}
}

// renderTemplate 渲染模板字符串
func renderTemplate(tplStr string, data TemplateData) (string, error) {
	t, err := template.New("alert").Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("模板语法错误: %v", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("模板渲染失败: %v", err)
	}
	return buf.String(), nil
}

// getTemplateForChannel 获取渠道对应场景的模板
// 按优先级查找：1. 该类型+场景的已启用模板（优先默认模板）; 2. 回退到硬编码
func getTemplateForChannel(channelType, scene string) *AlertTemplate {
	var tpl AlertTemplate
	// 先找默认模板
	err := database.DB.Where("type = ? AND scene = ? AND is_default = ? AND enabled = ? AND deleted_at IS NULL",
		channelType, scene, true, true).First(&tpl).Error
	if err == nil {
		return &tpl
	}
	// 再找任意已启用的模板
	err = database.DB.Where("type = ? AND scene = ? AND enabled = ? AND deleted_at IS NULL",
		channelType, scene, true).First(&tpl).Error
	if err == nil {
		return &tpl
	}
	return nil
}

// GetTemplates 获取所有模板
func GetTemplates(c *gin.Context) {
	var templates []AlertTemplate
	query := database.DB.Where("deleted_at IS NULL")

	if t := c.Query("type"); t != "" {
		query = query.Where("type = ?", t)
	}
	if scene := c.Query("scene"); scene != "" {
		query = query.Where("scene = ?", scene)
	}

	if err := query.Order("type, scene, is_default DESC, created_at DESC").Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// CreateTemplate 创建模板
func CreateTemplate(c *gin.Context) {
	var tpl AlertTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证模板语法
	if _, err := renderTemplate(tpl.ContentTpl, buildSampleData()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&tpl).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tpl})
}

// UpdateTemplate 更新模板
func UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var tpl AlertTemplate
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&tpl).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	var input AlertTemplate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证模板语法
	if _, err := renderTemplate(input.ContentTpl, buildSampleData()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&tpl).Updates(map[string]interface{}{
		"name":        input.Name,
		"type":        input.Type,
		"scene":       input.Scene,
		"title_tpl":   input.TitleTpl,
		"content_tpl": input.ContentTpl,
		"enabled":     input.Enabled,
		"description": input.Description,
	})

	c.JSON(http.StatusOK, gin.H{"data": tpl})
}

// DeleteTemplate 删除模板（软删除）
func DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	result := database.DB.Model(&AlertTemplate{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", &now)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// SetDefaultTemplate 设置默认模板
func SetDefaultTemplate(c *gin.Context) {
	id := c.Param("id")
	var tpl AlertTemplate
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&tpl).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	// 先取消同类型同场景的其他默认模板
	database.DB.Model(&AlertTemplate{}).Where("type = ? AND scene = ? AND is_default = ? AND deleted_at IS NULL",
		tpl.Type, tpl.Scene, true).Update("is_default", false)

	// 设置当前为默认
	database.DB.Model(&tpl).Update("is_default", true)

	c.JSON(http.StatusOK, gin.H{"message": "已设为默认模板"})
}

// PreviewTemplate 预览模板渲染结果
func PreviewTemplate(c *gin.Context) {
	var input struct {
		TitleTpl   string `json:"title_tpl"`
		ContentTpl string `json:"content_tpl"`
		Type       string `json:"type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	data := buildSampleData()

	titleResult := ""
	if input.TitleTpl != "" {
		var err error
		titleResult, err = renderTemplate(input.TitleTpl, data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "标题模板错误: " + err.Error()})
			return
		}
	}

	contentResult, err := renderTemplate(input.ContentTpl, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容模板错误: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"title":   titleResult,
		"content": contentResult,
		"data":    data,
	})
}

// buildSampleData 构建示例数据用于预览
func buildSampleData() TemplateData {
	return TemplateData{
		RuleName:      "DiskUsageHigh",
		Content:       "/dev/sda1 磁盘使用率达到 95%，请及时清理",
		Source:        "node-exporter:9100",
		Severity:      "critical",
		SeverityLabel: "严重",
		Status:        "firing",
		StatusLabel:   "告警中",
		Category:      "disk",
		CategoryLabel: "磁盘",
		Time:          FormatTimeLocal(time.Now()),
		Emoji:         "🔴",
	}
}
