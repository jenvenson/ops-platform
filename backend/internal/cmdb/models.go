package cmdb

import "time"

type Project struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex:name,deleted_at;size:100;not null"`
	Code        string     `json:"code" gorm:"size:50;not null"` // 项目编号
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (Project) TableName() string {
	return "projects"
}

type Environment struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex:name,deleted_at;size:100;not null"`
	Type        string     `json:"type" gorm:"size:10;default:'dev';not null"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func (Environment) TableName() string {
	return "environments"
}

type Server struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Hostname string `json:"hostname" gorm:"size:100;not null"`
	IP       string `json:"ip" gorm:"uniqueIndex:ip,deleted_at;size:45;not null"`
	OS       string `json:"os" gorm:"size:50"`
	Arch     string `json:"arch" gorm:"size:20"`
	Status   string `json:"status" gorm:"size:10;default:'offline';not null"`
	SSHPort  int    `json:"ssh_port" gorm:"default:22"`
	EnvID    uint   `json:"env_id" gorm:"index"`
	EnvIDs   string `json:"env_ids" gorm:"size:200"` // 多选环境，存储逗号分隔的ID

	// Agent 状态字段
	AgentRegistered bool       `json:"agent_registered" gorm:"default:false"` // 是否已注册 Agent
	AgentVersion    string     `json:"agent_version" gorm:"size:20"`          // Agent 版本
	AgentID         string     `json:"agent_id" gorm:"size:64;index"`         // Agent 唯一标识
	LastHeartbeat   *time.Time `json:"last_heartbeat" gorm:"index"`           // 最后心跳时间
	CPUUsage        float64    `json:"cpu_usage" gorm:"default:0"`            // CPU 使用率 %
	MemoryUsage     float64    `json:"memory_usage" gorm:"default:0"`         // 内存使用率 %
	DiskUsage       float64    `json:"disk_usage" gorm:"default:0"`           // 磁盘使用率 %
	LoadAvg         string     `json:"load_avg" gorm:"size:50"`               // 系统负载

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联
	Projects []Project `json:"projects,omitempty" gorm:"many2many:server_projects;"`
}

func (Server) TableName() string {
	return "servers"
}

type Application struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	Name              string     `json:"name" gorm:"index:name;size:100;not null"`
	CodeRepo          string     `json:"code_repo"`
	DeployPath        string     `json:"deploy_path"`
	JenkinsJob        string     `json:"jenkins_job"`
	JenkinsArchiveJob string     `json:"jenkins_archive_job"`
	EnvID             uint       `json:"env_id" gorm:"index"`
	ProjectID         uint       `json:"project_id" gorm:"index"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联
	Environment *Environment `json:"environment,omitempty" gorm:"foreignKey:EnvID"`
	Project     *Project     `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

func (Application) TableName() string {
	return "applications"
}

type ServerApp struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	ServerID   uint       `json:"server_id" gorm:"not null;uniqueIndex:server_app"`
	AppID      uint       `json:"app_id" gorm:"not null;index"`
	Version    string     `json:"version"`
	DeployedAt *time.Time `json:"deployed_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 关联
	Server *Server      `json:"server,omitempty" gorm:"foreignKey:ServerID"`
	App    *Application `json:"app,omitempty" gorm:"foreignKey:AppID"`
}

func (ServerApp) TableName() string {
	return "server_apps"
}

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"column:key;size:50;not null;uniqueIndex:key_value"`
	Value     string    `json:"value" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at"`
}

func (Tag) TableName() string {
	return "tags"
}

type AssetTag struct {
	AssetType string `json:"asset_type" gorm:"size:20;not null"`
	AssetID   uint   `json:"asset_id" gorm:"not null"`
	TagID     uint   `json:"tag_id" gorm:"not null"`

	// 关联
	Tag *Tag `json:"tag,omitempty" gorm:"foreignKey:TagID"`
}

func (AssetTag) TableName() string {
	return "asset_tags"
}

// DeployRecord 部署记录
type DeployRecord struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	AppID           uint       `json:"app_id" gorm:"index;not null"`
	AppName         string     `json:"app_name" gorm:"size:100;not null"`
	EnvID           uint       `json:"env_id" gorm:"index;not null"`
	EnvName         string     `json:"env_name" gorm:"size:50;not null"`
	EnvType         string     `json:"env_type" gorm:"size:10;not null"` // dev/test/prod
	ProjectCode     string     `json:"project_code" gorm:"size:50;not null"`
	DeployType      string     `json:"deploy_type" gorm:"size:20;not null"` // frontend/backend/all
	JenkinsJob      string     `json:"jenkins_job" gorm:"size:200"`
	JenkinsBuildNum int        `json:"jenkins_build_num"`
	JenkinsQueueID  int64      `json:"jenkins_queue_id"`
	Status          string     `json:"status" gorm:"size:20;default:'pending';index:idx_status_created;not null"` // pending/running/success/failed
	ErrorMessage    string     `json:"error_message,omitempty" gorm:"type:text"`
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Duration        int        `json:"duration"`                           // 秒
	TriggeredBy     string     `json:"triggered_by" gorm:"size:100;index"` // 触发人
	CreatedAt       time.Time  `json:"created_at" gorm:"index:idx_status_created"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// 关联
	App         *Application `json:"app,omitempty" gorm:"foreignKey:AppID"`
	Environment *Environment `json:"environment,omitempty" gorm:"foreignKey:EnvID"`
}

func (DeployRecord) TableName() string {
	return "deploy_records"
}

// ArchiveRecord 归档记录
type ArchiveRecord struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	AppID             uint       `json:"app_id" gorm:"index;not null"`
	AppName           string     `json:"app_name" gorm:"size:100;not null"`
	EnvID             uint       `json:"env_id" gorm:"index;not null"`
	EnvName           string     `json:"env_name" gorm:"size:50;not null"`
	EnvType           string     `json:"env_type" gorm:"size:10;not null"`     // dev/test/prod
	DeployType        string     `json:"deploy_type" gorm:"size:20;not null"`  // frontend/backend/all
	ProjectCode       string     `json:"project_code" gorm:"size:50;not null"` // 项目编号，用于下载地址
	JenkinsJob        string     `json:"jenkins_job" gorm:"size:200"`
	JenkinsBuildNum   int        `json:"jenkins_build_num"`
	JenkinsQueueID    int64      `json:"jenkins_queue_id"`
	JenkinsConsoleURL string     `json:"jenkins_console_url,omitempty"`
	DownloadURL       string     `json:"download_url,omitempty"`
	Status            string     `json:"status" gorm:"size:20;default:'pending';index:idx_status_created;not null"` // pending/running/success/failed
	ErrorMessage      string     `json:"error_message,omitempty" gorm:"type:text"`
	StartTime         *time.Time `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Operator          string     `json:"operator" gorm:"size:100;index"` // 操作人
	CreatedAt         time.Time  `json:"created_at" gorm:"index:idx_status_created"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// 关联
	App         *Application `json:"app,omitempty" gorm:"foreignKey:AppID"`
	Environment *Environment `json:"environment,omitempty" gorm:"foreignKey:EnvID"`
}

func (ArchiveRecord) TableName() string {
	return "archive_records"
}
