// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package models

import "time"

// FIMKnownHost 已知主机密钥
type FIMKnownHost struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Hostname    string `json:"hostname" gorm:"size:255;not null;uniqueIndex:uk_host_port_keytype"`
	Port        int    `json:"port" gorm:"not null;default:22;uniqueIndex:uk_host_port_keytype"`

	KeyType           string `json:"key_type" gorm:"size:50;not null;uniqueIndex:uk_host_port_keytype"`
	PublicKey         string `json:"public_key" gorm:"type:text;not null"`
	FingerprintSHA256 string `json:"fingerprint_sha256" gorm:"size:64;not null"`

	ServerID    *uint  `json:"server_id"`
	Description string `json:"description" gorm:"size:500"`
	Tags        string `json:"tags" gorm:"type:json"` // JSON array

	VerificationStatus string     `json:"verification_status" gorm:"size:20;default:verified"`
	VerifiedBy         string     `json:"verified_by" gorm:"size:100"`
	VerifiedAt         *time.Time `json:"verified_at"`

	LastUsedAt *time.Time `json:"last_used_at"`
	UseCount   int        `json:"use_count" gorm:"default:0"`

	AddedBy   string    `json:"added_by" gorm:"size:100;not null"`
	AddedAt   time.Time `json:"added_at" gorm:"autoCreateTime"`
	UpdatedBy string    `json:"updated_by" gorm:"size:100"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	IsEnabled bool `json:"is_enabled" gorm:"default:true"`
}

func (FIMKnownHost) TableName() string {
	return "fim_known_hosts"
}

// FIMKnownHostsHistory 主机密钥变更历史
type FIMKnownHostsHistory struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	HostID  uint   `json:"host_id" gorm:"not null;index"`
	Action  string `json:"action" gorm:"size:20;not null"`

	OldKeyType     *string `json:"old_key_type" gorm:"size:50"`
	OldPublicKey   *string `json:"old_public_key" gorm:"type:text"`
	OldFingerprint *string `json:"old_fingerprint" gorm:"size:64"`

	NewKeyType     *string `json:"new_key_type" gorm:"size:50"`
	NewPublicKey   *string `json:"new_public_key" gorm:"type:text"`
	NewFingerprint *string `json:"new_fingerprint" gorm:"size:64"`

	OperatedBy string    `json:"operated_by" gorm:"size:100;not null"`
	OperatedAt time.Time `json:"operated_at" gorm:"autoCreateTime"`
	Reason     string    `json:"reason" gorm:"type:text"`
	IPAddress  string    `json:"ip_address" gorm:"size:45"`
}

func (FIMKnownHostsHistory) TableName() string {
	return "fim_known_hosts_history"
}

// FIMSSHConnectionLog SSH连接尝试日志
type FIMSSHConnectionLog struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Hostname string `json:"hostname" gorm:"size:255;not null;index:idx_hostname_port"`
	Port     int    `json:"port" gorm:"not null;index:idx_hostname_port"`

	Result       string `json:"result" gorm:"size:20;not null;index"`
	ErrorMessage string `json:"error_message" gorm:"type:text"`

	PresentedKeyType     *string `json:"presented_key_type" gorm:"size:50"`
	PresentedFingerprint *string `json:"presented_fingerprint" gorm:"size:64"`
	ExpectedFingerprint  *string `json:"expected_fingerprint" gorm:"size:64"`

	AttemptedAt time.Time `json:"attempted_at" gorm:"autoCreateTime;index"`
	ServerID    *uint     `json:"server_id" gorm:"index"`
	PolicyID    *uint     `json:"policy_id"`
	SnapshotID  *uint     `json:"snapshot_id"`
}

func (FIMSSHConnectionLog) TableName() string {
	return "fim_ssh_connection_logs"
}