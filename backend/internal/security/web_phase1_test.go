package security

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

func TestPhase1WebFindingKeyIncludesURL(t *testing.T) {
	vuln := models.SecurityVulnerability{
		IP:            "10.0.0.8",
		Port:          443,
		FindingSource: "web-template",
		PrimaryCVEID:  "CVE-2024-0001",
		MatchMode:     "template",
	}

	key := phase1WebFindingKey(vuln, "https://demo.example.com/admin")

	for _, part := range []string{"web-template", "10.0.0.8", "443", "CVE-2024-0001", "template", "https://demo.example.com/admin"} {
		if !strings.Contains(key, part) {
			t.Fatalf("expected key %q to contain %q", key, part)
		}
	}
}

func TestPhase1WebAuthSnapshotOmitsSecrets(t *testing.T) {
	config := &WebScanConfig{
		AuthMode:      "login-token",
		Credential:    "top-secret-token",
		Username:      "scanner",
		Password:      "super-secret-password",
		LoginURL:      "https://demo.example.com/api/login",
		LoginMethod:   "POST",
		UsernameField: "user",
		PasswordField: "pass",
		TokenField:    "token",
	}

	snapshot := phase1WebAuthSnapshot("https://demo.example.com", config)
	if snapshot == nil {
		t.Fatal("expected auth snapshot")
	}
	if got := snapshot["auth_mode"]; got != "login-token" {
		t.Fatalf("expected auth_mode login-token, got %#v", got)
	}
	if got := snapshot["username"]; got != "scanner" {
		t.Fatalf("expected username scanner, got %#v", got)
	}
	if got := snapshot["has_credential"]; got != true {
		t.Fatalf("expected has_credential true, got %#v", got)
	}
	if _, exists := snapshot["password"]; exists {
		t.Fatal("did not expect password in auth snapshot")
	}
	if _, exists := snapshot["credential"]; exists {
		t.Fatal("did not expect credential in auth snapshot")
	}
}

func TestPhase1WebConfigSnapshotIncludesScanProfile(t *testing.T) {
	raw := phase1WebConfigSnapshot("https://demo.example.com", []string{"https://demo.example.com"}, &WebScanConfig{
		ScanProfile:       "deep",
		DiscoveryMode:     "browser",
		DiscoveryMaxURLs:  40,
		DiscoveryMaxDepth: 2,
	})
	if raw == nil {
		t.Fatal("expected config snapshot")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &payload); err != nil {
		t.Fatalf("expected valid json, got error: %v", err)
	}

	if got := payload["scan_profile"]; got != "deep" {
		t.Fatalf("expected scan_profile deep, got %#v", got)
	}
}

func TestPhase1WebCompletionMessageMentionsBrowserFallback(t *testing.T) {
	message := phase1WebCompletionMessage(12, 8, 4, 2, 1, 2, 0, []map[string]interface{}{
		phase1WebDiscoveryWarning("http://demo.example.com/app", "browser", "http", nil),
	})

	for _, fragment := range []string{"发现 12 个入口", "扫描 8 个目标", "跳过 4 个低优先级目标", "2 个低价值目标仅执行规则检测", "browser helper 不可达回退为 HTTP 发现"} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected message %q to contain %q", message, fragment)
		}
	}
}

func TestPhase1WebSummarySnapshotIncludesDiscoveryWarnings(t *testing.T) {
	raw := phase1WebSummarySnapshot(
		[]string{"http://demo.example.com/app"},
		12,
		8,
		4,
		2,
		"browser",
		[]map[string]interface{}{
			phase1WebDiscoveryWarning("http://demo.example.com/app", "browser", "http", assertErr("helper unavailable")),
		},
		1,
		2,
		0,
		time.Unix(1, 0).UTC(),
	)
	if raw == nil {
		t.Fatal("expected summary snapshot")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &payload); err != nil {
		t.Fatalf("expected valid json, got error: %v", err)
	}

	if got := payload["browser_fallback_count"]; got != float64(1) {
		t.Fatalf("expected browser_fallback_count 1, got %#v", got)
	}
	if got := payload["rule_only_target_count"]; got != float64(2) {
		t.Fatalf("expected rule_only_target_count 2, got %#v", got)
	}
	if got := payload["discovery_mode"]; got != "browser" {
		t.Fatalf("expected discovery_mode browser, got %#v", got)
	}
	warnings, ok := payload["discovery_warnings"].([]interface{})
	if !ok || len(warnings) != 1 {
		t.Fatalf("expected one discovery warning, got %#v", payload["discovery_warnings"])
	}
}

func TestPhase1WebTargetSnapshotIncludesRuleOnlyCount(t *testing.T) {
	raw := phase1WebTargetSnapshotWithMeta(
		[]string{"http://demo.example.com/app"},
		[]DiscoveredTarget{
			{URL: "http://demo.example.com/app", Kind: "page", Source: "entry"},
			{URL: "http://demo.example.com/base/custom/get", Kind: "api", Source: "browser-request"},
		},
		3,
		1,
		1,
		nil,
	)
	if raw == nil {
		t.Fatal("expected target snapshot")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &payload); err != nil {
		t.Fatalf("expected valid json, got error: %v", err)
	}

	if got := payload["rule_only_target_count"]; got != float64(1) {
		t.Fatalf("expected rule_only_target_count 1, got %#v", got)
	}
	if got := payload["skipped_target_count"]; got != float64(1) {
		t.Fatalf("expected skipped_target_count 1, got %#v", got)
	}
}

func assertErr(message string) error {
	return &staticError{message: message}
}

type staticError struct {
	message string
}

func (e *staticError) Error() string {
	return e.message
}
