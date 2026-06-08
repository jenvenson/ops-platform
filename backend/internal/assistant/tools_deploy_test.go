package assistant

import (
	"strings"
	"testing"

	"github.com/edy/ops-platform/internal/cmdb"
	"github.com/edy/ops-platform/internal/models"
)

func TestSummarizeDeployRecords(t *testing.T) {
	records := []cmdb.DeployRecord{
		{AppName: "web-api", Status: "failed"},
		{AppName: "web-api", Status: "failed"},
		{AppName: "console-ui", Status: "running"},
	}

	t.Run("failed deploys lead with failure conclusion", func(t *testing.T) {
		summary := summarizeDeployRecords("最近有哪些失败部署", records)
		if !strings.Contains(summary, "最近有 2 条失败部署") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("focused app conclusion highlights concentration", func(t *testing.T) {
		summary := summarizeDeployRecords("最近部署异常集中在哪些应用", records)
		if !strings.Contains(summary, "异常更集中在应用 web-api") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "2 条失败记录") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("falls back to overall concentration when there is no failure", func(t *testing.T) {
		summary := summarizeDeployRecords("最近部署异常集中在哪些应用", []cmdb.DeployRecord{
			{AppName: "web-api", Status: "success"},
			{AppName: "web-api", Status: "running"},
			{AppName: "console-ui", Status: "success"},
		})
		if !strings.Contains(summary, "最近没有明显失败集中") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})
}

func TestSummarizeArchiveRecords(t *testing.T) {
	records := []cmdb.ArchiveRecord{
		{AppName: "web-api", Status: "failed"},
		{AppName: "console-ui", Status: "success"},
	}

	t.Run("failed archives lead with risk conclusion", func(t *testing.T) {
		summary := summarizeArchiveRecords("最近有哪些归档失败", []cmdb.ArchiveRecord{
			{AppName: "web-api", Status: "failed"},
			{AppName: "web-api", Status: "failed"},
			{AppName: "console-ui", Status: "failed"},
		})
		if !strings.Contains(summary, "最近有 3 条归档失败") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "失败主要集中在 web-api（2 条）、console-ui（1 条）") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("download query returns operation guidance", func(t *testing.T) {
		summary := summarizeArchiveRecords("如何下载某次归档产物", records)
		if !strings.Contains(summary, "需要从归档历史页面的下载地址进入") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "复制链接后在新窗口打开下载") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("normal completion query highlights failed app first", func(t *testing.T) {
		summary := summarizeArchiveRecords("最近归档是否正常完成", records)
		if !strings.Contains(summary, "并非全部正常完成") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "优先关注应用 web-api") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("in progress query highlights active app", func(t *testing.T) {
		summary := summarizeArchiveRecords("最近归档是否正常完成", []cmdb.ArchiveRecord{
			{AppName: "web-api", Status: "running"},
			{AppName: "web-api", Status: "queued"},
			{AppName: "console-ui", Status: "success"},
		})
		if !strings.Contains(summary, "仍有 2 条任务在处理中") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "主要集中在应用 web-api") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("successful query highlights dominant app", func(t *testing.T) {
		summary := summarizeArchiveRecords("最近归档是否正常完成", []cmdb.ArchiveRecord{
			{AppName: "web-api", Status: "success"},
			{AppName: "web-api", Status: "success"},
			{AppName: "console-ui", Status: "success"},
		})
		if !strings.Contains(summary, "都已正常完成") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "主要集中在应用 web-api") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})
}

func TestFilterActionsForKnowledgeQueryPrefersArchivedPageForArtifactDownload(t *testing.T) {
	actions := []Action{
		{Type: "navigate", Label: "打开归档打包", Path: "/deploy/archive"},
		{Type: "navigate", Label: "打开归档历史", Path: "/deploy/archived"},
	}

	filtered := filterActionsForKnowledgeQuery("如何下载某次归档产物", actions)
	if len(filtered) != 1 || filtered[0].Path != "/deploy/archived" {
		t.Fatalf("expected archived page action, got %#v", filtered)
	}
}

func TestArchiveQueryKeywordSkipsGenericRecentFailurePrompt(t *testing.T) {
	if keyword := archiveQueryKeyword("最近有哪些归档失败"); keyword != "" {
		t.Fatalf("expected no keyword for generic recent failure query, got %q", keyword)
	}

	if keyword := archiveQueryKeyword("最近有哪些 web-api 归档失败"); keyword == "" || strings.Contains(keyword, "最近") || strings.Contains(keyword, "哪些") {
		t.Fatalf("expected a specific app keyword to survive, got %q", keyword)
	}
}

func TestSummarizeAggregatedHistory(t *testing.T) {
	t.Run("failed aggregate query is grouped by project", func(t *testing.T) {
		summary := summarizeAggregatedHistory("最近有哪些聚合失败", []models.AggregatedHistory{
			{ProjectName: "ops-core", Status: "failed"},
			{ProjectName: "ops-core", Status: "failed"},
			{ProjectName: "ops-ui", Status: "failed"},
		})
		if !strings.Contains(summary, "最近有 3 条聚合失败") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "失败主要集中在 ops-core（2 条）、ops-ui（1 条）") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})

	t.Run("download aggregate query returns operation guidance", func(t *testing.T) {
		summary := summarizeAggregatedHistory("如何从下载地址下载聚合包", []models.AggregatedHistory{
			{ProjectName: "ops-core", Status: "completed"},
		})
		if !strings.Contains(summary, "需要从聚合历史页面进入") {
			t.Fatalf("unexpected summary: %q", summary)
		}
		if !strings.Contains(summary, "在下载地址中直接点击下载链接下载") {
			t.Fatalf("unexpected summary: %q", summary)
		}
	})
}

func TestAggregateQueryKeywordSkipsGenericRecentFailurePrompt(t *testing.T) {
	if keyword := aggregateQueryKeyword("最近有哪些聚合失败"); keyword != "" {
		t.Fatalf("expected no keyword for generic aggregate failure query, got %q", keyword)
	}
}
