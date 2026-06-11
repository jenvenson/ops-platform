// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/auth"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/jenvenson/ops-platform/pkg/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	engine *ScanEngine
}

func NewHandler() *Handler {
	return &Handler{
		engine: NewScanEngine(),
	}
}

// RegisterRoutes 注册安全扫描路由（包级别函数）
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	// 初始化资产中心数据库
	go func() {
		if err := database.DB.AutoMigrate(
			&models.Asset{},
			&models.SecurityVulnerability{},
			&models.VulnTicket{},
			&models.SecurityAsset{},
			&FIMPolicy{},
			&FIMPolicyTarget{},
			&FIMWatchPath{},
			&FIMSnapshot{},
			&FIMSnapshotEntry{},
			&FIMDiffEvent{},
			&FIMAlert{},
			&models.FIMKnownHost{},
			&models.FIMKnownHostsHistory{},
			&models.FIMSSHConnectionLog{},
		); err != nil {
			println("[Asset] 数据库迁移失败:", err.Error())
		} else {
			println("[Asset] 资产中心、漏洞、工单和文件完整性巡检数据库迁移完成")
		}
	}()

	// 初始化漏洞知识库
	go func() {
		vulnDB := NewVulnDBService()
		if err := vulnDB.InitVulnDB(); err != nil {
			println("[VulnDB] 初始化失败:", err.Error())
		} else {
			println("[VulnDB] 漏洞知识库初始化完成")
		}
	}()

	h := NewHandler()
	startFIMScheduler()
	h.registerRoutes(r, cfg)
}

// registerRoutes 内部路由注册
func (h *Handler) registerRoutes(r *gin.Engine, cfg *config.Config) {
	security := r.Group("/api/security")
	security.Use(AuthMiddleware(cfg.JWT.Secret))
	{
		// 仪表盘统计
		security.GET("/statistics", h.GetStatistics())

		// 扫描任务管理
		security.GET("/tasks", h.GetTasks())
		security.POST("/tasks", h.CreateTask())
		security.POST("/auth-flow/generate", h.GenerateAuthFlow())
		security.GET("/tasks/:id", h.GetTask())
		security.DELETE("/tasks/:id", h.DeleteTask())

		// 任务控制
		security.POST("/tasks/:id/pause", h.PauseTask())
		security.POST("/tasks/:id/resume", h.ResumeTask())
		security.POST("/tasks/:id/cancel", h.CancelTask())

		// 任务结果
		security.GET("/tasks/:id/assets", h.GetTaskAssets())
		security.GET("/tasks/:id/targets", h.GetTaskTargets())
		security.GET("/tasks/:id/occurrences", h.GetTaskOccurrences())
		security.GET("/tasks/:id/evidences", h.GetTaskEvidences())
		security.GET("/tasks/:id/vulnerabilities", h.GetTaskVulnerabilities())

		// 漏洞管理
		security.GET("/vulnerabilities", h.GetVulnerabilities())
		security.GET("/vulnerabilities/:id", h.GetVulnerability())
		security.GET("/vulnerabilities/:id/detail", h.GetVulnerabilityDetail())
		security.PUT("/vulnerabilities/:id/status", h.UpdateVulnerabilityStatus())
		security.PUT("/vulnerabilities/:id/review", h.ReviewVulnerabilityCandidate())
		security.DELETE("/vulnerabilities/:id", h.DeleteVulnerability())

		// 导出报告
		security.GET("/tasks/:id/export", h.ExportReport())

		// 漏洞知识库
		security.GET("/vuln-db/list", h.GetVulnDBList())
		security.GET("/vuln-db/stats", h.GetVulnDBStats())
		security.GET("/vuln-db/search", h.SearchVulnDB())
		security.GET("/vuln-db/:cveId", h.GetVulnByCVE())
		security.POST("/vuln-db/import", h.ImportVulnDB())
		security.POST("/vuln-db/sync-nvd", h.SyncNVD())
		security.POST("/vuln-db/sync-nvd-full", h.SyncNFull())
		security.GET("/vuln-db/sync-tasks", h.GetSyncTasks())
		security.POST("/vuln-db/import-cnvd", h.ImportCNVD())
		security.POST("/vuln-db/import-cnnvd", h.ImportCNNVD())

		// 服务漏洞匹配
		security.POST("/vuln-db/match-service", h.MatchServiceVulns())

		// 资产中心
		security.GET("/assets", h.GetAssets())
		security.GET("/assets/:id", h.GetAsset())
		security.POST("/assets", h.CreateAsset())
		security.PUT("/assets/:id", h.UpdateAsset())
		security.DELETE("/assets/:id", h.DeleteAsset())
		security.GET("/assets/stats", h.GetAssetStats())

		// 漏洞工单
		security.GET("/tickets", h.GetTickets())
		security.GET("/tickets/:id", h.GetTicket())
		security.POST("/tickets", h.CreateTicket())
		security.PUT("/tickets/:id", h.UpdateTicket())
		security.DELETE("/tickets/:id", h.DeleteTicket())
		security.POST("/tickets/:id/assign", h.AssignTicket())
		security.POST("/tickets/:id/close", h.CloseTicket())

		h.registerFIMRoutes(security)
	}
}

// GetStatistics 获取安全统计信息
func (h *Handler) GetStatistics() gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats models.ScanStatistics

		// 任务统计
		database.DB.Model(&models.SecurityScanTask{}).Count(&stats.TotalTasks)
		database.DB.Model(&models.SecurityScanTask{}).Where("status = ?", "running").Count(&stats.RunningTasks)
		database.DB.Model(&models.SecurityScanTask{}).Where("status = ?", "completed").Count(&stats.CompletedTasks)

		// 资产统计
		database.DB.Model(&models.SecurityAsset{}).Count(&stats.TotalAssets)

		// 漏洞统计
		database.DB.Model(&models.SecurityVulnerability{}).Count(&stats.TotalVulnerabilities)
		database.DB.Model(&models.SecurityVulnerability{}).Where("severity = ?", "high").Count(&stats.HighRiskCount)
		database.DB.Model(&models.SecurityVulnerability{}).Where("severity = ?", "medium").Count(&stats.MediumRiskCount)
		database.DB.Model(&models.SecurityVulnerability{}).Where("severity = ?", "low").Count(&stats.LowRiskCount)

		c.JSON(http.StatusOK, stats)
	}
}

// GetTasks 获取扫描任务列表（支持分页）
func (h *Handler) GetTasks() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tasks []models.SecurityScanTask
		var total int64

		params := Paginate(c)
		offset := (params.Page - 1) * params.PageSize

		query := database.DB.Model(&models.SecurityScanTask{}).Order("created_at DESC")

		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if scanType := strings.TrimSpace(c.Query("scan_type")); scanType != "" {
			query = query.Where("scan_type = ?", scanType)
		}
		switch strings.TrimSpace(c.Query("task_group")) {
		case "vuln":
			query = query.Where("scan_type IN ?", []string{"web", "host-vuln", "all"})
		case "discovery":
			query = query.Where("scan_type IN ?", []string{"port", "host"})
		}

		// 获取总数
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tasks"})
			return
		}

		// 分页查询
		if err := query.Offset(offset).Limit(params.PageSize).Find(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tasks"})
			return
		}

		totalPages := int(total) / params.PageSize
		if int(total)%params.PageSize > 0 {
			totalPages++
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Total:      int(total),
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
			Data:       tasks,
		})
	}
}

// CreateTaskRequest 创建任务请求
// PageParams 分页请求参数
type PageParams struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

type SecurityScanRunDetail struct {
	ID              uint                   `json:"id"`
	TaskID          uint                   `json:"task_id"`
	Status          string                 `json:"status"`
	Progress        int                    `json:"progress"`
	TotalTargets    int                    `json:"total_targets"`
	ScannedTargets  int                    `json:"scanned_targets"`
	Message         string                 `json:"message"`
	HighRisk        int                    `json:"high_risk"`
	MediumRisk      int                    `json:"medium_risk"`
	LowRisk         int                    `json:"low_risk"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Phase           string                 `json:"phase,omitempty"`
	ConfigSnapshot  map[string]interface{} `json:"config_snapshot,omitempty"`
	TargetSnapshot  map[string]interface{} `json:"target_snapshot,omitempty"`
	SummarySnapshot map[string]interface{} `json:"summary_snapshot,omitempty"`
}

type SecurityScanTaskDetail struct {
	models.SecurityScanTask
	CurrentRun *SecurityScanRunDetail `json:"current_run,omitempty"`
	LatestRun  *SecurityScanRunDetail `json:"latest_run,omitempty"`
}

type SecurityScanTargetDetail struct {
	ID               uint                   `json:"id"`
	RunID            uint                   `json:"run_id"`
	TaskID           uint                   `json:"task_id"`
	ParentTargetID   *uint                  `json:"parent_target_id,omitempty"`
	TargetKind       string                 `json:"target_kind"`
	NormalizedTarget string                 `json:"normalized_target"`
	TargetURL        string                 `json:"target_url"`
	Host             string                 `json:"host"`
	Port             *int                   `json:"port,omitempty"`
	Scheme           string                 `json:"scheme"`
	Path             string                 `json:"path"`
	ServiceName      string                 `json:"service_name"`
	ProductName      string                 `json:"product_name"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"`
	DiscoverySource  string                 `json:"discovery_source"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type SecurityScanEvidenceDetail struct {
	ID              uint                   `json:"id"`
	RunID           uint                   `json:"run_id"`
	TaskID          uint                   `json:"task_id"`
	TargetID        *uint                  `json:"target_id,omitempty"`
	EvidenceType    string                 `json:"evidence_type"`
	SourceEngine    string                 `json:"source_engine"`
	Digest          string                 `json:"digest"`
	RequestExcerpt  string                 `json:"request_excerpt,omitempty"`
	ResponseExcerpt string                 `json:"response_excerpt,omitempty"`
	PayloadExcerpt  string                 `json:"payload_excerpt,omitempty"`
	StorageRef      string                 `json:"storage_ref,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type SecurityScanFindingOccurrenceDetail struct {
	ID                    uint                      `json:"id"`
	RunID                 uint                      `json:"run_id"`
	TaskID                uint                      `json:"task_id"`
	TargetID              *uint                     `json:"target_id,omitempty"`
	LegacyVulnerabilityID *uint                     `json:"legacy_vulnerability_id,omitempty"`
	FindingKey            string                    `json:"finding_key"`
	FindingFamily         string                    `json:"finding_family"`
	FindingSource         string                    `json:"finding_source"`
	Severity              string                    `json:"severity"`
	Confidence            string                    `json:"confidence"`
	MatchMode             string                    `json:"match_mode"`
	PrimaryCVEID          string                    `json:"primary_cve_id"`
	Title                 string                    `json:"title"`
	Status                string                    `json:"status"`
	VerificationStatus    string                    `json:"verification_status"`
	EvidenceCount         int                       `json:"evidence_count"`
	FirstSeenAt           *time.Time                `json:"first_seen_at,omitempty"`
	LastSeenAt            *time.Time                `json:"last_seen_at,omitempty"`
	Metadata              map[string]interface{}    `json:"metadata,omitempty"`
	EvidenceID            *uint                     `json:"evidence_id,omitempty"`
	Target                *SecurityScanTargetDetail `json:"target,omitempty"`
}

type SecurityVulnerabilityDetailResponse struct {
	Vulnerability models.SecurityVulnerability          `json:"vulnerability"`
	Occurrences   []SecurityScanFindingOccurrenceDetail `json:"occurrences"`
	Evidences     []SecurityScanEvidenceDetail          `json:"evidences"`
}

// Paginate 分页辅助函数
func Paginate(c *gin.Context) *PageParams {
	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	return &PageParams{Page: page, PageSize: pageSize}
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
	Data       interface{} `json:"data"`
}

type CreateTaskRequest struct {
	Name             string          `json:"name" binding:"required"`
	TargetType       string          `json:"target_type" binding:"required"` // ip_list 或 url (web扫描)
	Target           string          `json:"target" binding:"required"`
	ScanType         string          `json:"scan_type"`        // port, host-vuln, web, all (默认 port)
	WebScanOptions   string          `json:"web_scan_options"` // comma-separated: sql-injection,xss,ssrf,csrf,rce,etc.
	WebScanProfile   string          `json:"web_scan_profile"`
	AuthMode         string          `json:"auth_mode"` // none, cookie, bearer, basic, login-form, login-token, advanced
	DiscoveryMode    string          `json:"discovery_mode"`
	AuthCredential   string          `json:"auth_credential"` // 认证凭据
	AuthHeader       string          `json:"auth_header"`     // 自定义请求头名称
	AuthFlow         json.RawMessage `json:"auth_flow"`
	LoginURL         string          `json:"login_url"`
	LoginMethod      string          `json:"login_method"`
	LoginContentType string          `json:"login_content_type"`
	Username         string          `json:"username"`
	Password         string          `json:"password"`
	UsernameField    string          `json:"username_field"`
	PasswordField    string          `json:"password_field"`
	TokenField       string          `json:"token_field"`
}

func (h *Handler) GenerateAuthFlow() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GenerateAuthFlowRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		result, err := generateAuthFlow(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// WebScanConfig Web 扫描配置
type WebScanConfig struct {
	Options                []string // 扫描选项
	ScanProfile            string
	AuthMode               string // 认证模式
	Credential             string // 认证凭据
	AuthHeader             string // 自定义请求头
	AuthFlowRaw            json.RawMessage
	AuthFlow               *AuthFlowConfig
	DiscoveryMode          string
	DiscoveryMaxDepth      int
	DiscoveryMaxURLs       int
	VerificationMaxTargets int
	LoginURL               string
	LoginMethod            string
	LoginContentType       string
	Username               string
	Password               string
	UsernameField          string
	PasswordField          string
	TokenField             string
}

func normalizeWebScanProfile(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "deep":
		return "deep"
	default:
		return "standard"
	}
}

func applyWebScanProfileDefaults(config *WebScanConfig) {
	if config == nil {
		return
	}
	config.ScanProfile = normalizeWebScanProfile(config.ScanProfile)
	switch config.ScanProfile {
	case "deep":
		if config.DiscoveryMaxURLs < 40 {
			config.DiscoveryMaxURLs = 40
		}
		if config.VerificationMaxTargets < 12 {
			config.VerificationMaxTargets = 12
		}
	default:
		if config.DiscoveryMaxURLs <= 0 {
			config.DiscoveryMaxURLs = 25
		}
		if config.VerificationMaxTargets <= 0 {
			config.VerificationMaxTargets = 8
		}
	}
}

func riskCategoryForVulnerability(vuln *models.SecurityVulnerability) string {
	if vuln == nil {
		return ""
	}
	if strings.TrimSpace(vuln.FindingFamily) == "inventory" || strings.TrimSpace(vuln.ScanMethod) == "服务识别" || strings.Contains(strings.TrimSpace(vuln.VulnType), "资产识别") {
		return "inventory"
	}
	if strings.TrimSpace(vuln.PrimaryCVEID) != "" || strings.TrimSpace(vuln.CVEID) != "" {
		return "cve_risk"
	}
	if strings.Contains(strings.TrimSpace(vuln.VulnType), "配置") {
		return "config_risk"
	}
	return "generic_risk"
}

func decorateVulnerabilityRiskCategory(vuln *models.SecurityVulnerability) {
	if vuln == nil {
		return
	}
	decorateVulnerabilityMetadata(vuln, NewVulnDBService())
	vuln.RiskCategory = riskCategoryForVulnerability(vuln)
}

func decorateVulnerabilityListRiskCategory(vulns []models.SecurityVulnerability) {
	decorateVulnerabilityListMetadata(vulns)
	for i := range vulns {
		vulns[i].RiskCategory = riskCategoryForVulnerability(&vulns[i])
	}
}

func applyRiskCategoryFilter(query *gorm.DB, riskCategory string) *gorm.DB {
	switch strings.TrimSpace(riskCategory) {
	case "inventory":
		return query.Where("scan_method = ? OR vuln_type LIKE ?", "服务识别", "%资产识别%")
	case "cve_risk":
		return query.Where("cve_id IS NOT NULL AND cve_id <> ''")
	case "config_risk":
		return query.Where("vuln_type LIKE ?", "%配置%")
	case "generic_risk":
		return query.Where("(scan_method IS NULL OR scan_method <> ?) AND (vuln_type IS NULL OR vuln_type NOT LIKE ?) AND (cve_id IS NULL OR cve_id = '') AND (vuln_type IS NULL OR vuln_type NOT LIKE ?)", "服务识别", "%资产识别%", "%配置%")
	default:
		return query
	}
}

func applyFindingFamilyFilter(query *gorm.DB, findingFamily string) *gorm.DB {
	switch strings.TrimSpace(findingFamily) {
	case "inventory":
		return query.Where(
			"finding_family = ? OR ((finding_family IS NULL OR finding_family = '') AND (scan_method = ? OR vuln_type LIKE ?))",
			"inventory",
			"服务识别",
			"%资产识别%",
		)
	case "vulnerability":
		return query.Where(
			"finding_family = ? OR ((finding_family IS NULL OR finding_family = '') AND (scan_method IS NULL OR scan_method <> ?) AND (vuln_type IS NULL OR vuln_type NOT LIKE ?))",
			"vulnerability",
			"服务识别",
			"%资产识别%",
		)
	default:
		return query
	}
}

func applyConfidenceFilter(query *gorm.DB, confidence string) *gorm.DB {
	confidence = strings.TrimSpace(confidence)
	if confidence == "" {
		return query
	}
	return query.Where("confidence = ?", confidence)
}

func applyFindingSourceFilter(query *gorm.DB, findingSource string) *gorm.DB {
	findingSource = strings.TrimSpace(findingSource)
	if findingSource == "" {
		return query
	}
	return query.Where("finding_source = ?", findingSource)
}

func applyMatchModeFilter(query *gorm.DB, matchMode string) *gorm.DB {
	matchMode = strings.TrimSpace(matchMode)
	if matchMode == "" {
		return query
	}
	return query.Where("match_mode = ?", matchMode)
}

func applyHasKnowledgeFilter(query *gorm.DB, hasKnowledge string) *gorm.DB {
	switch strings.ToLower(strings.TrimSpace(hasKnowledge)) {
	case "true", "1", "yes":
		return query.Where(
			"(vuln_db_id IS NOT NULL AND vuln_db_id > 0) OR " +
				"(primary_cve_id IS NOT NULL AND primary_cve_id <> '') OR " +
				"(cve_id IS NOT NULL AND cve_id <> '') OR " +
				"(cnvd_id IS NOT NULL AND cnvd_id <> '') OR " +
				"(cnnvd_id IS NOT NULL AND cnnvd_id <> '') OR " +
				"(cncve_id IS NOT NULL AND cncve_id <> '')",
		)
	case "false", "0", "no":
		return query.Where(
			"(vuln_db_id IS NULL OR vuln_db_id = 0) AND " +
				"(primary_cve_id IS NULL OR primary_cve_id = '') AND " +
				"(cve_id IS NULL OR cve_id = '') AND " +
				"(cnvd_id IS NULL OR cnvd_id = '') AND " +
				"(cnnvd_id IS NULL OR cnnvd_id = '') AND " +
				"(cncve_id IS NULL OR cncve_id = '')",
		)
	default:
		return query
	}
}

func applyVulnerabilityQueryFilters(query *gorm.DB, c *gin.Context) *gorm.DB {
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if riskCategory := c.Query("risk_category"); riskCategory != "" {
		query = applyRiskCategoryFilter(query, riskCategory)
	}
	if findingSource := c.Query("finding_source"); findingSource != "" {
		query = applyFindingSourceFilter(query, findingSource)
	}
	if findingFamily := c.Query("finding_family"); findingFamily != "" {
		query = applyFindingFamilyFilter(query, findingFamily)
	}
	if confidence := c.Query("confidence"); confidence != "" {
		query = applyConfidenceFilter(query, confidence)
	}
	if matchMode := c.Query("match_mode"); matchMode != "" {
		query = applyMatchModeFilter(query, matchMode)
	}
	if hasKnowledge := c.Query("has_knowledge"); hasKnowledge != "" {
		query = applyHasKnowledgeFilter(query, hasKnowledge)
	}
	return query
}

// CreateTask 创建扫描任务
func (h *Handler) CreateTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 设置默认值
		scanType := req.ScanType
		if scanType == "" {
			scanType = "port" // 默认端口扫描
		}

		// 兼容旧的 scan_type 值
		if scanType == "host" {
			scanType = "port" // 旧值映射为新值
		}

		// 验证 target_type 与 scan_type 的组合
		switch scanType {
		case "web":
			// Web 扫描：只支持登录后 URL 扫描
			if req.TargetType != "url" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "authenticated web scan requires target_type 'url'"})
				return
			}
			targets := strings.Split(req.Target, ",")
			for _, target := range targets {
				target = strings.TrimSpace(target)
				if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
					c.JSON(http.StatusBadRequest, gin.H{"error": "web scan target must be http:// or https:// URL"})
					return
				}
			}
		case "port", "host-vuln":
			// 端口扫描/主机漏洞扫描：仅支持 IP 列表
			if req.TargetType != "ip_list" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "scan requires target_type 'ip_list'"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "scan_type must be 'port', 'host-vuln' or 'web'"})
			return
		}

		if req.TargetType == "ip_list" {
			for _, item := range strings.Split(req.Target, ",") {
				ip := strings.TrimSpace(item)
				if ip == "" {
					continue
				}
				if net.ParseIP(ip) == nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ip_list target, expected comma-separated IP addresses"})
					return
				}
			}
		}

		if scanType == "web" && (strings.TrimSpace(req.AuthMode) == "" || req.AuthMode == "none") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "web scan requires login auth; anonymous scan is disabled"})
			return
		}

		if scanType == "web" && req.AuthMode == "advanced" && len(req.AuthFlow) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "advanced auth requires auth_flow"})
			return
		}

		if scanType == "web" && (req.AuthMode == "login-form" || req.AuthMode == "login-token") {
			if strings.TrimSpace(req.LoginURL) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "login auth requires login_url"})
				return
			}
			if strings.TrimSpace(req.Username) == "" || req.Password == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "login auth requires username and password"})
				return
			}
		}

		var webConfig *WebScanConfig
		if scanType == "web" {
			// 解析扫描选项
			var options []string
			if req.WebScanOptions != "" {
				options = strings.Split(req.WebScanOptions, ",")
				for i := range options {
					options[i] = strings.TrimSpace(options[i])
				}
			}
			webConfig = &WebScanConfig{
				Options:                options,
				ScanProfile:            req.WebScanProfile,
				AuthMode:               req.AuthMode,
				Credential:             req.AuthCredential,
				AuthHeader:             req.AuthHeader,
				AuthFlowRaw:            req.AuthFlow,
				DiscoveryMode:          "browser",
				DiscoveryMaxDepth:      1,
				DiscoveryMaxURLs:       25,
				VerificationMaxTargets: 8,
				LoginURL:               req.LoginURL,
				LoginMethod:            req.LoginMethod,
				LoginContentType:       req.LoginContentType,
				Username:               req.Username,
				Password:               req.Password,
				UsernameField:          req.UsernameField,
				PasswordField:          req.PasswordField,
				TokenField:             req.TokenField,
			}
			if strings.TrimSpace(req.DiscoveryMode) != "" {
				webConfig.DiscoveryMode = strings.ToLower(strings.TrimSpace(req.DiscoveryMode))
			}
			applyWebScanProfileDefaults(webConfig)
			authFlow, err := parseAuthFlow(req.AuthFlow)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid auth_flow", "details": err.Error()})
				return
			}
			webConfig.AuthFlow = authFlow
		}

		userID := c.GetUint("user_id")

		task := models.SecurityScanTask{
			Name:       req.Name,
			TargetType: req.TargetType,
			Target:     req.Target,
			ScanType:   scanType,
			Status:     "pending",
			CreatedBy:  userID,
		}

		if err := database.DB.Create(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task", "details": err.Error()})
			return
		}

		// 异步执行扫描
		go AsyncScan(task.ID, req.Target, req.TargetType, scanType, webConfig)

		c.JSON(http.StatusOK, task)
	}
}

type securityScanRunDetailRow struct {
	ID              uint       `gorm:"column:id"`
	TaskID          uint       `gorm:"column:task_id"`
	Status          string     `gorm:"column:status"`
	Progress        int        `gorm:"column:progress"`
	TotalTargets    int        `gorm:"column:total_targets"`
	ScannedTargets  int        `gorm:"column:scanned_targets"`
	Message         string     `gorm:"column:message"`
	HighRisk        int        `gorm:"column:high_risk"`
	MediumRisk      int        `gorm:"column:medium_risk"`
	LowRisk         int        `gorm:"column:low_risk"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	CompletedAt     *time.Time `gorm:"column:completed_at"`
	Phase           string     `gorm:"column:phase"`
	ConfigSnapshot  *string    `gorm:"column:config_snapshot"`
	TargetSnapshot  *string    `gorm:"column:target_snapshot"`
	SummarySnapshot *string    `gorm:"column:summary_snapshot"`
}

func decodeScanSnapshot(raw *string) map[string]interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &payload); err != nil {
		return nil
	}
	return payload
}

func loadSecurityScanRunDetail(tx *gorm.DB, runID uint, includeSnapshots bool) (*SecurityScanRunDetail, error) {
	if tx == nil || runID == 0 {
		return nil, nil
	}

	selectFields := []string{
		"id",
		"task_id",
		"status",
		"progress",
		"total_targets",
		"scanned_targets",
		"message",
		"high_risk",
		"medium_risk",
		"low_risk",
		"started_at",
		"completed_at",
	}
	if includeSnapshots {
		selectFields = append(selectFields, "phase", "config_snapshot", "target_snapshot", "summary_snapshot")
	}

	var row securityScanRunDetailRow
	if err := tx.Table("security_scan_runs").Select(strings.Join(selectFields, ", ")).Where("id = ?", runID).Take(&row).Error; err != nil {
		return nil, err
	}

	return &SecurityScanRunDetail{
		ID:              row.ID,
		TaskID:          row.TaskID,
		Status:          row.Status,
		Progress:        row.Progress,
		TotalTargets:    row.TotalTargets,
		ScannedTargets:  row.ScannedTargets,
		Message:         row.Message,
		HighRisk:        row.HighRisk,
		MediumRisk:      row.MediumRisk,
		LowRisk:         row.LowRisk,
		StartedAt:       row.StartedAt,
		CompletedAt:     row.CompletedAt,
		Phase:           row.Phase,
		ConfigSnapshot:  decodeScanSnapshot(row.ConfigSnapshot),
		TargetSnapshot:  decodeScanSnapshot(row.TargetSnapshot),
		SummarySnapshot: decodeScanSnapshot(row.SummarySnapshot),
	}, nil
}

func formatSecurityScanTargetURL(target models.SecurityScanTarget) string {
	if strings.TrimSpace(target.Scheme) == "" || strings.TrimSpace(target.Host) == "" {
		return strings.TrimSpace(target.NormalizedTarget)
	}

	base := strings.ToLower(strings.TrimSpace(target.Scheme)) + "://" + strings.TrimSpace(target.Host)
	if target.Port != nil && *target.Port > 0 {
		defaultPort := 0
		switch strings.ToLower(strings.TrimSpace(target.Scheme)) {
		case "http":
			defaultPort = 80
		case "https":
			defaultPort = 443
		}
		if defaultPort == 0 || *target.Port != defaultPort {
			base += ":" + strconv.Itoa(*target.Port)
		}
	}

	path := strings.TrimSpace(target.Path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func buildSecurityScanTargetDetail(target models.SecurityScanTarget) SecurityScanTargetDetail {
	return SecurityScanTargetDetail{
		ID:               target.ID,
		RunID:            target.RunID,
		TaskID:           target.TaskID,
		ParentTargetID:   target.ParentTargetID,
		TargetKind:       target.TargetKind,
		NormalizedTarget: target.NormalizedTarget,
		TargetURL:        formatSecurityScanTargetURL(target),
		Host:             target.Host,
		Port:             target.Port,
		Scheme:           target.Scheme,
		Path:             target.Path,
		ServiceName:      target.ServiceName,
		ProductName:      target.ProductName,
		Version:          target.Version,
		Status:           target.Status,
		DiscoverySource:  target.DiscoverySource,
		StartedAt:        target.StartedAt,
		CompletedAt:      target.CompletedAt,
		Metadata:         decodeScanSnapshot(target.MetadataJSON),
	}
}

func buildSecurityScanEvidenceDetail(evidence models.SecurityScanEvidence) SecurityScanEvidenceDetail {
	return SecurityScanEvidenceDetail{
		ID:              evidence.ID,
		RunID:           evidence.RunID,
		TaskID:          evidence.TaskID,
		TargetID:        evidence.TargetID,
		EvidenceType:    evidence.EvidenceType,
		SourceEngine:    evidence.SourceEngine,
		Digest:          evidence.Digest,
		RequestExcerpt:  evidence.RequestExcerpt,
		ResponseExcerpt: evidence.ResponseExcerpt,
		PayloadExcerpt:  evidence.PayloadExcerpt,
		StorageRef:      evidence.StorageRef,
		CreatedAt:       evidence.CreatedAt,
		Metadata:        decodeScanSnapshot(evidence.MetadataJSON),
	}
}

func metadataUintValue(metadata map[string]interface{}, key string) *uint {
	if metadata == nil {
		return nil
	}
	value, exists := metadata[key]
	if !exists || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case float64:
		if typed < 0 {
			return nil
		}
		result := uint(typed)
		return &result
	case int:
		if typed < 0 {
			return nil
		}
		result := uint(typed)
		return &result
	case uint:
		result := typed
		return &result
	case string:
		parsed, err := strconv.ParseUint(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return nil
		}
		result := uint(parsed)
		return &result
	default:
		return nil
	}
}

func relatedVulnerabilityIDs(vuln models.SecurityVulnerability) []uint {
	seen := map[uint]struct{}{}
	result := make([]uint, 0, 3)
	appendID := func(value uint) {
		if value == 0 {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	appendID(vuln.ID)
	if vuln.SourceVulnID != nil {
		appendID(*vuln.SourceVulnID)
	}
	if vuln.ConfirmedVulnID != nil {
		appendID(*vuln.ConfirmedVulnID)
	}
	return result
}

func activeRunIDForTask(task models.SecurityScanTask) uint {
	if task.CurrentRunID != nil {
		return *task.CurrentRunID
	}
	if task.LatestRunID != nil {
		return *task.LatestRunID
	}
	return 0
}

func loadTaskTargetDetails(tx *gorm.DB, task models.SecurityScanTask) ([]SecurityScanTargetDetail, error) {
	support := getScanPhase1SchemaSupport(tx)
	if !support.targetsTable {
		return []SecurityScanTargetDetail{}, nil
	}
	runID := activeRunIDForTask(task)
	if runID == 0 {
		return []SecurityScanTargetDetail{}, nil
	}

	var targets []models.SecurityScanTarget
	if err := tx.
		Where("task_id = ? AND run_id = ?", task.ID, runID).
		Order("COALESCE(parent_target_id, 0) ASC, created_at ASC, id ASC").
		Find(&targets).Error; err != nil {
		return nil, err
	}

	result := make([]SecurityScanTargetDetail, 0, len(targets))
	for _, target := range targets {
		result = append(result, buildSecurityScanTargetDetail(target))
	}
	return result, nil
}

func loadTaskEvidenceDetails(tx *gorm.DB, task models.SecurityScanTask) ([]SecurityScanEvidenceDetail, error) {
	support := getScanPhase1SchemaSupport(tx)
	if !support.evidencesTable {
		return []SecurityScanEvidenceDetail{}, nil
	}
	runID := activeRunIDForTask(task)
	if runID == 0 {
		return []SecurityScanEvidenceDetail{}, nil
	}

	var evidences []models.SecurityScanEvidence
	if err := tx.
		Where("task_id = ? AND run_id = ?", task.ID, runID).
		Order("created_at DESC, id DESC").
		Find(&evidences).Error; err != nil {
		return nil, err
	}

	result := make([]SecurityScanEvidenceDetail, 0, len(evidences))
	for _, evidence := range evidences {
		result = append(result, buildSecurityScanEvidenceDetail(evidence))
	}
	return result, nil
}

func loadTaskOccurrenceDetails(tx *gorm.DB, task models.SecurityScanTask) ([]SecurityScanFindingOccurrenceDetail, error) {
	support := getScanPhase1SchemaSupport(tx)
	if !support.occurrencesTable {
		return []SecurityScanFindingOccurrenceDetail{}, nil
	}
	runID := activeRunIDForTask(task)
	if runID == 0 {
		return []SecurityScanFindingOccurrenceDetail{}, nil
	}

	var occurrences []models.SecurityScanFindingOccurrence
	if err := tx.
		Where("task_id = ? AND run_id = ?", task.ID, runID).
		Order("COALESCE(last_seen_at, created_at) DESC, id DESC").
		Find(&occurrences).Error; err != nil {
		return nil, err
	}

	targetDetails, err := loadTaskTargetDetails(tx, task)
	if err != nil {
		return nil, err
	}
	targetMap := make(map[uint]SecurityScanTargetDetail, len(targetDetails))
	for _, detail := range targetDetails {
		targetMap[detail.ID] = detail
	}

	result := make([]SecurityScanFindingOccurrenceDetail, 0, len(occurrences))
	for _, occurrence := range occurrences {
		metadata := decodeScanSnapshot(occurrence.MetadataJSON)
		evidenceID := metadataUintValue(metadata, "evidence_id")
		detail := SecurityScanFindingOccurrenceDetail{
			ID:                    occurrence.ID,
			RunID:                 occurrence.RunID,
			TaskID:                occurrence.TaskID,
			TargetID:              occurrence.TargetID,
			LegacyVulnerabilityID: occurrence.LegacyVulnerabilityID,
			FindingKey:            occurrence.FindingKey,
			FindingFamily:         occurrence.FindingFamily,
			FindingSource:         occurrence.FindingSource,
			Severity:              occurrence.Severity,
			Confidence:            occurrence.Confidence,
			MatchMode:             occurrence.MatchMode,
			PrimaryCVEID:          occurrence.PrimaryCVEID,
			Title:                 occurrence.Title,
			Status:                occurrence.Status,
			VerificationStatus:    occurrence.VerificationStatus,
			EvidenceCount:         occurrence.EvidenceCount,
			FirstSeenAt:           occurrence.FirstSeenAt,
			LastSeenAt:            occurrence.LastSeenAt,
			Metadata:              metadata,
			EvidenceID:            evidenceID,
		}
		if occurrence.TargetID != nil {
			if target, exists := targetMap[*occurrence.TargetID]; exists {
				targetCopy := target
				detail.Target = &targetCopy
			}
		}
		result = append(result, detail)
	}
	return result, nil
}

// GetTask 获取任务详情
func (h *Handler) GetTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		detail := SecurityScanTaskDetail{SecurityScanTask: task}
		support := getScanPhase1SchemaSupport(database.DB)
		includeSnapshots := support.runExtensions

		if task.CurrentRunID != nil {
			run, err := loadSecurityScanRunDetail(database.DB, *task.CurrentRunID, includeSnapshots)
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load current run"})
				return
			}
			detail.CurrentRun = run
		}
		if task.LatestRunID != nil {
			run, err := loadSecurityScanRunDetail(database.DB, *task.LatestRunID, includeSnapshots)
			if err != nil && err != gorm.ErrRecordNotFound {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load latest run"})
				return
			}
			detail.LatestRun = run
		}

		c.JSON(http.StatusOK, detail)
	}
}

// DeleteTask 删除任务
func (h *Handler) DeleteTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		tx := database.DB.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
			return
		}

		if err := deleteTaskVulnerabilities(tx, uint(id)); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task vulnerabilities"})
			return
		}
		if err := tx.Where("task_id = ?", id).Delete(&models.SecurityAsset{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task assets"})
			return
		}
		if err := deleteTaskRuns(tx, uint(id)); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task runs"})
			return
		}
		if err := tx.Delete(&models.SecurityScanTask{}, id).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
			return
		}
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
	}
}

// PauseTask 暂停任务
func (h *Handler) PauseTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		// 只有运行中的任务可以暂停
		if task.Status != models.TaskStatusRunning {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only running tasks can be paused"})
			return
		}

		if err := UpdateTaskAndCurrentRun(task.ID, map[string]interface{}{
			"status":  models.TaskStatusPaused,
			"message": "已收到暂停请求，将在当前阶段结束后停止",
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pause task"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message":        "pause requested; takes effect after the current stage finishes",
			"effective_mode": "stage_boundary",
		})
	}
}

// ResumeTask 恢复任务
func (h *Handler) ResumeTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		// 只有暂停的任务可以恢复
		if task.Status != models.TaskStatusPaused {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only paused tasks can be resumed"})
			return
		}

		c.JSON(http.StatusConflict, gin.H{
			"error": "resume is not supported reliably for current scan tasks; create a new task instead",
		})
	}
}

// CancelTask 取消任务
func (h *Handler) CancelTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		// 只有 pending 或 running 或 paused 状态的任务可以取消
		if task.Status != models.TaskStatusPending && task.Status != models.TaskStatusRunning && task.Status != models.TaskStatusPaused {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only pending, running or paused tasks can be cancelled"})
			return
		}

		completedAt := time.Now()
		if err := UpdateTaskAndCurrentRun(task.ID, map[string]interface{}{
			"status":       models.TaskStatusCancelled,
			"message":      "已收到取消请求，将在当前阶段结束后停止",
			"completed_at": &completedAt,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel task"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message":        "cancel requested; takes effect after the current stage finishes",
			"effective_mode": "stage_boundary",
		})
	}
}

// GetTaskAssets 获取任务的资产列表
func (h *Handler) GetTaskAssets() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var assets []models.SecurityAsset
		if err := database.DB.Where("task_id = ?", id).Order("ip ASC, port ASC").Find(&assets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get assets"})
			return
		}

		c.JSON(http.StatusOK, assets)
	}
}

// GetTaskTargets 获取任务的目标树
func (h *Handler) GetTaskTargets() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.Select("id", "current_run_id", "latest_run_id").First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		targets, err := loadTaskTargetDetails(database.DB, task)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get targets"})
			return
		}
		c.JSON(http.StatusOK, targets)
	}
}

// GetTaskOccurrences 获取任务的命中记录
func (h *Handler) GetTaskOccurrences() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.Select("id", "current_run_id", "latest_run_id").First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		occurrences, err := loadTaskOccurrenceDetails(database.DB, task)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get occurrences"})
			return
		}
		c.JSON(http.StatusOK, occurrences)
	}
}

// GetTaskEvidences 获取任务的证据列表
func (h *Handler) GetTaskEvidences() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var task models.SecurityScanTask
		if err := database.DB.Select("id", "current_run_id", "latest_run_id").First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		evidences, err := loadTaskEvidenceDetails(database.DB, task)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get evidences"})
			return
		}
		c.JSON(http.StatusOK, evidences)
	}
}

// GetTaskVulnerabilities 获取任务的漏洞列表
func (h *Handler) GetTaskVulnerabilities() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		var vulns []models.SecurityVulnerability
		query := taskVulnerabilityScope(database.DB, uint(id)).Order("severity DESC, created_at DESC")
		query = applyVulnerabilityQueryFilters(query, c)

		if err := query.Find(&vulns).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vulnerabilities"})
			return
		}

		decorateVulnerabilityListRiskCategory(vulns)
		c.JSON(http.StatusOK, vulns)
	}
}

// GetVulnerabilities 获取漏洞列表（支持分页）
func (h *Handler) GetVulnerabilities() gin.HandlerFunc {
	return func(c *gin.Context) {
		var vulns []models.SecurityVulnerability
		var total int64

		params := Paginate(c)
		offset := (params.Page - 1) * params.PageSize

		query := database.DB.Model(&models.SecurityVulnerability{}).Order("created_at DESC")

		if severity := c.Query("severity"); severity != "" {
			query = query.Where("severity = ?", severity)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if ip := c.Query("ip"); ip != "" {
			query = query.Where("ip LIKE ?", "%"+ip+"%")
		}
		query = applyVulnerabilityQueryFilters(query, c)

		// 获取总数
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count vulnerabilities"})
			return
		}

		// 分页查询
		if err := query.Offset(offset).Limit(params.PageSize).Find(&vulns).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vulnerabilities"})
			return
		}
		decorateVulnerabilityListRiskCategory(vulns)

		totalPages := int(total) / params.PageSize
		if int(total)%params.PageSize > 0 {
			totalPages++
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Total:      int(total),
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
			Data:       vulns,
		})
	}
}

// GetVulnerability 获取漏洞详情
func (h *Handler) GetVulnerability() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vulnerability id"})
			return
		}

		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}

		decorateVulnerabilityRiskCategory(&vuln)
		c.JSON(http.StatusOK, vuln)
	}
}

// GetVulnerabilityDetail 获取漏洞详情及 occurrence/evidence
func (h *Handler) GetVulnerabilityDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vulnerability id"})
			return
		}

		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}
		decorateVulnerabilityRiskCategory(&vuln)

		response := SecurityVulnerabilityDetailResponse{
			Vulnerability: vuln,
			Occurrences:   []SecurityScanFindingOccurrenceDetail{},
			Evidences:     []SecurityScanEvidenceDetail{},
		}

		support := getScanPhase1SchemaSupport(database.DB)
		if !support.occurrencesTable {
			c.JSON(http.StatusOK, response)
			return
		}

		relatedIDs := relatedVulnerabilityIDs(vuln)
		if len(relatedIDs) == 0 {
			c.JSON(http.StatusOK, response)
			return
		}

		var occurrences []models.SecurityScanFindingOccurrence
		if err := database.DB.
			Where("legacy_vulnerability_id IN ?", relatedIDs).
			Order("COALESCE(last_seen_at, created_at) DESC, id DESC").
			Find(&occurrences).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vulnerability occurrences"})
			return
		}
		if len(occurrences) == 0 {
			c.JSON(http.StatusOK, response)
			return
		}

		targetIDs := make([]uint, 0, len(occurrences))
		evidenceIDs := make([]uint, 0, len(occurrences))
		targetSeen := map[uint]struct{}{}
		evidenceSeen := map[uint]struct{}{}
		occurrenceDetails := make([]SecurityScanFindingOccurrenceDetail, 0, len(occurrences))

		for _, occurrence := range occurrences {
			metadata := decodeScanSnapshot(occurrence.MetadataJSON)
			evidenceID := metadataUintValue(metadata, "evidence_id")
			if evidenceID != nil {
				if _, exists := evidenceSeen[*evidenceID]; !exists {
					evidenceSeen[*evidenceID] = struct{}{}
					evidenceIDs = append(evidenceIDs, *evidenceID)
				}
			}
			if occurrence.TargetID != nil {
				if _, exists := targetSeen[*occurrence.TargetID]; !exists {
					targetSeen[*occurrence.TargetID] = struct{}{}
					targetIDs = append(targetIDs, *occurrence.TargetID)
				}
			}
			occurrenceDetails = append(occurrenceDetails, SecurityScanFindingOccurrenceDetail{
				ID:                    occurrence.ID,
				RunID:                 occurrence.RunID,
				TaskID:                occurrence.TaskID,
				TargetID:              occurrence.TargetID,
				LegacyVulnerabilityID: occurrence.LegacyVulnerabilityID,
				FindingKey:            occurrence.FindingKey,
				FindingFamily:         occurrence.FindingFamily,
				FindingSource:         occurrence.FindingSource,
				Severity:              occurrence.Severity,
				Confidence:            occurrence.Confidence,
				MatchMode:             occurrence.MatchMode,
				PrimaryCVEID:          occurrence.PrimaryCVEID,
				Title:                 occurrence.Title,
				Status:                occurrence.Status,
				VerificationStatus:    occurrence.VerificationStatus,
				EvidenceCount:         occurrence.EvidenceCount,
				FirstSeenAt:           occurrence.FirstSeenAt,
				LastSeenAt:            occurrence.LastSeenAt,
				Metadata:              metadata,
				EvidenceID:            evidenceID,
			})
		}

		targetMap := map[uint]SecurityScanTargetDetail{}
		if support.targetsTable && len(targetIDs) > 0 {
			var targets []models.SecurityScanTarget
			if err := database.DB.Where("id IN ?", targetIDs).Find(&targets).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vulnerability targets"})
				return
			}
			for _, target := range targets {
				targetMap[target.ID] = buildSecurityScanTargetDetail(target)
			}
		}

		evidenceMap := map[uint]SecurityScanEvidenceDetail{}
		if support.evidencesTable && len(evidenceIDs) > 0 {
			var evidences []models.SecurityScanEvidence
			if err := database.DB.Where("id IN ?", evidenceIDs).Order("created_at DESC, id DESC").Find(&evidences).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vulnerability evidences"})
				return
			}
			response.Evidences = make([]SecurityScanEvidenceDetail, 0, len(evidences))
			for _, evidence := range evidences {
				detail := buildSecurityScanEvidenceDetail(evidence)
				evidenceMap[evidence.ID] = detail
				response.Evidences = append(response.Evidences, detail)
			}
		}

		for i := range occurrenceDetails {
			if occurrenceDetails[i].TargetID != nil {
				if target, exists := targetMap[*occurrenceDetails[i].TargetID]; exists {
					targetCopy := target
					occurrenceDetails[i].Target = &targetCopy
				}
			}
			if occurrenceDetails[i].EvidenceID != nil {
				if _, exists := evidenceMap[*occurrenceDetails[i].EvidenceID]; !exists {
					occurrenceDetails[i].EvidenceID = nil
				}
			}
		}

		response.Occurrences = occurrenceDetails
		c.JSON(http.StatusOK, response)
	}
}

// UpdateVulnerabilityStatusRequest 更新漏洞状态请求
type UpdateVulnerabilityStatusRequest struct {
	Status string `json:"status" binding:"required"` // open, acknowledged, fixed, ignored
}

type ReviewVulnerabilityCandidateRequest struct {
	VerificationStatus string `json:"verification_status"`
	VerificationNote   string `json:"verification_note"`
	ReviewStatus       string `json:"review_status"`
	ReviewNote         string `json:"review_note"`
}

func (r ReviewVulnerabilityCandidateRequest) normalizedStatus() string {
	if value := strings.TrimSpace(r.VerificationStatus); value != "" {
		return value
	}
	return strings.TrimSpace(r.ReviewStatus)
}

func (r ReviewVulnerabilityCandidateRequest) normalizedNote() string {
	if value := strings.TrimSpace(r.VerificationNote); value != "" {
		return value
	}
	return strings.TrimSpace(r.ReviewNote)
}

// UpdateVulnerabilityStatus 更新漏洞状态
func (h *Handler) UpdateVulnerabilityStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vulnerability id"})
			return
		}

		var req UpdateVulnerabilityStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		if err := database.DB.Model(&models.SecurityVulnerability{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "status updated"})
	}
}

func (h *Handler) ReviewVulnerabilityCandidate() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vulnerability id"})
			return
		}

		var req ReviewVulnerabilityCandidateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		reviewStatus := req.normalizedStatus()
		if reviewStatus == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification_status is required"})
			return
		}
		switch reviewStatus {
		case "pending", "needs-test", "confirmed", "rejected":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification_status"})
			return
		}

		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}

		decorateVulnerabilityMetadata(&vuln, NewVulnDBService())
		if !isCandidateHostVersionMatch(vuln) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only host-version-match findings can be reviewed as candidates"})
			return
		}

		userID := c.GetUint("user_id")
		reviewedAt := time.Now()
		reviewNote := req.normalizedNote()
		updates := map[string]interface{}{
			"review_status": reviewStatus,
			"review_note":   reviewNote,
			"reviewed_by":   userID,
			"reviewed_at":   &reviewedAt,
		}
		if reviewStatus == "pending" && reviewNote == "" {
			updates["reviewed_by"] = nil
			updates["reviewed_at"] = nil
		}

		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.SecurityVulnerability{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}

			vuln.ReviewStatus = reviewStatus
			vuln.ReviewNote = reviewNote
			if reviewStatus == "pending" && reviewNote == "" {
				vuln.ReviewedBy = nil
				vuln.ReviewedAt = nil
			} else {
				vuln.ReviewedBy = &userID
				vuln.ReviewedAt = &reviewedAt
			}

			if reviewStatus == "confirmed" {
				_, err := upsertConfirmedVulnerabilityForCandidate(tx, &vuln)
				return err
			}

			return removeConfirmedVulnerabilityForCandidate(tx, &vuln)
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update review status"})
			return
		}

		if err := database.DB.First(&vuln, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load updated vulnerability"})
			return
		}
		decorateVulnerabilityRiskCategory(&vuln)
		c.JSON(http.StatusOK, vuln)
	}
}

// DeleteVulnerability 删除漏洞及其关联工单
func (h *Handler) DeleteVulnerability() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vulnerability id"})
			return
		}

		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load vulnerability"})
			return
		}

		tx := database.DB.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
			return
		}

		if vuln.ConfirmedVulnID != nil && *vuln.ConfirmedVulnID > 0 {
			if err := removeConfirmedVulnerabilityForCandidate(tx, &vuln); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete derived confirmed vulnerability"})
				return
			}
		}

		if vuln.SourceVulnID != nil && *vuln.SourceVulnID > 0 {
			if err := tx.Model(&models.SecurityVulnerability{}).
				Where("id = ?", *vuln.SourceVulnID).
				Update("confirmed_vuln_id", nil).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detach candidate confirmation"})
				return
			}
		}

		if err := tx.Where("vuln_id = ?", id).Delete(&models.VulnTicket{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete vulnerability tickets"})
			return
		}

		result := tx.Delete(&models.SecurityVulnerability{}, id)
		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete vulnerability"})
			return
		}
		if result.RowsAffected == 0 {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit vulnerability deletion"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "vulnerability deleted"})
	}
}

// ExportReport 导出报告（支持 HTML/JSON 格式）
func (h *Handler) ExportReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}

		// 获取任务信息
		var task models.SecurityScanTask
		if err := database.DB.First(&task, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		// 获取漏洞列表
		var vulns []models.SecurityVulnerability
		taskVulnerabilityScope(database.DB, uint(id)).Order("severity DESC").Find(&vulns)

		decorateVulnerabilityListRiskCategory(vulns)
		findingGroups := splitReportFindings(vulns)
		confirmedCounts := countReportSeverities(findingGroups.Confirmed)

		// 获取资产列表
		var assets []models.SecurityAsset
		database.DB.Where("task_id = ?", id).Order("ip ASC, port ASC").Find(&assets)

		// 支持 format 参数：html (默认)、json 或 csv
		format := c.DefaultQuery("format", "html")
		includeSnapshots := getScanPhase1SchemaSupport(database.DB).runExtensions
		var currentRun *SecurityScanRunDetail
		var targets []SecurityScanTargetDetail
		var occurrences []SecurityScanFindingOccurrenceDetail
		var evidences []SecurityScanEvidenceDetail
		if runID := activeRunIDForTask(task); runID > 0 {
			currentRun, _ = loadSecurityScanRunDetail(database.DB, runID, includeSnapshots)
		}
		targets, _ = loadTaskTargetDetails(database.DB, task)
		occurrences, _ = loadTaskOccurrenceDetails(database.DB, task)
		evidences, _ = loadTaskEvidenceDetails(database.DB, task)

		if format == "json" {
			// 生成 JSON 报告
			jsonData := generateJSONReportWithDetails(task, findingGroups, confirmedCounts, assets, currentRun, targets, occurrences, evidences)
			jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")

			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=security_report_"+task.Name+"_"+time.Now().Format("20060102")+".json")
			c.String(http.StatusOK, string(jsonBytes))
			return
		}

		if format == "csv" {
			// 生成 CSV 报告（详细漏洞报告）
			csvData := generateCSVReportWithDetails(findingGroups, task, occurrences, evidences)

			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=security_report_"+task.Name+"_"+time.Now().Format("20060102")+".csv")
			c.Header("Content-Transfer-Encoding", "binary")
			c.String(http.StatusOK, csvData)
			return
		}

		// 默认生成 HTML 报告
		html := generateHTMLReportWithDetails(task, findingGroups, confirmedCounts, assets, currentRun, targets, occurrences, evidences)

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=security_report_"+task.Name+"_"+time.Now().Format("20060102")+".html")
		c.String(http.StatusOK, html)
	}
}

// SecurityReport JSON 报告结构
type SecurityReport struct {
	ReportInfo           ReportInfo                            `json:"report_info"`
	TaskInfo             TaskInfo                              `json:"task_info"`
	Statistics           Statistics                            `json:"statistics"`
	Vulnerabilities      []Vulnerability                       `json:"vulnerabilities"`
	VerificationFindings []Vulnerability                       `json:"verification_findings,omitempty"`
	CandidateFindings    []Vulnerability                       `json:"candidate_findings,omitempty"`
	InventoryFindings    []Vulnerability                       `json:"inventory_findings,omitempty"`
	Assets               []Asset                               `json:"assets"`
	CurrentRun           *SecurityScanRunDetail                `json:"current_run,omitempty"`
	Targets              []SecurityScanTargetDetail            `json:"targets,omitempty"`
	Occurrences          []SecurityScanFindingOccurrenceDetail `json:"occurrences,omitempty"`
	Evidences            []SecurityScanEvidenceDetail          `json:"evidences,omitempty"`
}

type ReportInfo struct {
	GeneratedAt string `json:"generated_at"`
	Generator   string `json:"generator"`
	Version     string `json:"version"`
}

type TaskInfo struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	TargetType      string `json:"target_type"`
	Target          string `json:"target"`
	Status          string `json:"status"`
	NucleiVersion   string `json:"nuclei_version,omitempty"`
	TemplateVersion string `json:"template_version,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
}

type Statistics struct {
	TotalAssets          int64 `json:"total_assets"`
	TotalFindings        int64 `json:"total_findings"`
	TotalVulnerabilities int64 `json:"total_vulnerabilities"`
	VerificationFindings int64 `json:"verification_findings"`
	CandidateFindings    int64 `json:"candidate_findings"`
	InventoryFindings    int64 `json:"inventory_findings"`
	HighRisk             int64 `json:"high_risk"`
	MediumRisk           int64 `json:"medium_risk"`
	LowRisk              int64 `json:"low_risk"`
	InfoRisk             int64 `json:"info_risk"`
}

type Vulnerability struct {
	ID            uint    `json:"id"`
	Severity      string  `json:"severity"`
	CVEID         string  `json:"cve_id,omitempty"`
	PrimaryCVEID  string  `json:"primary_cve_id,omitempty"`
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	Solution      string  `json:"solution,omitempty"`
	VulnType      string  `json:"vuln_type,omitempty"`
	VulnURL       string  `json:"vuln_url,omitempty"`
	ScanMethod    string  `json:"scan_method,omitempty"`
	Scanner       string  `json:"scanner,omitempty"`
	FindingSource string  `json:"finding_source,omitempty"`
	FindingFamily string  `json:"finding_family,omitempty"`
	Confidence    string  `json:"confidence,omitempty"`
	MatchMode     string  `json:"match_mode,omitempty"`
	RiskCategory  string  `json:"risk_category,omitempty"`
	DisplayGroup  string  `json:"display_group,omitempty"`
	Payload       string  `json:"payload,omitempty"`
	Response      string  `json:"response,omitempty"`
	CVSSScore     float64 `json:"cvss_score"`
	IP            string  `json:"ip"`
	Port          int     `json:"port"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

type Asset struct {
	ID          uint   `json:"id"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ServiceName string `json:"service_name"`
	Version     string `json:"version,omitempty"`
	Banner      string `json:"banner,omitempty"`
}

// generateJSONReport 生成 JSON 格式报告
func generateJSONReport(task models.SecurityScanTask, findings reportFindingGroups, counts reportSeverityCounts, assets []models.SecurityAsset) SecurityReport {
	return generateJSONReportWithDetails(task, findings, counts, assets, nil, nil, nil, nil)
}

func generateJSONReportWithDetails(task models.SecurityScanTask, findings reportFindingGroups, counts reportSeverityCounts, assets []models.SecurityAsset, currentRun *SecurityScanRunDetail, targets []SecurityScanTargetDetail, occurrences []SecurityScanFindingOccurrenceDetail, evidences []SecurityScanEvidenceDetail) SecurityReport {
	// 格式化时间
	startedAt := ""
	completedAt := ""
	if task.StartedAt != nil {
		startedAt = task.StartedAt.Format("2006-01-02 15:04:05")
	}
	if task.CompletedAt != nil {
		completedAt = task.CompletedAt.Format("2006-01-02 15:04:05")
	}

	report := SecurityReport{
		ReportInfo: ReportInfo{
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
			Generator:   "OPS Platform Security Scanner",
			Version:     "1.0",
		},
		TaskInfo: TaskInfo{
			ID:              task.ID,
			Name:            task.Name,
			TargetType:      task.TargetType,
			Target:          task.Target,
			Status:          task.Status,
			NucleiVersion:   task.NucleiVersion,
			TemplateVersion: task.TemplateVersion,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		},
		Statistics: Statistics{
			TotalAssets:          int64(len(assets)),
			TotalFindings:        int64(len(findings.Confirmed) + len(findings.Candidate) + len(findings.Inventory)),
			TotalVulnerabilities: int64(len(findings.Confirmed)),
			VerificationFindings: int64(len(findings.Candidate)),
			CandidateFindings:    int64(len(findings.Candidate)),
			InventoryFindings:    int64(len(findings.Inventory)),
			HighRisk:             counts.High,
			MediumRisk:           counts.Medium,
			LowRisk:              counts.Low,
			InfoRisk:             counts.Info,
		},
	}

	report.Vulnerabilities = reportEntriesFromVulnerabilities(findings.Confirmed)
	report.VerificationFindings = reportEntriesFromVulnerabilities(findings.Candidate)
	report.CandidateFindings = reportEntriesFromVulnerabilities(findings.Candidate)
	report.InventoryFindings = reportEntriesFromVulnerabilities(findings.Inventory)
	report.CurrentRun = currentRun
	report.Targets = targets
	report.Occurrences = occurrences
	report.Evidences = evidences

	// 转换资产列表
	for _, a := range assets {
		report.Assets = append(report.Assets, Asset{
			ID:          a.ID,
			IP:          a.IP,
			Port:        a.Port,
			Protocol:    a.Protocol,
			ServiceName: a.ServiceName,
			Version:     a.Version,
			Banner:      a.Banner,
		})
	}

	return report
}

// AuthMiddleware 认证中间件（使用 auth 包的 JWT 验证）
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 将用户信息写入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("real_name", claims.RealName)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// VulnDB handlers - 漏洞知识库管理

// GetVulnDBList 获取漏洞知识库列表（支持分页）
func (h *Handler) GetVulnDBList() gin.HandlerFunc {
	return func(c *gin.Context) {
		keyword := c.Query("keyword")
		severity := c.Query("severity")
		vulnType := c.Query("vuln_type")

		params := Paginate(c)
		offset := (params.Page - 1) * params.PageSize

		var vulns []models.VulnerabilityDatabase
		var total int64

		query := database.DB.Model(&models.VulnerabilityDatabase{}).Order("cvss_score DESC")

		// 关键词搜索
		if keyword != "" {
			query = query.Where("cve_id LIKE ? OR title LIKE ? OR cnvd_id LIKE ? OR cnnvd_id LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}

		// 严重程度筛选
		if severity != "" {
			query = query.Where("severity = ?", severity)
		}

		// 漏洞类型筛选
		if vulnType != "" {
			query = query.Where("vuln_type LIKE ?", "%"+vulnType+"%")
		}

		// 获取总数
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count vuln list"})
			return
		}

		// 分页查询
		if err := query.Offset(offset).Limit(params.PageSize).Find(&vulns).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vuln list"})
			return
		}

		totalPages := int(total) / params.PageSize
		if int(total)%params.PageSize > 0 {
			totalPages++
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Total:      int(total),
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
			Data:       vulns,
		})
	}
}

// GetVulnDBStats 获取漏洞知识库统计
func (h *Handler) GetVulnDBStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		vulnDB := NewVulnDBService()
		stats, err := vulnDB.GetStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}

// ImportVulnDBRequest 导入请求
type ImportVulnDBRequest struct {
	CSVData string `json:"csv_data" binding:"required"`
}

// ImportVulnDB 导入漏洞数据
func (h *Handler) ImportVulnDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportVulnDBRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		vulnDB := NewVulnDBService()
		inserted, err := vulnDB.SyncFromCSV(req.CSVData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "import failed", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "import completed",
			"inserted": inserted,
		})
	}
}

// SyncNVD 从 NVD API 同步漏洞数据
func (h *Handler) SyncNVD() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusGone, gin.H{
			"error":   "vulnerability sync has been disabled",
			"message": "漏洞知识库同步功能已停用",
		})
	}
}

func hasRunningSyncTask(vulnDB *VulnDBService) bool {
	if vulnDB == nil || vulnDB.db == nil {
		return false
	}

	var count int64
	if err := vulnDB.db.Model(&SyncTask{}).Where("status = ?", "running").Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func (h *Handler) SyncNFull() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusGone, gin.H{
			"error":   "vulnerability sync has been disabled",
			"message": "漏洞知识库同步功能已停用",
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NVD API Response structures
type NVDResponse struct {
	Vulnerabilities []NVDVulnerability `json:"vulnerabilities"`
}

type NVDVulnerability struct {
	CVE *NVDCVE `json:"cve"`
}

type NVDCVE struct {
	ID           string           `json:"id"`
	Descriptions []NVDDescription `json:"descriptions"`
	Metrics      NVDMetrics       `json:"metrics"`
}

type NVDDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type NVDMetrics struct {
	CVSSMetricV31 []NVDCVSSMetric `json:"cvssMetricV31"`
	CVSSMetricV30 []NVDCVSSMetric `json:"cvssMetricV30"`
	CVSSMetricV2  []NVDCVSSMetric `json:"cvssMetricV2"`
}

type NVDCVSSMetric struct {
	CVSSData NVDScore `json:"cvssData"`
}

type NVDScore struct {
	BaseScore    float64 `json:"baseScore"`
	VectorString string  `json:"vectorString"`
}

// ImportCNVDRequest CNVD 导入请求
type ImportCNVDRequest struct {
	CSVData string `json:"csv_data" binding:"required"`
}

// ImportCNVD 导入 CNVD 数据并自动关联现有 CVE 漏洞
func (h *Handler) ImportCNVD() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportCNVDRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		lines := strings.Split(req.CSVData, "\n")
		var inserted, updated int

		for _, line := range lines[1:] { // 跳过表头
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, ",")
			if len(parts) < 3 {
				continue
			}

			cnvdID := strings.TrimSpace(parts[0])
			cveID := strings.TrimSpace(parts[1])
			title := strings.TrimSpace(parts[2])

			if cnvdID == "" || (!strings.HasPrefix(cnvdID, "CNVD-") && !strings.HasPrefix(cnvdID, "CNVD-")) {
				continue
			}

			// 查找是否已存在该 CVE 的记录
			var existing models.VulnerabilityDatabase
			searchID := cveID
			if !strings.HasPrefix(cveID, "CVE-") {
				searchID = "CVE-" + cveID
			}
			database.DB.Where("cve_id = ?", searchID).First(&existing)

			severity := "medium"
			cvssScore := 0.0
			description := ""
			solution := ""

			if len(parts) > 3 {
				severity = strings.TrimSpace(parts[3])
			}
			if len(parts) > 4 {
				if s, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64); err == nil {
					cvssScore = s
				}
			}
			if len(parts) > 5 {
				description = strings.TrimSpace(parts[5])
			}
			if len(parts) > 6 {
				solution = strings.TrimSpace(parts[6])
			}

			if existing.ID > 0 {
				// 更新现有记录，补充 CNVD ID
				existing.CNVDID = cnvdID
				if title != "" && existing.Title == "" {
					existing.Title = title
				}
				if description != "" && existing.Description == "" {
					existing.Description = description
				}
				if solution != "" && existing.Solution == "" {
					existing.Solution = solution
				}
				existing.LastUpdated = time.Now()
				database.DB.Save(&existing)
				updated++
			} else {
				// 新建记录
				vuln := models.VulnerabilityDatabase{
					CVEID:       searchID,
					CNVDID:      cnvdID,
					Title:       title,
					Description: description,
					Severity:    severity,
					CVSSScore:   cvssScore,
					Solution:    solution,
					Source:      "cnvd",
					LastUpdated: time.Now(),
				}
				database.DB.Create(&vuln)
				inserted++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "CNVD import completed",
			"inserted": inserted,
			"updated":  updated,
		})
	}
}

// ImportCNNVD 导入 CNNVD 数据并自动关联现有 CVE 漏洞
func (h *Handler) ImportCNNVD() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportCNVDRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		lines := strings.Split(req.CSVData, "\n")
		var inserted, updated int

		for _, line := range lines[1:] { // 跳过表头
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.Split(line, ",")
			if len(parts) < 3 {
				continue
			}

			cnnvdID := strings.TrimSpace(parts[0])
			cveID := strings.TrimSpace(parts[1])
			title := strings.TrimSpace(parts[2])

			if cnnvdID == "" || !strings.HasPrefix(cnnvdID, "CNNVD-") {
				continue
			}

			// 查找是否已存在该 CVE 的记录
			var existing models.VulnerabilityDatabase
			searchID := cveID
			if !strings.HasPrefix(cveID, "CVE-") {
				searchID = "CVE-" + cveID
			}
			database.DB.Where("cve_id = ?", searchID).First(&existing)

			severity := "medium"
			cvssScore := 0.0
			description := ""
			solution := ""
			vulnType := ""

			if len(parts) > 3 {
				severity = strings.TrimSpace(parts[3])
			}
			if len(parts) > 4 {
				if s, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64); err == nil {
					cvssScore = s
				}
			}
			if len(parts) > 5 {
				description = strings.TrimSpace(parts[5])
			}
			if len(parts) > 6 {
				solution = strings.TrimSpace(parts[6])
			}
			if len(parts) > 7 {
				vulnType = strings.TrimSpace(parts[7])
			}

			if existing.ID > 0 {
				// 更新现有记录，补充 CNNVD ID
				existing.CNNVDID = cnnvdID
				if title != "" && existing.Title == "" {
					existing.Title = title
				}
				if description != "" && existing.Description == "" {
					existing.Description = description
				}
				if solution != "" && existing.Solution == "" {
					existing.Solution = solution
				}
				if vulnType != "" && existing.VulnType == "" {
					existing.VulnType = vulnType
				}
				existing.LastUpdated = time.Now()
				database.DB.Save(&existing)
				updated++
			} else {
				// 新建记录
				vuln := models.VulnerabilityDatabase{
					CVEID:       searchID,
					CNNVDID:     cnnvdID,
					Title:       title,
					Description: description,
					VulnType:    vulnType,
					Severity:    severity,
					CVSSScore:   cvssScore,
					Solution:    solution,
					Source:      "cnnvd",
					LastUpdated: time.Now(),
				}
				database.DB.Create(&vuln)
				inserted++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "CNNVD import completed",
			"inserted": inserted,
			"updated":  updated,
		})
	}
}

// ==================== 资产中心 API ====================

// GetAssets 获取资产列表（支持分页、筛选）
func (h *Handler) GetAssets() gin.HandlerFunc {
	return func(c *gin.Context) {
		var assets []models.Asset
		var total int64

		params := Paginate(c)
		offset := (params.Page - 1) * params.PageSize

		query := database.DB.Model(&models.Asset{}).Order("last_seen DESC")

		// 筛选条件
		if ip := c.Query("ip"); ip != "" {
			query = query.Where("ip LIKE ?", "%"+ip+"%")
		}
		if assetType := c.Query("asset_type"); assetType != "" {
			query = query.Where("asset_type = ?", assetType)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if importance := c.Query("importance"); importance != "" {
			query = query.Where("importance = ?", importance)
		}
		if assetGroup := c.Query("asset_group"); assetGroup != "" {
			query = query.Where("asset_group = ?", assetGroup)
		}
		if owner := c.Query("owner"); owner != "" {
			query = query.Where("owner LIKE ?", "%"+owner+"%")
		}
		if keyword := c.Query("keyword"); keyword != "" {
			query = query.Where("ip LIKE ? OR service_name LIKE ? OR banner LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}

		// 获取总数
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count assets"})
			return
		}

		// 分页查询
		if err := query.Offset(offset).Limit(params.PageSize).Find(&assets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get assets"})
			return
		}

		totalPages := int(total) / params.PageSize
		if int(total)%params.PageSize > 0 {
			totalPages++
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Total:      int(total),
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
			Data:       assets,
		})
	}
}

// GetAsset 获取资产详情
func (h *Handler) GetAsset() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
			return
		}

		var asset models.Asset
		if err := database.DB.First(&asset, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}

		// 获取关联漏洞统计
		var vulnCount struct {
			Critical int64
			High     int64
			Medium   int64
			Low      int64
			Total    int64
		}

		database.DB.Model(&models.SecurityVulnerability{}).
			Where("ip = ?", asset.IP).
			Count(&vulnCount.Total)

		database.DB.Model(&models.SecurityVulnerability{}).
			Where("ip = ? AND severity = ?", asset.IP, "critical").
			Count(&vulnCount.Critical)

		database.DB.Model(&models.SecurityVulnerability{}).
			Where("ip = ? AND severity = ?", asset.IP, "high").
			Count(&vulnCount.High)

		database.DB.Model(&models.SecurityVulnerability{}).
			Where("ip = ? AND severity = ?", asset.IP, "medium").
			Count(&vulnCount.Medium)

		database.DB.Model(&models.SecurityVulnerability{}).
			Where("ip = ? AND severity = ?", asset.IP, "low").
			Count(&vulnCount.Low)

		c.JSON(http.StatusOK, gin.H{
			"asset":      asset,
			"vuln_count": vulnCount,
		})
	}
}

// CreateAssetRequest 创建资产请求
type CreateAssetRequest struct {
	IP          string `json:"ip" binding:"required"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ServiceName string `json:"service_name"`
	Version     string `json:"version"`
	OSInfo      string `json:"os_info"`
	Banner      string `json:"banner"`
	AssetType   string `json:"asset_type"`
	AssetGroup  string `json:"asset_group"`
	Tags        string `json:"tags"`
	Importance  string `json:"importance"`
	Owner       string `json:"owner"`
	Department  string `json:"department"`
	Status      string `json:"status"`
}

// CreateAsset 创建资产
func (h *Handler) CreateAsset() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		now := time.Now()
		asset := models.Asset{
			IP:          req.IP,
			Port:        req.Port,
			Protocol:    req.Protocol,
			ServiceName: req.ServiceName,
			Version:     req.Version,
			OSInfo:      req.OSInfo,
			Banner:      req.Banner,
			AssetType:   req.AssetType,
			AssetGroup:  req.AssetGroup,
			Tags:        req.Tags,
			Importance:  req.Importance,
			Owner:       req.Owner,
			Department:  req.Department,
			Status:      req.Status,
			FirstSeen:   now,
			LastSeen:    now,
		}

		if err := database.DB.Create(&asset).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create asset"})
			return
		}

		c.JSON(http.StatusCreated, asset)
	}
}

// UpdateAsset 更新资产
func (h *Handler) UpdateAsset() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
			return
		}

		var asset models.Asset
		if err := database.DB.First(&asset, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}

		var req CreateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		updates := map[string]interface{}{
			"ip":           req.IP,
			"port":         req.Port,
			"protocol":     req.Protocol,
			"service_name": req.ServiceName,
			"version":      req.Version,
			"os_info":      req.OSInfo,
			"banner":       req.Banner,
			"asset_type":   req.AssetType,
			"asset_group":  req.AssetGroup,
			"tags":         req.Tags,
			"importance":   req.Importance,
			"owner":        req.Owner,
			"department":   req.Department,
			"status":       req.Status,
			"last_seen":    time.Now(),
		}

		if err := database.DB.Model(&asset).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update asset"})
			return
		}

		c.JSON(http.StatusOK, asset)
	}
}

// DeleteAsset 删除资产
func (h *Handler) DeleteAsset() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
			return
		}

		if err := database.DB.Delete(&models.Asset{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete asset"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
	}
}

// GetAssetStats 获取资产统计
func (h *Handler) GetAssetStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		var total, online, offline, unknown int64

		database.DB.Model(&models.Asset{}).Count(&total)
		database.DB.Model(&models.Asset{}).Where("status = ?", "online").Count(&online)
		database.DB.Model(&models.Asset{}).Where("status = ?", "offline").Count(&offline)
		database.DB.Model(&models.Asset{}).Where("status = ?", "unknown").Count(&unknown)

		// 按资产类型统计
		var byType []struct {
			Type  string `json:"type"`
			Count int64  `json:"count"`
		}
		database.DB.Model(&models.Asset{}).
			Select("asset_type as type, count(*) as count").
			Group("asset_type").
			Scan(&byType)

		// 按重要性统计
		var byImportance []struct {
			Importance string `json:"importance"`
			Count      int64  `json:"count"`
		}
		database.DB.Model(&models.Asset{}).
			Select("importance, count(*) as count").
			Group("importance").
			Scan(&byImportance)

		c.JSON(http.StatusOK, gin.H{
			"total":         total,
			"online":        online,
			"offline":       offline,
			"unknown":       unknown,
			"by_type":       byType,
			"by_importance": byImportance,
		})
	}
}

// ==================== 漏洞工单 API ====================

// GetTickets 获取漏洞工单列表（支持分页、筛选）
func (h *Handler) GetTickets() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tickets []models.VulnTicket
		var total int64

		params := Paginate(c)
		offset := (params.Page - 1) * params.PageSize

		query := database.DB.Model(&models.VulnTicket{}).Order("created_at DESC")

		// 筛选条件
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if priority := c.Query("priority"); priority != "" {
			query = query.Where("priority = ?", priority)
		}
		if assignee := c.Query("assignee"); assignee != "" {
			a, _ := strconv.ParseUint(assignee, 10, 32)
			query = query.Where("assignee = ?", uint(a))
		}
		if department := c.Query("department"); department != "" {
			query = query.Where("department = ?", department)
		}

		// 获取总数
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tickets"})
			return
		}

		// 分页查询
		if err := query.Offset(offset).Limit(params.PageSize).Find(&tickets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tickets"})
			return
		}

		totalPages := int(total) / params.PageSize
		if int(total)%params.PageSize > 0 {
			totalPages++
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Total:      int(total),
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
			Data:       tickets,
		})
	}
}

// GetTicket 获取工单详情
func (h *Handler) GetTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		var ticket models.VulnTicket
		if err := database.DB.First(&ticket, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}

		// 获取关联漏洞信息
		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, ticket.VulnID).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{
				"ticket": ticket,
				"vuln":   vuln,
			})
			return
		}

		c.JSON(http.StatusOK, ticket)
	}
}

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	VulnID   uint   `json:"vuln_id" binding:"required"`
	Assignee uint   `json:"assignee"`
	Priority string `json:"priority"`
	DueDate  string `json:"due_date"`
	Notes    string `json:"notes"`
}

// CreateTicket 创建漏洞工单
func (h *Handler) CreateTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTicketRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 获取漏洞信息
		var vuln models.SecurityVulnerability
		if err := database.DB.First(&vuln, req.VulnID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}

		// 获取当前用户信息
		userID, _ := c.Get("user_id")
		realName, _ := c.Get("real_name")

		var dueDate *time.Time
		if req.DueDate != "" {
			if parsed, err := time.Parse("2006-01-02", req.DueDate); err == nil {
				dueDate = &parsed
			}
		}

		ticket := models.VulnTicket{
			VulnID:        req.VulnID,
			VulnTitle:     vuln.Title,
			Assignee:      req.Assignee,
			Priority:      req.Priority,
			DueDate:       dueDate,
			Notes:         req.Notes,
			Status:        models.VulnTicketStatusOpen,
			CreatedBy:     userID.(uint),
			CreatedByName: realName.(string),
		}

		if err := database.DB.Create(&ticket).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
			return
		}

		// 创建工单历史记录
		history := models.TicketHistory{
			TicketID:     ticket.ID,
			Action:       "created",
			NewStatus:    models.VulnTicketStatusOpen,
			Comment:      req.Notes,
			OperatorID:   userID.(uint),
			OperatorName: realName.(string),
		}
		database.DB.Create(&history)

		// 更新漏洞状态为已指派
		database.DB.Model(&vuln).Update("status", "acknowledged")

		c.JSON(http.StatusCreated, ticket)
	}
}

// UpdateTicketRequest 更新工单请求
type UpdateTicketRequest struct {
	Assignee     uint   `json:"assignee"`
	AssigneeName string `json:"assignee_name"`
	Priority     string `json:"priority"`
	Status       string `json:"status"`
	DueDate      string `json:"due_date"`
	Notes        string `json:"notes"`
	Comments     string `json:"comments"`
}

// UpdateTicket 更新工单
func (h *Handler) UpdateTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		var ticket models.VulnTicket
		if err := database.DB.First(&ticket, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}

		var req UpdateTicketRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		updates := map[string]interface{}{
			"assignee":      req.Assignee,
			"assignee_name": req.AssigneeName,
			"priority":      req.Priority,
			"status":        req.Status,
			"notes":         req.Notes,
			"comments":      req.Comments,
		}

		if req.DueDate != "" {
			if parsed, err := time.Parse("2006-01-02", req.DueDate); err == nil {
				updates["due_date"] = &parsed
			}
		}

		// 如果状态为已修复，记录解决时间
		if req.Status == models.VulnTicketStatusFixed {
			now := time.Now()
			updates["resolved_at"] = &now
		}

		if err := database.DB.Model(&ticket).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
			return
		}

		// 同步更新漏洞状态
		if req.Status == models.VulnTicketStatusFixed {
			database.DB.Model(&models.SecurityVulnerability{}).Where("id = ?", ticket.VulnID).Update("status", "fixed")
		}

		c.JSON(http.StatusOK, ticket)
	}
}

// DeleteTicket 删除工单
func (h *Handler) DeleteTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		if err := database.DB.Delete(&models.VulnTicket{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "ticket deleted"})
	}
}

// AssignTicketRequest 指派工单请求
type AssignTicketRequest struct {
	Assignee     uint   `json:"assignee" binding:"required"`
	AssigneeName string `json:"assignee_name" binding:"required"`
	Department   string `json:"department"`
	Priority     string `json:"priority"`
}

// AssignTicket 指派工单
func (h *Handler) AssignTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		var ticket models.VulnTicket
		if err := database.DB.First(&ticket, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}

		var req AssignTicketRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 获取当前用户信息
		userID, _ := c.Get("user_id")
		realName, _ := c.Get("real_name")

		// 记录旧的指派人
		oldAssignee := ticket.AssigneeName

		updates := map[string]interface{}{
			"assignee":      req.Assignee,
			"assignee_name": req.AssigneeName,
			"department":    req.Department,
			"status":        models.VulnTicketStatusProcessing,
		}

		if req.Priority != "" {
			updates["priority"] = req.Priority
		}

		if err := database.DB.Model(&ticket).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign ticket"})
			return
		}

		// 创建工单历史记录
		history := models.TicketHistory{
			TicketID:     ticket.ID,
			Action:       "assigned",
			OldStatus:    ticket.Status,
			NewStatus:    models.VulnTicketStatusProcessing,
			OldAssignee:  oldAssignee,
			NewAssignee:  req.AssigneeName,
			OperatorID:   userID.(uint),
			OperatorName: realName.(string),
		}
		database.DB.Create(&history)

		// 更新漏洞状态为处理中
		database.DB.Model(&models.SecurityVulnerability{}).Where("id = ?", ticket.VulnID).Update("status", "acknowledged")

		c.JSON(http.StatusOK, ticket)
	}
}

// CloseTicket 关闭工单
func (h *Handler) CloseTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		var ticket models.VulnTicket
		if err := database.DB.First(&ticket, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}

		var req struct {
			Status   string `json:"status"`
			Comments string `json:"comments"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// 获取当前用户信息
		userID, _ := c.Get("user_id")
		realName, _ := c.Get("real_name")

		// 记录旧状态
		oldStatus := ticket.Status

		now := time.Now()
		updates := map[string]interface{}{
			"status":      req.Status,
			"comments":    req.Comments,
			"resolved_at": &now,
		}

		if err := database.DB.Model(&ticket).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close ticket"})
			return
		}

		// 创建工单历史记录
		action := "closed"
		if req.Status == models.VulnTicketStatusFixed {
			action = "fixed"
		} else if req.Status == models.VulnTicketStatusRejected {
			action = "rejected"
		}
		history := models.TicketHistory{
			TicketID:     ticket.ID,
			Action:       action,
			OldStatus:    oldStatus,
			NewStatus:    req.Status,
			Comment:      req.Comments,
			OperatorID:   userID.(uint),
			OperatorName: realName.(string),
		}
		database.DB.Create(&history)

		// 同步更新漏洞状态
		if req.Status == models.VulnTicketStatusFixed {
			database.DB.Model(&models.SecurityVulnerability{}).Where("id = ?", ticket.VulnID).Update("status", "fixed")
		} else if req.Status == models.VulnTicketStatusRejected {
			database.DB.Model(&models.SecurityVulnerability{}).Where("id = ?", ticket.VulnID).Update("status", "ignored")
		}

		c.JSON(http.StatusOK, ticket)
	}
}

// GetTicketHistory 获取工单历史记录
func (h *Handler) GetTicketHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket id"})
			return
		}

		var history []models.TicketHistory
		if err := database.DB.Where("ticket_id = ?", id).Order("created_at DESC").Find(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get history"})
			return
		}

		c.JSON(http.StatusOK, history)
	}
}

// ============================================
// 新增漏洞库 API 处理函数
// ============================================

// SearchVulnDB 搜索漏洞库
func (h *Handler) SearchVulnDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		keyword := c.Query("keyword")
		severity := c.Query("severity")
		vulnType := c.Query("vuln_type")
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("page_size", "20")

		pageInt, _ := strconv.Atoi(page)
		pageSizeInt, _ := strconv.Atoi(pageSize)
		offset := (pageInt - 1) * pageSizeInt

		query := database.DB.Model(&models.VulnerabilityDatabase{})

		// 关键词搜索
		if keyword != "" {
			query = query.Where("cve_id LIKE ? OR title LIKE ? OR description LIKE ? OR affected_product LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}

		// 严重程度筛选
		if severity != "" {
			query = query.Where("severity = ?", severity)
		}

		// 漏洞类型筛选
		if vulnType != "" {
			query = query.Where("vuln_type = ?", vulnType)
		}

		// 统计总数
		var total int64
		query.Count(&total)

		// 分页查询
		var vulns []models.VulnerabilityDatabase
		query.Order("cvss_score DESC").Offset(offset).Limit(pageSizeInt).Find(&vulns)

		c.JSON(http.StatusOK, gin.H{
			"data":      vulns,
			"total":     total,
			"page":      pageInt,
			"page_size": pageSizeInt,
		})
	}
}

// GetVulnByCVE 根据 CVE ID 获取漏洞详情
func (h *Handler) GetVulnByCVE() gin.HandlerFunc {
	return func(c *gin.Context) {
		cveID := c.Param("cveId")
		if cveID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cve_id is required"})
			return
		}

		vulnDB := NewVulnDBService()
		vuln := vulnDB.LookupCVE(cveID)

		if vuln == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "vulnerability not found"})
			return
		}

		c.JSON(http.StatusOK, vuln)
	}
}

// GetSyncTasks 获取同步任务历史
func (h *Handler) GetSyncTasks() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"tasks":          []SyncTask{},
			"last_sync_time": "",
			"disabled":       true,
			"message":        "漏洞知识库同步功能已停用",
		})
	}
}

// MatchServiceVulnsRequest 服务漏洞匹配请求
type MatchServiceVulnsRequest struct {
	ServiceName string `json:"service_name" binding:"required"`
	ProductName string `json:"product_name"`
	Version     string `json:"version"`
}

// MatchServiceVulns 根据服务信息匹配漏洞
func (h *Handler) MatchServiceVulns() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MatchServiceVulnsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}

		// 使用漏洞匹配器
		matcher := NewVulnMatcher()

		// 创建 PortInfo 用于匹配
		portInfo := &PortInfo{
			Service: req.ServiceName,
			Product: req.ProductName,
			Version: req.Version,
		}

		// 执行匹配
		results := matcher.MatchByFingerprint(portInfo)

		// 转换为响应格式
		var matchedVulns []gin.H
		for _, r := range results {
			matchedVulns = append(matchedVulns, gin.H{
				"cve_id":      r.CVEID,
				"cvss":        r.CVSS,
				"severity":    r.Severity,
				"title":       r.Title,
				"description": r.Description,
				"solution":    r.Solution,
				"vuln_type":   r.VulnType,
				"matched_on":  r.MatchedOn,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"service": req.ServiceName,
			"version": req.Version,
			"total":   len(matchedVulns),
			"vulns":   matchedVulns,
		})
	}
}