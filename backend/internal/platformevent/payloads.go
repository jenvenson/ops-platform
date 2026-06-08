package platformevent

import "time"

type DeployRecordPayload struct {
	ID           uint
	AppID        uint
	AppName      string
	EnvID        uint
	EnvName      string
	ProjectCode  string
	DeployType   string
	Status       string
	ErrorMessage string
	StartTime    *time.Time
	EndTime      *time.Time
	TriggeredBy  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (DeployRecordPayload) TableName() string {
	return "deploy_records"
}

type ArchiveRecordPayload struct {
	ID           uint
	AppID        uint
	AppName      string
	EnvID        uint
	EnvName      string
	ProjectCode  string
	DeployType   string
	Status       string
	ErrorMessage string
	StartTime    *time.Time
	EndTime      *time.Time
	Operator     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ArchiveRecordPayload) TableName() string {
	return "archive_records"
}

type AlertEventPayload struct {
	ID           uint
	RuleName     string
	Severity     string
	Category     string
	Content      string
	Source       string
	Status       string
	FiredAt      time.Time
	AckedAt      *time.Time
	ResolvedAt   *time.Time
	ClosedAt     *time.Time
	AckedBy      string
	ClosedBy     string
	HandleType   string
	HandleNote   string
	Labels       string
	Fingerprint  string
	NotifyStatus string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (AlertEventPayload) TableName() string {
	return "alert_events"
}
