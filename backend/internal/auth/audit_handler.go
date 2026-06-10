// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package auth

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/audit"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPlatformAccessLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.PlatformAccessLog
		query := database.DB.Model(&models.PlatformAccessLog{})
		page, pageSize := getAuditPageParams(c)
		query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "accessed_at")

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count access logs"})
			return
		}
		if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch access logs"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "page": page, "page_size": pageSize})
	}
}

func GetPlatformAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.PlatformAuditLog
		query := database.DB.Model(&models.PlatformAuditLog{})
		page, pageSize := getAuditPageParams(c)
		query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "operated_at")
		if module := strings.TrimSpace(c.Query("module")); module != "" {
			query = query.Where("module = ?", module)
		}
		if action := strings.TrimSpace(c.Query("action")); action != "" {
			query = query.Where("action = ?", action)
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs"})
			return
		}
		if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "page": page, "page_size": pageSize})
	}
}

func GetPlatformLoginLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.PlatformLoginLog
		query := database.DB.Model(&models.PlatformLoginLog{})
		page, pageSize := getAuditPageParams(c)
		query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "logged_in_at")

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count login logs"})
			return
		}
		if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch login logs"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "page": page, "page_size": pageSize})
	}
}

func DeletePlatformAccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		deletePlatformAuditRecord(c, &models.PlatformAccessLog{}, "access log")
	}
}

func DeletePlatformAuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		deletePlatformAuditRecord(c, &models.PlatformAuditLog{}, "audit log")
	}
}

func DeletePlatformLoginLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		deletePlatformAuditRecord(c, &models.PlatformLoginLog{}, "login log")
	}
}

type AuditRetentionRequest struct {
	RetentionDays int `json:"retention_days"`
}

type AuditRetentionResponse struct {
	RetentionDays     int   `json:"retention_days"`
	AccessAffected    int64 `json:"access_affected"`
	OperationAffected int64 `json:"operation_affected"`
	LoginAffected     int64 `json:"login_affected"`
}

type PlatformArchivedLogItem struct {
	ID              uint      `json:"id"`
	ArchiveType     string    `json:"archive_type"`
	Username        string    `json:"username"`
	RealName        string    `json:"real_name"`
	Role            string    `json:"role"`
	Title           string    `json:"title"`
	RequestPath     string    `json:"request_path"`
	RequestMethod   string    `json:"request_method"`
	RequestIP       string    `json:"request_ip"`
	OperationStatus string    `json:"operation_status"`
	StatusCode      int       `json:"status_code"`
	DurationMS      int64     `json:"duration_ms"`
	ErrorMessage    string    `json:"error_message"`
	OccurredAt      time.Time `json:"occurred_at"`
	ArchivedAt      time.Time `json:"archived_at"`
}

type PlatformArchiveStats struct {
	AccessTotal    int64   `json:"access_total"`
	OperationTotal int64   `json:"operation_total"`
	LoginTotal     int64   `json:"login_total"`
	Total          int64   `json:"total"`
	LatestArchived *string `json:"latest_archived"`
}

func ArchivePlatformAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		retentionDays, ok := parseAuditRetentionRequest(c)
		if !ok {
			return
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		result, err := archivePlatformAuditLogs(cutoff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to archive audit logs"})
			return
		}
		result.RetentionDays = retentionDays
		c.JSON(http.StatusOK, result)
	}
}

func CleanupPlatformAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		retentionDays, ok := parseAuditRetentionRequest(c)
		if !ok {
			return
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		result, err := cleanupPlatformAuditLogs(cutoff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup audit logs"})
			return
		}
		result.RetentionDays = retentionDays
		c.JSON(http.StatusOK, result)
	}
}

func CleanupPlatformOnlineAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		retentionDays, ok := parseAuditRetentionRequest(c)
		if !ok {
			return
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		result, err := cleanupPlatformOnlineAuditLogs(cutoff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup online audit logs"})
			return
		}
		result.RetentionDays = retentionDays
		c.JSON(http.StatusOK, result)
	}
}

func GetPlatformArchiveStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		var accessTotal, operationTotal, loginTotal int64
		if err := database.DB.Model(&models.PlatformAccessLogArchive{}).Count(&accessTotal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count archived access logs"})
			return
		}
		if err := database.DB.Model(&models.PlatformAuditLogArchive{}).Count(&operationTotal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count archived audit logs"})
			return
		}
		if err := database.DB.Model(&models.PlatformLoginLogArchive{}).Count(&loginTotal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count archived login logs"})
			return
		}

		var latestAccess, latestOperation, latestLogin time.Time
		database.DB.Model(&models.PlatformAccessLogArchive{}).Select("MAX(archived_at)").Scan(&latestAccess)
		database.DB.Model(&models.PlatformAuditLogArchive{}).Select("MAX(archived_at)").Scan(&latestOperation)
		database.DB.Model(&models.PlatformLoginLogArchive{}).Select("MAX(archived_at)").Scan(&latestLogin)

		latest := latestAccess
		if latestOperation.After(latest) {
			latest = latestOperation
		}
		if latestLogin.After(latest) {
			latest = latestLogin
		}

		resp := PlatformArchiveStats{
			AccessTotal:    accessTotal,
			OperationTotal: operationTotal,
			LoginTotal:     loginTotal,
			Total:          accessTotal + operationTotal + loginTotal,
		}
		if !latest.IsZero() {
			formatted := latest.Format("2006-01-02 15:04:05")
			resp.LatestArchived = &formatted
		}
		c.JSON(http.StatusOK, resp)
	}
}

func GetPlatformArchivedLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize := getAuditPageParams(c)
		offset := (page - 1) * pageSize
		logType := strings.TrimSpace(c.Query("archive_type"))
		whereClauses := make([]string, 0, 8)
		args := make([]any, 0, 8)
		switch logType {
		case "", "all":
		case "access":
			whereClauses = append(whereClauses, "archive_type = ?")
			args = append(args, "access")
		case "operation":
			whereClauses = append(whereClauses, "archive_type = ?")
			args = append(args, "operation")
		case "login":
			whereClauses = append(whereClauses, "archive_type = ?")
			args = append(args, "login")
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported archive type"})
			return
		}
		whereClauses, args = appendArchiveAuditFilters(c, whereClauses, args)
		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
		}

		unionSQL := `
			SELECT id, 'access' AS archive_type, username, real_name, role, COALESCE(menu_title, '访问日志') AS title,
				request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
				accessed_at AS occurred_at, archived_at
			FROM platform_access_logs_archive
			UNION ALL
			SELECT id, 'operation' AS archive_type, username, real_name, role, COALESCE(resource_name, action_label, '操作审计') AS title,
				request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
				operated_at AS occurred_at, archived_at
			FROM platform_audit_logs_archive
			UNION ALL
			SELECT id, 'login' AS archive_type, username, real_name, role, '登录日志' AS title,
				request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
				logged_in_at AS occurred_at, archived_at
			FROM platform_login_logs_archive
		`

		var total int64
		countSQL := fmt.Sprintf("SELECT COUNT(*) AS total FROM (%s) AS archived_logs %s", unionSQL, whereClause)
		if err := database.DB.Raw(countSQL, args...).Scan(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count archived logs"})
			return
		}

		var items []PlatformArchivedLogItem
		listSQL := fmt.Sprintf(`
			SELECT * FROM (%s) AS archived_logs
			%s
			ORDER BY archived_at DESC, occurred_at DESC
			LIMIT ? OFFSET ?
		`, unionSQL, whereClause)
		listArgs := append(append([]any{}, args...), pageSize, offset)
		if err := database.DB.Raw(listSQL, listArgs...).Scan(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch archived logs"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "page": page, "page_size": pageSize})
	}
}

func DeletePlatformArchivedLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		archiveType := strings.TrimSpace(c.Param("type"))
		id := parsePositiveInt(strings.TrimSpace(c.Param("id")))
		if id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid archive log id"})
			return
		}

		var resp *gorm.DB
		switch archiveType {
		case "access":
			resp = database.DB.Delete(&models.PlatformAccessLogArchive{}, id)
		case "operation":
			resp = database.DB.Delete(&models.PlatformAuditLogArchive{}, id)
		case "login":
			resp = database.DB.Delete(&models.PlatformLoginLogArchive{}, id)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported archive type"})
			return
		}
		if resp.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete archived log"})
			return
		}
		if resp.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "archived log not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func ExportPlatformAuditLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		exportType := strings.TrimSpace(c.Query("type"))
		if exportType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing export type"})
			return
		}
		exporter, ok := resolvePlatformAuditExporter(exportType)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported export type"})
			return
		}

		filename := buildPlatformAuditExportFilename(c, exportType)
		audit.SetOperationAuditAfter(c, gin.H{
			"export_type":  exportType,
			"archive_type": strings.TrimSpace(c.Query("archive_type")),
			"start":        strings.TrimSpace(c.Query("start")),
			"end":          strings.TrimSpace(c.Query("end")),
			"module":       strings.TrimSpace(c.Query("module")),
			"action":       strings.TrimSpace(c.Query("action")),
			"status":       strings.TrimSpace(c.Query("status")),
			"request_ip":   strings.TrimSpace(c.Query("request_ip")),
			"request_path": strings.TrimSpace(c.Query("request_path")),
			"operator":     strings.TrimSpace(c.Query("operator")),
			"export_file":  filename,
			"exported_at":  time.Now().Format(time.DateTime),
		})
		audit.SetOperationAuditSummary(c, buildPlatformAuditExportSummary(c, exportType, filename))

		var csvBuffer bytes.Buffer
		writer := csv.NewWriter(&csvBuffer)
		err := exporter(c, writer)
		writer.Flush()
		if err != nil {
			statusCode := http.StatusInternalServerError
			if exportType == "archive" && strings.Contains(strings.ToLower(err.Error()), "unsupported archive type") {
				statusCode = http.StatusBadRequest
			}
			c.JSON(statusCode, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		c.Header("Cache-Control", "no-store")

		_, _ = c.Writer.Write([]byte("\uFEFF"))
		_, _ = c.Writer.Write(csvBuffer.Bytes())
	}
}

func resolvePlatformAuditExporter(exportType string) (func(*gin.Context, *csv.Writer) error, bool) {
	switch exportType {
	case "access":
		return exportAccessLogsCSV, true
	case "operation":
		return exportOperationLogsCSV, true
	case "login":
		return exportLoginLogsCSV, true
	case "archive":
		return exportArchivedLogsCSV, true
	default:
		return nil, false
	}
}

func buildPlatformAuditExportFilename(c *gin.Context, exportType string) string {
	parts := []string{"platform-audit", sanitizeAuditFilenameSegment(exportType)}
	if exportType == "archive" {
		if archiveType := sanitizeAuditFilenameSegment(strings.TrimSpace(c.Query("archive_type"))); archiveType != "" && archiveType != "all" {
			parts = append(parts, archiveType)
		}
	}
	if dateRange := buildAuditExportDateRangeSegment(c); dateRange != "" {
		parts = append(parts, dateRange)
	}
	parts = append(parts, time.Now().Format("20060102150405"))
	return strings.Join(parts, "-") + ".csv"
}

func buildAuditExportDateRangeSegment(c *gin.Context) string {
	start := sanitizeAuditFilenameSegment(strings.TrimSpace(c.Query("start")))
	end := sanitizeAuditFilenameSegment(strings.TrimSpace(c.Query("end")))
	switch {
	case start != "" && end != "":
		return start + "_to_" + end
	case start != "":
		return "from_" + start
	case end != "":
		return "until_" + end
	default:
		return ""
	}
}

func sanitizeAuditFilenameSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		".", "-",
		"_", "-",
	)
	value = replacer.Replace(value)
	var builder strings.Builder
	lastDash := false
	for _, ch := range value {
		isAllowed := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAllowed {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if ch == '-' && !lastDash && builder.Len() > 0 {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func buildPlatformAuditExportSummary(c *gin.Context, exportType, filename string) string {
	label := map[string]string{
		"access":    "访问日志",
		"operation": "操作审计",
		"login":     "登录日志",
		"archive":   "归档日志",
	}[exportType]
	if label == "" {
		label = "平台审计日志"
	}
	if exportType == "archive" {
		if archiveType := strings.TrimSpace(c.Query("archive_type")); archiveType != "" && archiveType != "all" {
			typeLabel := map[string]string{
				"access":    "访问日志",
				"operation": "操作审计",
				"login":     "登录日志",
			}[archiveType]
			if typeLabel != "" {
				label = "归档" + typeLabel
			}
		}
	}
	dateRange := buildAuditExportDateRangeLabel(c)
	if dateRange != "" {
		return fmt.Sprintf("导出了%s，时间范围 %s，文件 %s。", label, dateRange, filename)
	}
	return fmt.Sprintf("导出了%s，文件 %s。", label, filename)
}

func buildAuditExportDateRangeLabel(c *gin.Context) string {
	start := strings.TrimSpace(c.Query("start"))
	end := strings.TrimSpace(c.Query("end"))
	switch {
	case start != "" && end != "":
		return start + " 至 " + end
	case start != "":
		return start + " 之后"
	case end != "":
		return end + " 之前"
	default:
		return ""
	}
}

func deletePlatformAuditRecord(c *gin.Context, model any, label string) {
	id := parsePositiveInt(strings.TrimSpace(c.Param("id")))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + label + " id"})
		return
	}
	resp := database.DB.Delete(model, id)
	if resp.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete " + label})
		return
	}
	if resp.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": label + " not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func exportAccessLogsCSV(c *gin.Context, writer *csv.Writer) error {
	var items []models.PlatformAccessLog
	query := database.DB.Model(&models.PlatformAccessLog{})
	query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "accessed_at")
	if err := query.Order("id DESC").Find(&items).Error; err != nil {
		return err
	}
	rows := buildAccessLogCSVRows(items)
	return writer.WriteAll(rows)
}

func buildAccessLogCSVRows(items []models.PlatformAccessLog) [][]string {
	rows := [][]string{{"时间", "操作人", "菜单", "请求路径", "请求方法", "请求IP", "状态", "耗时", "错误信息"}}
	for _, item := range items {
		rows = append(rows, []string{
			formatAuditTime(item.AccessedAt),
			displayAuditOperator(item.RealName, item.Username),
			defaultAuditString(item.MenuTitle, "-"),
			item.RequestPath,
			item.RequestMethod,
			item.RequestIP,
			formatAuditStatus(item.OperationStatus),
			fmt.Sprintf("%dms", item.DurationMS),
			defaultAuditString(item.ErrorMessage, "-"),
		})
	}
	return rows
}

func exportOperationLogsCSV(c *gin.Context, writer *csv.Writer) error {
	var items []models.PlatformAuditLog
	query := database.DB.Model(&models.PlatformAuditLog{})
	query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "operated_at")
	if module := strings.TrimSpace(c.Query("module")); module != "" {
		query = query.Where("module = ?", module)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Order("id DESC").Find(&items).Error; err != nil {
		return err
	}
	rows := [][]string{{"时间", "操作人", "模块", "动作", "资源名称", "请求路径", "请求方法", "请求IP", "状态", "耗时", "错误信息"}}
	for _, item := range items {
		rows = append(rows, []string{
			formatAuditTime(item.OperatedAt),
			displayAuditOperator(item.RealName, item.Username),
			defaultAuditString(item.Module, "-"),
			defaultAuditString(firstNonEmpty(item.ActionLabel, item.Action), "-"),
			defaultAuditString(item.ResourceName, "-"),
			item.RequestPath,
			item.RequestMethod,
			item.RequestIP,
			formatAuditStatus(item.OperationStatus),
			fmt.Sprintf("%dms", item.DurationMS),
			defaultAuditString(item.ErrorMessage, "-"),
		})
	}
	return writer.WriteAll(rows)
}

func exportLoginLogsCSV(c *gin.Context, writer *csv.Writer) error {
	var items []models.PlatformLoginLog
	query := database.DB.Model(&models.PlatformLoginLog{})
	query = applyAuditCommonFilters(c, query, "username", "request_ip", "request_path", "operation_status", "logged_in_at")
	if err := query.Order("id DESC").Find(&items).Error; err != nil {
		return err
	}
	rows := [][]string{{"时间", "操作人", "角色", "登录类型", "请求路径", "请求方法", "请求IP", "状态", "耗时", "错误信息"}}
	for _, item := range items {
		rows = append(rows, []string{
			formatAuditTime(item.LoggedInAt),
			displayAuditOperator(item.RealName, item.Username),
			defaultAuditString(item.Role, "-"),
			defaultAuditString(item.LoginType, "password"),
			item.RequestPath,
			item.RequestMethod,
			item.RequestIP,
			formatAuditStatus(item.OperationStatus),
			fmt.Sprintf("%dms", item.DurationMS),
			defaultAuditString(item.ErrorMessage, "-"),
		})
	}
	return writer.WriteAll(rows)
}

func exportArchivedLogsCSV(c *gin.Context, writer *csv.Writer) error {
	logType := strings.TrimSpace(c.Query("archive_type"))
	whereClauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	switch logType {
	case "", "all":
	case "access", "operation", "login":
		whereClauses = append(whereClauses, "archive_type = ?")
		args = append(args, logType)
	default:
		return fmt.Errorf("unsupported archive type")
	}
	whereClauses, args = appendArchiveAuditFilters(c, whereClauses, args)
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	unionSQL := `
		SELECT id, 'access' AS archive_type, username, real_name, role, COALESCE(menu_title, '访问日志') AS title,
			request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
			accessed_at AS occurred_at, archived_at
		FROM platform_access_logs_archive
		UNION ALL
		SELECT id, 'operation' AS archive_type, username, real_name, role, COALESCE(resource_name, action_label, '操作审计') AS title,
			request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
			operated_at AS occurred_at, archived_at
		FROM platform_audit_logs_archive
		UNION ALL
		SELECT id, 'login' AS archive_type, username, real_name, role, '登录日志' AS title,
			request_path, request_method, request_ip, operation_status, status_code, duration_ms, error_message,
			logged_in_at AS occurred_at, archived_at
		FROM platform_login_logs_archive
	`
	listSQL := fmt.Sprintf(`
		SELECT * FROM (%s) AS archived_logs
		%s
		ORDER BY archived_at DESC, occurred_at DESC
	`, unionSQL, whereClause)

	var items []PlatformArchivedLogItem
	if err := database.DB.Raw(listSQL, args...).Scan(&items).Error; err != nil {
		return err
	}

	rows := [][]string{{"归档时间", "原始时间", "类型", "操作人", "标题", "请求路径", "请求方法", "请求IP", "状态", "错误信息"}}
	for _, item := range items {
		rows = append(rows, []string{
			formatAuditTime(item.ArchivedAt),
			formatAuditTime(item.OccurredAt),
			item.ArchiveType,
			displayAuditOperator(item.RealName, item.Username),
			item.Title,
			item.RequestPath,
			item.RequestMethod,
			item.RequestIP,
			formatAuditStatus(item.OperationStatus),
			defaultAuditString(item.ErrorMessage, "-"),
		})
	}
	return writer.WriteAll(rows)
}

func parseAuditRetentionRequest(c *gin.Context) (int, bool) {
	var req AuditRetentionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return 0, false
	}
	switch req.RetentionDays {
	case 7, 30, 180:
		return req.RetentionDays, true
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported retention days"})
		return 0, false
	}
}

func archivePlatformAuditLogs(cutoff time.Time) (AuditRetentionResponse, error) {
	result := AuditRetentionResponse{}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		accessCount, err := archiveAccessLogs(tx, cutoff)
		if err != nil {
			return err
		}
		operationCount, err := archiveOperationLogs(tx, cutoff)
		if err != nil {
			return err
		}
		loginCount, err := archiveLoginLogs(tx, cutoff)
		if err != nil {
			return err
		}
		result.AccessAffected = accessCount
		result.OperationAffected = operationCount
		result.LoginAffected = loginCount
		return nil
	})
	return result, err
}

func cleanupPlatformAuditLogs(cutoff time.Time) (AuditRetentionResponse, error) {
	result := AuditRetentionResponse{}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		accessResp := tx.Where("accessed_at < ?", cutoff).Delete(&models.PlatformAccessLogArchive{})
		if accessResp.Error != nil {
			return accessResp.Error
		}
		operationResp := tx.Where("operated_at < ?", cutoff).Delete(&models.PlatformAuditLogArchive{})
		if operationResp.Error != nil {
			return operationResp.Error
		}
		loginResp := tx.Where("logged_in_at < ?", cutoff).Delete(&models.PlatformLoginLogArchive{})
		if loginResp.Error != nil {
			return loginResp.Error
		}
		result.AccessAffected = accessResp.RowsAffected
		result.OperationAffected = operationResp.RowsAffected
		result.LoginAffected = loginResp.RowsAffected
		return nil
	})
	return result, err
}

func cleanupPlatformOnlineAuditLogs(cutoff time.Time) (AuditRetentionResponse, error) {
	result := AuditRetentionResponse{}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		accessResp := tx.Where("accessed_at < ?", cutoff).Delete(&models.PlatformAccessLog{})
		if accessResp.Error != nil {
			return accessResp.Error
		}
		operationResp := tx.Where("operated_at < ?", cutoff).Delete(&models.PlatformAuditLog{})
		if operationResp.Error != nil {
			return operationResp.Error
		}
		loginResp := tx.Where("logged_in_at < ?", cutoff).Delete(&models.PlatformLoginLog{})
		if loginResp.Error != nil {
			return loginResp.Error
		}
		result.AccessAffected = accessResp.RowsAffected
		result.OperationAffected = operationResp.RowsAffected
		result.LoginAffected = loginResp.RowsAffected
		return nil
	})
	return result, err
}

func archiveAccessLogs(tx *gorm.DB, cutoff time.Time) (int64, error) {
	var logs []models.PlatformAccessLog
	if err := tx.Where("accessed_at < ?", cutoff).Find(&logs).Error; err != nil {
		return 0, err
	}
	if len(logs) == 0 {
		return 0, nil
	}
	archives := make([]models.PlatformAccessLogArchive, 0, len(logs))
	now := time.Now()
	for _, log := range logs {
		archives = append(archives, models.PlatformAccessLogArchive{
			ID:              log.ID,
			TraceID:         log.TraceID,
			UserID:          log.UserID,
			Username:        log.Username,
			RealName:        log.RealName,
			Role:            log.Role,
			MenuKey:         log.MenuKey,
			MenuTitle:       log.MenuTitle,
			PagePath:        log.PagePath,
			RequestPath:     log.RequestPath,
			RequestMethod:   log.RequestMethod,
			RequestIP:       log.RequestIP,
			UserAgent:       log.UserAgent,
			Referer:         log.Referer,
			StatusCode:      log.StatusCode,
			OperationStatus: log.OperationStatus,
			DurationMS:      log.DurationMS,
			ErrorMessage:    log.ErrorMessage,
			AccessedAt:      log.AccessedAt,
			CreatedAt:       log.CreatedAt,
			ArchivedAt:      now,
		})
	}
	if err := tx.CreateInBatches(&archives, 500).Error; err != nil {
		return 0, err
	}
	ids := make([]uint, 0, len(logs))
	for _, log := range logs {
		ids = append(ids, log.ID)
	}
	if err := tx.Where("id IN ?", ids).Delete(&models.PlatformAccessLog{}).Error; err != nil {
		return 0, err
	}
	return int64(len(logs)), nil
}

func archiveOperationLogs(tx *gorm.DB, cutoff time.Time) (int64, error) {
	var logs []models.PlatformAuditLog
	if err := tx.Where("operated_at < ?", cutoff).Find(&logs).Error; err != nil {
		return 0, err
	}
	if len(logs) == 0 {
		return 0, nil
	}
	archives := make([]models.PlatformAuditLogArchive, 0, len(logs))
	now := time.Now()
	for _, log := range logs {
		archives = append(archives, models.PlatformAuditLogArchive{
			ID:              log.ID,
			TraceID:         log.TraceID,
			UserID:          log.UserID,
			Username:        log.Username,
			RealName:        log.RealName,
			Role:            log.Role,
			Module:          log.Module,
			ResourceType:    log.ResourceType,
			ResourceID:      log.ResourceID,
			ResourceName:    log.ResourceName,
			Action:          log.Action,
			ActionLabel:     log.ActionLabel,
			RequestPath:     log.RequestPath,
			RequestMethod:   log.RequestMethod,
			RequestIP:       log.RequestIP,
			StatusCode:      log.StatusCode,
			OperationStatus: log.OperationStatus,
			RequestParams:   log.RequestParams,
			BeforeData:      log.BeforeData,
			AfterData:       log.AfterData,
			ChangeSummary:   log.ChangeSummary,
			ErrorMessage:    log.ErrorMessage,
			DurationMS:      log.DurationMS,
			OperatedAt:      log.OperatedAt,
			CreatedAt:       log.CreatedAt,
			ArchivedAt:      now,
		})
	}
	if err := tx.CreateInBatches(&archives, 500).Error; err != nil {
		return 0, err
	}
	ids := make([]uint, 0, len(logs))
	for _, log := range logs {
		ids = append(ids, log.ID)
	}
	if err := tx.Where("id IN ?", ids).Delete(&models.PlatformAuditLog{}).Error; err != nil {
		return 0, err
	}
	return int64(len(logs)), nil
}

func archiveLoginLogs(tx *gorm.DB, cutoff time.Time) (int64, error) {
	var logs []models.PlatformLoginLog
	if err := tx.Where("logged_in_at < ?", cutoff).Find(&logs).Error; err != nil {
		return 0, err
	}
	if len(logs) == 0 {
		return 0, nil
	}
	archives := make([]models.PlatformLoginLogArchive, 0, len(logs))
	now := time.Now()
	for _, log := range logs {
		archives = append(archives, models.PlatformLoginLogArchive{
			ID:              log.ID,
			TraceID:         log.TraceID,
			UserID:          log.UserID,
			Username:        log.Username,
			RealName:        log.RealName,
			Role:            log.Role,
			RequestIP:       log.RequestIP,
			UserAgent:       log.UserAgent,
			RequestPath:     log.RequestPath,
			RequestMethod:   log.RequestMethod,
			StatusCode:      log.StatusCode,
			OperationStatus: log.OperationStatus,
			LoginType:       log.LoginType,
			ErrorMessage:    log.ErrorMessage,
			DurationMS:      log.DurationMS,
			LoggedInAt:      log.LoggedInAt,
			CreatedAt:       log.CreatedAt,
			ArchivedAt:      now,
		})
	}
	if err := tx.CreateInBatches(&archives, 500).Error; err != nil {
		return 0, err
	}
	ids := make([]uint, 0, len(logs))
	for _, log := range logs {
		ids = append(ids, log.ID)
	}
	if err := tx.Where("id IN ?", ids).Delete(&models.PlatformLoginLog{}).Error; err != nil {
		return 0, err
	}
	return int64(len(logs)), nil
}

func getAuditPageParams(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if value := strings.TrimSpace(c.Query("page")); value != "" {
		if parsed := parsePositiveInt(value); parsed > 0 {
			page = parsed
		}
	}
	if value := strings.TrimSpace(c.Query("page_size")); value != "" {
		if parsed := parsePositiveInt(value); parsed > 0 && parsed <= 200 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

func parsePositiveInt(value string) int {
	result := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0
		}
		result = result*10 + int(ch-'0')
	}
	return result
}

func applyAuditCommonFilters(c *gin.Context, db *gorm.DB, userField, ipField, pathField, statusField, timeField string) *gorm.DB {
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		keyword := "%" + username + "%"
		db = db.Where(fmt.Sprintf("(%s LIKE ? OR real_name LIKE ?)", userField), keyword, keyword)
	}
	if requestIP := strings.TrimSpace(c.Query("request_ip")); requestIP != "" {
		db = db.Where(ipField+" LIKE ?", "%"+requestIP+"%")
	}
	if requestPath := strings.TrimSpace(c.Query("request_path")); requestPath != "" {
		db = db.Where(pathField+" LIKE ?", "%"+requestPath+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		db = db.Where(statusField+" = ?", status)
	}
	start := strings.TrimSpace(c.Query("start"))
	end := strings.TrimSpace(c.Query("end"))
	if start != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", start); err == nil {
			db = db.Where(timeField+" >= ?", parsed)
		}
	}
	if end != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", end); err == nil {
			db = db.Where(timeField+" <= ?", parsed)
		}
	}
	return db
}

func appendArchiveAuditFilters(c *gin.Context, whereClauses []string, args []any) ([]string, []any) {
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		keyword := "%" + username + "%"
		whereClauses = append(whereClauses, "(username LIKE ? OR real_name LIKE ?)")
		args = append(args, keyword, keyword)
	}
	if requestIP := strings.TrimSpace(c.Query("request_ip")); requestIP != "" {
		whereClauses = append(whereClauses, "request_ip LIKE ?")
		args = append(args, "%"+requestIP+"%")
	}
	if requestPath := strings.TrimSpace(c.Query("request_path")); requestPath != "" {
		whereClauses = append(whereClauses, "request_path LIKE ?")
		args = append(args, "%"+requestPath+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		whereClauses = append(whereClauses, "operation_status = ?")
		args = append(args, status)
	}
	if start := strings.TrimSpace(c.Query("start")); start != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", start); err == nil {
			whereClauses = append(whereClauses, "occurred_at >= ?")
			args = append(args, parsed)
		}
	}
	if end := strings.TrimSpace(c.Query("end")); end != "" {
		if parsed, err := time.Parse("2006-01-02 15:04:05", end); err == nil {
			whereClauses = append(whereClauses, "occurred_at <= ?")
			args = append(args, parsed)
		}
	}
	return whereClauses, args
}

func formatAuditTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04:05")
}

func formatAuditStatus(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "success") {
		return "成功"
	}
	return "失败"
}

func displayAuditOperator(realName, username string) string {
	if strings.TrimSpace(realName) != "" {
		return realName
	}
	if strings.TrimSpace(username) != "" {
		return username
	}
	return "-"
}

func defaultAuditString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}