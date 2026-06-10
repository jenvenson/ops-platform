// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"testing"
	"time"
)

func TestIsStaticDiscoveryAssetSkipsDefaultBrandingDownloads(t *testing.T) {
	targets := []string{
		"http://198.51.100.1/file-manage/v2/downloadGet?uuid=default-favorite-logo",
		"http://198.51.100.1/file-manage/v2/downloadGet?uuid=default-login-background",
		"http://198.51.100.1/base/user/get/watermark",
	}

	for _, target := range targets {
		if !isStaticDiscoveryAsset(target) {
			t.Fatalf("expected %q to be treated as low-value discovery asset", target)
		}
	}
}

func TestPrioritizeVerificationTargetsPrefersAPIsWithinLimit(t *testing.T) {
	plan, skipped := prioritizeVerificationTargets(nil, []DiscoveredTarget{
		{URL: "http://demo.example.com/login", Kind: "page", Depth: 1, Source: "browser-frame"},
		{URL: "http://demo.example.com/base/custom/get", Kind: "api", Depth: 1, Source: "browser-request"},
		{URL: "http://demo.example.com/base/common/get-func", Kind: "api", Depth: 1, Source: "browser-request"},
		{URL: "http://demo.example.com/web_01", Kind: "page", Depth: 0, Source: "entry"},
	}, 3)

	if skipped != 1 {
		t.Fatalf("expected one target to be skipped, got %d", skipped)
	}
	if len(plan) != 3 {
		t.Fatalf("expected 3 prioritized targets, got %d", len(plan))
	}
	if plan[0].Source != "entry" {
		t.Fatalf("expected entry target first, got %+v", plan[0])
	}
	if plan[1].Kind != "api" || plan[2].Kind != "api" {
		t.Fatalf("expected API targets to outrank low-value pages, got %+v", plan)
	}
}

func TestPrioritizeVerificationTargetsDeepScanKeepsPageTargetsCompetitive(t *testing.T) {
	plan, skipped := prioritizeVerificationTargets(&WebScanConfig{ScanProfile: "deep"}, []DiscoveredTarget{
		{URL: "http://demo.example.com/page-a", Kind: "page", Depth: 1, Source: "browser-frame"},
		{URL: "http://demo.example.com/page-b", Kind: "page", Depth: 2, Source: "browser-dom"},
		{URL: "http://demo.example.com/base/common/get-func", Kind: "api", Depth: 1, Source: "browser-request"},
	}, 3)

	if skipped != 0 {
		t.Fatalf("expected no skipped targets, got %d", skipped)
	}
	if len(plan) != 3 {
		t.Fatalf("expected all deep-scan targets to remain, got %d", len(plan))
	}
	if plan[1].Kind != "page" {
		t.Fatalf("expected deep scan to keep page targets competitive, got %+v", plan)
	}
}

func TestShouldRunFullNucleiForTargetDowngradesPagesAndAgreementEndpoints(t *testing.T) {
	cases := []struct {
		name string
		item DiscoveredTarget
		want bool
	}{
		{
			name: "page target",
			item: DiscoveredTarget{URL: "http://demo.example.com/web_01", Kind: "page", Source: "entry"},
			want: false,
		},
		{
			name: "agreement api",
			item: DiscoveredTarget{URL: "http://demo.example.com/base/agreement/get", Kind: "api", Source: "browser-request"},
			want: false,
		},
		{
			name: "term api",
			item: DiscoveredTarget{URL: "http://demo.example.com/base/common/get-term?app_id=app-base", Kind: "api", Source: "browser-request"},
			want: false,
		},
		{
			name: "plugin installed list api",
			item: DiscoveredTarget{URL: "http://demo.example.com/base/open/plugin/installed/list", Kind: "api", Source: "browser-request"},
			want: false,
		},
		{
			name: "verify type api",
			item: DiscoveredTarget{URL: "http://demo.example.com/base/user/verify/type/web_01", Kind: "api", Source: "browser-request"},
			want: false,
		},
		{
			name: "business api",
			item: DiscoveredTarget{URL: "http://demo.example.com/base/common/get-func", Kind: "api", Source: "browser-request"},
			want: true,
		},
	}

	for _, tc := range cases {
		if got := shouldRunFullNucleiForTarget(nil, tc.item); got != tc.want {
			t.Fatalf("%s: expected %v, got %v for %+v", tc.name, tc.want, got, tc.item)
		}
	}
}

func TestShouldRunFullNucleiForTargetDeepScanDisablesRuleOnlyDowngrades(t *testing.T) {
	config := &WebScanConfig{ScanProfile: "deep"}
	cases := []DiscoveredTarget{
		{URL: "http://demo.example.com/web_01", Kind: "page", Source: "entry"},
		{URL: "http://demo.example.com/base/agreement/get", Kind: "api", Source: "browser-request"},
		{URL: "http://demo.example.com/base/open/plugin/installed/list", Kind: "api", Source: "browser-request"},
		{URL: "http://demo.example.com/base/user/verify/type/web_01", Kind: "api", Source: "browser-request"},
	}

	for _, item := range cases {
		if !shouldRunFullNucleiForTarget(config, item) {
			t.Fatalf("expected deep scan to keep full nuclei for %+v", item)
		}
	}
}

func TestWebVerificationNucleiTimeoutShortensOnlyKnownHighValueAPIs(t *testing.T) {
	if got := webVerificationNucleiTimeout(nil, DiscoveredTarget{
		URL:    "http://demo.example.com/base/common/get-func",
		Kind:   "api",
		Source: "browser-request",
	}); got != 20*time.Second {
		t.Fatalf("expected get-func timeout 20s, got %s", got)
	}

	if got := webVerificationNucleiTimeout(nil, DiscoveredTarget{
		URL:    "http://demo.example.com/base/custom/get",
		Kind:   "api",
		Source: "browser-request",
	}); got != 20*time.Second {
		t.Fatalf("expected custom/get timeout 20s, got %s", got)
	}

	if got := webVerificationNucleiTimeout(nil, DiscoveredTarget{
		URL:    "http://demo.example.com/base/agreement/get",
		Kind:   "api",
		Source: "browser-request",
	}); got != nucleiCommandTimeout() {
		t.Fatalf("expected agreement timeout to keep default command timeout %s, got %s", nucleiCommandTimeout(), got)
	}
}

func TestWebVerificationNucleiTimeoutDeepScanRestoresDefaultBudget(t *testing.T) {
	config := &WebScanConfig{ScanProfile: "deep"}
	for _, item := range []DiscoveredTarget{
		{URL: "http://demo.example.com/base/common/get-func", Kind: "api", Source: "browser-request"},
		{URL: "http://demo.example.com/base/custom/get", Kind: "api", Source: "browser-request"},
		{URL: "http://demo.example.com/base/agreement/get", Kind: "api", Source: "browser-request"},
	} {
		if got := webVerificationNucleiTimeout(config, item); got != nucleiCommandTimeout() {
			t.Fatalf("expected deep scan timeout %s, got %s for %+v", nucleiCommandTimeout(), got, item)
		}
	}
}

func TestVerificationTargetLimitDefaultsToEight(t *testing.T) {
	if got := verificationTargetLimit(nil); got != 8 {
		t.Fatalf("expected nil config limit to default to 8, got %d", got)
	}
	if got := verificationTargetLimit(&WebScanConfig{}); got != 8 {
		t.Fatalf("expected empty config limit to default to 8, got %d", got)
	}
	if got := verificationTargetLimit(&WebScanConfig{VerificationMaxTargets: 5}); got != 5 {
		t.Fatalf("expected explicit limit 5, got %d", got)
	}
}

func TestApplyWebScanProfileDefaults(t *testing.T) {
	deep := &WebScanConfig{ScanProfile: "deep"}
	applyWebScanProfileDefaults(deep)
	if deep.ScanProfile != "deep" {
		t.Fatalf("expected deep profile, got %q", deep.ScanProfile)
	}
	if deep.DiscoveryMaxURLs != 40 {
		t.Fatalf("expected deep discovery max urls 40, got %d", deep.DiscoveryMaxURLs)
	}
	if deep.VerificationMaxTargets != 12 {
		t.Fatalf("expected deep verification max targets 12, got %d", deep.VerificationMaxTargets)
	}

	standard := &WebScanConfig{}
	applyWebScanProfileDefaults(standard)
	if standard.ScanProfile != "standard" {
		t.Fatalf("expected standard profile, got %q", standard.ScanProfile)
	}
	if standard.DiscoveryMaxURLs != 25 {
		t.Fatalf("expected standard discovery max urls 25, got %d", standard.DiscoveryMaxURLs)
	}
	if standard.VerificationMaxTargets != 8 {
		t.Fatalf("expected standard verification max targets 8, got %d", standard.VerificationMaxTargets)
	}
}