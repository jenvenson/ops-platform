package models

import "time"

// ConsulKVReplaceRule Consul KV路径替换规则
type ConsulKVReplaceRule struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AppID           uint      `gorm:"uniqueIndex;not null" json:"app_id"`
	AppName         string    `gorm:"size:255;not null" json:"app_name"`
	ConsulPathPrefix string   `gorm:"size:255;not null" json:"consul_path_prefix"`
	Description     string    `gorm:"size:500" json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ConsulKVReplaceRule) TableName() string {
	return "consul_kv_replace_rules"
}
