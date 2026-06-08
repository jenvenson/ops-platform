package security

import (
	"testing"

	"github.com/edy/ops-platform/internal/models"
)

func TestClassifyMatchExact(t *testing.T) {
	matcher := NewVulnMatcher()
	vuln := &models.VulnerabilityDatabase{
		AffectedCPE: "cpe:2.3:a:oracle:mysql:*:*:*:*:*:*:*:*",
	}
	portInfo := &PortInfo{
		Service: "mysql",
		Version: "8.0.20",
		CPE:     "cpe:/a:mysql:mysql:8.0.20",
	}

	matchMode, confidence := matcher.classifyMatch(vuln, portInfo, false)
	if matchMode != "exact" {
		t.Fatalf("expected exact, got %s", matchMode)
	}
	if confidence != "high" {
		t.Fatalf("expected high confidence, got %s", confidence)
	}
}

func TestClassifyMatchVersionRange(t *testing.T) {
	matcher := NewVulnMatcher()
	vuln := &models.VulnerabilityDatabase{
		AffectedCPE:         "cpe:2.3:a:openbsd:openssh:*:*:*:*:*:*:*:*",
		VersionEndExcluding: "9.3",
	}
	portInfo := &PortInfo{
		Service: "ssh",
		Version: "8.9p1",
		CPE:     "cpe:2.3:a:openbsd:openssh:8.9p1:*:*:*:*:*:*:*",
	}

	matchMode, confidence := matcher.classifyMatch(vuln, portInfo, false)
	if matchMode != "version-range" {
		t.Fatalf("expected version-range, got %s", matchMode)
	}
	if confidence != "high" {
		t.Fatalf("expected high confidence, got %s", confidence)
	}
}

func TestClassifyMatchProductOnlyCPEWithStructuredVersionIsCandidate(t *testing.T) {
	matcher := NewVulnMatcher()
	vuln := &models.VulnerabilityDatabase{
		AffectedCPE:         "cpe:2.3:a:f5:nginx:*:*:*:*:*:*:*:*",
		VersionEndIncluding: "1.20.0",
	}
	portInfo := &PortInfo{
		Service: "nginx",
		CPE:     "cpe:/a:nginx:nginx",
	}

	matchMode, confidence := matcher.classifyMatch(vuln, portInfo, false)
	if matchMode != "fuzzy-product" {
		t.Fatalf("expected fuzzy-product, got %s", matchMode)
	}
	if confidence != "medium" {
		t.Fatalf("expected medium confidence, got %s", confidence)
	}
}

func TestClassifyMatchServiceFallbackIsLowConfidence(t *testing.T) {
	matcher := NewVulnMatcher()
	vuln := &models.VulnerabilityDatabase{
		Product: "openssh",
	}
	portInfo := &PortInfo{
		Service: "ssh",
		Version: "8.9p1",
	}

	matchMode, confidence := matcher.classifyMatch(vuln, portInfo, true)
	if matchMode != "fuzzy-product" {
		t.Fatalf("expected fuzzy-product, got %s", matchMode)
	}
	if confidence != "low" {
		t.Fatalf("expected low confidence, got %s", confidence)
	}
}

func TestDeduplicateResultsPrefersStrongerMatch(t *testing.T) {
	matcher := NewVulnMatcher()
	results := []VulnMatchResult{
		{
			CVEID:      "CVE-2023-28531",
			Severity:   "medium",
			CVSS:       5.3,
			MatchMode:  "fuzzy-product",
			Confidence: "low",
			MatchedOn:  "服务匹配: ssh",
		},
		{
			CVEID:      "CVE-2023-28531",
			Severity:   "medium",
			CVSS:       5.3,
			MatchMode:  "version-range",
			Confidence: "high",
			MatchedOn:  "版本 8.9p1 < 9.3",
		},
	}

	deduped := matcher.deduplicateResults(results)
	if len(deduped) != 1 {
		t.Fatalf("expected 1 result after dedupe, got %d", len(deduped))
	}
	if deduped[0].MatchMode != "version-range" {
		t.Fatalf("expected stronger version-range result to be kept, got %s", deduped[0].MatchMode)
	}
	if deduped[0].Confidence != "high" {
		t.Fatalf("expected stronger high confidence result to be kept, got %s", deduped[0].Confidence)
	}
}

func TestNewVulnMatchResultAnnotatesVersionEvidence(t *testing.T) {
	matcher := NewVulnMatcher()
	result := matcher.newVulnMatchResult(models.VulnerabilityDatabase{
		CVEID:               "CVE-2023-28531",
		VersionEndExcluding: "9.3",
	}, &PortInfo{
		Service: "ssh",
		Version: "8.9p1",
	}, "产品匹配: openssh", false)

	if result.MatchedOn != "产品匹配: openssh (版本: 8.9p1)" {
		t.Fatalf("unexpected matched_on annotation: %s", result.MatchedOn)
	}
}
