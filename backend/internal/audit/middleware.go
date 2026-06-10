// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/gin-gonic/gin"
)

const (
	auditBeforeDataKey    = "audit_before_data"
	auditAfterDataKey     = "audit_after_data"
	auditChangeSummaryKey = "audit_change_summary"
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

const accessLogDedupWindow = 15 * time.Second

var accessLogTracker = struct {
	sync.Mutex
	lastSeen map[string]time.Time
}{
	lastSeen: make(map[string]time.Time),
}

type auditLogSettingSnapshot struct {
	access    bool
	operation bool
	login     bool
	loadedAt  time.Time
}

var auditLogSettingCache = struct {
	sync.RWMutex
	value auditLogSettingSnapshot
}{}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	_, _ = w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api/health" {
			c.Next()
			return
		}

		traceID := buildTraceID()
		c.Set("trace_id", traceID)
		start := time.Now()
		requestBody := readRequestBody(c)
		writer := &bodyCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = writer

		c.Next()

		duration := time.Since(start).Milliseconds()
		ctx := buildRequestContext(c, traceID, requestBody, writer.body.String(), duration)
		go persistAuditLogs(ctx)
	}
}

type requestContext struct {
	TraceID         string
	UserID          uint
	Username        string
	RealName        string
	Role            string
	PagePathHint    string
	RequestPath     string
	RequestMethod   string
	RequestIP       string
	UserAgent       string
	Referer         string
	StatusCode      int
	OperationStatus string
	DurationMS      int64
	ErrorMessage    string
	RequestBody     string
	ResponseBody    string
	BeforeData      string
	AfterData       string
	ChangeSummary   string
	OccurredAt      time.Time
}

func buildRequestContext(c *gin.Context, traceID, requestBody, responseBody string, duration int64) requestContext {
	statusCode := c.Writer.Status()
	operationStatus := "success"
	if statusCode >= http.StatusBadRequest {
		operationStatus = "failed"
	}

	return requestContext{
		TraceID:         traceID,
		UserID:          c.GetUint("user_id"),
		Username:        c.GetString("username"),
		RealName:        c.GetString("real_name"),
		Role:            c.GetString("role"),
		PagePathHint:    strings.TrimSpace(c.GetHeader("X-Page-Path")),
		RequestPath:     c.FullPath(),
		RequestMethod:   c.Request.Method,
		RequestIP:       ExtractRequestIP(c),
		UserAgent:       c.Request.UserAgent(),
		Referer:         c.Request.Referer(),
		StatusCode:      statusCode,
		OperationStatus: operationStatus,
		DurationMS:      duration,
		ErrorMessage:    extractErrorMessage(responseBody, statusCode),
		RequestBody:     requestBody,
		ResponseBody:    responseBody,
		BeforeData:      marshalAuditContextValue(c, auditBeforeDataKey),
		AfterData:       marshalAuditContextValue(c, auditAfterDataKey),
		ChangeSummary:   strings.TrimSpace(c.GetString(auditChangeSummaryKey)),
		OccurredAt:      time.Now(),
	}
}

func ExtractRequestIP(c *gin.Context) string {
	if c == nil {
		return ""
	}
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(c.GetHeader(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			parts := strings.Split(value, ",")
			for _, part := range parts {
				candidate := strings.TrimSpace(part)
				if isValidIP(candidate) {
					return candidate
				}
			}
			continue
		}
		if isValidIP(value) {
			return value
		}
	}
	return c.ClientIP()
}

func isValidIP(value string) bool {
	return net.ParseIP(strings.TrimSpace(value)) != nil
}

func persistAuditLogs(ctx requestContext) {
	settings := getAuditLogSettingSnapshot()
	switch {
	case isLoginRequest(ctx):
		if settings.login {
			database.DB.Create(buildLoginLog(ctx))
		}
	case isAccessRequest(ctx):
		if settings.access {
			database.DB.Create(buildAccessLog(ctx))
		}
	case isOperationRequest(ctx):
		if settings.operation {
			database.DB.Create(buildOperationLog(ctx))
		}
	}
}

func getAuditLogSettingSnapshot() auditLogSettingSnapshot {
	auditLogSettingCache.RLock()
	cached := auditLogSettingCache.value
	auditLogSettingCache.RUnlock()
	if !cached.loadedAt.IsZero() && time.Since(cached.loadedAt) < 10*time.Second {
		return cached
	}

	snapshot := auditLogSettingSnapshot{
		access:    true,
		operation: true,
		login:     true,
		loadedAt:  time.Now(),
	}
	if database.DB != nil {
		var setting models.AuditLogSetting
		query := database.DB.Order("id ASC").Limit(1).Find(&setting)
		if query.Error == nil && query.RowsAffected > 0 {
			snapshot.access = setting.AccessLogEnabled
			snapshot.operation = setting.OperationLogEnabled
			snapshot.login = setting.LoginLogEnabled
		}
	}

	auditLogSettingCache.Lock()
	auditLogSettingCache.value = snapshot
	auditLogSettingCache.Unlock()
	return snapshot
}

func buildAccessLog(ctx requestContext) *models.PlatformAccessLog {
	meta := resolveRequestMeta(ctx.RequestPath, ctx.RequestMethod, ctx.PagePathHint)
	return &models.PlatformAccessLog{
		TraceID:         ctx.TraceID,
		UserID:          ctx.UserID,
		Username:        ctx.Username,
		RealName:        ctx.RealName,
		Role:            ctx.Role,
		MenuKey:         meta.MenuKey,
		MenuTitle:       meta.MenuTitle,
		PagePath:        meta.PagePath,
		RequestPath:     ctx.RequestPath,
		RequestMethod:   ctx.RequestMethod,
		RequestIP:       ctx.RequestIP,
		UserAgent:       truncateString(ctx.UserAgent, 512),
		Referer:         truncateString(ctx.Referer, 512),
		StatusCode:      ctx.StatusCode,
		OperationStatus: ctx.OperationStatus,
		DurationMS:      ctx.DurationMS,
		ErrorMessage:    ctx.ErrorMessage,
		AccessedAt:      ctx.OccurredAt,
	}
}

func buildOperationLog(ctx requestContext) *models.PlatformAuditLog {
	meta := resolveRequestMeta(ctx.RequestPath, ctx.RequestMethod, ctx.PagePathHint)
	return &models.PlatformAuditLog{
		TraceID:         ctx.TraceID,
		UserID:          ctx.UserID,
		Username:        ctx.Username,
		RealName:        ctx.RealName,
		Role:            ctx.Role,
		Module:          meta.Module,
		ResourceType:    meta.ResourceType,
		ResourceID:      meta.ResourceID,
		ResourceName:    meta.ResourceName,
		Action:          meta.Action,
		ActionLabel:     meta.ActionLabel,
		RequestPath:     ctx.RequestPath,
		RequestMethod:   ctx.RequestMethod,
		RequestIP:       ctx.RequestIP,
		StatusCode:      ctx.StatusCode,
		OperationStatus: ctx.OperationStatus,
		RequestParams:   ctx.RequestBody,
		BeforeData:      ctx.BeforeData,
		AfterData:       ctx.AfterData,
		ChangeSummary:   firstNonEmpty(ctx.ChangeSummary, meta.ChangeSummary),
		ErrorMessage:    ctx.ErrorMessage,
		DurationMS:      ctx.DurationMS,
		OperatedAt:      ctx.OccurredAt,
	}
}

func SetOperationAuditBefore(c *gin.Context, value any) {
	if c != nil {
		c.Set(auditBeforeDataKey, value)
	}
}

func SetOperationAuditAfter(c *gin.Context, value any) {
	if c != nil {
		c.Set(auditAfterDataKey, value)
	}
}

func SetOperationAuditSummary(c *gin.Context, summary string) {
	if c != nil {
		c.Set(auditChangeSummaryKey, strings.TrimSpace(summary))
	}
}

func marshalAuditContextValue(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(key)
	if !ok || value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func buildLoginLog(ctx requestContext) *models.PlatformLoginLog {
	username := ctx.Username
	if username == "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(ctx.RequestBody), &payload); err == nil {
			if value, ok := payload["username"].(string); ok {
				username = value
			}
		}
	}
	return &models.PlatformLoginLog{
		TraceID:         ctx.TraceID,
		UserID:          ctx.UserID,
		Username:        username,
		RealName:        ctx.RealName,
		Role:            ctx.Role,
		RequestIP:       ctx.RequestIP,
		UserAgent:       truncateString(ctx.UserAgent, 512),
		RequestPath:     ctx.RequestPath,
		RequestMethod:   ctx.RequestMethod,
		StatusCode:      ctx.StatusCode,
		OperationStatus: ctx.OperationStatus,
		LoginType:       "password",
		ErrorMessage:    ctx.ErrorMessage,
		DurationMS:      ctx.DurationMS,
		LoggedInAt:      ctx.OccurredAt,
	}
}

type requestMeta struct {
	MenuKey       string
	MenuTitle     string
	PagePath      string
	Module        string
	ResourceType  string
	ResourceID    string
	ResourceName  string
	Action        string
	ActionLabel   string
	ChangeSummary string
}

func resolveRequestMeta(path, method, pagePathHint string) requestMeta {
	meta := requestMeta{
		MenuKey:      "platform-api",
		MenuTitle:    "接口访问",
		PagePath:     firstNonEmptyPagePath(pagePathHint, path),
		Module:       "平台通用",
		ResourceType: "api",
		ResourceName: deriveGenericResourceName(path),
		Action:       inferAction(method, path),
		ActionLabel:  inferActionLabel(method, path),
	}

	if pageMeta, ok := resolveMenuMetaByPage(pagePathHint); ok {
		meta.MenuKey = pageMeta.MenuKey
		meta.MenuTitle = pageMeta.MenuTitle
		meta.PagePath = pageMeta.PagePath
		meta.Module = pageMeta.Module
		if meta.ResourceType == "api" {
			meta.ResourceType = pageMeta.ResourceType
		}
		if meta.ResourceName == "" || meta.ResourceName == "接口访问" {
			meta.ResourceName = pageMeta.ResourceName
		}
	}

	switch {
	case strings.HasPrefix(path, "/api/security/fim/"):
		meta.MenuKey = "security-fim"
		meta.MenuTitle = "文件完整性巡检"
		meta.PagePath = "/security/fim/policies"
		meta.Module = "安全中心"
		meta.ResourceType = "fim"
		meta.ResourceName = resolveFIMResourceName(path)
	case strings.HasPrefix(path, "/api/admin/settings"):
		meta.MenuKey = "admin-settings"
		meta.MenuTitle = "系统设置"
		meta.PagePath = "/admin/settings"
		meta.Module = "系统管理"
		meta.ResourceType = "settings"
		meta.ResourceName = "系统设置"
	case strings.HasPrefix(path, "/api/admin/audit"):
		meta.MenuKey = "platform-audit"
		meta.MenuTitle = "平台审计"
		meta.PagePath = "/platform/audit"
		meta.Module = "平台审计"
		meta.ResourceType = "audit"
		meta.ResourceName = "平台审计"
	case strings.HasPrefix(path, "/api/alert/"):
		meta.MenuKey = "alarm-center"
		meta.MenuTitle = "告警中心"
		meta.PagePath = "/alarm/events"
		meta.Module = "告警中心"
		meta.ResourceType = "alert"
		meta.ResourceName = "告警中心"
	case strings.HasPrefix(path, "/api/auth/login"):
		meta.MenuKey = "auth-login"
		meta.MenuTitle = "登录日志"
		meta.PagePath = "/login"
		meta.Module = "认证中心"
		meta.ResourceType = "login"
		meta.ResourceName = "后台管理平台"
	case strings.HasPrefix(path, "/api/user/menus"):
		meta.MenuKey = "platform-audit"
		meta.MenuTitle = "平台审计"
		meta.PagePath = "/platform/audit"
		meta.Module = "平台审计"
		meta.ResourceType = "menu_access"
		meta.ResourceName = "菜单访问"
	case strings.HasPrefix(path, "/api/user/"):
		meta.MenuKey = "platform-user"
		meta.MenuTitle = "用户中心"
		meta.PagePath = "/profile"
		meta.Module = "用户中心"
		meta.ResourceType = "user"
		meta.ResourceName = deriveGenericResourceName(path)
	case strings.HasPrefix(path, "/api/auth/"):
		meta.MenuKey = "auth-center"
		meta.MenuTitle = "认证中心"
		meta.PagePath = "/login"
		meta.Module = "认证中心"
		meta.ResourceType = "auth"
		meta.ResourceName = deriveGenericResourceName(path)
	case strings.HasPrefix(path, "/api/admin/"):
		meta.MenuKey = "system-admin"
		meta.MenuTitle = "系统管理"
		meta.PagePath = "/admin/settings"
		meta.Module = "系统管理"
		meta.ResourceType = "admin"
		meta.ResourceName = deriveGenericResourceName(path)
	}

	meta.ResourceID = resolveResourceID(path)
	meta.ChangeSummary = buildChangeSummary(meta, path)
	return meta
}

func firstNonEmptyPagePath(pagePathHint, path string) string {
	if pagePathHint != "" {
		return pagePathHint
	}
	return path
}

func resolveMenuMetaByPage(pagePath string) (requestMeta, bool) {
	switch {
	case pagePath == "/" || pagePath == "":
		return requestMeta{MenuKey: "dashboard", MenuTitle: "工作台", PagePath: "/", Module: "工作台", ResourceType: "dashboard", ResourceName: "工作台"}, true
	case strings.HasPrefix(pagePath, "/cmdb/projects"):
		return requestMeta{MenuKey: "cmdb-projects", MenuTitle: "项目管理", PagePath: "/cmdb/projects", Module: "资产中心", ResourceType: "project", ResourceName: "项目管理"}, true
	case strings.HasPrefix(pagePath, "/cmdb/environments"):
		return requestMeta{MenuKey: "cmdb-environments", MenuTitle: "环境管理", PagePath: "/cmdb/environments", Module: "资产中心", ResourceType: "environment", ResourceName: "环境管理"}, true
	case strings.HasPrefix(pagePath, "/cmdb/servers"):
		return requestMeta{MenuKey: "cmdb-servers", MenuTitle: "主机管理", PagePath: "/cmdb/servers", Module: "资产中心", ResourceType: "server", ResourceName: "主机管理"}, true
	case strings.HasPrefix(pagePath, "/cmdb/applications"):
		return requestMeta{MenuKey: "cmdb-applications", MenuTitle: "应用流水线管理", PagePath: "/cmdb/applications", Module: "资产中心", ResourceType: "application", ResourceName: "应用流水线管理"}, true
	case strings.HasPrefix(pagePath, "/deploy/release"):
		return requestMeta{MenuKey: "deploy-release", MenuTitle: "迭代部署", PagePath: "/deploy/release", Module: "变更发布", ResourceType: "deploy", ResourceName: "迭代部署"}, true
	case strings.HasPrefix(pagePath, "/deploy/history"):
		return requestMeta{MenuKey: "deploy-history", MenuTitle: "部署记录", PagePath: "/deploy/history", Module: "变更发布", ResourceType: "deploy_record", ResourceName: "部署记录"}, true
	case strings.HasPrefix(pagePath, "/deploy/archived"):
		return requestMeta{MenuKey: "deploy-archived", MenuTitle: "归档历史", PagePath: "/deploy/archived", Module: "变更发布", ResourceType: "archive_record", ResourceName: "归档历史"}, true
	case strings.HasPrefix(pagePath, "/deploy/archive"):
		return requestMeta{MenuKey: "deploy-archive", MenuTitle: "归档打包", PagePath: "/deploy/archive", Module: "变更发布", ResourceType: "archive", ResourceName: "归档打包"}, true
	case strings.HasPrefix(pagePath, "/deploy/aggregate-package"):
		return requestMeta{MenuKey: "deploy-aggregate-package", MenuTitle: "聚合打包", PagePath: "/deploy/aggregate-package", Module: "变更发布", ResourceType: "aggregate_package", ResourceName: "聚合打包"}, true
	case strings.HasPrefix(pagePath, "/deploy/aggregated-history"):
		return requestMeta{MenuKey: "aggregated-history", MenuTitle: "聚合历史", PagePath: "/deploy/aggregated-history", Module: "变更发布", ResourceType: "aggregate_history", ResourceName: "聚合历史"}, true
	case strings.HasPrefix(pagePath, "/consul/config"):
		return requestMeta{MenuKey: "consul-config", MenuTitle: "配置管理", PagePath: "/consul/config", Module: "变更发布", ResourceType: "consul_config", ResourceName: "配置管理"}, true
	case strings.HasPrefix(pagePath, "/consul/batch-all"):
		return requestMeta{MenuKey: "consul-batch-all", MenuTitle: "批量配置下发", PagePath: "/consul/batch-all", Module: "变更发布", ResourceType: "consul_batch", ResourceName: "批量配置下发"}, true
	case strings.HasPrefix(pagePath, "/consul/operations"):
		return requestMeta{MenuKey: "consul-operations", MenuTitle: "配置操作记录", PagePath: "/consul/operations", Module: "变更发布", ResourceType: "consul_operation", ResourceName: "配置操作记录"}, true
	case strings.HasPrefix(pagePath, "/jenkins/views"):
		return requestMeta{MenuKey: "jenkins-views", MenuTitle: "视图管理", PagePath: "/jenkins/views", Module: "变更发布", ResourceType: "jenkins_view", ResourceName: "视图管理"}, true
	case strings.HasPrefix(pagePath, "/monitor/bigscreen") || pagePath == "/monitor":
		return requestMeta{MenuKey: "monitor-bigscreen", MenuTitle: "监控大屏", PagePath: "/monitor/bigscreen", Module: "监控中心", ResourceType: "monitor", ResourceName: "监控大屏"}, true
	case strings.HasPrefix(pagePath, "/monitor/overview"):
		return requestMeta{MenuKey: "monitor-overview", MenuTitle: "监控概览", PagePath: "/monitor/overview", Module: "监控中心", ResourceType: "monitor", ResourceName: "监控概览"}, true
	case strings.HasPrefix(pagePath, "/monitor/dashboards"):
		return requestMeta{MenuKey: "monitor-dashboards", MenuTitle: "Grafana仪表盘", PagePath: "/monitor/dashboards", Module: "监控中心", ResourceType: "grafana", ResourceName: "Grafana仪表盘"}, true
	case strings.HasPrefix(pagePath, "/alarm/events") || pagePath == "/alarm":
		return requestMeta{MenuKey: "alarm-events", MenuTitle: "告警事件", PagePath: "/alarm/events", Module: "告警中心", ResourceType: "alert_event", ResourceName: "告警事件"}, true
	case strings.HasPrefix(pagePath, "/alarm/rules"):
		return requestMeta{MenuKey: "alarm-rules", MenuTitle: "告警规则", PagePath: "/alarm/rules", Module: "告警中心", ResourceType: "alert_rule", ResourceName: "告警规则"}, true
	case strings.HasPrefix(pagePath, "/alarm/contacts"):
		return requestMeta{MenuKey: "alarm-contacts", MenuTitle: "联系人管理", PagePath: "/alarm/contacts", Module: "告警中心", ResourceType: "alert_contact", ResourceName: "联系人管理"}, true
	case strings.HasPrefix(pagePath, "/alarm/channels"):
		return requestMeta{MenuKey: "alarm-channels", MenuTitle: "通知渠道", PagePath: "/alarm/channels", Module: "告警中心", ResourceType: "notify_channel", ResourceName: "通知渠道"}, true
	case strings.HasPrefix(pagePath, "/alarm/templates"):
		return requestMeta{MenuKey: "alarm-templates", MenuTitle: "通知模板", PagePath: "/alarm/templates", Module: "告警中心", ResourceType: "notify_template", ResourceName: "通知模板"}, true
	case strings.HasPrefix(pagePath, "/security/overview") || pagePath == "/security":
		return requestMeta{MenuKey: "security-overview", MenuTitle: "安全概览", PagePath: "/security/overview", Module: "安全中心", ResourceType: "security_overview", ResourceName: "安全概览"}, true
	case strings.HasPrefix(pagePath, "/security/fim/policies"):
		return requestMeta{MenuKey: "security-fim-policies", MenuTitle: "巡检策略", PagePath: "/security/fim/policies", Module: "安全中心", ResourceType: "fim_policy", ResourceName: "巡检策略"}, true
	case strings.HasPrefix(pagePath, "/security/fim/executions"):
		return requestMeta{MenuKey: "security-fim-executions", MenuTitle: "执行记录", PagePath: "/security/fim/executions", Module: "安全中心", ResourceType: "fim_snapshot", ResourceName: "执行记录"}, true
	case strings.HasPrefix(pagePath, "/security/fim/events"):
		return requestMeta{MenuKey: "security-fim-events", MenuTitle: "文件变更事件", PagePath: "/security/fim/events", Module: "安全中心", ResourceType: "fim_event", ResourceName: "文件变更事件"}, true
	case strings.HasPrefix(pagePath, "/security/fim/alerts"):
		return requestMeta{MenuKey: "security-fim-alerts", MenuTitle: "完整性告警", PagePath: "/security/fim/alerts", Module: "安全中心", ResourceType: "fim_alert", ResourceName: "完整性告警"}, true
	case strings.HasPrefix(pagePath, "/security/tasks"):
		return requestMeta{MenuKey: "security-tasks", MenuTitle: "扫描任务", PagePath: "/security/tasks", Module: "安全中心", ResourceType: "security_task", ResourceName: "扫描任务"}, true
	case strings.HasPrefix(pagePath, "/security/assets"):
		return requestMeta{MenuKey: "security-assets", MenuTitle: "安全资产", PagePath: "/security/assets", Module: "安全中心", ResourceType: "security_asset", ResourceName: "安全资产"}, true
	case strings.HasPrefix(pagePath, "/security/vulnerabilities"):
		return requestMeta{MenuKey: "security-vulnerabilities", MenuTitle: "漏洞管理", PagePath: "/security/vulnerabilities", Module: "安全中心", ResourceType: "security_vulnerability", ResourceName: "漏洞管理"}, true
	case strings.HasPrefix(pagePath, "/security/tickets"):
		return requestMeta{MenuKey: "security-tickets", MenuTitle: "漏洞工单", PagePath: "/security/tickets", Module: "安全中心", ResourceType: "security_ticket", ResourceName: "漏洞工单"}, true
	case strings.HasPrefix(pagePath, "/security/vuln-db"):
		return requestMeta{MenuKey: "security-vuln-db", MenuTitle: "漏洞知识库", PagePath: "/security/vuln-db", Module: "安全中心", ResourceType: "security_vuln_db", ResourceName: "漏洞知识库"}, true
	case strings.HasPrefix(pagePath, "/admin/users"):
		return requestMeta{MenuKey: "admin-users", MenuTitle: "用户管理", PagePath: "/admin/users", Module: "系统管理", ResourceType: "admin_user", ResourceName: "用户管理"}, true
	case strings.HasPrefix(pagePath, "/admin/roles"):
		return requestMeta{MenuKey: "admin-roles", MenuTitle: "角色管理", PagePath: "/admin/roles", Module: "系统管理", ResourceType: "admin_role", ResourceName: "角色管理"}, true
	case strings.HasPrefix(pagePath, "/admin/menus"):
		return requestMeta{MenuKey: "admin-menus", MenuTitle: "菜单管理", PagePath: "/admin/menus", Module: "系统管理", ResourceType: "admin_menu", ResourceName: "菜单管理"}, true
	case strings.HasPrefix(pagePath, "/admin/settings"):
		return requestMeta{MenuKey: "admin-settings", MenuTitle: "系统设置", PagePath: "/admin/settings", Module: "系统管理", ResourceType: "settings", ResourceName: "系统设置"}, true
	case strings.HasPrefix(pagePath, "/platform/audit"):
		return requestMeta{MenuKey: "platform-audit", MenuTitle: "平台审计", PagePath: "/platform/audit", Module: "平台审计", ResourceType: "audit", ResourceName: "平台审计"}, true
	case strings.HasPrefix(pagePath, "/platform/events"):
		return requestMeta{MenuKey: "platform-events", MenuTitle: "平台事件中心", PagePath: "/platform/events", Module: "平台事件", ResourceType: "platform_event", ResourceName: "平台事件中心"}, true
	case strings.HasPrefix(pagePath, "/profile"):
		return requestMeta{MenuKey: "profile", MenuTitle: "我的资料", PagePath: "/profile", Module: "用户中心", ResourceType: "profile", ResourceName: "我的资料"}, true
	case strings.HasPrefix(pagePath, "/user-manual"):
		return requestMeta{MenuKey: "user-manual", MenuTitle: "用户手册", PagePath: "/user-manual", Module: "平台通用", ResourceType: "manual", ResourceName: "用户手册"}, true
	default:
		return requestMeta{}, false
	}
}

func inferAction(method, path string) string {
	switch {
	case path == "/api/auth/login":
		return "login"
	case strings.Contains(path, "/export"):
		return "export"
	case strings.Contains(path, "/archive"):
		return "archive"
	case strings.Contains(path, "/cleanup"):
		return "cleanup"
	case strings.Contains(path, "/test"):
		return "execute"
	case strings.Contains(path, "/ack"):
		return "ack"
	case strings.Contains(path, "/resolve"):
		return "resolve"
	case strings.Contains(path, "/close"):
		return "close"
	case strings.Contains(path, "/enable"):
		return "enable"
	case strings.Contains(path, "/disable"):
		return "disable"
	case strings.Contains(path, "/scan") || strings.Contains(path, "/build"):
		return "execute"
	case method == http.MethodPost:
		return "create"
	case method == http.MethodPut || method == http.MethodPatch:
		return "update"
	case method == http.MethodDelete:
		return "delete"
	default:
		return "query"
	}
}

func inferActionLabel(method, path string) string {
	switch inferAction(method, path) {
	case "login":
		return "登录"
	case "export":
		return "导出日志"
	case "archive":
		return "归档日志"
	case "cleanup":
		return "清理日志"
	case "ack":
		return "确认"
	case "resolve":
		return "解决"
	case "close":
		return "关闭"
	case "enable":
		return "启用"
	case "disable":
		return "停用"
	case "execute":
		return "执行操作"
	case "create":
		return "新增"
	case "update":
		return "更新配置"
	case "delete":
		return "删除"
	default:
		return "访问页面"
	}
}

func resolveFIMResourceName(path string) string {
	switch {
	case strings.Contains(path, "/scan"):
		return "FIM 扫描任务"
	case strings.Contains(path, "/alerts"):
		return "完整性告警"
	case strings.Contains(path, "/events"):
		return "文件变更事件"
	case strings.Contains(path, "/watch-paths"):
		return "巡检目录配置"
	default:
		return "文件完整性巡检"
	}
}

func deriveGenericResourceName(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "接口访问"
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) <= 1 {
		return trimmed
	}
	last := parts[len(parts)-1]
	if _, err := strconv.Atoi(last); err == nil && len(parts) >= 2 {
		last = parts[len(parts)-2]
	}
	last = strings.ReplaceAll(last, "-", "_")
	switch last {
	case "profile":
		return "个人信息"
	case "menus":
		return "菜单访问"
	case "settings":
		return "系统设置"
	case "login":
		return "登录接口"
	default:
		return last
	}
}

func buildChangeSummary(meta requestMeta, path string) string {
	if meta.Action == "query" {
		return fmt.Sprintf("访问了 %s，查看当前列表或详情。", meta.MenuTitle)
	}
	return fmt.Sprintf("在 %s 中执行了%s，请求路径 %s。", meta.MenuTitle, meta.ActionLabel, path)
}

func resolveResourceID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if _, err := strconv.Atoi(parts[i]); err == nil {
			return parts[i]
		}
	}
	return ""
}

func readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return truncateString(string(body), 20000)
}

func extractErrorMessage(responseBody string, statusCode int) string {
	if responseBody == "" || statusCode < http.StatusBadRequest {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(responseBody), &payload); err != nil {
		return ""
	}
	if value, ok := payload["error"].(string); ok {
		return truncateString(value, 2000)
	}
	if value, ok := payload["message"].(string); ok {
		return truncateString(value, 2000)
	}
	return ""
}

func buildTraceID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

func isLoginRequest(ctx requestContext) bool {
	return ctx.RequestPath == "/api/auth/login"
}

func isAccessRequest(ctx requestContext) bool {
	if ctx.RequestMethod != http.MethodGet || ctx.RequestPath == "/api/auth/login" {
		return false
	}
	if ctx.RequestPath == "/api/admin/audit/export" {
		return false
	}
	if shouldIgnoreAccessRequest(ctx.RequestPath) {
		return false
	}
	pageMeta, ok := resolveMenuMetaByPage(ctx.PagePathHint)
	if !ok {
		return false
	}
	return shouldPersistAccessByWindow(ctx, pageMeta.PagePath)
}

func isOperationRequest(ctx requestContext) bool {
	return ctx.RequestPath == "/api/admin/audit/export" ||
		(ctx.RequestMethod != http.MethodGet && ctx.RequestPath != "/api/auth/login")
}

func shouldIgnoreAccessRequest(path string) bool {
	switch {
	case path == "/api/health":
		return true
	case strings.HasPrefix(path, "/api/admin/audit"):
		return true
	case strings.HasPrefix(path, "/api/user/menus"):
		return true
	case strings.HasPrefix(path, "/api/user/profile"):
		return true
	case strings.HasPrefix(path, "/api/auth/refresh"):
		return true
	default:
		return false
	}
}

func shouldPersistAccessByWindow(ctx requestContext, pagePath string) bool {
	key := buildAccessDedupKey(ctx, pagePath)
	now := ctx.OccurredAt

	accessLogTracker.Lock()
	defer accessLogTracker.Unlock()

	for existingKey, seenAt := range accessLogTracker.lastSeen {
		if now.Sub(seenAt) > accessLogDedupWindow {
			delete(accessLogTracker.lastSeen, existingKey)
		}
	}

	if seenAt, ok := accessLogTracker.lastSeen[key]; ok && now.Sub(seenAt) < accessLogDedupWindow {
		return false
	}

	accessLogTracker.lastSeen[key] = now
	return true
}

func buildAccessDedupKey(ctx requestContext, pagePath string) string {
	identity := ctx.Username
	if identity == "" {
		identity = fmt.Sprintf("uid:%d", ctx.UserID)
	}
	if identity == "" {
		identity = ctx.RequestIP
	}
	return identity + "|" + pagePath
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}