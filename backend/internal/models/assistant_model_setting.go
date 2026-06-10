// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

type AssistantModelSetting struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Provider    string    `json:"provider" gorm:"size:50;default:ollama"`
	Enabled     bool      `json:"enabled" gorm:"default:false"`
	APIKey      string    `json:"api_key" gorm:"size:512"`
	BaseURL     string    `json:"base_url" gorm:"size:512"`
	ChatModel   string    `json:"chat_model" gorm:"size:100"`
	EmbedModel  string    `json:"embed_model" gorm:"size:100"`
	Temperature float64   `json:"temperature" gorm:"type:decimal(3,2);default:0.20"`
	TimeoutSec  int       `json:"timeout_sec" gorm:"default:20"`
	UpdatedBy   string    `json:"updated_by" gorm:"size:100"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AssistantModelSetting) TableName() string {
	return "system_assistant_model_settings"
}