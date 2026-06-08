package models

import "time"

// PipelineTriggerMode 流水线触发方式
type PipelineTriggerMode string

const (
	PipelineTriggerManual    PipelineTriggerMode = "manual"    // 手动触发
	PipelineTriggerScheduled PipelineTriggerMode = "scheduled" // 定时触发
	PipelineTriggerWebhook   PipelineTriggerMode = "webhook"   // Webhook触发
)

// PipelineStatus 流水线状态
type PipelineStatus string

const (
	PipelineStatusPending   PipelineStatus = "pending"   // 待执行
	PipelineStatusRunning   PipelineStatus = "running"   // 执行中
	PipelineStatusSuccess   PipelineStatus = "success"   // 成功
	PipelineStatusFailed    PipelineStatus = "failed"    // 失败
	PipelineStatusCancelled PipelineStatus = "cancelled" // 已取消
)

// Pipeline 流水线模型
type Pipeline struct {
	ID                  uint                `json:"id" gorm:"primaryKey"`
	Name                string              `json:"name" gorm:"size:100;not null"`
	Description         string              `json:"description" gorm:"size:500"`
	Repository          string              `json:"repository" gorm:"size:255;not null"` // 关联仓库/项目
	TriggerMode         PipelineTriggerMode `json:"trigger_mode" gorm:"size:20;default:'manual'"`
	CronExpression      string              `json:"cron_expression" gorm:"size:50"`        // 定时触发Cron表达式
	Branch              string              `json:"branch" gorm:"size:100;default:'main'"` // 默认分支
	YAMLConfig          string              `json:"yaml_config" gorm:"type:text"`          // pipeline.yaml 配置
	LastExecutionID     *uint               `json:"last_execution_id"`                     // 最近一次执行ID
	LastExecutionStatus PipelineStatus      `json:"last_execution_status"`                 // 最近执行状态
	LastExecutionTime   *time.Time          `json:"last_execution_time"`                   // 最近执行时间
	// Jenkins 配置（可覆盖全局配置）
	JenkinsURL  string `json:"jenkins_url" gorm:"size:255"`  // Jenkins 地址（可选，覆盖全局）
	JenkinsView string `json:"jenkins_view" gorm:"size:100"` // Jenkins View 名称
	JenkinsJob  string `json:"jenkins_job" gorm:"size:100"`  // Jenkins Job 名称
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Pipeline) TableName() string {
	return "pipelines"
}

// PipelineExecution 流水线执行记录模型
type PipelineExecution struct {
	ID            uint                `json:"id" gorm:"primaryKey"`
	PipelineID    uint                `json:"pipeline_id" gorm:"index;not null"`
	TriggerMode   PipelineTriggerMode `json:"trigger_mode" gorm:"size:20"`
	Branch        string              `json:"branch" gorm:"size:100"`
	CommitID      string              `json:"commit_id" gorm:"size:40"`
	CommitMessage string              `json:"commit_message" gorm:"size:500"`
	Status        PipelineStatus      `json:"status" gorm:"size:20;default:'pending'"`
	StartedAt     *time.Time          `json:"started_at"`
	FinishedAt    *time.Time          `json:"finished_at"`
	Duration      int64               `json:"duration"`      // 耗时（秒）
	ExecutorID    *uint               `json:"executor_id"`   // 执行人ID
	ExecutorName  string              `json:"executor_name"` // 执行人名称
	TriggeredBy   string              `json:"triggered_by"`  // 触发方式描述
	// Jenkins 构建信息
	JenkinsBuildNumber int    `json:"jenkins_build_number"`              // Jenkins 构建号
	JenkinsBuildURL    string `json:"jenkins_build_url" gorm:"size:255"` // Jenkins 构建链接
	JenkinsQueueID     int64  `json:"jenkins_queue_id"`                  // Jenkins 队列 ID
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (PipelineExecution) TableName() string {
	return "pipeline_executions"
}

// PipelineStage 流水线阶段模型
type PipelineStage struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ExecutionID  uint           `json:"execution_id" gorm:"index;not null"`
	Name         string         `json:"name" gorm:"size:100;not null"`
	StageOrder   int            `json:"stage_order"` // 阶段顺序
	Status       PipelineStatus `json:"status" gorm:"size:20;default:'pending'"`
	StartedAt    *time.Time     `json:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at"`
	Duration     int64          `json:"duration"` // 耗时（秒）
	ErrorMessage string         `json:"error_message" gorm:"size:1000"`
	Logs         string         `json:"logs" gorm:"type:text"` // 阶段执行日志
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (PipelineStage) TableName() string {
	return "pipeline_stages"
}
