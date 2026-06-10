// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

// CVEDetail CVE 详细信息（用于漏洞分析和跟踪）
type CVEDetail struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	CVEID            string    `json:"cve_id" gorm:"size:50;uniqueIndex;not null"` // CVE-2021-12345
	Description      string    `json:"description" gorm:"type:text"`               // CVE 描述
	CVSSScore        float64   `json:"cvss_score"`                                   // CVSS 基础评分
	CVSSVector       string    `json:"cvss_vector" gorm:"size:200"`                  // CVSS 向量字符串
	AttackVector     string    `json:"attack_vector" gorm:"size:50"`                  // Network, Adjacent, Local, Physical
	AttackComplexity string    `json:"attack_complexity" gorm:"size:50"`             // Low, High
	PrivilegesRequired string  `json:"privileges_required" gorm:"size:50"`          // None, Low, High
	UserInteraction  string    `json:"user_interaction" gorm:"size:50"`             // None, Required
	Scope            string   `json:"scope" gorm:"size:50"`                         // Unchanged, Changed
	Confidentiality  string   `json:"confidentiality" gorm:"size:50"`                // None, Low, High
	Integrity        string  `json:"integrity" gorm:"size:50"`                     // None, Low, High
	Availability     string   `json:"availability" gorm:"size:50"`                  // None, Low, High

	// 漏洞分类
	CWEID            string   `json:"cwe_id" gorm:"size:20"`                       // CWE-79
	VulnerabilityType string  `json:"vulnerability_type" gorm:"size:50"`           // XSS, SQLi, RCE, etc.

	// 修复信息
	Severity         string   `json:"severity" gorm:"size:20"`                     // CRITICAL, HIGH, MEDIUM, LOW
	Solution         string   `json:"solution" gorm:"type:text"`                     // 修复建议
	Workaround       string   `json:"workaround" gorm:"type:text"`                   // 临时解决方案

	// 参考信息
	References       string   `json:"references" gorm:"type:text"`                  // JSON 数组存储的参考链接
	PatchInfo        string   `json:"patch_info" gorm:"size:255"`                   // 补丁信息

	// 利用信息
	ExploitAvailable  bool    `json:"exploit_available"`                           // 是否有公开利用代码
	ExploitInWild     bool    `json:"exploit_in_wild"`                             // 是否有野外利用
	ExploitFrameworks string  `json:"exploit_frameworks" gorm:"size:100"`          // Metasploit, Cobalt Strike, etc.

	// 时间信息
	PublishedAt      *time.Time `json:"published_at"`                              // CVE 发布时间
	LastModifiedAt   *time.Time `json:"last_modified_at"`                          // 最后修改时间

	// 状态跟踪
	Status           string   `json:"status" gorm:"size:20;default:'active'"`       // active, deprecated, reserved
	Notes            string   `json:"notes" gorm:"type:text"`                       // 内部备注

	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (CVEDetail) TableName() string {
	return "cve_details"
}

// CVESeverity CVSS 严重等级
type CVESeverity string

const (
	SeverityCritical CVESeverity = "CRITICAL"
	SeverityHigh     CVESeverity = "HIGH"
	SeverityMedium   CVESeverity = "MEDIUM"
	SeverityLow      CVESeverity = "LOW"
	SeverityNone     CVESeverity = "NONE"
)

// AttackVectorType 攻击向量类型
type AttackVectorType string

const (
	AttackVectorNetwork   AttackVectorType = "NETWORK"
	AttackVectorAdjacent AttackVectorType = "ADJACENT_NETWORK"
	AttackVectorLocal    AttackVectorType = "LOCAL"
	AttackVectorPhysical AttackVectorType = "PHYSICAL"
)

// VulnerabilityCategory 漏洞分类
type VulnerabilityCategory string

const (
	VulnCategoryXSS          VulnerabilityCategory = "XSS"
	VulnCategorySQLi         VulnerabilityCategory = "SQL_INJECTION"
	VulnCategoryRCE          VulnerabilityCategory = "RCE"
	VulnCategorySSRF         VulnerabilityCategory = "SSRF"
	VulnCategoryXXE          VulnerabilityCategory = "XXE"
	VulnCategoryIDOR         VulnerabilityCategory = "IDOR"
	VulnCategoryInfoLeak     VulnerabilityCategory = "INFORMATION_DISCLOSURE"
	VulnCategoryAuthBypass   VulnerabilityCategory = "AUTH_BYPASS"
	VulnCategoryCSRF         VulnerabilityCategory = "CSRF"
	VulnCategoryOther       VulnerabilityCategory = "OTHER"
)

// CVEComparisonResult CVE 对比结果
type CVEComparisonResult struct {
	CVEID             string   `json:"cve_id"`
	DetectedSeverity  string   `json:"detected_severity"`    // 从 Nuclei 检测到的严重程度
	OfficialSeverity  string   `json:"official_severity"`    // NVD 官方严重程度
	SeverityMatch     bool     `json:"severity_match"`       // 是否匹配
	CVSSDiff          float64  `json:"cvss_diff"`           // CVSS 分数差异
	References        []string `json:"references"`          // 相关参考
	SuggestedAction   string   `json:"suggested_action"`    // 建议操作
}