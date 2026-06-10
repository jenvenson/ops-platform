// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenvenson/ops-platform/internal/cmdb"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

func (s *Service) queryDeployRecords(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&cmdb.DeployRecord{})

	if shouldUseFocusedObjectQuery(message, pageContext, "deploy_record") {
		if recordID, ok := pageObjectID(pageContext); ok {
			var record cmdb.DeployRecord
			if err := database.DB.Where("id = ?", recordID).First(&record).Error; err == nil {
				return buildFocusedDeployRecordContext(message, record)
			}
		}
	}

	if appName := pageFilterValue(pageContext, "appName"); appName != "" {
		query = query.Where("app_name LIKE ?", "%"+appName+"%")
	}
	if envID, ok := pageFilterUint(pageContext, "envId"); ok {
		query = query.Where("env_id = ?", envID)
	}
	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if triggeredBy := pageFilterValue(pageContext, "triggeredBy"); triggeredBy != "" {
		query = query.Where("triggered_by LIKE ?", "%"+triggeredBy+"%")
	}
	query = applyCreatedAtRange(query, pageContext)

	if containsAny(msg, "失败", "failed") {
		query = query.Where("status = ?", "failed")
	} else if containsAny(msg, "成功", "success") {
		query = query.Where("status = ?", "success")
	}

	if keyword := extractPrimaryKeyword(message, "部署", "发布", "历史", "记录", "最近", "失败", "成功", "状态"); keyword != "" {
		query = query.Where(
			"app_name LIKE ? OR env_name LIKE ? OR project_code LIKE ? OR triggered_by LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var records []cmdb.DeployRecord
	if err := query.Order("created_at desc").Limit(5).Find(&records).Error; err != nil {
		return &toolContext{ToolName: "query_release_history", Summary: "部署历史查询失败，当前无法读取发布记录。", Error: "deploy history query failed"}
	}
	if len(records) == 0 {
		return &toolContext{ToolName: "query_release_history", Summary: "当前没有匹配的部署历史记录。"}
	}

	conclusion := summarizeDeployRecords(message, records)
	lines := make([]string, 0, len(records))
	cards := make([]ResultCard, 0, len(records))
	for _, record := range records {
		line := fmt.Sprintf("- 应用：%s，环境：%s，状态：%s，时间：%s", record.AppName, record.EnvName, record.Status, record.CreatedAt.Format("2006-01-02 15:04"))
		if record.ErrorMessage != "" && record.Status == "failed" {
			line += "，错误：" + shorten(record.ErrorMessage, 60)
		}
		lines = append(lines, line)
		cards = append(cards, ResultCard{
			Title:      record.AppName,
			Subtitle:   fmt.Sprintf("环境：%s | 状态：%s", record.EnvName, record.Status),
			Meta:       record.CreatedAt.Format("2006-01-02 15:04"),
			ToolName:   "deploy.records",
			SourceType: "deploy",
		})
	}

	return &toolContext{
		ToolName: "query_release_history",
		Summary:  conclusion + "\n最近记录：\n" + strings.Join(lines, "\n"),
		Cards:    cards,
	}
}

func (s *Service) queryArchiveHistory(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&cmdb.ArchiveRecord{})

	if shouldUseFocusedObjectQuery(message, pageContext, "archive_record") {
		if recordID, ok := pageObjectID(pageContext); ok {
			var record cmdb.ArchiveRecord
			if err := database.DB.Where("id = ?", recordID).First(&record).Error; err == nil {
				return buildFocusedArchiveRecordContext(message, record)
			}
		}
	}

	if appName := pageFilterValue(pageContext, "appName"); appName != "" {
		query = query.Where("app_name LIKE ?", "%"+appName+"%")
	}
	if envID, ok := pageFilterUint(pageContext, "envId"); ok {
		query = query.Where("env_id = ?", envID)
	}
	query = applyCreatedAtRange(query, pageContext)

	if containsAny(msg, "失败", "failed") {
		query = query.Where("status = ?", "failed")
	} else if containsAny(msg, "成功", "success") {
		query = query.Where("status = ?", "success")
	} else if containsAny(msg, "运行", "处理中", "running", "pending") {
		query = query.Where("status IN ?", []string{"pending", "running"})
	}

	if keyword := archiveQueryKeyword(message); keyword != "" {
		query = query.Where(
			"app_name LIKE ? OR env_name LIKE ? OR project_code LIKE ? OR operator LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var records []cmdb.ArchiveRecord
	if err := query.Order("created_at desc").Limit(5).Find(&records).Error; err != nil {
		return &toolContext{ToolName: "query_archive_history", Summary: "归档历史查询失败，当前无法读取归档记录。", Error: "archive history query failed"}
	}
	if len(records) == 0 {
		return &toolContext{ToolName: "query_archive_history", Summary: "当前没有匹配的归档历史记录。"}
	}

	conclusion := summarizeArchiveRecords(message, records)
	lines := make([]string, 0, len(records))
	cards := make([]ResultCard, 0, len(records))
	for _, record := range records {
		line := fmt.Sprintf("- 应用：%s，环境：%s，状态：%s，时间：%s", record.AppName, record.EnvName, record.Status, record.CreatedAt.Format("2006-01-02 15:04"))
		if record.Operator != "" {
			line += "，操作人：" + record.Operator
		}
		if record.ErrorMessage != "" && record.Status == "failed" {
			line += "，错误：" + shorten(record.ErrorMessage, 60)
		}
		lines = append(lines, line)
		cards = append(cards, ResultCard{
			Title:      record.AppName,
			Subtitle:   fmt.Sprintf("环境：%s | 状态：%s", record.EnvName, record.Status),
			Meta:       fmt.Sprintf("%s | 操作人：%s", record.CreatedAt.Format("2006-01-02 15:04"), defaultText(record.Operator, "未知")),
			ToolName:   "deploy.archive_history",
			SourceType: "deploy",
		})
	}

	summary := conclusion + "\n最近记录：\n" + strings.Join(lines, "\n")
	if containsAny(msg, "下载", "产物", "文件", "链接") {
		summary += "\n建议打开 /deploy/archived，在目标记录的下载地址中点击“查看文件”，再点击“下载”获取归档包，或先复制链接后在新窗口打开下载。"
	}
	return &toolContext{ToolName: "query_archive_history", Summary: summary, Cards: cards}
}

func (s *Service) queryAggregatedHistory(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&models.AggregatedHistory{})

	if projectName := pageFilterValue(pageContext, "projectName"); projectName != "" {
		query = query.Where("project_name LIKE ?", "%"+projectName+"%")
	}
	if environment := pageFilterValue(pageContext, "environment"); environment != "" {
		query = query.Where("environment LIKE ?", "%"+environment+"%")
	}
	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query = applyCreatedAtRange(query, pageContext)

	if containsAny(msg, "失败", "failed") {
		query = query.Where("status = ?", "failed")
	} else if containsAny(msg, "成功", "success", "正常完成", "完成") {
		query = query.Where("status IN ?", []string{"success", "completed"})
	} else if containsAny(msg, "运行", "处理中", "running", "pending", "排队") {
		query = query.Where("status IN ?", []string{"pending", "running", "queued", "archiving"})
	}

	if keyword := aggregateQueryKeyword(message); keyword != "" {
		query = query.Where(
			"project_name LIKE ? OR environment LIKE ? OR operator LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var records []models.AggregatedHistory
	if err := query.Order("created_at desc").Limit(5).Find(&records).Error; err != nil {
		return &toolContext{ToolName: "query_aggregate_history", Summary: "聚合历史查询失败，当前无法读取安装包聚合记录。", Error: "aggregate history query failed"}
	}
	if len(records) == 0 {
		return &toolContext{ToolName: "query_aggregate_history", Summary: "当前没有匹配的聚合历史记录。"}
	}

	conclusion := summarizeAggregatedHistory(message, records)
	lines := make([]string, 0, len(records))
	cards := make([]ResultCard, 0, len(records))
	for _, record := range records {
		line := fmt.Sprintf("- 项目：%s，标签：%s，状态：%s，时间：%s", record.ProjectName, record.Environment, record.Status, record.CreatedAt.Format("2006-01-02 15:04"))
		if record.ErrorMessage != nil && record.Status == "failed" && strings.TrimSpace(*record.ErrorMessage) != "" {
			line += "，错误：" + shorten(*record.ErrorMessage, 60)
		}
		lines = append(lines, line)
		cards = append(cards, ResultCard{
			Title:      record.ProjectName,
			Subtitle:   fmt.Sprintf("标签：%s | 状态：%s", record.Environment, record.Status),
			Meta:       record.CreatedAt.Format("2006-01-02 15:04"),
			ToolName:   "deploy.aggregated_history",
			SourceType: "deploy",
		})
	}

	summary := conclusion + "\n最近记录：\n" + strings.Join(lines, "\n")
	if containsAny(msg, "下载", "产物", "文件", "链接", "聚合包") {
		summary += "\n建议打开 /deploy/aggregated-history，从目标记录的下载地址直接点击下载链接获取聚合包。"
	}
	return &toolContext{ToolName: "query_aggregate_history", Summary: summary, Cards: cards}
}

func buildFocusedDeployRecordContext(message string, record cmdb.DeployRecord) *toolContext {
	summaryLines := []string{
		fmt.Sprintf("当前部署记录：应用 %s，环境 %s，状态 %s，触发时间 %s。", record.AppName, record.EnvName, record.Status, record.CreatedAt.Format("2006-01-02 15:04")),
	}

	if strings.TrimSpace(record.TriggeredBy) != "" {
		summaryLines = append(summaryLines, "操作人："+record.TriggeredBy)
	}
	if strings.TrimSpace(record.ErrorMessage) != "" && record.Status == "failed" {
		summaryLines = append(summaryLines, "失败原因："+shorten(record.ErrorMessage, 120))
	}
	return &toolContext{
		ToolName: "query_release_history",
		Summary:  strings.Join(summaryLines, "\n"),
		Cards: []ResultCard{{
			Title:      record.AppName,
			Subtitle:   fmt.Sprintf("环境：%s | 状态：%s", record.EnvName, record.Status),
			Meta:       record.CreatedAt.Format("2006-01-02 15:04"),
			ToolName:   "deploy.records",
			SourceType: "deploy",
		}},
	}
}

func buildFocusedArchiveRecordContext(message string, record cmdb.ArchiveRecord) *toolContext {
	summaryLines := []string{
		fmt.Sprintf("当前归档记录：应用 %s，环境 %s，状态 %s，归档时间 %s。", record.AppName, record.EnvName, record.Status, record.CreatedAt.Format("2006-01-02 15:04")),
	}

	if strings.TrimSpace(record.Operator) != "" {
		summaryLines = append(summaryLines, "操作人："+record.Operator)
	}
	if strings.TrimSpace(record.ErrorMessage) != "" && record.Status == "failed" {
		summaryLines = append(summaryLines, "失败原因："+shorten(record.ErrorMessage, 120))
	}
	if strings.TrimSpace(record.DownloadURL) != "" {
		summaryLines = append(summaryLines, "下载地址："+record.DownloadURL)
		if containsAny(strings.ToLower(message), "下载", "链接", "文件") {
			summaryLines = append(summaryLines, "可以先点“查看文件”确认归档文件，再直接下载或复制链接后在新窗口打开。")
		}
	}

	return &toolContext{
		ToolName: "query_archive_history",
		Summary:  strings.Join(summaryLines, "\n"),
		Cards: []ResultCard{{
			Title:      record.AppName,
			Subtitle:   fmt.Sprintf("环境：%s | 状态：%s", record.EnvName, record.Status),
			Meta:       record.CreatedAt.Format("2006-01-02 15:04"),
			ToolName:   "deploy.archive_history",
			SourceType: "deploy",
		}},
	}
}

func shorten(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len([]rune(text)) <= max {
		return text
	}
	runes := []rune(text)
	return string(runes[:max]) + "..."
}

func summarizeDeployRecords(message string, records []cmdb.DeployRecord) string {
	msg := strings.ToLower(message)
	failed := 0
	running := 0
	statusCounts := map[string]int{}
	appCounts := map[string]int{}
	failedAppCounts := map[string]int{}
	for _, record := range records {
		statusCounts[record.Status]++
		appCounts[record.AppName]++
		if record.Status == "failed" {
			failed++
			failedAppCounts[record.AppName]++
		}
		if record.Status == "running" || record.Status == "queued" || record.Status == "pending" {
			running++
		}
	}

	switch {
	case containsAny(msg, "失败"):
		if failed == 0 {
			return "结论：最近查询到的部署记录里没有失败项。"
		}
		return fmt.Sprintf("结论：最近有 %d 条失败部署，优先关注最上面的失败记录。", failed)
	case containsAny(msg, "执行中", "运行中", "排队"):
		if running == 0 {
			return "结论：当前最近记录里没有执行中或排队中的部署。"
		}
		return fmt.Sprintf("结论：当前有 %d 条部署仍在执行中或排队中。", running)
	case containsAny(msg, "异常集中", "集中", "哪些应用"):
		failedApp, failedCount := topCount(failedAppCounts)
		if failedApp != "" && failedCount > 0 {
			if failedCount == 1 {
				return fmt.Sprintf("结论：最近失败部署分散出现，当前排在前面的是应用 %s。", failedApp)
			}
			return fmt.Sprintf("结论：最近部署异常更集中在应用 %s，最近 5 条里有 %d 条失败记录。", failedApp, failedCount)
		}

		app, count := topCount(appCounts)
		if app == "" || count <= 1 {
			return "结论：最近部署记录分布较分散，暂未识别到明显集中的应用。"
		}
		return fmt.Sprintf("结论：最近没有明显失败集中，记录更集中在应用 %s，最近 5 条里出现了 %d 次。", app, count)
	default:
		if failed > 0 {
			return fmt.Sprintf("结论：最近部署记录里有 %d 条失败，建议优先排查失败项。", failed)
		}
		if running > 0 {
			return fmt.Sprintf("结论：最近部署整体可控，但还有 %d 条任务在执行中或排队中。", running)
		}
		return fmt.Sprintf("结论：最近部署记录以 %s 为主。", dominantStatusLabel(statusCounts))
	}
}

func summarizeArchiveRecords(message string, records []cmdb.ArchiveRecord) string {
	msg := strings.ToLower(message)
	failed := 0
	inProgress := 0
	success := 0
	appCounts := map[string]int{}
	failedAppCounts := map[string]int{}
	inProgressAppCounts := map[string]int{}
	for _, record := range records {
		appCounts[record.AppName]++
		switch record.Status {
		case "failed":
			failed++
			failedAppCounts[record.AppName]++
		case "success":
			success++
		case "running", "queued", "pending":
			inProgress++
			inProgressAppCounts[record.AppName]++
		}
	}

	switch {
	case containsAny(msg, "失败"):
		if failed == 0 {
			return "结论：最近查询到的归档记录里没有失败项。"
		}
		return fmt.Sprintf("结论：最近有 %d 条归档失败；%s建议优先查看失败记录的错误信息。", failed, summarizeTopApps(failedAppCounts, "失败主要集中在"))
	case containsAny(msg, "正常完成", "是否正常", "成功"):
		if failed == 0 && inProgress == 0 {
			app, count := topCount(appCounts)
			if app != "" && count > 0 {
				return fmt.Sprintf("结论：最近归档记录都已正常完成，共 %d 条成功记录；最近记录主要集中在应用 %s。", success, app)
			}
			return fmt.Sprintf("结论：最近归档记录都已正常完成，共 %d 条成功记录。", success)
		}
		if failed > 0 {
			app, count := topCount(failedAppCounts)
			if app != "" && count > 0 {
				return fmt.Sprintf("结论：最近归档并非全部正常完成，存在 %d 条失败记录，优先关注应用 %s。", failed, app)
			}
			return fmt.Sprintf("结论：最近归档并非全部正常完成，存在 %d 条失败记录。", failed)
		}
		app, count := topCount(inProgressAppCounts)
		if app != "" && count > 0 {
			return fmt.Sprintf("结论：最近归档仍有 %d 条任务在处理中，当前主要集中在应用 %s，建议稍后再确认最终状态。", inProgress, app)
		}
		return fmt.Sprintf("结论：最近归档仍有 %d 条任务在处理中，建议稍后再确认最终状态。", inProgress)
	case containsAny(msg, "下载", "产物", "文件", "链接"):
		return "结论：归档产物需要从归档历史页面的下载地址进入，先点“查看文件”，再下载归档包，或复制链接后在新窗口打开下载。"
	default:
		app, count := topCount(appCounts)
		if app != "" && count > 1 {
			return fmt.Sprintf("结论：最近归档记录更集中在应用 %s，最近 5 条里出现了 %d 次。", app, count)
		}
		if failed > 0 {
			return fmt.Sprintf("结论：最近归档记录里有 %d 条失败，建议优先排查失败项。", failed)
		}
		return "结论：最近归档记录整体稳定，可继续查看具体时间、应用和状态。"
	}
}

func topCount(counts map[string]int) (string, int) {
	var bestKey string
	bestCount := 0
	for key, count := range counts {
		if count > bestCount {
			bestKey = key
			bestCount = count
		}
	}
	return bestKey, bestCount
}

func dominantStatusLabel(counts map[string]int) string {
	status, count := topCount(counts)
	if status == "" || count == 0 {
		return "空结果"
	}
	switch status {
	case "success":
		return "成功"
	case "failed":
		return "失败"
	case "running":
		return "执行中"
	case "queued":
		return "排队中"
	case "pending":
		return "待执行"
	default:
		return status
	}
}

func summarizeTopApps(counts map[string]int, prefix string) string {
	type pair struct {
		app   string
		count int
	}
	items := make([]pair, 0, len(counts))
	for app, count := range counts {
		if strings.TrimSpace(app) == "" || count <= 0 {
			continue
		}
		items = append(items, pair{app: app, count: count})
	}
	if len(items) == 0 {
		return ""
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].app < items[j].app
		}
		return items[i].count > items[j].count
	})
	if len(items) == 1 {
		return fmt.Sprintf("%s %s（%d 条）。", prefix, items[0].app, items[0].count)
	}
	return fmt.Sprintf("%s %s（%d 条）、%s（%d 条）。", prefix, items[0].app, items[0].count, items[1].app, items[1].count)
}

func archiveQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近有哪些归档失败", "最近归档失败有哪些", "最近有哪些失败归档", "查看最近归档失败") {
		return ""
	}

	return extractPrimaryKeyword(message,
		"归档", "历史", "记录", "最近", "失败", "成功", "状态", "查看",
		"有哪些", "哪些", "最近有哪些", "最近归档失败有哪些", "一下", "看看", "帮我", "帮忙", "查询",
	)
}

func aggregateQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近有哪些聚合失败", "最近聚合失败有哪些", "最近有哪些失败聚合", "查看最近聚合失败") {
		return ""
	}

	return extractPrimaryKeyword(message,
		"聚合", "打包", "聚合历史", "历史", "记录", "最近", "失败", "成功", "状态", "查看",
		"有哪些", "哪些", "最近有哪些", "一下", "看看", "帮我", "帮忙", "查询",
	)
}

func summarizeAggregatedHistory(message string, records []models.AggregatedHistory) string {
	msg := strings.ToLower(message)
	failed := 0
	inProgress := 0
	success := 0
	projectCounts := map[string]int{}
	failedProjectCounts := map[string]int{}
	inProgressProjectCounts := map[string]int{}
	for _, record := range records {
		projectCounts[record.ProjectName]++
		switch record.Status {
		case "failed":
			failed++
			failedProjectCounts[record.ProjectName]++
		case "success", "completed":
			success++
		case "running", "queued", "pending", "archiving":
			inProgress++
			inProgressProjectCounts[record.ProjectName]++
		}
	}

	switch {
	case containsAny(msg, "失败"):
		if failed == 0 {
			return "结论：最近查询到的聚合历史里没有失败项。"
		}
		return fmt.Sprintf("结论：最近有 %d 条聚合失败；%s建议优先查看失败记录的错误信息。", failed, summarizeTopApps(failedProjectCounts, "失败主要集中在"))
	case containsAny(msg, "正常完成", "是否正常", "成功", "完成"):
		if failed == 0 && inProgress == 0 {
			project, count := topCount(projectCounts)
			if project != "" && count > 0 {
				return fmt.Sprintf("结论：最近聚合记录都已正常完成，共 %d 条成功记录；最近记录主要集中在项目 %s。", success, project)
			}
			return fmt.Sprintf("结论：最近聚合记录都已正常完成，共 %d 条成功记录。", success)
		}
		if failed > 0 {
			project, count := topCount(failedProjectCounts)
			if project != "" && count > 0 {
				return fmt.Sprintf("结论：最近聚合并非全部正常完成，存在 %d 条失败记录，优先关注项目 %s。", failed, project)
			}
			return fmt.Sprintf("结论：最近聚合并非全部正常完成，存在 %d 条失败记录。", failed)
		}
		project, count := topCount(inProgressProjectCounts)
		if project != "" && count > 0 {
			return fmt.Sprintf("结论：最近聚合仍有 %d 条任务在处理中，当前主要集中在项目 %s，建议稍后再确认最终状态。", inProgress, project)
		}
		return fmt.Sprintf("结论：最近聚合仍有 %d 条任务在处理中，建议稍后再确认最终状态。", inProgress)
	case containsAny(msg, "下载", "产物", "文件", "链接", "聚合包"):
		return "结论：聚合包需要从聚合历史页面进入，在下载地址中直接点击下载链接下载。"
	default:
		project, count := topCount(projectCounts)
		if project != "" && count > 1 {
			return fmt.Sprintf("结论：最近聚合记录更集中在项目 %s，最近 5 条里出现了 %d 次。", project, count)
		}
		if failed > 0 {
			return fmt.Sprintf("结论：最近聚合记录里有 %d 条失败，建议优先排查失败项。", failed)
		}
		return "结论：最近聚合记录整体稳定，可继续查看项目、标签和状态。"
	}
}