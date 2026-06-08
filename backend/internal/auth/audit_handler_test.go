package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edy/ops-platform/internal/models"
	"github.com/gin-gonic/gin"
)

func TestExportPlatformAuditLogsMissingType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit/export", nil)

	ExportPlatformAuditLogs()(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "missing export type") {
		t.Fatalf("expected missing export type error, got %s", w.Body.String())
	}
}

func TestExportPlatformAuditLogsUnsupportedType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit/export?type=invalid", nil)

	ExportPlatformAuditLogs()(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unsupported export type") {
		t.Fatalf("expected unsupported export type error, got %s", w.Body.String())
	}
}

func TestExportPlatformAuditLogsArchiveInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit/export?type=archive&archive_type=xxx", nil)

	ExportPlatformAuditLogs()(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unsupported archive type") {
		t.Fatalf("expected unsupported archive type error, got %s", w.Body.String())
	}
}

func TestBuildPlatformAuditExportFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/admin/audit/export?type=archive&archive_type=operation&start=2026-03-01&end=2026-03-31",
		nil,
	)

	filename := buildPlatformAuditExportFilename(c, "archive")
	if !strings.HasPrefix(filename, "platform-audit-archive-operation-2026-03-01_to_2026-03-31-") {
		t.Fatalf("unexpected filename prefix: %s", filename)
	}
	if !strings.HasSuffix(filename, ".csv") {
		t.Fatalf("expected csv suffix, got %s", filename)
	}
}

func TestSanitizeAuditFilenameSegment(t *testing.T) {
	value := sanitizeAuditFilenameSegment("  A/B\\C:d_e.1 @#中文  ")
	if value != "a-b-c-d-e-1" {
		t.Fatalf("unexpected sanitized value: %s", value)
	}
}

func TestBuildAuditExportDateRangeSegment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit/export?start=2020-01-01&end=2026-03-31", nil)

	segment := buildAuditExportDateRangeSegment(c)
	if segment != "2020-01-01_to_2026-03-31" {
		t.Fatalf("unexpected date range segment: %s", segment)
	}
}

func TestBuildAccessLogCSVRowsEmpty(t *testing.T) {
	rows := buildAccessLogCSVRows(nil)
	if len(rows) != 1 {
		t.Fatalf("expected only header row for empty data, got %d rows", len(rows))
	}
	if got := strings.Join(rows[0], ","); !strings.Contains(got, "请求路径") {
		t.Fatalf("unexpected header row: %s", got)
	}
}

func TestBuildAccessLogCSVRowsWithData(t *testing.T) {
	rows := buildAccessLogCSVRows([]models.PlatformAccessLog{
		{
			Username:        "admin",
			MenuTitle:       "平台审计",
			RequestPath:     "/api/admin/audit/export",
			RequestMethod:   "GET",
			RequestIP:       "127.0.0.1",
			OperationStatus: "success",
			DurationMS:      12,
		},
	})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1][2] != "平台审计" {
		t.Fatalf("expected menu title column to be 平台审计, got %s", rows[1][2])
	}
	if rows[1][7] != "12ms" {
		t.Fatalf("expected duration column to be 12ms, got %s", rows[1][7])
	}
}
