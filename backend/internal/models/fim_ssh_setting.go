// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

type FIMSSHSetting struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	AuthMode            string    `json:"auth_mode" gorm:"size:20;not null;default:'password'"`
	SSHUser             string    `json:"ssh_user" gorm:"size:100;not null"`
	PasswordEncrypted   string    `json:"-" gorm:"type:text"`
	PrivateKeyEncrypted string    `json:"-" gorm:"type:longtext"`
	TimeoutSec          int       `json:"timeout_sec" gorm:"default:15"`
	UpdatedBy           string    `json:"updated_by" gorm:"size:100"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (FIMSSHSetting) TableName() string {
	return "system_fim_ssh_settings"
}