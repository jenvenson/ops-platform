// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package alert

import "time"

// AlertRule 告警规则（从 Grafana 同步 + 本地分级配置）
type AlertRule struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	GrafanaUID     string     `json:"grafana_uid" gorm:"size:100;uniqueIndex"` // Grafana 规则 UID
	Name           string     `json:"name" gorm:"size:200;not null"`           // 规则名称
	RuleGroup      string     `json:"rule_group" gorm:"size:100"`              // 规则组名
	FolderTitle    string     `json:"folder_title" gorm:"size:100"`            // Grafana 文件夹
	Severity       string     `json:"severity" gorm:"size:20;default:'warning';index"` // critical/warning/info
	Category       string     `json:"category" gorm:"size:50;index"`           // disk/memory/cpu/instance 等
	Description    string     `json:"description" gorm:"type:text"`            // 规则描述
	Expression     string     `json:"expression" gorm:"type:text"`             // 告警表达式
	Condition      string     `json:"condition" gorm:"type:text"`              // 触发条件（可读描述）
	Enabled        bool       `json:"enabled" gorm:"default:true;index"`       // 是否启用
	AlertGroupID   *uint      `json:"alert_group_id" gorm:"index"`            // 关联报警组
	NotifyChannels string     `json:"notify_channels" gorm:"size:500"`         // 通知渠道 ID，逗号分隔
	SyncedAt       *time.Time `json:"synced_at"`                               // 最后同步时间
	GrafanaState   string     `json:"grafana_state" gorm:"size:20"`            // Grafana 中的状态
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联
	Group *AlertNotifyGroup `json:"group,omitempty" gorm:"foreignKey:AlertGroupID"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// AlertContact 联系人
type AlertContact struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Name     string `json:"name" gorm:"size:50;not null"`
	Email    string `json:"email" gorm:"size:100"`
	Phone    string `json:"phone" gorm:"size:20"`
	DingTalk string `json:"dingtalk" gorm:"size:100"` // 钉钉 UserID 或手机号
	WeChat   string `json:"wechat" gorm:"size:100"`   // 企微 UserID

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联
	Groups []AlertNotifyGroup `json:"groups,omitempty" gorm:"many2many:alert_group_contacts;"`
}

func (AlertContact) TableName() string {
	return "alert_contacts"
}

// AlertNotifyGroup 报警组
type AlertNotifyGroup struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:100;uniqueIndex:name,deleted_at;not null"`
	Description string `json:"description" gorm:"size:500"`
	Enabled     bool   `json:"enabled" gorm:"default:true"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联
	Contacts []AlertContact `json:"contacts,omitempty" gorm:"many2many:alert_group_contacts;"`
}

func (AlertNotifyGroup) TableName() string {
	return "alert_notify_groups"
}

// NotifyChannel 通知渠道
type NotifyChannel struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Type        string `json:"type" gorm:"size:20;not null;index"` // dingtalk/wechat/email
	WebhookURL  string `json:"webhook_url" gorm:"size:500"`       // 钉钉/企微机器人 Webhook
	Secret      string `json:"secret" gorm:"size:200"`            // 钉钉签名密钥
	SMTPHost    string `json:"smtp_host" gorm:"size:100"`         // 邮件 SMTP 服务器
	SMTPPort    int    `json:"smtp_port"`                         // SMTP 端口
	SMTPUser    string `json:"smtp_user" gorm:"size:100"`         // SMTP 用户名
	SMTPPass    string `json:"smtp_pass" gorm:"size:200"`         // SMTP 密码
	EmailFrom   string `json:"email_from" gorm:"size:100"`        // 发件人地址
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`
	Description string `json:"description" gorm:"size:500"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (NotifyChannel) TableName() string {
	return "notify_channels"
}

// AlertEvent 告警事件
type AlertEvent struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	RuleID       *uint      `json:"rule_id" gorm:"index"`                              // 关联规则
	RuleName     string     `json:"rule_name" gorm:"size:200;not null"`                // 规则名称（冗余）
	Severity     string     `json:"severity" gorm:"size:20;not null;index"`            // critical/warning/info
	Category     string     `json:"category" gorm:"size:50;index"`                     // disk/memory/cpu/instance
	Content      string     `json:"content" gorm:"type:text;not null"`                 // 告警内容
	Source       string     `json:"source" gorm:"size:200"`                            // 告警来源（主机/服务）
	Status       string     `json:"status" gorm:"size:20;default:'firing';index"`      // firing/acknowledged/resolved/closed
	FiredAt      time.Time  `json:"fired_at" gorm:"not null;index"`                    // 触发时间
	AckedAt      *time.Time `json:"acked_at"`                                          // 确认时间
	ResolvedAt   *time.Time `json:"resolved_at"`                                       // 恢复时间
	ClosedAt     *time.Time `json:"closed_at"`                                         // 关闭时间
	AckedBy      string     `json:"acked_by" gorm:"size:50"`                           // 确认人
	ClosedBy     string     `json:"closed_by" gorm:"size:50"`                          // 关闭人
	HandleType   string     `json:"handle_type" gorm:"size:20"`                        // ticket/auto/manual 处理方式
	HandleNote   string     `json:"handle_note" gorm:"type:text"`                      // 处理备注
	Labels       string     `json:"labels" gorm:"type:text"`                           // JSON 标签
	Fingerprint  string     `json:"fingerprint" gorm:"size:100;index"`                 // 去重指纹
	NotifyStatus string     `json:"notify_status" gorm:"size:20;default:'pending'"`    // pending/sent/failed
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联
	Rule *AlertRule      `json:"rule,omitempty" gorm:"foreignKey:RuleID"`
	Logs []AlertEventLog `json:"logs,omitempty" gorm:"foreignKey:EventID"`
}

func (AlertEvent) TableName() string {
	return "alert_events"
}

// AlertEventLog 告警事件生命周期日志
type AlertEventLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	EventID   uint      `json:"event_id" gorm:"not null;index"`
	Action    string    `json:"action" gorm:"size:30;not null"` // created/acked/resolved/closed/notified/note
	Operator  string    `json:"operator" gorm:"size:50"`        // 操作人
	Content   string    `json:"content" gorm:"type:text"`       // 日志内容
	CreatedAt time.Time `json:"created_at"`
}

func (AlertEventLog) TableName() string {
	return "alert_event_logs"
}

// AlertTemplate 告警通知模板
// 支持通过变量占位符自定义通知内容
//
// 可用变量（使用 {{.变量名}} 语法）：
//   {{.RuleName}}     - 规则名称
//   {{.Content}}      - 告警内容
//   {{.Source}}       - 告警来源（IP:Port）
//   {{.Severity}}     - 告警级别（英文: critical/warning/info）
//   {{.SeverityLabel}} - 告警级别（中文: 严重/警告/提醒）
//   {{.Status}}       - 告警状态（英文: firing/resolved）
//   {{.StatusLabel}}  - 告警状态（中文: 告警中/已恢复）
//   {{.Category}}     - 告警分类（英文: disk/memory/cpu 等）
//   {{.CategoryLabel}} - 告警分类（中文: 磁盘/内存/CPU 等）
//   {{.Time}}         - 触发/恢复时间
//   {{.Emoji}}        - 级别对应的 Emoji 图标
type AlertTemplate struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:100;not null"`                          // 模板名称
	Type        string `json:"type" gorm:"size:20;not null;index"`                     // dingtalk/wechat/email
	Scene       string `json:"scene" gorm:"size:20;not null;default:'firing';index"`   // firing=告警触发 / resolved=告警恢复 / test=测试
	TitleTpl    string `json:"title_tpl" gorm:"type:text"`                             // 标题模板
	ContentTpl  string `json:"content_tpl" gorm:"type:text;not null"`                  // 内容模板
	IsDefault   bool   `json:"is_default" gorm:"default:false;index"`                  // 是否为默认模板
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`                      // 是否启用
	Description string `json:"description" gorm:"size:500"`                            // 模板描述

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (AlertTemplate) TableName() string {
	return "alert_templates"
}