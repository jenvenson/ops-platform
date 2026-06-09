package security

import (
	"strings"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestSplitReportFindings(t *testing.T) {
	vulns := []models.SecurityVulnerability{
		{ID: 1, FindingSource: "host-template", FindingFamily: "vulnerability", Severity: "high"},
		{ID: 2, Scanner: "vuln-matcher", FindingSource: "host-version-match", FindingFamily: "vulnerability", Severity: "medium"},
		{ID: 3, FindingSource: "asset-inventory", FindingFamily: "inventory", Severity: "info"},
		{ID: 4, Scanner: manualReviewScanner, FindingSource: hostManualConfirmedFindingSource, FindingFamily: "vulnerability", Severity: "high"},
	}

	groups := splitReportFindings(vulns)
	if len(groups.Confirmed) != 2 || groups.Confirmed[0].ID != 1 || groups.Confirmed[1].ID != 4 {
		t.Fatalf("unexpected confirmed findings: %#v", groups.Confirmed)
	}
	if len(groups.Candidate) != 1 || groups.Candidate[0].ID != 2 {
		t.Fatalf("unexpected candidate findings: %#v", groups.Candidate)
	}
	if len(groups.Inventory) != 1 || groups.Inventory[0].ID != 3 {
		t.Fatalf("unexpected inventory findings: %#v", groups.Inventory)
	}
}

func TestGenerateJSONReportIncludesFindingGroups(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	report := generateJSONReport(
		models.SecurityScanTask{ID: 7, Name: "host-scan", Target: "10.0.0.1", Status: "completed", CreatedAt: now},
		reportFindingGroups{
			Confirmed: []models.SecurityVulnerability{{ID: 1, Title: "confirmed", FindingSource: "host-template", RiskCategory: "CVE 风险", CreatedAt: now}},
			Candidate: []models.SecurityVulnerability{{ID: 2, Title: "candidate", FindingSource: "host-version-match", Confidence: "high", CandidateTier: "strong", CreatedAt: now}},
			Inventory: []models.SecurityVulnerability{{ID: 3, Title: "inventory", FindingFamily: "inventory", FindingSource: "asset-inventory", CreatedAt: now}},
		},
		reportSeverityCounts{High: 1},
		nil,
	)

	if report.Statistics.TotalVulnerabilities != 1 || report.Statistics.VerificationFindings != 1 || report.Statistics.CandidateFindings != 1 || report.Statistics.InventoryFindings != 1 {
		t.Fatalf("unexpected statistics: %#v", report.Statistics)
	}
	if len(report.Vulnerabilities) != 1 || len(report.VerificationFindings) != 1 || len(report.CandidateFindings) != 1 || len(report.InventoryFindings) != 1 {
		t.Fatalf("unexpected grouped report sections: %#v", report)
	}
}

func TestGenerateCSVReportIncludesCategoryColumns(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	csv := generateCSVReport(reportFindingGroups{
		Confirmed: []models.SecurityVulnerability{{
			ID: 1, IP: "10.0.0.1", Port: 22, Title: "ssh vuln", Severity: "high",
			FindingSource: "host-template", Confidence: "high", MatchMode: "template", RiskCategory: "CVE 风险",
			VulnURL: "ssh://10.0.0.1:22", CreatedAt: now,
		}},
		Candidate: []models.SecurityVulnerability{{
			ID: 2, IP: "10.0.0.1", Port: 3306, Title: "mysql candidate", Severity: "medium",
			FindingSource: "host-version-match", CandidateTier: "strong", Confidence: "high", MatchMode: "version-range", RiskCategory: "CVE 风险",
			VulnURL: "10.0.0.1:3306", CreatedAt: now,
		}},
	}, models.SecurityScanTask{Name: "host-scan"})

	for _, fragment := range []string{"报告分类", "待验证", "版本匹配", "ssh://10.0.0.1:22"} {
		if !strings.Contains(csv, fragment) {
			t.Fatalf("expected csv to contain %q, got %s", fragment, csv)
		}
	}
}

func TestReportFindingSourceCNAndMatchModeCN(t *testing.T) {
	if got := reportFindingSourceCN(hostManualConfirmedFindingSource); got != "人工确认" {
		t.Fatalf("expected manual confirmed finding source label, got %q", got)
	}
	if got := reportMatchModeCN(manualReviewMatchMode); got != "人工复核" {
		t.Fatalf("expected manual review match mode label, got %q", got)
	}
}
