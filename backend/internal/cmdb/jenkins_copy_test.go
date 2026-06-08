package cmdb

import (
	"strings"
	"testing"
)

func TestApplyTagReplacements_PreservesNumericSuffix(t *testing.T) {
	input := strings.Join([]string{
		`string(name: 'tag', defaultValue: '15s_dev', description: 'deploy tag')`,
		`string(`,
		`  name: 'tag',`,
		`  defaultValue: '15s_dev',`,
		`  description: 'deploy tag'`,
		`)`,
		`tag='15s_dev'`,
		`tag=15s_dev`,
	}, "\n")

	got := applyTagReplacements(input, []TagReplacementRule{{
		OldPattern: "15s_dev",
		NewPattern: "prod166",
	}})

	if strings.Contains(got, "prod\n") || strings.Contains(got, "prod'") && !strings.Contains(got, "prod166'") {
		t.Fatalf("numeric suffix was truncated: %s", got)
	}
	if strings.Count(got, "prod166") != 4 {
		t.Fatalf("expected all tag occurrences to be replaced, got: %s", got)
	}
}

func TestApplyTagReplacements_DoesNotTouchOtherValues(t *testing.T) {
	input := strings.Join([]string{
		`string(name: 'tag', defaultValue: '15s_dev', description: 'deploy tag')`,
		`string(name: 'branch', defaultValue: '15s_dev', description: 'branch')`,
		`image_tag="15s_dev"`,
	}, "\n")

	got := applyTagReplacements(input, []TagReplacementRule{{
		OldPattern: "15s_dev",
		NewPattern: "prod166",
	}})

	if !strings.Contains(got, `string(name: 'tag', defaultValue: 'prod166', description: 'deploy tag')`) {
		t.Fatalf("tag defaultValue was not replaced: %s", got)
	}
	if !strings.Contains(got, `string(name: 'branch', defaultValue: '15s_dev', description: 'branch')`) {
		t.Fatalf("non-tag defaultValue should stay unchanged: %s", got)
	}
	if !strings.Contains(got, `image_tag="15s_dev"`) {
		t.Fatalf("non-tag key should stay unchanged: %s", got)
	}
}

func TestNormalizeJobNameReplacements_LastRuleWinsForSameOldPattern(t *testing.T) {
	rules := []JobNameReplacementRule{
		{OldPattern: "fat-190", NewPattern: "prod-166"},
		{OldPattern: "fat-190", NewPattern: "prod166"},
		{OldPattern: "foo", NewPattern: "bar"},
	}

	got := normalizeJobNameReplacements(rules)
	if len(got) != 2 {
		t.Fatalf("expected 2 normalized rules, got %d", len(got))
	}
	if got[0].OldPattern != "fat-190" || got[0].NewPattern != "prod166" {
		t.Fatalf("expected explicit replacement to win, got %#v", got[0])
	}
	if got[1].OldPattern != "foo" || got[1].NewPattern != "bar" {
		t.Fatalf("unexpected trailing rule: %#v", got[1])
	}
}

func TestExtractAppNameFromJob(t *testing.T) {
	cases := []struct {
		name    string
		view    string
		jobName string
		want    string
	}{
		{
			name:    "strip direct suffix-style prefix",
			view:    "6f_dev-187",
			jobName: "6f_dev-technical-research",
			want:    "technical-research",
		},
		{
			name:    "strip normalized view prefix",
			view:    "prod-166",
			jobName: "prod166-technical-research",
			want:    "technical-research",
		},
		{
			name:    "no matching prefix keeps original job name",
			view:    "prod-166",
			jobName: "other-technical-research",
			want:    "other-technical-research",
		},
		{
			name:    "strip prefix from versioned view name",
			view:    "jlsf_dev-V2.4.0",
			jobName: "jlsf_dev-firmware-research",
			want:    "firmware-research",
		},
		{
			name:    "strip same env prefix plus numeric segment",
			view:    "mscs_dev-185",
			jobName: "myapp_dev-180-app-awd",
			want:    "app-awd",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractAppNameFromJob(tc.view, tc.jobName); got != tc.want {
				t.Fatalf("extractAppNameFromJob(%q, %q) = %q, want %q", tc.view, tc.jobName, got, tc.want)
			}
		})
	}
}

func TestExtractAppNameFromJobWithExplicitPrefix(t *testing.T) {
	got := extractAppNameFromJobWithPrefix("ts_dev-185", "190-firmware-research", "190-")
	if got != "firmware-research" {
		t.Fatalf("expected explicit prefix cleanup to win, got %q", got)
	}

	got = extractAppNameFromJobWithPrefix("fat-70-V2.5.1", "fat-70-V2.5.1-firmware-research", "V2.5.1-")
	if got != "firmware-research" {
		t.Fatalf("expected explicit version prefix cleanup to win after derived prefix, got %q", got)
	}

	got = extractAppNameFromJobWithPrefix("ts_dev-185", "190-firmware-research", "")
	if got != "190-firmware-research" {
		t.Fatalf("expected job name to stay unchanged without explicit prefix, got %q", got)
	}
}
