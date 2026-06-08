package models

import "time"

// PlatformEvent 为跨模块统一事件流。
type PlatformEvent struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	EventID       string     `json:"event_id" gorm:"size:180;uniqueIndex;not null"`
	EventType     string     `json:"event_type" gorm:"size:80;index;not null"`
	EventCategory string     `json:"event_category" gorm:"size:50;index;not null"`
	SourceSystem  string     `json:"source_system" gorm:"size:50;index;not null"`
	SourceTable   string     `json:"source_table" gorm:"size:80;index;not null"`
	SourceID      string     `json:"source_id" gorm:"size:50;index;not null"`
	ObjectType    string     `json:"object_type" gorm:"size:50;index;not null"`
	ObjectID      string     `json:"object_id" gorm:"size:150;index;not null"`
	Title         string     `json:"title" gorm:"size:255;not null"`
	Summary       string     `json:"summary" gorm:"type:text"`
	Status        string     `json:"status" gorm:"size:50;index"`
	Severity      string     `json:"severity" gorm:"size:50;index"`
	OperatorID    string     `json:"operator_id" gorm:"size:100;index"`
	OperatorName  string     `json:"operator_name" gorm:"size:100"`
	TriggerMode   string     `json:"trigger_mode" gorm:"size:30;index"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	OccurredAt    time.Time  `json:"occurred_at" gorm:"index;not null"`
	MetadataJSON  string     `json:"metadata_json" gorm:"type:longtext"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PlatformEvent) TableName() string {
	return "platform_event_stream"
}
