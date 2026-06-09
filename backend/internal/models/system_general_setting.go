package models

import "time"

// SystemGeneralSetting 系统通用配置，单行表（只有一条记录）。
type SystemGeneralSetting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SiteName  string    `json:"site_name" gorm:"size:128;default:'运维管理平台'"`
	Timezone  string    `json:"timezone" gorm:"size:64;default:'Asia/Shanghai'"`
	Language  string    `json:"language" gorm:"size:16;default:'zh-CN'"`
	UpdatedBy string    `json:"updated_by" gorm:"size:64;default:''"`
	UpdatedAt time.Time `json:"updated_at"`
}
