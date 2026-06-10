// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"testing"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestBuildConfirmedVulnerabilityFromCandidate(t *testing.T) {
	firstTaskID := uint(3)
	lastTaskID := uint(7)
	assetID := uint(11)
	vulnDBID := uint(19)
	candidate := models.SecurityVulnerability{
		ID:            42,
		TaskID:        7,
		FirstTaskID:   &firstTaskID,
		LastTaskID:    &lastTaskID,
		AssetID:       assetID,
		IP:            "10.0.0.8",
		Port:          3306,
		Protocol:      "tcp",
		Severity:      "high",
		CVSSScore:     8.8,
		CVEID:         "CVE-2024-0001",
		PrimaryCVEID:  "CVE-2024-0001",
		Title:         "mysql candidate",
		Description:   "candidate description",
		VulnType:      "info",
		Solution:      "upgrade",
		MatchedOn:     "CPE匹配 mysql 8.0.0 < 8.0.36",
		ExploitPrereq: "reachable from jump host",
		Scanner:       "vuln-matcher",
		ScanMethod:    "版本匹配",
		VulnURL:       "10.0.0.8:3306",
		FindingSource: "host-version-match",
		FindingFamily: "vulnerability",
		Confidence:    "medium",
		VulnDBID:      &vulnDBID,
		MatchMode:     "version-range",
		Payload:       "n/a",
		Request:       "request",
		Response:      "response",
		ReferenceURL:  "https://example.test/CVE-2024-0001",
		Status:        "acknowledged",
		Priority:      "high",
		FalsePositive: false,
		ReviewStatus:  "confirmed",
		ReviewNote:    "人工验证已确认版本仍受影响",
	}

	confirmed := buildConfirmedVulnerabilityFromCandidate(candidate)

	if confirmed.SourceVulnID == nil || *confirmed.SourceVulnID != candidate.ID {
		t.Fatalf("expected source_vuln_id %d, got %#v", candidate.ID, confirmed.SourceVulnID)
	}
	if confirmed.FindingSource != hostManualConfirmedFindingSource {
		t.Fatalf("expected finding_source %q, got %q", hostManualConfirmedFindingSource, confirmed.FindingSource)
	}
	if confirmed.MatchMode != manualReviewMatchMode {
		t.Fatalf("expected match_mode %q, got %q", manualReviewMatchMode, confirmed.MatchMode)
	}
	if confirmed.Scanner != manualReviewScanner || confirmed.ScanMethod != manualReviewScanMethod {
		t.Fatalf("unexpected scanner/scan method: %q / %q", confirmed.Scanner, confirmed.ScanMethod)
	}
	if confirmed.Status != candidate.Status || confirmed.Priority != candidate.Priority {
		t.Fatalf("expected disposition to be preserved, got status=%q priority=%q", confirmed.Status, confirmed.Priority)
	}
	if confirmed.AssetID != assetID || confirmed.TaskID != candidate.TaskID || confirmed.Port != candidate.Port {
		t.Fatalf("expected target fields to be preserved, got %#v", confirmed)
	}
}

func TestUniqueUintValues(t *testing.T) {
	got := uniqueUintValues([]uint{7, 0, 7, 8, 8, 9})
	want := []uint{7, 8, 9}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected values: got %v want %v", got, want)
		}
	}
}