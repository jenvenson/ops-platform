package models

import "time"

type RoleMenu struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RoleID    uint      `json:"role_id" gorm:"not null;comment:角色ID"`
	MenuID    uint      `json:"menu_id" gorm:"not null;comment:菜单ID"`
	CreatedAt time.Time `json:"created_at"`
}

func (RoleMenu) TableName() string {
	return "role_menus"
}
