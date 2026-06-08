package assistant

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/cmdb"
	"github.com/edy/ops-platform/internal/database"
)

type applicationCountStat struct {
	AppName string
	Count   int
}

func (s *Service) queryProjects(message string, pageContext *AssistantPageContext) *toolContext {
	var projects []cmdb.Project
	query := database.DB.Model(&cmdb.Project{}).Where("deleted_at IS NULL")
	if projectName := pageFilterValue(pageContext, "projectName"); projectName != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+projectName+"%", "%"+projectName+"%")
	}
	if keyword := extractPrimaryKeyword(message, "项目", "project", "管理"); keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Order("id desc").Limit(5).Find(&projects).Error; err != nil {
		return &toolContext{ToolName: "query_projects", Summary: "项目查询失败，当前无法读取 CMDB 项目数据。", Error: "cmdb projects query failed"}
	}
	if len(projects) == 0 {
		return &toolContext{ToolName: "query_projects", Summary: "当前没有匹配的项目记录。"}
	}

	lines := make([]string, 0, len(projects))
	cards := make([]ResultCard, 0, len(projects))
	for _, project := range projects {
		lines = append(lines, fmt.Sprintf("- 项目：%s，编号：%s", project.Name, project.Code))
		cards = append(cards, ResultCard{
			Title:      project.Name,
			Subtitle:   "项目编号：" + project.Code,
			Meta:       shorten(project.Description, 60),
			ToolName:   "cmdb.projects",
			SourceType: "cmdb",
		})
	}
	return &toolContext{ToolName: "query_projects", Summary: "查询到的项目如下：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) queryEnvironments(message string, pageContext *AssistantPageContext) *toolContext {
	var environments []cmdb.Environment
	query := database.DB.Model(&cmdb.Environment{}).Where("deleted_at IS NULL")
	if envName := pageFilterValue(pageContext, "envName"); envName != "" {
		query = query.Where("name LIKE ?", "%"+envName+"%")
	}
	if keyword := extractPrimaryKeyword(message, "环境", "environment", "管理"); keyword != "" {
		query = query.Where("name LIKE ? OR type LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Order("id desc").Limit(5).Find(&environments).Error; err != nil {
		return &toolContext{ToolName: "query_environments", Summary: "环境查询失败，当前无法读取 CMDB 环境数据。", Error: "cmdb environments query failed"}
	}
	if len(environments) == 0 {
		return &toolContext{ToolName: "query_environments", Summary: "当前没有匹配的环境记录。"}
	}

	lines := make([]string, 0, len(environments))
	cards := make([]ResultCard, 0, len(environments))
	for _, environment := range environments {
		lines = append(lines, fmt.Sprintf("- 环境：%s，类型：%s", environment.Name, environment.Type))
		cards = append(cards, ResultCard{
			Title:      environment.Name,
			Subtitle:   "环境类型：" + environment.Type,
			Meta:       shorten(environment.Description, 60),
			ToolName:   "cmdb.environments",
			SourceType: "cmdb",
		})
	}
	return &toolContext{ToolName: "query_environments", Summary: "查询到的环境如下：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) queryServers(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	if containsAny(msg, "当前主机分布情况", "主机分布情况", "当前主机情况") {
		return s.queryCurrentServerOverview(pageContext)
	}
	if containsAny(msg, "删除") && containsAny(msg, "异常", "离线") {
		return s.queryDeletedAbnormalServers(message, pageContext)
	}

	var servers []cmdb.Server
	query := database.DB.Model(&cmdb.Server{}).Where("deleted_at IS NULL")
	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if containsAny(msg, "离线", "异常") {
		query = query.Where("status = ?", "offline")
	} else if containsAny(msg, "在线") {
		query = query.Where("status = ?", "online")
	}
	if keyword := serverQueryKeyword(message); keyword != "" {
		query = query.Where("hostname LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Preload("Projects").Order("id desc").Limit(5).Find(&servers).Error; err != nil {
		return &toolContext{ToolName: "query_servers", Summary: "主机查询失败，当前无法读取 CMDB 主机数据。", Error: "cmdb servers query failed"}
	}
	if len(servers) == 0 {
		return &toolContext{ToolName: "query_servers", Summary: "当前没有匹配的主机记录。"}
	}

	conclusion := summarizeServers(message, servers)
	lines := make([]string, 0, len(servers))
	cards := make([]ResultCard, 0, len(servers))
	for _, server := range servers {
		projectNames := make([]string, 0, len(server.Projects))
		for _, project := range server.Projects {
			projectNames = append(projectNames, project.Name)
		}
		projectText := "未绑定项目"
		if len(projectNames) > 0 {
			projectText = strings.Join(projectNames, "、")
		}
		lines = append(lines, fmt.Sprintf("- 主机：%s，IP：%s，状态：%s，项目：%s", server.Hostname, server.IP, server.Status, projectText))
		cards = append(cards, ResultCard{
			Title:      server.Hostname,
			Subtitle:   "IP：" + server.IP,
			Meta:       fmt.Sprintf("状态：%s | 项目：%s", server.Status, projectText),
			ToolName:   "cmdb.servers",
			SourceType: "cmdb",
		})
	}
	return &toolContext{ToolName: "query_servers", Summary: conclusion + "\n最近记录：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) queryCurrentServerOverview(pageContext *AssistantPageContext) *toolContext {
	servers, err := listCurrentServers(pageContext)
	if err != nil {
		return &toolContext{ToolName: "query_servers", Summary: "主机查询失败，当前无法读取 CMDB 主机数据。", Error: "cmdb current servers query failed"}
	}
	if len(servers) == 0 {
		return &toolContext{ToolName: "query_servers", Summary: "当前没有主机记录。"}
	}

	stats := buildServerInventoryStats(servers)
	summary := summarizeCurrentServerOverview(stats)
	lines := make([]string, 0, minInt(len(servers), 5))
	cards := make([]ResultCard, 0, minInt(len(servers), 5))
	for _, server := range servers[:minInt(len(servers), 5)] {
		osName := fallbackLabel(server.OS, "未标注系统")
		arch := fallbackLabel(server.Arch, "未标注架构")
		lines = append(lines, fmt.Sprintf("- 主机：%s，IP：%s，状态：%s，系统：%s，架构：%s", server.Hostname, server.IP, server.Status, osName, arch))
		cards = append(cards, ResultCard{
			Title:      server.Hostname,
			Subtitle:   "IP：" + server.IP,
			Meta:       fmt.Sprintf("状态：%s | %s | %s", server.Status, osName, arch),
			ToolName:   "cmdb.servers",
			SourceType: "cmdb",
		})
	}
	return &toolContext{ToolName: "query_servers", Summary: summary + "\n最近记录：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) queryDeletedAbnormalServers(message string, pageContext *AssistantPageContext) *toolContext {
	activeServers, err := listCurrentServers(pageContext)
	if err != nil {
		return &toolContext{ToolName: "query_servers", Summary: "主机查询失败，当前无法读取 CMDB 主机数据。", Error: "cmdb active servers query failed"}
	}

	var deletedServers []cmdb.Server
	query := database.DB.Unscoped().Model(&cmdb.Server{}).
		Where("deleted_at IS NOT NULL").
		Where("status = ?", "offline")
	if keyword := serverQueryKeyword(message); keyword != "" {
		query = query.Where("hostname LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Order("deleted_at desc").Limit(5).Find(&deletedServers).Error; err != nil {
		return &toolContext{ToolName: "query_servers", Summary: "主机查询失败，当前无法读取异常主机删除记录。", Error: "cmdb deleted servers query failed"}
	}

	stats := buildServerInventoryStats(activeServers)
	summary := summarizeDeletedAbnormalServers(deletedServers, stats)
	if len(deletedServers) == 0 {
		return &toolContext{ToolName: "query_servers", Summary: summary}
	}

	lines := make([]string, 0, len(deletedServers))
	cards := make([]ResultCard, 0, len(deletedServers))
	for _, server := range deletedServers {
		deletedAt := "-"
		if server.DeletedAt != nil {
			deletedAt = server.DeletedAt.Format("2006-01-02 15:04")
		}
		osName := fallbackLabel(server.OS, "未标注系统")
		arch := fallbackLabel(server.Arch, "未标注架构")
		lines = append(lines, fmt.Sprintf("- 主机：%s，IP：%s，删除时间：%s，系统：%s，架构：%s", server.Hostname, server.IP, deletedAt, osName, arch))
		cards = append(cards, ResultCard{
			Title:      server.Hostname,
			Subtitle:   "IP：" + server.IP,
			Meta:       fmt.Sprintf("已删除异常主机 | 删除时间：%s | %s | %s", deletedAt, osName, arch),
			ToolName:   "cmdb.servers",
			SourceType: "cmdb",
		})
	}

	return &toolContext{
		ToolName: "query_servers",
		Summary:  summary + "\n最近删除记录：\n" + strings.Join(lines, "\n"),
		Cards:    cards,
	}
}

func (s *Service) queryApplications(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)

	if containsAny(msg, "缺少", "配置") {
		var apps []cmdb.Application
		query := database.DB.Model(&cmdb.Application{}).Where("deleted_at IS NULL").
			Where("project_id = 0 OR env_id = 0 OR jenkins_job = '' OR jenkins_archive_job = ''")
		if projectName := pageFilterValue(pageContext, "projectName"); projectName != "" {
			query = query.Where("name LIKE ? OR code_repo LIKE ?", "%"+projectName+"%", "%"+projectName+"%")
		}
		if keyword := applicationQueryKeyword(message); keyword != "" {
			query = query.Where("name LIKE ?", "%"+keyword+"%")
		}
		if err := query.Preload("Project").Preload("Environment").Order("id desc").Limit(5).Find(&apps).Error; err != nil {
			return &toolContext{ToolName: "query_applications", Summary: "应用流水线查询失败，当前无法读取应用配置数据。", Error: "cmdb applications query failed"}
		}
		if len(apps) == 0 {
			return &toolContext{ToolName: "query_applications", Summary: "当前没有匹配的应用流水线记录。"}
		}

		lines := make([]string, 0, len(apps))
		cards := make([]ResultCard, 0, len(apps))
		for _, app := range apps {
			missing := make([]string, 0, 4)
			if app.ProjectID == 0 {
				missing = append(missing, "所属项目")
			}
			if app.EnvID == 0 {
				missing = append(missing, "项目环境")
			}
			if strings.TrimSpace(app.JenkinsJob) == "" {
				missing = append(missing, "Jenkins发布流水线")
			}
			if strings.TrimSpace(app.JenkinsArchiveJob) == "" {
				missing = append(missing, "Jenkins归档流水线")
			}
			lines = append(lines, fmt.Sprintf("- 应用：%s，缺少：%s", app.Name, strings.Join(missing, "、")))
			cards = append(cards, ResultCard{
				Title:      app.Name,
				Subtitle:   "配置缺失",
				Meta:       strings.Join(missing, "、"),
				ToolName:   "cmdb.applications",
				SourceType: "cmdb",
			})
		}
		return &toolContext{ToolName: "query_applications", Summary: fmt.Sprintf("结论：当前有 %d 个应用流水线存在关键信息缺失。\n最近记录：\n%s", len(apps), strings.Join(lines, "\n")), Cards: cards}
	}

	query := database.DB.Model(&cmdb.DeployRecord{})
	if appName := pageFilterValue(pageContext, "appName"); appName != "" {
		query = query.Where("app_name LIKE ?", "%"+appName+"%")
	}
	if projectCode := pageFilterValue(pageContext, "projectCode", "projectName"); projectCode != "" {
		query = query.Where("project_code LIKE ?", "%"+projectCode+"%")
	}
	if envID, ok := pageFilterUint(pageContext, "envId"); ok {
		query = query.Where("env_id = ?", envID)
	}
	if containsAny(msg, "失败") {
		query = query.Where("status = ?", "failed")
	}
	if keyword := applicationQueryKeyword(message); keyword != "" {
		query = query.Where("app_name LIKE ? OR project_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var counts []applicationCountStat
	if err := query.Select("app_name, COUNT(*) as count").Group("app_name").Order("count desc").Limit(5).Scan(&counts).Error; err != nil {
		return &toolContext{ToolName: "query_applications", Summary: "应用流水线查询失败，当前无法读取部署统计数据。", Error: "cmdb applications deploy stats query failed"}
	}
	if len(counts) == 0 {
		return &toolContext{ToolName: "query_applications", Summary: "当前没有匹配的应用流水线记录。"}
	}

	lines := make([]string, 0, len(counts))
	cards := make([]ResultCard, 0, len(counts))
	for _, item := range counts {
		label := "最近部署次数"
		if containsAny(msg, "失败") {
			label = "最近失败次数"
		}
		lines = append(lines, fmt.Sprintf("- 应用：%s，%s：%d", item.AppName, label, item.Count))
		cards = append(cards, ResultCard{
			Title:      item.AppName,
			Subtitle:   label,
			Meta:       fmt.Sprintf("%d", item.Count),
			ToolName:   "cmdb.applications",
			SourceType: "cmdb",
		})
	}

	return &toolContext{ToolName: "query_applications", Summary: summarizeApplications(message, counts) + "\n最近记录：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func extractPrimaryKeyword(message string, stopWords ...string) string {
	terms := extractKeywords(message)
	stopSet := make(map[string]struct{}, len(stopWords))
	for _, word := range stopWords {
		stopSet[strings.ToLower(word)] = struct{}{}
	}

	for _, term := range terms {
		if _, exists := stopSet[strings.ToLower(term)]; exists {
			continue
		}
		if len([]rune(term)) >= 2 {
			return term
		}
	}
	return ""
}

func serverQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近有哪些异常主机", "最近有哪些异常主机删除", "当前主机分布情况", "主机分布情况", "当前主机情况", "当前离线主机有哪些", "主机异常主要集中在哪个环境") {
		return ""
	}
	return extractPrimaryKeyword(message, "主机", "服务器", "server", "管理", "最近", "当前", "异常", "离线", "在线", "删除", "环境", "分布", "情况", "哪些", "有哪些")
}

func applicationQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近哪些应用发布最频繁", "哪些应用最近部署失败较多", "哪些应用缺少关键信息配置") {
		return ""
	}
	return extractPrimaryKeyword(message, "应用", "流水线", "最近", "部署", "发布", "失败", "较多", "频繁", "缺少", "关键信息", "配置", "哪些")
}

func summarizeServers(message string, servers []cmdb.Server) string {
	msg := strings.ToLower(message)
	statusCounts := map[string]int{}
	envCounts := map[string]int{}
	for _, server := range servers {
		statusCounts[server.Status]++
		for _, envID := range strings.Split(server.EnvIDs, ",") {
			envID = strings.TrimSpace(envID)
			if envID != "" {
				envCounts[envID]++
			}
		}
	}
	switch {
	case containsAny(msg, "离线"):
		return fmt.Sprintf("结论：当前匹配到 %d 台离线主机。", len(servers))
	case containsAny(msg, "异常"):
		return fmt.Sprintf("结论：当前匹配到 %d 台异常主机，默认按离线主机优先展示。", len(servers))
	case containsAny(msg, "环境", "集中"):
		envID, count := topCountString(envCounts)
		if envID != "" {
			return fmt.Sprintf("结论：主机异常更集中在环境 ID %s，最近返回结果里出现了 %d 次。", envID, count)
		}
		return "结论：最近没有识别到明显集中的异常环境。"
	default:
		return fmt.Sprintf("结论：当前匹配到 %d 台主机。", len(servers))
	}
}

type serverInventoryStats struct {
	Total      int
	StatusCounts map[string]int
	OSCounts   map[string]int
	ArchCounts map[string]int
}

func listCurrentServers(pageContext *AssistantPageContext) ([]cmdb.Server, error) {
	var servers []cmdb.Server
	query := database.DB.Model(&cmdb.Server{}).Where("deleted_at IS NULL")
	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("id desc").Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

func buildServerInventoryStats(servers []cmdb.Server) serverInventoryStats {
	stats := serverInventoryStats{
		Total:        len(servers),
		StatusCounts: make(map[string]int),
		OSCounts:     make(map[string]int),
		ArchCounts:   make(map[string]int),
	}
	for _, server := range servers {
		stats.StatusCounts[fallbackLabel(server.Status, "unknown")]++
		stats.OSCounts[fallbackLabel(server.OS, "未标注系统")]++
		stats.ArchCounts[fallbackLabel(server.Arch, "未标注架构")]++
	}
	return stats
}

func summarizeCurrentServerOverview(stats serverInventoryStats) string {
	return fmt.Sprintf(
		"当前主机分布情况如下，当前共有 %d 台主机；状态分布：%s；系统分布：%s；架构分布：%s。",
		stats.Total,
		formatServerDistribution(stats.StatusCounts),
		formatServerDistribution(stats.OSCounts),
		formatServerDistribution(stats.ArchCounts),
	)
}

func summarizeDeletedAbnormalServers(servers []cmdb.Server, stats serverInventoryStats) string {
	base := fmt.Sprintf(
		"结论：最近删除的异常主机有 %d 台。当前共有 %d 台主机；系统分布：%s；架构分布：%s。",
		len(servers),
		stats.Total,
		formatServerDistribution(stats.OSCounts),
		formatServerDistribution(stats.ArchCounts),
	)
	if len(servers) == 0 {
		return "结论：最近没有发现异常主机删除记录。" + strings.TrimPrefix(base, "结论：最近删除的异常主机有 0 台。")
	}
	latestDeletedAt := latestServerDeletedAt(servers)
	if latestDeletedAt.IsZero() {
		return base
	}
	return fmt.Sprintf("%s 最近一次删除时间：%s。", base, latestDeletedAt.Format("2006-01-02 15:04"))
}

func latestServerDeletedAt(servers []cmdb.Server) time.Time {
	var latest time.Time
	for _, server := range servers {
		if server.DeletedAt != nil && server.DeletedAt.After(latest) {
			latest = *server.DeletedAt
		}
	}
	return latest
}

func formatServerDistribution(counts map[string]int) string {
	if len(counts) == 0 {
		return "暂无数据"
	}

	type item struct {
		Label string
		Count int
	}
	items := make([]item, 0, len(counts))
	for label, count := range counts {
		items = append(items, item{Label: label, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Label < items[j].Label
		}
		return items[i].Count > items[j].Count
	})

	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %d 台", item.Label, item.Count))
	}
	return strings.Join(parts, "，")
}

func fallbackLabel(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func summarizeApplications(message string, counts []applicationCountStat) string {
	msg := strings.ToLower(message)
	if len(counts) == 0 {
		return "结论：当前没有匹配的应用流水线记录。"
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count == counts[j].Count {
			return counts[i].AppName < counts[j].AppName
		}
		return counts[i].Count > counts[j].Count
	})
	top := counts[0]
	switch {
	case containsAny(msg, "发布最频繁"):
		return fmt.Sprintf("结论：最近发布最频繁的是应用 %s，共 %d 次。", top.AppName, top.Count)
	case containsAny(msg, "失败较多", "部署失败"):
		return fmt.Sprintf("结论：最近部署失败较多的是应用 %s，共 %d 次。", top.AppName, top.Count)
	default:
		return fmt.Sprintf("结论：最近部署记录更集中在应用 %s，共 %d 次。", top.AppName, top.Count)
	}
}
