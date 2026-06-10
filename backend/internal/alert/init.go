// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package alert

import (
	"log"

	"github.com/jenvenson/ops-platform/internal/database"
)

// Init 初始化告警模块数据库表
func Init() error {
	if err := database.DB.AutoMigrate(
		&AlertRule{},
		&AlertContact{},
		&AlertNotifyGroup{},
		&NotifyChannel{},
		&AlertEvent{},
		&AlertEventLog{},
		&AlertTemplate{},
	); err != nil {
		log.Printf("Warning: alert migration warning (app may still work): %v", err)
	}

	// 初始化默认模板（如果不存在）
	seedDefaultTemplates()

	return nil
}

// seedDefaultTemplates 初始化内置默认告警模板
func seedDefaultTemplates() {
	defaults := []AlertTemplate{
		// ===== 钉钉模板 =====
		{
			Name:  "钉钉-告警触发",
			Type:  "dingtalk",
			Scene: "firing",
			TitleTpl: "{{.Emoji}} 【告警】{{.RuleName}}",
			ContentTpl: `### {{.Emoji}} 【告警】{{.RuleName}}

> **规则名称**：{{.RuleName}}

> **告警内容**：{{.Content}}

> **来源**：{{.Source}}

> **级别**：{{.SeverityLabel}}

> **分类**：{{.CategoryLabel}}

> **状态**：{{.StatusLabel}}

> **触发时间**：{{.Time}}`,
			IsDefault:   true,
			Enabled:     true,
			Description: "钉钉机器人默认告警触发模板",
		},
		{
			Name:  "钉钉-告警恢复",
			Type:  "dingtalk",
			Scene: "resolved",
			TitleTpl: "{{.Emoji}} 【恢复】{{.RuleName}}",
			ContentTpl: `### {{.Emoji}} 【恢复】{{.RuleName}}

> **规则名称**：{{.RuleName}}

> **告警内容**：{{.Content}}

> **来源**：{{.Source}}

> **级别**：{{.SeverityLabel}}

> **分类**：{{.CategoryLabel}}

> **状态**：{{.StatusLabel}}

> **恢复时间**：{{.Time}}`,
			IsDefault:   true,
			Enabled:     true,
			Description: "钉钉机器人默认告警恢复模板",
		},
		// ===== 企微模板 =====
		{
			Name:  "企微-告警触发",
			Type:  "wechat",
			Scene: "firing",
			TitleTpl: "【告警】{{.RuleName}}",
			ContentTpl: `## 【告警】{{.RuleName}}
> **规则名称**：{{.RuleName}}
> **告警内容**：{{.Content}}
> **来源**：{{.Source}}
> **级别**：{{.SeverityLabel}}
> **分类**：{{.CategoryLabel}}
> **状态**：{{.StatusLabel}}
> **触发时间**：{{.Time}}`,
			IsDefault:   true,
			Enabled:     true,
			Description: "企业微信默认告警触发模板",
		},
		{
			Name:  "企微-告警恢复",
			Type:  "wechat",
			Scene: "resolved",
			TitleTpl: "【恢复】{{.RuleName}}",
			ContentTpl: `## 【恢复】{{.RuleName}}
> **规则名称**：{{.RuleName}}
> **告警内容**：{{.Content}}
> **来源**：{{.Source}}
> **级别**：{{.SeverityLabel}}
> **分类**：{{.CategoryLabel}}
> **状态**：{{.StatusLabel}}
> **恢复时间**：{{.Time}}`,
			IsDefault:   true,
			Enabled:     true,
			Description: "企业微信默认告警恢复模板",
		},
		// ===== 邮件模板 =====
		{
			Name:  "邮件-告警触发",
			Type:  "email",
			Scene: "firing",
			TitleTpl: "[{{.SeverityLabel}}] 【告警】{{.RuleName}}",
			ContentTpl: `<table style="width:100%;border-collapse:collapse;font-size:14px;">
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;width:100px;font-weight:bold;">规则名称</td><td style="padding:12px 8px;">{{.RuleName}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">告警内容</td><td style="padding:12px 8px;">{{.Content}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">来源</td><td style="padding:12px 8px;font-family:monospace;">{{.Source}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">级别</td><td style="padding:12px 8px;">{{.SeverityLabel}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">分类</td><td style="padding:12px 8px;">{{.CategoryLabel}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">状态</td><td style="padding:12px 8px;">{{.StatusLabel}}</td></tr>
<tr><td style="padding:12px 8px;color:#666;font-weight:bold;">触发时间</td><td style="padding:12px 8px;">{{.Time}}</td></tr>
</table>`,
			IsDefault:   true,
			Enabled:     true,
			Description: "邮件默认告警触发模板",
		},
		{
			Name:  "邮件-告警恢复",
			Type:  "email",
			Scene: "resolved",
			TitleTpl: "[{{.SeverityLabel}}] 【恢复】{{.RuleName}}",
			ContentTpl: `<table style="width:100%;border-collapse:collapse;font-size:14px;">
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;width:100px;font-weight:bold;">规则名称</td><td style="padding:12px 8px;">{{.RuleName}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">告警内容</td><td style="padding:12px 8px;">{{.Content}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">来源</td><td style="padding:12px 8px;font-family:monospace;">{{.Source}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">级别</td><td style="padding:12px 8px;">{{.SeverityLabel}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">分类</td><td style="padding:12px 8px;">{{.CategoryLabel}}</td></tr>
<tr style="border-bottom:1px solid #f0f0f0;"><td style="padding:12px 8px;color:#666;font-weight:bold;">状态</td><td style="padding:12px 8px;">{{.StatusLabel}}</td></tr>
<tr><td style="padding:12px 8px;color:#666;font-weight:bold;">恢复时间</td><td style="padding:12px 8px;">{{.Time}}</td></tr>
</table>`,
			IsDefault:   true,
			Enabled:     true,
			Description: "邮件默认告警恢复模板",
		},
	}

	for _, tpl := range defaults {
		var count int64
		database.DB.Model(&AlertTemplate{}).Where("type = ? AND scene = ? AND is_default = ?", tpl.Type, tpl.Scene, true).Count(&count)
		if count == 0 {
			database.DB.Create(&tpl)
		}
	}
}