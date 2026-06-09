package assistant

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

func (s *Service) querySecurityVulnerabilities(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&models.SecurityVulnerability{})

	if shouldUseFocusedObjectQuery(message, pageContext, "security_vulnerability") {
		if vulnerabilityID, ok := pageObjectID(pageContext); ok {
			var vulnerability models.SecurityVulnerability
			if err := database.DB.Where("id = ?", vulnerabilityID).First(&vulnerability).Error; err == nil {
				return buildFocusedSecurityVulnerabilityContext(vulnerability)
			}
		}
	}

	if severity := pageFilterValue(pageContext, "severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	switch pageFilterValue(pageContext, "riskCategory") {
	case "资产识别":
		query = query.Where("scan_method = ? OR vuln_type LIKE ?", "服务识别", "%资产识别%")
	case "CVE 风险":
		query = query.Where("cve_id <> ''")
	case "配置风险":
		query = query.Where("vuln_type LIKE ?", "%配置%")
	}

	switch {
	case containsAny(msg, "严重", "critical"):
		query = query.Where("severity = ?", "critical")
	case containsAny(msg, "高危", "high"):
		query = query.Where("severity = ?", "high")
	case containsAny(msg, "中危", "medium"):
		query = query.Where("severity = ?", "medium")
	case containsAny(msg, "低危", "low"):
		query = query.Where("severity = ?", "low")
	}

	switch {
	case containsAny(msg, "处理中", "已确认", "acknowledged"):
		query = query.Where("status = ?", "acknowledged")
	case containsAny(msg, "已修复", "fixed"):
		query = query.Where("status = ?", "fixed")
	case containsAny(msg, "忽略", "ignored"):
		query = query.Where("status = ?", "ignored")
	case containsAny(msg, "待处理", "未处理", "open"):
		query = query.Where("status = ?", "open")
	}

	if keyword := vulnerabilityQueryKeyword(message); keyword != "" {
		query = query.Where(
			"title LIKE ? OR cve_id LIKE ? OR ip LIKE ? OR vuln_type LIKE ? OR status LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var total int64
	_ = query.Count(&total).Error
	var vulns []models.SecurityVulnerability
	if err := query.Order("created_at desc").Limit(5).Find(&vulns).Error; err != nil {
		return &toolContext{ToolName: "query_vulnerability_summary", Summary: "漏洞查询失败，当前无法读取安全模块漏洞数据。", Error: "security vulnerabilities query failed"}
	}
	if len(vulns) == 0 {
		return &toolContext{ToolName: "query_vulnerability_summary", Summary: "当前没有匹配的漏洞记录。"}
	}

	conclusion := summarizeSecurityVulnerabilities(message, vulns, total)
	lines := make([]string, 0, len(vulns))
	cards := make([]ResultCard, 0, len(vulns))
	for _, vuln := range vulns {
		identifier := defaultText(vuln.CVEID, defaultText(vuln.TemplateID, "无编号"))
		target := defaultText(vuln.IP, "未知目标")
		if vuln.Port > 0 {
			target = fmt.Sprintf("%s:%d", target, vuln.Port)
		}
		lines = append(lines, fmt.Sprintf("- 漏洞：%s，级别：%s，状态：%s，目标：%s，编号：%s", defaultText(vuln.Title, "未命名漏洞"), vuln.Severity, vuln.Status, target, identifier))
		cards = append(cards, ResultCard{
			Title:      defaultText(vuln.Title, "未命名漏洞"),
			Subtitle:   fmt.Sprintf("级别：%s | 状态：%s", vuln.Severity, vuln.Status),
			Meta:       fmt.Sprintf("目标：%s | 编号：%s", target, identifier),
			ToolName:   "security.vulnerabilities",
			SourceType: "security",
		})
	}

	return &toolContext{ToolName: "query_vulnerability_summary", Summary: conclusion + "\n最近记录：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) querySecurityScanTasks(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&models.SecurityScanTask{})

	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}

	switch {
	case containsAny(msg, "运行", "执行中", "running"):
		query = query.Where("status = ?", models.TaskStatusRunning)
	case containsAny(msg, "失败", "failed"):
		query = query.Where("status = ?", models.TaskStatusFailed)
	case containsAny(msg, "完成", "completed"):
		query = query.Where("status = ?", models.TaskStatusCompleted)
	case containsAny(msg, "待执行", "pending"):
		query = query.Where("status = ?", models.TaskStatusPending)
	}

	switch {
	case containsAny(msg, "web", "网页", "站点"):
		query = query.Where("scan_type = ?", string(models.ScanTypeWeb))
	case containsAny(msg, "主机", "host"):
		query = query.Where("scan_type = ?", string(models.ScanTypeHostVuln))
	case containsAny(msg, "端口", "port"):
		query = query.Where("scan_type = ?", string(models.ScanTypePort))
	}

	if keyword := securityTaskQueryKeyword(message); keyword != "" {
		query = query.Where("name LIKE ? OR target LIKE ? OR message LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var tasks []models.SecurityScanTask
	if err := query.Order("created_at desc").Limit(5).Find(&tasks).Error; err != nil {
		return &toolContext{ToolName: "query_security_scan_tasks", Summary: "扫描任务查询失败，当前无法读取安全扫描任务数据。", Error: "security scan tasks query failed"}
	}
	if len(tasks) == 0 {
		return &toolContext{ToolName: "query_security_scan_tasks", Summary: "当前没有匹配的扫描任务记录。"}
	}

	conclusion := summarizeSecurityScanTasks(message, tasks)
	lines := make([]string, 0, len(tasks))
	cards := make([]ResultCard, 0, len(tasks))
	for _, task := range tasks {
		lines = append(lines, fmt.Sprintf("- 任务：%s，类型：%s，状态：%s，进度：%d%%，目标：%s", task.Name, task.ScanType, task.Status, task.Progress, defaultText(task.Target, "未设置目标")))
		cards = append(cards, ResultCard{
			Title:      task.Name,
			Subtitle:   fmt.Sprintf("类型：%s | 状态：%s", task.ScanType, task.Status),
			Meta:       fmt.Sprintf("进度：%d%% | 目标：%s", task.Progress, defaultText(task.Target, "未设置目标")),
			ToolName:   "security.tasks",
			SourceType: "security",
		})
	}

	return &toolContext{ToolName: "query_security_scan_tasks", Summary: conclusion + "\n最近记录：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) querySecurityAssets(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&models.Asset{})

	switch {
	case containsAny(msg, "在线", "online"):
		query = query.Where("status = ?", models.AssetStatusOnline)
	case containsAny(msg, "离线", "offline"):
		query = query.Where("status = ?", models.AssetStatusOffline)
	case containsAny(msg, "未知", "unknown"):
		query = query.Where("status = ?", models.AssetStatusUnknown)
	}

	switch {
	case containsAny(msg, "web", "网页", "站点"):
		query = query.Where("asset_type = ?", models.AssetTypeWeb)
	case containsAny(msg, "数据库", "db"):
		query = query.Where("asset_type = ?", models.AssetTypeDatabase)
	case containsAny(msg, "服务器", "主机", "server"):
		query = query.Where("asset_type = ?", models.AssetTypeServer)
	case containsAny(msg, "网络", "network"):
		query = query.Where("asset_type = ?", models.AssetTypeNetwork)
	}

	if keyword := extractPrimaryKeyword(message, "资产", "安全", "最近", "状态", "在线", "离线", "web", "网页", "站点", "数据库", "服务器", "主机", "网络", "列表"); keyword != "" {
		query = query.Where(
			"ip LIKE ? OR service_name LIKE ? OR owner LIKE ? OR department LIKE ? OR tags LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var assets []models.Asset
	if err := query.Order("updated_at desc").Limit(5).Find(&assets).Error; err != nil {
		return &toolContext{ToolName: "query_security_assets", Summary: "安全资产查询失败，当前无法读取安全资产数据。", Error: "security assets query failed"}
	}
	if len(assets) == 0 {
		return &toolContext{ToolName: "query_security_assets", Summary: "当前没有匹配的安全资产记录。"}
	}

	lines := make([]string, 0, len(assets))
	cards := make([]ResultCard, 0, len(assets))
	for _, asset := range assets {
		target := asset.IP
		if asset.Port > 0 {
			target = fmt.Sprintf("%s:%d", asset.IP, asset.Port)
		}
		lines = append(lines, fmt.Sprintf("- 资产：%s，类型：%s，状态：%s，服务：%s，负责人：%s", target, asset.AssetType, asset.Status, defaultText(asset.ServiceName, "未知"), defaultText(asset.Owner, "未设置")))
		cards = append(cards, ResultCard{
			Title:      target,
			Subtitle:   fmt.Sprintf("类型：%s | 状态：%s", asset.AssetType, asset.Status),
			Meta:       fmt.Sprintf("服务：%s | 负责人：%s", defaultText(asset.ServiceName, "未知"), defaultText(asset.Owner, "未设置")),
			ToolName:   "security.assets",
			SourceType: "security",
		})
	}

	return &toolContext{ToolName: "query_security_assets", Summary: "查询到的安全资产如下：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func (s *Service) querySecurityTickets(message string, pageContext *AssistantPageContext) *toolContext {
	msg := strings.ToLower(message)
	query := database.DB.Model(&models.VulnTicket{})

	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}

	switch {
	case containsAny(msg, "处理中", "processing"):
		query = query.Where("status = ?", models.VulnTicketStatusProcessing)
	case containsAny(msg, "已修复", "fixed"):
		query = query.Where("status = ?", models.VulnTicketStatusFixed)
	case containsAny(msg, "已关闭", "closed"):
		query = query.Where("status = ?", models.VulnTicketStatusClosed)
	case containsAny(msg, "驳回", "rejected"):
		query = query.Where("status = ?", models.VulnTicketStatusRejected)
	case containsAny(msg, "待处理", "未处理", "open"):
		query = query.Where("status = ?", models.VulnTicketStatusOpen)
	}

	switch {
	case containsAny(msg, "高优先级", "高优先", "高"):
		query = query.Where("priority = ?", models.VulnTicketPriorityHigh)
	case containsAny(msg, "中优先级", "中优先", "中"):
		query = query.Where("priority = ?", models.VulnTicketPriorityMedium)
	case containsAny(msg, "低优先级", "低优先", "低"):
		query = query.Where("priority = ?", models.VulnTicketPriorityLow)
	}

	if keyword := extractPrimaryKeyword(message, "工单", "漏洞", "最近", "状态", "处理中", "待处理", "已修复", "列表", "记录", "高优先级", "中优先级", "低优先级"); keyword != "" {
		query = query.Where(
			"vuln_title LIKE ? OR assignee_name LIKE ? OR department LIKE ? OR status LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var tickets []models.VulnTicket
	if err := query.Order("created_at desc").Limit(5).Find(&tickets).Error; err != nil {
		return &toolContext{ToolName: "query_security_tickets", Summary: "漏洞工单查询失败，当前无法读取工单数据。", Error: "security tickets query failed"}
	}
	if len(tickets) == 0 {
		return &toolContext{ToolName: "query_security_tickets", Summary: "当前没有匹配的漏洞工单记录。"}
	}

	lines := make([]string, 0, len(tickets))
	cards := make([]ResultCard, 0, len(tickets))
	for _, ticket := range tickets {
		lines = append(lines, fmt.Sprintf("- 工单：%d，标题：%s，状态：%s，优先级：%s，指派人：%s", ticket.ID, defaultText(ticket.VulnTitle, "未命名工单"), ticket.Status, defaultText(ticket.Priority, "未设置"), defaultText(ticket.AssigneeName, "未指派")))
		cards = append(cards, ResultCard{
			Title:      fmt.Sprintf("工单 #%d", ticket.ID),
			Subtitle:   fmt.Sprintf("状态：%s | 优先级：%s", ticket.Status, defaultText(ticket.Priority, "未设置")),
			Meta:       fmt.Sprintf("漏洞：%s | 指派人：%s", defaultText(ticket.VulnTitle, "未命名工单"), defaultText(ticket.AssigneeName, "未指派")),
			ToolName:   "security.tickets",
			SourceType: "security",
		})
	}

	return &toolContext{ToolName: "query_security_tickets", Summary: "查询到的漏洞工单如下：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func buildFocusedSecurityVulnerabilityContext(vulnerability models.SecurityVulnerability) *toolContext {
	target := defaultText(vulnerability.IP, "未知目标")
	if vulnerability.Port > 0 {
		target = fmt.Sprintf("%s:%d", target, vulnerability.Port)
	}

	lines := []string{
		fmt.Sprintf("当前漏洞：%s，级别 %s，状态 %s，目标 %s。", defaultText(vulnerability.Title, "未命名漏洞"), vulnerability.Severity, vulnerability.Status, target),
	}
	if strings.TrimSpace(vulnerability.CVEID) != "" {
		lines = append(lines, "漏洞编号："+vulnerability.CVEID)
	}
	if strings.TrimSpace(vulnerability.Description) != "" {
		lines = append(lines, "说明："+shorten(vulnerability.Description, 160))
	}
	if strings.TrimSpace(vulnerability.Solution) != "" {
		lines = append(lines, "修复建议："+shorten(vulnerability.Solution, 160))
	}

	return &toolContext{
		ToolName: "query_vulnerability_summary",
		Summary:  strings.Join(lines, "\n"),
		Cards: []ResultCard{{
			Title:      defaultText(vulnerability.Title, "未命名漏洞"),
			Subtitle:   fmt.Sprintf("级别：%s | 状态：%s", vulnerability.Severity, vulnerability.Status),
			Meta:       fmt.Sprintf("目标：%s | 编号：%s", target, defaultText(vulnerability.CVEID, defaultText(vulnerability.TemplateID, "无编号"))),
			ToolName:   "security.vulnerabilities",
			SourceType: "security",
		}},
	}
}

func vulnerabilityQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近新增了哪些高危漏洞", "当前未处理的高危漏洞有多少", "哪些资产的漏洞风险最高") {
		return ""
	}

	return extractPrimaryKeyword(message,
		"漏洞", "风险", "最近", "状态", "高危", "严重", "中危", "低危", "处理中", "待处理", "数量", "列表", "多少", "哪些", "有哪些", "当前", "新增",
	)
}

func securityTaskQueryKeyword(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	if containsAny(msg, "最近有哪些失败扫描任务", "当前还有哪些扫描任务在运行", "最近扫描异常集中在哪些目标") {
		return ""
	}

	return extractPrimaryKeyword(message,
		"扫描", "任务", "最近", "状态", "运行", "失败", "完成", "列表", "记录", "web", "网页", "站点", "主机", "端口", "哪些", "有哪些", "当前",
	)
}

func summarizeSecurityVulnerabilities(message string, vulns []models.SecurityVulnerability, total int64) string {
	msg := strings.ToLower(message)
	assetCounts := map[string]int{}
	for _, vuln := range vulns {
		target := vuln.IP
		if vuln.Port > 0 {
			target = fmt.Sprintf("%s:%d", vuln.IP, vuln.Port)
		}
		assetCounts[target]++
	}

	switch {
	case containsAny(msg, "高危") && containsAny(msg, "新增", "最近新增"):
		return fmt.Sprintf("结论：最近匹配到 %d 条高危漏洞记录，建议优先处理最上面的高危项。", len(vulns))
	case containsAny(msg, "未处理", "待处理") && containsAny(msg, "多少", "有多少"):
		return fmt.Sprintf("结论：当前匹配到 %d 条未处理高危漏洞。", total)
	case containsAny(msg, "资产", "风险最高"):
		asset, count := topCountString(assetCounts)
		if asset != "" {
			return fmt.Sprintf("结论：当前漏洞风险更集中在资产 %s，最近返回结果里出现了 %d 次。", asset, count)
		}
		return "结论：当前没有识别到明确的高风险集中资产。"
	default:
		return fmt.Sprintf("结论：当前匹配到 %d 条漏洞记录。", total)
	}
}

func summarizeSecurityScanTasks(message string, tasks []models.SecurityScanTask) string {
	msg := strings.ToLower(message)
	targetCounts := map[string]int{}
	running := 0
	failed := 0
	for _, task := range tasks {
		targetCounts[defaultText(task.Target, "未设置目标")]++
		if task.Status == models.TaskStatusRunning {
			running++
		}
		if task.Status == models.TaskStatusFailed {
			failed++
		}
	}

	switch {
	case containsAny(msg, "失败"):
		return fmt.Sprintf("结论：最近匹配到 %d 条失败扫描任务，建议优先查看失败任务的报错信息。", failed)
	case containsAny(msg, "运行", "执行中"):
		return fmt.Sprintf("结论：当前有 %d 条扫描任务仍在运行。", running)
	case containsAny(msg, "异常集中", "集中", "哪些目标"):
		target, count := topCountString(targetCounts)
		if target != "" && count > 1 {
			return fmt.Sprintf("结论：最近扫描异常更集中在目标 %s，最近返回结果里出现了 %d 次。", target, count)
		}
		if target != "" {
			return fmt.Sprintf("结论：最近扫描异常暂未形成明显集中，当前排在前面的是目标 %s。", target)
		}
		return "结论：最近没有识别到明显集中的异常目标。"
	default:
		return fmt.Sprintf("结论：当前匹配到 %d 条扫描任务记录。", len(tasks))
	}
}

func topCountString(counts map[string]int) (string, int) {
	type pair struct {
		key   string
		count int
	}
	items := make([]pair, 0, len(counts))
	for key, count := range counts {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		items = append(items, pair{key: key, count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].key < items[j].key
		}
		return items[i].count > items[j].count
	})
	if len(items) == 0 {
		return "", 0
	}
	return items[0].key, items[0].count
}
