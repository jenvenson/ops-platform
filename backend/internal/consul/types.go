package consul

import (
	"time"

	"gorm.io/gorm"
)

// ConsulConfig Consul 连接配置
type ConsulConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`       // 配置名称
	Address   string         `gorm:"size:255;not null" json:"address"`    // Consul 地址，如 http://10.99.99.98:8500
	Datacenter string        `gorm:"size:50;default:dc1" json:"datacenter"` // 数据中心
	Token     string         `gorm:"size:255" json:"token,omitempty"`     // ACL Token（可选）
	Username  string         `gorm:"size:100" json:"username,omitempty"`  // 基本认证用户名（可选）
	Password  string         `gorm:"size:255" json:"password,omitempty"`  // 基本认证密码（可选）
	IsDefault bool           `gorm:"default:false" json:"is_default"`     // 是否默认配置
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// KVItem KV 键值项
type KVItem struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Flags       uint64 `json:"flags,omitempty"`
	CreateIndex uint64 `json:"create_index,omitempty"`
	ModifyIndex uint64 `json:"modify_index,omitempty"`
}

// KVNode KV 树节点
type KVNode struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	IsDir    bool      `json:"is_dir"`
	Children []KVNode  `json:"children,omitempty"`
}

// ReplaceRule 替换规则
type ReplaceRule struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`         // 规则名称
	Description string         `gorm:"size:500" json:"description"`            // 规则描述
	SourceType  string         `gorm:"size:20;default:text" json:"source_type"` // text: 文本替换, regex: 正则替换
	OldValue    string         `gorm:"size:500;not null" json:"old_value"`     // 原值/正则表达式
	NewValue    string         `gorm:"size:500;not null" json:"new_value"`     // 新值
	Enabled     bool           `gorm:"default:true" json:"enabled"`            // 是否启用
	SortOrder   int            `gorm:"default:0" json:"sort_order"`            // 执行顺序
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// CopyOperation 复制操作记录
type CopyOperation struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ConfigID     uint           `gorm:"not null;index" json:"config_id"`         // Consul 配置 ID
	SourceKey    string         `gorm:"size:500;not null" json:"source_key"`     // 源键
	TargetKey    string         `gorm:"size:500;not null" json:"target_key"`     // 目标键
	RulesApplied string         `gorm:"type:text" json:"rules_applied"`          // 应用的规则 JSON
	Status       string         `gorm:"size:20;default:pending" json:"status"`   // pending, success, failed
	Message      string         `gorm:"type:text" json:"message"`                // 操作消息
	Operator     string         `gorm:"size:100" json:"operator"`                // 操作人
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// CopyRequest 复制请求
type CopyRequest struct {
	ConfigID                uint          `json:"config_id" binding:"required"`
	SourceKey               string        `json:"source_key" binding:"required"`
	TargetKey               string        `json:"target_key" binding:"required"`
	TagReplacements         []ReplacePair `json:"tag_replacements"`
	ServerReplacements      []ReplacePair `json:"server_replacements"`
	BranchReplacements      []ReplacePair `json:"branch_replacements"`
	SubmoduleBranchReplacements []ReplacePair `json:"submodule_branch_replacements"`
	ReplaceRules            []RuleItem    `json:"replace_rules"`
	Recursive               bool          `json:"recursive"`
}

// RuleItem 规则项
type RuleItem struct {
	Type     string `json:"type"`      // text, regex
	OldValue string `json:"old_value"` // 原值/正则表达式
	NewValue string `json:"new_value"` // 新值
}

// ReplacePair 替换对（原模式 → 新模式）
type ReplacePair struct {
	OldPattern string `json:"old_pattern"` // 原模式
	NewPattern string `json:"new_pattern"` // 新模式
}

// CopyResult 复制结果
type CopyResult struct {
	Success     int      `json:"success"`      // 成功数量
	Failed      int      `json:"failed"`       // 失败数量
	Total       int      `json:"total"`        // 总数量
	CopiedKeys  []string `json:"copied_keys"`  // 已复制的键
	FailedKeys  []string `json:"failed_keys"`  // 失败的键
	Errors      []string `json:"errors"`       // 错误信息
}

// BatchCopyRequest 批量复制请求（参考脚本功能）
type BatchCopyRequest struct {
	ConfigID                    uint          `json:"config_id"`
	SourcePrefix                string        `json:"source_prefix" binding:"required"`
	TargetPrefix                string        `json:"target_prefix" binding:"required"`
	Recursive                   bool          `json:"recursive"`
	ReplaceRules                []RuleItem    `json:"replace_rules"`
	TagReplacements             []ReplacePair `json:"tag_replacements"`
	ServerReplacements          []ReplacePair `json:"server_replacements"`
	BranchReplacements          []ReplacePair `json:"branch_replacements"`
	SubmoduleBranchReplacements []ReplacePair `json:"submodule_branch_replacements"`
}

// BatchCopyResult 批量复制结果
type BatchCopyResult struct {
	Success     int      `json:"success"`       // 成功数量
	Failed      int      `json:"failed"`        // 失败数量
	Total       int      `json:"total"`         // 总数量
	CopiedKeys  []string `json:"copied_keys"`   // 已复制的键
	FailedKeys  []string `json:"failed_keys"`   // 失败的键
	Errors      []string `json:"errors"`        // 错误信息
	ElapsedTime string   `json:"elapsed_time"`  // 耗时
}

// BatchCopyAllProjectsRequest 批量复制所有项目请求
type BatchCopyAllProjectsRequest struct {
	ConfigID                    uint          `json:"config_id"`
	SourceSuffix                string        `json:"source_suffix" binding:"required"`
	TargetSuffix                string        `json:"target_suffix" binding:"required"`
	ReplaceInPlace              bool          `json:"replace_in_place"`
	Projects                    []string      `json:"projects"`
	ReplaceRules                []RuleItem    `json:"replace_rules"`
	TagReplacements             []ReplacePair `json:"tag_replacements"`
	ServerReplacements          []ReplacePair `json:"server_replacements"`
	BranchReplacements          []ReplacePair `json:"branch_replacements"`
	SubmoduleBranchReplacements []ReplacePair `json:"submodule_branch_replacements"`
	Recursive                   bool          `json:"recursive"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	ConfigID uint     `json:"config_id"`
	Keys     []string `json:"keys" binding:"required"`
}

// BatchDeleteResult 批量删除结果
type BatchDeleteResult struct {
	Deleted    int      `json:"deleted"`
	Failed     int      `json:"failed"`
	Total      int      `json:"total"`
	DeletedKeys []string `json:"deleted_keys"`
	FailedKeys  []string `json:"failed_keys"`
	Errors      []string `json:"errors"`
}
