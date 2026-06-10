// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

type Menu struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"size:50;not null;comment:菜单标题"`
	Key       string    `json:"key" gorm:"size:50;uniqueIndex;not null;comment:菜单唯一标识"`
	Path      string    `json:"path" gorm:"size:255;comment:菜单路径"`
	Icon      string    `json:"icon" gorm:"size:50;comment:菜单图标"`
	ParentID  uint      `json:"parent_id" gorm:"default:0;comment:父菜单ID"`
	Sort      int       `json:"sort" gorm:"default:0;comment:排序"`
	Status    int       `json:"status" gorm:"default:1;comment:状态: 1-启用, 0-禁用"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Children  []Menu    `json:"children,omitempty" gorm:"-"`
}

func (Menu) TableName() string {
	return "menus"
}