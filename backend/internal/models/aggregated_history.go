package models

import "time"

// AggregatedHistory 聚合历史记录模型
type AggregatedHistory struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	ProjectName string     `json:"project_name" gorm:"type:varchar(255);not null;comment:项目名称"`
	Environment string     `json:"environment" gorm:"type:varchar(100);comment:Tag名称"`
	Status      string     `json:"status" gorm:"type:varchar(50);default:'pending';comment:状态"`
	Progress    int        `json:"progress" gorm:"default:0;comment:进度百分比(0-100)"`
	StartTime   *time.Time `json:"start_time,omitempty" gorm:"comment:归档开始时间"`
	EndTime     *time.Time `json:"end_time,omitempty" gorm:"comment:归档结束时间"`
	DownloadURL *string    `json:"download_url,omitempty" gorm:"type:text;comment:下载地址"`
	Operator    string     `json:"operator" gorm:"type:varchar(100);comment:操作人"`
	OperatorName string    `json:"operator_name" gorm:"type:varchar(100);comment:操作人姓名"`
	Operation   string     `json:"operation,omitempty" gorm:"-"` // 前端计算字段，不存储在数据库中
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Jenkins相关字段
	JenkinsJobName    string  `json:"jenkins_job_name" gorm:"type:varchar(255);comment:Jenkins任务名称"`
	JenkinsBuildNum   *int    `json:"jenkins_build_num,omitempty" gorm:"comment:Jenkins构建编号"`
	JenkinsQueueID    *int64  `json:"jenkins_queue_id,omitempty" gorm:"comment:Jenkins队列ID"`
	JenkinsConsoleURL *string `json:"jenkins_console_url,omitempty" gorm:"type:text;comment:Jenkins控制台日志URL"`
	ErrorMessage      *string `json:"error_message,omitempty" gorm:"type:text;comment:错误信息"`

	// 关联聚合打包任务
	TaskID *uint `json:"task_id,omitempty" gorm:"comment:关联的任务ID"`
}
