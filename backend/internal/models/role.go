package models

import "time"

type Role struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:50;not null;comment:角色名称"`
	Code        string    `json:"code" gorm:"size:50;uniqueIndex;not null;comment:角色编码"`
	Description string    `json:"description" gorm:"size:255;comment:角色描述"`
	Status      int       `json:"status" gorm:"default:1;comment:状态: 1-启用, 0-禁用"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}
