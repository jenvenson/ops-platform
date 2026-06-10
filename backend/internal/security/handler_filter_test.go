// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jenvenson/ops-platform/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func newSecurityDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(testDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("failed to create dry-run db: %v", err)
	}
	return db
}

type testDialector struct{}

func (testDialector) Name() string {
	return "test"
}

func (testDialector) Initialize(*gorm.DB) error {
	return nil
}

func (testDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

func (testDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (testDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (testDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	_, _ = writer.WriteString("?")
}

func (testDialector) QuoteTo(writer clause.Writer, str string) {
	_, _ = writer.WriteString("`")
	_, _ = writer.WriteString(str)
	_, _ = writer.WriteString("`")
}

func (testDialector) Explain(sql string, _ ...interface{}) string {
	return sql
}

func buildSecurityVulnWhere(t *testing.T, query *gorm.DB) string {
	t.Helper()

	whereClause, ok := query.Statement.Clauses["WHERE"]
	if !ok {
		t.Fatalf("expected WHERE clause to exist")
	}
	return fmt.Sprintf("%v", whereClause.Expression)
}

func newFilterContext(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/security/vulnerabilities?"+rawQuery, nil)
	return c
}

func TestApplyFindingSourceFilter(t *testing.T) {
	db := newSecurityDryRunDB(t)

	where := buildSecurityVulnWhere(t, applyFindingSourceFilter(db.Model(&models.SecurityVulnerability{}), "web-rule"))
	if !strings.Contains(where, "finding_source") {
		t.Fatalf("expected where clause to include finding_source filter, got %s", where)
	}
}

func TestApplyMatchModeFilter(t *testing.T) {
	db := newSecurityDryRunDB(t)

	where := buildSecurityVulnWhere(t, applyMatchModeFilter(db.Model(&models.SecurityVulnerability{}), "version-range"))
	if !strings.Contains(where, "match_mode") {
		t.Fatalf("expected where clause to include match_mode filter, got %s", where)
	}
}

func TestApplyHasKnowledgeFilterTrue(t *testing.T) {
	db := newSecurityDryRunDB(t)

	where := buildSecurityVulnWhere(t, applyHasKnowledgeFilter(db.Model(&models.SecurityVulnerability{}), "true"))
	for _, fragment := range []string{"vuln_db_id", "primary_cve_id", "cve_id", "cnvd_id", "cnnvd_id", "cncve_id"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("expected where clause to include %s in knowledge filter, got %s", fragment, where)
		}
	}
}

func TestApplyHasKnowledgeFilterFalse(t *testing.T) {
	db := newSecurityDryRunDB(t)

	where := buildSecurityVulnWhere(t, applyHasKnowledgeFilter(db.Model(&models.SecurityVulnerability{}), "false"))
	for _, fragment := range []string{"vuln_db_id", "primary_cve_id", "cve_id", "cnvd_id", "cnnvd_id", "cncve_id"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("expected where clause to include %s in inverse knowledge filter, got %s", fragment, where)
		}
	}
}

func TestApplyVulnerabilityQueryFiltersReadsNewParams(t *testing.T) {
	db := newSecurityDryRunDB(t)
	c := newFilterContext("finding_source=web-rule&confidence=medium&match_mode=rule&has_knowledge=true")

	where := buildSecurityVulnWhere(t, applyVulnerabilityQueryFilters(db.Model(&models.SecurityVulnerability{}), c))
	for _, fragment := range []string{"finding_source", "confidence", "match_mode", "vuln_db_id"} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("expected where clause to include %s, got %s", fragment, where)
		}
	}
}