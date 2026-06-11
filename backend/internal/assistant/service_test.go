// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"context"
	"strings"
	"testing"

	"github.com/jenvenson/ops-platform/pkg/config"
)

func TestNewServiceKeepsModelDisabledByDefault(t *testing.T) {
	service := NewService(&config.Config{
		Assistant: config.AssistantConfig{
			Enabled:         false,
			Provider:        "ollama",
			OllamaBaseURL:   "http://docker-host:11434",
			OllamaChatModel: "qwen3:8b",
		},
	})
	if service.chatProvider != nil {
		t.Fatalf("expected chat provider to stay disabled, got %#v", service.chatProvider)
	}
}

func TestClassifyKnowledgeQuestion(t *testing.T) {
	cases := map[string]string{
		"运维小助手一期验收标准是什么": "acceptance",
		"推荐开发顺序是什么":      "roadmap",
		"一期范围包括什么":       "scope",
		"整体架构怎么设计":       "architecture",
		"如何使用平台":         "onboarding",
		"如何配置运维小助手":      "howto",
		"查看监控大屏":         "howto",
		"查看部署记录":         "howto",
	}

	for input, want := range cases {
		if got := classifyKnowledgeQuestion(input); got != want {
			t.Fatalf("input %q: want %q, got %q", input, want, got)
		}
	}
}

func TestClassifyAssistantIntent(t *testing.T) {
	intent := classifyAssistantIntent("查看最近告警状态")
	if intent.Name != "readonly_query" {
		t.Fatalf("expected readonly_query, got %#v", intent)
	}
	if intent.SubIntent != "alert_event_query" {
		t.Fatalf("expected alert_event_query, got %#v", intent)
	}
	if !intent.NeedTools {
		t.Fatalf("expected readonly intent to require tools, got %#v", intent)
	}

	intent = classifyAssistantIntent("最新告警动作")
	if intent.Name != "readonly_query" || intent.SubIntent != "alert_event_query" || !intent.NeedTools {
		t.Fatalf("expected latest alert action to use readonly alert query, got %#v", intent)
	}
}

func TestGenerateReplyBuildsCompatibleDecision(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			routes: loadRoutes(),
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看历史记录",
				},
			}),
		},
	}

	reply, _, _, _ := service.GenerateReply(context.Background(), "如何查看归档历史", nil, &AssistantPageContext{
		PagePath:  "/deploy/archived",
		ModuleKey: "deploy",
		PageTitle: "归档历史",
	}, "")
	if reply.Decision == nil {
		t.Fatal("expected decision to be populated")
	}
	if reply.Intent != reply.Decision.Intent.Name {
		t.Fatalf("expected top-level intent to mirror decision, got %#v", reply)
	}
	if reply.Answer != reply.Decision.Summary {
		t.Fatalf("expected top-level answer to mirror decision summary, got %#v", reply)
	}
	if reply.Decision.Context.Message != "如何查看归档历史" {
		t.Fatalf("expected context message, got %#v", reply.Decision.Context)
	}
	if reply.Decision.Context.PageContext == nil || reply.Decision.Context.PageContext.PagePath != "/deploy/archived" {
		t.Fatalf("expected page context to be preserved, got %#v", reply.Decision.Context.PageContext)
	}
	if reply.Decision.RiskLevel != "low" {
		t.Fatalf("expected low risk for knowledge qa, got %#v", reply.Decision)
	}
	if reply.Decision.ExecutionPlan == nil || len(reply.Decision.ExecutionPlan.Steps) == 0 {
		t.Fatalf("expected execution plan steps, got %#v", reply.Decision)
	}
	if reply.Model != "ops-assistant-fallback" {
		t.Fatalf("expected fallback model label, got %#v", reply.Model)
	}
}

func TestRunReadonlyToolsUsesRegisteredTool(t *testing.T) {
	service := &Service{
		readonlyTools: []assistantTool{
			{
				Name: "query_alert_events",
				Match: func(message string, intent AssistantIntent) bool {
					return intent.SubIntent == "alert_event_query"
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{ToolName: "query_alert_events", Summary: "ok"}
				},
			},
		},
	}
	result := service.runReadonlyTools("查看最近告警状态", AssistantIntent{
		Name:      "readonly_query",
		SubIntent: "alert_event_query",
		NeedTools: true,
	}, nil)
	if result == nil {
		t.Fatal("expected readonly tool result")
	}
	if result.ToolName != "query_alert_events" {
		t.Fatalf("expected query_alert_events, got %#v", result)
	}
}

func TestBuildExecutionPlanPrefersResolvedToolName(t *testing.T) {
	plan := buildExecutionPlan(AssistantIntent{
		Name:      "readonly_query",
		SubIntent: "alert_event_query",
		NeedTools: true,
	}, nil, &toolContext{ToolName: "query_alert_events"})
	if plan == nil || len(plan.Steps) == 0 {
		t.Fatalf("expected plan, got %#v", plan)
	}
	if plan.Steps[0].Tool != "query_alert_events" {
		t.Fatalf("expected resolved tool name, got %#v", plan)
	}
	if !plan.Steps[0].Readonly {
		t.Fatalf("expected readonly execution step, got %#v", plan)
	}
}

func TestBuildReadonlyToolRegistryIncludesPrimaryTools(t *testing.T) {
	registry := buildReadonlyToolRegistry()
	names := map[string]assistantTool{}
	for _, tool := range registry {
		names[tool.Name] = tool
	}

	for _, name := range []string{"query_release_history", "query_alert_events", "query_archive_history"} {
		tool, ok := names[name]
		if !ok {
			t.Fatalf("expected tool %s in registry", name)
		}
		if !tool.Readonly {
			t.Fatalf("expected tool %s to be readonly", name)
		}
		if tool.Match == nil || tool.Execute == nil {
			t.Fatalf("expected tool %s to define match and execute", name)
		}
	}
}

func TestRunReadonlyToolsUsesArchiveHistoryTool(t *testing.T) {
	service := &Service{
		readonlyTools: []assistantTool{
			{
				Name: "query_archive_history",
				Match: func(message string, intent AssistantIntent) bool {
					return intent.SubIntent == "archive_history_query"
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{ToolName: "query_archive_history", Summary: "ok"}
				},
				Readonly: true,
			},
		},
	}

	result := service.runReadonlyTools("查看归档历史", AssistantIntent{
		Name:      "readonly_query",
		SubIntent: "archive_history_query",
		NeedTools: true,
	}, nil)
	if result == nil || result.ToolName != "query_archive_history" {
		t.Fatalf("expected archive history tool result, got %#v", result)
	}
}

func TestResolveIntentWithPageContextPromotesFallbackToReadonlyQuery(t *testing.T) {
	intent := resolveIntentWithPageContext("最新动作", &AssistantPageContext{
		PagePath:  "/alarm/events",
		ModuleKey: "alert",
		PageTitle: "告警中心",
	}, AssistantIntent{Name: "fallback", Confidence: 0.4})

	if intent.Name != "readonly_query" {
		t.Fatalf("expected readonly_query, got %#v", intent)
	}
	if intent.SubIntent != "alert_event_query" {
		t.Fatalf("expected alert_event_query, got %#v", intent)
	}
	if !intent.NeedTools {
		t.Fatalf("expected tool requirement, got %#v", intent)
	}
}

func TestResolveIntentWithFocusedObjectPromotesKnowledgeQueryToReadonly(t *testing.T) {
	intent := resolveIntentWithPageContext("这个漏洞要不要先修", &AssistantPageContext{
		PagePath:   "/security/vulnerabilities",
		ModuleKey:  "security",
		ObjectType: "security_vulnerability",
		ObjectID:   "42",
	}, AssistantIntent{Name: "knowledge_qa", Confidence: 0.85})

	if intent.Name != "readonly_query" {
		t.Fatalf("expected readonly_query, got %#v", intent)
	}
	if intent.SubIntent != "vulnerability_query" {
		t.Fatalf("expected vulnerability_query, got %#v", intent)
	}
	if !intent.NeedTools {
		t.Fatalf("expected focused object query to require tools, got %#v", intent)
	}
}

func TestShouldUseFocusedObjectQuery(t *testing.T) {
	pageContext := &AssistantPageContext{
		ObjectType: "deploy_record",
		ObjectID:   "12",
	}

	if !shouldUseFocusedObjectQuery("这次部署失败点在哪", pageContext, "deploy_record") {
		t.Fatal("expected focused deploy record query to be recognized")
	}
	if shouldUseFocusedObjectQuery("最近有哪些失败部署", pageContext, "deploy_record") {
		t.Fatal("expected list query not to be forced into focused object mode")
	}
}

func TestRunReadonlyToolsPrefersToolFromPageContext(t *testing.T) {
	service := &Service{
		readonlyTools: []assistantTool{
			{
				Name: "query_release_history",
				Match: func(message string, intent AssistantIntent) bool {
					return false
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{ToolName: "query_release_history", Summary: "deploy page result"}
				},
			},
			{
				Name: "query_alert_events",
				Match: func(message string, intent AssistantIntent) bool {
					return true
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{ToolName: "query_alert_events", Summary: "alert result"}
				},
			},
		},
	}

	result := service.runReadonlyTools("最近有哪些失败记录", AssistantIntent{
		Name:      "readonly_query",
		SubIntent: "generic_readonly_query",
		NeedTools: true,
	}, &AssistantPageContext{
		PagePath:  "/deploy/history",
		ModuleKey: "deploy",
	})
	if result == nil {
		t.Fatal("expected readonly tool result")
	}
	if result.ToolName != "query_release_history" {
		t.Fatalf("expected deploy history tool, got %#v", result)
	}
}

func TestGenerateReplyReadonlyQueryBuildsDecisionWithResolvedTool(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		readonlyTools: []assistantTool{
			{
				Name: "query_archive_history",
				Match: func(message string, intent AssistantIntent) bool {
					return intent.SubIntent == "archive_history_query"
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{
						ToolName: "query_archive_history",
						Summary:  "查询到的归档历史如下：\n- 应用：web-api，环境：prod，状态：success",
					}
				},
				Readonly: true,
			},
		},
	}

	reply, _, _, _ := service.GenerateReply(context.Background(), "查看归档历史", nil, &AssistantPageContext{
		PagePath:  "/deploy/archived",
		ModuleKey: "deploy",
		PageTitle: "归档历史",
	}, "")
	if reply.Decision == nil {
		t.Fatal("expected decision to be populated")
	}
	if reply.Decision.Intent.Name != "readonly_query" {
		t.Fatalf("expected readonly_query intent, got %#v", reply.Decision)
	}
	if reply.Decision.ExecutionPlan == nil || len(reply.Decision.ExecutionPlan.Steps) == 0 {
		t.Fatalf("expected execution plan, got %#v", reply.Decision)
	}
	if reply.Decision.ExecutionPlan.Steps[0].Tool != "query_archive_history" {
		t.Fatalf("expected resolved archive history tool, got %#v", reply.Decision.ExecutionPlan)
	}
	if !reply.Decision.ExecutionPlan.Steps[0].Readonly {
		t.Fatalf("expected readonly execution step, got %#v", reply.Decision.ExecutionPlan)
	}
	if reply.Decision.Context.PageContext == nil || reply.Decision.Context.PageContext.PagePath != "/deploy/archived" {
		t.Fatalf("expected page context to be preserved, got %#v", reply.Decision.Context)
	}
}

func TestGenerateReplyUsesPageContextForGenericReadonlyQuestion(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		readonlyTools: []assistantTool{
			{
				Name: "query_release_history",
				Match: func(message string, intent AssistantIntent) bool {
					return false
				},
				Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
					return &toolContext{
						ToolName: "query_release_history",
						Summary:  "结论：最近有 2 条失败部署。",
					}
				},
				Readonly: true,
			},
		},
	}

	reply, _, _, _ := service.GenerateReply(context.Background(), "最近有哪些失败记录", nil, &AssistantPageContext{
		PagePath:  "/deploy/history",
		ModuleKey: "deploy",
		PageTitle: "部署记录",
	}, "")
	if reply.Decision == nil {
		t.Fatal("expected decision to be populated")
	}
	if reply.Decision.Intent.Name != "readonly_query" {
		t.Fatalf("expected readonly_query intent, got %#v", reply.Decision.Intent)
	}
	if reply.Decision.ExecutionPlan == nil || len(reply.Decision.ExecutionPlan.Steps) == 0 {
		t.Fatalf("expected execution plan, got %#v", reply.Decision.ExecutionPlan)
	}
	if reply.Decision.ExecutionPlan.Steps[0].Tool != "query_release_history" {
		t.Fatalf("expected deploy history tool, got %#v", reply.Decision.ExecutionPlan)
	}
	if !strings.Contains(reply.Answer, "失败部署") {
		t.Fatalf("expected tool-backed answer, got %#v", reply.Answer)
	}
}

func TestBuildActionsFallsBackToCurrentPageAction(t *testing.T) {
	service := &Service{
		knowledge: &knowledgeBase{
			routes: loadRoutes(),
		},
	}

	actions := service.buildActions("怎么下载", AssistantIntent{Name: "knowledge_qa"}, &AssistantPageContext{
		PagePath:  "/deploy/archived",
		ModuleKey: "deploy",
		PageTitle: "归档历史",
	})
	if len(actions) != 1 {
		t.Fatalf("expected current page action, got %#v", actions)
	}
	if actions[0].Path != "/deploy/archived" {
		t.Fatalf("expected archive history path, got %#v", actions[0])
	}
}

func TestSanitizePageContext(t *testing.T) {
	clean := sanitizePageContext(&AssistantPageContext{
		PagePath:          " /deploy/history ",
		ModuleKey:         " deploy ",
		PageTitle:         " 部署记录 ",
		SelectedRecordIDs: []string{" rec-1 ", "", "rec-2"},
		Filters: map[string]string{
			" status ":   " failed ",
			"appName":    " web-api ",
			"emptyValue": "   ",
			"   ":        "ignored",
		},
	})
	if clean == nil {
		t.Fatal("expected page context")
	}
	if clean.PagePath != "/deploy/history" || clean.ModuleKey != "deploy" || clean.PageTitle != "部署记录" {
		t.Fatalf("unexpected sanitized context: %#v", clean)
	}
	if len(clean.SelectedRecordIDs) != 2 {
		t.Fatalf("expected filtered selected ids, got %#v", clean.SelectedRecordIDs)
	}
	if clean.Filters["status"] != "failed" || clean.Filters["appName"] != "web-api" {
		t.Fatalf("unexpected sanitized filters: %#v", clean.Filters)
	}
	if _, exists := clean.Filters["emptyValue"]; exists {
		t.Fatalf("expected empty filter to be removed, got %#v", clean.Filters)
	}
	if sanitizePageContext(&AssistantPageContext{}) != nil {
		t.Fatal("expected empty page context to be omitted")
	}
}

func TestSummarizeCitationsForQuestionUsesTypedPrefix(t *testing.T) {
	citations := []Citation{
		{
			Title:   "运维小助手技术方案 / 17. 一期验收标准",
			Path:    "docs/运维小助手技术方案.md",
			Snippet: "- 能回答平台常见使用问题\n- 能返回正确页面导航\n- 工具调用有日志",
		},
	}

	answer := summarizeCitationsForQuestion("运维小助手一期验收标准是什么", citations, false)
	if !strings.HasPrefix(answer, "根据当前检索到的文档内容，验收标准主要包括：") {
		t.Fatalf("unexpected answer prefix: %q", answer)
	}
	if strings.Count(answer, "\n- ") != 3 {
		t.Fatalf("expected bullet summary, got %q", answer)
	}
}

func TestResponseModelNameMarksFallbackClearly(t *testing.T) {
	service := &Service{}

	if got := service.responseModelName("模型回答", nil); got != "ops-assistant" {
		t.Fatalf("expected default model name for successful answer, got %q", got)
	}
	if got := service.responseModelName("", context.DeadlineExceeded); got != "ops-assistant-fallback" {
		t.Fatalf("expected fallback model label, got %q", got)
	}
}

func TestSummarizeCitationsForPlatformUsageUsesOnboardingGuide(t *testing.T) {
	answer := summarizeCitationsForQuestion("如何使用平台", []Citation{
		{
			Title:   "用户手册 / 快速入门",
			Path:    "docs/user_manual.md",
			Snippet: "平台包含配置管理、发布管理、告警和安全管理等模块。",
		},
	}, false)

	if !strings.HasPrefix(answer, "可以按下面这个顺序上手平台：") {
		t.Fatalf("unexpected onboarding answer prefix: %q", answer)
	}
	for _, fragment := range []string{"/cmdb/projects", "/deploy/release", "/monitor/bigscreen", "/security/tasks", "/user-manual"} {
		if !strings.Contains(answer, fragment) {
			t.Fatalf("expected onboarding guide to include %q, got %q", fragment, answer)
		}
	}
}

func TestSummarizeCitationsForRoadmapSkipsIntroLine(t *testing.T) {
	citations := []Citation{
		{
			Title:   "运维小助手开发待办清单 / 6. 推荐开发顺序",
			Path:    "docs/运维小助手开发待办清单.md",
			Snippet: "建议按以下顺序推进：\n\n1. 完成鉴权、历史、降级、限流\n2. 完成 RAG 真检索第一版",
		},
	}

	answer := summarizeCitationsForQuestion("运维小助手推荐开发顺序是什么", citations, false)
	if strings.Contains(answer, "\n- 建议按以下顺序推进：") {
		t.Fatalf("unexpected intro bullet in %q", answer)
	}
}

func TestBuildCitationsOnlyUsesDocsForKnowledgeQA(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 配置管理\n可以在变更发布里查看配置管理。",
				},
			}),
		},
	}

	if citations := service.buildCitations("如何查看配置管理", "knowledge_qa", nil); len(citations) == 0 {
		t.Fatal("expected knowledge_qa to use citations")
	}
	if citations := service.buildCitations("查看配置管理", "readonly_query", nil); len(citations) != 0 {
		t.Fatalf("expected readonly_query to skip citations, got %#v", citations)
	}
}

func TestBuildCitationsForPlatformUsagePrefersUserManual(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 快速入门\n1. 导航至 /cmdb/projects\n2. 配置项目、环境、主机和应用\n3. 进入 /deploy/release 执行部署",
				},
				{
					Title:   "运维小助手技术方案",
					Path:    "docs/运维小助手技术方案.md",
					Module:  "assistant",
					Content: "# 设计说明\n这里是设计文档。",
				},
			}),
		},
	}

	citations := service.buildCitations("如何使用平台", "knowledge_qa", nil)
	if len(citations) == 0 {
		t.Fatal("expected onboarding citations")
	}
	if citations[0].Path != "docs/user_manual.md" {
		t.Fatalf("expected user manual citation first, got %#v", citations)
	}
}

func TestFilterCitationsForActionPrefersProjectManual(t *testing.T) {
	citations := []Citation{
		{Title: "user_manual / 执行聚合打包", Path: "docs/user_manual.md", Snippet: "1. 导航至 /deploy/aggregate-package"},
		{Title: "user_manual / 创建项目", Path: "docs/user_manual.md", Snippet: "1. 导航至 /cmdb/projects\n2. 点击新建项目"},
	}

	filtered := filterCitationsForAction("如何新建项目", citations, []Action{{Type: "navigate", Label: "打开项目管理", Path: "/cmdb/projects"}})
	if len(filtered) != 1 || !strings.Contains(filtered[0].Snippet, "/cmdb/projects") {
		t.Fatalf("expected project-focused citation, got %#v", filtered)
	}
}

func TestFilterCitationsForActionPrefersServerManual(t *testing.T) {
	citations := []Citation{
		{Title: "user_manual / 执行视图复制", Path: "docs/user_manual.md", Snippet: "1. 导航至 /jenkins/views\n2. 输入 Jenkins 服务器地址"},
		{Title: "user_manual / 添加服务器", Path: "docs/user_manual.md", Snippet: "1. 导航至 /cmdb/servers\n2. 点击添加服务器"},
	}

	filtered := filterCitationsForAction("如何新建服务器", citations, []Action{{Type: "navigate", Label: "打开主机管理", Path: "/cmdb/servers"}})
	if len(filtered) != 1 || !strings.Contains(filtered[0].Snippet, "/cmdb/servers") {
		t.Fatalf("expected server-focused citation, got %#v", filtered)
	}
}

func TestBuildCitationsForHowToUsesActionFocusedDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 2.1 项目管理 (/cmdb/projects)\n#### 创建项目\n1. 导航至 /cmdb/projects\n2. 点击新建项目",
				},
				{
					Title:   "安装包聚合打包功能需求文档",
					Path:    "docs/aggregate-package-feature-requirements.md",
					Module:  "deploy",
					Content: "# 用户操作流程\n1. 用户进入聚合打包页面\n2. 选择要进行聚合打包的项目",
				},
			}),
		},
	}

	citations := service.buildCitations("如何新建项目", "knowledge_qa", []Action{{Type: "navigate", Label: "打开项目管理", Path: "/cmdb/projects"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "/cmdb/projects") {
		t.Fatalf("expected action-focused project citation, got %#v", citations)
	}
}

func TestBuildCitationsForHowToUsesApplicationDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 2.4 应用管理 (/cmdb/applications)\n#### 新建流水线\n1. 导航至 /cmdb/applications\n2. 点击新建流水线\n3. 输入应用名称",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 2.2 环境管理 (/cmdb/environments)\n#### 创建环境\n1. 导航至 /cmdb/environments\n2. 点击新建环境",
				},
			}),
		},
	}

	citations := service.buildCitations("如何新建流水线", "knowledge_qa", []Action{{Type: "navigate", Label: "打开应用管理", Path: "/cmdb/applications"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "/cmdb/applications") {
		t.Fatalf("expected application-focused citation, got %#v", citations)
	}
}

func TestBuildCitationsForViewPageUsesActionFocusedDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.3 部署记录 (/deploy/history)\n#### 查看部署记录\n1. 导航至 /deploy/history\n2. 查看部署记录列表\n3. 根据应用名称、部署状态和时间定位目标记录",
				},
				{
					Title:   "安装包聚合打包功能需求文档",
					Path:    "docs/aggregate-package-feature-requirements.md",
					Module:  "deploy",
					Content: "# 1. 功能概述\n安装包聚合打包功能是针对运维管理平台的一个新增功能模块。",
				},
			}),
		},
	}

	citations := service.buildCitations("查看部署记录", "knowledge_qa", []Action{{Type: "navigate", Label: "打开部署记录", Path: "/deploy/history"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "/deploy/history") {
		t.Fatalf("expected deploy-history-focused citation, got %#v", citations)
	}
}

func TestBuildCitationsForArchiveHowToUsesArchiveDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 选择项目\n3. 选择需要归档的应用\n4. 点击开始归档",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看历史记录",
				},
			}),
		},
	}

	citations := service.buildCitations("如何归档", "knowledge_qa", []Action{{Type: "navigate", Label: "打开归档打包", Path: "/deploy/archive"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "/deploy/archive") {
		t.Fatalf("expected archive-focused citation, got %#v", citations)
	}
}

func TestBuildCitationsForArchiveHistoryHowToUsesArchivedDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 选择项目\n3. 选择需要归档的应用\n4. 点击开始归档",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看历史记录\n3. 可按应用名称筛选",
				},
			}),
		},
	}

	citations := service.buildCitations("如何查看归档历史", "knowledge_qa", []Action{{Type: "navigate", Label: "打开归档历史", Path: "/deploy/archived"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "/deploy/archived") {
		t.Fatalf("expected archive-history-focused citation, got %#v", citations)
	}
}

func TestBuildCitationsForDeleteSessionQueryDoesNotUseDeployDocs(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 选择项目\n3. 点击开始归档",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 8.4 我的资料 (/profile)\n#### 修改密码\n1. 点击右上角用户下拉菜单中的“我的资料”\n2. 在安全设置区域点击“修改密码”",
				},
			}),
		},
	}

	citations := service.buildCitations("如何删除会话", "knowledge_qa", nil)
	for _, citation := range citations {
		if strings.Contains(citation.Snippet, "/deploy/archive") || strings.Contains(citation.Snippet, "/profile") {
			t.Fatalf("expected delete-session query to avoid unrelated manual citations, got %#v", citations)
		}
	}
}

func TestBuildActionsForArchiveHistoryKeepsOnlyArchiveHistoryEntry(t *testing.T) {
	service := &Service{
		knowledge: &knowledgeBase{
			routes: loadRoutes(),
		},
	}

	actions := service.buildActions("如何查看归档历史", AssistantIntent{Name: "knowledge_qa"}, nil)
	if len(actions) != 1 {
		t.Fatalf("expected only one archive-history action, got %#v", actions)
	}
	if actions[0].Path != "/deploy/archived" {
		t.Fatalf("expected /deploy/archived action, got %#v", actions)
	}
}

func TestBuildActionsForArchiveArtifactDownloadKeepsOnlyArchiveHistoryEntry(t *testing.T) {
	service := &Service{
		knowledge: &knowledgeBase{
			routes: loadRoutes(),
		},
	}

	actions := service.buildActions("如何从下载地址下载归档包", AssistantIntent{Name: "knowledge_qa"}, nil)
	if len(actions) != 1 {
		t.Fatalf("expected only one archive-history action, got %#v", actions)
	}
	if actions[0].Path != "/deploy/archived" {
		t.Fatalf("expected /deploy/archived action, got %#v", actions)
	}
}

func TestBuildCitationsForArchiveArtifactDownloadUsesDownloadSection(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看历史记录\n\n#### 下载归档产物\n1. 导航至 /deploy/archived\n2. 在归档历史列表中找到目标记录的下载地址\n3. 点击查看文件打开归档文件列表\n4. 在文件列表中点击下载获取相关归档包，或点击复制链接后在新窗口打开下载",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 点击开始归档",
				},
			}),
			routes: loadRoutes(),
		},
	}

	citations := service.buildCitations("如何从下载地址下载归档包", "knowledge_qa", []Action{{Type: "navigate", Label: "打开归档历史", Path: "/deploy/archived"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Title != "用户手册 / 下载归档产物" {
		t.Fatalf("expected download section first, got %#v", citations[0])
	}
}

func TestBuildActionsForAggregateArtifactDownloadKeepsOnlyAggregateHistoryEntry(t *testing.T) {
	service := &Service{
		knowledge: &knowledgeBase{
			routes: loadRoutes(),
		},
	}

	actions := service.buildActions("如何从下载地址下载聚合包", AssistantIntent{Name: "knowledge_qa"}, nil)
	if len(actions) != 1 {
		t.Fatalf("expected only one aggregate-history action, got %#v", actions)
	}
	if actions[0].Path != "/deploy/aggregated-history" {
		t.Fatalf("expected /deploy/aggregated-history action, got %#v", actions)
	}
}

func TestBuildCitationsForAggregateArtifactDownloadUsesDownloadSection(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.6 聚合历史 (/deploy/aggregated-history)\n#### 查看聚合历史\n1. 导航至 /deploy/aggregated-history\n2. 查看历史记录\n\n#### 下载聚合包\n1. 导航至 /deploy/aggregated-history\n2. 在聚合历史列表中找到目标记录的下载地址\n3. 直接点击下载链接下载聚合包",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.5 聚合打包功能 (/deploy/aggregate-package)\n#### 执行聚合打包\n1. 导航至 /deploy/aggregate-package\n2. 点击开始聚合打包",
				},
			}),
			routes: loadRoutes(),
		},
	}

	citations := service.buildCitations("如何从下载地址下载聚合包", "knowledge_qa", []Action{{Type: "navigate", Label: "打开聚合历史", Path: "/deploy/aggregated-history"}})
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Title != "用户手册 / 下载聚合包" {
		t.Fatalf("expected aggregate download section first, got %#v", citations[0])
	}
}

func TestBuildCitationsForDeleteSessionQueryUsesAssistantManual(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			Assistant: config.AssistantConfig{TopK: 4},
		},
		knowledge: &knowledgeBase{
			chunks: buildChunks([]documentEntry{
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 9.4 会话管理\n#### 删除会话\n1. 点击右下角浮动按钮打开 AI 助手\n2. 点击左上角的会话按钮展开历史会话列表\n3. 在目标会话右侧点击删除\n4. 在确认框中点击确定，删除该会话及其全部消息",
				},
				{
					Title:   "用户手册",
					Path:    "docs/user_manual.md",
					Module:  "manual",
					Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 点击开始归档",
				},
			}),
		},
	}

	citations := service.buildCitations("如何删除会话", "knowledge_qa", nil)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if !strings.Contains(citations[0].Snippet, "删除该会话及其全部消息") {
		t.Fatalf("expected delete-session citation first, got %#v", citations)
	}
}