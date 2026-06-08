package security

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/edy/ops-platform/internal/audit"
	"github.com/edy/ops-platform/internal/database"
	"github.com/gin-gonic/gin"
)

type CreateFIMPolicyRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	Enabled         bool   `json:"enabled"`
	Severity        string `json:"severity"`
	NotifyChannels  string `json:"notify_channels"`
	ScanIntervalSec int    `json:"scan_interval_sec"`
	HashMode        string `json:"hash_mode"`
	CompareMode     string `json:"compare_mode"`
}

type UpdateFIMPolicyRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Enabled         *bool  `json:"enabled"`
	Severity        string `json:"severity"`
	NotifyChannels  string `json:"notify_channels"`
	ScanIntervalSec int    `json:"scan_interval_sec"`
	HashMode        string `json:"hash_mode"`
	CompareMode     string `json:"compare_mode"`
}

type AddFIMPolicyTargetsRequest struct {
	ServerIDs []uint `json:"server_ids" binding:"required"`
}

type CreateFIMWatchPathRequest struct {
	Path            string `json:"path" binding:"required"`
	ScanMode        string `json:"scan_mode"`
	Recursive       bool   `json:"recursive"`
	MaxDepth        int    `json:"max_depth"`
	FileGlob        string `json:"file_glob"`
	ExcludeGlob     string `json:"exclude_glob"`
	HashOnMatchOnly bool   `json:"hash_on_match_only"`
}

type UpdateFIMWatchPathRequest struct {
	Path            string `json:"path"`
	ScanMode        string `json:"scan_mode"`
	Recursive       *bool  `json:"recursive"`
	MaxDepth        int    `json:"max_depth"`
	FileGlob        string `json:"file_glob"`
	ExcludeGlob     string `json:"exclude_glob"`
	HashOnMatchOnly *bool  `json:"hash_on_match_only"`
}

type BuildFIMBaselineRequest struct {
	ServerID uint `json:"server_id" binding:"required"`
}

type RunFIMScanRequest struct {
	ServerID uint   `json:"server_id" binding:"required"`
	ScanType string `json:"scan_type"`
}

type UpdateFIMAlertStatusRequest struct {
	Comment string `json:"comment"`
}

func (h *Handler) registerFIMRoutes(security *gin.RouterGroup) {
	fim := security.Group("/fim")
	{
		fim.GET("/policies", h.GetFIMPolicies())
		fim.POST("/policies", h.CreateFIMPolicy())
		fim.PUT("/policies/:id", h.UpdateFIMPolicy())
		fim.DELETE("/policies/:id", h.DeleteFIMPolicy())
		fim.POST("/policies/:id/clear-history", h.ClearFIMPolicyHistory())
		fim.POST("/policies/:id/enable", h.EnableFIMPolicy())
		fim.POST("/policies/:id/disable", h.DisableFIMPolicy())

		fim.GET("/policies/:id/targets", h.GetFIMPolicyTargets())
		fim.POST("/policies/:id/targets", h.AddFIMPolicyTargets())
		fim.DELETE("/policies/:id/targets/:targetId", h.DeleteFIMPolicyTarget())

		fim.GET("/policies/:id/watch-paths", h.GetFIMWatchPaths())
		fim.POST("/policies/:id/watch-paths", h.CreateFIMWatchPath())
		fim.PUT("/watch-paths/:id", h.UpdateFIMWatchPath())
		fim.DELETE("/watch-paths/:id", h.DeleteFIMWatchPath())

		fim.POST("/policies/:id/baselines/build", h.BuildFIMBaseline())
		fim.POST("/policies/:id/scan", h.RunFIMScan())

		fim.GET("/snapshots", h.GetFIMSnapshots())
		fim.GET("/snapshots/:id", h.GetFIMSnapshotDetail())
		fim.POST("/snapshots/:id/activate-baseline", h.ActivateFIMBaseline())

		fim.GET("/events", h.GetFIMDiffEvents())
		fim.DELETE("/events/:id", h.DeleteFIMDiffEvent())
		fim.GET("/alerts", h.GetFIMAlerts())
		fim.GET("/alerts/:id", h.GetFIMAlertDetail())
		fim.DELETE("/alerts/:id", h.DeleteFIMAlert())
		fim.POST("/alerts/:id/ack", h.AckFIMAlert())
		fim.POST("/alerts/:id/resolve", h.ResolveFIMAlert())
		fim.POST("/alerts/:id/close", h.CloseFIMAlert())

		// Known Hosts Management (SSH Host Keys Whitelist)
		fim.GET("/known-hosts", ListKnownHosts)
		fim.GET("/known-hosts/:id", GetKnownHost)
		fim.POST("/known-hosts", AddKnownHost)
		fim.PUT("/known-hosts/:id", UpdateKnownHost)
		fim.DELETE("/known-hosts/:id", DeleteKnownHost)
		fim.POST("/known-hosts/import", ImportKnownHosts)
		fim.GET("/known-hosts/export", ExportKnownHosts)
		fim.POST("/known-hosts/batch", BatchAddKnownHosts)
		fim.GET("/connection-logs", GetConnectionLogs)
	}
}

func (h *Handler) fimService() *FIMService {
	return NewFIMService()
}

func parseUintParam(c *gin.Context, key string) (uint, error) {
	raw := c.Param(key)
	value, err := strconv.ParseUint(raw, 10, 64)
	return uint(value), err
}

func (h *Handler) GetFIMPolicies() gin.HandlerFunc {
	return func(c *gin.Context) {
		service := h.fimService()
		params := Paginate(c)
		items, total, err := service.ListPolicies(FIMPolicyFilter{
			Keyword:  c.Query("keyword"),
			Page:     params.Page,
			PageSize: params.PageSize,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim policies"})
			return
		}
		c.JSON(http.StatusOK, PaginatedResponse{Total: int(total), Page: params.Page, PageSize: params.PageSize, TotalPages: calcTotalPages(int(total), params.PageSize), Data: items})
	}
}

func (h *Handler) CreateFIMPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateFIMPolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		item, err := h.fimService().CreatePolicy(req, currentOperator(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fim policy"})
			return
		}
		audit.SetOperationAuditAfter(c, item)
		audit.SetOperationAuditSummary(c, "创建了 FIM 巡检策略。")
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) UpdateFIMPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var req UpdateFIMPolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		var before FIMPolicy
		if err := database.DB.First(&before, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "fim policy not found"})
			return
		}
		item, err := h.fimService().UpdatePolicy(id, req, currentOperator(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fim policy"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, item)
		audit.SetOperationAuditSummary(c, "更新了 FIM 巡检策略。")
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) DeleteFIMPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var before FIMPolicy
		if err := database.DB.First(&before, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "fim policy not found"})
			return
		}
		if err := h.fimService().DeletePolicy(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fim policy"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditSummary(c, "删除了 FIM 巡检策略。")
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) ClearFIMPolicyHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		if err := h.fimService().ClearPolicyHistory(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear fim policy history"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) EnableFIMPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		if err := h.fimService().SetPolicyEnabled(id, true, currentOperator(c)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable fim policy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) DisableFIMPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		if err := h.fimService().SetPolicyEnabled(id, false, currentOperator(c)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable fim policy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) GetFIMPolicyTargets() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		items, err := h.fimService().ListTargets(policyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim targets"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func (h *Handler) AddFIMPolicyTargets() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var req AddFIMPolicyTargetsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		if err := h.fimService().AddTargets(policyID, req.ServerIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add fim targets"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) DeleteFIMPolicyTarget() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		targetID, err := parseUintParam(c, "targetId")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target id"})
			return
		}
		if err := h.fimService().DeleteTarget(policyID, targetID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fim target"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) GetFIMWatchPaths() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		items, err := h.fimService().ListWatchPaths(policyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim watch paths"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func (h *Handler) CreateFIMWatchPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var req CreateFIMWatchPathRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		item, err := h.fimService().CreateWatchPath(policyID, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create fim watch path"})
			return
		}
		audit.SetOperationAuditAfter(c, item)
		audit.SetOperationAuditSummary(c, "新增了 FIM 监控目录配置。")
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) UpdateFIMWatchPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watch path id"})
			return
		}
		var req UpdateFIMWatchPathRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		var before FIMWatchPath
		if err := database.DB.First(&before, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "fim watch path not found"})
			return
		}
		item, err := h.fimService().UpdateWatchPath(id, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fim watch path"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditAfter(c, item)
		audit.SetOperationAuditSummary(c, "更新了 FIM 监控目录配置。")
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) DeleteFIMWatchPath() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watch path id"})
			return
		}
		var before FIMWatchPath
		if err := database.DB.First(&before, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "fim watch path not found"})
			return
		}
		if err := h.fimService().DeleteWatchPath(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fim watch path"})
			return
		}
		audit.SetOperationAuditBefore(c, before)
		audit.SetOperationAuditSummary(c, "删除了 FIM 监控目录配置。")
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) BuildFIMBaseline() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var req BuildFIMBaselineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		item, err := h.fimService().BuildBaseline(policyID, req.ServerID, currentOperator(c))
		if err != nil {
			writeFIMExecutionError(c, err, "failed to build fim baseline")
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) RunFIMScan() gin.HandlerFunc {
	return func(c *gin.Context) {
		policyID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
			return
		}
		var req RunFIMScanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		item, err := h.fimService().RunScan(policyID, req.ServerID, req.ScanType, currentOperator(c))
		if err != nil {
			writeFIMExecutionError(c, err, "failed to run fim scan")
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func writeFIMExecutionError(c *gin.Context, err error, fallback string) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
		return
	}
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "already running"):
		c.JSON(http.StatusConflict, gin.H{"error": message})
	case strings.Contains(lower, "not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": message})
	case strings.Contains(lower, "has no watch paths") || strings.Contains(lower, "invalid"):
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

func (h *Handler) GetFIMSnapshots() gin.HandlerFunc {
	return func(c *gin.Context) {
		params := Paginate(c)
		filter := FIMSnapshotFilter{
			PolicyID:     parseUintQuery(c, "policy_id"),
			ServerID:     parseUintQuery(c, "server_id"),
			SnapshotType: c.Query("snapshot_type"),
			Status:       c.Query("status"),
			Page:         params.Page,
			PageSize:     params.PageSize,
		}
		items, total, err := h.fimService().ListSnapshots(filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim snapshots"})
			return
		}
		c.JSON(http.StatusOK, PaginatedResponse{Total: int(total), Page: params.Page, PageSize: params.PageSize, TotalPages: calcTotalPages(int(total), params.PageSize), Data: items})
	}
}

func (h *Handler) GetFIMSnapshotDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}
		item, err := h.fimService().GetSnapshotDetail(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim snapshot detail"})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) ActivateFIMBaseline() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
			return
		}
		if err := h.fimService().ActivateBaseline(id, currentOperator(c)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate fim baseline"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) GetFIMDiffEvents() gin.HandlerFunc {
	return func(c *gin.Context) {
		params := Paginate(c)
		items, total, err := h.fimService().ListDiffEvents(params.Page, params.PageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim events"})
			return
		}
		c.JSON(http.StatusOK, PaginatedResponse{Total: int(total), Page: params.Page, PageSize: params.PageSize, TotalPages: calcTotalPages(int(total), params.PageSize), Data: items})
	}
}

func (h *Handler) DeleteFIMDiffEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		if err := h.fimService().DeleteDiffEvent(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fim event"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) GetFIMAlerts() gin.HandlerFunc {
	return func(c *gin.Context) {
		params := Paginate(c)
		items, total, err := h.fimService().ListAlerts(FIMAlertFilter{
			Status:   c.Query("status"),
			Severity: c.Query("severity"),
			Page:     params.Page,
			PageSize: params.PageSize,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim alerts"})
			return
		}
		c.JSON(http.StatusOK, PaginatedResponse{Total: int(total), Page: params.Page, PageSize: params.PageSize, TotalPages: calcTotalPages(int(total), params.PageSize), Data: items})
	}
}

func (h *Handler) GetFIMAlertDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
			return
		}
		item, err := h.fimService().GetAlertDetail(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fim alert detail"})
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func (h *Handler) DeleteFIMAlert() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
			return
		}
		if err := h.fimService().DeleteAlert(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete fim alert"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

func (h *Handler) AckFIMAlert() gin.HandlerFunc {
	return func(c *gin.Context) {
		updateFIMAlertStatus(c, h, "acknowledged")
	}
}

func (h *Handler) ResolveFIMAlert() gin.HandlerFunc {
	return func(c *gin.Context) {
		updateFIMAlertStatus(c, h, "resolved")
	}
}

func (h *Handler) CloseFIMAlert() gin.HandlerFunc {
	return func(c *gin.Context) {
		updateFIMAlertStatus(c, h, "closed")
	}
}

func updateFIMAlertStatus(c *gin.Context, h *Handler, status string) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}
	var req UpdateFIMAlertStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.fimService().UpdateAlertStatus(id, status, currentOperator(c), req.Comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fim alert status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func currentOperator(c *gin.Context) string {
	if value, exists := c.Get("username"); exists {
		if username, ok := value.(string); ok && username != "" {
			return username
		}
	}
	return "system"
}

func parseUintQuery(c *gin.Context, key string) uint {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

func calcTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}
	return totalPages
}
