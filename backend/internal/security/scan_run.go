// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"time"

	"github.com/jenvenson/ops-platform/internal/database"
	"github.com/jenvenson/ops-platform/internal/models"
	"gorm.io/gorm"
)

func isTerminalTaskStatus(status string) bool {
	switch status {
	case models.TaskStatusCompleted, models.TaskStatusFailed, models.TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func cloneUpdateMap(updates map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(updates))
	for key, value := range updates {
		cloned[key] = value
	}
	return cloned
}

func buildTaskUpdates(updates map[string]interface{}) map[string]interface{} {
	taskUpdates := cloneUpdateMap(updates)
	for _, key := range []string{"phase", "config_snapshot", "target_snapshot", "summary_snapshot", "cancelled_at", "run_no"} {
		delete(taskUpdates, key)
	}
	return taskUpdates
}

func buildScanRunUpdates(taskUpdates map[string]interface{}) map[string]interface{} {
	runUpdates := make(map[string]interface{})
	for key, value := range taskUpdates {
		switch key {
		case "status", "progress", "message", "high_risk", "medium_risk", "low_risk", "started_at", "completed_at", "phase", "config_snapshot", "target_snapshot", "summary_snapshot", "cancelled_at", "run_no":
			runUpdates[key] = value
		case "total_ips":
			runUpdates["total_targets"] = value
		case "scanned_ips":
			runUpdates["scanned_targets"] = value
		}
	}
	return runUpdates
}

func createScanRunForTask(taskID uint) (*models.SecurityScanRun, error) {
	var run models.SecurityScanRun
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var task models.SecurityScanTask
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}

		run = models.SecurityScanRun{
			TaskID:         task.ID,
			TaskName:       task.Name,
			TargetType:     task.TargetType,
			Target:         task.Target,
			ScanType:       task.ScanType,
			Status:         task.Status,
			Progress:       task.Progress,
			TotalTargets:   task.TotalIPs,
			ScannedTargets: task.ScannedIPs,
			Message:        task.Message,
			HighRisk:       task.HighRisk,
			MediumRisk:     task.MediumRisk,
			LowRisk:        task.LowRisk,
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			TriggeredBy:    task.CreatedBy,
		}
		if err := tx.Create(&run).Error; err != nil {
			return err
		}

		if err := initializePhase1RunFields(tx, task, run.ID); err != nil {
			return err
		}

		return tx.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"current_run_id": run.ID,
			"latest_run_id":  run.ID,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func UpdateTaskAndCurrentRun(taskID uint, updates map[string]interface{}) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var task models.SecurityScanTask
		if err := tx.Select("current_run_id").First(&task, taskID).Error; err != nil {
			return err
		}

		taskUpdates := buildTaskUpdates(updates)
		runUpdates := buildScanRunUpdates(updates)
		support := getScanPhase1SchemaSupport(tx)
		if !support.runExtensions {
			for _, key := range []string{"phase", "config_snapshot", "target_snapshot", "summary_snapshot", "cancelled_at", "run_no"} {
				delete(runUpdates, key)
			}
		}

		status, hasStatus := taskUpdates["status"].(string)
		if hasStatus && isTerminalTaskStatus(status) {
			taskUpdates["current_run_id"] = nil
			if _, exists := runUpdates["completed_at"]; !exists {
				runUpdates["completed_at"] = time.Now()
			}
			if support.runExtensions {
				if _, exists := runUpdates["phase"]; !exists {
					runUpdates["phase"] = phase1TerminalRunPhase(status)
				}
				if status == models.TaskStatusCancelled {
					if _, exists := runUpdates["cancelled_at"]; !exists {
						runUpdates["cancelled_at"] = time.Now()
					}
				}
			}
		}

		if err := tx.Model(&models.SecurityScanTask{}).Where("id = ?", taskID).Updates(taskUpdates).Error; err != nil {
			return err
		}

		if task.CurrentRunID != nil && len(runUpdates) > 0 {
			if err := tx.Model(&models.SecurityScanRun{}).Where("id = ?", *task.CurrentRunID).Updates(runUpdates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func deleteTaskRuns(tx *gorm.DB, taskID uint) error {
	return tx.Where("task_id = ?", taskID).Delete(&models.SecurityScanRun{}).Error
}