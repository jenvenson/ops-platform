package assistant

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/pkg/config"
)

type documentEntry struct {
	Title   string
	Path    string
	Module  string
	Content string
}

type documentChunk struct {
	DocumentTitle string
	Path          string
	Module        string
	Heading       string
	Content       string
	Keywords      []string
	Embedding     []float64
}

type routeEntry struct {
	Title    string
	Path     string
	Keywords []string
}

type scoredChunk struct {
	index         int
	chunk         documentChunk
	lexicalScore  float64
	semanticScore float64
	score         float64
}

type knowledgeBase struct {
	docs          []documentEntry
	chunks        []documentChunk
	routes        []routeEntry
	embedProvider EmbedProvider
	embedModel    string
	embedEnabled  bool
	embeddingsErr error
	embeddingsMu  sync.Mutex
}

func loadKnowledgeBase(cfg *config.Config, embedProvider EmbedProvider) *knowledgeBase {
	docs := loadDocuments()
	embedModel := ""
	embedEnabled := false
	if cfg != nil && cfg.Assistant.Enabled && cfg.Assistant.OllamaEmbedModel != "" && embedProvider != nil {
		embedModel = cfg.Assistant.OllamaEmbedModel
		embedEnabled = true
	}
	if cfg != nil && cfg.Assistant.Enabled && cfg.Assistant.EmbedModel != "" && embedProvider != nil {
		embedModel = cfg.Assistant.EmbedModel
		embedEnabled = true
	}
	return &knowledgeBase{
		docs:          docs,
		chunks:        buildChunks(docs),
		routes:        loadRoutes(),
		embedProvider: embedProvider,
		embedModel:    embedModel,
		embedEnabled:  embedEnabled,
	}
}

func loadDocuments() []documentEntry {
	root, ok := findDocsRoot()
	if !ok {
		return defaultDocuments()
	}

	docsDir := filepath.Join(root, "docs")
	paths := []string{
		"deploy.md",
		"design.md",
		"testing.md",
		"user_manual.md",
		"project-structure.md",
		"aggregate-package-feature-requirements.md",
		"aggregate-package-feature-design.md",
		"运维小助手技术方案.md",
		"运维小助手开发待办清单.md",
		"基于qwen3-8b的运维小助手详细落地设计.md",
	}

	docs := make([]documentEntry, 0, len(paths))
	for _, path := range paths {
		fullPath := filepath.Join(docsDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		docs = append(docs, documentEntry{
			Title:   strings.TrimSuffix(path, filepath.Ext(path)),
			Path:    filepath.ToSlash(filepath.Join("docs", path)),
			Module:  inferModule(path),
			Content: normalizeContent(string(content)),
		})
	}

	if len(docs) == 0 {
		return defaultDocuments()
	}
	return docs
}

func findDocsRoot() (string, bool) {
	baseDir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	candidates := []string{baseDir}
	current := baseDir
	for i := 0; i < 4; i++ {
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		candidates = append(candidates, parent)
		current = parent
	}

	for _, candidate := range candidates {
		target := filepath.Join(candidate, "docs", "user_manual.md")
		if _, err := os.Stat(target); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func defaultDocuments() []documentEntry {
	return []documentEntry{
		{
			Title:   "部署说明",
			Path:    "docs/deploy.md",
			Module:  "deploy",
			Content: "# 部署说明\n## 发布入口\n- 首次部署使用 deploy/deploy-init.sh\n- 日常发布使用 deploy/deploy-update.sh\n- 生产环境发布由本地脚本通过 ssh/scp 同步产物到服务器",
		},
		{
			Title:   "用户手册",
			Path:    "docs/user_manual.md",
			Module:  "manual",
			Content: "# 运维管理平台用户使用手册\n## 3.3 归档历史 (/deploy/archived)\n#### 查看归档历史\n1. 导航至 /deploy/archived\n2. 查看归档历史记录列表\n3. 根据应用名称或归档结果筛选目标记录\n4. 按需查看归档详情和版本信息\n\n#### 下载归档产物\n1. 导航至 /deploy/archived\n2. 在归档历史列表中找到目标记录的“下载地址”\n3. 点击“查看文件”打开归档文件列表\n4. 在文件列表中点击“下载”获取相关归档包，或点击“复制链接”后在新窗口打开下载\n\n## 3.6 聚合历史 (/deploy/aggregated-history)\n#### 下载聚合包\n1. 导航至 /deploy/aggregated-history\n2. 在聚合历史列表中找到目标记录的“下载地址”\n3. 可直接点击下载链接下载聚合包\n4. 如需查看当前聚合产物列表，可点击“查看文件”打开文件列表后再下载或复制链接",
		},
	}
}

func buildChunks(docs []documentEntry) []documentChunk {
	chunks := make([]documentChunk, 0, len(docs)*4)
	for _, doc := range docs {
		docChunks := splitMarkdownDocument(doc)
		chunks = append(chunks, docChunks...)
	}
	return chunks
}

func splitMarkdownDocument(doc documentEntry) []documentChunk {
	scanner := bufio.NewScanner(strings.NewReader(doc.Content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var chunks []documentChunk
	currentHeading := doc.Title
	var currentLines []string

	flush := func() {
		content := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if content == "" {
			return
		}
		for _, part := range splitChunkContent(content, 420, 80) {
			chunks = append(chunks, documentChunk{
				DocumentTitle: doc.Title,
				Path:          doc.Path,
				Module:        doc.Module,
				Heading:       currentHeading,
				Content:       part,
				Keywords:      extractKeywords(doc.Title + "\n" + currentHeading + "\n" + part + "\n" + doc.Module),
			})
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isHeading(line) {
			flush()
			currentHeading = cleanHeading(line)
			currentLines = currentLines[:0]
			continue
		}
		currentLines = append(currentLines, line)
	}
	flush()

	if len(chunks) == 0 {
		return []documentChunk{{
			DocumentTitle: doc.Title,
			Path:          doc.Path,
			Module:        doc.Module,
			Heading:       doc.Title,
			Content:       capContent(doc.Content, 900),
			Keywords:      extractKeywords(doc.Title + "\n" + doc.Content + "\n" + doc.Module),
		}}
	}
	return chunks
}

func splitChunkContent(content string, size, overlap int) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	if size <= 0 {
		return []string{content}
	}

	runes := []rune(content)
	if len(runes) <= size {
		return []string{content}
	}

	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}

	step := size - overlap
	if step <= 0 {
		step = size
	}

	parts := make([]string, 0, (len(runes)/step)+1)
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		part := strings.TrimSpace(string(runes[start:end]))
		if part != "" {
			parts = append(parts, part)
		}
		if end == len(runes) {
			break
		}
	}
	return parts
}

func loadRoutes() []routeEntry {
	return []routeEntry{
		{Title: "工作台", Path: "/", Keywords: []string{"工作台", "首页", "仪表盘"}},
		{Title: "资产中心", Path: "/cmdb/projects", Keywords: []string{"资产中心", "cmdb"}},
		{Title: "变更发布", Path: "/deploy/release", Keywords: []string{"变更发布", "发布中心", "自动化部署"}},
		{Title: "监控中心", Path: "/monitor/bigscreen", Keywords: []string{"监控中心", "监控"}},
		{Title: "告警中心", Path: "/alarm/events", Keywords: []string{"告警中心", "告警事件", "报警中心"}},
		{Title: "安全中心", Path: "/security/overview", Keywords: []string{"安全中心", "安全管理"}},
		{Title: "系统管理", Path: "/admin/users", Keywords: []string{"系统管理", "后台管理"}},
		{Title: "用户手册", Path: "/user-manual", Keywords: []string{"用户手册", "使用手册", "帮助文档"}},
		{Title: "我的资料", Path: "/profile", Keywords: []string{"我的资料", "个人资料", "个人信息", "profile", "修改密码", "登录密码", "更改密码", "修改用户密码", "修改个人密码"}},
		{Title: "项目管理", Path: "/cmdb/projects", Keywords: []string{"项目", "项目管理", "cmdb"}},
		{Title: "环境管理", Path: "/cmdb/environments", Keywords: []string{"环境", "环境管理"}},
		{Title: "主机管理", Path: "/cmdb/servers", Keywords: []string{"主机", "服务器"}},
		{Title: "应用流水线管理", Path: "/cmdb/applications", Keywords: []string{"流水线", "应用管理", "应用流水线管理"}},
		{Title: "迭代部署", Path: "/deploy/release", Keywords: []string{"部署", "发布", "应用发布", "迭代部署"}},
		{Title: "部署记录", Path: "/deploy/history", Keywords: []string{"发布历史", "部署历史", "部署记录"}},
		{Title: "归档打包", Path: "/deploy/archive", Keywords: []string{"归档", "应用归档", "归档管理", "归档打包"}},
		{Title: "归档历史", Path: "/deploy/archived", Keywords: []string{"归档历史", "历史归档", "下载地址", "查看文件", "归档文件", "归档产物", "归档包", "下载归档产物", "下载归档包"}},
		{Title: "聚合打包", Path: "/deploy/aggregate-package", Keywords: []string{"聚合", "打包", "安装包", "聚合打包"}},
		{Title: "聚合历史", Path: "/deploy/aggregated-history", Keywords: []string{"聚合历史", "查看聚合历史", "聚合记录", "聚合状态", "下载地址", "查看文件", "聚合包", "下载聚合包"}},
		{Title: "平台事件中心", Path: "/platform/events", Keywords: []string{"平台事件中心", "平台事件", "事件中心", "跨模块事件", "对象时间线"}},
		{Title: "监控大屏", Path: "/monitor/bigscreen", Keywords: []string{"监控", "大屏", "监控大屏"}},
		{Title: "监控概览", Path: "/monitor/overview", Keywords: []string{"监控概览"}},
		{Title: "Grafana仪表盘", Path: "/monitor/dashboards", Keywords: []string{"grafana", "仪表盘"}},
		{Title: "告警事件", Path: "/alarm/events", Keywords: []string{"告警", "报警", "告警事件", "事件", "事件中心"}},
		{Title: "告警规则", Path: "/alarm/rules", Keywords: []string{"告警规则", "报警规则"}},
		{Title: "联系人管理", Path: "/alarm/contacts", Keywords: []string{"联系人"}},
		{Title: "通知渠道", Path: "/alarm/channels", Keywords: []string{"通知渠道", "渠道"}},
		{Title: "通知模板", Path: "/alarm/templates", Keywords: []string{"模板", "通知模板"}},
		{Title: "安全概览", Path: "/security/overview", Keywords: []string{"安全概览"}},
		{Title: "扫描任务", Path: "/security/tasks", Keywords: []string{"安全", "扫描", "扫描任务"}},
		{Title: "安全资产", Path: "/security/assets", Keywords: []string{"资产", "安全资产", "资产管理"}},
		{Title: "漏洞管理", Path: "/security/vulnerabilities", Keywords: []string{"漏洞", "漏洞管理"}},
		{Title: "漏洞工单", Path: "/security/tickets", Keywords: []string{"工单", "漏洞工单"}},
		{Title: "漏洞知识库", Path: "/security/vuln-db", Keywords: []string{"漏洞知识库", "知识库"}},
		{Title: "批量配置下发", Path: "/consul/batch-all", Keywords: []string{"consul", "流水线配置", "批量配置下发"}},
		{Title: "Consul配置变更", Path: "/consul/config", Keywords: []string{"consul配置变更", "consul变更", "配置变更"}},
		{Title: "配置管理", Path: "/consul/config", Keywords: []string{"consul配置", "配置管理", "consul配置变更", "Consul配置"}},
		{Title: "配置操作记录", Path: "/consul/operations", Keywords: []string{"consul", "操作记录", "配置操作记录"}},
		{Title: "Jenkins任务", Path: "/jenkins", Keywords: []string{"jenkins", "构建", "jenkins任务"}},
		{Title: "视图管理", Path: "/jenkins/views", Keywords: []string{"jenkins视图", "视图管理", "jenkins视图管理"}},
		{Title: "用户管理", Path: "/admin/users", Keywords: []string{"用户管理", "账号管理", "新增用户", "创建用户", "平台用户"}},
		{Title: "角色管理", Path: "/admin/roles", Keywords: []string{"角色", "角色管理"}},
		{Title: "菜单管理", Path: "/admin/menus", Keywords: []string{"菜单", "菜单管理"}},
		{Title: "系统设置", Path: "/admin/settings", Keywords: []string{"设置", "系统设置"}},
	}
}

func (k *knowledgeBase) searchDocuments(query string, limit int) []Citation {
	if k == nil || len(k.chunks) == 0 {
		return nil
	}

	queryTerms := extractSearchTerms(query)
	queryCore := trimKnowledgeQuery(query)
	preferredModules := extractQueryModules(query, queryTerms)
	questionType := classifyKnowledgeQuestion(query)
	results := make([]scoredChunk, 0, len(k.chunks))
	for idx, chunk := range k.chunks {
		score := float64(scoreKeywords(queryTerms, chunk.Keywords))
		score += float64(scoreModule(queryTerms, chunk.Module))
		score += scorePreferredModule(preferredModules, chunk.Module)
		score += scoreDocumentTermMatches(queryTerms, chunk)
		score += scoreQuestionTypeChunk(questionType, chunk)
		score += scoreFeatureDocPreference(queryTerms, chunk)
		score += float64(documentPriorityForQuery(chunk.Path, preferredModules, questionType))

		headingLower := strings.ToLower(chunk.Heading)
		titleLower := strings.ToLower(chunk.DocumentTitle)
		contentLower := strings.ToLower(chunk.Content)
		if queryCore != "" {
			switch {
			case headingLower == queryCore || titleLower == queryCore:
				score += 6
			case strings.Contains(headingLower, queryCore) || strings.Contains(titleLower, queryCore):
				score += 3
			case strings.Contains(contentLower, queryCore):
				score += 1.5
			}
		}
		if score > 0 {
			results = append(results, scoredChunk{index: idx, chunk: chunk, lexicalScore: score, score: score})
		}
	}

	semanticScores := k.semanticScores(query)
	if len(semanticScores) > 0 {
		if len(results) == 0 {
			for idx, chunk := range k.chunks {
				if semanticScores[idx] < 0.35 {
					continue
				}
				results = append(results, scoredChunk{
					index:         idx,
					chunk:         chunk,
					semanticScore: semanticScores[idx],
					score:         semanticScores[idx] * 8,
				})
			}
		} else {
			for i := range results {
				results[i].semanticScore = semanticScores[results[i].index]
				results[i].score += semanticScores[results[i].index] * 8
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			leftPriority := documentPriorityForQuery(results[i].chunk.Path, preferredModules, questionType)
			rightPriority := documentPriorityForQuery(results[j].chunk.Path, preferredModules, questionType)
			if leftPriority != rightPriority {
				return leftPriority > rightPriority
			}
			if results[i].lexicalScore != results[j].lexicalScore {
				return results[i].lexicalScore > results[j].lexicalScore
			}
			if results[i].semanticScore != results[j].semanticScore {
				return results[i].semanticScore > results[j].semanticScore
			}
			if results[i].chunk.Path == results[j].chunk.Path {
				return results[i].chunk.Heading < results[j].chunk.Heading
			}
			return results[i].chunk.Path < results[j].chunk.Path
		}
		return results[i].score > results[j].score
	})

	if limit <= 0 {
		limit = 4
	}

	orderedResults := prioritizeResultsByModule(results, preferredModules)
	orderedResults = trimResultsToPreferredModules(orderedResults, preferredModules, limit)
	orderedResults = filterResultsForSpecificIntent(query, queryTerms, orderedResults)
	return buildCitationsFromResults(orderedResults, queryTerms, limit)
}

func (k *knowledgeBase) searchDocumentsForAction(action Action, limit int) []Citation {
	if k == nil || len(k.chunks) == 0 || limit <= 0 {
		return nil
	}

	tokens := actionFocusTokens(action)
	if len(tokens) == 0 {
		return nil
	}

	results := make([]scoredChunk, 0, len(k.chunks))
	for idx, chunk := range k.chunks {
		path := filepath.ToSlash(chunk.Path)
		title := strings.ToLower(strings.TrimSpace(chunk.DocumentTitle))
		heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
		content := strings.ToLower(strings.TrimSpace(chunk.Content))

		score := 0.0
		matchedFocus := false
		actionable := containsAny(heading, "创建", "新建", "添加", "修改密码", "执行", "步骤", "操作流程", "用户操作流程") ||
			containsAny(content, "1.", "2.", "3.", "点击", "输入", "导航至")
		if path == "docs/user_manual.md" {
			score += 4
		}
		if containsAny(heading, "创建", "新建", "添加", "修改密码", "执行", "步骤", "操作流程", "用户操作流程") {
			score += 3
		}
		if containsAny(content, "1.", "2.", "3.", "点击", "输入", "导航至") {
			score += 2
		}

		for _, token := range tokens {
			token = strings.ToLower(strings.TrimSpace(token))
			if token == "" {
				continue
			}
			switch {
			case strings.Contains(content, token):
				score += 5
				matchedFocus = true
			case strings.Contains(heading, token):
				score += 4
				matchedFocus = true
			case strings.Contains(title, token):
				score += 3
				matchedFocus = true
			}
		}

		if score > 0 && matchedFocus && actionable {
			results = append(results, scoredChunk{index: idx, chunk: chunk, lexicalScore: score, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			if results[i].chunk.Path == results[j].chunk.Path {
				return results[i].chunk.Heading < results[j].chunk.Heading
			}
			if results[i].chunk.Path == "docs/user_manual.md" {
				return true
			}
			if results[j].chunk.Path == "docs/user_manual.md" {
				return false
			}
			return results[i].chunk.Path < results[j].chunk.Path
		}
		return results[i].score > results[j].score
	})

	exactPathResults := make([]scoredChunk, 0, len(results))
	for _, result := range results {
		if strings.Contains(strings.ToLower(result.chunk.Content), strings.ToLower(action.Path)) {
			exactPathResults = append(exactPathResults, result)
		}
	}
	if len(exactPathResults) > 0 {
		sort.SliceStable(exactPathResults, func(i, j int) bool {
			leftSpecificity := actionSpecificityScore(action, exactPathResults[i].chunk)
			rightSpecificity := actionSpecificityScore(action, exactPathResults[j].chunk)
			if leftSpecificity != rightSpecificity {
				return leftSpecificity > rightSpecificity
			}
			if exactPathResults[i].score != exactPathResults[j].score {
				return exactPathResults[i].score > exactPathResults[j].score
			}
			return exactPathResults[i].chunk.Heading < exactPathResults[j].chunk.Heading
		})
		topSpecificity := actionSpecificityScore(action, exactPathResults[0].chunk)
		if topSpecificity > 0 {
			filtered := make([]scoredChunk, 0, len(exactPathResults))
			for _, result := range exactPathResults {
				if actionSpecificityScore(action, result.chunk) < topSpecificity {
					continue
				}
				filtered = append(filtered, result)
			}
			if len(filtered) > 0 {
				exactPathResults = filtered
			}
		}
		results = exactPathResults
	}

	return buildCitationsFromResults(results, tokens, limit)
}

func actionSpecificityScore(action Action, chunk documentChunk) int {
	score := 0
	title := strings.ToLower(strings.TrimSpace(chunk.DocumentTitle))
	heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
	content := strings.ToLower(strings.TrimSpace(chunk.Content))
	label := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(action.Label, "打开")))

	if action.Path != "" {
		pathLower := strings.ToLower(action.Path)
		if strings.Contains(heading, pathLower) {
			score += 8
		}
		if strings.Contains(title, pathLower) {
			score += 6
		}
	}

	if label != "" {
		if heading == label {
			score += 20
		} else if strings.Contains(heading, label) {
			score += 12
		}
		if title == label {
			score += 10
		} else if strings.Contains(title, label) {
			score += 6
		}
		if strings.Contains(content, label) {
			score += 4
		}
	}

	for _, token := range actionFocusTokens(action) {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" || token == strings.ToLower(action.Path) {
			continue
		}
		switch {
		case heading == token:
			score += 10
		case strings.Contains(heading, token):
			score += 6
		case strings.Contains(title, token):
			score += 4
		case strings.Contains(content, token):
			score += 2
		}
	}

	return score
}

func filterResultsForSpecificIntent(query string, queryTerms []string, results []scoredChunk) []scoredChunk {
	if len(results) == 0 {
		return results
	}
	questionType := classifyKnowledgeQuestion(query)

	hasTerm := func(target string) bool {
		for _, term := range queryTerms {
			if strings.Contains(term, target) || strings.Contains(target, term) {
				return true
			}
		}
		return false
	}

	if questionType == "howto" && !hasTerm("运维小助手") && !hasTerm("assistant") {
		filtered := make([]scoredChunk, 0, len(results))
		for _, result := range results {
			path := filepath.ToSlash(result.chunk.Path)
			heading := strings.ToLower(strings.TrimSpace(result.chunk.Heading))
			content := strings.ToLower(strings.TrimSpace(result.chunk.Content))
			isStepDoc := containsAny(heading, "创建", "新建", "添加", "修改密码", "执行", "步骤", "操作流程", "用户操作流程") ||
				containsAny(content, "1.", "2.", "3.", "点击", "输入", "导航至", "新建项目", "添加服务器", "新建服务器")
			matchesTopic := matchesHowtoTopic(result.chunk, queryTerms)
			if strings.Contains(path, "助手") {
				continue
			}
			if isStepDoc && matchesTopic {
				filtered = append(filtered, result)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}

	if hasTerm("修改密码") || hasTerm("用户密码") || hasTerm("登录密码") || hasTerm("个人密码") {
		filtered := make([]scoredChunk, 0, len(results))
		for _, result := range results {
			heading := strings.ToLower(strings.TrimSpace(result.chunk.Heading))
			content := strings.ToLower(strings.TrimSpace(result.chunk.Content))
			if containsAny(heading, "我的资料", "修改密码") ||
				containsAny(content, "/profile", "我的资料", "登录密码", "当前密码", "新密码", "修改密码") {
				filtered = append(filtered, result)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}

	if hasSecurityScanIntent(queryTerms) {
		filtered := make([]scoredChunk, 0, len(results))
		for _, result := range results {
			path := filepath.ToSlash(result.chunk.Path)
			if strings.Contains(path, "助手") {
				continue
			}
			if matchesSecurityScanTopic(result.chunk, queryTerms) {
				filtered = append(filtered, result)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}

	return results
}

func matchesHowtoTopic(chunk documentChunk, queryTerms []string) bool {
	title := strings.ToLower(strings.TrimSpace(chunk.DocumentTitle))
	heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
	content := strings.ToLower(strings.TrimSpace(chunk.Content))

	for _, term := range queryTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || isGenericHowtoTerm(term) {
			continue
		}
		topics := expandHowtoTopicTerms(term)
		for _, topic := range topics {
			if topic == "" || isGenericHowtoTerm(topic) {
				continue
			}
			if strings.Contains(title, topic) || strings.Contains(heading, topic) || strings.Contains(content, topic) {
				return true
			}
		}
		if strings.Contains(title, term) || strings.Contains(heading, term) || strings.Contains(content, term) {
			return true
		}
	}
	return false
}

func hasSecurityScanIntent(queryTerms []string) bool {
	hasWeb := false
	hasScan := false
	hasPrecheck := false
	for _, term := range queryTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		if strings.Contains(term, "网站漏洞") || strings.Contains(term, "web扫描") || strings.Contains(term, "漏洞扫描") || strings.Contains(term, "安全扫描") {
			hasWeb = true
			hasScan = true
		}
		if strings.Contains(term, "网站") || strings.Contains(term, "web") || strings.Contains(term, "漏洞") {
			hasWeb = true
		}
		if strings.Contains(term, "扫描") {
			hasScan = true
		}
		if strings.Contains(term, "接入预检查") || strings.Contains(term, "预检查") {
			hasPrecheck = true
		}
	}
	return hasPrecheck || (hasWeb && hasScan)
}

func matchesSecurityScanTopic(chunk documentChunk, queryTerms []string) bool {
	title := strings.ToLower(strings.TrimSpace(chunk.DocumentTitle))
	heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
	content := strings.ToLower(strings.TrimSpace(chunk.Content))
	if !containsAny(title, "安全", "漏洞", "扫描") &&
		!containsAny(heading, "安全", "漏洞", "扫描", "网站漏洞") &&
		!containsAny(content, "/security/tasks", "网站漏洞", "主机漏洞", "接入预检查", "登录后 web 扫描", "登录后web扫描") {
		return false
	}
	for _, term := range queryTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || isGenericHowtoTerm(term) {
			continue
		}
		if strings.Contains(term, "接入预检查") && containsAny(content, "接入预检查", "可直接接入", "需要定制认证流", "暂不建议自动扫描") {
			return true
		}
		if strings.Contains(term, "网站漏洞") && containsAny(content, "网站漏洞", "目标 url", "登录后 web 扫描", "登录后web扫描") {
			return true
		}
		if strings.Contains(title, term) || strings.Contains(heading, term) || strings.Contains(content, term) {
			return true
		}
	}
	return false
}

func isGenericHowtoTerm(term string) bool {
	switch term {
	case "如何", "怎么", "怎样", "步骤", "使用", "配置", "创建", "新建", "添加", "操作", "处理":
		return true
	default:
		return false
	}
}

func expandHowtoTopicTerms(term string) []string {
	expanded := []string{term}
	trimmed := term
	for _, prefix := range []string{"新建", "创建", "添加", "查看", "修改", "操作"} {
		trimmed = strings.TrimPrefix(trimmed, prefix)
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed != "" && trimmed != term {
		expanded = append(expanded, trimmed)
	}
	if strings.Contains(term, "服务器") || strings.Contains(term, "主机") {
		expanded = append(expanded, "服务器", "主机")
	}
	if strings.Contains(term, "项目") {
		expanded = append(expanded, "项目")
	}
	return expanded
}

func buildCitationsFromResults(results []scoredChunk, queryTerms []string, limit int) []Citation {
	if len(results) == 0 || limit <= 0 {
		return nil
	}

	distinctPaths := map[string]struct{}{}
	for _, result := range results {
		distinctPaths[result.chunk.Path] = struct{}{}
	}
	stopAfterDiversity := minInt(limit, 3)

	citations := make([]Citation, 0, limit)
	seenHeadings := map[string]struct{}{}
	pathCounts := map[string]int{}

	appendCitation := func(result scoredChunk) bool {
		key := result.chunk.Path + "::" + normalizeHeadingKey(result.chunk.Heading)
		if _, exists := seenHeadings[key]; exists {
			return false
		}
		seenHeadings[key] = struct{}{}
		pathCounts[result.chunk.Path]++
		citations = append(citations, Citation{
			Title:   result.chunk.DocumentTitle + " / " + result.chunk.Heading,
			Path:    result.chunk.Path,
			Snippet: extractSnippet(result.chunk.Content, queryTerms),
		})
		return true
	}

	for _, maxPerPath := range []int{1, 2, limit} {
		for _, result := range results {
			if len(citations) >= limit {
				return citations
			}
			if pathCounts[result.chunk.Path] >= maxPerPath {
				continue
			}
			appendCitation(result)
		}
		if maxPerPath == 1 && len(distinctPaths) >= stopAfterDiversity && len(citations) >= stopAfterDiversity {
			return citations
		}
	}

	return citations
}

func prioritizeResultsByModule(results []scoredChunk, preferredModules []string) []scoredChunk {
	if len(results) == 0 || len(preferredModules) == 0 {
		return results
	}

	preferred := make([]scoredChunk, 0, len(results))
	others := make([]scoredChunk, 0, len(results))

	for _, result := range results {
		if moduleIn(result.chunk.Module, preferredModules) {
			preferred = append(preferred, result)
			continue
		}
		others = append(others, result)
	}
	if len(preferred) == 0 {
		return results
	}
	return append(preferred, others...)
}

func trimResultsToPreferredModules(results []scoredChunk, preferredModules []string, limit int) []scoredChunk {
	if len(results) == 0 || len(preferredModules) == 0 {
		return results
	}

	preferred := make([]scoredChunk, 0, len(results))
	distinctPaths := map[string]struct{}{}
	for _, result := range results {
		if !moduleIn(result.chunk.Module, preferredModules) {
			continue
		}
		preferred = append(preferred, result)
		distinctPaths[result.chunk.Path] = struct{}{}
	}

	requiredDistinct := minInt(limit, 3)
	if len(distinctPaths) >= requiredDistinct {
		return preferred
	}
	return results
}

func (k *knowledgeBase) semanticScores(query string) []float64 {
	if k == nil || !k.embedEnabled || k.embedModel == "" || k.embedProvider == nil || strings.TrimSpace(query) == "" {
		return nil
	}
	if err := k.ensureEmbeddings(); err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	vectors, err := k.embedProvider.Embed(ctx, k.embedModel, []string{query})
	if err != nil || len(vectors) == 0 {
		return nil
	}
	queryEmbedding := vectors[0]

	scores := make([]float64, len(k.chunks))
	for i, chunk := range k.chunks {
		if len(chunk.Embedding) == 0 {
			continue
		}
		scores[i] = cosineSimilarity(queryEmbedding, chunk.Embedding)
	}
	return scores
}

func (k *knowledgeBase) ensureEmbeddings() error {
	if k == nil || !k.embedEnabled || k.embedModel == "" || k.embedProvider == nil {
		return nil
	}

	k.embeddingsMu.Lock()
	defer k.embeddingsMu.Unlock()

	if k.embeddingsErr != nil {
		return k.embeddingsErr
	}
	ready := true
	for _, chunk := range k.chunks {
		if len(chunk.Embedding) == 0 {
			ready = false
			break
		}
	}
	if ready {
		return nil
	}

	inputs := make([]string, 0, len(k.chunks))
	indexes := make([]int, 0, len(k.chunks))
	for idx, chunk := range k.chunks {
		if len(chunk.Embedding) > 0 {
			continue
		}
		inputs = append(inputs, strings.TrimSpace(chunk.DocumentTitle+"\n"+chunk.Heading+"\n"+chunk.Content))
		indexes = append(indexes, idx)
	}
	if len(inputs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	vectors, err := k.embedProvider.Embed(ctx, k.embedModel, inputs)
	if err != nil {
		k.embeddingsErr = err
		return err
	}
	if len(vectors) != len(indexes) {
		k.embeddingsErr = fmt.Errorf("embedding count mismatch: got %d want %d", len(vectors), len(indexes))
		return k.embeddingsErr
	}
	for i, idx := range indexes {
		k.chunks[idx].Embedding = vectors[i]
	}
	return nil
}

func (k *knowledgeBase) searchRoutes(query string, limit int) []Action {
	if k == nil || len(k.routes) == 0 {
		return nil
	}

	queryTerms := extractKeywords(query)
	normalizedQuery := normalizeContent(strings.ToLower(query))
	coreQuery := trimNavigationQuery(normalizedQuery)
	type scoredRoute struct {
		route routeEntry
		score int
		exact bool
	}

	results := make([]scoredRoute, 0, len(k.routes))
	for _, route := range k.routes {
		score := scoreKeywords(queryTerms, route.Keywords)
		exact := false
		titleLower := strings.ToLower(route.Title)
		if coreQuery == titleLower || normalizedQuery == titleLower {
			score += 100
			exact = true
		} else if strings.Contains(coreQuery, titleLower) || strings.Contains(normalizedQuery, titleLower) {
			score += 20
		}
		for _, keyword := range route.Keywords {
			keywordLower := strings.ToLower(keyword)
			if coreQuery == keywordLower || normalizedQuery == keywordLower {
				score += 60
				exact = true
				continue
			}
			if strings.Contains(coreQuery, keywordLower) || strings.Contains(normalizedQuery, keywordLower) {
				score += 8
			}
		}
		if score > 0 {
			results = append(results, scoredRoute{route: route, score: score, exact: exact})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].route.Path < results[j].route.Path
		}
		return results[i].score > results[j].score
	})

	if limit <= 0 {
		limit = 2
	}

	hasExact := false
	for _, result := range results {
		if result.exact {
			hasExact = true
			break
		}
	}

	filtered := make([]scoredRoute, 0, len(results))
	seenPaths := make(map[string]struct{}, len(results))
	for _, result := range results {
		if hasExact && !result.exact {
			continue
		}
		if _, exists := seenPaths[result.route.Path]; exists {
			continue
		}
		seenPaths[result.route.Path] = struct{}{}
		filtered = append(filtered, result)
		if len(filtered) >= limit {
			break
		}
	}

	actions := make([]Action, 0, len(filtered))
	for _, result := range filtered {
		actions = append(actions, Action{
			Type:  "navigate",
			Label: "打开" + result.route.Title,
			Path:  result.route.Path,
		})
	}
	return actions
}

func inferModule(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "deploy"):
		return "deploy"
	case strings.Contains(lower, "aggregate-package"):
		return "deploy"
	case strings.Contains(lower, "manual"):
		return "manual"
	case strings.Contains(lower, "project-structure"):
		return "structure"
	case strings.Contains(lower, "testing"):
		return "testing"
	case strings.Contains(lower, "design"):
		return "design"
	case strings.Contains(lower, "助手"):
		return "assistant"
	default:
		return "general"
	}
}

func isHeading(line string) bool {
	return strings.HasPrefix(line, "#")
}

func cleanHeading(line string) string {
	return strings.TrimSpace(strings.TrimLeft(line, "#"))
}

func capContent(content string, maxRunes int) string {
	if maxRunes <= 0 {
		return strings.TrimSpace(content)
	}
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}

func extractKeywords(text string) []string {
	text = normalizeContent(strings.ToLower(text))
	replacer := strings.NewReplacer(
		"：", " ", "，", " ", "。", " ", "、", " ", "；", " ", "（", " ", "）", " ",
		"(", " ", ")", " ", "/", " ", "-", " ", "_", " ", "|", " ", "\n", " ", "\t", " ",
	)
	text = replacer.Replace(text)
	parts := strings.Fields(text)
	keywords := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) < 2 {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
	}
	return keywords
}

func extractSearchTerms(text string) []string {
	terms := extractKeywords(text)
	seen := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		seen[term] = struct{}{}
	}

	normalized := trimKnowledgeQuery(text)
	for _, term := range splitSemanticTerms(normalized) {
		if _, exists := seen[term]; exists {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}

	sort.SliceStable(terms, func(i, j int) bool {
		return len([]rune(terms[i])) > len([]rune(terms[j]))
	})
	return terms
}

func splitSemanticTerms(text string) []string {
	text = normalizeContent(strings.ToLower(text))
	replacer := strings.NewReplacer(
		"什么是", " ", "是什么", " ", "有哪些", " ", "有哪", " ", "哪里", " ", "在哪", " ",
		"如何", " ", "怎么", " ", "怎样", " ", "帮我", " ", "请问", " ", "一下", " ", "一下子", " ",
		"当前", " ", "最近", " ", "有关", " ", "关于", " ", "相关", " ", "问题", " ", "内容", " ",
		"说明", " ", "介绍", " ", "查看", " ", "查询", " ", "告诉我", " ", "什么", " ",
		"？", " ", "?", " ", "：", " ", "，", " ", "。", " ", "、", " ", "；", " ", "\n", " ",
	)
	text = replacer.Replace(text)

	knownPhrases := []string{
		"运维小助手",
		"验收标准",
		"技术方案",
		"待办清单",
		"开发待办清单",
		"真检索",
		"向量检索",
		"会话历史",
		"只读查询",
		"页面导航",
		"知识问答",
		"一期",
		"二期",
		"三期",
		"安全模块",
		"扫描任务",
		"安全资产",
		"漏洞工单",
		"漏洞管理",
		"用户手册",
		"我的资料",
		"修改密码",
		"登录密码",
		"用户密码",
		"个人密码",
		"部署记录",
		"迭代部署",
		"配置管理",
		"配置变更",
	}

	terms := make([]string, 0, 16)
	seen := map[string]struct{}{}
	addTerm := func(term string) {
		term = strings.TrimSpace(term)
		if len([]rune(term)) < 2 {
			return
		}
		if _, exists := seen[term]; exists {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}

	for _, part := range strings.Fields(text) {
		addTerm(part)
		for _, phrase := range knownPhrases {
			if strings.Contains(part, phrase) {
				addTerm(phrase)
			}
		}
	}

	if strings.Contains(text, "密码") {
		addTerm("密码")
	}
	if strings.Contains(text, "修改") && strings.Contains(text, "密码") {
		addTerm("修改密码")
	}
	if strings.Contains(text, "用户") && strings.Contains(text, "密码") {
		addTerm("用户密码")
	}
	if strings.Contains(text, "登录") && strings.Contains(text, "密码") {
		addTerm("登录密码")
	}
	if strings.Contains(text, "个人") && strings.Contains(text, "密码") {
		addTerm("个人密码")
	}

	return terms
}

func extractQueryModules(query string, terms []string) []string {
	lower := strings.ToLower(normalizeContent(query))
	moduleKeywords := []struct {
		module   string
		keywords []string
	}{
		{module: "assistant", keywords: []string{"运维小助手", "assistant", "rag", "真检索", "会话历史", "页面导航", "知识问答"}},
		{module: "manual", keywords: []string{"用户手册", "手册", "帮助文档", "我的资料", "修改密码", "登录密码", "profile"}},
		{module: "deploy", keywords: []string{"部署", "发布", "迭代部署", "部署记录", "归档", "聚合打包", "jenkins"}},
		{module: "testing", keywords: []string{"测试", "smoke", "验收测试"}},
		{module: "design", keywords: []string{"设计", "架构", "方案"}},
		{module: "structure", keywords: []string{"结构", "目录", "project-structure"}},
	}

	modules := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, entry := range moduleKeywords {
		for _, keyword := range entry.keywords {
			if strings.Contains(lower, keyword) || containsTerm(terms, keyword) {
				if _, exists := seen[entry.module]; exists {
					break
				}
				seen[entry.module] = struct{}{}
				modules = append(modules, entry.module)
				break
			}
		}
	}
	return modules
}

func trimKnowledgeQuery(text string) string {
	text = normalizeContent(strings.ToLower(text))
	replacer := strings.NewReplacer(
		"什么是", " ", "是什么", " ", "有哪些", " ", "有哪", " ", "哪里", " ", "在哪", " ",
		"如何", " ", "怎么", " ", "怎样", " ", "帮我", " ", "请问", " ", "一下", " ", "当前", " ",
		"最近", " ", "有关", " ", "关于", " ", "相关", " ", "问题", " ", "内容", " ", "说明", " ",
		"介绍", " ", "查看", " ", "查询", " ", "告诉我", " ", "？", " ", "?", " ",
	)
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func scoreDocumentTermMatches(queryTerms []string, chunk documentChunk) float64 {
	if len(queryTerms) == 0 {
		return 0
	}

	title := strings.ToLower(chunk.DocumentTitle)
	heading := strings.ToLower(chunk.Heading)
	path := strings.ToLower(chunk.Path)
	content := strings.ToLower(chunk.Content)

	score := 0.0
	for _, term := range queryTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if len([]rune(term)) < 2 {
			continue
		}

		switch {
		case title == term || heading == term:
			score += 6
		case strings.Contains(title, term):
			score += 4
		case strings.Contains(heading, term):
			score += 3.5
		case strings.Contains(path, term):
			score += 2.5
		case strings.Contains(content, term):
			score += 1
		}
	}
	return score
}

func scorePreferredModule(preferredModules []string, module string) float64 {
	if len(preferredModules) == 0 {
		return 0
	}
	if moduleIn(module, preferredModules) {
		return 4
	}
	return 0
}

func scoreQuestionTypeChunk(questionType string, chunk documentChunk) float64 {
	heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
	content := strings.ToLower(strings.TrimSpace(chunk.Content))

	switch questionType {
	case "howto":
		score := 0.0
		if containsAny(heading, "执行", "步骤", "操作流程", "使用", "实践") {
			score += 5
		}
		if containsAny(heading, "主流程", "流程", "打包参数", "用户操作流程") {
			score += 4
		}
		if containsAny(content, "1.", "2.", "3.", "步骤", "点击", "输入", "导航至") {
			score += 3
		}
		if containsAny(heading, "安全", "架构", "组件职责", "功能概述", "目标") {
			score -= 4
		}
		return score
	case "acceptance":
		if containsAny(heading, "验收标准", "验收", "标准") {
			return 5
		}
	case "roadmap":
		if containsAny(heading, "推荐开发顺序", "开发顺序", "phase", "阶段") {
			return 5
		}
	}

	return 0
}

func scoreFeatureDocPreference(queryTerms []string, chunk documentChunk) float64 {
	path := filepath.ToSlash(chunk.Path)
	heading := strings.ToLower(strings.TrimSpace(chunk.Heading))
	title := strings.ToLower(strings.TrimSpace(chunk.DocumentTitle))
	content := strings.ToLower(strings.TrimSpace(chunk.Content))

	hasTerm := func(target string) bool {
		for _, term := range queryTerms {
			if strings.Contains(term, target) || strings.Contains(target, term) {
				return true
			}
		}
		return false
	}

	score := 0.0
	if hasTerm("聚合打包") {
		switch path {
		case "docs/user_manual.md":
			if strings.Contains(heading, "聚合打包") {
				score += 8
			}
		case "docs/aggregate-package-feature-requirements.md":
			score += 7
			if containsAny(heading, "用户操作流程", "聚合打包") {
				score += 3
			}
		case "docs/aggregate-package-feature-design.md":
			score += 6
			if containsAny(heading, "聚合打包", "前端实现", "接口设计") {
				score += 2
			}
		case "docs/deploy.md":
			if !strings.Contains(heading, "聚合打包") && !strings.Contains(title, "聚合打包") {
				score -= 2
			}
		}
	}

	if hasTerm("修改密码") || hasTerm("用户密码") || hasTerm("登录密码") || hasTerm("个人密码") {
		matchesProfilePasswordDoc := containsAny(heading, "我的资料", "修改密码") ||
			containsAny(content, "/profile", "我的资料", "登录密码", "当前密码", "新密码", "修改密码")
		switch path {
		case "docs/user_manual.md":
			score += 6
			if matchesProfilePasswordDoc {
				score += 4
			}
			if containsAny(content, "/profile", "修改密码", "登录密码", "右上角用户下拉菜单") {
				score += 3
			}
		default:
			if matchesProfilePasswordDoc {
				score += 2
			} else {
				score -= 12
			}
		}
	}

	if hasSecurityScanIntent(queryTerms) {
		matchesSecurityDoc := matchesSecurityScanTopic(chunk, queryTerms)
		switch path {
		case "docs/user_manual.md":
			score += 7
			if matchesSecurityDoc {
				score += 6
			}
		default:
			if strings.Contains(path, "助手") {
				score -= 12
			} else if matchesSecurityDoc {
				score += 2
			}
		}
	}

	return score
}

func moduleIn(module string, preferredModules []string) bool {
	for _, preferred := range preferredModules {
		if preferred == module {
			return true
		}
	}
	return false
}

func containsTerm(terms []string, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	for _, term := range terms {
		if strings.EqualFold(term, keyword) {
			return true
		}
	}
	return false
}

func documentPriority(path string) int {
	switch filepath.ToSlash(path) {
	case "docs/运维小助手开发待办清单.md":
		return 6
	case "docs/运维小助手技术方案.md":
		return 5
	case "docs/基于qwen3-8b的运维小助手详细落地设计.md":
		return 4
	case "docs/user_manual.md":
		return 2
	case "docs/deploy.md", "docs/aggregate-package-design.md", "docs/testing.md", "docs/project-structure.md":
		return 1
	default:
		return 0
	}
}

func documentPriorityForQuery(path string, preferredModules []string, questionType string) int {
	normalizedPath := filepath.ToSlash(path)
	basePriority := documentPriority(normalizedPath)

	if questionType == "howto" && !moduleIn("assistant", preferredModules) {
		switch normalizedPath {
		case "docs/user_manual.md":
			return 9
		case "docs/aggregate-package-feature-requirements.md":
			return 8
		case "docs/aggregate-package-feature-design.md":
			return 7
		}
		if strings.Contains(normalizedPath, "助手") {
			return 0
		}
	}

	if len(preferredModules) == 0 {
		return basePriority
	}

	if moduleIn("deploy", preferredModules) || moduleIn("manual", preferredModules) {
		switch normalizedPath {
		case "docs/user_manual.md":
			return 8
		case "docs/aggregate-package-feature-requirements.md":
			return 7
		case "docs/aggregate-package-feature-design.md":
			return 6
		case "docs/deploy.md":
			return 5
		}
		if questionType == "howto" && strings.Contains(normalizedPath, "助手") {
			return 0
		}
	}

	return basePriority
}

func normalizeHeadingKey(heading string) string {
	heading = strings.ToLower(strings.TrimSpace(heading))
	heading = strings.TrimLeft(heading, "0123456789. ")
	heading = strings.Join(strings.Fields(heading), " ")
	return heading
}

func scoreKeywords(queryTerms, targetTerms []string) int {
	score := 0
	for _, queryTerm := range queryTerms {
		for _, targetTerm := range targetTerms {
			if queryTerm == targetTerm {
				score += 3
				break
			}
			if strings.Contains(targetTerm, queryTerm) || strings.Contains(queryTerm, targetTerm) {
				score += 1
				break
			}
		}
	}
	return score
}

func scoreModule(queryTerms []string, module string) int {
	for _, term := range queryTerms {
		if strings.Contains(term, module) || strings.Contains(module, term) {
			return 1
		}
	}
	return 0
}

func extractSnippet(content string, queryTerms []string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	contentRunes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	for _, term := range queryTerms {
		termRunes := []rune(strings.ToLower(strings.TrimSpace(term)))
		idx := indexRunes(lowerRunes, termRunes)
		if idx >= 0 {
			start := idx - 80
			if start < 0 {
				start = 0
			}
			end := idx + 180
			if end > len(contentRunes) {
				end = len(contentRunes)
			}
			return strings.TrimSpace(string(contentRunes[start:end]))
		}
	}
	if len(contentRunes) > 180 {
		return string(contentRunes[:180])
	}
	return content
}

func indexRunes(haystack, needle []rune) int {
	if len(haystack) == 0 || len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func normalizeContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\t", " ")
	return strings.TrimSpace(content)
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func trimNavigationQuery(query string) string {
	query = strings.TrimSpace(query)
	prefixes := []string{
		"打开",
		"进入",
		"跳转到",
		"跳到",
		"跳转",
		"去",
		"到",
		"查看",
		"看下",
		"看",
		"如何",
		"怎么",
		"怎样",
	}
	for _, prefix := range prefixes {
		query = strings.TrimSpace(strings.TrimPrefix(query, prefix))
	}
	return query
}
