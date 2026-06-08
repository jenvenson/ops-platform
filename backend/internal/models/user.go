package models

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Password  string    `json:"-" gorm:"not null"`
	RealName  string    `json:"real_name" gorm:"size:50"` // 姓名
	Email     string    `json:"email" gorm:"size:100"`
	Role      string    `json:"role" gorm:"size:10;default:'user'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
