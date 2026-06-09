package models

import "time"

type PasswordResetToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;size:64;not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
