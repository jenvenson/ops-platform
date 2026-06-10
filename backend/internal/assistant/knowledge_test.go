// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import "testing"

func TestExtractSearchTermsIncludesSemanticPhrases(t *testing.T) {
	terms := extractSearchTerms("运维小助手一期验收标准是什么")
	expected := []string{"运维小助手", "一期", "验收标准"}

	for _, want := range expected {
		found := false
		for _, term := range terms {
			if term == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected term %q in %#v", want, terms)
		}
	}
}

func TestSearchDocumentsRanksAssistantAcceptanceDocs(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 7. 验收标准建议\n满足以下条件可认为一期达到可用状态：文档问答能返回真实引用。",
			},
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 菜单说明\n平台主要功能菜单、页面入口和使用步骤。",
			},
		}),
	}

	citations := kb.searchDocuments("运维小助手一期验收标准是什么", 2)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/运维小助手开发待办清单.md" {
		t.Fatalf("expected first citation to be assistant todo doc, got %#v", citations[0])
	}
}

func TestExtractQueryModulesRecognizesAssistantQuery(t *testing.T) {
	modules := extractQueryModules("运维小助手一期验收标准是什么", extractSearchTerms("运维小助手一期验收标准是什么"))
	if len(modules) == 0 || modules[0] != "assistant" {
		t.Fatalf("expected assistant module, got %#v", modules)
	}
}

func TestSearchDocumentsPrefersAssistantDocsOverGenericManual(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 验收\n平台功能菜单与使用步骤。",
			},
			{
				Title:   "运维小助手技术方案",
				Path:    "docs/运维小助手技术方案.md",
				Module:  "assistant",
				Content: "# 17. 一期验收标准\n能返回正确页面导航。",
			},
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 7. 验收标准建议\n文档问答能返回真实引用。",
			},
		}),
	}

	citations := kb.searchDocuments("运维小助手一期验收标准是什么", 2)
	if len(citations) < 2 {
		t.Fatalf("expected at least 2 citations, got %#v", citations)
	}
	for _, citation := range citations {
		if citation.Path == "docs/user_manual.md" {
			t.Fatalf("expected assistant docs to be prioritized over manual, got %#v", citations)
		}
	}
}

func TestBuildCitationsFromResultsPrefersDocumentDiversity(t *testing.T) {
	results := []scoredChunk{
		{
			chunk: documentChunk{
				DocumentTitle: "运维小助手技术方案",
				Path:          "docs/运维小助手技术方案.md",
				Heading:       "17. 一期验收标准",
				Content:       "能返回正确页面导航。",
			},
			score: 10,
		},
		{
			chunk: documentChunk{
				DocumentTitle: "运维小助手技术方案",
				Path:          "docs/运维小助手技术方案.md",
				Heading:       "16. 一期范围",
				Content:       "一期只做文档问答与页面导航。",
			},
			score: 9,
		},
		{
			chunk: documentChunk{
				DocumentTitle: "运维小助手开发待办清单",
				Path:          "docs/运维小助手开发待办清单.md",
				Heading:       "7. 验收标准建议",
				Content:       "文档问答能返回真实引用。",
			},
			score: 8,
		},
	}

	citations := buildCitationsFromResults(results, []string{"运维小助手", "验收标准"}, 2)
	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %#v", citations)
	}
	if citations[0].Path == citations[1].Path {
		t.Fatalf("expected citations from different docs first, got %#v", citations)
	}
}

func TestSearchDocumentsSkipsCrossModuleFillWhenPreferredDocsAreEnough(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "运维小助手技术方案",
				Path:    "docs/运维小助手技术方案.md",
				Module:  "assistant",
				Content: "# 17. 一期验收标准\n能返回正确页面导航。",
			},
			{
				Title:   "基于qwen3-8b的运维小助手详细落地设计",
				Path:    "docs/基于qwen3-8b的运维小助手详细落地设计.md",
				Module:  "assistant",
				Content: "# 16. 一期验收标准\n能引用文档来源。",
			},
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 7. 验收标准建议\n文档问答能返回真实引用。",
			},
			{
				Title:   "testing",
				Path:    "docs/testing.md",
				Module:  "testing",
				Content: "# 1. 并发测试\n验证系统稳定性。",
			},
		}),
	}

	citations := kb.searchDocuments("运维小助手一期验收标准是什么", 4)
	if len(citations) != 3 {
		t.Fatalf("expected 3 citations after diversity cutoff, got %#v", citations)
	}
	for _, citation := range citations {
		if citation.Path == "docs/testing.md" {
			t.Fatalf("expected preferred assistant docs to be enough, got %#v", citations)
		}
	}
}

func TestSearchDocumentsPrefersFeatureDocsForHowToQuery(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 3.3 聚合打包功能\n#### 执行聚合打包\n1. 导航至 /deploy/aggregate-package\n2. 输入项目名称\n3. 点击开始聚合打包。",
			},
			{
				Title:   "安装包聚合打包功能需求文档",
				Path:    "docs/aggregate-package-feature-requirements.md",
				Module:  "deploy",
				Content: "# 1. 功能概述\n聚合打包用于集中化打包多个应用。\n# 用户操作流程\n1. 用户进入聚合打包页面\n2. 选择项目\n3. 选择应用\n4. 点击开始聚合打包。",
			},
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 1. 说明\n- `P0`：一期上线前必须完成\n- `P1`：一期可用后应尽快补齐\n- `P2`：中期增强能力",
			},
		}),
	}

	citations := kb.searchDocuments("如何操作聚合打包", 3)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/user_manual.md" && citations[0].Title != "安装包聚合打包功能需求文档 / 用户操作流程" {
		t.Fatalf("expected feature doc first, got %#v", citations[0])
	}
	for _, citation := range citations {
		if citation.Path == "docs/运维小助手开发待办清单.md" {
			t.Fatalf("unexpected assistant todo doc in how-to citations: %#v", citations)
		}
	}
}

func TestSearchRoutesPrefersProfileForPasswordChangeQuery(t *testing.T) {
	kb := &knowledgeBase{
		routes: loadRoutes(),
	}

	actions := kb.searchRoutes("如何修改用户密码", 2)
	if len(actions) == 0 {
		t.Fatal("expected actions")
	}
	if actions[0].Path != "/profile" {
		t.Fatalf("expected /profile for password change query, got %#v", actions)
	}
}

func TestSearchRoutesPrefersArchivePackagingPageForArchiveHowToQuery(t *testing.T) {
	kb := &knowledgeBase{
		routes: loadRoutes(),
	}

	actions := kb.searchRoutes("如何归档打包", 2)
	if len(actions) == 0 {
		t.Fatal("expected actions")
	}
	if actions[0].Path != "/deploy/archive" {
		t.Fatalf("expected /deploy/archive for archive packaging query, got %#v", actions)
	}
}

func TestSearchRoutesPrefersArchiveHistoryPageForArchiveHistoryQuery(t *testing.T) {
	kb := &knowledgeBase{
		routes: loadRoutes(),
	}

	actions := kb.searchRoutes("如何查看归档历史", 2)
	if len(actions) == 0 {
		t.Fatal("expected actions")
	}
	if actions[0].Path != "/deploy/archived" {
		t.Fatalf("expected /deploy/archived for archive history query, got %#v", actions)
	}
}

func TestSearchRoutesDoesNotMisrouteDeleteSessionQuery(t *testing.T) {
	kb := &knowledgeBase{
		routes: loadRoutes(),
	}

	actions := kb.searchRoutes("如何删除会话", 2)
	if len(actions) != 0 {
		t.Fatalf("expected no unrelated page actions for delete-session query, got %#v", actions)
	}
}

func TestSearchDocumentsPrefersSecurityManualForWebScanPrecheckQuery(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 7.1 安全扫描\n#### 网站漏洞扫描说明\n- 当前 Web 扫描主路径已收敛为登录后 Web 扫描\n- 接入预检查会给出 可直接接入、需要定制认证流、暂不建议自动扫描 三类结论",
			},
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 3. 推荐开发顺序\n- 完成鉴权、历史、降级、限流\n- 完成 RAG 真检索第一版\n- 接入安全模块查询\n- 接入 Jenkins / 发布联动查询",
			},
		}),
	}

	citations := kb.searchDocuments("网站漏洞扫描里的接入预检查是什么意思？", 3)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/user_manual.md" {
		t.Fatalf("expected security manual first, got %#v", citations[0])
	}
	for _, citation := range citations {
		if citation.Path == "docs/运维小助手开发待办清单.md" {
			t.Fatalf("unexpected assistant todo doc in security scan citations: %#v", citations)
		}
	}
}

func TestSearchDocumentsPrefersProfileManualForPasswordQuery(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 8.4 我的资料 (/profile)\n#### 修改密码\n1. 点击右上角用户下拉菜单中的“我的资料”\n2. 在安全设置区域点击“修改密码”\n3. 输入当前密码和新密码。",
			},
			{
				Title:   "运维小助手技术方案",
				Path:    "docs/运维小助手技术方案.md",
				Module:  "assistant",
				Content: "# 6.3 执行类能力\n- 触发某应用重新发布\n- 重试某 Jenkins 构建",
			},
			{
				Title:   "deploy",
				Path:    "docs/deploy.md",
				Module:  "deploy",
				Content: "# Redis 密码（必须修改）\nREDIS_PASSWORD=your_redis_password",
			},
		}),
	}

	citations := kb.searchDocuments("如何修改用户密码", 2)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/user_manual.md" {
		t.Fatalf("expected password change query to prefer user manual, got %#v", citations)
	}
	for _, citation := range citations {
		if citation.Path == "docs/deploy.md" {
			t.Fatalf("expected unrelated deploy password doc to be filtered out, got %#v", citations)
		}
	}
}

func TestSearchDocumentsPrefersManualStepsForProjectHowToQuery(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 2.1 项目管理 (/cmdb/projects)\n#### 创建项目\n1. 导航至 /cmdb/projects\n2. 点击\"新建项目\"按钮\n3. 输入项目名称\n4. 点击保存。",
			},
			{
				Title:   "运维小助手开发待办清单",
				Path:    "docs/运维小助手开发待办清单.md",
				Module:  "assistant",
				Content: "# 1. 说明\n- `P0`：一期上线前必须完成\n- `P1`：一期可用后应尽快补齐\n- `P2`：中期增强能力",
			},
		}),
	}

	citations := kb.searchDocuments("如何新建项目", 3)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/user_manual.md" {
		t.Fatalf("expected project how-to query to prefer user manual, got %#v", citations)
	}
	for _, citation := range citations {
		if citation.Path == "docs/运维小助手开发待办清单.md" {
			t.Fatalf("unexpected assistant todo doc in project how-to citations: %#v", citations)
		}
	}
}

func TestSearchDocumentsForArchiveHistoryActionPrefersHistoryChunkOverArchivePage(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 3.2 归档打包 (/deploy/archive)\n#### 归档应用\n1. 导航至 /deploy/archive\n2. 选择要归档的项目\n3. 选择需要归档的应用\n4. 点击开始归档\n5. 如需查看历史记录，可进入 /deploy/archived",
			},
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看历史记录\n3. 可按应用名称筛选",
			},
		}),
	}

	citations := kb.searchDocumentsForAction(Action{Type: "navigate", Label: "打开归档历史", Path: "/deploy/archived"}, 2)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Title != "用户手册 / 查看归档历史" {
		t.Fatalf("expected archive history chunk first, got %#v", citations[0])
	}
}

func TestSearchDocumentsForActionUsesArchiveHistorySectionFromRealDocs(t *testing.T) {
	kb := &knowledgeBase{
		docs:   loadDocuments(),
		chunks: buildChunks(loadDocuments()),
	}

	citations := kb.searchDocumentsForAction(Action{Type: "navigate", Label: "打开归档历史", Path: "/deploy/archived"}, 2)
	if len(citations) == 0 {
		t.Fatal("expected citations from real docs")
	}
	if citations[0].Title != "user_manual / 查看归档历史" {
		t.Fatalf("expected real docs to prefer archive history section, got %#v", citations[0])
	}
	for _, citation := range citations[1:] {
		if citation.Title == "user_manual / 归档应用" {
			t.Fatalf("expected archive history action to exclude archive page citation, got %#v", citations)
		}
	}
}

func TestSearchDocumentsPrefersManualStepsForServerHowToQuery(t *testing.T) {
	kb := &knowledgeBase{
		chunks: buildChunks([]documentEntry{
			{
				Title:   "用户手册",
				Path:    "docs/user_manual.md",
				Module:  "manual",
				Content: "# 2.3 服务器管理 (/cmdb/servers)\n#### 添加服务器\n1. 导航至 /cmdb/servers\n2. 点击\"添加服务器\"按钮\n3. 输入主机名和 IP 地址\n4. 点击保存。",
			},
			{
				Title:   "运维小助手技术方案",
				Path:    "docs/运维小助手技术方案.md",
				Module:  "assistant",
				Content: "# 6.3 执行类能力\n- 触发某应用重新发布\n- 同步某环境配置",
			},
		}),
	}

	citations := kb.searchDocuments("如何新建服务器", 3)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
	if citations[0].Path != "docs/user_manual.md" {
		t.Fatalf("expected server how-to query to prefer user manual, got %#v", citations)
	}
	for _, citation := range citations {
		if citation.Path == "docs/运维小助手技术方案.md" {
			t.Fatalf("unexpected assistant design doc in server how-to citations: %#v", citations)
		}
	}
}