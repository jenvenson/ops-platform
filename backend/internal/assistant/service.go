package assistant

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/edy/ops-platform/pkg/config"
)

type Service struct {
	cfg           *config.Config
	ollama        *ollamaClient
	knowledge     *knowledgeBase
	readonlyTools []assistantTool
}

func NewService(cfg *config.Config) *Service {
	var client *ollamaClient
	if cfg != nil && cfg.Assistant.Enabled && strings.EqualFold(cfg.Assistant.Provider, "ollama") {
		client = newOllamaClient(cfg.Assistant.OllamaBaseURL, cfg.Assistant.OllamaChatModel, cfg.Assistant.Temperature)
	}

	return &Service{
		cfg:           cfg,
		ollama:        client,
		knowledge:     loadKnowledgeBase(cfg, client),
		readonlyTools: buildReadonlyToolRegistry(),
	}
}

func (s *Service) GenerateReply(ctx context.Context, message string, history []historyMessage, pageContext *AssistantPageContext) (MessageResponse, int, int, int64) {
	start := time.Now()
	pageContext = sanitizePageContext(pageContext)
	intent := resolveIntentWithPageContext(message, pageContext, classifyAssistantIntent(message))
	actions := s.buildActions(message, intent, pageContext)
	citations := s.buildCitations(message, intent.Name, actions)
	toolResult := s.runReadonlyTools(message, intent, pageContext)

	answer, promptTokens, completionTokens, err := s.generateModelAnswer(ctx, intent.Name, message, history, citations, actions, toolResult, pageContext)
	model := s.responseModelName(answer, err)
	if err != nil || strings.TrimSpace(answer) == "" {
		answer = fallbackAnswer(message, intent.Name, citations, actions, toolResult)
	}

	decision := s.buildDecision(intent, message, history, pageContext, answer, citations, actions, toolResult)
	return newMessageResponse(decision, model),
		promptTokens,
		completionTokens,
		time.Since(start).Milliseconds()
}

func newMessageResponse(decision AssistantDecision, model string) MessageResponse {
	return MessageResponse{
		Intent:      decision.Intent.Name,
		Answer:      decision.Summary,
		Citations:   decision.Citations,
		Actions:     decision.Actions,
		ResultCards: decision.ResultCards,
		Decision:    &decision,
		Model:       model,
	}
}

func (s *Service) buildDecision(intent AssistantIntent, message string, history []historyMessage, pageContext *AssistantPageContext, answer string, citations []Citation, actions []Action, toolResult *toolContext) AssistantDecision {
	decision := AssistantDecision{
		Intent: intent,
		Context: AssistantContext{
			Message:      message,
			HistoryCount: len(history),
			PageContext:  sanitizePageContext(pageContext),
		},
		Summary:          answer,
		Citations:        citations,
		Actions:          actions,
		ResultCards:      toolCards(toolResult),
		RiskLevel:        deriveRiskLevel(intent),
		NeedConfirmation: intent.Name == "execution",
	}
	if plan := buildExecutionPlan(intent, actions, toolResult); plan != nil {
		decision.ExecutionPlan = plan
	}
	return decision
}

func sanitizePageContext(pageContext *AssistantPageContext) *AssistantPageContext {
	if pageContext == nil {
		return nil
	}

	clean := &AssistantPageContext{
		PagePath:   strings.TrimSpace(pageContext.PagePath),
		ModuleKey:  strings.TrimSpace(pageContext.ModuleKey),
		ObjectType: strings.TrimSpace(pageContext.ObjectType),
		ObjectID:   strings.TrimSpace(pageContext.ObjectID),
		PageTitle:  strings.TrimSpace(pageContext.PageTitle),
	}
	for _, id := range pageContext.SelectedRecordIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			clean.SelectedRecordIDs = append(clean.SelectedRecordIDs, id)
		}
	}
	for key, value := range pageContext.Filters {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if clean.Filters == nil {
			clean.Filters = make(map[string]string)
		}
		clean.Filters[key] = value
	}

	if clean.PagePath == "" && clean.ModuleKey == "" && clean.ObjectType == "" && clean.ObjectID == "" && clean.PageTitle == "" && len(clean.SelectedRecordIDs) == 0 && len(clean.Filters) == 0 {
		return nil
	}
	return clean
}

func toolCards(toolResult *toolContext) []ResultCard {
	if toolResult == nil || len(toolResult.Cards) == 0 {
		return nil
	}
	return toolResult.Cards
}

func deriveRiskLevel(intent AssistantIntent) string {
	switch intent.Name {
	case "execution":
		return "high"
	case "analysis":
		return "medium"
	default:
		return "low"
	}
}

func buildExecutionPlan(intent AssistantIntent, actions []Action, toolResult *toolContext) *AssistantExecutionPlan {
	if !intent.NeedTools && len(actions) == 0 {
		return nil
	}

	steps := make([]AssistantExecutionPlanStep, 0, 2)
	if intent.NeedTools {
		toolName := strings.TrimSpace(intent.SubIntent)
		if toolResult != nil && strings.TrimSpace(toolResult.ToolName) != "" {
			toolName = strings.TrimSpace(toolResult.ToolName)
		}
		if toolName == "" {
			toolName = intent.Name
		}
		steps = append(steps, AssistantExecutionPlanStep{
			Tool:     toolName,
			Purpose:  "resolve_context",
			Readonly: intent.Name == "readonly_query",
		})
	}
	if len(actions) > 0 && actions[0].Path != "" {
		steps = append(steps, AssistantExecutionPlanStep{
			Tool:     "navigate_to_page",
			Purpose:  actions[0].Path,
			Readonly: true,
		})
	}
	if len(steps) == 0 && toolResult == nil {
		return nil
	}

	return &AssistantExecutionPlan{
		PlanID: executionPlanID(intent, steps),
		Steps:  steps,
	}
}

func executionPlanID(intent AssistantIntent, steps []AssistantExecutionPlanStep) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(intent.Name))
	_, _ = hasher.Write([]byte(intent.SubIntent))
	for _, step := range steps {
		_, _ = hasher.Write([]byte(step.Tool))
		_, _ = hasher.Write([]byte(step.Purpose))
	}
	return fmt.Sprintf("plan_%x", hasher.Sum32())
}

func (s *Service) modelName() string {
	if s != nil && s.ollama != nil && s.ollama.model != "" {
		return s.ollama.model
	}
	return "ops-assistant"
}

func (s *Service) responseModelName(answer string, err error) string {
	if err == nil && strings.TrimSpace(answer) != "" {
		return s.modelName()
	}
	return "ops-assistant-fallback"
}

func (s *Service) generateModelAnswer(ctx context.Context, intent, message string, history []historyMessage, citations []Citation, actions []Action, toolResult *toolContext, pageContext *AssistantPageContext) (string, int, int, error) {
	if s == nil || s.ollama == nil {
		return "", 0, 0, fmt.Errorf("assistant model unavailable")
	}

	promptHistory := append(make([]historyMessage, 0, len(history)+1), history...)
	promptHistory = append(promptHistory, historyMessage{
		Role:    "user",
		Content: buildUserPrompt(intent, message, citations, actions, toolResult, pageContext),
	})

	return s.ollama.chat(ctx, systemPrompt, promptHistory)
}

const systemPrompt = "你是 OPS Platform 的运维小助手。你的职责是帮助用户理解平台功能、定位页面入口、总结查询结果。你只能根据给定上下文回答，不要编造不存在的页面、状态或操作结果。请使用中文，回答简洁、明确、可执行。"

func buildUserPrompt(intent, message string, citations []Citation, actions []Action, toolResult *toolContext, pageContext *AssistantPageContext) string {
	var builder strings.Builder
	builder.WriteString("用户问题：")
	builder.WriteString(message)
	builder.WriteString("\n\n意图：")
	builder.WriteString(intent)

	if contextSummary := summarizePageContext(pageContext); contextSummary != "" {
		builder.WriteString("\n\n当前页面上下文：\n")
		builder.WriteString(contextSummary)
	}

	if len(citations) > 0 {
		builder.WriteString("\n\n可参考资料：")
		for _, citation := range citations {
			builder.WriteString("\n- ")
			builder.WriteString(citation.Title)
			if citation.Path != "" {
				builder.WriteString(" (")
				builder.WriteString(citation.Path)
				builder.WriteString(")")
			}
			if citation.Snippet != "" {
				builder.WriteString("：")
				builder.WriteString(citation.Snippet)
			}
		}
	}

	if len(actions) > 0 {
		builder.WriteString("\n\n可导航页面：")
		for _, action := range actions {
			if action.Path == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(action.Label)
			builder.WriteString(" -> ")
			builder.WriteString(action.Path)
		}
	}

	if toolResult != nil && strings.TrimSpace(toolResult.Summary) != "" {
		builder.WriteString("\n\n实时查询结果：\n")
		builder.WriteString(toolResult.Summary)
	}

	builder.WriteString("\n\n请先给结论，再给简短操作建议。如果有页面入口，请自然提到路径。")
	return builder.String()
}

func resolveIntentWithPageContext(message string, pageContext *AssistantPageContext, intent AssistantIntent) AssistantIntent {
	if pageContext == nil {
		return intent
	}

	if intent.Name == "readonly_query" && intent.SubIntent != "" && intent.SubIntent != "generic_readonly_query" {
		return intent
	}

	if !shouldPromoteContextualReadonlyQuery(message, intent, pageContext) {
		return intent
	}

	intent.Name = "readonly_query"
	intent.NeedTools = true
	if intent.Confidence < 0.82 {
		intent.Confidence = 0.82
	}
	if strings.TrimSpace(intent.SubIntent) == "" || intent.SubIntent == "generic_readonly_query" {
		if subIntent := pageContextReadonlySubIntent(pageContext); subIntent != "" {
			intent.SubIntent = subIntent
		}
	}
	return intent
}

func shouldPromoteContextualReadonlyQuery(message string, intent AssistantIntent, pageContext *AssistantPageContext) bool {
	if pageContext == nil || pageContextReadonlyTool(pageContext) == "" {
		return shouldUseFocusedObjectQuery(message, pageContext)
	}

	if intent.Name == "readonly_query" {
		return true
	}

	if intent.Name != "fallback" {
		return shouldUseFocusedObjectQuery(message, pageContext)
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	return shouldUseFocusedObjectQuery(message, pageContext) || containsAny(msg,
		"最近", "最新", "当前", "哪些", "有哪些", "多少", "状态", "失败", "成功", "异常", "恢复", "运行",
		"处理中", "待处理", "在线", "离线", "高危", "严重", "下载", "产物", "文件", "链接", "动作", "变化",
	)
}

func pageContextReadonlySubIntent(pageContext *AssistantPageContext) string {
	switch pageContextReadonlyTool(pageContext) {
	case "query_release_history":
		return "deploy_history_query"
	case "query_archive_history":
		return "archive_history_query"
	case "query_aggregate_history":
		return "aggregate_history_query"
	case "query_alert_events":
		return "alert_event_query"
	case "query_vulnerability_summary":
		return "vulnerability_query"
	case "query_security_scan_tasks":
		return "security_scan_task_query"
	case "query_security_assets":
		return "security_asset_query"
	case "query_security_tickets":
		return "security_ticket_query"
	case "query_projects":
		return "project_query"
	case "query_environments":
		return "environment_query"
	case "query_servers":
		return "server_query"
	case "query_applications":
		return "application_query"
	default:
		return ""
	}
}

func pageContextReadonlyTool(pageContext *AssistantPageContext) string {
	if pageContext == nil {
		return ""
	}

	switch normalizedPagePath(pageContext.PagePath) {
	case "/deploy/history":
		return "query_release_history"
	case "/deploy/archived":
		return "query_archive_history"
	case "/deploy/aggregated-history":
		return "query_aggregate_history"
	case "/alarm", "/alarm/events":
		return "query_alert_events"
	case "/security/tasks":
		return "query_security_scan_tasks"
	case "/security/assets":
		return "query_security_assets"
	case "/security/vulnerabilities":
		return "query_vulnerability_summary"
	case "/security/tickets":
		return "query_security_tickets"
	case "/cmdb/projects":
		return "query_projects"
	case "/cmdb/environments":
		return "query_environments"
	case "/cmdb/servers":
		return "query_servers"
	case "/cmdb/applications":
		return "query_applications"
	default:
		return ""
	}
}

func normalizedPagePath(path string) string {
	path = strings.TrimSpace(strings.ToLower(path))
	switch path {
	case "/alarm":
		return "/alarm"
	default:
		return path
	}
}

func summarizePageContext(pageContext *AssistantPageContext) string {
	if pageContext == nil {
		return ""
	}

	lines := make([]string, 0, 5)
	if pageContext.PageTitle != "" {
		lines = append(lines, "- 页面标题："+pageContext.PageTitle)
	}
	if pageContext.PagePath != "" {
		lines = append(lines, "- 页面路径："+pageContext.PagePath)
	}
	if pageContext.ModuleKey != "" {
		lines = append(lines, "- 所属模块："+pageContext.ModuleKey)
	}
	if pageContext.ObjectType != "" || pageContext.ObjectID != "" {
		objectParts := make([]string, 0, 2)
		if pageContext.ObjectType != "" {
			objectParts = append(objectParts, pageContext.ObjectType)
		}
		if pageContext.ObjectID != "" {
			objectParts = append(objectParts, pageContext.ObjectID)
		}
		if len(objectParts) == 0 {
			objectParts = append(objectParts, "unknown")
		}
		lines = append(lines, "- 当前对象："+strings.Join(objectParts, " "))
	}
	if len(pageContext.SelectedRecordIDs) > 0 {
		lines = append(lines, "- 已选记录："+strings.Join(pageContext.SelectedRecordIDs, ", "))
	}
	if len(pageContext.Filters) > 0 {
		filterLines := make([]string, 0, len(pageContext.Filters))
		for _, key := range sortedFilterKeys(pageContext.Filters) {
			filterLines = append(filterLines, key+"="+pageContext.Filters[key])
		}
		lines = append(lines, "- 当前筛选："+strings.Join(filterLines, "，"))
	}
	return strings.Join(lines, "\n")
}

func sortedFilterKeys(filters map[string]string) []string {
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func classifyIntent(message string) string {
	return classifyAssistantIntent(message).Name
}

func classifyAssistantIntent(message string) AssistantIntent {
	msg := strings.ToLower(message)
	switch {
	case containsAny(msg, "打开", "进入", "跳转", "在哪", "哪里", "页面", "菜单"):
		return AssistantIntent{Name: "page_navigation", Confidence: 0.95}
	case containsAny(msg, "如何查看", "怎么看", "查看监控大屏", "查看部署记录", "如何查看监控大屏", "如何查看部署记录"):
		return AssistantIntent{Name: "knowledge_qa", SubIntent: classifyKnowledgeQuestion(message), Confidence: 0.9}
	case containsAny(msg, "告警", "报警") && containsAny(msg, "最新", "最近") && containsAny(msg, "动作", "变化", "处理"):
		return AssistantIntent{Name: "readonly_query", SubIntent: "alert_event_query", Confidence: 0.88, NeedTools: true}
	case containsAny(msg, "查看", "查询", "最近", "有哪些", "多少", "状态"):
		return AssistantIntent{Name: "readonly_query", SubIntent: inferReadonlySubIntent(msg), Confidence: 0.85, NeedTools: true}
	case containsAny(msg, "如何", "怎么", "帮助", "说明", "介绍", "是什么", "什么是", "验收", "标准", "文档", "手册"):
		return AssistantIntent{Name: "knowledge_qa", SubIntent: classifyKnowledgeQuestion(message), Confidence: 0.85}
	default:
		return AssistantIntent{Name: "fallback", Confidence: 0.4}
	}
}

func inferReadonlySubIntent(msg string) string {
	switch {
	case containsAny(msg, "归档历史", "归档记录"):
		return "archive_history_query"
	case containsAny(msg, "部署记录", "发布记录", "部署历史"):
		return "deploy_history_query"
	case containsAny(msg, "告警", "报警"):
		return "alert_event_query"
	case containsAny(msg, "漏洞"):
		return "vulnerability_query"
	default:
		return "generic_readonly_query"
	}
}

func (s *Service) buildCitations(message, intent string, actions []Action) []Citation {
	if s != nil && s.knowledge != nil {
		switch intent {
		case "knowledge_qa":
			if classifyKnowledgeQuestion(message) == "onboarding" && isPlatformUsageQuestion(message) {
				citations := s.onboardingCitations()
				if len(citations) > 0 {
					return citations
				}
			}
			if shouldUseActionGuidedCitations(message, actions) {
				if shouldPreferMessageSpecificCitations(message, actions) {
					citationAction := actionForMessageSpecificCitations(message, actions[0])
					citations := s.knowledge.searchDocumentsForAction(citationAction, s.cfg.Assistant.TopK)
					if len(citations) > 0 {
						return citations
					}
				}
				citations := s.knowledge.searchDocumentsForAction(actions[0], s.cfg.Assistant.TopK)
				if len(citations) > 0 {
					return citations
				}
			}
			citations := s.knowledge.searchDocuments(message, s.cfg.Assistant.TopK)
			if len(citations) > 0 {
				citations = filterCitationsForAction(message, citations, actions)
				if shouldSuppressUnrelatedCitations(message, citations, actions) {
					return nil
				}
				return citations
			}
		}
	}

	switch intent {
	case "knowledge_qa":
		return defaultDocumentsAsCitations()
	default:
		return nil
	}
}

func (s *Service) onboardingCitations() []Citation {
	return []Citation{{
		Title:   "user_manual / 快速入门",
		Path:    "docs/user_manual.md",
		Snippet: "建议先配置项目、环境、主机、应用，再执行部署与制品操作，最后查看监控告警和安全页面。",
	}}
}

func shouldSuppressUnrelatedCitations(message string, citations []Citation, actions []Action) bool {
	if len(citations) == 0 || len(actions) > 0 {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	if !containsAny(msg, "会话", "聊天记录", "聊天历史", "消息记录", "删除会话", "归档会话") {
		return false
	}

	for _, citation := range citations {
		text := strings.ToLower(citation.Title + "\n" + citation.Path + "\n" + citation.Snippet)
		if containsAny(text, "会话", "聊天", "消息") {
			return false
		}
	}

	return true
}

func filterCitationsForAction(message string, citations []Citation, actions []Action) []Citation {
	if len(citations) == 0 || !shouldUseActionGuidedCitations(message, actions) {
		return citations
	}

	exactPathMatches := make([]Citation, 0, len(citations))
	for _, citation := range citations {
		text := strings.ToLower(citation.Title + "\n" + citation.Path + "\n" + citation.Snippet)
		if actions[0].Path != "" && strings.Contains(text, strings.ToLower(actions[0].Path)) {
			exactPathMatches = append(exactPathMatches, citation)
		}
	}
	if len(exactPathMatches) > 0 {
		return exactPathMatches
	}

	tokens := actionFocusTokens(actions[0])
	if len(tokens) == 0 {
		return citations
	}

	filtered := make([]Citation, 0, len(citations))
	for _, citation := range citations {
		text := strings.ToLower(citation.Title + "\n" + citation.Path + "\n" + citation.Snippet)
		for _, token := range tokens {
			if token != "" && strings.Contains(text, strings.ToLower(token)) {
				filtered = append(filtered, citation)
				break
			}
		}
	}
	if len(filtered) > 0 {
		return filtered
	}
	return citations
}

func shouldUseActionGuidedCitations(message string, actions []Action) bool {
	if len(actions) == 0 {
		return false
	}
	switch classifyKnowledgeQuestion(message) {
	case "howto":
		return true
	default:
		msg := strings.ToLower(strings.TrimSpace(message))
		return containsAny(msg, "查看", "查看一下", "看看", "如何查看", "怎么看")
	}
}

func shouldPreferMessageSpecificCitations(message string, actions []Action) bool {
	if len(actions) == 0 {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	if actions[0].Path == "/deploy/archived" && containsAny(msg, "下载", "下载地址", "查看文件", "归档产物", "归档包", "复制链接", "新窗口") {
		return true
	}
	if actions[0].Path == "/deploy/aggregated-history" && containsAny(msg, "下载", "下载地址", "查看文件", "聚合包", "复制链接", "新窗口") {
		return true
	}

	return false
}

func actionForMessageSpecificCitations(message string, action Action) Action {
	msg := strings.ToLower(strings.TrimSpace(message))
	if action.Path == "/deploy/archived" && containsAny(msg, "下载", "下载地址", "查看文件", "归档产物", "归档包", "复制链接", "新窗口") {
		action.Label = "下载归档产物"
		return action
	}
	if action.Path == "/deploy/aggregated-history" && containsAny(msg, "下载", "下载地址", "查看文件", "聚合包", "复制链接", "新窗口") {
		action.Label = "下载聚合包"
		return action
	}

	return action
}

func actionFocusTokens(action Action) []string {
	label := strings.TrimPrefix(action.Label, "打开")
	switch action.Path {
	case "/cmdb/projects":
		return []string{action.Path, label, "项目管理", "创建项目", "新建项目"}
	case "/cmdb/servers":
		return []string{action.Path, label, "服务器管理", "主机管理", "添加服务器", "新建服务器"}
	case "/cmdb/environments":
		return []string{action.Path, label, "环境管理", "创建环境", "新建环境"}
	case "/cmdb/applications":
		return []string{action.Path, label, "应用管理", "应用流水线管理", "流水线", "新增应用流水线", "新增流水线", "新建流水线", "创建流水线"}
	case "/deploy/aggregate-package":
		return []string{action.Path, label, "聚合打包", "安装包聚合打包"}
	case "/deploy/aggregated-history":
		return []string{action.Path, label, "聚合历史", "查看聚合历史", "聚合记录", "下载地址", "查看文件", "聚合包", "下载聚合包"}
	case "/deploy/history":
		return []string{action.Path, label, "部署记录", "部署历史", "发布历史", "查看部署记录", "部署结果", "部署状态"}
	case "/deploy/archive":
		return []string{action.Path, label, "归档打包", "归档管理", "应用归档", "归档", "如何归档", "执行归档", "归档版本", "版本归档"}
	case "/deploy/archived":
		return []string{action.Path, label, "归档历史", "历史归档", "查看归档历史", "归档记录", "历史记录", "归档详情", "归档文件", "归档产物", "下载归档产物"}
	case "/monitor/bigscreen":
		return []string{action.Path, label, "监控大屏", "监控中心", "大屏", "自定义大屏", "仪表板", "实时监控"}
	case "/consul/config":
		return []string{action.Path, label, "Consul配置", "配置管理", "添加配置", "新增配置", "编辑配置", "测试连接"}
	case "/consul/batch-all":
		return []string{action.Path, label, "批量配置下发", "执行批量配置下发", "源后缀", "目标后缀", "Consul配置"}
	case "/profile":
		return []string{action.Path, label, "我的资料", "修改密码", "登录密码"}
	case "/user-manual":
		return []string{action.Path, label, "用户手册", "使用手册", "快速入门", "平台使用", "如何使用平台"}
	default:
		return []string{action.Path, label}
	}
}

func defaultDocumentsAsCitations() []Citation {
	docs := defaultDocuments()
	citations := make([]Citation, 0, len(docs))
	for _, doc := range docs {
		citations = append(citations, Citation{
			Title:   doc.Title,
			Path:    doc.Path,
			Snippet: doc.Content,
		})
	}
	return citations
}

func (s *Service) buildActions(message string, intent AssistantIntent, pageContext *AssistantPageContext) []Action {
	if s != nil && s.knowledge != nil {
		actions := s.knowledge.searchRoutes(message, 2)
		if len(actions) > 0 {
			return filterActionsForKnowledgeQuery(message, actions)
		}
	}

	if action := actionForPageContext(s, pageContext); action != nil && shouldSuggestCurrentPageAction(message, intent) {
		return []Action{*action}
	}

	return nil
}

func actionForPageContext(s *Service, pageContext *AssistantPageContext) *Action {
	if s == nil || s.knowledge == nil || pageContext == nil {
		return nil
	}

	path := normalizedPagePath(pageContext.PagePath)
	if path == "" {
		return nil
	}

	for _, route := range s.knowledge.routes {
		if route.Path != path {
			continue
		}
		return &Action{
			Type:  "navigate",
			Label: "打开" + route.Title,
			Path:  route.Path,
		}
	}
	return nil
}

func shouldSuggestCurrentPageAction(message string, intent AssistantIntent) bool {
	if intent.Name == "page_navigation" || intent.Name == "readonly_query" {
		return true
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	return containsAny(msg, "如何", "怎么", "步骤", "打开", "进入", "下载", "查看文件", "本页", "当前页", "这里")
}

func filterActionsForKnowledgeQuery(message string, actions []Action) []Action {
	if len(actions) <= 1 {
		return actions
	}

	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "归档历史", "历史归档", "查看归档历史") || (containsAny(msg, "归档") && containsAny(msg, "下载", "产物", "文件", "链接")) {
		filtered := make([]Action, 0, len(actions))
		for _, action := range actions {
			if action.Path == "/deploy/archived" {
				filtered = append(filtered, action)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}
	if containsAny(msg, "聚合历史", "查看聚合历史") || (containsAny(msg, "聚合") && containsAny(msg, "下载", "文件", "链接", "聚合包")) {
		filtered := make([]Action, 0, len(actions))
		for _, action := range actions {
			if action.Path == "/deploy/aggregated-history" {
				filtered = append(filtered, action)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
	}

	return actions
}

func fallbackAnswer(message, intent string, citations []Citation, actions []Action, toolResult *toolContext) string {
	switch intent {
	case "page_navigation":
		if len(actions) > 0 {
			return fmt.Sprintf("可以从对应功能页面进入，建议直接打开 %s。", actions[0].Path)
		}
		return "可以描述一下具体模块名称，我再帮你定位页面入口。"
	case "readonly_query":
		if toolResult != nil && toolResult.Summary != "" {
			return toolResult.Summary
		}
		if len(actions) > 0 {
			return fmt.Sprintf("这个问题更适合结合页面数据查看，建议先打开 %s，再根据列表结果继续分析。", actions[0].Path)
		}
		return "当前版本优先提供入口定位和基础查询建议，如需实时数据查询，我可以继续帮你收敛到具体模块。"
	case "knowledge_qa":
		if len(citations) > 0 {
			return summarizeCitationsForQuestion(message, citations)
		}
		return "运维小助手一期主要支持平台使用问答、页面导航和只读查询。你可以直接问我某个功能怎么用，或者让我带你到对应页面。"
	default:
		return "我是 OPS Platform 的运维小助手。目前已支持基础问答、页面导航和只读查询建议。你可以继续问我部署、监控、告警、安全或系统管理相关问题。"
	}
}

func summarizeCitationsForQuestion(message string, citations []Citation) string {
	if len(citations) == 0 {
		return ""
	}

	questionType := classifyKnowledgeQuestion(message)
	if questionType == "onboarding" && isPlatformUsageQuestion(message) {
		return summarizePlatformUsageQuestion()
	}
	bullets := make([]string, 0, 6)
	seenBullets := map[string]struct{}{}
	for _, citation := range citations {
		for _, bullet := range extractSummaryBullets(citation.Snippet) {
			if _, exists := seenBullets[bullet]; exists {
				continue
			}
			seenBullets[bullet] = struct{}{}
			bullets = append(bullets, bullet)
			if len(bullets) >= 6 {
				break
			}
		}
		if len(bullets) >= 6 {
			break
		}
	}
	if len(bullets) > 0 {
		maxBullets := summaryBulletLimit(questionType)
		if len(bullets) > maxBullets {
			bullets = bullets[:maxBullets]
		}
		return summaryPrefix(questionType) + "\n- " + strings.Join(bullets, "\n- ")
	}

	parts := make([]string, 0, minInt(len(citations), 2))
	for i, citation := range citations {
		if i >= 2 {
			break
		}
		text := strings.TrimSpace(citation.Snippet)
		if text == "" {
			text = strings.TrimSpace(citation.Title)
		}
		if text == "" {
			continue
		}
		parts = append(parts, shorten(text, 120))
	}
	if len(parts) == 0 {
		return "我找到了相关文档，但暂时无法提炼出有效摘要。你可以继续缩小问题范围，我再进一步检索。"
	}
	return summaryPrefix(questionType) + "\n- " + strings.Join(parts, "\n- ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func containsAny(message string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func classifyKnowledgeQuestion(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case isPlatformUsageQuestion(msg):
		return "onboarding"
	case containsAny(msg, "验收", "标准", "要求", "条件"):
		return "acceptance"
	case containsAny(msg, "顺序", "路线图", "roadmap", "阶段", "优先级", "下一步"):
		return "roadmap"
	case containsAny(msg, "范围", "一期做什么", "二期做什么", "三期做什么", "支持哪些", "支持什么"):
		return "scope"
	case containsAny(msg, "架构", "设计", "模块", "原理", "组成"):
		return "architecture"
	case containsAny(msg, "如何", "怎么", "步骤", "怎样", "使用", "配置", "如何查看", "怎么看", "查看", "看看"):
		return "howto"
	default:
		return "general"
	}
}

func summaryPrefix(questionType string) string {
	switch questionType {
	case "onboarding":
		return "可以按下面这个顺序上手平台："
	case "acceptance":
		return "根据当前检索到的文档内容，验收标准主要包括："
	case "roadmap":
		return "根据当前检索到的文档内容，建议的开发顺序如下："
	case "scope":
		return "根据当前检索到的文档内容，当前范围主要包括："
	case "architecture":
		return "根据当前检索到的文档内容，核心设计要点如下："
	case "howto":
		return "根据当前检索到的文档内容，可以按下面方式处理："
	default:
		return "根据当前检索到的文档内容，结论如下："
	}
}

func summaryBulletLimit(questionType string) int {
	switch questionType {
	case "onboarding":
		return 6
	case "acceptance", "roadmap", "scope", "architecture", "howto":
		return 5
	default:
		return 6
	}
}

func isPlatformUsageQuestion(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	return containsAny(msg, "平台") && containsAny(msg, "如何使用", "怎么用", "怎么使用", "如何用", "上手", "入门")
}

func summarizePlatformUsageQuestion() string {
	return strings.Join([]string{
		"可以按下面这个顺序上手平台：",
		"- 先配置基础数据：`/cmdb/projects`、`/cmdb/environments`、`/cmdb/servers`、`/cmdb/applications`",
		"- 再做发布和制品操作：`/deploy/release`、`/deploy/history`、`/deploy/archived`、`/deploy/aggregated-history`",
		"- 日常运行状态主要看：`/monitor/bigscreen`、`/alarm/events`",
		"- 安全排查主要看：`/security/tasks`、`/security/vulnerabilities`",
		"- 如果想系统了解全部功能，可以打开 `/user-manual` 查看完整说明",
	}, "\n")
}

func extractSummaryBullets(snippet string) []string {
	lines := strings.Split(strings.TrimSpace(snippet), "\n")
	bullets := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "：") || strings.HasSuffix(line, ":") {
			continue
		}
		line = strings.TrimLeft(line, "-*0123456789. ")
		line = strings.TrimSpace(line)
		if line == "" || len([]rune(line)) < 4 {
			continue
		}
		bullets = append(bullets, shorten(line, 80))
	}
	return bullets
}
