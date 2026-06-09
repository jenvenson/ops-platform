package security

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestBuildTaskUpdatesStripsRunOnlyFields(t *testing.T) {
	taskUpdates := buildTaskUpdates(map[string]interface{}{
		"status":           "running",
		"message":          "scanning",
		"phase":            "verification",
		"target_snapshot":  `{"expanded_count":2}`,
		"summary_snapshot": `{"high_risk":1}`,
	})

	if taskUpdates["status"] != "running" {
		t.Fatalf("expected status to remain on task updates, got %#v", taskUpdates["status"])
	}
	for _, key := range []string{"phase", "target_snapshot", "summary_snapshot"} {
		if _, exists := taskUpdates[key]; exists {
			t.Fatalf("expected %s to be removed from task updates", key)
		}
	}
}

func TestPhase1FindingKeyUsesStableFields(t *testing.T) {
	vuln := models.SecurityVulnerability{
		IP:            "10.0.0.8",
		Port:          3306,
		FindingSource: "host-version-match",
		PrimaryCVEID:  "CVE-2020-14641",
		MatchMode:     "version-range",
	}

	key := phase1FindingKey(vuln)

	for _, part := range []string{"host-version-match", "10.0.0.8", "3306", "CVE-2020-14641", "version-range"} {
		if !strings.Contains(key, part) {
			t.Fatalf("expected key %q to contain %q", key, part)
		}
	}
}

func TestPhase1TrimTextSanitizesInvalidUTF8AndRespectsLimit(t *testing.T) {
	raw := "abc" + string([]byte{0xe3, 0x80}) + "中"
	got := phase1TrimText(raw, 5)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8, got %q", got)
	}
	if len(got) > 5 {
		t.Fatalf("expected trimmed text within limit, got %d bytes", len(got))
	}
}
