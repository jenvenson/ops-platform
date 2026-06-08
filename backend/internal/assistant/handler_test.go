package assistant

import "testing"

func TestNormalizeSessionPagination(t *testing.T) {
	page, limit := normalizeSessionPagination(0, 100)
	if page != 1 || limit != 50 {
		t.Fatalf("unexpected pagination result: page=%d limit=%d", page, limit)
	}
}

func TestBuildSessionTitle(t *testing.T) {
	got := buildSessionTitle("  请帮我查看今天生产环境的部署失败原因，并告诉我受影响应用  ")
	if got == "" {
		t.Fatal("expected non-empty title")
	}
	if got == "新会话" {
		t.Fatalf("expected message-derived title, got %q", got)
	}
}

func TestBuildSessionTitleFallsBackForBlankMessage(t *testing.T) {
	if got := buildSessionTitle("   "); got != "新会话" {
		t.Fatalf("expected fallback title, got %q", got)
	}
}

func TestUpdateSessionSummaryPrefersLatestRound(t *testing.T) {
	got := updateSessionSummary("旧摘要", "查看生产环境部署记录", "最近一次部署失败，应用 web-api 受影响。")
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	if got == "旧摘要" {
		t.Fatalf("expected updated summary, got %q", got)
	}
}

func TestUpdateSessionTitleKeepsExistingNonDefaultTitle(t *testing.T) {
	got := updateSessionTitle("生产部署排查", "查看今天部署")
	if got != "生产部署排查" {
		t.Fatalf("expected existing title to be kept, got %q", got)
	}
}

func TestSingleLine(t *testing.T) {
	got := singleLine("第一行\n第二行\t第三行")
	if got != "第一行 第二行 第三行" {
		t.Fatalf("unexpected singleLine result: %q", got)
	}
}
