package assistant

import (
	"strings"
	"testing"
	"time"

	"github.com/edy/ops-platform/internal/cmdb"
)

func TestServerQueryKeywordSkipsGenericPrompts(t *testing.T) {
	if keyword := serverQueryKeyword("当前离线主机有哪些"); keyword != "" {
		t.Fatalf("expected no keyword for generic server query, got %q", keyword)
	}
	if keyword := serverQueryKeyword("当前主机分布情况"); keyword != "" {
		t.Fatalf("expected no keyword for server overview prompt, got %q", keyword)
	}
}

func TestApplicationQueryKeywordSkipsGenericPrompts(t *testing.T) {
	if keyword := applicationQueryKeyword("哪些应用最近部署失败较多"); keyword != "" {
		t.Fatalf("expected no keyword for generic application query, got %q", keyword)
	}
}

func TestSummarizeApplications(t *testing.T) {
	summary := summarizeApplications("最近哪些应用发布最频繁", []applicationCountStat{
		{AppName: "web-api", Count: 5},
		{AppName: "console-ui", Count: 3},
	})
	if summary != "结论：最近发布最频繁的是应用 web-api，共 5 次。" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestSummarizeDeletedAbnormalServers(t *testing.T) {
	deletedAt := time.Date(2026, 3, 28, 9, 30, 0, 0, time.Local)
	summary := summarizeDeletedAbnormalServers([]cmdb.Server{
		{Hostname: "srv-offline-1", DeletedAt: &deletedAt},
	}, serverInventoryStats{
		Total: 3,
		OSCounts: map[string]int{
			"Ubuntu 22.04": 2,
			"CentOS 7":     1,
		},
		ArchCounts: map[string]int{
			"x86_64": 2,
			"arm64":  1,
		},
	})

	if !containsAll(summary, []string{
		"最近删除的异常主机有 1 台",
		"当前共有 3 台主机",
		"Ubuntu 22.04 2 台",
		"x86_64 2 台",
		"2026-03-28 09:30",
	}) {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestSummarizeCurrentServerOverview(t *testing.T) {
	summary := summarizeCurrentServerOverview(serverInventoryStats{
		Total: 3,
		StatusCounts: map[string]int{
			"online":  2,
			"offline": 1,
		},
		OSCounts: map[string]int{
			"Ubuntu 22.04": 2,
			"CentOS 7":     1,
		},
		ArchCounts: map[string]int{
			"x86_64": 2,
			"arm64":  1,
		},
	})

	if !containsAll(summary, []string{
		"当前主机分布情况如下",
		"当前共有 3 台主机",
		"online 2 台",
		"offline 1 台",
		"Ubuntu 22.04 2 台",
		"arm64 1 台",
	}) {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func containsAll(text string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
