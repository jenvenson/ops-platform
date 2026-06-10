// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import "strings"

type toolContext struct {
	ToolName string
	Summary  string
	Cards    []ResultCard
	Error    string
}

type assistantTool struct {
	Name     string
	Match    func(message string, intent AssistantIntent) bool
	Execute  func(s *Service, message string, pageContext *AssistantPageContext) *toolContext
	Purpose  string
	Readonly bool
}

func (s *Service) runReadonlyTools(message string, intent AssistantIntent, pageContext *AssistantPageContext) *toolContext {
	if intent.Name != "readonly_query" {
		return nil
	}

	if preferred := pageContextReadonlyTool(pageContext); preferred != "" {
		if result := s.runReadonlyToolByName(preferred, message, pageContext); result != nil {
			return result
		}
	}

	for _, tool := range s.readonlyTools {
		if tool.Match == nil || !tool.Match(message, intent) {
			continue
		}
		result := tool.Execute(s, strings.ToLower(message), pageContext)
		if result == nil {
			return nil
		}
		if strings.TrimSpace(result.ToolName) == "" {
			result.ToolName = tool.Name
		}
		if strings.TrimSpace(result.Error) != "" && strings.TrimSpace(result.Summary) == "" {
			result.Summary = result.Error
		}
		return result
	}
	return nil
}

func (s *Service) runReadonlyToolByName(name, message string, pageContext *AssistantPageContext) *toolContext {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	for _, tool := range s.readonlyTools {
		if tool.Name != name || tool.Execute == nil {
			continue
		}
		result := tool.Execute(s, strings.ToLower(message), pageContext)
		if result == nil {
			return nil
		}
		if strings.TrimSpace(result.ToolName) == "" {
			result.ToolName = tool.Name
		}
		if strings.TrimSpace(result.Error) != "" && strings.TrimSpace(result.Summary) == "" {
			result.Summary = result.Error
		}
		return result
	}

	return nil
}

func buildReadonlyToolRegistry() []assistantTool {
	return []assistantTool{
		{
			Name: "query_alert_events",
			Match: func(message string, intent AssistantIntent) bool {
				return intent.SubIntent == "alert_event_query" || containsAny(message, "告警", "报警", "事件")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryAlertEvents(message, pageContext)
			},
			Purpose:  "load_alert_events",
			Readonly: true,
		},
		{
			Name: "query_applications",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "应用", "流水线") && containsAny(message, "最近", "部署", "发布", "失败", "频繁", "配置", "缺少")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryApplications(message, pageContext)
			},
			Purpose:  "load_applications",
			Readonly: true,
		},
		{
			Name: "query_release_history",
			Match: func(message string, intent AssistantIntent) bool {
				return intent.SubIntent == "deploy_history_query" || (containsAny(message, "部署", "发布") && containsAny(message, "历史", "记录", "失败", "成功", "状态", "最近"))
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryDeployRecords(message, pageContext)
			},
			Purpose:  "load_release_history",
			Readonly: true,
		},
		{
			Name: "query_archive_history",
			Match: func(message string, intent AssistantIntent) bool {
				return intent.SubIntent == "archive_history_query" || (containsAny(message, "归档") && containsAny(message, "历史", "记录", "失败", "成功", "状态", "最近"))
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryArchiveHistory(message, pageContext)
			},
			Purpose:  "load_archive_history",
			Readonly: true,
		},
		{
			Name: "query_aggregate_history",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "聚合", "打包") && containsAny(message, "历史", "记录", "失败", "成功", "状态", "最近")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryAggregatedHistory(message, pageContext)
			},
			Purpose:  "load_aggregate_history",
			Readonly: true,
		},
		{
			Name: "query_security_tickets",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "工单", "ticket") && containsAny(message, "漏洞", "最近", "状态", "处理中", "待处理", "已修复", "列表", "记录")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.querySecurityTickets(message, pageContext)
			},
			Purpose:  "load_security_tickets",
			Readonly: true,
		},
		{
			Name: "query_vulnerability_summary",
			Match: func(message string, intent AssistantIntent) bool {
				return intent.SubIntent == "vulnerability_query" || (containsAny(message, "漏洞", "风险") && containsAny(message, "最近", "状态", "高危", "严重", "处理中", "待处理", "数量", "列表"))
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.querySecurityVulnerabilities(message, pageContext)
			},
			Purpose:  "load_vulnerability_summary",
			Readonly: true,
		},
		{
			Name: "query_security_assets",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "资产") && containsAny(message, "安全", "最近", "状态", "在线", "离线", "web", "数据库", "服务器", "列表")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.querySecurityAssets(message, pageContext)
			},
			Purpose:  "load_security_assets",
			Readonly: true,
		},
		{
			Name: "query_security_scan_tasks",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "扫描", "任务") && containsAny(message, "最近", "状态", "运行", "失败", "完成", "列表", "记录")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.querySecurityScanTasks(message, pageContext)
			},
			Purpose:  "load_security_scan_tasks",
			Readonly: true,
		},
		{
			Name: "query_projects",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "项目", "project")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryProjects(message, pageContext)
			},
			Purpose:  "load_projects",
			Readonly: true,
		},
		{
			Name: "query_environments",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "环境", "environment")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryEnvironments(message, pageContext)
			},
			Purpose:  "load_environments",
			Readonly: true,
		},
		{
			Name: "query_servers",
			Match: func(message string, intent AssistantIntent) bool {
				return containsAny(message, "主机", "服务器", "server")
			},
			Execute: func(s *Service, message string, pageContext *AssistantPageContext) *toolContext {
				return s.queryServers(message, pageContext)
			},
			Purpose:  "load_servers",
			Readonly: true,
		},
	}
}