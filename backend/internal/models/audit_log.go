package models

import "time"

type PlatformAccessLog struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	MenuKey         string    `json:"menu_key" gorm:"size:64;index"`
	MenuTitle       string    `json:"menu_title" gorm:"size:128"`
	PagePath        string    `json:"page_path" gorm:"size:255"`
	RequestPath     string    `json:"request_path" gorm:"size:255;index"`
	RequestMethod   string    `json:"request_method" gorm:"size:16;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	UserAgent       string    `json:"user_agent" gorm:"size:512"`
	Referer         string    `json:"referer" gorm:"size:512"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	DurationMS      int64     `json:"duration_ms"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	AccessedAt      time.Time `json:"accessed_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PlatformAccessLog) TableName() string { return "platform_access_logs" }

type PlatformAccessLogArchive struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	MenuKey         string    `json:"menu_key" gorm:"size:64;index"`
	MenuTitle       string    `json:"menu_title" gorm:"size:128"`
	PagePath        string    `json:"page_path" gorm:"size:255"`
	RequestPath     string    `json:"request_path" gorm:"size:255;index"`
	RequestMethod   string    `json:"request_method" gorm:"size:16;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	UserAgent       string    `json:"user_agent" gorm:"size:512"`
	Referer         string    `json:"referer" gorm:"size:512"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	DurationMS      int64     `json:"duration_ms"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	AccessedAt      time.Time `json:"accessed_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	ArchivedAt      time.Time `json:"archived_at" gorm:"index"`
}

func (PlatformAccessLogArchive) TableName() string { return "platform_access_logs_archive" }

type PlatformAuditLog struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	Module          string    `json:"module" gorm:"size:64;index"`
	ResourceType    string    `json:"resource_type" gorm:"size:64;index"`
	ResourceID      string    `json:"resource_id" gorm:"size:64"`
	ResourceName    string    `json:"resource_name" gorm:"size:255"`
	Action          string    `json:"action" gorm:"size:64;index"`
	ActionLabel     string    `json:"action_label" gorm:"size:128"`
	RequestPath     string    `json:"request_path" gorm:"size:255;index"`
	RequestMethod   string    `json:"request_method" gorm:"size:16;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	RequestParams   string    `json:"request_params_json" gorm:"type:longtext"`
	BeforeData      string    `json:"before_data_json" gorm:"type:longtext"`
	AfterData       string    `json:"after_data_json" gorm:"type:longtext"`
	ChangeSummary   string    `json:"change_summary" gorm:"type:text"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	DurationMS      int64     `json:"duration_ms"`
	OperatedAt      time.Time `json:"operated_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PlatformAuditLog) TableName() string { return "platform_audit_logs" }

type PlatformAuditLogArchive struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	Module          string    `json:"module" gorm:"size:64;index"`
	ResourceType    string    `json:"resource_type" gorm:"size:64;index"`
	ResourceID      string    `json:"resource_id" gorm:"size:64"`
	ResourceName    string    `json:"resource_name" gorm:"size:255"`
	Action          string    `json:"action" gorm:"size:64;index"`
	ActionLabel     string    `json:"action_label" gorm:"size:128"`
	RequestPath     string    `json:"request_path" gorm:"size:255;index"`
	RequestMethod   string    `json:"request_method" gorm:"size:16;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	RequestParams   string    `json:"request_params_json" gorm:"type:longtext"`
	BeforeData      string    `json:"before_data_json" gorm:"type:longtext"`
	AfterData       string    `json:"after_data_json" gorm:"type:longtext"`
	ChangeSummary   string    `json:"change_summary" gorm:"type:text"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	DurationMS      int64     `json:"duration_ms"`
	OperatedAt      time.Time `json:"operated_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	ArchivedAt      time.Time `json:"archived_at" gorm:"index"`
}

func (PlatformAuditLogArchive) TableName() string { return "platform_audit_logs_archive" }

type PlatformLoginLog struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	UserAgent       string    `json:"user_agent" gorm:"size:512"`
	RequestPath     string    `json:"request_path" gorm:"size:255"`
	RequestMethod   string    `json:"request_method" gorm:"size:16"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	LoginType       string    `json:"login_type" gorm:"size:32;default:'password'"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	DurationMS      int64     `json:"duration_ms"`
	LoggedInAt      time.Time `json:"logged_in_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PlatformLoginLog) TableName() string { return "platform_login_logs" }

type PlatformLoginLogArchive struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	TraceID         string    `json:"trace_id" gorm:"size:64;index"`
	UserID          uint      `json:"user_id" gorm:"index"`
	Username        string    `json:"username" gorm:"size:64;index"`
	RealName        string    `json:"real_name" gorm:"size:64"`
	Role            string    `json:"role" gorm:"size:32;index"`
	RequestIP       string    `json:"request_ip" gorm:"size:64;index"`
	UserAgent       string    `json:"user_agent" gorm:"size:512"`
	RequestPath     string    `json:"request_path" gorm:"size:255"`
	RequestMethod   string    `json:"request_method" gorm:"size:16"`
	StatusCode      int       `json:"status_code" gorm:"index"`
	OperationStatus string    `json:"operation_status" gorm:"size:16;index"`
	LoginType       string    `json:"login_type" gorm:"size:32;default:'password'"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	DurationMS      int64     `json:"duration_ms"`
	LoggedInAt      time.Time `json:"logged_in_at" gorm:"index"`
	CreatedAt       time.Time `json:"created_at"`
	ArchivedAt      time.Time `json:"archived_at" gorm:"index"`
}

func (PlatformLoginLogArchive) TableName() string { return "platform_login_logs_archive" }
