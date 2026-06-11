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
		MenuTitle:    "API Access",
		PagePath:     firstNonEmptyPagePath(pagePathHint, path),
		Module:       "general",
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
		if meta.ResourceName == "" || meta.ResourceName == "API Access" {
			meta.ResourceName = pageMeta.ResourceName
		}
	}

	switch {
	case strings.HasPrefix(path, "/api/security/fim/"):
		meta.MenuKey = "security-fim"
		meta.MenuTitle = "File Integrity Monitoring"
		meta.PagePath = "/security/fim/policies"
		meta.Module = "security"
		meta.ResourceType = "fim"
		meta.ResourceName = resolveFIMResourceName(path)
	case strings.HasPrefix(path, "/api/admin/settings"):
		meta.MenuKey = "admin-settings"
		meta.MenuTitle = "System Settings"
		meta.PagePath = "/admin/settings"
		meta.Module = "system"
		meta.ResourceType = "settings"
		meta.ResourceName = "System Settings"
	case strings.HasPrefix(path, "/api/admin/audit"):
		meta.MenuKey = "platform-audit"
		meta.MenuTitle = "Platform Audit"
		meta.PagePath = "/platform/audit"
		meta.Module = "audit"
		meta.ResourceType = "audit"
		meta.ResourceName = "Platform Audit"
	case strings.HasPrefix(path, "/api/alert/"):
		meta.MenuKey = "alarm-center"
		meta.MenuTitle = "Alert Center"
		meta.PagePath = "/alarm/events"
		meta.Module = "alert"
		meta.ResourceType = "alert"
		meta.ResourceName = "Alert Center"
	case strings.HasPrefix(path, "/api/auth/login"):
		meta.MenuKey = "auth-login"
		meta.MenuTitle = "Login Log"
		meta.PagePath = "/login"
		meta.Module = "auth"
		meta.ResourceType = "login"
		meta.ResourceName = "Management Platform"
	case strings.HasPrefix(path, "/api/user/menus"):
		meta.MenuKey = "platform-audit"
		meta.MenuTitle = "Platform Audit"
		meta.PagePath = "/platform/audit"
		meta.Module = "audit"
		meta.ResourceType = "menu_access"
		meta.ResourceName = "Menu Access"
	case strings.HasPrefix(path, "/api/user/"):
		meta.MenuKey = "platform-user"
		meta.MenuTitle = "User Center"
		meta.PagePath = "/profile"
		meta.Module = "user"
		meta.ResourceType = "user"
		meta.ResourceName = deriveGenericResourceName(path)
	case strings.HasPrefix(path, "/api/auth/"):
		meta.MenuKey = "auth-center"
		meta.MenuTitle = "Auth Center"
		meta.PagePath = "/login"
		meta.Module = "auth"
		meta.ResourceType = "auth"
		meta.ResourceName = deriveGenericResourceName(path)
	case strings.HasPrefix(path, "/api/admin/"):
		meta.MenuKey = "system-admin"
		meta.MenuTitle = "System Management"
		meta.PagePath = "/admin/settings"
		meta.Module = "system"
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
		return requestMeta{MenuKey: "dashboard", MenuTitle: "Dashboard", PagePath: "/", Module: "dashboard", ResourceType: "dashboard", ResourceName: "Dashboard"}, true
	case strings.HasPrefix(pagePath, "/cmdb/projects"):
		return requestMeta{MenuKey: "cmdb-projects", MenuTitle: "Projects", PagePath: "/cmdb/projects", Module: "cmdb", ResourceType: "project", ResourceName: "Projects"}, true
	case strings.HasPrefix(pagePath, "/cmdb/environments"):
		return requestMeta{MenuKey: "cmdb-environments", MenuTitle: "Environments", PagePath: "/cmdb/environments", Module: "cmdb", ResourceType: "environment", ResourceName: "Environments"}, true
	case strings.HasPrefix(pagePath, "/cmdb/servers"):
		return requestMeta{MenuKey: "cmdb-servers", MenuTitle: "Servers", PagePath: "/cmdb/servers", Module: "cmdb", ResourceType: "server", ResourceName: "Servers"}, true
	case strings.HasPrefix(pagePath, "/cmdb/applications"):
		return requestMeta{MenuKey: "cmdb-applications", MenuTitle: "Applications", PagePath: "/cmdb/applications", Module: "cmdb", ResourceType: "application", ResourceName: "Applications"}, true
	case strings.HasPrefix(pagePath, "/deploy/release"):
		return requestMeta{MenuKey: "deploy-release", MenuTitle: "Release Deploy", PagePath: "/deploy/release", Module: "deploy", ResourceType: "deploy", ResourceName: "Release Deploy"}, true
	case strings.HasPrefix(pagePath, "/deploy/history"):
		return requestMeta{MenuKey: "deploy-history", MenuTitle: "Deploy History", PagePath: "/deploy/history", Module: "deploy", ResourceType: "deploy_record", ResourceName: "Deploy History"}, true
	case strings.HasPrefix(pagePath, "/deploy/archived"):
		return requestMeta{MenuKey: "deploy-archived", MenuTitle: "Archive History", PagePath: "/deploy/archived", Module: "deploy", ResourceType: "archive_record", ResourceName: "Archive History"}, true
	case strings.HasPrefix(pagePath, "/deploy/archive"):
		return requestMeta{MenuKey: "deploy-archive", MenuTitle: "Archive Package", PagePath: "/deploy/archive", Module: "deploy", ResourceType: "archive", ResourceName: "Archive Package"}, true
	case strings.HasPrefix(pagePath, "/deploy/aggregate-package"):
		return requestMeta{MenuKey: "deploy-aggregate-package", MenuTitle: "Aggregate Package", PagePath: "/deploy/aggregate-package", Module: "deploy", ResourceType: "aggregate_package", ResourceName: "Aggregate Package"}, true
	case strings.HasPrefix(pagePath, "/deploy/aggregated-history"):
		return requestMeta{MenuKey: "aggregated-history", MenuTitle: "Aggregated History", PagePath: "/deploy/aggregated-history", Module: "deploy", ResourceType: "aggregate_history", ResourceName: "Aggregated History"}, true
	case strings.HasPrefix(pagePath, "/consul/config"):
		return requestMeta{MenuKey: "consul-config", MenuTitle: "Config Management", PagePath: "/consul/config", Module: "deploy", ResourceType: "consul_config", ResourceName: "Config Management"}, true
	case strings.HasPrefix(pagePath, "/consul/batch-all"):
		return requestMeta{MenuKey: "consul-batch-all", MenuTitle: "Batch Config Push", PagePath: "/consul/batch-all", Module: "deploy", ResourceType: "consul_batch", ResourceName: "Batch Config Push"}, true
	case strings.HasPrefix(pagePath, "/consul/operations"):
		return requestMeta{MenuKey: "consul-operations", MenuTitle: "Config Operations", PagePath: "/consul/operations", Module: "deploy", ResourceType: "consul_operation", ResourceName: "Config Operations"}, true
	case strings.HasPrefix(pagePath, "/jenkins/views"):
		return requestMeta{MenuKey: "jenkins-views", MenuTitle: "Jenkins Views", PagePath: "/jenkins/views", Module: "deploy", ResourceType: "jenkins_view", ResourceName: "Jenkins Views"}, true
	case strings.HasPrefix(pagePath, "/monitor/bigscreen") || pagePath == "/monitor":
		return requestMeta{MenuKey: "monitor-bigscreen", MenuTitle: "Monitor Dashboard", PagePath: "/monitor/bigscreen", Module: "monitor", ResourceType: "monitor", ResourceName: "Monitor Dashboard"}, true
	case strings.HasPrefix(pagePath, "/monitor/overview"):
		return requestMeta{MenuKey: "monitor-overview", MenuTitle: "Monitor Overview", PagePath: "/monitor/overview", Module: "monitor", ResourceType: "monitor", ResourceName: "Monitor Overview"}, true
	case strings.HasPrefix(pagePath, "/monitor/dashboards"):
		return requestMeta{MenuKey: "monitor-dashboards", MenuTitle: "Grafana Dashboards", PagePath: "/monitor/dashboards", Module: "monitor", ResourceType: "grafana", ResourceName: "Grafana Dashboards"}, true
	case strings.HasPrefix(pagePath, "/alarm/events") || pagePath == "/alarm":
		return requestMeta{MenuKey: "alarm-events", MenuTitle: "Alert Events", PagePath: "/alarm/events", Module: "alert", ResourceType: "alert_event", ResourceName: "Alert Events"}, true
	case strings.HasPrefix(pagePath, "/alarm/rules"):
		return requestMeta{MenuKey: "alarm-rules", MenuTitle: "Alert Rules", PagePath: "/alarm/rules", Module: "alert", ResourceType: "alert_rule", ResourceName: "Alert Rules"}, true
	case strings.HasPrefix(pagePath, "/alarm/contacts"):
		return requestMeta{MenuKey: "alarm-contacts", MenuTitle: "Contacts", PagePath: "/alarm/contacts", Module: "alert", ResourceType: "alert_contact", ResourceName: "Contacts"}, true
	case strings.HasPrefix(pagePath, "/alarm/channels"):
		return requestMeta{MenuKey: "alarm-channels", MenuTitle: "Notify Channels", PagePath: "/alarm/channels", Module: "alert", ResourceType: "notify_channel", ResourceName: "Notify Channels"}, true
	case strings.HasPrefix(pagePath, "/alarm/templates"):
		return requestMeta{MenuKey: "alarm-templates", MenuTitle: "Notify Templates", PagePath: "/alarm/templates", Module: "alert", ResourceType: "notify_template", ResourceName: "Notify Templates"}, true
	case strings.HasPrefix(pagePath, "/security/overview") || pagePath == "/security":
		return requestMeta{MenuKey: "security-overview", MenuTitle: "Security Overview", PagePath: "/security/overview", Module: "security", ResourceType: "security_overview", ResourceName: "Security Overview"}, true
	case strings.HasPrefix(pagePath, "/security/fim/policies"):
		return requestMeta{MenuKey: "security-fim-policies", MenuTitle: "FIM Policies", PagePath: "/security/fim/policies", Module: "security", ResourceType: "fim_policy", ResourceName: "FIM Policies"}, true
	case strings.HasPrefix(pagePath, "/security/fim/executions"):
		return requestMeta{MenuKey: "security-fim-executions", MenuTitle: "FIM Executions", PagePath: "/security/fim/executions", Module: "security", ResourceType: "fim_snapshot", ResourceName: "FIM Executions"}, true
	case strings.HasPrefix(pagePath, "/security/fim/events"):
		return requestMeta{MenuKey: "security-fim-events", MenuTitle: "File Change Events", PagePath: "/security/fim/events", Module: "security", ResourceType: "fim_event", ResourceName: "File Change Events"}, true
	case strings.HasPrefix(pagePath, "/security/fim/alerts"):
		return requestMeta{MenuKey: "security-fim-alerts", MenuTitle: "Integrity Alerts", PagePath: "/security/fim/alerts", Module: "security", ResourceType: "fim_alert", ResourceName: "Integrity Alerts"}, true
	case strings.HasPrefix(pagePath, "/security/tasks"):
		return requestMeta{MenuKey: "security-tasks", MenuTitle: "Scan Tasks", PagePath: "/security/tasks", Module: "security", ResourceType: "security_task", ResourceName: "Scan Tasks"}, true
	case strings.HasPrefix(pagePath, "/security/assets"):
		return requestMeta{MenuKey: "security-assets", MenuTitle: "Security Assets", PagePath: "/security/assets", Module: "security", ResourceType: "security_asset", ResourceName: "Security Assets"}, true
	case strings.HasPrefix(pagePath, "/security/vulnerabilities"):
		return requestMeta{MenuKey: "security-vulnerabilities", MenuTitle: "Vulnerabilities", PagePath: "/security/vulnerabilities", Module: "security", ResourceType: "security_vulnerability", ResourceName: "Vulnerabilities"}, true
	case strings.HasPrefix(pagePath, "/security/tickets"):
		return requestMeta{MenuKey: "security-tickets", MenuTitle: "Vulnerability Tickets", PagePath: "/security/tickets", Module: "security", ResourceType: "security_ticket", ResourceName: "Vulnerability Tickets"}, true
	case strings.HasPrefix(pagePath, "/security/vuln-db"):
		return requestMeta{MenuKey: "security-vuln-db", MenuTitle: "Vulnerability Knowledge DB", PagePath: "/security/vuln-db", Module: "security", ResourceType: "security_vuln_db", ResourceName: "Vulnerability Knowledge DB"}, true
	case strings.HasPrefix(pagePath, "/admin/users"):
		return requestMeta{MenuKey: "admin-users", MenuTitle: "Users", PagePath: "/admin/users", Module: "system", ResourceType: "admin_user", ResourceName: "Users"}, true
	case strings.HasPrefix(pagePath, "/admin/roles"):
		return requestMeta{MenuKey: "admin-roles", MenuTitle: "Roles", PagePath: "/admin/roles", Module: "system", ResourceType: "admin_role", ResourceName: "Roles"}, true
	case strings.HasPrefix(pagePath, "/admin/menus"):
		return requestMeta{MenuKey: "admin-menus", MenuTitle: "Menus", PagePath: "/admin/menus", Module: "system", ResourceType: "admin_menu", ResourceName: "Menus"}, true
	case strings.HasPrefix(pagePath, "/admin/settings"):
		return requestMeta{MenuKey: "admin-settings", MenuTitle: "System Settings", PagePath: "/admin/settings", Module: "system", ResourceType: "settings", ResourceName: "System Settings"}, true
	case strings.HasPrefix(pagePath, "/platform/audit"):
		return requestMeta{MenuKey: "platform-audit", MenuTitle: "Platform Audit", PagePath: "/platform/audit", Module: "audit", ResourceType: "audit", ResourceName: "Platform Audit"}, true
	case strings.HasPrefix(pagePath, "/platform/events"):
		return requestMeta{MenuKey: "platform-events", MenuTitle: "Platform Events", PagePath: "/platform/events", Module: "events", ResourceType: "platform_event", ResourceName: "Platform Events"}, true
	case strings.HasPrefix(pagePath, "/profile"):
		return requestMeta{MenuKey: "profile", MenuTitle: "My Profile", PagePath: "/profile", Module: "user", ResourceType: "profile", ResourceName: "My Profile"}, true
	case strings.HasPrefix(pagePath, "/user-manual"):
		return requestMeta{MenuKey: "user-manual", MenuTitle: "User Manual", PagePath: "/user-manual", Module: "general", ResourceType: "manual", ResourceName: "User Manual"}, true
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
		return "login"
	case "export":
		return "export_logs"
	case "archive":
		return "archive_logs"
	case "cleanup":
		return "cleanup_logs"
	case "ack":
		return "acknowledge"
	case "resolve":
		return "resolve"
	case "close":
		return "close"
	case "enable":
		return "enable"
	case "disable":
		return "disable"
	case "execute":
		return "execute"
	case "create":
		return "create"
	case "update":
		return "update"
	case "delete":
		return "delete"
	default:
		return "visit_page"
	}
}

func resolveFIMResourceName(path string) string {
	switch {
	case strings.Contains(path, "/scan"):
		return "FIM Scan Task"
	case strings.Contains(path, "/alerts"):
		return "Integrity Alert"
	case strings.Contains(path, "/events"):
		return "File Change Event"
	case strings.Contains(path, "/watch-paths"):
		return "Watch Path Config"
	default:
		return "File Integrity Monitoring"
	}
}

func deriveGenericResourceName(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "API Access"
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
		return "Profile"
	case "menus":
		return "Menu Access"
	case "settings":
		return "System Settings"
	case "login":
		return "Login"
	default:
		return last
	}
}

func buildChangeSummary(meta requestMeta, path string) string {
	if meta.Action == "query" {
		return fmt.Sprintf("Viewed %s page, browsing current list or details.", meta.MenuTitle)
	}
	return fmt.Sprintf("Performed %s in %s at %s.", meta.ActionLabel, meta.MenuTitle, path)
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