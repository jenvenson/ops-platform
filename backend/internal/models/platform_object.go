package models

import "time"

// PlatformObject 为跨模块统一对象索引。
type PlatformObject struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ObjectUID    string    `json:"object_uid" gorm:"size:150;uniqueIndex;not null"`
	ObjectType   string    `json:"object_type" gorm:"size:50;index;not null"`
	SourceModule string    `json:"source_module" gorm:"size:50;index;not null"`
	SourcePK     string    `json:"source_pk" gorm:"size:50;index;not null"`
	Title        string    `json:"title" gorm:"size:255;not null"`
	Summary      string    `json:"summary" gorm:"type:text"`
	Status       string    `json:"status" gorm:"size:50;index"`
	OwnerID      string    `json:"owner_id" gorm:"size:100;index"`
	MetadataJSON string    `json:"metadata_json" gorm:"type:longtext"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (PlatformObject) TableName() string {
	return "platform_object_index"
}
