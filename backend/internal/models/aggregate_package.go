// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

// AggregatePackageTask 存储聚合打包任务基本信息
type AggregatePackageTask struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TaskName         string    `gorm:"type:varchar(255);not null" json:"task_name"`
	ProjectName      string    `gorm:"type:varchar(255);not null" json:"project_name"`
	AppNames         []string  `gorm:"serializer:json" json:"app_names"`  // 参与打包的应用名称列表
	JenkinsJobName   string    `gorm:"type:varchar(255)" json:"jenkins_job_name"`
	JenkinsJobUrl    string    `gorm:"type:varchar(500)" json:"jenkins_job_url"`
	ConsulConfigPath string    `gorm:"type:varchar(500)" json:"consul_config_path"`
	BuildParams      map[string]string `gorm:"serializer:json" json:"build_params"` // 构建参数
	Status           string    `gorm:"type:enum('pending','building','success','failed','cancelled');default:'pending'" json:"status"`
	TriggeredBy      string    `gorm:"type:varchar(255)" json:"triggered_by"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	Duration         int       `json:"duration"`  // 耗时(秒)
	ErrorMessage     *string   `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	JenkinsQueueID   *int64    `json:"jenkins_queue_id,omitempty"`
}

// AggregatePackageResult 存储每个应用的打包结果
type AggregatePackageResult struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TaskID           uint      `gorm:"index;not null" json:"task_id"`
	AppName          string    `gorm:"type:varchar(255);not null" json:"app_name"`
	JenkinsBuildNum  *int      `json:"jenkins_build_num,omitempty"`
	JenkinsQueueID   *int64    `json:"jenkins_queue_id,omitempty"`
	JenkinsConsoleURL *string  `gorm:"type:varchar(500)" json:"jenkins_console_url,omitempty"`
	ConsulTag        string    `gorm:"type:varchar(255)" json:"consul_tag"`  // 从Consul获取的标签
	Status           string    `gorm:"type:enum('pending','building','success','failed');default:'pending'" json:"status"`
	ErrorMessage     *string   `json:"error_message,omitempty"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	Duration         int       `json:"duration"`  // 耗时(秒)
	DownloadURL      *string   `gorm:"type:varchar(500)" json:"download_url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}