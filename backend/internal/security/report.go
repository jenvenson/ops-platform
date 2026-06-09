package security

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/jenvenson/ops-platform/internal/models"
)

type reportOccurrenceSummary struct {
	Count        int
	FirstTarget  string
	SourceEngine string
}

type reportFindingGroups struct {
	Confirmed []models.SecurityVulnerability
	Candidate []models.SecurityVulnerability
	Inventory []models.SecurityVulnerability
}

type reportSeverityCounts struct {
	High   int64
	Medium int64
	Low    int64
	Info   int64
}

func splitReportFindings(vulns []models.SecurityVulnerability) reportFindingGroups {
	groups := reportFindingGroups{
		Confirmed: make([]models.SecurityVulnerability, 0),
		Candidate: make([]models.SecurityVulnerability, 0),
		Inventory: make([]models.SecurityVulnerability, 0),
	}

	for _, vuln := range vulns {
		switch {
		case inferFindingFamily(&vuln) == "inventory":
			groups.Inventory = append(groups.Inventory, vuln)
		case isCandidateHostVersionMatch(vuln):
			groups.Candidate = append(groups.Candidate, vuln)
		default:
			groups.Confirmed = append(groups.Confirmed, vuln)
		}
	}

	return groups
}

func countReportSeverities(vulns []models.SecurityVulnerability) reportSeverityCounts {
	counts := reportSeverityCounts{}
	for _, vuln := range vulns {
		switch strings.ToLower(strings.TrimSpace(vuln.Severity)) {
		case "critical", "high":
			counts.High++
		case "medium":
			counts.Medium++
		case "low":
			counts.Low++
		case "info":
			counts.Info++
		}
	}
	return counts
}

func reportEntriesFromVulnerabilities(vulns []models.SecurityVulnerability) []Vulnerability {
	entries := make([]Vulnerability, 0, len(vulns))
	for _, v := range vulns {
		entries = append(entries, Vulnerability{
			ID:            v.ID,
			Severity:      v.Severity,
			CVEID:         v.CVEID,
			PrimaryCVEID:  v.PrimaryCVEID,
			Title:         v.Title,
			Description:   v.Description,
			Solution:      v.Solution,
			VulnType:      v.VulnType,
			VulnURL:       v.VulnURL,
			ScanMethod:    v.ScanMethod,
			Scanner:       v.Scanner,
			FindingSource: v.FindingSource,
			FindingFamily: v.FindingFamily,
			Confidence:    v.Confidence,
			MatchMode:     v.MatchMode,
			RiskCategory:  v.RiskCategory,
			DisplayGroup:  v.DisplayGroup,
			Payload:       v.Payload,
			Response:      v.Response,
			CVSSScore:     v.CVSSScore,
			IP:            v.IP,
			Port:          v.Port,
			Status:        v.Status,
			CreatedAt:     v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return entries
}

// generateHTMLReport 生成简洁的安全扫描报告
func generateHTMLReport(task models.SecurityScanTask, findings reportFindingGroups, counts reportSeverityCounts, assets []models.SecurityAsset) string {
	return generateHTMLReportWithDetails(task, findings, counts, assets, nil, nil, nil, nil)
}

func generateHTMLReportWithDetails(task models.SecurityScanTask, findings reportFindingGroups, counts reportSeverityCounts, assets []models.SecurityAsset, currentRun *SecurityScanRunDetail, targets []SecurityScanTargetDetail, occurrences []SecurityScanFindingOccurrenceDetail, evidences []SecurityScanEvidenceDetail) string {
	totalConfirmed := counts.High + counts.Medium + counts.Low + counts.Info

	// 格式化时间
	startedAt := ""
	completedAt := ""
	if task.StartedAt != nil {
		startedAt = task.StartedAt.Format("2006-01-02 15:04:05")
	}
	if task.CompletedAt != nil {
		completedAt = task.CompletedAt.Format("2006-01-02 15:04:05")
	}

	executionSummaryHTML := generateExecutionSummaryHTML(task, currentRun, targets, occurrences, evidences)
	targetTableHTML := generateTargetDetailTable(targets)
	targetSectionTitle := "8. 扫描目标"
	if task.ScanType == string(models.ScanTypeWeb) {
		targetSectionTitle = "8. 攻击面目标"
	}

	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>安全扫描报告 - ` + task.Name + `</title>
<style>
* { box-sizing: border-box; }
body { font-family: "Microsoft YaHei", "Segoe UI", Arial, sans-serif; margin: 0; padding: 20px; background: #f5f7fa; color: #333; }
.container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); overflow: hidden; }
.header { background: linear-gradient(135deg, #1a365d 0%, #2d3748 100%); color: white; padding: 40px; text-align: center; }
.header h1 { margin: 0 0 10px 0; font-size: 28px; font-weight: 600; }
.header .subtitle { font-size: 14px; opacity: 0.9; }
.section { padding: 30px 40px; border-bottom: 1px solid #eee; }
.section:last-child { border-bottom: none; }
.section-title { font-size: 20px; font-weight: 600; color: #1a365d; margin-bottom: 20px; padding-bottom: 10px; border-bottom: 2px solid #3182ce; }

.summary-cards { display: flex; gap: 20px; margin-bottom: 20px; }
.card { flex: 1; padding: 20px; border-radius: 8px; text-align: center; }
.card.high { background: linear-gradient(135deg, #e53e3e 0%, #c53030 100%); color: white; }
.card.medium { background: linear-gradient(135deg, #dd6b20 0%, #c05621 100%); color: white; }
.card.low { background: linear-gradient(135deg, #38a169 0%, #2f855a 100%); color: white; }
.card.info { background: linear-gradient(135deg, #3182ce 0%, #2b6cb0 100%); color: white; }
.card .value { font-size: 36px; font-weight: 700; }
.card .label { font-size: 14px; margin-top: 5px; opacity: 0.9; }

.task-info { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
.info-table { width: 100%; border-collapse: collapse; }
.info-table th, .info-table td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #e2e8f0; }
.info-table th { background: #f7fafc; font-weight: 600; color: #4a5568; width: 120px; }
.info-table td { color: #2d3748; }
.risk-badge { display: inline-block; padding: 4px 12px; border-radius: 4px; font-weight: 600; }
.risk-badge.high { background: #fed7d7; color: #c53030; }
.risk-badge.medium { background: #feebc8; color: #c05621; }
.risk-badge.low { background: #c6f6d5; color: #2f855a; }

.data-table { width: 100%; border-collapse: collapse; margin: 15px 0; }
.data-table th, .data-table td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #e2e8f0; }
.data-table th { background: #f7fafc; font-weight: 600; color: #4a5568; }
.data-table tr:hover { background: #f7fafc; }
.data-table .severity-high { color: #c53030; font-weight: 600; }
.data-table .severity-medium { color: #c05621; font-weight: 600; }
.data-table .severity-low { color: #2f855a; font-weight: 600; }
.data-table .severity-info { color: #3182ce; }

.vuln-card { border: 1px solid #e2e8f0; border-radius: 8px; margin: 15px 0; overflow: hidden; }
.vuln-header { padding: 15px 20px; background: #f7fafc; border-bottom: 1px solid #e2e8f0; display: flex; align-items: center; gap: 15px; }
.vuln-header.high { background: linear-gradient(135deg, #fff5f5 0%, #fed7d7 100%); }
.vuln-header.medium { background: linear-gradient(135deg, #fffaf0 0%, #feebc8 100%); }
.vuln-header.low { background: linear-gradient(135deg, #f0fff4 0%, #c6f6d5 100%); }
.vuln-title { font-weight: 600; font-size: 15px; color: #2d3748; }
.vuln-body { padding: 20px; }
.vuln-row { display: flex; margin: 8px 0; }
.vuln-row .label { width: 100px; color: #718096; flex-shrink: 0; }
.vuln-row .value { color: #2d3748; }
.vuln-cve { font-family: "Consolas", monospace; background: #edf2f7; padding: 2px 8px; border-radius: 4px; font-size: 13px; }

.host-card { border: 1px solid #e2e8f0; border-radius: 8px; margin: 10px 0; }
.host-header { padding: 12px 20px; background: #f7fafc; display: flex; justify-content: space-between; align-items: center; }
.host-ip { font-weight: 600; color: #2d3748; }
.host-risk { font-size: 14px; }
.host-stats { display: flex; gap: 30px; }
.host-stat { text-align: center; }
.host-stat .num { font-size: 20px; font-weight: 700; }
.host-stat .num.high { color: #c53030; }
.host-stat .num.medium { color: #c05621; }
.host-stat .num.low { color: #2f855a; }
.host-stat .txt { font-size: 12px; color: #718096; }

.distribution-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
.distribution-item { background: #f7fafc; padding: 20px; border-radius: 8px; }
.distribution-title { font-weight: 600; margin-bottom: 15px; color: #2d3748; }
.distribution-bar { display: flex; align-items: center; margin: 8px 0; }
.distribution-bar .name { width: 100px; font-size: 13px; color: #4a5568; }
.distribution-bar .bar { flex: 1; height: 20px; background: #e2e8f0; border-radius: 4px; overflow: hidden; }
.distribution-bar .fill { height: 100%; border-radius: 4px; transition: width 0.3s; }
.distribution-bar .fill.high { background: linear-gradient(90deg, #fc8181 0%, #e53e3e 100%); }
.distribution-bar .fill.medium { background: linear-gradient(90deg, #f6ad55 0%, #dd6b20 100%); }
.distribution-bar .fill.low { background: linear-gradient(90deg, #68d391 0%, #38a169 100%); }
.distribution-bar .count { width: 50px; text-align: right; font-size: 13px; color: #4a5568; }
.scope-note { margin: 18px 0 0 0; padding: 14px 16px; border-radius: 8px; background: #fffaf0; color: #744210; border: 1px solid #f6ad55; }
.scope-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-top: 20px; }
.scope-card { padding: 18px; border-radius: 8px; background: #f7fafc; border: 1px solid #e2e8f0; }
.scope-card .value { font-size: 28px; font-weight: 700; color: #1a365d; }
.scope-card .label { margin-top: 6px; font-size: 14px; color: #4a5568; }
.scope-card .hint { margin-top: 8px; font-size: 12px; color: #718096; line-height: 1.5; }
.finding-badge { display: inline-block; padding: 4px 10px; border-radius: 999px; font-size: 12px; font-weight: 600; }
.finding-badge.confirmed { background: #e6fffa; color: #234e52; }
.finding-badge.candidate { background: #fffaf0; color: #9c4221; }
.finding-badge.inventory { background: #ebf8ff; color: #2a4365; }

.footer { padding: 20px 40px; background: #f7fafc; text-align: center; color: #718096; font-size: 13px; }
</style>
</head>
<body>
<div class="container">
<div class="header">
    <h1>安全扫描报告</h1>
    <div class="subtitle">任务: ` + task.Name + ` | 目标: ` + task.Target + `</div>
    <div class="subtitle">扫描时间: ` + task.CreatedAt.Format("2006-01-02 15:04:05") + `</div>
</div>

<!-- 综述信息 -->
<div class="section">
    <div class="section-title">1. 综述信息</div>
    <div class="summary-cards">
        <div class="card high">
            <div class="value">` + fmt.Sprintf("%d", counts.High) + `</div>
            <div class="label">已确认高危</div>
        </div>
        <div class="card medium">
            <div class="value">` + fmt.Sprintf("%d", counts.Medium) + `</div>
            <div class="label">已确认中危</div>
        </div>
        <div class="card low">
            <div class="value">` + fmt.Sprintf("%d", counts.Low) + `</div>
            <div class="label">已确认低危</div>
        </div>
        <div class="card info">
            <div class="value">` + fmt.Sprintf("%d", len(assets)) + `</div>
            <div class="label">开放端口</div>
        </div>
    </div>
    <div class="scope-grid">
        <div class="scope-card">
            <div class="value">` + fmt.Sprintf("%d", len(findings.Confirmed)) + `</div>
            <div class="label">已确认漏洞</div>
            <div class="hint">计入任务风险统计和本报告风险分布。</div>
        </div>
        <div class="scope-card">
            <div class="value">` + fmt.Sprintf("%d", len(findings.Candidate)) + `</div>
            <div class="label">待验证</div>
            <div class="hint">根据目标开放服务的产品名、版本号或 CPE 信息，与漏洞知识库自动比对后得到的风险线索，仅作为排查依据，不并入正式风险。</div>
        </div>
        <div class="scope-card">
            <div class="value">` + fmt.Sprintf("%d", len(findings.Inventory)) + `</div>
            <div class="label">资产识别</div>
            <div class="hint">用于资产盘点和版本枚举，不作为漏洞结论。</div>
        </div>
    </div>
    <div class="scope-note">报告口径说明：风险评分、主机风险列表和威胁分布仅统计“已确认漏洞”；待验证结果和资产识别会单独展示，不混入正式风险。</div>

    <div class="task-info">
        <div>
            <table class="info-table">
                <tr>
                    <th>任务名称</th>
                    <td>` + task.Name + `</td>
                </tr>
                <tr>
                    <th>扫描目标</th>
                    <td>` + task.Target + `</td>
                </tr>
                <tr>
                    <th>任务状态</th>
                    <td>` + getStatusCN(task.Status) + `</td>
                </tr>
            </table>
        </div>
        <div>
            <table class="info-table">
                <tr>
                    <th>扫描时间</th>
                    <td>` + startedAt + ` ~ ` + completedAt + `</td>
                </tr>
                <tr>
                    <th>扫描主机</th>
                    <td>` + fmt.Sprintf("%d", task.ScannedIPs) + `</td>
                </tr>
                <tr>
                    <th>Nuclei版本</th>
                    <td>` + task.NucleiVersion + `</td>
                </tr>
                <tr>
                    <th>当前阶段</th>
                    <td>` + html.EscapeString(reportRunPhaseCN(currentRun)) + `</td>
                </tr>
            </table>
        </div>
    </div>
    ` + executionSummaryHTML + `
</div>

<!-- 风险分布 -->
<div class="section">
    <div class="section-title">2. 风险分布</div>
    <div class="distribution-grid">
        <div class="distribution-item">
            <div class="distribution-title">按严重程度</div>
            <div class="distribution-bar">
                <div class="name">高危</div>
                <div class="bar"><div class="fill high" style="width: ` + getPercent(counts.High, totalConfirmed) + `%"></div></div>
                <div class="count">` + fmt.Sprintf("%d", counts.High) + `</div>
            </div>
            <div class="distribution-bar">
                <div class="name">中危</div>
                <div class="bar"><div class="fill medium" style="width: ` + getPercent(counts.Medium, totalConfirmed) + `%"></div></div>
                <div class="count">` + fmt.Sprintf("%d", counts.Medium) + `</div>
            </div>
            <div class="distribution-bar">
                <div class="name">低危</div>
                <div class="bar"><div class="fill low" style="width: ` + getPercent(counts.Low, totalConfirmed) + `%"></div></div>
                <div class="count">` + fmt.Sprintf("%d", counts.Low) + `</div>
            </div>
            <div class="distribution-bar">
                <div class="name">信息</div>
                <div class="bar"><div class="fill low" style="width: ` + getPercent(counts.Info, totalConfirmed) + `%"></div></div>
                <div class="count">` + fmt.Sprintf("%d", counts.Info) + `</div>
            </div>
        </div>
        <div class="distribution-item">
            <div class="distribution-title">按威胁类型</div>
            ` + generateThreatDistribution(findings.Confirmed) + `
        </div>
    </div>
</div>

<!-- 主机风险列表 -->
<div class="section">
    <div class="section-title">3. 主机风险列表</div>
    <table class="data-table">
        <thead>
            <tr>
                <th>IP地址</th>
                <th>操作系统</th>
                <th>高危</th>
                <th>中危</th>
                <th>低危</th>
                <th>风险评分</th>
            </tr>
        </thead>
        <tbody>
            ` + generateHostTable(findings.Confirmed, assets) + `
        </tbody>
    </table>
</div>

<!-- 已确认漏洞 -->
<div class="section">
    <div class="section-title">4. 已确认漏洞</div>
    ` + generateVulnDetails(findings.Confirmed, "confirmed") + `
</div>

<!-- 待验证 -->
<div class="section">
    <div class="section-title">5. 待验证</div>
    ` + generateVulnDetails(findings.Candidate, "candidate") + `
</div>

<!-- 资产识别 -->
<div class="section">
    <div class="section-title">6. 资产识别</div>
    ` + generateVulnDetails(findings.Inventory, "inventory") + `
</div>

<!-- 资产清单 -->
<div class="section">
    <div class="section-title">7. 资产清单</div>
    <table class="data-table">
        <thead>
            <tr>
                <th>IP地址</th>
                <th>端口</th>
                <th>协议</th>
                <th>服务</th>
                <th>版本</th>
            </tr>
        </thead>
        <tbody>
            ` + generateAssetTable(assets) + `
        </tbody>
    </table>
</div>

<!-- 扫描目标 -->
<div class="section">
    <div class="section-title">` + targetSectionTitle + `</div>
    ` + targetTableHTML + `
</div>

<div class="footer">
    <p>报告生成时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
    <p>Powered by OPS Platform Security Scanner</p>
</div>
</div>
</body>
</html>`

	return html
}

func reportSnapshotInt(snapshot map[string]interface{}, key string) int {
	if snapshot == nil {
		return 0
	}
	switch value := snapshot[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func reportSnapshotString(snapshot map[string]interface{}, key string) string {
	if snapshot == nil {
		return ""
	}
	if value, ok := snapshot[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func reportRunPhaseCN(run *SecurityScanRunDetail) string {
	if run == nil {
		return "-"
	}
	switch strings.TrimSpace(run.Phase) {
	case "discovery":
		return "发现阶段"
	case "verification":
		return "验证阶段"
	case "completed":
		return "已完成"
	case "failed":
		return "失败"
	case "cancelled":
		return "已取消"
	default:
		if strings.TrimSpace(run.Phase) != "" {
			return run.Phase
		}
		return "-"
	}
}

func reportDiscoveryModeCN(mode string) string {
	switch strings.TrimSpace(mode) {
	case "browser":
		return "浏览器发现"
	case "http":
		return "HTTP 发现"
	case "none":
		return "不自动发现"
	default:
		if strings.TrimSpace(mode) == "" {
			return "-"
		}
		return mode
	}
}

func reportScanProfileCN(profile string) string {
	switch strings.TrimSpace(profile) {
	case "deep":
		return "专项扫描"
	case "standard":
		return "标准扫描"
	default:
		if strings.TrimSpace(profile) == "" {
			return "-"
		}
		return profile
	}
}

func generateExecutionSummaryHTML(task models.SecurityScanTask, currentRun *SecurityScanRunDetail, targets []SecurityScanTargetDetail, occurrences []SecurityScanFindingOccurrenceDetail, evidences []SecurityScanEvidenceDetail) string {
	if currentRun == nil && len(targets) == 0 && len(occurrences) == 0 && len(evidences) == 0 {
		return ""
	}

	discoveredCount := reportSnapshotInt(currentRun.TargetSnapshot, "discovered_count")
	scannedCount := reportSnapshotInt(currentRun.SummarySnapshot, "scanned_targets")
	ruleOnlyCount := reportSnapshotInt(currentRun.SummarySnapshot, "rule_only_target_count")
	skippedCount := reportSnapshotInt(currentRun.SummarySnapshot, "skipped_target_count")
	fallbackCount := reportSnapshotInt(currentRun.SummarySnapshot, "browser_fallback_count")
	if discoveredCount == 0 {
		discoveredCount = len(targets)
	}
	if scannedCount == 0 && currentRun != nil {
		scannedCount = currentRun.ScannedTargets
	}

	rows := []string{}
	appendRow := func(label, value string) {
		if strings.TrimSpace(value) == "" || value == "-" {
			return
		}
		rows = append(rows, fmt.Sprintf(`<tr><th>%s</th><td>%s</td></tr>`, html.EscapeString(label), html.EscapeString(value)))
	}

	appendRow("执行阶段", reportRunPhaseCN(currentRun))
	appendRow("扫描策略", reportScanProfileCN(reportSnapshotString(currentRun.ConfigSnapshot, "scan_profile")))
	appendRow("发现方式", reportDiscoveryModeCN(firstNonEmpty(reportSnapshotString(currentRun.SummarySnapshot, "discovery_mode"), reportSnapshotString(currentRun.ConfigSnapshot, "discovery_mode"))))
	appendRow("发现目标", fmt.Sprintf("%d", discoveredCount))
	appendRow("实际扫描", fmt.Sprintf("%d", scannedCount))
	appendRow("仅规则检测", fmt.Sprintf("%d", ruleOnlyCount))
	appendRow("预算跳过", fmt.Sprintf("%d", skippedCount))
	appendRow("浏览器回退", fmt.Sprintf("%d", fallbackCount))
	appendRow("命中记录", fmt.Sprintf("%d", len(occurrences)))
	appendRow("证据条数", fmt.Sprintf("%d", len(evidences)))

	if len(rows) == 0 {
		return ""
	}

	return `<div style="margin-top: 20px;">
        <table class="info-table">` + strings.Join(rows, "") + `</table>
    </div>`
}

func generateTargetDetailTable(targets []SecurityScanTargetDetail) string {
	if len(targets) == 0 {
		return `<p style='color: #718096;'>暂无新模型目标数据</p>`
	}

	var rows strings.Builder
	for _, target := range targets {
		parent := "-"
		if target.ParentTargetID != nil {
			parent = fmt.Sprintf("#%d", *target.ParentTargetID)
		}
		rows.WriteString(fmt.Sprintf(`
            <tr>
                <td>#%d</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>`,
			target.ID,
			html.EscapeString(target.TargetURL),
			html.EscapeString(target.TargetKind),
			html.EscapeString(target.DiscoverySource),
			html.EscapeString(parent),
			html.EscapeString(target.Status),
			html.EscapeString(firstNonEmpty(target.Host, "-")),
		))
	}
	return `<table class="data-table">
        <thead>
            <tr>
                <th>ID</th>
                <th>目标</th>
                <th>类型</th>
                <th>来源</th>
                <th>父目标</th>
                <th>状态</th>
                <th>主机</th>
            </tr>
        </thead>
        <tbody>` + rows.String() + `</tbody>
    </table>`
}

// getPercent 计算百分比
func getPercent(count int64, total int64) string {
	if total == 0 {
		return "0"
	}
	return fmt.Sprintf("%.1f", float64(count)*100/float64(total))
}

// getStatusCN 获取状态中文
func getStatusCN(status string) string {
	switch status {
	case "pending":
		return "等待中"
	case "running":
		return "扫描中"
	case "completed":
		return "扫描完成"
	case "failed":
		return "扫描失败"
	default:
		return status
	}
}

// generateThreatDistribution 生成威胁类型分布
func generateThreatDistribution(vulns []models.SecurityVulnerability) string {
	if len(vulns) == 0 {
		return `<div style="color: #718096;">暂无数据</div>`
	}

	typeCount := make(map[string]int)
	for _, v := range vulns {
		vulnType := getVulnTypeCN(v.VulnType, v.Title)
		typeCount[vulnType]++
	}

	var html string
	keys := make([]string, 0, len(typeCount))
	for vulnType := range typeCount {
		keys = append(keys, vulnType)
	}
	sort.Strings(keys)
	for _, vulnType := range keys {
		count := typeCount[vulnType]
		percent := float64(count) * 100 / float64(len(vulns))
		html += fmt.Sprintf(`
            <div class="distribution-bar">
                <div class="name">%s</div>
                <div class="bar"><div class="fill low" style="width: %.1f%%"></div></div>
                <div class="count">%d</div>
            </div>`, vulnType, percent, count)
	}
	return html
}

// generateHostTable 生成主机风险表格
func generateHostTable(vulns []models.SecurityVulnerability, assets []models.SecurityAsset) string {
	// 按IP分组统计漏洞
	hostVulns := make(map[string]map[string]int)
	assetOS := make(map[string]string)

	for _, a := range assets {
		assetOS[a.IP] = a.OSInfo
	}

	for _, v := range vulns {
		if hostVulns[v.IP] == nil {
			hostVulns[v.IP] = map[string]int{"high": 0, "medium": 0, "low": 0}
		}
		switch strings.ToLower(strings.TrimSpace(v.Severity)) {
		case "critical", "high":
			hostVulns[v.IP]["high"]++
		case "medium":
			hostVulns[v.IP]["medium"]++
		case "low":
			hostVulns[v.IP]["low"]++
		}
	}

	var html string
	for ip, counts := range hostVulns {
		riskScore := float64(counts["high"]*10 + counts["medium"]*5 + counts["low"]*1)
		if riskScore > 10 {
			riskScore = 10
		}

		severityClass := "low"
		if counts["high"] > 0 {
			severityClass = "high"
		} else if counts["medium"] > 0 {
			severityClass = "medium"
		}

		html += fmt.Sprintf(`
            <tr>
                <td><span class="risk-badge %s">%s</span></td>
                <td>%s</td>
                <td class="severity-high">%d</td>
                <td class="severity-medium">%d</td>
                <td class="severity-low">%d</td>
                <td>%.1f</td>
            </tr>`, severityClass, ip, assetOS[ip], counts["high"], counts["medium"], counts["low"], riskScore)
	}
	return html
}

// generateVulnDetails 生成漏洞详情
func generateVulnDetails(vulns []models.SecurityVulnerability, section string) string {
	if len(vulns) == 0 {
		switch section {
		case "candidate":
			return "<p style='color: #718096;'>暂无待验证结果</p>"
		case "inventory":
			return "<p style='color: #718096;'>暂无资产识别结果</p>"
		default:
			return "<p style='color: #718096;'>未发现已确认漏洞</p>"
		}
	}

	var html string
	for _, v := range vulns {
		severityCN := "低危"
		severityClass := "low"
		if v.Severity == "critical" {
			severityCN = "严重"
			severityClass = "high"
		} else if v.Severity == "high" {
			severityCN = "高危"
			severityClass = "high"
		} else if v.Severity == "medium" {
			severityCN = "中危"
			severityClass = "medium"
		} else if v.Severity == "info" {
			severityCN = "信息"
			severityClass = "low"
		}

		cveDisplay := ""
		if v.CVEID != "" {
			cveDisplay = fmt.Sprintf(`<span class="vuln-cve">%s</span>`, v.CVEID)
		}

		sectionBadge := `<span class="finding-badge confirmed">已确认漏洞</span>`
		switch section {
		case "candidate":
			sectionBadge = `<span class="finding-badge candidate">待验证</span>`
		case "inventory":
			sectionBadge = `<span class="finding-badge inventory">资产识别</span>`
		}

		html += fmt.Sprintf(`
        <div class="vuln-card">
            <div class="vuln-header %s">
                <span class="risk-badge %s">%s</span>
                %s
                <span class="vuln-title">%s %s</span>
            </div>
            <div class="vuln-body">
                <div class="vuln-row">
                    <div class="label">影响主机</div>
                    <div class="value">%s:%d</div>
                </div>
                %s
                <div class="vuln-row">
                    <div class="label">CVSS评分</div>
                    <div class="value">%.1f %s</div>
                </div>
                %s
				%s
				%s
				%s
				%s
				%s
				%s
			</div>
		</div>`, severityClass, severityClass, severityCN, sectionBadge, v.Title, cveDisplay, v.IP, v.Port,
			getVulnRow("漏洞地址", v.VulnURL),
			v.CVSSScore, getCVSSLevel(v.CVSSScore),
			getVulnRow("漏洞类型", v.VulnType),
			getVulnRow("结果来源", reportFindingSourceCN(v.FindingSource)),
			getVulnRow("置信度", reportConfidenceCN(v.Confidence)),
			getVulnRow("匹配模式", reportMatchModeCN(v.MatchMode)),
			getVulnRow("发现方式", firstNonEmpty(v.ScanMethod, v.Scanner)),
			getVulnRow("描述", v.Description),
			getVulnRow("解决方案", v.Solution)+getVulnRow("命中依据", firstNonEmpty(v.Payload, v.MatchedOn))+getVulnRow("响应预览", v.Response))
	}
	return html
}

// getVulnRow 生成漏洞信息行
func getVulnRow(label, content string) string {
	if content == "" {
		return ""
	}
	// 限制描述长度
	if len(content) > 500 {
		content = content[:500] + "..."
	}
	content = html.EscapeString(content)
	content = strings.ReplaceAll(content, "\n", "<br/>")
	return fmt.Sprintf(`<div class="vuln-row"><div class="label">%s</div><div class="value">%s</div></div>`, html.EscapeString(label), content)
}

// getCVSSLevel 获取CVSS等级描述
func getCVSSLevel(score float64) string {
	if score >= 9.0 {
		return "(严重)"
	} else if score >= 7.0 {
		return "(高)"
	} else if score >= 4.0 {
		return "(中)"
	} else if score > 0 {
		return "(低)"
	}
	return ""
}

// generateAssetTable 生成资产表格
func generateAssetTable(assets []models.SecurityAsset) string {
	var html string
	for _, a := range assets {
		version := a.Version
		if version == "" {
			version = a.Banner
		}
		if version == "" {
			version = "-"
		}
		html += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%d</td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
            </tr>`, a.IP, a.Port, a.Protocol, a.ServiceName, version)
	}
	return html
}

// generateCSVReport 生成 CSV 格式的漏洞详细报告
func generateCSVReport(findings reportFindingGroups, task models.SecurityScanTask) string {
	return generateCSVReportWithDetails(findings, task, nil, nil)
}

func buildOccurrenceSummaryMap(occurrences []SecurityScanFindingOccurrenceDetail, evidences []SecurityScanEvidenceDetail) map[uint]reportOccurrenceSummary {
	result := map[uint]reportOccurrenceSummary{}
	evidenceMap := make(map[uint]SecurityScanEvidenceDetail, len(evidences))
	for _, evidence := range evidences {
		evidenceMap[evidence.ID] = evidence
	}
	for _, occurrence := range occurrences {
		if occurrence.LegacyVulnerabilityID == nil {
			continue
		}
		current := result[*occurrence.LegacyVulnerabilityID]
		current.Count++
		if current.FirstTarget == "" {
			current.FirstTarget = firstNonEmpty(
				func() string {
					if occurrence.Target != nil {
						return occurrence.Target.TargetURL
					}
					return ""
				}(),
				reportSnapshotString(occurrence.Metadata, "vuln_url"),
			)
		}
		if current.SourceEngine == "" && occurrence.EvidenceID != nil {
			if evidence, exists := evidenceMap[*occurrence.EvidenceID]; exists {
				current.SourceEngine = firstNonEmpty(evidence.SourceEngine, evidence.EvidenceType)
			}
		}
		result[*occurrence.LegacyVulnerabilityID] = current
	}
	return result
}

func generateCSVReportWithDetails(findings reportFindingGroups, task models.SecurityScanTask, occurrences []SecurityScanFindingOccurrenceDetail, evidences []SecurityScanEvidenceDetail) string {
	// CSV 表头
	headers := []string{
		"报告分类",
		"结果来源",
		"置信度",
		"匹配模式",
		"风险类别",
		"主机IP",
		"漏洞名称",
		"详细描述",
		"漏洞类型",
		"协议",
		"漏洞端口",
		"CVSS得分",
		"CVSS等级",
		"处置优先级",
		"CVE编号",
		"主CVE编号",
		"CNVD编号",
		"CNNVD编号",
		"CNCVE编号",
		"修复建议",
		"返回信息",
		"漏洞地址",
		"扫描方式",
		"命中记录数",
		"首个命中目标",
		"证据来源",
		"是否误报",
	}

	occurrenceMap := buildOccurrenceSummaryMap(occurrences, evidences)

	// 构建 CSV 内容
	var csv strings.Builder
	csv.WriteString(strings.Join(headers, ",") + "\n")

	appendRows := func(category string, vulns []models.SecurityVulnerability) {
		for _, v := range vulns {
			// 转义 CSV 中的特殊字符
			escapeCSV := func(s string) string {
				s = strings.ReplaceAll(s, "\"", "\"\"")
				s = strings.ReplaceAll(s, "\n", " ")
				s = strings.ReplaceAll(s, "\r", " ")
				if strings.Contains(s, ",") || strings.Contains(s, "\"") {
					s = "\"" + s + "\""
				}
				return s
			}

			// 获取 CVSS 等级
			cvssLevel := "低危"
			if v.CVSSScore >= 9.0 {
				cvssLevel = "严重"
			} else if v.CVSSScore >= 7.0 {
				cvssLevel = "高危"
			} else if v.CVSSScore >= 4.0 {
				cvssLevel = "中危"
			}

			// 优先级映射
			priority := "低"
			if v.Severity == "critical" || v.Severity == "high" {
				priority = "高"
			} else if v.Severity == "medium" {
				priority = "中"
			}

			// 漏洞类型中文
			vulnTypeCN := getVulnTypeCN(v.VulnType, v.Title)

			// 是否误报
			falsePositive := "否"
			if v.FalsePositive {
				falsePositive = "是"
			}

			// 协议
			protocol := v.Protocol
			if protocol == "" {
				protocol = "TCP"
			}
			occurrenceSummary := occurrenceMap[v.ID]

			row := []string{
				escapeCSV(category),
				escapeCSV(reportFindingSourceCN(v.FindingSource)),
				escapeCSV(reportConfidenceCN(v.Confidence)),
				escapeCSV(reportMatchModeCN(v.MatchMode)),
				escapeCSV(v.RiskCategory),
				escapeCSV(v.IP),
				escapeCSV(v.Title),
				escapeCSV(v.Description),
				escapeCSV(vulnTypeCN),
				escapeCSV(protocol),
				fmt.Sprintf("%d", v.Port),
				fmt.Sprintf("%.1f", v.CVSSScore),
				escapeCSV(cvssLevel),
				escapeCSV(priority),
				escapeCSV(v.CVEID),
				escapeCSV(v.PrimaryCVEID),
				escapeCSV(v.CNVDID),
				escapeCSV(v.CNNVDID),
				escapeCSV(v.CNCVEID),
				escapeCSV(v.Solution),
				escapeCSV(v.Response),
				escapeCSV(firstNonEmpty(v.VulnURL, fmt.Sprintf("%s:%d", v.IP, v.Port))),
				escapeCSV(v.ScanMethod),
				fmt.Sprintf("%d", occurrenceSummary.Count),
				escapeCSV(occurrenceSummary.FirstTarget),
				escapeCSV(occurrenceSummary.SourceEngine),
				falsePositive,
			}
			csv.WriteString(strings.Join(row, ",") + "\n")
		}
	}

	appendRows("已确认漏洞", findings.Confirmed)
	appendRows("待验证", findings.Candidate)
	appendRows("资产识别", findings.Inventory)

	return csv.String()
}

func reportFindingSourceCN(source string) string {
	switch strings.TrimSpace(source) {
	case "web-template":
		return "Web 模板"
	case "web-rule":
		return "Web 规则"
	case "host-template":
		return "服务模板"
	case "host-version-match":
		return "版本匹配"
	case hostManualConfirmedFindingSource:
		return "人工确认"
	case "asset-inventory":
		return "资产识别"
	default:
		return source
	}
}

func reportConfidenceCN(confidence string) string {
	switch strings.TrimSpace(confidence) {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return confidence
	}
}

func reportMatchModeCN(mode string) string {
	switch strings.TrimSpace(mode) {
	case "template":
		return "模板命中"
	case "rule":
		return "规则匹配"
	case "version-range":
		return "版本区间"
	case "fuzzy-product":
		return "产品模糊匹配"
	case "inventory":
		return "资产识别"
	case "exact":
		return "精确匹配"
	case manualReviewMatchMode:
		return "人工复核"
	default:
		return mode
	}
}

// getVulnTypeCN 获取漏洞类型中文
func getVulnTypeCN(vulnType, title string) string {
	// 如果有明确的类型字段，直接返回
	if vulnType != "" {
		typeCN := map[string]string{
			"rce":        "远程代码执行",
			"xss":        "跨站脚本",
			"sqli":       "SQL注入",
			"ssrf":       "服务器端请求伪造",
			"ssti":       "服务器端模板注入",
			"csrf":       "跨站请求伪造",
			"lfi":        "本地文件包含",
			"fi":         "文件包含",
			"info":       "信息泄露",
			"disclosure": "信息泄露",
		}
		if cn, ok := typeCN[strings.ToLower(vulnType)]; ok {
			return cn
		}
	}

	// 从标题推断
	titleLower := strings.ToLower(title)
	if strings.Contains(titleLower, "xss") || strings.Contains(titleLower, "cross-site") {
		return "跨站脚本"
	} else if strings.Contains(titleLower, "sql") || strings.Contains(titleLower, "injection") {
		return "SQL注入"
	} else if strings.Contains(titleLower, "rce") || strings.Contains(titleLower, "remote code") || strings.Contains(titleLower, "command injection") {
		return "远程代码执行"
	} else if strings.Contains(titleLower, "ssrf") {
		return "服务器端请求伪造"
	} else if strings.Contains(titleLower, "xxe") {
		return "XML外部实体注入"
	} else if strings.Contains(titleLower, "csrf") || strings.Contains(titleLower, "cross-site request") {
		return "跨站请求伪造"
	} else if strings.Contains(titleLower, "lfi") || strings.Contains(titleLower, "file inclusion") {
		return "本地文件包含"
	} else if strings.Contains(titleLower, "ssti") || strings.Contains(titleLower, "template injection") {
		return "服务器端模板注入"
	} else if strings.Contains(titleLower, "information") || strings.Contains(titleLower, "info") || strings.Contains(titleLower, "disclosure") || strings.Contains(titleLower, "泄露") {
		return "信息泄露"
	} else if strings.Contains(titleLower, "auth") || strings.Contains(titleLower, "bypass") || strings.Contains(titleLower, "authentication") {
		return "认证绕过"
	} else if strings.Contains(titleLower, "sensitive") || strings.Contains(titleLower, "credentials") || strings.Contains(titleLower, "secret") {
		return "敏感信息"
	} else if strings.Contains(titleLower, "file") || strings.Contains(titleLower, "traversal") {
		return "文件遍历"
	}

	return "其他"
}
