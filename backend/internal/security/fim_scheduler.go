// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jenvenson/ops-platform/internal/cmdb"
	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
)

const (
	fimSchedulerOperator     = "fim-scheduler"
	fimSchedulerPollInterval = 15 * time.Second
)

var fimSchedulerOnce sync.Once

type fimScheduler struct {
	service *FIMService
	running sync.Map
}

func startFIMScheduler() {
	fimSchedulerOnce.Do(func() {
		scheduler := &fimScheduler{service: NewFIMService()}
		go scheduler.loop()
		log.Println("[FIM Scheduler] started")
	})
}

func (s *fimScheduler) loop() {
	ticker := time.NewTicker(fimSchedulerPollInterval)
	defer ticker.Stop()

	s.runOnce()
	for range ticker.C {
		s.runOnce()
	}
}

func (s *fimScheduler) runOnce() {
	var policies []FIMPolicy
	if err := database.DB.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		log.Printf("[FIM Scheduler] load policies failed: %v", err)
		return
	}

	now := time.Now()
	for _, policy := range policies {
		var targets []FIMPolicyTarget
		if err := database.DB.Where("policy_id = ? AND enabled = ?", policy.ID, true).Find(&targets).Error; err != nil {
			log.Printf("[FIM Scheduler] load targets failed policy=%d err=%v", policy.ID, err)
			continue
		}

		for _, target := range targets {
			if !shouldRunFIMScheduledScan(now, target.LastScanAt, policy.ScanIntervalSec) {
				continue
			}
			s.runTarget(policy, target)
		}
	}
}

func (s *fimScheduler) runTarget(policy FIMPolicy, target FIMPolicyTarget) {
	key := fmt.Sprintf("%d:%d", policy.ID, target.ServerID)
	if _, loaded := s.running.LoadOrStore(key, struct{}{}); loaded {
		return
	}

	go func() {
		defer s.running.Delete(key)

		skipReason, err := s.preflightTarget(target.ServerID)
		if err != nil {
			log.Printf("[FIM Scheduler] preflight failed policy=%d server=%d err=%v", policy.ID, target.ServerID, err)
			return
		}
		if skipReason != "" {
			now := time.Now()
			_ = database.DB.Model(&FIMPolicyTarget{}).
				Where("policy_id = ? AND server_id = ?", policy.ID, target.ServerID).
				Updates(map[string]any{
					"last_scan_at":     &now,
					"last_scan_status": "failed",
					"updated_at":       now,
				}).Error
			log.Printf("[FIM Scheduler] scheduled scan skipped policy=%d server=%d reason=%s", policy.ID, target.ServerID, skipReason)
			return
		}

		if _, err := s.service.RunScan(policy.ID, target.ServerID, "scheduled", fimSchedulerOperator); err != nil {
			log.Printf("[FIM Scheduler] scheduled scan failed policy=%d server=%d err=%v", policy.ID, target.ServerID, err)
			return
		}
		log.Printf("[FIM Scheduler] scheduled scan completed policy=%d server=%d", policy.ID, target.ServerID)
	}()
}

func (s *fimScheduler) preflightTarget(serverID uint) (string, error) {
	var server cmdb.Server
	if err := database.DB.Select("id", "hostname", "ip", "ssh_port").First(&server, serverID).Error; err != nil {
		return "", err
	}

	port := defaultInt(server.SSHPort, 22)
	hostnames := []string{server.IP}
	if server.Hostname != "" && server.Hostname != server.IP {
		hostnames = append(hostnames, server.Hostname)
	}

	var knownHost models.FIMKnownHost
	result := database.DB.
		Where("is_enabled = ? AND port = ?", true, port).
		Where("hostname IN ?", hostnames).
		Limit(1).
		Find(&knownHost)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected > 0 {
		return "", nil
	}

	serverScopedResult := database.DB.
		Model(&models.FIMKnownHost{}).
		Where("is_enabled = ? AND port = ? AND server_id = ?", true, port, server.ID).
		Limit(1).
		Find(&knownHost)
	if serverScopedResult.Error != nil {
		return "", serverScopedResult.Error
	}
	if serverScopedResult.RowsAffected > 0 {
		return fmt.Sprintf("server %s:%d has known-host records, but hostname does not match current server address", server.IP, port), nil
	}

	return fmt.Sprintf("server %s:%d has no enabled known-host entry", server.IP, port), nil
}

func shouldRunFIMScheduledScan(now time.Time, lastScanAt *time.Time, intervalSec int) bool {
	if intervalSec <= 0 {
		intervalSec = 300
	}
	if lastScanAt == nil || lastScanAt.IsZero() {
		return true
	}
	return now.Sub(*lastScanAt) >= time.Duration(intervalSec)*time.Second
}