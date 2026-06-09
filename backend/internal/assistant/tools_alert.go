package assistant

import (
	"fmt"
	"strings"

	"github.com/jenvenson/ops-platform/internal/alert"
	"github.com/jenvenson/ops-platform/internal/database"
)

type alertQueryOptions struct {
	statuses []string
	orderBy  string
	keyword  string
}

func (s *Service) queryAlertEvents(message string, pageContext *AssistantPageContext) *toolContext {
	options := deriveAlertQueryOptions(message)
	query := database.DB.Model(&alert.AlertEvent{})

	if shouldUseFocusedObjectQuery(message, pageContext, "alert_event") {
		if eventID, ok := pageObjectID(pageContext); ok {
			var event alert.AlertEvent
			if err := database.DB.Where("id = ?", eventID).First(&event).Error; err == nil {
				return buildFocusedAlertEventContext(event)
			}
		}
	}

	if status := pageFilterValue(pageContext, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if severity := pageFilterValue(pageContext, "severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if category := pageFilterValue(pageContext, "category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if source := pageFilterValue(pageContext, "source"); source != "" {
		query = query.Where("source LIKE ?", "%"+source+"%")
	}

	switch len(options.statuses) {
	case 0:
	case 1:
		query = query.Where("status = ?", options.statuses[0])
	default:
		query = query.Where("status IN ?", options.statuses)
	}

	msg := strings.ToLower(message)
	switch {
	case containsAny(msg, "已恢复", "resolved"):
		query = query.Where("status = ?", "resolved")
	case containsAny(msg, "已确认", "ack", "acknowledged"):
		query = query.Where("status = ?", "acknowledged")
	}

	switch {
	case containsAny(msg, "严重", "critical"):
		query = query.Where("severity = ?", "critical")
	case containsAny(msg, "警告", "warning"):
		query = query.Where("severity = ?", "warning")
	case containsAny(msg, "提醒", "info"):
		query = query.Where("severity = ?", "info")
	}

	if options.keyword != "" {
		query = query.Where(
			"rule_name LIKE ? OR content LIKE ? OR source LIKE ? OR category LIKE ?",
			"%"+options.keyword+"%", "%"+options.keyword+"%", "%"+options.keyword+"%", "%"+options.keyword+"%",
		)
	}

	var events []alert.AlertEvent
	orderBy := options.orderBy
	if strings.TrimSpace(orderBy) == "" {
		orderBy = "fired_at desc"
	}
	if err := query.Order(orderBy).Limit(5).Find(&events).Error; err != nil {
		return &toolContext{ToolName: "query_alert_events", Summary: "告警中心查询失败，当前无法读取告警数据。", Error: "alert events query failed"}
	}
	if len(events) == 0 {
		return &toolContext{ToolName: "query_alert_events", Summary: "当前没有匹配的告警。"}
	}

	lines := make([]string, 0, len(events))
	cards := make([]ResultCard, 0, len(events))
	for _, event := range events {
		line := fmt.Sprintf("- 规则：%s，级别：%s，状态：%s，来源：%s，时间：%s", event.RuleName, event.Severity, event.Status, defaultText(event.Source, "未知来源"), event.FiredAt.Format("2006-01-02 15:04"))
		if strings.TrimSpace(event.Content) != "" {
			line += "，内容：" + shorten(event.Content, 50)
		}
		lines = append(lines, line)
		cards = append(cards, ResultCard{
			Title:      event.RuleName,
			Subtitle:   fmt.Sprintf("级别：%s | 状态：%s", event.Severity, event.Status),
			Meta:       fmt.Sprintf("%s | %s", defaultText(event.Source, "未知来源"), event.FiredAt.Format("2006-01-02 15:04")),
			ToolName:   "alert.events",
			SourceType: "alert",
		})
	}

	return &toolContext{ToolName: "query_alert_events", Summary: "查询到的告警如下：\n" + strings.Join(lines, "\n"), Cards: cards}
}

func buildFocusedAlertEventContext(event alert.AlertEvent) *toolContext {
	lines := []string{
		fmt.Sprintf("当前告警：规则 %s，级别 %s，状态 %s，来源 %s。", event.RuleName, event.Severity, event.Status, defaultText(event.Source, "未知来源")),
	}
	if strings.TrimSpace(event.Content) != "" {
		lines = append(lines, "告警内容："+shorten(event.Content, 160))
	}
	if strings.TrimSpace(event.HandleType) != "" {
		lines = append(lines, "处理方式："+event.HandleType)
	}
	if strings.TrimSpace(event.HandleNote) != "" {
		lines = append(lines, "处理备注："+shorten(event.HandleNote, 120))
	}

	return &toolContext{
		ToolName: "query_alert_events",
		Summary:  strings.Join(lines, "\n"),
		Cards: []ResultCard{{
			Title:      event.RuleName,
			Subtitle:   fmt.Sprintf("级别：%s | 状态：%s", event.Severity, event.Status),
			Meta:       fmt.Sprintf("%s | %s", defaultText(event.Source, "未知来源"), event.FiredAt.Format("2006-01-02 15:04")),
			ToolName:   "alert.events",
			SourceType: "alert",
		}},
	}
}

func deriveAlertQueryOptions(message string) alertQueryOptions {
	msg := strings.ToLower(strings.TrimSpace(message))
	options := alertQueryOptions{
		orderBy: "fired_at desc",
	}

	switch {
	case containsAny(msg, "已恢复", "resolved"):
		options.statuses = []string{"resolved"}
	case containsAny(msg, "已确认", "ack", "acknowledged"):
		options.statuses = []string{"acknowledged"}
	case containsAny(msg, "已关闭", "closed"):
		options.statuses = []string{"closed"}
	case containsAny(msg, "异常", "未恢复", "处理中", "告警中", "firing"):
		options.statuses = []string{"firing", "acknowledged"}
	case containsAny(msg, "最新", "最近") && containsAny(msg, "告警", "报警") && containsAny(msg, "动作", "变化", "处理"):
		options.statuses = []string{"firing", "acknowledged"}
		options.orderBy = "updated_at desc"
	}

	if containsAny(msg, "动作", "变化", "处理") && !containsAny(msg, "已恢复", "resolved") {
		options.orderBy = "updated_at desc"
	}

	options.keyword = extractAlertKeyword(message)
	return options
}

func extractAlertKeyword(message string) string {
	replacer := strings.NewReplacer(
		"查看", " ",
		"查询", " ",
		"最新", " ",
		"最近", " ",
		"当前", " ",
		"异常", " ",
		"动作", " ",
		"变化", " ",
		"处理", " ",
		"状态", " ",
		"列表", " ",
		"告警", " ",
		"报警", " ",
		"事件", " ",
		"未恢复", " ",
		"已恢复", " ",
		"已确认", " ",
		"已关闭", " ",
	)
	cleaned := replacer.Replace(strings.ToLower(message))
	return extractPrimaryKeyword(cleaned, "严重", "警告", "提醒", "critical", "warning", "info", "firing", "resolved", "ack", "acknowledged", "closed")
}

func defaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
