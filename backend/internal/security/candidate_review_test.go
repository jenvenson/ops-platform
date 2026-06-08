package security

import (
	"testing"

	"github.com/edy/ops-platform/internal/models"
)

func TestIsCandidateHostVersionMatch(t *testing.T) {
	tests := []struct {
		name     string
		vuln     models.SecurityVulnerability
		expected bool
	}{
		{
			name:     "host version match is candidate",
			vuln:     models.SecurityVulnerability{Scanner: "vuln-matcher", FindingSource: "host-version-match"},
			expected: true,
		},
		{
			name:     "host template is not candidate",
			vuln:     models.SecurityVulnerability{Scanner: "nuclei", FindingSource: "host-template"},
			expected: false,
		},
		{
			name:     "manual confirmed finding is not candidate",
			vuln:     models.SecurityVulnerability{Scanner: manualReviewScanner, FindingSource: hostManualConfirmedFindingSource},
			expected: false,
		},
		{
			name:     "inventory is not candidate",
			vuln:     models.SecurityVulnerability{FindingSource: "asset-inventory", FindingFamily: "inventory"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCandidateHostVersionMatch(tt.vuln); got != tt.expected {
				t.Fatalf("isCandidateHostVersionMatch() = %v, want %v", got, tt.expected)
			}
		})
	}
}
