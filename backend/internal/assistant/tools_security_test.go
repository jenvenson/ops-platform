package assistant

import (
	"strings"
	"testing"

	"github.com/edy/ops-platform/internal/models"
)

func TestVulnerabilityQueryKeywordSkipsGenericPrompts(t *testing.T) {
	if keyword := vulnerabilityQueryKeyword("当前未处理的高危漏洞有多少"); keyword != "" {
		t.Fatalf("expected no keyword for generic vulnerability query, got %q", keyword)
	}
}

func TestSecurityTaskQueryKeywordSkipsGenericPrompts(t *testing.T) {
	if keyword := securityTaskQueryKeyword("最近有哪些失败扫描任务"); keyword != "" {
		t.Fatalf("expected no keyword for generic task query, got %q", keyword)
	}
}

func TestSummarizeSecurityScanTasks(t *testing.T) {
	summary := summarizeSecurityScanTasks("最近扫描异常集中在哪些目标", []models.SecurityScanTask{
		{Target: "10.0.0.1", Status: models.TaskStatusFailed},
		{Target: "10.0.0.1", Status: models.TaskStatusFailed},
		{Target: "10.0.0.2", Status: models.TaskStatusFailed},
	})
	if !strings.Contains(summary, "目标 10.0.0.1") {
		t.Fatalf("unexpected summary: %q", summary)
	}
}
