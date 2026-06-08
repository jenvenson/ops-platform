package models

import "time"

// AssistantSession 记录运维小助手会话。
type AssistantSession struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"size:100;uniqueIndex;not null"`
	UserID    string    `json:"user_id" gorm:"size:100"`
	Scene     string    `json:"scene" gorm:"size:50;default:web"`
	Status    string    `json:"status" gorm:"size:20;default:active"`
	Title     string    `json:"title" gorm:"size:200"`
	Pinned    bool      `json:"pinned" gorm:"default:false"`
	UserAgent string    `json:"user_agent" gorm:"size:500"`
	IPAddress string    `json:"ip_address" gorm:"size:50"`
	Summary   string    `json:"summary" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AssistantSession) TableName() string {
	return "assistant_sessions"
}

// AssistantMessage 记录会话消息和模型元数据。
type AssistantMessage struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	SessionID        string    `json:"session_id" gorm:"size:100;index;not null"`
	MessageID        string    `json:"message_id" gorm:"size:100;uniqueIndex;not null"`
	UserID           string    `json:"user_id" gorm:"size:100"`
	Role             string    `json:"role" gorm:"size:20;not null"`
	Intent           string    `json:"intent" gorm:"size:50"`
	Content          string    `json:"content" gorm:"type:text;not null"`
	ModelName        string    `json:"model_name" gorm:"size:100"`
	ActionsJSON      string    `json:"actions_json" gorm:"type:longtext"`
	ResultCardsJSON  string    `json:"result_cards_json" gorm:"type:longtext"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	LatencyMS        int64     `json:"latency_ms"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (AssistantMessage) TableName() string {
	return "assistant_messages"
}

// AssistantCitation 记录回答引用来源。
type AssistantCitation struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	MessageID   string    `json:"message_id" gorm:"size:100;index;not null"`
	SourceType  string    `json:"source_type" gorm:"size:50"`
	SourceTitle string    `json:"source_title" gorm:"size:255"`
	SourcePath  string    `json:"source_path" gorm:"size:500"`
	Snippet     string    `json:"snippet" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AssistantCitation) TableName() string {
	return "assistant_citations"
}
