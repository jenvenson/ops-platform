package models

import "time"

type AuditLogSetting struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	AccessLogEnabled    bool      `json:"access_log_enabled" gorm:"default:true"`
	OperationLogEnabled bool      `json:"operation_log_enabled" gorm:"default:true"`
	LoginLogEnabled     bool      `json:"login_log_enabled" gorm:"default:true"`
	UpdatedBy           string    `json:"updated_by" gorm:"size:100"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (AuditLogSetting) TableName() string {
	return "system_audit_log_settings"
}
