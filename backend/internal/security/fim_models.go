// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import "time"

type FIMPolicy struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"size:120;uniqueIndex;not null"`
	Description     string    `json:"description" gorm:"size:500"`
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	Severity        string    `json:"severity" gorm:"size:20;default:'high'"`
	NotifyChannels  string    `json:"notify_channels" gorm:"size:500"`
	ScanIntervalSec int       `json:"scan_interval_sec" gorm:"default:300"`
	HashMode        string    `json:"hash_mode" gorm:"size:20;default:'changed_only'"`
	CompareMode     string    `json:"compare_mode" gorm:"size:20;default:'baseline'"`
	CreatedBy       string    `json:"created_by" gorm:"size:100"`
	UpdatedBy       string    `json:"updated_by" gorm:"size:100"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (FIMPolicy) TableName() string { return "security_fim_policies" }

type FIMPolicyTarget struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	PolicyID       uint       `json:"policy_id" gorm:"index;not null"`
	ServerID       uint       `json:"server_id" gorm:"index;not null"`
	ServerName     string     `json:"server_name,omitempty" gorm:"-"`
	ServerIP       string     `json:"server_ip,omitempty" gorm:"-"`
	Enabled        bool       `json:"enabled" gorm:"default:true"`
	LastScanAt     *time.Time `json:"last_scan_at"`
	LastScanStatus string     `json:"last_scan_status" gorm:"size:20"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (FIMPolicyTarget) TableName() string { return "security_fim_policy_targets" }

type FIMWatchPath struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	PolicyID        uint      `json:"policy_id" gorm:"index;not null"`
	Path            string    `json:"path" gorm:"size:500;not null"`
	ScanMode        string    `json:"scan_mode" gorm:"size:20;default:'full_hash'"`
	Recursive       bool      `json:"recursive" gorm:"default:true"`
	MaxDepth        int       `json:"max_depth" gorm:"default:0"`
	FileGlob        string    `json:"file_glob" gorm:"size:255"`
	ExcludeGlob     string    `json:"exclude_glob" gorm:"size:255"`
	HashOnMatchOnly bool      `json:"hash_on_match_only" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (FIMWatchPath) TableName() string { return "security_fim_watch_paths" }

type FIMSnapshot struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	PolicyID     uint       `json:"policy_id" gorm:"index;not null"`
	ServerID     uint       `json:"server_id" gorm:"index;not null"`
	PolicyName   string     `json:"policy_name,omitempty" gorm:"-"`
	ServerName   string     `json:"server_name,omitempty" gorm:"-"`
	ServerIP     string     `json:"server_ip,omitempty" gorm:"-"`
	OriginType   string     `json:"origin_type" gorm:"size:20;default:'scheduled'"`
	SnapshotType string     `json:"snapshot_type" gorm:"size:20;default:'scheduled'"`
	Status       string     `json:"status" gorm:"size:20;default:'running'"`
	Operator     string     `json:"operator" gorm:"size:100;default:'system'"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	EntryCount   int64      `json:"entry_count"`
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (FIMSnapshot) TableName() string { return "security_fim_snapshots" }

type FIMSnapshotEntry struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	SnapshotID uint       `json:"snapshot_id" gorm:"index;not null"`
	Path       string     `json:"path" gorm:"size:1000;not null"`
	EntryType  string     `json:"entry_type" gorm:"size:20;default:'file'"`
	Size       int64      `json:"size"`
	Mode       string     `json:"mode" gorm:"size:16"`
	Owner      string     `json:"owner" gorm:"size:100"`
	GroupName  string     `json:"group_name" gorm:"size:100"`
	Mtime      *time.Time `json:"mtime"`
	SHA256     string     `json:"sha256" gorm:"size:64"`
	TargetPath string     `json:"target_path" gorm:"size:1000"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (FIMSnapshotEntry) TableName() string { return "security_fim_snapshot_entries" }

type FIMDiffEvent struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	PolicyID           uint      `json:"policy_id" gorm:"index;not null"`
	ServerID           uint      `json:"server_id" gorm:"index;not null"`
	PolicyName         string    `json:"policy_name,omitempty" gorm:"-"`
	ServerName         string    `json:"server_name,omitempty" gorm:"-"`
	ServerIP           string    `json:"server_ip,omitempty" gorm:"-"`
	BaselineSnapshotID *uint     `json:"baseline_snapshot_id"`
	CurrentSnapshotID  uint      `json:"current_snapshot_id" gorm:"index;not null"`
	Path               string    `json:"path" gorm:"size:1000;not null"`
	EventType          string    `json:"event_type" gorm:"size:20;not null"`
	Severity           string    `json:"severity" gorm:"size:20;default:'high'"`
	OldValueJSON       string    `json:"old_value_json" gorm:"type:longtext"`
	NewValueJSON       string    `json:"new_value_json" gorm:"type:longtext"`
	OccurredAt         time.Time `json:"occurred_at"`
	CreatedAt          time.Time `json:"created_at"`
}

func (FIMDiffEvent) TableName() string { return "security_fim_diff_events" }

type FIMAlert struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	DiffEventID     uint      `json:"diff_event_id" gorm:"index;not null"`
	PolicyID        uint      `json:"policy_id" gorm:"index;not null"`
	ServerID        uint      `json:"server_id" gorm:"index;not null"`
	Path            string    `json:"path" gorm:"size:1000;index:idx_security_fim_alerts_path,length:255"`
	EventType       string    `json:"event_type" gorm:"size:20;index"`
	PolicyName      string    `json:"policy_name,omitempty" gorm:"-"`
	ServerName      string    `json:"server_name,omitempty" gorm:"-"`
	ServerIP        string    `json:"server_ip,omitempty" gorm:"-"`
	Title           string    `json:"title" gorm:"size:255;not null"`
	Summary         string    `json:"summary" gorm:"size:1000"`
	Severity        string    `json:"severity" gorm:"size:20;default:'high'"`
	Status          string    `json:"status" gorm:"size:20;default:'open'"`
	OccurrenceCount int       `json:"occurrence_count" gorm:"default:1"`
	Assignee        string    `json:"assignee" gorm:"size:100"`
	FirstSeenAt     time.Time `json:"first_seen_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (FIMAlert) TableName() string { return "security_fim_alerts" }