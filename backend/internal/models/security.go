package models

import "time"

// ScanType 扫描类型
type ScanType string

const (
	ScanTypePort     ScanType = "port"      // 端口扫描（只做端口发现和服务识别）
	ScanTypeHostVuln ScanType = "host-vuln" // 主机漏洞扫描（SSH/数据库/Redis等）
	ScanTypeWeb      ScanType = "web"       // Web 漏洞扫描
)

// TaskStatus 任务状态
const (
	TaskStatusPending   = "pending"   // 待执行
	TaskStatusRunning   = "running"   // 执行中
	TaskStatusPaused    = "paused"    // 已暂停
	TaskStatusCancelled = "cancelled" // 已取消
	TaskStatusCompleted = "completed" // 已完成
	TaskStatusFailed    = "failed"    // 执行失败
)

// SecurityScanTask 安全扫描任务
type SecurityScanTask struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"size:100;not null"`
	TargetType      string     `json:"target_type" gorm:"size:20;not null"`     // cidr: 网段, ip_list: IP列表
	Target          string     `json:"target" gorm:"size:500"`                  // 10.1.0.0/24 或 10.1.0.1,10.1.0.2
	ScanType        string     `json:"scan_type" gorm:"size:20;default:'port'"` // port: 端口扫描, host-vuln: 主机漏洞, web: Web漏洞
	Status          string     `json:"status" gorm:"size:20;default:'pending'"` // pending, running, paused, cancelled, completed, failed
	Progress        int        `json:"progress" gorm:"default:0"`               // 进度百分比 0-100
	TotalIPs        int        `json:"total_ips" gorm:"default:0"`              // 总共需要扫描的 IP 数量
	ScannedIPs      int        `json:"scanned_ips" gorm:"default:0"`            // 已扫描的 IP 数量
	Message         string     `json:"message" gorm:"size:255"`                 // 当前状态信息
	NucleiVersion   string     `json:"nuclei_version" gorm:"size:50"`           // nuclei 版本
	TemplateVersion string     `json:"template_version" gorm:"size:50"`         // 模板版本
	HighRisk        int        `json:"high_risk" gorm:"default:0"`
	MediumRisk      int        `json:"medium_risk" gorm:"default:0"`
	LowRisk         int        `json:"low_risk" gorm:"default:0"`
	CurrentRunID    *uint      `json:"current_run_id,omitempty" gorm:"index"`
	LatestRunID     *uint      `json:"latest_run_id,omitempty" gorm:"index"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedBy       uint       `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (SecurityScanTask) TableName() string {
	return "security_scan_tasks"
}

// SecurityScanRun 安全扫描执行记录
type SecurityScanRun struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	TaskID         uint       `json:"task_id" gorm:"index;not null"`
	TaskName       string     `json:"task_name" gorm:"size:100;not null"`
	TargetType     string     `json:"target_type" gorm:"size:20;not null"`
	Target         string     `json:"target" gorm:"size:500"`
	ScanType       string     `json:"scan_type" gorm:"size:20;default:'port'"`
	Status         string     `json:"status" gorm:"size:20;default:'pending'"`
	Progress       int        `json:"progress" gorm:"default:0"`
	TotalTargets   int        `json:"total_targets" gorm:"default:0"`
	ScannedTargets int        `json:"scanned_targets" gorm:"default:0"`
	Message        string     `json:"message" gorm:"size:255"`
	HighRisk       int        `json:"high_risk" gorm:"default:0"`
	MediumRisk     int        `json:"medium_risk" gorm:"default:0"`
	LowRisk        int        `json:"low_risk" gorm:"default:0"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	TriggeredBy    uint       `json:"triggered_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (SecurityScanRun) TableName() string {
	return "security_scan_runs"
}

// SecurityScanTarget 扫描运行目标单元
type SecurityScanTarget struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	RunID            uint       `json:"run_id" gorm:"index;not null"`
	TaskID           uint       `json:"task_id" gorm:"index;not null"`
	ParentTargetID   *uint      `json:"parent_target_id,omitempty" gorm:"index"`
	TargetKind       string     `json:"target_kind" gorm:"size:20;not null"`
	NormalizedTarget string     `json:"normalized_target" gorm:"size:500;not null"`
	Host             string     `json:"host" gorm:"size:255"`
	Port             *int       `json:"port,omitempty"`
	Scheme           string     `json:"scheme" gorm:"size:20"`
	Path             string     `json:"path" gorm:"size:500"`
	ServiceName      string     `json:"service_name" gorm:"size:100"`
	ProductName      string     `json:"product_name" gorm:"size:100"`
	Version          string     `json:"version" gorm:"size:100"`
	Status           string     `json:"status" gorm:"size:20;default:'pending'"`
	DiscoverySource  string     `json:"discovery_source" gorm:"size:30"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	MetadataJSON     *string    `json:"metadata_json,omitempty" gorm:"type:json"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (SecurityScanTarget) TableName() string {
	return "security_scan_targets"
}

// SecurityScanEvidence 扫描原始证据
type SecurityScanEvidence struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	RunID           uint      `json:"run_id" gorm:"index;not null"`
	TaskID          uint      `json:"task_id" gorm:"index;not null"`
	TargetID        *uint     `json:"target_id,omitempty" gorm:"index"`
	EvidenceType    string    `json:"evidence_type" gorm:"size:30;not null"`
	SourceEngine    string    `json:"source_engine" gorm:"size:30"`
	Digest          string    `json:"digest" gorm:"size:64;index"`
	RequestExcerpt  string    `json:"request_excerpt" gorm:"type:mediumtext"`
	ResponseExcerpt string    `json:"response_excerpt" gorm:"type:mediumtext"`
	PayloadExcerpt  string    `json:"payload_excerpt" gorm:"type:mediumtext"`
	MetadataJSON    *string   `json:"metadata_json,omitempty" gorm:"type:json"`
	RawJSON         *string   `json:"raw_json,omitempty" gorm:"type:longtext"`
	StorageRef      string    `json:"storage_ref" gorm:"size:500"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (SecurityScanEvidence) TableName() string {
	return "security_scan_evidences"
}

// SecurityScanFindingOccurrence 扫描运行命中记录
type SecurityScanFindingOccurrence struct {
	ID                    uint       `json:"id" gorm:"primaryKey"`
	RunID                 uint       `json:"run_id" gorm:"index;not null"`
	TaskID                uint       `json:"task_id" gorm:"index;not null"`
	TargetID              *uint      `json:"target_id,omitempty" gorm:"index"`
	LegacyVulnerabilityID *uint      `json:"legacy_vulnerability_id,omitempty" gorm:"index"`
	FindingKey            string     `json:"finding_key" gorm:"size:255;index"`
	FindingFamily         string     `json:"finding_family" gorm:"size:20"`
	FindingSource         string     `json:"finding_source" gorm:"size:30;index"`
	Severity              string     `json:"severity" gorm:"size:20"`
	Confidence            string     `json:"confidence" gorm:"size:20"`
	MatchMode             string     `json:"match_mode" gorm:"size:30"`
	PrimaryCVEID          string     `json:"primary_cve_id" gorm:"size:50;index"`
	VulnDBID              *uint      `json:"vuln_db_id,omitempty"`
	Title                 string     `json:"title" gorm:"size:200"`
	Status                string     `json:"status" gorm:"size:20;default:'open';index"`
	VerificationStatus    string     `json:"verification_status" gorm:"size:20;default:'pending';index"`
	EvidenceCount         int        `json:"evidence_count" gorm:"default:0"`
	FirstSeenAt           *time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt            *time.Time `json:"last_seen_at,omitempty"`
	MetadataJSON          *string    `json:"metadata_json,omitempty" gorm:"type:json"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (SecurityScanFindingOccurrence) TableName() string {
	return "security_scan_finding_occurrences"
}

// SecurityAsset 发现的安全资产（主机/端口）
type SecurityAsset struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"index;not null"`
	IP          string    `json:"ip" gorm:"size:45;index"`
	Port        int       `json:"port" gorm:"index"`
	Protocol    string    `json:"protocol" gorm:"size:10"`
	ServiceName string    `json:"service_name" gorm:"size:50"`
	Version     string    `json:"version" gorm:"size:100"`
	OSInfo      string    `json:"os_info" gorm:"size:100"`
	Banner      string    `json:"banner" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SecurityAsset) TableName() string {
	return "security_assets"
}

// AssetImportance 资产重要性
const (
	AssetImportanceCritical = "critical"
	AssetImportanceHigh     = "high"
	AssetImportanceMedium   = "medium"
	AssetImportanceLow      = "low"
)

// AssetStatus 资产状态
const (
	AssetStatusOnline  = "online"
	AssetStatusOffline = "offline"
	AssetStatusUnknown = "unknown"
)

// AssetType 资产类型
const (
	AssetTypeServer   = "server"
	AssetTypeNetwork  = "network"
	AssetTypeWeb      = "web"
	AssetTypeDatabase = "database"
	AssetTypeOther    = "other"
)

// Asset 统一资产库（独立于扫描任务）
type Asset struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	IP          string    `json:"ip" gorm:"size:45;index;not null"`
	Port        int       `json:"port" gorm:"index"`
	Protocol    string    `json:"protocol" gorm:"size:10"`
	ServiceName string    `json:"service_name" gorm:"size:50"`
	Version     string    `json:"version" gorm:"size:100"`
	OSInfo      string    `json:"os_info" gorm:"size:100"`
	Banner      string    `json:"banner" gorm:"type:text"`
	AssetType   string    `json:"asset_type" gorm:"size:20"`               // server, network, web, database, other
	AssetGroup  string    `json:"asset_group" gorm:"size:50"`              // 资产分组
	Tags        string    `json:"tags" gorm:"size:200"`                    // 标签，逗号分隔
	Importance  string    `json:"importance" gorm:"size:20"`               // critical, high, medium, low
	Owner       string    `json:"owner" gorm:"size:50"`                    // 负责人
	Department  string    `json:"department" gorm:"size:50"`               // 部门
	Status      string    `json:"status" gorm:"size:20;default:'unknown'"` // online, offline, unknown
	FirstSeen   time.Time `json:"first_seen"`                              // 首次发现时间
	LastSeen    time.Time `json:"last_seen"`                               // 最后发现时间
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Asset) TableName() string {
	return "security_assets_new"
}

// AssetVulnCount 资产关联漏洞数（用于统计）
type AssetVulnCount struct {
	AssetID     uint `json:"asset_id" gorm:"primaryKey"`
	CriticalNum int  `json:"critical_num"`
	HighNum     int  `json:"high_num"`
	MediumNum   int  `json:"medium_num"`
	LowNum      int  `json:"low_num"`
	TotalNum    int  `json:"total_num"`
}

// SecurityVulnerability 安全漏洞
type SecurityVulnerability struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	TaskID      uint    `json:"task_id" gorm:"index;not null"`
	FirstTaskID *uint   `json:"first_task_id,omitempty" gorm:"index"`
	LastTaskID  *uint   `json:"last_task_id,omitempty" gorm:"index"`
	AssetID     uint    `json:"asset_id" gorm:"index"`
	IP          string  `json:"ip" gorm:"size:45;index"`
	Port        int     `json:"port" gorm:"index"`
	Protocol    string  `json:"protocol" gorm:"size:10"`          // TCP/UDP
	Severity    string  `json:"severity" gorm:"size:20;not null"` // critical, high, medium, low, info
	CVSSScore   float64 `json:"cvss_score"`
	CVSSVector  string  `json:"cvss_vector" gorm:"size:200"`

	// 漏洞标识
	CVEID   string `json:"cve_id" gorm:"size:50"`   // CVE-2021-12345
	CNVDID  string `json:"cnvd_id" gorm:"size:50"`  // CNVD-2021-12345
	CNNVDID string `json:"cnnvd_id" gorm:"size:50"` // CNNVD-2021-12345
	CNCVEID string `json:"cncve_id" gorm:"size:50"` // CNCVE-2021-12345

	// 漏洞信息
	Title         string `json:"title" gorm:"size:200"`
	Description   string `json:"description" gorm:"type:text"`
	VulnType      string `json:"vuln_type" gorm:"size:50"`        // 漏洞类型: rce, xss, sqli, etc.
	Solution      string `json:"solution" gorm:"type:text"`       // 修复建议
	MatchedOn     string `json:"matched_on" gorm:"size:255"`      // 命中依据
	ExploitPrereq string `json:"exploit_prereq" gorm:"type:text"` // 利用前提

	// 扫描信息
	Scanner       string `json:"scanner" gorm:"size:20"`                        // 扫描引擎: nuclei, nmap
	TemplateID    string `json:"template_id" gorm:"size:100"`                   // Nuclei 模板ID
	ScanMethod    string `json:"scan_method" gorm:"size:50"`                    // 扫描方式: 主动探测, 被动检测, 非授权扫描
	VulnURL       string `json:"vuln_url" gorm:"size:500"`                      // 漏洞地址
	FindingSource string `json:"finding_source,omitempty" gorm:"size:30;index"` // web-template, web-rule, host-template, host-version-match, asset-inventory
	FindingFamily string `json:"finding_family,omitempty" gorm:"size:20;index"` // vulnerability, inventory
	Confidence    string `json:"confidence,omitempty" gorm:"size:20;index"`     // high, medium, low
	PrimaryCVEID  string `json:"primary_cve_id,omitempty" gorm:"size:50"`       // 主 CVE，用于稳定展示和关联
	VulnDBID      *uint  `json:"vuln_db_id,omitempty" gorm:"index"`             // 关联漏洞库主记录
	MatchMode     string `json:"match_mode,omitempty" gorm:"size:30"`           // template, rule, version-range, fuzzy-product, inventory
	SourceVulnID  *uint  `json:"source_vuln_id,omitempty" gorm:"index"`         // 派生正式漏洞时关联的候选记录 ID

	// 验证信息
	Payload      string `json:"payload" gorm:"type:text"`      // 探测使用的 payload
	Request      string `json:"request" gorm:"type:text"`      // HTTP 请求片段
	Response     string `json:"response" gorm:"type:text"`     // 响应片段
	ReferenceURL string `json:"reference_url" gorm:"size:500"` // 参考链接

	// 处置信息
	Status             string                  `json:"status" gorm:"size:20;default:'open'"`     // open, acknowledged, fixed, ignored
	Priority           string                  `json:"priority" gorm:"size:20"`                  // 处置优先级: high, medium, low
	FalsePositive      bool                    `json:"false_positive" gorm:"default:false"`      // 是否误报
	ConfirmedVulnID    *uint                   `json:"confirmed_vuln_id,omitempty" gorm:"index"` // 候选确认后派生出的正式漏洞 ID
	ReviewStatus       string                  `json:"-" gorm:"size:20;default:'pending'"`       // pending, needs-test, confirmed, rejected
	ReviewNote         string                  `json:"-" gorm:"type:text"`
	ReviewedBy         *uint                   `json:"-" gorm:"index"`
	ReviewedAt         *time.Time              `json:"-"`
	VerificationStatus string                  `json:"verification_status,omitempty" gorm:"-"`
	VerificationNote   string                  `json:"verification_note,omitempty" gorm:"-"`
	VerifiedBy         *uint                   `json:"verified_by,omitempty" gorm:"-"`
	VerifiedAt         *time.Time              `json:"verified_at,omitempty" gorm:"-"`
	CandidateTier      string                  `json:"-" gorm:"-"`
	RiskCategory       string                  `json:"risk_category,omitempty" gorm:"-"`
	DisplayGroup       string                  `json:"display_group,omitempty" gorm:"-"`
	Knowledge          *VulnerabilityKnowledge `json:"knowledge,omitempty" gorm:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SecurityVulnerability) TableName() string {
	return "security_vulnerabilities"
}

type VulnerabilityKnowledge struct {
	ID           uint    `json:"id"`
	Title        string  `json:"title"`
	Severity     string  `json:"severity"`
	CVSSScore    float64 `json:"cvss_score"`
	CNVDID       string  `json:"cnvd_id"`
	CNNVDID      string  `json:"cnnvd_id"`
	CNCVEID      string  `json:"cncve_id"`
	HasReference bool    `json:"has_reference"`
}

// ScanStatistics 扫描统计（用于仪表盘）
type ScanStatistics struct {
	TotalTasks           int64 `json:"total_tasks"`
	RunningTasks         int64 `json:"running_tasks"`
	CompletedTasks       int64 `json:"completed_tasks"`
	TotalAssets          int64 `json:"total_assets"`
	TotalVulnerabilities int64 `json:"total_vulnerabilities"`
	HighRiskCount        int64 `json:"high_risk_count"`
	MediumRiskCount      int64 `json:"medium_risk_count"`
	LowRiskCount         int64 `json:"low_risk_count"`
}

// VulnTicketStatus 漏洞工单状态
const (
	VulnTicketStatusOpen       = "open"       // 待处理
	VulnTicketStatusProcessing = "processing" // 处理中
	VulnTicketStatusFixed      = "fixed"      // 已修复
	VulnTicketStatusClosed     = "closed"     // 已关闭
	VulnTicketStatusRejected   = "rejected"   // 已驳回
)

// VulnTicketPriority 漏洞工单优先级
const (
	VulnTicketPriorityHigh   = "high"
	VulnTicketPriorityMedium = "medium"
	VulnTicketPriorityLow    = "low"
)

// VulnTicket 漏洞工单
type VulnTicket struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	VulnID        uint       `json:"vuln_id" gorm:"index;not null"`        // 关联漏洞ID
	VulnTitle     string     `json:"vuln_title" gorm:"size:200"`           // 漏洞标题（冗余存储）
	Assignee      uint       `json:"assignee" gorm:"index"`                // 指派人ID
	AssigneeName  string     `json:"assignee_name" gorm:"size:50"`         // 指派人姓名
	Department    string     `json:"department" gorm:"size:50"`            // 部门
	Status        string     `json:"status" gorm:"size:20;default:'open'"` // open, processing, fixed, closed, rejected
	Priority      string     `json:"priority" gorm:"size:20"`              // high, medium, low
	DueDate       *time.Time `json:"due_date"`                             // 截止日期
	Notes         string     `json:"notes" gorm:"type:text"`               // 备注
	Comments      string     `json:"comments" gorm:"type:text"`            // 评论/处理记录
	CreatedBy     uint       `json:"created_by" gorm:"index"`              // 创建人
	CreatedByName string     `json:"created_by_name" gorm:"size:50"`       // 创建人姓名
	ResolvedAt    *time.Time `json:"resolved_at"`                          // 解决时间
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (VulnTicket) TableName() string {
	return "security_vuln_tickets"
}

// TicketHistory 工单处理历史记录
type TicketHistory struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TicketID     uint      `json:"ticket_id" gorm:"index;not null"`
	Action       string    `json:"action" gorm:"size:50;not null"` // created, assigned, status_changed, closed, commented
	OldStatus    string    `json:"old_status" gorm:"size:20"`
	NewStatus    string    `json:"new_status" gorm:"size:20"`
	OldAssignee  string    `json:"old_assignee" gorm:"size:50"`
	NewAssignee  string    `json:"new_assignee" gorm:"size:50"`
	Comment      string    `json:"comment" gorm:"type:text"`
	OperatorID   uint      `json:"operator_id"`                  // 操作人ID
	OperatorName string    `json:"operator_name" gorm:"size:50"` // 操作人姓名
	CreatedAt    time.Time `json:"created_at"`
}

func (TicketHistory) TableName() string {
	return "security_ticket_history"
}
